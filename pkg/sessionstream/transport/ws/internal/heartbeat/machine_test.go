package heartbeat

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var epoch = time.Unix(1_700_000_000, 0)

func TestNewRejectsNonPositiveTimeout(t *testing.T) {
	for _, timeout := range []time.Duration{0, -time.Second} {
		_, err := New(Config{PongTimeout: timeout})
		require.ErrorIs(t, err, ErrInvalidPongTimeout)
	}
}

func TestReadySchedulesFirstTick(t *testing.T) {
	m := newMachine(t)
	actions := step(t, m, Event{Kind: EventReady, At: epoch})
	require.Equal(t, PhaseIdle, m.State().Phase)
	requireActionKinds(t, actions, ActionScheduleTick)
}

func TestMatchingPongBeforeDeadlineReturnsIdle(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "nonce-1")
	writtenAt := epoch.Add(time.Second)
	actions := step(t, m, Event{Kind: EventPingWritten, At: writtenAt, Generation: generation, Nonce: nonce})
	require.Equal(t, writtenAt.Add(5*time.Second), m.State().Deadline)
	requireActionKinds(t, actions, ActionArmDeadline)

	actions = step(t, m, Event{Kind: EventPongReceived, At: writtenAt.Add(time.Second), Nonce: nonce})
	require.Equal(t, PhaseIdle, m.State().Phase)
	require.Equal(t, generation, m.State().Generation)
	require.Empty(t, m.State().Nonce)
	requireActionKinds(t, actions, ActionCancelDeadline, ActionRecordPong, ActionScheduleTick)
}

func TestPongAtOrAfterDeadlineIsStale(t *testing.T) {
	for _, offset := range []time.Duration{5 * time.Second, 5*time.Second + time.Nanosecond} {
		t.Run(offset.String(), func(t *testing.T) {
			m := readyMachine(t)
			generation, nonce := beginChallenge(t, m, epoch, "nonce")
			step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})
			actions := step(t, m, Event{Kind: EventPongReceived, At: epoch.Add(offset), Nonce: nonce})
			require.Equal(t, PhaseAwaiting, m.State().Phase)
			requireActionKinds(t, actions, ActionRecordStalePong)
		})
	}
}

func TestStalePongNeverAcknowledgesCurrentChallenge(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})

	actions := step(t, m, Event{Kind: EventPongReceived, At: epoch.Add(time.Second), Nonce: "stale"})
	require.Equal(t, PhaseAwaiting, m.State().Phase)
	require.Equal(t, nonce, m.State().Nonce)
	requireActionKinds(t, actions, ActionRecordStalePong)
}

func TestStaleDeadlineNeverExpiresCurrentChallenge(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})

	actions := step(t, m, Event{Kind: EventDeadlineElapsed, At: epoch.Add(10 * time.Second), Generation: generation - 1})
	require.Empty(t, actions)
	require.Equal(t, PhaseAwaiting, m.State().Phase)
}

func TestMatchingDeadlineSuspectsAndCloses(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})

	actions := step(t, m, Event{Kind: EventDeadlineElapsed, At: epoch.Add(5 * time.Second), Generation: generation})
	require.Equal(t, PhaseSuspected, m.State().Phase)
	requireActionKinds(t, actions, ActionRecordSuspected, ActionCloseConnection)
	require.ErrorIs(t, actions[0].Reason, ErrPongDeadlineExceeded)
}

func TestEarlyMatchingDeadlineIsInvariantError(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})

	_, err := m.Step(Event{Kind: EventDeadlineElapsed, At: epoch.Add(4 * time.Second), Generation: generation})
	require.ErrorIs(t, err, ErrEarlyDeadline)
	require.Equal(t, PhaseAwaiting, m.State().Phase)
}

func TestWriteFailureOnlyAffectsMatchingChallenge(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")

	actions := step(t, m, Event{Kind: EventPingWriteFailed, At: epoch, Generation: generation + 1, Nonce: nonce, Err: errors.New("old")})
	require.Empty(t, actions)
	require.Equal(t, PhaseWriting, m.State().Phase)

	writeErr := errors.New("socket failed")
	actions = step(t, m, Event{Kind: EventPingWriteFailed, At: epoch, Generation: generation, Nonce: nonce, Err: writeErr})
	require.Equal(t, PhaseSuspected, m.State().Phase)
	requireActionKinds(t, actions, ActionRecordSuspected, ActionCloseConnection)
	require.ErrorIs(t, actions[0].Reason, ErrPingWriteFailed)
	require.ErrorIs(t, actions[0].Reason, writeErr)
}

func TestStopIsAbsorbingAndCancelsActiveDeadline(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "current")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})

	actions := step(t, m, Event{Kind: EventStop, At: epoch.Add(time.Second)})
	require.Equal(t, PhaseStopped, m.State().Phase)
	requireActionKinds(t, actions, ActionCancelDeadline, ActionStop)

	for _, kind := range allEventKinds() {
		actions, err := m.Step(Event{Kind: kind, At: epoch.Add(2 * time.Second), Generation: generation, Nonce: nonce})
		require.NoError(t, err)
		require.Empty(t, actions)
		require.Equal(t, PhaseStopped, m.State().Phase)
	}
}

