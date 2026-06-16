package sessionstream

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/modules/express"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
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

func TestSchemasBulkRegisterGeneratedPrototypeNamespaces(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	value, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		const schemas = ss.schemas({
		  commands: { ChatStartInference: pb.StartInferenceCommand },
		  events: { ChatUserMessageAccepted: pb.UserMessageAcceptedEvent },
		  uiEvents: { ChatMessageAccepted: pb.ChatMessageUpdate },
		  entities: { ChatMessage: pb.ChatMessageEntity },
		});
		const hub = ss.hub({ schemas });
		hub.command("ChatStartInference", (cmd, session, pub) => {
		  return pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
		    .messageId("m-bulk").role("user").content(cmd.payload.prompt).streaming(false).build());
		});
		hub.uiProjection((event) => [{
		  name: "ChatMessageAccepted",
		  payload: pb.ChatMessageUpdate.builder()
		    .messageId(event.payload.messageId).role("user").content(event.payload.content).streaming(false).build(),
		}]);
		hub.timelineProjection((event) => [{
		  kind: "ChatMessage",
		  id: event.payload.messageId,
		  payload: pb.ChatMessageEntity.builder()
		    .messageId(event.payload.messageId).role("user").content(event.payload.content).status("accepted").streaming(false).build(),
		}]);
		hub.submit("s-bulk", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("typed").build());
	`)
	require.NoError(t, err)
	requireFulfilledPromise(t, value)
}

func TestSchemasBulkRegisterStringFullNames(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	value, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		const schemas = ss.schemas({
		  commands: { ChatStartInference: "sessionstream.examples.chatdemo.v1.StartInferenceCommand" },
		});
		const hub = ss.hub({ schemas });
		hub.command("ChatStartInference", () => {});
		hub.submit("s-string", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("typed").build());
	`)
	require.NoError(t, err)
	requireFulfilledPromise(t, value)
}

func TestSchemasRejectPlainObjectDescriptors(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec string
	}{
		{name: "typeName", spec: `{ typeName: "sessionstream.examples.chatdemo.v1.StartInferenceCommand" }`},
		{name: "type", spec: `{ type: "sessionstream.examples.chatdemo.v1.StartInferenceCommand" }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vm := goja.New()
			reg := noderequire.NewRegistry()
			Register(reg, Options{})
			require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
			reg.Enable(vm)
			_, err := vm.RunString(`
				const ss = require("sessionstream");
				ss.schemas({ commands: { ChatStartInference: ` + tc.spec + ` } });
			`)
			require.Error(t, err)
			require.Contains(t, err.Error(), "schema must be a generated message namespace or protobuf full name")
		})
	}
}

func requireFulfilledPromise(t *testing.T, value goja.Value) {
	t.Helper()
	promise, ok := value.Export().(*goja.Promise)
	require.True(t, ok, "expected a Promise, got %T", value.Export())
	require.Equal(t, goja.PromiseStateFulfilled, promise.State(), "promise result: %s", jsValueString(promise.Result()))
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

func TestHubPublishEventFromJavaScript(t *testing.T) {
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
		  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
		  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);
		const hub = ss.hub({ schemas });
		hub.timelineProjection((event) => [{
		  kind: "ChatMessage",
		  id: event.payload.messageId,
		  payload: pb.ChatMessageEntity.builder().messageId(event.payload.messageId).role("user").content(event.payload.content).status("accepted").streaming(false).build(),
		}]);
		hub.publish("s-publish", "ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
		  .messageId("m-publish").role("user").content("published from js").streaming(false).build());
		JSON.stringify(hub.snapshot("s-publish"));
	`)
	require.NoError(t, err)
	require.Contains(t, value.String(), "published from js")
	require.Contains(t, value.String(), "m-publish")
}

