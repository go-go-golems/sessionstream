package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

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

// ConnectionConfig bounds one upgraded websocket connection and configures
// application-level ping/pong liveness using the generated transport frames.
type ConnectionConfig struct {
	MaxReadBytes      int64
	SendQueueSize     int
	WriteTimeout      time.Duration
	HeartbeatInterval time.Duration
	PongTimeout       time.Duration
}

// DefaultConnectionConfig returns conservative production-safe connection bounds.
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxReadBytes:      1 << 20,
		SendQueueSize:     128,
		WriteTimeout:      10 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		PongTimeout:       10 * time.Second,
	}
}

func (c ConnectionConfig) validate() error {
	if c.MaxReadBytes <= 0 || c.SendQueueSize <= 0 {
		return fmt.Errorf("websocket read limit and send queue size must be positive")
	}
	if c.WriteTimeout <= 0 || c.HeartbeatInterval <= 0 || c.PongTimeout <= 0 {
		return fmt.Errorf("websocket write/heartbeat/pong timeouts must be positive")
	}
	return nil
}

// SnapshotProvider provides snapshot lookup for subscribe flows.
type SnapshotProvider interface {
	Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error)
}

// Option configures a websocket Server.
type Option func(*Server) error

// WithConnectionConfig replaces the default connection bounds and heartbeat policy.
func WithConnectionConfig(config ConnectionConfig) Option {
	return func(s *Server) error {
		if err := config.validate(); err != nil {
			return err
		}
		s.connectionConfig = config
		return nil
	}
}

// SubscribeAuthorizer authorizes a session before snapshot hydration begins.
type SubscribeAuthorizer func(ctx context.Context, sid sessionstream.SessionId) error

