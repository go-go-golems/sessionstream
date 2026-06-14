# Tasks

## Investigation

- [x] Create ticket workspace, design document, diary, tasks, and changelog.
- [x] Identify current synchronous JS callback adapters in `pkg/js/modules/sessionstream/api_callbacks.go`.
- [x] Identify TypeScript declarations that currently expose synchronous callback return types.

## Implementation

### 1. Promise detection and awaiting helper

- [ ] Investigate existing go-go-goja runtime/event-loop helpers for Promise awaiting or microtask draining.
- [ ] Add a small sessionstream JS helper that accepts a `goja.Value` and returns the resolved value if it is a Promise/thenable.
- [ ] Ensure rejected Promises are converted into Go errors with useful callback context.
- [ ] Ensure non-Promise values continue to work without behavior changes.
- [ ] Ensure waiting is context-aware and respects cancellation/deadlines.

### 2. Command handler support

- [ ] Update `moduleRuntime.commandHandler` so JS command callbacks may return `Promise<void>`.
- [ ] Ensure `hub.submit(...)` waits for the command Promise before returning.
- [ ] Ensure `publisher.publish(...)` still uses `runtimebridge.CurrentOwnerContext(m.vm)` and remains safe from Express-handler reentry deadlocks.
- [ ] Add tests for async command success, async command rejection, and sync command compatibility.

### 3. Projection support

- [ ] Update `moduleRuntime.uiProjection` so callbacks may return `UIEvent[] | Promise<UIEvent[]>`.
- [ ] Update `moduleRuntime.timelineProjection` so callbacks may return `TimelineEntity[] | Promise<TimelineEntity[]>`.
- [ ] Decode projection arrays only after Promise resolution.
- [ ] Preserve projection error policy behavior for rejected Promises.
- [ ] Add tests for async UI projection success/rejection and async timeline projection success/rejection.

### 4. Event loop/runtime-owner integration

- [ ] Add tests using xgoja/go-go-goja runtime owner rather than only bare `goja.Runtime`.
- [ ] Verify async callbacks can be invoked from Express handlers without reentrant runtime-owner deadlocks.
- [ ] Verify callbacks invoked by bus consumers preserve the owner context and do not call Goja from arbitrary goroutines.
- [ ] Decide whether Promise support requires a go-go-goja runtime/event-loop dependency in the sessionstream module options.

### 5. TypeScript and docs

- [ ] Update `pkg/js/modules/sessionstream/typescript.go` declarations:
  - `command(... handler: (...) => void | Promise<void>)`
  - `uiProjection(... handler: (...) => UIEvent[] | Promise<UIEvent[]>)`
  - `timelineProjection(... handler: (...) => TimelineEntity[] | Promise<TimelineEntity[]>)`
- [ ] Update `pkg/js/modules/sessionstream/README.md` with async callback examples and caveats.
- [ ] Update `examples/goja-chatdemo-server/README.md` if the example switches to a real async model call.

### 6. Validation and bookkeeping

- [ ] Run `go test ./pkg/js/modules/sessionstream ./pkg/sessionstream -count=1`.
- [ ] Run `go test ./... -count=1` in sessionstream.
- [ ] Run `make -C examples/goja-chatdemo-server smoke` if the example remains present and buildable.
- [ ] Update this diary after each implementation step.
- [ ] Relate modified files to the design document with absolute paths.
- [ ] Update changelog with each committed step.
- [ ] Run `docmgr --root ttmp doctor --ticket SS-GOJA-PROMISE-CALLBACKS-001 --stale-after 30`.
