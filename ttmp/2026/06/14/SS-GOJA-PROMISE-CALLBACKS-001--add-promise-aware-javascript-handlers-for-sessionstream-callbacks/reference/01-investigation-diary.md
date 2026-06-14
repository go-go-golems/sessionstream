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

## Step 3: Redesign the JavaScript API around Promise-native submit/publish and enqueue

I paused implementation to revisit the API shape. The earlier `submitAsync` and `publishAsync` names solved the runtime-owner deadlock, but they exposed an implementation detail instead of the JavaScript semantic model. The revised direction is to make JavaScript `hub.submit(...)` and `publisher.publish(...)` Promise-returning by default, remove sync variants from the JS surface, and reserve `hub.enqueue(...)` for accepted-for-processing semantics.

This creates a cleaner contract: `await hub.submit(...)` means processing completed, while `await hub.enqueue(...)` means the job was accepted and will be processed by a per-hub background worker. That distinction matters for LLM/chatbot flows where some callers want completion and others want fire-and-forget queue behavior.

### Prompt Context

**User prompt (verbatim):** "Ok, update, and remove the submidSync and publishSync entirely. Add a new design document. Also add enqueue. Then add new tasks and implement them one by one, keeping a diary as you work. Commit at appropriate intervals"

**Assistant interpretation:** Redesign the API before continuing: no JS sync variants, natural Promise-returning `submit`/`publish`, a separate `enqueue` API, a new design document, updated tasks, incremental implementation, diary updates, and focused commits.

**Inferred user intent:** Ensure the sessionstream JS API is ergonomic and semantically correct before the Promise-aware implementation becomes baked into the PR.

### What I did

- Added `design-doc/02-js-async-submit-publish-and-enqueue-api.md`.
- Updated `tasks.md` with explicit Promise-native API redesign tasks and queue tasks.
- Captured the distinction between completion semantics (`submit`/`publish`) and accepted-for-processing semantics (`enqueue`).

### Why

- `submitAsync`/`publishAsync` are workable bridge names, but idiomatic JavaScript should use Promise-returning `submit`/`publish`.
- Queue semantics are useful but materially different from direct submission, so they need a separate method instead of changing what `submit` means.

### What worked

- The new design gives implementation steps that can be committed incrementally: first API rename, then enqueue worker, then docs/tests/validation.

### What didn't work

- N/A for this design-only step.

### What I learned

- The API should communicate semantics first and implementation strategy second. The runtime-owner workaround should be invisible to JavaScript users.

### What was tricky to build

- The subtle distinction is that a Promise-returning `submit` still means completion, while `enqueue` only means acceptance. Both are asynchronous in JavaScript, but they should not be interchangeable.

### What warrants a second pair of eyes

- Review whether the first `enqueue` implementation should be in-memory only, or whether it should immediately integrate with Watermill once commands have a bus representation.

### What should be done in the future

- Implement the Promise-native API rename.
- Implement the in-memory per-hub enqueue worker and receipt.

### Code review instructions

- Start with `design-doc/02-js-async-submit-publish-and-enqueue-api.md` and the new sections in `tasks.md`.

### Technical details

Target API:

```js
await hub.submit(sessionId, commandName, payload)
await publisher.publish(eventName, payload)
const receipt = await hub.enqueue(sessionId, commandName, payload)
```

## Step 4: Implement Promise-native submit/publish and in-memory enqueue

I replaced the bridge-shaped `submitAsync` / `publishAsync` JavaScript API with Promise-native `hub.submit(...)` and `publisher.publish(...)`. There are no JS sync variants. Synchronous Go semantics remain in the core `Hub.Submit` API, while JavaScript callers now always see a Promise so async command/projection callbacks can settle without blocking the runtime owner.

I also added `hub.enqueue(...)` as a distinct accepted-for-processing API. It uses a per-hub in-memory FIFO worker in the JavaScript wrapper. `enqueue` resolves with a receipt after the command is accepted into the queue; background processing errors are logged instead of being reported through the already-resolved receipt.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Implement the redesigned Promise-native JS API, add queue semantics, update tests/examples/docs, validate, and record the work.

**Inferred user intent:** Land an ergonomic JS API now, before consumers start relying on the temporary `submitAsync` / `publishAsync` names.

### What I did

