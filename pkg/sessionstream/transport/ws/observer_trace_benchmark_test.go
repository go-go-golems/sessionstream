package ws

import (
	"context"
	"testing"
)

func BenchmarkObserverDispatchTracingDisabled(b *testing.B) {
	server := &Server{
		observer:      TransportObserverFunc(func(context.Context, TransportRecord) {}),
		observerQueue: make(chan observedTransportRecord, b.N+1),
	}
	ctx := context.Background()
	record := TransportRecord{Stage: TransportStageConnected}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.observe(ctx, record)
	}
}

func BenchmarkObserverDispatchTracingEnabled(b *testing.B) {
	sink := &discardObserverTraceSink{}
	server := &Server{
		observer:      TransportObserverFunc(func(context.Context, TransportRecord) {}),
		observerQueue: make(chan observedTransportRecord, b.N+1),
		observerTrace: &observerTraceState{config: ObserverTraceConfig{
			RunID: "benchmark", DispatcherID: "observer", Sink: sink,
		}},
	}
	server.observerTrace.start(context.Background())
	defer server.observerTrace.finish()
	ctx := context.Background()
	record := TransportRecord{Stage: TransportStageConnected}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.observe(ctx, record)
	}
}

type discardObserverTraceSink struct{}

func (*discardObserverTraceSink) OnObserverModelEvent(ObserverModelEvent)       {}
func (*discardObserverTraceSink) OnObserverIntervalEvent(ObserverIntervalEvent) {}
