package main

import (
	wstransport "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

func clonePhase5RunResponse(in phase5RunResponse) phase5RunResponse {
	out := in
	out.Trace = cloneTraceEntries(in.Trace)
	out.Connections = append([]wstransport.ConnectionInfo(nil), in.Connections...)
	out.PreRestart = cloneMap(in.PreRestart)
	out.PostRestart = cloneMap(in.PostRestart)
	out.Checks = cloneBoolMap(in.Checks)
	return out
}
