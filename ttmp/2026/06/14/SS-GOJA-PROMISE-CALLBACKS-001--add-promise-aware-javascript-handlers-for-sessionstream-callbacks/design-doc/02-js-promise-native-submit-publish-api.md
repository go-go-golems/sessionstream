---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat_goja.pb.go
      Note: Generated Goja builder for custom trace event
    - Path: examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto
      Note: Custom InferenceTraceEvent protobuf schema
    - Path: examples/goja-chatdemo-server/README.md
      Note: Documents intentional streaming delay
    - Path: examples/goja-chatdemo-server/assets/public/app.css
      Note: Trace pane styling
    - Path: examples/goja-chatdemo-server/assets/public/app.js
      Note: |-
        Displays trace status in browser UI
        Trace UI event rendering
    - Path: examples/goja-chatdemo-server/assets/public/index.html
      Note: Separate trace pane markup
    - Path: examples/goja-chatdemo-server/cmd/smoke-client/main.go
      Note: Cross-process smoke client post/listen split
    - Path: examples/goja-chatdemo-server/verbs/chatbot.js
      Note: |-
        Uses timer.sleep and Promise-native publish to slow fake streaming
        Publishes custom trace protobuf from xgoja JavaScript
        Trace UI event projection no longer overwrites chat message
        Shared server source without Redis-only CLI verbs
    - Path: examples/goja-chatdemo-server/xgoja.yaml
      Note: Adds go-go-goja core timer module for visible streaming delay
    - Path: examples/goja-redis-chatdemo-server/README.md
      Note: Documents multi-source jsverb pattern and Redis CLI semantics
    - Path: examples/goja-redis-chatdemo-server/cmd/redis-host/main.go
      Note: Custom Redis/Watermill xgoja host
    - Path: examples/goja-redis-chatdemo-server/docker-compose.yml
      Note: Redis service for Watermill example
    - Path: examples/goja-redis-chatdemo-server/verbs/redis_tools.js
      Note: Redis-only CLI jsverbs
    - Path: examples/goja-redis-chatdemo-server/xgoja.yaml
      Note: |-
        Multiple jsverb source split for Redis-only tools
        Runtime-package generation and multi-source jsverbs
    - Path: pkg/js/modules/sessionstream/api_callbacks.go
      Note: Promise-native publisher.publish
    - Path: pkg/js/modules/sessionstream/api_hub.go
      Note: |-
        Promise-native submit without local enqueue
        JS hub.publish for typed event injection
    - Path: pkg/js/modules/sessionstream/api_promises.go
      Note: Shared Promise settlement helper
    - Path: pkg/js/modules/sessionstream/module_test.go
      Note: Async submit/publish regressions with enqueue removed
    - Path: pkg/js/modules/sessionstream/provider/provider.go
      Note: Host-service based hub option injection
    - Path: pkg/js/modules/sessionstream/typescript.go
      Note: Final Promise-native JS declarations
    - Path: pkg/sessionstream/hub.go
      Note: Go Hub.Publish event path
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---







# JS Promise-native submit and publish API

## Executive summary

The JavaScript-facing sessionstream API should be Promise-native and small: `hub.submit(...)` and `publisher.publish(...)` return Promises, and there is no JavaScript `submitSync`, `publishSync`, `submitAsync`, `publishAsync`, or `enqueue` API.

The earlier in-memory `hub.enqueue(...)` experiment was removed. It blurred two separate concepts: command submission and event-bus publication. Sessionstream's existing Watermill integration is event-oriented, so a future queue/bus API should be designed around the event bus or an explicit command-bus extension rather than a local per-hub goroutine queue.

## Problem statement

The core Go API, `(*Hub).Submit`, is synchronous: once it returns, command handling and local projection work has completed or failed. JavaScript has a different constraint: an async function returns a pending Promise and can only continue after the current JavaScript stack unwinds. If a JS-facing bridge blocks that same stack while waiting for the Promise, the continuation cannot run.

The JavaScript API therefore needs asynchronous completion semantics:

- `await hub.submit(...)`: command handler, event publication, projections, storage, and fanout completed or failed.
- `await publisher.publish(...)`: event publication/projection completed or failed.

It should not expose a local command enqueue API until the system has a clear command-bus design.

## Goals

1. Make the JavaScript API idiomatic: Promise-returning methods use the natural names `submit` and `publish`.
2. Avoid exposing sync variants in JavaScript.
3. Avoid exposing `Async` suffixes for normal Promise-returning JS methods.
4. Preserve Go's synchronous `Hub.Submit` behavior.
5. Keep command queueing out of this API until a durable command-bus design exists.

## Non-goals

- This design does not replace the core Go command dispatch API.
- This design does not introduce command queueing.
- This design does not introduce a durable external command bus.
- This design does not change existing event-bus semantics.

## Proposed JavaScript API

```ts
interface Hub {
  submit(sessionId: string, name: string, payload: unknown): Promise<void>
  snapshot(sessionId: string): Snapshot
  command(name: string, handler: CommandHandler): this
  uiProjection(handler: UIProjectionHandler): this
  timelineProjection(handler: TimelineProjectionHandler): this
  run(): void
  shutdown(): void
}

interface Publisher {
  publish(name: string, payload: unknown): Promise<void>
}

type CommandHandler = (
  cmd: Command,
  session: Session,
  publisher: Publisher,
) => void | Promise<void>
```

## Semantics

### `await hub.submit(...)`

`submit` decodes the command payload synchronously enough to reject immediately for type errors, then runs `Hub.Submit` off the current JavaScript stack and returns a Promise. The Promise resolves when `Hub.Submit` completes and rejects if command handling, event publication, projection, storage, or fanout returns an error.

This is the JavaScript equivalent of Go's synchronous `Hub.Submit`, but expressed as a Promise so async JS callbacks can make progress.

### `await publisher.publish(...)`

`publish` decodes the event payload and returns a Promise. The Promise resolves when the underlying `EventPublisher.Publish` call completes and rejects on publication/projection errors.

Command handlers should either `await publisher.publish(...)` or `return publisher.publish(...)` if publication must be part of command completion.

## Queueing decision

Do not expose `hub.enqueue(...)` in this ticket.

Rationale:

1. A local in-memory command queue is not the same semantic boundary as a durable/event-bus queue.
2. Sessionstream's current Watermill integration is event-oriented: it encodes events and consumes them for projection/application.
3. Command queueing would require an explicit command envelope, topic, consumer, ordering policy, and failure model.
4. Adding a local queue now risks hardening the wrong API.

Future queue work should be a separate design. Depending on the use case, it may be one of:

- event-bus publication semantics through `publisher.publish(...)`, using existing Watermill event infrastructure;
- a new explicit command-bus API such as `commands.enqueue(...)` or `hub.enqueueCommand(...)`, backed by a command envelope and consumer;
- application-level queueing outside sessionstream.

## Decision records

### Decision: JS `submit` and `publish` are always Promise-returning

- Context: JS async callbacks cannot be awaited by blocking the same JS stack.
- Options: keep `submitAsync`, add sync variants, or make `submit` Promise-native.
- Decision: use Promise-native `submit`/`publish` and no sync JS variants.
- Rationale: this is idiomatic JavaScript and avoids overloaded sync/async behavior.
- Consequences: existing JS snippets must `await`, `return`, or deliberately ignore returned Promises.
- Status: accepted.

### Decision: remove local `hub.enqueue`

- Context: an in-memory per-hub queue was implemented as an experiment, but the desired queue boundary is likely Watermill/event-bus or a future command bus.
- Options: keep local enqueue, rename it, or remove it.
- Decision: remove it.
- Rationale: avoid baking in a misleading command-queue abstraction.
- Consequences: JS has only completion-oriented `submit` and `publish` for now.
- Status: accepted.

## Implementation plan

1. Remove `hub.enqueue(...)` implementation, TypeScript declarations, tests, and README references.
2. Keep Promise-native `hub.submit(...)` and `publisher.publish(...)`.
3. Update task list, diary, changelog, and doc relations.
4. Validate focused and full test suites.

## Validation

Run:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```
