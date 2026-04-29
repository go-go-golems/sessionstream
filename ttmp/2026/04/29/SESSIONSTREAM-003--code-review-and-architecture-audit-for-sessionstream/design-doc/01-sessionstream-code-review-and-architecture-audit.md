---
Title: Sessionstream code review and architecture audit
Ticket: SESSIONSTREAM-003
Status: active
Topics:
    - architecture
    - backend
    - event-streaming
    - framework
    - onboarding
    - code-review
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/phase2_lab.go
      Note: Systemlab long-file and repeated phase runtime patterns reviewed
    - Path: cmd/sessionstream-systemlab/phase5_lab.go
      Note: Persistence/restart demo reviewed for ordinal and cursor semantics
    - Path: examples/chatdemo/chat.go
      Note: Reference domain example reviewed for schema and projection patterns
    - Path: pkg/sessionstream/consumer.go
      Note: Watermill consumer decode and acknowledgement behavior reviewed
    - Path: pkg/sessionstream/hub.go
      Note: Core hub runtime
    - Path: pkg/sessionstream/hydration.go
      Note: HydrationStore API contract reviewed
    - Path: pkg/sessionstream/hydration/sqlite/store.go
      Note: Persistent current-state store
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Websocket subscribe
ExternalSources: []
Summary: Architecture map and evidence-backed code review of sessionstream for onboarding, API clarity, cleanup, persistence, websocket transport, and systemlab maintainability.
LastUpdated: 2026-04-29T14:44:36-04:00
WhatFor: Use this when onboarding to sessionstream or planning cleanup/refactor work after the extraction.
WhenToUse: Read before changing the hub, hydration stores, websocket transport, examples/chatdemo, or sessionstream-systemlab phases.
---


# Sessionstream code review and architecture audit

## Executive summary

`sessionstream` is a small Go framework for session-keyed streaming applications. Its core idea is sound: commands enter through a `Hub`, command handlers publish backend events, projections turn those backend events into both live UI events and persistent timeline entities, transports fan UI events out to clients, and hydration stores provide reconnect snapshots. The repository is intentionally framework-owned rather than product-owned, as stated in `AGENT.md` and `README.md`.

The core package is compact and mostly easy to read. The public seams are concentrated in a few files:

- `pkg/sessionstream/types.go` defines `SessionId`, `ConnectionId`, `Command`, `Event`, and `Session`.
- `pkg/sessionstream/hub.go` wires commands, sessions, projections, fanout, event publishing, and optional Watermill consumption.
- `pkg/sessionstream/projection.go` defines `UIProjection`, `TimelineProjection`, and `TimelineView`.
- `pkg/sessionstream/hydration.go` defines `HydrationStore` and `Snapshot`.
- `pkg/sessionstream/transport/ws/server.go` is the websocket transport and also implements `UIFanout`.
- `pkg/sessionstream/hydration/memory/store.go` and `pkg/sessionstream/hydration/sqlite/store.go` are the two store implementations.

The main risks are not basic readability bugs. They are contract mismatches between APIs and behavior:

1. `HydrationStore.Snapshot(ctx, sid, asOf)` exposes an `asOf` parameter, but both stores ignore it and only keep current state.
2. The websocket subscribe frame accepts `sinceOrdinal`, stores it, and echoes it, but does not use it to filter or replay events.
3. Local ordinal assignment starts from zero per fresh hub and does not consult `HydrationStore.Cursor`, so a non-bus hub with a persistent store can emit low ordinals after restart.
4. The default projection policy advances the cursor even when projections fail, which can permanently skip timeline state unless applications opt into fail-fast behavior.
5. Watermill consumer decode errors are silently acknowledged by returning `nil` from `handleMessage`, which can hide malformed envelopes.
6. The framework still exposes the old `evtstream` name in topics, metadata keys, and SQLite table names.
7. The systemlab code demonstrates useful concepts, but several phase files are long and repeat the same runtime scaffolding, trace handling, websocket hooks, response construction, and check logic.

The recommended direction is to keep the core framework small, but make the contracts sharper:

- decide whether `sessionstream` is a current-state hydrator or an event-log/replay framework;
- if it is current-state only, remove or rename `asOf` and make `sinceOrdinal` explicitly advisory;
- if it should support replay, add a real per-session event/UI-event log and implement `Snapshot(asOf)` plus `Subscribe(sinceOrdinal)` semantics;
- seed local ordinals from the store cursor;
- report or dead-letter malformed bus messages;
- split systemlab phase runtime boilerplate into reusable helpers.

## Problem statement and scope

This review answers four questions for a new intern:

1. What is `sessionstream`, and how does data flow through it?
2. Which APIs are stable and clear enough to build on?
3. Which parts are obtuse, under-specified, overgrown, duplicated, or still carrying extraction-era names?
4. What cleanup work should be done first, and how should it be tested?

The review covers Go code under `pkg/`, `examples/`, and `cmd/sessionstream-systemlab/`, plus repository-level docs and CI context. It does not audit the browser-side JavaScript/CSS in detail, except where server APIs shape frontend behavior.

## Repository map

At the time of review, the Go surface area is 39 files and 6,472 total lines. The largest files are concentrated in systemlab:

```text
797 cmd/sessionstream-systemlab/phase2_lab.go
495 cmd/sessionstream-systemlab/lab_environment.go
484 cmd/sessionstream-systemlab/phase5_lab.go
455 pkg/sessionstream/transport/ws/server.go
425 examples/chatdemo/chat.go
423 cmd/sessionstream-systemlab/phase3_lab.go
350 pkg/sessionstream/hub.go
296 cmd/sessionstream-systemlab/server.go
288 pkg/sessionstream/hydration/sqlite/store.go
230 pkg/sessionstream/bus.go
```

