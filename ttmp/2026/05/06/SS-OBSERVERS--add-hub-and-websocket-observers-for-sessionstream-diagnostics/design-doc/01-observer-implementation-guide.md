---
Title: Observer Implementation Guide
Ticket: SS-OBSERVERS
Status: active
Topics:
    - sessionstream
    - observability
    - websocket
    - hydration
    - debugging
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sessionstream/hub.go
      Note: Hub pipeline observer targets projectAndApply and rebuildTimelineEvent.
    - Path: pkg/sessionstream/hydration.go
      Note: Defines snapshot and store seams referenced by observer records.
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: WebSocket transport observer targets subscribe
    - Path: ttmp/2026/pkg/sessionstream/bus.go
    - Path: ttmp/2026/pkg/sessionstream/hub.go
    - Path: ttmp/2026/pkg/sessionstream/hydration.go
    - Path: ttmp/2026/pkg/sessionstream/projection.go
    - Path: ttmp/2026/pkg/sessionstream/transport/ws/server.go
ExternalSources: []
Summary: Design and implementation guide for adding hub pipeline and websocket transport observers to sessionstream.
LastUpdated: 2026-05-06T21:35:00-04:00
WhatFor: Guide a new contributor through implementing generic sessionstream observers that Pinocchio can use for streaming diagnostics.
WhenToUse: Use before changing sessionstream observability, websocket hooks, or debugging APIs that need event/projection/hydration/fanout evidence.
---


# Observer Implementation Guide

## Executive summary

This document explains how to add two generic observer seams to `sessionstream`: a **Hub pipeline observer** and a **WebSocket transport observer**. These observers are designed to support investigations like Pinocchio's `PINO-STREAM-DEBUG` ticket without embedding Pinocchio-specific debug storage, HTTP endpoints, or UI behavior in the reusable `sessionstream` library.

The Hub observer watches the core event-processing pipeline:

```text
backend event
  -> optional event-store append
  -> timeline view load
  -> UI projection
  -> timeline projection
  -> hydration-store apply
  -> projection cursor advance
  -> UI fanout
```

The WebSocket observer watches the client transport path:

```text
HTTP upgrade
  -> hello frame
  -> raw client frame
  -> decoded subscribe
  -> snapshot load
  -> snapshot frame queued/written
  -> subscription registered
  -> live UI frame fanout
  -> disconnect / errors
```

Together, these two observers let an application reconstruct the path from a backend event to the browser-visible frame. The library supplies neutral observations; applications decide whether to log them, store them, expose them over a debug endpoint, or ignore them.

## Why this exists

`sessionstream` is deliberately small. Its job is to route commands, persist events and timeline state, project backend events into UI events, and deliver those UI events to clients. It should not know about Pinocchio, CoinVault, Redux, browser debug overlays, or reMarkable reports. But when a streaming UI misbehaves, the missing information is often inside `sessionstream` itself.

A common debugging question looks like this:

> The browser missed an assistant message chunk after reload. Did the backend publish the event? Did the UI projection produce a frame? Did the timeline projection persist durable state? Did the WebSocket server queue the frame? Did the write loop actually write it?

Without observers, the answer requires piecing together logs from several layers. With observers, the application receives structured records at the exact seams where state changes.

The design goal is **observation without ownership inversion**:

- `sessionstream` owns the generic event/projection/hydration/transport seams.
- Pinocchio owns debug recording, HTTP endpoints, UI panels, and reconciliation tools.
- Tests can use observers as deterministic probes without parsing log text.

## Mental model: three ledgers

A streaming system maintains three different ledgers. Bugs happen when we confuse them.

```text
                 durable backend history
                +------------------------+
                | EventStore             |
                | ordinal -> backend ev  |
                +-----------+------------+
                            |
                            v
              +-------------+--------------+
              | Hub projectAndApply         |
              | UIProjection + TimelineProj |
              +--------+-------------+-----+
                       |             |
                       v             v
           live UI delivery       durable timeline state
        +--------------------+    +----------------------+
        | WebSocket frames   |    | HydrationStore       |
        | per connection     |    | entity snapshots     |
        +--------------------+    +----------------------+
```

The backend event ledger is the canonical description of what happened. The timeline ledger is the durable state used for reconnect and hydration. The WebSocket ledger is the transient live delivery path. A reload bug may affect only the WebSocket ledger; the event and timeline ledgers can remain correct.

The observers in this ticket deliberately cover the two places where the ledgers diverge:

- The **Hub pipeline observer** explains how one backend event became UI events and timeline entities.
- The **WebSocket transport observer** explains how UI events and snapshots moved to each connection.