// WithSubscribeAuthorizer installs a fail-closed subscription authorization hook.
func WithSubscribeAuthorizer(authorize SubscribeAuthorizer) Option {
	return func(s *Server) error {
		if authorize == nil {
			return fmt.Errorf("websocket subscribe authorizer is nil")
		}
		s.authorizeSubscribe = authorize
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
// sinceSnapshotOrdinal is accepted, stored, echoed, and surfaced to observers for
// teaching and diagnostics, but it is advisory for now: this reference adapter does not
// replay missed UI events from the event store. Replayed event history belongs
// behind an explicit replay API rather than being hidden inside websocket
// subscribe semantics.
//
// Production callers should wrap this handler with their own authentication,
// authorization, origin policy, and rate limiting. The default upgrader is
// intentionally permissive for local labs and examples; use WithUpgrader to
// install a stricter CheckOrigin policy.
type Server struct {
	snapshots          SnapshotProvider
	upgrader           websocket.Upgrader
	observer           TransportObserver
	connectionConfig   ConnectionConfig
	authorizeSubscribe SubscribeAuthorizer

	maxHydrationBufferedBatches int

	nextID uint64

	mu          sync.RWMutex
	conns       map[sessionstream.ConnectionId]*connection
	bySession   map[sessionstream.SessionId]map[sessionstream.ConnectionId]struct{}
	lifecycleMu sync.Mutex
	closing     bool
	wg          sync.WaitGroup
}

type connection struct {
	id       sessionstream.ConnectionId
	ws       *websocket.Conn
	send     chan outboundFrame
	close    sync.Once
	closed   atomic.Bool
	done     chan struct{}
	pongs    chan string
	requests chan *sessionstreamv1.ClientFrame

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
	written   chan error
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
		connectionConfig:            DefaultConnectionConfig(),
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
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	s.lifecycleMu.Lock()
	if s.closing {
		s.lifecycleMu.Unlock()
		http.Error(w, "websocket server is closing", http.StatusServiceUnavailable)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.lifecycleMu.Unlock()
		s.observe(r.Context(), TransportRecord{Stage: TransportStageUpgradeError, Err: err})
		return
	}
	cid := sessionstream.ConnectionId(fmt.Sprintf("conn-%d", atomic.AddUint64(&s.nextID, 1)))
	c := &connection{
		id:       cid,
		ws:       conn,
		send:     make(chan outboundFrame, s.connectionConfig.SendQueueSize),
		subs:     map[sessionstream.SessionId]subscription{},
		done:     make(chan struct{}),
		pongs:    make(chan string, 1),
		requests: make(chan *sessionstreamv1.ClientFrame, s.connectionConfig.SendQueueSize),
	}
	s.mu.Lock()
	s.conns[cid] = c
	s.mu.Unlock()
	s.wg.Add(4)
	s.lifecycleMu.Unlock()
	s.observe(r.Context(), TransportRecord{Stage: TransportStageConnected, ConnectionId: cid})
	defer s.wg.Done()
	go func() { defer s.wg.Done(); s.writeLoop(ctx, c) }()
	go func() { defer s.wg.Done(); s.heartbeatLoop(ctx, c) }()
	go func() { defer s.wg.Done(); s.requestLoop(ctx, c) }()
	_ = s.sendFrame(c, newHelloFrame(cid))
	s.readLoop(ctx, c)
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
	deliveryErrs := make([]error, 0)
	for _, c := range targets {
		if err := s.deliverUIEvents(ctx, c, sid, ord, events); err != nil {
			deliveryErrs = append(deliveryErrs, fmt.Errorf("connection %s: %w", c.id, err))
			s.closeConnection(c)
		}
	}
	s.observe(ctx, TransportRecord{Stage: TransportStageFanoutCompleted, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events), FanoutTargetIds: targetIDs, Err: errors.Join(deliveryErrs...)})
	return errors.Join(deliveryErrs...)
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
	c.ws.SetReadLimit(s.connectionConfig.MaxReadBytes)
	defer close(c.requests)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageReadError, Direction: FrameDirectionClientToServer, ConnectionId: c.id, Err: err})
			return
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageClientFrameRead, Direction: FrameDirectionClientToServer, ConnectionId: c.id, RawBytes: len(raw)})
		frame := &sessionstreamv1.ClientFrame{}
		if err := unmarshalOpts.Unmarshal(raw, frame); err != nil {
			s.observe(ctx, TransportRecord{Stage: TransportStageClientFrameDecodeError, Direction: FrameDirectionClientToServer, ConnectionId: c.id, RawBytes: len(raw), Err: err})
			_ = s.sendFrame(c, newErrorFrame("bad_client_frame", err.Error(), ""))
			continue
		}
		s.observe(ctx, clientFrameRecord(TransportStageClientFrameDecoded, c.id, frame, len(raw)))
		switch typed := frame.GetFrame().(type) {
		case *sessionstreamv1.ClientFrame_Ping:
			if err := s.sendFrame(c, newPongFrame(typed.Ping.GetNonce())); err != nil {
				return
			}
			continue
		case *sessionstreamv1.ClientFrame_Pong:
			select {
			case c.pongs <- typed.Pong.GetNonce():
			default:
			}
			s.observe(ctx, TransportRecord{Stage: TransportStageHeartbeatPongReceived, Direction: FrameDirectionClientToServer, ConnectionId: c.id, FrameType: "pong"})
			continue
		}
		select {
		case c.requests <- frame:
		case <-ctx.Done():
			return
		case <-c.done:
			return
		default:
			err := fmt.Errorf("connection %s client request queue full", c.id)
			rec := clientFrameRecord(TransportStageProtocolError, c.id, frame, 0)
			rec.Err = err
			s.observe(ctx, rec)
			_ = s.sendFrame(c, newErrorFrame("request_queue_full", err.Error(), ""))
			return
		}
	}
}

func (s *Server) requestLoop(ctx context.Context, c *connection) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case frame, ok := <-c.requests:
			if !ok {
				return
			}
			if err := s.handleRequestFrame(ctx, c, frame); err != nil {
				rec := clientFrameRecord(TransportStageProtocolError, c.id, frame, 0)
				rec.Err = err
				s.observe(ctx, rec)
				_ = s.sendFrame(c, newErrorFrame("protocol_error", err.Error(), ""))
			}
		}
	}
}

