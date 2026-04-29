package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) handlePhase2Command(ctx context.Context, cmd sessionstream.Command, sess *sessionstream.Session, pub sessionstream.EventPublisher) error {
	payload := protoStructMap(cmd.Payload)
	label := strings.TrimSpace(toString(payload["label"]))
	if label == "" {
		label = fmt.Sprintf("%s-%d", cmd.SessionId, time.Now().UnixNano())
	}
	e.mu.Lock()
	e.phase2AppendTraceLocked("handler", "phase 2 handler invoked", map[string]any{
		"sessionId":  string(cmd.SessionId),
		"label":      label,
		"hasSession": sess != nil,
	})
	e.mu.Unlock()
	eventPayload, err := structpb.NewStruct(map[string]any{
		"label":     label,
		"sessionId": string(cmd.SessionId),
	})
	if err != nil {
		return err
	}
	return pub.Publish(ctx, sessionstream.Event{Name: phase2EventName, SessionId: cmd.SessionId, Payload: eventPayload})
}

func (e *labEnvironment) phase2UIProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
	payload := protoStructMap(ev.Payload)
	payload["ordinal"] = fmt.Sprintf("%d", ev.Ordinal)
	pb, err := structpb.NewStruct(payload)
	if err != nil {
		return nil, err
	}
	return []sessionstream.UIEvent{{Name: phase2UIEventName, Payload: pb}}, nil
}

func (e *labEnvironment) phase2TimelineProjection(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
	payload := protoStructMap(ev.Payload)
	label := toString(payload["label"])
	entityPayload, err := structpb.NewStruct(map[string]any{
		"label":     label,
		"sessionId": payload["sessionId"],
		"ordinal":   fmt.Sprintf("%d", ev.Ordinal),
	})
	if err != nil {
		return nil, err
	}
	return []sessionstream.TimelineEntity{{Kind: phase2TimelineEntity, Id: label, Payload: entityPayload}}, nil
}

func (e *labEnvironment) phase2MessageMutator(_ context.Context, _ sessionstream.Event, msg *message.Message) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 == nil {
		return nil
	}
	e.phase2.syntheticSequence++
	seq := e.phase2.syntheticSequence
	switch e.phase2.streamMode {
	case "missing":
		return nil
	case "invalid":
		msg.Metadata.Set(sessionstream.MetadataKeyStreamID, fmt.Sprintf("invalid-%d", seq))
	default:
		msg.Metadata.Set(sessionstream.MetadataKeyStreamID, fmt.Sprintf("1713560000123-%d", seq))
	}
	return nil
}

func (e *labEnvironment) phase2Published(_ context.Context, ev sessionstream.Event, rec sessionstream.BusRecord) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 == nil {
		return
	}
	record := e.phase2RecordLocked(rec.MessageID)
	record.MessageID = rec.MessageID
	record.SessionID = string(ev.SessionId)
	record.EventName = ev.Name
	record.Label = toString(protoStructMap(ev.Payload)["label"])
	record.Topic = rec.Topic
	record.PublishedOrdinal = ev.Ordinal
	record.PublishMetadata = cloneStringMap(rec.Metadata)
	e.phase2AppendTraceLocked("publish", "phase 2 event published", map[string]any{
		"messageId": rec.MessageID,
		"sessionId": record.SessionID,
		"streamId":  rec.Metadata[sessionstream.MetadataKeyStreamID],
	})
}

func (e *labEnvironment) phase2Consumed(_ context.Context, ev sessionstream.Event, rec sessionstream.BusRecord) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 == nil {
		return
	}
	record := e.phase2RecordLocked(rec.MessageID)
	record.MessageID = rec.MessageID
	record.SessionID = string(ev.SessionId)
	record.EventName = ev.Name
	record.Label = toString(protoStructMap(ev.Payload)["label"])
	record.Topic = rec.Topic
	record.AssignedOrdinal = ev.Ordinal
	record.ConsumeMetadata = cloneStringMap(rec.Metadata)
	e.phase2.ordinals[string(ev.SessionId)] = append(e.phase2.ordinals[string(ev.SessionId)], ev.Ordinal)
	e.phase2AppendTraceLocked("consume", "phase 2 event consumed", map[string]any{
		"messageId": rec.MessageID,
		"sessionId": string(ev.SessionId),
		"ordinal":   fmt.Sprintf("%d", ev.Ordinal),
		"streamId":  rec.Metadata[sessionstream.MetadataKeyStreamID],
	})
}

func (e *labEnvironment) phase2RecordLocked(messageID string) *phase2MessageRecord {
	record := e.phase2.messages[messageID]
	if record == nil {
		record = &phase2MessageRecord{}
		e.phase2.messages[messageID] = record
		e.phase2.messageOrder = append(e.phase2.messageOrder, messageID)
	}
	return record
}

func (e *labEnvironment) phase2AppendTraceLocked(kind, message string, details map[string]any) {
	appendTraceEntry(&e.phase2.trace, kind, message, details)
}
