---
Title: sessionstream design patterns — intern study
Ticket: SS-DESIGN-PATTERNS
Status: draft
Topics:
    - architecture
    - framework
    - event-streaming
    - sessionstream
    - streaming
    - hydration
    - reconnect
    - websocket
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://examples/chatdemo/chat.go
      Note: 'The batch-patch-into-delta reference: runDemoInference delta emit + timelineProjection whole-state upsert (Pattern 2)'
    - Path: repo://examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto
      Note: TokensDeltaEvent carrying chunk + accumulated text (Pattern 2)
    - Path: repo://pkg/analysis/sessionstreamschema/analyzer.go
      Note: Schema-vet analyzer rejecting top-level Struct (Pattern 10)
    - Path: repo://pkg/sessionstream/bus.go
      Note: Watermill envelope, metadata, partition key (Patterns 3,8)
    - Path: repo://pkg/sessionstream/consumer.go
      Note: Bus consumer loop and ordinal assignment at consume time (Pattern 3)
    - Path: repo://pkg/sessionstream/fanout.go
      Note: UIFanout seam (Pattern 1)
    - Path: repo://pkg/sessionstream/hub.go
      Note: The Hub wiring; projectAndApply, RebuildTimeline, projection error policies (Patterns 1,4)
    - Path: repo://pkg/sessionstream/hydration.go
      Note: HydrationStore/EventStore/ProjectionCursorStore/ErrorStore interfaces (Pattern 4)
    - Path: repo://pkg/sessionstream/hydration/sqlite/store.go
      Note: SQLite realization of append-log + materialized view + snapshot (Pattern 4)
    - Path: repo://pkg/sessionstream/ordinals.go
      Note: OrdinalAssigner, DeriveOrdinalFromStreamID (Pattern 3)
    - Path: repo://pkg/sessionstream/projection.go
      Note: UIEvent, TimelineEntity, TimelineView, UIProjection/TimelineProjection (Patterns 1,2)
    - Path: repo://pkg/sessionstream/schema.go
      Note: SchemaRegistry typed registration and decode (Patterns 8,10)
    - Path: repo://pkg/sessionstream/transport/ws/heartbeat.go
      Note: Heartbeat supervisor with timers/writes (Pattern 6)
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Pure heartbeat state machine (Pattern 6)
    - Path: repo://pkg/sessionstream/transport/ws/observer.go
      Note: Best-effort in-order observer dispatcher with drop reporting (Pattern 9)
    - Path: repo://pkg/sessionstream/types.go
      Note: Command, Event, Session (Pattern 1)
    - Path: repo://ttmp/2026/04/20/EVT-STREAM-013--streaming-custom-backend-events-progressive-widgets-and-authoritative-commit-patterns-for-evtstream-chat-apps/design-doc/01-intern-guide-to-streaming-custom-events-progressive-widgets-and-authoritative-commit-in-evtstream-chat-apps.md
      Note: Canonical preview-vs-authoritative-commit design (Pattern 2 discipline)
ExternalSources: []
Summary: 'An in-depth, intern-ready study of the design patterns in the sessionstream framework: command/event/projection/hydration split, the batch-patch-into-delta (preview vs authoritative commit) pattern, ordinals and ordering, the hydration snapshot/replay model, the WebSocket snapshot-before-live contract, the timed-failure-detector heartbeat state machine, and the observer/trace diagnostics layer. With prose, bullets, pseudocode, diagrams, API references, and file references.'
LastUpdated: 2026-08-15T00:00:00-04:00
WhatFor: Onboard a new engineer onto the design patterns used across sessionstream and ground future framework work.
WhenToUse: Read this first when joining sessionstream or touching the hub, projections, hydration, transport, or streaming event families.
---


# sessionstream Design Patterns — Intern Study

> **Audience:** a new engineer (intern) joining `sessionstream`. No prior
> context assumed. This document is the result of an in-depth read of the
> framework code and its design history; it explains the recurring design
> *patterns* the framework is built on — what each one is, why it exists, how
> the code expresses it, and where the sharp edges are — with prose, bullets,
> pseudocode, ASCII diagrams, API references, and file references.
>
> **Scope:** `github.com/go-go-golems/sessionstream` at `HEAD` of
> `task/experiment-verification` (recent log: `d62dca9`). The framework's own
> README and the EVT-STREAM-013 design doc are the companion product-level
> references; this study is the *patterns* companion.

---

## 1. Executive summary

`sessionstream` is a Go framework for **session-scoped, event-driven**
applications where a command produces canonical *backend events*, *projections*
derive both live UI state and durable timeline state from those events, and
clients reconnect through *hydrated snapshots*. The whole framework is a small
set of interfaces arranged around one load-bearing idea:

> **Handlers publish events instead of returning UI state.** Once work is
> described as canonical backend events, different projections can derive
> different views of the same history.

Across the codebase the same handful of design patterns recur. They are the
vocabulary of the framework, and learning them is the fastest way to become
productive:

1. **Command → Event → Projection split** (`pkg/sessionstream/{hub,handler,projection}.go`).
   The single most important pattern. Backend events are the source of truth;
   UI projections and timeline projections are *separate* derivations.
2. **Batch-patch-into-delta (preview vs authoritative commit)** — the streaming
   pattern the user specifically asked about. A streaming producer emits
   *delta* events carrying both the incremental chunk and the accumulated full
   state; *preview* events are live-only and speculative, and a final
   *authoritative commit* event is the only thing that becomes durable. See
   `examples/chatdemo/chat.go` and the EVT-STREAM-013 design doc.
