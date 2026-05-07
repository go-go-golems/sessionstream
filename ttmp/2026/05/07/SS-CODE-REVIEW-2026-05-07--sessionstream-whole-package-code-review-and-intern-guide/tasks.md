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

- [x] Add a deterministic websocket test for late hydration-buffer batches and filter late batches by `ordinal > snapshotOrdinal`.
- [x] Decide websocket fanout failure policy: return aggregate delivery errors to Hub.
- [x] Replace destructive SQLite migrations with additive migrations and preservation tests.
- [x] Change SQLite `AppendEvent` to allow only idempotent identical duplicates and reject conflicting duplicates.
- [x] Make `sqlite.NewInMemory` isolated by default.
- [x] Make `ErrorObserver` panic-safe, clone error records before delivery, and surface error persistence failures to observers.
- [ ] Split `transport/ws/server.go`, `hydration/sqlite/store.go`, `pkg/sessionstream/hub.go`, and optionally `examples/chatdemo/chat.go` after correctness fixes.
- [ ] Add focused coverage tests for websocket ping/unsubscribe/helper branches, store view helpers, and noop-store behavior.
- [ ] Update README historical `evtstream` link context after this review is accepted.
