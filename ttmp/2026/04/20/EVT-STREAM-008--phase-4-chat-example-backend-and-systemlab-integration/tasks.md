# Tasks

## TODO

### Framework
- [x] Add `examples/chat` package tree.
- [x] Register chat command/event/UI/entity schemas.
- [x] Implement `StartInference` and `StopInference` handlers.
- [x] Implement timeline projection reducing token deltas into `ChatMessage`.

### Systemlab and validation
- [x] Implement UI projection emitting message append/finish UI events.
- [x] Add Systemlab Phase 4 page with prompt control, presets, event panels, and timeline panel.
- [x] Add a cancellation/stop scenario and verify final state is coherent.
- [x] Document which pieces of the example are teaching material vs production candidates.
- [x] Capture at least one exported transcript to use in later docs or regression fixtures.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