func TestHubPromiseAwareCallbacksFromJavaScript(t *testing.T) {
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{DefaultHydrationStore: store})},
			gggengine.NativeModuleRegistrar{ModuleName: chatdemov1.GojaBuilderFileChatProtoModuleName(), Loader: chatdemov1.NewGojaBuilderFileChatProtoLoader("")},
		).
		Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := rt.Owner.Call(ctx, "sessionstream.promise.success", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`(async () => {
			const ss = require("sessionstream");
			const pb = require("sessionstream.examples.chatdemo.v1");
			const schemas = ss.schemas()
			  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
			  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
			  .registerUIEvent("ChatMessageAccepted", pb.ChatMessageUpdate)
			  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);
			const hub = ss.hub({ schemas });
			hub.command("ChatStartInference", async (cmd, session, pub) => {
			  const prompt = await Promise.resolve(cmd.payload.prompt + ":async-command");
			  await pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
			    .messageId("m-async").role("user").content(prompt).streaming(false).build());
			});
			hub.uiProjection(async (event) => {
			  const content = await Promise.resolve(event.payload.content + ":async-ui");
			  return [{
			    name: "ChatMessageAccepted",
			    payload: pb.ChatMessageUpdate.builder().messageId(event.payload.messageId).role("user").content(content).streaming(false).build(),
			  }];
			});
			hub.timelineProjection(async (event) => {
			  const content = await Promise.resolve(event.payload.content + ":async-timeline");
			  return [{
			    kind: "ChatMessage",
			    id: event.payload.messageId,
			    payload: pb.ChatMessageEntity.builder().messageId(event.payload.messageId).role("user").content(content).status("accepted").streaming(false).build(),
			  }];
			});
			await hub.submit("s-async", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("hello").build());
			return JSON.stringify(hub.snapshot("s-async"));
		})()`)
	})
	require.NoError(t, err)
	resolved, err := waitTestPromise(ctx, rt, value.(goja.Value))
	require.NoError(t, err)
	require.Contains(t, resolved.String(), "hello:async-command:async-timeline")
	require.Contains(t, resolved.String(), "m-async")
}

func TestHubPromiseRejectedCommandReturnsError(t *testing.T) {
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{DefaultHydrationStore: store})},
			gggengine.NativeModuleRegistrar{ModuleName: chatdemov1.GojaBuilderFileChatProtoModuleName(), Loader: chatdemov1.NewGojaBuilderFileChatProtoLoader("")},
		).
		Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := rt.Owner.Call(ctx, "sessionstream.promise.reject", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`(async () => {
			const ss = require("sessionstream");
			const pb = require("sessionstream.examples.chatdemo.v1");
			const schemas = ss.schemas().registerCommand("ChatStartInference", pb.StartInferenceCommand);
			const hub = ss.hub({ schemas });
			hub.command("ChatStartInference", async () => {
			  throw new Error("async boom");
			});
			await hub.submit("s-reject", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("hello").build());
		})()`)
	})
	require.NoError(t, err)
	_, err = waitTestPromise(ctx, rt, value.(goja.Value))
	require.Error(t, err)
	require.Contains(t, err.Error(), "async boom")
}

func TestHubPromiseRejectedProjectionReturnsError(t *testing.T) {
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{DefaultHydrationStore: store})},
			gggengine.NativeModuleRegistrar{ModuleName: chatdemov1.GojaBuilderFileChatProtoModuleName(), Loader: chatdemov1.NewGojaBuilderFileChatProtoLoader("")},
		).
		Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })

	cases := []struct {
		name       string
		setup      string
		wantLabel  string
		wantReason string
	}{
		{
			name: "ui projection",
			setup: `
				hub.uiProjection(async () => {
				  await Promise.resolve();
				  throw new Error("ui boom");
				});
				hub.timelineProjection(() => []);
			`,
			wantLabel:  "sessionstream.uiProjection.ChatUserMessageAccepted",
			wantReason: "ui boom",
		},
		{
			name: "timeline projection",
			setup: `
				hub.uiProjection(() => []);
				hub.timelineProjection(async () => {
				  await Promise.resolve();
				  throw new Error("timeline boom");
				});
			`,
			wantLabel:  "sessionstream.timelineProjection.ChatUserMessageAccepted",
			wantReason: "timeline boom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			value, err := rt.Owner.Call(ctx, "sessionstream.projection.reject", func(_ context.Context, vm *goja.Runtime) (any, error) {
				return vm.RunString(`(async () => {
					const ss = require("sessionstream");
					const pb = require("sessionstream.examples.chatdemo.v1");
					const schemas = ss.schemas()
					  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
					  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent);
					const hub = ss.hub({ schemas });
					hub.command("ChatStartInference", (cmd, session, pub) => {
					  return pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
					    .messageId("m-reject").role("user").content(cmd.payload.prompt).streaming(false).build());
					});
					` + tc.setup + `
					await hub.submit("s-reject", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("hello").build());
				})()`)
			})
			require.NoError(t, err)
			_, err = waitTestPromise(ctx, rt, value.(goja.Value))
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantLabel)
			require.Contains(t, err.Error(), tc.wantReason)
		})
	}
}

