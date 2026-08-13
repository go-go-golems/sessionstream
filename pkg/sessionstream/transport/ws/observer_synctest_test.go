package ws

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestObserverDispatcherSynctestWaitBlocksUntilCallbackReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		callbackEntered := make(chan struct{})
		releaseCallback := make(chan struct{})
		var waitReturned atomic.Bool

		server := &Server{
			observer: TransportObserverFunc(func(context.Context, TransportRecord) {
				close(callbackEntered)
				<-releaseCallback
			}),
		}
		server.startObserverDispatcher()
		server.observe(context.Background(), TransportRecord{Stage: TransportStageConnected})

		synctest.Wait()
		select {
		case <-callbackEntered:
		default:
			t.Fatal("observer callback did not start before the bubble became durably blocked")
		}

		server.stopObserverDispatcher()
		go func() {
			server.waitObserverDispatcher()
			waitReturned.Store(true)
		}()

		synctest.Wait()
		require.False(t, waitReturned.Load(), "Wait returned while the accepted callback was blocked")

		close(releaseCallback)
		synctest.Wait()
		require.True(t, waitReturned.Load(), "Wait did not return after the callback completed")
	})
}

func TestObserverDispatcherSynctestDrainsAfterPanicAndRejectsAfterStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var callbacks atomic.Uint64
		server := &Server{
			observer: TransportObserverFunc(func(_ context.Context, rec TransportRecord) {
				callbacks.Add(1)
				if rec.Ordinal == 1 {
					panic("intentional observer panic")
				}
			}),
		}
		server.startObserverDispatcher()
		server.observe(context.Background(), TransportRecord{Stage: TransportStageConnected, Ordinal: 1})
		server.observe(context.Background(), TransportRecord{Stage: TransportStageDisconnected, Ordinal: 2})
		server.stopObserverDispatcher()
		server.waitObserverDispatcher()

		require.Equal(t, uint64(2), callbacks.Load(), "panic recovery must preserve later accepted delivery")

		server.observe(context.Background(), TransportRecord{Stage: TransportStageConnected, Ordinal: 3})
		synctest.Wait()
		require.Equal(t, uint64(2), callbacks.Load(), "post-stop observation must be rejected")
	})
}
