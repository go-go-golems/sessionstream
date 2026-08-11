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
    - Path: repo://README.md
      Note: Operational documentation completed in Step 5
    - Path: repo://pkg/sessionstream/bus_test.go
      Note: Per-session ordering assertion corrected in Step 8
    - Path: repo://pkg/sessionstream/hub_test.go
      Note: Race-safe shared hydration and event-store fixture from Step 8
    - Path: repo://pkg/sessionstream/transport/ws/heartbeat.go
      Note: |-
        Integrated supervisor in Step 4 at commit dbfbf02
        P1 overdue deadline fix and shared helper (commit 5a1d9eb)
    - Path: repo://pkg/sessionstream/transport/ws/heartbeat_arbitration_test.go
      Note: Pre-fix reproducer, regressions, seeds, oracle, and campaign (commit 5a1d9eb)
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/logcopter.go
      Note: Generated package metadata required by CI
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Implemented in Step 3 at commit d0693bf
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go
      Note: |-
        Validated reducer invariants in Step 3 at commit d0693bf
        State-aware fuzzing implemented and run in Step 7 at commit a7a49e4
    - Path: repo://pkg/sessionstream/transport/ws/observer.go
      Note: |-
        Moved callbacks off critical paths in Step 4 at commit dbfbf02
        Detaches accepted observations from producer cancellation while preserving values (commit 5005973)
    - Path: repo://pkg/sessionstream/transport/ws/server.go
      Note: Primary implementation evidence inspected during Step 1
    - Path: repo://pkg/sessionstream/transport/ws/server_test.go
      Note: Deterministic queued-context lifetime regression coverage (commit 5005973)
    - Path: repo://proto/sessionstream/v1/transport.proto
      Note: Wire-contract evidence inspected during Step 1
    - Path: repo://ttmp/2026/08/10/SESSIONSTREAM-005--timed-failure-detector-and-websocket-heartbeat-state-machine/design-doc/01-intern-guide-to-the-timed-failure-detector-and-websocket-heartbeat-state-machine.md
      Note: Primary design deliverable produced by the investigation
    - Path: repo://ttmp/2026/08/10/SESSIONSTREAM-005--timed-failure-detector-and-websocket-heartbeat-state-machine/design-doc/03-intern-guide-to-heartbeat-supervisor-event-arbitration-and-runtime-fuzzing.md
      Note: Supervisor arbitration and runtime fuzzing design delivered to reMarkable
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
- A peer can receive and answer bytes before the local `WriteMessage` call returns; local write completion is the deadline origin, not a valid lower bound on pong arrival.
- Observer isolation necessarily changes callback timing; tests and API comments must assert eventual ordered delivery rather than synchronous completion.

### What was tricky to build

The supervisor must remain responsive while waiting for four independent inputs: decoded pong events, writer acknowledgement, idle timer, and deadline timer. It uses nil-disabled select channels and keeps timer identity local to the sole owner. Reducer generations independently protect against stale deadline events.

Writer acknowledgement and pong arrival can be observed in either order. The updated `PhaseWriting` stores the earliest matching pong timestamp. On `PingWritten`, it accepts that response only when:

```text
PendingPongAt < WrittenAt + PongTimeout
```

This converts scheduler nondeterminism into explicit timestamp semantics.

Observer shutdown required a separate lifecycle from connection workers. `Close` first closes connections and waits for all handlers/workers, then linearizes observer shutdown, drains already accepted records, and waits for the dispatcher. If a callback blocks, `Close(ctx)` returns its deadline while a later `Close` can continue waiting after the callback is released.

### What warrants a second pair of eyes

- Review the recursive action application used for immediate queue/write/timer failures in `runHeartbeatSupervisor`.
- Review whether `heartbeatEventQueueSize = 8` is sufficient and whether overflow-close is the desired abuse policy.
- Review the public addition of `ObserverDroppedRecords` and the fixed 1,024-record dispatcher capacity.
- Review storing original callback contexts until asynchronous delivery; they may already be canceled when consumed.
- Review whether nonce generation failure should have a dedicated transport stage rather than `protocol_error`.
- Confirm earliest-pending-pong semantics and the exclusive deadline upper bound.

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

## Step 5: Remove scheduler ordering from write and deadline semantics

This step followed repeated stress failures back to two deeper serialization mistakes. The code originally timestamped `PingWritten` when the supervisor selected the acknowledgement, and it allowed a simultaneously ready timer to win over an already admitted timely pong.

Tracked writes now carry the timestamp captured by the sole writer immediately when `WriteMessage` returns. Before applying a deadline event, the supervisor drains the bounded heartbeat control queue so a pong admitted before the timer cannot lose solely because Go's `select` chose the timer case.

### Prompt Context

**User prompt (verbatim):** (same as Step 3)

**Assistant interpretation:** Treat repeated stress failures as evidence of model/adapter defects, correct the event-time boundaries, and continue only after high-repetition validation.

**Inferred user intent:** Achieve a heartbeat implementation whose correctness follows explicit event-time rules rather than probabilistic scheduler behavior.

**Commit (code):** afe9496efff266bc207f73f076fcd40658d9cddc — "fix(ws): order heartbeat events by wire completion"

### What I did