3. **Ordinals as the universal ordering key** (`pkg/sessionstream/ordinals.go`).
   Every event gets a monotonic per-session ordinal; snapshots, entities, and
   UI frames all carry ordinals so clients can reason about freshness.
4. **Hydration as snapshot + replay** (`pkg/sessionstream/hydration.go`,
   `pkg/sessionstream/hydration/sqlite/store.go`). Durable state is a
   materialized projection over an append-only event log; reconnect reads a
   snapshot, then live events.
5. **Snapshot-before-live transport contract** (`pkg/sessionstream/transport/ws/`).
   The WebSocket reconnect protocol: snapshot first, then future live UI
   events, identified by ordinals.
6. **Pure state machine + supervisor (timed failure detector)**
   (`pkg/sessionstream/transport/ws/internal/heartbeat/`). The heartbeat is a
   *pure* reducer separated from I/O, with a supervisor that owns timers and
   writes — a deliberate testability pattern.
7. **Capability seam via small interfaces + functional adapters**
   (`UIProjectionFunc`, `TimelineProjectionFunc`, `UIFanoutFunc`,
   `HubOption`). The framework prefers many tiny interfaces over few fat ones,
   and lets callers pass plain functions.
8. **Capabilities/behavior as data, not branches** (`pkg/sessionstream/schema.go`,
   `pkg/sessionstream/bus.go`). Schemas, bus config, and message mutators are
   *records* consumed by generic code; adding a command/event/machine is adding
   a registration, not editing the hub.
9. **Observers + trace as a best-effort side channel** (`pkg/sessionstream/transport/ws/observer*.go`).
   Diagnostics never run on critical paths; they go through a bounded,
   in-order, drop-reporting dispatcher.
10. **Schema-vet as a compile-time policy** (`pkg/analysis/sessionstreamschema/`,
    `cmd/sessionstream-lint/`). A Go analyzer enforces "no top-level
    `google.protobuf.Struct`" so contracts stay typed end-to-end.

The rest of this document studies each pattern in depth.

---

## 2. The big picture (one diagram)

```text
                         ┌─────────── Command (Name + proto.Message + SessionId) ───────────┐
                         │                                                                │
                         ▼                                                                │
   ┌──────────────────────────────────────────────────────────────┐                 │
   │  Hub  (pkg/sessionstream/hub.go)                              │                 │
   │    ├─ SchemaRegistry   (typed prototypes by name)             │                 │
   │    ├─ commandRegistry  (name -> CommandHandler)              │                 │
   │    ├─ sessionRegistry   (SessionId -> *Session + metadata)    │                 │
   │    ├─ HydrationStore    (Apply / Snapshot / View / Cursor)    │                 │
   │    ├─ UIProjection      (Event -> []UIEvent)                 │                 │
   │    ├─ TimelineProjection (Event -> []TimelineEntity)         │                 │
   │    ├─ UIFanout          (live UI delivery seam)              │                 │
   │    └─ bus? (Watermill pub/sub) -> eventConsumer              │                 │
   └──────────────────────────────────────────────────────────────┘                 │
              │  dispatch(cmd) -> handler(ctx, cmd, sess, pub)                       │
              ▼                                                                      │
   ┌──────────────────────────┐    pub.Publish(Event)    ┌────────────────────────┐  │
   │ CommandHandler (your app)│ ───────────────────────▶ │ EventPublisher          │  │
   │  - validates command     │                          │  localEventPublisher:   │  │
   │  - does work over time   │                          │   assign ordinal,        │  │
   │  - publishes canonical   │                          │   projectAndApply()      │  │
   │   backend events         │                          │  OR watermillEventPub:   │  │
   └──────────────────────────┘                          │   encode -> bus topic    │  │
                                                         └────────────────────────┘  │
              ┌───────────────────────────────────────────────────┘
              ▼
   projectAndApply(ev):
     1. (if EventStore) AppendEvent(ev)            ── append-only log
     2. View = store.View(sid)                      ── current materialized state
     3. uiEvents   = UIProjection.Project(ev, view)
     4. entities   = TimelineProjection.Project(ev, view)
     5. store.Apply(sid, ev.Ordinal, entities)      ── upsert materialized state
     6. (if ProjectionCursorStore) AdvanceProjectionCursor("timeline", sid, ord)
     7. fanout.PublishUI(sid, ord, clone(uiEvents)) ── live delivery
              │
              ▼
   ┌──────────────────────────────┐        ┌──────────────────────────────────────┐
   │ HydrationStore (durable)     │        │ UIFanout -> WebSocket server          │
   │  sessionstream_entities      │◀───────│  snapshot-before-live:                │
   │  sessionstream_entity_versions│       │   1. Snapshot(snapshotOrdinal)        │
   │  sessionstream_events (log)  │        │   2. live UiEventFrame(eventOrdinal)   │
   │  sessionstream_projection_   │        │  + heartbeat state machine + observers │
   │    cursors                    │        └──────────────────────────────────────┘
   └──────────────────────────────┘
```

The vertical seams are the patterns. Read this diagram once; the rest of the
document walks each box.

---

## 3. Pattern 1 — Command → Event → Projection split

### 3.1 What it is

A **command** is an external request to do work. A **backend event** is the
canonical record of something that happened. A **projection** translates one
backend event into a particular view. The framework keeps three projections
distinct on purpose:

- the **command handler** does work and *publishes backend events* (it does
  **not** return UI state);
- the **UI projection** turns a backend event into live, client-facing `UIEvent`s;
- the **timeline projection** turns the same backend event into durable
  `TimelineEntity`s.

### 3.2 Why it exists

> "This separation is the design's most important property. Backend events
> describe what happened. UI projections decide what live clients should see
> now. Timeline projections decide what durable state should exist after the
> event." — README