This distribution is healthy for the core package but less healthy for the teaching app. The longest files mix setup, state, handlers, projections, trace logging, response construction, and correctness checks. For a new intern, that makes the labs harder to navigate than the framework itself.

### Top-level ownership

```text
sessionstream/
├── pkg/sessionstream/                 # core substrate interfaces and runtime
│   ├── hydration/memory/               # in-memory current-state store
│   ├── hydration/sqlite/               # SQLite current-state store
│   └── transport/ws/                   # websocket transport and fanout
├── examples/chatdemo/                  # framework-owned demo domain
├── cmd/sessionstream-systemlab/        # interactive teaching app
├── ttmp/                               # docmgr tickets and design notes
├── Makefile                            # test/build/lint/release targets
└── .github/workflows/                  # CI, lint, security, release
```

## Mental model for interns

A `sessionstream` application has four moving parts:

1. **Commands** are requests from clients or application code. They have a name, a session id, and a protobuf payload.
2. **Backend events** are the durable logical facts emitted by command handlers.
3. **Projections** turn backend events into:
   - live UI events for currently connected clients;
   - timeline entities for hydration/reconnect snapshots.
4. **Transports and stores** are adapters. The websocket server sends UI events. Hydration stores keep current timeline state.

The shortest conceptual flow is:

```text
client/app
  │
  │ Submit(ctx, sid, commandName, commandPayload)
  ▼
Hub
  │ validates command schema
  │ finds command handler
  ▼
CommandHandler(ctx, Command, Session, EventPublisher)
  │ publishes one or more backend events
  ▼
EventPublisher
  ├─ local path: assign local ordinal and project immediately
  └─ bus path: serialize event to Watermill; consumer later assigns ordinal
        │
        ▼
projectAndApply
  │ loads current TimelineView
  │ runs UIProjection
  │ runs TimelineProjection
  │ applies timeline entities to HydrationStore
  │ fans UI events to UIFanout
  ▼
websocket clients + reconnect snapshots
```

A simplified pseudocode version of `Hub.Submit` is:

```go
func Submit(ctx, sid, name, payload) error {
    validate session id
    validate command payload type against SchemaRegistry
    handler := commandRegistry[name]
    session := sessions.GetOrCreate(ctx, sid)
    publisher := localEventPublisher or watermillEventPublisher
    return handler(ctx, Command{Name: name, SessionId: sid, Payload: payload}, session, publisher)
}
```

A simplified pseudocode version of `projectAndApply` is:

```go
func projectAndApply(ctx, event) ([]UIEvent, error) {
    session := sessions.GetOrCreate(ctx, event.SessionId)
    view := store.View(ctx, event.SessionId)       // current timeline before event

    uiEvents, uiErr := uiProjection.Project(ctx, event, session, view)
    entities, tlErr := timelineProjection.Project(ctx, event, session, view)

    if policy == Fail && (uiErr != nil || tlErr != nil) {
        return nil, firstProjectionError
    }

    if tlErr != nil {
        entities = nil                            // default policy still advances cursor
    }
    store.Apply(ctx, event.SessionId, event.Ordinal, entities)

    if uiErr == nil {
        fanout.PublishUI(ctx, event.SessionId, event.Ordinal, uiEvents)
    }
    return uiEvents, nil
}
```

That pseudocode is the most important piece of the framework. Most architecture questions reduce to what guarantees this function makes.

## Current-state architecture with file references

### 1. Core domain types

`pkg/sessionstream/types.go` is intentionally small. It defines the identifiers and data shapes that cross every boundary:

- `SessionId` is the universal routing key (`types.go:6`).
- `ConnectionId` identifies a transport-level connection (`types.go:9`).
- `Command` includes `Name`, `Payload`, `SessionId`, and `ConnectionId` (`types.go:12`).
- `Event` includes `Name`, `Payload`, `SessionId`, and `Ordinal` (`types.go:22`).
- `Session` includes `Id` and arbitrary `Metadata` (`types.go:31`).

The API is easy to read, but two things are worth noting for new engineers:

- `Command.ConnectionId` exists but is not populated by `Hub.Submit`; the current websocket transport does not decode command frames into hub submissions. It mainly manages subscriptions and fanout.
- `Payload` is `proto.Message`, and the examples use `structpb.Struct` rather than generated domain protobufs. That keeps demos flexible but weakens compile-time schema clarity.

### 2. Schema registry

`pkg/sessionstream/schema.go` stores prototype protobuf messages in four maps:

```text
commands: command name -> proto prototype
events: backend event name -> proto prototype
uiEvents: UI event name -> proto prototype
entities: timeline entity kind -> proto prototype
```

Important functions:

- `RegisterCommand`, `RegisterEvent`, `RegisterUIEvent`, and `RegisterTimelineEntity` call a shared `register` helper (`schema.go:25-41`, `schema.go:78-101`).
- `DecodeCommandJSON` instantiates a registered command prototype and unmarshals JSON into it (`schema.go:61-72`).
- `MarshalProtoJSON` centralizes protobuf JSON output (`schema.go:74-82`).
- `instantiate` calls `prototype.ProtoReflect().New().Interface()` (`schema.go:113-124`).

The registry works, but its contract should be documented more explicitly:

- Are names global across a hub or scoped by module?
- Should callers register generated protobuf types instead of `structpb.Struct`?
- Are returned schema prototypes immutable? Today `CommandSchema`, `EventSchema`, and friends return the registered prototype directly.

### 3. Hub and local execution path

`pkg/sessionstream/hub.go` owns the framework's main runtime state. `NewHub` initializes default components: a fresh schema registry, a no-op hydration store, a session registry, a command registry, `ProjectionErrorPolicyAdvance`, and an empty `localOrdinal` map (`hub.go:90-105`).