- Replaced tracked write `chan error` with `chan frameWriteResult` carrying both completion timestamp and error.
- Captured that timestamp inside the sole writer immediately after write setup or `WriteMessage` returns.
- Used the captured timestamp—not supervisor receive time—for `PingWritten` and deadline derivation.
- Preserved pending matching pongs while the writer acknowledgement is waiting to be selected.
- Kept the earliest pending matching pong so a late duplicate cannot overwrite timely evidence.
- Drained at most the bounded heartbeat queue capacity before applying a ready deadline event.
- Added explicit reducer tests for pong-before-write-return and duplicate pending pongs.
- Increased the real-time responsive-pong test margin while retaining fake-time boundary authority.
- Added operational README documentation covering suspicion semantics, interval/deadline behavior, observer dispatch, drop accounting, and shutdown.
- Ran 300 repeated race-enabled heartbeat integration tests after the corrections.
- Ran 100 full transport-package race repetitions successfully.

### Why

- The timestamp at supervisor channel selection can be arbitrarily later than actual write completion and can invert pong/write ordering.
- A Go `select` chooses pseudo-randomly among ready cases; that choice must not override a pong already admitted before its deadline.
- The fake-time reducer tests should own exact boundary assertions, while real-socket tests should use enough timing margin to test integration rather than host scheduler speed.

### What worked

- Three hundred repeated race-enabled responsive, hydration-blocked, observer-blocked, and fake-clock heartbeat tests passed after the final event-time correction.
- One hundred complete transport-package race repetitions passed.
- Full repository tests, vet, and lint passed before commit.
- Pre-commit full tests and lint passed.
- The timeout still begins at local write completion, while a pong physically received before the call returns is accepted as valid.

### What didn't work

The first full 100-run race attempt after integration failed once in the hydration-blocked test at a 25 ms deadline. Raising that deadline exposed further failures rather than merely masking them. A 200-run focused test still produced timeout disconnects after several successful ping/pong cycles.

The diagnostic stage sequence was especially useful:

```text
... heartbeat_ping_queued
server_frame_written
client_frame_read
client_frame_decoded
heartbeat_pong_received
...
heartbeat_timeout
disconnected
```

This proved the client was responding and the reader was decoding pongs. The defect was downstream event ordering.

The first attempted correction required `PendingPongAt >= PingWritten.At`. That was also wrong. A peer can receive kernel-written bytes and answer before the local `WriteMessage` function returns, so the recorded pong can legitimately precede local completion. Removing that lower bound and retaining only the exclusive deadline upper bound fixed the stress failures.

### What I learned

- “Actually written” has multiple observable moments: kernel/network visibility and local API completion are not globally ordered with remote response processing.
- Local API completion remains a conservative deadline origin, but it cannot reject evidence already returned by the peer.
- A timer channel and input channel need an explicit tie policy; state-machine purity alone does not solve adapter-level event serialization.
- Bounded queue draining is safe here because there are at most eight admitted control events and the reducer validates each timestamp and nonce.

### What was tricky to build

The machine and adapter observe different orders. The writer owns the authoritative local completion time, while the supervisor owns state mutation. Passing a timestamped result preserves both ownership rules. A pong selected during `Writing` remains pending; when the timestamped acknowledgement arrives, the reducer either completes the cycle or arms a deadline from the exact completion time.

At deadline selection, the supervisor processes up to `heartbeatEventQueueSize` queued pong events first. A timely matching pong transitions the machine to idle, making the subsequent stale deadline event harmless. A pong at or after the deadline remains stale by reducer timestamp comparison, so draining does not extend the deadline.

### What warrants a second pair of eyes

- Confirm that bounded queue draining before deadline is the clearest linearization policy.
- Confirm accepting a matching pending pong before local write return is appropriate given cryptographically random challenge nonces and non-authentication semantics.
- Review whether tracked timestamps should use the heartbeat clock for all frames or only heartbeat frames; the implementation uses the injected clock consistently.
- Review whether real-time integration margins are sufficient under slow CI race instrumentation.

### What should be done in the future

- Run final full validation, including repository-wide race tests and release hooks.
- Resolve any unrelated full-race fixture issue explicitly rather than ignoring it.
- Complete ticket validation, diary, changelog, and final branch audit.

### Code review instructions

- Begin at `frameWriteResult`, `writeLoop`, and the `writeAck` supervisor case.
- Review the deadline queue-drain block alongside reducer deadline timestamp checks.
- Run the four focused heartbeat integration tests 300 times under `-race`.
- Run all transport packages 100 times under `-race`.

### Technical details

```text
Writer result:
  frameWriteResult{at: heartbeatNow(), err: writeError}

Deadline:
  result.at + PongTimeout

Deadline tie policy:
  drain at most 8 admitted heartbeat events
  then apply generation-tagged deadline event
```

## Step 6: Scope and publish the pragmatic fuzzing design

This step documented a deliberately bounded improvement to the reducer fuzzer before changing its implementation. The plan strengthens useful state-space exploration without adding a formal-tool runtime, external property-testing dependency, generated production code, or mandatory long CI campaign.

The design uses one mutation-friendly byte to encode an operation, identity mode, and time mode. A state-sensitive advance operation guarantees that mutations can traverse repeated healthy generations, while direct operations still inject stale identities, deadline boundaries, failures, duplicates, and stop.

### Prompt Context

**User prompt (verbatim):** "ok, create a design doc for the better fuzzing, while we are at it. Don't over do it, we still want to ship. Only stuff that is reasonable to add and interesting to document. Upload the design doc to remarkable. Then implement it and run it."

**Assistant interpretation:** Add a concise design document for practical state-aware fuzzing, publish it to reMarkable, then implement exactly that limited scope and run a bounded native Go fuzz campaign.

