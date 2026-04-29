package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) newPhase2State() (*phase2State, error) {
	state := &phase2State{
		streamMode:      "derived",
		publishCounters: map[string]int{},
		sessionMeta:     map[string]map[string]any{},
		messages:        map[string]*phase2MessageRecord{},
		messageOrder:    []string{},
		ordinals:        map[string][]uint64{},
		fanout:          map[string][]namedPayload{},
	}
	reg := sessionstream.NewSchemaRegistry()
	for _, err := range []error{
		reg.RegisterCommand(phase2CommandName, &structpb.Struct{}),
		reg.RegisterEvent(phase2EventName, &structpb.Struct{}),
		reg.RegisterUIEvent(phase2UIEventName, &structpb.Struct{}),
		reg.RegisterTimelineEntity(phase2TimelineEntity, &structpb.Struct{}),
	} {
		if err != nil {
			return nil, err
		}
	}

	store, err := storesqlite.NewInMemory(reg)
	if err != nil {
		return nil, err
	}

	pubsub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 256}, watermill.NopLogger{})
	hub, err := sessionstream.NewHub(
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithSessionMetadataFactory(func(_ context.Context, sid sessionstream.SessionId) (any, error) {
			meta := map[string]any{
				"sessionId": string(sid),
				"createdBy": "sessionstream-systemlab",
				"lab":       "phase2",
			}
			e.mu.Lock()
			defer e.mu.Unlock()
			if e.phase2 != nil {
				e.phase2.sessionMeta[string(sid)] = cloneMap(meta)
				e.phase2AppendTraceLocked("session", "phase 2 session created", meta)
			}
			return cloneMap(meta), nil
		}),
		sessionstream.WithUIFanout(sessionstream.UIFanoutFunc(func(_ context.Context, sid sessionstream.SessionId, ord uint64, events []sessionstream.UIEvent) error {
			e.mu.Lock()
			defer e.mu.Unlock()
			if e.phase2 == nil {
				return nil
			}
			for _, uiEvent := range events {
				payload := protoStructMap(uiEvent.Payload)
				payload["ordinal"] = fmt.Sprintf("%d", ord)
				e.phase2.fanout[string(sid)] = append(e.phase2.fanout[string(sid)], namedPayload{Name: uiEvent.Name, Payload: payload})
			}
			return nil
		})),
		sessionstream.WithEventBus(pubsub, pubsub,
			sessionstream.WithBusTopic("sessionstream.phase2"),
			sessionstream.WithBusMessageMutator(e.phase2MessageMutator),
			sessionstream.WithBusObserver(sessionstream.BusObserverHooks{
				OnPublished: e.phase2Published,
				OnConsumed:  e.phase2Consumed,
			}),
		),
	)
	if err != nil {
		return nil, err
	}
	if err := hub.RegisterCommand(phase2CommandName, e.handlePhase2Command); err != nil {
		return nil, err
	}
	if err := hub.RegisterUIProjection(sessionstream.UIProjectionFunc(e.phase2UIProjection)); err != nil {
		return nil, err
	}
	if err := hub.RegisterTimelineProjection(sessionstream.TimelineProjectionFunc(e.phase2TimelineProjection)); err != nil {
		return nil, err
	}

	state.hub = hub
	state.store = store
	return state, nil
}

func (e *labEnvironment) startPhase2() error {
	e.mu.Lock()
	state := e.phase2
	e.mu.Unlock()
	if state == nil || state.hub == nil {
		return fmt.Errorf("phase 2 state is not initialized")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := state.hub.Run(ctx); err != nil {
		cancel()
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 != state {
		cancel()
		return nil
	}
	state.cancel = cancel
	state.trace = nil
	state.messages = map[string]*phase2MessageRecord{}
	state.messageOrder = nil
	state.ordinals = map[string][]uint64{}
	state.fanout = map[string][]namedPayload{}
	state.publishCounters = map[string]int{}
	state.syntheticSequence = 0
	state.sessionMeta = map[string]map[string]any{}
	e.phase2AppendTraceLocked("consumer", "phase 2 consumer started", map[string]any{"topic": "sessionstream.phase2"})
	return nil
}

func (e *labEnvironment) shutdownPhase2() error {
	e.mu.Lock()
	state := e.phase2
	e.phase2 = nil
	e.mu.Unlock()
	if state == nil || state.hub == nil {
		return nil
	}
	if state.cancel != nil {
		state.cancel()
	}
	return state.hub.Shutdown(context.Background())
}

func (e *labEnvironment) restartPhase2Consumer() error {
	e.mu.Lock()
	state := e.phase2
	e.mu.Unlock()
	if state == nil || state.hub == nil {
		return fmt.Errorf("phase 2 state is not initialized")
	}
	if state.cancel != nil {
		state.cancel()
	}
	if err := state.hub.Shutdown(context.Background()); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := state.hub.Run(ctx); err != nil {
		cancel()
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 != state {
		cancel()
		return nil
	}
	state.cancel = cancel
	e.phase2AppendTraceLocked("consumer", "phase 2 consumer restarted", nil)
	return nil
}

func (e *labEnvironment) resetPhase2Only() error {
	e.mu.Lock()
	state := e.phase2
	e.mu.Unlock()
	if state == nil {
		return fmt.Errorf("phase 2 state is not initialized")
	}
	if state.cancel != nil {
		state.cancel()
	}
	if err := state.hub.Shutdown(context.Background()); err != nil {
		return err
	}
	newState, err := e.newPhase2State()
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.phase2 = newState
	e.mu.Unlock()
	return e.startPhase2()
}
