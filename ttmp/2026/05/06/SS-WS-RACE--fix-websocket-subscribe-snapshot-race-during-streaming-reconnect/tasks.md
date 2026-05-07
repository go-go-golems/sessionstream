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

- [ ] Read `design-doc/01-websocket-subscribe-race-fix-guide.md` completely.
- [ ] Confirm the race with the current subscribe order: snapshot load/send happens before subscription registration.
- [ ] Confirm whether `SS-OBSERVERS` is implemented first; if yes, use transport records in race tests.
- [ ] Confirm no protobuf schema changes are needed for the first fix.
- [ ] Confirm no durable UI-event replay is required in this ticket.

## Phase 1: Subscription state model

- [ ] Replace the current `subscription` struct with a stateful struct.
- [ ] Add `subscriptionState` type.
- [ ] Add `subscriptionStateHydrating`.
- [ ] Add `subscriptionStateLive`.
- [ ] Add `snapshotOrdinal uint64` to `subscription`.
- [ ] Add per-subscription `buffer []bufferedUIBatch`.
- [ ] Add `bufferedUIBatch` with `ordinal uint64` and cloned `events []sessionstream.UIEvent`.
- [ ] Decide and document initial buffer limits.

## Phase 2: Subscription helper methods

- [ ] Add `registerHydrating(c, sid, since)` that updates both `c.subs` and `s.bySession` before snapshot loading.
- [ ] Add `subscriptionState(c, sid) (subscriptionState, bool)`.
- [ ] Add `bufferHydrationEvents(c, sid, ord, events)`.
- [ ] Ensure `bufferHydrationEvents` clones UI events before storing them.
- [ ] Add `drainHydrationBuffer(c, sid, snapshotOrdinal)`.
- [ ] Ensure `drainHydrationBuffer` filters to `ordinal > snapshotOrdinal`.
- [ ] Ensure `drainHydrationBuffer` sorts batches by ordinal before returning them.
- [ ] Add `markLive(c, sid, snapshotOrdinal)`.
- [ ] Add `sendUIBatch(c, sid, ord, events)` helper to avoid duplicating frame construction.
- [ ] Ensure none of the send helpers are called while holding `s.mu` or `c.mu`.

## Phase 3: Modify subscribe flow

- [ ] In the subscribe branch, validate session ID first.
- [ ] Call `registerHydrating` before `SnapshotProvider.Snapshot`.
- [ ] Add cleanup logic so a failed snapshot or failed send removes the hydrating subscription.
- [ ] Load snapshot.
- [ ] Queue/send snapshot frame.
- [ ] Drain buffered events with `ordinal > snapshotOrdinal`.
- [ ] Queue/send buffered UI event frames after snapshot.
- [ ] Mark subscription live.
- [ ] Queue/send subscribed frame.
- [ ] Preserve existing hook behavior unless explicitly changed and documented.

## Phase 4: Modify PublishUI

- [ ] Keep `connectionsForSession` returning a copied target slice.
- [ ] For each target, inspect that connection's subscription state for the session.
- [ ] If state is `hydrating`, buffer the batch.
- [ ] If state is `live`, send the batch directly.
- [ ] If subscription disappeared between target selection and state lookup, skip that connection.
- [ ] Return errors consistently with the existing `PublishUI` behavior.
- [ ] If buffer overflow occurs, send an error frame if possible, close the connection, and remove subscription.
- [ ] Do not silently drop buffered live events.

## Phase 5: Buffer limits and overflow behavior

- [ ] Add default constants for maximum hydrating buffered batches/events.
- [ ] Optionally add `WithHydrationBufferLimits(maxBatches, maxEvents int) Option`.
- [ ] Validate option inputs if added.
- [ ] Add overflow observation if `SS-OBSERVERS` is present.
- [ ] On overflow, close the connection rather than serving an incomplete stream.
- [ ] Document overflow behavior in code comments.

## Phase 6: Observer integration

- [ ] If `SS-OBSERVERS` has landed, observe `subscription_registered` with state `hydrating`.
- [ ] Observe `ui_event_buffered` or equivalent when hydrating fanout is buffered.
- [ ] Observe `hydration_buffer_flushed` with flushed ordinals.
- [ ] Observe `subscription_live` after mark-live.
- [ ] Observe buffer overflow.
- [ ] Add test assertions against observer records for the deterministic race test.

## Phase 7: Deterministic race tests

- [ ] Add `TestSubscribeBuffersFanoutDuringSnapshotLoad` using a snapshot provider blocked on channels.
- [ ] In the test, send subscribe and wait until snapshot loading starts.
- [ ] Publish UI event ordinal 101 while snapshot is blocked.
- [ ] Release snapshot returning ordinal 100.
- [ ] Assert client receives snapshot 100, then uiEvent 101, then subscribed.
- [ ] Add `TestSubscribeDoesNotFlushBufferedEventsAtOrBeforeSnapshotOrdinal`.
- [ ] Add `TestFanoutAfterSubscriptionLiveSendsDirectly`.
- [ ] Add `TestHydratingConnectionDoesNotBlockLiveConnection`.
- [ ] Add `TestHydrationBufferOverflowClosesConnection` if buffer limits are implemented.

## Phase 8: Regression tests for existing behavior

- [ ] Ensure `TestServerSubscribeEmptySnapshotThenLive` still passes.
- [ ] Ensure `TestServerReconnectGetsSnapshotThenNextLive` still passes or update expected ordering only if intentionally changed.
- [ ] Ensure unsubscribe still removes the connection from `bySession`.
- [ ] Ensure disconnect still removes all subscriptions and closes the send channel once.
- [ ] Ensure malformed subscribe still returns protocol error and does not leave stale hydrating state.

## Phase 9: Concurrency and race validation

- [ ] Run `go test ./pkg/sessionstream/transport/ws -count=1`.
- [ ] Run `go test ./pkg/sessionstream/transport/ws -race -count=1` if feasible.
- [ ] Run `go test ./pkg/sessionstream/... -count=1`.
- [ ] Inspect lock ordering manually: avoid calling `sendFrame`, observers, or close logic while holding `s.mu` or `c.mu` unless already safe.
- [ ] Manually review channel-close behavior so overflow/cleanup does not double-close `c.send`.

## Phase 10: Documentation and changelog

- [ ] Add comments explaining the hydrating/live subscription state machine.
- [ ] Update WebSocket transport documentation to state the reconnect guarantee: snapshot first, then buffered live events with ordinal greater than snapshot ordinal.
- [ ] Update changelog with race fix and tests.
- [ ] If Pinocchio consumes this behavior, update the PINO-STREAM-DEBUG ticket to mention the fixed subscribe semantics.
