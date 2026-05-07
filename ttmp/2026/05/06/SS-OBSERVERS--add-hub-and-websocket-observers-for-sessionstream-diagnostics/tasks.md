---
Title: Tasks
Ticket: SS-OBSERVERS
Status: active
Topics:
  - sessionstream
  - observability
  - websocket
  - hydration
  - debugging
DocType: tasks
Intent: operational
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Detailed implementation tasks for Hub pipeline and WebSocket transport observers.
LastUpdated: 2026-05-06T21:40:00-04:00
---

# Tasks

## Phase 0: Confirm API boundaries

- [x] Verify that `sessionstream` will expose generic observer hooks only, with no Pinocchio-specific debug endpoints, tables, or browser concepts.
- [x] Confirm that `pkg/sessionstream/hydration.go` interfaces remain unchanged in this ticket.
- [x] Confirm that existing `transport/ws.Hooks` remain supported and unchanged for existing callers.
- [x] Decide whether new observer types live in existing files (`hub.go`, `server.go`) or new files (`pipeline_observer.go`, `transport/ws/observer.go`).

## Phase 1: Hub pipeline observer API

- [x] Add `PipelineMode` with values `PipelineModeLive` and `PipelineModeRebuild`.
- [x] Add `PipelineRecord` with event identity fields: `Mode`, `SessionId`, `Ordinal`, `EventName`, and cloned `Event`.
- [x] Add append-stage fields: `EventAppended` and `AppendErr`.
- [x] Add session/view-stage fields: `SessionErr`, `ViewOrdinal`, and `ViewErr`.
- [x] Add projection-stage fields: `UIEvents`, `UIProjectionErr`, `TimelineEntities`, and `TimelineProjectionErr`.
- [x] Add hydration-stage fields: `AppliedEntities`, `ApplyErr`, `TimelineCursorAdvanced`, and `CursorErr`.
- [x] Add fanout-stage fields: `FanoutEvents` and `FanoutErr`.
- [x] Add `PipelineObserver` interface with `OnPipeline(ctx context.Context, rec PipelineRecord)`.
- [x] Add `PipelineObserverFunc` adapter.
- [x] Add `PipelineObserverHooks` adapter for callback-style tests and applications.
- [x] Add `pipelineObserver PipelineObserver` field to `Hub`.
- [x] Add `WithPipelineObserver(observer PipelineObserver) HubOption`.

## Phase 2: Hub observer safety and cloning

- [x] Add `cloneEvent(ev Event) Event` that clones protobuf payloads.
- [x] Reuse or add `cloneUIEvents([]UIEvent) []UIEvent` for observer-safe payloads.
- [x] Add `cloneTimelineEntities([]TimelineEntity) []TimelineEntity` that clones entity payloads.
- [x] Add `clonePipelineRecord(PipelineRecord) PipelineRecord`.
- [x] Add `observePipeline(ctx, rec)` on `Hub` with nil checks and panic recovery.
- [x] Ensure observer panic never changes `Submit`, local publish, bus consume, or rebuild behavior.

## Phase 3: Instrument live Hub pipeline

- [x] Add a `defer h.observePipeline(ctx, rec)` at the start of `projectAndApply` after initializing a live `PipelineRecord`.
- [x] Record `AppendErr` and `EventAppended` around `EventStore.AppendEvent`.
- [x] Record `SessionErr` around `h.sessions.GetOrCreate`.
- [x] Record `ViewErr` and `ViewOrdinal` around `h.store.View`.
- [x] Record `UIEvents` and `UIProjectionErr` around `h.uiProjection.Project`.
- [x] Record `TimelineEntities` and `TimelineProjectionErr` around `h.timelineProjection.Project`.
- [x] Preserve existing projection error policy behavior exactly.
- [x] Record `AppliedEntities` and `ApplyErr` around `h.store.Apply`.
- [x] Record `TimelineCursorAdvanced` and `CursorErr` around `AdvanceProjectionCursor`.
- [x] Record `FanoutEvents` and `FanoutErr` around `h.fanout.PublishUI`.
- [x] Preserve existing fanout error handling exactly.

## Phase 4: Instrument rebuild pipeline

- [x] Initialize a `PipelineRecord{Mode: PipelineModeRebuild}` inside `rebuildTimelineEvent`.
- [x] Use `defer h.observePipeline(ctx, rec)` so rebuild errors are observed.
- [x] Record session/view load stages.
- [x] Record timeline projection output and errors.
- [x] Record apply output and errors.
- [x] Record projection cursor advance output and errors.
- [x] Do not record UI projection or fanout fields for rebuild mode.

## Phase 5: WebSocket transport observer API

