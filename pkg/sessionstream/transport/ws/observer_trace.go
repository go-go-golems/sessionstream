package ws

import (
	"context"
	"fmt"
	"runtime/trace"
	"sync"
	"sync/atomic"
)

const ObserverTraceSchemaVersion = 1

// ObserverTraceSink receives refinement events for the bounded observer
// dispatcher. Implementations must return promptly and must not call back into
// Server. A nil sink disables all observer refinement instrumentation.
type ObserverTraceSink interface {
	OnObserverModelEvent(ObserverModelEvent)
	OnObserverIntervalEvent(ObserverIntervalEvent)
}

// ObserverTraceConfig enables refinement tracing for one dispatcher instance.
// RunID groups one harvested execution; DispatcherID identifies this observer
// dispatcher within that run. Both IDs are required when Sink is non-nil.
type ObserverTraceConfig struct {
	RunID        string
	DispatcherID string
	Sink         ObserverTraceSink
}

// ObserverModelEvent records one abstract transition and partial state update.
// OperationID links it to invocation/return bounds in ObserverIntervalEvent.
type ObserverModelEvent struct {
	SchemaVersion int            `json:"schema_version"`
	RunID         string         `json:"run_id"`
	DispatcherID  string         `json:"dispatcher_id"`
	Sequence      uint64         `json:"sequence"`
	OperationID   string         `json:"operation_id"`
	Operation     string         `json:"operation"`
	Action        string         `json:"action"`
	ItemID        uint64         `json:"item_id,omitempty"`
	Updates       map[string]any `json:"updates,omitempty"`
	Evidence      map[string]any `json:"evidence,omitempty"`
}

// ObserverIntervalEvent bounds one operation. Linearize events share an
// OperationID with ObserverModelEvent; sequence orders observations only, not
// the abstract operations themselves.
type ObserverIntervalEvent struct {
	SchemaVersion int    `json:"schema_version"`
	RunID         string `json:"run_id"`
	DispatcherID  string `json:"dispatcher_id"`
	Sequence      uint64 `json:"sequence"`
	OperationID   string `json:"operation_id"`
	Operation     string `json:"operation"`
	Phase         string `json:"phase"`
	Action        string `json:"action,omitempty"`
}

type observerTraceState struct {
	config       ObserverTraceConfig
	emitMu       sync.Mutex
	modelSeq     uint64
	intervalSeq  uint64
	operationSeq atomic.Uint64
	itemSeq      atomic.Uint64
	runtimeCtx   context.Context
	runtimeTask  *trace.Task
	finishOnce   sync.Once
}

type observerTraceOperation struct {
	state     *observerTraceState
	id        string
	operation string
	ctx       context.Context
	region    *trace.Region
}

func WithObserverTrace(config ObserverTraceConfig) Option {
	return func(s *Server) error {
		if config.Sink == nil {
			s.observerTrace = nil
			return nil
		}
		if config.RunID == "" {
			return fmt.Errorf("observer trace run ID is empty")
		}
		if config.DispatcherID == "" {
			return fmt.Errorf("observer trace dispatcher ID is empty")
		}
		s.observerTrace = &observerTraceState{config: config}
		return nil
	}
}

func (t *observerTraceState) start(ctx context.Context) {
	if t == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	t.runtimeCtx, t.runtimeTask = trace.NewTask(ctx, "ws.observer_dispatcher")
	trace.Logf(t.runtimeCtx, "observer.identity", "run=%s dispatcher=%s", t.config.RunID, t.config.DispatcherID)
}

func (t *observerTraceState) finish() {
	if t == nil {
		return
	}
	t.finishOnce.Do(func() {
		if t.runtimeTask != nil {
			t.runtimeTask.End()
		}
	})
}

func (t *observerTraceState) begin(operation string) observerTraceOperation {
	if t == nil {
		return observerTraceOperation{}
	}
	id := fmt.Sprintf("op-%d", t.operationSeq.Add(1))
	ctx := t.runtimeCtx
	if ctx == nil {
		ctx = context.Background()
	}
	region := trace.StartRegion(ctx, "observer."+operation)
	t.emitInterval(id, operation, "invoke", "")
	trace.Logf(ctx, "observer.operation", "id=%s phase=invoke operation=%s", id, operation)
	return observerTraceOperation{state: t, id: id, operation: operation, ctx: ctx, region: region}
}

func (op observerTraceOperation) linearize(action string, itemID uint64, updates, evidence map[string]any) {
	if op.state == nil {
		return
	}
	op.state.emitMu.Lock()
	op.state.modelSeq++
	op.state.config.Sink.OnObserverModelEvent(ObserverModelEvent{
		SchemaVersion: ObserverTraceSchemaVersion,
		RunID:         op.state.config.RunID,
		DispatcherID:  op.state.config.DispatcherID,
		Sequence:      op.state.modelSeq,
		OperationID:   op.id,
		Operation:     op.operation,
		Action:        action,
		ItemID:        itemID,
		Updates:       updates,
		Evidence:      evidence,
	})
	op.state.emitMu.Unlock()
	op.state.emitInterval(op.id, op.operation, "linearize", action)
	trace.Logf(op.ctx, "observer.linearize", "id=%s action=%s item=%d", op.id, action, itemID)
}

func (op observerTraceOperation) end() {
	if op.state == nil {
		return
	}
	op.state.emitInterval(op.id, op.operation, "return", "")
	trace.Logf(op.ctx, "observer.operation", "id=%s phase=return operation=%s", op.id, op.operation)
	if op.region != nil {
		op.region.End()
	}
}

func (op observerTraceOperation) cancel() {
	if op.state == nil {
		return
	}
	op.state.emitInterval(op.id, op.operation, "cancel", "")
	trace.Logf(op.ctx, "observer.operation", "id=%s phase=cancel operation=%s", op.id, op.operation)
	if op.region != nil {
		op.region.End()
	}
}

func (t *observerTraceState) emitInterval(operationID, operation, phase, action string) {
	t.emitMu.Lock()
	defer t.emitMu.Unlock()
	t.intervalSeq++
	t.config.Sink.OnObserverIntervalEvent(ObserverIntervalEvent{
		SchemaVersion: ObserverTraceSchemaVersion,
		RunID:         t.config.RunID,
		DispatcherID:  t.config.DispatcherID,
		Sequence:      t.intervalSeq,
		OperationID:   operationID,
		Operation:     operation,
		Phase:         phase,
		Action:        action,
	})
}

func (t *observerTraceState) nextItemID() uint64 {
	if t == nil {
		return 0
	}
	return t.itemSeq.Add(1)
}