**Inferred user intent:** Capture the interesting verification reasoning while preventing fuzz infrastructure from delaying shipment.

### What I did

- Added `design-doc/02-pragmatic-stateful-fuzzing-plan-for-the-heartbeat-reducer.md`.
- Documented the limitations of the existing nonce-mismatched fuzzer.
- Defined a one-byte operation/identity/time encoding.
- Defined a phase-sensitive happy-path advance operation.
- Limited properties to state shape, error atomicity, monotonic generation, and stable action contracts.
- Defined a readable seed corpus and a 60-second local campaign.
- Explicitly rejected external frameworks, formal code generation, supervisor/socket fuzzing, and mandatory long CI fuzzing.
- Related the design to the reducer and test implementation.
- Validated frontmatter and ticket health.
- Ran a reMarkable dry-run and uploaded the design PDF.

### Why

- The current fuzzer often stalls in `Writing` because generated write and pong nonces do not match the tick nonce.
- A small state-aware decoder provides much better exploration without creating a second specification project.
- Publishing the design first keeps implementation reviewable against an agreed boundary.

### What worked

- Frontmatter validation passed.
- Ticket-scoped `docmgr doctor` passed.
- The dry-run showed only the requested fuzzing design in the bundle.
- Upload succeeded at `/ai/2026/08/10/SESSIONSTREAM-005/SESSIONSTREAM-005 Heartbeat Fuzzing Plan.pdf`.

### What didn't work

N/A

### What I learned

- Eight operations, four identity modes, and eight time modes fit exactly in one byte, which is both compact and friendly to Go's corpus minimizer.
- A single state-aware operation is enough to make healthy multi-cycle traces reachable without hiding malformed direct-event combinations.

### What was tricky to build

The scope needed to avoid a tautological second reducer while still asserting more than “did not panic.” The plan therefore checks coarse action contracts and error atomicity, leaving exact transition order to the independent deterministic table tests already present.

### What warrants a second pair of eyes

- Review whether the identity-mode distribution gives enough matching events without making stale events rare.
- Review the chosen 4,096-operation per-input bound.
- Confirm a 60-second local campaign is sufficient for this shipment.

### What should be done in the future

- Implement the decoder, seeds, state/action assertions, and bounded input loop.
- Run seed regressions, race regressions, and the 60-second fuzz campaign.
- Preserve any generated failure as a corpus fixture and named regression if one appears.

### Code review instructions

- Compare the implementation directly against the operation, identity, and time tables in the fuzzing design.
- Ensure all helpers remain in `_test.go` and production APIs remain unchanged.

### Technical details

```text
bits 0..2: operation
bits 3..4: identity mode
bits 5..7: time mode
campaign: go test -run='^$' -fuzz=FuzzMachinePreservesInvariants -fuzztime=60s
```

## Step 7: Implement and run state-aware reducer fuzzing

This step implemented exactly the bounded fuzzing plan from Step 6. The native Go target can now move through repeated healthy generations while independently mutating event kind, challenge identity, and boundary time.

The campaign completed without finding a reducer violation. The new harness stays entirely in `machine_test.go`; it adds no runtime API, dependency, generated code, or CI-duration commitment.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Implement the approved small fuzzing enhancement, validate seeds under ordinary and race execution, run the planned bounded campaign, and record concrete evidence.

**Inferred user intent:** Gain meaningful property-testing confidence now without expanding the shipment into a formal verification project.

**Commit (code):** a7a49e450ee3a4302536edd667bde37e4b0fb820 — "test: add state-aware heartbeat fuzzing"

### What I did

- Replaced direct byte-to-event mapping with an 8/4/8 operation, identity, and time decoder.
- Added phase-sensitive `Advance` behavior for healthy multi-generation traversal.
- Added current, previous, future/stale, and empty identity modes.
- Added before/exact/after deadline, write-time, duplicate-time, timeout, and one-second time modes.
- Bounded each fuzz execution to 4,096 operations.
- Added seven readable seeds for healthy cycles, pending pong, stale pong, early/exact deadline, write failure, active stop, and post-stop events.
- Added error-atomicity checks for `ErrMissingNonce` and `ErrEarlyDeadline`.
- Added stable action-contract assertions for send, arm, pong, suspicion/close pairing, schedule, and stop.
- Added a deterministic test proving seven `Advance` bytes complete two healthy generations.
- Ran 100 ordinary seed regressions.
- Ran 100 race-enabled seed regressions.
- Ran the planned 60-second fuzz campaign.

### Why

- State-aware advance prevents random traces from becoming trapped by accidental nonce mismatch.
- Separate direct operations preserve malformed and stale-event exploration.
- One-byte encoding lets Go's mutator and minimizer alter one semantic dimension cheaply.
- Error atomicity is a strong property not tied to the reducer's internal switch layout.

### What worked

The bounded campaign reported:

```text
baseline coverage: 7/7 completed
workers: 8
executions: 111066
new interesting: 153
total interesting: 160
result: PASS
elapsed: 61.075s
```

No failure corpus was produced. Seed regressions, race-enabled seed regressions, lint, full pre-commit tests, and pre-commit lint all passed.

### What didn't work

The campaign periodically reported zero executions per second for several three-second reporting windows after reaching larger interesting inputs. It resumed repeatedly and completed successfully. The per-input 4,096-operation cap bounded work; no timeout, crash, or test failure occurred.

### What I learned

