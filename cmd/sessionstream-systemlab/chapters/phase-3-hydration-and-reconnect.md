# Phase 3 — Hydration and Reconnect

## What this chapter is about

Phase 2 showed you how the consumer assigns ordinals and how the framework establishes consumed order. Phase 1 showed you how commands become events and how projections derive state. But neither phase answered a practical question: what happens when a client connects, leaves, and comes back?

This chapter answers that question. By the end, you should understand why snapshot-before-live is non-negotiable, how ConnectionId and SessionId work together, and what the framework guarantees when a client reconnects.

---

## 1. Why reconnect is a framework problem

Many systems treat reconnect as a frontend concern. The browser dropped, so the frontend reconnects. The socket reopened, so the UI asks for state.

That is not wrong, but it is incomplete.

Reconnect is a framework problem because the framework owns:

- current durable state (the hydration store)
- live event delivery (the UIFanout)
- session routing (by SessionId)
- transport identity (by ConnectionId)

If the framework has no coherent answer to reconnect, the frontend can reconnect all it wants and still end up with duplicated updates, missing updates, or inconsistent state.

Phase 3 comes after hydration and ordering because reconnect only makes sense when the system already knows what state is current and how event order is defined.

---

## 2. The central rule: snapshot before live

> A reconnecting client should receive a coherent snapshot first, and only then continue with live UI events.

This is the most important sentence in Phase 3.

Consider what happens if this rule is violated. A client reconnects. It receives live events. Then it receives a snapshot. The snapshot contradicts the live events. The client must reconcile them, or it ends up wrong.

Now consider what happens when the rule holds. A client reconnects. It receives a snapshot. That snapshot represents the current state. Then it receives live events that continue from that state. No reconciliation needed. The client just continues.

The framework prevents an entire category of bugs by sequencing correctly.

---

## 3. What a subscribe looks like

Here is the sequence when a client subscribes to a session:

```text
Client connects
    -> receives ConnectionId
    ->
Client subscribes to SessionId
    ->
Framework loads snapshot from HydrationStore
    ->
Framework sends SnapshotFrame first (with snapshotOrdinal and entities)
    ->
Framework sends live UiEventFrame messages after snapshot
```

The snapshot arrives before any live events. That is the rule.

---

## 4. ConnectionId vs SessionId

Phase 0 introduced these concepts. Phase 3 makes them operational.

**SessionId** is the business-level routing key. It identifies the unit of work. All events for a session share the same SessionId. The hydration store tracks state per SessionId.

**ConnectionId** is the transport-level identity. It identifies one socket connection. One session can have multiple connections over time (reconnect), or multiple simultaneous connections (multiple tabs watching the same session).

```text
SessionId: "session-abc"     <- business identity, stable
ConnectionId: "conn-123"     <- transport identity, changes on reconnect
ConnectionId: "conn-456"     <- different connection, same session
```

This distinction matters because:

- A client disconnects and reconnects. The SessionId stays the same. The ConnectionId changes.
- Multiple tabs can watch the same session. Each has its own ConnectionId.
- The framework tracks subscriptions by SessionId. It tracks presence by ConnectionId.

---

## 5. The transport architecture

The websocket transport sits downstream of the consumer:

```text
Commands
   |
   v
Handlers
   |
   v
Canonical backend events
   |
   v
Consumer
   |------------------> TimelineProjection -> HydrationStore
   |
   +------------------> UIProjection -> UIFanout -> Websocket transport
                                                   |
                                                   v
                                              Client connections
```

The transport is not the source of truth. It is the live-delivery mechanism for one derived view of the truth.

The transport should:
- accept connections and assign ConnectionIds
- track subscriptions by SessionId
- deliver snapshots on subscribe
- deliver live UI events after snapshots
- stay unaware of application semantics

The transport should not:
- invent application semantics
- assign ordinals
- interpret command meanings
- accept command frames
- become the place where business logic lives

The websocket protocol accepts protobuf JSON `ClientFrame` messages with `subscribe`, `unsubscribe`, `ping`, and `pong` oneof arms. Command ingress is intentionally not implemented in this adapter. Submit commands through the backend API or runtime boundary, then let the normal command handler → event → projection path produce UI fanout.

`subscribe` includes a `sinceSnapshotOrdinal` field for teaching and diagnostics. Today it is advisory: the server parses, stores, echoes, and traces it, but subscribe still means "send the current snapshot, then future live UI events." It does not replay missed UI events from the event log. If replay is needed, use an explicit replay API so recovery behavior remains visible and testable.

---

## 6. Why this matters for correctness

Here is what the framework guarantees when a client subscribes:

1. The snapshot reflects the current hydration store state.
2. The snapshot arrives before any live events.
3. The snapshot carries `snapshotOrdinal`; each entity carries `createdOrdinal` and `lastEventOrdinal`.
4. Live events continue after the snapshot and carry `eventOrdinal`.
5. Events arrive in ordinal order.

Here is what the framework does not guarantee:

- That multiple connections see identical delivery timing.
- That websocket subscribe replays every missed UI event.
- That the client never misses an event (the client must handle that).

The framework establishes the correct sequence. The transport delivers it as protobuf JSON `ServerFrame` messages. The client handles delivery confirmation and `Any` payload unpacking with its application schema registry. The reference websocket adapter is intentionally small: production deployments should add authentication, authorization, strict origin checks, rate limiting, and operational backpressure policy around it.

