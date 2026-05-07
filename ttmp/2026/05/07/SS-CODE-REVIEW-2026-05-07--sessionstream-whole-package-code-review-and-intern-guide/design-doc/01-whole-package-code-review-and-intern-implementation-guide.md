---
Title: Whole-package code review and intern implementation guide
Ticket: SS-CODE-REVIEW-2026-05-07
Status: active
Topics:
    - sessionstream
    - code-review
    - cleanup
    - architecture
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/README.md
      Note: Systemlab boundary and onboarding app reviewed as part of package architecture
    - Path: examples/chatdemo/chat.go
      Note: Reference schema-first chat demo reviewed for onboarding value and file organization
    - Path: pkg/sessionstream/hub.go
      Note: Core Hub command/event/projection/replay pipeline reviewed in the architecture guide
    - Path: pkg/sessionstream/hydration/sqlite/store.go
      Note: SQLite hydration/replay/error store reviewed for migration
    - Path: pkg/sessionstream/pipeline_observer.go
      Note: Recent Hub pipeline observer API reviewed for diagnostics and panic-safety comparisons
    - Path: pkg/sessionstream/transport/ws/observer.go
      Note: Recent transport observer API reviewed for diagnostics clarity
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Websocket snapshot/fanout/hydration buffering implementation reviewed for recent race fix and remaining issues
ExternalSources: []
Summary: Evidence-backed whole-package Sessionstream architecture review, recent-change audit, cleanup plan, and intern onboarding guide.
LastUpdated: 2026-05-07T17:02:00-04:00
WhatFor: Use this as the onboarding and cleanup guide for reviewing or improving the Sessionstream package after the observer, replay-store, schema-vet, and websocket race work.
WhenToUse: Read before making public API, websocket hydration, SQLite replay-store, Systemlab, or diagnostics changes in sessionstream.
---


# Whole-package code review and intern implementation guide

## Executive summary

`sessionstream` is a small but dense Go framework for session-scoped event streaming applications. Its core promise is: application handlers publish typed backend events; the framework validates payloads, assigns event ordinals, runs live UI and durable timeline projections, stores replay/hydration state, and fans out live UI events to reconnectable clients. The repository also contains a SQLite replay/hydration store, a protobuf-JSON websocket transport, a chat demo, a schema vet analyzer, embedded documentation, and an educational browser app called Systemlab.

The package is in good shape for an actively extracted framework. The most important improvements from the recent diaries are real and visible in code:

- the misleading map-backed memory store was removed in favor of in-memory SQLite;
- generated protobuf payloads replaced top-level `structpb.Struct` in the chat demo;
- replay primitives now exist in SQLite: backend event log, historical entity versions, projection cursors, durable errors, retry helpers, and scratch rebuild;
- Systemlab Phase 2 and Phase 5 were split out of monolithic files;
- Hub and websocket observers now expose pipeline and transport-level diagnostics;
- the websocket reconnect race was fixed with hydrating subscriptions and fanout buffering.

The highest-priority remaining review findings are not broad rewrites. They are targeted correctness and maintainability issues:

1. **Potential duplicate live events in a narrow websocket hydration window.** `drainHydrationBuffer` filters buffered events to `ordinal > snapshotOrdinal`, but `markLive` returns late batches without the same filter. A delayed fanout for an event already included in the snapshot can be sent again after the snapshot. See `pkg/sessionstream/transport/ws/server.go:329-346` and `pkg/sessionstream/transport/ws/server.go:506-518`.
2. **Websocket fanout errors are swallowed by `PublishUI`.** Per-connection delivery failures close the connection but do not return an error to the Hub, so `ErrorKindFanout` is not emitted for websocket queue/full/hydration overflow failures. See `pkg/sessionstream/transport/ws/server.go:212-229` and `pkg/sessionstream/transport/ws/server.go:453-479`.
3. **SQLite migration strategy can drop durable data.** `migrate` drops all tables when `user_version < 2`. That was acceptable during extraction, but it is dangerous once users treat the store as durable. See `pkg/sessionstream/hydration/sqlite/store.go:581-599`.
4. **SQLite event append silently overwrites ordinal conflicts.** `AppendEvent` uses `ON CONFLICT(session_id, ordinal) DO UPDATE`. This makes replays idempotent, but it also permits a different event with the same session/ordinal to replace the original event. See `pkg/sessionstream/hydration/sqlite/store.go:370-382`.
5. **The largest files are still too large for new contributors.** `pkg/sessionstream/hydration/sqlite/store.go` is 745 lines, `pkg/sessionstream/transport/ws/server.go` is 712 lines, and `pkg/sessionstream/hub.go` is 608 lines. Generated protobuf files are larger but not review targets.
6. **Diagnostics APIs are useful but fragmented.** The package now exposes `BusObserver`, `PipelineObserver`, websocket `Hooks`, and websocket `TransportObserver`. They work, but a new user has to learn four observation concepts with different naming and record styles.

Validation status during this review:

- `go test ./...` passed.
- `go test ./pkg/sessionstream/transport/ws -race -count=1` passed.
- `make lint` passed with 0 issues.
- Coverage after `go clean -cache`: total statement coverage is about 66.9%; notable uncovered or weak areas are noop store paths, view helpers, websocket ping/unsubscribe/error helper branches, schema lookup getters, and some SQLite edge/error paths.

## Problem statement and scope

The user asked for a new docmgr ticket under `sessionstream/ttmp` that reviews the whole package, especially recently added code and improvements, while looking beyond ordinary bugs. The requested review includes:

- code quality issues;
- deprecated or stale code;
- unclear APIs;
- files that are too long;
- code that could be organized better;
- packages that are too large;
- overengineered or underdocumented functionality;
- a clear, technical intern-oriented architecture and implementation guide;
- diary, docmgr storage, validation, and reMarkable delivery.

This document is therefore a hybrid artifact:

1. **Intern guide.** It explains the system, runtime flows, public APIs, and package map.
2. **Code review.** It records concrete findings with file references, examples, impact, and cleanup sketches.
3. **Implementation guide.** It proposes a phased cleanup plan that a new contributor can execute safely.

