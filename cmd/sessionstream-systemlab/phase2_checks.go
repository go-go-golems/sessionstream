package main

import (
	"strings"
)

func normalizeStreamMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "missing", "invalid":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "derived"
	}
}

func phase2PublishOrdinalsZero(history []phase2MessageRecord) bool {
	if len(history) == 0 {
		return false
	}
	for _, record := range history {
		if record.PublishedOrdinal != 0 {
			return false
		}
	}
	return true
}

func phase2Monotonic(ordinals map[string][]uint64) bool {
	if len(ordinals) == 0 {
		return false
	}
	for _, values := range ordinals {
		for i := 1; i < len(values); i++ {
			if values[i] <= values[i-1] {
				return false
			}
		}
	}
	return true
}

func phase2SessionIsolation(ordinals map[string][]uint64) bool {
	if len(ordinals) == 0 {
		return false
	}
	for _, values := range ordinals {
		if len(values) == 0 {
			return false
		}
	}
	return true
}

func phase2ConsumedCount(history []phase2MessageRecord) int {
	count := 0
	for _, record := range history {
		if record.AssignedOrdinal > 0 {
			count++
		}
	}
	return count
}