The local path is used when no event bus is configured:

- `Submit` validates the command and dispatches it (`hub.go:146-160`).
- `dispatch` looks up the handler and gets or creates the session (`hub.go:230-240`).
- `publisher` returns `localEventPublisher` unless `h.bus != nil` (`hub.go:242-247`).
- `localEventPublisher.Publish` validates the event, assigns an ordinal with `nextLocalOrdinal`, and calls `projectAndApply` (`hub.go:253-269`).
- `nextLocalOrdinal` increments an in-memory map (`hub.go:334-339`).

This is the right shape for a minimal framework. The important caveat is that local ordinals are not seeded from the hydration store cursor. If a persistent store is reused after a process restart, the next locally emitted UI event can carry ordinal `1` even though the store cursor is already higher. The store may preserve the cursor because it applies `max(existing, ord)`, but the live event ordinal can still be misleading.

### 4. Watermill bus path

`pkg/sessionstream/bus.go` integrates an optional Watermill publisher/subscriber pair:

- `WithEventBus` validates the publisher and subscriber and stores a `busConfig` (`bus.go:101-128`).
- `watermillEventPublisher.Publish` validates the event, marshals the payload to proto JSON, wraps it in an `eventEnvelope`, assigns Watermill metadata, and publishes it (`bus.go:139-185`).
- `decodeEventEnvelope` unmarshals the envelope and instantiates the registered event payload type (`bus.go:188-214`).

`pkg/sessionstream/consumer.go` consumes bus messages:

- `newEventConsumer` creates an `OrdinalAssigner` whose cursor source is `h.store.Cursor(ctx, sid)` (`consumer.go:18-27`).
- `consume` subscribes, then handles messages one at a time (`consumer.go:46-78`).
- `handleMessage` decodes the envelope, assigns an ordinal, notifies the observer, and calls `projectAndApply` (`consumer.go:80-99`).

The bus path is more restart-safe than the local path because its ordinal assigner consults the store cursor on first use. However, malformed messages are currently acknowledged silently: `handleMessage` returns `nil` if `decodeEventEnvelope` fails (`consumer.go:81-84`). That behavior should either be documented as a deliberate poison-message policy or changed to emit a structured error/dead-letter observation.

### 5. Ordinal assignment

`pkg/sessionstream/ordinals.go` assigns monotonically increasing per-session ordinals. It has two inputs:

- the current cursor loaded from the store on first use (`ordinals.go:30-42`);
- optional stream metadata from `MetadataKeyStreamID`, `xid`, or `redis_xid` (`ordinals.go:51-58`, `ordinals.go:60-68`).

For Redis-style IDs, `DeriveOrdinalFromStreamID` parses `<milliseconds>-<sequence>` and returns `ms*1_000_000 + seq` (`ordinals.go:70-84`). This works as a stable ordering proxy, but the unit and overflow assumptions should be documented in public API docs if users are expected to interpret ordinals.

### 6. Projection model

`pkg/sessionstream/projection.go` defines the framework's most important extension interfaces:

```go
type UIProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]UIEvent, error)
}

type TimelineProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]TimelineEntity, error)
}
```

Both projections receive the same pre-event `TimelineView`. This is useful because the UI projection and timeline projection can make decisions based on current state. It is also important because UI projection does not see timeline changes emitted by the same backend event unless it recomputes them itself.

The actual execution order in `projectAndApply` is:

1. load session;
2. load current view;
3. run UI projection;
4. run timeline projection;
5. depending on error policy, possibly discard timeline entities;
6. apply entities and cursor;
7. publish UI events.

This order should be treated as part of the public contract. If changed later, examples may behave differently.

### 7. Hydration stores

The core interface is in `pkg/sessionstream/hydration.go`:

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

The in-memory store (`pkg/sessionstream/hydration/memory/store.go`) keeps one map per session. `Apply` upserts or deletes entities and advances the cursor if `ord > cursor` (`memory/store.go:34-49`). `Snapshot` returns the current entities sorted by kind and id (`memory/store.go:52-70`). `View` wraps the current snapshot in a small read-only map (`memory/store.go:72-77`).

The SQLite store (`pkg/sessionstream/hydration/sqlite/store.go`) stores current state in two tables:

- `evtstream_sessions(session_id, cursor)` (`sqlite/store.go:197-202`);
- `evtstream_entities(session_id, kind, entity_id, payload_json)` (`sqlite/store.go:203-209`).

`Apply` runs in a transaction, deletes tombstones, upserts entities, and updates the cursor with `max(existing, excluded)` semantics (`sqlite/store.go:71-118`). `Snapshot` selects current entities and decodes each payload using the schema registry (`sqlite/store.go:120-166`).

Both stores ignore the `asOf` parameter. They are current-state stores, not historical stores.

### 8. Websocket transport

`pkg/sessionstream/transport/ws/server.go` is both an `http.Handler` and a `sessionstream.UIFanout` (`server.go:116-117`). It owns connections and a reverse index from session id to connection ids:

```text
Server
├── conns: ConnectionId -> connection
└── bySession: SessionId -> set(ConnectionId)
```

A connection has:

- a Gorilla websocket;
- a buffered send channel of 128 messages;
- a subscription map keyed by session id (`server.go:77-91`).

The protocol frames include:

- server-to-client: `hello`, `subscribed`, `unsubscribed`, `snapshot`, `ui-event`, `error`, `pong`;
- client-to-server: `subscribe`, `unsubscribe`, `ping`.

Subscribe flow:

