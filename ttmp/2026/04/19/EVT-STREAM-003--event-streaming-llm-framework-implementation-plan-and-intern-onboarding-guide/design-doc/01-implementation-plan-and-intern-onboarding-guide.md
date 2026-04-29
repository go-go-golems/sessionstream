---
Title: Implementation Plan and Intern Onboarding Guide
Ticket: EVT-STREAM-003
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - backend
    - implementation
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md
      Note: |-
        Source-of-truth API and runtime design for the substrate.
        Primary API and runtime source of truth
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md
      Note: |-
        Reuse matrix and donor-vs-do-not-reuse guidance.
        Reuse matrix and donor-code guidance
    - Path: pinocchio/go.mod
      Note: |-
        Existing Go module with Watermill dependency; recommended initial implementation home.
        Existing Go module and Watermill dependency baseline
    - Path: pinocchio/pkg/persistence/chatstore/timeline_store.go
      Note: |-
        Existing hydration-store-like contract and implementations to adapt.
        Current hydration-store-like contract to adapt
    - Path: pinocchio/pkg/webchat/connection_pool.go
      Note: |-
        Existing websocket fan-out/backpressure implementation worth reusing internally.
        Reusable websocket backpressure/fan-out pattern
    - Path: pinocchio/pkg/webchat/stream_backend.go
      Note: |-
        Existing bus seam and subscriber construction ideas.
        Application-owned bus seam and subscriber construction
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: |-
        Existing ordinal/stream-id logic worth carrying over.
        Consumption-time ordering and stream-id-derived sequence logic
    - Path: pinocchio/pkg/webchat/timeline_registry.go
      Note: Counterexample showing why global projection registries should not survive
ExternalSources: []
Summary: Implementation blueprint for the event-streaming LLM framework aimed at a new engineer joining the work. Explains the architecture, donor code, target package layout, runtime flows, file-by-file implementation plan, testing strategy, and first execution slices. Recommends building the first implementation in pinocchio/pkg/evtstream while keeping the package layout extraction-friendly.
LastUpdated: 2026-04-19T20:58:00-04:00
WhatFor: Turn EVT-STREAM-002's clean-room architecture into an actionable implementation plan that a new intern can follow without having to reverse-engineer the repository history.
WhenToUse: When starting the first implementation of the reusable event-streaming framework, onboarding a new contributor, planning milestones, or reviewing where donor code from pinocchio/webchat should and should not be reused.
---


# Implementation Plan and Intern Onboarding Guide

## Executive Summary

This ticket is the **implementation bridge** between the clean-room architecture in EVT-STREAM-002 and the actual code that still needs to be written. The two key source documents already exist:

1. the clean-room technical architecture, which defines the target API and runtime model, and
2. the reuse analysis against `pinocchio/pkg/webchat`, which explains which parts of the old system are good donors and which architectural habits must not be carried forward.

A new engineer should read this document as the **practical build manual**. It explains what the framework is, what abstractions are core, where the donor code lives, what files should be created first, what runtime invariants must hold, and how to proceed phase by phase without accidentally rebuilding “webchat with more knobs” instead of the intended generic substrate.

The main recommendation in this guide is:

- implement the first version in **`pinocchio/pkg/evtstream`**,
- keep the package layout neutral enough that it can later be extracted to a standalone module,
- treat `pinocchio/pkg/webchat` as a **donor** and later as a **consumer/backend example**,
- keep the substrate centered on a single canonical identity, **`SessionId`**,
- make **backend events** the canonical internal stream,
- run **UI projection** and **timeline projection** as sibling consumers of that canonical stream.

If you remember only one sentence, remember this one:

> The framework is not a websocket server and it is not a chat app; it is a reusable **event-streaming substrate** that accepts typed commands, carries typed backend events over an application-owned Watermill bus, and projects them into both live UI updates and durable hydration state.

---

## Problem Statement

The repository now has enough design material to implement the new framework, but not enough code. That creates a common onboarding failure mode: a new engineer opens the repo, sees `pinocchio/pkg/webchat`, sees the clean-room doc, sees an older refactor analysis, and has no obvious answer to three practical questions:

1. **What exactly is the system we are trying to build?**
2. **What existing code is safe to copy or adapt?**
3. **What files do I create first, and in what order?**

That confusion is amplified by the fact that current webchat already contains good ideas and bad ideas mixed together:

- good: transport/core separation, subscriber seams, idle eviction, backpressure handling, stream-id-based ordering,
- bad for the new substrate: dual ids, SEM-first projection chain, chat-specific command APIs, package-global projection registries.

This guide solves that problem by turning the architectural intent into a **specific implementation plan**.

### In scope

This document covers:

- the mental model of the new framework,
- the most important source files to read,
- the recommended initial code location,
- the package and file layout to implement,
- the runtime flows that must exist,
- a phased implementation plan,
- a testing strategy,
- the main pitfalls to avoid.

### Out of scope

This document does **not** attempt to fully resolve all open questions from EVT-STREAM-002. In particular, these remain partially open and should not block the first implementation slices:

- final liveness/tick protocol,
- final `SessionId` allocation strategy,
- whether the TypeScript client ships in the first milestone,
- whether the framework is eventually extracted into its own standalone repository/module.

The goal here is not to remove every future decision. The goal is to let an intern build the first correct substrate slices **now**, without creating architecture debt that fights the existing design.

---

## Recommended Reading Order for a New Intern

Read in this exact order.

### 1) Target architecture first

1. `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md`
   - Why: this is the source-of-truth public API and runtime contract.
   - Most important sections:
     - three layers and three slots (`:56-108`)
     - core types (`:140-180`)
     - handlers and projections (`:184-290`)
     - hydration store (`:500-540`)
     - transports and Hub (`:541-650`)
     - sessions, connections, and TS client (`:652-760`)

