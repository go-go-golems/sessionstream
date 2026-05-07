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

- [ ] Verify that `sessionstream` will expose generic observer hooks only, with no Pinocchio-specific debug endpoints, tables, or browser concepts.
- [ ] Confirm that `pkg/sessionstream/hydration.go` interfaces remain unchanged in this ticket.
- [ ] Confirm that existing `transport/ws.Hooks` remain supported and unchanged for existing callers.
- [ ] Decide whether new observer types live in existing files (`hub.go`, `server.go`) or new files (`pipeline_observer.go`, `transport/ws/observer.go`).

## Phase 1: Hub pipeline observer API

- [ ] Add `PipelineMode` with values `PipelineModeLive` and `PipelineModeRebuild`.
- [ ] Add `PipelineRecord` with event identity fields: `Mode`, `SessionId`, `Ordinal`, `EventName`, and cloned `Event`.
- [ ] Add append-stage fields: `EventAppended` and `AppendErr`.
- [ ] Add session/view-stage fields: `SessionErr`, `ViewOrdinal`, and `ViewErr`.
- [ ] Add projection-stage fields: `UIEvents`, `UIProjectionErr`, `TimelineEntities`, and `TimelineProjectionErr`.
- [ ] Add hydration-stage fields: `AppliedEntities`, `ApplyErr`, `TimelineCursorAdvanced`, and `CursorErr`.
- [ ] Add fanout-stage fields: `FanoutEvents` and `FanoutErr`.
- [ ] Add `PipelineObserver` interface with `OnPipeline(ctx context.Context, rec PipelineRecord)`.
- [ ] Add `PipelineObserverFunc` adapter.
- [ ] Add `PipelineObserverHooks` adapter for callback-style tests and applications.
- [ ] Add `pipelineObserver PipelineObserver` field to `Hub`.
- [ ] Add `WithPipelineObserver(observer PipelineObserver) HubOption`.

## Phase 2: Hub observer safety and cloning

- [ ] Add `cloneEvent(ev Event) Event` that clones protobuf payloads.
- [ ] Reuse or add `cloneUIEvents([]UIEvent) []UIEvent` for observer-safe payloads.
- [ ] Add `cloneTimelineEntities([]TimelineEntity) []TimelineEntity` that clones entity payloads.
- [ ] Add `clonePipelineRecord(PipelineRecord) PipelineRecord`.
- [ ] Add `observePipeline(ctx, rec)` on `Hub` with nil checks and panic recovery.
- [ ] Ensure observer panic never changes `Submit`, local publish, bus consume, or rebuild behavior.

## Phase 3: Instrument live Hub pipeline

- [ ] Add a `defer h.observePipeline(ctx, rec)` at the start of `projectAndApply` after initializing a live `PipelineRecord`.
- [ ] Record `AppendErr` and `EventAppended` around `EventStore.AppendEvent`.
- [ ] Record `SessionErr` around `h.sessions.GetOrCreate`.
- [ ] Record `ViewErr` and `ViewOrdinal` around `h.store.View`.
- [ ] Record `UIEvents` and `UIProjectionErr` around `h.uiProjection.Project`.
- [ ] Record `TimelineEntities` and `TimelineProjectionErr` around `h.timelineProjection.Project`.
- [ ] Preserve existing projection error policy behavior exactly.
- [ ] Record `AppliedEntities` and `ApplyErr` around `h.store.Apply`.
- [ ] Record `TimelineCursorAdvanced` and `CursorErr` around `AdvanceProjectionCursor`.
- [ ] Record `FanoutEvents` and `FanoutErr` around `h.fanout.PublishUI`.
- [ ] Preserve existing fanout error handling exactly.

## Phase 4: Instrument rebuild pipeline

- [ ] Initialize a `PipelineRecord{Mode: PipelineModeRebuild}` inside `rebuildTimelineEvent`.
- [ ] Use `defer h.observePipeline(ctx, rec)` so rebuild errors are observed.
- [ ] Record session/view load stages.
- [ ] Record timeline projection output and errors.
- [ ] Record apply output and errors.
- [ ] Record projection cursor advance output and errors.
- [ ] Do not record UI projection or fanout fields for rebuild mode.

## Phase 5: WebSocket transport observer API

- [ ] Add `TransportStage` enum/string type.
- [ ] Add stages for upgrade/connect/disconnect: `upgrade_error`, `connected`, `disconnected`.
- [ ] Add stages for client frames: `client_frame_read`, `client_frame_decoded`, `client_frame_decode_error`, `protocol_error`.
- [ ] Add stages for subscribe/hydration: `subscribe_received`, `snapshot_load_started`, `snapshot_loaded`, `subscription_registered`.
- [ ] Add stages for outgoing frames: `server_frame_marshal_error`, `server_frame_queued`, `server_frame_queue_full`, `server_frame_written`, `server_frame_write_error`.
- [ ] Add stages for fanout: `fanout_started`, `fanout_no_targets`, `fanout_completed`.
- [ ] Add `FrameDirection` with `client_to_server` and `server_to_client`.
- [ ] Add `TransportRecord` with connection/session/frame/ordinal/snapshot/fanout/queue/error fields.
- [ ] Add `TransportObserver` interface with `OnTransport(ctx context.Context, rec TransportRecord)`.
- [ ] Add `TransportObserverFunc` adapter.
- [ ] Add `observer TransportObserver` field to `ws.Server`.
- [ ] Add `WithTransportObserver(observer TransportObserver) Option`.

