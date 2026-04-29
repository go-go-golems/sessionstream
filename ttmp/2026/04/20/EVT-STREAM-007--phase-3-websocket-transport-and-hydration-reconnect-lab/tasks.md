# Tasks

## TODO

### Framework
- [x] Implement websocket transport package under `evtstream/transport/ws`.
- [x] Implement connection registry keyed by `ConnectionId`.
- [x] Implement subscribe/unsubscribe protocol and session subscription updates.
- [x] Ensure snapshot retrieval and delivery occurs before live stream fan-out.

### Systemlab and validation
- [x] Add tests for empty snapshot, populated snapshot, and reconnect after disconnection.
- [x] Build Systemlab Lab 03 with two simulated clients, connection controls, and live trace panels.
- [x] Show store snapshot and client-local received events in one screen.
- [x] Add invariant badges for snapshot-before-live and convergence of final state.
- [x] Prepare hooks for future liveness/tick support without blocking this phase.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