## Existing code map

### Core Hub files

| File | Why it matters |
|------|----------------|
| `pkg/sessionstream/hub.go` | Defines `Hub`, `HubOption`, command dispatch, event publishing, projection execution, hydration apply, fanout, errors, and rebuild. |
| `pkg/sessionstream/projection.go` | Defines `UIProjection`, `TimelineProjection`, `UIEvent`, `TimelineEntity`, and `TimelineView`. |
| `pkg/sessionstream/hydration.go` | Defines `HydrationStore`, `EventStore`, `ProjectionCursorStore`, `Snapshot`, and optional store capabilities. |
| `pkg/sessionstream/bus.go` | Defines Watermill bus integration and the existing `BusObserver` pattern. |

### WebSocket files

| File | Why it matters |
|------|----------------|
| `pkg/sessionstream/transport/ws/server.go` | Defines `Server`, `Hooks`, `PublishUI`, `ServeHTTP`, `readLoop`, `handleClientFrame`, `sendFrame`, and `writeLoop`. |
| `pkg/sessionstream/transport/ws/server_test.go` | Existing behavioral tests for subscribe, reconnect, and live fanout. New observer tests should live near these. |

## Existing observer patterns

Sessionstream already has two observer styles.

### Error observer

`hub.go` defines:

```go
type ErrorObserver interface {
    OnSessionstreamError(ctx context.Context, rec ErrorRecord)
}

type ErrorObserverFunc func(ctx context.Context, rec ErrorRecord)

func WithErrorObserver(observer ErrorObserver) HubOption
```

This is a best-effort hook for runtime errors. The Hub reports errors through `reportError`, and stores may also persist them if they implement `ErrorStore`.

### Bus observer

`bus.go` defines:

```go
type BusObserver interface {
    Published(ctx context.Context, ev Event, rec BusRecord)
    Consumed(ctx context.Context, ev Event, rec BusRecord)
}

type BusObserverHooks struct {
    OnPublished func(ctx context.Context, ev Event, rec BusRecord)
    OnConsumed  func(ctx context.Context, ev Event, rec BusRecord)
}
```

This is an application-facing observation seam for Watermill publish/consume behavior.

The new observers should follow the same conventions:

- Use neutral names such as `PipelineObserver` and `TransportObserver`, not `DebugObserver`.
- Provide an interface, a `Func` adapter where useful, and a `Hooks` adapter for optional callbacks.
- Invoke observers best-effort. Observer failure must not break the streaming pipeline.
- Clone protobuf payloads before handing them to observers, or clearly document that observers must not retain or mutate values. This guide recommends cloning.

## Observer 1: Hub pipeline observer

### Purpose

The Hub pipeline observer records what happened to one backend event inside `Hub.projectAndApply`. It answers the question:

> For backend event ordinal N, what did each stage produce, store, or fail to do?

### Stages to observe

The Hub pipeline has these stages:

| Stage | Existing code | Observation question |
|-------|---------------|----------------------|
| Event append | `eventStore.AppendEvent(ctx, ev)` | Was the backend event written to the replay log? |
| Session load | `h.sessions.GetOrCreate` | Did the session state exist or get created? |
| View load | `h.store.View` | What timeline ordinal did projections see? |
| UI projection | `h.uiProjection.Project` | Which UI events were produced? |
| Timeline projection | `h.timelineProjection.Project` | Which durable entities were produced? |
| Store apply | `h.store.Apply` | Which entities were applied to hydration state? |
| Cursor advance | `AdvanceProjectionCursor` | Did materialization cursor advance? |
| Fanout | `h.fanout.PublishUI` | Were UI events handed to the transport layer? |

### Proposed types

Add these types to `pkg/sessionstream/hub.go` or a new `pkg/sessionstream/pipeline_observer.go` file.

```go
type PipelineMode string

const (
    PipelineModeLive    PipelineMode = "live"
    PipelineModeRebuild PipelineMode = "rebuild"
)

type PipelineRecord struct {
    Mode PipelineMode

    SessionId SessionId
    Ordinal   uint64
    EventName string
    Event     Event

    EventAppended bool
    AppendErr     error

    SessionErr error

    ViewOrdinal uint64
    ViewErr     error

    UIEvents        []UIEvent
    UIProjectionErr error

    TimelineEntities      []TimelineEntity
    TimelineProjectionErr error

    AppliedEntities []TimelineEntity
    ApplyErr        error

    TimelineCursorAdvanced bool
    CursorErr              error

    FanoutEvents []UIEvent
    FanoutErr    error
}

type PipelineObserver interface {
    OnPipeline(ctx context.Context, rec PipelineRecord)
}

type PipelineObserverFunc func(ctx context.Context, rec PipelineRecord)

func (f PipelineObserverFunc) OnPipeline(ctx context.Context, rec PipelineRecord) {
    if f != nil {
        f(ctx, rec)
    }
}

type PipelineObserverHooks struct {
    OnPipeline func(ctx context.Context, rec PipelineRecord)
}

func (h PipelineObserverHooks) OnPipeline(ctx context.Context, rec PipelineRecord) {
    if h.OnPipeline != nil {
        h.OnPipeline(ctx, rec)
    }
}
```

