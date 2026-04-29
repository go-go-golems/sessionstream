package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func phase2MessageHistoryView(history []phase2MessageRecord) []map[string]any {
	out := make([]map[string]any, 0, len(history))
	for _, record := range history {
		out = append(out, map[string]any{
			"messageId":        record.MessageID,
			"sessionId":        record.SessionID,
			"eventName":        record.EventName,
			"label":            record.Label,
			"topic":            record.Topic,
			"publishedOrdinal": fmt.Sprintf("%d", record.PublishedOrdinal),
			"assignedOrdinal":  fmt.Sprintf("%d", record.AssignedOrdinal),
			"publishMetadata":  cloneStringMap(record.PublishMetadata),
			"consumeMetadata":  cloneStringMap(record.ConsumeMetadata),
		})
	}
	return out
}

func phase2OrdinalStrings(in map[string][]uint64) map[string][]string {
	out := make(map[string][]string, len(in))
	for sid, values := range in {
		converted := make([]string, 0, len(values))
		for _, value := range values {
			converted = append(converted, fmt.Sprintf("%d", value))
		}
		out[sid] = converted
	}
	return out
}

func renderPhase2Markdown(resp phase2RunResponse) string {
	var b strings.Builder
	b.WriteString("# Phase 2 Transcript\n\n")
	_, _ = fmt.Fprintf(&b, "- Action: `%s`\n", resp.Action)
	_, _ = fmt.Fprintf(&b, "- Stream mode: `%s`\n", resp.StreamMode)
	if resp.Error != "" {
		_, _ = fmt.Fprintf(&b, "- Error: `%s`\n", resp.Error)
	}
	b.WriteString("\n## Checks\n\n")
	keys := make([]string, 0, len(resp.Checks))
	for key := range resp.Checks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		status := "FAIL"
		if resp.Checks[key] {
			status = "PASS"
		}
		_, _ = fmt.Fprintf(&b, "- %s: %s\n", key, status)
	}
	b.WriteString("\n## Message History\n\n")
	for _, record := range resp.MessageHistory {
		buf, _ := json.Marshal(record)
		_, _ = fmt.Fprintf(&b, "- `%s`\n", string(buf))
	}
	b.WriteString("\n## Per-Session Ordinals\n\n```json\n")
	ordinals, _ := json.MarshalIndent(resp.PerSessionOrdinals, "", "  ")
	b.Write(ordinals)
	b.WriteString("\n```\n")
	b.WriteString("\n## Trace\n\n")
	for _, entry := range resp.Trace {
		_, _ = fmt.Fprintf(&b, "%d. **%s** — %s\n", entry.Step, entry.Kind, entry.Message)
		if len(entry.Details) > 0 {
			buf, _ := json.Marshal(entry.Details)
			_, _ = fmt.Fprintf(&b, "   - details: `%s`\n", string(buf))
		}
	}
	return b.String()
}

func uniqueStrings(values ...string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