2. `le-chat/ttmp/.../design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`
   - Why: this tells you what to reuse and what to avoid.
   - Most important sections:
     - executive summary (`:45-74`)
     - reuse matrix (`:123-139`)
     - direct donor pieces (`§3`)
     - mismatches that must not become substrate core (`§4`)

### 2) Then read the donor code

3. `pinocchio/pkg/webchat/doc.go:1-12`
   - Why: proves the current package already learned that apps should own transport route composition.

4. `pinocchio/pkg/webchat/http/api.go:34-100,128-255`
   - Why: shows the request-policy seam and thin HTTP adapter style that the new transport layer should preserve.

5. `pinocchio/pkg/webchat/stream_backend.go:14-20,29-90`
   - Why: shows a practical seam between the runtime and in-memory/Redis-backed event plumbing.

6. `pinocchio/pkg/webchat/stream_coordinator.go:23-42,125-213,215-301`
   - Why: this is the best donor for subscriber-side ordering and stream-id-derived sequence logic.

7. `pinocchio/pkg/webchat/connection_pool.go:22-125,186-248`
   - Why: shows the correct internal backpressure pattern for websocket fan-out.

8. `pinocchio/pkg/webchat/conversation.go:23-78,262-379`
   - Why: useful lifecycle shape, but also the clearest proof of what **not** to preserve in the substrate (dual ids and SEM-first projection chain).

9. `pinocchio/pkg/webchat/conv_manager_eviction.go:18-139`
   - Why: operationally useful cleanup logic.

10. `pinocchio/pkg/persistence/chatstore/timeline_store.go:9-33`
    - Why: shows the storage seam we can adapt into the new `HydrationStore`.

11. `pinocchio/pkg/webchat/timeline_registry.go:11-124`
    - Why: useful warning sign that package-global projection registries are not appropriate for the new Hub-scoped model.

12. `pinocchio/pkg/webchat/llm_loop_runner.go:203-255`
    - Why: concrete example of a legacy shortcut that bypasses the desired canonical-event path.

---

## Current-State Architecture: What Already Exists

## 1. The target design is already surprisingly concrete

The clean-room technical architecture is not a vague RFC. It already specifies:

- the three layers,
- the three application slots,
- the core Go types,
- the transport seam,
- the hydration seam,
- the Hub lifecycle,
- the TypeScript client contract,
- the role of Watermill,
- and a worked chat example.

Evidence:
- `design/02-technical-architecture-event-streaming-llm-framework.md:56-108` defines the three-layer model and three application slots.
- `...:140-180` defines `SessionId`, `ConnectionId`, `Command`, `Event`, and `Session`.
- `...:184-234` defines `CommandHandler`, `EventPublisher`, and consumption-time ordinal stamping.
- `...:236-290` defines `UIProjection`, `TimelineProjection`, `TimelineView`, and the projection contract.
- `...:500-650` defines `HydrationStore`, transport interfaces, and `Hub`.

That means the job is **not** “invent the architecture from scratch.” The job is “implement this API shape faithfully.”

## 2. The clean-room design already chose the most important boundaries

The most important choices are already made:

- **one canonical routing id**: `SessionId`
- **commands are synchronous** and registered by name
- **backend events** ride on an application-owned Watermill bus
- **ordinals are assigned at consumption time**, not publish time
- **two projections** run from the same backend-event stream
- **hydration is substrate-owned**, not app-owned
- **connections are substrate-owned and opaque to handlers**

These choices are the reason the new framework can support chat, agents, document writers, scrapers, and replay tools without each app rebuilding the same hard parts.

## 3. The donor code already solved several hard runtime problems

The reuse analysis and donor code show that webchat already solved several operational problems well enough to reuse the ideas directly.

### 3.1 Transport ownership and request resolution

`pinocchio/pkg/webchat/doc.go:1-12` explicitly says applications own `/chat` and `/ws`, and package helpers only expose reusable handlers. `pinocchio/pkg/webchat/http/api.go:62-100` further isolates request resolution into `ConversationRequestResolver` and separates chat/stream/timeline handler interfaces.

This is important because it means the new framework does **not** need a monolithic “router” abstraction at all. It can expose:

- a generic `Transport` interface,
- optional adapters,
- and app-owned route composition.

### 3.2 Bus seam and subscriber construction

`pinocchio/pkg/webchat/stream_backend.go:14-20` defines a `StreamBackend` with:

- `Publisher()`
- `BuildSubscriber(...)`
- `Close()`

That is conceptually very close to EVT-STREAM-002's choice that the **application owns the Watermill bus**, while the substrate consumes it.

### 3.3 Subscriber-side ordering

`pinocchio/pkg/webchat/stream_coordinator.go:191-241` derives a monotonic `seq` from a Redis stream id when available, and falls back to an atomic counter otherwise. This is one of the highest-value donor pieces because EVT-STREAM-002 explicitly wants **consumption-time ordinal stamping**.

### 3.4 Backpressure handling

`pinocchio/pkg/webchat/connection_pool.go:22-125` and `:186-248` use one buffered channel per client and drop slow clients rather than stalling the entire broadcast loop. This is exactly the right transport-internal behavior for a reusable framework.

### 3.5 Idle cleanup

`pinocchio/pkg/webchat/conv_manager_eviction.go:18-139` contains a practical conversation eviction loop with sane conditions:

- do not evict if connections still exist,
- do not evict if a stream is still running,
- do not evict if work is still in flight,
- evict once the session is truly idle long enough.

That logic should survive, even though the types around it will change.

---

## Why We Must Not Implement the New Framework by Renaming `pkg/webchat`

This is the most important warning in the whole guide.

Current webchat is a **good donor** and a **bad direct substrate**.

### 1. The identity model is wrong for the new substrate

`pinocchio/pkg/webchat/conversation.go:23-27` defines:

```go
type Conversation struct {
    ID        string
    SessionID string
    ...
}
```

