# sessionstream Goja module

`pkg/js/modules/sessionstream` exposes the core sessionstream runtime to Goja as
`require("sessionstream")`.

Phase-1 coverage:

- `schemas()` wraps `sessionstream.SchemaRegistry`.
- Schema entries can be registered from generated protobuf builder namespace
  tokens, for example `pb.StartInferenceCommand`.
- `hub({ schemas, fanout })` wraps `sessionstream.Hub`.
- `hub.submit(sessionId, commandName, payload)` accepts generated protobuf builder
  values via `protogoja.MessageFromValue`, and also accepts plain JS objects that
  can be decoded by the registered protobuf schema.
- `hub.command(name, fn)` adapts synchronous JavaScript command handlers.
- `publisher.publish(eventName, payload)` publishes typed backend events.
- `hub.uiProjection(fn)` and `hub.timelineProjection(fn)` adapt synchronous JS
  projections with read-only `TimelineView` wrappers.
- `eventEmitterFanout(emitter)` bridges `UIFanout` batches to Goja's Go-native
  `events.EventEmitter` through `jsevents.Manager`.
- `webSocket.server(hub)` wraps the existing `transport/ws.Server` and exposes
  connection introspection; provider-level HTTP mounting is the recommended next
  integration step.
- `TypeScriptModule()` exposes handwritten DTS for xgoja declaration bundles.

Minimal JavaScript:

```js
const ss = require("sessionstream");
const pb = require("sessionstream.examples.chatdemo.v1");

const schemas = ss.schemas()
  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
  .registerUIEvent("ChatMessageAccepted", pb.ChatMessageUpdate)
  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);

const hub = ss.hub({ schemas });

hub.command("ChatStartInference", (cmd, session, pub) => {
  pub.publish("ChatUserMessageAccepted",
    pb.UserMessageAcceptedEvent.builder()
      .messageId("m1-user")
      .role("user")
      .content(cmd.payload.prompt)
      .build());
});
```

Validation:

```bash
go test ./pkg/js/modules/sessionstream/... -count=1
go test ./examples/chatdemo ./examples/goja-chatdemo/... -count=1
```
