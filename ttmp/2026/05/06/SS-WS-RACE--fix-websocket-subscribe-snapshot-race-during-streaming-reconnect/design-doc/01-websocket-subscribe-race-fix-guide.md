---
Title: WebSocket Subscribe Race Fix Guide
Ticket: SS-WS-RACE
Status: active
Topics:
    - sessionstream
    - websocket
    - hydration
    - reconnect
    - streaming
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sessionstream/hydration.go
      Note: Defines Snapshot semantics and snapshot ordinal used to filter buffered events.
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Contains current snapshot-before-subscribe flow and PublishUI fanout logic to fix.
    - Path: pkg/sessionstream/transport/ws/server_test.go
      Note: Existing websocket tests and target location for deterministic race regression tests.
    - Path: ttmp/2026/pkg/sessionstream/hub.go
    - Path: ttmp/2026/pkg/sessionstream/hydration.go
    - Path: ttmp/2026/pkg/sessionstream/transport/ws/server.go
    - Path: ttmp/2026/pkg/sessionstream/transport/ws/server_test.go
ExternalSources: []
Summary: Design and implementation guide for fixing the WebSocket snapshot-before-subscribe race that can drop live UI events during reconnect.
LastUpdated: 2026-05-06T21:35:00-04:00
WhatFor: Guide a new contributor through understanding and fixing the reconnect race in sessionstream's websocket transport.
WhenToUse: Use before changing websocket subscribe semantics, snapshot hydration ordering, live UI fanout buffering, or reconnect behavior.
---


# WebSocket Subscribe Race Fix Guide

## Executive summary

`sessionstream`'s WebSocket transport currently handles subscribe like this:

```text
client subscribe
  -> load current snapshot
  -> send snapshot frame
  -> register the connection as subscribed
  -> send subscribed frame
  -> future live UI events reach this connection
```

This ordering creates a race during streaming reconnects. If a live UI event is published after the snapshot is loaded but before the connection is registered, the reloaded browser tab can miss that event. The backend event is not necessarily lost. The durable timeline may not be lost. The transient WebSocket frame for the reconnecting tab is lost.

The robust fix is to close the gap between snapshot and live delivery. This document recommends a **subscribe-first hydration buffer**:

```text
client subscribe
  -> register the connection as hydrating for the session
  -> buffer live UI events for that connection while hydrating
  -> load current snapshot
  -> send snapshot frame
  -> flush buffered UI events with ordinal > snapshot ordinal
  -> mark subscription live
  -> send subscribed frame
  -> future live UI events are sent directly
```

This preserves the important client contract: a reconnecting browser receives a snapshot first, then live events that happen after that snapshot.

## Problem statement

Streaming applications emit many events while the browser is connected. During a single assistant response, the server may publish:

```text
ChatMessageStarted ordinal=100
ChatMessageAppended ordinal=101
ChatMessageAppended ordinal=102
ChatReasoningAppended ordinal=103
ChatMessageAppended ordinal=104
ChatMessageFinished ordinal=105
```

If the user reloads the browser tab during this sequence, the old WebSocket disconnects and a new WebSocket subscribes to the same session. Reconnect should feel like this:

```text
snapshot includes everything through ordinal N
then live frames continue at ordinal N+1, N+2, ...
```

The current code does not guarantee that property. It sends a snapshot before adding the connection to the live fanout map. That means live events emitted during the snapshot load/send window do not target the new connection.

## The current code path

The relevant file is `pkg/sessionstream/transport/ws/server.go`.

### Subscribe handling today

In `handleClientFrame`, the subscribe branch does roughly this:

```go
case *sessionstreamv1.ClientFrame_Subscribe:
    sid := sessionstream.SessionId(sub.GetSessionId())
    since := sub.GetSinceSnapshotOrdinal()

    snap, err := s.snapshots.Snapshot(ctx, sid)
    if err != nil { return err }

    if err := s.sendFrame(c, newSnapshotFrame(sid, snap)); err != nil {
        return err
    }

    c.mu.Lock()
    c.subs[sid] = subscription{sinceSnapshotOrdinal: since}
    c.mu.Unlock()

    s.mu.Lock()
    set := s.bySession[sid]
    if set == nil { set = map[ConnectionId]struct{}{} }
    set[c.id] = struct{}{}
    s.bySession[sid] = set
    s.mu.Unlock()

    return s.sendFrame(c, newSubscribedFrame(sid, since))
```

