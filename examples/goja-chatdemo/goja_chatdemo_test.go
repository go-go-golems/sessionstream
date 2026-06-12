package gojachatdemo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	chatdemo "github.com/go-go-golems/sessionstream/examples/chatdemo"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	"github.com/stretchr/testify/require"
)

func TestGeneratedBuilderBuildsChatdemoCommandFromJavaScript(t *testing.T) {
	vm, exports := runChatdemoScript(t)
	_ = vm

	msg, ok := protogoja.MessageFromValue(exports.Get("command"))
	require.True(t, ok, "script command export should carry a Go proto.Message ref")
	cmd, ok := msg.(*chatdemov1.StartInferenceCommand)
	require.True(t, ok, "command export type = %T", msg)
	require.Equal(t, "Explain ordinals", cmd.GetPrompt())
}

func TestJavaScriptBuiltCommandRunsThroughChatdemoHub(t *testing.T) {
	_, exports := runChatdemoScript(t)
	msg, ok := protogoja.MessageFromValue(exports.Get("command"))
	require.True(t, ok, "script command export should carry a Go proto.Message ref")
	cmd, ok := msg.(*chatdemov1.StartInferenceCommand)
	require.True(t, ok, "command export type = %T", msg)

	fanout := &recordingFanout{}
	engine := chatdemo.NewEngine(chatdemo.WithChunkDelay(time.Millisecond))
	hub := newChatdemoHub(t, engine, fanout)

	ctx := context.Background()
	sid := sessionstream.SessionId("goja-chatdemo-1")
	require.NoError(t, hub.Submit(ctx, sid, chatdemo.CommandStartInference, cmd))
	require.NoError(t, engine.WaitIdle(ctx, sid))

	snap, err := hub.Snapshot(ctx, sid)
	require.NoError(t, err)
	require.GreaterOrEqual(t, snap.SnapshotOrdinal, uint64(2))
	require.Len(t, snap.Entities, 2)

	user := findEntity(t, snap, "chat-msg-1-user")
	userPayload, ok := user.Payload.(*chatdemov1.ChatMessageEntity)
	require.True(t, ok, "user entity payload type = %T", user.Payload)
	require.Equal(t, "user", userPayload.GetRole())
	require.Equal(t, "Explain ordinals", userPayload.GetContent())

	assistant := findEntity(t, snap, "chat-msg-1")
	assistantPayload, ok := assistant.Payload.(*chatdemov1.ChatMessageEntity)
	require.True(t, ok, "assistant entity payload type = %T", assistant.Payload)
	require.Equal(t, "finished", assistantPayload.GetStatus())
	require.Equal(t, "Answer: Explain ordinals", assistantPayload.GetText())

	batches := fanout.Batches()
	require.NotEmpty(t, batches, "chatdemo should publish UI batches through fanout")
	require.Equal(t, sid, batches[0].SessionID)
	require.NotEmpty(t, batches[0].Events)
	firstPayload, ok := batches[0].Events[0].Payload.(*chatdemov1.ChatMessageUpdate)
	require.True(t, ok, "first UI event payload type = %T", batches[0].Events[0].Payload)
	require.Equal(t, "Explain ordinals", firstPayload.GetContent())
}

func runChatdemoScript(t *testing.T) (*goja.Runtime, *goja.Object) {
	t.Helper()
	vm := goja.New()
	registry := noderequire.NewRegistry()
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(registry, ""))
	registry.Enable(vm)

	script, err := os.ReadFile(filepath.Join("scripts", "start-inference.js"))
	require.NoError(t, err)
	moduleObj := vm.NewObject()
	exports := vm.NewObject()
	require.NoError(t, moduleObj.Set("exports", exports))
	wrapped := "(function(exports, require) {\n" + string(script) + "\n})"
	fnValue, err := vm.RunString(wrapped)
	require.NoError(t, err)
	fn, ok := goja.AssertFunction(fnValue)
	require.True(t, ok, "wrapped script should be callable")
	_, err = fn(goja.Undefined(), exports, vm.Get("require"))
	require.NoError(t, err)
	return vm, exports
}

func newChatdemoHub(t *testing.T, engine *chatdemo.Engine, fanout sessionstream.UIFanout) *sessionstream.Hub {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, chatdemo.RegisterSchemas(reg))
	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithUIFanout(fanout),
	)
	require.NoError(t, err)
	require.NoError(t, chatdemo.Install(hub, engine))
	return hub
}

func findEntity(t *testing.T, snap sessionstream.Snapshot, id string) sessionstream.TimelineEntity {
	t.Helper()
	for _, entity := range snap.Entities {
		if entity.Id == id {
			return entity
		}
	}
	t.Fatalf("entity %s not found in snapshot: %#v", id, snap.Entities)
	return sessionstream.TimelineEntity{}
}

type uiBatch struct {
	SessionID sessionstream.SessionId
	Ordinal   uint64
	Events    []sessionstream.UIEvent
}

type recordingFanout struct {
	mu      sync.Mutex
	batches []uiBatch
}

func (f *recordingFanout) PublishUI(_ context.Context, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := append([]sessionstream.UIEvent(nil), events...)
	f.batches = append(f.batches, uiBatch{SessionID: sid, Ordinal: ord, Events: copied})
	return nil
}

func (f *recordingFanout) Batches() []uiBatch {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uiBatch(nil), f.batches...)
}

func TestScriptUsesGeneratedChatdemoModuleName(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("scripts", "start-inference.js"))
	require.NoError(t, err)
	require.True(t, strings.Contains(string(script), chatdemov1.GojaBuilderFileChatProtoModuleName()))
}
