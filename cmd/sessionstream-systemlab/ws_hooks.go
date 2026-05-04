package main

import (
	"fmt"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	wstransport "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

type websocketTraceOptions struct {
	Phase             int
	AppendTrace       func(kind, message string, details map[string]any)
	IncludeUIPayload  bool
	RecordClientFrame bool
}

func newWebsocketTraceHooks(opts websocketTraceOptions) wstransport.Hooks {
	phaseLabel := fmt.Sprintf("phase %d", opts.Phase)
	appendTrace := opts.AppendTrace
	if appendTrace == nil {
		appendTrace = func(string, string, map[string]any) {}
	}

	hooks := wstransport.Hooks{
		OnConnect: func(cid sessionstream.ConnectionId) {
			appendTrace("transport", phaseLabel+" websocket connected", map[string]any{"connectionId": string(cid)})
		},
		OnDisconnect: func(cid sessionstream.ConnectionId) {
			appendTrace("transport", phaseLabel+" websocket disconnected", map[string]any{"connectionId": string(cid)})
		},
		OnSubscribe: func(cid sessionstream.ConnectionId, sid sessionstream.SessionId, since uint64) {
			appendTrace("transport", phaseLabel+" subscribed", map[string]any{"connectionId": string(cid), "sessionId": string(sid), "sinceSnapshotOrdinal": fmt.Sprintf("%d", since)})
		},
		OnUnsubscribe: func(cid sessionstream.ConnectionId, sid sessionstream.SessionId) {
			appendTrace("transport", phaseLabel+" unsubscribed", map[string]any{"connectionId": string(cid), "sessionId": string(sid)})
		},
		OnSnapshotSent: func(cid sessionstream.ConnectionId, sid sessionstream.SessionId, snap sessionstream.Snapshot) {
			appendTrace("transport", phaseLabel+" snapshot sent", map[string]any{"connectionId": string(cid), "sessionId": string(sid), "snapshotOrdinal": fmt.Sprintf("%d", snap.SnapshotOrdinal), "entityCount": len(snap.Entities)})
		},
		OnUIEventSent: func(cid sessionstream.ConnectionId, sid sessionstream.SessionId, ord uint64, event sessionstream.UIEvent) {
			details := map[string]any{}
			if opts.IncludeUIPayload {
				details = protoStructMap(event.Payload)
			}
			details["connectionId"] = string(cid)
			details["sessionId"] = string(sid)
			details["eventOrdinal"] = fmt.Sprintf("%d", ord)
			details["uiEvent"] = event.Name
			appendTrace("transport", phaseLabel+" ui event sent", details)
		},
	}

	if opts.RecordClientFrame {
		hooks.OnClientFrame = func(cid sessionstream.ConnectionId, frame map[string]any) {
			frame["connectionId"] = string(cid)
			appendTrace("client-frame", phaseLabel+" client frame received", frame)
		}
	}

	return hooks
}
