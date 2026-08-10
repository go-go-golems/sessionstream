---
Title: Investigation Diary
Ticket: SESSIONSTREAM-005
Status: active
Topics:
    - websocket
    - architecture
    - event-streaming
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/sessionstream/transport/ws/heartbeat.go
      Note: Integrated supervisor in Step 4 at commit dbfbf02
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Implemented in Step 3 at commit d0693bf
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go
      Note: Validated reducer invariants in Step 3 at commit d0693bf
    - Path: repo://pkg/sessionstream/transport/ws/observer.go
      Note: Moved callbacks off critical paths in Step 4 at commit dbfbf02
    - Path: repo://pkg/sessionstream/transport/ws/server.go
      Note: Primary implementation evidence inspected during Step 1
    - Path: repo://proto/sessionstream/v1/transport.proto
      Note: Wire-contract evidence inspected during Step 1
    - Path: repo://ttmp/2026/08/10/SESSIONSTREAM-005--timed-failure-detector-and-websocket-heartbeat-state-machine/design-doc/01-intern-guide-to-the-timed-failure-detector-and-websocket-heartbeat-state-machine.md
      Note: Primary design deliverable produced by the investigation
ExternalSources:
    - https://github.com/go-go-golems/sessionstream/pull/10
Summary: Chronological research and design record for the timed WebSocket failure-detector state machine.
LastUpdated: 2026-08-10T20:10:00-04:00
WhatFor: Continuing the heartbeat state-machine implementation with evidence, decisions, validation commands, and review guidance intact.
WhenToUse: Read before implementing or reviewing SESSIONSTREAM-005.
---




# Diary

## Goal

Capture the evidence-led investigation and design of a timed failure-detector state machine for Sessionstream WebSocket heartbeat handling, including the theory, current code constraints, proposed API, implementation phases, and delivery evidence.

## Step 1: Create the ticket and design the timed failure detector

This step created SESSIONSTREAM-005 as a focused follow-up to the WebSocket lifecycle review work in PR #10. The immediate objective was not to patch another heartbeat edge case, but to model the protocol explicitly enough that a new intern can implement and review it systematically.

I traced the current wire schema, server lifecycle, reader and writer ownership, heartbeat loop, snapshot hydration boundary, observers, browser clients, tests, and shutdown behavior. I then turned those findings into a detailed design centered on a pure event/action reducer and a single per-connection supervisor.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new docmgr ticket in sessionstream to build that and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new Sessionstream docmgr ticket for implementing a mathematically explicit heartbeat failure detector, author a comprehensive intern-facing analysis/design/implementation guide, keep ticket bookkeeping, validate it, and upload the documentation bundle to reMarkable.

**Inferred user intent:** Replace reactive review-driven heartbeat patches with a teachable and verifiable architecture that gives a new engineer enough theory, repository context, API detail, pseudocode, file evidence, and implementation sequencing to build it safely.

### What I did

- Created ticket `SESSIONSTREAM-005` with architecture, event-streaming, onboarding, and WebSocket topics.
- Created the primary design document and this investigation diary.
- Inspected `pkg/sessionstream/transport/ws/server.go`, including connection configuration, `ServeHTTP`, `readLoop`, request processing, `writeLoop`, heartbeat handling, `closeConnection`, and `Close`.
- Inspected `pkg/sessionstream/transport/ws/observer.go` and identified synchronous observer invocation as a control-path interference risk.
- Inspected `proto/sessionstream/v1/transport.proto` and preserved the existing application-level ping/pong wire contract.
- Inspected `pkg/sessionstream/hydration.go` and `pkg/sessionstream/hub.go` to explain how snapshots and UI fanout relate to transport behavior.
- Located all shipped browser heartbeat handlers and relevant heartbeat/lifecycle tests.
- Researched unreliable failure detectors, phi-accrual detection, RFC 6455 Ping/Pong semantics, and Gorilla WebSocket's single-reader/single-writer contract.
- Wrote an intern-facing guide with system orientation, theory, current-state evidence, gap analysis, architecture diagrams, APIs, transition table, pseudocode, invariants, decision records, implementation phases, tests, risks, alternatives, open questions, and file references.

### Why

- Repeated review findings clustered around implicit lifecycle and heartbeat state rather than isolated syntax defects.
- A pure reducer separates mathematical protocol correctness from goroutine scheduling and network I/O.
- Explicit invariants make review and testing finite: every state/event pair can be covered without timing-sensitive sleeps.
- Intern onboarding requires explaining both why the design exists and how it fits the current repository.

