package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	sessionstreamv1 "github.com/go-go-golems/sessionstream/pkg/sessionstream/pb/proto/sessionstream/v1"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	marshalOptions = protojson.MarshalOptions{EmitUnpopulated: false, UseProtoNames: false}
	unmarshalOpts  = protojson.UnmarshalOptions{DiscardUnknown: false}
)

const defaultMaxHydrationBufferedBatches = 1024

// SnapshotProvider provides snapshot lookup for subscribe flows.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error)
}

// Hooks observes websocket lifecycle and payload sequencing for debugging and labs.
type Hooks struct {
	OnUpgradeError  func(*http.Request, error)
	OnConnect       func(sessionstream.ConnectionId)
	OnDisconnect    func(sessionstream.ConnectionId)
	OnReadError     func(sessionstream.ConnectionId, error)
	OnWriteError    func(sessionstream.ConnectionId, error)
	OnSendError     func(sessionstream.ConnectionId, error)
	OnProtocolError func(sessionstream.ConnectionId, error)
	OnSubscribe     func(sessionstream.ConnectionId, sessionstream.SessionId, uint64)
	OnUnsubscribe   func(sessionstream.ConnectionId, sessionstream.SessionId)
	OnSnapshotSent  func(sessionstream.ConnectionId, sessionstream.SessionId, sessionstream.Snapshot)
	OnUIEventSent   func(sessionstream.ConnectionId, sessionstream.SessionId, uint64, sessionstream.UIEvent)
	OnClientFrame   func(sessionstream.ConnectionId, map[string]any)
}

// Option configures a websocket Server.
type Option func(*Server) error

// WithHooks installs optional lifecycle hooks.
func WithHooks(h Hooks) Option {
	return func(s *Server) error {
		s.hooks = h
		return nil
	}
}

// WithUpgrader overrides the default websocket upgrader.
func WithUpgrader(u websocket.Upgrader) Option {
	return func(s *Server) error {
		s.upgrader = u
		return nil
	}
}

// WithHydrationBufferLimit configures how many live UI batches a hydrating
// subscription may buffer while its snapshot is loading. A non-positive value
// disables buffering and is rejected.
func WithHydrationBufferLimit(maxBatches int) Option {
	return func(s *Server) error {
		if maxBatches <= 0 {
			return fmt.Errorf("hydration buffer limit must be positive")
		}
		s.maxHydrationBufferedBatches = maxBatches
		return nil
	}
}

// Server is a websocket snapshot/fanout adapter. It is both an HTTP handler
// and a sessionstream.UIFanout: clients may subscribe/unsubscribe to sessions,
// receive snapshots, and receive live UI events. Command ingress is deliberately
// out of scope for this adapter.
//
// Subscribe always sends a current snapshot followed by future live UI fanout.
// sinceSnapshotOrdinal is accepted, stored, echoed, and surfaced to hooks for teaching
// and diagnostics, but it is advisory for now: this reference adapter does not
// replay missed UI events from the event store. Replayed event history belongs
// behind an explicit replay API rather than being hidden inside websocket
// subscribe semantics.
//
// Production callers should wrap this handler with their own authentication,
// authorization, origin policy, and rate limiting. The default upgrader is
// intentionally permissive for local labs and examples; use WithUpgrader to
// install a stricter CheckOrigin policy.
type Server struct {
	snapshots SnapshotProvider
	upgrader  websocket.Upgrader
	hooks     Hooks
	observer  TransportObserver

	maxHydrationBufferedBatches int

	nextID uint64

	mu        sync.RWMutex
	conns     map[sessionstream.ConnectionId]*connection
	bySession map[sessionstream.SessionId]map[sessionstream.ConnectionId]struct{}
}

type connection struct {
	id     sessionstream.ConnectionId
	ws     *websocket.Conn
	send   chan outboundFrame
	close  sync.Once
	closed atomic.Bool

	mu   sync.RWMutex
	subs map[sessionstream.SessionId]subscription
}

type subscriptionState string

const (
	subscriptionStateHydrating subscriptionState = "hydrating"
	subscriptionStateLive      subscriptionState = "live"
)

