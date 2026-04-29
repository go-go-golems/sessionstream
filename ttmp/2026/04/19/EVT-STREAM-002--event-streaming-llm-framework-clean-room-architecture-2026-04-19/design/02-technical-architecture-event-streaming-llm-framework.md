---
Title: Technical Architecture — Event Streaming LLM Framework
Ticket: EVT-STREAM-002
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - chat
    - websocket
    - backend
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design/01-architecture-analysis-event-streaming-llm-framework.md
      Note: "Clean-room architecture analysis (the source of truth this technical design follows)."
    - Path: reference/01-source-image-transcription-2026-04-19-sketches.md
      Note: "Verbatim transcription of the original 2026-04-19 sketches."
ExternalSources: []
Summary: "End-to-end technical architecture for a reusable Go + TypeScript framework for realtime, websocket-driven LLM/agent applications. Three-layer design (Backend / Generic / Client). Three application slots (CommandHandler, UIProjection, TimelineProjection). Application-owned watermill bus for distributed event publishing, substrate-side ordinal stamping, defensive-copy timeline view, protobuf single source of truth via buf for both Go and TypeScript."
LastUpdated: 2026-04-19T15:30:00-04:00
WhatFor: "Specify the public API of the framework — types, signatures, lifecycle, contracts, and seams — at the level of detail required to begin implementation. No implementation code; only Go and TypeScript signatures with their semantics."
WhenToUse: "When designing or implementing any part of the framework's substrate, when wiring a new backend application against it, when reviewing the API surface for additions/changes, or when onboarding a contributor."
---

# Technical Architecture — Event Streaming LLM Framework

## 1. Purpose & scope

This document specifies the public API of a reusable framework for **realtime, websocket-driven LLM/agent applications** — chat, autonomous agents, replayable scrapers, document writers, anything that fits the shape of "structured commands in, structured events out, multiple clients sharing live sessions, hydration on reconnect."

In scope:

- The Go API of the substrate (the **Generic layer**).
- The Go authoring contract for backend applications (the **Backend layer**).
- The TypeScript client API and the UI/Timeline projection contracts (the **Client layer**, whose projections execute substrate-side but are user-supplied).
- The wire/transport seams.
- The application-owned event bus (watermill).
- The hydration / timeline-entity store.
- The protobuf single-source-of-truth schema flow.

Out of scope (intentionally):

- Concrete UI framework choices (React/Vue/Svelte). The TS client is store-agnostic; per-app processors feed app-chosen state.
- Authentication and authorisation. The connection object is the future home of user/identity context (see §13).
- Concrete watermill backend selection (gochannel vs NATS vs Redis vs SQL). The substrate consumes any `message.Publisher`/`message.Subscriber` pair the application configures.
- Stale-connection / liveness contract — see Open Questions §17.

**Source of truth for intent.** This design follows from `design/01-architecture-analysis-event-streaming-llm-framework.md`, which is the clean-room reading of the 2026-04-19 sketches plus four post-sketch clarifications from the author. Where this document and the analysis disagree, the analysis wins and this document is wrong.

## 2. Architectural overview

### 2.1 The three layers

```
+--------------------------------------------------------------------------+
| Backend layer (per app: chat / agent / scraper / document writer / …)    |
|   - typed event schemas                                                  |
|   - typed command set                                                    |
|   - typed Session.Metadata                                               |
|   - command handlers                  (cmd, *Session, EventPublisher)    |
|   - distributed event publishers      (workers on other hosts)           |
+-----------------------------^--------------------------------------------+
                              |  registers handlers + projections,
                              |  publishes backend events
+-----------------------------v--------------------------------------------+
| Generic layer (the substrate)                                            |
|   transport:    ws lifecycle (establish, disconnect, ticks*)             |
|   identity:     ConnectionId (substrate-only),                           |
|                 Session{Id, Metadata}                                    |
|   routing:      SessionId-keyed dispatch, fan-out, multiplexing          |
|   commands:     synchronous dispatch of (Name, Payload, ConnId, SessId)  |
|   bus:          consumer of an application-owned watermill bus           |
|   pipeline:     single backend-event stream → 2 user-supplied            |
|                 projections (UI + Timeline), run in parallel             |
|   storage:      hydration / timeline-entity store, fed by                |
|                 TimelineProjection                                       |
|   ordinals:     monotonic per SessionId, stamped at consumption          |
|   liveness:     ticks + stale-connection reaper (OPEN QUESTION)          |
+-----------------------------^--------------------------------------------+
                              |  UI events out, UI commands in
+-----------------------------v--------------------------------------------+
| Client layer (TS — projections execute substrate-side)                   |
|   - UIProjection        (Event, Session, View) -> []UIEvent              |
|   - TimelineProjection  (Event, Session, View) -> []TimelineEntity       |
|   - UI command set + mapping to backend commands                         |
|   - hydration consumer  (snapshot -> view-state reconciliation)          |
|   - typed UI event schemas + timeline entity schemas                     |
+--------------------------------------------------------------------------+
```