- A small semantic decoder significantly increased interesting corpus discovery while remaining easy to inspect.
- The deterministic two-generation test is valuable because corpus execution alone does not report which phases were reached.
- Coarse action properties complement rather than duplicate the exact transition tests.

### What was tricky to build

Expected reducer errors must not be treated as fuzz failures. The harness accepts only missing nonce and early deadline errors, then proves both are atomic by comparing complete before/after state and requiring no actions. Any other error fails immediately.

Action assertions inspect the final state after the complete ordered action slice. This permits combinations such as cancel-plus-stop while still enforcing that send leaves `Writing`, arm leaves `Awaiting`, successful pong leaves `Idle`, and suspicion is paired with close.

### What warrants a second pair of eyes

- Review the decoder's bit masks and enum coverage.
- Review whether the action generation upper-bound assertion has the right exception for generation-zero lifecycle actions.
- Review the expected-error allowlist when future reducer errors are introduced.

### What should be done in the future

N/A for this shipment. Future reducer changes should add readable seeds only when they introduce a new semantic category.

### Code review instructions

- Start at `TestHeartbeatFuzzAdvanceTraversesTwoHealthyCycles` and `FuzzMachinePreservesInvariants`.
- Follow `decodeFuzzEvent` into operation, identity, and time helpers.
- Run the documented 60-second command when changing reducer semantics.

### Technical details

```text
Commit: a7a49e4
Seed regressions: 100 ordinary + 100 race-enabled
Campaign: 60 seconds, 111066 executions, PASS
Production changes: none
```

## Step 8: Complete repository and release validation

This step completed the shipment-oriented validation after fuzzing. It also repaired two pre-existing test-fixture defects that prevented high-confidence repeated and repository-wide race runs from becoming clean evidence.

All configured CI checks, full tests, build/generation, vet, lint, vulnerability analysis, JavaScript syntax checks, repository-wide race tests, temporary-workspace tests, hooks, and release snapshot now pass, with one separately documented local gosec loader limitation.

### Prompt Context

**User prompt (verbatim):** (same as Step 6)

**Assistant interpretation:** Finish the requested design/fuzz work without losing the broader heartbeat shipment gate, and document every validation failure and correction.

**Inferred user intent:** Ship a clean, reproducible branch rather than stopping after one successful fuzz command.

**Commit (test fixtures):** 18812254e824d287d60bf8ecbb8e274d7e6d3d86 — "test: stabilize concurrent bus fixtures"

**Commit (generated metadata):** 217ab0b0a06a408e84a4140cb718b4dcfe7f1118 — "chore: generate heartbeat package logging metadata"

### What I did

- Ran repository-wide race tests.
- Added locking to the shared in-memory hydration/event-store fixture after the race detector identified concurrent map and protobuf clone access.
- Corrected a GoChannel test to assert ordering within session `s-a`, not callback order across independently consumed sessions.
- Ran the sessionstream package 100 times and 20 times under race detection.
- Ran `make ci-check`, including format, lint, boundary, schema-vet, logcopter, glazed-lint, tests, generation, and build.
- Generated required `internal/heartbeat/logcopter.go` metadata.
- Ran `govulncheck ./...` with no reachable vulnerabilities.
- Ran all changed shipped JavaScript clients through `node --check`.
- Ran the complete pre-push hook, including test, lint, and GoReleaser snapshot.
- Ran a fresh full repository race test after fuzz implementation.
- Tested and built through a temporary standalone `go.work` after the parent workspace rejected this worktree because it is not listed.

### Why

- The ticket's completion criteria require evidence beyond focused reducer tests.
- Existing fixture races made a full race command fail even though the runtime heartbeat packages were clean; fixing the fixture produces durable repository-wide evidence.
- Cross-session callback order is not a GoChannel contract, while per-session stream ordinal order is.
- New packages must participate in repository generation conventions.

### What worked

- `make ci-check` passed completely after generation.
- `GOWORK=off go test -race ./... -count=1` passed completely.
- `GOWORK=off go test ./pkg/sessionstream -count=100` passed.
- `GOWORK=off go test -race ./pkg/sessionstream -count=20` passed.
- `govulncheck ./...` reported no vulnerabilities in called code.
- All six JavaScript syntax checks passed.
- Pre-push tests, lint, and release snapshot passed.
- Temporary-workspace full tests and build passed.
- GoReleaser produced the `0.1.1-next` Linux archive successfully.

### What didn't work

The first full race run reproduced the historical test-helper race:

```text
WARNING: DATA RACE
Write: (*testHydrationStore).Apply at pkg/sessionstream/hub_test.go:384
Read:  (*testHydrationStore).Cursor at pkg/sessionstream/hub_test.go:409
Test:  TestHubEventBusFallsBackWithoutStreamID
```

Adding one `sync.RWMutex` around snapshot and event fixture state removed both map races and concurrent protobuf-clone races. One hundred focused race runs and the full race suite then passed.

A subsequent 20-run package test exposed the known cross-session ordering assumption:

```text
TestHubEventBusGoChannelRoundTrip
"1700000000000000002" is not greater than "1700000000000000003"
```

The consumer may interleave different sessions. The test now filters consumed records to `s-a` and asserts order only for that session.

The first `make ci-check` failed with:

```text
logcopter-gen: generated file is not current:
pkg/sessionstream/transport/ws/internal/heartbeat/logcopter.go
```

Running `make logcopter-generate` created the expected package metadata; the rerun passed.

The repository's `make gosec` command reported zero security issues but exited nonzero because the locally installed `gosec` identifies itself as `dev` and failed package loading for internal-package imports:

```text
could not import .../transport/ws/internal/heartbeat (invalid package name: "")
use of internal package .../internal/xgojaruntime not allowed
Issues : 0
```

This is a scanner/package-loader limitation rather than a reported finding. `govulncheck`, configured lint, build, tests, and prior GitHub security jobs provide the usable security evidence; the branch should run GitHub's gosec job when pushed.

Direct workspace mode initially failed because the parent `go.work` does not list `sessionstream-p111`:

```text
pattern ./pkg/sessionstream/transport/ws/...: directory prefix ... does not contain modules listed in go.work
```

A temporary `go.work` containing this module passed full tests and build, while `GOWORK=off` remained the repository's configured mode.

### What I learned

- High-repetition validation can turn “known flaky” behavior into a precise contract correction.
- Repository generation checks are effective at keeping even logic-only internal packages integrated with logging metadata.
- Workspace validation must distinguish source failure from a parent workspace that simply does not include the worktree.

### What was tricky to build

The hydration fixture embeds the same store in event-store tests, so one mutex needed to protect snapshots, cloned protobuf payloads, event cursor, events, and recorded errors without introducing nested-lock calls. `View` delegates to the already locked `Snapshot`, while direct methods take their own appropriate read or write lock.

The GoChannel test needed to preserve its real assertion—stream-derived ordinals increase for one session—without imposing global consumer callback order across sessions.

### What warrants a second pair of eyes

- Review the fixture mutex coverage and ensure no test method recursively reacquires it.
- Confirm the gosec package-loader failure reproduces independently of this branch before changing security tooling.
- Review the final branch diff against `21724ab` before deciding whether to stack or fold it into PR #10.

### What should be done in the future

- Push the branch and run GitHub checks.
- Decide whether SESSIONSTREAM-005 should be a stacked PR or folded into PR #10.

### Code review instructions

- Run `make ci-check` and `GOWORK=off go test -race ./... -count=1`.
- Run `lefthook run pre-push` for release evidence.
- Review `bus_test.go` and `hub_test.go` separately from heartbeat runtime code; they are validation-fixture repairs.

### Technical details

```text
Configured CI: PASS
Full repository race: PASS
Temporary go.work test/build: PASS
Govulncheck: no reachable vulnerabilities
Pre-push release snapshot: PASS
Local gosec: 0 issues, nonzero loader failure (documented)
```

## Step 9: Detach accepted observer records from producer cancellation

PR #11 review correctly identified that the bounded observer dispatcher retained each producer's original context. A record accepted from `ServeHTTP` or `readLoop` could wait behind a slow callback until the producer returned and canceled that context, causing the eventual observer callback to receive `context.Canceled` and potentially discard telemetry that shutdown promised to drain.

The queue now stores `context.WithoutCancel(ctx)`. This preserves request-scoped values while deliberately removing producer cancellation and deadlines. A deterministic regression test blocks the dispatcher, enqueues a value-bearing context with a deadline, cancels the producer, and then proves the drained callback retains the value while observing no error or deadline.

### Prompt Context

**User prompt (verbatim):** "Address
pkg/sessionstream/transport/ws/observer.go
    if ctx == nil {
        ctx = context.Background()
    }
    item := observedTransportRecord{ctx: ctx, rec: cloneTransportRecord(rec)}
@chatgpt-codex-connector
chatgpt-codex-connector Bot
16 minutes ago

P2 Badge Detach contexts before queuing observer callbacks

When observer delivery falls behind, records emitted from ServeHTTP or readLoop can remain queued until after the handler returns and its deferred cancellation runs. Because the queue retains the original context, OnTransport then receives context.Canceled; observers that use it for database writes or telemetry export can discard an accepted record even though shutdown claims to drain accepted observations. Preserve context values while detaching cancellation (for example, with context.WithoutCancel) before enqueueing.

Useful? React with 👍 / 👎."

**Assistant interpretation:** Apply the review finding by detaching queued observer contexts from producer cancellation, preserve context values, add a deterministic backlog regression test, validate, commit, and update PR #11.

**Inferred user intent:** Ensure every accepted best-effort observation remains usable when the asynchronous dispatcher drains it after the originating request or connection operation has ended.

**Commit (code):** 5005973cda9fecca3cdedbd5616b8ca6126f01d0 — "fix(ws): detach queued observer contexts"

### What I did

- Changed `observe` to enqueue `context.WithoutCancel(ctx)` after normalizing `nil` to `context.Background()`.
- Expanded the `TransportObserver` contract comment to state that values survive while producer cancellation and deadlines do not.
- Added `TestTransportObserverQueueDetachesCancellationAndPreservesValues`.
- Blocked the first observer callback so the target record necessarily remained queued while its parent context was canceled.
- Ran the focused test 100 times, its race-enabled form 50 times, full repository tests, `go vet`, and the pre-commit test/lint hook.

### Why

- Accepted observer records are server-owned queue entries once admission succeeds.
- Retaining producer cancellation makes queue delay alter delivery semantics and weakens graceful-drain guarantees.
- Context values may carry trace or correlation metadata that observers still need after producer completion.

### What worked

- `context.WithoutCancel` provided the exact contract: values preserved, `Done()` detached, `Err()` nil, and no inherited deadline.
- The deterministic backlog test passed 100 ordinary repetitions and 50 race-enabled repetitions.
- Full tests, vet, and the final pre-commit hook passed.

### What didn't work

The first two commit attempts failed lint because the regression test used a non-exhaustive switch over the typed `TransportStage` enum:

```text
pkg/sessionstream/transport/ws/server_test.go:869:3: missing cases in switch of type ws.TransportStage: ... (exhaustive)
```

Adding a `default` branch did not satisfy this repository's exhaustive-linter policy. Replacing the two-case switch with explicit `if` conditions removed the enum-exhaustiveness claim and the final pre-commit run passed.

### What I learned

- Asynchronous queue ownership must include cancellation ownership; copying a context pointer does not transfer a producer's lifetime safely.
- The repository's exhaustive linter treats a switch over a closed enum as a declaration that every member should be named, even when a default exists.

### What was tricky to build

The regression needed to prove actual queue delay rather than merely pass an already-canceled context. It first blocks the dispatcher on a `Connected` callback, then admits the target record, cancels its value-bearing deadline context, and only then releases dispatch. This ordering reproduces the review scenario deterministically without sleeps.

### What warrants a second pair of eyes

- Confirm that detaching deadlines is the intended observer contract for database and telemetry exporters; the API comment now makes this behavior explicit.
- Review whether any observer intentionally depended on producer cancellation. Such a dependency is incompatible with queued graceful delivery and should use its own operation timeout instead.

### What should be done in the future

- Watch PR #11 GitHub checks and resolve the review thread after the pushed commit is visible.

### Code review instructions

- Start with `pkg/sessionstream/transport/ws/observer.go:observe` and the new test beside the bounded-queue test in `server_test.go`.
- Run `GOWORK=off go test -race ./pkg/sessionstream/transport/ws -run TestTransportObserverQueueDetachesCancellationAndPreservesValues -count=50`.
- Run `GOWORK=off go test ./... -count=1` and `GOWORK=off go vet ./...`.

### Technical details

```text
Producer context: value=request-value, deadline=set, then canceled
Queued observer context: value=request-value, deadline=none, err=nil
Focused ordinary repetitions: 100 PASS
Focused race repetitions: 50 PASS
Full repository tests: PASS
Go vet: PASS
Pre-commit test/lint: PASS
```

## Step 10: Design supervisor arbitration and its runtime fuzz harness

A second PR #11 finding exposed a boundary not covered by the reducer fuzzer: the supervisor can apply an overdue deadline recursively while handling a writer acknowledgement, before reading a timely pong already admitted to the bounded event queue. The reducer is correct for the sequence it receives, but the runtime serializes the concurrent history into the wrong reducer trace.

I created a 7,299-word intern guide that reconstructs the execution, defines the supervisor refinement invariant, specifies one shared deadline-arbitration helper for timer and overdue-on-arm paths, and designs deterministic and state-aware runtime coverage. The guide includes labeled-transition-system, partial-order, happens-before, linearization, trace-refinement, safety/liveness, failure-detector, and observational-equivalence foundations, followed by fuzzing techniques, APIs, pseudocode, diagrams, phases, decisions, risks, and review checklists. It passed docmgr validation and was uploaded successfully to reMarkable.

### Prompt Context

**User prompt (verbatim):** "write it out in a new design doc, and also add a section that goes into computer science theoretic underpinnings of the design and how it impacts invariants and creating fuzz harnesses, and what some design technique for fuzzing are. Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new comprehensive SESSIONSTREAM-005 design document that teaches the reported supervisor-ordering problem, its theoretical basis, the proposed shared arbitration API, and a practical runtime fuzzing strategy, then validate and upload it to reMarkable.

**Inferred user intent:** Give a new intern enough theory, repository orientation, implementation detail, and verification guidance to fix the bug without conflating pure reducer correctness with concurrent runtime serialization.

### What I did

- Added `design-doc/03-intern-guide-to-heartbeat-supervisor-event-arbitration-and-runtime-fuzzing.md` through docmgr.
- Inspected line-numbered reducer, supervisor, and existing fuzz implementation evidence.
- Documented the exact `tw < tp < deadline < resume` schedule and why `PendingPongAt` and timer-only draining do not cover it.
- Specified `applyDeadlineAfterAdmittedEvents` and both production call sites in pseudocode.
- Defined reducer-local and supervisor-refinement invariants.
- Designed `FuzzHeartbeatDeadlineArbitration`, its byte encoding, oracle, seeds, properties, campaign commands, and bounded resource contract.
- Added theoretical sections on labeled transition systems, trace refinement, concurrent histories, partial orders, happens-before, linearization points, safety, liveness, failure detectors, and observational equivalence.
- Added practical sections on property-based, model-based, stateful, structure-aware, boundary, equivalence-class, metamorphic, differential, schedule, shrinking, and coverage-guided fuzzing.
- Related the four primary source/test files and updated the ticket changelog.
- Ran frontmatter validation, `docmgr doctor`, Markdown fence checks, and `git diff --check`.
- Performed the required reMarkable dry run and real upload.

### Why

- The existing reducer harness validates the transition function after event ordering has already been chosen.
- The reported defect lives in the refinement from concurrent source history to reducer event trace.
- A separate runtime arbitration harness preserves the purity and fault localization of the reducer fuzzer while testing the missing contract.
- The implementation is small, but a new engineer needs a precise model to avoid replacing one scheduler-dependent ordering rule with another.

### What worked

- The generated design document passed `docmgr validate frontmatter`.
- `docmgr doctor --ticket SESSIONSTREAM-005 --stale-after 30` reported all checks passed.
- The 1,403-line document contains balanced code fences and no placeholders.
- reMarkable dry-run rendering completed successfully.
- The real upload returned:

```text
OK: uploaded SESSIONSTREAM-005 Supervisor Arbitration and Runtime Fuzzing Guide.pdf -> /ai/2026/08/10/SESSIONSTREAM-005
```

### What didn't work

N/A

### What I learned

- A reducer fuzzer establishes properties of `delta(state,event)`, while the supervisor needs a separate refinement oracle for `serialize(concurrent history)`.
- Native Go fuzzing is most effective here when scheduler state is encoded as bounded mailbox data rather than delegated to nondeterministic goroutine scheduling.
- The core oracle can remain independent and small: the final result is successful exactly when the admitted queue contains a current-nonce pong strictly before the deadline.

### What was tricky to build

The guide needed to distinguish three order concepts without collapsing them: event timestamps, channel admission order, and supervisor selection order. The proposed linearization point is the nonblocking drain's empty observation. It guarantees consideration of events admitted before the deadline decision but does not claim that every remotely timely pong has already been admitted. That limitation preserves accurate fixed-threshold failure-detector semantics.

The fuzz design also needed to cover production arbitration without adding channels or scheduler randomness to the pure reducer harness. The solution is a second target over a directly testable bounded helper, combined with one deterministic production-wiring regression and existing real WebSocket race tests.

### What warrants a second pair of eyes

- Review whether `heartbeatEventQueueSize` or `cap(events)` should define the drain bound; the guide recommends the named constant.
- Confirm the factored test seam exercises the overdue `ActionArmDeadline` path rather than only the helper in isolation.
- Review the independent oracle and ensure future event kinds cannot silently invalidate its pong-only assumption.

### What should be done in the future

- Implement the shared arbitration helper and route both deadline paths through it.
- Add deterministic helper and production-wiring regressions.
- Add and campaign the separate runtime arbitration fuzz target.
- Reply to and resolve the PR #11 review thread after validation evidence is pushed.

### Code review instructions

- Read the new guide from Executive Summary through Sections 3–6 before reviewing pseudocode.
- Compare its current-state citations against `heartbeat.go:121-130` and `heartbeat.go:192-212`.
- Review Sections 8–10 for fuzz oracle independence and coverage-layer boundaries.
- Validate with `docmgr doctor --ticket SESSIONSTREAM-005 --stale-after 30`.

### Technical details

```text
Document: design-doc/03-intern-guide-to-heartbeat-supervisor-event-arbitration-and-runtime-fuzzing.md
Length: 1,403 lines; 7,299 words
Related source/test files: 4
Frontmatter: PASS
Docmgr doctor: PASS
Markdown fences: balanced
reMarkable: /ai/2026/08/10/SESSIONSTREAM-005/SESSIONSTREAM-005 Supervisor Arbitration and Runtime Fuzzing Guide.pdf
Implementation status: proposed; no runtime code changed in this step
```

## Step 11: Reproduce and fix overdue deadline arbitration with a focused fuzzer

I implemented the runtime-seam fuzzer before completing the drain logic and deliberately routed the overdue-on-arm production branch through an incomplete helper that applied the deadline immediately. The checked-in timely-pong seed failed at baseline, reproducing the review finding without relying on scheduler luck. No failing intermediate state was committed.

I then implemented the bounded admitted-event drain, routed both timer expiry and overdue-on-arm expiry through it, and retained the same seed as a permanent regression. The final harness constructs only reachable `Awaiting` detector states, generates bounded pong mailboxes, computes its expected phase from an independent identity/time predicate, and checks queue, phase, generation, and action contracts.

### Prompt Context

**User prompt (verbatim):** "ok, let's do that. Implement the fuzzer, keep a detailed diary as you work, and commit at appropriate intervals."

**Assistant interpretation:** Build the separate supervisor-arbitration fuzz target, prove that it fails on the current defect before fixing production, implement shared deadline arbitration, validate broadly, and preserve detailed chronological evidence in code and documentation commits.

**Inferred user intent:** Establish that the new fuzz harness genuinely detects the P1 review bug rather than merely passing against a pre-corrected helper, then ship a durable and reviewable fix.

**Commit (code):** 5a1d9ebfe00e00b9712777d8a1db617753e6f00a — "fix(ws): arbitrate admitted pongs before deadlines"

### What I did

- Added the package-private `applyHeartbeatDeadlineAfterAdmittedEvents` production seam.
- Initially implemented it as direct deadline application and routed the overdue-on-arm branch through it solely to test whether the new harness reproduced the defect.
- Added `pkg/sessionstream/transport/ws/heartbeat_arbitration_test.go`.
- Added deterministic timely, empty, exact, late, stale, mixed-order, and final-queue-slot regressions.
- Added `FuzzHeartbeatDeadlineArbitration` with ten readable seeds.
- Encoded current/stale/empty/future nonce classes and before/exact/after/before-write timestamp classes in each event byte.
- Added a bounded eight-event mailbox, independently computed timely-current-pong oracle, reachable reducer setup, and action collector.
- Ran the seed target against the intentionally incomplete production seam and captured the expected failure.
- Replaced the incomplete helper with a bounded nonblocking drain followed by exactly one deadline application.
- Replaced the duplicated timer-channel drain with the shared helper.
- Captured `heartbeatNow()` once in the overdue-on-arm action branch.
- Ran focused ordinary and race-enabled repetitions, a 60-second campaign, full tests, full repository race, vet, lint, and `make ci-check`.
- Committed the green code and tests as one focused code commit.

