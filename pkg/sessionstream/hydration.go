package sessionstream

import "context"

// HydrationStore is the substrate-owned persistence seam.
type HydrationStore interface {
	Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
	Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
	View(ctx context.Context, sid SessionId) (TimelineView, error)
	Cursor(ctx context.Context, sid SessionId) (uint64, error)
}

// EventStore is the replay-log seam for stores that persist backend events.
type EventStore interface {
	AppendEvent(ctx context.Context, ev Event) error
	Events(ctx context.Context, sid SessionId, after uint64, limit int) ([]Event, error)
	EventCursor(ctx context.Context, sid SessionId) (uint64, error)
}

// ProjectionCursorStore tracks per-projector materialization progress.
type ProjectionCursorStore interface {
	ProjectionCursor(ctx context.Context, projector string, sid SessionId) (uint64, error)
	AdvanceProjectionCursor(ctx context.Context, projector string, sid SessionId, ord uint64) error
}

// TimelineResetStore clears materialized timeline state for stores that support
// full timeline rebuilds from the event log.
type TimelineResetStore interface {
	ClearTimeline(ctx context.Context, sid SessionId) error
}

// ErrorStore is the durable runtime-error/DLQ seam for stores that persist
// projection, fanout, decode, and store errors.
type ErrorStore interface {
	RecordError(ctx context.Context, rec ErrorRecord) error
}

// ErrorRecordStore exposes persisted runtime errors for tests and operator tools.
type ErrorRecordStore interface {
	ErrorRecords(ctx context.Context, sid SessionId, limit int) ([]ErrorRecord, error)
}

// Snapshot is the reconnect payload returned by the store.
type Snapshot struct {
	SessionId       SessionId        `json:"sessionId"`
	SnapshotOrdinal uint64           `json:"snapshotOrdinal"`
	Entities        []TimelineEntity `json:"entities"`
}
