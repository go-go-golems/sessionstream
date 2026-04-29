package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Store struct {
	db  *sql.DB
	reg *sessionstream.SchemaRegistry
}

var _ sessionstream.HydrationStore = (*Store)(nil)
var _ sessionstream.EventStore = (*Store)(nil)
var _ sessionstream.ProjectionCursorStore = (*Store)(nil)
var _ sessionstream.TimelineResetStore = (*Store)(nil)
var _ sessionstream.ErrorStore = (*Store)(nil)
var _ sessionstream.ErrorRecordStore = (*Store)(nil)

func New(dsn string, reg *sessionstream.SchemaRegistry) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("sqlite hydration store dsn is empty")
	}
	if reg == nil {
		return nil, fmt.Errorf("sqlite hydration store schema registry is nil")
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, reg: reg}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func FileDSN(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("sqlite hydration store path is empty")
	}
	return fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", path), nil
}

func MemoryDSN(name string) string {
	if name == "" {
		name = "sessionstream-memory"
	}
	return fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000&_foreign_keys=on", name)
}

func NewInMemory(reg *sessionstream.SchemaRegistry) (*Store, error) {
	return New(MemoryDSN(""), reg)
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Reset(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, stmt := range []string{
		`DELETE FROM sessionstream_errors`,
		`DELETE FROM sessionstream_projection_cursors`,
		`DELETE FROM sessionstream_entity_versions`,
		`DELETE FROM sessionstream_entities`,
		`DELETE FROM sessionstream_events`,
		`DELETE FROM sessionstream_sessions`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Apply(ctx context.Context, sid sessionstream.SessionId, ord uint64, entities []sessionstream.TimelineEntity) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cursor, err := uint64ToInt64(ord)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, entity := range entities {
		payload, err := s.reg.MarshalProtoJSON(entity.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessionstream_entity_versions(session_id, kind, entity_id, ordinal, tombstone, payload_json)
			VALUES(?, ?, ?, ?, ?, ?)
			ON CONFLICT(session_id, kind, entity_id, ordinal) DO UPDATE SET
				tombstone = excluded.tombstone,
				payload_json = excluded.payload_json
		`, string(sid), entity.Kind, entity.Id, cursor, boolToInt(entity.Tombstone), string(payload)); err != nil {
			return err
		}
		if entity.Tombstone {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessionstream_entities WHERE session_id = ? AND kind = ? AND entity_id = ?`, string(sid), entity.Kind, entity.Id); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessionstream_entities(session_id, kind, entity_id, payload_json)
			VALUES(?, ?, ?, ?)
			ON CONFLICT(session_id, kind, entity_id) DO UPDATE SET payload_json = excluded.payload_json
		`, string(sid), entity.Kind, entity.Id, string(payload)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessionstream_sessions(session_id, cursor)
		VALUES(?, ?)
		ON CONFLICT(session_id) DO UPDATE SET cursor = CASE
			WHEN excluded.cursor > sessionstream_sessions.cursor THEN excluded.cursor
			ELSE sessionstream_sessions.cursor
		END
	`, string(sid), cursor); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context, sid sessionstream.SessionId, asOf uint64) (sessionstream.Snapshot, error) {
	if s == nil || s.db == nil {
		return sessionstream.Snapshot{}, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cursor, err := s.Cursor(ctx, sid)
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	if asOf > 0 && asOf < cursor {
		cursor = asOf
	}

	var rows *sql.Rows
	if asOf == 0 {
		rows, err = s.db.QueryContext(ctx, `SELECT kind, entity_id, 0 AS tombstone, payload_json FROM sessionstream_entities WHERE session_id = ? ORDER BY kind ASC, entity_id ASC`, string(sid))
	} else {
		versionCursor, err := uint64ToInt64(asOf)
		if err != nil {
			return sessionstream.Snapshot{}, err
		}
		rows, err = s.db.QueryContext(ctx, `
			SELECT v.kind, v.entity_id, v.tombstone, v.payload_json
			FROM sessionstream_entity_versions v
			JOIN (
				SELECT kind, entity_id, MAX(ordinal) AS ordinal
				FROM sessionstream_entity_versions
				WHERE session_id = ? AND ordinal <= ?
				GROUP BY kind, entity_id
			) latest
			ON latest.kind = v.kind AND latest.entity_id = v.entity_id AND latest.ordinal = v.ordinal
			WHERE v.session_id = ?
			ORDER BY v.kind ASC, v.entity_id ASC
		`, string(sid), versionCursor, string(sid))
	}
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	defer func() { _ = rows.Close() }()
	entities := make([]sessionstream.TimelineEntity, 0)
	for rows.Next() {
		var (
			kind      string
			id        string
			tombstone int
			rawJSON   string
		)
		if err := rows.Scan(&kind, &id, &tombstone, &rawJSON); err != nil {
			return sessionstream.Snapshot{}, err
		}
		if tombstone != 0 {
			continue
		}
		prototype, ok := s.reg.TimelineEntitySchema(kind)
		if !ok {
			return sessionstream.Snapshot{}, fmt.Errorf("unknown timeline entity %q", kind)
		}
		msg := prototype.ProtoReflect().New().Interface()
		if err := protojson.Unmarshal([]byte(rawJSON), msg); err != nil {
			return sessionstream.Snapshot{}, err
		}
		entities = append(entities, sessionstream.TimelineEntity{Kind: kind, Id: id, Payload: msg})
	}
	if err := rows.Err(); err != nil {
		return sessionstream.Snapshot{}, err
	}
	sort.Slice(entities, func(i, j int) bool {
		if entities[i].Kind == entities[j].Kind {
			return entities[i].Id < entities[j].Id
		}
		return entities[i].Kind < entities[j].Kind
	})
	return sessionstream.Snapshot{SessionId: sid, Ordinal: cursor, Entities: entities}, nil
}

func (s *Store) View(ctx context.Context, sid sessionstream.SessionId) (sessionstream.TimelineView, error) {
	snap, err := s.Snapshot(ctx, sid, 0)
	if err != nil {
		return nil, err
	}
	return newView(snap), nil
}

func (s *Store) Cursor(ctx context.Context, sid sessionstream.SessionId) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var cursor sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT cursor FROM sessionstream_sessions WHERE session_id = ?`, string(sid)).Scan(&cursor)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !cursor.Valid {
		return 0, nil
	}
	return int64ToUint64(cursor.Int64)
}

func (s *Store) ClearTimeline(ctx context.Context, sid sessionstream.SessionId) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`DELETE FROM sessionstream_entities WHERE session_id = ?`,
		`DELETE FROM sessionstream_entity_versions WHERE session_id = ?`,
		`DELETE FROM sessionstream_projection_cursors WHERE session_id = ? AND projector = ?`,
		`UPDATE sessionstream_sessions SET cursor = 0 WHERE session_id = ?`,
	} {
		if stmt == `DELETE FROM sessionstream_projection_cursors WHERE session_id = ? AND projector = ?` {
			if _, err := tx.ExecContext(ctx, stmt, string(sid), sessionstream.TimelineProjectorName); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt, string(sid)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) EventCursor(ctx context.Context, sid sessionstream.SessionId) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var cursor sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT MAX(ordinal) FROM sessionstream_events WHERE session_id = ?`, string(sid)).Scan(&cursor)
	if err != nil {
		return 0, err
	}
	if !cursor.Valid {
		return 0, nil
	}
	return int64ToUint64(cursor.Int64)
}

func (s *Store) AppendEvent(ctx context.Context, ev sessionstream.Event) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ev.SessionId == "" {
		return fmt.Errorf("event %q missing session id", ev.Name)
	}
	if ev.Name == "" {
		return fmt.Errorf("event name is empty")
	}
	ordinal, err := uint64ToInt64(ev.Ordinal)
	if err != nil {
		return err
	}
	payload, err := s.reg.MarshalProtoJSON(ev.Payload)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessionstream_events(session_id, ordinal, name, payload_json)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(session_id, ordinal) DO UPDATE SET
			name = excluded.name,
			payload_json = excluded.payload_json
	`, string(ev.SessionId), ordinal, ev.Name, string(payload))
	return err
}

func (s *Store) Events(ctx context.Context, sid sessionstream.SessionId, after uint64, limit int) ([]sessionstream.Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	afterCursor, err := uint64ToInt64(after)
	if err != nil {
		return nil, err
	}
	query := `SELECT ordinal, name, payload_json FROM sessionstream_events WHERE session_id = ? AND ordinal > ? ORDER BY ordinal ASC`
	args := []any{string(sid), afterCursor}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]sessionstream.Event, 0)
	for rows.Next() {
		var (
			ordinal int64
			name    string
			rawJSON string
		)
		if err := rows.Scan(&ordinal, &name, &rawJSON); err != nil {
			return nil, err
		}
		prototype, ok := s.reg.EventSchema(name)
		if !ok {
			return nil, fmt.Errorf("unknown event %q", name)
		}
		msg := prototype.ProtoReflect().New().Interface()
		if err := protojson.Unmarshal([]byte(rawJSON), msg); err != nil {
			return nil, err
		}
		ord, err := int64ToUint64(ordinal)
		if err != nil {
			return nil, err
		}
		out = append(out, sessionstream.Event{Name: name, SessionId: sid, Ordinal: ord, Payload: msg})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ProjectionCursor(ctx context.Context, projector string, sid sessionstream.SessionId) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if projector == "" {
		return 0, fmt.Errorf("projector is empty")
	}
	var cursor sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT cursor FROM sessionstream_projection_cursors WHERE projector = ? AND session_id = ?`, projector, string(sid)).Scan(&cursor)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !cursor.Valid {
		return 0, nil
	}
	return int64ToUint64(cursor.Int64)
}

