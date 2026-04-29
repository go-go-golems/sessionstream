package main

import (
	"fmt"
	"strings"
)

func normalizePhase5Mode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "sql") {
		return "sql"
	}
	return "memory"
}

func phase5CursorPreserved(mode string, pre, post map[string]any) bool {
	if pre == nil {
		return true
	}
	if mode != "sql" {
		return toString(post["ordinal"]) == "0"
	}
	return toString(pre["ordinal"]) == toString(post["ordinal"])
}

func phase5EntitiesPreserved(mode string, pre, post map[string]any) bool {
	if pre == nil {
		return true
	}
	preJSON := fmt.Sprintf("%v", pre["entities"])
	postJSON := fmt.Sprintf("%v", post["entities"])
	if mode != "sql" {
		return postJSON == "[]"
	}
	return preJSON == postJSON
}

func phase5ResumeWithoutGaps(trace []traceEntry, sessionID string) bool {
	prev := uint64(0)
	for _, entry := range trace {
		if entry.Message != "phase 5 ui projection emitted event" {
			continue
		}
		if toString(entry.Details["sessionId"]) != sessionID {
			continue
		}
		var current uint64
		_, _ = fmt.Sscanf(toString(entry.Details["ordinal"]), "%d", &current)
		if current == 0 {
			continue
		}
		if prev > 0 && current != prev+1 {
			return false
		}
		prev = current
	}
	return true
}