```text
client sends {type:"subscribe", sessionId:"s", sinceOrdinal:"..."}
server parses sinceOrdinal
server loads current snapshot from SnapshotProvider
server sends snapshot
server records subscription in connection and bySession
server sends subscribed acknowledgement
```

The implementation is in `handleClientFrame` (`server.go:238-292`). Live fanout is implemented by `PublishUI`, which finds subscribed connections and enqueues one `ui-event` envelope per UI event (`server.go:170-193`).

The transport is simple and useful for labs. The biggest ambiguity is `sinceOrdinal`: the server parses and stores it (`server.go:245-267`) but does not use it to filter snapshot entities, replay missed UI events, or suppress old live messages. This needs to be either implemented or renamed/documented.

### 9. Chat demo example

`examples/chatdemo/chat.go` is a framework-owned demonstration domain. It registers chat command/event/UI/entity names (`chat.go:13-27`, `chat.go:72-96`), installs handlers and projections into a hub (`chat.go:98-125`), and simulates token streaming with a goroutine (`chat.go:188-229`).

The demo is valuable because it shows how a real application might use the framework:

- command handler publishes accepted/started/delta/finished events;
- UI projection maps backend events to client-facing events (`chat.go:289-310`);
- timeline projection upserts a `ChatMessage` entity (`chat.go:312-359`).

For onboarding, this file is probably the best starting point after reading `pkg/sessionstream/hub.go`.

### 10. Systemlab teaching app

`cmd/sessionstream-systemlab` is an interactive teaching app. It contains five phases:

- Phase 1: local command -> event -> projections -> hydration.
- Phase 2: Watermill ordering and per-session ordinals.
- Phase 3: websocket subscription and reconnect snapshots.
- Phase 4: chat demo integration.
- Phase 5: persistence and restart behavior.

`lab_environment.go` initializes all phases in `newLabEnvironment` (`lab_environment.go:83-152`). `server.go` exposes HTTP routes for each phase (`server.go:26-47`).

The app is useful but currently carries the highest maintainability cost. Phase 2 is 797 lines and Phase 5 is 484 lines. The same shapes repeat in each phase:

- register schemas;
- create store;
- create websocket server;
- create hub;
- register command/UI/timeline projections;
- append traces;
- run action switch;
- build response;
- compute checks;
- clone response state.

That repetition is acceptable during prototyping, but it should be extracted before the teaching app grows further.

## Detailed findings and cleanup recommendations

### Finding 1 — `asOf` in `HydrationStore.Snapshot` is an API promise the stores do not keep

**Problem:** The interface says `Snapshot(ctx, sid, asOf)` but both stores ignore `asOf`. This makes the API look like it supports historical snapshots even though the implementations only store current state.

**Where to look:**

- `pkg/sessionstream/hydration.go:6-10`
- `pkg/sessionstream/hydration/memory/store.go:52`
- `pkg/sessionstream/hydration/sqlite/store.go:120`
- `pkg/sessionstream/hub.go:163-168`

**Example:**

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

The memory and SQLite methods both name the parameter `_ uint64`, confirming that it is not used.

**Why it matters:** New users will assume `Snapshot(..., asOf=10)` returns the view as of ordinal 10. That is not true. If clients use this during reconnect, they may believe they can recover a precise historical state when they only get the latest state.

**Cleanup options:**

Option A: make current-state semantics explicit.

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

Option B: implement historical semantics.

```text
evtstream_events/sessionstream_events
  session_id
  ordinal
  event_name
  payload_json

timeline_entity_versions
  session_id
  kind
  entity_id
  ordinal
  tombstone
  payload_json
```

Then `Snapshot(asOf)` replays or queries the latest entity version with `ordinal <= asOf`.

**Recommendation:** Use Option A unless replay is a near-term requirement. It is better to be honest and small than to expose a historical API without storage support.

### Finding 2 — `sinceOrdinal` is accepted by websocket subscribe but does not affect behavior

**Problem:** The websocket protocol accepts `sinceOrdinal`, stores it in the subscription, and echoes it in the `subscribed` frame. It does not use it to filter snapshots, replay missed UI events, or suppress older events.

**Where to look:**

- `pkg/sessionstream/transport/ws/server.go:98-113` for frame fields.
- `pkg/sessionstream/transport/ws/server.go:238-280` for subscribe handling.
- `pkg/sessionstream/transport/ws/server.go:170-193` for fanout.

**Example:**

```go
since, err := parseUint(frame.SinceOrdinal)
// ...
snap, err := s.snapshots.Snapshot(ctx, sid)
// ...
c.subs[sid] = subscription{sinceOrdinal: since}
```

`PublishUI` later sends every new UI event to every subscribed connection. It never reads `subscription.sinceOrdinal`.

**Why it matters:** The name `sinceOrdinal` implies replay or catch-up semantics. A client reconnecting with `sinceOrdinal=42` might expect only changes after 42, but the server always sends a full current snapshot plus future live events. Full snapshot plus future live events is a valid strategy, but the field name is misleading.

**Cleanup sketch:**

If the framework remains snapshot-only:

```go
type clientFrame struct {
    Type      string `json:"type"`
    SessionID string `json:"sessionId,omitempty"`
    // LastSeenOrdinal is informational; the server always replies with current snapshot.
    LastSeenOrdinal string `json:"lastSeenOrdinal,omitempty"`
}
```

If replay is desired:

```go
func subscribe(sid, since) {
    snap := store.Snapshot(ctx, sid, since)
    missed := eventLog.UIEventsAfter(ctx, sid, since)
    send(snapshot)
    for _, ev := range missed { send(uiEvent) }
    markSubscribed(sid, since)
}
```

**Recommendation:** Rename/document the field now, or add a ticket for actual replay. Do not leave it looking implemented when it is advisory.

### Finding 3 — local ordinals are not seeded from persistent store cursor