### Live fanout today

`PublishUI` does this:

```go
func (s *Server) PublishUI(ctx context.Context, sid SessionId, ord uint64, events []UIEvent) error {
    targets := s.connectionsForSession(sid)
    for _, c := range targets {
        for _, event := range events {
            frame := newUIEventFrame(sid, ord, event)
            s.sendFrame(c, frame)
        }
    }
    return nil
}
```

The target list comes from `s.bySession[sid]`. A connection is invisible to live fanout until subscribe registration has completed.

## Race timeline

Here is the bad interleaving.

```text
Time   Actor            Action
----   -----            ------
T1     browser          reloads while assistant response is streaming
T2     old websocket    disconnects
T3     new websocket    connects
T4     browser          sends subscribe(session=S, since=0)
T5     server           starts Snapshot(S)
T6     store            returns snapshot ordinal=100
T7     backend/hub      publishes UI event ordinal=101
T8     websocket        PublishUI(session=S, ordinal=101) asks bySession[S]
T9     websocket        new connection is not registered yet, targets=[]
T10    server           sends snapshot ordinal=100 to new connection
T11    server           registers new connection in bySession[S]
T12    backend/hub      publishes UI event ordinal=102
T13    websocket        sends ordinal=102 to new connection
```

The browser receives:

```text
snapshot ordinal=100
uiEvent ordinal=102
uiEvent ordinal=103
...
```

The browser misses:

```text
uiEvent ordinal=101
```

The event may still exist in the backend event store, and the timeline projection may still have applied durable state. But the browser state after reconnect can diverge from the state of a browser that never reloaded.

## Why `sinceSnapshotOrdinal` does not currently fix it

The subscribe request includes `sinceSnapshotOrdinal`. The WebSocket server stores and echoes it, but the current code comments state that it is advisory:

```go
// sinceSnapshotOrdinal is accepted, stored, echoed, and surfaced to hooks for teaching
// and diagnostics, but it is advisory for now: this reference adapter does not
// replay missed UI events from the event store.
```

That means the server does not replay UI events after the snapshot ordinal. Without replay, the only way to avoid missed live events is to buffer them while the connection is hydrating.

## Design goals

The fix should satisfy these requirements:

- A reconnecting client should receive a snapshot before live UI event frames for that subscription.
- Live UI events emitted while a subscription is hydrating should not be dropped for that connection.
- Events with ordinals less than or equal to the snapshot ordinal should not be replayed after the snapshot.
- The fix should not require durable UI event storage.
- The fix should not change `HydrationStore`, `UIFanout`, or protobuf schemas unless absolutely necessary.
- The fix should preserve existing behavior for already-live subscribers.
- The fix should be observable through the new WebSocket transport observer from `SS-OBSERVERS`.

## Proposed solution: subscribe-first hydration buffer

A subscription should have a state machine.

```text
absent
  -> hydrating
  -> live
  -> absent
```

### Meaning of each state

| State | Meaning | Fanout behavior |
|-------|---------|-----------------|
| `absent` | The connection is not subscribed to this session. | Do not target the connection. |
| `hydrating` | The connection has requested the session and is loading/sending snapshot. | Buffer live UI events for this connection/session. |
| `live` | Snapshot has been sent and buffered events have been flushed. | Send live UI events directly. |

### New subscription struct

Replace the current subscription struct:

```go
type subscription struct {
    sinceSnapshotOrdinal uint64
}
```

with:

