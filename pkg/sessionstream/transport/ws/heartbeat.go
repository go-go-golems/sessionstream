package ws

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws/internal/heartbeat"
)

const heartbeatEventQueueSize = 8

type heartbeatTimer interface {
	C() <-chan time.Time
	Stop() bool
}

type realHeartbeatTimer struct {
	timer *time.Timer
}

func (t realHeartbeatTimer) C() <-chan time.Time { return t.timer.C }
func (t realHeartbeatTimer) Stop() bool          { return t.timer.Stop() }

type heartbeatRuntime struct {
	machine *heartbeat.Machine
	events  chan heartbeat.Event
}

func newHeartbeatRuntime(config ConnectionConfig) (*heartbeatRuntime, error) {
	machine, err := heartbeat.New(heartbeat.Config{PongTimeout: config.PongTimeout})
	if err != nil {
		return nil, err
	}
	return &heartbeatRuntime{
		machine: machine,
		events:  make(chan heartbeat.Event, heartbeatEventQueueSize),
	}, nil
}

func defaultHeartbeatNonce(_ sessionstream.ConnectionId, _ uint64) (string, error) {
	var raw [18]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate heartbeat nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func (s *Server) offerHeartbeatPong(c *connection, nonce string) error {
	if c == nil || c.heartbeat == nil {
		return fmt.Errorf("heartbeat runtime is unavailable")
	}
	event := heartbeat.Event{Kind: heartbeat.EventPongReceived, At: s.heartbeatNow(), Nonce: nonce}
	select {
	case c.heartbeat.events <- event:
		return nil
	case <-c.done:
		return fmt.Errorf("connection %s is closed", c.id)
	default:
		return fmt.Errorf("connection %s heartbeat event queue full", c.id)
	}
}

func (s *Server) runHeartbeatSupervisor(ctxDone <-chan struct{}, c *connection) {
	if c == nil || c.heartbeat == nil {
		return
	}

	var tickTimer heartbeatTimer
	var tickC <-chan time.Time
	var deadlineTimer heartbeatTimer
	var deadlineC <-chan time.Time
	var deadlineGeneration uint64
	var writeAck <-chan error
	var writeGeneration uint64
	var writeNonce string
	stopped := false

	stopTimer := func(timer heartbeatTimer) {
		if timer != nil {
			_ = timer.Stop()
		}
	}
	defer func() {
		stopTimer(tickTimer)
		stopTimer(deadlineTimer)
	}()

	var apply func(heartbeat.Event)
	apply = func(event heartbeat.Event) {
		if stopped {
			return
		}
		actions, err := c.heartbeat.machine.Step(event)
		if err != nil {
			s.observeHeartbeatInvariant(c, err)
			s.closeConnection(c)
			stopped = true
			return
		}
		for _, action := range actions {
			switch action.Kind {
			case heartbeat.ActionScheduleTick:
				stopTimer(tickTimer)
				tickTimer = s.newHeartbeatTimer(s.connectionConfig.HeartbeatInterval)
				tickC = tickTimer.C()
			case heartbeat.ActionSendPing:
				written, queueErr := s.sendFrameTracked(c, newPingFrame(action.Nonce))
				if queueErr != nil {
					apply(heartbeat.Event{Kind: heartbeat.EventPingWriteFailed, At: s.heartbeatNow(), Generation: action.Generation, Nonce: action.Nonce, Err: queueErr})
					continue
				}
				writeAck = written
				writeGeneration = action.Generation
				writeNonce = action.Nonce
				s.observe(context.Background(), TransportRecord{Stage: TransportStageHeartbeatPingQueued, Direction: FrameDirectionServerToClient, ConnectionId: c.id, FrameType: "ping"})
			case heartbeat.ActionArmDeadline:
				stopTimer(deadlineTimer)
				delay := action.Deadline.Sub(s.heartbeatNow())
				if delay <= 0 {
					apply(heartbeat.Event{Kind: heartbeat.EventDeadlineElapsed, At: s.heartbeatNow(), Generation: action.Generation})
					continue
				}
				deadlineTimer = s.newHeartbeatTimer(delay)
				deadlineC = deadlineTimer.C()
				deadlineGeneration = action.Generation
			case heartbeat.ActionCancelDeadline:
				stopTimer(deadlineTimer)
				deadlineTimer = nil
				deadlineC = nil
				deadlineGeneration = 0
			case heartbeat.ActionRecordPong:
				// Pong receipt is observed by readLoop for both matching and stale
				// responses, preserving the existing transport record contract.
			case heartbeat.ActionRecordStalePong:
				// Stale input is intentionally ignored after reducer validation.
			case heartbeat.ActionRecordSuspected:
				if errors.Is(action.Reason, heartbeat.ErrPongDeadlineExceeded) {
					err := fmt.Errorf("connection %s heartbeat pong timeout: %w", c.id, action.Reason)
					s.observe(context.Background(), TransportRecord{Stage: TransportStageHeartbeatTimeout, ConnectionId: c.id, Err: err})
				}
			case heartbeat.ActionCloseConnection:
				s.closeConnection(c)
			case heartbeat.ActionStop:
				stopped = true
			default:
				s.observeHeartbeatInvariant(c, fmt.Errorf("unknown heartbeat action %d", action.Kind))
				s.closeConnection(c)
				stopped = true
			}
		}
	}

	select {
	case <-ctxDone:
		return
	case <-c.done:
		return
	case <-c.ready:
	}
	apply(heartbeat.Event{Kind: heartbeat.EventReady, At: s.heartbeatNow()})
	for !stopped {
		select {
		case <-ctxDone:
			apply(heartbeat.Event{Kind: heartbeat.EventStop, At: s.heartbeatNow()})
		case <-c.done:
			apply(heartbeat.Event{Kind: heartbeat.EventStop, At: s.heartbeatNow()})
		case event := <-c.heartbeat.events:
			apply(event)
		case at := <-tickC:
			tickC = nil
			generation := c.heartbeat.machine.State().Generation + 1
			nonce, err := s.heartbeatNonce(c.id, generation)
			if err != nil {
				s.observeHeartbeatInvariant(c, err)
				s.closeConnection(c)
				stopped = true
				continue
			}
			apply(heartbeat.Event{Kind: heartbeat.EventTick, At: at, Nonce: nonce})
		case err := <-writeAck:
			writeAck = nil
			kind := heartbeat.EventPingWritten
			if err != nil {
				kind = heartbeat.EventPingWriteFailed
			}
			apply(heartbeat.Event{Kind: kind, At: s.heartbeatNow(), Generation: writeGeneration, Nonce: writeNonce, Err: err})
		case at := <-deadlineC:
			deadlineC = nil
			apply(heartbeat.Event{Kind: heartbeat.EventDeadlineElapsed, At: at, Generation: deadlineGeneration})
		}
	}
}

func (s *Server) observeHeartbeatInvariant(c *connection, err error) {
	cid := sessionstream.ConnectionId("")
	if c != nil {
		cid = c.id
	}
	s.observe(context.Background(), TransportRecord{Stage: TransportStageProtocolError, ConnectionId: cid, Err: fmt.Errorf("heartbeat invariant: %w", err)})
}
