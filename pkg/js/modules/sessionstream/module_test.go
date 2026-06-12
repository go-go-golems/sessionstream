package sessionstream

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	chatdemo "github.com/go-go-golems/sessionstream/examples/chatdemo"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	"github.com/stretchr/testify/require"
)

func TestModuleExports(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	reg.Enable(vm)
	value, err := vm.RunString(`
		const ss = require("sessionstream");
		JSON.stringify({
			version: ss.version,
			schemas: typeof ss.schemas,
			hub: typeof ss.hub,
			fanout: typeof ss.eventEmitterFanout,
			ws: typeof ss.webSocket.server,
		});
	`)
	require.NoError(t, err)
	require.JSONEq(t, `{"version":"0.1.0","schemas":"function","hub":"function","fanout":"function","ws":"function"}`, value.String())
}

func TestSchemasRegisterGeneratedPrototypeTokens(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	_, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		const schemas = ss.schemas();
		schemas.registerCommand("ChatStartInference", pb.StartInferenceCommand);
		const hub = ss.hub({ schemas });
		hub.command("ChatStartInference", () => {});
		hub.submit("s-1", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("typed").build());
	`)
	require.NoError(t, err)
}

func TestHubCommandProjectionAndSnapshotFromJavaScript(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	Register(reg, Options{DefaultHydrationStore: store})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)

	value, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		const schemas = ss.schemas()
		  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
		  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
		  .registerUIEvent("ChatMessageAccepted", pb.ChatMessageUpdate)
		  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);
		const hub = ss.hub({ schemas });
		hub.command("ChatStartInference", (cmd, session, pub) => {
		  pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
		    .messageId("m1-user").role("user").content(cmd.payload.prompt).streaming(false).build());
		});
		hub.uiProjection((event) => [{
		  name: "ChatMessageAccepted",
		  payload: pb.ChatMessageUpdate.builder().messageId(event.payload.messageId).role("user").content(event.payload.content).streaming(false).build(),
		}]);
		hub.timelineProjection((event) => [{
		  kind: "ChatMessage",
		  id: event.payload.messageId,
		  payload: pb.ChatMessageEntity.builder().messageId(event.payload.messageId).role("user").content(event.payload.content).status("accepted").streaming(false).build(),
		}]);
		hub.submit("s-js", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("hello from js").build());
		JSON.stringify(hub.snapshot("s-js"));
	`)
	require.NoError(t, err)
	require.Contains(t, value.String(), "hello from js")
	require.Contains(t, value.String(), "m1-user")
}

func TestEventEmitterFanoutReceivesUIBatch(t *testing.T) {
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	var rt *gggengine.Runtime
	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithRuntimeInitializers(jsevents.Install()).
		WithModules(
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{DefaultHydrationStore: store, EventEmitterManagerResolver: func() (*jsevents.Manager, bool) { return jsevents.FromRuntime(rt) }})},
			gggengine.NativeModuleRegistrar{ModuleName: chatdemov1.GojaBuilderFileChatProtoModuleName(), Loader: chatdemov1.NewGojaBuilderFileChatProtoLoader("")},
		).
		Build()
	require.NoError(t, err)
	rt, err = factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })

	_, err = rt.Owner.Call(context.Background(), "sessionstream.fanout.test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			const ss = require("sessionstream");
			const pb = require("sessionstream.examples.chatdemo.v1");
			globalThis.batches = [];
			const ee = new EventEmitter();
			ee.on("ui", batch => globalThis.batches.push(batch));
			const schemas = ss.schemas()
			  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
			  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
			  .registerUIEvent("ChatMessageAccepted", pb.ChatMessageUpdate)
			  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);
			const hub = ss.hub({ schemas, fanout: ss.eventEmitterFanout(ee) });
			hub.command("ChatStartInference", (cmd, session, pub) => pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder().messageId("m1-user").role("user").content(cmd.payload.prompt).build()));
			hub.uiProjection((event) => [{ name: "ChatMessageAccepted", payload: pb.ChatMessageUpdate.builder().messageId(event.payload.messageId).role("user").content(event.payload.content).build() }]);
			hub.timelineProjection((event) => []);
			hub.submit("s-ee", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("fanout").build());
		`)
		return nil, err
	})
	require.NoError(t, err)
	_, err = rt.Owner.Call(context.Background(), "sessionstream.fanout.assert", func(_ context.Context, vm *goja.Runtime) (any, error) {
		value, err := vm.RunString(`JSON.stringify(globalThis.batches)`)
		if err != nil {
			return nil, err
		}
		require.Contains(t, value.String(), "fanout")
		return nil, nil
	})
	require.NoError(t, err)
}

func TestTypeScriptModuleDescriptor(t *testing.T) {
	mod := TypeScriptModule()
	require.Equal(t, ModuleName, mod.Name)
	require.NotEmpty(t, mod.RawDTS)
}

func TestMessageFromValueStillWorksForGeneratedBuilder(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	value, err := vm.RunString(`require("sessionstream.examples.chatdemo.v1").StartInferenceCommand.builder().prompt("x").build()`)
	require.NoError(t, err)
	msg, ok := protogoja.MessageFromValue(value)
	require.True(t, ok)
	require.Equal(t, "x", msg.(*chatdemov1.StartInferenceCommand).GetPrompt())
}