type bufferedUIBatch struct {
	ordinal uint64
	events  []sessionstream.UIEvent
}

type subscription struct {
	sinceSnapshotOrdinal uint64
	state                subscriptionState
	snapshotOrdinal      uint64
	buffer               []bufferedUIBatch
}

type outboundFrame struct {
	body      []byte
	frameType string
}

// ConnectionInfo describes the current transport-visible state of one connection.
type ConnectionInfo struct {
	ConnectionId  string   `json:"connectionId"`
	Subscriptions []string `json:"subscriptions"`
}

var _ http.Handler = (*Server)(nil)
var _ sessionstream.UIFanout = (*Server)(nil)

// NewServer builds a websocket transport server.
func NewServer(snapshots SnapshotProvider, opts ...Option) (*Server, error) {
	if snapshots == nil {
		return nil, fmt.Errorf("snapshot provider is nil")
	}
	server := &Server{
		snapshots:                   snapshots,
		maxHydrationBufferedBatches: defaultMaxHydrationBufferedBatches,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		conns:     map[sessionstream.ConnectionId]*connection{},
		bySession: map[sessionstream.SessionId]map[sessionstream.ConnectionId]struct{}{},
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(server); err != nil {
			return nil, err
		}
	}
	return server, nil
}

// ServeHTTP upgrades a connection and serves the websocket protocol.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.observe(r.Context(), TransportRecord{Stage: TransportStageUpgradeError, Err: err})
		if s.hooks.OnUpgradeError != nil {
			s.hooks.OnUpgradeError(r, err)
		}
		return
	}
	cid := sessionstream.ConnectionId(fmt.Sprintf("conn-%d", atomic.AddUint64(&s.nextID, 1)))
	c := &connection{
		id:   cid,
		ws:   conn,
		send: make(chan outboundFrame, 128),
		subs: map[sessionstream.SessionId]subscription{},
	}
	s.mu.Lock()
	s.conns[cid] = c
	s.mu.Unlock()
	s.observe(r.Context(), TransportRecord{Stage: TransportStageConnected, ConnectionId: cid})
	if s.hooks.OnConnect != nil {
		s.hooks.OnConnect(cid)
	}

	go s.writeLoop(c)
	_ = s.sendFrame(c, newHelloFrame(cid))
	s.readLoop(r.Context(), c)
	s.closeConnection(c)
}

// PublishUI fans projected UI events out to subscribed websocket clients.
func (s *Server) PublishUI(ctx context.Context, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	if len(events) == 0 {
		return nil
	}
	targets := s.connectionsForSession(sid)
	targetIDs := fanoutTargetIDs(targets)
	if len(targets) == 0 {
		s.observe(ctx, TransportRecord{Stage: TransportStageFanoutNoTargets, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events)})
		return nil
	}
	s.observe(ctx, TransportRecord{Stage: TransportStageFanoutStarted, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events), FanoutTargetIds: targetIDs})
	for _, c := range targets {
		state, ok := s.subscriptionState(c, sid)
		if !ok {
			continue
		}
		switch state {
		case subscriptionStateHydrating:
			if err := s.bufferHydrationEvents(ctx, c, sid, ord, events); err != nil {
				_ = s.sendFrame(c, newErrorFrame("hydration_buffer_overflow", err.Error(), ""))
				s.closeConnection(c)
			}
		case subscriptionStateLive:
			if err := s.sendUIBatch(c, sid, ord, events); err != nil {
				s.closeConnection(c)
			}
		}
	}
	s.observe(ctx, TransportRecord{Stage: TransportStageFanoutCompleted, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events), FanoutTargetIds: targetIDs})
	return nil
}

// Connections returns a stable snapshot of current connections and subscriptions.
func (s *Server) Connections() []ConnectionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ConnectionInfo, 0, len(s.conns))
	for id, conn := range s.conns {
		conn.mu.RLock()
		subs := make([]string, 0, len(conn.subs))
		for sid := range conn.subs {
			subs = append(subs, string(sid))
		}
		conn.mu.RUnlock()
		sort.Strings(subs)
		out = append(out, ConnectionInfo{ConnectionId: string(id), Subscriptions: subs})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ConnectionId < out[j].ConnectionId })
	return out
}