func (s *Server) handleRequestFrame(ctx context.Context, c *connection, frame *sessionstreamv1.ClientFrame) error {
	switch typed := frame.GetFrame().(type) {
	case *sessionstreamv1.ClientFrame_Subscribe:
		sub := typed.Subscribe
		sid := sessionstream.SessionId(sub.GetSessionId())
		if sid == "" {
			return fmt.Errorf("subscribe missing session id")
		}
		if s.authorizeSubscribe != nil {
			if err := s.authorizeSubscribe(ctx, sid); err != nil {
				s.observe(ctx, TransportRecord{Stage: TransportStageSubscribeDenied, ConnectionId: c.id, SessionId: sid, Err: err})
				return fmt.Errorf("subscribe session %q: %w", sid, err)
			}
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
		snapshotRecord := TransportRecord{Stage: TransportStageSnapshotLoaded, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since, SnapshotOrdinal: snap.SnapshotOrdinal, SnapshotEntityCount: len(snap.Entities), SnapshotEntities: summarizeEntities(snap.Entities)}
		s.observe(ctx, snapshotRecord)
		if err := s.sendFrame(c, newSnapshotFrame(sid, snap)); err != nil {
			return err
		}
		snapshotRecord.Stage = TransportStageSnapshotSent
		s.observe(ctx, snapshotRecord)
		buffered := s.drainHydrationBuffer(c, sid, snap.SnapshotOrdinal)
		if len(buffered) > 0 {
			s.observe(ctx, TransportRecord{Stage: TransportStageHydrationBufferFlushed, ConnectionId: c.id, SessionId: sid, SnapshotOrdinal: snap.SnapshotOrdinal, FanoutEventCount: countBufferedEvents(buffered)})
		}
		for _, batch := range buffered {
			if err := s.sendUIBatch(ctx, c, sid, batch.ordinal, batch.events); err != nil {
				return err
			}
		}
		lateEventCount, err := s.flushLateHydrationBufferAndMarkLive(ctx, c, sid, snap.SnapshotOrdinal)
		if err != nil {
			return err
		}
		if lateEventCount > 0 {
			s.observe(ctx, TransportRecord{Stage: TransportStageHydrationBufferFlushed, ConnectionId: c.id, SessionId: sid, SnapshotOrdinal: snap.SnapshotOrdinal, FanoutEventCount: lateEventCount})
		}
		if err := s.sendFrame(c, newSubscribedFrame(sid, since)); err != nil {
			return err
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageSubscribed, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: since, SnapshotOrdinal: snap.SnapshotOrdinal})
		completed = true
		return nil
	case *sessionstreamv1.ClientFrame_Unsubscribe:
		sid := sessionstream.SessionId(typed.Unsubscribe.GetSessionId())
		if sid == "" {
			return fmt.Errorf("unsubscribe missing session id")
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageUnsubscribeReceived, Direction: FrameDirectionClientToServer, ConnectionId: c.id, SessionId: sid, FrameType: "unsubscribe"})
		s.removeSubscription(c, sid)
		if err := s.sendFrame(c, newUnsubscribedFrame(sid)); err != nil {
			return err
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageUnsubscribed, ConnectionId: c.id, SessionId: sid})
		return nil
	default:
		return fmt.Errorf("unknown client frame")
	}
}

func (s *Server) writeLoop(ctx context.Context, c *connection) {
	defer s.closeConnection(c)
	for {
		var msg outboundFrame
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case msg = <-c.send:
		}
		if err := c.ws.SetWriteDeadline(time.Now().Add(s.connectionConfig.WriteTimeout)); err != nil {
			notifyFrameWritten(msg, err)
			s.observe(ctx, TransportRecord{Stage: TransportStageServerFrameWriteError, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: msg.frameType, RawBytes: len(msg.body), Err: err})
			return
		}
		if err := c.ws.WriteMessage(websocket.TextMessage, msg.body); err != nil {
			notifyFrameWritten(msg, err)
			s.observe(ctx, TransportRecord{Stage: TransportStageServerFrameWriteError, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: msg.frameType, RawBytes: len(msg.body), Err: err})
			return
		}
		notifyFrameWritten(msg, nil)
		s.observe(ctx, TransportRecord{Stage: TransportStageServerFrameWritten, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: msg.frameType, RawBytes: len(msg.body)})
	}
}