### What worked

- The current code has strong reusable boundaries: one reader, one writer, ordered request processing, tracked write completion, hello readiness, bounded queues, and idempotent close.
- The wire schema already carries the nonce needed for exact challenge matching.
- The proposed design preserves public `ConnectionConfig` and protobuf-JSON compatibility.
- The pure `(state, event) -> (state, actions)` model cleanly expresses stale nonce and stale timer behavior.
- The implementation can proceed incrementally with characterization tests before deleting old mechanics.

### What didn't work

- An initial source inventory looked for `pkg/sessionstream/pb/proto/sessionstream/v1/transport.proto`, but the source schema is actually `proto/sessionstream/v1/transport.proto`; `pkg/sessionstream/pb/proto/sessionstream/v1/transport.pb.go` is generated output.
- The broad `docmgr status --summary-only` reports all 25 historical tickets as stale. Those repository-wide historical warnings are unrelated to this new ticket; ticket-scoped doctor output is the relevant completion gate.

### What I learned

- Sessionstream implements application-level heartbeat because browser JavaScript cannot directly send arbitrary RFC 6455 control frames.
- The response deadline must exclude local outbound queue delay; successful socket write is the correct start boundary.
- The one-slot pong channel currently mixes event transport with freshness policy. Generation-checked direct state transitions are easier to prove.
- Timer cancellation alone is not a correctness proof; stale deadline events need explicit generation identity.
- `TransportObserver` is called synchronously, so panic recovery does not prevent a blocked callback from delaying control flow.
- Fixed-threshold failure detection is sufficient for the first refactor. Phi-accrual behavior is a future operational optimization, not a prerequisite for correct ownership.

### What was tricky to build

The difficult part was drawing a boundary that is mathematically useful without creating an overgeneralized framework. The reducer must remain pure, but nonce generation, actual write acknowledgement, timer creation, observation, and connection closure are effects. The design therefore represents those effects as ordered actions executed by a supervisor adapter.

Another sharp edge is the race between a matching pong and a deadline event. Channel selection order is not a stable semantic authority. The guide recommends timestamp-based boundary semantics and generation-checked timer events so scheduler ordering cannot silently redefine the protocol.

Observer isolation is also nontrivial because Go cannot forcibly stop a user callback that blocks forever. The design proposes bounded best-effort dispatch and requires the implementation to define whether callback completion participates in graceful shutdown. It explicitly forbids observer execution on the heartbeat supervisor path.

### What warrants a second pair of eyes

- The exact deadline boundary: whether a pong timestamp equal to the deadline is accepted or classified as late.
- Whether the next heartbeat interval begins after matching pong or follows a fixed periodic epoch.
- Whether observer dispatch should be per connection or per server.
- Whether nonce generation belongs immediately outside the reducer or should be represented as a separate action/event handshake.
- Whether heartbeat event-queue overflow should always close the connection.
- Whether write failure should produce detector suspicion or a distinct transport-failure terminal state.
- Whether general observer isolation belongs in this ticket or a tightly coupled follow-up.

### What should be done in the future

- Implement phases 0 through 7 from the design guide.
- Resolve the open design questions before finalizing reducer types.
- Keep the state machine internal until a second transport proves a public abstraction is useful.
- Update the diary after each implementation phase with exact commands, failures, commits, and review guidance.

### Code review instructions

- Begin with `proto/sessionstream/v1/transport.proto` to understand the stable wire contract.
- Read `pkg/sessionstream/transport/ws/server.go` in lifecycle order: configuration, `ServeHTTP`, `readLoop`, `writeLoop`, heartbeat, close.
- Read `pkg/sessionstream/transport/ws/observer.go` to understand callback interference.
- Review the design's HB-1 through HB-10 invariants before reviewing proposed APIs.
- Validate future implementation first with deterministic reducer tests, then fake-time integration tests, then repeated race-enabled transport tests.

### Technical details

```text
Proposed kernel:
  (State, Event) -> (State, []Action)

Proposed phases:
  Booting -> Idle -> Writing -> Awaiting -> Idle
                                   |
                                   +-> Suspected -> Stopped

Core identity:
  local generation + opaque wire nonce

Core deadline:
  deadline = successful WriteMessage completion time + PongTimeout
```

Research references:

- Chandra and Toueg, unreliable failure detectors.
- Hayashibara et al., phi-accrual failure detector.
- RFC 6455 Sections 5.5.2 and 5.5.3.
- Gorilla WebSocket package concurrency contract.

## Step 2: Validate and deliver the documentation bundle

This step validated the ticket's metadata and structural health, rendered the complete documentation to PDF, corrected a font compatibility problem in an ASCII topology diagram, and delivered the final bundle to reMarkable.

The upload includes the ticket overview, 1,400-plus-line intern guide, investigation diary, and phased implementation task list. The implementation tasks remain open by design; this ticket is now ready for an intern to begin Phase 0 characterization work.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete ticket quality checks and publish the requested material to the user's reMarkable device.

**Inferred user intent:** Make the technical design durable, portable, and immediately usable away from the development environment.

### What I did

- Validated frontmatter for the index, design document, and diary.
- Ran ticket-scoped `docmgr doctor` with a 30-day stale threshold.
- Rendered the documentation through Pandoc/XeLaTeX as a 39-page PDF smoke test.
- Replaced unsupported Unicode tree-drawing glyphs with ASCII equivalents.
- Ran a reMarkable bundle dry-run with a depth-two table of contents.
- Uploaded the real bundle to `/ai/2026/08/10/SESSIONSTREAM-005`.
- Marked the documentation-delivery task complete and updated the ticket changelog.

### Why

- Frontmatter and doctor checks ensure the ticket remains searchable and structurally valid.
- Local PDF rendering catches LaTeX/font failures before a remote upload.
- The dry-run verifies paths and bundle composition without mutating the device.

### What worked

- All three frontmatter validations passed.
- `docmgr doctor --ticket SESSIONSTREAM-005 --stale-after 30` reported all checks passed.
- PDF generation produced 39 pages.
- The reMarkable command reported: `OK: uploaded SESSIONSTREAM-005 Timed Failure Detector Guide.pdf -> /ai/2026/08/10/SESSIONSTREAM-005`.

### What didn't work

