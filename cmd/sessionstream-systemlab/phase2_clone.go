package main

func clonePhase2MessageRecord(in phase2MessageRecord) phase2MessageRecord {
	out := in
	out.PublishMetadata = cloneStringMap(in.PublishMetadata)
	out.ConsumeMetadata = cloneStringMap(in.ConsumeMetadata)
	return out
}

func clonePhase2Ordinals(in map[string][]uint64) map[string][]uint64 {
	if in == nil {
		return nil
	}
	out := make(map[string][]uint64, len(in))
	for sid, values := range in {
		out[sid] = append([]uint64(nil), values...)
	}
	return out
}

func cloneNamedPayloadMap(in map[string][]namedPayload) map[string][]namedPayload {
	if in == nil {
		return nil
	}
	out := make(map[string][]namedPayload, len(in))
	for sid, values := range in {
		cloned := make([]namedPayload, 0, len(values))
		for _, value := range values {
			cloned = append(cloned, namedPayload{Name: value.Name, Payload: cloneMap(value.Payload)})
		}
		out[sid] = cloned
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePhase2RunResponse(in phase2RunResponse) phase2RunResponse {
	out := in
	out.Trace = cloneTraceEntries(in.Trace)
	out.MessageHistory = make([]map[string]any, 0, len(in.MessageHistory))
	for _, record := range in.MessageHistory {
		out.MessageHistory = append(out.MessageHistory, cloneMap(record))
	}
	out.PerSessionOrdinals = cloneStringSlicesMap(in.PerSessionOrdinals)
	out.Fanout = cloneNamedPayloadMap(in.Fanout)
	out.Snapshots = make(map[string]map[string]any, len(in.Snapshots))
	for sid, snap := range in.Snapshots {
		out.Snapshots[sid] = cloneMap(snap)
	}
	out.Checks = cloneBoolMap(in.Checks)
	return out
}

func cloneStringSlicesMap(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for sid, values := range in {
		out[sid] = append([]string(nil), values...)
	}
	return out
}
