package sessionstream

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// PipelineMode identifies which Hub processing path produced a PipelineRecord.
type PipelineMode string

const (
	// PipelineModeLive observes normal command/event processing through projectAndApply.
	PipelineModeLive PipelineMode = "live"
	// PipelineModeRebuild observes timeline replay through rebuildTimelineEvent.
	PipelineModeRebuild PipelineMode = "rebuild"
)

// PipelineRecord describes how one backend event moved through the Hub pipeline.
//
// Observers receive cloned protobuf payloads so they may retain records for
// diagnostics without mutating the live pipeline state.
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

// PipelineObserver receives best-effort Hub pipeline observations.
type PipelineObserver interface {
	OnPipeline(ctx context.Context, rec PipelineRecord)
}

// PipelineObserverFunc adapts a function to PipelineObserver.
type PipelineObserverFunc func(ctx context.Context, rec PipelineRecord)

func (f PipelineObserverFunc) OnPipeline(ctx context.Context, rec PipelineRecord) {
	if f != nil {
		f(ctx, rec)
	}
}

// PipelineObserverHooks adapts optional callbacks to PipelineObserver.
type PipelineObserverHooks struct {
	OnPipelineFunc func(ctx context.Context, rec PipelineRecord)
}

func (h PipelineObserverHooks) OnPipeline(ctx context.Context, rec PipelineRecord) {
	if h.OnPipelineFunc != nil {
		h.OnPipelineFunc(ctx, rec)
	}
}

// WithPipelineObserver installs a best-effort Hub pipeline observer.
func WithPipelineObserver(observer PipelineObserver) HubOption {
	return func(h *Hub) error {
		h.pipelineObserver = observer
		return nil
	}
}

func (h *Hub) observePipeline(ctx context.Context, rec PipelineRecord) {
	if h == nil || h.pipelineObserver == nil {
		return
	}
	safe := clonePipelineRecord(rec)
	defer func() { _ = recover() }()
	h.pipelineObserver.OnPipeline(ctx, safe)
}

func clonePipelineRecord(in PipelineRecord) PipelineRecord {
	out := in
	out.Event = cloneEvent(in.Event)
	out.UIEvents = cloneUIEvents(in.UIEvents)
	out.TimelineEntities = cloneTimelineEntities(in.TimelineEntities)
	out.AppliedEntities = cloneTimelineEntities(in.AppliedEntities)
	out.FanoutEvents = cloneUIEvents(in.FanoutEvents)
	return out
}

func cloneEvent(in Event) Event {
	out := in
	if in.Payload != nil {
		out.Payload = proto.Clone(in.Payload)
	}
	return out
}

func cloneTimelineEntities(in []TimelineEntity) []TimelineEntity {
	if len(in) == 0 {
		return nil
	}
	out := make([]TimelineEntity, 0, len(in))
	for _, entity := range in {
		cloned := entity
		if entity.Payload != nil {
			cloned.Payload = proto.Clone(entity.Payload)
		}
		out = append(out, cloned)
	}
	return out
}