Concretely: if a frontend rendering detail changes, you can update the UI
projection without touching the canonical event or the durable model. If the
persistence shape changes, you update the timeline projection without touching
the handler. The backend event stream stays **authoritative** and stable.

### 3.3 How the code expresses it

```go
// pkg/sessionstream/types.go
type Command struct { Name string; Payload proto.Message; SessionId SessionId }
type Event   struct { Name string; Payload proto.Message; SessionId SessionId; Ordinal uint64 }

// pkg/sessionstream/handler.go
type CommandHandler func(ctx context.Context, cmd Command, sess *Session, pub EventPublisher) error
type EventPublisher  interface { Publish(ctx context.Context, ev Event) error }

// pkg/sessionstream/projection.go
type UIEvent       struct { Name string; Payload proto.Message }
type TimelineEntity struct {
    Kind, Id string; CreatedOrdinal, LastEventOrdinal uint64
    Payload proto.Message; Tombstone bool
}
type UIProjection       interface { Project(ctx, ev, sess, view) ([]UIEvent, error) }
type TimelineProjection interface { Project(ctx, ev, sess, view) ([]TimelineEntity, error) }
```

Note the `TimelineView` argument to projections: a projection can read the
*current* materialized state (`view.Get(kind, id)`) to compute its output. This
is what lets the chatdemo timeline projection *accumulate* deltas into a single
`ChatMessage` entity (Pattern 2).

The hub wires exactly one of each projection (`hub.go`, `RegisterUIProjection`
/ `RegisterTimelineProjection` reject a second registration), and the central
method `projectAndApply` runs both and persists only the timeline output:

```go
// pkg/sessionstream/hub.go — projectAndApply (simplified)
func (h *Hub) projectAndApply(ctx, ev) ([]UIEvent, error) {
    if es, ok := h.store.(EventStore); ok { es.AppendEvent(ctx, ev) }   // append-only log
    sess, _ := h.sessions.GetOrCreate(ctx, ev.SessionId)
    view, _ := h.store.View(ctx, ev.SessionId)                          // current state
    uiEvents, uiErr := h.uiProjection.Project(ctx, ev, sess, view)      // live
    entities, tlErr  := h.timelineProjection.Project(ctx, ev, sess, view)// durable
    // policy: fail or advance-on-error, per-projection
    h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entities)               // upsert
    if pcs, ok := h.store.(ProjectionCursorStore); ok && tlErr == nil {
        pcs.AdvanceProjectionCursor(ctx, "timeline", ev.SessionId, ev.Ordinal)
    }
    if h.fanout != nil && uiErr == nil && len(uiEvents) > 0 {
        h.fanout.PublishUI(ctx, ev.SessionId, ev.Ordinal, cloneUIEvents(uiEvents))
    }
    return uiEvents, nil
}
```

### 3.4 API references

- `Hub.Submit(ctx, sid, name, payload)` — validates the payload type against
  the registry, builds a `Command`, dispatches to the handler
  (`hub.go`).
- `Hub.RegisterCommand(name, handler)`, `RegisterUIProjection`,
  `RegisterTimelineProjection` — one-shot registrations.
- `Hub.Snapshot(ctx, sid)`, `Hub.Cursor(ctx, sid)`,
  `Hub.ProjectionCursor(ctx, projector, sid)` — read-side.
- `Hub.RebuildTimeline(ctx, sid, from)` /
  `RebuildTimelineFromScratch` — replay the event log through the timeline
  projection to repair materialized state.

### 3.5 Sharp edges

- **Projection error policy is explicit** (`ProjectionPolicies{UI, Timeline}`,
  each `Fail` or `Advance`). Default is `Fail`. `Advance` lets a projection
  error not block cursor movement — useful but it means a projection can
  silently skip; choose deliberately (`hub.go`).
- **UI events are cloned before fanout** (`cloneUIEvents`) so a transport that
  mutates payloads can't corrupt the in-memory projection output.
- The hub reports errors to an optional `ErrorStore` (`reportError`) without
  swallowing the operation's real return value.

---

## 4. Pattern 2 — Batch-patch-into-delta (preview vs authoritative commit)

This is the pattern the user asked about. It has two layers: (a) the **delta
event** shape that carries both the chunk and the accumulated state, and (b)
the **two-phase preview/commit** discipline that keeps streaming speculation
out of durable state.

### 4.1 The delta event shape (the "batch patch into delta")

When a producer streams tokens, the worst design is to send only the chunk and
make the client accumulate, *or* to send only the full text and re-transmit
everything each tick. The chatdemo does both, in one event:

```proto
// examples/chatdemo/proto/.../chat.proto
message TokensDeltaEvent {
  string message_id = 1;
  string role       = 2;
  string chunk      = 3;   // the DELTA: just this token batch
  string text       = 4;   // accumulated full text so far
  string content    = 5;   // alias of text (display string)
  string status     = 6;   // "streaming" | "finished" | "stopped"
  bool   streaming  = 7;
}
```

The handler accumulates server-side and publishes one event per chunk carrying
**both** the incremental `chunk` and the full `text`:

```go
// examples/chatdemo/chat.go — runDemoInference (simplified)
accumulated := ""
for _, chunk := range chunkText(answer, 10) {
    select {
    case <-ctx.Done():
        stopped := &chatdemov1.InferenceStoppedEvent{ /* ..., Text: accumulated, Status: "stopped" */ }
        _ = e.publish(ctx, sid, pub, EventInferenceStopped, stopped)
        return
    case <-time.After(e.chunkDelay):
    }
    accumulated += chunk
    delta := &chatdemov1.TokensDeltaEvent{
        MessageId: messageID, Role: "assistant",
        Chunk:  chunk,            // delta (this batch)
        Text:   accumulated,       // accumulated full state
        Content: accumulated,
        Status: "streaming", Streaming: true,
    }
    _ = e.publish(ctx, sid, pub, EventTokensDelta, delta)
}
finished := &chatdemov1.InferenceFinishedEvent{ /* ..., Text: accumulated, Status: "finished" */ }
_ = e.publish(ctx, sid, pub, EventInferenceFinished, finished)
```

Why carry both? Two consumers, two needs:

- The **UI projection** can render progressively from `chunk` (cheap append) or
  re-render from `text` (idempotent). A late-joining live client that missed
  earlier deltas can still render the current `text`.
- The **timeline projection** upserts the *whole* `ChatMessage` entity from
  `text`/`content` each tick — a "batch patch" of the materialized entity
  rather than a list of patches. The store's `Apply` is an idempotent upsert
  keyed by `(session, kind, entity_id)`, so re-applying the full state is safe.

```go
// examples/chatdemo/chat.go — timelineProjection (simplified)
case *chatdemov1.TokensDeltaEvent:
    entity.Role = "assistant"
    entity.Content = payload.GetContent()   // whole-state patch
    entity.Text    = payload.GetText()
    entity.Status  = "streaming"
    entity.Streaming = true
// ... returns []TimelineEntity{{Kind: "ChatMessage", Id: messageID, Payload: entity}}
```

This is the **batch-patch-into-delta** shape: the wire event is a *delta*
(small, streaming), but the projection *patches the durable entity with the
full accumulated state* (a batch upsert), so materialized state is always a
single row per message — never a log of patches the client must fold.

### 4.2 The two-phase discipline (preview vs authoritative commit)

Carrying deltas is not enough when the streaming producer is *speculative* —
e.g. a structured-output parser that incrementally parses YAML while tokens
flow. There the partial parse may be wrong or incomplete. The framework's
discipline (documented in EVT-STREAM-013 and reinforced by the Geppetto
`FilteringSink` comment) is to split events into two classes:

```text
PROVISIONAL (preview)                    AUTHORITATIVE (commit)
─────────────────────                    ───────────────────────
emitted while streaming                  emitted only at the final boundary
may be incomplete or wrong               derived from the full output
UIProjection: live UI event             UIProjection: live UI event (+ clear preview)
TimelineProjection: SKIP (live-only)    TimelineProjection: upsert durable entity
not hydrated, not replayed               hydrated, replayable, survives reconnect
```

Concretely, for an "agent mode switch" widget:

```go
// projection split (EVT-STREAM-013, sketch)
func uiProjection(ev Event) []UIEvent {
    switch ev.Name {
    case "ChatAgentModePreviewUpdated":
        return []UIEvent{{Name: "AgentModePreviewUpdated", Payload: ev.Payload}}
    case "ChatAgentModeCommitted":
        return []UIEvent{
            {Name: "AgentModeCommitted", Payload: ev.Payload},
            {Name: "AgentModePreviewCleared", Payload: ev.Payload}, // clear the widget
        }
    }
    return nil
}

func timelineProjection(ev Event) []TimelineEntity {
    switch ev.Name {
    case "ChatAgentModeCommitted":                       // commit ONLY
        return []TimelineEntity{{Kind: "AgentMode", Id: "session", Payload: ev.Payload}}
    default:
        return nil // preview stays live-only in v1
    }
}
```

### 4.3 Why preview is not persisted

> "Persisting that preview into hydrated state would blur a very important
> boundary: 'this is the best current guess' versus 'this is the application's
> final conclusion.'" — EVT-STREAM-013

If a stream is interrupted, a persisted preview would survive a reconnect and
show stale speculative UI. The discipline: the **final middleware pass** (which
sees the full assistant output) is the only authoritative boundary. Streaming
extraction is for UX/telemetry; the commit is for durable state.

### 4.4 How deltas and the snapshot/replay model compose

This is where Pattern 2 and Pattern 4 reinforce each other:

```text
live client (connected):
  ChatTokensDelta(ordinal=5) -> UI: ChatMessageAppended -> render chunk + accumulated text
  ChatTokensDelta(ordinal=6) -> UI: ChatMessageAppended -> render chunk + accumulated text
  ...
reconnecting client (missed 1..9):
  Snapshot(snapshotOrdinal=9) -> entities: [ChatMessage{ text: full, lastEventOrdinal: 9 }]
  then live: ChatTokensDelta(ordinal=10) -> ChatMessageAppended
```

Because the timeline projection upserts the **whole** entity each tick, the
snapshot contains the full accumulated message at ordinal 9 — the reconnecting
client doesn't need deltas 1..9. Then live deltas resume from ordinal 10. The
batch-patch-into-delta shape is what makes snapshot-before-live work for
streaming content.

### 4.5 Sharp edges / rules of thumb

- **Always carry the accumulated state in a delta event** if the entity is
  rendered as a whole (a message). Carry only the patch if the entity is a
  collection and the patch is an append to a list.
- **Never persist preview events** in v1. If you must, give them a distinct
  entity kind with explicit cleanup on commit/stop.
- **The commit event should clear the preview** in the UI projection so a
  connected client doesn't keep showing a stale widget.
- **Ordinal-tag every event** so a late delta after a snapshot can be detected
  and ignored by the client (see Pattern 3).

---

## 5. Pattern 3 — Ordinals as the universal ordering key

### 5.1 What it is

Every backend event gets a `uint64` **ordinal**, monotonic per session.
Ordinals are not just sequence numbers — they are the join key across the
event log, materialized entities, projection cursors, snapshots, and live UI
frames.

