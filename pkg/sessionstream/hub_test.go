package sessionstream

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	testCommandName = "LabStart"
	testEventName   = "LabStarted"
	testEntityKind  = "LabMessage"
)

func TestHubSubmitRunsHandlerProjectionAndStore(t *testing.T) {
	hub := newTestHub(t)
	registerTestHandler(t, hub)

	uiEvents := make([]UIEvent, 0)
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]UIEvent, error) {
		uiEvents = append(uiEvents, UIEvent{Name: "LabMessageStarted", Payload: ev.Payload})
		return []UIEvent{{Name: "LabMessageStarted", Payload: ev.Payload}}, nil
	})))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		payload := ev.Payload.(*structpb.Struct).AsMap()
		entityPayload, err := structpb.NewStruct(map[string]any{"prompt": payload["prompt"]})
		require.NoError(t, err)
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: entityPayload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))

	session, err := hub.Session(context.Background(), "s-1")
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, map[string]any{"sessionId": "s-1"}, session.Metadata)

	cursor, err := hub.Cursor(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), cursor)
	require.Len(t, uiEvents, 1)

	snap, err := hub.Snapshot(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), snap.SnapshotOrdinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "hello", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["prompt"])
}

func TestHubSubmitUnknownCommand(t *testing.T) {
	hub := newTestHub(t)
	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)

	err = hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload)
	require.Error(t, err)
	require.ErrorContains(t, err, `unknown command "LabStart"`)
}

func TestHubProjectionErrorPolicyFailReturnsProjectionError(t *testing.T) {
	hub := newTestHub(t)
	registerTestHandler(t, hub)
	boom := errors.New("projection exploded")
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]TimelineEntity, error) {
		return nil, boom
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	err = hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload)
	require.ErrorIs(t, err, boom)

	cursor, err := hub.Cursor(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
}

func TestHubProjectionErrorPolicyAdvanceStillAdvancesCursor(t *testing.T) {
	hub := newTestHub(t, WithTimelineProjectionErrorPolicy(ProjectionErrorPolicyAdvance))
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]TimelineEntity, error) {
		return nil, errors.New("projection exploded")
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))

	cursor, err := hub.Cursor(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), cursor)
}

func TestHubProjectionPoliciesSplitUIAndTimeline(t *testing.T) {
	observed := make([]ErrorRecord, 0)
	hub := newTestHub(t,
		WithUIProjectionErrorPolicy(ProjectionErrorPolicyAdvance),
		WithTimelineProjectionErrorPolicy(ProjectionErrorPolicyFail),
		WithErrorObserver(ErrorObserverFunc(func(_ context.Context, rec ErrorRecord) {
			observed = append(observed, rec)
		})),
	)
	registerTestHandler(t, hub)
	uiBoom := errors.New("ui projection exploded")
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]UIEvent, error) {
		return nil, uiBoom
	})))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))

	cursor, err := hub.Cursor(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), cursor)
	require.Len(t, observed, 1)
	require.Equal(t, ErrorKindUIProjection, observed[0].Kind)
	require.ErrorIs(t, observed[0].Err, uiBoom)
}

func TestHubErrorObserverPanicRecovered(t *testing.T) {
	hub := newTestHub(t, WithErrorObserver(ErrorObserverFunc(func(context.Context, ErrorRecord) {
		panic("observer panic should be recovered")
	})))
	registerTestHandler(t, hub)
	boom := errors.New("ui projection exploded")
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]UIEvent, error) {
		return nil, boom
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	err = hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload)
	require.ErrorIs(t, err, boom)
}

func TestHubReportsErrorPersistenceFailureToObserver(t *testing.T) {
	storeErr := errors.New("error store unavailable")
	store := &failingErrorStore{testHydrationStore: newTestHydrationStore().(*testHydrationStore), err: storeErr}
	observed := make([]ErrorRecord, 0)
	hub := newTestHub(t,
		WithHydrationStore(store),
		WithErrorObserver(ErrorObserverFunc(func(_ context.Context, rec ErrorRecord) {
			observed = append(observed, rec)
		})),
	)
	registerTestHandler(t, hub)
	boom := errors.New("ui projection exploded")
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]UIEvent, error) {
		return nil, boom
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	err = hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload)
	require.ErrorIs(t, err, boom)
	require.Len(t, observed, 2)
	require.Equal(t, ErrorKindStore, observed[0].Kind)
	require.ErrorIs(t, observed[0].Err, storeErr)
	require.Equal(t, string(ErrorKindUIProjection), observed[0].Metadata["originalKind"])
	require.Equal(t, ErrorKindUIProjection, observed[1].Kind)
	require.ErrorIs(t, observed[1].Err, boom)
}