**Problem:** The local publishing path uses `Hub.localOrdinal`, an in-memory map initialized empty in `NewHub`. It does not consult `HydrationStore.Cursor`. With a persistent store, a new hub can emit UI events with ordinals lower than the stored cursor after restart.

**Where to look:**

- `pkg/sessionstream/hub.go:36` for `localOrdinal`.
- `pkg/sessionstream/hub.go:90-105` for initialization.
- `pkg/sessionstream/hub.go:253-269` for local publish.
- `pkg/sessionstream/hub.go:334-339` for `nextLocalOrdinal`.
- `cmd/sessionstream-systemlab/phase5_lab.go:76-176` for persistent runtime setup.

**Example:**

```go
func (h *Hub) nextLocalOrdinal(sid SessionId) uint64 {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.localOrdinal[sid]++
    return h.localOrdinal[sid]
}
```

**Why it matters:** Phase 5 is specifically about persistence and restart. The SQLite store preserves the cursor, but local ordinals start from one after a hub restart. That can confuse clients and correctness checks that assume ordinals are monotonic across restart.

**Cleanup sketch:**

Make local assignment use the same cursor-seeded logic as the bus path:

```go
type Hub struct {
    localOrdinals *OrdinalAssigner
}

func NewHub(...) *Hub {
    h.localOrdinals = NewOrdinalAssigner(func(ctx context.Context, sid SessionId) (uint64, error) {
        return h.store.Cursor(ctx, sid)
    })
}

func (p localEventPublisher) Publish(ctx context.Context, ev Event) error {
    ord, err := p.hub.localOrdinals.Next(ctx, ev.SessionId, nil)
    if err != nil { return err }
    ev.Ordinal = ord
    _, err = p.hub.projectAndApply(ctx, ev)
    return err
}
```

Be careful that the cursor function captures the final store after `WithHydrationStore` options are applied.

**Test to add:** Create a SQLite store, submit one event, close/reopen the store, create a new hub with the same store, submit another event, and assert the second UI event ordinal is greater than the first stored cursor.

### Finding 4 — default projection error policy can advance cursors while dropping state

**Problem:** `ProjectionErrorPolicyAdvance` is the default. If the timeline projection fails, `projectAndApply` sets `entitiesToApply = nil` but still calls `store.Apply` with the event ordinal. That advances the cursor without applying the intended timeline update.

**Where to look:**

- `pkg/sessionstream/hub.go:12-19` for policy docs.
- `pkg/sessionstream/hub.go:90-105` for default policy.
- `pkg/sessionstream/hub.go:272-317` for projection/apply logic.
- `pkg/sessionstream/hub_test.go:84` for the test that confirms advance behavior.

**Example:**

```go
entitiesToApply := entities
if tlErr != nil {
    entitiesToApply = nil
}
if err := h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entitiesToApply); err != nil {
    return nil, err
}
```

**Why it matters:** In event-driven systems, cursor advancement usually means "all effects up to this point are reflected." Here it can mean "we saw the event, but skipped its timeline effects." That may be acceptable for lossy projections, but it should not be the surprising default for durable hydration.

**Cleanup options:**

- Make `ProjectionErrorPolicyFail` the default for framework correctness.
- Keep `Advance`, but add an error observer hook and write projection failures to a dead-letter/status channel.
- Split UI and timeline policies so a transient UI fanout issue does not imply the same behavior as a durable timeline projection issue.

**API sketch:**

```go
type ProjectionErrorObserver interface {
    ProjectionFailed(ctx context.Context, ev Event, kind string, err error)
}

type ProjectionPolicies struct {
    UI       ProjectionErrorPolicy
    Timeline ProjectionErrorPolicy
}
```

### Finding 5 — malformed Watermill envelopes are silently acknowledged

**Problem:** `eventConsumer.handleMessage` returns `nil` when `decodeEventEnvelope` fails. Because `consume` calls `msg.Ack()` after a nil error, malformed messages disappear without an error path.

**Where to look:**

- `pkg/sessionstream/consumer.go:80-84`
- `pkg/sessionstream/consumer.go:64-76`
- `pkg/sessionstream/bus.go:188-214`

**Example:**

```go
ev, err := decodeEventEnvelope(c.hub.reg, msg.Payload)
if err != nil {
    return nil
}
```

**Why it matters:** This hides integration bugs and schema drift. A publisher can send an unknown event name or invalid JSON and the consumer will ack it. Operators will not know that data was dropped unless an external Watermill logger catches it elsewhere.

**Cleanup sketch:**

Add a policy:

```go
type DecodeErrorPolicy int
const (
    DecodeErrorAckAndObserve DecodeErrorPolicy = iota
    DecodeErrorNack
    DecodeErrorDeadLetter
)
```

At minimum, notify an observer:

```go
if err != nil {
    observer.DecodeFailed(ctx, newBusRecord(msg, topic), err)
    return nil // only if ack-and-observe is deliberate
}
```

### Finding 6 — old `evtstream` names remain in public constants and SQLite schema

**Problem:** The repository has been extracted as `sessionstream`, but bus constants, metadata keys, and SQLite table names still say `evtstream`.

**Where to look:**

- `pkg/sessionstream/bus.go:15-22`
- `pkg/sessionstream/bus.go:39`
- `pkg/sessionstream/hydration/sqlite/store.go:61-62`
- `pkg/sessionstream/hydration/sqlite/store.go:90-112`
- `pkg/sessionstream/hydration/sqlite/store.go:197-210`

**Example:**

```go
DefaultEventBusTopic = "evtstream.events"
MetadataKeyEventName = "evtstream_event_name"
```

