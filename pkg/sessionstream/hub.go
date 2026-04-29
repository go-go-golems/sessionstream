package sessionstream

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
)

// ProjectionErrorPolicy controls how event processing reacts to projection failures.
type ProjectionErrorPolicy int

const (
	// ProjectionErrorPolicyFail stops processing and returns the projection error.
	ProjectionErrorPolicyFail ProjectionErrorPolicy = iota
	// ProjectionErrorPolicyAdvance advances the relevant cursor even if a projection fails.
	ProjectionErrorPolicyAdvance
)

// ProjectionPolicies configures UI and timeline projection failure handling independently.
type ProjectionPolicies struct {
	UI       ProjectionErrorPolicy
	Timeline ProjectionErrorPolicy
}

// ErrorKind classifies framework-observed runtime errors.
type ErrorKind string

const (
	TimelineProjectorName                 = "timeline"
	ErrorKindDecode             ErrorKind = "decode"
	ErrorKindOrdinal            ErrorKind = "ordinal"
	ErrorKindUIProjection       ErrorKind = "ui-projection"
	ErrorKindTimelineProjection ErrorKind = "timeline-projection"
	ErrorKindFanout             ErrorKind = "fanout"
	ErrorKindStore              ErrorKind = "store"
)

// ErrorRecord describes an error observed while applying or delivering an event.
type ErrorRecord struct {
	Kind       ErrorKind
	SessionId  SessionId
	Ordinal    uint64
	EventName  string
	Err        error
	RawMessage []byte
	Metadata   map[string]string
}

// ErrorObserver receives best-effort runtime error notifications.
type ErrorObserver interface {
	OnSessionstreamError(ctx context.Context, rec ErrorRecord)
}

// ErrorObserverFunc adapts a function to ErrorObserver.
type ErrorObserverFunc func(ctx context.Context, rec ErrorRecord)

func (f ErrorObserverFunc) OnSessionstreamError(ctx context.Context, rec ErrorRecord) {
	f(ctx, rec)
}

// Hub is the substrate entrypoint.
type Hub struct {
	reg      *SchemaRegistry
	store    HydrationStore
	sessions *sessionRegistry
	commands *commandRegistry

	uiProjection       UIProjection
	timelineProjection TimelineProjection
	fanout             UIFanout
	bus                *busConfig

	projectionPolicies ProjectionPolicies
	errorObserver      ErrorObserver

	mu           sync.Mutex
	localOrdinal map[SessionId]uint64

	runMu     sync.Mutex
	runCancel context.CancelFunc
	consumer  *eventConsumer
}

// HubOption configures a Hub.
type HubOption func(*Hub) error

func WithSchemaRegistry(r *SchemaRegistry) HubOption {
	return func(h *Hub) error {
		if r == nil {
			return fmt.Errorf("schema registry is nil")
		}
		h.reg = r
		return nil
	}
}

func WithHydrationStore(s HydrationStore) HubOption {
	return func(h *Hub) error {
		if s == nil {
			return fmt.Errorf("hydration store is nil")
		}
		h.store = s
		return nil
	}
}

func WithSessionMetadataFactory(f SessionMetadataFactory) HubOption {
	return func(h *Hub) error {
		h.sessions = newSessionRegistry(f)
		return nil
	}
}

func WithProjectionPolicies(policies ProjectionPolicies) HubOption {
	return func(h *Hub) error {
		h.projectionPolicies = policies
		return nil
	}
}

func WithUIProjectionErrorPolicy(policy ProjectionErrorPolicy) HubOption {
	return func(h *Hub) error {
		h.projectionPolicies.UI = policy
		return nil
	}
}

func WithTimelineProjectionErrorPolicy(policy ProjectionErrorPolicy) HubOption {
	return func(h *Hub) error {
		h.projectionPolicies.Timeline = policy
		return nil
	}
}

func WithProjectionErrorPolicy(policy ProjectionErrorPolicy) HubOption {
	return WithProjectionPolicies(ProjectionPolicies{UI: policy, Timeline: policy})
}