func (s *Server) readLoop(ctx context.Context, c *connection) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageReadError, Direction: FrameDirectionClientToServer, ConnectionId: c.id, Err: err})
			if s.hooks.OnReadError != nil {
				s.hooks.OnReadError(c.id, err)
			}
			return
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageClientFrameRead, Direction: FrameDirectionClientToServer, ConnectionId: c.id, RawBytes: len(raw)})
		frame := &sessionstreamv1.ClientFrame{}
		if err := unmarshalOpts.Unmarshal(raw, frame); err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageClientFrameDecodeError, Direction: FrameDirectionClientToServer, ConnectionId: c.id, RawBytes: len(raw), Err: err})
			if s.hooks.OnProtocolError != nil {
				s.hooks.OnProtocolError(c.id, err)
			}
			if sendErr := s.sendFrame(c, newErrorFrame("bad_client_frame", err.Error(), "")); sendErr != nil && s.hooks.OnSendError != nil {
				s.hooks.OnSendError(c.id, sendErr)
			}
			continue
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageClientFrameDecoded, Direction: FrameDirectionClientToServer, ConnectionId: c.id, FrameType: clientFrameType(frame), RawBytes: len(raw)})
		if s.hooks.OnClientFrame != nil {
			s.hooks.OnClientFrame(c.id, protoMessageAsMap(frame))
		}
		if err := s.handleClientFrame(ctx, c, frame); err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageProtocolError, ConnectionId: c.id, FrameType: clientFrameType(frame), Err: err})
			if s.hooks.OnProtocolError != nil {
				s.hooks.OnProtocolError(c.id, err)
			}
			if sendErr := s.sendFrame(c, newErrorFrame("protocol_error", err.Error(), "")); sendErr != nil && s.hooks.OnSendError != nil {
				s.hooks.OnSendError(c.id, sendErr)
			}
		}
	}
}

func (s *Server) handleClientFrame(ctx context.Context, c *connection, frame *sessionstreamv1.ClientFrame) error {
	switch typed := frame.GetFrame().(type) {
	case *sessionstreamv1.ClientFrame_Ping:
		return s.sendFrame(c, newPongFrame(typed.Ping.GetNonce()))
	case *sessionstreamv1.ClientFrame_Pong:
		return nil
	case *sessionstreamv1.ClientFrame_Subscribe:
		sub := typed.Subscribe
		sid := sessionstream.SessionId(sub.GetSessionId())
		if sid == "" {
			return fmt.Errorf("subscribe missing session id")
		}
		since := sub.GetSinceSnapshotOrdinal()
		s.observe(ctx, TransportRecord{Stage: TransportStageSubscribeReceived, Direction: FrameDirectionClientToServer, ConnectionId: c.id, SessionId: sid, FrameType: "subscribe", SinceSnapshotOrdinal: since})
		s.registerHydrating(c, sid, since)
		completed := false
		defer func() {
			if !completed {
				s.removeSubscription(c, sid)
			}
		}()

		s.observe(ctx, TransportRecord{Stage: TransportStageSnapshotLoadStarted, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since})
		snap, err := s.snapshots.Snapshot(ctx, sid)
		if err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageProtocolError, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since, Err: err})
			return fmt.Errorf("load snapshot for %q: %w", sid, err)
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageSnapshotLoaded, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since, SnapshotOrdinal: snap.SnapshotOrdinal, SnapshotEntityCount: len(snap.Entities), SnapshotEntities: summarizeEntities(snap.Entities)})
		if err := s.sendFrame(c, newSnapshotFrame(sid, snap)); err != nil {
			return err
		}
		if s.hooks.OnSnapshotSent != nil {
			s.hooks.OnSnapshotSent(c.id, sid, cloneSnapshot(snap))
		}
		buffered := s.drainHydrationBuffer(c, sid, snap.SnapshotOrdinal)
		if len(buffered) > 0 {
			s.observe(ctx, TransportRecord{Stage: TransportStageHydrationBufferFlushed, ConnectionId: c.id, SessionId: sid, SnapshotOrdinal: snap.SnapshotOrdinal, FanoutEventCount: countBufferedEvents(buffered)})
		}
		for _, batch := range buffered {
			if err := s.sendUIBatch(c, sid, batch.ordinal, batch.events); err != nil {
				return err
			}
		}
		s.markLive(c, sid, snap.SnapshotOrdinal)
		if s.hooks.OnSubscribe != nil {
			s.hooks.OnSubscribe(c.id, sid, since)
		}
		if err := s.sendFrame(c, newSubscribedFrame(sid, since)); err != nil {
			return err
		}
		completed = true
		return nil
	case *sessionstreamv1.ClientFrame_Unsubscribe:
		sid := sessionstream.SessionId(typed.Unsubscribe.GetSessionId())
		if sid == "" {
			return fmt.Errorf("unsubscribe missing session id")
		}
		s.removeSubscription(c, sid)
		if s.hooks.OnUnsubscribe != nil {
			s.hooks.OnUnsubscribe(c.id, sid)
		}
		return s.sendFrame(c, newUnsubscribedFrame(sid))
	default:
		return fmt.Errorf("unknown client frame")
	}
}

