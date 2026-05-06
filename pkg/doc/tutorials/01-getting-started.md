---
Title: "Getting Started with Sessionstream"
Slug: "sessionstream-getting-started"
Short: "Build a small sessionstream app by registering typed schemas, installing a hub, publishing events, and reading a snapshot."
Topics:
  - sessionstream
  - getting-started
  - commands
  - projections
  - hydration
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

This guide walks through the smallest useful `sessionstream` application. The goal is not to build a production service. The goal is to see the shape of the framework: commands enter a session, handlers publish backend events, projections derive UI and timeline state, and snapshots make the result recoverable.

The runnable reference is `examples/chatdemo`. Use that example when you want complete code with generated protobuf messages and tests.

## What you will build

You will wire a small chat-style service that accepts a prompt and produces timeline state. The moving pieces are:

1. A `SchemaRegistry` with concrete protobuf payloads.
2. A `Hub` configured with that registry.
3. Command handlers that publish backend events.
4. UI and timeline projections.
5. A snapshot call that reads hydrated state.

## Step 1: Register payload schemas

Every command, backend event, UI event, and timeline entity has a name and a concrete protobuf payload type. The schema registry stores those prototypes so the framework can validate, marshal, unmarshal, and hydrate payloads consistently.

```go
reg := sessionstream.NewSchemaRegistry()
if err := chatdemo.RegisterSchemas(reg); err != nil {
    return err
}
```

The reference implementation registers messages like this:

```go
reg.RegisterCommand(chatdemo.CommandStartInference, &chatdemov1.StartInferenceCommand{})
reg.RegisterEvent(chatdemo.EventInferenceStarted, &chatdemov1.InferenceStartedEvent{})
reg.RegisterUIEvent(chatdemo.UIMessageStarted, &chatdemov1.ChatMessageUpdate{})
reg.RegisterTimelineEntity(chatdemo.TimelineEntityChatMessage, &chatdemov1.ChatMessageEntity{})
```

Use concrete protobuf messages. Do not register top-level `*structpb.Struct`; `sessionstream-lint` rejects those registrations.

## Step 2: Create the hub

The `Hub` is the routing center. It receives commands, looks up handlers, gives handlers an event publisher, runs projections, applies timeline entities, and fans out UI events.

```go
hub, err := sessionstream.NewHub(
    sessionstream.WithSchemaRegistry(reg),
)
if err != nil {
    return err
}
```

The default hub uses an in-memory/no-op hydration store. Production or restart-sensitive applications should install a real store such as the SQLite hydration implementation.

## Step 3: Install handlers and projections

Handlers and projections are application code. `sessionstream` does not know what a chat message, inventory widget, or workflow step means. It only knows how to route and apply the events those features publish.

```go
engine := chatdemo.NewEngine()
if err := chatdemo.Install(hub, engine); err != nil {
    return err
}
```

Inside the installer, the application registers command handlers and projections:

```go
hub.RegisterCommand(chatdemo.CommandStartInference, engine.handleStartInference)
hub.RegisterUIProjection(sessionstream.UIProjectionFunc(uiProjection))
hub.RegisterTimelineProjection(sessionstream.TimelineProjectionFunc(timelineProjection))
```

A handler publishes events instead of returning UI state:

```go
return pub.Publish(ctx, sessionstream.Event{
    Name:      chatdemo.EventUserMessageAccepted,
    SessionId: cmd.SessionId,
    Payload: &chatdemov1.UserMessageAcceptedEvent{
        MessageId: userMessageID,
        Role:      "user",
        Content:   prompt,
    },
})
```

That choice is the core design. Events are the canonical history. UI and timeline state are projections of that history.

## Step 4: Submit a command

Commands always belong to a session. The same command name can be submitted to different sessions without sharing timeline state.

```go
service, err := chatdemo.NewService(hub, engine)
if err != nil {
    return err
}

if err := service.SubmitPrompt(ctx, "session-1", "Explain ordinals"); err != nil {
    return err
}
```

The command path is:

```text
Submit -> Hub -> CommandHandler -> EventPublisher -> backend Event -> projections -> UIEvent + TimelineEntity
```

## Step 5: Read a snapshot

A snapshot returns the current hydrated timeline state for a session.

```go
snapshot, err := service.Snapshot(ctx, "session-1")
if err != nil {
    return err
}
```

Snapshots are what make reconnects work. A websocket client receives the current snapshot first and then future live UI events.

## Complete local validation

From the Sessionstream repository:

```bash
go test ./examples/chatdemo -count=1
make test
```

If you are changing schema registrations, also run:

```bash
make schema-vet
```

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `unknown command` | The command name was not registered with the hub. | Call `hub.RegisterCommand` during installation. |
| `payload type mismatch` | The submitted payload does not match the schema registry prototype. | Submit the generated protobuf type registered for that command. |
| No UI events appear | No UI projection is registered, or the projection ignores the backend event name. | Register `UIProjectionFunc` and handle the event name explicitly. |
| Snapshot is empty | No timeline projection emitted entities, or the store is no-op for your scenario. | Register a timeline projection and use a hydration store appropriate for your app. |
| `sessionstream-lint` rejects `*structpb.Struct` | A top-level schema registration is generic. | Define a concrete protobuf message and register that type. |

## See Also

- `sessionstream-user-guide` for the broader mental model.
- `sessionstream-reference` for API and package responsibilities.
- `sessionstream-schema-vet-playbook` for schema-vet usage.
- `examples/chatdemo/chat.go` for a complete small application.