func waitTestPromise(ctx context.Context, rt *gggengine.Runtime, value goja.Value) (goja.Value, error) {
	promise, ok := value.Export().(*goja.Promise)
	if !ok {
		return value, nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		ret, err := rt.Owner.Call(ctx, "sessionstream.test.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{State: promise.State(), Result: promise.Result()}, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("promise rejected: %s", jsValueString(snapshot.Result))
		case goja.PromiseStateFulfilled:
			return snapshot.Result, nil
		}
	}
}

func TestWebSocketServerMountsInExpress(t *testing.T) {
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(
			express.NewRegistrar(host),
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{})},
		).
		Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })
	host.SetRuntime(rt.Owner)

	_, err = rt.Owner.Call(context.Background(), "sessionstream.websocket.mount", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const ss = require("sessionstream");
			const app = express.app();
			const schemas = ss.schemas();
			const hub = ss.hub({ schemas });
			app.mount("/ws", ss.webSocket.server(hub));
		`)
		return nil, err
	})
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	host.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ws/rooms/general", nil))
	require.NotEqual(t, http.StatusNotFound, rr.Code)
}

func TestHubSubmitFromExpressHandlerUsesCurrentOwnerContext(t *testing.T) {
	registry := ss.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(registry))
	store, err := storesqlite.NewInMemory(registry)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithModules(
			express.NewRegistrar(host),
			gggengine.NativeModuleRegistrar{ModuleName: ModuleName, Loader: NewLoader(Options{DefaultHydrationStore: store})},
			gggengine.NativeModuleRegistrar{ModuleName: chatdemov1.GojaBuilderFileChatProtoModuleName(), Loader: chatdemov1.NewGojaBuilderFileChatProtoLoader("")},
		).
		Build()
	require.NoError(t, err)
	rt, err := factory.NewRuntime(gggengine.WithStartupContext(context.Background()), gggengine.WithLifetimeContext(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rt.Close(context.Background())) })
	host.SetRuntime(rt.Owner)

	_, err = rt.Owner.Call(context.Background(), "sessionstream.express.submit.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const express = require("express");
			const ss = require("sessionstream");
			const pb = require("sessionstream.examples.chatdemo.v1");
			const app = express.app();
			const schemas = ss.schemas()
			  .registerCommand("ChatStartInference", pb.StartInferenceCommand)
			  .registerEvent("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent)
			  .registerUIEvent("ChatMessageAccepted", pb.ChatMessageUpdate)
			  .registerTimelineEntity("ChatMessage", pb.ChatMessageEntity);
			const hub = ss.hub({ schemas });
			hub.command("ChatStartInference", (cmd, _session, pub) => {
			  return pub.publish("ChatUserMessageAccepted", pb.UserMessageAcceptedEvent.builder()
			    .messageId("m-http").role("user").content(cmd.payload.prompt).streaming(false).build());
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
			app.post("/api/chat", async (_req, res) => {
			  await hub.submit("s-http", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("from http").build());
			  res.json({ ok: true, snapshot: hub.snapshot("s-http") });
			});
		`)
		return nil, err
	})
	require.NoError(t, err)

	type result struct {
		code int
		body string
	}
	done := make(chan result, 1)
	go func() {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(`{}`))
		req.Header.Set("content-type", "application/json")
		host.ServeHTTP(rr, req)
		done <- result{code: rr.Code, body: rr.Body.String()}
	}()

	select {
	case got := <-done:
		require.Equal(t, http.StatusOK, got.code, got.body)
		require.Contains(t, got.body, `"ok":true`)
		require.True(t, strings.Contains(got.body, "from http"), got.body)
	case <-time.After(2 * time.Second):
		t.Fatal("POST /api/chat deadlocked while hub.submit re-entered the Goja runtime owner")
	}
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

	value, err := rt.Owner.Call(context.Background(), "sessionstream.fanout.test", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`(async () => {
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
			await hub.submit("s-ee", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("fanout").build());
		})()`)
	})
	require.NoError(t, err)
	_, err = waitTestPromise(context.Background(), rt, value.(goja.Value))
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
