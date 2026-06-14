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
		_, err := m.callJSCallback(ctx, "sessionstream.command."+cmd.Name, func(vm *goja.Runtime) (goja.Value, error) {
			cmdValue, err := m.commandToJS(cmd)
			if err != nil {
				return nil, err
			}
			publisher := m.wrapPublisher(schemas, cmd.SessionId, pub)
			return fn(goja.Undefined(), cmdValue, m.sessionToJS(sess), publisher)
		}, nil)
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
	m.mustSet(obj, "publishAsync", func(name string, payload goja.Value) goja.Value {
		msg, err := m.jsValueToProto(schemas, schemaKindEvent, name, payload)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		services, ok := runtimebridge.Lookup(m.vm)
		if !ok || services.Owner == nil {
			panic(m.vm.NewGoError(fmt.Errorf("publishAsync requires runtime services")))
		}
		promise, resolve, reject := m.vm.NewPromise()
		callCtx := runtimebridge.CurrentOwnerContext(m.vm)
		go func() {
			err := pub.Publish(callCtx, ss.Event{Name: name, SessionId: sid, Payload: msg})
			_ = services.PostWithCustomContext(callCtx, "sessionstream.publishAsync.settle", func(context.Context, *goja.Runtime) {
				if err != nil {
					_ = reject(m.vm.ToValue(err.Error()))
					return
				}
				_ = resolve(goja.Undefined())
			})
		}()
		return m.vm.ToValue(promise)
	})
	return obj
}

func (m *moduleRuntime) uiProjection(schemas *ss.SchemaRegistry, fn goja.Callable) func(context.Context, ss.Event, *ss.Session, ss.TimelineView) ([]ss.UIEvent, error) {
	return func(ctx context.Context, ev ss.Event, sess *ss.Session, view ss.TimelineView) ([]ss.UIEvent, error) {
		out, err := m.callJSCallback(ctx, "sessionstream.uiProjection."+ev.Name, func(vm *goja.Runtime) (goja.Value, error) {
			eventValue, err := m.eventToJS(ev)
			if err != nil {
				return nil, err
			}
			return fn(goja.Undefined(), eventValue, m.sessionToJS(sess), m.wrapTimelineView(view))
		}, func(_ *goja.Runtime, value goja.Value) (any, error) {
			return m.decodeUIEvents(schemas, value)
		})
		if err != nil {
			return nil, err
		}
		return out.([]ss.UIEvent), nil
	}
}

func (m *moduleRuntime) timelineProjection(schemas *ss.SchemaRegistry, fn goja.Callable) func(context.Context, ss.Event, *ss.Session, ss.TimelineView) ([]ss.TimelineEntity, error) {
	return func(ctx context.Context, ev ss.Event, sess *ss.Session, view ss.TimelineView) ([]ss.TimelineEntity, error) {
		out, err := m.callJSCallback(ctx, "sessionstream.timelineProjection."+ev.Name, func(vm *goja.Runtime) (goja.Value, error) {
			eventValue, err := m.eventToJS(ev)
			if err != nil {
				return nil, err
			}
			return fn(goja.Undefined(), eventValue, m.sessionToJS(sess), m.wrapTimelineView(view))
		}, func(_ *goja.Runtime, value goja.Value) (any, error) {
			return m.decodeTimelineEntities(schemas, value)
		})
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