```go
type subscriptionState string

const (
    subscriptionStateHydrating subscriptionState = "hydrating"
    subscriptionStateLive      subscriptionState = "live"
)

type bufferedUIBatch struct {
    ordinal uint64
    events  []sessionstream.UIEvent
}

type subscription struct {
    sinceSnapshotOrdinal uint64
    state                subscriptionState
    snapshotOrdinal      uint64
    buffer               []bufferedUIBatch
}
```

The buffer is per connection per session. It is not global. That matters because one tab can be hydrating while another tab is already live.

## New subscribe flow

The new subscribe algorithm is:

```text
1. Validate session id.
2. Register the connection as hydrating for this session.
3. Load the snapshot.
4. Send/queue the snapshot frame.
5. Drain buffered UI events whose ordinal is greater than the snapshot ordinal.
6. Mark the subscription live.
7. Send/queue subscribed frame.
```

Pseudocode:

```go
func (s *Server) handleSubscribe(ctx context.Context, c *connection, sub *SubscribeRequest) error {
    sid := sessionstream.SessionId(sub.GetSessionId())
    since := sub.GetSinceSnapshotOrdinal()
    if sid == "" {
        return fmt.Errorf("subscribe missing session id")
    }

    s.registerHydrating(c, sid, since)
    defer func() {
        // If the function exits before markLive, clean up hydrating state.
        if !s.isLive(c, sid) {
            s.removeSubscription(c, sid)
        }
    }()

    snap, err := s.snapshots.Snapshot(ctx, sid)
    if err != nil {
        return fmt.Errorf("load snapshot for %q: %w", sid, err)
    }

    if err := s.sendFrame(c, newSnapshotFrame(sid, snap)); err != nil {
        return err
    }

    buffered := s.drainHydrationBuffer(c, sid, snap.SnapshotOrdinal)
    for _, batch := range buffered {
        for _, event := range batch.events {
            if err := s.sendUIEventFrame(c, sid, batch.ordinal, event); err != nil {
                return err
            }
        }
    }

    s.markLive(c, sid, snap.SnapshotOrdinal)

    return s.sendFrame(c, newSubscribedFrame(sid, since))
}
```

The exact ordering of `markLive` relative to flushing and `subscribed` should be chosen carefully. This document recommends:

```text
snapshot queued
buffered UI queued
mark live
subscribed queued
```

This means future fanout after `markLive` sends directly. If the `subscribed` frame is used by clients as a readiness signal, consider queuing it before marking live, but then the direct-live window remains smaller but still worth reasoning about. Tests should lock this down.

## Register hydrating helper

```go
func (s *Server) registerHydrating(c *connection, sid SessionId, since uint64) {
    c.mu.Lock()
    c.subs[sid] = subscription{
        sinceSnapshotOrdinal: since,
        state: subscriptionStateHydrating,
        buffer: nil,
    }
    c.mu.Unlock()

    s.mu.Lock()
    set := s.bySession[sid]
    if set == nil {
        set = map[ConnectionId]struct{}{}
        s.bySession[sid] = set
    }
    set[c.id] = struct{}{}
    s.mu.Unlock()
}
```

The key is that the connection is now in `bySession` before snapshot loading starts. `PublishUI` will see it as a target, but will buffer rather than send directly because its subscription state is hydrating.

## PublishUI with hydrating subscriptions

`PublishUI` should target live and hydrating connections differently.

```go
func (s *Server) PublishUI(ctx context.Context, sid SessionId, ord uint64, events []UIEvent) error {
    targets := s.connectionsForSession(sid)
    for _, c := range targets {
        state := s.subscriptionState(c, sid)
        switch state {
        case subscriptionStateHydrating:
            s.bufferHydrationEvents(c, sid, ord, events)
        case subscriptionStateLive:
            s.sendUIBatch(c, sid, ord, events)
        default:
            // Connection disappeared between target selection and state lookup.
        }
    }
    return nil
}
```

### Buffer clone rule

Always clone events before buffering:

```go
func (s *Server) bufferHydrationEvents(c *connection, sid SessionId, ord uint64, events []UIEvent) {
    c.mu.Lock()
    defer c.mu.Unlock()

    sub, ok := c.subs[sid]
    if !ok || sub.state != subscriptionStateHydrating {
        return
    }
    sub.buffer = append(sub.buffer, bufferedUIBatch{
        ordinal: ord,
        events: cloneUIEvents(events),
    })
    c.subs[sid] = sub
}
```

