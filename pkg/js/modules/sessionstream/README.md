# sessionstream Goja module

`pkg/js/modules/sessionstream` exposes the core sessionstream runtime to Goja as
`require("sessionstream")`.

Phase-1 coverage:

- `schemas()` wraps `sessionstream.SchemaRegistry`.
- Schema entries can be registered from generated protobuf builder namespace
  tokens, for example `pb.StartInferenceCommand`.
- `hub({ schemas, fanout })` wraps `sessionstream.Hub`.
- `hub.submit(sessionId, commandName, payload)` returns a Promise that resolves
  after command handling and local projection work completes. It accepts
  generated protobuf builder values via `protogoja.MessageFromValue`, and also
  accepts plain JS objects that can be decoded by the registered protobuf schema.
- `hub.command(name, fn)` adapts synchronous JavaScript command handlers and
  Promise-returning handlers.
- `publisher.publish(eventName, payload)` publishes typed backend events and
  returns a Promise that resolves after publication/projection completes.
- `hub.uiProjection(fn)` and `hub.timelineProjection(fn)` adapt synchronous or
  Promise-returning JS projections with read-only `TimelineView` wrappers.
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
  return pub.publish("ChatUserMessageAccepted",
    pb.UserMessageAcceptedEvent.builder()
      .messageId("m1-user")
      .role("user")
      .content(cmd.payload.prompt)
      .build());
});
```

JavaScript submit/publish APIs are Promise-native so the JavaScript stack can
unwind while Promises settle on the runtime owner:

```js
hub.command("ChatStartInference", async (cmd, session, pub) => {
  const answer = await model.ask(cmd.payload.prompt);
  await pub.publish("ChatUserMessageAccepted",
    pb.UserMessageAcceptedEvent.builder()
      .messageId("m1-user")
      .role("assistant")
      .content(answer)
      .build());
});

await hub.submit("s-1", "ChatStartInference",
  pb.StartInferenceCommand.builder().prompt("hello").build());
```

Validation:

```bash
go test ./pkg/js/modules/sessionstream/... -count=1
go test ./examples/chatdemo ./examples/goja-chatdemo/... -count=1
```
