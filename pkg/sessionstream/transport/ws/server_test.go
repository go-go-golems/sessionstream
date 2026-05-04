package ws

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	sessionstreamv1 "github.com/go-go-golems/sessionstream/pkg/sessionstream/pb/proto/sessionstream/v1"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	testCommandName = "TestStart"
	testEventName   = "TestEvent"
	testUIEventName = "TestUIEvent"
	testEntityKind  = "TestEntity"
)

func TestServerSubscribeEmptySnapshotThenLive(t *testing.T) {
	hub, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()

	hello := readServerFrame(t, conn)
	require.NotNil(t, hello.GetHello())
	require.NotEmpty(t, hello.GetHello().GetConnectionId())

	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-1", SinceSnapshotOrdinal: 0}}})
	snapshot := readServerFrame(t, conn)
	require.NotNil(t, snapshot.GetSnapshot())
	require.Equal(t, "s-1", snapshot.GetSnapshot().GetSessionId())
	require.Equal(t, uint64(0), snapshot.GetSnapshot().GetSnapshotOrdinal())
	require.Empty(t, snapshot.GetSnapshot().GetEntities())

	subscribed := readServerFrame(t, conn)
	require.NotNil(t, subscribed.GetSubscribed())
	require.Equal(t, uint64(0), subscribed.GetSubscribed().GetSinceSnapshotOrdinal())

	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-1"), testCommandName, payload))

	live := readServerFrame(t, conn)
	require.NotNil(t, live.GetUiEvent())
	require.Equal(t, "s-1", live.GetUiEvent().GetSessionId())
	require.Equal(t, uint64(1), live.GetUiEvent().GetEventOrdinal())
	require.Equal(t, testUIEventName, live.GetUiEvent().GetName())
}

func TestServerReconnectGetsSnapshotThenNextLive(t *testing.T) {
	hub, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	payload1, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload1))

	conn := dialWS(t, httpServer.URL)
	_ = readServerFrame(t, conn) // hello
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-2", SinceSnapshotOrdinal: 0}}})
	snapshot := readServerFrame(t, conn)
	require.Equal(t, uint64(1), snapshot.GetSnapshot().GetSnapshotOrdinal())
	_ = readServerFrame(t, conn) // subscribed
	require.NoError(t, conn.Close())

	payload2, err := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload2))

	reconnected := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, reconnected.Close()) }()
	_ = readServerFrame(t, reconnected) // hello
	writeClientFrame(t, reconnected, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-2", SinceSnapshotOrdinal: 1}}})
	snapshot2 := readServerFrame(t, reconnected)
	require.Equal(t, uint64(2), snapshot2.GetSnapshot().GetSnapshotOrdinal())
	_ = readServerFrame(t, reconnected) // subscribed

	payload3, err := structpb.NewStruct(map[string]any{"text": "three"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), sessionstream.SessionId("s-2"), testCommandName, payload3))

	live := readServerFrame(t, reconnected)
	require.Equal(t, uint64(3), live.GetUiEvent().GetEventOrdinal())
}

func TestServerConnectionsTracksSubscriptions(t *testing.T) {
	_, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-3"}}})
	_ = readServerFrame(t, conn) // snapshot
	_ = readServerFrame(t, conn) // subscribed

	infos := server.Connections()
	require.Len(t, infos, 1)
	require.Equal(t, []string{"s-3"}, infos[0].Subscriptions)
}

func TestServerRejectsCommandFramesAsUnsupported(t *testing.T) {
	_, server := newTestHubAndServer(t)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello

	require.NoError(t, conn.WriteJSON(map[string]any{
		"command": map[string]any{
			"sessionId": "s-command",
			"name":      testCommandName,
			"payload":   map[string]any{"text": "should not enter through websocket"},
		},
	}))

	frame := readServerFrame(t, conn)
	require.NotNil(t, frame.GetError())
	require.Equal(t, "bad_client_frame", frame.GetError().GetCode())
	require.Contains(t, frame.GetError().GetMessage(), "unknown field \"command\"")

	require.Empty(t, server.Connections()[0].Subscriptions)
}

func newTestHubAndServer(t *testing.T) (*sessionstream.Hub, *Server) {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterCommand(testCommandName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterEvent(testEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterUIEvent(testUIEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity(testEntityKind, &structpb.Struct{}))

	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
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

type snapshotAdapter struct{ store *storesqlite.Store }

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

func writeClientFrame(t *testing.T, conn *websocket.Conn, frame *sessionstreamv1.ClientFrame) {
	t.Helper()
	body, err := protojson.Marshal(frame)
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, body))
}

func readServerFrame(t *testing.T, conn *websocket.Conn) *sessionstreamv1.ServerFrame {
	t.Helper()
	_, body, err := conn.ReadMessage()
	require.NoError(t, err)
	frame := &sessionstreamv1.ServerFrame{}
	require.NoError(t, protojson.Unmarshal(body, frame))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	return frame
}