The scope is the `sessionstream` repository excluding generated code as a primary review target. Generated protobuf files are counted in inventory but not treated as maintainability debt unless they leak into hand-written APIs.

## Evidence collected

Commands and artifacts are stored in this ticket under `sources/` and `scripts/`.

Key evidence files:

- `sources/01-inventory-output.txt` — package list, file inventory, line-count hotspots, marker search, recent git history.
- `sources/02-key-files-lines-1.txt` — line-numbered excerpts of key files.
- `sources/03-issue-snippets.txt` — line-numbered snippets for review findings.
- `sources/04-validation-output.txt` — successful `go test`, websocket race test, and lint output.
- `sources/05-coverage-output.txt` — coverage output after cache cleanup.
- `scripts/01-inventory.sh` — reproducible inventory script.

Recent diaries that shaped this review:

- `ttmp/2026/05/06/SS-OBSERVERS--add-hub-and-websocket-observers-for-sessionstream-diagnostics/reference/01-implementation-diary.md`
- `ttmp/2026/05/06/SS-WS-RACE--fix-websocket-subscribe-snapshot-race-during-streaming-reconnect/reference/01-implementation-diary.md`
- `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md`

## Mental model for interns

Sessionstream separates product behavior from streaming infrastructure. The product code decides what commands and events mean. Sessionstream coordinates the mechanics.

### One-sentence model

A `Command` enters a session; a handler publishes canonical backend `Event` values; projections derive live `UIEvent` values and durable `TimelineEntity` values; stores hydrate reconnecting clients; transports deliver live events.

### Core flow diagram

```mermaid
flowchart TD
    Client[Client or application] -->|Submit command| Hub[sessionstream.Hub]
    Hub --> Registry[SchemaRegistry validates command]
    Hub --> Handler[CommandHandler]
    Handler -->|Publish Event| Publisher[EventPublisher]
    Publisher --> EventStore[optional EventStore append]
    Publisher --> UIProj[UIProjection]
    Publisher --> TLProj[TimelineProjection]
    UIProj --> UIEvents[[]UIEvent]
    TLProj --> Entities[[]TimelineEntity]
    Entities --> Store[HydrationStore / SQLite]
    Store --> Snapshot[Snapshot on reconnect]
    UIEvents --> Fanout[UIFanout]
    Fanout --> WS[WebSocket clients]
    EventStore --> Replay[Retry/RebuildTimeline]
    Replay --> TLProj
```

The most important design rule is that backend events are the source of truth. UI events are delivery artifacts. Timeline entities are durable projected state. A reconnecting browser should not need to ask the original command handler to run again; it should receive a snapshot from the store and then future live UI events.

### Pseudocode for the live path

This is the conceptual shape of `Hub.Submit` through `projectAndApply`:

```go
func Submit(ctx, sid, commandName, commandPayload) error {
    validate commandPayload against SchemaRegistry
    handler := commandRegistry[commandName]
    session := sessions.GetOrCreate(sid)
    publisher := localPublisher or busPublisher
    return handler(ctx, Command{sid, commandName, commandPayload}, session, publisher)
}

func Publish(event) error {
    validate event payload against SchemaRegistry
    event.Ordinal = next ordinal for this session

    if store implements EventStore {
        store.AppendEvent(event)       // canonical replay source
    }

    view := store.View(event.SessionId)
    uiEvents := uiProjection.Project(event, view)
    timelineEntities := timelineProjection.Project(event, view)

    store.Apply(event.SessionId, event.Ordinal, timelineEntities)
    projectionCursor.Advance("timeline", event.Ordinal)

    fanout.PublishUI(event.SessionId, event.Ordinal, uiEvents)
}
```

The actual implementation adds error policies, durable error records, observers, cloning, nil checks, and replay/rebuild paths. Start in `pkg/sessionstream/hub.go` and read `projectAndApply` around `pkg/sessionstream/hub.go:386-470`.

### Pseudocode for websocket reconnect

The reconnect contract is snapshot-before-live. Recent race-fix work made the subscription visible to fanout before snapshot loading starts, while marking it as `hydrating` so live events are buffered until after the snapshot is sent.

```go
func handleSubscribe(conn, sessionID, sinceSnapshotOrdinal) error {
    registerHydrating(conn, sessionID)        // fanout can now target this conn
    snapshot := store.Snapshot(sessionID)     // may take time
    send(snapshotFrame(snapshot))

    buffered := drainHydrationBuffer(conn, sessionID, snapshot.Ordinal)
    send only buffered events with ordinal > snapshot.Ordinal

    lateBuffered := markLive(conn, sessionID, snapshot.Ordinal)
    send late buffered events

    send(subscribedFrame)
}

func PublishUI(sessionID, ordinal, events) {
    for each subscribed connection:
        if subscription is hydrating:
            buffer the batch
        else if subscription is live:
            send UI frames immediately
}
```

The current review recommends applying the same `ordinal > snapshotOrdinal` filter to `lateBuffered` before it is sent.

## Repository and package map

| Path | Role | Review notes |
|---|---|---|
| `pkg/sessionstream` | Core public framework: Hub, commands, events, projections, ordinals, bus, schema, replay interfaces, observers. | Correct abstraction boundary, but `hub.go` is now too dense. |
| `pkg/sessionstream/hydration/sqlite` | SQLite implementation of hydration, event log, entity versions, projection cursors, error records. | Semantically important and too large; migration strategy needs hardening. |
| `pkg/sessionstream/transport/ws` | Protobuf-JSON websocket snapshot/fanout adapter. | Recently improved; still too large and has a narrow late-buffer filtering concern. |
| `pkg/sessionstream/pb/proto/sessionstream/v1` | Generated Go bindings for websocket transport protobuf. | Do not hand-edit; ignore generated `Deprecated` comments. |
| `proto/sessionstream/v1` | Source `.proto` transport schema. | Clear and small. |
| `examples/chatdemo` | Runnable schema-first chat example. | Good teaching value; `chat.go` can be split. |
| `cmd/sessionstream-systemlab` | Browser lab and long-form chapters. | Much improved after file splits; still a large `main` package with global helper namespace. |
| `cmd/sessionstream-lint` and `pkg/analysis/sessionstreamschema` | Schema vet analyzer that rejects top-level `Struct` registration. | Good boundary enforcement. |
| `pkg/doc` | Embedded Glazed help docs. | Useful; should be kept in sync with code changes. |
| `ttmp` | Ticket docs, diaries, design artifacts. | Rich history; not runtime code. |