That is already two first-class ids. Later, the stream callback filters by `event.Metadata().SessionID` while routing and storage still center `convID` (`conversation.go:353-372`).

The new framework explicitly rejects that split. `design/02:150-168` makes `SessionId` the universal routing key and `ConnectionId` only a transport-local concern.

### 2. The projection pipeline is shaped incorrectly

Webchat currently does:

```text
Geppetto event -> SEM frame -> websocket
                     \
                      -> timeline projector parses SEM frame -> store
```

Evidence:
- `conversation.go:353-372` broadcasts the translated frame and then feeds the same frame into `TimelineProjector.ApplySemFrame(...)`.
- `timeline_projector.go:82-149` parses a SEM envelope and switches on event types like `llm.start`, `llm.delta`, and `llm.final`.

The new framework explicitly wants:

```text
Backend event -> UIProjection -> UIEvent -> wire
             \
              -> TimelineProjection -> TimelineEntity batch -> store
```

This is the architecture difference that matters most. The new substrate must project from the **canonical backend event**, not from a downstream UI envelope.

### 3. Some legacy paths bypass the canonical event stream entirely

`pinocchio/pkg/webchat/llm_loop_runner.go:203-255` publishes a synthetic user chat SEM event and then separately writes timeline state by hand. That kind of shortcut is fine in an app package, but it is explicitly not what EVT-STREAM-002 is trying to normalize.

### 4. Package-global projection registries are not acceptable in the new model

`pinocchio/pkg/webchat/timeline_registry.go:30-34` stores handlers and runtime bridges in package-global variables. That works for one embedded webchat subsystem, but it is the wrong scope for a reusable framework where multiple `Hub`s may exist in tests or in the same process.

The new framework should keep registration **instance-scoped**, not package-global.

---

## Recommended Initial Code Location

## Recommendation

Implement the first version in:

```text
pinocchio/pkg/evtstream/
```

### Why this is the best starting point

1. `pinocchio` is already a Go module: `pinocchio/go.mod:1-10`.
2. `pinocchio` already depends on Watermill: `pinocchio/go.mod:7-10`.
3. The best donor code (`pkg/webchat`, `pkg/persistence/chatstore`) is already there.
4. The chat backend migration path is clearest if the substrate lives next to the donor implementation.

### Why not implement directly under `le-chat/` first

Observed current state:

- `le-chat/pkg/` is effectively empty.
- `le-chat` currently has no established Go module in this workspace.

That means starting in `le-chat` would force the intern to solve two problems at once:

- framework design and implementation,
- repository/module bootstrap.

That is unnecessary churn for v1.

### How to keep future extraction easy

Even if implementation begins in `pinocchio/pkg/evtstream`, keep the package layout and imports neutral enough that later extraction is mostly mechanical.

Use names like:

- `evtstream`
- `evtstream/bus`
- `evtstream/transport/ws`
- `evtstream/hydration/memory`
- `evtstream/hydration/sql`
- `evtstream/examples/chat`

Do **not** let the new package depend on:

- SEM-specific concepts,
- webchat UI envelope types,
- Glazed layers,
- app-specific request policies,
- webchat router abstractions.

---

## Target Package Layout

This is the recommended initial file tree.

```text
pinocchio/pkg/evtstream/
  types.go
  handler.go
  projection.go
  schema.go
  hub.go
  session_registry.go
  command_registry.go

  bus/
    watermill.go
    consumer.go

  hydration/
    hydration.go
    memory/
      store.go
      store_test.go
    sql/
      store.go
      migrations.go

  transport/
    transport.go
    ws/
      transport.go
      connection_registry.go
      protocol.go
      transport_test.go

  internal/
    ordinals/
      ordinals.go
    copies/
      proto_clone.go

  examples/
    chat/
      schemas.go
      handlers.go
      timeline_projection.go
      ui_projection.go
      register.go

  testdata/
    proto/
```

### Notes on deliberate deviations from the clean-room layout

The clean-room doc groups some things more conceptually than operationally. For actual code, splitting out these files helps a new engineer:

- `session_registry.go`: so session lifecycle does not get buried inside `hub.go`
- `command_registry.go`: so registration and dispatch logic are easy to test without the entire Hub
- `bus/consumer.go`: so event-consumption and projection logic are testable independently from websocket transport
- `transport/ws/connection_registry.go`: so socket bookkeeping is clearly transport-internal

These are structural refinements, not architectural deviations.

---

## Core Mental Model: The Objects You Must Understand

## 1. `SessionId`

This is the most important type in the entire system.

It answers all of these questions:

- which command stream is this work part of?
- which subscribers receive the resulting UI events?
- which hydration snapshot should be loaded?
- which ordinal counter should be resumed?
- which timeline entities belong together?

If you find yourself introducing another substrate-level routing id, stop and re-check the design.

## 2. `ConnectionId`

This identifies a transport-level connection only.

It exists so the substrate can:

- route UI events to specific subscribers,
- remember per-connection liveness state,
- later attach auth/profile context,
- keep websocket transport details out of backend handlers.

Handlers never receive a connection object.

## 3. `Command`

A command is the synchronous request shape entering the system.

Canonical fields:

```go
type Command struct {
    Name         string
    Payload      proto.Message
    SessionId    SessionId
    ConnectionId ConnectionId
}
```

Meaning:

- `Name`: which registered handler to invoke
- `Payload`: typed arguments for that handler
- `SessionId`: which logical session this command belongs to
- `ConnectionId`: where the command came from, if it came from a transport at all

A `Hub.Submit(...)` call is just a command without a meaningful live connection.

## 4. `Event`

A backend event is the canonical internal stream item.

```go
type Event struct {
    Name      string
    Payload   proto.Message
    SessionId SessionId
    Ordinal   uint64
}
```

Very important rule:

- `Ordinal` is **not** assigned by the handler when it publishes.
- `Ordinal` is assigned when the substrate **consumes** the event from the bus.

