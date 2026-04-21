package ws

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream"
	storememory "github.com/go-go-golems/sessionstream/hydration/memory"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	testCommandName = "TestStart"
	testEventName   = "TestEvent"
	testUIEventName = "TestUIEvent"
	testEntityKind  = "TestEntity"
)

type frameMap map[string]any

func TestServerSubscribeEmptySnapshotThenLive(t *testing.T) {
	hub, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer conn.Close()

	hello := readFrame(t, conn)
	require.Equal(t, frameTypeHello, hello["type"])
	require.NotEmpty(t, hello["connectionId"])

	require.NoError(t, conn.WriteJSON(map[string]any{"type": "subscribe", "sessionId": "s-1", "sinceOrdinal": "0"}))
	snapshot := readFrame(t, conn)
	require.Equal(t, frameTypeSnapshot, snapshot["type"])
	require.Equal(t, "s-1", snapshot["sessionId"])
	require.Equal(t, "0", snapshot["ordinal"])
	require.Empty(t, snapshot["entities"])

	subscribed := readFrame(t, conn)
	require.Equal(t, frameTypeSubscribed, subscribed["type"])

	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-1"), testCommandName, payload))

	live := readFrame(t, conn)
	require.Equal(t, frameTypeUIEvent, live["type"])
	require.Equal(t, "s-1", live["sessionId"])
	require.Equal(t, "1", live["ordinal"])
	require.Equal(t, testUIEventName, live["name"])
}

func TestServerReconnectGetsSnapshotThenNextLive(t *testing.T) {
	hub, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	payload1, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload1))

	conn := dialWS(t, httpServer.URL)
	_ = readFrame(t, conn) // hello
	require.NoError(t, conn.WriteJSON(map[string]any{"type": "subscribe", "sessionId": "s-2", "sinceOrdinal": "0"}))
	snapshot := readFrame(t, conn)
	require.Equal(t, frameTypeSnapshot, snapshot["type"])
	require.Equal(t, "1", snapshot["ordinal"])
	readFrame(t, conn) // subscribed
	require.NoError(t, conn.Close())

	payload2, err := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload2))

	reconnected := dialWS(t, httpServer.URL)
	defer reconnected.Close()
	_ = readFrame(t, reconnected) // hello
	require.NoError(t, reconnected.WriteJSON(map[string]any{"type": "subscribe", "sessionId": "s-2", "sinceOrdinal": "1"}))
	snapshot2 := readFrame(t, reconnected)
	require.Equal(t, frameTypeSnapshot, snapshot2["type"])
	require.Equal(t, "2", snapshot2["ordinal"])
	readFrame(t, reconnected) // subscribed

	payload3, err := structpb.NewStruct(map[string]any{"text": "three"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload3))

	live := readFrame(t, reconnected)
	require.Equal(t, frameTypeUIEvent, live["type"])
	require.Equal(t, "3", live["ordinal"])
}

func TestServerConnectionsTracksSubscriptions(t *testing.T) {
	_, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer conn.Close()
	_ = readFrame(t, conn) // hello
	require.NoError(t, conn.WriteJSON(map[string]any{"type": "subscribe", "sessionId": "s-3"}))
	_ = readFrame(t, conn) // snapshot
	_ = readFrame(t, conn) // subscribed

	infos := server.Connections()
	require.Len(t, infos, 1)
	require.Equal(t, []string{"s-3"}, infos[0].Subscriptions)
}

func newTestHubAndServer(t *testing.T) (*sessionstream.Hub, *Server) {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterCommand(testCommandName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterEvent(testEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterUIEvent(testUIEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity(testEntityKind, &structpb.Struct{}))

	store := storememory.New()
	server, err := NewServer(snapshotAdapter{store: store})
	require.NoError(t, err)

	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithUIFanout(server),
	)
	require.NoError(t, err)
	registerTestFlow(t, hub)
	return hub, server
}

func registerTestFlow(t *testing.T, hub *sessionstream.Hub) {
	t.Helper()
	require.NoError(t, hub.RegisterCommand(testCommandName, func(ctx context.Context, cmd sessionstream.Command, _ *sessionstream.Session, pub sessionstream.EventPublisher) error {
		return pub.Publish(ctx, sessionstream.Event{Name: testEventName, SessionId: cmd.SessionId, Payload: cmd.Payload})
	}))
	require.NoError(t, hub.RegisterUIProjection(sessionstream.UIProjectionFunc(func(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
		return []sessionstream.UIEvent{{Name: testUIEventName, Payload: ev.Payload}}, nil
	})))
	require.NoError(t, hub.RegisterTimelineProjection(sessionstream.TimelineProjectionFunc(func(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
		return []sessionstream.TimelineEntity{{Kind: testEntityKind, Id: string(ev.SessionId), Payload: ev.Payload}}, nil
	})))
}

type snapshotAdapter struct{ store *storememory.Store }

func (a snapshotAdapter) Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
	return a.store.Snapshot(ctx, sid, 0)
}

func dialWS(t *testing.T, rawURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + rawURL[len("http"):] // http:// -> ws://
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	return conn
}

func readFrame(t *testing.T, conn *websocket.Conn) frameMap {
	t.Helper()
	var frame frameMap
	require.NoError(t, conn.ReadJSON(&frame))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	return frame
}
