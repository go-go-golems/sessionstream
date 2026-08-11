package ws

import (
	"testing"
	"time"

	"github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws/internal/heartbeat"
	"github.com/stretchr/testify/require"
)

const (
	arbitrationIdentityCurrent byte = iota
	arbitrationIdentityStale
	arbitrationIdentityEmpty
	arbitrationIdentityFuture
)

const (
	arbitrationTimeBeforeDeadline byte = iota
	arbitrationTimeAtDeadline
	arbitrationTimeAfterDeadline
	arbitrationTimeBeforeWrite
)

func TestHeartbeatDeadlineArbitrationDrainsAdmittedTimelyPong(t *testing.T) {
	machine, generation, nonce, _, deadline := newAwaitingHeartbeatMachine(t)
	events := make(chan heartbeat.Event, heartbeatEventQueueSize)
	events <- heartbeat.Event{Kind: heartbeat.EventPongReceived, At: deadline.Add(-time.Nanosecond), Nonce: nonce}

	actions := applyArbitratedHeartbeatDeadline(t, machine, events, heartbeat.Event{
		Kind:       heartbeat.EventDeadlineElapsed,
		At:         deadline.Add(time.Second),
		Generation: generation,
	})

	require.Equal(t, heartbeat.PhaseIdle, machine.State().Phase)
	require.Equal(t, 0, countHeartbeatActions(actions, heartbeat.ActionRecordSuspected))
	require.Equal(t, 0, countHeartbeatActions(actions, heartbeat.ActionCloseConnection))
	require.Empty(t, events)
}

func TestHeartbeatDeadlineArbitrationBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		events    []byte
		wantPhase heartbeat.Phase
	}{
		{name: "empty queue", wantPhase: heartbeat.PhaseSuspected},
		{name: "timely current pong", events: []byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 0)}, wantPhase: heartbeat.PhaseIdle},
		{name: "current pong at deadline", events: []byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAtDeadline, 0)}, wantPhase: heartbeat.PhaseSuspected},
		{name: "late current pong", events: []byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAfterDeadline, 0)}, wantPhase: heartbeat.PhaseSuspected},
		{name: "stale timely pong", events: []byte{arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0)}, wantPhase: heartbeat.PhaseSuspected},
		{name: "timely current pong after stale pong", events: []byte{
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0),
			arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 1),
		}, wantPhase: heartbeat.PhaseIdle},
		{name: "timely current pong in final queue slot", events: []byte{
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 1),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 2),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 3),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 4),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 5),
			arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 6),
			arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 7),
		}, wantPhase: heartbeat.PhaseIdle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, actions, remaining := runHeartbeatDeadlineArbitration(t, tt.events, 0)
			require.Equal(t, tt.wantPhase, phase)
			require.Zero(t, remaining)
			assertHeartbeatArbitrationActions(t, tt.wantPhase, actions)
		})
	}
}

func FuzzHeartbeatDeadlineArbitration(f *testing.F) {
	f.Add([]byte{}, byte(0))
	f.Add([]byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 0)}, byte(1))
	f.Add([]byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAtDeadline, 0)}, byte(2))
	f.Add([]byte{arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAfterDeadline, 0)}, byte(3))
	f.Add([]byte{arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0)}, byte(4))
	f.Add([]byte{
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0),
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 1),
	}, byte(5))
	f.Add([]byte{
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 0),
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAfterDeadline, 1),
	}, byte(6))
	f.Add([]byte{
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeAfterDeadline, 0),
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 1),
	}, byte(7))
	f.Add([]byte{
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 1),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 2),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 3),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 4),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 5),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 6),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 7),
	}, byte(8))
	f.Add([]byte{
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 0),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 1),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 2),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 3),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 4),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 5),
		arbitrationByte(arbitrationIdentityStale, arbitrationTimeBeforeDeadline, 6),
		arbitrationByte(arbitrationIdentityCurrent, arbitrationTimeBeforeDeadline, 7),
	}, byte(9))

	f.Fuzz(func(t *testing.T, input []byte, expiryOffset byte) {
		if len(input) > heartbeatEventQueueSize {
			input = input[:heartbeatEventQueueSize]
		}
		wantPhase := heartbeat.PhaseSuspected
		for _, raw := range input {
			if arbitrationIdentity(raw) == arbitrationIdentityCurrent && arbitrationTime(raw) != arbitrationTimeAtDeadline && arbitrationTime(raw) != arbitrationTimeAfterDeadline {
				wantPhase = heartbeat.PhaseIdle
				break
			}
		}

		phase, actions, remaining := runHeartbeatDeadlineArbitration(t, input, expiryOffset)
		require.Equal(t, wantPhase, phase)
		require.Zero(t, remaining)
		assertHeartbeatArbitrationActions(t, wantPhase, actions)
	})
}