That rule is what makes distributed workers and restart-safe projection possible.

## 5. `UIProjection`

This is not the frontend itself. It is a substrate-side function contributed by the application.

It translates:

```text
Backend event -> zero or more UI events
```

These UI events are then fanned out to connections subscribed to the same `SessionId`.

## 6. `TimelineProjection`

This translates:

```text
Backend event -> zero or more timeline entities
```

The result is persisted into the hydration store.

It is a sibling of `UIProjection`, not something downstream of it.

## 7. `HydrationStore`

This is the substrate-owned persistence seam.

It must support four things:

- `Apply(...)`: persist entity mutations plus cursor advance atomically
- `Snapshot(...)`: produce reconnect state
- `View(...)`: give projections a read-only current view
- `Cursor(...)`: let the consumer resume ordinals on restart

## 8. `Hub`

This is the substrate entry point and orchestration object.

Its responsibilities are:

- registration of handlers and projections,
- session creation and lookup,
- transport startup,
- bus consumer startup,
- dispatching commands,
- publishing errors and lifecycle events appropriately.

It is **not** a router and it is **not** a chat service.

---

## Runtime Flow Diagrams

## 1. Commands flow down

```text
UI / CLI / test harness
        |
        v
Transport frame or Hub.Submit()
        |
        v
Decode payload using SchemaRegistry
        |
        v
Lookup SessionId -> get/create Session
        |
        v
Lookup CommandHandler by Name
        |
        v
handler(ctx, cmd, sess, pub)
        |
        v
Publish backend event(s) to Watermill bus
```

### Properties that must hold

- dispatch is synchronous with respect to handler invocation,
- the handler sees `Session`, not connection internals,
- the handler publishes backend events; it does not write directly to the wire,
- `Hub.Submit` and websocket-delivered commands follow the same path after decoding.

## 2. Backend events flow up

```text
Command handler or remote worker
        |
        v
Publish Event{Name, Payload, SessionId, Ordinal:0}
        |
        v
Watermill bus
        |
        v
Substrate consumer
        |
        +--> assign ordinal for SessionId
        |
        +--> load TimelineView as-of ordinal-1
        |
        +--> run UIProjection(ctx, ev, sess, view)
        |
        +--> run TimelineProjection(ctx, ev, sess, view)
        |         |
        |         +--> HydrationStore.Apply(session, ordinal, entities)
        |
        +--> route UI events to subscribed ConnectionIds
```

### Properties that must hold

- both projections see the same event and the same pre-event view,
- storage cursor advance happens with timeline apply,
- projection failure does not corrupt ordinal sequencing,
- wire fan-out is keyed by `SessionId`, not by an app-specific conversation id.

## 3. Reconnect / subscribe flow

```text
WS connect
   |
   v
allocate ConnectionId
   |
   v
client sends { subscribe: SessionId, sinceOrdinal }
   |
   v
register ConnectionId in session subscription set
   |
   v
HydrationStore.Snapshot(SessionId, asOf=0/current)
   |
   v
send snapshot first
   |
   v
send live UI events after snapshot
```

The reconnection contract must ensure the client can:

- render a coherent initial view,
- know the last ordinal represented by that snapshot,
- resume live updates without gaps.

---

## API Reference: Minimal Contracts the Intern Should Keep Open While Coding

These are the high-value interfaces from the target design, reproduced here for convenience.

### Core handler and publisher contracts

```go
type CommandHandler func(
    ctx  context.Context,
    cmd  Command,
    sess *Session,
    pub  EventPublisher,
) error

type EventPublisher interface {
    Publish(ctx context.Context, ev Event) error
}
```

Source: `design/02:184-210`

### Projection contracts

```go
type UIProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]UIEvent, error)
}

type TimelineProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]TimelineEntity, error)
}
```

Source: `design/02:269-279`

### Hydration contract

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

Source: `design/02:505-529`

### Transport contract

```go
type Transport interface {
    Start(ctx context.Context, in chan<- IncomingCommand, out <-chan OutgoingMessage) error
}
```

Source: `design/02:546-568`

### Hub registration contract

```go
func (h *Hub) RegisterCommand(name string, handler CommandHandler) error
func (h *Hub) RegisterUIProjection(p UIProjection) error
func (h *Hub) RegisterTimelineProjection(p TimelineProjection) error
func (h *Hub) Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error
func (h *Hub) Run(ctx context.Context) error
```

Source: `design/02:639-649`

---

## Detailed File-by-File Implementation Guide

This section is the most practical part of the document. Read it while coding.

## 1. `types.go`

### Purpose

Define the canonical substrate nouns and keep them boring.

### Must contain

- `SessionId`
- `ConnectionId`
- `Command`
- `Event`
- `Session`
- maybe lightweight `SessionState` enum if needed internally

### Rules

- do not add app-specific fields,
- do not add transport-specific connection objects,
- do not add SEM-specific fields,
- do not rename `SessionId` back to conversation/conv ids.

### Suggested tests

- compile-level and JSON/proto roundtrip tests if helper methods are added
- equality/zero-value sanity tests if helper methods exist

---

## 2. `schema.go`

### Purpose

Central registry of command/event/UI/timeline protobuf message prototypes.

### Why this file matters

The transport receives bytes. The handler and projection layers want typed `proto.Message`. The registry is the bridge.

### Must contain

- `RegisterCommand(name, msg)`
- `RegisterEvent(name, msg)`
- `RegisterUIEvent(name, msg)`
- `RegisterTimelineEntity(kind, msg)`
- lookup helpers for each category
- cloning/new-instance helpers so callers do not accidentally reuse prototype instances

### Implementation note

Prefer storing message prototypes and cloning them when decoding rather than storing concrete runtime instances.

### Suggested pseudocode