**Why it matters:** Names are architecture. A new intern will wonder whether `evtstream` is a different project, a compatibility layer, or the old name. For a framework repository, stale names leak extraction history into the public API.

**Cleanup sketch:**

- For new public APIs, use `sessionstream.events`, `sessionstream_event_name`, and `sessionstream_sessions`/`sessionstream_entities`.
- If backward compatibility is required, support explicit aliases through options rather than default names.
- Since the repository is still bootstrap-mode, consider a breaking cleanup before the first stable release.

### Finding 7 — websocket server needs production-readiness boundaries documented

**Problem:** The websocket server is good for labs, but defaults are permissive and incomplete for production. It allows all origins, has no read limit, no read/write deadlines, no ping/pong deadlines, no close frame handling policy, and a fixed send buffer size.

**Where to look:**

- `pkg/sessionstream/transport/ws/server.go:120-136` for default upgrader with `CheckOrigin: true`.
- `pkg/sessionstream/transport/ws/server.go:144-168` for connection setup.
- `pkg/sessionstream/transport/ws/server.go:214-236` for read loop.
- `pkg/sessionstream/transport/ws/server.go:296-302` for write loop.
- `pkg/sessionstream/transport/ws/server.go:367-391` for send buffer behavior.

**Example:**

```go
upgrader: websocket.Upgrader{
    CheckOrigin: func(_ *http.Request) bool { return true },
},
```

**Why it matters:** If users see `transport/ws` under `pkg/`, they may assume it is production-ready. The permissive origin default is especially easy to miss.

**Cleanup sketch:**

```go
type Config struct {
    CheckOrigin func(*http.Request) bool
    SendBuffer int
    ReadLimit int64
    WriteTimeout time.Duration
    PongTimeout time.Duration
    Logger Logger
}
```

Make unsafe defaults explicit:

```go
func NewServer(snapshots SnapshotProvider, opts ...Option) (*Server, error) {
    // If no origin policy is provided, allow only same-origin by default.
}
```

If the current behavior is intentionally lab-first, document it in package docs and README: "The websocket transport is a reference adapter, not a hardened edge server."

### Finding 8 — websocket transport and core hub are not wired for incoming command frames

**Problem:** `pkg/sessionstream/transport/transport.go` defines generic `IncomingCommand` and `OutgoingMessage`, and `Command.ConnectionId` exists in core types, but the websocket server only handles subscribe/unsubscribe/ping. It does not decode command frames and submit them to the hub.

**Where to look:**

- `pkg/sessionstream/transport/transport.go:10-27`
- `pkg/sessionstream/types.go:12-18`
- `pkg/sessionstream/transport/ws/server.go:238-292`
- `pkg/sessionstream/hub.go:146-160`

**Why it matters:** This creates an unclear API story. Is `transport/ws` only a fanout/subscription adapter? Or is it intended to be a full bidirectional command transport? Both designs are valid, but the current names suggest more than the implementation provides.

**Cleanup options:**

Option A: make websocket explicitly fanout-only.

```go
package wsfanout
// Server handles snapshot subscribe/unsubscribe and live UI fanout only.
```

Option B: add command ingress.

```go
type CommandSubmitter interface {
    Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error
}

type CommandDecoder interface {
    DecodeCommandJSON(name string, payload json.RawMessage) (proto.Message, error)
}
```

Then support:

```json
{"type":"command","sessionId":"s-1","name":"ChatStartInference","payload":{"prompt":"hello"}}
```

### Finding 9 — store view implementation is duplicated between memory and SQLite stores

**Problem:** Both stores implement a private `entityKey`, `view`, `newView`, `Get`, `List`, and clone logic. The code is short, but duplicated enough to drift.

**Where to look:**

- `pkg/sessionstream/hydration/memory/store.go:99-145`
- `pkg/sessionstream/hydration/sqlite/store.go:219-274`

**Why it matters:** The view behavior is part of the projection contract. If memory and SQLite sorting or cloning behavior diverges, tests may pass against one store and fail against the other.

**Cleanup sketch:**

Add a small unexported or exported helper in the core package:

```go
func NewSnapshotView(snap Snapshot) TimelineView
func CloneTimelineEntity(entity TimelineEntity) TimelineEntity
func SortTimelineEntities(entities []TimelineEntity)
```

Then both stores use the same implementation.

### Finding 10 — session registry calls the metadata factory while holding a global lock

**Problem:** `sessionRegistry.GetOrCreate` calls `factory(ctx, sid)` while holding `r.mu.Lock()`.

**Where to look:**

- `pkg/sessionstream/session_registry.go:32-54`

**Example:**

```go
r.mu.Lock()
defer r.mu.Unlock()
// ...
if r.factory != nil {
    metadata, err = r.factory(ctx, sid)
}
```

**Why it matters:** If the factory performs I/O or blocks, all other new sessions are blocked. The current default factory is nil, so this is not a bug today, but the API invites users to install arbitrary metadata creation.

**Cleanup sketch:**

Use a per-session singleflight or reserve placeholder:

```go
if existing := get(sid); existing != nil { return existing }
metadata, err := factory(ctx, sid) // outside global lock
lock
if someoneCreated := byID[sid]; someoneCreated != nil { return someoneCreated }
byID[sid] = &Session{Id: sid, Metadata: metadata}
```

If duplicate factory calls are unacceptable, use `singleflight.Group` keyed by session id.

### Finding 11 — schema registry returns mutable prototypes directly

**Problem:** The `CommandSchema`, `EventSchema`, `UIEventSchema`, and `TimelineEntitySchema` methods return the stored prototype message directly. Callers can mutate the prototype after registration or after lookup.

**Where to look:**

- `pkg/sessionstream/schema.go:45-59`
- `pkg/sessionstream/schema.go:103-111`

