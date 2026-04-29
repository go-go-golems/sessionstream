package main

import (
	"context"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	wstransport "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

const (
	phase5CommandName = "PersistRecord"
	phase5EventName   = "RecordPersisted"
	phase5UIEventName = "RecordObserved"
	phase5EntityKind  = "RecordEntity"
	phase5BusTopic    = "sessionstream.phase5"
)

type phase5RunRequest struct {
	Action    string `json:"action"`
	Mode      string `json:"mode"`
	SessionID string `json:"sessionId"`
	Text      string `json:"text"`
}

type phase5RunResponse struct {
	Action      string                       `json:"action"`
	Mode        string                       `json:"mode"`
	SessionID   string                       `json:"sessionId"`
	Text        string                       `json:"text"`
	Trace       []traceEntry                 `json:"trace"`
	Connections []wstransport.ConnectionInfo `json:"connections"`
	PreRestart  map[string]any               `json:"preRestart"`
	PostRestart map[string]any               `json:"postRestart"`
	Checks      map[string]bool              `json:"checks"`
	Error       string                       `json:"error,omitempty"`
}

type phase5Runtime struct {
	mode   string
	hub    *sessionstream.Hub
	ws     *wstransport.Server
	store  sessionstream.HydrationStore
	close  func() error
	cancel context.CancelFunc
	dbPath string
	reg    *sessionstream.SchemaRegistry
}

type phase5State struct {
	runtime    *phase5Runtime
	trace      []traceEntry
	preRestart map[string]any
	lastRun    phase5RunResponse
	seq        int
}