func TestHubLocalOrdinalSeedsFromStoreCursor(t *testing.T) {
	store := newTestHydrationStore().(*testHydrationStore)
	store.snapshots["s-1"] = Snapshot{SessionId: "s-1", SnapshotOrdinal: 41}
	hub := newTestHub(t, WithHydrationStore(store))
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))

	cursor, err := hub.Cursor(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(42), cursor)
}

func TestHubLocalOrdinalPrefersEventCursorWhenAvailable(t *testing.T) {
	store := newTestEventStore()
	store.eventCursor = 7
	store.snapshots["s-1"] = Snapshot{SessionId: "s-1", SnapshotOrdinal: 2}
	hub := newTestHub(t, WithHydrationStore(store))
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))
	require.Len(t, store.events, 1)
	require.Equal(t, uint64(8), store.events[0].Ordinal)
}

func TestHubAdvancesTimelineProjectionCursor(t *testing.T) {
	store := newTestEventStore()
	hub := newTestHub(t, WithHydrationStore(store))
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))
	cursor, err := hub.ProjectionCursor(context.Background(), TimelineProjectorName, "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(1), cursor)
}

func TestEventConsumerRecordsDecodeErrors(t *testing.T) {
	store := newTestEventStore()
	hub := newTestHub(t, WithHydrationStore(store))
	consumer := newEventConsumer(hub)

	err := consumer.handleMessage(context.Background(), message.NewMessage("bad", []byte(`{`)))
	require.NoError(t, err)
	require.Len(t, store.errors, 1)
	require.Equal(t, ErrorKindDecode, store.errors[0].Kind)
	require.Equal(t, []byte(`{`), store.errors[0].RawMessage)
}

func TestEventConsumerRecordsOrdinalErrors(t *testing.T) {
	store := newFailingEventCursorStore(errors.New("cursor failed"))
	hub := newTestHub(t, WithHydrationStore(store))
	consumer := newEventConsumer(hub)
	body, err := json.Marshal(eventEnvelope{Name: testEventName, SessionID: "s-1", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	err = consumer.handleMessage(context.Background(), message.NewMessage("event", body))
	require.ErrorContains(t, err, "cursor failed")
	require.Len(t, store.errors, 1)
	require.Equal(t, ErrorKindOrdinal, store.errors[0].Kind)
	require.Equal(t, SessionId("s-1"), store.errors[0].SessionId)
}

func TestHubRebuildTimelineReplaysStoredEventsWithoutFanout(t *testing.T) {
	store := newTestEventStore()
	payload1, err := structpb.NewStruct(map[string]any{"prompt": "one"})
	require.NoError(t, err)
	payload2, err := structpb.NewStruct(map[string]any{"prompt": "two"})
	require.NoError(t, err)
	require.NoError(t, store.AppendEvent(context.Background(), Event{Name: testEventName, SessionId: "s-1", Ordinal: 1, Payload: payload1}))
	require.NoError(t, store.AppendEvent(context.Background(), Event{Name: testEventName, SessionId: "s-1", Ordinal: 2, Payload: payload2}))
	fanoutCalls := 0
	hub := newTestHub(t, WithHydrationStore(store), WithUIFanout(UIFanoutFunc(func(context.Context, SessionId, uint64, []UIEvent) error {
		fanoutCalls++
		return nil
	})))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	require.NoError(t, hub.RebuildTimeline(context.Background(), "s-1", 0))
	cursor, err := hub.ProjectionCursor(context.Background(), TimelineProjectorName, "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(2), cursor)
	require.Equal(t, 0, fanoutCalls)
	snap, err := hub.Snapshot(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, "two", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["prompt"])
}

func TestHubRetryTimelineStartsFromProjectionCursor(t *testing.T) {
	store := newTestEventStore()
	payload1, err := structpb.NewStruct(map[string]any{"prompt": "one"})
	require.NoError(t, err)
	payload2, err := structpb.NewStruct(map[string]any{"prompt": "two"})
	require.NoError(t, err)
	require.NoError(t, store.AppendEvent(context.Background(), Event{Name: testEventName, SessionId: "s-1", Ordinal: 1, Payload: payload1}))
	require.NoError(t, store.AppendEvent(context.Background(), Event{Name: testEventName, SessionId: "s-1", Ordinal: 2, Payload: payload2}))
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), TimelineProjectorName, "s-1", 1))
	hub := newTestHub(t, WithHydrationStore(store))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	require.NoError(t, hub.RetryTimeline(context.Background(), "s-1"))
	cursor, err := hub.ProjectionCursor(context.Background(), TimelineProjectorName, "s-1")
	require.NoError(t, err)
	require.Equal(t, uint64(2), cursor)
	snap, err := hub.Snapshot(context.Background(), "s-1")
	require.NoError(t, err)
	require.Equal(t, "two", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["prompt"])
}

