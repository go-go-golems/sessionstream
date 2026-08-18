package mysql

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// mysqlTestDSN returns a DSN for the local docker-compose MySQL when one is
// configured. Tests skip (not fail) when no DSN is set, so `go test ./...`
// stays green without MySQL. Set SESSIONSTREAM_MYSQL_DSN to run.
func mysqlTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("SESSIONSTREAM_MYSQL_DSN")
	if dsn == "" {
		if os.Getenv("CI") == "true" {
			t.Fatal("SESSIONSTREAM_MYSQL_DSN must be set in CI; refusing to skip MySQL integration tests")
		}
		t.Skipf("SESSIONSTREAM_MYSQL_DSN not set; skipping MySQL hydration store integration test")
	}
	return dsn
}

func newTestRegistry(t *testing.T) *sessionstream.SchemaRegistry {
	t.Helper()
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, reg.RegisterEvent("TestEvent", &structpb.Struct{}))
	require.NoError(t, reg.RegisterTimelineEntity("TestEntity", &structpb.Struct{}))
	return reg
}

// newTestStore opens a Store against the shared coinvault_chat_dev database.
// Each test uses a distinct session id so rows do not collide; the tables are
// shared (created once).
func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), mysqlTestDSN(t), newTestRegistry(t))
	require.NoError(t, err, "Open")
	// Clear all hydration tables at the start so each test (and each re-run
	// against the persistent shared DB) starts clean. Tests in this package run
	// sequentially, and RecordError is a non-idempotent auto-increment INSERT.
	require.NoError(t, store.Reset(context.Background()), "reset")
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func testSession(t *testing.T) string {
	// Include a unique-per-process suffix so re-running against a populated
	// shared database (leftover rows from a prior run) never collides with this
	// test's assertions about initial-empty state.
	return "s-mysql-" + sanitize(t.Name()) + "-" + uniqueSuffix()
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, byte(r))
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

var uniqueSeq atomic.Uint64

func uniqueSuffix() string {
	return strconv.FormatUint(uniqueSeq.Add(1), 36)
}

func TestMySQLStoreInitializesSchemaVersion(t *testing.T) {
	store := newTestStore(t)
	var version int64
	err := store.db.QueryRowContext(context.Background(), `
		SELECT schema_version FROM sessionstream_schema_version WHERE component = ?
	`, schemaComponent).Scan(&version)
	require.NoError(t, err)
	require.Equal(t, schemaVersion, version)
}

func TestMySQLStoreRejectsUnknownSchemaVersion(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	_, err := store.db.ExecContext(ctx, `UPDATE sessionstream_schema_version SET schema_version = 999 WHERE component = ?`, schemaComponent)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(ctx, `UPDATE sessionstream_schema_version SET schema_version = ? WHERE component = ?`, schemaVersion, schemaComponent)
	})

	_, err = Open(ctx, mysqlTestDSN(t), newTestRegistry(t))
	require.ErrorContains(t, err, "unsupported mysql hydration schema version 999")
}

