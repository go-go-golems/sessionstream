# Tasks

## TODO

### Framework
- [x] Design SQL schema for timeline entities and per-session cursor rows.
- [x] Implement SQL `HydrationStore` with atomic `Apply` + cursor advancement.
- [x] Add migration/bootstrap path for the SQL store.
- [x] Implement store reset helper for local lab/testing use.

### Systemlab and validation
- [x] Add integration tests for restart/resume correctness.
- [x] Build Systemlab Phase 5 page with memory/SQL mode toggle and restart controls.
- [x] Show pre/post restart cursor and snapshot state side by side.
- [x] Add comparison badges highlighting any divergence between backends.
- [x] Document dev setup for local SQL store and cleanup procedure.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate and inspect the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