### 2.2 The three application slots

A backend application contributes exactly **three pluggable slots** to the substrate:

1. **`CommandHandler`** — `(ctx, Command, *Session, EventPublisher) -> error`. Services one registered command name. Synchronous.
2. **`UIProjection`** — `(ctx, Event, *Session, TimelineView) -> []UIEvent`. Turns one backend event into UI events for fan-out to subscribed connections.
3. **`TimelineProjection`** — `(ctx, Event, *Session, TimelineView) -> []TimelineEntity`. Reduces (current entity, event) → next entity, persisted by the substrate into the hydration store.

Plus a **`SessionMetadataFactory`** to populate the typed slot in `Session.Metadata` on first reference to a `SessionId`.

### 2.3 Two perpendicular flows

- **Commands flow down**: UI → Generic → Backend handler. Entry can be either a websocket frame or `Hub.Submit` (the page-1 SRC→PKG non-persistent path).
- **Backend events flow up**: Backend handler (or distributed worker) publishes onto the watermill bus → substrate consumes → both projections run in parallel for every event → UI events fan out to subscribed connections; timeline entities apply to the hydration store.

## 3. Package layout

```
github.com/wesen/evtstream/
  evtstream/                  # core substrate (Generic layer)
    types.go                  # SessionId, ConnectionId, Command, Event, Session
    hub.go                    # Hub, registration, lifecycle, Submit
    handler.go                # CommandHandler, EventPublisher
    projection.go             # UIProjection, TimelineProjection, UIEvent, TimelineEntity, TimelineView
    schema.go                 # SchemaRegistry — proto descriptors per name
    hydration.go              # HydrationStore, Snapshot
    transport/
      transport.go            # Transport interface + IncomingCommand/OutgoingMessage
      ws/                     # websocket transport
    bus/
      bus.go                  # EventPublisher, consumer wiring
      watermill.go            # watermill helpers (publisher/subscriber adapters)
    liveness/                 # ticks + stale-connection reaper (open-question)
  proto/
    api/v1/                   # *.proto: commands, events, ui_events, timeline_entities
  examples/
    chat/                     # backend example: chat (uses ChatMessage timeline)
    agent/                    # backend example: autonomous agent
    scraper/                  # backend example: replayable scraper
  ts/                         # TypeScript client (published separately)
    src/client.ts
    src/projections.ts
    src/types/                # generated from proto via buf + @bufbuild/protobuf
```

## 4. Core types (`evtstream/types.go`)

```go
package evtstream

import (
    "context"
    "google.golang.org/protobuf/proto"
)

// SessionId is the universal routing key. Substrate-owned.
type SessionId string

// ConnectionId identifies a single wire-level connection. Substrate-owned and
// opaque to backend handlers — they never receive a connection object.
type ConnectionId string

// Command is the wire-and-handler shape of an incoming request. The Payload
// carries typed command arguments (a proto-generated message).
type Command struct {
    Name         string
    Payload      proto.Message
    SessionId    SessionId
    ConnectionId ConnectionId   // substrate-only routing metadata
}

// Event is a backend-emitted message carried on the watermill bus. Ordinal
// is stamped at consumption (subscriber-side), monotonic per SessionId, so
// projections and hydration can resume.
type Event struct {
    Name      string
    Payload   proto.Message
    SessionId SessionId
    Ordinal   uint64           // 0 at publish; assigned on consumption
}

// Session is the substrate-owned per-session object. Metadata holds the
// application's typed session shape, populated by SessionMetadataFactory.
type Session struct {
    Id       SessionId
    Metadata any
}
```

## 5. Command handlers & event publishing (`evtstream/handler.go`)