func (s *Server) writeLoop(c *connection) {
	for msg := range c.send {
		if err := c.ws.WriteMessage(websocket.TextMessage, msg.body); err != nil {
			s.observe(context.Background(), TransportRecord{Stage: TransportStageServerFrameWriteError, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: msg.frameType, RawBytes: len(msg.body), Err: err})
			if s.hooks.OnWriteError != nil {
				s.hooks.OnWriteError(c.id, err)
			}
			return
		}
		s.observe(context.Background(), TransportRecord{Stage: TransportStageServerFrameWritten, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: msg.frameType, RawBytes: len(msg.body)})
	}
}

func (s *Server) closeConnection(c *connection) {
	if c == nil {
		return
	}
	c.close.Do(func() {
		c.mu.Lock()
		subs := make([]sessionstream.SessionId, 0, len(c.subs))
		for sid := range c.subs {
			subs = append(subs, sid)
		}
		c.subs = map[sessionstream.SessionId]subscription{}
		c.mu.Unlock()

		s.mu.Lock()
		delete(s.conns, c.id)
		for _, sid := range subs {
			delete(s.bySession[sid], c.id)
			if len(s.bySession[sid]) == 0 {
				delete(s.bySession, sid)
			}
		}
		s.mu.Unlock()

		c.closed.Store(true)
		close(c.send)
		_ = c.ws.Close()
		s.observe(context.Background(), TransportRecord{Stage: TransportStageDisconnected, ConnectionId: c.id})
		if s.hooks.OnDisconnect != nil {
			s.hooks.OnDisconnect(c.id)
		}
	})
}

func (s *Server) connectionsForSession(sid sessionstream.SessionId) []*connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := s.bySession[sid]
	if len(set) == 0 {
		return nil
	}
	out := make([]*connection, 0, len(set))
	for cid := range set {
		if conn := s.conns[cid]; conn != nil {
			out = append(out, conn)
		}
	}
	return out
}

func (s *Server) registerHydrating(c *connection, sid sessionstream.SessionId, since uint64) {
	c.mu.Lock()
	c.subs[sid] = subscription{sinceSnapshotOrdinal: since, state: subscriptionStateHydrating}
	c.mu.Unlock()

	s.mu.Lock()
	set := s.bySession[sid]
	if set == nil {
		set = map[sessionstream.ConnectionId]struct{}{}
		s.bySession[sid] = set
	}
	set[c.id] = struct{}{}
	s.mu.Unlock()
	s.observe(context.Background(), TransportRecord{Stage: TransportStageSubscriptionRegistered, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since})
}

func (s *Server) subscriptionState(c *connection, sid sessionstream.SessionId) (subscriptionState, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	sub, ok := c.subs[sid]
	return sub.state, ok
}