Add a field to `Hub`:

```go
type Hub struct {
    // existing fields...
    pipelineObserver PipelineObserver
}
```

Add a `HubOption`:

```go
func WithPipelineObserver(observer PipelineObserver) HubOption {
    return func(h *Hub) error {
        h.pipelineObserver = observer
        return nil
    }
}
```

### Invocation strategy

Use `defer`, not a single happy-path call at the end. The observer must see early failures too.

Pseudocode:

```go
func (h *Hub) projectAndApply(ctx context.Context, ev Event) (out []UIEvent, err error) {
    rec := PipelineRecord{
        Mode:      PipelineModeLive,
        SessionId: ev.SessionId,
        Ordinal:   ev.Ordinal,
        EventName: ev.Name,
        Event:     cloneEvent(ev),
    }
    defer func() {
        h.observePipeline(ctx, rec)
    }()

    if eventStore, ok := h.store.(EventStore); ok {
        if err := eventStore.AppendEvent(ctx, ev); err != nil {
            rec.AppendErr = err
            h.reportError(...)
            return nil, err
        }
        rec.EventAppended = true
    }

    sess, err := h.sessions.GetOrCreate(ctx, ev.SessionId)
    if err != nil {
        rec.SessionErr = err
        return nil, err
    }

    view, err := h.store.View(ctx, ev.SessionId)
    if err != nil {
        rec.ViewErr = err
        return nil, err
    }
    rec.ViewOrdinal = view.Ordinal()

    if h.uiProjection != nil {
        uiEvents, uiErr = h.uiProjection.Project(ctx, ev, sess, view)
        rec.UIEvents = cloneUIEvents(uiEvents)
        rec.UIProjectionErr = uiErr
    }

    if h.timelineProjection != nil {
        entities, tlErr = h.timelineProjection.Project(ctx, ev, sess, view)
        rec.TimelineEntities = cloneTimelineEntities(entities)
        rec.TimelineProjectionErr = tlErr
    }

    // Preserve existing error policy behavior.

    entitiesToApply := entities
    if tlErr != nil {
        entitiesToApply = nil
    }
    if err := h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entitiesToApply); err != nil {
        rec.ApplyErr = err
        h.reportError(...)
        return nil, err
    }
    rec.AppliedEntities = cloneTimelineEntities(entitiesToApply)

    if tlErr == nil {
        if projectionStore, ok := h.store.(ProjectionCursorStore); ok {
            if err := projectionStore.AdvanceProjectionCursor(ctx, TimelineProjectorName, ev.SessionId, ev.Ordinal); err != nil {
                rec.CursorErr = err
                h.reportError(...)
                return nil, err
            }
            rec.TimelineCursorAdvanced = true
        }
    }

    if uiErr == nil && h.fanout != nil && len(uiEvents) > 0 {
        fanoutEvents := cloneUIEvents(uiEvents)
        if err := h.fanout.PublishUI(ctx, ev.SessionId, ev.Ordinal, fanoutEvents); err != nil {
            rec.FanoutErr = err
            h.reportError(...)
            return nil, err
        }
        rec.FanoutEvents = cloneUIEvents(fanoutEvents)
    }

    return uiEvents, nil
}
```

### Safe observer helper

Observers are diagnostic tools. They must not break production behavior.

```go
func (h *Hub) observePipeline(ctx context.Context, rec PipelineRecord) {
    if h == nil || h.pipelineObserver == nil {
        return
    }
    safe := clonePipelineRecord(rec)
    defer func() {
        _ = recover()
    }()
    h.pipelineObserver.OnPipeline(ctx, safe)
}
```

This deliberately ignores observer panics. If applications want durable errors from their observers, they should handle those errors inside their observer implementation.

### Rebuild mode

`RebuildTimeline` does not use `projectAndApply`; it calls `rebuildTimelineEvent`. That path should be instrumented too, using `PipelineModeRebuild` and only the stages that apply.

