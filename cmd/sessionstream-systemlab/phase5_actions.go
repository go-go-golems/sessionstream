package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) RunPhase5(ctx context.Context, in phase5RunRequest) (phase5RunResponse, error) {
	action := strings.TrimSpace(in.Action)
	if action == "" {
		action = "state"
	}
	mode := normalizePhase5Mode(in.Mode)
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		sessionID = "persist-demo"
	}
	text := strings.TrimSpace(in.Text)
	if text == "" {
		text = "persist this record"
	}

	e.mu.Lock()
	state := e.phase5
	e.mu.Unlock()
	if state == nil || state.runtime == nil || state.runtime.mode != mode {
		if err := e.resetPhase5Only(mode); err != nil {
			return phase5RunResponse{}, err
		}
		e.mu.Lock()
		state = e.phase5
		e.mu.Unlock()
	}

	var err error
	switch action {
	case "seed-session":
		e.appendPhase5Trace("control", "phase 5 seed requested", map[string]any{"mode": mode, "sessionId": sessionID, "text": text})
		payload, buildErr := structpb.NewStruct(map[string]any{"text": text})
		if buildErr != nil {
			return phase5RunResponse{}, buildErr
		}
		before, _ := state.runtime.hub.Cursor(ctx, sessionstream.SessionId(sessionID))
		err = state.runtime.hub.Submit(ctx, sessionstream.SessionId(sessionID), phase5CommandName, payload)
		if err == nil {
			err = e.waitForPhase5Cursor(sessionID, before+1)
		}
	case "restart-backend":
		pre, buildErr := e.phase5SnapshotFor(state.runtime, sessionID)
		if buildErr != nil {
			return phase5RunResponse{}, buildErr
		}
		e.mu.Lock()
		if e.phase5 != nil {
			e.phase5.preRestart = pre
		}
		e.mu.Unlock()
		dbPath := ""
		if mode == "sql" {
			dbPath = state.runtime.dbPath
		}
		if err = e.shutdownPhase5Runtime(state.runtime); err != nil {
			return phase5RunResponse{}, err
		}
		newState, buildErr := e.newPhase5State(mode, dbPath)
		if buildErr != nil {
			return phase5RunResponse{}, buildErr
		}
		e.mu.Lock()
		newState.preRestart = pre
		newState.trace = append([]traceEntry(nil), state.trace...)
		newState.seq = state.seq
		newState.trace = append(newState.trace, traceEntry{Step: len(newState.trace) + 1, Kind: "control", Message: "phase 5 backend restarted", Details: map[string]any{"mode": mode}})
		e.phase5 = newState
		e.mu.Unlock()
	case "reset-phase5":
		err = e.resetPhase5Only(mode)
	case "state":
		// no-op
	default:
		err = fmt.Errorf("unknown phase 5 action %q", action)
	}

	resp, buildErr := e.buildPhase5Response(action, mode, sessionID, text)
	if buildErr != nil {
		return phase5RunResponse{}, buildErr
	}
	if err != nil {
		resp.Error = err.Error()
	}
	e.mu.Lock()
	if e.phase5 != nil {
		e.phase5.lastRun = clonePhase5RunResponse(resp)
	}
	e.mu.Unlock()
	return resp, nil
}

func (e *labEnvironment) buildPhase5Response(action, mode, sessionID, text string) (phase5RunResponse, error) {
	e.mu.Lock()
	state := e.phase5
	if state == nil || state.runtime == nil {
		e.mu.Unlock()
		return phase5RunResponse{}, fmt.Errorf("phase 5 state is not initialized")
	}
	trace := append([]traceEntry(nil), state.trace...)
	preRestart := cloneMap(state.preRestart)
	runtime := state.runtime
	e.mu.Unlock()
	postRestart, err := e.phase5SnapshotFor(runtime, sessionID)
	if err != nil {
		return phase5RunResponse{}, err
	}
	checks := map[string]bool{
		"cursorPreserved":   phase5CursorPreserved(mode, preRestart, postRestart),
		"entitiesPreserved": phase5EntitiesPreserved(mode, preRestart, postRestart),
		"resumeWithoutGaps": phase5ResumeWithoutGaps(trace, sessionID),
		"modeIsSQL":         mode == "sql",
	}
	return phase5RunResponse{
		Action:      action,
		Mode:        mode,
		SessionID:   sessionID,
		Text:        text,
		Trace:       trace,
		Connections: runtime.ws.Connections(),
		PreRestart:  preRestart,
		PostRestart: postRestart,
		Checks:      checks,
	}, nil
}

func (e *labEnvironment) phase5SnapshotFor(runtime *phase5Runtime, sessionID string) (map[string]any, error) {
	snap, err := runtime.hub.Snapshot(context.Background(), sessionstream.SessionId(sessionID))
	if err != nil {
		return nil, err
	}
	encoded := encodeSnapshot(snap)
	encoded["ordinal"] = fmt.Sprintf("%d", snap.Ordinal)
	return encoded, nil
}

func (e *labEnvironment) waitForPhase5Cursor(sessionID string, want uint64) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		state := e.phase5
		e.mu.Unlock()
		if state == nil || state.runtime == nil {
			return fmt.Errorf("phase 5 runtime is not initialized")
		}
		cursor, err := state.runtime.hub.Cursor(context.Background(), sessionstream.SessionId(sessionID))
		if err != nil {
			return err
		}
		if cursor >= want {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for phase 5 cursor %d", want)
}