### Why

- A seeded fuzz harness should demonstrate sensitivity to the known defect before it is trusted as regression coverage.
- Both deadline origins represent the same protocol decision and require one shared arbitration policy.
- A bounded direct helper turns concurrent mailbox scheduling into deterministic input data that Go's fuzzer can mutate and shrink.
- Keeping the fuzzer in package `ws` tests the runtime refinement seam without adding concurrency or transport behavior to the pure reducer harness.

### What worked

The first baseline seed run caught the defect immediately. Seed 1 represented one current-nonce pong one nanosecond before the deadline. The oracle expected `Idle`; the incomplete production seam applied deadline first and produced `Suspected`:

```text
--- FAIL: FuzzHeartbeatDeadlineArbitration/seed#1
Error: Not equal:
  expected: 0x1
  actual  : 0x4
```

Additional seeds failed because the queue remained undrained, confirming broad sensitivity rather than one hard-coded assertion.

After the production fix:

- 100 focused ordinary repetitions passed.
- 100 focused race-enabled repetitions passed.
- The 60-second campaign passed 877,473 executions.
- It found 16 new interesting inputs, for 26 total corpus entries.
- A later uninterrupted 10-minute campaign reused that 26-input cache, passed 8,837,183 executions, found no additional coverage-interesting input, and produced no failure.
- Full repository tests passed.
- Full repository race tests passed.
- `go vet`, lint, and `make ci-check` passed.
- No failure corpus was produced after the fix.

### What didn't work

The first lint run after the successful campaign failed on an ineffectual initialization in the event decoder:

```text
pkg/sessionstream/transport/ws/heartbeat_arbitration_test.go:204:2: ineffectual assignment to at (ineffassign)
    at := deadline.Add(-delta)
```

Every switch branch replaced `at`, so I changed the declaration to `var at time.Time`, formatted, and reran lint successfully.

The deliberately incomplete helper caused seeds 1–9 to fail before the fix. This was expected reproduction evidence, not an attempted final implementation. It was never committed.

The first attempted 10-minute post-fix campaign was interrupted at approximately 6m21s when a new user question arrived. It had completed roughly 5.59 million executions with no failure. I did not count that partial run as the requested campaign; I started a fresh uninterrupted 10-minute run, which passed 8,837,183 executions.

### What I learned

- A baseline seed can serve as a deterministic counterexample while still participating in later mutation and minimization.
- The strongest oracle is smaller than the runtime: success exists if and only if the bounded admitted queue contains a current-nonce pong strictly before the deadline.
- Applying the deadline after draining remains useful even after a pong succeeds because the reducer's phase and generation rules make it harmlessly stale.
- The final queue-slot seed is important: it proves the work bound does not stop one event too early.

### What was tricky to build

The fuzzer needed to exercise the real production policy without reproducing nondeterministic Go `select` scheduling. The extracted helper is now called by both production deadline paths, so encoding the admitted queue as bytes tests the exact arbitration operation while keeping every failure replayable.

The action oracle had to account for multiple queued pongs. The first timely current pong returns the machine to `Idle`; later pongs become stale, and the subsequent deadline is a no-op. Therefore a successful trace must contain exactly one `ActionRecordPong` and no suspicion/close pair, regardless of duplicates.

The pre-fix experiment also needed to preserve a green commit history. I created and executed the incomplete seam locally, captured its failures, then replaced it with the correct bounded implementation before staging the code commit.

### What warrants a second pair of eyes

- Review the helper's deadline linearization point: nonblocking queue-empty observation followed by deadline application.
- Confirm that limiting receives to `heartbeatEventQueueSize` remains correct if the queue's event kinds or capacity change.
- Review the fuzzer oracle if anything besides pong events is later admitted to `heartbeatRuntime.events`.
- Confirm both production branches continue to call the shared helper in future refactors.

### What should be done in the future

- Push the code and documentation commits to PR #11.
- Run a longer arbitration fuzz campaign if review requests it.
- Reply to the P1 thread with pre-fix reproduction, commit, campaign, and validation evidence, then resolve it.

### Code review instructions

- Start at `applyHeartbeatDeadlineAfterAdmittedEvents` in `heartbeat.go`.
- Verify both `ActionArmDeadline` and `deadlineC` branches call it.
- Review the ten seeds and the independent oracle in `heartbeat_arbitration_test.go`.
- Reproduce seeds with `GOWORK=off go test ./pkg/sessionstream/transport/ws -run='^FuzzHeartbeatDeadlineArbitration$' -count=100`.
- Campaign with `GOWORK=off go test ./pkg/sessionstream/transport/ws -run='^$' -fuzz='^FuzzHeartbeatDeadlineArbitration$' -fuzztime=60s`.
- Run `GOWORK=off go test -race ./... -count=1` and `make ci-check`.

### Technical details

```text
Pre-fix seed reproduction: FAIL as expected
Shared production helper: applyHeartbeatDeadlineAfterAdmittedEvents
Queue work bound: 8 events
Deadline applications per arbitration: 1
Readable seeds: 10
Focused ordinary repetitions: 100 PASS
Focused race repetitions: 100 PASS
Initial 60-second fuzz executions: 877473
Initial new interesting: 16
Cached interesting corpus: 26
Complete 10-minute fuzz executions: 8837183
Additional interesting in cached campaign: 0
Post-fix failures: 0
Full test/race/vet/lint/CI: PASS
Code commit: 5a1d9ebfe00e00b9712777d8a1db617753e6f00a
```