```go
// pkg/sessionstream/types.go
type Event struct { Name string; Payload proto.Message; SessionId SessionId; Ordinal uint64 }

// pkg/sessionstream/projection.go
type TimelineEntity struct {
    Kind, Id string
    CreatedOrdinal   uint64   // ordinal that first created this entity
    LastEventOrdinal uint64   // latest ordinal that updated it
    Payload proto.Message; Tombstone bool
}
```

### 5.2 Two assignment paths

The framework supports two ordinal sources, switchable by whether a bus is
configured:

1. **Local hub** (`localEventPublisher` → `Hub.nextLocalOrdinal`): an
   in-memory `map[SessionId]uint64` seeded from the store's event cursor, then
   incremented. Simple, single-writer, no ordering hazard.
2. **Bus-backed** (`eventConsumer` → `OrdinalAssigner`): ordinals are assigned
   at *consume* time. The assigner can **derive** an ordinal from a Redis-style
   stream id in the message metadata (`DeriveOrdinalFromStreamID`) so that
   across consumers/restarts the ordinal is stable and monotonic, falling back
   to `current+1` when no stream id is present.

```go
// pkg/sessionstream/ordinals.go — OrdinalAssigner.Next (simplified)
next := current + 1
if streamID := ExtractStreamID(metadata); streamID != "" {
    if derived, ok := DeriveOrdinalFromStreamID(streamID); ok && derived > current {
        next = derived
    }
}
if next <= current { next = current + 1 }   // never go backwards
o.current[sid] = next
return next, nil
```

### 5.3 Why it matters

- **Ordering**: the bus can shard by `PartitionKeyForSession(sid)` so all events
  for a session reach one consumer in order; ordinals make that order durable
  and queryable.
- **Reconnect freshness**: `Snapshot.snapshotOrdinal` tells the client the
  highest materialized ordinal; `UiEventFrame.eventOrdinal` tells it which
  backend event produced a live frame; `SnapshotEntity.lastEventOrdinal` tells
  it how stale each entity is.
- **Idempotent apply**: the SQLite `Apply` writes a version row keyed by
  `(session, kind, entity_id, ordinal)` — re-applying the same ordinal is a
  no-op/conflict-detect, not a duplicate.

### 5.4 Sharp edges

- **Browser clients must treat `uint64` ordinals as strings** (protobuf JSON
  emits them as strings to avoid JS `number` precision loss). Don't coerce
  through `Number()`.
- **Deriving from a stream id can jump** (ms*1e6+seq), which is fine —
  monotonicity is what's required, not contiguity.

---

## 6. Pattern 4 — Hydration: append-only log + materialized projection + snapshot

### 6.1 What it is

Durable state is a **materialized projection** over an **append-only event
log**, with **snapshot** as the read path for reconnect. The store interface is
intentionally split into optional capabilities:

```go
// pkg/sessionstream/hydration.go
type HydrationStore interface {
    Apply(ctx, sid, ord, entities []TimelineEntity) error   // upsert materialized state
    Snapshot(ctx, sid, asOf uint64) (Snapshot, error)        // read materialized state
    View(ctx, sid) (TimelineView, error)                     // current state for projections
    Cursor(ctx, sid) (uint64, error)                         // highest applied ordinal
}
type EventStore interface {            // optional: persist the raw event log
    AppendEvent(ctx, ev) error
    Events(ctx, sid, after uint64, limit int) ([]Event, error)
    EventCursor(ctx, sid) (uint64, error)
}
type ProjectionCursorStore interface { // optional: per-projector progress
    ProjectionCursor(ctx, projector, sid) (uint64, error)
    AdvanceProjectionCursor(ctx, projector, sid, ord) error
}
type TimelineResetStore interface { ClearTimeline(ctx, sid) error }   // optional: rebuild
type ErrorStore interface { RecordError(ctx, rec) error }             // optional: DLQ
```

This is a classic CQRS/event-sourcing split, but deliberately *partial*: a
store can implement just `HydrationStore` (the noop store does) and the hub
still works. The SQLite store implements **all** of them.

### 6.2 The SQLite realization (the pattern made concrete)

Schema (`pkg/sessionstream/hydration/sqlite/store.go` migrate):

```text
sessionstream_events          (session_id, ordinal) PK     -- append-only log
sessionstream_entities        (session_id, kind, entity_id) PK  -- materialized "current"
sessionstream_entity_versions(session_id, kind, entity_id, ordinal) PK  -- point-in-time
sessionstream_projection_cursors(projector, session_id) PK   -- per-projector cursor
sessionstream_sessions(session_id) PK, snapshot_ordinal       -- per-session high-water
sessionstream_errors(id autoincrement)                        -- DLQ
```

`Apply` is a single transaction that, for each entity:

1. inserts a **version row** at this ordinal (point-in-time state, upsertable
   at the same ordinal — idempotent),
2. if not a tombstone, upserts the **current** row (the materialized view),
3. if a tombstone, deletes the current row (so snapshots skip it),
4. advances the session `snapshot_ordinal` monotonically (`CASE WHEN
   excluded.snapshot_ordinal > ... THEN ...`).

`Snapshot(asOf=0)` reads the current rows; `Snapshot(asOf=k)` does a
point-in-time query by joining `entity_versions` against `MAX(ordinal) WHERE
ordinal <= k` per entity — true temporal lookup.

### 6.3 Why this shape

- **Append-only log + materialized view** gives both replay (`RebuildTimeline`
  re-runs the timeline projection from `Events(after)`) and fast reconnect
  (read `entities`).