func (s *Store) AdvanceProjectionCursor(ctx context.Context, projector string, sid sessionstream.SessionId, ord uint64) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if projector == "" {
		return fmt.Errorf("projector is empty")
	}
	cursor, err := uint64ToInt64(ord)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessionstream_projection_cursors(projector, session_id, cursor)
		VALUES(?, ?, ?)
		ON CONFLICT(projector, session_id) DO UPDATE SET cursor = CASE
			WHEN excluded.cursor > sessionstream_projection_cursors.cursor THEN excluded.cursor
			ELSE sessionstream_projection_cursors.cursor
		END
	`, projector, string(sid), cursor)
	return err
}

func (s *Store) RecordError(ctx context.Context, rec sessionstream.ErrorRecord) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ordinal, err := uint64ToInt64(rec.Ordinal)
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(rec.Metadata)
	if err != nil {
		return err
	}
	errText := ""
	if rec.Err != nil {
		errText = rec.Err.Error()
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessionstream_errors(kind, session_id, ordinal, event_name, error, raw_message, metadata_json)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, string(rec.Kind), string(rec.SessionId), ordinal, rec.EventName, errText, rec.RawMessage, string(metadata))
	return err
}

func (s *Store) ErrorRecords(ctx context.Context, sid sessionstream.SessionId, limit int) ([]sessionstream.ErrorRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	query := `SELECT kind, session_id, ordinal, event_name, error, raw_message, metadata_json FROM sessionstream_errors`
	args := []any{}
	if sid != "" {
		query += ` WHERE session_id = ?`
		args = append(args, string(sid))
	}
	query += ` ORDER BY id ASC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]sessionstream.ErrorRecord, 0)
	for rows.Next() {
		var (
			kind        string
			sessionID   sql.NullString
			ordinal     sql.NullInt64
			eventName   sql.NullString
			errText     string
			rawMessage  []byte
			metadataRaw string
		)
		if err := rows.Scan(&kind, &sessionID, &ordinal, &eventName, &errText, &rawMessage, &metadataRaw); err != nil {
			return nil, err
		}
		rec := sessionstream.ErrorRecord{Kind: sessionstream.ErrorKind(kind), RawMessage: append([]byte(nil), rawMessage...)}
		if sessionID.Valid {
			rec.SessionId = sessionstream.SessionId(sessionID.String)
		}
		if ordinal.Valid {
			ord, err := int64ToUint64(ordinal.Int64)
			if err != nil {
				return nil, err
			}
			rec.Ordinal = ord
		}
		if eventName.Valid {
			rec.EventName = eventName.String
		}
		if errText != "" {
			rec.Err = fmt.Errorf("%s", errText)
		}
		if metadataRaw != "" {
			if err := json.Unmarshal([]byte(metadataRaw), &rec.Metadata); err != nil {
				return nil, err
			}
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) migrate() error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS sessionstream_sessions (
		  session_id TEXT PRIMARY KEY,
		  cursor INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS sessionstream_events (
		  session_id TEXT NOT NULL,
		  ordinal INTEGER NOT NULL,
		  name TEXT NOT NULL,
		  payload_json TEXT NOT NULL,
		  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		  PRIMARY KEY(session_id, ordinal)
		);`,
		`CREATE TABLE IF NOT EXISTS sessionstream_entities (
		  session_id TEXT NOT NULL,
		  kind TEXT NOT NULL,
		  entity_id TEXT NOT NULL,
		  payload_json TEXT NOT NULL,
		  PRIMARY KEY(session_id, kind, entity_id)
		);`,
		`CREATE TABLE IF NOT EXISTS sessionstream_entity_versions (
		  session_id TEXT NOT NULL,
		  kind TEXT NOT NULL,
		  entity_id TEXT NOT NULL,
		  ordinal INTEGER NOT NULL,
		  tombstone INTEGER NOT NULL DEFAULT 0,
		  payload_json TEXT NOT NULL,
		  PRIMARY KEY(session_id, kind, entity_id, ordinal)
		);`,
		`CREATE TABLE IF NOT EXISTS sessionstream_projection_cursors (
		  projector TEXT NOT NULL,
		  session_id TEXT NOT NULL,
		  cursor INTEGER NOT NULL DEFAULT 0,
		  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		  PRIMARY KEY(projector, session_id)
		);`,
		`CREATE TABLE IF NOT EXISTS sessionstream_errors (
		  id INTEGER PRIMARY KEY AUTOINCREMENT,
		  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		  kind TEXT NOT NULL,
		  session_id TEXT,
		  ordinal INTEGER,
		  event_name TEXT,
		  error TEXT NOT NULL,
		  raw_message BLOB,
		  metadata_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`CREATE INDEX IF NOT EXISTS sessionstream_events_by_session ON sessionstream_events(session_id, ordinal);`,
		`CREATE INDEX IF NOT EXISTS sessionstream_entities_by_session ON sessionstream_entities(session_id);`,
		`CREATE INDEX IF NOT EXISTS sessionstream_entity_versions_by_session ON sessionstream_entity_versions(session_id, ordinal);`,
		`CREATE INDEX IF NOT EXISTS sessionstream_projection_cursors_by_session ON sessionstream_projection_cursors(session_id, cursor);`,
		`CREATE INDEX IF NOT EXISTS sessionstream_errors_by_session ON sessionstream_errors(session_id, ordinal);`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

type entityKey struct {
	kind string
	id   string
}

type view struct {
	ordinal uint64
	index   map[entityKey]sessionstream.TimelineEntity
}

func newView(snap sessionstream.Snapshot) *view {
	index := map[entityKey]sessionstream.TimelineEntity{}
	for _, entity := range snap.Entities {
		cloned := entity
		if entity.Payload != nil {
			cloned.Payload = proto.Clone(entity.Payload)
		}
		index[entityKey{kind: entity.Kind, id: entity.Id}] = cloned
	}
	return &view{ordinal: snap.Ordinal, index: index}
}

func (v *view) Get(kind, id string) (sessionstream.TimelineEntity, bool) {
	entity, ok := v.index[entityKey{kind: kind, id: id}]
	if !ok {
		return sessionstream.TimelineEntity{}, false
	}
	cloned := entity
	if entity.Payload != nil {
		cloned.Payload = proto.Clone(entity.Payload)
	}
	return cloned, true
}

func (v *view) List(kind string) []sessionstream.TimelineEntity {
	out := make([]sessionstream.TimelineEntity, 0)
	for _, entity := range v.index {
		if kind != "" && entity.Kind != kind {
			continue
		}
		cloned := entity
		if entity.Payload != nil {
			cloned.Payload = proto.Clone(entity.Payload)
		}
		out = append(out, cloned)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Id < out[j].Id
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func (v *view) Ordinal() uint64 { return v.ordinal }

func uint64ToInt64(v uint64) (int64, error) {
	if v > math.MaxInt64 {
		return 0, fmt.Errorf("value %d overflows int64", v)
	}
	return int64(v), nil
}

func int64ToUint64(v int64) (uint64, error) {
	if v < 0 {
		return 0, fmt.Errorf("value %d cannot be represented as uint64", v)
	}
	return uint64(v), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
