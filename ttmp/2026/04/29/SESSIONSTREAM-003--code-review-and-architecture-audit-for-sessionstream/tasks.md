# Tasks

## DONE

- [x] Create SESSIONSTREAM-003 ticket workspace.
- [x] Add primary architecture/code-review design document.
- [x] Add chronological investigation diary.
- [x] Inventory Go files and line-count hotspots.
- [x] Review core hub, schema, bus, consumer, ordinal, projection, and hydration APIs.
- [x] Review memory and SQLite store implementations.
- [x] Review websocket transport semantics.
- [x] Review chat demo and systemlab phase maintainability.
- [x] Run tests and coverage to support review findings.
- [x] Add remediation plan for replay store and API cleanup direction.
- [x] Start Phase 1 implementation: rename runtime `evtstream` names outside `ttmp`.
- [x] Start Phase 1 implementation: remove orphan generic command transport abstraction for now.
- [x] Start Phase 1 implementation: document websocket adapter as fanout/subscription only.
- [x] Start Phase 1 implementation: remove `Command.ConnectionId` while command ingress is out of scope.
- [x] Start Phase 2 implementation: seed local ordinals from store cursor.
- [x] Start Phase 3 implementation: make fail policy the default, split UI/timeline projection policy options, and add error observer hook.
- [x] Start Phase 5 implementation: clone schema prototypes on register and lookup.

## Phase 1 — Remove misleading names and API surface

- [x] Rename runtime bus topic and metadata defaults from `evtstream_*` to `sessionstream_*`.
- [x] Rename SQLite tables and indexes from `evtstream_*` to `sessionstream_*`.
- [x] Remove or quarantine unused `pkg/sessionstream/transport/transport.go` command-ingress abstraction.
- [x] Document websocket server as snapshot/fanout-only.
- [x] Remove `Command.ConnectionId` while websocket command ingress is not implemented.
- [ ] Update systemlab chapter prose if it still implies command ingress through websocket.
- [ ] Decide whether `ConnectionId` should remain in core package or move under websocket/fanout transport.

## Phase 2 — Replay store and ordinal correctness

- [x] Seed local ordinals from `HydrationStore.Cursor` to avoid restart regressions.
- [x] Add a unit test for local ordinal seeding from an existing cursor.
- [x] Add initial `EventStore` interface for event append/query/cursor semantics.
- [x] Implement SQLite event log table `sessionstream_events`.
- [x] Implement projection cursor table `sessionstream_projection_cursors`.
- [x] Implement timeline entity versioning and `Snapshot(asOf)` for SQLite.
- [x] Add `Events(ctx, sid, after, limit)` query API.
- [x] Add `NewInMemoryStore(reg)` backed by named in-memory SQLite.
- [x] Port local users to in-memory SQLite and remove the map-backed memory store.

## Phase 3 — Projection policies, errors, and DLQ

- [x] Make fail-closed projection behavior the default.
- [x] Split UI and timeline projection policy options.
- [x] Add `ErrorObserver` and `ErrorRecord` hook for projection/fanout errors.
- [x] Add tests for split UI/timeline policy behavior and observer records.
- [x] Add durable `sessionstream_errors` table.
- [x] Record projection, fanout, and store errors into durable error store when the store supports it.
- [x] Record decode and ordinal errors into durable error store.
- [x] Persist raw error payloads and metadata in SQLite error records.
- [ ] Add configurable bus decode nack/ack policy instead of hard-coded record-and-ack behavior.
- [x] Add initial `Hub.RebuildTimeline` helper that reprojects stored events without UI fanout.
- [x] Add retry-from-projection-cursor helper (`Hub.RetryTimeline`).
- [x] Add scratch rebuild helper (`Hub.RebuildTimelineFromScratch`) for stores that can clear materialized timeline state.
- [x] Add systemlab replay inspection endpoint for cursors and persisted errors.
- [x] Add systemlab UI controls/visualization for replay cursors and persisted errors.

## Phase 4 — Websocket fanout-only semantics

- [x] Make websocket comments explicit: subscribe, snapshot, live UI fanout only.
- [ ] Rename `sinceOrdinal` or document it as advisory until replay is implemented.
- [ ] Add a test that command frames are rejected/ignored as unsupported.
- [ ] Add production-readiness options or document reference-adapter limits.
- [ ] Decide whether replayed UI events should ever be part of websocket subscribe.

## Phase 5 — Schema hardening and protobuf example

- [x] Clone schema prototypes on registration.
- [x] Return cloned schema prototypes from lookup methods.
- [x] Add tests proving registration and lookup are defensive.
- [ ] Add descriptor-oriented schema lookup methods if needed.
- [x] Add `examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto`.
- [x] Add code generation wiring for the chat demo protobuf namespace.
- [x] Update chat demo to use generated protobuf messages instead of only `structpb.Struct`.
- [x] Update chat demo tests to assert generated payload types.

## Phase 6 — Systemlab refactor

- [x] Extract initial shared trace append/clone helpers.
- [x] Extract shared snapshot encoding helpers.
- [x] Extract shared websocket hook builder.
- [ ] Split Phase 2 into runtime/actions/projections/checks/render files.
- [ ] Split Phase 5 into runtime/actions/projections/checks/render files.
- [x] Add Phase 5 chapter prose for replay inspection, event cursor, and timeline cursor.
- [ ] Update chapter file references after future file moves.
- [ ] Keep educational phase logic readable and avoid over-abstracting lesson-specific behavior.

## Phase 7 — Validation and delivery

- [x] Run `go test ./...` after initial implementation work.
- [x] Run `make check` after the next implementation slice.
- [ ] Run `rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'` before finalizing the cleanup.
- [ ] Upload refreshed ticket bundle to reMarkable after documenting this implementation slice.
- [ ] Split remaining open work into implementation tickets if this ticket becomes too large.