- [x] Add `TransportStage` enum/string type.
- [x] Add stages for upgrade/connect/disconnect: `upgrade_error`, `connected`, `disconnected`.
- [x] Add stages for client frames: `client_frame_read`, `client_frame_decoded`, `client_frame_decode_error`, `protocol_error`.
- [x] Add stages for subscribe/hydration: `subscribe_received`, `snapshot_load_started`, `snapshot_loaded`, `subscription_registered`.
- [x] Add stages for outgoing frames: `server_frame_marshal_error`, `server_frame_queued`, `server_frame_queue_full`, `server_frame_written`, `server_frame_write_error`.
- [x] Add stages for fanout: `fanout_started`, `fanout_no_targets`, `fanout_completed`.
- [x] Add `FrameDirection` with `client_to_server` and `server_to_client`.
- [x] Add `TransportRecord` with connection/session/frame/ordinal/snapshot/fanout/queue/error fields.
- [x] Add `TransportObserver` interface with `OnTransport(ctx context.Context, rec TransportRecord)`.
- [x] Add `TransportObserverFunc` adapter.
- [x] Add `observer TransportObserver` field to `ws.Server`.
- [x] Add `WithTransportObserver(observer TransportObserver) Option`.

## Phase 6: WebSocket observer helpers

- [x] Add `observe(ctx, rec)` on `ws.Server` with nil checks and panic recovery.
- [x] Add `clientFrameType(*sessionstreamv1.ClientFrame) string`.
- [x] Add `serverFrameType(*sessionstreamv1.ServerFrame) string`.
- [x] Add `TimelineEntitySummary` for snapshot observations.
- [x] Add `summarizeEntities([]sessionstream.TimelineEntity) []TimelineEntitySummary`.
- [x] Add target ID helper for fanout observations.
- [x] Ensure transport observations avoid retaining mutable protobuf message pointers unless explicitly cloned.

## Phase 7: Instrument WebSocket lifecycle and reads

- [x] Observe upgrade errors in `ServeHTTP`.
- [x] Observe successful connection creation after connection ID allocation.
- [x] Observe hello frame queue through `sendFrame` instrumentation.
- [x] Observe disconnect in `closeConnection`.
- [x] Observe raw client frame reads before protobuf JSON decode.
- [x] Observe decode errors with raw byte length and error.
- [x] Observe decoded client frames with frame type.
- [x] Preserve existing `Hooks.OnClientFrame`, `Hooks.OnProtocolError`, and error-frame behavior.

## Phase 8: Instrument WebSocket subscribe/hydration

- [x] Observe `subscribe_received` with connection ID, session ID, and `sinceSnapshotOrdinal`.
- [x] Observe `snapshot_load_started` immediately before calling `SnapshotProvider.Snapshot`.
- [x] Observe `snapshot_loaded` with snapshot ordinal, entity count, and entity summaries.
- [x] Observe snapshot load errors as protocol/error records.
- [x] Observe `subscription_registered` exactly when the connection enters `bySession`.
- [x] Preserve existing `Hooks.OnSnapshotSent` and `Hooks.OnSubscribe` behavior.
- [x] Document that observer stages are ordered evidence for diagnosing the subscribe race.

## Phase 9: Instrument fanout and outgoing writes

- [x] Change `PublishUI(_ context.Context, ...)` to use the passed context for observations.
- [x] Observe `fanout_no_targets` when no connections are subscribed to a session.
- [x] Observe `fanout_started` with target connection IDs before sending events.
- [x] Observe `fanout_completed` after all frames were queued or attempted.
- [x] Instrument `sendFrame` for marshal error, queued, and queue-full stages.
- [x] Include frame type, raw byte count, queue length, and queue capacity in send observations.
- [x] Instrument `writeLoop` for successful writes and write errors.
- [x] Preserve existing `Hooks.OnUIEventSent`, `Hooks.OnSendError`, and `Hooks.OnWriteError` semantics.

## Phase 10: Tests for Hub observer

- [x] Add `TestHubPipelineObserverSuccess`.
- [x] Add `TestHubPipelineObserverAppendError`.
- [x] Add `TestHubPipelineObserverUIProjectionError`.
- [x] Add `TestHubPipelineObserverTimelineProjectionErrorAdvancePolicy`.
- [x] Add `TestHubPipelineObserverRebuild`.
- [x] Add `TestHubPipelineObserverPanicIsRecovered`.
- [x] Assert cloned payloads are safe to retain in observer records.

## Phase 11: Tests for WebSocket observer

- [x] Add `TestTransportObserverSubscribeSequence`.
- [x] Add `TestTransportObserverBadClientFrame`.
- [x] Add `TestTransportObserverFanoutTargets`.
- [x] Add `TestTransportObserverFanoutNoTargets`.
- [x] Add `TestTransportObserverQueueFull` or a lower-level send buffer test if direct queue-full is hard to force.
- [x] Add `TestTransportObserverPanicIsRecovered`.
- [x] Assert existing hook callbacks still fire as before.

## Phase 12: Documentation and validation

- [x] Add package-level documentation or README section explaining the observer APIs.
- [x] Add example wiring snippet for applications.
- [x] Run `go test ./pkg/sessionstream/... -count=1`.
- [x] Run any repository lint/format targets normally used for sessionstream.
- [x] Update changelog with implemented API names and any intentional behavior notes.