The buffer should store a batch because one ordinal may produce multiple UI events.

## Draining the hydration buffer

After snapshot has been queued, flush buffered batches with `ordinal > snap.SnapshotOrdinal`.

```go
func (s *Server) drainHydrationBuffer(c *connection, sid SessionId, snapshotOrdinal uint64) []bufferedUIBatch {
    c.mu.Lock()
    defer c.mu.Unlock()

    sub, ok := c.subs[sid]
    if !ok {
        return nil
    }

    out := make([]bufferedUIBatch, 0, len(sub.buffer))
    for _, batch := range sub.buffer {
        if batch.ordinal > snapshotOrdinal {
            out = append(out, cloneBufferedBatch(batch))
        }
    }
    sort.SliceStable(out, func(i, j int) bool {
        return out[i].ordinal < out[j].ordinal
    })

    sub.buffer = nil
    sub.snapshotOrdinal = snapshotOrdinal
    c.subs[sid] = sub
    return out
}
```

Filtering by snapshot ordinal prevents duplicate delivery. If an event arrived after the connection entered hydrating but before the snapshot was read, the snapshot may already include its timeline state. Sending a live UI event for that same ordinal after the snapshot could duplicate a message or status update.

## Marking live

```go
func (s *Server) markLive(c *connection, sid SessionId, snapshotOrdinal uint64) {
    c.mu.Lock()
    defer c.mu.Unlock()

    sub, ok := c.subs[sid]
    if !ok {
        return
    }
    sub.state = subscriptionStateLive
    sub.snapshotOrdinal = snapshotOrdinal
    sub.buffer = nil
    c.subs[sid] = sub
}
```

## Buffer bounds and failure behavior

A hydrating subscription should not buffer unbounded data. Add constants:

```go
const (
    defaultMaxHydrationBufferedBatches = 1024
    defaultMaxHydrationBufferedEvents  = 4096
)
```

Expose options only if needed:

```go
func WithHydrationBufferLimits(maxBatches, maxEvents int) Option
```

For the first implementation, a simple max batch count may be enough. If the buffer overflows, close the connection with a protocol error or send an error frame and remove the subscription. Do not silently drop buffered events, because that reintroduces the bug.

Recommended behavior:

```text
buffer overflow
  -> observe hydration_buffer_overflow
  -> queue error frame if possible
  -> close connection
  -> remove hydrating subscription
```

This is strict, but streaming correctness is better served by forcing the client to reconnect than by serving an incomplete stream.

## Observability requirements

This race is hard to prove without structured observations. The `SS-OBSERVERS` ticket should add WebSocket records for:

```text
subscribe_received
subscription_registered(state=hydrating)
snapshot_load_started
snapshot_loaded(snapshotOrdinal=N)
fanout_started(ordinal=N+1, target=[conn])
ui_event_buffered(ordinal=N+1, conn=...)
snapshot_queued(snapshotOrdinal=N)
buffer_flushed(ordinal=N+1)
subscription_live
subscribed_queued
```

The expected fixed trace looks like this:

```text
T1 subscribe_received conn=C sid=S
T2 subscription_registered conn=C sid=S state=hydrating
T3 snapshot_load_started conn=C sid=S
T4 fanout_started sid=S ordinal=101 targets=[C]
T5 ui_event_buffered conn=C sid=S ordinal=101
T6 snapshot_loaded conn=C sid=S snapshotOrdinal=100
T7 server_frame_queued conn=C frame=snapshot ordinal=100
T8 hydration_buffer_flushed conn=C sid=S ordinals=[101]
T9 server_frame_queued conn=C frame=uiEvent ordinal=101
T10 subscription_live conn=C sid=S
T11 server_frame_queued conn=C frame=subscribed
```

The unfixed trace looks like this:

```text
T1 subscribe_received conn=C sid=S
T2 snapshot_load_started conn=C sid=S
T3 snapshot_loaded conn=C sid=S snapshotOrdinal=100
T4 fanout_started sid=S ordinal=101 targets=[]
T5 server_frame_queued conn=C frame=snapshot ordinal=100
T6 subscription_registered conn=C sid=S
```

A test should be able to assert the fixed trace order.

## Test strategy

### Unit-level deterministic race test

Create a test in `pkg/sessionstream/transport/ws/server_test.go` that controls snapshot loading with channels.

Pseudocode:

```go
func TestSubscribeBuffersFanoutDuringSnapshotLoad(t *testing.T) {
    snapshotStarted := make(chan struct{})
    releaseSnapshot := make(chan struct{})

    provider := SnapshotProviderFunc(func(ctx context.Context, sid SessionId) (Snapshot, error) {
        close(snapshotStarted)
        <-releaseSnapshot
        return Snapshot{SessionId: sid, SnapshotOrdinal: 100, Entities: []TimelineEntity{...}}, nil
    })

    wsServer := NewServer(provider)
    conn := connectWebSocket(wsServer)

    writeSubscribe(conn, "s-1", 0)
    <-snapshotStarted

    // This event happens while subscribe is hydrating.
    require.NoError(t, wsServer.PublishUI(context.Background(), "s-1", 101, []UIEvent{...}))

    close(releaseSnapshot)

    snapshotFrame := readServerFrame(conn)
    require.Equal(t, uint64(100), snapshotFrame.GetSnapshot().GetSnapshotOrdinal())

    uiFrame := readServerFrame(conn)
    require.Equal(t, uint64(101), uiFrame.GetUiEvent().GetEventOrdinal())

    subscribed := readServerFrame(conn)
    require.NotNil(t, subscribed.GetSubscribed())
}
```

This test should fail on the old implementation because `PublishUI` sees no targets during snapshot load.

### Duplicate prevention test

Create a test where the buffered event ordinal is less than or equal to the snapshot ordinal.

```go
func TestSubscribeDoesNotFlushBufferedEventsAtOrBeforeSnapshotOrdinal(t *testing.T) {
    // Register hydrating.
    // Publish ordinal 100 while snapshot is loading.
    // Snapshot returns ordinal 100.
    // Assert no uiEvent ordinal 100 is sent after snapshot.
}
```

This proves the `ordinal > snapshotOrdinal` filter.

### Live-after-mark test

```go
func TestFanoutAfterSubscriptionLiveSendsDirectly(t *testing.T) {
    // Complete subscribe.
    // Publish ordinal 102.
    // Assert uiEvent 102 is sent normally and not buffered.
}
```

### Buffer overflow test

```go
func TestHydrationBufferOverflowClosesConnection(t *testing.T) {
    // Configure low max buffer size.
    // Hold snapshot load.
    // Publish more batches than limit.
    // Assert error observation and disconnect.
}
```

### Multiple connections test

```go
func TestHydratingConnectionDoesNotBlockLiveConnection(t *testing.T) {
    // Tab A is live.
    // Tab B subscribes and snapshot blocks.
    // Publish ordinal 101.
    // Assert Tab A receives immediately.
    // Assert Tab B receives after snapshot.
}
```

This is crucial. The buffer is per hydrating connection, not per session globally.

## Locking and concurrency guidance

The WebSocket server currently uses:

- `s.mu` for `s.conns` and `s.bySession`.
- `c.mu` for `c.subs`.
- `c.send` channel for outgoing bytes.

Avoid holding locks while sending frames. `sendFrame` can block briefly on channel operations and can call observers. Holding `c.mu` or `s.mu` while sending risks deadlocks and latency spikes.

Recommended pattern:

```go
// Under lock: copy state and buffered batches.
buffered := s.drainHydrationBuffer(...)

// Outside lock: send frames.
for _, batch := range buffered {
    sendUIBatch(...)
}

// Under lock: mark live.
s.markLive(...)
```

