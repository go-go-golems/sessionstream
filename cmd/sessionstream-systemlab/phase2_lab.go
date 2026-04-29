package main

import (
	"context"

	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
)

const (
	phase2CommandName    = "PublishOrderedEvent"
	phase2EventName      = "OrderedEvent"
	phase2UIEventName    = "OrderedEventObserved"
	phase2TimelineEntity = "OrderedEventRecord"
)

type phase2RunRequest struct {
	Action     string `json:"action"`
	SessionA   string `json:"sessionA"`
	SessionB   string `json:"sessionB"`
	BurstCount int    `json:"burstCount"`
	StreamMode string `json:"streamMode"`
}

type phase2RunResponse struct {
	Action             string                    `json:"action"`
	SessionA           string                    `json:"sessionA"`
	SessionB           string                    `json:"sessionB"`
	BurstCount         int                       `json:"burstCount"`
	StreamMode         string                    `json:"streamMode"`
	Trace              []traceEntry              `json:"trace"`
	MessageHistory     []map[string]any          `json:"messageHistory"`
	PerSessionOrdinals map[string][]string       `json:"perSessionOrdinals"`
	Fanout             map[string][]namedPayload `json:"fanout"`
	Snapshots          map[string]map[string]any `json:"snapshots"`
	Checks             map[string]bool           `json:"checks"`
	Error              string                    `json:"error,omitempty"`
}

type phase2MessageRecord struct {
	MessageID        string            `json:"messageId"`
	SessionID        string            `json:"sessionId"`
	EventName        string            `json:"eventName"`
	Label            string            `json:"label,omitempty"`
	Topic            string            `json:"topic"`
	PublishedOrdinal uint64            `json:"publishedOrdinal"`
	AssignedOrdinal  uint64            `json:"assignedOrdinal"`
	PublishMetadata  map[string]string `json:"publishMetadata,omitempty"`
	ConsumeMetadata  map[string]string `json:"consumeMetadata,omitempty"`
}

type phase2State struct {
	hub               *sessionstream.Hub
	store             *storesqlite.Store
	cancel            context.CancelFunc
	streamMode        string
	syntheticSequence uint64
	publishCounters   map[string]int
	sessionMeta       map[string]map[string]any
	trace             []traceEntry
	messages          map[string]*phase2MessageRecord
	messageOrder      []string
	ordinals          map[string][]uint64
	fanout            map[string][]namedPayload
	lastRun           phase2RunResponse
}