```go
package evtstream

// CommandHandler is what an application registers to service one command Name.
// The backend never sees a connection — only (ctx, Command, *Session, pub).
type CommandHandler func(
    ctx  context.Context,
    cmd  Command,
    sess *Session,
    pub  EventPublisher,
) error

// EventPublisher publishes backend Events to the application's watermill bus.
// The same interface is used by:
//   - in-process command handlers (passed in by the substrate dispatch path),
//   - distributed workers running on other hosts (constructed directly from
//     the application's bus by NewEventPublisher).
type EventPublisher interface {
    Publish(ctx context.Context, ev Event) error
}

// NewEventPublisher constructs a publisher against an application-owned
// watermill Publisher. The SchemaRegistry is consulted to validate the
// proto.Message type matches the registered schema for ev.Name.
func NewEventPublisher(pub message.Publisher, reg *SchemaRegistry, opts ...PublisherOption) EventPublisher
```

### 5.1 Distributed publishing

The application owns the watermill bus. Both the substrate and the application's distributed workers publish/consume against the same backend (gochannel for in-process, NATS/Redis/SQL for distributed).

```go
// On a remote worker host — no Hub, no Transport, no websocket. Just a
// publisher into the same bus.
pub := evtstream.NewEventPublisher(natsPub, reg)
_ = pub.Publish(ctx, evtstream.Event{
    Name:      "TaskStepCompleted",
    Payload:   &agentpb.TaskStepCompleted{StepId: "s-42", Output: "…"},
    SessionId: sid,
})
```

### 5.2 Ordinal assignment

`Event.Ordinal` is **0 at publish time** and **assigned on consumption** by the substrate's subscriber. The substrate keeps a monotonic counter per `SessionId` (in memory; persisted via the hydration store's cursor for restart durability — see §9.4). This means:

- Watermill must preserve publish-order per partition; partition by `SessionId`.
- Two distributed publishers racing on the same `SessionId` are ordered by the bus, not by wall-clock.
- Restarts of the substrate's consumer resume ordinal assignment from the hydration store's last-known cursor (so projections never see duplicate ordinals on a clean restart).

## 6. Projections (`evtstream/projection.go`)

```go
package evtstream

// UIEvent is what the front-end UI consumes (per-app schemas; proto-typed).
type UIEvent struct {
    Name    string
    Payload proto.Message
}

// TimelineEntity is the single timeline value type. Tombstone=true with a nil
// Payload signals deletion of (Kind, Id) within the SessionId.
type TimelineEntity struct {
    Kind      string
    Id        string
    Payload   proto.Message
    Tombstone bool
}

// TimelineView is a read-only handle on the session's current timeline state,
// passed to both projections so they can reduce (current entity, event) ->
// next entity. The view is consistent as-of just-before the event being
// processed; the substrate refreshes it for each event.
//
// Get/List return defensive copies — projections may freely mutate the
// returned proto.Message without affecting the underlying store.
type TimelineView interface {
    Get(kind, id string) (TimelineEntity, bool)
    List(kind string) []TimelineEntity
    Ordinal() uint64
}

// UIProjection turns one backend Event into zero or more UIEvents.
type UIProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]UIEvent, error)
}

// TimelineProjection turns one backend Event into zero or more TimelineEntity
// values. A returned entity is upserted; Tombstone=true deletes. Returning
// nil leaves all existing entities unchanged.
type TimelineProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]TimelineEntity, error)
}
```

### 6.1 Projection contract

For every consumed `Event`, the substrate runs both projections in parallel:

1. Build a `TimelineView` consistent at `Event.Ordinal - 1`.
2. Pass `(ctx, ev, sess, view)` to each projection.
3. UI projection's results are routed to the wire (see §10); timeline projection's results are written via `HydrationStore.Apply` (see §9) atomically with the cursor advance to `Event.Ordinal`.

If either projection returns an error, the substrate logs it and continues — the cursor still advances. Projection errors must not block the event stream. (Error policy is configurable; see §15.)

## 7. Worked example — `ChatMessage` from streaming token deltas

This is the canonical example for the "first event creates the entity, subsequent events reduce into it" pattern.

### 7.1 Schemas (`proto/api/v1/chat.proto`)

