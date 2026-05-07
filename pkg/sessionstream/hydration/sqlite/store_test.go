package sqlite

import (
	"context"
	"database/sql"
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
	require.Equal(t, uint64(7), snap.SnapshotOrdinal)
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
	defer func() { require.NoError(t, reopened.Close()) }()
	snap, err := reopened.Snapshot(context.Background(), sessionstream.SessionId("s-2"), 0)
	require.NoError(t, err)
	require.Equal(t, uint64(9), snap.SnapshotOrdinal)
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

func TestStoreAppendEventAllowsOnlyIdenticalDuplicate(t *testing.T) {
	store := newTestStore(t)
	payload1, err := structpb.NewStruct(map[string]any{"text": "one"})
	require.NoError(t, err)
	payload2, err := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, err)
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-conflict", Ordinal: 1, Payload: payload1}))
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-conflict", Ordinal: 1, Payload: payload1}))

	err = store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: "s-conflict", Ordinal: 1, Payload: payload2})
	require.Error(t, err)
	require.ErrorContains(t, err, "event conflict")

	events, err := store.Events(context.Background(), "s-conflict", 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "one", events[0].Payload.(*structpb.Struct).AsMap()["text"])
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
	require.Equal(t, uint64(1), snap.SnapshotOrdinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "one", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])

	snap, err = store.Snapshot(context.Background(), "s-5", 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2), snap.SnapshotOrdinal)
	require.Equal(t, "two", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestStoreSnapshotCurrentOrdersByCreatedThenLastEventOrdinal(t *testing.T) {
	store := newTestStore(t)
	first, err := structpb.NewStruct(map[string]any{"text": "first"})
	require.NoError(t, err)
	second, err := structpb.NewStruct(map[string]any{"text": "second"})
	require.NoError(t, err)
	updatedFirst, err := structpb.NewStruct(map[string]any{"text": "first updated"})
	require.NoError(t, err)

	require.NoError(t, store.Apply(context.Background(), "s-order", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "first", Payload: first}}))
	require.NoError(t, store.Apply(context.Background(), "s-order", 2, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "second", Payload: second}}))
	require.NoError(t, store.Apply(context.Background(), "s-order", 3, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "first", Payload: updatedFirst}}))

	snap, err := store.Snapshot(context.Background(), "s-order", 0)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 2)
	require.Equal(t, "first", snap.Entities[0].Id)
	require.Equal(t, uint64(1), snap.Entities[0].CreatedOrdinal)
	require.Equal(t, uint64(3), snap.Entities[0].LastEventOrdinal)
	require.Equal(t, "second", snap.Entities[1].Id)
	require.Equal(t, uint64(2), snap.Entities[1].CreatedOrdinal)
	require.Equal(t, uint64(2), snap.Entities[1].LastEventOrdinal)
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
	defer func() { require.NoError(t, store.Close()) }()
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), "s-7", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	snap, err := store.Snapshot(context.Background(), "s-7", 0)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 1)
}

func TestNewInMemoryStoresAreIsolated(t *testing.T) {
	reg := newTestRegistry(t)
	first, err := NewInMemory(reg)
	require.NoError(t, err)
	defer func() { require.NoError(t, first.Close()) }()
	second, err := NewInMemory(reg)
	require.NoError(t, err)
	defer func() { require.NoError(t, second.Close()) }()
	payload, err := structpb.NewStruct(map[string]any{"text": "isolated"})
	require.NoError(t, err)
	require.NoError(t, first.Apply(context.Background(), "s-isolated", 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))

	snap, err := second.Snapshot(context.Background(), "s-isolated", 0)
	require.NoError(t, err)
	require.Empty(t, snap.Entities)
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

func TestMigratePreservesExistingRowsAndAddsErrorColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessionstream.sqlite")
	dsn, err := FileDSN(path)
	require.NoError(t, err)
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE sessionstream_sessions (
		  session_id TEXT PRIMARY KEY,
		  snapshot_ordinal INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE sessionstream_errors (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  kind TEXT NOT NULL,
		  session_id TEXT,
		  ordinal INTEGER,
		  event_name TEXT,
		  error TEXT NOT NULL
		);
		INSERT INTO sessionstream_sessions(session_id, snapshot_ordinal) VALUES('s-old', 12);
		PRAGMA user_version = 1;
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	store, err := New(dsn, newTestRegistry(t))
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	cursor, err := store.Cursor(context.Background(), "s-old")
	require.NoError(t, err)
	require.Equal(t, uint64(12), cursor)
	require.NoError(t, store.RecordError(context.Background(), sessionstream.ErrorRecord{Kind: sessionstream.ErrorKindDecode, RawMessage: []byte("raw"), Metadata: map[string]string{"k": "v"}, Err: errTestFailure}))
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