func (s *Server) bufferHydrationEvents(ctx context.Context, c *connection, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subs[sid]
	if !ok || sub.state != subscriptionStateHydrating {
		return nil
	}
	if len(sub.buffer) >= s.maxHydrationBufferedBatches {
		err := fmt.Errorf("connection %s hydration buffer full for session %s", c.id, sid)
		s.observe(ctx, TransportRecord{Stage: TransportStageHydrationBufferOverflow, ConnectionId: c.id, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events), Err: err})
		return err
	}
	sub.buffer = append(sub.buffer, bufferedUIBatch{ordinal: ord, events: cloneUIEvents(events)})
	c.subs[sid] = sub
	s.observe(ctx, TransportRecord{Stage: TransportStageUIEventBuffered, ConnectionId: c.id, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events)})
	return nil
}

func (s *Server) drainHydrationBuffer(c *connection, sid sessionstream.SessionId, snapshotOrdinal uint64) []bufferedUIBatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subs[sid]
	if !ok {
		return nil
	}
	out := make([]bufferedUIBatch, 0, len(sub.buffer))
	for _, batch := range sub.buffer {
		if batch.ordinal > snapshotOrdinal {
			out = append(out, cloneBufferedUIBatch(batch))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ordinal < out[j].ordinal })
	sub.buffer = nil
	sub.snapshotOrdinal = snapshotOrdinal
	c.subs[sid] = sub
	return out
}

func (s *Server) markLive(c *connection, sid sessionstream.SessionId, snapshotOrdinal uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subs[sid]
	if !ok {
		return
	}
	sub.state = subscriptionStateLive
	sub.snapshotOrdinal = snapshotOrdinal
	sub.buffer = nil
	c.subs[sid] = sub
	s.observe(context.Background(), TransportRecord{Stage: TransportStageSubscriptionLive, ConnectionId: c.id, SessionId: sid, SnapshotOrdinal: snapshotOrdinal})
}

func (s *Server) sendUIBatch(c *connection, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	for _, event := range events {
		frame, err := newUIEventFrame(sid, ord, event)
		if err != nil {
			if s.hooks.OnSendError != nil {
				s.hooks.OnSendError(c.id, err)
			}
			return err
		}
		if err := s.sendFrame(c, frame); err != nil {
			return err
		}
		if s.hooks.OnUIEventSent != nil {
			s.hooks.OnUIEventSent(c.id, sid, ord, cloneUIEvent(event))
		}
	}
	return nil
}

func cloneBufferedUIBatch(batch bufferedUIBatch) bufferedUIBatch {
	return bufferedUIBatch{ordinal: batch.ordinal, events: cloneUIEvents(batch.events)}
}

func countBufferedEvents(batches []bufferedUIBatch) int {
	total := 0
	for _, batch := range batches {
		total += len(batch.events)
	}
	return total
}

func (s *Server) removeSubscription(c *connection, sid sessionstream.SessionId) {
	if c == nil || sid == "" {
		return
	}
	c.mu.Lock()
	delete(c.subs, sid)
	c.mu.Unlock()
	s.mu.Lock()
	delete(s.bySession[sid], c.id)
	if len(s.bySession[sid]) == 0 {
		delete(s.bySession, sid)
	}
	s.mu.Unlock()
}

func (s *Server) sendFrame(c *connection, frame *sessionstreamv1.ServerFrame) (err error) {
	frameType := serverFrameType(frame)
	if c == nil {
		return fmt.Errorf("connection is nil")
	}
	if c.closed.Load() {
		return fmt.Errorf("connection %s is closed", c.id)
	}
	body, err := marshalOptions.Marshal(frame)
	if err != nil {
		s.observe(context.Background(), TransportRecord{Stage: TransportStageServerFrameMarshalError, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: frameType, Err: err})
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("connection %s is closed", c.id)
		}
	}()
	queueLen := len(c.send)
	queueCap := cap(c.send)
	select {
	case c.send <- outboundFrame{body: body, frameType: frameType}:
		s.observe(context.Background(), TransportRecord{Stage: TransportStageServerFrameQueued, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: frameType, RawBytes: len(body), QueueLen: queueLen, QueueCap: queueCap})
		return nil
	default:
		if c.closed.Load() {
			return fmt.Errorf("connection %s is closed", c.id)
		}
		err := fmt.Errorf("connection %s send buffer full", c.id)
		s.observe(context.Background(), TransportRecord{Stage: TransportStageServerFrameQueueFull, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: frameType, RawBytes: len(body), QueueLen: queueLen, QueueCap: queueCap, Err: err})
		return err
	}
}

