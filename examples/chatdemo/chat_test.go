package chatdemo

import (
	"context"
	"testing"
	"time"

	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	"github.com/stretchr/testify/require"
)

func TestChatDemoHappyPath(t *testing.T) {
	engine := NewEngine(WithChunkDelay(time.Millisecond))
	hub := newTestHub(t, engine)
	payload := &chatdemov1.StartInferenceCommand{Prompt: "Explain ordinals"}
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-1"), CommandStartInference, payload))
	require.NoError(t, engine.WaitIdle(context.Background(), sessionstream.SessionId("chat-1")))

	snap, err := hub.Snapshot(context.Background(), sessionstream.SessionId("chat-1"))
	require.NoError(t, err)
	require.Equal(t, uint64(6), snap.SnapshotOrdinal)
	require.Len(t, snap.Entities, 2)

	assistant, ok := findEntity(snap, "chat-msg-1")
	require.True(t, ok)
	assistantPayload := assistant.Payload.(*chatdemov1.ChatMessageEntity)
	require.Equal(t, "finished", assistantPayload.GetStatus())
	require.Equal(t, "Answer: Explain ordinals", assistantPayload.GetText())

	user, ok := findEntity(snap, "chat-msg-1-user")
	require.True(t, ok)
	userPayload := user.Payload.(*chatdemov1.ChatMessageEntity)
	require.Equal(t, "user", userPayload.GetRole())
	require.Equal(t, "Explain ordinals", userPayload.GetContent())
}

func TestChatDemoStopPath(t *testing.T) {
	engine := NewEngine(WithChunkDelay(10 * time.Millisecond))
	hub := newTestHub(t, engine)
	payload := &chatdemov1.StartInferenceCommand{Prompt: "Stop me"}
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-2"), CommandStartInference, payload))
	time.Sleep(12 * time.Millisecond)
	stop := &chatdemov1.StopInferenceCommand{}
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-2"), CommandStopInference, stop))
	require.NoError(t, engine.WaitIdle(context.Background(), sessionstream.SessionId("chat-2")))

	snap, err := hub.Snapshot(context.Background(), sessionstream.SessionId("chat-2"))
	require.NoError(t, err)
	assistant, ok := findEntity(snap, "chat-msg-1")
	require.True(t, ok)
	assistantPayload := assistant.Payload.(*chatdemov1.ChatMessageEntity)
	require.Equal(t, "stopped", assistantPayload.GetStatus())
	require.Equal(t, false, assistantPayload.GetStreaming())
}

func newTestHub(t *testing.T, engine *Engine) *sessionstream.Hub {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, RegisterSchemas(reg))
	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
	)
	require.NoError(t, err)
	require.NoError(t, Install(hub, engine))
	return hub
}

func findEntity(snap sessionstream.Snapshot, id string) (sessionstream.TimelineEntity, bool) {
	for _, entity := range snap.Entities {
		if entity.Id == id {
			return entity, true
		}
	}
	return sessionstream.TimelineEntity{}, false
}