```go
type SchemaRegistry struct {
    mu sync.RWMutex
    commands map[string]proto.Message
    events map[string]proto.Message
    uiEvents map[string]proto.Message
    entities map[string]proto.Message
}

func (r *SchemaRegistry) newMessageForCommand(name string) (proto.Message, error) {
    protoType := r.commands[name]
    if protoType == nil { return nil, ErrUnknownCommand }
    return proto.Clone(protoType).ProtoReflect().New().Interface(), nil
}
```

### Suggested tests

- duplicate registration behavior
- unknown-name lookup
- decode into fresh instance, not shared mutable prototype

---

## 3. `handler.go`

### Purpose

Define `CommandHandler` and `EventPublisher`; provide a Watermill-backed publisher implementation.

### Why this file matters

The handler contract is the clearest boundary between “generic framework” and “application code”. Keep it simple.

### Must contain

- `CommandHandler`
- `EventPublisher`
- `NewEventPublisher(pub message.Publisher, reg *SchemaRegistry, ...)`

### Rules

- publisher validates that event name and payload type match the schema registry,
- publisher emits `Ordinal: 0`; ordinals are stamped on consumption,
- handler code should not know or care which Watermill backend is in use.

### Suggested tests

- publish succeeds for registered event type
- publish fails for unknown event name
- publish fails for mismatched payload type

---

## 4. `session_registry.go`

### Purpose

Own in-memory session lifecycle.

### Why this deserves its own file

If session lookup/creation, subscriber sets, and lifecycle cleanup all live inside `hub.go`, the core becomes difficult to test.

### Suggested responsibilities

- `GetOrCreate(sid SessionId)`
- `Get(sid SessionId)`
- `AddSubscription(sid, cid)`
- `RemoveSubscription(sid, cid)`
- `Subscribers(sid)`
- optional idle/liveness bookkeeping

### Important design choice

This registry should own exactly one substrate-level identity: `SessionId`.

Do **not** recreate the `Conversation.ID` + `SessionID` split from `pinocchio/pkg/webchat/conversation.go:23-27`.

### Suggested pseudocode

```go
type sessionState struct {
    sess *Session
    subscribers map[ConnectionId]struct{}
    lastActivity time.Time
}

type SessionRegistry struct {
    mu sync.RWMutex
    byID map[SessionId]*sessionState
    factory SessionMetadataFactory
}
```

### Suggested tests

- get/create is idempotent
- first reference triggers metadata factory exactly once
- subscriptions are maintained correctly

---

## 5. `command_registry.go`

### Purpose

Map command names to handlers and centralize validation.

### Must contain

- registration with duplicate-name protection
- handler lookup
- maybe an internal dispatch helper used by the Hub

### Rules

- one name -> one handler
- dispatch must fail clearly on unknown name
- command dispatch must not depend on websocket types

---

## 6. `hydration/hydration.go`

### Purpose

Define `HydrationStore`, `Snapshot`, `TimelineEntity`, and `TimelineView` if they do not already live in root `projection.go`.

### Important implementation invariant

`Apply(ctx, sid, ord, entities)` must be **atomic with cursor advance**. That is the storage guarantee the rest of the runtime depends on.

### Why this is stricter than current webchat storage

Current `TimelineStore` in `pinocchio/pkg/persistence/chatstore/timeline_store.go:23-33` is a good donor, but it is missing the full cursor/view contract of the new design.

### Suggested storage model

For memory store:

```go
type sessionTimeline struct {
    cursor uint64
    entities map[key]TimelineEntity
}
```

For SQL store:

- one table for per-session cursor
- one table for `(session_id, kind, id)` current entity state
- transaction around entity upserts/deletes plus cursor update

### Suggested tests

- batch apply + cursor advance
- snapshot current and snapshot as-of
- view returns defensive copies
- tombstone deletes entity
- restart cursor resumes correctly

---

## 7. `projection.go`

### Purpose

Define the projection interfaces and the read-only `TimelineView` abstraction.

### Important implementation invariant

Both projections must observe the same **pre-event** state.

That means when the consumer handles event `N`, it should build a view representing state as of `N-1`, then pass that same view to both projections.

### Suggested test cases

- first event creates a new entity
- second event appends to the existing entity
- UI and timeline projection both see the same previous state
- projection errors do not mutate shared store state accidentally

---

## 8. `bus/watermill.go`

### Purpose

Wrap Watermill-specific plumbing.

### Why this file matters

The substrate design assumes an application-owned Watermill bus (`design/02:215-234`, `:615-616`). The rest of the framework should depend on small interfaces, not Watermill directly everywhere.

### Suggested responsibilities

- helper to build `EventPublisher`
- helper to subscribe using app-supplied subscriber
- maybe topic naming helper if the framework standardizes one

### Topic/partitioning rule

Partition by `SessionId`.

This is not optional. The ordinal and projection model depends on ordered consumption per session.

---

## 9. `bus/consumer.go`

### Purpose

Own the canonical backend-event consumption loop.

### This is one of the most important files in the whole implementation

It is where the following things happen together:

- bus subscription,
- backend event decoding,
- ordinal assignment,
- session lookup,
- timeline view retrieval,
- parallel projection execution,
- store apply,
- UI event fan-out.

### Suggested pseudocode

```go
func (c *Consumer) handleMessage(ctx context.Context, msg *message.Message) error {
    ev := decodeEvent(msg)

    sess := sessions.GetOrCreate(ev.SessionId)

    ord := ordinals.Next(ev.SessionId, msg.Metadata)
    ev.Ordinal = ord

    view := store.View(ctx, ev.SessionId) // must represent ord-1 state

    var uiEvents []UIEvent
    var entities []TimelineEntity
    var uiErr, tlErr error

    parallel(
        func() { uiEvents, uiErr = uiProjection.Project(ctx, ev, sess, view) },
        func() { entities, tlErr = timelineProjection.Project(ctx, ev, sess, view) },
    )

    if tlErr == nil {
        if err := store.Apply(ctx, ev.SessionId, ord, entities); err != nil {
            return err // storage failure is a hard failure
        }
    } else {
        // configurable projection error policy; default log+advance
        _ = store.Apply(ctx, ev.SessionId, ord, nil)
    }

    if uiErr == nil {
        out <- routedMessages(ev.SessionId, sessions.Subscribers(ev.SessionId), uiEvents)
    }

    return nil
}
```

