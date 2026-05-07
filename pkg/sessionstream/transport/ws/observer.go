package ws

import (
	"context"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	sessionstreamv1 "github.com/go-go-golems/sessionstream/pkg/sessionstream/pb/proto/sessionstream/v1"
)

// TransportStage identifies a websocket transport observation point.
type TransportStage string

const (
	TransportStageUpgradeError            TransportStage = "upgrade_error"
	TransportStageConnected               TransportStage = "connected"
	TransportStageDisconnected            TransportStage = "disconnected"
	TransportStageClientFrameRead         TransportStage = "client_frame_read"
	TransportStageClientFrameDecoded      TransportStage = "client_frame_decoded"
	TransportStageClientFrameDecodeError  TransportStage = "client_frame_decode_error"
	TransportStageReadError               TransportStage = "read_error"
	TransportStageProtocolError           TransportStage = "protocol_error"
	TransportStageSubscribeReceived       TransportStage = "subscribe_received"
	TransportStageUnsubscribeReceived     TransportStage = "unsubscribe_received"
	TransportStageSnapshotLoadStarted     TransportStage = "snapshot_load_started"
	TransportStageSnapshotLoaded          TransportStage = "snapshot_loaded"
	TransportStageSnapshotSent            TransportStage = "snapshot_sent"
	TransportStageSubscriptionRegistered  TransportStage = "subscription_registered"
	TransportStageSubscribed              TransportStage = "subscribed"
	TransportStageUnsubscribed            TransportStage = "unsubscribed"
	TransportStageUIEventBuffered         TransportStage = "ui_event_buffered"
	TransportStageUIEventSent             TransportStage = "ui_event_sent"
	TransportStageHydrationBufferFlushed  TransportStage = "hydration_buffer_flushed"
	TransportStageSubscriptionLive        TransportStage = "subscription_live"
	TransportStageHydrationBufferOverflow TransportStage = "hydration_buffer_overflow"
	TransportStageServerFrameMarshalError TransportStage = "server_frame_marshal_error"
	TransportStageServerFrameQueued       TransportStage = "server_frame_queued"
	TransportStageServerFrameQueueFull    TransportStage = "server_frame_queue_full"
	TransportStageServerFrameWritten      TransportStage = "server_frame_written"
	TransportStageServerFrameWriteError   TransportStage = "server_frame_write_error"
	TransportStageFanoutStarted           TransportStage = "fanout_started"
	TransportStageFanoutNoTargets         TransportStage = "fanout_no_targets"
	TransportStageFanoutCompleted         TransportStage = "fanout_completed"
)

// FrameDirection identifies whether a frame moved into or out of the server.
type FrameDirection string

const (
	FrameDirectionClientToServer FrameDirection = "client_to_server"
	FrameDirectionServerToClient FrameDirection = "server_to_client"
)

// TimelineEntitySummary is a compact, payload-safe snapshot entity description.
type TimelineEntitySummary struct {
	Kind             string
	Id               string
	CreatedOrdinal   uint64
	LastEventOrdinal uint64
	PayloadType      string
	Tombstone        bool
}

// TransportRecord describes one websocket transport observation.
type TransportRecord struct {
	Stage     TransportStage
	Direction FrameDirection

	ConnectionId sessionstream.ConnectionId
	SessionId    sessionstream.SessionId

	FrameType   string
	Ordinal     uint64
	EventName   string
	PayloadType string

	SinceSnapshotOrdinal uint64
	SnapshotOrdinal      uint64
	SnapshotEntityCount  int
	SnapshotEntities     []TimelineEntitySummary

	FanoutEventCount int
	FanoutTargetIds  []sessionstream.ConnectionId

	// UIEvent is present for TransportStageUIEventSent records. Its payload is cloned
	// before observer delivery so diagnostics can retain it safely.
	UIEvent sessionstream.UIEvent

	RawBytes int
	QueueLen int
	QueueCap int

	Err error
}

// TransportObserver receives best-effort websocket transport observations.
type TransportObserver interface {
	OnTransport(ctx context.Context, rec TransportRecord)
}

// TransportObserverFunc adapts a function to TransportObserver.
type TransportObserverFunc func(ctx context.Context, rec TransportRecord)

func (f TransportObserverFunc) OnTransport(ctx context.Context, rec TransportRecord) {
	if f != nil {
		f(ctx, rec)
	}
}

// WithTransportObserver installs a best-effort websocket transport observer.
func WithTransportObserver(observer TransportObserver) Option {
	return func(s *Server) error {
		s.observer = observer
		return nil
	}
}

func (s *Server) observe(ctx context.Context, rec TransportRecord) {
	if s == nil || s.observer == nil {
		return
	}
	safe := cloneTransportRecord(rec)
	defer func() { _ = recover() }()
	s.observer.OnTransport(ctx, safe)
}

func cloneTransportRecord(in TransportRecord) TransportRecord {
	out := in
	out.UIEvent = cloneUIEvent(in.UIEvent)
	if len(in.SnapshotEntities) > 0 {
		out.SnapshotEntities = append([]TimelineEntitySummary(nil), in.SnapshotEntities...)
	}
	if len(in.FanoutTargetIds) > 0 {
		out.FanoutTargetIds = append([]sessionstream.ConnectionId(nil), in.FanoutTargetIds...)
	}
	return out
}

func summarizeEntities(in []sessionstream.TimelineEntity) []TimelineEntitySummary {
	if len(in) == 0 {
		return nil
	}
	out := make([]TimelineEntitySummary, 0, len(in))
	for _, entity := range in {
		payloadType := ""
		if entity.Payload != nil {
			payloadType = string(entity.Payload.ProtoReflect().Descriptor().FullName())
		}
		out = append(out, TimelineEntitySummary{
			Kind:             entity.Kind,
			Id:               entity.Id,
			CreatedOrdinal:   entity.CreatedOrdinal,
			LastEventOrdinal: entity.LastEventOrdinal,
			PayloadType:      payloadType,
			Tombstone:        entity.Tombstone,
		})
	}
	return out
}

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

func fanoutTargetIDs(targets []*connection) []sessionstream.ConnectionId {
	if len(targets) == 0 {
		return nil
	}
	out := make([]sessionstream.ConnectionId, 0, len(targets))
	for _, target := range targets {
		if target != nil {
			out = append(out, target.id)
		}
	}
	return out
}