For `PublishUI`, do not hold `s.mu` while iterating sends. The existing `connectionsForSession` helper already returns a copied target slice. Keep that pattern.

## Alternative approaches considered

### Alternative A: Replay missed UI events from the event store

This would make `sinceSnapshotOrdinal` a real cursor. The server would load a snapshot at ordinal N, then replay UI projection for backend events after N.

Pros:

- Strong reconnect semantics.
- Could support longer disconnects, not just the subscribe window.

Cons:

- Requires durable UI event storage or deterministic replay of UI projections.
- UI projections may depend on a timeline view at the time of each event, which is not trivial to reconstruct unless the projection model is designed for replay.
- Larger change than needed to fix the immediate race.

Decision: defer. Buffering fixes the local race without changing storage semantics.

### Alternative B: Register subscription before snapshot and send live events immediately

This would avoid lost events but can violate ordering. The browser might receive `uiEvent 101` before `snapshot 100`, then the snapshot clears/replaces Redux state and erases the live event.

Decision: reject. Snapshot must be first for hydration correctness.

### Alternative C: Rely on another reload

This is the current practical workaround: if a tab misses a live event, a later reload gets a new snapshot that may include durable state.

Decision: reject for robust streaming UX. The system should not require reloads to repair reconnects.

## API impact

The recommended fix can be implemented without changing the public protobuf protocol or core `sessionstream.UIFanout` interface. It changes only internal WebSocket server subscription state and optionally adds server options for buffer limits.

Potential additive option:

```go
func WithHydrationBufferLimits(maxBatches int) Option
```

This is not required for the first pass if constants are acceptable.

## File reference

| File | Symbols to inspect/change |
|------|---------------------------|
| `pkg/sessionstream/transport/ws/server.go` | `subscription`, `handleClientFrame`, `PublishUI`, `connectionsForSession`, `removeSubscription`, `sendFrame`, `writeLoop`. |
| `pkg/sessionstream/transport/ws/server_test.go` | Existing subscribe/reconnect tests; add race tests here. |
| `pkg/sessionstream/hydration.go` | `Snapshot`, `HydrationStore`; no interface changes expected. |
| `pkg/sessionstream/projection.go` | `UIEvent`; use `cloneUIEvent`/`cloneUIEvents` when buffering. |

## Implementation tasks

1. Extend subscription state with `hydrating`, `live`, `snapshotOrdinal`, and a per-subscription buffer.
2. Add helper methods: `registerHydrating`, `subscriptionState`, `bufferHydrationEvents`, `drainHydrationBuffer`, `markLive`, and `sendUIBatch`.
3. Modify subscribe handling to register hydrating before snapshot load.
4. Modify `PublishUI` to buffer for hydrating targets and send directly for live targets.
5. Filter buffered batches by `ordinal > snapshotOrdinal` before flush.
6. Add buffer limits and strict overflow behavior.
7. Add observer records if `SS-OBSERVERS` has landed.
8. Add deterministic tests for the race, duplicate prevention, multiple connections, live-after-mark, and overflow.
9. Run `go test ./pkg/sessionstream/transport/ws -count=1` and full `go test ./pkg/sessionstream/... -count=1`.

## Review checklist

A reviewer should verify:

- The connection enters `bySession` before snapshot load starts.
- Hydrating subscriptions buffer events rather than sending them before snapshot.
- Buffered ordinals less than or equal to snapshot ordinal are discarded.
- Buffered ordinals greater than snapshot ordinal are sent after snapshot.
- Already-live connections are not delayed by another connection's hydration.
- No lock is held while writing or queueing websocket frames.
- Buffer overflow fails loudly instead of silently dropping events.
- Existing tests for subscribe, reconnect, unsubscribe, and live fanout still pass.

## Closing guidance for the intern

Do not think of this as a generic WebSocket problem. It is specifically a **handoff problem** between two sources of truth: the snapshot and the live fanout stream. The fix is correct only if the browser sees a single ordered story: durable state up to N, then live events after N. Any implementation that can deliver live event N+1 before snapshot N, or skip N+1 entirely, is still broken.