func runHeartbeatDeadlineArbitration(t testing.TB, input []byte, expiryOffset byte) (heartbeat.Phase, []heartbeat.Action, int) {
	t.Helper()
	machine, generation, nonce, writtenAt, deadline := newAwaitingHeartbeatMachine(t)
	events := make(chan heartbeat.Event, heartbeatEventQueueSize)
	for index, raw := range input {
		events <- decodeHeartbeatArbitrationPong(raw, index, nonce, writtenAt, deadline)
	}
	deadlineEvent := heartbeat.Event{
		Kind:       heartbeat.EventDeadlineElapsed,
		At:         deadline.Add(time.Duration(expiryOffset) * time.Nanosecond),
		Generation: generation,
	}
	actions := applyArbitratedHeartbeatDeadline(t, machine, events, deadlineEvent)
	return machine.State().Phase, actions, len(events)
}

func newAwaitingHeartbeatMachine(t testing.TB) (*heartbeat.Machine, uint64, string, time.Time, time.Time) {
	t.Helper()
	const timeout = 5 * time.Second
	start := time.Unix(1_700_000_000, 0)
	writtenAt := start.Add(time.Second)
	nonce := "current-nonce"
	machine, err := heartbeat.New(heartbeat.Config{PongTimeout: timeout})
	require.NoError(t, err)
	_, err = machine.Step(heartbeat.Event{Kind: heartbeat.EventReady, At: start})
	require.NoError(t, err)
	actions, err := machine.Step(heartbeat.Event{Kind: heartbeat.EventTick, At: start, Nonce: nonce})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	generation := actions[0].Generation
	actions, err = machine.Step(heartbeat.Event{Kind: heartbeat.EventPingWritten, At: writtenAt, Generation: generation, Nonce: nonce})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, heartbeat.ActionArmDeadline, actions[0].Kind)
	return machine, generation, nonce, writtenAt, actions[0].Deadline
}

func applyArbitratedHeartbeatDeadline(
	t testing.TB,
	machine *heartbeat.Machine,
	events <-chan heartbeat.Event,
	deadline heartbeat.Event,
) []heartbeat.Action {
	t.Helper()
	var allActions []heartbeat.Action
	applyHeartbeatDeadlineAfterAdmittedEvents(events, deadline, func(event heartbeat.Event) {
		actions, err := machine.Step(event)
		require.NoError(t, err)
		allActions = append(allActions, actions...)
	})
	return allActions
}

func decodeHeartbeatArbitrationPong(raw byte, index int, currentNonce string, writtenAt, deadline time.Time) heartbeat.Event {
	delta := (time.Duration((raw>>4)&0x0f) + 1) * time.Nanosecond
	nonce := currentNonce
	switch arbitrationIdentity(raw) {
	case arbitrationIdentityCurrent:
	case arbitrationIdentityStale:
		nonce = "stale-nonce"
	case arbitrationIdentityEmpty:
		nonce = ""
	case arbitrationIdentityFuture:
		nonce = "future-nonce"
	default:
		nonce = "stale-nonce"
	}
	var at time.Time
	switch arbitrationTime(raw) {
	case arbitrationTimeBeforeDeadline:
		at = deadline.Add(-delta)
	case arbitrationTimeAtDeadline:
		at = deadline
	case arbitrationTimeAfterDeadline:
		at = deadline.Add(delta)
	case arbitrationTimeBeforeWrite:
		at = writtenAt.Add(-delta)
	default:
		at = deadline.Add(time.Duration(index+1) * time.Nanosecond)
	}
	return heartbeat.Event{Kind: heartbeat.EventPongReceived, At: at, Nonce: nonce}
}

func arbitrationByte(identity, timeMode, delta byte) byte {
	return identity&0x03 | (timeMode&0x03)<<2 | (delta&0x0f)<<4
}

func arbitrationIdentity(raw byte) byte { return raw & 0x03 }

func arbitrationTime(raw byte) byte { return (raw >> 2) & 0x03 }

func assertHeartbeatArbitrationActions(t testing.TB, phase heartbeat.Phase, actions []heartbeat.Action) {
	t.Helper()
	suspected := countHeartbeatActions(actions, heartbeat.ActionRecordSuspected)
	closed := countHeartbeatActions(actions, heartbeat.ActionCloseConnection)
	require.Equal(t, suspected, closed)
	if phase == heartbeat.PhaseIdle {
		require.Zero(t, suspected)
		require.Equal(t, 1, countHeartbeatActions(actions, heartbeat.ActionRecordPong))
		return
	}
	require.Equal(t, heartbeat.PhaseSuspected, phase)
	require.Equal(t, 1, suspected)
	require.Equal(t, 1, closed)
}

func countHeartbeatActions(actions []heartbeat.Action, kind heartbeat.ActionKind) int {
	count := 0
	for _, action := range actions {
		if action.Kind == kind {
			count++
		}
	}
	return count
}
