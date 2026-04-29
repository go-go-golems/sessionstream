# Tasks

## TODO

### Framework
- [x] Create `evtstream` package root and compile-only files listed in EVT-STREAM-002.
- [x] Create separate `systemlab` app/package/repo shell with its own README and dev entrypoint.
- [x] Choose public integration seam for Systemlab (HTTP/WS preferred; in-process allowed only if it mirrors the future public API).
- [x] Add architecture overview page in Systemlab.

### Systemlab and validation
- [x] Add placeholder navigation for Phase 1 through Phase 6 labs.
- [x] Write boundary contract document inside the phase design doc and copy key rules into Systemlab README.
- [x] Add compile/build commands for both codebases to CI or local Make targets.
- [x] Verify that no code in Systemlab imports legacy webchat internals.
- [x] Record follow-up tasks for Phase 1 implementation handoff.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
