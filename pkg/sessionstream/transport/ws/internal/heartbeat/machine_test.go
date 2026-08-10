package heartbeat

import (
	"errors"
	"fmt"
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

func TestMatchingPongProcessedBeforeWriteAckCompletesCycle(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "nonce-1")
	pongAt := epoch.Add(time.Second)
	actions := step(t, m, Event{Kind: EventPongReceived, At: pongAt, Nonce: nonce})
	require.Empty(t, actions)
	require.Equal(t, PhaseWriting, m.State().Phase)
	require.Equal(t, pongAt, m.State().PendingPongAt)

	actions = step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})
	require.Equal(t, PhaseIdle, m.State().Phase)
	requireActionKinds(t, actions, ActionRecordPong, ActionScheduleTick)
}

func TestPendingPongKeepsEarliestMatchingArrival(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "nonce-1")
	first := epoch.Add(time.Second)
	step(t, m, Event{Kind: EventPongReceived, At: first, Nonce: nonce})
	step(t, m, Event{Kind: EventPongReceived, At: epoch.Add(20 * time.Second), Nonce: nonce})
	require.Equal(t, first, m.State().PendingPongAt)

	actions := step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})
	require.Equal(t, PhaseIdle, m.State().Phase)
	requireActionKinds(t, actions, ActionRecordPong, ActionScheduleTick)
}

func TestMatchingPongBeforeWriteCompletionIsAccepted(t *testing.T) {
	m := readyMachine(t)
	generation, nonce := beginChallenge(t, m, epoch, "nonce-1")
	step(t, m, Event{Kind: EventPongReceived, At: epoch.Add(-time.Second), Nonce: nonce})
	actions := step(t, m, Event{Kind: EventPingWritten, At: epoch, Generation: generation, Nonce: nonce})
	require.Equal(t, PhaseIdle, m.State().Phase)
	requireActionKinds(t, actions, ActionRecordPong, ActionScheduleTick)
}