- **Per-projector cursors** let the timeline projection be rebuilt
  independently of the event log (e.g. after a projection bug fix) via
  `RetryTimeline` / `RebuildTimelineFromScratch`, without losing events.
- **Idempotent `Apply`** (version rows keyed by ordinal, `INSERT OR IGNORE`
  on events with conflict detection) makes redelivery safe — essential for a
  bus consumer that may nack/retry.

### 6.4 Sharp edges

- `Snapshot(asOf)` is only meaningful for stores that keep version history
  (SQLite does). A store that only keeps current state can ignore `asOf`.
- `ClearTimeline` resets the projection cursor **and** `snapshot_ordinal`, so
  a from-scratch rebuild re-applies from ordinal 0.
- The store is the **only** writer to materialized state; projections return
  *new* entity values and the store upserts them. Projections must not mutate
  the `view` they read (the SQLite `view.Get` returns `proto.Clone`s).

---

## 7. Pattern 5 — Snapshot-before-live transport contract

### 7.1 What it is

The WebSocket reconnect protocol is the user-visible payoff of Patterns 2–4.
On subscribe, the server sends the current snapshot, then future live UI
events — never a replay of past deltas to a reconnecting client.

```text
client subscribes(sid)
  -> server: Snapshot(snapshotOrdinal=N, entities=[...])     // current materialized state
  -> server: UiEventFrame(eventOrdinal=N+1, ... )             // future live events only
  -> server: UiEventFrame(eventOrdinal=N+2, ... )
  ...
```

The wire format is **protobuf JSON** over WebSocket text frames
(`proto/sessionstream/v1/transport.proto`), so browser JS can consume it with
`JSON.parse` while still following a typed contract.

### 7.2 Why snapshot-first

A reconnecting client should not ask handlers to replay product logic. It
receives durable timeline state (the snapshot, which is the *result* of all
past deltas) and then subscribes to future live events. This is exactly the
"batch patch into delta" property made resilient: the snapshot *is* the batch,
and deltas only resume *after* it.

### 7.3 The ordinal join

The contract fields are ordinal-tagged so the client can detect ordering
issues:

- `Snapshot.snapshotOrdinal` — highest timeline ordinal in the snapshot.
- `SnapshotEntity.createdOrdinal`, `lastEventOrdinal` — per-entity freshness.
- `UiEventFrame.eventOrdinal` — backend event that produced this live frame.

A client that receives a live frame with `eventOrdinal <= snapshotOrdinal`
knows it's a duplicate/stale and can drop it.

### 7.4 Where it lives

`pkg/sessionstream/transport/ws/server.go` (the WS server), the snapshot/fanout
adapter, and `proto/sessionstream/v1/transport.proto` (the frame schema). The
README's "WebSocket and hydration contract" section is the canonical
description.

---

## 8. Pattern 6 — Pure state machine + supervisor (timed failure detector)

### 8.1 What it is

The WebSocket heartbeat is a **timed failure detector**: the server sends an
application-level `PingFrame` with an opaque nonce; the client must echo it in
a `PongFrame`. If a pong doesn't arrive within `PongTimeout`, the connection is
"suspected" and closed so the client can reconnect and hydrate.

The crucial design choice: the detector is a **pure state machine** separated
from all I/O and timers, with a **supervisor** that owns the side effects.

### 8.2 The machine

```go
// pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
type Phase uint8 // Booting | Idle | Writing | Awaiting | Suspected | Stopped
type EventKind uint8 // Ready | Tick | PingWritten | PingWriteFailed | PongReceived | ...

type Machine struct { phase Phase; ... }
func (m *Machine) Step(ev Event) (Phase, []Effect, error)  // PURE: no I/O, no time
```