func (s *Server) heartbeatLoop(ctx context.Context, c *connection) {
	ticker := time.NewTicker(s.connectionConfig.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
		}
		nonce := fmt.Sprintf("%s-%d", c.id, time.Now().UnixNano())
		written, err := s.sendFrameTracked(c, newPingFrame(nonce))
		if err != nil {
			s.closeConnection(c)
			return
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageHeartbeatPingQueued, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: "ping"})
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case err := <-written:
			if err != nil {
				s.closeConnection(c)
				return
			}
		}
		timer := time.NewTimer(s.connectionConfig.PongTimeout)
		matched := false
		for !matched {
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-c.done:
				if !timer.Stop() {
					<-timer.C
				}
				return
			case pong := <-c.pongs:
				matched = pong == nonce
			case <-timer.C:
				err := fmt.Errorf("connection %s heartbeat pong timeout", c.id)
				s.observe(ctx, TransportRecord{Stage: TransportStageHeartbeatTimeout, ConnectionId: c.id, Err: err})
				s.closeConnection(c)
				return
			}
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
}

func (s *Server) closeConnection(c *connection) {
	if c == nil {
		return
	}
	c.close.Do(func() {
		close(c.done)
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
		if c.ws != nil {
			_ = c.ws.Close()
		}
		s.observe(context.Background(), TransportRecord{Stage: TransportStageDisconnected, ConnectionId: c.id})
	})
}

// Close closes all upgraded connections and waits for their read, write, and
// heartbeat loops to exit or for ctx to expire.
func (s *Server) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("close websocket server context is nil")
	}
	s.lifecycleMu.Lock()
	s.closing = true
	s.lifecycleMu.Unlock()
	s.mu.RLock()
	connections := make([]*connection, 0, len(s.conns))
	for _, connection := range s.conns {
		connections = append(connections, connection)
	}
	s.mu.RUnlock()
	for _, connection := range connections {
		s.closeConnection(connection)
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
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

// deliverUIEvents routes a UI event batch to one connection. It checks the
// subscription state under the connection lock and either buffers (hydrating)
// or sends directly (live). This eliminates the TOCTOU race where PublishUI
// reads a stale subscription state while another goroutine transitions the
// subscription between hydration and live delivery.
func (s *Server) deliverUIEvents(ctx context.Context, c *connection, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	c.mu.Lock()
	sub, ok := c.subs[sid]
	if !ok {
		c.mu.Unlock()
		return nil
	}
	switch sub.state {
	case subscriptionStateLive:
		c.mu.Unlock()
		return s.sendUIBatch(ctx, c, sid, ord, events)
	case subscriptionStateHydrating:
		if len(sub.buffer) >= s.maxHydrationBufferedBatches {
			err := fmt.Errorf("connection %s hydration buffer full for session %s", c.id, sid)
			c.mu.Unlock()
			s.observe(ctx, TransportRecord{Stage: TransportStageHydrationBufferOverflow, ConnectionId: c.id, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events), Err: err})
			_ = s.sendFrame(c, newErrorFrame("hydration_buffer_overflow", err.Error(), ""))
			return err
		}
		sub.buffer = append(sub.buffer, bufferedUIBatch{ordinal: ord, events: cloneUIEvents(events)})
		c.subs[sid] = sub
		c.mu.Unlock()
		s.observe(ctx, TransportRecord{Stage: TransportStageUIEventBuffered, ConnectionId: c.id, SessionId: sid, Ordinal: ord, FanoutEventCount: len(events)})
		return nil
	default:
		c.mu.Unlock()
		return nil
	}
}

func (s *Server) drainHydrationBuffer(c *connection, sid sessionstream.SessionId, snapshotOrdinal uint64) []bufferedUIBatch {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subs[sid]
	if !ok {
		return nil
	}
	out := filterBufferedAfterSnapshot(sub.buffer, snapshotOrdinal)
	sub.buffer = nil
	sub.snapshotOrdinal = snapshotOrdinal
	c.subs[sid] = sub
	return out
}

