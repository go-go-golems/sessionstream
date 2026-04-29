---
Title: Remediation plan for replay store and API cleanup
Ticket: SESSIONSTREAM-003
Status: active
Topics:
    - architecture
    - backend
    - event-streaming
    - framework
    - onboarding
    - code-review
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab
      Note: systemlab refactor plan covers phase runtime
    - Path: examples/chatdemo/chat.go
      Note: Generated protobuf chat example migration planned here
    - Path: pkg/sessionstream/consumer.go
      Note: Bus decode error and DLQ reporting behavior planned here
    - Path: pkg/sessionstream/hub.go
      Note: Hub ordinal
    - Path: pkg/sessionstream/hydration.go
      Note: Current HydrationStore contract to be replaced/expanded into replay store semantics
    - Path: pkg/sessionstream/hydration/memory/store.go
      Note: Map-backed memory store replacement with in-memory SQLite evaluated here
    - Path: pkg/sessionstream/hydration/sqlite/store.go
      Note: Primary SQLite replay backend and in-memory SQLite constructor planned here
    - Path: pkg/sessionstream/schema.go
      Note: Schema registry clone-on-register and descriptor API cleanup planned here
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Websocket fanout-only scope and subscribe semantics planned here
ExternalSources: []
Summary: Implementation design for turning sessionstream into a replay-capable event/projection store, tightening failure policies, cleaning extraction-era names, keeping websocket as fanout-only, cloning schemas defensively, and adding a self-contained protobuf chat example.
LastUpdated: 2026-04-29T15:08:15-04:00
WhatFor: Use this as the implementation blueprint after the SESSIONSTREAM-003 code review findings are accepted.
WhenToUse: Read before changing event storage, ordinal assignment, projection policies, error reporting, websocket scope, schema registry cloning, or examples/chatdemo protobuf contracts.
---


# Remediation plan for replay store and API cleanup

## Executive summary

This document turns the SESSIONSTREAM-003 review findings into an implementation plan. The project direction is:

- no backwards compatibility is required for extracted `evtstream` names or ambiguous bootstrap APIs;
- local users should still work with a simple default setup;
- the framework should become a real replay-capable event/projection store instead of only a current-state hydration map;
- local and bus paths should assign ordinals consistently from persisted state;
- projection failure should fail closed by default, with explicit observer/DLQ reporting;
- UI and timeline projection failure policies should be split;
- websocket should be documented and shaped as fanout/subscription only for now, not command ingress;
- schema registration should defensively clone prototypes;
- the chat example should include a self-contained protobuf namespace so users see the intended schema-first pattern.

The core design is to separate three concepts that are currently partially collapsed:

1. **Event log**: append-only source of truth for backend events and ordinals.
2. **Projection state**: current timeline entity state and projection cursors.
3. **Live fanout**: best-effort delivery of UI events to subscribed clients.

A backend event can be persisted successfully while its projections fail. In that case the event remains replayable, the timeline projection cursor does not advance past the failed event, and an error/DLQ record explains what happened. This gives operators a path to retry after fixing projection code or bad data.

## Questions answered before implementation

### 1. In finding 4, is there an update to the intended timeline to apply when projection fails?

No. There is no generally correct timeline update to apply when the timeline projection fails.

The timeline projection is the only component that knows the intended entity changes for a backend event. If it returns an error, the framework cannot safely infer whether it should:

- apply no entities;
- apply a partial entity list;
- apply a tombstone;
- preserve previous state;
- retry later;
- skip permanently.

The current `ProjectionErrorPolicyAdvance` behavior applies no timeline entities but still advances the cursor. That is lossy: the store cursor says the event was handled, while the event's intended timeline state may be missing.

The replay-store design should instead distinguish event persistence from projection progress:

```text
backend event accepted
  ├─ append event to event log with ordinal N
  ├─ run UI projection
  ├─ run timeline projection
  │    ├─ success: apply entities and advance timeline projector cursor to N
  │    └─ failure: do not advance timeline projector cursor; write projection error
  └─ live fanout only if UI policy allows it
```

