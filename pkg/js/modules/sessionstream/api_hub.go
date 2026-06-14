package sessionstream

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

const defaultHubQueueDepth = 64

type hubQueue struct {
	m    *moduleRuntime
	hub  *ss.Hub
	jobs chan hubQueueJob
}

type hubQueueJob struct {
	ctx       context.Context
	id        string
	sessionID ss.SessionId
	name      string
	payload   proto.Message
}

func newHubQueue(m *moduleRuntime, hub *ss.Hub) *hubQueue {
	q := &hubQueue{m: m, hub: hub, jobs: make(chan hubQueueJob, defaultHubQueueDepth)}
	go q.run()
	return q
}

func (q *hubQueue) enqueue(ctx context.Context, sessionID ss.SessionId, name string, payload proto.Message) goja.Value {
	promise, resolve, reject := q.m.vm.NewPromise()
	if _, ok := runtimebridge.Lookup(q.m.vm); !ok {
		_ = reject(q.m.vm.ToValue("sessionstream enqueue requires runtime services"))
		return q.m.vm.ToValue(promise)
	}
	job := hubQueueJob{ctx: ctx, id: uuid.NewString(), sessionID: sessionID, name: name, payload: payload}
	select {
	case q.jobs <- job:
		_ = resolve(q.m.vm.ToValue(map[string]any{
			"accepted":  true,
			"id":        job.id,
			"sessionId": string(sessionID),
			"command":   name,
			"depth":     len(q.jobs),
		}))
	default:
		_ = reject(q.m.vm.ToValue(fmt.Sprintf("sessionstream enqueue queue full for command %q", name)))
	}
	return q.m.vm.ToValue(promise)
}

func (q *hubQueue) run() {
	for job := range q.jobs {
		if err := q.hub.Submit(job.ctx, job.sessionID, job.name, job.payload); err != nil {
			q.m.logger.Error().Err(err).Str("enqueue_id", job.id).Str("session_id", string(job.sessionID)).Str("command", job.name).Msg("sessionstream enqueue job failed")
		}
	}
}

func (m *moduleRuntime) hubBuilder(call goja.FunctionCall) goja.Value {
	var schemasValue goja.Value
	var fanoutValue goja.Value
	projectionPolicy := ""
	if arg := call.Argument(0); !goja.IsUndefined(arg) && !goja.IsNull(arg) {
		obj := arg.ToObject(m.vm)
		schemasValue = obj.Get("schemas")
		fanoutValue = obj.Get("fanout")
		if policy := obj.Get("projectionPolicy"); policy != nil && !goja.IsUndefined(policy) && !goja.IsNull(policy) {
			projectionPolicy = policy.String()
		}
	}
	registry := m.defaultSchemaRegistry
	if ref, ok := m.schemaRef(schemasValue); ok {
		registry = ref.registry
	}
	if registry == nil {
		registry = ss.NewSchemaRegistry()
	}
	hubOpts := []ss.HubOption{ss.WithSchemaRegistry(registry)}
	if m.defaultHydrationStore != nil {
		hubOpts = append(hubOpts, ss.WithHydrationStore(m.defaultHydrationStore))
	}
	if ref, ok := m.fanoutRef(fanoutValue); ok {
		hubOpts = append(hubOpts, ss.WithUIFanout(ref.fanout))
	}
	if projectionPolicy != "" {
		hubOpts = append(hubOpts, ss.WithProjectionErrorPolicy(m.projectionPolicy(projectionPolicy)))
	}
	hub, err := ss.NewHub(hubOpts...)
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
	return m.wrapHub(hub, registry)
}

func (m *moduleRuntime) projectionPolicy(policy string) ss.ProjectionErrorPolicy {
	switch policy {
	case "advance":
		return ss.ProjectionErrorPolicyAdvance
	case "fail", "":
		return ss.ProjectionErrorPolicyFail
	default:
		panic(m.vm.NewTypeError("unknown projectionPolicy %q", policy))
	}
}

func (m *moduleRuntime) wrapHub(hub *ss.Hub, schemas *ss.SchemaRegistry) goja.Value {
	obj := m.vm.NewObject()
	ref := &hubRef{hub: hub, schemas: schemas, queue: newHubQueue(m, hub)}
	m.attachRef(obj, ref)
	m.mustSet(obj, "submit", func(sessionID, name string, payload goja.Value) goja.Value {
		msg, err := m.jsValueToProto(schemas, schemaKindCommand, name, payload)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		callCtx := runtimebridge.CurrentOwnerContext(m.vm)
		return m.promiseFromGo(callCtx, "sessionstream.submit", func(ctx context.Context) error {
			return hub.Submit(ctx, ss.SessionId(sessionID), name, msg)
		})
	})
	m.mustSet(obj, "enqueue", func(sessionID, name string, payload goja.Value) goja.Value {
		msg, err := m.jsValueToProto(schemas, schemaKindCommand, name, payload)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return ref.queue.enqueue(runtimebridge.CurrentOwnerContext(m.vm), ss.SessionId(sessionID), name, msg)
	})
	m.mustSet(obj, "snapshot", func(sessionID string) goja.Value {
		snap, err := hub.Snapshot(runtimebridge.CurrentOwnerContext(m.vm), ss.SessionId(sessionID))
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		value, err := m.snapshotToJS(snap)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return value
	})
	m.mustSet(obj, "command", func(name string, handler goja.Value) goja.Value {
		fn, ok := goja.AssertFunction(handler)
		if !ok {
			panic(m.vm.NewTypeError("command handler must be a function"))
		}
		if err := hub.RegisterCommand(name, m.commandHandler(schemas, fn)); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "uiProjection", func(handler goja.Value) goja.Value {
		fn, ok := goja.AssertFunction(handler)
		if !ok {
			panic(m.vm.NewTypeError("uiProjection handler must be a function"))
		}
		if err := hub.RegisterUIProjection(ss.UIProjectionFunc(m.uiProjection(schemas, fn))); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "timelineProjection", func(handler goja.Value) goja.Value {
		fn, ok := goja.AssertFunction(handler)
		if !ok {
			panic(m.vm.NewTypeError("timelineProjection handler must be a function"))
		}
		if err := hub.RegisterTimelineProjection(ss.TimelineProjectionFunc(m.timelineProjection(schemas, fn))); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return obj
	})
	m.mustSet(obj, "run", func() goja.Value {
		if err := hub.Run(runtimebridge.CurrentOwnerContext(m.vm)); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	m.mustSet(obj, "shutdown", func() goja.Value {
		if err := hub.Shutdown(runtimebridge.CurrentOwnerContext(m.vm)); err != nil {
			panic(m.vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	return obj
}

func (m *moduleRuntime) snapshotToJS(snap ss.Snapshot) (goja.Value, error) {
	entities := make([]map[string]any, 0, len(snap.Entities))
	for _, ent := range snap.Entities {
		converted, err := m.timelineEntityToJS(ent)
		if err != nil {
			return nil, err
		}
		entities = append(entities, converted)
	}
	return m.vm.ToValue(map[string]any{
		"sessionId":       string(snap.SessionId),
		"snapshotOrdinal": strconv.FormatUint(snap.SnapshotOrdinal, 10),
		"entities":        entities,
	}), nil
}