func TestHubRebuildTimelineFromScratchClearsTimeline(t *testing.T) {
	store := newTestEventStore()
	oldPayload, err := structpb.NewStruct(map[string]any{"prompt": "old"})
	require.NoError(t, err)
	newPayload, err := structpb.NewStruct(map[string]any{"prompt": "new"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), "s-1", 99, []TimelineEntity{{Kind: testEntityKind, Id: "stale", Payload: oldPayload}}))
	require.NoError(t, store.AppendEvent(context.Background(), Event{Name: testEventName, SessionId: "s-1", Ordinal: 1, Payload: newPayload}))
	hub := newTestHub(t, WithHydrationStore(store))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "fresh", Payload: ev.Payload}}, nil
	})))

	require.NoError(t, hub.RebuildTimelineFromScratch(context.Background(), "s-1"))
	snap, err := hub.Snapshot(context.Background(), "s-1")
	require.NoError(t, err)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "fresh", snap.Entities[0].Id)
	require.Equal(t, "new", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["prompt"])
}

func newTestHub(t *testing.T, opts ...HubOption) *Hub {
	t.Helper()
	reg := NewSchemaRegistry()
	require.NoError(t, reg.RegisterCommand(testCommandName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterEvent(testEventName, &structpb.Struct{}))
	require.NoError(t, reg.RegisterUIEvent("LabMessageStarted", &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity(testEntityKind, &structpb.Struct{}))

	allOpts := append([]HubOption{
		WithSchemaRegistry(reg),
		WithHydrationStore(newTestHydrationStore()),
		WithSessionMetadataFactory(func(_ context.Context, sid SessionId) (any, error) {
			return map[string]any{"sessionId": string(sid)}, nil
		}),
	}, opts...)
	hub, err := NewHub(allOpts...)
	require.NoError(t, err)
	return hub
}

func registerTestHandler(t *testing.T, hub *Hub) {
	t.Helper()
	require.NoError(t, hub.RegisterCommand(testCommandName, func(ctx context.Context, cmd Command, _ *Session, pub EventPublisher) error {
		payload := cmd.Payload.(*structpb.Struct).AsMap()
		evPayload, err := structpb.NewStruct(map[string]any{"prompt": payload["prompt"]})
		require.NoError(t, err)
		return pub.Publish(ctx, Event{Name: testEventName, SessionId: cmd.SessionId, Payload: evPayload})
	}))
}

type testHydrationStore struct {
	snapshots map[SessionId]Snapshot
}

func newTestHydrationStore() HydrationStore {
	return &testHydrationStore{snapshots: map[SessionId]Snapshot{}}
}

func (s *testHydrationStore) Apply(_ context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error {
	snap := s.snapshots[sid]
	snap.SessionId = sid
	if ord > snap.SnapshotOrdinal {
		snap.SnapshotOrdinal = ord
	}
	entityMap := map[string]TimelineEntity{}
	for _, entity := range snap.Entities {
		entityMap[entity.Kind+"/"+entity.Id] = cloneTestEntity(entity)
	}
	for _, entity := range entities {
		key := entity.Kind + "/" + entity.Id
		if entity.Tombstone {
			delete(entityMap, key)
			continue
		}
		entityMap[key] = cloneTestEntity(entity)
	}
	snap.Entities = snap.Entities[:0]
	for _, entity := range entityMap {
		snap.Entities = append(snap.Entities, cloneTestEntity(entity))
	}
	s.snapshots[sid] = snap
	return nil
}

func (s *testHydrationStore) Snapshot(_ context.Context, sid SessionId, _ uint64) (Snapshot, error) {
	snap, ok := s.snapshots[sid]
	if !ok {
		return Snapshot{SessionId: sid}, nil
	}
	out := Snapshot{SessionId: snap.SessionId, SnapshotOrdinal: snap.SnapshotOrdinal, Entities: make([]TimelineEntity, 0, len(snap.Entities))}
	for _, entity := range snap.Entities {
		out.Entities = append(out.Entities, cloneTestEntity(entity))
	}
	return out, nil
}

func (s *testHydrationStore) View(ctx context.Context, sid SessionId) (TimelineView, error) {
	snap, err := s.Snapshot(ctx, sid, 0)
	if err != nil {
		return nil, err
	}
	return testTimelineView{snapshot: snap}, nil
}

func (s *testHydrationStore) Cursor(_ context.Context, sid SessionId) (uint64, error) {
	return s.snapshots[sid].SnapshotOrdinal, nil
}

type testEventStore struct {
	*testHydrationStore
	eventCursor uint64
	events      []Event
	errors      []ErrorRecord
}

func newTestEventStore() *testEventStore {
	return &testEventStore{testHydrationStore: newTestHydrationStore().(*testHydrationStore)}
}

func (s *testEventStore) AppendEvent(_ context.Context, ev Event) error {
	s.events = append(s.events, Event{Name: ev.Name, SessionId: ev.SessionId, Ordinal: ev.Ordinal, Payload: proto.Clone(ev.Payload)})
	if ev.Ordinal > s.eventCursor {
		s.eventCursor = ev.Ordinal
	}
	return nil
}

func (s *testEventStore) Events(_ context.Context, sid SessionId, after uint64, limit int) ([]Event, error) {
	out := make([]Event, 0)
	for _, ev := range s.events {
		if ev.SessionId != sid || ev.Ordinal <= after {
			continue
		}
		out = append(out, Event{Name: ev.Name, SessionId: ev.SessionId, Ordinal: ev.Ordinal, Payload: proto.Clone(ev.Payload)})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *testEventStore) EventCursor(context.Context, SessionId) (uint64, error) {
	return s.eventCursor, nil
}

func (s *testEventStore) ProjectionCursor(_ context.Context, projector string, sid SessionId) (uint64, error) {
	return s.snapshots[sid].SnapshotOrdinal, nil
}

func (s *testEventStore) AdvanceProjectionCursor(_ context.Context, projector string, sid SessionId, ord uint64) error {
	snap := s.snapshots[sid]
	snap.SessionId = sid
	if ord > snap.SnapshotOrdinal {
		snap.SnapshotOrdinal = ord
	}
	s.snapshots[sid] = snap
	return nil
}

func (s *testEventStore) ClearTimeline(_ context.Context, sid SessionId) error {
	delete(s.snapshots, sid)
	return nil
}

func (s *testEventStore) RecordError(_ context.Context, rec ErrorRecord) error {
	s.errors = append(s.errors, rec)
	return nil
}

type failingErrorStore struct {
	*testHydrationStore
	err error
}

func (s *failingErrorStore) RecordError(context.Context, ErrorRecord) error { return s.err }

type failingEventCursorStore struct {
	*testHydrationStore
	err    error
	errors []ErrorRecord
}

func newFailingEventCursorStore(err error) *failingEventCursorStore {
	return &failingEventCursorStore{testHydrationStore: newTestHydrationStore().(*testHydrationStore), err: err}
}

func (s *failingEventCursorStore) AppendEvent(context.Context, Event) error { return nil }

func (s *failingEventCursorStore) Events(context.Context, SessionId, uint64, int) ([]Event, error) {
	return nil, nil
}

func (s *failingEventCursorStore) EventCursor(context.Context, SessionId) (uint64, error) {
	return 0, s.err
}

func (s *failingEventCursorStore) RecordError(_ context.Context, rec ErrorRecord) error {
	s.errors = append(s.errors, rec)
	return nil
}

type testTimelineView struct {
	snapshot Snapshot
}

func (v testTimelineView) Get(kind, id string) (TimelineEntity, bool) {
	for _, entity := range v.snapshot.Entities {
		if entity.Kind == kind && entity.Id == id {
			return cloneTestEntity(entity), true
		}
	}
	return TimelineEntity{}, false
}

func (v testTimelineView) List(kind string) []TimelineEntity {
	ret := make([]TimelineEntity, 0)
	for _, entity := range v.snapshot.Entities {
		if kind != "" && entity.Kind != kind {
			continue
		}
		ret = append(ret, cloneTestEntity(entity))
	}
	return ret
}

func (v testTimelineView) Ordinal() uint64 { return v.snapshot.SnapshotOrdinal }

func cloneTestEntity(entity TimelineEntity) TimelineEntity {
	out := entity
	if entity.Payload != nil {
		out.Payload = proto.Clone(entity.Payload)
	}
	return out
}

func TestHubPipelineObserverSuccess(t *testing.T) {
	observed := make([]PipelineRecord, 0)
	fanout := UIFanoutFunc(func(context.Context, SessionId, uint64, []UIEvent) error { return nil })
	hub := newTestHub(t,
		WithHydrationStore(newTestEventStore()),
		WithUIFanout(fanout),
		WithPipelineObserver(PipelineObserverFunc(func(_ context.Context, rec PipelineRecord) {
			observed = append(observed, rec)
		})),
	)
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]UIEvent, error) {
		return []UIEvent{{Name: "LabMessageStarted", Payload: ev.Payload}}, nil
	})))
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))

	require.Len(t, observed, 1)
	rec := observed[0]
	require.Equal(t, PipelineModeLive, rec.Mode)
	require.Equal(t, SessionId("s-1"), rec.SessionId)
	require.Equal(t, uint64(1), rec.Ordinal)
	require.Equal(t, testEventName, rec.EventName)
	require.True(t, rec.EventAppended)
	require.NoError(t, rec.AppendErr)
	require.Len(t, rec.UIEvents, 1)
	require.Len(t, rec.TimelineEntities, 1)
	require.Len(t, rec.AppliedEntities, 1)
	require.True(t, rec.TimelineCursorAdvanced)
	require.Len(t, rec.FanoutEvents, 1)
}

