---
Title: "Sessionstream User Guide"
Slug: "sessionstream-user-guide"
Short: "Understand the Sessionstream application model: sessions, commands, backend events, projections, timelines, hydration, and transports."
Topics:
  - sessionstream
  - user-guide
  - architecture
  - events
  - projections
  - hydration
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

`sessionstream` helps you build applications where backend work unfolds over time and the UI needs both live updates and durable state. The framework is deliberately small: it routes commands, records backend events, runs projections, applies timeline state, and exposes snapshot/live-event seams for clients.

The best way to understand it is to follow one command through the system.

## The application shape

A sessionstream application starts when a client submits a command for a session. The hub routes that command to a handler. The handler publishes backend events. Projections consume those events and derive live UI events and durable timeline entities.

```text
Client command
  -> Hub
  -> CommandHandler
  -> backend Event
  -> UIProjection       -> live UIEvent
  -> TimelineProjection -> durable TimelineEntity
  -> Snapshot on reconnect
```

The handler does not return UI state. This is intentional. If handlers returned UI state directly, they would need to know too much about every consumer. Events let the backend describe what happened once, and let projections decide how different views should respond.

## Sessions

A session is the unit of routing and recovery. Commands, backend events, UI events, cursors, snapshots, and timeline entities are all interpreted relative to a `SessionId`.

Use sessions when users or workflows need isolated timelines. A chat thread is a session. A lab run is a session. A long-running operator workflow can be a session.

## Commands

A command is a request to do work. It has a logical name, a session ID, and a concrete protobuf payload. Commands are validated against the `SchemaRegistry` before the hub dispatches them.

```go
err := hub.Submit(ctx, "session-1", "ChatStartInference", &chatv1.StartInferenceCommand{
    Prompt: "Explain ordinals",
})
```

The command handler receives a publisher:

```go
type CommandHandler func(
    ctx context.Context,
    cmd sessionstream.Command,
    sess *sessionstream.Session,
    pub sessionstream.EventPublisher,
) error
```

The publisher is the handler's way to describe progress. A streaming handler might publish `Started`, many `Delta`, and `Finished` events.

## Backend events

Backend events are canonical. They are the record of what happened, not a rendering instruction. A good event name describes a domain fact:

- `ChatInferenceStarted`
- `ChatTokensDelta`
- `ChatInferenceFinished`
- `CoinVaultInventoryCardsProjected`
- `LabFinished`

Events carry protobuf payloads. Those payloads should be concrete message types owned by the feature that publishes them.

## Projections

A projection derives a view from backend events. Sessionstream has two projection families.

| Projection | Produces | Purpose |
|---|---|---|
| UI projection | `[]UIEvent` | Live client updates. |
| Timeline projection | `[]TimelineEntity` | Durable state for snapshots and hydration. |

The same backend event can produce both a live UI event and a durable timeline entity. It can also produce one without the other.

This split keeps the backend event stream authoritative. If a frontend rendering detail changes, you can often update the UI projection without changing the canonical event.

## Timeline entities

A timeline entity is durable projected state. It has a kind, ID, ordinals, and a protobuf payload.

Timeline entities are not limited to chat messages. They can represent tool calls, reasoning segments, inventory widgets, lab records, errors, or any other durable piece of UI state.

Use stable IDs. If an event updates an existing concept, emit the same kind and ID. If an event creates a new concept, choose a new ID. Deletion is represented with tombstones where the store/projection path supports it.

## Hydration and reconnects

Hydration is the recovery path. A client that reconnects should not ask command handlers to replay work. Instead, it receives a snapshot of durable timeline state and then subscribes to future live UI events.

The websocket contract is snapshot-before-live:

1. Client subscribes to a session.
2. Server sends the current snapshot.
3. Server sends future live UI events.

This contract lets the backend restart, the browser reload, and the UI still recover the current timeline.

## Ordinals

Ordinals define event order. Backend events receive ordinals, snapshots report the highest materialized timeline ordinal, and live UI frames identify which backend event produced them.

In browser-facing protobuf JSON, `uint64` ordinals appear as strings. Treat them as strings or big integers in JavaScript if precision matters.

## Schema registry

The schema registry maps logical names to protobuf message prototypes:

```go
reg.RegisterCommand("ChatStartInference", &chatv1.StartInferenceCommand{})
reg.RegisterEvent("ChatInferenceStarted", &chatv1.InferenceStartedEvent{})
reg.RegisterUIEvent("ChatMessageStarted", &chatv1.ChatMessageUpdate{})
reg.RegisterTimelineEntity("ChatMessage", &chatv1.ChatMessageEntity{})
```

The registry is the contract among runtime code, projections, persistence, websocket transport, and frontend parsing. Use concrete protobuf messages. Do not use top-level `*structpb.Struct` as a shortcut.

## How to design a feature

When adding a feature, work in this order:

1. Name the backend events that describe what happens.
2. Define protobuf messages for command, event, UI, and entity payloads.
3. Register those schemas.
4. Implement command handlers that publish backend events.
5. Implement UI projections for live updates.
6. Implement timeline projections for durable state.
7. Exercise both the live path and the snapshot path.
8. Run `make schema-vet`.

This order prevents a common mistake: designing the UI payload first and then treating backend events as transport envelopes. Backend events should remain meaningful even if the UI changes.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| A command validates but nothing changes. | The handler may not publish any backend events, or projections ignore those events. | Inspect the handler and projection event-name switch statements. |
| Live UI works but reload loses state. | UI projection exists, but timeline projection does not emit durable entities. | Add or fix the timeline projection and hydration store. |
| Hydrated state differs from live state. | UI and timeline payloads use different shapes or names. | Use concrete protobuf messages and test live plus snapshot paths. |
| Events arrive out of expected order. | Ordinal assignment or bus stream IDs are not configured as expected. | Inspect event ordinals and bus metadata. |
| Schema-vet fails. | A top-level registration uses `*structpb.Struct`. | Replace it with a concrete protobuf message. |

## See Also

- `sessionstream-getting-started` for a step-by-step first app.
- `sessionstream-reference` for API and package details.
- `sessionstream-schema-vet-playbook` for schema policy enforcement.
- `proto/sessionstream/v1/transport.proto` for websocket frame schemas.