- Changed JS `hub.submit(...)` to return a Promise and execute `Hub.Submit` off the current JS stack when runtime services are available.
- Removed the JS `hub.submitAsync(...)` public API.
- Changed JS `publisher.publish(...)` to return a Promise and execute `EventPublisher.Publish` off the current JS stack when runtime services are available.
- Removed the JS `publisher.publishAsync(...)` public API.
- Added `hub.enqueue(...)` with an in-memory per-hub FIFO queue and receipt fields: `accepted`, `id`, `sessionId`, `command`, `depth`.
- Updated TypeScript declarations, README, module tests, Express regression, fanout regression, and the xgoja chatdemo server to use `await hub.submit(...)` and `await pub.publish(...)`.
- Added `TestHubEnqueueAcceptsAndProcessesInBackground`.

### Why

- Promise-native `submit`/`publish` is the idiomatic JavaScript API and avoids exposing implementation-specific `Async` suffixes.
- A separate `enqueue` method preserves a clear semantic split between “processing completed” and “accepted for background processing”.

### What worked

Focused and full validation passed:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```

The Express regression now uses an async route handler with `await hub.submit(...)`, and the fanout regression awaits submission before checking emitted UI batches.

### What didn't work

After changing `hub.submit(...)` to return a Promise, existing tests that called `hub.submit(...)` without awaiting it observed empty snapshots or missing fanout events. The fix was to update the tests and examples to either `await hub.submit(...)` or return/await `pub.publish(...)` from command handlers.

The first enqueue receipt assertion expected the command-generated message id in the receipt, but the receipt intentionally describes queue acceptance, not processing output. I changed the assertion to check receipt metadata and then poll the snapshot for background processing results.

### What I learned

- Making `submit` Promise-native forces call sites to be honest about whether they wait for completion.
- `return pub.publish(...)` is a compact sync-looking command handler style that still gives the command pipeline completion semantics because the handler returns the Promise.
- Queue receipts should not contain command output; they should only describe acceptance.

### What was tricky to build

- The runtime-owner boundary remains the critical invariant. `promiseFromGo` runs Go work outside the current JS stack, then posts Promise settlement back to the runtime services.
- The queue worker cannot directly mutate Goja state from its goroutine; it only calls the Go hub. Any JS callback re-entry goes through the existing runtime-owner callback helper.
- Existing sync-looking examples needed careful updates because ignoring a Promise now deliberately means fire-and-forget behavior.

### What warrants a second pair of eyes

- Review whether the in-memory queue should be explicitly configurable for depth instead of using a fixed default of 64.
- Review whether failed enqueue jobs should call a JS-visible error hook in addition to structured logging.
- Review whether `enqueue` should be disabled without runtime services, as currently implemented, or should support a synchronous no-owner fallback.

### What should be done in the future

- Add queue rejection tests for full queues or invalid payloads.
- Add explicit rejected async UI/timeline projection tests.
- Consider durable command queue support if command enqueue semantics need to survive process restarts.

### Code review instructions

- Start with `pkg/js/modules/sessionstream/api_hub.go` for `submit` and `enqueue`.
- Then review `pkg/js/modules/sessionstream/api_callbacks.go` for Promise-returning `publish`.
- Check `pkg/js/modules/sessionstream/api_promises.go` for runtime-owner-safe Promise settlement.
- Validate with:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```

### Technical details

Current JS API shape:

```js
hub.command("Ask", async (cmd, session, pub) => {
  const answer = await model.ask(cmd.payload.prompt)
  await pub.publish("AnswerDone", buildAnswer(answer))
})

await hub.submit("s-1", "Ask", buildAsk("hello"))
const receipt = await hub.enqueue("s-1", "Ask", buildAsk("background"))
```

## Step 5: Remove local enqueue and keep only async submit/publish

I removed the experimental JavaScript `hub.enqueue(...)` API. After revisiting the semantics, the local in-memory command queue did not fit sessionstream's current model: sessionstream already has event-oriented Watermill infrastructure, while command queueing would require its own command envelope, topic, consumer, ordering model, and failure policy. Keeping a local per-hub queue would risk hardening the wrong abstraction.

The JavaScript API is now intentionally smaller: `hub.submit(...)` and `publisher.publish(...)` are Promise-returning completion APIs, and there are no sync variants, async-suffixed variants, or enqueue method on the JS hub.

### Prompt Context

**User prompt (verbatim):** "ok, remvoe enqueue entirely. stick with async submit / publish"

**Assistant interpretation:** Remove the enqueue API and implementation from the current PR, keep Promise-native submit/publish, update docs/tasks/diary, validate, commit, and push.