**Why it matters:** Protobuf messages are mutable. A caller that gets a schema and modifies fields could affect later instantiation or validation in surprising ways. For empty generated messages this is less visible, but `structpb.Struct` prototypes can carry fields.

**Cleanup sketch:**

- Clone on register to isolate from caller-owned values.
- Return clones or descriptors from lookup methods.

```go
func (r *SchemaRegistry) register(..., msg proto.Message) error {
    m[name] = proto.Clone(msg)
}

func (r *SchemaRegistry) CommandSchema(name string) (protoreflect.MessageDescriptor, bool) {
    msg, ok := r.lookup(...)
    if !ok { return nil, false }
    return msg.ProtoReflect().Descriptor(), true
}
```

### Finding 12 — systemlab phase files are too long and repeat runtime boilerplate

**Problem:** `cmd/sessionstream-systemlab` is the least navigable part of the repository. Phase 2 is 797 lines; Phase 5 is 484 lines; Phase 3 is 423 lines; `lab_environment.go` is 495 lines. The files combine multiple responsibilities.

**Where to look:**

- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `cmd/sessionstream-systemlab/lab_environment.go`

**Repeated patterns:**

- `newPhaseNState` registers schemas, stores, hubs, websocket hooks, command handlers, and projections.
- `RunPhaseN` normalizes inputs, switches on actions, mutates state, and builds response.
- `appendPhaseNTrace` clones details and increments step numbers.
- `buildPhaseNResponse` snapshots state and computes checks.
- `clonePhaseNRunResponse` duplicates response copying.

**Why it matters:** Systemlab is supposed to teach the framework. If the teaching implementation is harder to read than the framework, interns may learn the wrong abstraction boundaries.

**Cleanup sketch:**

Introduce a small phase runtime toolkit:

```text
cmd/sessionstream-systemlab/
├── phase_runtime.go        # common trace, response, ws hooks helpers
├── phase_runtime_test.go
├── phase1_lab.go
├── phase2_lab.go
├── phase3_lab.go
├── phase4_lab.go
└── phase5_lab.go
```

Possible helper types:

```go
type traceLog struct {
    mu sync.Mutex
    entries []traceEntry
}
func (t *traceLog) Append(kind, message string, details map[string]any)
func (t *traceLog) Snapshot() []traceEntry

type phaseHubRuntime struct {
    Reg *sessionstream.SchemaRegistry
    Store sessionstream.HydrationStore
    Hub *sessionstream.Hub
    WS *wstransport.Server
}
```

The target is not fewer lines for its own sake. The target is that each phase file reads like a lesson rather than a framework bootstrap script.

### Finding 13 — examples rely on `structpb.Struct`, which hides the intended schema-first story

**Problem:** The chat demo and systemlab phases register every command/event/UI/entity as `&structpb.Struct{}`. This makes the examples flexible, but it makes the schema registry look like a stringly typed JSON registry rather than a protobuf schema registry.

**Where to look:**

- `examples/chatdemo/chat.go:72-96`
- `cmd/sessionstream-systemlab/lab_environment.go:89-98`
- `cmd/sessionstream-systemlab/phase2_lab.go:91-98`
- `cmd/sessionstream-systemlab/phase3_lab.go:58-65`
- `cmd/sessionstream-systemlab/phase5_lab.go:80-83`

**Why it matters:** A new user may copy the example and never define real protobuf schemas. That may be acceptable for rapid prototyping, but the framework's imports and validation imply protobuf is meant to provide stronger contracts.

**Cleanup sketch:**

Add one tiny generated protobuf example, even if the labs keep `structpb.Struct`:

```proto
message StartInferenceCommand { string prompt = 1; }
message InferenceStartedEvent { string message_id = 1; string prompt = 2; }
message ChatMessageEntity { string message_id = 1; string role = 2; string content = 3; }
```

Then show:

```go
reg.RegisterCommand(chat.CommandStartInference, &chatpb.StartInferenceCommand{})
```

### Finding 14 — `transport/transport.go` is currently an orphan abstraction

**Problem:** The generic `Transport` interface is defined but not implemented by the websocket server and not referenced by the hub. This makes it unclear whether it is a planned abstraction or leftover design scaffolding.

**Where to look:**

- `pkg/sessionstream/transport/transport.go:10-27`
- `pkg/sessionstream/transport/ws/server.go:116-117`

**Why it matters:** Orphan abstractions are confusing in small frameworks. They invite users to implement interfaces that the runtime never consumes.

**Cleanup options:**

- Delete it until a second transport exists.
- Move it to an internal design doc.
- Wire it into the hub and websocket implementation.

### Finding 15 — current tests cover the happy paths but miss several contract edges

**Problem:** Tests exist across all main packages, but coverage output shows important public behavior untested or lightly tested. The generated coverage run reported:

```text
cmd/sessionstream-systemlab coverage: 64.4%
examples/chatdemo coverage: 70.6%
pkg/sessionstream coverage: 72.4%
pkg/sessionstream/hydration/memory coverage: 64.1%
pkg/sessionstream/hydration/sqlite coverage: 48.9%
pkg/sessionstream/transport/ws coverage: 62.7%
```

Many server handlers in `cmd/sessionstream-systemlab/server.go` are at 0% in the coverage report, and SQLite view methods are also at 0%.

**Why it matters:** The missing tests correspond to contract ambiguity: websocket subscribe semantics, restart ordinals, `Snapshot(asOf)`, error policies, malformed bus messages, and production transport behavior.

**High-value tests to add:**

