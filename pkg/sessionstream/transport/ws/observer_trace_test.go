package ws

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

type recordingObserverTraceSink struct {
	mu        sync.Mutex
	model     []ObserverModelEvent
	intervals []ObserverIntervalEvent
}

func (s *recordingObserverTraceSink) OnObserverModelEvent(event ObserverModelEvent) {
	s.mu.Lock()
	s.model = append(s.model, event)
	s.mu.Unlock()
}

func (s *recordingObserverTraceSink) OnObserverIntervalEvent(event ObserverIntervalEvent) {
	s.mu.Lock()
	s.intervals = append(s.intervals, event)
	s.mu.Unlock()
}

func (s *recordingObserverTraceSink) snapshot() ([]ObserverModelEvent, []ObserverIntervalEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ObserverModelEvent(nil), s.model...), append([]ObserverIntervalEvent(nil), s.intervals...)
}

func TestObserverTraceRequiresPartitionIdentity(t *testing.T) {
	sink := &recordingObserverTraceSink{}
	server := &Server{}
	require.Error(t, WithObserverTrace(ObserverTraceConfig{DispatcherID: "d", Sink: sink})(server))
	require.Error(t, WithObserverTrace(ObserverTraceConfig{RunID: "r", Sink: sink})(server))
	require.NoError(t, WithObserverTrace(ObserverTraceConfig{})(server))
	require.Nil(t, server.observerTrace)
}

func TestObserverTraceProductionLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sink := &recordingObserverTraceSink{}
		callbackEntered := make(chan struct{})
		release := make(chan struct{})
		server := &Server{
			observer: TransportObserverFunc(func(_ context.Context, record TransportRecord) {
				if record.Ordinal == 1 {
					close(callbackEntered)
					<-release
				}
				if record.Ordinal == 2 {
					panic("trace panic")
				}
			}),
			observerTrace: &observerTraceState{config: ObserverTraceConfig{
				RunID:        "run-production",
				DispatcherID: "observer-production",
				Sink:         sink,
			}},
		}
		server.startObserverDispatcher()
		server.observe(context.Background(), TransportRecord{Ordinal: 1})
		synctest.Wait()
		select {
		case <-callbackEntered:
		default:
			t.Fatal("first callback did not enter")
		}
		server.observe(context.Background(), TransportRecord{Ordinal: 2})
		server.observe(context.Background(), TransportRecord{Ordinal: 3})
		server.stopObserverDispatcher()
		close(release)
		server.waitObserverDispatcher()

		model, intervals := sink.snapshot()
		require.NotEmpty(t, model)
		require.NotEmpty(t, intervals)

		actions := map[string]int{}
		linearizations := map[string]map[string]bool{}
		for i, event := range model {
			require.Equal(t, uint64(i+1), event.Sequence)
			require.Equal(t, "run-production", event.RunID)
			require.Equal(t, "observer-production", event.DispatcherID)
			require.NotEmpty(t, event.OperationID)
			require.NotEmpty(t, event.Operation)
			actions[event.Action]++
		}
		for i, event := range intervals {
			require.Equal(t, uint64(i+1), event.Sequence)
			require.Equal(t, "run-production", event.RunID)
			require.Equal(t, "observer-production", event.DispatcherID)
			if event.Phase == "linearize" {
				if linearizations[event.OperationID] == nil {
					linearizations[event.OperationID] = map[string]bool{}
				}
				linearizations[event.OperationID][event.Action] = true
			}
		}
		for _, event := range model {
			require.True(t, linearizations[event.OperationID][event.Action], "missing linked interval event for %+v", event)
		}

		require.GreaterOrEqual(t, actions["submit_accepted"], 3)
		require.GreaterOrEqual(t, actions["receive"], 3)
		require.Equal(t, 1, actions["panic_recovered"])
		require.Equal(t, 2, actions["offered"])
		require.Equal(t, 1, actions["close_effective"])
		require.Equal(t, 1, actions["worker_exit"])
		require.Equal(t, 1, actions["wait_returned"])
	})
}