func TestTickRequiresNonceAndDoesNotAdvanceGenerationOnError(t *testing.T) {
	m := readyMachine(t)
	_, err := m.Step(Event{Kind: EventTick, At: epoch})
	require.ErrorIs(t, err, ErrMissingNonce)
	require.Equal(t, State{Phase: PhaseIdle}, m.State())
}

func TestGenerationMonotonicallyAdvancesPerChallenge(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "one")
	step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})
	step(t, m, Event{Kind: EventPongReceived, At: epoch.Add(time.Second), Nonce: nonce})

	second, _ := beginChallenge(t, m, epoch.Add(2*time.Second), "two")
	require.Equal(t, generation+1, second)
}

func TestEveryPhaseEventCombinationIsDefined(t *testing.T) {
	for _, phase := range []Phase{PhaseBooting, PhaseIdle, PhaseWriting, PhaseAwaiting, PhaseSuspected, PhaseStopped} {
		for _, kind := range allEventKinds() {
			t.Run(phase.String()+"/event-"+string(rune('0'+kind)), func(t *testing.T) {
				m := newMachine(t)
				m.state = stateForPhase(phase)
				event := Event{Kind: kind, At: epoch.Add(10 * time.Second), Generation: 1, Nonce: "nonce"}
				_, err := m.Step(event)
				if phase == PhaseIdle && kind == EventTick {
					require.NoError(t, err)
				} else {
					require.NoError(t, err)
				}
				assertStateWellFormed(t, m.State())
			})
		}
	}
}

func FuzzMachinePreservesInvariants(f *testing.F) {
	f.Add([]byte{0, 1, 2, 4, 5, 6})
	f.Add([]byte{0, 1, 3, 6})
	f.Fuzz(func(t *testing.T, input []byte) {
		m, err := New(Config{PongTimeout: 5 * time.Second})
		require.NoError(t, err)
		lastGeneration := uint64(0)
		for i, raw := range input {
			kind := EventKind(raw % byte(len(allEventKinds())))
			state := m.State()
			generation := state.Generation
			nonce := "nonce"
			if kind == EventTick {
				nonce = "next"
			}
			at := epoch.Add(time.Duration(i+20) * time.Second)
			_, _ = m.Step(Event{Kind: kind, At: at, Generation: generation, Nonce: nonce})
			current := m.State()
			require.GreaterOrEqual(t, current.Generation, lastGeneration)
			assertStateWellFormed(t, current)
			lastGeneration = current.Generation
		}
	})
}

func newMachine(t *testing.T) *Machine {
	t.Helper()
	m, err := New(Config{PongTimeout: 5 * time.Second})
	require.NoError(t, err)
	return m
}

func readyMachine(t *testing.T) *Machine {
	t.Helper()
	m := newMachine(t)
	step(t, m, Event{Kind: EventReady, At: epoch})
	return m
}

func beginChallenge(t *testing.T, m *Machine, at time.Time, nonce string) (uint64, string) {
	t.Helper()
	actions := step(t, m, Event{Kind: EventTick, At: at, Nonce: nonce})
	requireActionKinds(t, actions, ActionSendPing)
	return actions[0].Generation, actions[0].Nonce
}

func step(t *testing.T, m *Machine, event Event) []Action {
	t.Helper()
	actions, err := m.Step(event)
	require.NoError(t, err)
	assertStateWellFormed(t, m.State())
	return actions
}

func requireActionKinds(t *testing.T, actions []Action, want ...ActionKind) {
	t.Helper()
	require.Len(t, actions, len(want))
	for i := range want {
		require.Equal(t, want[i], actions[i].Kind)
	}
}

func assertStateWellFormed(t *testing.T, state State) {
	t.Helper()
	switch state.Phase {
	case PhaseWriting:
		require.NotZero(t, state.Generation)
		require.NotEmpty(t, state.Nonce)
		require.True(t, state.WrittenAt.IsZero())
		require.True(t, state.Deadline.IsZero())
	case PhaseAwaiting:
		require.NotZero(t, state.Generation)
		require.NotEmpty(t, state.Nonce)
		require.False(t, state.WrittenAt.IsZero())
		require.Equal(t, state.WrittenAt.Add(5*time.Second), state.Deadline)
	case PhaseBooting, PhaseIdle, PhaseSuspected, PhaseStopped:
	default:
		t.Fatalf("invalid phase %d", state.Phase)
	}
}

func stateForPhase(phase Phase) State {
	switch phase {
	case PhaseWriting:
		return State{Phase: phase, Generation: 1, Nonce: "nonce"}
	case PhaseAwaiting:
		return State{Phase: phase, Generation: 1, Nonce: "nonce", WrittenAt: epoch, Deadline: epoch.Add(5 * time.Second)}
	case PhaseBooting, PhaseIdle, PhaseSuspected, PhaseStopped:
		return State{Phase: phase, Generation: 1}
	default:
		return State{Phase: phase, Generation: 1}
	}
}

func allEventKinds() []EventKind {
	return []EventKind{EventReady, EventTick, EventPingWritten, EventPingWriteFailed, EventPongReceived, EventDeadlineElapsed, EventStop}
}