```proto
syntax = "proto3";
package evtstream.api.v1.chat;

import "google/protobuf/timestamp.proto";

// Backend events
message InferenceStarted   { string message_id = 1; string model = 2; }
message TokensDelta        { string message_id = 1; string tokens = 2; }
message InferenceFinished  { string message_id = 1; string finish_reason = 2; }

// Backend commands
message StartInference     { string prompt = 1; string model = 2; }
message StopInference      { string message_id = 1; }

// Timeline entity — the canonical ChatMessage
message ChatMessage {
    string                    id         = 1;
    string                    role       = 2;     // "user" | "assistant" | …
    string                    text       = 3;
    google.protobuf.Timestamp created_at = 4;
    bool                      streaming  = 5;
}

// UI events
message MessageAppended { string message_id = 1; string text = 2; }
message MessageFinished { string message_id = 1; string finish_reason = 2; }
```

### 7.2 Timeline projection (`examples/chat/timeline_projection.go`)

```go
package chat

type timelineProjection struct{}

func (timelineProjection) Project(
    ctx  context.Context,
    ev   evtstream.Event,
    sess *evtstream.Session,
    view evtstream.TimelineView,
) ([]evtstream.TimelineEntity, error) {

    switch ev.Name {

    case "TokensDelta":
        d := ev.Payload.(*chatpb.TokensDelta)

        if existing, ok := view.Get("ChatMessage", d.MessageId); ok {
            // Subsequent delta — reduce: append text to the existing message.
            msg := existing.Payload.(*chatpb.ChatMessage)   // defensive copy from view
            msg.Text += d.Tokens
            return []evtstream.TimelineEntity{{
                Kind:    "ChatMessage",
                Id:      d.MessageId,
                Payload: msg,
            }}, nil
        }

        // First delta for this message_id — create the ChatMessage entity.
        return []evtstream.TimelineEntity{{
            Kind: "ChatMessage",
            Id:   d.MessageId,
            Payload: &chatpb.ChatMessage{
                Id:        d.MessageId,
                Role:      "assistant",
                Text:      d.Tokens,
                CreatedAt: timestamppb.Now(),
                Streaming: true,
            },
        }}, nil

    case "InferenceFinished":
        f := ev.Payload.(*chatpb.InferenceFinished)
        existing, ok := view.Get("ChatMessage", f.MessageId)
        if !ok {
            return nil, nil
        }
        msg := existing.Payload.(*chatpb.ChatMessage)
        msg.Streaming = false
        return []evtstream.TimelineEntity{{
            Kind:    "ChatMessage",
            Id:      f.MessageId,
            Payload: msg,
        }}, nil
    }

    return nil, nil
}
```

### 7.3 UI projection (`examples/chat/ui_projection.go`)

```go
package chat

type uiProjection struct{}

func (uiProjection) Project(
    ctx  context.Context,
    ev   evtstream.Event,
    sess *evtstream.Session,
    view evtstream.TimelineView,
) ([]evtstream.UIEvent, error) {

    switch ev.Name {
    case "TokensDelta":
        d := ev.Payload.(*chatpb.TokensDelta)
        return []evtstream.UIEvent{{
            Name:    "MessageAppended",
            Payload: &chatpb.MessageAppended{MessageId: d.MessageId, Text: d.Tokens},
        }}, nil

    case "InferenceFinished":
        f := ev.Payload.(*chatpb.InferenceFinished)
        return []evtstream.UIEvent{{
            Name:    "MessageFinished",
            Payload: &chatpb.MessageFinished{MessageId: f.MessageId, FinishReason: f.FinishReason},
        }}, nil
    }
    return nil, nil
}
```

### 7.4 Command handler (`examples/chat/handlers.go`)

```go
package chat

func startInference(
    ctx  context.Context,
    cmd  evtstream.Command,
    sess *evtstream.Session,
    pub  evtstream.EventPublisher,
) error {
    args := cmd.Payload.(*chatpb.StartInference)
    msgId := newMessageId()

    if err := pub.Publish(ctx, evtstream.Event{
        Name:      "InferenceStarted",
        Payload:   &chatpb.InferenceStarted{MessageId: msgId, Model: args.Model},
        SessionId: cmd.SessionId,
    }); err != nil {
        return err
    }

    // The actual inference loop runs asynchronously and publishes TokensDelta
    // and InferenceFinished events to pub. Could be in-process or on a remote
    // worker — same EventPublisher interface.
    go runInference(args, msgId, cmd.SessionId, pub)

    return nil
}
```

## 8. Schema registry (`evtstream/schema.go`)

