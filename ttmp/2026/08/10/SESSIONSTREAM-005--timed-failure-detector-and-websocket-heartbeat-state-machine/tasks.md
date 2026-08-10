# Tasks

## TODO

- [x] Freeze current heartbeat wire, configuration, hello-first, timeout, and shutdown behavior with characterization tests <!-- t:d4cb -->
- [x] Implement the pure internal heartbeat state-event-action reducer and typed invariants <!-- t:kk21 -->
- [x] Add exhaustive transition-table, stale-event, boundary-time, and property/fuzz tests <!-- t:9v77 -->
- [x] Implement the per-connection supervisor with generation-safe timer and deterministic nonce seams <!-- t:6ybx -->
- [x] Integrate reader and writer acknowledgements while preserving one-reader/one-writer ownership <!-- t:myhd -->
- [x] Remove observer callbacks from heartbeat-critical paths with bounded best-effort dispatch <!-- t:9mm6 -->
- [x] Remove the pong channel, latest-pong helper, ticker loop, and duplicate legacy heartbeat mechanics <!-- t:sfy1 -->
- [ ] Run workspace and GOWORK=off tests, repeated race tests, vet, build, lint, vulnerability, hook, and release validation <!-- t:7tpx -->
- [x] Update package and operational documentation with suspicion semantics, timeout tuning, and compatibility <!-- t:i37x -->
- [x] Complete evidence-backed architecture analysis and intern implementation guide <!-- t:pyzo -->
- [x] Create and maintain the investigation diary, file relations, and ticket changelog <!-- t:5vtl -->
- [x] Validate ticket documentation and upload the design bundle to reMarkable <!-- t:4drb -->
- [ ] Design and implement pragmatic state-aware heartbeat reducer fuzzing and run a bounded campaign <!-- t:kk99 -->
