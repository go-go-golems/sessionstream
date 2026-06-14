---
Title: Investigation diary
Ticket: SS-GOJA-PROMISE-CALLBACKS-001
Status: active
Topics:
    - goja
    - js-bindings
    - sessionstream
    - async
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Diary for adding Promise-aware JavaScript callback handling to sessionstream Goja bindings."
LastUpdated: 2026-06-14T00:00:00Z
WhatFor: "Chronological implementation notes for Promise-aware sessionstream JS callbacks."
WhenToUse: "Read before implementing or reviewing SS-GOJA-PROMISE-CALLBACKS-001."
---

# Diary

## Goal

This diary captures the design and implementation work for adding Promise-aware JavaScript command and projection callbacks to the sessionstream Goja bindings.

## Step 1: Create ticket and define async callback scope

I created a dedicated docmgr ticket for Promise-aware sessionstream callbacks. The initial design documents the current synchronous callback bridge, the desired `Promise` return behavior for commands and projections, and the runtime-owner constraints that make this more than a simple `goja.Promise` type check.

The ticket scopes async support to the JavaScript/xgoja binding layer. The core Go `CommandHandler` type is already context-aware and can do blocking/asynchronous Go work internally, but the JS callback adapter currently ignores returned Promises or tries to decode them immediately.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to add promise aware JS handlers for sessionstream callbacks"

**Assistant interpretation:** Create a structured sessionstream docmgr ticket describing the work needed to support Promise-returning JavaScript command and projection callbacks.

**Inferred user intent:** Preserve the design and task breakdown before implementing async JS callback support so future work can proceed safely around Goja runtime ownership and callback semantics.

### What I did

- Created ticket `SS-GOJA-PROMISE-CALLBACKS-001` under `sessionstream/ttmp`.
- Added design document:
  - `design-doc/01-promise-aware-sessionstream-js-callback-design.md`
- Added this investigation diary.
- Replaced the placeholder task list with a detailed implementation plan covering:
  - Promise detection/awaiting helper;
  - async command handlers;
  - async UI/timeline projections;
  - runtime-owner/event-loop integration;
  - TypeScript declaration updates;
  - docs, validation, and bookkeeping.
- Reviewed current callback and type declaration files:
  - `pkg/js/modules/sessionstream/api_callbacks.go`
  - `pkg/js/modules/sessionstream/typescript.go`
  - `pkg/sessionstream/handler.go`
  - `pkg/sessionstream/hub.go`

### Why

- Current JS callbacks are synchronous: command callbacks discard the return value, while projection callbacks immediately decode the return value as arrays.
- Real chatbot/model integrations need idiomatic `async` JavaScript handlers that can `await` model calls and publish completion events.
- Promise support must preserve the runtime-owner constraints that prevent Goja concurrent access and reentrant Express-handler deadlocks.

### What worked

- `docmgr --root ttmp ticket create-ticket` created the ticket workspace successfully.
- `docmgr --root ttmp doc add` created the design doc and diary.
- The existing code has a clear adapter boundary in `api_callbacks.go`, which is the right implementation target.
- The current TypeScript declarations make the required API change easy to see: callback return types should widen from synchronous-only returns to `T | Promise<T>`.

### What didn't work

N/A. This step was ticket creation and design scoping only.

### What I learned

- Promise support should be added for all callback classes, not just command handlers, because projections also need async enrichment and have deterministic returned data shapes.
- `hub.submit` should wait for a command handler's Promise to settle to preserve the synchronous Go `Submit` contract.
- Rejected Promises should use existing error paths rather than becoming background logs.

### What was tricky to build

- The hard part will be event-loop ownership, not callback registration. Sessionstream already routes callbacks through `runtimeOwner.Call`, and recent fixes use `runtimebridge.CurrentOwnerContext(m.vm)` to avoid reentrant deadlocks. Promise awaiting must preserve those invariants.
- Projection callbacks need resolved values before decoding. Returning a Promise from `uiProjection` or `timelineProjection` cannot be treated as an array until it is fulfilled.