## Public API reference map

### Core setup API

Start with these APIs when building an app:

```go
reg := sessionstream.NewSchemaRegistry()
reg.RegisterCommand("CommandName", &appv1.CommandPayload{})
reg.RegisterEvent("EventName", &appv1.EventPayload{})
reg.RegisterUIEvent("UIEventName", &appv1.UIEventPayload{})
reg.RegisterTimelineEntity("EntityKind", &appv1.EntityPayload{})

hub, err := sessionstream.NewHub(
    sessionstream.WithSchemaRegistry(reg),
    sessionstream.WithHydrationStore(store),
    sessionstream.WithUIFanout(wsServer),
)
```

Important symbols:

- `NewHub`, `WithSchemaRegistry`, `WithHydrationStore`, `WithUIFanout`, `WithEventBus` — setup.
- `RegisterCommand`, `RegisterUIProjection`, `RegisterTimelineProjection` — app behavior wiring.
- `Submit` — command entrypoint.
- `Snapshot`, `Cursor`, `EventCursor`, `ProjectionCursor` — state/cursor reads.
- `RetryTimeline`, `RebuildTimeline`, `RebuildTimelineFromScratch` — replay/repair helpers.
- `WithPipelineObserver`, `WithErrorObserver` — diagnostics.

### Projection API

```go
type UIProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]UIEvent, error)
}

type TimelineProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]TimelineEntity, error)
}
```

Use UI projections for ephemeral live output and timeline projections for durable snapshot state. Do not mix the two unless there is a concrete reason.

### Store API families

`HydrationStore` is the baseline:

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

SQLite additionally implements:

- `EventStore` — append/query backend events and read event cursor.
- `ProjectionCursorStore` — track timeline projector progress.
- `TimelineResetStore` — clear materialized timeline state for scratch rebuild.
- `ErrorStore` and `ErrorRecordStore` — persist and query framework errors.

### Websocket transport API

The websocket package exposes a `Server` that is both `http.Handler` and `sessionstream.UIFanout`.

```go
server, err := ws.NewServer(snapshotProvider,
    ws.WithTransportObserver(observer),
    ws.WithHydrationBufferLimit(1024),
)
```

Client frames are defined in `proto/sessionstream/v1/transport.proto`:

- `subscribe`
- `unsubscribe`
- `ping`
- `pong`

Server frames include:

- `hello`
- `snapshot`
- `subscribed`
- `unsubscribed`
- `ui_event`
- `error`
- `pong`

`since_snapshot_ordinal` is advisory today. It is echoed and observable, but websocket subscribe does not replay missed UI events.

## Current-state architecture by subsystem

### 1. Hub and event pipeline

`Hub` owns the high-level command/event/projection path. It keeps registries for schemas, commands, sessions, projection functions, fanout, optional Watermill bus config, error observer, pipeline observer, and local ordinal assignment.

Evidence:

- setup and options: `pkg/sessionstream/hub.go:48-151`;
- local publish path: `pkg/sessionstream/hub.go:291-319`;
- replay helpers: `pkg/sessionstream/hub.go:321-384`;
- live projection path: `pkg/sessionstream/hub.go:386-470`;
- error reporting: `pkg/sessionstream/hub.go:548-565`.

Strengths:

- Payload validation happens before command dispatch and event publish.
- The default projection policy is fail-closed, which is safer than advancing through missing timeline state.
- Replay helpers do not publish UI events, preventing duplicate live delivery during repairs.
- Pipeline observers receive cloned payloads and are panic-safe.

Weaknesses:

- `hub.go` combines public options, command dispatch, local publisher, replay, projection, cloning, ordinals, and error reporting in one file.
- Error observer callbacks are not panic-safe, unlike pipeline and transport observers.
- `reportError` ignores persistence failures from `ErrorStore.RecordError`; that is defensible for best-effort telemetry but should be explicit in docs and tests.

### 2. Schema registry

`SchemaRegistry` stores cloned protobuf prototypes by logical name. It validates that submitted commands and published events use the registered concrete message type.

Evidence:

- registration and lookup: `pkg/sessionstream/schema.go:20-117`.

Strengths:

- Registration clones prototypes and lookup returns clones, which prevents callers from mutating registry state.
- The analyzer under `pkg/analysis/sessionstreamschema` reinforces the protobuf-first contract by rejecting top-level `structpb.Struct` registrations.

Weaknesses:

- The registry currently has four parallel maps and nearly identical methods. This is fine now, but if descriptor metadata grows, consider one generic typed category table internally.
- Some exported lookup methods have no direct coverage according to coverage output.

### 3. SQLite hydration/replay store

SQLite is now the single real store implementation. It stores current timeline entities, historical entity versions, backend events, projection cursors, and durable errors.

Evidence:

- constructor and DSNs: `pkg/sessionstream/hydration/sqlite/store.go:22-58`;
- apply/snapshot/cursor: `pkg/sessionstream/hydration/sqlite/store.go:79-305`;
- event log and projection cursor: `pkg/sessionstream/hydration/sqlite/store.go:338-486`;
- errors: `pkg/sessionstream/hydration/sqlite/store.go:488-579`;
- migration: `pkg/sessionstream/hydration/sqlite/store.go:581-671`.

Strengths:

- Removing the map-backed memory store eliminated semantic drift.
- Historical `Snapshot(asOf)` is implemented through `sessionstream_entity_versions`.
- Projection cursors are separate from event cursors, which is the right replay model.
- Error records now persist raw bytes and metadata, which helps diagnose decode failures.

Weaknesses:

- The file is too long and mixes migrations, SQL CRUD, error records, views, and helper conversion functions.
- `migrate` drops tables on old `user_version`, which is unsafe for durable users.
- `AppendEvent` overwrites conflicts instead of detecting event identity mismatches.
- `NewInMemory` uses a fixed shared memory DSN name.

