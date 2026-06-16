package sessionstream

import (
	"github.com/dop251/goja"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/proto"
)

type schemaRegistrar func(name string, msg proto.Message) error

func (m *moduleRuntime) schemasBuilder(call goja.FunctionCall) goja.Value {
	registry := ss.NewSchemaRegistry()
	if m.defaultSchemaRegistry != nil && goja.IsUndefined(call.Argument(0)) {
		return m.wrapSchemaRegistry(m.defaultSchemaRegistry)
	}
	if arg := call.Argument(0); !goja.IsUndefined(arg) && !goja.IsNull(arg) {
		input, ok := arg.(*goja.Object)
		if !ok || input == nil {
			panic(m.vm.NewTypeError("schema input must be an object"))
		}
		m.registerSchemaSection("commands", input.Get("commands"), registry.RegisterCommand)
		m.registerSchemaSection("events", input.Get("events"), registry.RegisterEvent)
		m.registerSchemaSection("uiEvents", input.Get("uiEvents"), registry.RegisterUIEvent)
		m.registerSchemaSection("entities", input.Get("entities"), registry.RegisterTimelineEntity)
	}
	return m.wrapSchemaRegistry(registry)
}

func (m *moduleRuntime) registerSchemaSection(section string, value goja.Value, register schemaRegistrar) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return
	}
	obj, ok := value.(*goja.Object)
	if !ok || obj == nil {
		panic(m.vm.NewTypeError("schema %s must be an object", section))
	}
	for _, name := range obj.Keys() {
		msg := m.mustResolveSchemaValue(obj.Get(name))
		if err := register(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
}

func (m *moduleRuntime) wrapSchemaRegistry(registry *ss.SchemaRegistry) goja.Value {
	obj := m.vm.NewObject()
	m.attachRef(obj, &schemaRegistryRef{registry: registry})
	m.mustSet(obj, "registerCommand", func(name string, schema goja.Value) goja.Value {
		msg := m.mustResolveSchemaValue(schema)
		if err := registry.RegisterCommand(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "registerEvent", func(name string, schema goja.Value) goja.Value {
		msg := m.mustResolveSchemaValue(schema)
		if err := registry.RegisterEvent(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "registerUIEvent", func(name string, schema goja.Value) goja.Value {
		msg := m.mustResolveSchemaValue(schema)
		if err := registry.RegisterUIEvent(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "registerTimelineEntity", func(name string, schema goja.Value) goja.Value {
		msg := m.mustResolveSchemaValue(schema)
		if err := registry.RegisterTimelineEntity(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	return obj
}

func (m *moduleRuntime) mustResolveSchemaValue(value goja.Value) proto.Message {
	if msg, _, ok := m.prototypeFromValue(value); ok {
		return msg
	}
	if value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
		if full, ok := value.Export().(string); ok && full != "" {
			msg, err := m.resolvePrototype(full)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			return msg
		}
	}
	panic(m.vm.NewTypeError("schema must be a generated message namespace or protobuf full name"))
}
