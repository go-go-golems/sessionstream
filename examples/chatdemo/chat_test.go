package chatdemo

import (
	"context"
	"testing"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storememory "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/memory"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestChatDemoHappyPath(t *testing.T) {
	engine := NewEngine(WithChunkDelay(time.Millisecond))
	hub := newTestHub(t, engine)
	payload, err := structpb.NewStruct(map[string]any{"prompt": "Explain ordinals"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-1"), CommandStartInference, payload))
	require.NoError(t, engine.WaitIdle(context.Background(), sessionstream.SessionId("chat-1")))

	snap, err := hub.Snapshot(context.Background(), sessionstream.SessionId("chat-1"))
	require.NoError(t, err)
	require.Equal(t, uint64(6), snap.Ordinal)
	require.Len(t, snap.Entities, 2)

	assistant, ok := findEntity(snap, "chat-msg-1")
	require.True(t, ok)
	assistantPayload := assistant.Payload.(*structpb.Struct).AsMap()
	require.Equal(t, "finished", assistantPayload["status"])
	require.Equal(t, "Answer: Explain ordinals", assistantPayload["text"])

	user, ok := findEntity(snap, "chat-msg-1-user")
	require.True(t, ok)
	userPayload := user.Payload.(*structpb.Struct).AsMap()
	require.Equal(t, "user", userPayload["role"])
	require.Equal(t, "Explain ordinals", userPayload["content"])
}

func TestChatDemoStopPath(t *testing.T) {
	engine := NewEngine(WithChunkDelay(10 * time.Millisecond))
	hub := newTestHub(t, engine)
	payload, err := structpb.NewStruct(map[string]any{"prompt": "Stop me"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-2"), CommandStartInference, payload))
	time.Sleep(12 * time.Millisecond)
	stop, err := structpb.NewStruct(map[string]any{})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("chat-2"), CommandStopInference, stop))
	require.NoError(t, engine.WaitIdle(context.Background(), sessionstream.SessionId("chat-2")))

	snap, err := hub.Snapshot(context.Background(), sessionstream.SessionId("chat-2"))
	require.NoError(t, err)
	assistant, ok := findEntity(snap, "chat-msg-1")
	require.True(t, ok)
	assistantPayload := assistant.Payload.(*structpb.Struct).AsMap()
	require.Equal(t, "stopped", assistantPayload["status"])
	require.Equal(t, false, assistantPayload["streaming"])
}

func newTestHub(t *testing.T, engine *Engine) *sessionstream.Hub {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, RegisterSchemas(reg))
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(storememory.New()),
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