```go
func (h *Hub) rebuildTimelineEvent(ctx context.Context, ev Event) (err error) {
    rec := PipelineRecord{
        Mode:      PipelineModeRebuild,
        SessionId: ev.SessionId,
        Ordinal:   ev.Ordinal,
        EventName: ev.Name,
        Event:     cloneEvent(ev),
    }
    defer func() { h.observePipeline(ctx, rec) }()

    sess, err := h.sessions.GetOrCreate(ctx, ev.SessionId)
    if err != nil { rec.SessionErr = err; return err }

    view, err := h.store.View(ctx, ev.SessionId)
    if err != nil { rec.ViewErr = err; return err }
    rec.ViewOrdinal = view.Ordinal()

    entities, err := h.timelineProjection.Project(ctx, ev, sess, view)
    rec.TimelineEntities = cloneTimelineEntities(entities)
    rec.TimelineProjectionErr = err
    if err != nil { return err }

    err = h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entities)
    rec.AppliedEntities = cloneTimelineEntities(entities)
    rec.ApplyErr = err
    if err != nil { return err }

    // cursor advance as existing code does
    return nil
}
```

The rebuild observation helps diagnose cases where live streaming looked correct but a later timeline rebuild produces different durable entities.

## Observer 2: WebSocket transport observer

### Purpose

The WebSocket observer records what happened to connection lifecycle, client frames, snapshot hydration, server frame queuing, server frame writes, and fanout target selection. It answers the question:

> Did the server receive, understand, queue, write, or drop the frame that the browser expected?

### Why existing `Hooks` are not enough

`transport/ws/server.go` already has hooks:

```go
type Hooks struct {
    OnUpgradeError  func(*http.Request, error)
    OnConnect       func(ConnectionId)
    OnDisconnect    func(ConnectionId)
    OnReadError     func(ConnectionId, error)
    OnWriteError    func(ConnectionId, error)
    OnSendError     func(ConnectionId, error)
    OnProtocolError func(ConnectionId, error)
    OnSubscribe     func(ConnectionId, SessionId, uint64)
    OnUnsubscribe   func(ConnectionId, SessionId)
    OnSnapshotSent  func(ConnectionId, SessionId, Snapshot)
    OnUIEventSent   func(ConnectionId, SessionId, uint64, UIEvent)
    OnClientFrame   func(ConnectionId, map[string]any)
}
```

These are useful, but they lose important distinctions:

- `OnClientFrame` only fires after successful protobuf JSON decode; malformed raw frames are invisible except as errors.
- `OnSnapshotSent` means the snapshot frame was queued through `sendFrame`, not necessarily written to the socket.
- `OnUIEventSent` means a UI event frame was queued, not necessarily written.
- There is no structured record for queue pressure, queue full, marshal failures, fanout target lists, or subscription registration order.

The observer should preserve the existing hooks for compatibility but add a record-style API for richer diagnostics.

### Proposed types

Add to `pkg/sessionstream/transport/ws/server.go` or a new `observer.go` in the same package.

```go
type TransportStage string

const (
    TransportStageUpgradeError           TransportStage = "upgrade_error"
    TransportStageConnected              TransportStage = "connected"
    TransportStageDisconnected           TransportStage = "disconnected"
    TransportStageClientFrameRead        TransportStage = "client_frame_read"
    TransportStageClientFrameDecoded     TransportStage = "client_frame_decoded"
    TransportStageClientFrameDecodeError TransportStage = "client_frame_decode_error"
    TransportStageProtocolError          TransportStage = "protocol_error"
    TransportStageSubscribeReceived      TransportStage = "subscribe_received"
    TransportStageSnapshotLoadStarted    TransportStage = "snapshot_load_started"
    TransportStageSnapshotLoaded         TransportStage = "snapshot_loaded"
    TransportStageSubscriptionRegistered TransportStage = "subscription_registered"
    TransportStageServerFrameMarshalError TransportStage = "server_frame_marshal_error"
    TransportStageServerFrameQueued      TransportStage = "server_frame_queued"
    TransportStageServerFrameQueueFull   TransportStage = "server_frame_queue_full"
    TransportStageServerFrameWritten     TransportStage = "server_frame_written"
    TransportStageServerFrameWriteError  TransportStage = "server_frame_write_error"
    TransportStageFanoutStarted          TransportStage = "fanout_started"
    TransportStageFanoutNoTargets        TransportStage = "fanout_no_targets"
    TransportStageFanoutCompleted        TransportStage = "fanout_completed"
)

type FrameDirection string

const (
    FrameDirectionClientToServer FrameDirection = "client_to_server"
    FrameDirectionServerToClient FrameDirection = "server_to_client"
)

type TransportRecord struct {
    Stage     TransportStage
    Direction FrameDirection

    ConnectionId sessionstream.ConnectionId
    SessionId    sessionstream.SessionId

    FrameType string // hello, snapshot, subscribed, uiEvent, error, subscribe, ping, pong
    Ordinal   uint64 // UI event ordinal or snapshot ordinal when applicable
    EventName string // UI event name when applicable
    PayloadType string

    SinceSnapshotOrdinal uint64
    SnapshotOrdinal      uint64
    SnapshotEntityCount  int
    SnapshotEntities     []TimelineEntitySummary

    FanoutEventCount int
    FanoutTargetIds  []sessionstream.ConnectionId

    RawBytes int
    QueueLen int
    QueueCap int

    Err error
}

type TransportObserver interface {
    OnTransport(ctx context.Context, rec TransportRecord)
}

type TransportObserverFunc func(ctx context.Context, rec TransportRecord)

func (f TransportObserverFunc) OnTransport(ctx context.Context, rec TransportRecord) {
    if f != nil {
        f(ctx, rec)
    }
}
```