```go
package evtstream

// SchemaRegistry holds the proto descriptors the substrate needs to decode
// inbound payloads (commands), encode outbound ones (UI events), persist
// timeline entities, and validate publish-time event payloads.
type SchemaRegistry struct{ /* unexported */ }

func NewSchemaRegistry() *SchemaRegistry

func (r *SchemaRegistry) RegisterCommand(name string,        msg proto.Message) error
func (r *SchemaRegistry) RegisterEvent(name string,          msg proto.Message) error
func (r *SchemaRegistry) RegisterUIEvent(name string,        msg proto.Message) error
func (r *SchemaRegistry) RegisterTimelineEntity(kind string, msg proto.Message) error

// Lookups (used by the substrate, not by application code, but exposed so
// tooling can introspect):
func (r *SchemaRegistry) CommandSchema(name string)        (proto.Message, bool)
func (r *SchemaRegistry) EventSchema(name string)          (proto.Message, bool)
func (r *SchemaRegistry) UIEventSchema(name string)        (proto.Message, bool)
func (r *SchemaRegistry) TimelineEntitySchema(kind string) (proto.Message, bool)
```

### 8.1 Codegen flow

Single source of truth in `proto/api/v1/*.proto`. **`buf generate`** produces:

- Go: `proto/api/v1/*.pb.go` via `protoc-gen-go`.
- TS: `ts/src/types/*.ts` via `protoc-gen-es` (`@bufbuild/protobuf`).

`buf.gen.yaml`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: proto
    opt: paths=source_relative
  - remote: buf.build/bufbuild/es
    out: ts/src/types
    opt: target=ts
```

The same names used in `RegisterCommand("StartInference", &chatpb.StartInference{})` are the same wire names used by the TS client — there is no separate wire schema.

## 9. Hydration store (`evtstream/hydration.go`)

```go
package evtstream

// HydrationStore is the substrate-owned persistence seam for timeline
// entities. Backends never call it directly; the substrate calls Apply on
// every TimelineProjection result and Snapshot on subscribe/reconnect.
type HydrationStore interface {
    // Apply persists a batch of entity mutations atomically with advancing
    // the per-session ordinal cursor to ord.
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error

    // Snapshot returns the timeline state for a session as-of asOf (exclusive
    // upper bound). Pass 0 to mean "current".
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)

    // View returns a TimelineView consistent at the current ordinal. Used by
    // the substrate to feed projections.
    View(ctx context.Context, sid SessionId) (TimelineView, error)

    // Cursor returns the last-applied ordinal for a session, or 0 if none.
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}

type Snapshot struct {
    SessionId SessionId
    Ordinal   uint64
    Entities  []TimelineEntity
}
```

### 9.1 In-tree implementations

- `hydration/memory` — in-memory, for tests and single-process dev.
- `hydration/sql` — Postgres-backed, for production. Schema: one row per `(SessionId, Kind, Id)` plus a per-session cursor row.

### 9.2 Cursor durability

`Cursor` is what the substrate's bus consumer reads at startup to resume ordinal assignment. Because `Apply` is atomic with cursor advance, the system survives crashes without duplicate or skipped ordinals.

## 10. Transports (`evtstream/transport/`)

```go
package transport

// Transport is the seam between the substrate and the wire. The Hub starts
// every registered Transport and pumps IncomingCommand into the dispatcher
// while consuming OutgoingMessage produced by UIProjection fan-out.
type Transport interface {
    Start(
        ctx context.Context,
        in chan<- IncomingCommand,
        out <-chan OutgoingMessage,
    ) error
}

type IncomingCommand struct {
    ConnectionId ConnectionId
    SessionId    SessionId
    Name         string
    PayloadBytes []byte    // substrate decodes via SchemaRegistry
}

type OutgoingMessage struct {
    SessionId     SessionId
    ConnectionIds []ConnectionId   // substrate already routed; transport just delivers
    UIEvent       UIEvent
}
```

### 10.1 Websocket transport (`evtstream/transport/ws/`)

```go
package ws

func New(addr string, opts ...Option) transport.Transport

type Option func(*config)
func WithTLS(cert, key string) Option
func WithSubprotocol(s string) Option
func WithReadLimit(bytes int) Option
```

On connect, the websocket transport allocates a `ConnectionId` and accepts a subscription frame from the client (`{ subscribe: SessionId, sinceOrdinal: uint64 }`). The substrate replies with a `Snapshot` (the hydration payload), then live `UIEvent` frames as they are produced.

### 10.2 In-process submit (Hub.Submit)

The page-1 SRC→PKG slice ("no need for persistent connection") is exposed as a method on the Hub rather than a separate transport. Same dispatch pipeline; no `ConnectionId`.

```go
func (h *Hub) Submit(
    ctx     context.Context,
    sid     SessionId,
    name    string,
    payload proto.Message,
) error
```

Use cases:
- **REPLAY** — a Go-side test harness or CLI re-issues commands.
- **Batch / cron** — scheduled jobs that submit work without a UI client.
- **Server-side seeding** — bootstrap a session before any UI connects.

Submitted commands flow through the same `CommandHandler` registration; events fan out to whichever connections are subscribed (possibly none).

## 11. The Hub — substrate entry point (`evtstream/hub.go`)

```go
package evtstream

