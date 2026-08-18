// Package mysql provides a MySQL/Aurora implementation of the sessionstream
// hydration stores (HydrationStore, EventStore, ProjectionCursorStore,
// TimelineResetStore, ErrorStore, ErrorRecordStore). It is a dialect
// translation of the sqlite store: the same six tables, the same columns, and
// the same per-event write paths. It is selected by a non-empty timeline-dsn;
// an empty DSN keeps the sqlite store.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Store is a MySQL-backed implementation of the sessionstream hydration
// interfaces. It owns a bounded database/sql connection pool.
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

const (
	schemaVersionTable = "sessionstream_schema_version"
	schemaComponent    = "hydration"
	schemaVersion      = int64(1)
)

var managedTables = []string{
	"sessionstream_sessions",
	"sessionstream_events",
	"sessionstream_entities",
	"sessionstream_entity_versions",
	"sessionstream_projection_cursors",
	"sessionstream_errors",
}

// Open opens a bounded MySQL pool, migrates the six-table schema if needed,
// and returns a Store. The dsn must be a go-sql-driver/mysql DSN with
// parseTime=true.
func Open(ctx context.Context, dsn string, reg *sessionstream.SchemaRegistry) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("mysql hydration store dsn is empty")
	}
	if reg == nil {
		return nil, fmt.Errorf("mysql hydration store schema registry is nil")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0) // long-lived; tune in Phase 4
	store := &Store{db: db, reg: reg}
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("mysql hydration store migrate: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Reset clears all timeline tables (used by tests). Production callers use
// ClearTimeline for per-session resets.
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

func (s *Store) migrate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := s.ensureSchemaVersion(ctx); err != nil {
		return err
	}
	// Six tables mirroring the sqlite schema with MySQL types. Opaque identity
	// columns are VARBINARY so MySQL never applies collation, normalization, or
	// trailing-space rules to domain keys. Lengths are byte limits, not rune
	// limits; oversized keys fail rather than being silently truncated.
	// created_at uses DATETIME(3) (millisecond precision) with server-side
	// defaults, replacing sqlite's strftime('%Y-%m-%dT%H:%M:%fZ','now').
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS sessionstream_sessions (
		  session_id VARBINARY(191) NOT NULL PRIMARY KEY,
		  snapshot_ordinal BIGINT NOT NULL DEFAULT 0
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
		`CREATE TABLE IF NOT EXISTS sessionstream_events (
		  session_id VARBINARY(191) NOT NULL,
		  ordinal BIGINT NOT NULL,
		  name VARBINARY(128) NOT NULL,
		  payload_json MEDIUMTEXT NOT NULL,
		  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		  PRIMARY KEY(session_id, ordinal),
		  KEY sessionstream_events_by_session (session_id, ordinal)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
		`CREATE TABLE IF NOT EXISTS sessionstream_entities (
		  session_id VARBINARY(191) NOT NULL,
		  kind VARBINARY(128) NOT NULL,
		  entity_id VARBINARY(255) NOT NULL,
		  created_ordinal BIGINT NOT NULL,
		  last_event_ordinal BIGINT NOT NULL,
		  payload_json MEDIUMTEXT NOT NULL,
		  PRIMARY KEY(session_id, kind, entity_id),
		  KEY sessionstream_entities_by_session (session_id, created_ordinal, last_event_ordinal)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
		`CREATE TABLE IF NOT EXISTS sessionstream_entity_versions (
		  session_id VARBINARY(191) NOT NULL,
		  kind VARBINARY(128) NOT NULL,
		  entity_id VARBINARY(255) NOT NULL,
		  ordinal BIGINT NOT NULL,
		  created_ordinal BIGINT NOT NULL,
		  last_event_ordinal BIGINT NOT NULL,
		  tombstone TINYINT NOT NULL DEFAULT 0,
		  payload_json MEDIUMTEXT NOT NULL,
		  PRIMARY KEY(session_id, kind, entity_id, ordinal),
		  KEY sessionstream_entity_versions_by_session (session_id, ordinal),
		  KEY sessionstream_entity_versions_ordering (session_id, created_ordinal, last_event_ordinal)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
		`CREATE TABLE IF NOT EXISTS sessionstream_projection_cursors (
		  projector VARBINARY(128) NOT NULL,
		  session_id VARBINARY(191) NOT NULL,
		  cursor_ordinal BIGINT NOT NULL DEFAULT 0,
		  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
		  PRIMARY KEY(projector, session_id),
		  KEY sessionstream_projection_cursors_by_session (session_id, cursor_ordinal)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
		`CREATE TABLE IF NOT EXISTS sessionstream_errors (
		  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		  kind VARBINARY(64) NOT NULL,
		  session_id VARBINARY(191),
		  ordinal BIGINT,
		  event_name VARBINARY(128),
		  error TEXT NOT NULL,
		  raw_message BLOB,
		  metadata_json MEDIUMTEXT NOT NULL,
		  KEY sessionstream_errors_by_session (session_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensureSchemaVersion(ctx context.Context) error {
	exists, err := s.tableExists(ctx, schemaVersionTable)
	if err != nil {
		return fmt.Errorf("check mysql hydration schema version table: %w", err)
	}
	if !exists {
		for _, table := range managedTables {
			exists, err := s.tableExists(ctx, table)
			if err != nil {
				return fmt.Errorf("check mysql hydration table %s: %w", table, err)
			}
			if exists {
				return fmt.Errorf("mysql hydration store found unversioned prototype table %q; remove the prototype schema or migrate it to schema version %d before opening", table, schemaVersion)
			}
		}
		if _, err := s.db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS sessionstream_schema_version (
			  component VARBINARY(64) NOT NULL PRIMARY KEY,
			  schema_version BIGINT NOT NULL
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin
		`); err != nil {
			return fmt.Errorf("create mysql hydration schema version table: %w", err)
		}
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO sessionstream_schema_version(component, schema_version)
			VALUES(?, ?)
			ON DUPLICATE KEY UPDATE schema_version = sessionstream_schema_version.schema_version
		`, schemaComponent, schemaVersion); err != nil {
			return fmt.Errorf("initialize mysql hydration schema version: %w", err)
		}
	}

	var version int64
	err = s.db.QueryRowContext(ctx, `
		SELECT schema_version
		FROM sessionstream_schema_version
		WHERE component = ?
	`, schemaComponent).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("mysql hydration schema version row %q is missing; refusing to guess schema compatibility", schemaComponent)
	}
	if err != nil {
		return fmt.Errorf("read mysql hydration schema version: %w", err)
	}
	if version != schemaVersion {
		return fmt.Errorf("unsupported mysql hydration schema version %d for component %q; supported version is %d", version, schemaComponent, schemaVersion)
	}
	return nil
}

func (s *Store) tableExists(ctx context.Context, table string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = DATABASE() AND table_name = ?
		)
	`, table).Scan(&exists)
	return exists != 0, err
}

// Apply is the entity-materialization write path, ported from the sqlite
// store. It runs in one transaction: upsert each entity version, then upsert
// (or tombstone-delete) the current entity, then advance the session snapshot
// ordinal.
func (s *Store) Apply(ctx context.Context, sid sessionstream.SessionId, ord uint64, entities []sessionstream.TimelineEntity) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	eventOrdinal, err := uint64ToInt64(ord)
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

		createdOrdinal := entity.CreatedOrdinal
		lastEventOrdinal := entity.LastEventOrdinal
		if createdOrdinal == 0 {
			createdOrdinal = ord
		}
		if lastEventOrdinal == 0 {
			lastEventOrdinal = ord
		}

		var existingCreated sql.NullInt64
		err = tx.QueryRowContext(ctx, `
			SELECT created_ordinal
			FROM sessionstream_entities
			WHERE session_id = ? AND kind = ? AND entity_id = ?
		`, string(sid), entity.Kind, entity.Id).Scan(&existingCreated)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && existingCreated.Valid && entity.CreatedOrdinal == 0 {
			createdOrdinal, err = int64ToUint64(existingCreated.Int64)
			if err != nil {
				return err
			}
		}

		createdOrdinalDB, err := uint64ToInt64(createdOrdinal)
		if err != nil {
			return err
		}
		lastEventOrdinalDB, err := uint64ToInt64(lastEventOrdinal)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessionstream_entity_versions(session_id, kind, entity_id, ordinal, created_ordinal, last_event_ordinal, tombstone, payload_json)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?) AS new
			ON DUPLICATE KEY UPDATE
				created_ordinal = new.created_ordinal,
				last_event_ordinal = new.last_event_ordinal,
				tombstone = new.tombstone,
				payload_json = new.payload_json
		`, string(sid), entity.Kind, entity.Id, eventOrdinal, createdOrdinalDB, lastEventOrdinalDB, boolToInt(entity.Tombstone), string(payload)); err != nil {
			return err
		}
		if entity.Tombstone {
			if _, err := tx.ExecContext(ctx, `DELETE FROM sessionstream_entities WHERE session_id = ? AND kind = ? AND entity_id = ?`, string(sid), entity.Kind, entity.Id); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sessionstream_entities(session_id, kind, entity_id, created_ordinal, last_event_ordinal, payload_json)
			VALUES(?, ?, ?, ?, ?, ?) AS new
			ON DUPLICATE KEY UPDATE
				created_ordinal = new.created_ordinal,
				last_event_ordinal = new.last_event_ordinal,
				payload_json = new.payload_json
		`, string(sid), entity.Kind, entity.Id, createdOrdinalDB, lastEventOrdinalDB, string(payload)); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessionstream_sessions(session_id, snapshot_ordinal)
		VALUES(?, ?) AS new
		ON DUPLICATE KEY UPDATE snapshot_ordinal = CASE
			WHEN new.snapshot_ordinal > sessionstream_sessions.snapshot_ordinal THEN new.snapshot_ordinal
			ELSE sessionstream_sessions.snapshot_ordinal
		END
	`, string(sid), eventOrdinal); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context, sid sessionstream.SessionId, asOf uint64) (sessionstream.Snapshot, error) {
	if s == nil || s.db == nil {
		return sessionstream.Snapshot{}, fmt.Errorf("mysql hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return sessionstream.Snapshot{}, fmt.Errorf("begin mysql hydration snapshot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var cursor sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT snapshot_ordinal
		FROM sessionstream_sessions
		WHERE session_id = ?
	`, string(sid)).Scan(&cursor)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return sessionstream.Snapshot{}, err
		}
		return sessionstream.Snapshot{SessionId: sid}, nil
	}
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	snapshotOrdinal, err := int64ToUint64(cursor.Int64)
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	if asOf > 0 && asOf < snapshotOrdinal {
		snapshotOrdinal = asOf
	}
	versionOrdinal, err := uint64ToInt64(snapshotOrdinal)
	if err != nil {
		return sessionstream.Snapshot{}, err
	}

	var rows *sql.Rows
	if asOf == 0 {
		rows, err = tx.QueryContext(ctx, `
			SELECT kind, entity_id, created_ordinal, last_event_ordinal, 0 AS tombstone, payload_json
			FROM sessionstream_entities
			WHERE session_id = ? AND created_ordinal <= ? AND last_event_ordinal <= ?
			ORDER BY created_ordinal ASC, last_event_ordinal ASC, kind ASC, entity_id ASC
		`, string(sid), versionOrdinal, versionOrdinal)
	} else {
		rows, err = tx.QueryContext(ctx, `
			SELECT v.kind, v.entity_id, v.created_ordinal, v.last_event_ordinal, v.tombstone, v.payload_json
			FROM sessionstream_entity_versions v
			JOIN (
				SELECT kind, entity_id, MAX(ordinal) AS ordinal
				FROM sessionstream_entity_versions
				WHERE session_id = ? AND ordinal <= ?
				GROUP BY kind, entity_id
			) latest
			ON latest.kind = v.kind AND latest.entity_id = v.entity_id AND latest.ordinal = v.ordinal
			WHERE v.session_id = ? AND v.created_ordinal <= ? AND v.last_event_ordinal <= ?
			ORDER BY v.created_ordinal ASC, v.last_event_ordinal ASC, v.kind ASC, v.entity_id ASC
		`, string(sid), versionOrdinal, string(sid), versionOrdinal, versionOrdinal)
	}
	if err != nil {
		return sessionstream.Snapshot{}, err
	}
	defer func() { _ = rows.Close() }()
	entities := make([]sessionstream.TimelineEntity, 0)
	for rows.Next() {
		var (
			kind             string
			id               string
			createdOrdinal   int64
			lastEventOrdinal int64
			tombstone        int
			rawJSON          string
		)
		if err := rows.Scan(&kind, &id, &createdOrdinal, &lastEventOrdinal, &tombstone, &rawJSON); err != nil {
			return sessionstream.Snapshot{}, err
		}
		createdOrdinalU64, err := int64ToUint64(createdOrdinal)
		if err != nil {
			return sessionstream.Snapshot{}, err
		}
		lastEventOrdinalU64, err := int64ToUint64(lastEventOrdinal)
		if err != nil {
			return sessionstream.Snapshot{}, err
		}
		if createdOrdinalU64 > snapshotOrdinal || lastEventOrdinalU64 > snapshotOrdinal {
			return sessionstream.Snapshot{}, fmt.Errorf("mysql hydration snapshot returned entity %q/%q at event ordinal %d beyond snapshot ordinal %d", kind, id, lastEventOrdinalU64, snapshotOrdinal)
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
		entities = append(entities, sessionstream.TimelineEntity{Kind: kind, Id: id, CreatedOrdinal: createdOrdinalU64, LastEventOrdinal: lastEventOrdinalU64, Payload: msg})
	}
	if err := rows.Err(); err != nil {
		return sessionstream.Snapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return sessionstream.Snapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return sessionstream.Snapshot{}, err
	}
	return sessionstream.Snapshot{SessionId: sid, SnapshotOrdinal: snapshotOrdinal, Entities: entities}, nil
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
		return 0, fmt.Errorf("mysql hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var cursor sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT snapshot_ordinal FROM sessionstream_sessions WHERE session_id = ?`, string(sid)).Scan(&cursor)
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
		return fmt.Errorf("mysql hydration store db is nil")
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
		`UPDATE sessionstream_sessions SET snapshot_ordinal = 0 WHERE session_id = ?`,
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
		return 0, fmt.Errorf("mysql hydration store db is nil")
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

// AppendEvent is the durable event-log write path. It uses INSERT ... ON
// DUPLICATE KEY UPDATE to enforce the same idempotency contract as the sqlite
// store: re-appending an identical (session, ordinal) is a no-op; re-appending
// a differing event at the same ordinal returns an "event conflict" error.
func (s *Store) AppendEvent(ctx context.Context, ev sessionstream.Event) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql hydration store db is nil")
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
	// Duplicate detection is independent of RowsAffected semantics, which the
	// caller's DSN can change (clientFoundRows=true reports matched duplicates
	// as affected, defeating an inserted>0 check). Instead: first SELECT the
	// existing row; if present, enforce the sqlite store's identical-duplicate
	// contract; if absent, INSERT, and on a concurrent dup-key race re-check.
	sid := string(ev.SessionId)
	var existingName, existingPayload string
	err = s.db.QueryRowContext(ctx, `
		SELECT name, payload_json FROM sessionstream_events WHERE session_id = ? AND ordinal = ?
	`, sid, ordinal).Scan(&existingName, &existingPayload)
	if err == nil {
		// Row already exists: identical re-append is a no-op; a differing event
		// at the same ordinal is a conflict.
		if existingName == ev.Name && existingPayload == string(payload) {
			return nil
		}
		return fmt.Errorf("event conflict for session %s ordinal %d: existing event %q differs from %q", ev.SessionId, ev.Ordinal, existingName, ev.Name)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	// No existing row: insert. A concurrent appender can race the same
	// (session, ordinal); on dup-key, re-read and apply the same comparison.
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessionstream_events(session_id, ordinal, name, payload_json)
		VALUES(?, ?, ?, ?)
	`, sid, ordinal, ev.Name, string(payload)); err != nil {
		// mysql driver exposes dup-key as a MySQLError with Number 1062.
		if isMySQLDupKey(err) {
			if err := s.db.QueryRowContext(ctx, `
				SELECT name, payload_json FROM sessionstream_events WHERE session_id = ? AND ordinal = ?
			`, sid, ordinal).Scan(&existingName, &existingPayload); err != nil {
				return err
			}
			if existingName == ev.Name && existingPayload == string(payload) {
				return nil
			}
			return fmt.Errorf("event conflict for session %s ordinal %d: existing event %q differs from %q", ev.SessionId, ev.Ordinal, existingName, ev.Name)
		}
		return err
	}
	return nil
}

func (s *Store) Events(ctx context.Context, sid sessionstream.SessionId, after uint64, limit int) ([]sessionstream.Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("mysql hydration store db is nil")
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
		return 0, fmt.Errorf("mysql hydration store db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if projector == "" {
		return 0, fmt.Errorf("projector is empty")
	}
	var cursor sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT cursor_ordinal FROM sessionstream_projection_cursors WHERE projector = ? AND session_id = ?`, projector, string(sid)).Scan(&cursor)
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
		return fmt.Errorf("mysql hydration store db is nil")
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
		INSERT INTO sessionstream_projection_cursors(projector, session_id, cursor_ordinal)
		VALUES(?, ?, ?) AS new
		ON DUPLICATE KEY UPDATE cursor_ordinal = CASE
			WHEN new.cursor_ordinal > sessionstream_projection_cursors.cursor_ordinal THEN new.cursor_ordinal
			ELSE sessionstream_projection_cursors.cursor_ordinal
		END
	`, projector, string(sid), cursor)
	return err
}

func (s *Store) RecordError(ctx context.Context, rec sessionstream.ErrorRecord) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("mysql hydration store db is nil")
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
	// metadata_json is NOT NULL without a server default (MySQL strict mode forbids
	// defaults on TEXT columns); ensure a valid JSON object even for nil metadata.
	if len(metadata) == 0 || string(metadata) == "null" {
		metadata = []byte("{}")
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
		return nil, fmt.Errorf("mysql hydration store db is nil")
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

// ---- view (mirrors the sqlite store's in-memory view) ----

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
	return &view{ordinal: snap.SnapshotOrdinal, index: index}
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

// ---- int conversion helpers (mirror the sqlite store) ----

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

// mysqlDupKeyNumber is the go-sql-driver/mysql error number for a duplicate-key
// violation (ER_DUP_ENTRY), used to detect a concurrent insert of the same
// (session_id, ordinal) without relying on RowsAffected semantics.
const mysqlDupKeyNumber = 1062

// isMySQLDupKey reports whether err is a MySQL duplicate-key error (1062).
func isMySQLDupKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDupKeyNumber
}