That means the event is durable and replayable, but the timeline cursor remains at the last successfully projected ordinal. There is no fake timeline update.

If an application wants partial timeline application, it should return those partial entities without returning an error, and optionally publish an explicit domain event like `ProjectionDegraded` or `MessagePartiallyIndexed`. Framework projection errors should mean "do not treat this projection as successfully applied."

### 2. How should error / DLQ reporting work?

Use two layers:

1. **Synchronous observer hook** for application/runtime visibility.
2. **Durable error store / DLQ table** for replay, operations, and tests.

The observer is an interface installed on the hub:

```go
type ErrorObserver interface {
    OnSessionstreamError(ctx context.Context, rec ErrorRecord)
}

type ErrorKind string

const (
    ErrorKindDecode       ErrorKind = "decode"
    ErrorKindOrdinal      ErrorKind = "ordinal"
    ErrorKindUIProjection ErrorKind = "ui-projection"
    ErrorKindTimeline     ErrorKind = "timeline-projection"
    ErrorKindFanout       ErrorKind = "fanout"
    ErrorKindStore        ErrorKind = "store"
)

type ErrorRecord struct {
    Kind        ErrorKind
    SessionId   SessionId
    Ordinal     uint64
    EventName   string
    CommandName string
    Error       error
    Retryable   bool
    RawMessage  []byte
    Metadata    map[string]string
}
```

The durable store keeps error records in SQLite. Suggested tables:

```sql
CREATE TABLE sessionstream_errors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL,
  kind TEXT NOT NULL,
  session_id TEXT,
  ordinal INTEGER,
  event_name TEXT,
  command_name TEXT,
  retryable INTEGER NOT NULL,
  error TEXT NOT NULL,
  raw_message BLOB,
  metadata_json TEXT
);

CREATE TABLE sessionstream_projection_runs (
  projector TEXT NOT NULL,
  session_id TEXT NOT NULL,
  cursor INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(projector, session_id)
);
```

For bus decode errors, the event may not have a valid session id or event name. Those records still go into `sessionstream_errors` with `kind='decode'`, raw payload, and Watermill metadata.

For projection errors, the event has an ordinal and should already be in the event log. The error record references `(session_id, ordinal, projector)` so a replay command can retry from the failed ordinal.

Policy sketch:

```go
func handleProjection(ctx, ev) error {
    uiEvents, uiErr := runUIProjection(...)
    timelineEntities, tlErr := runTimelineProjection(...)

    if uiErr != nil {
        report(ErrorKindUIProjection, ev, uiErr)
        if policies.UI == ProjectionErrorPolicyFail {
            return uiErr
        }
    }

    if tlErr != nil {
        report(ErrorKindTimeline, ev, tlErr)
        if policies.Timeline == ProjectionErrorPolicyFail {
            return tlErr // timeline cursor does not advance
        }
    }

    if tlErr == nil {
        store.ApplyTimeline(ctx, ev.SessionId, ev.Ordinal, timelineEntities)
        store.AdvanceProjectionCursor(ctx, "timeline", ev.SessionId, ev.Ordinal)
    }

    if uiErr == nil {
        fanout.PublishUI(...)
    }
    return nil
}
```

### 3. How should systemlab be split/refactored?

Split by reusable runtime mechanics first, not by arbitrary line count.

Current systemlab phase files repeat:

- schema registration;
- hub/store/websocket setup;
- websocket hook construction;
- trace logging;
- action dispatch;
- response construction;
- snapshot encoding;
- checks;
- clone helpers.

Proposed structure:

```text
cmd/sessionstream-systemlab/
├── main.go
├── server.go                    # routes only; small handlers delegate to phase controllers
├── http_helpers.go              # decode JSON, write JSON, export helpers
├── phase_runtime.go             # common hub/store/ws runtime assembly helpers
├── trace_log.go                 # shared traceEntry + concurrent append/snapshot
├── snapshot_encoding.go         # encodeSnapshot, encode entities, clone maps
├── ws_hooks.go                  # common websocket hook builder for traces
├── checks.go                    # shared check helpers where genuinely common
├── phase1_lab.go                # Phase 1 lesson/domain-specific behavior
├── phase2/
│   ├── runtime.go               # bus runtime, publisher/subscriber setup
│   ├── actions.go               # publish-a, publish-b, burst-a, restart-consumer
│   ├── projections.go
│   ├── checks.go
│   └── render.go
├── phase3/
│   ├── runtime.go
│   ├── actions.go
│   ├── projections.go
│   ├── checks.go
│   └── render.go
├── phase4/
│   ├── runtime.go
│   ├── actions.go
│   └── checks.go
└── phase5/
    ├── runtime.go               # memory/sqlite runtime, restart lifecycle
    ├── actions.go               # seed-session, restart-backend, reset
    ├── projections.go
    ├── checks.go
    └── render.go
```

Keep each phase's educational narrative visible. Do not over-abstract the actual lesson. Extract only mechanics that obscure the lesson.

A useful intermediate abstraction:

```go
type traceLog struct {
    mu      sync.Mutex
    entries []traceEntry
}

func (l *traceLog) Append(kind, message string, details map[string]any)
func (l *traceLog) Snapshot() []traceEntry
func (l *traceLog) Reset()

type phaseRuntime struct {
    Reg    *sessionstream.SchemaRegistry
    Store  sessionstream.ReplayStore
    Hub    *sessionstream.Hub
    Fanout *wstransport.Server
    Trace  *traceLog
}
```

The first refactor should target trace handling and websocket hook duplication, because those are repeated in phases 3, 4, and 5 and do not teach core sessionstream concepts.

### 4. Can the memory store be an in-memory SQLite database so there is basically a single backend?

Yes, and that is a good direction if the goal is a real replay store with one consistent semantics surface.

Instead of maintaining separate in-memory and SQLite store implementations, provide one SQLite-backed store with two constructors:

```go
func NewSQLiteStore(dsn string, reg *SchemaRegistry) (*sqlite.Store, error)
func NewInMemoryStore(reg *SchemaRegistry) (*sqlite.Store, error) {
    return NewSQLiteStore("file:sessionstream?mode=memory&cache=shared&_foreign_keys=on", reg)
}
```

However, a plain `:memory:` DSN creates one database per connection. Since `database/sql` uses a connection pool, use one of these safer approaches:

```text
file:sessionstream-memory?mode=memory&cache=shared&_foreign_keys=on
```

and either:

```go
db.SetMaxOpenConns(1)
```

or keep a named shared-cache DSN and ensure the store owns the DB for the process lifetime.

Pros:

- one schema;
- one set of replay semantics;
- one implementation of snapshots, event log, cursors, and errors;
- tests exercise the same backend shape as production;
- less duplicate view cloning/sorting logic.

Cons:

- SQLite and CGO become part of the default local path unless a pure-Go driver is chosen;
- very small examples pay a little setup cost;
- true in-memory Go map store is simpler for unit tests that do not care about replay.

Recommendation: make SQLite the primary replay backend and expose an in-memory SQLite constructor as the default local store. Keep a tiny no-op store only if needed for tests or explicit dry hubs. Remove the separate map-backed memory store once the SQLite replay store is stable.

## Target architecture

### Before

```text
CommandHandler publishes Event
  ├─ local path assigns in-memory ordinal
  └─ bus path serializes event and consumer assigns cursor-seeded ordinal

projectAndApply
  ├─ loads current view
  ├─ runs UI projection
  ├─ runs timeline projection
  ├─ applies timeline entities to current-state store
  └─ sends UI events to fanout
```

### After