type Hub struct{ /* unexported */ }

type HubOption func(*Hub)

// Bus — application owns the watermill backend; substrate consumes from it.
func WithEventBus(pub message.Publisher, sub message.Subscriber, opts ...BusOption) HubOption

// Storage.
func WithHydrationStore(s HydrationStore) HubOption

// Wire.
func WithTransport(t transport.Transport) HubOption // may be called multiple times

// Schemas.
func WithSchemaRegistry(r *SchemaRegistry) HubOption

// Sessions.
func WithSessionMetadataFactory(f SessionMetadataFactory) HubOption

// Liveness (open-question).
func WithLiveness(r liveness.Reaper) HubOption

func NewHub(opts ...HubOption) (*Hub, error)

// SessionMetadataFactory builds the typed Session.Metadata for a SessionId.
// Called by the substrate on first reference.
type SessionMetadataFactory func(ctx context.Context, sid SessionId) (any, error)

// The three application slots.
func (h *Hub) RegisterCommand(name string, handler CommandHandler) error
func (h *Hub) RegisterUIProjection(p UIProjection) error            // exactly one per Hub
func (h *Hub) RegisterTimelineProjection(p TimelineProjection) error // exactly one per Hub

// Programmatic command path (see §10.2).
func (h *Hub) Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error

// Lifecycle.
func (h *Hub) Run(ctx context.Context) error
func (h *Hub) Shutdown(ctx context.Context) error
```

## 12. Sessions

### 12.1 Lifecycle

A `SessionId` is created by the substrate on first reference (first command, first subscription, first `Submit`). On creation:

1. The substrate calls `SessionMetadataFactory(ctx, sid)` to populate `Session.Metadata`.
2. `HydrationStore.Cursor(ctx, sid)` is consulted; if non-zero, ordinal assignment resumes from there.

A `Session` lives in the substrate's in-memory map for the duration of the process; persistent state lives in the hydration store. There is no explicit `CreateSession` call — sessions exist by being referenced.

### 12.2 SessionMetadataFactory example

```go
hub, _ := evtstream.NewHub(
    evtstream.WithSessionMetadataFactory(func(ctx context.Context, sid evtstream.SessionId) (any, error) {
        return &chatpb.ChatSessionMetadata{
            CreatedAt:  timestamppb.Now(),
            ModelHint:  "claude-sonnet-4-6",
            TokenLimit: 200_000,
        }, nil
    }),
)
```

### 12.3 SessionId allocation — open question

Whether a `SessionId` is generated by the client, by the backend on `Submit`, or by the substrate on first command, is unsettled (Open Question §17.1). The current API accepts any `SessionId` the caller supplies and creates the session lazily.

## 13. Connections

`ConnectionId` is the substrate's handle on a single websocket connection. It is **never visible to backend handlers**. The substrate owns:

- The mapping `ConnectionId → ws.Conn`.
- The mapping `SessionId → []ConnectionId` (the subscription set).
- Liveness state (last-tick, stale-flag).
- Future user/identity context (see §13.1).

The substrate's transport plumbing, dispatch path, and middleware pipeline can read and mutate connection state. The application's three slots cannot.

### 13.1 User/identity context (future work)

The page-1 sketch shows a `profile-id` field on the inbound command annotation. Per the post-sketch clarification, `profile-id` is **not** a command field — it migrates onto the connection object as user/identity context. The framework reserves space on the connection for this:

```go
// Future shape (illustrative; not in v1):
type ConnectionUser struct {
    ProfileId string
    AuthClaims map[string]any
}
```

A concrete population mechanism — on connect, on first authenticated command, or via an explicit hook — is Open Question §17.6.

## 14. TypeScript client (`ts/src/`)

```ts
// ts/src/types-runtime.ts
import type { Message } from "@bufbuild/protobuf";

