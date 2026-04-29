# Tasks

## Completed

- [x] Create dedicated ticket workspace `EVT-STREAM-013` for progressive custom-event previews and authoritative custom-event commit patterns in `evtstream` chat apps.
- [x] Gather evidence from `evtstream`, app-grade chat, `agentmode`, `cmd/web-chat`, frontend websocket/state code, and Geppetto structured parsing helpers.
- [x] Write a detailed intern-facing design and implementation guide with prose, diagrams, pseudocode, API references, and file references.
- [x] Record a chronological diary entry capturing commands, current migration context, and review instructions.
- [x] Validate the ticket with `docmgr doctor`.
- [x] Upload the ticket bundle to reMarkable.

## Follow-up implementation work

### Phase 0 — Runtime contract cleanup

- [x] Add a runtime-owned sink decorator contract such as `WrapSink` to `pinocchio/pkg/inference/runtime/composer.go`, while temporarily retaining legacy `Sink` for `pkg/webchat` compatibility.
- [x] Update `pinocchio/cmd/web-chat/runtime_composer.go` so the runtime composer returns sink-decoration behavior when `agentmode` middleware is enabled.
- [x] Change `pinocchio/cmd/web-chat/agentmode_sink.go` from returning a chat-app `SinkWrapper` to returning a runtime-composer-owned decorator.
- [x] Remove `ResolvedRuntime` from `pinocchio/pkg/evtstream/apps/chat/service.go`.
- [x] Change `PromptRequest.Runtime` to `*infruntime.ComposedRuntime`.
- [x] Update `pinocchio/pkg/evtstream/apps/chat/chat.go` to consume `*infruntime.ComposedRuntime` directly and apply `WrapSink` around the canonical chat sink.
- [x] Update `pinocchio/cmd/web-chat/app` runtime-resolver plumbing to return `*infruntime.ComposedRuntime` directly.
- [x] Update `pinocchio/cmd/web-chat/canonical_runtime_resolver.go` to stop constructing a chat-layer runtime wrapper and instead return the composed runtime directly.
- [x] Delete or fully retire `pinocchio/cmd/web-chat/agentmode_sink_wrapper.go` and its direct test coverage.
- [x] Run focused cleanup validation for the new runtime composition story (`go test ./pkg/inference/runtime ./pkg/evtstream/apps/chat ./cmd/web-chat/app ./cmd/web-chat/... -count=1` and `go test ./pkg/webchat/... -count=1`).
- [x] Remove legacy `ComposedRuntime.Sink` entirely once `pkg/webchat` no longer depends on a concrete sink field.

### Phase 1 — Progressive streaming parser

- [x] Add progressive YAML parsing to `pkg/middlewares/agentmode/structured_extractor.go` using `parsehelpers.YAMLController`.
- [x] Emit provisional preview events from `OnRaw(...)` without committing durable mode state.
- [x] Add extractor/parser tests for newline-triggered snapshots and analysis/new-mode preview emission.
- [ ] Add more churn-oriented tests for malformed intermediate YAML and edge-case debounce behavior.

### Phase 2 — Canonical event translation and projection

- [x] Define and register canonical preview and committed event schemas in `pkg/evtstream/apps/chat`.
- [x] Extend the canonical runtime sink to translate custom preview/commit events instead of dropping them.
- [x] Project committed custom events to durable timeline entities and hydration state.
- [x] Keep preview events live-only in v1 unless a later slice explicitly adds provisional hydration.
- [x] Add focused backend tests covering preview-event publication, committed-event publication, UI projection, and committed timeline projection.

### Phase 3 — Frontend widget and browser validation

- [x] Add frontend widget rendering for progressive preview state and committed mode state.
- [x] Clear preview UI state on commit, stop, or interruption.
- [x] Add focused frontend tests for snapshot hydration and live preview updates.
- [x] Run real browser validation with an agentmode-enabled profile and `gpt-5-nano-low`.

### Phase 4 — Playbook extraction

- [x] Extract a shorter contributor playbook from the implementation once the feature lands.
- [x] Relate the final implementation artifacts back to this ticket so the long-form guide remains the deep reference.