## Phase 6: WebSocket observer helpers

- [ ] Add `observe(ctx, rec)` on `ws.Server` with nil checks and panic recovery.
- [ ] Add `clientFrameType(*sessionstreamv1.ClientFrame) string`.
- [ ] Add `serverFrameType(*sessionstreamv1.ServerFrame) string`.
- [ ] Add `TimelineEntitySummary` for snapshot observations.
- [ ] Add `summarizeEntities([]sessionstream.TimelineEntity) []TimelineEntitySummary`.
- [ ] Add target ID helper for fanout observations.
- [ ] Ensure transport observations avoid retaining mutable protobuf message pointers unless explicitly cloned.

## Phase 7: Instrument WebSocket lifecycle and reads

- [ ] Observe upgrade errors in `ServeHTTP`.
- [ ] Observe successful connection creation after connection ID allocation.
- [ ] Observe hello frame queue through `sendFrame` instrumentation.
- [ ] Observe disconnect in `closeConnection`.
- [ ] Observe raw client frame reads before protobuf JSON decode.
- [ ] Observe decode errors with raw byte length and error.
- [ ] Observe decoded client frames with frame type.
- [ ] Preserve existing `Hooks.OnClientFrame`, `Hooks.OnProtocolError`, and error-frame behavior.

## Phase 8: Instrument WebSocket subscribe/hydration

- [ ] Observe `subscribe_received` with connection ID, session ID, and `sinceSnapshotOrdinal`.
- [ ] Observe `snapshot_load_started` immediately before calling `SnapshotProvider.Snapshot`.
- [ ] Observe `snapshot_loaded` with snapshot ordinal, entity count, and entity summaries.
- [ ] Observe snapshot load errors as protocol/error records.
- [ ] Observe `subscription_registered` exactly when the connection enters `bySession`.
- [ ] Preserve existing `Hooks.OnSnapshotSent` and `Hooks.OnSubscribe` behavior.
- [ ] Document that observer stages are ordered evidence for diagnosing the subscribe race.

## Phase 9: Instrument fanout and outgoing writes

- [ ] Change `PublishUI(_ context.Context, ...)` to use the passed context for observations.
- [ ] Observe `fanout_no_targets` when no connections are subscribed to a session.
- [ ] Observe `fanout_started` with target connection IDs before sending events.
- [ ] Observe `fanout_completed` after all frames were queued or attempted.
- [ ] Instrument `sendFrame` for marshal error, queued, and queue-full stages.
- [ ] Include frame type, raw byte count, queue length, and queue capacity in send observations.
- [ ] Instrument `writeLoop` for successful writes and write errors.
- [ ] Preserve existing `Hooks.OnUIEventSent`, `Hooks.OnSendError`, and `Hooks.OnWriteError` semantics.

## Phase 10: Tests for Hub observer

- [ ] Add `TestHubPipelineObserverSuccess`.
- [ ] Add `TestHubPipelineObserverAppendError`.
- [ ] Add `TestHubPipelineObserverUIProjectionError`.
- [ ] Add `TestHubPipelineObserverTimelineProjectionErrorAdvancePolicy`.
- [ ] Add `TestHubPipelineObserverRebuild`.
- [ ] Add `TestHubPipelineObserverPanicIsRecovered`.
- [ ] Assert cloned payloads are safe to retain in observer records.

## Phase 11: Tests for WebSocket observer

- [ ] Add `TestTransportObserverSubscribeSequence`.
- [ ] Add `TestTransportObserverBadClientFrame`.
- [ ] Add `TestTransportObserverFanoutTargets`.
- [ ] Add `TestTransportObserverFanoutNoTargets`.
- [ ] Add `TestTransportObserverQueueFull` or a lower-level send buffer test if direct queue-full is hard to force.
- [ ] Add `TestTransportObserverPanicIsRecovered`.
- [ ] Assert existing hook callbacks still fire as before.

## Phase 12: Documentation and validation

- [ ] Add package-level documentation or README section explaining the observer APIs.
- [ ] Add example wiring snippet for applications.
- [ ] Run `go test ./pkg/sessionstream/... -count=1`.
- [ ] Run any repository lint/format targets normally used for sessionstream.
- [ ] Update changelog with implemented API names and any intentional behavior notes.