export type SessionId    = string & { readonly __brand: "SessionId" };
export type ConnectionId = string & { readonly __brand: "ConnectionId" };

export interface UIEvent<P extends Message = Message> {
    name: string;
    payload: P;
}

export interface TimelineEntity<P extends Message = Message> {
    kind: string;
    id: string;
    payload: P;
    tombstone?: boolean;
}

export interface Snapshot {
    sessionId: SessionId;
    ordinal: number;
    entities: TimelineEntity[];
}
```

```ts
// ts/src/client.ts
export interface EvtStreamClientOptions {
    url: string;
    onUIEvent?:   (sid: SessionId, ev: UIEvent) => void;
    onSnapshot?:  (s: Snapshot) => void;
    onConnState?: (state: "connecting" | "open" | "reconnecting" | "closed") => void;
}

export class EvtStreamClient {
    constructor(opts: EvtStreamClientOptions);

    connect(): Promise<void>;
    close(): void;

    // Send a typed command. Resolves when the substrate's CommandHandler returns.
    send<P extends Message>(
        sid: SessionId,
        name: string,
        payload: P,
    ): Promise<void>;

    // Subscribe to a session: emits the snapshot first, then live UIEvents.
    subscribe(sid: SessionId, sinceOrdinal?: number): Promise<void>;
    unsubscribe(sid: SessionId): Promise<void>;
}
```

```ts
// ts/src/projections.ts
// Per-app processor: turns UI events into store-shaped actions.
// Store technology (Redux / Zustand / signals / …) is application-chosen.
export type UIProcessor<E extends Message, A> = (ev: E) => A[];

// Hydration consumer: turns a Snapshot into store-shaped actions.
export type HydrationProcessor<A> = (s: Snapshot) => A[];
```

A TS-side application typically wires:

1. A `Map<string, UIProcessor>` keyed by `UIEvent.name`.
2. A `HydrationProcessor` that maps `TimelineEntity` arrays to view-state.
3. A command-mapping layer that translates UI intents into `client.send(sid, name, payload)` calls.

## 15. Error policy

| Surface                  | Error path                                                                        |
|--------------------------|-----------------------------------------------------------------------------------|
| `CommandHandler` returns | Surfaced to the caller (websocket frame error / `Submit` returns the error).      |
| `EventPublisher.Publish` | Surfaced to the caller (handler or worker).                                        |
| `UIProjection`           | Logged; cursor advances; offending `Event` does not produce UI events.            |
| `TimelineProjection`     | Logged; cursor advances; offending `Event` produces no timeline change. (Default.)|
| `HydrationStore.Apply`   | Bus consumer halts; supervisor restarts. Cursor durability ensures no skipped ord.|
| `Transport.Start`        | Hub shutdown.                                                                     |

The projection error policy is configurable via `WithProjectionErrorPolicy(policy)` — alternatives include "halt" (stop the consumer until manually resumed) and "dead-letter" (publish offending events to a side topic).

## 16. End-to-end registration (chat example)

```go
package main

func main() {
    ctx := context.Background()

    reg := evtstream.NewSchemaRegistry()
    chat.RegisterSchemas(reg)

    pub, sub := watermill_gochannel.NewBuiltin()  // single-process
    store    := hydration_memory.New()

    hub, err := evtstream.NewHub(
        evtstream.WithSchemaRegistry(reg),
        evtstream.WithEventBus(pub, sub),
        evtstream.WithHydrationStore(store),
        evtstream.WithTransport(ws.New(":8080")),
        evtstream.WithSessionMetadataFactory(chat.NewSession),
    )
    if err != nil { log.Fatal(err) }

    chat.RegisterHandlers(hub)
    chat.RegisterUIProjection(hub)
    chat.RegisterTimelineProjection(hub)

    if err := hub.Run(ctx); err != nil { log.Fatal(err) }
}
```

```go
// examples/chat/register.go
func RegisterSchemas(r *evtstream.SchemaRegistry) {
    r.RegisterCommand("StartInference",       &chatpb.StartInference{})
    r.RegisterCommand("StopInference",        &chatpb.StopInference{})
    r.RegisterEvent  ("InferenceStarted",     &chatpb.InferenceStarted{})
    r.RegisterEvent  ("TokensDelta",          &chatpb.TokensDelta{})
    r.RegisterEvent  ("InferenceFinished",    &chatpb.InferenceFinished{})
    r.RegisterUIEvent("MessageAppended",      &chatpb.MessageAppended{})
    r.RegisterUIEvent("MessageFinished",      &chatpb.MessageFinished{})
    r.RegisterTimelineEntity("ChatMessage",   &chatpb.ChatMessage{})
}

