package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

var errTestFailure = errors.New("test failure")

func TestStoreApplySnapshotAndCursor(t *testing.T) {
	store := newTestStore(t)
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), sessionstream.SessionId("s-1"), 7, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))

	snap, err := store.Snapshot(context.Background(), sessionstream.SessionId("s-1"), 0)
	require.NoError(t, err)
	require.Equal(t, uint64(7), snap.Ordinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "hello", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestStorePersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessionstream.sqlite")
	dsn, err := FileDSN(path)
	require.NoError(t, err)
	reg := newTestRegistry(t)
	store, err := New(dsn, reg)
	require.NoError(t, err)
	payload, err := structpb.NewStruct(map[string]any{"text": "persisted"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), sessionstream.SessionId("s-2"), 9, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, store.Close())

	reopened, err := New(dsn, reg)
	require.NoError(t, err)
	defer reopened.Close()
	snap, err := reopened.Snapshot(context.Background(), sessionstream.SessionId("s-2"), 0)
	require.NoError(t, err)
	require.Equal(t, uint64(9), snap.Ordinal)
	require.Equal(t, "persisted", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestStoreReset(t *testing.T) {
	store := newTestStore(t)
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), sessionstream.SessionId("s-3"), 3, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-3", Ordinal: 3, Payload: payload}))
	require.NoError(t, store.Reset(context.Background()))
	cursor, err := store.Cursor(context.Background(), sessionstream.SessionId("s-3"))
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
	eventCursor, err := store.EventCursor(context.Background(), sessionstream.SessionId("s-3"))
	require.NoError(t, err)
	require.Equal(t, uint64(0), eventCursor)
}

func TestStoreAppendsAndReplaysEvents(t *testing.T) {
	store := newTestStore(t)
	payload1, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	payload2, err := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, err)
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-4", Ordinal: 1, Payload: payload1}))
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-4", Ordinal: 2, Payload: payload2}))

	cursor, err := store.EventCursor(context.Background(), "s-4")
	require.NoError(t, err)
	require.Equal(t, uint64(2), cursor)
	events, err := store.Events(context.Background(), "s-4", 1, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(2), events[0].Ordinal)
	require.Equal(t, "two", events[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestStoreSnapshotAsOfUsesEntityVersions(t *testing.T) {
	store := newTestStore(t)
	payload1, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	payload2, err := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), "s-5", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload1}}))
	require.NoError(t, store.Apply(context.Background(), "s-5", 2, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload2}}))

	snap, err := store.Snapshot(context.Background(), "s-5", 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), snap.Ordinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "one", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])

	snap, err = store.Snapshot(context.Background(), "s-5", 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2), snap.Ordinal)
	require.Equal(t, "two", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestStoreSnapshotAsOfRespectsTombstones(t *testing.T) {
	store := newTestStore(t)
	payload, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), "s-6", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, store.Apply(context.Background(), "s-6", 2, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Tombstone: true}}))

	snap, err := store.Snapshot(context.Background(), "s-6", 1)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 1)
	snap, err = store.Snapshot(context.Background(), "s-6", 2)
	require.NoError(t, err)
	require.Empty(t, snap.Entities)
}

func TestNewInMemoryStore(t *testing.T) {
	store, err := NewInMemory(newTestRegistry(t))
	require.NoError(t, err)
	defer store.Close()
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), "s-7", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	snap, err := store.Snapshot(context.Background(), "s-7", 0)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 1)
}

func TestProjectionCursorAdvancesMonotonically(t *testing.T) {
	store := newTestStore(t)
	cursor, err := store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-8")
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-8", 3))
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-8", 2))
	cursor, err = store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-8")
	require.NoError(t, err)
	require.Equal(t, uint64(3), cursor)
}

func TestStoreRecordsAndReadsErrors(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.RecordError(context.Background(), sessionstream.ErrorRecord{
		Kind:       sessionstream.ErrorKindDecode,
		SessionId:  "s-err",
		Ordinal:    4,
		EventName:  "TestEvent",
		Err:        errTestFailure,
		RawMessage: []byte(`{"broken":true}`),
		Metadata:   map[string]string{"messageId": "m-1"},
	}))

	records, err := store.ErrorRecords(context.Background(), "s-err", 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, sessionstream.ErrorKindDecode, records[0].Kind)
	require.Equal(t, sessionstream.SessionId("s-err"), records[0].SessionId)
	require.Equal(t, uint64(4), records[0].Ordinal)
	require.Equal(t, "TestEvent", records[0].EventName)
	require.Equal(t, []byte(`{"broken":true}`), records[0].RawMessage)
	require.Equal(t, "m-1", records[0].Metadata["messageId"])
	require.ErrorContains(t, records[0].Err, "test failure")
}

func TestClearTimelineClearsMaterializedStateButKeepsEvents(t *testing.T) {
	store := newTestStore(t)
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-9", Ordinal: 1, Payload: payload}))
	require.NoError(t, store.Apply(context.Background(), "s-9", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-9", 1))

	require.NoError(t, store.ClearTimeline(context.Background(), "s-9"))
	snap, err := store.Snapshot(context.Background(), "s-9", 0)
	require.NoError(t, err)
	require.Empty(t, snap.Entities)
	cursor, err := store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, "s-9")
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
	events, err := store.Events(context.Background(), "s-9", 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn, err := FileDSN(filepath.Join(t.TempDir(), "sessionstream.sqlite"))
	require.NoError(t, err)
	store, err := New(dsn, newTestRegistry(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func newTestRegistry(t *testing.T) *sessionstream.SchemaRegistry {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterEvent("TestEvent", &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity("TestEntity", &structpb.Struct{}))
	return reg
}
