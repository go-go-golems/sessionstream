package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

type replayInspectResponse struct {
	Phase          string              `json:"phase"`
	SessionID      string              `json:"sessionId"`
	EventCursor    uint64              `json:"eventCursor"`
	TimelineCursor uint64              `json:"timelineCursor"`
	Errors         []replayErrorRecord `json:"errors"`
}

type replayErrorRecord struct {
	Kind             string            `json:"kind"`
	SessionID        string            `json:"sessionId,omitempty"`
	Ordinal          uint64            `json:"ordinal,omitempty"`
	EventName        string            `json:"eventName,omitempty"`
	Error            string            `json:"error,omitempty"`
	RawMessageBase64 string            `json:"rawMessageBase64,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func (s *systemlabServer) handlePhase5ReplayInspect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	sessionID := req.URL.Query().Get("sessionId")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "sessionId is required"})
		return
	}
	limit := 25
	if raw := req.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "limit must be a non-negative integer"})
			return
		}
		limit = parsed
	}
	resp, err := s.env.inspectPhase5Replay(req.Context(), sessionstream.SessionId(sessionID), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (e *labEnvironment) inspectPhase5Replay(ctx context.Context, sid sessionstream.SessionId, limit int) (replayInspectResponse, error) {
	e.mu.Lock()
	state := e.phase5
	e.mu.Unlock()
	if state == nil || state.runtime == nil || state.runtime.hub == nil {
		return replayInspectResponse{}, fmt.Errorf("phase 5 runtime is not initialized")
	}
	hub := state.runtime.hub
	eventCursor, err := hub.EventCursor(ctx, sid)
	if err != nil {
		return replayInspectResponse{}, err
	}
	timelineCursor, err := hub.ProjectionCursor(ctx, sessionstream.TimelineProjectorName, sid)
	if err != nil {
		return replayInspectResponse{}, err
	}
	var errorsOut []replayErrorRecord
	if errorStore, ok := state.runtime.store.(sessionstream.ErrorRecordStore); ok {
		records, err := errorStore.ErrorRecords(ctx, sid, limit)
		if err != nil {
			return replayInspectResponse{}, err
		}
		errorsOut = encodeReplayErrors(records)
	}
	return replayInspectResponse{
		Phase:          "phase5",
		SessionID:      string(sid),
		EventCursor:    eventCursor,
		TimelineCursor: timelineCursor,
		Errors:         errorsOut,
	}, nil
}

func encodeReplayErrors(records []sessionstream.ErrorRecord) []replayErrorRecord {
	if len(records) == 0 {
		return []replayErrorRecord{}
	}
	out := make([]replayErrorRecord, 0, len(records))
	for _, rec := range records {
		errText := ""
		if rec.Err != nil {
			errText = rec.Err.Error()
		}
		raw := ""
		if len(rec.RawMessage) > 0 {
			raw = base64.StdEncoding.EncodeToString(rec.RawMessage)
		}
		out = append(out, replayErrorRecord{
			Kind:             string(rec.Kind),
			SessionID:        string(rec.SessionId),
			Ordinal:          rec.Ordinal,
			EventName:        rec.EventName,
			Error:            errText,
			RawMessageBase64: raw,
			Metadata:         cloneReplayStringMap(rec.Metadata),
		})
	}
	return out
}

func cloneReplayStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