func RegisterHandlers(h *evtstream.Hub) {
    h.RegisterCommand("StartInference", startInference)
    h.RegisterCommand("StopInference",  stopInference)
}

func RegisterUIProjection(h *evtstream.Hub)        { h.RegisterUIProjection(uiProjection{}) }
func RegisterTimelineProjection(h *evtstream.Hub)  { h.RegisterTimelineProjection(timelineProjection{}) }

func NewSession(ctx context.Context, sid evtstream.SessionId) (any, error) {
    return &chatpb.ChatSessionMetadata{ /* … */ }, nil
}
```

## 17. Open questions (carried forward)

These are unresolved and need decisions before or during implementation. Numbering is not the same as the analysis doc; this is the technical-design view.

1. **`SessionId` allocation.** Substrate-generated, client-supplied, or backend-supplied? Affects idempotency and wire shape. Default in this design: caller-supplied.
2. **Persistent ordinal allocation across restarts.** Currently relies on `HydrationStore.Cursor` for resume; if the bus has un-consumed events at restart, the consumer re-stamps continuing from the cursor. Confirm that's acceptable.
3. **Liveness contract** (sketch §3.4 / §7.1 / §7.2). Bidirectional ticks at what cadence? Heartbeat semantics? Stale-connection reaper trigger?
4. **Per-session state machine ownership** (sketch annotation "+state machine (up to PROC, guess)"). Lives at `PROC` (the command handler), at the `Session.Metadata` slot, or distributed across both?
5. **The `X`-marked storage path on page 1.** What store did the author reject and why? This constrains the eventual production `HydrationStore` implementation.
6. **Connection user/identity population.** When does `ConnectionUser` populate — on connect, on first authenticated command, lazily? What hook does the substrate expose?
7. **Projection error policy default.** Currently "log + advance"; confirm vs "halt" / "dead-letter".
8. **REPLAY semantics.** Re-issue commands (current `Hub.Submit` design)? Or replay events directly into the bus? Or replay against a deterministic processor mode?
9. **What `DD` and `DO` mean on page-1 per-session arrows.** Captured for posterity; may not affect the API.
10. **Multiple `UIProjection` / `TimelineProjection` instances per Hub.** Currently restricted to one each (matches the page-2 outline of "the projection"). Open whether composition (chain / merge) is needed.

## 18. Glossary

| Term                  | Meaning                                                                          |
|-----------------------|----------------------------------------------------------------------------------|
| **Substrate**         | The Generic layer — everything in `evtstream/*` except the application slots.    |
| **Session**           | The substrate's per-logical-session object (page-1 `conv` glyph).                |
| **`SessionId`**       | The universal routing key.                                                       |
| **`ConnectionId`**    | A single wire connection's id; substrate-only.                                   |
| **Backend layer**     | The application's command handlers, schemas, and `SessionMetadataFactory`.       |
| **Generic layer**     | The substrate.                                                                   |
| **Client layer**      | The application's `UIProjection`, `TimelineProjection`, UI command set.          |
| **Application slot**  | One of the three pluggable contracts: `CommandHandler`, `UIProjection`, `TimelineProjection`. |
| **Backend event**     | An `Event` produced by a command handler or distributed worker, carried on the bus. |
| **UI event**          | An `UIEvent` produced by `UIProjection`, delivered over the wire.                |
| **Timeline entity**   | A `TimelineEntity` produced by `TimelineProjection`, persisted in the hydration store. |
| **Hydration**         | Reconciling a (re)connecting client to current session state from the store.     |
| **Ordinal**           | Monotonic per-`SessionId` counter on `Event`, stamped at consumption.            |
| **Transport**         | A wire adapter (websocket, …) implementing the substrate's `Transport` interface.|
| **`Hub.Submit`**      | The page-1 SRC→PKG non-persistent command-submission path (in-process).          |
| **REPLAY**            | Re-issuing commands (typically via `Hub.Submit`) to reproduce a session's events.|

## 19. Related

- Clean-room architecture analysis: `01-architecture-analysis-event-streaming-llm-framework.md`
- Source transcription: `../reference/01-source-image-transcription-2026-04-19-sketches.md`
- Source images: `../sources/diagram-page-1.png`, `../sources/building-blocks-page-2.png`