func WithErrorObserver(observer ErrorObserver) HubOption {
	return func(h *Hub) error {
		h.errorObserver = observer
		return nil
	}
}

func WithUIFanout(f UIFanout) HubOption {
	return func(h *Hub) error {
		if f == nil {
			return fmt.Errorf("ui fanout is nil")
		}
		h.fanout = f
		return nil
	}
}

func NewHub(opts ...HubOption) (*Hub, error) {
	h := &Hub{
		reg:                NewSchemaRegistry(),
		store:              newNoopHydrationStore(),
		sessions:           newSessionRegistry(nil),
		commands:           newCommandRegistry(),
		projectionPolicies: ProjectionPolicies{UI: ProjectionErrorPolicyFail, Timeline: ProjectionErrorPolicyFail},
		localOrdinal:       map[SessionId]uint64{},
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(h); err != nil {
			return nil, err
		}
	}
	return h, nil
}

func (h *Hub) RegisterCommand(name string, handler CommandHandler) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	return h.commands.Register(name, handler)
}

func (h *Hub) RegisterUIProjection(p UIProjection) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	if p == nil {
		return fmt.Errorf("ui projection is nil")
	}
	if h.uiProjection != nil {
		return fmt.Errorf("ui projection already registered")
	}
	h.uiProjection = p
	return nil
}

func (h *Hub) RegisterTimelineProjection(p TimelineProjection) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	if p == nil {
		return fmt.Errorf("timeline projection is nil")
	}
	if h.timelineProjection != nil {
		return fmt.Errorf("timeline projection already registered")
	}
	h.timelineProjection = p
	return nil
}

// Submit executes a command through the configured publisher path.
func (h *Hub) Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	if payload == nil {
		return fmt.Errorf("submit payload for %q is nil", name)
	}
	if sid == "" {
		return fmt.Errorf("session id is empty")
	}
	if err := h.validatePayloadType(h.reg.commands, "command", name, payload); err != nil {
		return err
	}
	cmd := Command{Name: name, SessionId: sid, Payload: payload}
	return h.dispatch(ctx, cmd)
}

func (h *Hub) Snapshot(ctx context.Context, sid SessionId) (Snapshot, error) {
	if h == nil {
		return Snapshot{}, fmt.Errorf("hub is nil")
	}
	return h.store.Snapshot(ctx, sid, 0)
}

func (h *Hub) Session(ctx context.Context, sid SessionId) (*Session, error) {
	if h == nil {
		return nil, fmt.Errorf("hub is nil")
	}
	return h.sessions.GetOrCreate(ctx, sid)
}

func (h *Hub) Cursor(ctx context.Context, sid SessionId) (uint64, error) {
	if h == nil {
		return 0, fmt.Errorf("hub is nil")
	}
	return h.store.Cursor(ctx, sid)
}

func (h *Hub) EventCursor(ctx context.Context, sid SessionId) (uint64, error) {
	if h == nil {
		return 0, fmt.Errorf("hub is nil")
	}
	return h.eventCursor(ctx, sid)
}

func (h *Hub) ProjectionCursor(ctx context.Context, projector string, sid SessionId) (uint64, error) {
	if h == nil {
		return 0, fmt.Errorf("hub is nil")
	}
	if projector == "" {
		return 0, fmt.Errorf("projector is empty")
	}
	if projectionStore, ok := h.store.(ProjectionCursorStore); ok {
		return projectionStore.ProjectionCursor(ctx, projector, sid)
	}
	return h.store.Cursor(ctx, sid)
}

// Run starts the configured bus consumer if an event bus is present.
func (h *Hub) Run(ctx context.Context) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	if h.bus == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("ctx is nil")
	}

	h.runMu.Lock()
	defer h.runMu.Unlock()
	if h.runCancel != nil {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	consumer := newEventConsumer(h)
	if err := consumer.start(runCtx); err != nil {
		cancel()
		return err
	}
	h.runCancel = cancel
	h.consumer = consumer
	return nil
}