```text
CommandHandler publishes Event
  │
  ▼
EventAppender assigns ordinal from durable session cursor
  │
  ├─ append event to sessionstream_events
  │
  ▼
ProjectionRunner
  ├─ loads TimelineView from projection state
  ├─ runs UI projection under UI policy
  ├─ runs timeline projection under Timeline policy
  ├─ on timeline success: apply entity versions/current state and advance timeline cursor
  ├─ on projection failure: record error/DLQ and leave failed projection cursor unadvanced
  └─ on UI success: send UI events to fanout
```

The durable store should know about:

- event log;
- current timeline state;
- projection cursors;
- error/DLQ records.

## Proposed API changes

### Store interfaces

Replace the current `HydrationStore` with a replay-oriented store.

```go
type ReplayStore interface {
    AppendEvent(ctx context.Context, ev Event) (Event, error)
    Events(ctx context.Context, sid SessionId, after uint64, limit int) ([]Event, error)

    ApplyTimeline(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)

    EventCursor(ctx context.Context, sid SessionId) (uint64, error)
    ProjectionCursor(ctx context.Context, projector string, sid SessionId) (uint64, error)
    AdvanceProjectionCursor(ctx context.Context, projector string, sid SessionId, ord uint64) error

    RecordError(ctx context.Context, rec ErrorRecord) error
}
```

If this feels too large, split it internally but keep one constructor for users:

```go
type EventStore interface { AppendEvent(...); Events(...); EventCursor(...) }
type ProjectionStore interface { ApplyTimeline(...); Snapshot(...); View(...); ProjectionCursor(...); AdvanceProjectionCursor(...) }
type ErrorStore interface { RecordError(...) }
```

`Hub` can accept the combined interface while implementation packages keep smaller interfaces internally.

### Projection policies

Replace the single policy with split policies and make fail the default.

```go
type ProjectionErrorPolicy int

const (
    ProjectionErrorPolicyFail ProjectionErrorPolicy = iota
    ProjectionErrorPolicyAdvance
)

type ProjectionPolicies struct {
    UI       ProjectionErrorPolicy
    Timeline ProjectionErrorPolicy
}

func WithProjectionPolicies(p ProjectionPolicies) HubOption
func WithUIProjectionErrorPolicy(p ProjectionErrorPolicy) HubOption
func WithTimelineProjectionErrorPolicy(p ProjectionErrorPolicy) HubOption
```

Default:

```go
ProjectionPolicies{
    UI:       ProjectionErrorPolicyFail,
    Timeline: ProjectionErrorPolicyFail,
}
```

Potentially useful local-user default:

- Fail by default, but errors are returned from `Submit` in the local path, so local tests fail loudly.
- Users can explicitly choose `Advance` for lossy UI projection or best-effort demos.

### Error observer

```go
func WithErrorObserver(observer ErrorObserver) HubOption
```

Rules:

- Always call observer before returning the error.
- Best effort: observer failures should not hide the original error unless a strict observer option is added later.
- Also write to `ReplayStore.RecordError` when a store is configured.

### Schema registry defensive cloning

Change register and lookup behavior:

```go
func (r *SchemaRegistry) register(..., msg proto.Message) error {
    m[name] = proto.Clone(msg)
}

func (r *SchemaRegistry) lookup(..., name string) (proto.Message, bool) {
    // return clone to avoid mutating stored prototype
}

func (r *SchemaRegistry) instantiate(...) (proto.Message, error) {
    // instantiate from stored prototype descriptor
}
```

If callers need descriptors, add descriptor-oriented methods instead of returning mutable prototypes:

```go
func (r *SchemaRegistry) CommandDescriptor(name string) (protoreflect.MessageDescriptor, bool)
```

### Websocket fanout-only scope

Make the package contract explicit:

- The websocket server subscribes clients to sessions.
- On subscribe it sends snapshots.
- It fans out live UI events.
- It does not accept command frames.

Rename or document generic transport abstractions accordingly. Options:

1. Delete `pkg/sessionstream/transport/transport.go` until command ingress exists.
2. Rename websocket docs/comments to "fanout transport".
3. Keep `Command.ConnectionId` only if near-term command ingress exists elsewhere; otherwise remove it in the no-backcompat cleanup.

Since no backwards compatibility is required, prefer deletion/removal of orphan abstractions.

### Clean `evtstream` names outside `ttmp`

Rename runtime names outside ticket docs:

```text
DefaultEventBusTopic: evtstream.events -> sessionstream.events
MetadataKeyEventName: evtstream_event_name -> sessionstream_event_name
MetadataKeySessionID: evtstream_session_id -> sessionstream_session_id
MetadataKeyPartitionKey: evtstream_partition_key -> sessionstream_partition_key
MetadataKeyPublishedOrd: evtstream_published_ordinal -> sessionstream_published_ordinal
MetadataKeyStreamID: evtstream_stream_id -> sessionstream_stream_id
SQLite tables: evtstream_sessions/entities -> sessionstream_sessions/entities
```

Do not attempt compatibility migrations unless a real deployed database must be preserved. The user's direction is no backwards compatibility needed.

### Self-contained protobuf chat example

Add a namespaced protobuf example rather than relying only on `structpb.Struct`.

Suggested layout:

```text
examples/chatdemo/proto/chatdemo/v1/chat.proto
examples/chatdemo/gen/chatdemo/v1/chat.pb.go
examples/chatdemo/chat.go
```

Proto sketch:

```proto
syntax = "proto3";

package sessionstream.examples.chatdemo.v1;

option go_package = "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/chatdemo/v1;chatdemov1";

message StartInferenceCommand {
  string prompt = 1;
}

message StopInferenceCommand {}

message UserMessageAcceptedEvent {
  string message_id = 1;
  string role = 2;
  string content = 3;
}

message InferenceStartedEvent {
  string message_id = 1;
  string prompt = 2;
  string role = 3;
}

message TokensDeltaEvent {
  string message_id = 1;
  string chunk = 2;
  string text = 3;
}

message InferenceFinishedEvent {
  string message_id = 1;
  string text = 2;
}

message InferenceStoppedEvent {
  string message_id = 1;
  string text = 2;
}

message ChatMessageEntity {
  string message_id = 1;
  string role = 2;
  string content = 3;
  string status = 4;
  bool streaming = 5;
}
```

The demo can still expose convenient Go methods like `SubmitPrompt(ctx, sid, prompt)`, but internally it should build generated protobuf messages.

## Implementation plan

### Phase 1: Rename and remove misleading API surface

1. Rename `evtstream` runtime constants, metadata keys, and SQLite table names to `sessionstream`.
2. Remove or quarantine `pkg/sessionstream/transport/transport.go` if it remains unused.
3. Update comments that call websocket "Phase 3" or imply more than fanout-only behavior.
4. Make README/package docs state websocket is fanout/subscription only.

Validation:

```bash
rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'
go test ./...
```

### Phase 2: Introduce replay store schema

1. Create new SQLite schema with:
   - `sessionstream_events`;
   - `sessionstream_timeline_entities` or entity current-state table;
   - optional `sessionstream_timeline_entity_versions` if `Snapshot(asOf)` must be efficient;
   - `sessionstream_projection_cursors`;
   - `sessionstream_errors`.
2. Add `NewInMemoryStore(reg)` using named in-memory SQLite.
3. Port current memory-store tests to the SQLite in-memory constructor.
4. Keep local users simple:

```go
hub, err := sessionstream.NewHub()
```

should still create a working hub with an in-memory replay store, or clear docs should require:

```go
store, _ := sqlite.NewInMemory(reg)
hub, _ := sessionstream.NewHub(sessionstream.WithReplayStore(store))
```

Recommendation: keep `NewHub()` working by installing an in-memory SQLite replay store after its default registry is created.

### Phase 3: Seed ordinals from the event store

