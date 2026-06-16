package sessionstream

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type schemaKind string

const (
	schemaKindCommand schemaKind = "command"
	schemaKindEvent   schemaKind = "event"
	schemaKindUIEvent schemaKind = "ui event"
	schemaKindEntity  schemaKind = "timeline entity"
)

var protoJSONMarshal = protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: false}
var protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: false}

func (m *moduleRuntime) protoToJSValue(msg proto.Message) (goja.Value, error) {
	if msg == nil {
		return goja.Null(), nil
	}
	b, err := protoJSONMarshal.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	return m.vm.ToValue(decoded), nil
}

func (m *moduleRuntime) jsValueToProto(registry *ss.SchemaRegistry, kind schemaKind, name string, value goja.Value) (proto.Message, error) {
	if registry == nil {
		return nil, fmt.Errorf("schema registry is nil")
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, fmt.Errorf("%s %q payload is null or undefined", kind, name)
	}
	if msg, ok := protogoja.MessageFromValue(value); ok {
		if err := validateMessageType(registry, kind, name, msg); err != nil {
			return nil, err
		}
		return proto.Clone(msg), nil
	}
	prototype, err := lookupPrototype(registry, kind, name)
	if err != nil {
		return nil, err
	}
	payload := value.Export()
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal JS %s %q payload: %w", kind, name, err)
	}
	msg := prototype.ProtoReflect().New().Interface()
	if err := protoJSONUnmarshal.Unmarshal(b, msg); err != nil {
		return nil, fmt.Errorf("decode %s %q: %w", kind, name, err)
	}
	return msg, nil
}

func validateMessageType(registry *ss.SchemaRegistry, kind schemaKind, name string, msg proto.Message) error {
	prototype, err := lookupPrototype(registry, kind, name)
	if err != nil {
		return err
	}
	if prototype.ProtoReflect().Descriptor().FullName() != msg.ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("%s %q expects %s, got %s", kind, name, prototype.ProtoReflect().Descriptor().FullName(), msg.ProtoReflect().Descriptor().FullName())
	}
	return nil
}

func lookupPrototype(registry *ss.SchemaRegistry, kind schemaKind, name string) (proto.Message, error) {
	var (
		msg proto.Message
		ok  bool
	)
	switch kind {
	case schemaKindCommand:
		msg, ok = registry.CommandSchema(name)
	case schemaKindEvent:
		msg, ok = registry.EventSchema(name)
	case schemaKindUIEvent:
		msg, ok = registry.UIEventSchema(name)
	case schemaKindEntity:
		msg, ok = registry.TimelineEntitySchema(name)
	default:
		return nil, fmt.Errorf("unknown schema kind %q", kind)
	}
	if !ok || msg == nil {
		return nil, fmt.Errorf("unknown %s %q", kind, name)
	}
	return msg, nil
}

func (m *moduleRuntime) commandToJS(cmd ss.Command) (goja.Value, error) {
	payload, err := m.protoToJSValue(cmd.Payload)
	if err != nil {
		return nil, err
	}
	obj := m.vm.NewObject()
	m.mustSet(obj, "name", cmd.Name)
	m.mustSet(obj, "sessionId", string(cmd.SessionId))
	m.mustSet(obj, "payload", payload)
	return obj, nil
}

func (m *moduleRuntime) eventToJS(ev ss.Event) (goja.Value, error) {
	payload, err := m.protoToJSValue(ev.Payload)
	if err != nil {
		return nil, err
	}
	obj := m.vm.NewObject()
	m.mustSet(obj, "name", ev.Name)
	m.mustSet(obj, "sessionId", string(ev.SessionId))
	m.mustSet(obj, "ordinal", strconv.FormatUint(ev.Ordinal, 10))
	m.mustSet(obj, "payload", payload)
	return obj, nil
}

func (m *moduleRuntime) sessionToJS(sess *ss.Session) goja.Value {
	obj := m.vm.NewObject()
	if sess == nil {
		return obj
	}
	m.mustSet(obj, "id", string(sess.Id))
	m.mustSet(obj, "metadata", sess.Metadata)
	return obj
}

func (m *moduleRuntime) uiEventToJS(ev ss.UIEvent) (map[string]any, error) {
	value, err := m.protoToJSValue(ev.Payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{"name": ev.Name, "payload": value.Export()}, nil
}

func (m *moduleRuntime) timelineEntityToJS(ent ss.TimelineEntity) (map[string]any, error) {
	value, err := m.protoToJSValue(ent.Payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"kind":             ent.Kind,
		"id":               ent.Id,
		"createdOrdinal":   strconv.FormatUint(ent.CreatedOrdinal, 10),
		"lastEventOrdinal": strconv.FormatUint(ent.LastEventOrdinal, 10),
		"payload":          value.Export(),
		"tombstone":        ent.Tombstone,
	}, nil
}
