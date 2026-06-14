---
Title: Promise-aware sessionstream JS callback design
Ticket: SS-GOJA-PROMISE-CALLBACKS-001
Status: active
Topics:
    - goja
    - js-bindings
    - sessionstream
    - async
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/js/modules/sessionstream/api_callbacks.go
      Note: Current synchronous JS command/projection callback adapters to update
    - Path: pkg/js/modules/sessionstream/api_hub.go
      Note: JS hub methods and owner-context pattern relevant to async handlers
    - Path: pkg/js/modules/sessionstream/module_test.go
      Note: Existing Goja/xgoja integration tests to extend
    - Path: pkg/js/modules/sessionstream/typescript.go
      Note: TypeScript callback signatures that need Promise return types
    - Path: pkg/sessionstream/handler.go
      Note: Core synchronous/context-aware Go command handler contract
    - Path: pkg/sessionstream/hub.go
      Note: Submit/projection pipeline semantics that async JS callbacks must preserve
ExternalSources: []
Summary: Design plan for supporting Promise-returning JavaScript command and projection callbacks in sessionstream's Goja bindings.
LastUpdated: 2026-06-14T00:00:00Z
WhatFor: Implementing async/Promise-aware sessionstream JavaScript handlers.
WhenToUse: Use when updating sessionstream Goja callback adapters, tests, and TypeScript declarations.
---


# Promise-aware sessionstream JS callback design

## Executive Summary

The current sessionstream JavaScript bindings adapt JS callbacks as synchronous Go callbacks. This works for simple command handlers and projections, but it does not support idiomatic JavaScript handlers that return Promises from `async` functions. A callback such as `hub.command("Ask", async (...) => { ... })` currently returns immediately with a Promise value that the Go adapter ignores; projection callbacks are even more constrained because the Go adapter tries to decode the returned value as an array immediately.

This ticket adds Promise-aware callback support for the Goja/xgoja sessionstream module. The desired behavior is that command handlers may return `void | Promise<void>`, UI projections may return `UIEvent[] | Promise<UIEvent[]>`, and timeline projections may return `TimelineEntity[] | Promise<TimelineEntity[]>`. Rejected Promises should become Go errors in the same places synchronous throws are errors today.

The main implementation risk is Goja runtime ownership. The existing bindings deliberately route callbacks through `runtimeowner.RuntimeOwner` and use `runtimebridge.CurrentOwnerContext(m.vm)` for reentrant calls from Express handlers. Promise awaiting must preserve that invariant: it must not call Goja from arbitrary goroutines, and it must not reintroduce the deadlocks that were recently fixed for `hub.submit` and `publisher.publish`.

## Problem Statement

Current callback adapters in `pkg/js/modules/sessionstream/api_callbacks.go` call JavaScript functions directly and use the immediate return value:

- `commandHandler` calls the JS function and discards the returned value.
- `uiProjection` calls the JS function and immediately decodes the return value as `[]UIEvent`.
- `timelineProjection` calls the JS function and immediately decodes the return value as `[]TimelineEntity`.

This means these handlers are synchronous only:

```js
hub.command("Ask", (cmd, session, pub) => {
  pub.publish("AnswerStarted", { prompt: cmd.payload.prompt })
  pub.publish("AnswerDone", { text: fakeAnswer(cmd.payload.prompt) })
})
```

But this idiomatic JavaScript shape is not supported correctly:

```js
hub.command("Ask", async (cmd, session, pub) => {
  pub.publish("AnswerStarted", { prompt: cmd.payload.prompt })
  const answer = await model.ask(cmd.payload.prompt)
  pub.publish("AnswerDone", { text: answer })
})
```

The callback returns a Promise, and the Go adapter returns before the Promise resolves. For projections, a Promise return cannot be decoded as an array.

## Current Architecture

### Core Go sessionstream API

The core command handler type in `pkg/sessionstream/handler.go` is synchronous from the caller's perspective but context-aware:

```go
type CommandHandler func(ctx context.Context, cmd Command, sess *Session, pub EventPublisher) error
```

`Hub.Submit` validates the command payload, loads the session, and calls the registered handler. It returns when the handler returns. A Go handler can perform blocking or asynchronous I/O internally, but `Submit` observes it as one synchronous operation.

### JS callback bridge

`pkg/js/modules/sessionstream/api_callbacks.go` creates Go callbacks around JS functions. When `runtimeOwner` is present, calls are routed through:

```go
m.runtimeOwner.Call(ctx, "sessionstream.command."+cmd.Name, call)
```

This is important because Goja runtimes are not safe to access concurrently. The bridge must keep all JS execution on the runtime owner.

### Reentrant owner context

Recent sessionstream work fixed deadlocks by using `runtimebridge.CurrentOwnerContext(m.vm)` for JS-facing methods such as `hub.submit`, `hub.snapshot`, `hub.run`, `hub.shutdown`, and `publisher.publish`. Promise support must keep this pattern: JS callbacks that publish events while being invoked from Express or xgoja-owned code need to preserve the current owner context.

### TypeScript declarations

`pkg/js/modules/sessionstream/typescript.go` currently declares only synchronous callback signatures:

```ts
command(name: string, handler: (cmd: Command, session: Session, publisher: Publisher) => void): this
uiProjection(handler: (event: Event, session: Session, view: TimelineView) => UIEvent[]): this
timelineProjection(handler: (event: Event, session: Session, view: TimelineView) => TimelineEntity[]): this
```