// Shutdown stops the active bus consumer.
func (h *Hub) Shutdown(ctx context.Context) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	h.runMu.Lock()
	cancel := h.runCancel
	consumer := h.consumer
	h.runCancel = nil
	h.consumer = nil
	h.runMu.Unlock()
	if cancel == nil || consumer == nil {
		return nil
	}
	cancel()
	return consumer.wait(ctx)
}

func (h *Hub) dispatch(ctx context.Context, cmd Command) error {
	handler, ok := h.commands.Lookup(cmd.Name)
	if !ok {
		return fmt.Errorf("unknown command %q", cmd.Name)
	}
	sess, err := h.sessions.GetOrCreate(ctx, cmd.SessionId)
	if err != nil {
		return err
	}
	return handler(ctx, cmd, sess, h.publisher())
}

func (h *Hub) publisher() EventPublisher {
	if h.bus != nil {
		return watermillEventPublisher{hub: h}
	}
	return localEventPublisher{hub: h}
}

type localEventPublisher struct {
	hub *Hub
}

func (p localEventPublisher) Publish(ctx context.Context, ev Event) error {
	if p.hub == nil {
		return fmt.Errorf("hub is nil")
	}
	if ev.SessionId == "" {
		return fmt.Errorf("event %q missing session id", ev.Name)
	}
	if ev.Payload == nil {
		return fmt.Errorf("event %q payload is nil", ev.Name)
	}
	if err := p.hub.validatePayloadType(p.hub.reg.events, "event", ev.Name, ev.Payload); err != nil {
		return err
	}
	ord, err := p.hub.nextLocalOrdinal(ctx, ev.SessionId)
	if err != nil {
		return err
	}
	ev.Ordinal = ord
	_, err = p.hub.projectAndApply(ctx, ev)
	return err
}

func (h *Hub) RetryTimeline(ctx context.Context, sid SessionId) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	cursor, err := h.ProjectionCursor(ctx, TimelineProjectorName, sid)
	if err != nil {
		return err
	}
	return h.RebuildTimeline(ctx, sid, cursor)
}

func (h *Hub) RebuildTimelineFromScratch(ctx context.Context, sid SessionId) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	resetStore, ok := h.store.(TimelineResetStore)
	if !ok {
		return fmt.Errorf("hydration store does not support timeline reset")
	}
	if err := resetStore.ClearTimeline(ctx, sid); err != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: sid, Err: err})
		return err
	}
	return h.RebuildTimeline(ctx, sid, 0)
}

func (h *Hub) RebuildTimeline(ctx context.Context, sid SessionId, from uint64) error {
	if h == nil {
		return fmt.Errorf("hub is nil")
	}
	if sid == "" {
		return fmt.Errorf("session id is empty")
	}
	if h.timelineProjection == nil {
		return fmt.Errorf("timeline projection is not registered")
	}
	eventStore, ok := h.store.(EventStore)
	if !ok {
		return fmt.Errorf("hydration store does not support event replay")
	}
	for {
		events, err := eventStore.Events(ctx, sid, from, 100)
		if err != nil {
			h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: sid, Ordinal: from, Err: err})
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for _, ev := range events {
			if err := h.rebuildTimelineEvent(ctx, ev); err != nil {
				return err
			}
			from = ev.Ordinal
		}
	}
}

func (h *Hub) rebuildTimelineEvent(ctx context.Context, ev Event) error {
	sess, err := h.sessions.GetOrCreate(ctx, ev.SessionId)
	if err != nil {
		return err
	}
	view, err := h.store.View(ctx, ev.SessionId)
	if err != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
		return err
	}
	entities, err := h.timelineProjection.Project(ctx, ev, sess, view)
	if err != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindTimelineProjection, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
		return err
	}
	if err := h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entities); err != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
		return err
	}
	if projectionStore, ok := h.store.(ProjectionCursorStore); ok {
		if err := projectionStore.AdvanceProjectionCursor(ctx, TimelineProjectorName, ev.SessionId, ev.Ordinal); err != nil {
			h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
			return err
		}
	}
	return nil
}

