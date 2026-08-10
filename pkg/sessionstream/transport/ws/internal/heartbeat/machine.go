// Package heartbeat implements the pure state machine behind websocket
// application-level heartbeat failure detection.
package heartbeat

import (
	"errors"
	"fmt"
	"time"
)

// Phase is the detector's current lifecycle phase.
type Phase uint8

const (
	PhaseBooting Phase = iota
	PhaseIdle
	PhaseWriting
	PhaseAwaiting
	PhaseSuspected
	PhaseStopped
)

func (p Phase) String() string {
	switch p {
	case PhaseBooting:
		return "booting"
	case PhaseIdle:
		return "idle"
	case PhaseWriting:
		return "writing"
	case PhaseAwaiting:
		return "awaiting"
	case PhaseSuspected:
		return "suspected"
	case PhaseStopped:
		return "stopped"
	default:
		return fmt.Sprintf("phase(%d)", p)
	}
}

// EventKind identifies one input to Machine.Step.
type EventKind uint8

const (
	EventReady EventKind = iota
	EventTick
	EventPingWritten
	EventPingWriteFailed
	EventPongReceived
	EventDeadlineElapsed
	EventStop
)

// Event is an explicit input to the detector. At must come from the same
// monotonic clock domain for all events in one Machine.
type Event struct {
	Kind       EventKind
	At         time.Time
	Generation uint64
	Nonce      string
	Err        error
}

// ActionKind identifies one side effect requested by the pure detector.
type ActionKind uint8

const (
	ActionScheduleTick ActionKind = iota
	ActionSendPing
	ActionArmDeadline
	ActionCancelDeadline
	ActionRecordPong
	ActionRecordStalePong
	ActionRecordSuspected
	ActionCloseConnection
	ActionStop
)

// Action is an effect for the websocket adapter to execute in order.
type Action struct {
	Kind       ActionKind
	At         time.Time
	Generation uint64
	Nonce      string
	Deadline   time.Time
	Reason     error
}

// State is an immutable snapshot of detector state.
type State struct {
	Phase         Phase
	Generation    uint64
	Nonce         string
	WrittenAt     time.Time
	Deadline      time.Time
	PendingPongAt time.Time
}

// Config controls the fixed-threshold detector.
type Config struct {
	PongTimeout time.Duration
}

var (
	// ErrInvalidPongTimeout reports an unusable fixed timeout.
	ErrInvalidPongTimeout = errors.New("heartbeat pong timeout must be positive")
	// ErrMissingNonce reports a tick without a prepared challenge identity.
	ErrMissingNonce = errors.New("heartbeat nonce is empty")
	// ErrEarlyDeadline reports a timer event timestamped before its deadline.
	ErrEarlyDeadline = errors.New("heartbeat deadline event arrived early")
	// ErrPongDeadlineExceeded is the detector's suspicion reason.
	ErrPongDeadlineExceeded = errors.New("heartbeat pong deadline exceeded")
	// ErrPingWriteFailed classifies a local transport write failure.
	ErrPingWriteFailed = errors.New("heartbeat ping write failed")
)

// Machine is a deterministic fixed-threshold failure detector. It owns no
// goroutines, channels, timers, clocks, sockets, protobufs, or observers.
type Machine struct {
	state       State
	pongTimeout time.Duration
}

// New returns a detector in PhaseBooting.
func New(config Config) (*Machine, error) {
	if config.PongTimeout <= 0 {
		return nil, ErrInvalidPongTimeout
	}
	return &Machine{
		state:       State{Phase: PhaseBooting},
		pongTimeout: config.PongTimeout,
	}, nil
}

// State returns a copy of the current state.
func (m *Machine) State() State {
	if m == nil {
		return State{Phase: PhaseStopped}
	}
	return m.state
}

// Step applies one event and returns ordered effects. Stale events are ignored
// or recorded; they never acknowledge or expire the current generation.
func (m *Machine) Step(event Event) ([]Action, error) {
	if m == nil {
		return nil, errors.New("heartbeat machine is nil")
	}
	if m.state.Phase == PhaseStopped {
		return nil, nil
	}
	if event.Kind == EventStop {
		return m.stop(event), nil
	}

	switch m.state.Phase {
	case PhaseBooting:
		return m.stepBooting(event)
	case PhaseIdle:
		return m.stepIdle(event)
	case PhaseWriting:
		return m.stepWriting(event)
	case PhaseAwaiting:
		return m.stepAwaiting(event)
	case PhaseSuspected, PhaseStopped:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid heartbeat phase %d", m.state.Phase)
	}
}

