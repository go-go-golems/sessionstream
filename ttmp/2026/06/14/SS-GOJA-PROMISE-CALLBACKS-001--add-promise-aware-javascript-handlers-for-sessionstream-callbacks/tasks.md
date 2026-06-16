# Tasks

## Investigation

- [x] Create ticket workspace, design document, diary, tasks, and changelog.
- [x] Identify current synchronous JS callback adapters in `pkg/js/modules/sessionstream/api_callbacks.go`.
- [x] Identify TypeScript declarations that currently expose synchronous callback return types.

## Implementation

### 1. Promise detection and awaiting helper

- [x] Investigate existing go-go-goja runtime/event-loop helpers for Promise awaiting or microtask draining.
- [x] Add a small sessionstream JS helper that accepts a `goja.Value` and returns the resolved value if it is a Promise/thenable.
- [x] Ensure rejected Promises are converted into Go errors with useful callback context.
- [x] Ensure non-Promise values continue to work without behavior changes.
- [x] Ensure waiting is context-aware and respects cancellation/deadlines.

### 2. Command handler support

- [x] Update `moduleRuntime.commandHandler` so JS command callbacks may return `Promise<void>`.
- [x] Add `hub.submitAsync(...)` for Promise-returning command handlers invoked from JavaScript.
- [x] Ensure synchronous `hub.submit(...)` reports pending Promise callbacks instead of deadlocking.
- [x] Ensure `publisher.publish(...)` still uses `runtimebridge.CurrentOwnerContext(m.vm)` and remains safe from Express-handler reentry deadlocks.
- [x] Add `publisher.publishAsync(...)` for async handlers that publish events into async projections.
- [x] Add tests for async command success, async command rejection, and sync command compatibility.

### 2a. Promise-native JavaScript API redesign

- [x] Add a second design document for Promise-native `submit`/`publish` and the decision not to expose enqueue.
- [x] Replace JS `hub.submitAsync(...)` with Promise-returning `hub.submit(...)`.
- [x] Replace JS `publisher.publishAsync(...)` with Promise-returning `publisher.publish(...)`.
- [x] Remove JS `submitAsync(...)` and `publishAsync(...)` from the public API.
- [x] Ensure there are no JS `submitSync(...)` or `publishSync(...)` APIs.
- [x] Update tests, README, TypeScript declarations, and examples to use `await hub.submit(...)` and `await pub.publish(...)`.

### 2b. Queue/accepted-for-processing API

- [x] Remove the experimental local `hub.enqueue(...)` API.
- [x] Remove per-hub FIFO queue state from JavaScript hub wrappers.
- [x] Remove enqueue TypeScript declarations, tests, and README references.
- [x] Document that future queueing should be a separate event-bus or command-bus design, not a local in-memory command queue.

### 3. Projection support

- [x] Update `moduleRuntime.uiProjection` so callbacks may return `UIEvent[] | Promise<UIEvent[]>`.
- [x] Update `moduleRuntime.timelineProjection` so callbacks may return `TimelineEntity[] | Promise<TimelineEntity[]>`.
- [x] Decode projection arrays only after Promise resolution.
- [x] Preserve projection error policy behavior for rejected Promises.
- [x] Add tests for async UI projection success and async timeline projection success.
- [x] Add explicit rejected async UI/timeline projection tests.

### 4. Event loop/runtime-owner integration

- [x] Add tests using xgoja/go-go-goja runtime owner rather than only bare `goja.Runtime`.
- [x] Verify async callback support avoids reentrant runtime-owner deadlocks by using `submitAsync` / `publishAsync`.
- [x] Add an Express-handler regression using `await hub.submit(...)`.
- [ ] Verify callbacks invoked by bus consumers preserve the owner context and do not call Goja from arbitrary goroutines.
- [x] Decide whether Promise support requires a go-go-goja runtime/event-loop dependency in the sessionstream module options.

### 5. TypeScript and docs

- [x] Update `pkg/js/modules/sessionstream/typescript.go` declarations:
  - `command(... handler: (...) => void | Promise<void>)`
  - `uiProjection(... handler: (...) => UIEvent[] | Promise<UIEvent[]>)`
  - `timelineProjection(... handler: (...) => TimelineEntity[] | Promise<TimelineEntity[]>)`
- [x] Update `pkg/js/modules/sessionstream/README.md` with async callback examples and caveats.
- [x] Update `examples/goja-chatdemo-server/verbs/chatbot.js` to use literal `require("fs:assets")` after parser-backed sourcegraph support.
- [ ] Update `examples/goja-chatdemo-server/README.md` if the example switches to a real async model call.

### 6. Validation and bookkeeping

- [x] Run `go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1`.
- [x] Run `go test ./... -count=1` in sessionstream.
- [x] Run `make -C examples/goja-chatdemo-server smoke` if the example remains present and buildable.
- [x] Update this diary after each implementation step.
- [x] Relate modified files to the design document with absolute paths.
- [x] Update changelog with each committed step.
- [x] Run `docmgr --root ttmp doctor --ticket SS-GOJA-PROMISE-CALLBACKS-001 --stale-after 30`.