### What warrants a second pair of eyes

- Which go-go-goja runtime/event-loop helper should be used to await Promises safely.
- Whether non-native thenables should be supported or only Goja native Promises.
- Whether async projection latency needs explicit documentation for hydration/fanout users.

### What should be done in the future

- Inspect go-go-goja Promise/event-loop support before implementing a custom wait loop.
- Add failing tests for Promise-returning command and projection callbacks.
- Implement Promise-aware adapters and update TypeScript declarations.

### Code review instructions

- Start with `pkg/js/modules/sessionstream/api_callbacks.go` to see the synchronous callback bridge.
- Review `pkg/js/modules/sessionstream/typescript.go` to see current callback type declarations.
- Review `pkg/sessionstream/hub.go` and `pkg/sessionstream/handler.go` for the synchronous Go command pipeline contract.

### Technical details

The target JavaScript shapes are:

```js
hub.command("Ask", async (cmd, session, pub) => {
  const answer = await model.ask(cmd.payload.prompt)
  pub.publish("AnswerDone", { text: answer })
})

hub.uiProjection(async (event, session, view) => {
  return [{ name: "DisplayUpdated", payload: await enrich(event.payload) }]
})

hub.timelineProjection(async (event, session, view) => {
  return [{ kind: "message", id: "1", payload: await buildEntity(event) }]
})
```

The corresponding TypeScript callback returns should become:

```ts
void | Promise<void>
UIEvent[] | Promise<UIEvent[]>
TimelineEntity[] | Promise<TimelineEntity[]>
```

## Step 2: Add Promise-aware callbacks plus async submit/publish APIs

I implemented the first Promise-aware callback slice. Command handlers, UI projections, and timeline projections now pass callback return values through a Promise-aware helper that waits for fulfilled Promises, converts rejected Promises into Go errors with callback labels, and decodes projection arrays only after resolution.

During testing I found an important JavaScript-stack constraint: a synchronous `hub.submit(...)` call made from JavaScript cannot safely block waiting for a pending Promise returned by a JS command handler, because the JavaScript stack must unwind before Goja can run the async continuation. To support idiomatic async handlers, I added explicit async APIs: `hub.submitAsync(...)` and `publisher.publishAsync(...)`. Async command handlers should `await hub.submitAsync(...)`, and async handlers that publish into async projections should `await pub.publishAsync(...)`.

### Prompt Context

**User prompt (verbatim):** "Continue"

**Assistant interpretation:** Start implementing the newly-created Promise-aware callback task list, validate behavior, update docs/diary/tasks, and commit when appropriate.

**Inferred user intent:** Move beyond the design ticket and make the sessionstream JS bindings actually support Promise-returning callbacks.

### What I did

- Added `pkg/js/modules/sessionstream/api_promises.go` with Promise-aware callback execution helpers.
- Updated `commandHandler`, `uiProjection`, and `timelineProjection` in `api_callbacks.go` to resolve Promise return values before continuing.
- Added `hub.submitAsync(...)`, which returns a JavaScript Promise and runs `Hub.Submit` from a Go goroutine so the JS stack can unwind.
- Added `publisher.publishAsync(...)`, which returns a Promise and runs event publication from a Go goroutine so async projections can settle without reentrant owner deadlocks.
- Updated module runtime initialization to discover runtime services from `runtimebridge.Lookup(vm)` when an explicit `Options.RuntimeOwner` was not supplied.
- Updated TypeScript declarations for Promise-capable handlers plus `submitAsync` and `publishAsync`.
- Updated `pkg/js/modules/sessionstream/README.md` with async callback examples and caveats.
- Updated `examples/goja-chatdemo-server/verbs/chatbot.js` to use literal `require("fs:assets")`, now that the go-go-goja sourcegraph parser supports configured colon aliases.
- Added runtime-owner tests for async command success/rejection and async UI/timeline projections.