func newHelloFrame(cid sessionstream.ConnectionId) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Hello{Hello: &sessionstreamv1.HelloFrame{ConnectionId: string(cid)}}}
}

func newSubscribedFrame(sid sessionstream.SessionId, since uint64) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Subscribed{Subscribed: &sessionstreamv1.SubscribedFrame{SessionId: string(sid), SinceSnapshotOrdinal: since}}}
}

func newUnsubscribedFrame(sid sessionstream.SessionId) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Unsubscribed{Unsubscribed: &sessionstreamv1.UnsubscribedFrame{SessionId: string(sid)}}}
}

func newSnapshotFrame(sid sessionstream.SessionId, snap sessionstream.Snapshot) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Snapshot{Snapshot: &sessionstreamv1.SnapshotFrame{SessionId: string(sid), SnapshotOrdinal: snap.SnapshotOrdinal, Entities: encodeSnapshotEntities(snap.Entities)}}}
}

func newUIEventFrame(sid sessionstream.SessionId, ord uint64, event sessionstream.UIEvent) (*sessionstreamv1.ServerFrame, error) {
	payload, err := packAny(event.Payload)
	if err != nil {
		return nil, err
	}
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_UiEvent{UiEvent: &sessionstreamv1.UiEventFrame{SessionId: string(sid), EventOrdinal: ord, Name: event.Name, Payload: payload}}}, nil
}

func newErrorFrame(code, message, sessionID string) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Error{Error: &sessionstreamv1.ErrorFrame{Code: code, Message: message, SessionId: sessionID}}}
}

func newPongFrame(nonce string) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Pong{Pong: &sessionstreamv1.PongFrame{Nonce: nonce}}}
}

func encodeSnapshotEntities(in []sessionstream.TimelineEntity) []*sessionstreamv1.SnapshotEntity {
	if len(in) == 0 {
		return []*sessionstreamv1.SnapshotEntity{}
	}
	out := make([]*sessionstreamv1.SnapshotEntity, 0, len(in))
	for _, entity := range in {
		out = append(out, &sessionstreamv1.SnapshotEntity{
			Kind:             entity.Kind,
			Id:               entity.Id,
			CreatedOrdinal:   entity.CreatedOrdinal,
			LastEventOrdinal: entity.LastEventOrdinal,
			Tombstone:        entity.Tombstone,
			Payload:          mustPackAny(entity.Payload),
		})
	}
	return out
}

func packAny(msg proto.Message) (*anypb.Any, error) {
	if msg == nil {
		return nil, nil
	}
	return anypb.New(msg)
}

func mustPackAny(msg proto.Message) *anypb.Any {
	packed, err := packAny(msg)
	if err != nil {
		return &anypb.Any{Value: []byte(err.Error())}
	}
	return packed
}

func protoMessageAsMap(msg proto.Message) map[string]any {
	if msg == nil {
		return map[string]any{}
	}
	body, err := marshalOptions.Marshal(msg)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return map[string]any{"error": err.Error()}
	}
	return out
}

func cloneSnapshot(snap sessionstream.Snapshot) sessionstream.Snapshot {
	out := sessionstream.Snapshot{SessionId: snap.SessionId, SnapshotOrdinal: snap.SnapshotOrdinal}
	out.Entities = make([]sessionstream.TimelineEntity, 0, len(snap.Entities))
	for _, entity := range snap.Entities {
		cloned := entity
		if entity.Payload != nil {
			cloned.Payload = proto.Clone(entity.Payload)
		}
		out.Entities = append(out.Entities, cloned)
	}
	return out
}

func cloneUIEvent(ev sessionstream.UIEvent) sessionstream.UIEvent {
	out := ev
	if ev.Payload != nil {
		out.Payload = proto.Clone(ev.Payload)
	}
	return out
}

func cloneUIEvents(in []sessionstream.UIEvent) []sessionstream.UIEvent {
	if len(in) == 0 {
		return nil
	}
	out := make([]sessionstream.UIEvent, 0, len(in))
	for _, event := range in {
		out = append(out, cloneUIEvent(event))
	}
	return out
}