Add to server options:

```go
type Server struct {
    // existing fields
    observer TransportObserver
}

func WithTransportObserver(observer TransportObserver) Option {
    return func(s *Server) error {
        s.observer = observer
        return nil
    }
}
```

Add a safe helper:

```go
func (s *Server) observe(ctx context.Context, rec TransportRecord) {
    if s == nil || s.observer == nil {
        return
    }
    safe := cloneTransportRecord(rec)
    defer func() { _ = recover() }()
    s.observer.OnTransport(ctx, safe)
}
```

### Where to fire records

#### HTTP upgrade and hello

In `ServeHTTP`, observe connection creation and hello frame queue.

```go
conn, err := s.upgrader.Upgrade(w, r, nil)
if err != nil {
    s.observe(r.Context(), TransportRecord{Stage: TransportStageUpgradeError, Err: err})
    return
}

s.observe(r.Context(), TransportRecord{Stage: TransportStageConnected, ConnectionId: cid})
_ = s.sendFrame(c, newHelloFrame(cid))
```

#### Raw read and decode

In `readLoop`, observe before and after decode.

```go
_, raw, err := c.ws.ReadMessage()
if err != nil {
    s.observe(ctx, TransportRecord{Stage: TransportStageServerFrameReadError, ConnectionId: c.id, Err: err})
    return
}
s.observe(ctx, TransportRecord{
    Stage: TransportStageClientFrameRead,
    Direction: FrameDirectionClientToServer,
    ConnectionId: c.id,
    RawBytes: len(raw),
})

frame := &sessionstreamv1.ClientFrame{}
if err := unmarshalOpts.Unmarshal(raw, frame); err != nil {
    s.observe(ctx, TransportRecord{
        Stage: TransportStageClientFrameDecodeError,
        Direction: FrameDirectionClientToServer,
        ConnectionId: c.id,
        RawBytes: len(raw),
        Err: err,
    })
    // existing error frame behavior
    continue
}

s.observe(ctx, TransportRecord{
    Stage: TransportStageClientFrameDecoded,
    Direction: FrameDirectionClientToServer,
    ConnectionId: c.id,
    FrameType: clientFrameType(frame),
})
```

#### Subscribe and snapshot

In `handleClientFrame`, observe the hydration path.

```go
case *sessionstreamv1.ClientFrame_Subscribe:
    sid := sessionstream.SessionId(sub.GetSessionId())
    since := sub.GetSinceSnapshotOrdinal()

    s.observe(ctx, TransportRecord{
        Stage: TransportStageSubscribeReceived,
        ConnectionId: c.id,
        SessionId: sid,
        FrameType: "subscribe",
        SinceSnapshotOrdinal: since,
    })

    s.observe(ctx, TransportRecord{
        Stage: TransportStageSnapshotLoadStarted,
        ConnectionId: c.id,
        SessionId: sid,
        SinceSnapshotOrdinal: since,
    })

    snap, err := s.snapshots.Snapshot(ctx, sid)
    if err != nil {
        s.observe(ctx, TransportRecord{
            Stage: TransportStageProtocolError,
            ConnectionId: c.id,
            SessionId: sid,
            SinceSnapshotOrdinal: since,
            Err: err,
        })
        return fmt.Errorf("load snapshot for %q: %w", sid, err)
    }

    s.observe(ctx, TransportRecord{
        Stage: TransportStageSnapshotLoaded,
        ConnectionId: c.id,
        SessionId: sid,
        SinceSnapshotOrdinal: since,
        SnapshotOrdinal: snap.SnapshotOrdinal,
        SnapshotEntityCount: len(snap.Entities),
        SnapshotEntities: summarizeEntities(snap.Entities),
    })
```

