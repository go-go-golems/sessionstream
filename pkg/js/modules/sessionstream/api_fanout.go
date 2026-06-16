package sessionstream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

type eventEmitterFanout struct {
	module *moduleRuntime
	ref    *jsevents.EmitterRef
}

func (m *moduleRuntime) eventEmitterFanoutBuilder(call goja.FunctionCall) goja.Value {
	manager, err := m.resolveEventEmitterManager()
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	emitterRef, err := manager.AdoptEmitterOnOwner(call.Argument(0))
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	fanout := &eventEmitterFanout{module: m, ref: emitterRef}
	obj := m.vm.NewObject()
	m.attachRef(obj, &fanoutRef{fanout: fanout, close: func() error { return emitterRef.Close(context.Background()) }})
	m.mustSet(obj, "close", func() { _ = emitterRef.Close(context.Background()) })
	m.mustSet(obj, "id", emitterRef.ID())
	return obj
}

func (m *moduleRuntime) resolveEventEmitterManager() (*jsevents.Manager, error) {
	if m.eventEmitterManager != nil {
		return m.eventEmitterManager, nil
	}
	if m.eventEmitterManagerResolver != nil {
		if mgr, ok := m.eventEmitterManagerResolver(); ok && mgr != nil {
			return mgr, nil
		}
	}
	return nil, fmt.Errorf("sessionstream: EventEmitter manager is not configured")
}

func (f *eventEmitterFanout) PublishUI(ctx context.Context, sid ss.SessionId, ord uint64, events []ss.UIEvent) error {
	if f == nil || f.ref == nil || f.module == nil {
		return fmt.Errorf("sessionstream: nil event emitter fanout")
	}
	copied := append([]ss.UIEvent(nil), events...)
	return f.ref.EmitWithBuilder(ctx, "ui", func(vm *goja.Runtime) ([]goja.Value, error) {
		out := make([]map[string]any, 0, len(copied))
		for _, ev := range copied {
			converted, err := f.module.uiEventToJS(ev)
			if err != nil {
				return nil, err
			}
			out = append(out, converted)
		}
		batch := map[string]any{"sessionId": string(sid), "ordinal": strconv.FormatUint(ord, 10), "events": out}
		return []goja.Value{vm.ToValue(batch)}, nil
	})
}
