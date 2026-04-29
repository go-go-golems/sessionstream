package main

func appendTraceEntry(entries *[]traceEntry, kind, message string, details map[string]any) {
	step := len(*entries) + 1
	*entries = append(*entries, traceEntry{Step: step, Kind: kind, Message: message, Details: cloneMap(details)})
}

func cloneTraceEntries(in []traceEntry) []traceEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]traceEntry, 0, len(in))
	for _, entry := range in {
		out = append(out, traceEntry{Step: entry.Step, Kind: entry.Kind, Message: entry.Message, Details: cloneMap(entry.Details)})
	}
	return out
}

func cloneNamedPayloads(in []namedPayload) []namedPayload {
	if len(in) == 0 {
		return nil
	}
	out := make([]namedPayload, 0, len(in))
	for _, event := range in {
		out = append(out, namedPayload{Name: event.Name, Payload: cloneMap(event.Payload)})
	}
	return out
}
