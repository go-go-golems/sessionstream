package sessionstream

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

type callbackPromise struct {
	Promise   *goja.Promise
	Reentrant bool
}

func (m *moduleRuntime) promiseFromGo(ctx context.Context, label string, run func(context.Context) error) goja.Value {
	promise, resolve, reject := m.vm.NewPromise()
	settle := func(err error) {
		if err != nil {
			_ = reject(m.vm.ToValue(err.Error()))
			return
		}
		_ = resolve(goja.Undefined())
	}
	services, ok := runtimebridge.Lookup(m.vm)
	if !ok || services.Owner == nil {
		settle(run(ctx))
		return m.vm.ToValue(promise)
	}
	go func() {
		err := run(ctx)
		_ = services.PostWithCustomContext(ctx, label+".settle", func(context.Context, *goja.Runtime) {
			settle(err)
		})
	}()
	return m.vm.ToValue(promise)
}

func (m *moduleRuntime) callJSCallback(ctx context.Context, label string, call func(*goja.Runtime) (goja.Value, error), finish func(*goja.Runtime, goja.Value) (any, error)) (any, error) {
	if finish == nil {
		finish = func(*goja.Runtime, goja.Value) (any, error) { return nil, nil }
	}
	if m.runtimeOwner != nil {
		ret, err := m.runtimeOwner.Call(ctx, label, func(callCtx context.Context, vm *goja.Runtime) (any, error) {
			value, err := call(vm)
			if err != nil {
				return nil, err
			}
			if promise, ok := value.Export().(*goja.Promise); ok {
				return callbackPromise{Promise: promise, Reentrant: callCtx == ctx}, nil
			}
			return finish(vm, value)
		})
		if err != nil {
			return nil, err
		}
		if pending, ok := ret.(callbackPromise); ok {
			if pending.Reentrant && pending.Promise.State() == goja.PromiseStatePending {
				return nil, fmt.Errorf("%s returned a pending Promise during a synchronous owner call; use Promise-returning submit/publish from JavaScript or call from Go", label)
			}
			return m.awaitPromise(ctx, label, pending.Promise, finish)
		}
		return ret, nil
	}

	value, err := call(m.vm)
	if err != nil {
		return nil, err
	}
	if promise, ok := value.Export().(*goja.Promise); ok {
		return m.awaitPromiseDirect(ctx, label, promise, finish)
	}
	return finish(m.vm, value)
}

func (m *moduleRuntime) awaitPromise(ctx context.Context, label string, promise *goja.Promise, finish func(*goja.Runtime, goja.Value) (any, error)) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, err := m.runtimeOwner.Call(ctx, label+".promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			snapshot := promiseSnapshot{State: promise.State(), Result: promise.Result()}
			if snapshot.State == goja.PromiseStateFulfilled {
				return finish(vm, snapshot.Result)
			}
			return snapshot, nil
		})
		if err != nil {
			return nil, err
		}
		if _, ok := ret.(promiseSnapshot); !ok {
			return ret, nil
		}
		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateRejected:
			return nil, fmt.Errorf("%s promise rejected: %s", label, jsValueString(snapshot.Result))
		case goja.PromiseStateFulfilled:
			return ret, nil
		}
	}
}

func (m *moduleRuntime) awaitPromiseDirect(ctx context.Context, label string, promise *goja.Promise, finish func(*goja.Runtime, goja.Value) (any, error)) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	snapshot := promiseSnapshot{State: promise.State(), Result: promise.Result()}
	switch snapshot.State {
	case goja.PromiseStatePending:
		return nil, fmt.Errorf("%s returned a pending Promise without a runtime owner", label)
	case goja.PromiseStateRejected:
		return nil, fmt.Errorf("%s promise rejected: %s", label, jsValueString(snapshot.Result))
	case goja.PromiseStateFulfilled:
		return finish(m.vm, snapshot.Result)
	default:
		return nil, fmt.Errorf("%s promise has unknown state %v", label, snapshot.State)
	}
}

func jsValueString(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	return value.String()
}