func TestMySQLStoreRejectsUnversionedPrototypeSchema(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	_, err := store.db.ExecContext(ctx, `DROP TABLE sessionstream_schema_version`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS sessionstream_schema_version (
			  component VARBINARY(64) NOT NULL PRIMARY KEY,
			  schema_version BIGINT NOT NULL
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin
		`)
		_, _ = store.db.ExecContext(ctx, `
			INSERT INTO sessionstream_schema_version(component, schema_version) VALUES(?, ?)
		`, schemaComponent, schemaVersion)
	})

	_, err = Open(ctx, mysqlTestDSN(t), newTestRegistry(t))
	require.ErrorContains(t, err, "unversioned prototype table")
}

func TestMySQLStorePreservesOpaqueIdentityBytes(t *testing.T) {
	store := newTestStore(t)
	payload, err := structpb.NewStruct(map[string]any{"text": "identity"})
	require.NoError(t, err)

	for i, entityID := range []string{"café", "cafe\u0301", "key", "key "} {
		err := store.Apply(context.Background(), "identity-session", uint64(i+1), []sessionstream.TimelineEntity{{
			Kind:    "TestEntity",
			Id:      entityID,
			Payload: payload,
		}})
		require.NoError(t, err)
	}
	for i, sid := range []sessionstream.SessionId{"identity-session", "Identity-Session"} {
		err := store.Apply(context.Background(), sid, uint64(10+i), []sessionstream.TimelineEntity{{
			Kind:    "TestEntity",
			Id:      "session-specific",
			Payload: payload,
		}})
		require.NoError(t, err)
	}

	snap, err := store.Snapshot(context.Background(), "identity-session", 0)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 4)
	gotIDs := make(map[string]bool, len(snap.Entities))
	for _, entity := range snap.Entities {
		gotIDs[entity.Id] = true
	}
	for _, want := range []string{"café", "cafe\u0301", "key", "key "} {
		require.True(t, gotIDs[want], "missing exact entity id %q", want)
	}

	other, err := store.Snapshot(context.Background(), "Identity-Session", 0)
	require.NoError(t, err)
	require.Len(t, other.Entities, 1)
	require.Equal(t, "session-specific", other.Entities[0].Id)

	maxID := strings.Repeat("x", 255)
	require.NoError(t, store.Apply(context.Background(), "identity-session", 20, []sessionstream.TimelineEntity{{
		Kind: "TestEntity", Id: maxID, Payload: payload,
	}}))
	err = store.Apply(context.Background(), "identity-session", 21, []sessionstream.TimelineEntity{{
		Kind: "TestEntity", Id: maxID + "x", Payload: payload,
	}})
	require.Error(t, err, "an entity id beyond the VARBINARY byte limit must not be truncated")
}

func TestMySQLStoreApplySnapshotAndCursor(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	payload, err := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NoError(t, store.Apply(context.Background(), sid, 7, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))

	snap, err := store.Snapshot(context.Background(), sid, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(7), snap.SnapshotOrdinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "hello", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])

	cursor, err := store.Cursor(context.Background(), sid)
	require.NoError(t, err)
	require.Equal(t, uint64(7), cursor)
}

func TestMySQLStorePersistsAcrossReopen(t *testing.T) {
	dsn := mysqlTestDSN(t)
	reg := newTestRegistry(t)
	sid := sessionstream.SessionId(testSession(t))

	first, err := Open(context.Background(), dsn, reg)
	require.NoError(t, err)
	require.NoError(t, first.Reset(context.Background()), "reset before reopen test")
	payload, err := structpb.NewStruct(map[string]any{"text": "persisted"})
	require.NoError(t, err)
	require.NoError(t, first.Apply(context.Background(), sid, 9, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, first.Close())

	reopened, err := Open(context.Background(), dsn, reg)
	require.NoError(t, err)
	defer func() { require.NoError(t, reopened.Close()) }()
	snap, err := reopened.Snapshot(context.Background(), sid, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(9), snap.SnapshotOrdinal)
	require.Equal(t, "persisted", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestMySQLStoreAppendsAndReplaysEvents(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	p1, _ := structpb.NewStruct(map[string]any{"text": "one"})
	p2, _ := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p1}))
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 2, Payload: p2}))

	cursor, err := store.EventCursor(context.Background(), sid)
	require.NoError(t, err)
	require.Equal(t, uint64(2), cursor)
	events, err := store.Events(context.Background(), sid, 1, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, uint64(2), events[0].Ordinal)
	require.Equal(t, "two", events[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestMySQLStoreAppendEventConflictCheckWithClientFoundRows(t *testing.T) {
	dsn := mysqlTestDSN(t)
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	store, err := Open(context.Background(), dsn+separator+"clientFoundRows=true", newTestRegistry(t))
	require.NoError(t, err)
	require.NoError(t, store.Reset(context.Background()))
	t.Cleanup(func() { _ = store.Close() })
	sid := sessionstream.SessionId(testSession(t))
	p1, _ := structpb.NewStruct(map[string]any{"text": "one"})
	p2, _ := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p1}))
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p1}))
	require.ErrorContains(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p2}), "event conflict")
}

func TestMySQLStoreAppendEventAllowsOnlyIdenticalDuplicate(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	p1, _ := structpb.NewStruct(map[string]any{"text": "one"})
	p2, _ := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p1}))
	// identical re-append is a no-op
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p1}))
	// differing event at same ordinal is a conflict
	err := store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: p2})
	require.Error(t, err)
	require.ErrorContains(t, err, "event conflict")
	events, err := store.Events(context.Background(), sid, 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "one", events[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestMySQLStoreSnapshotAsOfUsesEntityVersions(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	p1, _ := structpb.NewStruct(map[string]any{"text": "one"})
	p2, _ := structpb.NewStruct(map[string]any{"text": "two"})
	require.NoError(t, store.Apply(context.Background(), sid, 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: p1}}))
	require.NoError(t, store.Apply(context.Background(), sid, 2, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: p2}}))

	snap, err := store.Snapshot(context.Background(), sid, 1)
	require.NoError(t, err)
	require.Equal(t, uint64(1), snap.SnapshotOrdinal)
	require.Len(t, snap.Entities, 1)
	require.Equal(t, "one", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])

	snap, err = store.Snapshot(context.Background(), sid, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2), snap.SnapshotOrdinal)
	require.Equal(t, "two", snap.Entities[0].Payload.(*structpb.Struct).AsMap()["text"])
}

func TestMySQLStoreProjectionCursorAdvanceMonotonic(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	cursor, err := store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid)
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid, 3))
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid, 2)) // not monotonic
	cursor, err = store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid)
	require.NoError(t, err)
	require.Equal(t, uint64(3), cursor)
}

func TestMySQLStoreRecordAndReadError(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	require.NoError(t, store.RecordError(context.Background(), sessionstream.ErrorRecord{
		Kind: sessionstream.ErrorKindDecode, SessionId: sid, Ordinal: 5, EventName: "TestEvent",
		RawMessage: []byte("raw"), Metadata: map[string]string{"k": "v"}, Err: errTestFailure,
	}))
	recs, err := store.ErrorRecords(context.Background(), sid, 10)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	require.Equal(t, sessionstream.ErrorKindDecode, recs[0].Kind)
	require.Equal(t, "test failure", recs[0].Err.Error())
	require.Equal(t, "raw", string(recs[0].RawMessage))
	require.Equal(t, map[string]string{"k": "v"}, recs[0].Metadata)
}

func TestMySQLStoreClearTimelineKeepsEvents(t *testing.T) {
	store := newTestStore(t)
	sid := sessionstream.SessionId(testSession(t))
	payload, _ := structpb.NewStruct(map[string]any{"text": "hello"})
	require.NoError(t, store.AppendEvent(context.Background(), sessionstream.Event{Name: "TestEvent", SessionId: sid, Ordinal: 1, Payload: payload}))
	require.NoError(t, store.Apply(context.Background(), sid, 1, []sessionstream.TimelineEntity{{Kind: "TestEntity", Id: "msg-1", Payload: payload}}))
	require.NoError(t, store.AdvanceProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid, 1))

	require.NoError(t, store.ClearTimeline(context.Background(), sid))
	snap, err := store.Snapshot(context.Background(), sid, 0)
	require.NoError(t, err)
	require.Empty(t, snap.Entities)
	cursor, err := store.ProjectionCursor(context.Background(), sessionstream.TimelineProjectorName, sid)
	require.NoError(t, err)
	require.Equal(t, uint64(0), cursor)
	events, err := store.Events(context.Background(), sid, 0, 10)
	require.NoError(t, err)
	require.Len(t, events, 1) // events survive ClearTimeline
}

var errTestFailure = sentinelError("test failure")

type sentinelError string

func (e sentinelError) Error() string { return string(e) }