Observe subscription registration separately. This is crucial because the race ticket depends on knowing whether live fanout happened before or after registration.

```go
c.mu.Lock()
c.subs[sid] = subscription{sinceSnapshotOrdinal: since}
c.mu.Unlock()
// update s.bySession
s.observe(ctx, TransportRecord{
    Stage: TransportStageSubscriptionRegistered,
    ConnectionId: c.id,
    SessionId: sid,
    SinceSnapshotOrdinal: since,
    SnapshotOrdinal: snap.SnapshotOrdinal,
})
```

#### Fanout target selection

At the start of `PublishUI`:

```go
targets := s.connectionsForSession(sid)
targetIds := idsForConnections(targets)
if len(targets) == 0 {
    s.observe(context.Background(), TransportRecord{
        Stage: TransportStageFanoutNoTargets,
        SessionId: sid,
        Ordinal: ord,
        FanoutEventCount: len(events),
    })
    return nil
}
s.observe(context.Background(), TransportRecord{
    Stage: TransportStageFanoutStarted,
    SessionId: sid,
    Ordinal: ord,
    FanoutEventCount: len(events),
    FanoutTargetIds: targetIds,
})
```

`PublishUI` currently receives `_ context.Context`; change it to `ctx context.Context` so observations carry the upstream context.

#### Send queue and actual writes

`sendFrame` currently marshals the frame and queues bytes into `c.send`. Observe both success and queue-full failure.

```go
func (s *Server) sendFrame(c *connection, frame *sessionstreamv1.ServerFrame) (err error) {
    frameType := serverFrameType(frame)
    body, err := marshalOptions.Marshal(frame)
    if err != nil {
        s.observe(context.Background(), TransportRecord{
            Stage: TransportStageServerFrameMarshalError,
            Direction: FrameDirectionServerToClient,
            ConnectionId: c.id,
            FrameType: frameType,
            Err: err,
        })
        return err
    }

    qLen := len(c.send)
    qCap := cap(c.send)
    select {
    case c.send <- body:
        s.observe(context.Background(), TransportRecord{
            Stage: TransportStageServerFrameQueued,
            Direction: FrameDirectionServerToClient,
            ConnectionId: c.id,
            FrameType: frameType,
            RawBytes: len(body),
            QueueLen: qLen,
            QueueCap: qCap,
        })
        return nil
    default:
        err := fmt.Errorf("connection %s send buffer full", c.id)
        s.observe(context.Background(), TransportRecord{
            Stage: TransportStageServerFrameQueueFull,
            Direction: FrameDirectionServerToClient,
            ConnectionId: c.id,
            FrameType: frameType,
            RawBytes: len(body),
            QueueLen: qLen,
            QueueCap: qCap,
            Err: err,
        })
        return err
    }
}
```

In `writeLoop`, observe actual writes:

```go
for msg := range c.send {
    if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
        s.observe(context.Background(), TransportRecord{
            Stage: TransportStageServerFrameWriteError,
            Direction: FrameDirectionServerToClient,
            ConnectionId: c.id,
            RawBytes: len(msg),
            Err: err,
        })
        return
    }
    s.observe(context.Background(), TransportRecord{
        Stage: TransportStageServerFrameWritten,
        Direction: FrameDirectionServerToClient,
        ConnectionId: c.id,
        RawBytes: len(msg),
    })
}
```

This distinction matters: a frame can be queued and then fail on write later.

## Helper functions

### Frame type helpers

Add helpers that do not require full serialization.

```go
func clientFrameType(frame *sessionstreamv1.ClientFrame) string {
    switch frame.GetFrame().(type) {
    case *sessionstreamv1.ClientFrame_Subscribe:
        return "subscribe"
    case *sessionstreamv1.ClientFrame_Unsubscribe:
        return "unsubscribe"
    case *sessionstreamv1.ClientFrame_Ping:
        return "ping"
    case *sessionstreamv1.ClientFrame_Pong:
        return "pong"
    default:
        return "unknown"
    }
}

func serverFrameType(frame *sessionstreamv1.ServerFrame) string {
    switch frame.GetFrame().(type) {
    case *sessionstreamv1.ServerFrame_Hello:
        return "hello"
    case *sessionstreamv1.ServerFrame_Snapshot:
        return "snapshot"
    case *sessionstreamv1.ServerFrame_Subscribed:
        return "subscribed"
    case *sessionstreamv1.ServerFrame_Unsubscribed:
        return "unsubscribed"
    case *sessionstreamv1.ServerFrame_UiEvent:
        return "uiEvent"
    case *sessionstreamv1.ServerFrame_Error:
        return "error"
    case *sessionstreamv1.ServerFrame_Pong:
        return "pong"
    default:
        return "unknown"
    }
}
```

