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
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine.go
      Note: Implemented in Step 3 at commit d0693bf
    - Path: repo://pkg/sessionstream/transport/ws/internal/heartbeat/machine_test.go
      Note: Validated reducer invariants in Step 3 at commit d0693bf
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