**Inferred user intent:** Avoid adding an unclear local command-queue abstraction; keep this ticket focused on async JavaScript submission/publication semantics.

### What I did

- Removed the per-hub in-memory queue implementation from `api_hub.go`.
- Removed the hidden `queue` field from `hubRef`.
- Removed the JS `hub.enqueue(...)` method.
- Removed the enqueue regression test.
- Removed `EnqueueReceipt` and `Hub.enqueue(...)` from TypeScript declarations.
- Removed enqueue references from the Goja module README.
- Rewrote the second design document as `02-js-promise-native-submit-publish-api.md`, documenting why enqueue is intentionally absent.
- Updated `tasks.md` to mark enqueue removal as the accepted path.

### Why

- A local in-memory command queue is not equivalent to event-bus publication and could mislead users about durability, ordering, and failure semantics.
- Existing Watermill support is event-oriented. If command enqueueing is needed later, it should be a separate command-bus design.
- This ticket should remain focused on JS async callback support through Promise-native `submit` and `publish`.

### What worked

Validation passed:

```bash
go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
make -C examples/goja-chatdemo-server smoke
```

### What didn't work

The first focused validation run hit an existing flaky ordering assertion in `pkg/sessionstream`:

```text
--- FAIL: TestHubEventBusGoChannelRoundTrip (0.01s)
    bus_test.go:73:
        Error: "1700000000000000002" is not greater than "1700000000000000003"
```

Rerunning the full suite passed. I did not change `pkg/sessionstream` bus behavior as part of this enqueue removal.

### What I learned

- “Async submit” and “enqueue” are different abstractions: async submit waits for completion via a Promise, while enqueue should mean broker/queue acceptance.
- Because the current Watermill path is event-oriented, command enqueueing should not be improvised in the JS wrapper.

### What was tricky to build

- The main trap was documentation drift: the previous design doc and task list described enqueue as accepted work. I rewrote the design to explicitly record the reversal so future readers understand why enqueue disappeared.

### What warrants a second pair of eyes

- Review whether future command queueing should be represented as `hub.enqueueCommand(...)`, a separate `commands` namespace, or left entirely to application code.
- Review the unrelated bus test ordering flake separately.

### What should be done in the future

- Add explicit rejected async UI/timeline projection tests.
- If queueing is needed, open a separate command-bus/event-bus design ticket.

### Code review instructions

- Start with `pkg/js/modules/sessionstream/api_hub.go` and confirm the public JS hub only exposes Promise-native `submit`.
- Review `pkg/js/modules/sessionstream/api_callbacks.go` for Promise-native `publisher.publish`.
- Confirm `pkg/js/modules/sessionstream/typescript.go` and README have no enqueue API.
- Read `design-doc/02-js-promise-native-submit-publish-api.md` for the rationale.

### Technical details

Final intended JS shape for this ticket:

```js
hub.command("Ask", async (cmd, session, pub) => {
  await pub.publish("AnswerStarted", started)
  await pub.publish("AnswerDone", done)
})

await hub.submit("s-1", "Ask", command)
```

## Step 6: Add rejected async projection regressions

I added explicit tests for rejected Promise-returning UI and timeline projections. The earlier success test proved that async projections can resolve and be decoded, but it did not prove that rejected projection Promises propagate back through `publisher.publish(...)`, command handler completion, and finally `hub.submit(...)`.

The new table-driven regression covers both projection paths and asserts that the final rejected submit Promise includes the projection callback label and original rejection reason.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the remaining implementation checklist after removing enqueue; the obvious open item was explicit rejected async projection tests.

**Inferred user intent:** Finish cleanup and validation tasks for the Promise-native submit/publish work.

### What I did

- Added `TestHubPromiseRejectedProjectionReturnsError` in `pkg/js/modules/sessionstream/module_test.go`.
- Covered an async UI projection that throws after `await Promise.resolve()`.
- Covered an async timeline projection that throws after `await Promise.resolve()`.
- Verified errors contain callback labels:
  - `sessionstream.uiProjection.ChatUserMessageAccepted`
  - `sessionstream.timelineProjection.ChatUserMessageAccepted`
- Marked the rejected projection test task complete in `tasks.md`.

### Why

- Projection rejections need to behave like synchronous projection errors.
- The failure should be visible to the submit caller because `hub.submit(...)` is the JS completion API.
- Callback labels are important for diagnosing which registered handler failed.

### What worked

Focused validation passed:

```bash
go test ./pkg/js/modules/sessionstream -run 'TestHubPromiseRejectedProjectionReturnsError|TestHubPromiseRejectedCommandReturnsError' -count=1 -v
```

