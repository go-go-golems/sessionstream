package main

import (
	"context"
	"fmt"

	wstransport "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

type websocketTraceOptions struct {
	Phase             int
	AppendTrace       func(kind, message string, details map[string]any)
	IncludeUIPayload  bool
	RecordClientFrame bool
}

func newWebsocketTraceObserver(opts websocketTraceOptions) wstransport.TransportObserver {
	phaseLabel := fmt.Sprintf("phase %d", opts.Phase)
	appendTrace := opts.AppendTrace
	if appendTrace == nil {
		appendTrace = func(string, string, map[string]any) {}
	}

	return wstransport.TransportObserverFunc(func(_ context.Context, rec wstransport.TransportRecord) {
		//nolint:exhaustive // Systemlab renders a curated teaching trace, not every low-level transport observation.
		switch rec.Stage {
		case wstransport.TransportStageConnected:
			appendTrace("transport", phaseLabel+" websocket connected", traceConnectionDetails(rec))
		case wstransport.TransportStageDisconnected:
			appendTrace("transport", phaseLabel+" websocket disconnected", traceConnectionDetails(rec))
		case wstransport.TransportStageSubscribed:
			details := traceSessionDetails(rec)
			if rec.SinceSnapshotOrdinal > 0 {
				details["sinceSnapshotOrdinal"] = fmt.Sprintf("%d", rec.SinceSnapshotOrdinal)
			} else {
				details["sinceSnapshotOrdinal"] = "0"
			}
			appendTrace("transport", phaseLabel+" subscribed", details)
		case wstransport.TransportStageUnsubscribed:
			appendTrace("transport", phaseLabel+" unsubscribed", traceSessionDetails(rec))
		case wstransport.TransportStageSnapshotSent:
			details := traceSessionDetails(rec)
			details["snapshotOrdinal"] = fmt.Sprintf("%d", rec.SnapshotOrdinal)
			details["entityCount"] = rec.SnapshotEntityCount
			appendTrace("transport", phaseLabel+" snapshot sent", details)
		case wstransport.TransportStageUIEventSent:
			details := map[string]any{}
			if opts.IncludeUIPayload {
				details = protoStructMap(rec.UIEvent.Payload)
			}
			addConnectionAndSession(details, rec)
			details["eventOrdinal"] = fmt.Sprintf("%d", rec.Ordinal)
			details["uiEvent"] = rec.EventName
			appendTrace("transport", phaseLabel+" ui event sent", details)
		case wstransport.TransportStageClientFrameDecoded:
			if opts.RecordClientFrame {
				details := traceSessionDetails(rec)
				details["frameType"] = rec.FrameType
				if rec.RawBytes > 0 {
					details["rawBytes"] = rec.RawBytes
				}
				if rec.SinceSnapshotOrdinal > 0 {
					details["sinceSnapshotOrdinal"] = fmt.Sprintf("%d", rec.SinceSnapshotOrdinal)
				}
				appendTrace("client-frame", phaseLabel+" client frame received", details)
			}
		default:
			// Systemlab renders a curated teaching trace, not every low-level
			// transport observation.
		}
	})
}

func traceConnectionDetails(rec wstransport.TransportRecord) map[string]any {
	details := map[string]any{}
	addConnectionAndSession(details, rec)
	return details
}

func traceSessionDetails(rec wstransport.TransportRecord) map[string]any {
	details := map[string]any{}
	addConnectionAndSession(details, rec)
	return details
}

func addConnectionAndSession(details map[string]any, rec wstransport.TransportRecord) {
	if rec.ConnectionId != "" {
		details["connectionId"] = string(rec.ConnectionId)
	}
	if rec.SessionId != "" {
		details["sessionId"] = string(rec.SessionId)
	}
}
