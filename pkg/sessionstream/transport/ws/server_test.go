package ws

import (
	"context"
	"net/http/httptest"
	"sync"
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

func readQueuedServerFrame(t *testing.T, conn *connection) *sessionstreamv1.ServerFrame {
	t.Helper()
	select {
	case queued := <-conn.send:
		frame := &sessionstreamv1.ServerFrame{}
		require.NoError(t, protojson.Unmarshal(queued.body, frame))
		return frame
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued server frame")
		return nil
	}
}

func TestTransportObserverSubscribeAndFanoutSequence(t *testing.T) {
	records := newRecordingTransportObserver()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterCommand(testCommandName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterEvent(testEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterUIEvent(testUIEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity(testEntityKind, &structpb.Struct{}))
	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	server, err := NewServer(snapshotAdapter{store: store}, WithTransportObserver(records))
	require.NoError(t, err)
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithUIFanout(server),
	)
	require.NoError(t, err)
	registerTestFlow(t, hub)

	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-observe"}}})
	_ = readServerFrame(t, conn) // snapshot
	_ = readServerFrame(t, conn) // subscribed

	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-observe", testCommandName, payload))
	_ = readServerFrame(t, conn) // live

	require.Eventually(t, func() bool {
		stages := records.stages()
		return containsStage(stages, TransportStageConnected) &&
			containsStage(stages, TransportStageClientFrameRead) &&
			containsStage(stages, TransportStageClientFrameDecoded) &&
			containsStage(stages, TransportStageSubscribeReceived) &&
			containsStage(stages, TransportStageSnapshotLoadStarted) &&
			containsStage(stages, TransportStageSnapshotLoaded) &&
			containsStage(stages, TransportStageSubscriptionRegistered) &&
			containsStage(stages, TransportStageFanoutStarted) &&
			containsStage(stages, TransportStageFanoutCompleted) &&
			containsStage(stages, TransportStageServerFrameQueued) &&
			containsStage(stages, TransportStageServerFrameWritten)
	}, time.Second, 10*time.Millisecond)

	fanout := records.first(TransportStageFanoutStarted)
	require.Equal(t, sessionstream.SessionId("s-observe"), fanout.SessionId)
	require.Equal(t, uint64(1), fanout.Ordinal)
	require.Len(t, fanout.FanoutTargetIds, 1)
}

func TestTransportObserverBadClientFrameAndPanicRecovery(t *testing.T) {
	records := newRecordingTransportObserver()
	panicObserver := TransportObserverFunc(func(ctx context.Context, rec TransportRecord) {
		records.OnTransport(ctx, rec)
		panic("observer panic should be recovered")
	})
	_, server := newTestHubAndServerWithOptions(t, WithTransportObserver(panicObserver))
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello

	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"command":{"name":"nope"}}`)))
	frame := readServerFrame(t, conn)
	require.NotNil(t, frame.GetError())

	require.Eventually(t, func() bool {
		stages := records.stages()
		return containsStage(stages, TransportStageClientFrameRead) && containsStage(stages, TransportStageClientFrameDecodeError)
	}, time.Second, 10*time.Millisecond)
}

func TestTransportObserverFanoutNoTargets(t *testing.T) {
	records := newRecordingTransportObserver()
	server, err := NewServer(snapshotProviderFunc(func(_ context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		return sessionstream.Snapshot{SessionId: sid}, nil
	}), WithTransportObserver(records))
	require.NoError(t, err)
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "missing", 7, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))

	rec := records.first(TransportStageFanoutNoTargets)
	require.Equal(t, sessionstream.SessionId("missing"), rec.SessionId)
	require.Equal(t, uint64(7), rec.Ordinal)
}

func newTestHubAndServerWithOptions(t *testing.T, opts ...Option) (*sessionstream.Hub, *Server) {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterCommand(testCommandName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterEvent(testEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterUIEvent(testUIEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity(testEntityKind, &structpb.Struct{}))
	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	server, err := NewServer(snapshotAdapter{store: store}, opts...)
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

type snapshotProviderFunc func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error)

func (f snapshotProviderFunc) Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
	return f(ctx, sid)
}

type recordingTransportObserver struct {
	mu      sync.Mutex
	records []TransportRecord
}

func newRecordingTransportObserver() *recordingTransportObserver {
	return &recordingTransportObserver{}
}

func (r *recordingTransportObserver) OnTransport(_ context.Context, rec TransportRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
}

func (r *recordingTransportObserver) stages() []TransportStage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TransportStage, 0, len(r.records))
	for _, rec := range r.records {
		out = append(out, rec.Stage)
	}
	return out
}

func (r *recordingTransportObserver) first(stage TransportStage) TransportRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.records {
		if rec.Stage == stage {
			return rec
		}
	}
	return TransportRecord{}
}

func containsStage(stages []TransportStage, stage TransportStage) bool {
	for _, got := range stages {
		if got == stage {
			return true
		}
	}
	return false
}

func TestSubscribeBuffersFanoutDuringSnapshotLoad(t *testing.T) {
	snapshotStarted := make(chan struct{})
	releaseSnapshot := make(chan struct{})
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		closeOnce(snapshotStarted)
		select {
		case <-releaseSnapshot:
		case <-ctx.Done():
			return sessionstream.Snapshot{}, ctx.Err()
		}
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 100}, nil
	})
	records := newRecordingTransportObserver()
	server, err := NewServer(provider, WithTransportObserver(records))
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-race"}}})
	<-snapshotStarted

	payload, err := structpb.NewStruct(map[string]any{"text": "during snapshot"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-race", 101, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))

	close(releaseSnapshot)
	snapshot := readServerFrame(t, conn)
	require.Equal(t, uint64(100), snapshot.GetSnapshot().GetSnapshotOrdinal())
	live := readServerFrame(t, conn)
	require.NotNil(t, live.GetUiEvent())
	require.Equal(t, uint64(101), live.GetUiEvent().GetEventOrdinal())
	subscribed := readServerFrame(t, conn)
	require.NotNil(t, subscribed.GetSubscribed())

	require.Eventually(t, func() bool {
		stages := records.stages()
		return containsStage(stages, TransportStageUIEventBuffered) &&
			containsStage(stages, TransportStageHydrationBufferFlushed) &&
			containsStage(stages, TransportStageSubscriptionLive)
	}, time.Second, 10*time.Millisecond)
}

func TestSubscribeDoesNotFlushBufferedEventsAtOrBeforeSnapshotOrdinal(t *testing.T) {
	snapshotStarted := make(chan struct{})
	releaseSnapshot := make(chan struct{})
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		closeOnce(snapshotStarted)
		select {
		case <-releaseSnapshot:
		case <-ctx.Done():
			return sessionstream.Snapshot{}, ctx.Err()
		}
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 100}, nil
	})
	server, err := NewServer(provider)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn)
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-dup"}}})
	<-snapshotStarted
	payload, err := structpb.NewStruct(map[string]any{"text": "already in snapshot"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-dup", 100, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))
	close(releaseSnapshot)

	snapshot := readServerFrame(t, conn)
	require.Equal(t, uint64(100), snapshot.GetSnapshot().GetSnapshotOrdinal())
	subscribed := readServerFrame(t, conn)
	require.NotNil(t, subscribed.GetSubscribed())
}

func TestHydratingConnectionDoesNotBlockLiveConnection(t *testing.T) {
	blockSnapshot := make(chan struct{})
	releaseSnapshot := make(chan struct{})
	var snapshotCalls int
	var mu sync.Mutex
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		mu.Lock()
		snapshotCalls++
		call := snapshotCalls
		mu.Unlock()
		if call == 2 {
			closeOnce(blockSnapshot)
			select {
			case <-releaseSnapshot:
			case <-ctx.Done():
				return sessionstream.Snapshot{}, ctx.Err()
			}
		}
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 100}, nil
	})
	server, err := NewServer(provider)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	liveConn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, liveConn.Close()) }()
	_ = readServerFrame(t, liveConn)
	writeClientFrame(t, liveConn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-multi"}}})
	_ = readServerFrame(t, liveConn)
	_ = readServerFrame(t, liveConn)

	hydratingConn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, hydratingConn.Close()) }()
	_ = readServerFrame(t, hydratingConn)
	writeClientFrame(t, hydratingConn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-multi"}}})
	<-blockSnapshot

	payload, err := structpb.NewStruct(map[string]any{"text": "both tabs"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-multi", 101, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))

	liveFrame := readServerFrame(t, liveConn)
	require.Equal(t, uint64(101), liveFrame.GetUiEvent().GetEventOrdinal())

	close(releaseSnapshot)
	_ = readServerFrame(t, hydratingConn) // snapshot
	hydratedLive := readServerFrame(t, hydratingConn)
	require.Equal(t, uint64(101), hydratedLive.GetUiEvent().GetEventOrdinal())
}

func TestHydrationBufferOverflowClosesConnection(t *testing.T) {
	snapshotStarted := make(chan struct{})
	releaseSnapshot := make(chan struct{})
	records := newRecordingTransportObserver()
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		closeOnce(snapshotStarted)
		select {
		case <-releaseSnapshot:
		case <-ctx.Done():
			return sessionstream.Snapshot{}, ctx.Err()
		}
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 0}, nil
	})
	server, err := NewServer(provider, WithHydrationBufferLimit(1), WithTransportObserver(records))
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { _ = conn.Close() }()
	_ = readServerFrame(t, conn)
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-overflow"}}})
	<-snapshotStarted
	payload, err := structpb.NewStruct(map[string]any{"text": "overflow"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-overflow", 1, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))
	err = server.PublishUI(context.Background(), "s-overflow", 2, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}})
	require.Error(t, err)
	require.ErrorContains(t, err, "hydration buffer full")

	require.Eventually(t, func() bool {
		return containsStage(records.stages(), TransportStageHydrationBufferOverflow)
	}, time.Second, 10*time.Millisecond)
	close(releaseSnapshot)
}

func closeOnce(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}

func TestFlushLateHydrationBufferAndMarkLiveFiltersBatchesAlreadyCoveredBySnapshot(t *testing.T) {
	server, err := NewServer(snapshotProviderFunc(func(context.Context, sessionstream.SessionId) (sessionstream.Snapshot, error) {
		return sessionstream.Snapshot{}, nil
	}))
	require.NoError(t, err)
	oldPayload, err := structpb.NewStruct(map[string]any{"text": "already in snapshot"})
	require.NoError(t, err)
	newPayload, err := structpb.NewStruct(map[string]any{"text": "after snapshot"})
	require.NoError(t, err)
	conn := &connection{
		id:   "conn-test",
		send: make(chan outboundFrame, 8),
		subs: map[sessionstream.SessionId]subscription{},
	}
	conn.subs["s-filter"] = subscription{
		state: subscriptionStateHydrating,
		buffer: []bufferedUIBatch{
			{ordinal: 9, events: []sessionstream.UIEvent{{Name: testUIEventName, Payload: oldPayload}}},
			{ordinal: 10, events: []sessionstream.UIEvent{{Name: testUIEventName, Payload: oldPayload}}},
			{ordinal: 11, events: []sessionstream.UIEvent{{Name: testUIEventName, Payload: newPayload}}},
		},
	}

	lateEventCount, err := server.flushLateHydrationBufferAndMarkLive(context.Background(), conn, "s-filter", 10)
	require.NoError(t, err)
	require.Equal(t, 1, lateEventCount)
	require.Equal(t, subscriptionStateLive, conn.subs["s-filter"].state)

	queued := <-conn.send
	frame := &sessionstreamv1.ServerFrame{}
	require.NoError(t, unmarshalOpts.Unmarshal(queued.body, frame))
	require.NotNil(t, frame.GetUiEvent())
	require.Equal(t, uint64(11), frame.GetUiEvent().GetEventOrdinal())
	require.Empty(t, conn.send)
}

func TestFlushLateHydrationBufferAndMarkLiveQueuesLateBeforeConcurrentLiveFanout(t *testing.T) {
	latePayload, err := structpb.NewStruct(map[string]any{"text": "late"})
	require.NoError(t, err)
	livePayload, err := structpb.NewStruct(map[string]any{"text": "live"})
	require.NoError(t, err)

	conn := &connection{
		id:   "conn-test",
		send: make(chan outboundFrame, 8),
		subs: map[sessionstream.SessionId]subscription{},
	}
	conn.subs["s-order"] = subscription{
		state: subscriptionStateHydrating,
		buffer: []bufferedUIBatch{
			{ordinal: 101, events: []sessionstream.UIEvent{{Name: testUIEventName, Payload: latePayload}}},
		},
	}

	publishDone := make(chan error, 1)
	publishStarted := make(chan struct{})
	var publishOnce sync.Once
	var server *Server
	server, err = NewServer(
		snapshotProviderFunc(func(context.Context, sessionstream.SessionId) (sessionstream.Snapshot, error) {
			return sessionstream.Snapshot{}, nil
		}),
		WithTransportObserver(TransportObserverFunc(func(ctx context.Context, rec TransportRecord) {
			if rec.Stage == TransportStageUIEventSent && rec.Ordinal == 101 {
				publishOnce.Do(func() {
					close(publishStarted)
					go func() {
						publishDone <- server.PublishUI(context.Background(), "s-order", 102, []sessionstream.UIEvent{{Name: testUIEventName, Payload: livePayload}})
					}()
				})
			}
		})),
	)
	require.NoError(t, err)
	server.mu.Lock()
	server.conns[conn.id] = conn
	server.bySession["s-order"] = map[sessionstream.ConnectionId]struct{}{conn.id: {}}
	server.mu.Unlock()

	lateEventCount, err := server.flushLateHydrationBufferAndMarkLive(context.Background(), conn, "s-order", 100)
	require.NoError(t, err)
	require.Equal(t, 1, lateEventCount)
	<-publishStarted
	require.NoError(t, <-publishDone)

	first := readQueuedServerFrame(t, conn)
	require.NotNil(t, first.GetUiEvent())
	require.Equal(t, uint64(101), first.GetUiEvent().GetEventOrdinal())
	second := readQueuedServerFrame(t, conn)
	require.NotNil(t, second.GetUiEvent())
	require.Equal(t, uint64(102), second.GetUiEvent().GetEventOrdinal())
}

// TestSubscribeLateBufferNotDroppedByLiveTransition is a regression test for
// the race where PublishUI buffers events after drainHydrationBuffer clears the
// buffer but before the subscription transitions to live. The live transition
// must queue those late-buffered batches rather than silently discarding them.
func TestSubscribeLateBufferNotDroppedByLiveTransition(t *testing.T) {
	snapshotReleased := make(chan struct{})
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		<-snapshotReleased
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 50}, nil
	})

	records := newRecordingTransportObserver()
	server, err := NewServer(provider, WithTransportObserver(records))
	require.NoError(t, err)

	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello

	// Subscribe — snapshot blocks until we release it.
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-late"}}})

	// Give the subscribe handler time to enter Snapshot().
	require.Eventually(t, func() bool {
		return containsStage(records.stages(), TransportStageSnapshotLoadStarted)
	}, time.Second, 10*time.Millisecond)

	// Publish an event while the snapshot is loading → buffered during hydration.
	payload1, err := structpb.NewStruct(map[string]any{"text": "during-snapshot"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-late", 51, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload1}}))
	require.Eventually(t, func() bool {
		return containsStage(records.stages(), TransportStageUIEventBuffered)
	}, time.Second, 10*time.Millisecond)

	// Release the snapshot. The subscribe handler will drain the buffer, send the
	// buffered event, then flush any late-buffered events and go live.
	close(snapshotReleased)

	// Wait for the subscribe handler to complete the live transition.
	//
	// We can't deterministically interleave, but with -race and -count=100 this
	// would surface the bug. Instead, we verify the stronger invariant: both
	// events arrive at the client.
	require.Eventually(t, func() bool {
		return containsStage(records.stages(), TransportStageSubscriptionLive)
	}, time.Second, 10*time.Millisecond)

	// Now publish a post-live event to verify live fanout works.
	payload2, err := structpb.NewStruct(map[string]any{"text": "post-live"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-late", 52, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload2}}))

	// The client should receive: snapshot(50), uiEvent(51), subscribed, uiEvent(52).
	snap := readServerFrame(t, conn)
	require.Equal(t, uint64(50), snap.GetSnapshot().GetSnapshotOrdinal())

	event1 := readServerFrame(t, conn)
	require.NotNil(t, event1.GetUiEvent())
	require.Equal(t, uint64(51), event1.GetUiEvent().GetEventOrdinal())

	sub := readServerFrame(t, conn)
	require.NotNil(t, sub.GetSubscribed())

	event2 := readServerFrame(t, conn)
	require.NotNil(t, event2.GetUiEvent())
	require.Equal(t, uint64(52), event2.GetUiEvent().GetEventOrdinal())
}

// TestDeliverUIEventsSendsDirectlyWhenStateChangedToLive verifies that
// deliverUIEvents (the unified state check + buffer/send) routes events
// directly when the subscription has transitioned to live, rather than
// silently dropping them. This is the TOCTOU companion to the live-transition fix.
func TestDeliverUIEventsSendsDirectlyWhenStateChangedToLive(t *testing.T) {
	provider := snapshotProviderFunc(func(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
		return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: 10}, nil
	})
	server, err := NewServer(provider)
	require.NoError(t, err)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	conn := dialWS(t, httpServer.URL)
	defer func() { require.NoError(t, conn.Close()) }()
	_ = readServerFrame(t, conn) // hello

	// Fully subscribe and go live.
	writeClientFrame(t, conn, &sessionstreamv1.ClientFrame{Frame: &sessionstreamv1.ClientFrame_Subscribe{Subscribe: &sessionstreamv1.SubscribeRequest{SessionId: "s-direct"}}})
	_ = readServerFrame(t, conn) // snapshot
	_ = readServerFrame(t, conn) // subscribed

	// Now the subscription is live. PublishUI should send directly.
	payload, err := structpb.NewStruct(map[string]any{"text": "live-direct"})
	require.NoError(t, err)
	require.NoError(t, server.PublishUI(context.Background(), "s-direct", 11, []sessionstream.UIEvent{{Name: testUIEventName, Payload: payload}}))

	live := readServerFrame(t, conn)
	require.NotNil(t, live.GetUiEvent())
	require.Equal(t, uint64(11), live.GetUiEvent().GetEventOrdinal())
}