### Entity summaries

Observers should not force applications to retain large payloads when a summary is enough.

```go
type TimelineEntitySummary struct {
    Kind             string
    Id               string
    CreatedOrdinal   uint64
    LastEventOrdinal uint64
    PayloadType       string
    Tombstone         bool
}

func summarizeEntities(in []sessionstream.TimelineEntity) []TimelineEntitySummary {
    out := make([]TimelineEntitySummary, 0, len(in))
    for _, ent := range in {
        payloadType := ""
        if ent.Payload != nil {
            payloadType = string(ent.Payload.ProtoReflect().Descriptor().FullName())
        }
        out = append(out, TimelineEntitySummary{
            Kind: ent.Kind,
            Id: ent.Id,
            CreatedOrdinal: ent.CreatedOrdinal,
            LastEventOrdinal: ent.LastEventOrdinal,
            PayloadType: payloadType,
            Tombstone: ent.Tombstone,
        })
    }
    return out
}
```

The Hub observer may pass full cloned entities, while the transport observer can pass summaries for snapshots. This keeps the WebSocket observer cheaper.

## Integration from Pinocchio

Pinocchio should wire observers from `cmd/web-chat/app/server.go`, not from `pkg/chatapp`.

```go
recorder := NewDebugRecorder(DebugRecorderOptions{MaxRecords: 10000})

ws, err := wstransport.NewServer(snapshotProvider,
    wstransport.WithTransportObserver(wstransport.TransportObserverFunc(func(ctx context.Context, rec wstransport.TransportRecord) {
        recorder.RecordTransport(ctx, rec)
    })),
)

hub, err := sessionstream.NewHub(
    sessionstream.WithSchemaRegistry(reg),
    sessionstream.WithHydrationStore(store),
    sessionstream.WithUIFanout(ws),
    sessionstream.WithPipelineObserver(sessionstream.PipelineObserverFunc(func(ctx context.Context, rec sessionstream.PipelineRecord) {
        recorder.RecordPipeline(ctx, rec)
    })),
)
```

The recorder may expose app-specific endpoints such as:

```text
GET /api/debug/sessions/{id}/stream-records
GET /api/debug/sessions/{id}/pipeline
GET /api/debug/sessions/{id}/transport
GET /api/debug/sessions/{id}/reconcile
```

Those endpoints do not belong in `sessionstream`; they belong in the application.

## Implementation plan

### Phase 1: Hub observer types and plumbing

1. Add `PipelineMode`, `PipelineRecord`, `PipelineObserver`, `PipelineObserverFunc`, and `PipelineObserverHooks`.
2. Add `pipelineObserver PipelineObserver` to `Hub`.
3. Add `WithPipelineObserver`.
4. Add clone helpers for events, UI events, timeline entities, and records.
5. Add `observePipeline` with panic recovery.
6. Instrument `projectAndApply` with a `defer` record.
7. Instrument `rebuildTimelineEvent` with `PipelineModeRebuild`.
8. Add tests that prove observers receive records on success and early failure.

### Phase 2: WebSocket transport observer types and plumbing

1. Add `TransportStage`, `FrameDirection`, `TransportRecord`, `TransportObserver`, and `TransportObserverFunc`.
2. Add `observer TransportObserver` to `ws.Server`.
3. Add `WithTransportObserver`.
4. Add `observe` with panic recovery.
5. Add frame type helpers and entity summary helpers.
6. Instrument connect/disconnect, raw read, decode success/failure, subscribe, snapshot load, subscription registration, fanout target selection, send queue, and write loop.
7. Add tests that prove records are emitted in the expected sequence.

### Phase 3: Backward compatibility and tests

1. Keep existing `Hooks` behavior unchanged.
2. Avoid changing public interfaces such as `HydrationStore`.
3. Run `go test ./pkg/sessionstream/... ./pkg/sessionstream/transport/ws/... -count=1`.
4. Add targeted race/regression tests in `transport/ws/server_test.go`.
5. Document the observers in package comments or README sections.

