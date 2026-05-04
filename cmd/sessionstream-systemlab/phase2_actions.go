package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/types/known/structpb"
)

func (e *labEnvironment) RunPhase2(ctx context.Context, in phase2RunRequest) (phase2RunResponse, error) {
	action := strings.TrimSpace(in.Action)
	if action == "" {
		action = "publish-a"
	}
	sessionA := strings.TrimSpace(in.SessionA)
	if sessionA == "" {
		sessionA = "s-a"
	}
	sessionB := strings.TrimSpace(in.SessionB)
	if sessionB == "" {
		sessionB = "s-b"
	}
	burstCount := in.BurstCount
	if burstCount <= 0 {
		burstCount = 4
	}
	streamMode := normalizeStreamMode(in.StreamMode)

	e.mu.Lock()
	if e.phase2 == nil {
		e.mu.Unlock()
		return phase2RunResponse{}, fmt.Errorf("phase 2 state is not initialized")
	}
	e.phase2.streamMode = streamMode
	beforeA := len(e.phase2.ordinals[sessionA])
	beforeB := len(e.phase2.ordinals[sessionB])
	e.phase2AppendTraceLocked("control", "phase 2 action requested", map[string]any{
		"action":     action,
		"sessionA":   sessionA,
		"sessionB":   sessionB,
		"burstCount": burstCount,
		"streamMode": streamMode,
	})
	hub := e.phase2.hub
	e.mu.Unlock()

	var err error
	switch action {
	case "publish-a":
		err = e.submitPhase2Command(ctx, hub, sessionstream.SessionId(sessionA), e.nextPhase2Label(sessionA))
		if err == nil {
			err = e.waitForPhase2Consumed(sessionA, beforeA+1)
		}
	case "publish-b":
		err = e.submitPhase2Command(ctx, hub, sessionstream.SessionId(sessionB), e.nextPhase2Label(sessionB))
		if err == nil {
			err = e.waitForPhase2Consumed(sessionB, beforeB+1)
		}
	case "burst-a":
		for i := 0; i < burstCount; i++ {
			if err = e.submitPhase2Command(ctx, hub, sessionstream.SessionId(sessionA), e.nextPhase2Label(sessionA)); err != nil {
				break
			}
		}
		if err == nil {
			err = e.waitForPhase2Consumed(sessionA, beforeA+burstCount)
		}
	case "restart-consumer":
		err = e.restartPhase2Consumer()
	case "reset-phase2":
		err = e.resetPhase2Only()
	default:
		err = fmt.Errorf("unknown phase 2 action %q", action)
	}

	resp, buildErr := e.buildPhase2Response(action, sessionA, sessionB, burstCount, streamMode)
	if buildErr != nil {
		return phase2RunResponse{}, buildErr
	}
	if err != nil {
		resp.Error = err.Error()
	}

	e.mu.Lock()
	if e.phase2 != nil {
		e.phase2.lastRun = clonePhase2RunResponse(resp)
	}
	e.mu.Unlock()
	return resp, nil
}

func (e *labEnvironment) submitPhase2Command(ctx context.Context, hub *sessionstream.Hub, sid sessionstream.SessionId, label string) error {
	payload, err := structpb.NewStruct(map[string]any{"label": label})
	if err != nil {
		return err
	}
	return hub.Submit(ctx, sid, phase2CommandName, payload)
}

func (e *labEnvironment) nextPhase2Label(sessionID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase2 == nil {
		return sessionID + "-1"
	}
	e.phase2.publishCounters[sessionID]++
	return fmt.Sprintf("%s-%02d", sessionID, e.phase2.publishCounters[sessionID])
}

func (e *labEnvironment) waitForPhase2Consumed(sessionID string, want int) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		count := 0
		if e.phase2 != nil {
			count = len(e.phase2.ordinals[sessionID])
		}
		e.mu.Unlock()
		if count >= want {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for phase 2 consumed count %d for session %q", want, sessionID)
}

func (e *labEnvironment) buildPhase2Response(action, sessionA, sessionB string, burstCount int, streamMode string) (phase2RunResponse, error) {
	e.mu.Lock()
	state := e.phase2
	if state == nil {
		e.mu.Unlock()
		return phase2RunResponse{}, fmt.Errorf("phase 2 state is not initialized")
	}
	trace := append([]traceEntry(nil), state.trace...)
	messageHistoryRaw := make([]phase2MessageRecord, 0, len(state.messageOrder))
	for _, id := range state.messageOrder {
		if record := state.messages[id]; record != nil {
			messageHistoryRaw = append(messageHistoryRaw, clonePhase2MessageRecord(*record))
		}
	}
	ordinalsRaw := clonePhase2Ordinals(state.ordinals)
	fanout := cloneNamedPayloadMap(state.fanout)
	hub := state.hub
	e.mu.Unlock()

	snapshots := map[string]map[string]any{}
	for _, sid := range uniqueStrings(sessionA, sessionB) {
		snap, err := hub.Snapshot(context.Background(), sessionstream.SessionId(sid))
		if err != nil {
			return phase2RunResponse{}, err
		}
		encoded := encodeSnapshot(snap)
		encoded["ordinal"] = fmt.Sprintf("%d", snap.SnapshotOrdinal)
		snapshots[sid] = encoded
	}

	resp := phase2RunResponse{
		Action:             action,
		SessionA:           sessionA,
		SessionB:           sessionB,
		BurstCount:         burstCount,
		StreamMode:         streamMode,
		Trace:              trace,
		MessageHistory:     phase2MessageHistoryView(messageHistoryRaw),
		PerSessionOrdinals: phase2OrdinalStrings(ordinalsRaw),
		Fanout:             fanout,
		Snapshots:          snapshots,
		Checks: map[string]bool{
			"publishOrdinalZero":  phase2PublishOrdinalsZero(messageHistoryRaw),
			"monotonicPerSession": phase2Monotonic(ordinalsRaw),
			"sessionIsolation":    phase2SessionIsolation(ordinalsRaw),
			"messagesConsumed":    phase2ConsumedCount(messageHistoryRaw) > 0,
		},
	}
	return resp, nil
}

func (e *labEnvironment) ExportPhase2(format string) (string, string, []byte, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "json"
	}
	e.mu.Lock()
	state := e.phase2
	if state == nil {
		e.mu.Unlock()
		return "", "", nil, fmt.Errorf("phase 2 state is not initialized")
	}
	resp := clonePhase2RunResponse(state.lastRun)
	e.mu.Unlock()
	if resp.Action == "" {
		return "", "", nil, fmt.Errorf("no phase 2 transcript available")
	}
	switch format {
	case "json":
		body, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return "", "", nil, err
		}
		return "phase2-transcript.json", "application/json", body, nil
	case "md", "markdown":
		return "phase2-transcript.md", "text/markdown; charset=utf-8", []byte(renderPhase2Markdown(resp)), nil
	default:
		return "", "", nil, fmt.Errorf("unsupported export format %q", format)
	}
}