### What didn't work

- N/A. The projection rejection propagation behaved as expected.

### What I learned

- The current error chain preserves enough context to identify both the top-level command and the exact failing projection callback.

### What was tricky to build

- The projection rejection is nested: `hub.submit(...)` waits for a command handler, the command handler returns `pub.publish(...)`, and publication runs projections. The test therefore needs to assert on the final submit rejection rather than only on the projection helper.

### What warrants a second pair of eyes

- Review whether rejected JS Promises should reject with strings or JS `Error` objects at the outer `submit`/`publish` API boundary.

### What should be done in the future

- N/A for rejected projection coverage.

### Code review instructions

- Review `TestHubPromiseRejectedProjectionReturnsError` in `pkg/js/modules/sessionstream/module_test.go`.
- Validate with the focused test command above or the full package suite.

### Technical details

The test pattern is:

```js
hub.command("ChatStartInference", (cmd, session, pub) => {
  return pub.publish("ChatUserMessageAccepted", event)
})
hub.uiProjection(async () => {
  await Promise.resolve()
  throw new Error("ui boom")
})
await hub.submit("s-reject", "ChatStartInference", command)
```

## Step 7: Add visible streaming delay to the xgoja chatdemo

I updated the xgoja chatdemo server so the fake assistant response is visibly streamed instead of appearing almost immediately. The example now includes the go-go-goja core provider, imports the `timer` module from JavaScript, and awaits short sleeps between published sessionstream events.

This keeps the app fully self-contained while exercising the Promise-native `pub.publish(...)` and `hub.submit(...)` APIs in a more realistic async command handler. The browser can now show the user event, assistant-started state, token deltas, and final completion over time.

### Prompt Context

**User prompt (verbatim):** "can we add some delayed to see the streaming take place? this is almost immediate."

**Assistant interpretation:** Slow down the fake chatbot response path so manual browser testing visibly shows websocket streaming/projection updates.

**Inferred user intent:** Make the demo useful for visually verifying that the sessionstream WebSocket UI updates stream incrementally, not just that the final state is correct.

### What I did

- Added the `go-go-goja-core` xgoja provider to `examples/goja-chatdemo-server/xgoja.yaml`.
- Selected the `timer` runtime module as `timer`.
- Imported `const timer = require("timer")` in `verbs/chatbot.js`.
- Added `publishWithDelay(...)` helper using `await timer.sleep(...)`.
- Inserted delays after the user-accepted event, assistant-started event, and each token delta.
- Updated the example README to mention the intentional streaming delay.

### Why

- The app already produced multiple sessionstream events, but they were published so quickly that a human tester could barely observe streaming behavior.
- Delays make WebSocket fanout behavior and projection state transitions visible during manual testing.

### What worked

The xgoja smoke test still passed:

```bash
make -C examples/goja-chatdemo-server smoke
```

Focused package validation passed after rerunning the known flaky bus package separately:

```bash
go test ./pkg/sessionstream -count=1
go test ./pkg/js/modules/sessionstream -count=1
```

### What didn't work

The combined focused command hit the existing event-bus ordering flake again:

```text
--- FAIL: TestHubEventBusGoChannelRoundTrip (0.01s)
    bus_test.go:73:
        Error: "1700000000000000002" is not greater than "1700000000000000003"
```

Rerunning `go test ./pkg/sessionstream -count=1` passed. This appears unrelated to the chatdemo timer changes.

### What I learned

- The xgoja core provider is the right way to expose `timer.sleep(ms)` to generated xgoja apps.
- Delayed publishing is a good real-world exercise for the Promise-native callback work because the command handler awaits native Promise-returning helpers and Promise-returning sessionstream publication.

### What was tricky to build

- The delay needed to live inside the JS command handler rather than the frontend, so the server-side sessionstream pipeline itself demonstrates incremental publication over WebSocket.
- Adding `timer` required selecting the core provider/runtime module in `xgoja.yaml`; just requiring it from JS would not be enough for generated xgoja runtime plans.

### What warrants a second pair of eyes

- Review whether the delay durations feel right for manual demos: currently 200ms after the user event, 350ms after assistant start, and 650ms after each token delta.

### What should be done in the future

- Replace `fakeAnswer(...)` with a real async model call when ready; the timer-based delay can then be removed or kept as a dev/demo option.

### Code review instructions

