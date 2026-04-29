# Tasks

## TODO

### Framework
- [x] Implement `SessionRegistry` with lazy `SessionMetadataFactory` integration.
- [x] Implement `CommandRegistry` with duplicate-registration checks.
- [x] Implement in-memory `HydrationStore` including cursor and defensive-copy `View` semantics.
- [x] Wire `Hub.Submit(...)` to the real dispatch path.

### Systemlab and validation
- [x] Create one minimal example handler plus UI/timeline projections for lab use.
- [x] Build Systemlab Lab 01 page with command form, trace panel, session panel, and hydration snapshot panel.
- [x] Add transcript export (JSON or markdown) for a lab run.
- [x] Add automated tests for happy path, unknown command, and projection error handling policy.
- [x] Document exactly which seams Phase 2 will replace with Watermill-backed behavior.

## Exit Criteria

- [x] Framework capability implemented behind the intended public seam.
- [x] Systemlab page exists and can demonstrate the feature.
- [x] Automated tests cover the main invariants of the phase.
- [x] At least one trace, screenshot, or transcript exists as reference material.
