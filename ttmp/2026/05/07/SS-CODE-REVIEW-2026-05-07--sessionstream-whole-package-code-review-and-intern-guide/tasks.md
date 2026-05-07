# Tasks

## Completed

- [x] Inventory sessionstream source, docs, tests, and recent diaries.
- [x] Map runtime architecture and public APIs for intern onboarding.
- [x] Inspect code quality hot spots, deprecated code, package boundaries, and overengineering.
- [x] Write detailed analysis/design/implementation guide with diagrams and pseudocode.
- [x] Validate codebase with tests, websocket race test, lint, and coverage artifact.
- [x] Validate docmgr ticket with doctor.
- [x] Upload ticket bundle to reMarkable and verify remote listing.

## Recommended follow-up implementation tasks

- [ ] Add a deterministic websocket test for late hydration-buffer batches and filter late batches by `ordinal > snapshotOrdinal`.
- [ ] Decide websocket fanout failure policy: close-and-continue with explicit docs, or return aggregate errors to Hub.
- [ ] Replace destructive SQLite migrations with additive versioned migrations and preservation tests.
- [ ] Change SQLite `AppendEvent` to allow only idempotent identical duplicates and reject conflicting duplicates.
- [ ] Make `sqlite.NewInMemory` isolated by default; add explicit shared-memory constructor if needed.
- [ ] Make `ErrorObserver` panic-safe and clone error records before delivery.
- [ ] Split `transport/ws/server.go`, `hydration/sqlite/store.go`, `pkg/sessionstream/hub.go`, and optionally `examples/chatdemo/chat.go` after correctness fixes.
- [ ] Add focused coverage tests for websocket ping/unsubscribe/helper branches, store view helpers, and noop-store behavior.
- [ ] Update README historical `evtstream` link context after this review is accepted.