func (m *Machine) stepBooting(event Event) ([]Action, error) {
	switch event.Kind {
	case EventReady:
		m.state.Phase = PhaseIdle
		return []Action{{Kind: ActionScheduleTick, At: event.At}}, nil
	case EventPongReceived:
		return stalePongAction(event), nil
	case EventTick, EventPingWritten, EventPingWriteFailed, EventDeadlineElapsed, EventStop:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid heartbeat event %d", event.Kind)
	}
}

func (m *Machine) stepIdle(event Event) ([]Action, error) {
	switch event.Kind {
	case EventTick:
		if event.Nonce == "" {
			return nil, ErrMissingNonce
		}
		generation := m.state.Generation + 1
		m.state = State{Phase: PhaseWriting, Generation: generation, Nonce: event.Nonce}
		return []Action{{Kind: ActionSendPing, At: event.At, Generation: generation, Nonce: event.Nonce}}, nil
	case EventPongReceived:
		return stalePongAction(event), nil
	case EventReady, EventPingWritten, EventPingWriteFailed, EventDeadlineElapsed, EventStop:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid heartbeat event %d", event.Kind)
	}
}

func (m *Machine) stepWriting(event Event) ([]Action, error) {
	switch event.Kind {
	case EventPingWritten:
		if !m.matches(event) {
			return nil, nil
		}
		deadline := event.At.Add(m.pongTimeout)
		if !m.state.PendingPongAt.IsZero() && m.state.PendingPongAt.Before(deadline) {
			pongAt := m.state.PendingPongAt
			generation := m.state.Generation
			nonce := m.state.Nonce
			m.state = State{Phase: PhaseIdle, Generation: generation}
			return []Action{
				{Kind: ActionRecordPong, At: pongAt, Generation: generation, Nonce: nonce},
				{Kind: ActionScheduleTick, At: pongAt, Generation: generation},
			}, nil
		}
		m.state.Phase = PhaseAwaiting
		m.state.WrittenAt = event.At
		m.state.Deadline = deadline
		m.state.PendingPongAt = time.Time{}
		return []Action{{Kind: ActionArmDeadline, At: event.At, Generation: event.Generation, Nonce: event.Nonce, Deadline: deadline}}, nil
	case EventPingWriteFailed:
		if !m.matches(event) {
			return nil, nil
		}
		reason := errors.Join(ErrPingWriteFailed, event.Err)
		m.state.Phase = PhaseSuspected
		return terminalActions(event, reason), nil
	case EventPongReceived:
		if event.Nonce == m.state.Nonce {
			m.state.PendingPongAt = event.At
			return nil, nil
		}
		return stalePongAction(event), nil
	case EventReady, EventTick, EventDeadlineElapsed, EventStop:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid heartbeat event %d", event.Kind)
	}
}

func (m *Machine) stepAwaiting(event Event) ([]Action, error) {
	switch event.Kind {
	case EventPongReceived:
		if event.Nonce != m.state.Nonce || !event.At.Before(m.state.Deadline) {
			return stalePongAction(event), nil
		}
		generation := m.state.Generation
		nonce := m.state.Nonce
		m.state = State{Phase: PhaseIdle, Generation: generation}
		return []Action{
			{Kind: ActionCancelDeadline, At: event.At, Generation: generation},
			{Kind: ActionRecordPong, At: event.At, Generation: generation, Nonce: nonce},
			{Kind: ActionScheduleTick, At: event.At, Generation: generation},
		}, nil
	case EventDeadlineElapsed:
		if event.Generation != m.state.Generation {
			return nil, nil
		}
		if event.At.Before(m.state.Deadline) {
			return nil, ErrEarlyDeadline
		}
		m.state.Phase = PhaseSuspected
		return terminalActions(event, ErrPongDeadlineExceeded), nil
	case EventPingWritten, EventPingWriteFailed, EventReady, EventTick, EventStop:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid heartbeat event %d", event.Kind)
	}
}

func (m *Machine) matches(event Event) bool {
	return event.Generation == m.state.Generation && event.Nonce == m.state.Nonce
}

func (m *Machine) stop(event Event) []Action {
	actions := make([]Action, 0, 2)
	if m.state.Phase == PhaseAwaiting {
		actions = append(actions, Action{Kind: ActionCancelDeadline, At: event.At, Generation: m.state.Generation})
	}
	m.state = State{Phase: PhaseStopped, Generation: m.state.Generation}
	return append(actions, Action{Kind: ActionStop, At: event.At})
}

func stalePongAction(event Event) []Action {
	return []Action{{Kind: ActionRecordStalePong, At: event.At, Nonce: event.Nonce}}
}

func terminalActions(event Event, reason error) []Action {
	return []Action{
		{Kind: ActionRecordSuspected, At: event.At, Generation: event.Generation, Nonce: event.Nonce, Reason: reason},
		{Kind: ActionCloseConnection, At: event.At, Generation: event.Generation, Nonce: event.Nonce, Reason: reason},
	}
}
