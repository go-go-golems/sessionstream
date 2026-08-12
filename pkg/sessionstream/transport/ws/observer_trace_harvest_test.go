package ws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// TestObserverTraceHarvestConcurrent writes production dispatcher traces only
// when SESSIONSTREAM_OBSERVER_TRACE_DIR is set. The external TLC harness runs
// this test under several GOMAXPROCS values and captures runtime/trace output.
func TestObserverTraceHarvestConcurrent(t *testing.T) {
	directory := os.Getenv("SESSIONSTREAM_OBSERVER_TRACE_DIR")
	if directory == "" {
		t.Skip("SESSIONSTREAM_OBSERVER_TRACE_DIR is not set")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	runID := fmt.Sprintf("gomaxprocs-%d", runtime.GOMAXPROCS(0))
	dispatcherID := "ws-observer-1"
	modelFile, err := os.Create(filepath.Join(directory, "model.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = modelFile.Close() }()
	intervalFile, err := os.Create(filepath.Join(directory, "intervals.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = intervalFile.Close() }()
	sink, err := NewObserverJSONLTraceSink(modelFile, intervalFile)
	if err != nil {
		t.Fatal(err)
	}

	server := &Server{
		observer: TransportObserverFunc(func(_ context.Context, record TransportRecord) {
			if record.Ordinal%17 == 0 {
				panic("intentional harvested callback panic")
			}
		}),
		observerTrace: &observerTraceState{config: ObserverTraceConfig{
			RunID: runID, DispatcherID: dispatcherID, Sink: sink,
		}},
	}
	server.startObserverDispatcher()

	const producers = 4
	const submissions = 40
	start := make(chan struct{})
	var producersWG sync.WaitGroup
	for producer := 0; producer < producers; producer++ {
		producersWG.Add(1)
		go func(producer int) {
			defer producersWG.Done()
			<-start
			for i := 0; i < submissions; i++ {
				server.observe(context.Background(), TransportRecord{
					Ordinal: uint64(producer*submissions + i + 1),
				})
			}
		}(producer)
	}
	close(start)

	closeStarted := make(chan struct{})
	go func() {
		<-start
		close(closeStarted)
		server.stopObserverDispatcher()
	}()
	<-closeStarted

	waiters := 3
	var waitWG sync.WaitGroup
	for i := 0; i < waiters; i++ {
		waitWG.Add(1)
		go func() {
			defer waitWG.Done()
			server.waitObserverDispatcher()
		}()
	}
	producersWG.Wait()
	server.stopObserverDispatcher()
	waitWG.Wait()
	if err := sink.Err(); err != nil {
		t.Fatal(err)
	}
}