// flushLateHydrationBufferAndMarkLive transitions a subscription from hydrating
// to live only after any batches buffered since drainHydrationBuffer have been
// queued to the connection. Holding c.mu while queueing those late batches keeps
// concurrent PublishUI calls from observing the live state and enqueueing newer
// events ahead of older late-hydration events.
func (s *Server) flushLateHydrationBufferAndMarkLive(ctx context.Context, c *connection, sid sessionstream.SessionId, snapshotOrdinal uint64) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sub, ok := c.subs[sid]
	if !ok {
		return 0, nil
	}
	late := filterBufferedAfterSnapshot(sub.buffer, snapshotOrdinal)
	for _, batch := range late {
		if err := s.sendUIBatch(ctx, c, sid, batch.ordinal, batch.events); err != nil {
			return 0, err
		}
	}
	sub.state = subscriptionStateLive
	sub.snapshotOrdinal = snapshotOrdinal
	sub.buffer = nil
	c.subs[sid] = sub
	s.observe(context.Background(), TransportRecord{Stage: TransportStageSubscriptionLive, ConnectionId: c.id, SessionId: sid, SinceSnapshotOrdinal: sub.sinceSnapshotOrdinal, SnapshotOrdinal: snapshotOrdinal})
	return countBufferedEvents(late), nil
}

func (s *Server) sendUIBatch(ctx context.Context, c *connection, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
	for _, event := range events {
		frame, err := newUIEventFrame(sid, ord, event)
		if err != nil {
			return err
		}
		if err := s.sendFrame(c, frame); err != nil {
			return err
		}
		s.observe(ctx, TransportRecord{Stage: TransportStageUIEventSent, Direction: FrameDirectionServerToClient, ConnectionId: c.id, SessionId: sid, Ordinal: ord, EventName: event.Name, PayloadType: payloadType(event.Payload), UIEvent: cloneUIEvent(event)})
	}
	return nil
}

func filterBufferedAfterSnapshot(batches []bufferedUIBatch, snapshotOrdinal uint64) []bufferedUIBatch {
	out := make([]bufferedUIBatch, 0, len(batches))
	for _, batch := range batches {
		if batch.ordinal > snapshotOrdinal {
			out = append(out, cloneBufferedUIBatch(batch))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ordinal < out[j].ordinal })
	return out
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

func (s *Server) sendFrame(c *connection, frame *sessionstreamv1.ServerFrame) error {
	return s.queueFrame(c, frame, nil)
}

func (s *Server) sendFrameTracked(c *connection, frame *sessionstreamv1.ServerFrame) (<-chan error, error) {
	written := make(chan error, 1)
	if err := s.queueFrame(c, frame, written); err != nil {
		return nil, err
	}
	return written, nil
}

func (s *Server) queueFrame(c *connection, frame *sessionstreamv1.ServerFrame, written chan error) error {
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
	queueLen := len(c.send)
	queueCap := cap(c.send)
	select {
	case c.send <- outboundFrame{body: body, frameType: frameType, written: written}:
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

func notifyFrameWritten(frame outboundFrame, err error) {
	if frame.written != nil {
		frame.written <- err
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

func newPingFrame(nonce string) *sessionstreamv1.ServerFrame {
	return &sessionstreamv1.ServerFrame{Frame: &sessionstreamv1.ServerFrame_Ping{Ping: &sessionstreamv1.PingFrame{Nonce: nonce}}}
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

func clientFrameRecord(stage TransportStage, cid sessionstream.ConnectionId, frame *sessionstreamv1.ClientFrame, rawBytes int) TransportRecord {
	rec := TransportRecord{Stage: stage, Direction: FrameDirectionClientToServer, ConnectionId: cid, FrameType: clientFrameType(frame), RawBytes: rawBytes}
	switch typed := frame.GetFrame().(type) {
	case *sessionstreamv1.ClientFrame_Subscribe:
		rec.SessionId = sessionstream.SessionId(typed.Subscribe.GetSessionId())
		rec.SinceSnapshotOrdinal = typed.Subscribe.GetSinceSnapshotOrdinal()
	case *sessionstreamv1.ClientFrame_Unsubscribe:
		rec.SessionId = sessionstream.SessionId(typed.Unsubscribe.GetSessionId())
	}
	return rec
}

func payloadType(msg proto.Message) string {
	if msg == nil {
		return ""
	}
	return string(msg.ProtoReflect().Descriptor().FullName())
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