func TestNonmatchingPongWhileWritingIsStale(t *testing.T) {
	m := readyMachine(t)
	_, _ = beginChallenge(t, m, epoch, "current")
	actions := step(t, m, Event{Kind: EventPongReceived, At: epoch, Nonce: "old"})
	require.Equal(t, PhaseWriting, m.State().Phase)
	require.True(t, m.State().PendingPongAt.IsZero())
	requireActionKinds(t, actions, ActionRecordStalePong)
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

const fuzzMaxOperations = 4096

type fuzzOperation uint8

const (
	fuzzAdvance fuzzOperation = iota
	fuzzReady
	fuzzTick
	fuzzPingWritten
	fuzzPingWriteFailed
	fuzzPongReceived
	fuzzDeadlineElapsed
	fuzzStop
)

type fuzzIdentityMode uint8

const (
	fuzzIdentityCurrent fuzzIdentityMode = iota
	fuzzIdentityPrevious
	fuzzIdentityFutureStale
	fuzzIdentityEmpty
)

type fuzzTimeMode uint8

const (
	fuzzTimeNext fuzzTimeMode = iota
	fuzzTimeBeforeDeadline
	fuzzTimeAtDeadline
	fuzzTimeAfterDeadline
	fuzzTimeAtWrite
	fuzzTimeSame
	fuzzTimeAfterTimeout
	fuzzTimeAfterSecond
)

func TestHeartbeatFuzzAdvanceTraversesTwoHealthyCycles(t *testing.T) {
	m := newMachine(t)
	cursor := epoch
	advance := fuzzByte(fuzzAdvance, fuzzIdentityCurrent, fuzzTimeNext)
	for range 7 {
		event, nextCursor := decodeFuzzEvent(advance, m.State(), cursor)
		actions, err := m.Step(event)
		require.NoError(t, err)
		assertFuzzActionContracts(t, m.State(), actions)
		cursor = nextCursor
	}
	require.Equal(t, PhaseIdle, m.State().Phase)
	require.Equal(t, uint64(2), m.State().Generation)
}

func FuzzMachinePreservesInvariants(f *testing.F) {
	addHeartbeatFuzzSeeds(f)
	f.Fuzz(func(t *testing.T, input []byte) {
		m, err := New(Config{PongTimeout: 5 * time.Second})
		require.NoError(t, err)
		cursor := epoch
		lastGeneration := uint64(0)
		if len(input) > fuzzMaxOperations {
			input = input[:fuzzMaxOperations]
		}
		for _, raw := range input {
			before := m.State()
			event, nextCursor := decodeFuzzEvent(raw, before, cursor)
			actions, stepErr := m.Step(event)
			after := m.State()

			if stepErr != nil {
				require.True(t, errors.Is(stepErr, ErrMissingNonce) || errors.Is(stepErr, ErrEarlyDeadline), "unexpected reducer error: %v", stepErr)
				require.Equal(t, before, after, "expected input errors must be atomic")
				require.Empty(t, actions)
			} else {
				assertFuzzActionContracts(t, after, actions)
			}
			if before.Phase == PhaseStopped {
				require.Equal(t, before, after, "stopped must be absorbing")
				require.Empty(t, actions)
			}
			require.GreaterOrEqual(t, after.Generation, lastGeneration)
			assertStateWellFormed(t, after)
			cursor = nextCursor
			lastGeneration = after.Generation
		}
	})
}

func addHeartbeatFuzzSeeds(f *testing.F) {
	advance := fuzzByte(fuzzAdvance, fuzzIdentityCurrent, fuzzTimeNext)
	f.Add([]byte{advance, advance, advance, advance, advance, advance, advance, advance}) // two healthy cycles
	f.Add([]byte{advance, advance, fuzzByte(fuzzPongReceived, fuzzIdentityCurrent, fuzzTimeNext), advance})
	f.Add([]byte{advance, advance, advance, fuzzByte(fuzzPongReceived, fuzzIdentityFutureStale, fuzzTimeNext), advance})
	f.Add([]byte{advance, advance, advance, fuzzByte(fuzzDeadlineElapsed, fuzzIdentityCurrent, fuzzTimeBeforeDeadline), fuzzByte(fuzzDeadlineElapsed, fuzzIdentityCurrent, fuzzTimeAtDeadline)})
	f.Add([]byte{advance, advance, fuzzByte(fuzzPingWriteFailed, fuzzIdentityCurrent, fuzzTimeNext)})
	f.Add([]byte{advance, advance, advance, fuzzByte(fuzzStop, fuzzIdentityCurrent, fuzzTimeNext)})
	f.Add([]byte{advance, fuzzByte(fuzzStop, fuzzIdentityCurrent, fuzzTimeNext), advance, fuzzByte(fuzzPongReceived, fuzzIdentityCurrent, fuzzTimeAfterDeadline)})
}

func fuzzByte(operation fuzzOperation, identity fuzzIdentityMode, timeMode fuzzTimeMode) byte {
	return byte(operation&0x07) | byte(identity&0x03)<<3 | byte(timeMode&0x07)<<5
}

func decodeFuzzEvent(raw byte, state State, cursor time.Time) (Event, time.Time) {
	operation := fuzzOperation(raw & 0x07)
	identity := fuzzIdentityMode((raw >> 3) & 0x03)
	timeMode := fuzzTimeMode((raw >> 5) & 0x07)
	at := fuzzEventTime(timeMode, state, cursor)
	generation, nonce := fuzzEventIdentity(identity, state)
	kind := fuzzEventKind(operation)

	if operation == fuzzAdvance {
		kind, generation, nonce, at = fuzzAdvanceEvent(state, cursor)
	} else if kind == EventTick {
		switch identity {
		case fuzzIdentityCurrent:
			nonce = fmt.Sprintf("fuzz-%d", state.Generation+1)
		case fuzzIdentityPrevious:
			nonce = fmt.Sprintf("stale-%d", state.Generation)
		case fuzzIdentityFutureStale:
			nonce = fmt.Sprintf("future-%d", state.Generation+2)
		case fuzzIdentityEmpty:
			nonce = ""
		default:
			nonce = ""
		}
	}
	if at.After(cursor) {
		cursor = at
	}
	return Event{Kind: kind, At: at, Generation: generation, Nonce: nonce, Err: errors.New("fuzz write failure")}, cursor
}

func fuzzEventKind(operation fuzzOperation) EventKind {
	switch operation {
	case fuzzAdvance, fuzzReady:
		return EventReady
	case fuzzTick:
		return EventTick
	case fuzzPingWritten:
		return EventPingWritten
	case fuzzPingWriteFailed:
		return EventPingWriteFailed
	case fuzzPongReceived:
		return EventPongReceived
	case fuzzDeadlineElapsed:
		return EventDeadlineElapsed
	case fuzzStop:
		return EventStop
	default:
		return EventStop
	}
}

func fuzzAdvanceEvent(state State, cursor time.Time) (EventKind, uint64, string, time.Time) {
	next := cursor.Add(time.Millisecond)
	switch state.Phase {
	case PhaseBooting:
		return EventReady, state.Generation, "", next
	case PhaseIdle:
		return EventTick, state.Generation, fmt.Sprintf("fuzz-%d", state.Generation+1), next
	case PhaseWriting:
		return EventPingWritten, state.Generation, state.Nonce, next
	case PhaseAwaiting:
		return EventPongReceived, state.Generation, state.Nonce, state.Deadline.Add(-time.Nanosecond)
	case PhaseSuspected, PhaseStopped:
		return EventStop, state.Generation, state.Nonce, next
	default:
		return EventStop, state.Generation, state.Nonce, next
	}
}

func fuzzEventIdentity(mode fuzzIdentityMode, state State) (uint64, string) {
	switch mode {
	case fuzzIdentityCurrent:
		return state.Generation, state.Nonce
	case fuzzIdentityPrevious:
		generation := state.Generation
		if generation > 0 {
			generation--
		}
		return generation, state.Nonce
	case fuzzIdentityFutureStale:
		return state.Generation + 1, "stale"
	case fuzzIdentityEmpty:
		return state.Generation, ""
	default:
		return state.Generation, state.Nonce
	}
}

func fuzzEventTime(mode fuzzTimeMode, state State, cursor time.Time) time.Time {
	switch mode {
	case fuzzTimeNext:
		return cursor.Add(time.Millisecond)
	case fuzzTimeBeforeDeadline:
		if !state.Deadline.IsZero() {
			return state.Deadline.Add(-time.Nanosecond)
		}
		return cursor.Add(time.Millisecond)
	case fuzzTimeAtDeadline:
		if !state.Deadline.IsZero() {
			return state.Deadline
		}
		return cursor
	case fuzzTimeAfterDeadline:
		if !state.Deadline.IsZero() {
			return state.Deadline.Add(time.Nanosecond)
		}
		return cursor.Add(5*time.Second + time.Nanosecond)
	case fuzzTimeAtWrite:
		if !state.WrittenAt.IsZero() {
			return state.WrittenAt
		}
		return cursor
	case fuzzTimeSame:
		return cursor
	case fuzzTimeAfterTimeout:
		return cursor.Add(5 * time.Second)
	case fuzzTimeAfterSecond:
		return cursor.Add(time.Second)
	default:
		return cursor
	}
}

func assertFuzzActionContracts(t *testing.T, state State, actions []Action) {
	t.Helper()
	suspected := 0
	closed := 0
	for _, action := range actions {
		if action.Generation != 0 {
			require.LessOrEqual(t, action.Generation, state.Generation)
		}
		switch action.Kind {
		case ActionSendPing:
			require.Equal(t, PhaseWriting, state.Phase)
			require.Equal(t, state.Generation, action.Generation)
			require.Equal(t, state.Nonce, action.Nonce)
		case ActionArmDeadline:
			require.Equal(t, PhaseAwaiting, state.Phase)
			require.Equal(t, state.Generation, action.Generation)
			require.Equal(t, state.Deadline, action.Deadline)
		case ActionRecordPong, ActionScheduleTick:
			require.Equal(t, PhaseIdle, state.Phase)
		case ActionRecordSuspected:
			suspected++
			require.Equal(t, PhaseSuspected, state.Phase)
		case ActionCloseConnection:
			closed++
			require.Equal(t, PhaseSuspected, state.Phase)
		case ActionStop:
			require.Equal(t, PhaseStopped, state.Phase)
		case ActionCancelDeadline, ActionRecordStalePong:
		default:
			t.Fatalf("unknown action kind %d", action.Kind)
		}
	}
	require.Equal(t, suspected, closed, "suspicion and close actions must be paired")
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