- Review `examples/goja-chatdemo-server/xgoja.yaml` for the added core/timer module.
- Review `examples/goja-chatdemo-server/verbs/chatbot.js` for `publishWithDelay(...)` and delayed event publication.
- Validate with:

```bash
make -C examples/goja-chatdemo-server smoke
```

### Technical details

The important JS helper is:

```js
async function publishWithDelay(pub, name, payload, delayMs = 250) {
  await pub.publish(name, payload)
  await timer.sleep(delayMs)
}
```

## Step 8: Add a custom protobuf trace event to the xgoja chatbot

I added a custom protobuf message to the chatdemo schema and used it from the xgoja chatbot JavaScript. The new `InferenceTraceEvent` is generated into the existing `sessionstream.examples.chatdemo.v1` Goja builder module, registered as a sessionstream event in JavaScript, published during the fake assistant run, and projected into the browser UI as transient assistant status updates.

This demonstrates the full custom-protobuf path: define a proto message, regenerate Go and Goja builder bindings, expose the builders through the xgoja provider, register the message in the sessionstream schema registry, and use the generated JS builder from an xgoja verb.

### Prompt Context

**User prompt (verbatim):** "nvm...\n\nNow let's add a custom protobuf to the xgoja, is that possible? and use that in the chatbot?"

**Assistant interpretation:** Show that custom protobuf messages can be added to the xgoja app by adding one to the chatdemo schema and using its generated builder in the chatbot runtime.

**Inferred user intent:** Verify the end-to-end extensibility story for application-specific protobuf payloads in xgoja/sessionstream JavaScript.

### What I did

- Added `InferenceTraceEvent` to `examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto`.
- Regenerated:
  - `examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat.pb.go`
  - `examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat_goja.pb.go`
- Registered `ChatInferenceTrace` in `examples/chatdemo/chat.go` for the Go chatdemo schema helper.
- Registered `ChatInferenceTrace` from JavaScript in `examples/goja-chatdemo-server/verbs/chatbot.js`.
- Published trace events with `pb.InferenceTraceEvent.builder()` during fake assistant execution.
- Projected trace events into `ChatMessageUpdate` UI events and timeline entity status updates.
- Updated the browser client to show message `status` in the metadata line.

### Why

- This proves custom protobuf messages are not limited to built-in Go-side examples; xgoja can consume newly generated protobuf builders through the provider module.
- Trace events give a visible use for the custom payload without changing the transport protocol.

### What worked

Generation and validation passed:

```bash
PATH="$PWD/.bin:$PATH" go generate ./examples/chatdemo
make -C examples/goja-chatdemo-server smoke
go test ./examples/chatdemo ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
go test ./... -count=1
```

### What didn't work

- N/A. The generated Goja builder exposed the expected `elapsedMs(value: number | string)` method, and the xgoja smoke test passed.

### What I learned

- Adding a custom message to the existing chatdemo proto is enough for the current provider because `sessionstream-chatdemo` exposes the generated builder file module.
- The JavaScript side only needs to register the generated message namespace with `schemas.registerEvent(...)` before publishing it.

### What was tricky to build

- The custom event should not be confused with UI events. It is a backend event (`ChatInferenceTrace`) that projections translate into existing UI/timeline protobufs.
- Timeline projection for trace events needs to preserve any currently accumulated assistant text when updating status, so trace updates do not wipe the displayed message content.

### What warrants a second pair of eyes

- Review whether `InferenceTraceEvent` belongs in the shared `examples/chatdemo` schema or should eventually live in an app-local proto package when xgoja supports app-local protobuf providers more directly.

### What should be done in the future

- If we add app-local proto support to xgoja specs, move this style of custom message into `examples/goja-chatdemo-server/proto/...` and generate an app-local provider.

### Code review instructions

- Start with `examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto` and the generated `chat_goja.pb.go` diff.
- Review `examples/goja-chatdemo-server/verbs/chatbot.js` for `pb.InferenceTraceEvent.builder()` usage.
- Validate with:

```bash
make -C examples/goja-chatdemo-server smoke
go test ./examples/chatdemo ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1
```

### Technical details

The custom event registration and publication pattern is:

```js
schemas.registerEvent("ChatInferenceTrace", pb.InferenceTraceEvent)

await pub.publish("ChatInferenceTrace", pb.InferenceTraceEvent.builder()
  .messageId(assistantID)
  .stage("planning")
  .detail("Custom protobuf trace: planning a fake response")
  .elapsedMs(Date.now() - startedAt)
  .build())
```