### Why

- Awaiting a pending Promise from inside the same synchronous JavaScript call deadlocks: the caller is blocking the stack that must unwind before Promise continuations can run.
- Explicit async APIs make the scheduling model clear and preserve backward-compatible synchronous APIs for existing handlers.
- `publishAsync` is needed because command handlers often publish events that immediately run projections; if those projections are async, synchronous `publish` has the same stack-unwind problem as synchronous `submit`.

### What worked

Focused and full tests passed:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
```

The xgoja chatdemo smoke passed after changing the assets import to a literal alias:

```bash
make -C examples/goja-chatdemo-server smoke
```

### What didn't work

The first bare-Goja async test hung because `hub.submit(...)` was called synchronously from JavaScript while the command handler returned a pending Promise. I changed the direct no-owner Promise wait path to reject pending Promises instead of polling forever.

The first runtime-owner test still timed out when it used synchronous `hub.submit(...)`: even with a runtime owner, the Promise continuation could not run while the original JS `hub.submit` call was blocked. This led to the `submitAsync` design.

The next test failed when an async command used synchronous `pub.publish(...)` and the UI projection returned a Promise:

```text
sessionstream.uiProjection.ChatUserMessageAccepted returned a pending Promise during a synchronous owner call; use submitAsync or call from Go
```

That uncovered the same stack-unwind problem for event publication, so I added `publisher.publishAsync(...)` and updated the test to `await pub.publishAsync(...)`.

### What I learned

- Promise-aware callback support is not just a matter of checking `*goja.Promise`; synchronous JS-to-Go entry points need async alternatives.
- `submitAsync` and `publishAsync` are the safe user-facing APIs for Promise-returning callback chains initiated from JavaScript.
- Runtime services can be discovered through `runtimebridge.Lookup(vm)` for engine-created runtimes, which avoids requiring every module registration to explicitly pass `Options.RuntimeOwner`.

### What was tricky to build

- The hardest invariant is avoiding reentrant owner deadlocks. A Promise that is pending during a synchronous owner call must not be waited on by blocking that same JavaScript stack.
- The implementation distinguishes fulfilled/rejected Promises from pending Promises. Fulfilled Promises can be handled immediately; pending Promises from synchronous owner calls produce a clear error telling users to use async APIs.
- Async projections required `publishAsync`, not just `submitAsync`, because event publication runs projections as part of the same pipeline.

### What warrants a second pair of eyes

- Review whether `submitAsync` and `publishAsync` should be the final API names.
- Review whether rejected Promises should reject with strings or JS Error objects when settling `submitAsync`/`publishAsync` Promises. The current implementation rejects with `err.Error()`.
- Add explicit tests for rejected async UI/timeline projections and an Express route that uses `await hub.submitAsync(...)`.

### What should be done in the future

- Add explicit rejected projection tests.
- Add an Express handler regression for async submit.
- Consider updating the chatdemo server to use `submitAsync` if it gains a real async model call.

### Code review instructions

- Start with `pkg/js/modules/sessionstream/api_promises.go`.
- Then review:
  - `pkg/js/modules/sessionstream/api_callbacks.go`
  - `pkg/js/modules/sessionstream/api_hub.go`
  - `pkg/js/modules/sessionstream/module.go`
  - `pkg/js/modules/sessionstream/typescript.go`
  - `pkg/js/modules/sessionstream/module_test.go`
- Validate with:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```

### Technical details

The safe JavaScript pattern is now:

```js
hub.command("ChatStartInference", async (cmd, session, pub) => {
  const answer = await model.ask(cmd.payload.prompt)
  await pub.publishAsync("ChatUserMessageAccepted", buildEvent(answer))
})

await hub.submitAsync("s-1", "ChatStartInference", buildCommand("hello"))
```

Synchronous APIs remain for synchronous callback chains:

```js
hub.submit("s-1", "ChatStartInference", command)
pub.publish("ChatUserMessageAccepted", event)
```
