package sessionstream

import (
	"fmt"

	"github.com/dop251/goja"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/proto"
)

type schemaInput struct {
	Commands map[string]any `json:"commands"`
	Events   map[string]any `json:"events"`
	UIEvents map[string]any `json:"uiEvents"`
	Entities map[string]any `json:"entities"`
}

func (m *moduleRuntime) schemasBuilder(call goja.FunctionCall) goja.Value {
	registry := ss.NewSchemaRegistry()
	if m.defaultSchemaRegistry != nil && goja.IsUndefined(call.Argument(0)) {
		return m.wrapSchemaRegistry(m.defaultSchemaRegistry)
	}
	input := schemaInput{}
	if arg := call.Argument(0); !goja.IsUndefined(arg) && !goja.IsNull(arg) {
		if err := m.vm.ExportTo(arg, &input); err != nil {
			panic(m.vm.NewGoError(fmt.Errorf("decode schema input: %w", err)))
		}
	}
	for name, spec := range input.Commands {
		msg := m.mustResolveSchemaSpec(spec)
		if err := registry.RegisterCommand(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	for name, spec := range input.Events {
		msg := m.mustResolveSchemaSpec(spec)
		if err := registry.RegisterEvent(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	for name, spec := range input.UIEvents {
		msg := m.mustResolveSchemaSpec(spec)
		if err := registry.RegisterUIEvent(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	for name, spec := range input.Entities {
		msg := m.mustResolveSchemaSpec(spec)
		if err := registry.RegisterTimelineEntity(name, msg); err != nil {
			panic(m.vm.NewGoError(err))
		}
	}
	return m.wrapSchemaRegistry(registry)
}

func (m *moduleRuntime) mustResolveSchemaSpec(spec any) proto.Message {
	switch v := spec.(type) {
	case string:
		msg, err := m.resolvePrototype(v)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return msg
	case map[string]any:
		if full, ok := v["type"].(string); ok {
			msg, err := m.resolvePrototype(full)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			return msg
		}
	}
	value := m.vm.ToValue(spec)
	if msg, _, ok := m.prototypeFromValue(value); ok {
		return msg
	}
	panic(m.vm.NewTypeError("schema values must be protobuf full-name strings or generated message namespace objects"))
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
		if full := value.String(); full != "" && full != "[object Object]" {
			msg, err := m.resolvePrototype(full)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			return msg
		}
	}
	panic(m.vm.NewTypeError("schema must be a generated message namespace or protobuf full name"))
}