### 4. Watermill bus adapter and consumer

The bus adapter serializes event envelopes to Watermill messages. The consumer decodes messages, assigns ordinals, observes consumption, and feeds events through `projectAndApply`.

Evidence:

- publisher: `pkg/sessionstream/bus.go:115-167`;
- decoder: `pkg/sessionstream/bus.go:176-196`;
- consumer loop: `pkg/sessionstream/consumer.go:38-67`;
- message handling: `pkg/sessionstream/consumer.go:69-91`.

Strengths:

- The bus path preserves the same projection/store/fanout path as local publish.
- Decode errors are now durable error records instead of silent drops.
- Ordinal errors are recorded before the consumer returns an error.

Weaknesses:

- Decode errors are still acked after recording. This is intentionally deferred to GitHub issue #1, but the default policy should be documented near `consumer.go`.
- Ack/Nack return values are not surfaced. This is common with Watermill but leaves no telemetry if the broker rejects an ack/nack.

### 5. Websocket transport

The websocket server is a reference snapshot/fanout adapter. It upgrades HTTP requests, reads protobuf-JSON client frames, sends a hello frame, supports subscribe/unsubscribe/ping, sends snapshots, and fans out live UI events.

Evidence:

- server contract comment: `pkg/sessionstream/transport/ws/server.go:76-96`;
- subscribe flow: `pkg/sessionstream/transport/ws/server.go:296-360`;
- fanout path: `pkg/sessionstream/transport/ws/server.go:212-229`;
- hydrating/live delivery: `pkg/sessionstream/transport/ws/server.go:453-518`;
- frame helpers: `pkg/sessionstream/transport/ws/server.go:568-711`;
- transport observer types: `pkg/sessionstream/transport/ws/observer.go:13-188`.

Strengths:

- The hydrating subscription model closes the original no-target reconnect gap.
- Per-subscription buffering prevents live events from arriving before snapshots.
- Transport observations distinguish queued vs written frames.
- The server clearly rejects command ingress and documents production wrapper needs.

Weaknesses:

- The late-buffer path should apply the same duplicate-prevention filter as the main buffer drain.
- Per-connection fanout delivery errors are not returned to the Hub.
- `server.go` is still a large file with multiple responsibilities.
- There are two diagnostics APIs in the websocket package: old `Hooks` and new `TransportObserver`.

### 6. Chat demo

The chat demo is the best small reference app. It registers generated protobuf payloads, installs command handlers and projections, simulates streaming inference, and exposes a service API.

Evidence:

- schema registration and setup: `examples/chatdemo/chat.go:58-119`;
- command handlers and detached inference goroutine: `examples/chatdemo/chat.go:164-235`;
- UI/timeline projections: `examples/chatdemo/chat.go:249-333`.

Strengths:

- It now demonstrates the schema-first pattern with generated protobuf payloads.
- It covers streaming updates, finalization, stop behavior, and snapshot state.

Weaknesses:

- `chat.go` is 440 lines and mixes schemas, service, engine lifecycle, command handlers, projections, text chunking, and helper cloning.
- The demo intentionally uses detached goroutines and `context.Background()` publishes for streaming chunks. That is acceptable for a demo but deserves a comment near the pattern explaining application-level cancellation expectations.

### 7. Systemlab

Systemlab is a browser app and textbook-style lab. It is not core framework code, but it is important because new contributors use it to understand behavior.

Strengths:

- The UI and phase chapters are separated from the framework.
- Phase 2 and Phase 5 are now split into role-specific files.
- Replay cursor/error inspection makes the new replay model visible.

Weaknesses:

- All Go files are still in package `main`, so helper names can collide across phases. The previous diary recorded a real `cloneStringMap` collision.
- Some phase APIs remain broad and are optimized for teaching rather than library reuse. That is okay, but it should stay quarantined from `pkg/sessionstream`.

## Detailed findings and cleanup sketches

### Finding 1 — Late hydration buffer can replay snapshot-covered UI events

**Severity:** High correctness risk, narrow race window.

**Problem:** The main hydration buffer drain filters out batches whose ordinal is already represented by the snapshot. The later buffer collected between `drainHydrationBuffer` and `markLive` is sent without that filter. If a delayed fanout for an event already included in the loaded snapshot enters that window, the browser can receive the same state twice: once in the snapshot and once as a live UI event.

**Where to look:**

- `pkg/sessionstream/transport/ws/server.go:329-346` — filtered `drainHydrationBuffer` path.
- `pkg/sessionstream/transport/ws/server.go:506-518` — `markLive` returns late batches without filtering.

**Example snippet:**

```go
buffered := s.drainHydrationBuffer(c, sid, snap.SnapshotOrdinal)
for _, batch := range buffered {
    if err := s.sendUIBatch(c, sid, batch.ordinal, batch.events); err != nil {
        return err
    }
}
lateBuffered := s.markLive(c, sid, snap.SnapshotOrdinal)
for _, batch := range lateBuffered {
    if err := s.sendUIBatch(c, sid, batch.ordinal, batch.events); err != nil {
        return err
    }
}
```

**Why it matters:** The central websocket contract is snapshot-before-live without duplicates for snapshot-covered state. The race fix closed the missing-event window, but this late-buffer path can create the opposite problem in rare interleavings.

**Cleanup sketch:**

```go
func filterBufferedAfterSnapshot(batches []bufferedUIBatch, snapshotOrdinal uint64) []bufferedUIBatch {
    out := batches[:0]
    for _, batch := range batches {
        if batch.ordinal > snapshotOrdinal {
            out = append(out, cloneBufferedUIBatch(batch))
        }
    }
    sort by ordinal
    return out
}

buffered := filterBufferedAfterSnapshot(s.drainHydrationBuffer(...), snap.Ordinal)
late := filterBufferedAfterSnapshot(s.markLive(...), snap.Ordinal)
```

Add a deterministic test with a snapshot provider that blocks, a publish that is already in the snapshot, and a second delayed `PublishUI` call inserted after the first drain but before live transition. If that exact interleaving is hard to expose without test hooks, extract the filtering helper and unit-test `drain + markLive` semantics directly.