### Direct donor logic to reuse

Take ordinal derivation inspiration from `pinocchio/pkg/webchat/stream_coordinator.go:191-241`.

### Rules

- ordinals are assigned only here,
- the consumer is the only place that advances the durable cursor,
- the consumer should not know about websocket connection objects directly,
- the consumer must not translate into SEM envelopes or other UI-specific formats.

### Suggested tests

- message with Redis stream id yields stable ordinal
- fallback ordinal increments correctly when metadata missing
- timeline apply happens before UI fan-out cursor exposure
- projection errors do not prevent cursor advance under default policy
- storage error halts consumer

---

## 10. `hub.go`

### Purpose

Top-level orchestration and public entrypoint.

### Keep `Hub` small

The Hub should mainly:

- store references to registries, transports, store, and consumer,
- expose registration methods,
- expose `Submit(...)`,
- start/stop transports and the consumer.

Avoid burying the implementation of every subsystem inside `Hub` itself.

### Suggested pseudocode

```go
type Hub struct {
    schemas *SchemaRegistry
    sessions *SessionRegistry
    commands *CommandRegistry
    hydration HydrationStore
    transports []transport.Transport
    consumer *bus.Consumer
    incoming chan transport.IncomingCommand
    outgoing chan transport.OutgoingMessage
}

func (h *Hub) Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error {
    cmd := Command{Name: name, Payload: payload, SessionId: sid}
    return h.dispatch(ctx, cmd)
}
```

### Rules

- `Submit` should reuse the same dispatch path as transport-delivered commands,
- registration should happen before `Run`,
- exactly one `UIProjection` and one `TimelineProjection` per Hub in v1,
- the Hub should own shutdown semantics cleanly.

### Suggested tests

- register command/projection duplicates fail
- `Submit` reaches handler
- `Run` starts consumer + transports
- `Shutdown` is idempotent and clean

---

## 11. `transport/transport.go`

### Purpose

Define the generic transport seam and message types.

### Why this file matters

This is where you protect the substrate from becoming websocket-specific.

### Rules

- `IncomingCommand.PayloadBytes` should remain raw bytes until the core decodes them via the schema registry,
- `OutgoingMessage` should carry already-routed `ConnectionId`s,
- transport should not know how to run projections,
- transport should not know how to assign ordinals.

---

## 12. `transport/ws/transport.go`

### Purpose

Implement the websocket transport.

### What to reuse conceptually

Take these ideas from current webchat:

- keep websocket-specific state inside the transport,
- use one buffered send queue per client,
- drop slow clients rather than stalling the entire system,
- keep ping/pong or heartbeat logic transport-local.

Evidence:
- `pinocchio/pkg/webchat/connection_pool.go:22-125,186-248`

### What **not** to reuse

- do not expose `*websocket.Conn` to handlers,
- do not make websocket events the canonical internal representation,
- do not make the transport own session business logic.

### Suggested subscribe protocol for v1

Client frame:

```json
{ "subscribe": "<session-id>", "sinceOrdinal": 0 }
```

Server behavior:

1. allocate `ConnectionId`
2. register subscription in `SessionRegistry`
3. fetch snapshot from `HydrationStore`
4. send snapshot first
5. then send live `UIEvent` messages

### Suggested tests

- subscribe returns snapshot before live event
- unsubscribe removes subscription
- slow client gets dropped safely
- malformed command payload returns error frame or disconnect according to policy

---

## 13. `examples/chat/*`

### Purpose

Provide the first application on top of the substrate.

### Why this matters

A framework without an example is hard to verify and hard to teach.

The clean-room design already includes a worked example for:

- `StartInference`
- `InferenceStarted`
- `TokensDelta`
- `InferenceFinished`
- `ChatMessage`
- `MessageAppended`
- `MessageFinished`

Source:
- `design/02:296-453`

### Suggested file roles

- `schemas.go`: register command/event/UI/entity schemas
- `handlers.go`: command handlers
- `timeline_projection.go`: backend event -> `ChatMessage`
- `ui_projection.go`: backend event -> UI events
- `register.go`: helper that wires everything into a Hub

### Important rule

This example is where chat-specific naming belongs.

Do **not** move chat-specific behavior back into the substrate.

---

## Implementation Phases

This is the recommended implementation order for a new intern.

## Phase 0 — Bootstrap and skeleton

### Goal

Create the package tree and compile-only interfaces without implementing the full runtime.

### Files to create

- `pinocchio/pkg/evtstream/types.go`
- `pinocchio/pkg/evtstream/handler.go`
- `pinocchio/pkg/evtstream/projection.go`
- `pinocchio/pkg/evtstream/schema.go`
- `pinocchio/pkg/evtstream/hub.go`
- `pinocchio/pkg/evtstream/transport/transport.go`
- `pinocchio/pkg/evtstream/hydration/hydration.go`

### Definition of done

- package compiles,
- interfaces match the clean-room design,
- no transport or store implementation yet,
- minimal constructor tests compile.

### Why this phase matters

It freezes naming and file boundaries before behavior becomes hard to move.

---

## Phase 1 — In-memory core without websocket transport

### Goal

Get `Hub.Submit(...)` working end-to-end before touching websockets.

This follows the spirit of the earlier architectural recommendation: start with the simpler non-persistent command path first.

### Files to implement

- `session_registry.go`
- `command_registry.go`
- `bus/consumer.go` (in-process mode)
- `hydration/memory/store.go`
- minimal `hub.go` orchestration

### Definition of done

A test can:

1. register schemas,
2. register one handler,
3. register one UI projection,
4. register one timeline projection,
5. call `Hub.Submit(...)`,
6. observe timeline state in the memory hydration store.

### Tests to write first

- `TestSubmitDispatchesHandler`
- `TestConsumerAssignsOrdinals`
- `TestTimelineProjectionAppliesEntities`
- `TestSnapshotReturnsCurrentEntities`

---

## Phase 2 — Watermill-backed event path

### Goal

Replace pure in-memory shortcuts with real publisher/subscriber flow via Watermill.

### Files to implement

- `bus/watermill.go`
- real Watermill publisher adapter in `handler.go`
- real consumer loop in `bus/consumer.go`

### Key behavior

- event publisher serializes to Watermill message
- consumer decodes back into typed event
- ordinals are assigned on consume
- cursor resumes from store on restart

### Direct donor to use

- `pinocchio/pkg/webchat/stream_coordinator.go:191-241` for stream-id-derived sequence logic

### Definition of done

A test with Watermill `gochannel` can:

- publish two events to the same `SessionId`,
- consume them in order,
- produce strictly increasing ordinals,
- persist the final cursor.

---

## Phase 3 — Websocket transport

### Goal

Add live subscriptions and fan-out.

### Files to implement

- `transport/ws/transport.go`
- `transport/ws/connection_registry.go`
- `transport/ws/protocol.go`

### Direct donor to use

- `pinocchio/pkg/webchat/connection_pool.go:22-125,186-248`

### Definition of done

A test can:

- open websocket,
- subscribe to a session,
- receive snapshot,
- receive subsequent UI events after a `Submit(...)` or bus-published event.

### Important warning

Do not make the websocket transport responsible for translating backend events to UI events. That must still happen in the consumer/projection layer.

---

## Phase 4 — Example chat backend

### Goal

Prove the substrate with one concrete application.

### Files to implement

- `examples/chat/schemas.go`
- `examples/chat/handlers.go`
- `examples/chat/timeline_projection.go`
- `examples/chat/ui_projection.go`
- `examples/chat/register.go`

### Definition of done

A chat example can:

- submit `StartInference`,
- publish `InferenceStarted`, `TokensDelta`, `InferenceFinished`,
- create/update `ChatMessage` timeline entities,
- fan out `MessageAppended` and `MessageFinished` UI events.

### Why this phase matters

This example becomes both:

- the first real validation that the substrate works, and
- the first teaching example for future engineers.

---

## Phase 5 — SQL hydration store and restart behavior

### Goal

Support crash-safe cursor resume and durable snapshots.

### Files to implement

- `hydration/sql/store.go`
- `hydration/sql/migrations.go`

### Reuse donor code from

- `pinocchio/pkg/persistence/chatstore/timeline_store.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_memory.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go`

### Definition of done

An integration test can:

1. publish events,
2. persist entities and cursor,
3. restart consumer,
4. continue from prior cursor without duplicate or skipped ordinals.

---

## Phase 6 — Bridge old webchat or reimplement webchat backend on top

### Goal

Start consuming the substrate from a real app package.

### Recommended direction

Do **not** try to make the old webchat package itself become the substrate.

Instead:

- create a new backend package or example using `evtstream`,
- gradually adapt old webchat semantics to it,
- keep SEM translation, if needed, in the adapter/application layer.

This phase is where legacy compatibility questions belong.

---

## Pseudocode for the Most Important Runtime Paths

## 1. Command dispatch

```go
func (h *Hub) dispatch(ctx context.Context, cmd Command) error {
    sess, err := h.sessions.GetOrCreate(ctx, cmd.SessionId)
    if err != nil { return err }

    handler, ok := h.commands.Lookup(cmd.Name)
    if !ok { return ErrUnknownCommand }

    return handler(ctx, cmd, sess, h.publisher)
}
```

## 2. Consumer-side ordinal assignment

```go
func (o *OrdinalAssigner) Next(sid SessionId, metadata message.Metadata) uint64 {
    if streamID := extractStreamID(metadata); streamID != "" {
        if derived, ok := deriveFromStreamID(streamID); ok {
            return max(derived, current[sid]+1)
        }
    }
    return current[sid] + 1
}
```

## 3. Snapshot-before-live subscribe

```go
func (t *WSTransport) subscribe(cid ConnectionId, sid SessionId, since uint64) error {
    sessions.AddSubscription(sid, cid)

    snap, err := store.Snapshot(ctx, sid, 0)
    if err != nil { return err }

    send(cid, SnapshotMessage{Snapshot: snap})
    return nil
}
```

## 4. Projection pair execution

```go
view, _ := store.View(ctx, sid)

uiEvents, uiErr := uiProjection.Project(ctx, ev, sess, view)
entities, tlErr := timelineProjection.Project(ctx, ev, sess, view)

if tlErr == nil {
    _ = store.Apply(ctx, sid, ev.Ordinal, entities)
} else {
    _ = store.Apply(ctx, sid, ev.Ordinal, nil) // default policy: log + advance
}

if uiErr == nil {
    fanOut(sid, uiEvents)
}
```

---

## Testing and Validation Strategy

The intern should not try to “test everything through websockets” first. That creates slow and ambiguous failures.

Use a testing pyramid.

## 1. Unit tests

Write pure unit tests for:

- schema registry behavior
- command registry behavior
- session registry behavior
- ordinal derivation logic
- memory hydration store behavior
- projection behavior

These tests should be fast and deterministic.

## 2. Component tests

Write component tests for:

- Watermill publisher + consumer with `gochannel`
- `Hub.Submit(...)` -> handler -> event -> projection -> store
- websocket subscribe path with mock or in-memory transport pieces

## 3. Integration tests

Write integration tests for:

- websocket transport over real HTTP server
- snapshot + live stream ordering
- durable SQL store + restart cursor resume
- example chat backend end-to-end

## 4. Regression tests specifically inspired by donor code

