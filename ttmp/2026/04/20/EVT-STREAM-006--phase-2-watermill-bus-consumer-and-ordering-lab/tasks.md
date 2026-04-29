# Tasks

## TODO

### Framework
- [x] Implement Watermill-backed `EventPublisher` with schema validation.
- [x] Implement bus consumer loop and event decode/dispatch pipeline.
- [x] Implement ordinal assigner with stream-id-aware and fallback modes.
- [x] Document required topic/partitioning rule keyed by `SessionId`.

### Systemlab and validation
- [x] Add tests for monotonic ordering under stream-id and no-stream-id cases.
- [x] Build Systemlab Lab 02 with publish controls, burst controls, and restart/reset controls.
- [x] Show raw message metadata and derived ordinal side by side in the lab UI.
- [x] Add assertion panel for monotonic ordering and per-session isolation.
- [x] Prepare the consumer output seam that websocket transport will subscribe to in Phase 3.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
