package main

import (
	"encoding/json"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func encodeSnapshot(snap sessionstream.Snapshot) map[string]any {
	entities := make([]map[string]any, 0, len(snap.Entities))
	for _, entity := range snap.Entities {
		entities = append(entities, map[string]any{
			"kind":             entity.Kind,
			"id":               entity.Id,
			"createdOrdinal":   entity.CreatedOrdinal,
			"lastEventOrdinal": entity.LastEventOrdinal,
			"payload":          protoStructMap(entity.Payload),
		})
	}
	return map[string]any{
		"sessionId":       string(snap.SessionId),
		"snapshotOrdinal": snap.SnapshotOrdinal,
		"ordinal":         snap.SnapshotOrdinal,
		"entities":        entities,
	}
}

func currentEntityMap(view sessionstream.TimelineView, id string) map[string]any {
	return currentEntityMapForKind(view, phase1TimelineEntity, id)
}

func currentEntityMapForKind(view sessionstream.TimelineView, kind, id string) map[string]any {
	entity, ok := view.Get(kind, id)
	if !ok || entity.Payload == nil {
		return map[string]any{}
	}
	return protoStructMap(entity.Payload)
}

func protoStructMap(msg proto.Message) map[string]any {
	if pb, ok := msg.(*structpb.Struct); ok && pb != nil {
		return cloneMap(pb.AsMap())
	}
	if msg == nil {
		return map[string]any{}
	}
	body, err := protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: false}.Marshal(msg)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]any{}
	}
	return out
}