1. Local persistent ordinal resume test.
2. `Snapshot(asOf)` test that either proves it is unsupported or validates historical behavior.
3. Websocket subscribe with non-zero `sinceOrdinal` test that asserts the intended semantics.
4. Malformed bus message test that asserts observer/nack/dead-letter behavior.
5. Projection timeline failure test that asserts cursor and entity state for both policies.
6. SQLite `View`, `Get`, `List`, and tombstone tests matching memory store behavior.

## Proposed cleanup plan

### Phase 1 — Clarify public contracts before adding features

Goal: remove ambiguity without large architecture changes.

Tasks:

1. Decide and document whether hydration is current-state only.
2. If current-state only, remove or deprecate `asOf` from `HydrationStore.Snapshot`.
3. Document websocket subscribe as full-snapshot-plus-live, or rename `sinceOrdinal` to `lastSeenOrdinal`.
4. Mark websocket server as reference/lab adapter unless hardened.
5. Rename old `evtstream` constants/tables if compatibility constraints allow it.

Validation:

```bash
go test ./...
go test ./pkg/sessionstream/... -count=1
```

### Phase 2 — Fix correctness-sensitive ordinal and error behavior

Goal: make cursor/ordinal behavior reliable across restarts.

Tasks:

1. Replace `Hub.localOrdinal` with cursor-seeded `OrdinalAssigner`.
2. Add a persistent-store local-publisher restart test.
3. Add a decode error policy or observer for Watermill consumer.
4. Reconsider `ProjectionErrorPolicyAdvance` as the default.

Validation pseudocode:

```go
func TestLocalPublisherSeedsOrdinalFromStoreCursor(t *testing.T) {
    store := sqliteStore(tempDB)
    hub1 := newHub(store)
    submit(hub1, "s", "first")
    require.Equal(1, cursor(store, "s"))

    hub2 := newHub(reopenStore(tempDB))
    captureFanout := newRecordingFanout()
    submit(hub2, "s", "second")

    require.Equal(2, captureFanout.lastOrdinal("s"))
    require.Equal(2, cursor(store, "s"))
}
```

### Phase 3 — Reduce duplication in stores and systemlab

Goal: make future examples and stores cheaper to maintain.

Tasks:

1. Extract shared timeline view helpers.
2. Extract systemlab trace log helper.
3. Extract systemlab websocket hook builder.
4. Split Phase 2 into smaller files:
   - `phase2_runtime.go`
   - `phase2_actions.go`
   - `phase2_checks.go`
   - `phase2_render.go`
5. Split Phase 5 similarly around runtime persistence, actions, and checks.

### Phase 4 — Decide whether to support event replay

Goal: either keep the framework current-state focused or implement durable replay semantics.

If current-state only:

- remove replay-looking parameters;
- keep stores small;
- document reconnect as full snapshot plus live tail.

If replay is required:

- add an event log or UI-event log interface;
- persist events before projection;
- implement `Snapshot(asOf)` or `UIEventsAfter(since)`;
- make websocket subscribe replay missed events after snapshot.

## Suggested intern reading path

For an intern new to the codebase, read in this order:

1. `README.md` and `AGENT.md` for repository purpose and guardrails.
2. `pkg/sessionstream/types.go` for the vocabulary.
3. `pkg/sessionstream/projection.go` and `pkg/sessionstream/hydration.go` for extension points.
4. `pkg/sessionstream/hub.go`, especially `Submit`, `localEventPublisher.Publish`, and `projectAndApply`.
5. `examples/chatdemo/chat.go` for a complete small domain.
6. `pkg/sessionstream/transport/ws/server.go` for reconnect and fanout behavior.
7. `pkg/sessionstream/hydration/memory/store.go` before SQLite.
8. `pkg/sessionstream/hydration/sqlite/store.go` for persistence details.
9. `cmd/sessionstream-systemlab/lab_environment.go` and then individual phase files.

## Glossary

- **Backend event:** A logical fact emitted by a command handler. Example: `ChatTokensDelta`.
- **UI event:** A client-facing event derived from a backend event. Example: `ChatMessageAppended`.
- **Timeline entity:** A persistent current-state object used to hydrate clients. Example: `ChatMessage`.
- **Hydration:** Loading current timeline state for a session, usually after reconnect.
- **Ordinal:** Per-session sequence number assigned to backend events as they are applied.
- **Projection:** Function that maps backend events to UI events or timeline entities.
- **Fanout:** Delivery of live UI events to subscribed clients.
- **Snapshot:** Current timeline entities plus the latest store cursor for a session.

## References

Primary files:

- `README.md`
- `AGENT.md`
- `pkg/sessionstream/types.go`
- `pkg/sessionstream/schema.go`
- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/bus.go`
- `pkg/sessionstream/consumer.go`
- `pkg/sessionstream/ordinals.go`
- `pkg/sessionstream/projection.go`
- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hydration/memory/store.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/transport/transport.go`
- `examples/chatdemo/chat.go`
- `cmd/sessionstream-systemlab/lab_environment.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `cmd/sessionstream-systemlab/server.go`

Commands run during review:

```bash
docmgr --root ttmp status --summary-only
docmgr --root ttmp ticket create-ticket --ticket SESSIONSTREAM-003 --title "Code review and architecture audit for sessionstream" --topics architecture,backend,event-streaming,framework,onboarding,code-review,cleanup
docmgr --root ttmp doc add --ticket SESSIONSTREAM-003 --doc-type design-doc --title "Sessionstream code review and architecture audit"
docmgr --root ttmp doc add --ticket SESSIONSTREAM-003 --doc-type reference --title "Investigation diary"
rg --files -g '*.go' -g '!ttmp/**' -g '!dist/**'
wc -l $(rg --files -g '*.go' -g '!ttmp/**' -g '!dist/**')
go test ./... -coverprofile=/tmp/sessionstream-cover.out
go tool cover -func=/tmp/sessionstream-cover.out
```
