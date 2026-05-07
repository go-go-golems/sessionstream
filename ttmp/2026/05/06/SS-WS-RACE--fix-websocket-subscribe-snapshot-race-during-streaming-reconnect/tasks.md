---
Title: Tasks
Ticket: SS-WS-RACE
Status: active
Topics:
  - sessionstream
  - websocket
  - hydration
  - reconnect
  - streaming
DocType: tasks
Intent: operational
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Detailed implementation tasks for fixing the websocket subscribe snapshot race.
LastUpdated: 2026-05-06T21:40:00-04:00
---

# Tasks

## Phase 0: Confirm behavior and dependencies

- [x] Read `design-doc/01-websocket-subscribe-race-fix-guide.md` completely.
- [x] Confirm the race with the current subscribe order: snapshot load/send happens before subscription registration.
- [x] Confirm whether `SS-OBSERVERS` is implemented first; if yes, use transport records in race tests.
- [x] Confirm no protobuf schema changes are needed for the first fix.
- [x] Confirm no durable UI-event replay is required in this ticket.

## Phase 1: Subscription state model

- [x] Replace the current `subscription` struct with a stateful struct.
- [x] Add `subscriptionState` type.
- [x] Add `subscriptionStateHydrating`.
- [x] Add `subscriptionStateLive`.
- [x] Add `snapshotOrdinal uint64` to `subscription`.
- [x] Add per-subscription `buffer []bufferedUIBatch`.
- [x] Add `bufferedUIBatch` with `ordinal uint64` and cloned `events []sessionstream.UIEvent`.
- [x] Decide and document initial buffer limits.

## Phase 2: Subscription helper methods

- [x] Add `registerHydrating(c, sid, since)` that updates both `c.subs` and `s.bySession` before snapshot loading.
- [x] Add `subscriptionState(c, sid) (subscriptionState, bool)`.
- [x] Add `bufferHydrationEvents(c, sid, ord, events)`.
- [x] Ensure `bufferHydrationEvents` clones UI events before storing them.
- [x] Add `drainHydrationBuffer(c, sid, snapshotOrdinal)`.
- [x] Ensure `drainHydrationBuffer` filters to `ordinal > snapshotOrdinal`.
- [x] Ensure `drainHydrationBuffer` sorts batches by ordinal before returning them.
- [x] Add `markLive(c, sid, snapshotOrdinal)`.
- [x] Add `sendUIBatch(c, sid, ord, events)` helper to avoid duplicating frame construction.
- [x] Ensure none of the send helpers are called while holding `s.mu` or `c.mu`.

## Phase 3: Modify subscribe flow

- [x] In the subscribe branch, validate session ID first.
- [x] Call `registerHydrating` before `SnapshotProvider.Snapshot`.
- [x] Add cleanup logic so a failed snapshot or failed send removes the hydrating subscription.
- [x] Load snapshot.
- [x] Queue/send snapshot frame.
- [x] Drain buffered events with `ordinal > snapshotOrdinal`.
- [x] Queue/send buffered UI event frames after snapshot.
- [x] Mark subscription live.
- [x] Queue/send subscribed frame.
- [x] Preserve existing hook behavior unless explicitly changed and documented.

## Phase 4: Modify PublishUI

- [x] Keep `connectionsForSession` returning a copied target slice.
- [x] For each target, inspect that connection's subscription state for the session.
- [x] If state is `hydrating`, buffer the batch.
- [x] If state is `live`, send the batch directly.
- [x] If subscription disappeared between target selection and state lookup, skip that connection.
- [x] Return errors consistently with the existing `PublishUI` behavior.
- [x] If buffer overflow occurs, send an error frame if possible, close the connection, and remove subscription.
- [x] Do not silently drop buffered live events.

## Phase 5: Buffer limits and overflow behavior

- [x] Add default constants for maximum hydrating buffered batches/events.
- [x] Optionally add `WithHydrationBufferLimits(maxBatches, maxEvents int) Option`.
- [x] Validate option inputs if added.
- [x] Add overflow observation if `SS-OBSERVERS` is present.
- [x] On overflow, close the connection rather than serving an incomplete stream.
- [x] Document overflow behavior in code comments.

## Phase 6: Observer integration

- [x] If `SS-OBSERVERS` has landed, observe `subscription_registered` with state `hydrating`.
- [x] Observe `ui_event_buffered` or equivalent when hydrating fanout is buffered.
- [x] Observe `hydration_buffer_flushed` with flushed ordinals.
- [x] Observe `subscription_live` after mark-live.
- [x] Observe buffer overflow.
- [x] Add test assertions against observer records for the deterministic race test.

## Phase 7: Deterministic race tests

- [x] Add `TestSubscribeBuffersFanoutDuringSnapshotLoad` using a snapshot provider blocked on channels.
- [x] In the test, send subscribe and wait until snapshot loading starts.
- [x] Publish UI event ordinal 101 while snapshot is blocked.
- [x] Release snapshot returning ordinal 100.
- [x] Assert client receives snapshot 100, then uiEvent 101, then subscribed.
- [x] Add `TestSubscribeDoesNotFlushBufferedEventsAtOrBeforeSnapshotOrdinal`.
- [x] Add `TestFanoutAfterSubscriptionLiveSendsDirectly`.
- [x] Add `TestHydratingConnectionDoesNotBlockLiveConnection`.
- [x] Add `TestHydrationBufferOverflowClosesConnection` if buffer limits are implemented.

## Phase 8: Regression tests for existing behavior

- [x] Ensure `TestServerSubscribeEmptySnapshotThenLive` still passes.
- [x] Ensure `TestServerReconnectGetsSnapshotThenNextLive` still passes or update expected ordering only if intentionally changed.
- [x] Ensure unsubscribe still removes the connection from `bySession`.
- [x] Ensure disconnect still removes all subscriptions and closes the send channel once.
- [x] Ensure malformed subscribe still returns protocol error and does not leave stale hydrating state.

## Phase 9: Concurrency and race validation

- [x] Run `go test ./pkg/sessionstream/transport/ws -count=1`.
- [x] Run `go test ./pkg/sessionstream/transport/ws -race -count=1` if feasible.
- [x] Run `go test ./pkg/sessionstream/... -count=1`.
- [x] Inspect lock ordering manually: avoid calling `sendFrame`, observers, or close logic while holding `s.mu` or `c.mu` unless already safe.
- [x] Manually review channel-close behavior so overflow/cleanup does not double-close `c.send`.

## Phase 10: Documentation and changelog

- [x] Add comments explaining the hydrating/live subscription state machine.
- [x] Update WebSocket transport documentation to state the reconnect guarantee: snapshot first, then buffered live events with ordinal greater than snapshot ordinal.
- [x] Update changelog with race fix and tests.
- [x] If Pinocchio consumes this behavior, update the PINO-STREAM-DEBUG ticket to mention the fixed subscribe semantics.
