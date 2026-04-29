package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) handlePhase5Command(ctx context.Context, cmd sessionstream.Command, _ *sessionstream.Session, pub sessionstream.EventPublisher) error {
	payload := protoStructMap(cmd.Payload)
	text := strings.TrimSpace(toString(payload["text"]))
	if text == "" {
		text = fmt.Sprintf("record-%d", time.Now().UnixNano())
	}
	e.appendPhase5Trace("handler", "phase 5 handler invoked", map[string]any{"sessionId": string(cmd.SessionId), "text": text})
	pb, err := structpb.NewStruct(map[string]any{"text": text})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, sessionstream.Event{Name: phase5EventName, SessionId: cmd.SessionId, Payload: pb})
}

func (e *labEnvironment) phase5UIProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
	payload := protoStructMap(ev.Payload)
	payload["ordinal"] = fmt.Sprintf("%d", ev.Ordinal)
	pb, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, err
	}
	e.appendPhase5Trace("ui-projection", "phase 5 ui projection emitted event", map[string]any{"sessionId": string(ev.SessionId), "ordinal": fmt.Sprintf("%d", ev.Ordinal), "text": payload["text"]})
	return []sessionstream.UIEvent{{Name: phase5UIEventName, Payload: pb}}, nil
}

func (e *labEnvironment) phase5TimelineProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
	payload := protoStructMap(ev.Payload)
	id := fmt.Sprintf("record-%d", ev.Ordinal)
	pb, err := structpb.NewStruct(map[string]any{"text": payload["text"], "ordinal": fmt.Sprintf("%d", ev.Ordinal)})
	if err != nil {
		return nil, err
	}
	e.appendPhase5Trace("timeline-projection", "phase 5 timeline projection upserted entity", map[string]any{"sessionId": string(ev.SessionId), "ordinal": fmt.Sprintf("%d", ev.Ordinal), "entityId": id})
	return []sessionstream.TimelineEntity{{Kind: phase5EntityKind, Id: id, Payload: pb}}, nil
}

func (e *labEnvironment) appendPhase5Trace(kind, message string, details map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase5 == nil {
		return
	}
	appendTraceEntry(&e.phase5.trace, kind, message, details)
}
