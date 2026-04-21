package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"

	sessionstream "github.com/go-go-golems/sessionstream"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Store struct {
	db  *sql.DB
	reg *sessionstream.SchemaRegistry
}

var _ sessionstream.HydrationStore = (*Store)(nil)

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
		`DELETE FROM evtstream_entities`,
		`DELETE FROM evtstream_sessions`,
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
		if entity.Tombstone {
			if _, err := tx.ExecContext(ctx, `DELETE FROM evtstream_entities WHERE session_id = ? AND kind = ? AND entity_id = ?`, string(sid), entity.Kind, entity.Id); err != nil {
				return err
			}
			continue
		}
		payload, err := s.reg.MarshalProtoJSON(entity.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO evtstream_entities(session_id, kind, entity_id, payload_json)
			VALUES(?, ?, ?, ?)
			ON CONFLICT(session_id, kind, entity_id) DO UPDATE SET payload_json = excluded.payload_json
		`, string(sid), entity.Kind, entity.Id, string(payload)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO evtstream_sessions(session_id, cursor)
		VALUES(?, ?)
		ON CONFLICT(session_id) DO UPDATE SET cursor = CASE
			WHEN excluded.cursor > evtstream_sessions.cursor THEN excluded.cursor
			ELSE evtstream_sessions.cursor
		END
	`, string(sid), cursor); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context, sid sessionstream.SessionId, _ uint64) (sessionstream.Snapshot, error) {
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
	rows, err := s.db.QueryContext(ctx, `SELECT kind, entity_id, payload_json FROM evtstream_entities WHERE session_id = ? ORDER BY kind ASC, entity_id ASC`, string(sid))
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	defer func() { _ = rows.Close() }()
	entities := make([]sessionstream.TimelineEntity, 0)
	for rows.Next() {
		var (
			kind    string
			id      string
			rawJSON string
		)
		if err := rows.Scan(&kind, &id, &rawJSON); err != nil {
			return sessionstream.Snapshot{}, err
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
	err := s.db.QueryRowContext(ctx, `SELECT cursor FROM evtstream_sessions WHERE session_id = ?`, string(sid)).Scan(&cursor)
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

func (s *Store) migrate() error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS evtstream_sessions (
		  session_id TEXT PRIMARY KEY,
		  cursor INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS evtstream_entities (
		  session_id TEXT NOT NULL,
		  kind TEXT NOT NULL,
		  entity_id TEXT NOT NULL,
		  payload_json TEXT NOT NULL,
		  PRIMARY KEY(session_id, kind, entity_id)
		);`,
		`CREATE INDEX IF NOT EXISTS evtstream_entities_by_session ON evtstream_entities(session_id);`,
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