- The first PDF smoke render warned that Latin Modern Mono lacked Unicode tree glyphs `U+2514`, `U+2500`, and `U+251C`.
- Exact warning examples: `[WARNING] Missing character: There is no └ (U+2514)` and `[WARNING] Missing character: There is no ├ (U+251C)`.
- Replacing the four tree entries with ASCII ``|--`` and `` `--`` removed dependence on those glyphs.

### What I learned

- Even when Pandoc succeeds, missing-glyph warnings can make diagrams incomplete on the device; PDF smoke rendering is worth retaining before upload.
- The ticket bundle is more useful when tasks are included alongside design and diary because the reader can move directly from architecture to implementation phases.

### What was tricky to build

The documentation skill calls for dry-run and delivery verification, while the current reMarkable workflow treats a successful `OK: uploaded` result as authoritative and discourages a redundant cloud listing. The chosen sequence used the required dry-run, then one real upload, and accepted the explicit success output without an extra expensive listing.

### What warrants a second pair of eyes

- Review the PDF's wide transition and interference tables on the physical device for comfortable scaling.
- Confirm that future updates should overwrite this document only intentionally because reMarkable annotations may otherwise be lost.

### What should be done in the future

- Begin Phase 0 characterization tests under the open implementation tasks.
- Upload a new bundle revision only when the implementation materially changes the guide.

### Code review instructions

- Run `docmgr doctor --ticket SESSIONSTREAM-005 --stale-after 30`.
- Render the same four Markdown files with Pandoc or use the reMarkable dry-run before a future upload.
- Inspect `tasks.md` to distinguish completed research/delivery work from open code implementation.

### Technical details

```text
Bundle: SESSIONSTREAM-005 Timed Failure Detector Guide.pdf
Remote directory: /ai/2026/08/10/SESSIONSTREAM-005
ToC depth: 2
PDF smoke test: 39 pages
```

## Step 3: Freeze compatibility and implement the pure detector kernel

This step began implementation by freezing the existing heartbeat wire/default configuration contract and adding the pure state/event/action reducer. The kernel is internal to the WebSocket transport and has no goroutines, channels, clocks, timers, sockets, protobuf dependencies, contexts, or observers.

The reducer now makes readiness, writing, awaiting, suspicion, stopping, nonce matching, generation matching, deadline boundaries, and ordered side effects explicit. Exhaustive matrix tests and fuzzed event sequences exercise the model independently of real-time scheduling.

### Prompt Context

**User prompt (verbatim):** "ok, implement the new heartbeat system and replace the old one, keep a detailed diary as you work, and commit at appropriate intervals."

**Assistant interpretation:** Implement SESSIONSTREAM-005 completely, remove the old heartbeat mechanism, maintain detailed chronological evidence, and make focused commits at meaningful milestones.

**Inferred user intent:** Turn the theoretical design into production code while preserving compatibility and avoiding another long sequence of unstructured concurrency patches.

**Commit (code):** d0693bf1d4990261f2ef3abe2ca45bc9ab94f161 — "feat(ws): add heartbeat failure detector kernel"

### What I did

- Created branch `task/sessionstream-005-heartbeat-machine` from PR #10 head `21724ab` so the new work does not alter the active PR branch.
- Committed the ticket design bundle separately as `1d1b0e9`.
- Added `pkg/sessionstream/transport/ws/internal/heartbeat/machine.go`.
- Defined six phases, seven event kinds, nine action kinds, explicit state snapshots, configuration, and typed errors.
- Implemented exact nonce and generation matching.
- Defined a pong at or after its deadline as late/stale.
- Made matching timer events before their deadline return `ErrEarlyDeadline`.
- Made `Stopped` absorbing and cancellation explicit when stopping from `Awaiting`.
- Added characterization coverage for protobuf JSON ping/pong frames and all default `ConnectionConfig` values.
- Added deterministic transition, boundary, stale-event, write-failure, stop, generation, exhaustive phase/event, and fuzz tests.

### Why

- Compatibility fixtures provide evidence that the later integration does not drift public behavior.
- A pure kernel permits correctness testing without `time.Sleep` or scheduler assumptions.
- Local generation identity makes stale timer and write-completion events harmless even when nonce values are opaque.
- Ordered actions separate policy effects from transition mechanics.

### What worked

- The kernel passed 100 race-enabled repetitions before commit.
- The kernel and current transport passed 20 race-enabled repetitions together.
- Repository lint passed with zero issues after exhaustive-switch corrections.
- Full pre-commit tests and lint passed.
- The wire fixtures remain `{"ping":{"nonce":"n-1"}}` and `{"pong":{"nonce":"n-1"}}`.

### What didn't work

The first commit attempt failed because the repository enables the `exhaustive` linter. The exact findings were:

```text
pkg/sessionstream/transport/ws/internal/heartbeat/machine.go:156:2: missing cases in switch of type heartbeat.Phase: heartbeat.PhaseStopped (exhaustive)
pkg/sessionstream/transport/ws/internal/heartbeat/machine.go:173:2: missing cases in switch of type heartbeat.EventKind: heartbeat.EventTick, heartbeat.EventPingWritten, heartbeat.EventPingWriteFailed, heartbeat.EventDeadlineElapsed, heartbeat.EventStop (exhaustive)
pkg/sessionstream/transport/ws/internal/heartbeat/machine.go:185:2: missing cases in switch of type heartbeat.EventKind: heartbeat.EventReady, heartbeat.EventPingWritten, heartbeat.EventPingWriteFailed, heartbeat.EventDeadlineElapsed, heartbeat.EventStop (exhaustive)
pkg/sessionstream/transport/ws/internal/heartbeat/machine.go:201:2: missing cases in switch of type heartbeat.EventKind: heartbeat.EventReady, heartbeat.EventTick, heartbeat.EventDeadlineElapsed, heartbeat.EventStop (exhaustive)
pkg/sessionstream/transport/ws/internal/heartbeat/machine.go:226:2: missing cases in switch of type heartbeat.EventKind: heartbeat.EventReady, heartbeat.EventTick, heartbeat.EventStop (exhaustive)
pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go:248:2: missing cases in switch of type heartbeat.Phase: heartbeat.PhaseBooting, heartbeat.PhaseIdle, heartbeat.PhaseSuspected, heartbeat.PhaseStopped (exhaustive)
```

I corrected each switch to enumerate every known enum value explicitly, reran focused race tests and `make lint`, and only then committed.

### What I learned

- The exhaustive linter reinforces the state-machine goal: newly added phases or events force a compile-time-adjacent review of every transition function.
- Keeping nonce creation outside the reducer preserves purity; `EventTick` carries a prepared nonce while the reducer owns generation advancement.
- Separating `PhaseWriting` from `PhaseAwaiting` is necessary to prove that no pong timeout exists before write acknowledgement.

### What was tricky to build

The boundary between late pong and timeout cannot depend on Go `select` ordering. The reducer stores the successful write timestamp and exact deadline, then compares the pong event timestamp to that deadline. A pong exactly at the deadline is intentionally stale. A matching timer event timestamped before the deadline is an invariant error rather than a suspicion transition.

The fuzz test also needs to accept arbitrary event order without weakening state invariants. Illegal or stale runtime combinations generally become no-ops, while structurally impossible timing such as an early matching deadline produces a typed error for adapter-level handling.

### What warrants a second pair of eyes

- Confirm the “pong exactly at deadline is late” boundary policy.
- Confirm that write failure should use `PhaseSuspected` while retaining a distinct joined `ErrPingWriteFailed` reason.
- Review whether unknown numeric enum values should return invariant errors in every phase; known stale events are intentionally ignored.
- Review the decision to have the adapter prepare nonce identity before `EventTick` rather than add a separate nonce-generation action.

### What should be done in the future

- Integrate one per-connection supervisor around the kernel.
- Replace the old pong channel, ticker loop, and direct timeout-close behavior.
- Isolate heartbeat-critical observation from synchronous callbacks.

### Code review instructions

- Start with `machine.go` types and the transition table in the design guide.
- Compare `Machine.Step` and each phase handler against HB-1 through HB-10.
- Run `GOWORK=off go test -race ./pkg/sessionstream/transport/ws/internal/heartbeat -count=100`.
- Review `TestEveryPhaseEventCombinationIsDefined` and `FuzzMachinePreservesInvariants` for complete state-space pressure.

### Technical details

```text
Kernel input:  Event{Kind, At, Generation, Nonce, Err}
Kernel output: []Action in execution order
Timeout basis: PingWritten.At + PongTimeout
Challenge identity: monotonically increasing generation + opaque nonce
```

## Step 4: Replace the legacy loops with the supervisor and isolate observers

This step integrated the pure detector into the production WebSocket lifecycle. Each upgraded connection now owns one heartbeat runtime and one supervisor; the old ticker loop, one-slot pong channel, and freshness helper are gone.

The integration also moved transport observers onto one bounded ordered dispatcher. Socket reads, writes, heartbeat transitions, request processing, and connection closure now enqueue immutable records without invoking extension code on their critical paths.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Continue from the kernel milestone through complete runtime replacement, observer isolation, repeated concurrency validation, and focused commits.

**Inferred user intent:** Ensure the mathematical model actually governs production behavior and removes—not merely wraps—the old concurrency mechanism.

**Commit (code):** dbfbf0221e11a7ca70c2679f45b0bfdfaa5b972a — "refactor(ws): supervise heartbeat state machine"

### What I did

- Added `pkg/sessionstream/transport/ws/heartbeat.go` as the only runtime adapter around the pure machine.
- Added one bounded heartbeat event queue per connection.
- Added cryptographically random opaque default nonces and deterministic test injection seams.
- Added timer and clock seams and a fake-clock supervisor test.
- Started the supervisor before read processing but gated its `Ready` event on the existing hello-written barrier.
- Routed decoded pongs to the reducer before any observer record.
- Kept one socket reader and one socket writer.
- Converted tracked writer completion into `PingWritten` or `PingWriteFailed` events.
- Moved write acknowledgements before observation dispatch.
- Replaced a free-running ticker with one scheduled idle timer after readiness or successful pong.
- Removed `connection.pongs`, `offerLatestPong`, `heartbeatLoopAfterReady`, and `heartbeatLoop`.
- Added a bounded 1,024-record ordered observer dispatcher with drop accounting through `ObserverDroppedRecords`.
- Made `Close(ctx)` stop and drain accepted observer records after connection workers finish while retaining retryable deadline behavior.
- Added test cleanup that closes observer workers.
- Added deterministic and repeated race coverage for the supervisor, observer backpressure, hydration interference, write acknowledgement, and shutdown.

### Why

- A single supervisor is the only mutator of detector state, eliminating timer/pong channel policy races.
- Actual write acknowledgement remains the transition from `Writing` to `Awaiting`, so local queue delay never consumes pong budget.
- A bounded observer dispatcher prevents extension callbacks from starving heartbeat control while avoiding unbounded goroutines.
- One-shot idle timers prevent accumulated ticker ticks from causing catch-up ping bursts.

### What worked

- Twenty full transport-package race repetitions passed before the integration commit.
- One hundred full transport-package race repetitions passed after fixing the reconnect test synchronization.
- `make lint` passed with zero issues.
- `GOWORK=off go vet ./...` passed.
- Full pre-commit tests and lint passed.
- Existing heartbeat timeout, responsive pong, hello-first, blocked hydration, blocked observer, close deadline, and browser wire tests continue to pass.
- The fake clock proves no pong deadline exists before writer acknowledgement.

### What didn't work

The first 100-run race suite exposed four failures in `TestServerReconnectGetsSnapshotThenNextLive`:

```text
Error: Received unexpected error:
       connection conn-1: connection conn-1 is closed
