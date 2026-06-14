---
Title: ""
Ticket: ""
Status: ""
Topics: []
DocType: ""
Intent: ""
Owners: []
RelatedFiles:
    - Path: examples/goja-chatdemo-server/verbs/chatbot.js
      Note: xgoja chatdemo uses await submit/publish
    - Path: pkg/js/modules/sessionstream/api_callbacks.go
      Note: Promise-native publisher.publish implementation
    - Path: pkg/js/modules/sessionstream/api_hub.go
      Note: Promise-native submit and in-memory enqueue implementation
    - Path: pkg/js/modules/sessionstream/api_promises.go
      Note: Shared Promise settlement helper
    - Path: pkg/js/modules/sessionstream/module_test.go
      Note: Promise-native submit/publish and enqueue regressions
    - Path: pkg/js/modules/sessionstream/typescript.go
      Note: Promise-native TypeScript declarations
ExternalSources: []
Summary: ""
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# JS async submit/publish and enqueue API

## Executive summary

The JavaScript-facing sessionstream API should be Promise-native. `hub.submit(...)` and `publisher.publish(...)` should always return Promises in JavaScript, because Promise-returning handlers and projections need the JavaScript stack to unwind before continuations can run. The earlier `submitAsync(...)` and `publishAsync(...)` names are implementation-shaped bridge names; they should not be the primary API.

This design removes separate sync entrypoints from the JavaScript API. There should be no `submitSync(...)` and no `publishSync(...)`, and this ticket should replace `submitAsync(...)` / `publishAsync(...)` with Promise-returning `submit(...)` / `publish(...)`. For fire-and-forget or background work, add a distinct `hub.enqueue(...)` API with explicitly weaker semantics: it acknowledges queue acceptance, not processing completion.

## Problem statement

The core Go API, `(*Hub).Submit`, is synchronous: once it returns, command handling and local projection work has completed or failed. JavaScript has a different constraint: an async function returns a pending Promise and can only continue after the current JavaScript stack unwinds. If a JS-facing synchronous bridge blocks that same stack while waiting for the Promise, the continuation cannot run.

The API therefore needs to make async boundaries visible:

- `await hub.submit(...)`: wait until the command path has completed.
- `await publisher.publish(...)`: wait until event publication/projection has completed.
- `await hub.enqueue(...)`: wait only until work was accepted for background processing.

## Goals

1. Make the JavaScript API idiomatic: Promise-returning methods use the natural names `submit` and `publish`.
2. Avoid exposing sync variants in JavaScript. Existing synchronous handler bodies can still work by returning or awaiting the Promise.
3. Preserve Go's synchronous `Hub.Submit` behavior.
4. Add queue semantics as a separate API instead of overloading `submit`.
5. Keep errors actionable: rejected command/projection Promises reject `submit`/`publish`; queue processing failures are not reported through the already-resolved enqueue Promise.

## Non-goals

- This design does not replace the core Go command dispatch API.
- This design does not introduce a durable external queue. The first implementation can use an in-memory worker queue attached to the JS hub wrapper.
- This design does not provide a queue status API yet, although the receipt shape leaves room for it.

## Proposed JavaScript API

```ts
interface Hub {
  submit(sessionId: string, name: string, payload: unknown): Promise<void>
  enqueue(sessionId: string, name: string, payload: unknown): Promise<EnqueueReceipt>
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

interface EnqueueReceipt {
  accepted: true
  id: string
  sessionId: string
  command: string
  depth: number
}

type CommandHandler = (
  cmd: Command,
  session: Session,
  publisher: Publisher,
) => void | Promise<void>
```

## Semantics

### `await hub.submit(...)`

`submit` decodes the command payload synchronously enough to reject immediately for type errors, then runs `Hub.Submit` off the current JavaScript stack and returns a Promise. The Promise resolves when `Hub.Submit` completes and rejects if command handling, event publication, or projection returns an error.

This is the JavaScript equivalent of Go's synchronous `Hub.Submit`, but expressed as a Promise so async JS callbacks can make progress.

### `await publisher.publish(...)`

`publish` decodes the event payload and returns a Promise. The Promise resolves when the underlying `EventPublisher.Publish` call completes and rejects on publication/projection errors.

Command handlers should either `await publisher.publish(...)` or `return publisher.publish(...)` if publication must be part of command completion.

### `await hub.enqueue(...)`

`enqueue` decodes the command payload and attempts to place a job into the hub's in-memory queue. The returned Promise resolves when the job is accepted, not when it has finished processing. The first implementation should use a per-hub FIFO worker so accepted commands are processed in submission order for that hub wrapper.

A rejected `enqueue` Promise means the command could not be decoded or accepted into the queue. Runtime errors during later processing should be surfaced through logging/error observer mechanisms, not through the enqueue receipt.

## Decision records

### Decision: JS `submit` and `publish` are always Promise-returning

- Context: JS async callbacks cannot be awaited by blocking the same JS stack.
- Options: keep `submitAsync`, add `submitSync`, or make `submit` Promise-native.
- Decision: use Promise-native `submit`/`publish` and no sync JS variants.
- Rationale: this is idiomatic JavaScript and avoids overloaded sync/async behavior.
- Consequences: existing JS snippets must `await`, `return`, or deliberately ignore returned Promises.
- Status: proposed.

### Decision: `enqueue` is separate from `submit`

- Context: queues have weaker completion semantics than direct submission.
- Options: make `submit` enqueue internally, add `fireAndForget`, or add `enqueue`.
- Decision: add `enqueue`.
- Rationale: `enqueue` clearly communicates accepted-for-processing semantics.
- Consequences: applications must choose between completion semantics and background semantics.
- Status: proposed.

### Decision: first queue is in-memory and per JS hub wrapper

- Context: sessionstream already has optional Watermill event-bus support, but commands are not currently a bus-level concept.
- Options: implement a durable command queue now, use Watermill for commands, or start with an in-memory queue.
- Decision: start with a bounded/in-memory per-wrapper FIFO queue.
- Rationale: it validates the JavaScript API without expanding the core Go command model.
- Consequences: enqueue receipts are process-local and not durable.
- Status: proposed.

## Implementation plan

1. Add a new design document and update the task list.
2. Rename the JS API by replacing `submitAsync` with Promise-returning `submit` and replacing `publishAsync` with Promise-returning `publish`.
3. Update tests, README, TypeScript declarations, and examples to use `await hub.submit(...)` / `await pub.publish(...)`.
4. Add a per-hub enqueue worker and `hub.enqueue(...)` receipt.
5. Add tests for queue acceptance and background processing.
6. Validate focused and full test suites, update diary/changelog, then commit in focused intervals.

## Validation

Run:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```
