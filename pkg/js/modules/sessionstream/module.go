package sessionstream

import (
	"context"
	"fmt"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	ws "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
	"github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

const (
	ModuleName   = "sessionstream"
	hiddenRefKey = "__sessionstream_ref"
)

type Options struct {
	RuntimeOwner                runtimeowner.RuntimeOwner
	EventEmitterManager         *jsevents.Manager
	EventEmitterManagerResolver func() (*jsevents.Manager, bool)
	DefaultSchemaRegistry       *ss.SchemaRegistry
	DefaultHydrationStore       ss.HydrationStore
	Prototypes                  map[string]proto.Message
	Logger                      zerolog.Logger
}

func NewLoader(opts Options) require.ModuleLoader {
	return (&module{opts: opts}).Loader
}

func Register(reg *require.Registry, opts Options) {
	if reg == nil {
		return
	}
	reg.RegisterNativeModule(ModuleName, NewLoader(opts))
}

type module struct{ opts Options }

type callbackRuntimeOwner interface {
	Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error)
}

type runtimeOwnerAdapter struct{ owner runtimeowner.RuntimeOwner }

func (a runtimeOwnerAdapter) Call(ctx context.Context, op string, fn func(context.Context, *goja.Runtime) (any, error)) (any, error) {
	return a.owner.Call(ctx, op, runtimeowner.CallFunc(fn))
}

type moduleRuntime struct {
	vm                          *goja.Runtime
	runtimeOwner                callbackRuntimeOwner
	eventEmitterManager         *jsevents.Manager
	eventEmitterManagerResolver func() (*jsevents.Manager, bool)
	defaultSchemaRegistry       *ss.SchemaRegistry
	defaultHydrationStore       ss.HydrationStore
	prototypes                  map[string]proto.Message
	logger                      zerolog.Logger
}

type schemaRegistryRef struct{ registry *ss.SchemaRegistry }
type hubRef struct {
	hub     *ss.Hub
	schemas *ss.SchemaRegistry
	queue   *hubQueue
}
type fanoutRef struct {
	fanout ss.UIFanout
	close  func() error
}
type websocketRef struct{ server *ws.Server }

func (m *module) Loader(vm *goja.Runtime, moduleObj *goja.Object) {
	rt := newRuntime(vm, m.opts)
	exports := moduleObj.Get("exports").ToObject(vm)
	rt.installExports(exports)
}

func newRuntime(vm *goja.Runtime, opts Options) *moduleRuntime {
	var runtimeOwner callbackRuntimeOwner
	if opts.RuntimeOwner != nil {
		runtimeOwner = runtimeOwnerAdapter{owner: opts.RuntimeOwner}
	}
	if runtimeOwner == nil {
		if services, ok := runtimebridge.Lookup(vm); ok {
			runtimeOwner = services.Owner
		}
	}
	lg := opts.Logger
	if lg.GetLevel() == zerolog.NoLevel {
		lg = zerologlog.Logger
	}
	prototypes := map[string]proto.Message{}
	for name, msg := range opts.Prototypes {
		if strings.TrimSpace(name) != "" && msg != nil {
			prototypes[strings.TrimSpace(name)] = proto.Clone(msg)
		}
	}
	return &moduleRuntime{
		vm:                          vm,
		runtimeOwner:                runtimeOwner,
		eventEmitterManager:         opts.EventEmitterManager,
		eventEmitterManagerResolver: opts.EventEmitterManagerResolver,
		defaultSchemaRegistry:       opts.DefaultSchemaRegistry,
		defaultHydrationStore:       opts.DefaultHydrationStore,
		prototypes:                  prototypes,
		logger:                      lg,
	}
}

func (m *moduleRuntime) installExports(exports *goja.Object) {
	m.mustSet(exports, "version", "0.1.0")
	m.mustSet(exports, "schemas", m.schemasBuilder)
	m.mustSet(exports, "hub", m.hubBuilder)
	m.mustSet(exports, "eventEmitterFanout", m.eventEmitterFanoutBuilder)
	m.installFanoutNamespace(exports)
	m.installWebSocketNamespace(exports)
}

func (m *moduleRuntime) installFanoutNamespace(exports *goja.Object) {
	obj := m.vm.NewObject()
	m.mustSet(obj, "eventEmitter", m.eventEmitterFanoutBuilder)
	m.mustSet(exports, "fanout", obj)
}

func (m *moduleRuntime) installWebSocketNamespace(exports *goja.Object) {
	obj := m.vm.NewObject()
	m.mustSet(obj, "server", m.webSocketServerBuilder)
	m.mustSet(exports, "webSocket", obj)
}

func (m *moduleRuntime) mustSet(o *goja.Object, key string, value any) {
	if err := o.Set(key, value); err != nil {
		panic(m.vm.NewGoError(fmt.Errorf("set %s: %w", key, err)))
	}
}

func (m *moduleRuntime) attachRef(o *goja.Object, ref any) {
	_ = o.Set(hiddenRefKey, ref)
	_ = o.DefineDataProperty(hiddenRefKey, o.Get(hiddenRefKey), goja.FLAG_FALSE, goja.FLAG_FALSE, goja.FLAG_FALSE)
}

func (m *moduleRuntime) getRef(v goja.Value) any {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	obj, ok := v.(*goja.Object)
	if !ok || obj == nil {
		return nil
	}
	raw := obj.Get(hiddenRefKey)
	if raw == nil || goja.IsUndefined(raw) || goja.IsNull(raw) {
		return nil
	}
	return raw.Export()
}

func (m *moduleRuntime) schemaRef(v goja.Value) (*schemaRegistryRef, bool) {
	ref, ok := m.getRef(v).(*schemaRegistryRef)
	return ref, ok && ref != nil && ref.registry != nil
}

func (m *moduleRuntime) hubRef(v goja.Value) (*hubRef, bool) {
	ref, ok := m.getRef(v).(*hubRef)
	return ref, ok && ref != nil && ref.hub != nil
}

func (m *moduleRuntime) fanoutRef(v goja.Value) (*fanoutRef, bool) {
	ref, ok := m.getRef(v).(*fanoutRef)
	return ref, ok && ref != nil && ref.fanout != nil
}

func (m *moduleRuntime) resolvePrototype(fullName string) (proto.Message, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, fmt.Errorf("protobuf full name is empty")
	}
	if msg := m.prototypes[fullName]; msg != nil {
		return proto.Clone(msg), nil
	}
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(fullName))
	if err != nil {
		return nil, fmt.Errorf("resolve protobuf message %q: %w", fullName, err)
	}
	return mt.New().Interface(), nil
}

func (m *moduleRuntime) prototypeFromValue(value goja.Value) (proto.Message, string, bool) {
	if ref, ok := protogoja.MessagePrototypeFromValue(value); ok {
		msg := ref.NewMessage()
		if msg != nil {
			return msg, string(ref.TypeName()), true
		}
	}
	return nil, "", false
}