Test:  TestServerReconnectGetsSnapshotThenNextLive
```

The test closed the browser socket and immediately submitted the next event before the server reader had removed the old subscription. `PublishUI` legitimately found that closing target and reported delivery failure. I added an explicit wait for `server.Connections()` to become empty before the next submit. One hundred repeated race runs then passed.

After observation became asynchronous, two tests made immediate assertions before the dispatcher consumed their records:

```text
TestLiveQueueOverflowClosesConnection: stages did not yet contain server_frame_queue_full
TestTransportObserverFanoutNoTargets: expected session "missing", observed zero-value record
```

I changed those assertions to bounded `require.Eventually` checks, matching the documented best-effort asynchronous contract.

### What I learned

- A pong can arrive after the socket write but before the supervisor selects the write-ack channel. Treating every pong in `PhaseWriting` as stale would make correctness depend on `select` ordering.
- The reducer therefore needs a pending-pong substate even with one reader and writer.
- Keeping the **earliest** matching pending timestamp is correct: a later duplicate must not overwrite a timely valid response while write acknowledgement is delayed.
- A pong timestamped before successful write must not acknowledge the challenge even if its nonce happens to match.
- Observer isolation necessarily changes callback timing; tests and API comments must assert eventual ordered delivery rather than synchronous completion.

### What was tricky to build

The supervisor must remain responsive while waiting for four independent inputs: decoded pong events, writer acknowledgement, idle timer, and deadline timer. It uses nil-disabled select channels and keeps timer identity local to the sole owner. Reducer generations independently protect against stale deadline events.

Writer acknowledgement and pong arrival can be observed in either order. The updated `PhaseWriting` stores the earliest matching pong timestamp. On `PingWritten`, it accepts that response only when:

```text
WrittenAt <= PendingPongAt < WrittenAt + PongTimeout
```

This converts scheduler nondeterminism into explicit timestamp semantics.

Observer shutdown required a separate lifecycle from connection workers. `Close` first closes connections and waits for all handlers/workers, then linearizes observer shutdown, drains already accepted records, and waits for the dispatcher. If a callback blocks, `Close(ctx)` returns its deadline while a later `Close` can continue waiting after the callback is released.

### What warrants a second pair of eyes

- Review the recursive action application used for immediate queue/write/timer failures in `runHeartbeatSupervisor`.
- Review whether `heartbeatEventQueueSize = 8` is sufficient and whether overflow-close is the desired abuse policy.
- Review the public addition of `ObserverDroppedRecords` and the fixed 1,024-record dispatcher capacity.
- Review storing original callback contexts until asynchronous delivery; they may already be canceled when consumed.
- Review whether nonce generation failure should have a dedicated transport stage rather than `protocol_error`.
- Confirm earliest-pending-pong semantics and the inclusive lower/exclusive upper timestamp bounds.

### What should be done in the future

- Complete full repository, vulnerability, build, hook, release, JavaScript, and documentation validation.
- Update operational docs and design status to match the implementation.
- Run a final dead-code, race, resource-bound, and diff audit before pushing.

### Code review instructions

- Read `heartbeat.go` from supervisor setup through each select input and action effect.
- Confirm `server.go` contains no legacy pong channel or heartbeat loop.
- Review `observer.go` queue admission and stop/drain linearization alongside `Server.Close`.
- Run `GOWORK=off go test -race ./pkg/sessionstream/transport/ws/... -count=100`.
- Start with `TestHeartbeatSupervisorUsesWriteAckAndGenerationSafeTimer`, the blocked observer/hydration tests, and the pure pending-pong tests.

### Technical details

```text
Per connection:
  one socket reader
  one socket writer
  one ordered request worker
  one heartbeat supervisor
  one heartbeat event queue (capacity 8)
  at most one idle timer and one deadline timer

Per server with observer:
  one observer worker
  one observer queue (capacity 1024)
  dropped-record counter
```