`Step` takes an input event and returns the new phase **plus a list of
effects** (things the supervisor should do, like "start a timer", "write a
ping", "close"). The machine itself does none of that. This makes the detector
**trivially testable** — every transition is a pure function, and the repo has
`machine_test.go`, `heartbeat_arbitration_test.go`, and a pragmatic fuzzing plan
(`design-doc/02-pragmatic-stateful-fuzzing-plan-for-the-heartbeat-reducer.md`).

### 8.3 The supervisor

The supervisor (`pkg/sessionstream/transport/ws/heartbeat.go`) owns the real
timer, the real socket write, and the nonce generation. It feeds observed
events into the machine and executes the machine's effects. Critical
correctness properties (from the README and SESSIONSTREAM-005):

- `PongTimeout` starts **only after the sole writer successfully writes the
  ping** — time spent in the server's outbound queue is not charged to the
  client.
- **One challenge outstanding at a time**; the next interval begins after a
  matching pong.
- **Stale nonces and stale timer generations cannot acknowledge or expire the
  current challenge** (generation counters guard against timer races).
- A heartbeat timeout means the connection is **suspected under the configured
  timing assumption**, not *proven* crashed — network delay, a paused browser
  event loop, or scheduler delay can produce the same silence.

### 8.4 Why this pattern

- **Testability**: the hard logic (state transitions, race arbitration) is pure
  and unit-testable; only the boring glue is impure.
- **Race safety**: timers and writes are serialized through the supervisor,
  and generation counters reject late callbacks.
- **Honesty**: "suspected" is not "dead"; the framework says what it actually
  measured.

### 8.5 Sharp edges

- The detector closes a *suspected* connection so the client reconnects and
  hydrates from a fresh snapshot — it does not try to repair the live stream.
- Application-level ping/pong is used because **browser JS cannot originate
  RFC 6455 Ping/Pong control frames**. The nonce must be echoed **unchanged**
  (opaque to the client).

---

## 9. Pattern 7 — Small interfaces + functional adapters

The framework prefers **many tiny interfaces** over few fat ones, and lets
callers pass plain functions wherever an interface is needed.

```go
// pkg/sessionstream/projection.go
type UIProjectionFunc func(ctx, ev, sess, view) ([]UIEvent, error)
func (f UIProjectionFunc) Project(...) (...) { return f(...) }

type TimelineProjectionFunc func(ctx, ev, sess, view) ([]TimelineEntity, error)
func (f TimelineProjectionFunc) Project(...) (...) { return f(...) }

// pkg/sessionstream/fanout.go
type UIFanoutFunc func(ctx, sid, ord, events) error
func (f UIFanoutFunc) PublishUI(...) error { return f(...) }

// pkg/sessionstream/hub.go — functional options for construction
type HubOption func(*Hub) error
func WithSchemaRegistry(r) HubOption { ... }
func WithHydrationStore(s) HubOption { ... }
func WithEventBus(pub, sub, opts...) HubOption { ... }
```

Why:

- A downstream app can install a projection as a closure (`chatdemo.Install`
  registers `UIProjectionFunc(uiProjection)`), no boilerplate type.
- `HubOption` makes the hub constructible from composable, validated options
  (each option can return an error, e.g. "schema registry is nil").
- The interfaces are narrow enough that faking them in tests is trivial
  (`UIFanoutFunc` that captures into a slice).

---

## 10. Pattern 8 — Capabilities/behavior as data, not branches

Adding a command, event, UI event, entity, or machine is **adding a
registration record**, not editing the hub. The hub never switches on a name;
it looks up a prototype in the `SchemaRegistry` and a handler in the
`commandRegistry`.

```go
// pkg/sessionstream/schema.go — registration is data
reg.RegisterCommand("ChatStartInference", &chatdemov1.StartInferenceCommand{})
reg.RegisterEvent("ChatTokensDelta", &chatdemov1.TokensDeltaEvent{})
reg.RegisterUIEvent("ChatMessageAppended", &chatdemov1.ChatMessageUpdate{})
reg.RegisterTimelineEntity("ChatMessage", &chatdemov1.ChatMessageEntity{})
```

The schema registry also owns **typed decode**: `DecodeCommandJSON(name, bytes)`
instantiates the registered prototype via `protoReflect().New()` and
`protojson.Unmarshal`es into it — so the wire is JSON but the in-memory is
strongly typed. Payload type is validated on every publish/submit
(`Hub.validatePayloadType` compares `Descriptor().FullName()`).

The same pattern appears for the bus: `busConfig` is a record
(`publisher`, `subscriber`, `topic`, `messageMutator`); `WithEventBus` is a
`HubOption` that builds it; `BusMessageMutator` is a function you can inject to
attach backend-specific metadata (e.g. synthetic stream ids in a lab).

---

## 11. Pattern 9 — Observers + trace as a best-effort side channel

### 11.1 What it is

Diagnostics (hub events, websocket lifecycle, heartbeat arbitration) are
delivered to **observers** through a **bounded, in-order, best-effort
dispatcher** that never runs on a critical path and **reports drops**.

```text
critical path (socket reader/writer, heartbeat, request-worker, lifecycle)
        │
        ▼  (non-blocking send)
bounded observer dispatcher ── in-order ──▶ Observer callbacks
        │
        ▼ (on backpressure)
Server.ObserverDroppedRecords++   (reported, not silently lost)
```

### 11.2 Why

- Observers must not block or kill a connection because they're slow.
- Observers must not be *lossy without reporting* — silent loss makes
  diagnostics useless. So the dispatcher counts dropped records
  (`Server.ObserverDroppedRecords`).
- Order matters for reconstructing timelines, so the dispatcher is in-order.

### 11.3 The trace subsystem

`pkg/sessionstream/transport/ws/observer_trace*.go` implements a structured
trace harvest (the recent commits `957c906`, `229a47e`, `4754f78` hardened
this): observer records are accumulated, JSONL-serializable, and harvested
without blocking. There's an `observer_trace_benchmark_test.go` and a
`synctest`-based test (`observer_synctest_test.go`) — the framework treats
observer concurrency as a first-class correctness concern.

### 11.4 Sharp edges

- Observers are **best-effort**: don't build correctness on them. If you need a
  durable record, use the `ErrorStore` (Pattern 4), not an observer.
- `Server.Close(ctx)` drains accepted observations during shutdown; the context
  bounds a callback that does not return.

---

## 12. Pattern 10 — Schema-vet as a compile-time policy

### 12.1 What it is

A Go analyzer (`pkg/analysis/sessionstreamschema/analyzer.go`, run via
`cmd/sessionstream-lint` / `make schema-vet`) rejects schema registrations
whose top-level payload is `*google.protobuf.Struct`:

```go
// Rejected by sessionstream-lint:
reg.RegisterEvent("ToolCallUpdated", &structpb.Struct{})
```

### 12.2 Why

A top-level `Struct` is an arbitrary JSON object. It hides the contract from
Go, from generated frontend code, from hydration, and from reviewers. The
policy: top-level command/event/UI-event/entity must be a **named, concrete
protobuf message**. `Struct` may still appear *inside* a message when a field is
intentionally open-ended metadata.

### 12.3 Why it's a pattern worth naming

It's the framework's way of making "contracts stay typed end-to-end" a
**build-time guarantee** rather than a convention. This pairs with Pattern 8
(registration is data): the analyzer scans `Register*` calls and checks the
prototype's descriptor — the same data the registry uses at runtime.

---

## 13. Cross-cutting: how the patterns compose in one request

Walk a streaming chat command through every pattern:

```text
1. client POSTs StartInference  ──────────────────────────────────────────────── Pattern 1 (command)
2. Hub.Submit validates payload type vs SchemaRegistry, dispatches ─────────── Patterns 1,8,10
3. handler publishes ChatUserMessageAccepted, ChatInferenceStarted ─────────── Pattern 1
4. localEventPublisher assigns ordinal (nextLocalOrdinal, seeded from store) ── Pattern 3
5. projectAndApply:
   a. EventStore.AppendEvent(ev)            ─────────────────────────────────── Pattern 4 (log)
   b. store.View(sid)                       ─────────────────────────────────── Pattern 4
   c. UIProjection -> ChatMessageStarted    ─────────────────────────────────── Pattern 1
   d. TimelineProjection -> ChatMessage{empty} ──────────────────────────────── Pattern 1
   e. store.Apply(upsert)                   ─────────────────────────────────── Pattern 4
   f. fanout.PublishUI(ordinal, uiEvents)   ─────────────────────────────────── Pattern 1
6. handler streams: for each token batch, publish ChatTokensDelta(chunk+text) ── Pattern 2 (delta)
   └ repeat 4-5; timeline upserts the WHOLE ChatMessage each tick (batch patch) ─ Pattern 2
7. handler publishes ChatInferenceFinished (whole text) ──────────────────────── Patterns 1,2
8. websocket server: live UiEventFrame(eventOrdinal) to connected clients ────── Pattern 5
9. a client disconnects; heartbeat machine -> PhaseSuspected -> close ────────── Pattern 6
10. client reconnects: server sends Snapshot(snapshotOrdinal=N) ──────────────── Patterns 4,5
    then live UiEventFrame(ordinal=N+1..) ─────────────────────────────────────── Patterns 3,5
11. observer records the lifecycle (best-effort, drop-counted) ───────────────── Pattern 9
```

Every pattern fires in one request. That's why they're worth learning as a
vocabulary rather than as scattered features.

---

## 14. Where to read next (file references, absolute paths)

**Core substrate:**
- `/home/manuel/code/wesen/go-go-golems/sessionstream/pkg/sessionstream/types.go` — `Command`, `Event`, `Session`.
- `.../pkg/sessionstream/handler.go` — `CommandHandler`, `EventPublisher`.
- `.../pkg/sessionstream/projection.go` — `UIEvent`, `TimelineEntity`, `TimelineView`, projections.
- `.../pkg/sessionstream/hub.go` — the wiring; `projectAndApply`, `RebuildTimeline`, policies.
- `.../pkg/sessionstream/schema.go` — `SchemaRegistry`, typed decode.
- `.../pkg/sessionstream/command_registry.go`, `session_registry.go` — registries.
- `.../pkg/sessionstream/ordinals.go` — `OrdinalAssigner`, stream-id derivation.
- `.../pkg/sessionstream/bus.go` — Watermill envelope, metadata, decode.
- `.../pkg/sessionstream/consumer.go` — bus consumer loop.
- `.../pkg/sessionstream/hydration.go` — store interfaces (Pattern 4).
- `.../pkg/sessionstream/hydration/sqlite/store.go` — the full realization (Pattern 4).

**The batch-patch-into-delta reference app:**
- `/home/manuel/code/wesen/go-go-golems/sessionstream/examples/chatdemo/chat.go` — `runDemoInference` (delta emit), `timelineProjection` (batch upsert).
- `.../examples/chatdemo/proto/.../chat.proto` — `TokensDeltaEvent` (chunk + text).
- `.../examples/chatdemo/chat_test.go` — executable expectations.

**Transport & heartbeat:**
- `/home/manuel/code/wesen/go-go-golems/sessionstream/proto/sessionstream/v1/transport.proto` — wire frames.
- `.../pkg/sessionstream/transport/ws/server.go` — WS server + snapshot/fanout.
- `.../pkg/sessionstream/transport/ws/heartbeat.go` — supervisor.
- `.../pkg/sessionstream/transport/ws/internal/heartbeat/machine.go` — pure machine (Pattern 6).
- `.../pkg/sessionstream/transport/ws/observer*.go` — observer dispatcher + trace (Pattern 9).

**Schema vet:**
- `.../pkg/analysis/sessionstreamschema/analyzer.go`, `.../cmd/sessionstream-lint/` (Pattern 10).

**Design history (the "why"):**
- `.../ttmp/2026/04/20/EVT-STREAM-013--.../design-doc/01-intern-guide-to-streaming-custom-events-progressive-widgets-and-authoritative-commit-in-evtstream-chat-apps.md` — the canonical preview/commit pattern (Pattern 2 discipline).
- `.../ttmp/2026/08/10/SESSIONSTREAM-005--.../design-doc/01-...-heartbeat-state-machine.md`, `02-pragmatic-stateful-fuzzing-plan...`, `03-...-supervisor-event-arbitration...` — Pattern 6.
- `.../ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--.../design-doc/01-whole-package-code-review-and-intern-implementation-guide.md` — whole-package review.
- `.../pkg/doc/topics/01-user-guide.md` — the conceptual user guide.

---

## 15. How to validate your understanding (quick commands)

```bash
cd ~/code/wesen/go-go-golems/sessionstream
make test            # whole framework
go test ./pkg/sessionstream ./examples/chatdemo -count=1   # core + delta demo
go test ./pkg/sessionstream/transport/ws/... -count=1     # heartbeat + observers
make schema-vet      # Pattern 10 policy
```

Read `examples/chatdemo/chat_test.go` first — it asserts the delta/snapshot
behavior end to end and is the fastest way to confirm Pattern 2 and Pattern 4
in action.