### Finding 2 — Websocket `PublishUI` hides fanout failures from the Hub

**Severity:** Medium-to-high observability and correctness risk.

**Problem:** `PublishUI` closes failed connections but always returns `nil`. The Hub can only record `ErrorKindFanout` when `fanout.PublishUI` returns an error. Therefore queue-full, closed-connection, or hydration-buffer-overflow failures in the websocket transport are visible to `TransportObserver` but not to Hub-level error observers or durable error records.

**Where to look:**

- `pkg/sessionstream/transport/ws/server.go:212-229` — `PublishUI` ignores delivery errors.
- `pkg/sessionstream/transport/ws/server.go:453-479` — hydration buffer overflow returns an error to `deliverUIEvents`.
- `pkg/sessionstream/hub.go:456-467` — Hub reports fanout errors only when `PublishUI` returns one.

**Example snippet:**

```go
for _, c := range targets {
    if err := s.deliverUIEvents(ctx, c, sid, ord, events); err != nil {
        s.closeConnection(c)
    }
}
return nil
```

**Why it matters:** The new observer system exists to diagnose end-to-end delivery. If Hub-level error records never see websocket fanout failures, application-level diagnostics may show a clean pipeline even when the transport dropped a connection due to full queues.

**Cleanup sketch:**

```go
var errs []error
for _, c := range targets {
    if err := s.deliverUIEvents(ctx, c, sid, ord, events); err != nil {
        errs = append(errs, fmt.Errorf("%s: %w", c.id, err))
        s.closeConnection(c)
    }
}
if len(errs) > 0 {
    return errors.Join(errs...)
}
return nil
```

If returning errors would make one bad websocket client abort command processing too aggressively, add an option:

```go
type FanoutFailurePolicy int
const (
    FanoutFailureCloseAndContinue FanoutFailurePolicy = iota
    FanoutFailureReturnError
)
```

Default can remain close-and-continue, but emit a structured `TransportObserver` record and document that Hub fanout error records only happen in strict mode.

### Finding 3 — SQLite migrations drop durable data on old schema versions

**Severity:** High once any user treats SQLite as durable.

**Problem:** `migrate` drops all store tables when `PRAGMA user_version` is lower than the target. This was useful during fast extraction, but it is incompatible with a durable replay/hydration store.

**Where to look:** `pkg/sessionstream/hydration/sqlite/store.go:581-599`.

**Example snippet:**

```go
if userVersion < targetUserVersion {
    for _, stmt := range []string{
        `DROP TABLE IF EXISTS sessionstream_errors;`,
        `DROP TABLE IF EXISTS sessionstream_projection_cursors;`,
        ...
    } { ... }
}
```

**Why it matters:** The package now advertises event logs, historical snapshots, projection cursors, and error records. Dropping all tables on upgrade would lose exactly the data those APIs are meant to protect.

**Cleanup sketch:**

```go
func (s *Store) migrate() error {
    version := currentUserVersion()
    if version < 1 { create v1 tables }
    if version < 2 { alter/add v2 tables and indexes without dropping }
    if version < 3 { next additive migration }
    set user_version after each successful migration
}
```

Add tests that create a v1 database with sample rows, reopen with the new code, and assert that rows survive.

### Finding 4 — `AppendEvent` silently overwrites session/ordinal conflicts

**Severity:** Medium correctness risk.

**Problem:** `AppendEvent` uses upsert by `(session_id, ordinal)`. This permits a later event with the same session and ordinal but a different name or payload to replace the original.

**Where to look:** `pkg/sessionstream/hydration/sqlite/store.go:370-382`.

**Example snippet:**

```go
INSERT INTO sessionstream_events(session_id, ordinal, name, payload_json)
VALUES(?, ?, ?, ?)
ON CONFLICT(session_id, ordinal) DO UPDATE SET
    name = excluded.name,
    payload_json = excluded.payload_json
```

**Why it matters:** Event logs are supposed to be canonical. Idempotent append is useful, but overwriting different event content at the same ordinal undermines replay correctness.

**Cleanup sketch:**

```go
func AppendEvent(ev Event) error {
    try INSERT
    if conflict {
        existing := SELECT name, payload_json
        if existing equals new serialized event {
            return nil // idempotent retry
        }
        return ErrEventOrdinalConflict{SessionID: ev.SessionId, Ordinal: ev.Ordinal}
    }
}
```

Add tests for exact duplicate append and conflicting duplicate append.

### Finding 5 — `NewInMemory` uses a shared SQLite memory database name

**Severity:** Medium test isolation and application surprise risk.

**Problem:** `NewInMemory` calls `MemoryDSN("")`, and `MemoryDSN` substitutes the fixed name `sessionstream-memory`. With SQLite `mode=memory&cache=shared`, two stores using the same DSN in the same process can share state unintentionally while both connections are alive.

**Where to look:** `pkg/sessionstream/hydration/sqlite/store.go:48-58`.

**Example snippet:**

```go
func MemoryDSN(name string) string {
    if name == "" {
        name = "sessionstream-memory"
    }
    return fmt.Sprintf("file:%s?mode=memory&cache=shared...", name)
}

func NewInMemory(reg *sessionstream.SchemaRegistry) (*Store, error) {
    return New(MemoryDSN(""), reg)
}
```

**Why it matters:** In-memory helpers are often used in tests, examples, and multi-session local labs. Hidden state sharing makes tests order-dependent and makes independent stores less independent than the name suggests.

**Cleanup sketch:**

```go
func NewInMemory(reg *SchemaRegistry) (*Store, error) {
    name := "sessionstream-" + uuid.NewString()
    return New(MemoryDSN(name), reg)
}

func NewSharedInMemory(name string, reg *SchemaRegistry) (*Store, error) {
    if name == "" { return nil, error }
    return New(MemoryDSN(name), reg)
}
```

Keep `MemoryDSN(name)` for explicit shared-cache tests, but make `NewInMemory` isolated by default.

### Finding 6 — Error observers are not panic-safe and persistence failures are invisible

**Severity:** Medium robustness risk.

**Problem:** `PipelineObserver` and `TransportObserver` recover from observer panics. `ErrorObserver` does not. Also, `reportError` ignores `ErrorStore.RecordError` failures.