## Tests to write

### Hub observer tests

| Test | What it proves |
|------|----------------|
| `TestHubPipelineObserverSuccess` | Command event produces one observer record with UI events, timeline entities, applied entities, cursor advance, and fanout events. |
| `TestHubPipelineObserverAppendError` | Early append failure still emits a record with `AppendErr`. |
| `TestHubPipelineObserverUIProjectionError` | UI projection error is captured and respects projection error policy. |
| `TestHubPipelineObserverTimelineProjectionErrorAdvancePolicy` | Timeline projection errors are captured and `AppliedEntities` is nil under advance policy. |
| `TestHubPipelineObserverRebuild` | `RebuildTimeline` emits `Mode=rebuild` records. |
| `TestHubPipelineObserverPanicIsRecovered` | Observer panic does not break command submission. |

### WebSocket observer tests

| Test | What it proves |
|------|----------------|
| `TestTransportObserverSubscribeSequence` | Connect, hello queued/written, client frame read/decoded, snapshot loaded, snapshot queued, subscription registered, subscribed queued. |
| `TestTransportObserverBadClientFrame` | Raw frame and decode error are observed for malformed protobuf JSON. |
| `TestTransportObserverFanoutTargets` | Live fanout records target connection IDs. |
| `TestTransportObserverFanoutNoTargets` | Fanout with no subscribers emits `fanout_no_targets`. |
| `TestTransportObserverQueueFull` | Send queue pressure emits `server_frame_queue_full`. |
| `TestTransportObserverPanicIsRecovered` | Observer panic does not break websocket handling. |

## Design decisions

### Keep observers generic

The observer records should not mention Pinocchio, CoinVault, Redux, browser localStorage, or specific protobuf message names. They should expose enough generic information for applications to build their own debugging tools.

### Do not add debug persistence to sessionstream

A `debug_projection_trace` table is useful for Pinocchio, but it should live in Pinocchio if needed. `sessionstream` should offer hooks; applications choose storage.

### Do not change `HydrationStore` for this ticket

Changing `HydrationStore` would force every store implementation to update. The two-observer plan avoids that. If we later want store-operation traces, add a decorator instead of changing the interface.

### Preserve existing `Hooks`

The WebSocket `Hooks` API already exists. Do not remove or rename it. The new observer is additive and structured.

### Prefer record-style observers over many tiny callbacks

Many tiny callbacks are easy to add but hard to correlate. A record-style observer gives applications one structured envelope that can be filtered, stored, and serialized.

## How to review the implementation

Start with these files:

1. `pkg/sessionstream/hub.go` or `pkg/sessionstream/pipeline_observer.go` for Hub observer types and `WithPipelineObserver`.
2. `pkg/sessionstream/hub.go` for `projectAndApply` and `rebuildTimelineEvent` instrumentation.
3. `pkg/sessionstream/transport/ws/server.go` or `observer.go` for transport observer types and `WithTransportObserver`.
4. `pkg/sessionstream/transport/ws/server.go` for subscribe, fanout, send, and write-loop observations.
5. Tests in `pkg/sessionstream/*_test.go` and `pkg/sessionstream/transport/ws/server_test.go`.

Validation command:

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/sessionstream
go test ./pkg/sessionstream/... -count=1
```

## API reference

### Hub observer API

```go
func WithPipelineObserver(observer PipelineObserver) HubOption

type PipelineObserver interface {
    OnPipeline(ctx context.Context, rec PipelineRecord)
}
```

### WebSocket observer API

```go
func WithTransportObserver(observer TransportObserver) Option

type TransportObserver interface {
    OnTransport(ctx context.Context, rec TransportRecord)
}
```

### Application wiring sketch

```go
hub, err := sessionstream.NewHub(
    sessionstream.WithHydrationStore(store),
    sessionstream.WithUIFanout(wsServer),
    sessionstream.WithPipelineObserver(myRecorder),
)

wsServer, err := ws.NewServer(snapshotProvider,
    ws.WithTransportObserver(myRecorder),
)
```

## Closing guidance for the intern

When implementing observers, resist the temptation to add logging strings first. Logging is a consumer of observations, not the observation mechanism itself. The library should emit structured records at the right seams. Once those records exist, applications can turn them into logs, tables, JSON downloads, browser overlays, metrics, or tests.

The hard part is not defining an interface. The hard part is preserving the exact runtime behavior of the existing pipeline while observing it. Keep existing error handling and ordering intact. Add tests that prove the observer sees what happened, including failures, and that observer panics do not change application behavior.