These declarations should be widened after runtime support lands.

## Proposed Solution

### Add a Promise-aware value resolver

Add a small helper in the JS module layer that resolves callback return values:

```go
func (m *moduleRuntime) awaitCallbackValue(ctx context.Context, label string, value goja.Value) (goja.Value, error) {
    // If value is not a Promise/thenable, return it unchanged.
    // If value is fulfilled, return the fulfillment value.
    // If value is rejected, return a Go error annotated with label.
    // Respect ctx cancellation/deadline.
}
```

The exact implementation should first inspect go-go-goja's runtime/event-loop support. If the engine already has Promise-draining or event-loop helpers, reuse them rather than inventing a second event loop. If no helper exists, implement the minimal owner-safe wait mechanism and add tests around it.

### Command handler behavior

Update `commandHandler`:

```go
value, err := fn(goja.Undefined(), cmdValue, m.sessionToJS(sess), publisher)
if err != nil {
    return nil, err
}
_, err = m.awaitCallbackValue(ctx, "sessionstream.command."+cmd.Name, value)
return nil, err
```

Expected JavaScript behavior:

```js
hub.command("Ask", async (cmd, session, pub) => {
  const answer = await model.ask(cmd.payload.prompt)
  pub.publish("AnswerDone", { text: answer })
})
```

`hub.submit(...)` should not return until the async function resolves or rejects.

### Projection behavior

Update `uiProjection` and `timelineProjection` similarly, but decode after Promise resolution:

```go
value, err := fn(...)
if err != nil { return nil, err }
resolved, err := m.awaitCallbackValue(ctx, "sessionstream.uiProjection."+ev.Name, value)
if err != nil { return nil, err }
return m.decodeUIEvents(schemas, resolved)
```

Expected JavaScript behavior:

```js
hub.uiProjection(async (event, session, view) => {
  const display = await enrich(event.payload)
  return [{ name: "DisplayUpdated", payload: display }]
})
```

Projection error policy should treat rejected Promises exactly like synchronous throws: the projection returns an error, and the existing fail/advance policy decides whether the pipeline stops or advances.

## Design Decisions

### Decision: `Submit` waits for command Promise resolution

The core Go API is synchronous from `Submit`'s perspective. Promise-aware JS command handlers should preserve that model: `hub.submit` returns after the handler's Promise settles. If a handler wants fire-and-forget background work, it should explicitly schedule that work and return quickly.

### Decision: projections may be async but still return deterministic arrays

Projection callbacks may return Promises, but the resolved value must still be the same array shape as synchronous projections. This keeps persistence and fanout behavior unchanged.

### Decision: rejected Promises become Go callback errors

A rejected Promise should not be logged and ignored. It should behave like a thrown JS exception, carrying callback context in the Go error message.

### Decision: runtime-owner safety is mandatory

All Promise resolution, callbacks, and continuations that touch Goja must remain on the Goja runtime owner. This is more important than a quick local helper that works only in bare `goja.Runtime` tests.

## Alternatives Considered

### Leave JS callbacks synchronous

Rejected. The xgoja chatbot and real model integrations need idiomatic async JavaScript. Requiring all async work to be implemented in Go defeats the purpose of JS composition.

### Fire-and-forget async command handlers

Rejected as the default. If `hub.submit` returns before the Promise settles, errors become hard to route through the existing pipeline, tests become nondeterministic, and projections cannot use the same model.

### Support async commands but not async projections

Possible, but inconsistent. Projections often need async enrichment when building UI events or timeline entities. If the runtime can safely await one callback kind, it should support all three callback kinds.

## Implementation Plan

1. Inspect go-go-goja runtime/event-loop Promise support and choose the owner-safe waiting primitive.
2. Add tests that demonstrate current behavior fails or is insufficient for Promise-returning command/projection callbacks.
3. Implement `awaitCallbackValue` or equivalent in `pkg/js/modules/sessionstream`.
4. Update `commandHandler`, `uiProjection`, and `timelineProjection` to resolve returned Promises before continuing.
5. Add success and rejection tests for commands, UI projections, and timeline projections.
6. Add xgoja/runtime-owner integration tests, including an Express-handler path if practical.
7. Update TypeScript declarations to include `Promise` return types.
8. Update README/example docs with async callback examples.
9. Run focused and full validation.
10. Update diary, changelog, and file relationships.

## Testing Strategy

Minimum tests:

- sync command still works;
- async command publishes after `await` and `hub.submit` waits;
- rejected async command returns an error;
- sync UI/timeline projections still work;
- async UI/timeline projections resolve arrays correctly;
- rejected async projections enter existing projection error path;
- Promise support works under runtime-owner, not only bare Goja;
- publisher calls inside async handlers preserve owner context.

## Risks and Open Questions

- Which go-go-goja event-loop helper should be reused for Promise waiting?
- Should there be a timeout default, or should only `ctx` control cancellation?
- Should thenables be supported, or only native Goja Promises?
- Are async projections acceptable for all hydration backends, or should docs warn about latency?
- How should stack traces from rejected Promises be formatted?

## References

- `pkg/js/modules/sessionstream/api_callbacks.go`
- `pkg/js/modules/sessionstream/api_hub.go`
- `pkg/js/modules/sessionstream/typescript.go`
- `pkg/js/modules/sessionstream/module_test.go`
- `pkg/sessionstream/handler.go`
- `pkg/sessionstream/hub.go`
- `pkg/js/modules/sessionstream/README.md`