**Where to look:**

- `pkg/sessionstream/hub.go:548-565` — `reportError`.
- `pkg/sessionstream/pipeline_observer.go:88-94` — panic-safe pipeline observer for comparison.
- `pkg/sessionstream/transport/ws/observer.go:108-114` — panic-safe transport observer for comparison.

**Example snippet:**

```go
func (h *Hub) reportError(ctx context.Context, rec ErrorRecord) {
    if errorStore, ok := h.store.(ErrorStore); ok {
        _ = errorStore.RecordError(ctx, rec)
    }
    if h.errorObserver != nil {
        h.errorObserver.OnSessionstreamError(ctx, rec)
    }
}
```

**Why it matters:** Error observers are diagnostic code and should not be able to crash the main event pipeline. Durable error persistence failures also deserve at least optional visibility, because they mean the system failed to record its own failure.

**Cleanup sketch:**

```go
func (h *Hub) reportError(ctx context.Context, rec ErrorRecord) {
    if errorStore, ok := h.store.(ErrorStore); ok {
        if err := errorStore.RecordError(ctx, cloneErrorRecord(rec)); err != nil {
            // optional: send to observer as ErrorKindStore without recursion
        }
    }
    if h.errorObserver != nil {
        safe := cloneErrorRecord(rec)
        defer recoverObserverPanic()
        h.errorObserver.OnSessionstreamError(ctx, safe)
    }
}
```

Add a test mirroring the pipeline observer panic recovery test.

### Finding 7 — Observer APIs are useful but fragmented

**Severity:** Medium API clarity risk.

**Problem:** Diagnostics were added incrementally, and each layer now has its own shape:

- `BusObserver` with `Published` and `Consumed` methods.
- `PipelineObserver` with one `PipelineRecord`.
- Websocket `Hooks` with callback fields.
- `TransportObserver` with one `TransportRecord`.

**Where to look:**

- `pkg/sessionstream/bus.go:41-68`.
- `pkg/sessionstream/pipeline_observer.go:13-85`.
- `pkg/sessionstream/transport/ws/server.go:29-47`.
- `pkg/sessionstream/transport/ws/observer.go:13-105`.

**Why it matters:** The new diagnostics are powerful, but a new application author may not know which observer to install or whether `Hooks` are deprecated by `TransportObserver`.

**Cleanup sketch:**

Do not collapse everything immediately. First document the intended layering:

```text
BusObserver        Broker envelope and ordinal assignment evidence.
PipelineObserver   Hub event lifecycle evidence: append, project, apply, fanout.
TransportObserver  Websocket connection/frame/queue evidence.
Hooks              Legacy/lab convenience callbacks, narrower than TransportObserver.
```

Then consider marking `Hooks` as lab/convenience API in comments, and add one end-to-end example that installs all three record-style observers.

### Finding 8 — Core `hub.go` is doing too many jobs

**Severity:** Medium maintainability risk.

**Problem:** `hub.go` is 608 lines and contains public options, command registration, run/shutdown, local publishing, replay helpers, live projection, clone helpers, ordinal helpers, and error reporting.

**Where to look:** `pkg/sessionstream/hub.go`.

**Why it matters:** The code is readable now, but future additions will make it harder to navigate. New interns should not have to scroll through replay helpers to find `NewHub` options.

**Cleanup sketch:** split without changing package or public symbols:

```text
pkg/sessionstream/hub.go              // Hub type, NewHub, options, register methods
pkg/sessionstream/hub_submit.go       // Submit, dispatch, local publisher
pkg/sessionstream/hub_pipeline.go     // projectAndApply and projection policies
pkg/sessionstream/hub_replay.go       // RetryTimeline/RebuildTimeline helpers
pkg/sessionstream/hub_errors.go       // ErrorRecord/reportError
pkg/sessionstream/clone.go            // cloneEvent/cloneUIEvents/cloneTimelineEntities
```

Keep the package flat. Do not introduce subpackages unless cycles and public API pressure require it.

### Finding 9 — Websocket `server.go` should be split by responsibility

**Severity:** Medium maintainability risk.

**Problem:** `server.go` is 712 lines and contains HTTP upgrade, connection lifecycle, subscribe protocol, subscription state machine, fanout, frame constructors, protobuf `Any` packing, and clone helpers.

**Where to look:** `pkg/sessionstream/transport/ws/server.go`.

**Cleanup sketch:**

```text
transport/ws/server.go          // Server type, options, ServeHTTP, Connections
transport/ws/protocol.go        // readLoop, handleClientFrame, writeLoop
transport/ws/subscription.go    // registerHydrating, deliverUIEvents, buffer/drain/markLive
transport/ws/fanout.go          // PublishUI and target lookup
transport/ws/frames.go          // frame constructors, packAny, frame maps
transport/ws/clone.go           // cloneSnapshot/cloneUIEvent helpers
```

Before splitting, land Finding 1's filter fix so the state machine behavior is stable.

### Finding 10 — SQLite `store.go` should be split and migrated with tests

**Severity:** Medium maintainability risk.

**Problem:** `store.go` is 745 lines and mixes migrations, public constructors, current state, historical snapshots, events, projection cursors, error records, view implementation, and integer conversion helpers.

**Cleanup sketch:**

```text
hydration/sqlite/store.go        // Store type, New, Close, Reset
hydration/sqlite/migrate.go      // additive migrations and schema constants
hydration/sqlite/entities.go     // Apply, Snapshot, View, Cursor, ClearTimeline
hydration/sqlite/events.go       // AppendEvent, Events, EventCursor
hydration/sqlite/cursors.go      // ProjectionCursor, AdvanceProjectionCursor
hydration/sqlite/errors.go       // RecordError, ErrorRecords
hydration/sqlite/view.go         // read-only TimelineView implementation
hydration/sqlite/convert.go      // uint64/int64 helpers, bool conversion
```

This split should happen after migration behavior is tested, because migration tests will guide where schema constants belong.

### Finding 11 — Chat demo is good but still monolithic

**Severity:** Low-to-medium onboarding clarity issue.