---

## 7. What the protobuf frames look like

A subscribe request is a `ClientFrame` with a `subscribe` arm:

```json
{
  "subscribe": {
    "sessionId": "session-abc",
    "sinceSnapshotOrdinal": "42"
  }
}
```

The server responds with a `SnapshotFrame` before it confirms the subscription:

```json
{
  "snapshot": {
    "sessionId": "session-abc",
    "snapshotOrdinal": "45",
    "entities": [
      {
        "kind": "Message",
        "id": "message-1",
        "createdOrdinal": "40",
        "lastEventOrdinal": "45",
        "payload": {
          "@type": "type.googleapis.com/example.Message",
          "text": "hello"
        }
      }
    ]
  }
}
```

Then future live updates arrive as `UiEventFrame` messages:

```json
{
  "uiEvent": {
    "sessionId": "session-abc",
    "eventOrdinal": "46",
    "name": "MessageAppended",
    "payload": {
      "@type": "type.googleapis.com/example.MessageDelta",
      "text": " world"
    }
  }
}
```

The important details are the field names and the `Any` payload shape. JavaScript clients should decode with generated protobuf schemas and a registry, not by assuming a hand-written JSON envelope.

---

## 8. The Phase 3 page

The Phase 3 page simulates two clients to make reconnect semantics visible.

**Client A** and **Client B** each have their own connection lifecycle. They can subscribe to the same session or different sessions. They can disconnect and reconnect independently.

This forces you to think about:
- independent connection lifecycle
- shared session state
- different subscribe timings
- reconvergence toward the same final session view

---

## 9. Things to try

**Connect Client A, subscribe to a session.** The client connects. A snapshot arrives. Live events continue from there.

**Generate activity while Client A is connected.** Client A receives live events as they happen.

**Disconnect Client A.** The connection closes. The framework stops sending to that ConnectionId.

**Reconnect Client A.** The client reconnects. It subscribes. It receives a snapshot of current state. Then it receives live events. Notice: the live events continue naturally from where the snapshot left off.

**Connect Client B to the same session while activity is ongoing.** Client B subscribes. It receives a snapshot of current state. Then it receives live events. Notice: both clients converge to the same final session view.

**Disconnect Client A, keep Client B connected, generate more activity.** Client A misses the activity. Client B receives it.

**Reconnect Client A.** Client A receives a new snapshot (current state). Then it receives live events. Notice: Client A and Client B are back in sync.

---

## 10. What the checks prove

| Check | What it proves |
|-------|----------------|
| `snapshotBeforeLive` | The snapshot arrived before any live events |
| `convergence` | Multiple clients converged to the same session state |
| `connectionIsolation` | Connection lifecycle does not affect session state |
| `ordinalOrder` | Events arrived in correct ordinal order |

---

## 11. Common mistakes

**Mistake: live before snapshot.** A client receives live events before it has a coherent base state. The framework must sequence snapshot before live.

**Mistake: transport owning business semantics.** If websocket code interprets application event meanings, the framework boundary is polluted. The transport should only deliver what the UIProjection produces and pack typed payloads into `google.protobuf.Any`.

**Mistake: one connection equals one session.** One session can have multiple connections over time (reconnect) or simultaneously (multiple tabs). The framework must handle this.

---

## Key Points

- Reconnect is a framework problem, not just a frontend concern. The framework owns the relationship between durable state and live delivery.
- Snapshot before live is non-negotiable. The framework must establish the correct state before delivering live events.
- SessionId is the business routing key. ConnectionId is the transport identity.
- One session can have multiple connections over time (reconnect) or simultaneously (multiple tabs).
- The transport sits downstream of the consumer. It delivers protobuf `ServerFrame` messages; it does not interpret application payload semantics.
- Clients with different live histories should converge to the same session truth.

---

## API Reference

- **`Subscribe(sessionId, connectionId, sinceSnapshotOrdinal)`**: Subscribe a connection to a session and receive a snapshot first.
- **`Unsubscribe(sessionId, connectionId)`**: Remove a subscription.
- **`DeliverSnapshot(connectionId, snapshot)`**: Deliver the current state as a protobuf `SnapshotFrame`.
- **`DeliverEvent(connectionId, event)`**: Deliver a live UI event as a protobuf `UiEventFrame`.
- **`HydrationStore.Snapshot(sessionId, asOf)`**: Load current or as-of state for a session.

---

## File References

### Framework files

- `proto/sessionstream/v1/transport.proto` — websocket transport frame schema
- `pkg/sessionstream/transport/ws/server.go` — protobuf JSON websocket adapter
- `pkg/sessionstream/fanout.go` — UI event fanout
- `pkg/sessionstream/hydration.go` — hydration store interface
- `pkg/sessionstream/hydration/sqlite/store.go` — SQLite-backed store, using in-memory SQLite for local reconnect labs

### Systemlab files

- `cmd/sessionstream-systemlab/phase3_lab.go` — Phase 3 lab setup
- `cmd/sessionstream-systemlab/static/partials/phase3.html` — page layout
- `cmd/sessionstream-systemlab/static/js/pages/phase3.js` — page behavior

### Tests

- `pkg/sessionstream/transport/ws/server_test.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`