func TestHubPipelineObserverProjectionErrorAndPanicRecovery(t *testing.T) {
	boom := errors.New("ui projection exploded")
	observed := make([]PipelineRecord, 0)
	hub := newTestHub(t,
		WithUIProjectionErrorPolicy(ProjectionErrorPolicyFail),
		WithPipelineObserver(PipelineObserverFunc(func(_ context.Context, rec PipelineRecord) {
			observed = append(observed, rec)
			panic("observer panic should be recovered")
		})),
	)
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterUIProjection(UIProjectionFunc(func(context.Context, Event, *Session, TimelineView) ([]UIEvent, error) {
		return nil, boom
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	err = hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload)
	require.ErrorIs(t, err, boom)

	require.Len(t, observed, 1)
	require.ErrorIs(t, observed[0].UIProjectionErr, boom)
}

func TestHubPipelineObserverRebuild(t *testing.T) {
	store := newTestEventStore()
	observed := make([]PipelineRecord, 0)
	hub := newTestHub(t,
		WithHydrationStore(store),
		WithPipelineObserver(PipelineObserverFunc(func(_ context.Context, rec PipelineRecord) {
			observed = append(observed, rec)
		})),
	)
	registerTestHandler(t, hub)
	require.NoError(t, hub.RegisterTimelineProjection(TimelineProjectionFunc(func(_ context.Context, ev Event, _ *Session, _ TimelineView) ([]TimelineEntity, error) {
		return []TimelineEntity{{Kind: testEntityKind, Id: "msg-1", Payload: ev.Payload}}, nil
	})))

	cmdPayload, err := structpb.NewStruct(map[string]any{"prompt": "hello"})
	require.NoError(t, err)
	require.NoError(t, hub.Submit(context.Background(), "s-1", testCommandName, cmdPayload))
	observed = nil

	require.NoError(t, hub.RebuildTimelineFromScratch(context.Background(), "s-1"))
	require.Len(t, observed, 1)
	require.Equal(t, PipelineModeRebuild, observed[0].Mode)
	require.Equal(t, uint64(1), observed[0].Ordinal)
	require.Len(t, observed[0].TimelineEntities, 1)
	require.Len(t, observed[0].AppliedEntities, 1)
	require.True(t, observed[0].TimelineCursorAdvanced)
}