1. Replace `Hub.localOrdinal` with `OrdinalAssigner` backed by `ReplayStore.EventCursor`.
2. Ensure bus consumer and local publisher share ordinal assignment semantics.
3. Append events before projection.
4. Add restart tests for local path with persistent SQLite.

### Phase 4: Split projection policies and add error reporting

1. Add `ProjectionPolicies` with fail/fail default.
2. Add `ErrorObserver` and durable `RecordError` path.
3. Change projection behavior so timeline cursor advances only on timeline projection success.
4. Add tests for:
   - UI fail policy;
   - UI advance policy;
   - timeline fail policy;
   - timeline advance policy;
   - observer called with expected `ErrorRecord`;
   - durable error record written.

### Phase 5: Make replay real

1. Implement `Events(ctx, sid, after, limit)`.
2. Implement `Snapshot(ctx, sid, asOf)` either from entity versions or replay.
3. Add replay command/helper:

```go
func (h *Hub) Replay(ctx context.Context, sid SessionId, from uint64) error
```

or lower-level projector method:

```go
func (h *Hub) RebuildTimeline(ctx context.Context, sid SessionId, from uint64) error
```

4. Decide whether UI events are replayed live during rebuild. Default should probably be no.

### Phase 6: Add protobuf chat example

1. Add `examples/chatdemo/proto/chatdemo/v1/chat.proto`.
2. Add codegen config or documented `go generate` command.
3. Update `examples/chatdemo/chat.go` to use generated messages.
4. Keep the example self-contained and namespaced.
5. Update tests to assert generated protobuf payload types.

### Phase 7: Refactor systemlab

1. Extract trace log helper.
2. Extract snapshot encoding helper.
3. Extract websocket hook helper.
4. Move Phase 2 and Phase 5 into subdirectories or smaller files.
5. Update systemlab docs/chapter text if file paths change.

## Test strategy

High-priority tests:

```go
func TestLocalPublisherSeedsOrdinalFromReplayStoreCursor(t *testing.T)
func TestTimelineProjectionFailureDoesNotAdvanceTimelineCursorByDefault(t *testing.T)
func TestUIProjectionFailurePolicyIsIndependentFromTimelinePolicy(t *testing.T)
func TestProjectionErrorObserverAndStoreReceiveFailureRecord(t *testing.T)
func TestDecodeErrorRecordsDeadLetter(t *testing.T)
func TestSnapshotAsOfReturnsHistoricalState(t *testing.T)
func TestWebsocketIsFanoutOnlyAndRejectsCommandFrames(t *testing.T)
func TestSchemaRegistryClonesOnRegisterAndLookup(t *testing.T)
func TestChatDemoUsesGeneratedProtoSchemas(t *testing.T)
```

Regression command set:

```bash
go test ./...
rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'
go test ./pkg/sessionstream/... -race
```

## Risks and tradeoffs

### SQLite as the only backend

Using SQLite for both persistent and in-memory modes reduces semantic drift, but it makes SQLite a core dependency. This is acceptable if `sessionstream` wants replay semantics soon. If zero-CGO portability is a hard requirement, evaluate a pure-Go SQLite driver before making this the default.

### Event append before projection

Appending the event before projections means the event log can contain events that failed projection. That is intentional. Projection cursors and error tables tell operators what has and has not been materialized.

### No backwards compatibility

Renaming tables and constants is cleaner, but existing local databases created during bootstrap will not migrate. This matches the user's instruction. Keep this out of `ttmp` docs: historical tickets can still mention `evtstream`; runtime code should not.

### Local users still work

`NewHub()` should remain useful for tests and demos. The cleanest implementation is to install an in-memory SQLite replay store by default. If construction can fail because SQLite setup can fail, `NewHub()` already returns `(*Hub, error)`, so this remains compatible with the existing error-returning constructor style.

## References

- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/bus.go`
- `pkg/sessionstream/consumer.go`
- `pkg/sessionstream/ordinals.go`
- `pkg/sessionstream/schema.go`
- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/hydration/memory/store.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `examples/chatdemo/chat.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
