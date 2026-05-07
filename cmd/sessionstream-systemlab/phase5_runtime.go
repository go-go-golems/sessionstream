package main

import (
	"context"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	wstransport "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) newPhase5State(mode string, existingDBPath string) (*phase5State, error) {
	state := &phase5State{}
	runtime, err := e.buildPhase5Runtime(mode, existingDBPath)
	if err != nil {
		return nil, err
	}
	state.runtime = runtime
	return state, nil
}

func (e *labEnvironment) buildPhase5Runtime(mode string, existingDBPath string) (*phase5Runtime, error) {
	mode = normalizePhase5Mode(mode)
	reg := sessionstream.NewSchemaRegistry()
	for _, err := range []error{
		reg.RegisterCommand(phase5CommandName, &structpb.Struct{}),
		reg.RegisterEvent(phase5EventName, &structpb.Struct{}),
		reg.RegisterUIEvent(phase5UIEventName, &structpb.Struct{}),
		reg.RegisterTimelineEntity(phase5EntityKind, &structpb.Struct{}),
	} {
		if err != nil {
			return nil, err
		}
	}

	var (
		store   sessionstream.HydrationStore
		closeFn func() error
		dbPath  string
	)
	if mode == "sql" {
		dbPath = existingDBPath
		if dbPath == "" {
			file, err := os.CreateTemp("", "sessionstream-systemlab-phase5-*.sqlite")
			if err != nil {
				return nil, err
			}
			dbPath = file.Name()
			_ = file.Close()
		}
		dsn, err := storesqlite.FileDSN(dbPath)
		if err != nil {
			return nil, err
		}
		sqlStore, err := storesqlite.New(dsn, reg)
		if err != nil {
			return nil, err
		}
		store = sqlStore
		closeFn = sqlStore.Close
	} else {
		memStore, err := storesqlite.NewInMemory(reg)
		if err != nil {
			return nil, err
		}
		store = memStore
		closeFn = memStore.Close
	}

	wsServer, err := wstransport.NewServer(hydrationSnapshotProvider{store: store}, wstransport.WithTransportObserver(newWebsocketTraceObserver(websocketTraceOptions{
		Phase:            5,
		AppendTrace:      e.appendPhase5Trace,
		IncludeUIPayload: true,
	})))
	if err != nil {
		return nil, err
	}

	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 256}, watermill.NopLogger{})
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithUIFanout(wsServer),
		sessionstream.WithEventBus(pubsub, pubsub, sessionstream.WithBusTopic(phase5BusTopic)),
	)
	if err != nil {
		_ = closeFn()
		return nil, err
	}
	if err := hub.RegisterCommand(phase5CommandName, e.handlePhase5Command); err != nil {
		_ = closeFn()
		return nil, err
	}
	if err := hub.RegisterUIProjection(sessionstream.UIProjectionFunc(e.phase5UIProjection)); err != nil {
		_ = closeFn()
		return nil, err
	}
	if err := hub.RegisterTimelineProjection(sessionstream.TimelineProjectionFunc(e.phase5TimelineProjection)); err != nil {
		_ = closeFn()
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := hub.Run(ctx); err != nil {
		cancel()
		_ = closeFn()
		return nil, err
	}
	return &phase5Runtime{mode: mode, hub: hub, ws: wsServer, store: store, close: closeFn, cancel: cancel, dbPath: dbPath, reg: reg}, nil
}

func (e *labEnvironment) resetPhase5Only(mode string) error {
	e.mu.Lock()
	old := e.phase5
	e.mu.Unlock()
	if old != nil && old.runtime != nil {
		_ = e.shutdownPhase5Runtime(old.runtime)
	}
	newState, err := e.newPhase5State(mode, "")
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.phase5 = newState
	e.mu.Unlock()
	return nil
}

func (e *labEnvironment) shutdownPhase5Runtime(runtime *phase5Runtime) error {
	if runtime == nil {
		return nil
	}
	if runtime.cancel != nil {
		runtime.cancel()
	}
	_ = runtime.hub.Shutdown(context.Background())
	if runtime.close != nil {
		return runtime.close()
	}
	return nil
}

func (e *labEnvironment) shutdownPhase5() error {
	e.mu.Lock()
	state := e.phase5
	e.phase5 = nil
	e.mu.Unlock()
	if state == nil {
		return nil
	}
	return e.shutdownPhase5Runtime(state.runtime)
}
