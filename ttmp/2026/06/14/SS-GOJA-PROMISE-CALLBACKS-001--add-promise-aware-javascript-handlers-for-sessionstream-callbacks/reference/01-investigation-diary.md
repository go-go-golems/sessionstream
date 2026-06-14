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