**Problem:** `examples/chatdemo/chat.go` is 440 lines and mixes schema constants, engine state, service wrapper, handlers, async demo inference, projections, payload conversion, and helpers.

**Where to look:** `examples/chatdemo/chat.go`.

**Cleanup sketch:**

```text
examples/chatdemo/schema.go       // names and RegisterSchemas
examples/chatdemo/service.go      // Service, SubmitPrompt, Stop, Snapshot, WaitIdle
examples/chatdemo/engine.go       // Engine, activeRun, lifecycle helpers
examples/chatdemo/handlers.go     // handleStartInference, handleStopInference
examples/chatdemo/projections.go  // uiProjection, timelineProjection, conversion helpers
examples/chatdemo/text.go         // renderAnswer, chunkText
```

Keep it as one package so the example remains easy to copy.

### Finding 12 — README still links to an old `evtstream` ticket path

**Severity:** Low documentation clarity issue.

**Problem:** Runtime code no longer uses the old `evtstream` naming outside generated or historical docs, but `README.md` still links to an old ticket path containing `evtstream` in the title. This is not a bug, but it can confuse a new reader who thinks old names are still active.

**Where to look:** `README.md:324` from `rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'`.

**Cleanup sketch:** Add a note in the “Further reading” section that older tickets may use the historical `evtstream` name, or replace the link with newer `SESSIONSTREAM-003` and this ticket once this review lands.

### Finding 13 — Public footguns need stronger comments

**Severity:** Low-to-medium API clarity issue.

Examples:

- `ProjectionErrorPolicyAdvance` is useful for compatibility and experiments, but advancing through projection failures can hide missing durable state.
- `since_snapshot_ordinal` is advisory in websocket subscribe, not replay semantics.
- `NewInMemory` currently implies isolation but uses shared memory DSN behavior.
- `Hooks.OnUIEventSent` means queued, not written; `TransportObserver` distinguishes queued vs written.

**Cleanup sketch:** Strengthen comments and reference docs before changing APIs. If breaking changes are acceptable later, consider names like `ProjectionErrorPolicyUnsafeAdvance` and `WithSharedInMemoryName`.

### Finding 14 — Coverage gaps align with edge-case risk

**Severity:** Medium test planning issue.

The full coverage run after cache cleanup reported total coverage around 66.9%. Important gaps include:

- `pkg/sessionstream/noop_store.go` methods at 0% except constructor.
- SQLite `View`, `newView`, `Get`, `List`, and `Ordinal` at 0% in the aggregate coverage output.
- Websocket `newUnsubscribedFrame`, `newPongFrame`, `protoMessageAsMap`, and `cloneSnapshot` at 0%.
- Websocket `handleClientFrame` at 60% and `sendFrame` at 59.1%.

These are not all urgent, but they overlap with edge behavior: unsubscribe, ping/pong, frame helper errors, snapshot cloning, and store view reads.

**Cleanup sketch:** Add focused tests:

```text
TestServerPingPong
TestServerUnsubscribeStopsFanout
TestServerHookPayloadCloneIsolation
TestStoreViewGetListCloneIsolation
TestStoreAppendEventConflictMismatch
TestNewInMemoryStoresAreIsolated
TestErrorObserverPanicRecovered
```

## Deprecated, stale, and removed-code review

The search for `Deprecated`, `deprecated`, `legacy`, `obsolete`, `TODO`, `FIXME`, `HACK`, and `XXX` found no active hand-written deprecated runtime paths that need immediate deletion.

Important distinctions:

- The many `Deprecated: Use ... ProtoReflect.Descriptor` comments are generated protobuf getters in `*.pb.go`; ignore them.
- `pkg/sessionstream/doc.go` says product-specific “legacy webchat envelopes” belong in consuming apps. That is boundary documentation, not deprecated code.
- Systemlab and README still mention legacy concepts for teaching/history. That is acceptable if clearly framed.
- The map-backed memory store appears to be removed from active code. The only active mention found is explanatory prose in the Phase 5 chapter saying local mode now uses in-memory SQLite.

The biggest stale-doc issue is the README link to an older `evtstream`-named ticket. It should be contextualized or replaced.

## Package-size and organization assessment

### Current line-count hotspots

From `sources/01-inventory-output.txt`:

```text
745 pkg/sessionstream/hydration/sqlite/store.go
712 pkg/sessionstream/transport/ws/server.go
650 pkg/sessionstream/transport/ws/server_test.go
608 pkg/sessionstream/hub.go
575 pkg/sessionstream/hub_test.go
463 cmd/sessionstream-systemlab/lab_environment.go
440 examples/chatdemo/chat.go
395 cmd/sessionstream-systemlab/phase3_lab.go
```

Generated files are larger but excluded from cleanup recommendations.

### Recommended organization threshold

For this repository, use these rough thresholds:

- Over 500 lines in hand-written framework code: split by responsibility unless the file is generated or table-driven.
- Over 300 lines in examples/labs: split if responsibilities are mixed.
- Over 600 lines in tests: split only if test themes are distinct; long test files are less harmful than long production files.

### Split order

1. Fix correctness issues first: websocket late buffer filter, event append conflict policy, migration safety.
2. Split websocket server after the state machine stabilizes.
3. Split SQLite store after additive migration tests exist.
4. Split Hub into pipeline/replay/error files.
5. Split chat demo if interns keep using it as the first copied example.

## Phased implementation plan

### Phase 0 — Safety tests before behavior changes

Goal: pin down the current risky semantics.

Tasks:

- Add a websocket test for late hydrating buffered batches with ordinals at or below snapshot ordinal.
- Add a SQLite test for conflicting event append at the same session ordinal.
- Add a SQLite migration preservation test using a manually created old-version database.
- Add a test proving two `NewInMemory` stores do not share state; this should fail before the fix if simultaneous stores share a DSN.
- Add an `ErrorObserver` panic recovery test.

Validation:

```bash
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite ./pkg/sessionstream/transport/ws -count=1
go test ./pkg/sessionstream/transport/ws -race -count=1
```

### Phase 1 — Correctness fixes

Goal: fix behavior with minimal public API churn.

Tasks:

- Filter late buffered websocket batches by `ordinal > snapshotOrdinal` before sending.
- Decide websocket fanout failure policy and either return `errors.Join` or document close-and-continue with explicit observer records.
- Make `AppendEvent` idempotent only for identical events; return a typed conflict error otherwise.
- Make `NewInMemory` generate a unique name by default and add an explicit shared-memory constructor.
- Make `ErrorObserver` panic-safe and clone error metadata/raw bytes before delivery.

Validation:

```bash
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
make check
```

### Phase 2 — Durable SQLite migrations

Goal: replace destructive migration with additive schema evolution.

Tasks:

- Introduce `migrateFrom0To1`, `migrateFrom1To2`, and future migration function structure.
- Preserve data when adding new tables or columns.
- Add tests that create old schema versions and reopen them.
- Document that pre-release destructive migration is no longer acceptable.

Validation:

```bash
go test ./pkg/sessionstream/hydration/sqlite -count=1
```

### Phase 3 — File splits with no behavior change

Goal: make the package easier to review.

Tasks:

- Split `transport/ws/server.go` into protocol, subscription, fanout, frames, and clone files.
- Split `hydration/sqlite/store.go` into migration/entity/event/cursor/error/view files.
- Split `pkg/sessionstream/hub.go` into setup, submit, pipeline, replay, errors, and clone files.
- Split `examples/chatdemo/chat.go` if time permits.

Validation:

```bash
go test ./...
make lint
make check
git diff --stat
```

The diff should be mostly moved code plus import cleanup.

### Phase 4 — Diagnostics API consolidation docs

Goal: make observers understandable without removing useful APIs.

Tasks:

- Add a diagnostics reference section explaining Bus vs Pipeline vs Transport observers.
- Mark websocket `Hooks` as convenience/lab callbacks, not the richest telemetry path.
- Add a small example installing all record-style observers and correlating event ordinal/session id across records.
- Decide whether `TransportStage` and `PipelineMode` need stable compatibility guarantees.

### Phase 5 — Coverage and onboarding polish

Goal: improve confidence in edge behavior and intern onboarding.

Tasks:

- Add tests listed in Finding 14.
- Update README old `evtstream` link context.
- Add file-level comments after splits.
- Add a short “how to read this repo in one hour” guide in docs or README.

## Intern reading path

A new intern should read in this order:

1. `README.md` — project purpose and core model.
2. `pkg/doc/tutorials/01-getting-started.md` — first application flow.
3. `examples/chatdemo/chat.go` — concrete generated-protobuf example.
4. `examples/chatdemo/chat_test.go` — executable expected behavior.
5. `pkg/sessionstream/types.go`, `projection.go`, `hydration.go` — core data contracts.
6. `pkg/sessionstream/schema.go` — schema validation.
7. `pkg/sessionstream/hub.go` — live and replay event pipeline.
8. `pkg/sessionstream/hydration/sqlite/store.go` — durable store model.
9. `proto/sessionstream/v1/transport.proto` — websocket wire contract.
10. `pkg/sessionstream/transport/ws/server.go` — reconnect/hydration/fanout behavior.
11. `cmd/sessionstream-systemlab/README.md` and chapters — teaching app behavior.
12. Recent diaries listed in this ticket — why the latest code changed.

## Review checklist for future PRs

Use this checklist for changes to the package:

- Does every new command/event/UI/timeline payload use a concrete protobuf type?
- Does a handler publish canonical backend events instead of directly mutating UI state?
- Does a timeline projection update durable entities with correct `CreatedOrdinal` and `LastEventOrdinal`?
- Does any new replay behavior avoid live UI fanout during rebuild?
- Does websocket subscribe preserve snapshot-before-live ordering?
- Are ordinals treated as `uint64` and kept as strings in JavaScript/protobuf JSON where needed?
- Does any new observer clone retained payloads and recover from panics?
- Does SQLite migration preserve existing data?
- Does a fanout failure have an intentional policy and observable record?
- Are new Systemlab helpers kept out of the core package unless they are general framework APIs?

## Risks, alternatives, and open questions

### Risks

- Returning websocket fanout errors to Hub may change command success semantics for applications with flaky clients. If that is too disruptive, use an explicit policy option.
- Additive migrations require careful tests but are mandatory before external durable use.
- Splitting files can create noisy diffs. Keep those commits mechanical and separate from behavior changes.

### Alternatives considered

- **Collapse all observers into one global event bus.** Rejected for now. The layers observe different things, and immediate collapse would be churn. Documenting the layers is lower risk.
- **Implement websocket replay based on `since_snapshot_ordinal`.** Rejected for now. The package intentionally keeps subscribe as snapshot plus future live events; replay belongs behind explicit APIs.
- **Reintroduce a map-backed memory store for tests.** Rejected. In-memory SQLite keeps test/local semantics close to durable semantics.

### Open questions

- Should websocket `Hooks` eventually be deprecated in favor of `TransportObserver`, or remain as a stable simple callback API?
- Should `ProjectionErrorPolicyAdvance` be renamed in a breaking cleanup to make the risk obvious?
- Should SQLite error record queries support filtering by kind and time range?
- Should `AppendEvent` conflict errors expose existing and attempted event names for operator diagnostics?

## References

Primary code references:

- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/schema.go`
- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/bus.go`
- `pkg/sessionstream/consumer.go`
- `pkg/sessionstream/pipeline_observer.go`
- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/transport/ws/observer.go`
- `proto/sessionstream/v1/transport.proto`
- `examples/chatdemo/chat.go`
- `cmd/sessionstream-systemlab/README.md`

Recent docs and diaries:

- `ttmp/2026/05/06/SS-OBSERVERS--add-hub-and-websocket-observers-for-sessionstream-diagnostics/reference/01-implementation-diary.md`
- `ttmp/2026/05/06/SS-WS-RACE--fix-websocket-subscribe-snapshot-race-during-streaming-reconnect/reference/01-implementation-diary.md`
- `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md`
- `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/01-sessionstream-code-review-and-architecture-audit.md`

Validation artifacts in this ticket:

- `sources/01-inventory-output.txt`
- `sources/03-issue-snippets.txt`
- `sources/04-validation-output.txt`
- `sources/05-coverage-output.txt`