func (h *Hub) projectAndApply(ctx context.Context, ev Event) ([]UIEvent, error) {
	if eventStore, ok := h.store.(EventStore); ok {
		if err := eventStore.AppendEvent(ctx, ev); err != nil {
			h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
			return nil, err
		}
	}

	sess, err := h.sessions.GetOrCreate(ctx, ev.SessionId)
	if err != nil {
		return nil, err
	}
	view, err := h.store.View(ctx, ev.SessionId)
	if err != nil {
		return nil, err
	}

	var (
		uiEvents []UIEvent
		entities []TimelineEntity
		uiErr    error
		tlErr    error
	)
	if h.uiProjection != nil {
		uiEvents, uiErr = h.uiProjection.Project(ctx, ev, sess, view)
	}
	if h.timelineProjection != nil {
		entities, tlErr = h.timelineProjection.Project(ctx, ev, sess, view)
	}

	if uiErr != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindUIProjection, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: uiErr})
		if h.projectionPolicies.UI == ProjectionErrorPolicyFail {
			return nil, uiErr
		}
	}
	if tlErr != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindTimelineProjection, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: tlErr})
		if h.projectionPolicies.Timeline == ProjectionErrorPolicyFail {
			return nil, tlErr
		}
	}

	entitiesToApply := entities
	if tlErr != nil {
		entitiesToApply = nil
	}
	if err := h.store.Apply(ctx, ev.SessionId, ev.Ordinal, entitiesToApply); err != nil {
		h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
		return nil, err
	}
	if tlErr == nil {
		if projectionStore, ok := h.store.(ProjectionCursorStore); ok {
			if err := projectionStore.AdvanceProjectionCursor(ctx, TimelineProjectorName, ev.SessionId, ev.Ordinal); err != nil {
				h.reportError(ctx, ErrorRecord{Kind: ErrorKindStore, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
				return nil, err
			}
		}
	}
	if uiErr == nil && h.fanout != nil && len(uiEvents) > 0 {
		if err := h.fanout.PublishUI(ctx, ev.SessionId, ev.Ordinal, cloneUIEvents(uiEvents)); err != nil {
			h.reportError(ctx, ErrorRecord{Kind: ErrorKindFanout, SessionId: ev.SessionId, Ordinal: ev.Ordinal, EventName: ev.Name, Err: err})
			return nil, err
		}
	}
	return uiEvents, nil
}

func cloneUIEvents(in []UIEvent) []UIEvent {
	if len(in) == 0 {
		return nil
	}
	out := make([]UIEvent, 0, len(in))
	for _, event := range in {
		clonedEvent := event
		if event.Payload != nil {
			clonedEvent.Payload = proto.Clone(event.Payload)
		}
		out = append(out, clonedEvent)
	}
	return out
}

func (h *Hub) nextLocalOrdinal(ctx context.Context, sid SessionId) (uint64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	current, ok := h.localOrdinal[sid]
	if !ok {
		cursor, err := h.eventCursor(ctx, sid)
		if err != nil {
			return 0, err
		}
		current = cursor
	}
	current++
	h.localOrdinal[sid] = current
	return current, nil
}

func (h *Hub) eventCursor(ctx context.Context, sid SessionId) (uint64, error) {
	if eventStore, ok := h.store.(EventStore); ok {
		return eventStore.EventCursor(ctx, sid)
	}
	return h.store.Cursor(ctx, sid)
}

func (h *Hub) reportError(ctx context.Context, rec ErrorRecord) {
	if errorStore, ok := h.store.(ErrorStore); ok {
		_ = errorStore.RecordError(ctx, rec)
	}
	if h.errorObserver != nil {
		h.errorObserver.OnSessionstreamError(ctx, rec)
	}
}

func (h *Hub) validatePayloadType(m map[string]proto.Message, kind, name string, payload proto.Message) error {
	prototype, ok := h.reg.lookup(m, name)
	if !ok {
		return fmt.Errorf("unknown %s %q", kind, name)
	}
	if prototype.ProtoReflect().Descriptor().FullName() != payload.ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("%s %q payload type mismatch: got %s want %s", kind, name, payload.ProtoReflect().Descriptor().FullName(), prototype.ProtoReflect().Descriptor().FullName())
	}
	return nil
}
