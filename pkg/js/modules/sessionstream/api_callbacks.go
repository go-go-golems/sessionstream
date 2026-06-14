package sessionstream

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

func (m *moduleRuntime) commandHandler(schemas *ss.SchemaRegistry, fn goja.Callable) ss.CommandHandler {
	return func(ctx context.Context, cmd ss.Command, sess *ss.Session, pub ss.EventPublisher) error {
		call := func(_ context.Context, vm *goja.Runtime) (any, error) {
			cmdValue, err := m.commandToJS(cmd)
			if err != nil {
				return nil, err
			}
			publisher := m.wrapPublisher(schemas, cmd.SessionId, pub)
			_, err = fn(goja.Undefined(), cmdValue, m.sessionToJS(sess), publisher)
			return nil, err
		}
		if m.runtimeOwner != nil {
			_, err := m.runtimeOwner.Call(ctx, "sessionstream.command."+cmd.Name, call)
			return err
		}
		_, err := call(ctx, m.vm)
		return err
	}
}

func (m *moduleRuntime) wrapPublisher(schemas *ss.SchemaRegistry, sid ss.SessionId, pub ss.EventPublisher) goja.Value {
	obj := m.vm.NewObject()
	m.mustSet(obj, "publish", func(name string, payload goja.Value) goja.Value {
		msg, err := m.jsValueToProto(schemas, schemaKindEvent, name, payload)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		if err := pub.Publish(runtimebridge.CurrentOwnerContext(m.vm), ss.Event{Name: name, SessionId: sid, Payload: msg}); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	return obj
}

func (m *moduleRuntime) uiProjection(schemas *ss.SchemaRegistry, fn goja.Callable) func(context.Context, ss.Event, *ss.Session, ss.TimelineView) ([]ss.UIEvent, error) {
	return func(ctx context.Context, ev ss.Event, sess *ss.Session, view ss.TimelineView) ([]ss.UIEvent, error) {
		call := func(_ context.Context, vm *goja.Runtime) (any, error) {
			eventValue, err := m.eventToJS(ev)
			if err != nil {
				return nil, err
			}
			out, err := fn(goja.Undefined(), eventValue, m.sessionToJS(sess), m.wrapTimelineView(view))
			if err != nil {
				return nil, err
			}
			return m.decodeUIEvents(schemas, out)
		}
		if m.runtimeOwner != nil {
			out, err := m.runtimeOwner.Call(ctx, "sessionstream.uiProjection."+ev.Name, call)
			if err != nil {
				return nil, err
			}
			return out.([]ss.UIEvent), nil
		}
		out, err := call(ctx, m.vm)
		if err != nil {
			return nil, err
		}
		return out.([]ss.UIEvent), nil
	}
}

func (m *moduleRuntime) timelineProjection(schemas *ss.SchemaRegistry, fn goja.Callable) func(context.Context, ss.Event, *ss.Session, ss.TimelineView) ([]ss.TimelineEntity, error) {
	return func(ctx context.Context, ev ss.Event, sess *ss.Session, view ss.TimelineView) ([]ss.TimelineEntity, error) {
		call := func(_ context.Context, vm *goja.Runtime) (any, error) {
			eventValue, err := m.eventToJS(ev)
			if err != nil {
				return nil, err
			}
			out, err := fn(goja.Undefined(), eventValue, m.sessionToJS(sess), m.wrapTimelineView(view))
			if err != nil {
				return nil, err
			}
			return m.decodeTimelineEntities(schemas, out)
		}
		if m.runtimeOwner != nil {
			out, err := m.runtimeOwner.Call(ctx, "sessionstream.timelineProjection."+ev.Name, call)
			if err != nil {
				return nil, err
			}
			return out.([]ss.TimelineEntity), nil
		}
		out, err := call(ctx, m.vm)
		if err != nil {
			return nil, err
		}
		return out.([]ss.TimelineEntity), nil
	}
}

func (m *moduleRuntime) decodeUIEvents(schemas *ss.SchemaRegistry, value goja.Value) ([]ss.UIEvent, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	items, err := m.arrayValues(value)
	if err != nil {
		return nil, fmt.Errorf("decode UI projection result: %w", err)
	}
	out := make([]ss.UIEvent, 0, len(items))
	for _, item := range items {
		obj := item.ToObject(m.vm)
		name := getProperty(obj, "name").String()
		msg, err := m.jsValueToProto(schemas, schemaKindUIEvent, name, getProperty(obj, "payload"))
		if err != nil {
			return nil, err
		}
		out = append(out, ss.UIEvent{Name: name, Payload: msg})
	}
	return out, nil
}

func (m *moduleRuntime) decodeTimelineEntities(schemas *ss.SchemaRegistry, value goja.Value) ([]ss.TimelineEntity, error) {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, nil
	}
	items, err := m.arrayValues(value)
	if err != nil {
		return nil, fmt.Errorf("decode timeline projection result: %w", err)
	}
	out := make([]ss.TimelineEntity, 0, len(items))
	for _, item := range items {
		obj := item.ToObject(m.vm)
		kind := getProperty(obj, "kind").String()
		msg, err := m.jsValueToProto(schemas, schemaKindEntity, kind, getProperty(obj, "payload"))
		if err != nil {
			return nil, err
		}
		out = append(out, ss.TimelineEntity{Kind: kind, Id: getProperty(obj, "id").String(), Payload: msg, Tombstone: getProperty(obj, "tombstone").ToBoolean()})
	}
	return out, nil
}

func getProperty(obj *goja.Object, key string) goja.Value {
	if obj == nil {
		return goja.Undefined()
	}
	value := obj.Get(key)
	if value == nil {
		return goja.Undefined()
	}
	return value
}

func (m *moduleRuntime) arrayValues(value goja.Value) ([]goja.Value, error) {
	obj := value.ToObject(m.vm)
	length := obj.Get("length")
	if length == nil || goja.IsUndefined(length) {
		return nil, fmt.Errorf("expected an array-like value")
	}
	n := int(length.ToInteger())
	out := make([]goja.Value, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, obj.Get(fmt.Sprintf("%d", i)))
	}
	return out, nil
}