Add tests for the mistakes current webchat warns us about:

- no dual-id routing confusion
- no UI-envelope-first timeline projection
- no package-global projection registry pollution
- no manual timeline version increments outside canonical event consume path
- slow websocket client cannot stall all subscribers

---

## Common Mistakes to Avoid

## 1. Reintroducing `ConversationId`

If you find yourself adding both `ConversationId` and `SessionId` to core substrate types, stop. That is rebuilding the old ambiguity.

## 2. Letting UI formats become canonical again

If timeline projection starts consuming websocket/SEM/UI envelopes instead of backend events, stop. That is the wrong pipeline shape.

## 3. Hiding too much logic inside `Hub`

If `Hub` becomes 800 lines long, stop. Move registries and consumers into their own files.

## 4. Making transport own business logic

If websocket transport starts deciding how commands map to backend behavior, stop. That belongs to handlers and application code.

## 5. Writing directly to the timeline store from handlers

Handlers should publish backend events. Timeline state should be derived through `TimelineProjection`. Direct store writes should be reserved for tightly controlled internal machinery, not normal backend behavior.

## 6. Copying webchat globals

If you feel tempted to build global registries like `timeline_registry.go:30-34`, stop. Make them Hub-scoped.

---

## Suggested First-Week Plan for a New Intern

## Day 1 — Read and map

Read:

- EVT-STREAM-002 design/02
- EVT-STREAM-002 design/03
- the donor files listed earlier

Then write a short local note answering:

- what is a `SessionId`?
- where are ordinals assigned?
- why are there two projections?
- why is current webchat not the substrate?

If you cannot answer those four questions, do not start coding yet.

## Day 2 — Skeleton package

Create:

- `types.go`
- `handler.go`
- `projection.go`
- `schema.go`
- `hub.go`
- `transport/transport.go`
- `hydration/hydration.go`

Goal: compile cleanly, no runtime yet.

## Day 3 — In-memory core

Implement:

- `session_registry.go`
- `command_registry.go`
- `hydration/memory/store.go`
- minimal `Hub.Submit(...)`

Goal: handler + projection + store test passes without websocket.

## Day 4 — Watermill consumer

Implement:

- publisher adapter
- consumer loop
- ordinal assigner

Goal: one event roundtrip through real Watermill `gochannel`.

## Day 5 — Websocket transport draft

Implement:

- connection registry
- subscribe protocol
- snapshot-before-live behavior

Goal: transport test receives snapshot then live event.

This first week should end with a very small but real substrate, not with a fully polished framework.

---

## Decisions Recommended Up Front

These are my recommended defaults for the implementation ticket unless a lead engineer explicitly overrides them.

### 1. Initial code home

- **Default**: `pinocchio/pkg/evtstream`
- **Reason**: existing module, Watermill dependency, donor code nearby

### 2. Schema technology

- **Default**: protobuf via the same general approach already described in EVT-STREAM-002
- **Reason**: the design doc is already written that way and the repo already uses protobuf heavily

### 3. First transport

- **Default**: implement `Hub.Submit(...)` and in-memory/Watermill core first; websocket comes next
- **Reason**: lower complexity and better testability

### 4. First store

- **Default**: in-memory store first, SQL later
- **Reason**: lets the runtime and projection contract stabilize before persistence complexity arrives

### 5. First application example

- **Default**: chat example from the design doc
- **Reason**: best-documented worked example already exists

---

## Open Questions That Should Not Block Phase 0–3

These are real questions, but the intern should not stall on them while building the early slices.

1. **`SessionId` allocation**
   - client-generated, backend-generated, or substrate-generated?
   - safe default for early work: caller-supplied with lazy creation

2. **Liveness / ticks**
   - exact cadence and protocol can wait until after websocket transport exists

3. **Projection error policy options**
   - safe default: log + advance

4. **TS client timing**
   - can be deferred until Go substrate and chat example work

5. **Standalone module extraction timing**
   - can wait until package boundaries prove stable

---

## Final Guidance to the Intern

You do **not** need to understand every detail of old webchat to implement the new framework correctly. You need to understand four things deeply:

1. why `SessionId` is the only substrate routing key,
2. why backend events are the canonical internal stream,
3. why UI and timeline projections are siblings,
4. why the bus consumer, not the handler, owns ordinal assignment.

Everything else is implementation detail around those invariants.

When in doubt, compare your code against these two tests:

- **Architectural test**: does this make the substrate more generic, or am I leaking chat/websocket/SEM specifics into the core?
- **Runtime test**: if a remote worker publishes an event and no browser is connected, will the framework still store the right hydration state and later let a browser reconnect to it?

If the answer to either question is “no”, step back and simplify.

---

## References

### Primary design docs

- `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md`
- `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`

### Key donor code

- `pinocchio/go.mod:1-10`
- `pinocchio/pkg/webchat/doc.go:1-12`
- `pinocchio/pkg/webchat/http/api.go:34-100,128-255`
- `pinocchio/pkg/webchat/stream_backend.go:14-20,29-90`
- `pinocchio/pkg/webchat/conversation.go:23-78,262-379`
- `pinocchio/pkg/webchat/conv_manager_eviction.go:18-139`
- `pinocchio/pkg/webchat/stream_coordinator.go:23-42,125-301`
- `pinocchio/pkg/webchat/connection_pool.go:22-125,186-248`
- `pinocchio/pkg/persistence/chatstore/timeline_store.go:9-33`
- `pinocchio/pkg/webchat/timeline_registry.go:11-124`
- `pinocchio/pkg/webchat/llm_loop_runner.go:203-255`

### Important conceptual checkpoints

- `design/02:56-108` — layers, slots, flows
- `design/02:140-180` — core types
- `design/02:184-290` — handlers and projections
- `design/02:500-650` — hydration, transport, Hub
- `design/03:45-74` — donor vs non-donor summary
- `design/03:123-139` — reuse matrix
