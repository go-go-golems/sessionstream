---
Title: Investigation diary
Ticket: SESSIONSTREAM-003
Status: active
Topics:
    - architecture
    - backend
    - event-streaming
    - framework
    - onboarding
    - code-review
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/chapters/phase-3-hydration-and-reconnect.md
      Note: Teaches fanout-only subscribe semantics
    - Path: cmd/sessionstream-systemlab/phase2_actions.go
      Note: Phase 2 scenario actions split
    - Path: cmd/sessionstream-systemlab/phase2_checks.go
      Note: Phase 2 checks split
    - Path: cmd/sessionstream-systemlab/phase2_lab.go
      Note: Phase 2 constants and DTOs after file split
    - Path: cmd/sessionstream-systemlab/phase2_projections.go
      Note: Phase 2 projections and bus hooks split
    - Path: cmd/sessionstream-systemlab/phase2_render.go
      Note: Phase 2 transcript rendering split
    - Path: cmd/sessionstream-systemlab/phase2_runtime.go
      Note: Phase 2 runtime setup/lifecycle split
    - Path: cmd/sessionstream-systemlab/phase3_lab.go
      Note: Phase 3 now uses shared websocket trace hooks (commit aaac81d34cadad21820f68b7d335db701c0fc8b8)
    - Path: cmd/sessionstream-systemlab/phase4_lab.go
      Note: Phase 4 now uses shared websocket trace hooks (commit aaac81d34cadad21820f68b7d335db701c0fc8b8)
    - Path: cmd/sessionstream-systemlab/phase5_actions.go
      Note: Phase 5 scenario actions split
    - Path: cmd/sessionstream-systemlab/phase5_checks.go
      Note: Phase 5 checks split
    - Path: cmd/sessionstream-systemlab/phase5_lab.go
      Note: |-
        Phase 5 now uses shared websocket trace hooks (commit aaac81d34cadad21820f68b7d335db701c0fc8b8)
        Phase 5 constants and DTOs after file split
    - Path: cmd/sessionstream-systemlab/phase5_projections.go
      Note: Phase 5 projections split
    - Path: cmd/sessionstream-systemlab/phase5_runtime.go
      Note: Phase 5 runtime setup/lifecycle split
    - Path: cmd/sessionstream-systemlab/snapshot_helpers.go
      Note: Shared snapshot and protobuf payload encoding helpers (commit aaac81d34cadad21820f68b7d335db701c0fc8b8)
    - Path: cmd/sessionstream-systemlab/ws_hooks.go
      Note: Shared websocket trace hook builder (commit aaac81d34cadad21820f68b7d335db701c0fc8b8)
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Documents fanout-only websocket semantics and advisory sinceOrdinal
    - Path: pkg/sessionstream/transport/ws/server_test.go
      Note: Tests unsupported websocket command frames
    - Path: ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/changelog.md
      Note: Ticket changelog updated during review
    - Path: ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/01-sessionstream-code-review-and-architecture-audit.md
      Note: Primary review deliverable whose creation and evidence gathering are recorded by this diary
    - Path: ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/tasks.md
      Note: Task status for review and follow-up work
ExternalSources: []
Summary: Chronological diary for the SESSIONSTREAM-003 architecture audit and code review.
LastUpdated: 2026-04-29T14:44:36-04:00
WhatFor: Use this to understand how the review was performed and what evidence was collected.
WhenToUse: Read before continuing the code review, acting on recommendations, or validating the reMarkable/docmgr delivery.
---





# Investigation diary

## Goal

This diary records the SESSIONSTREAM-003 code review of the `sessionstream` repository. It captures ticket setup, repository inventory, commands run, evidence gathered, document writing, validation, and reMarkable delivery steps.

## Step 1: Created the docmgr ticket and scoped the review

I created a new docmgr ticket under `sessionstream/ttmp` so the review has a durable home in the repository's ticket documentation system. I added a primary design-doc for the audit and a reference diary for chronological notes.

The scope is intentionally broad: core APIs, architecture, unclear contracts, duplicated code, overgrown files, extraction-era naming, tests, and systemlab maintainability.

### Prompt Context

**User prompt (verbatim):**

```text
Now create a new docmgr ticket in sessionstream/ with docmgr --root sessionstream/ttmp and do an in depth code review of sessionstream. Keep an eye out for unclear apis, obtuse code, files that are too long, deprecated code, problematic architecture, etc...

 reate a detailed analysis / design / code review that is very detailed for a new intern, explaining all the parts of the     
 system needed to understand what it is, with prose paragraphs and bullet                                                              
 point sand pseudocode and diagrams and api references and file                                                                        
   references.                                                                                                                         
   It should be very clear and detailed. Store in the ticket and the nupload to remarkable.
```

**Assistant interpretation:** Create a new docmgr ticket, perform a deep evidence-backed code review of `sessionstream`, write a detailed intern-oriented design/code-review document with diagrams, pseudocode, and file references, store it in the ticket, validate it, and upload it to reMarkable.

**Inferred user intent:** The user wants a durable onboarding and cleanup-planning artifact, not just a short chat summary.

### What I did

- Ran `docmgr --root ttmp status --summary-only` inside `sessionstream`.
- Created ticket `SESSIONSTREAM-003` with title `Code review and architecture audit for sessionstream`.
- Added a design document titled `Sessionstream code review and architecture audit`.
- Added this diary document titled `Investigation diary`.

### Why

A broad architecture audit can become hard to resume if it only exists in chat. The ticket creates stable paths for the report, diary, tasks, changelog, and later follow-up work.

### What worked

- `docmgr` created the workspace cleanly.
- The repository already had vocabulary topics for the main review categories.

### What didn't work

- Nothing failed in this step.

### What I learned

- Existing tickets `SESSIONSTREAM-001` and `SESSIONSTREAM-002` already document extraction and systemlab planning, so this ticket can focus on review and cleanup rather than original design intent.

### What was tricky to build

The prompt asked for a code review, onboarding guide, design analysis, and reMarkable upload in one pass. I treated the primary document as a hybrid architecture audit and cleanup plan, with evidence-backed recommendations.

### What warrants a second pair of eyes

- Whether the ticket topics should include a more specific future topic like `api-contracts`.

### What should be done in the future

- Convert high-priority recommendations into separate implementation tickets after review.

### Code review instructions

- Start with `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/01-sessionstream-code-review-and-architecture-audit.md`.
- Check that the recommendations match the current code before implementing them.

### Technical details

Commands:

```bash
cd sessionstream
docmgr --root ttmp status --summary-only
docmgr --root ttmp ticket create-ticket --ticket SESSIONSTREAM-003 --title "Code review and architecture audit for sessionstream" --topics architecture,backend,event-streaming,framework,onboarding,code-review,cleanup
docmgr --root ttmp doc add --ticket SESSIONSTREAM-003 --doc-type design-doc --title "Sessionstream code review and architecture audit"
docmgr --root ttmp doc add --ticket SESSIONSTREAM-003 --doc-type reference --title "Investigation diary"
```

## Step 2: Inventoried the codebase and identified hotspots

I inventoried all Go files outside `ttmp` and `dist`, then counted lines per file. This identified the conceptual surface area and highlighted the largest files before making qualitative claims.

The core package is compact. The largest and most duplication-prone files are in `cmd/sessionstream-systemlab`, especially Phase 2 and Phase 5.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Gather evidence before writing conclusions.

**Inferred user intent:** The review should be grounded in concrete file structure and line counts.

### What I did

- Listed Go files with `rg --files`.
- Counted lines with `wc -l`.
- Read core files in `pkg/sessionstream`.
- Read store implementations, websocket transport, chat demo, systemlab server, and phase files.
- Ran coverage to identify untested public behaviors.

### Why

Line counts do not prove bad design by themselves, but they are useful signals. They showed that the framework core is small while the teaching app carries most complexity.

### What worked

The inventory clearly separated:

- compact framework core;
- optional store/transport adapters;
- chat demo;
- systemlab app with long phase files.

### What didn't work

No command failed in this step.

### What I learned

Important counts:

```text
797 cmd/sessionstream-systemlab/phase2_lab.go
495 cmd/sessionstream-systemlab/lab_environment.go
484 cmd/sessionstream-systemlab/phase5_lab.go
455 pkg/sessionstream/transport/ws/server.go
425 examples/chatdemo/chat.go
423 cmd/sessionstream-systemlab/phase3_lab.go
350 pkg/sessionstream/hub.go
288 pkg/sessionstream/hydration/sqlite/store.go
6472 total Go lines
```

Coverage run:

```text
cmd/sessionstream-systemlab coverage: 64.4%
examples/chatdemo coverage: 70.6%
pkg/sessionstream coverage: 72.4%
pkg/sessionstream/hydration/memory coverage: 64.1%
pkg/sessionstream/hydration/sqlite coverage: 48.9%
pkg/sessionstream/transport/ws coverage: 62.7%
```

### What was tricky to build

Some architecture issues are cross-file contract issues rather than single-line bugs. For example, `sinceOrdinal` is parsed in the websocket server, but the missing behavior is only obvious after comparing subscribe handling with fanout behavior.

### What warrants a second pair of eyes

- The local ordinal restart concern should be verified with a focused failing test before implementation.
- The intended semantics of `Snapshot(asOf)` and websocket `sinceOrdinal` need owner confirmation.

### What should be done in the future

- Add follow-up tickets for the highest-priority findings: local ordinal seeding, hydration contract cleanup, websocket subscribe semantics, and systemlab refactor.

### Code review instructions

Start with these files:

- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/consumer.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`

Validation commands:

```bash
cd sessionstream
go test ./... -coverprofile=/tmp/sessionstream-cover.out
go tool cover -func=/tmp/sessionstream-cover.out
```

### Technical details

Commands:

```bash
cd sessionstream
rg --files -g '*.go' -g '!ttmp/**' -g '!dist/**' | sort
wc -l $(rg --files -g '*.go' -g '!ttmp/**' -g '!dist/**' | sort) | sort -n
go test ./... -coverprofile=/tmp/sessionstream-cover.out
go tool cover -func=/tmp/sessionstream-cover.out
```

## Step 3: Wrote the primary architecture audit

I wrote the primary design/code-review document as an intern-oriented guide. It explains the system's mental model, maps current architecture, documents concrete findings, and proposes phased cleanup work.

The report intentionally distinguishes observed behavior from recommendations. Each issue includes file references, why it matters, and a cleanup sketch or pseudocode.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce a long-form, clear, detailed architecture/code-review artifact in the ticket.

**Inferred user intent:** The final report should be readable by a new intern and actionable by maintainers.

### What I did

- Rewrote `design-doc/01-sessionstream-code-review-and-architecture-audit.md`.
- Included an executive summary, repository map, mental model diagram, pseudocode, current-state architecture, detailed findings, and cleanup phases.
- Covered core hub behavior, schema registry, stores, websocket transport, bus consumer, ordinals, chat demo, and systemlab.

### Why

A chat-only response would be too shallow for the requested outcome. The document is structured to serve as both onboarding material and a cleanup roadmap.

### What worked

The report now gives interns a reading path and maintainers a prioritized list of issues.

### What didn't work

No write operation failed.

### What I learned

The strongest cleanup candidates are contract mismatches, not ordinary code style issues:

- `Snapshot(asOf)` is not implemented as historical lookup.
- `sinceOrdinal` is not replay semantics.
- local ordinals are in-memory-only.
- default projection error handling can advance cursor despite dropped state.
- malformed Watermill messages are acked silently.

### What was tricky to build

The document had to be detailed without over-claiming. For example, the websocket server may intentionally be a lab/reference transport; the report therefore recommends documenting or hardening it rather than simply calling it wrong.

### What warrants a second pair of eyes

- Whether to remove `asOf` or implement historical snapshots.
- Whether to keep `evtstream` names for compatibility.
- Whether `ProjectionErrorPolicyAdvance` should remain the default.

### What should be done in the future

- Turn findings into ticketed work with acceptance tests.

### Code review instructions

Review the report sections in this order:

1. Executive summary.
2. Mental model.
3. Findings 1 through 6 for API/correctness issues.
4. Findings 12 through 15 for maintainability and tests.

### Technical details

Primary document path:

```text
ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/01-sessionstream-code-review-and-architecture-audit.md
```

## Step 4: Validated docmgr metadata and uploaded the bundle to reMarkable

I validated the ticket with `docmgr doctor`, fixed missing vocabulary entries, and uploaded the ticket bundle to reMarkable. The first real upload attempt failed because the diary stored the verbatim prompt as an inline quoted string containing literal `\n` sequences; Pandoc treated those as LaTeX commands during PDF generation. I moved the verbatim prompt into a fenced `text` block and reran the dry-run and upload successfully.

The final reMarkable bundle includes the ticket index, primary architecture audit, investigation diary, tasks, and changelog.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the ticket and deliver the generated documentation to reMarkable.

**Inferred user intent:** The user wants the code review available as a readable PDF on the reMarkable, not only in the repository.

### What I did

- Ran `docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30`.
- Added vocabulary entries for `cleanup` and `code-review`.
- Reran doctor successfully.
- Ran a dry-run `remarquee upload bundle`.
- Fixed the Pandoc/LaTeX issue caused by literal `\n` in the inline prompt text.
- Uploaded the bundle to `/ai/2026/04/29/SESSIONSTREAM-003`.
- Verified the upload with `remarquee cloud ls`.

### Why

Docmgr validation catches stale metadata and unknown vocabulary before the ticket becomes part of the long-term knowledge base. The dry-run prevents accidental upload mistakes and catches PDF-rendering errors early.

### What worked

- `docmgr doctor` passed after adding the two vocabulary entries.
- `remarquee upload bundle --dry-run` passed after the prompt formatting fix.
- The final upload succeeded.
- The cloud listing showed the uploaded document.

### What didn't work

The first real upload failed with this error:

```text
Error: pandoc failed: Error producing PDF.
! Undefined control sequence.
l.2222 architecture, etc…\n

: exit status 43
```

Cause: the diary had a long inline Markdown string containing literal `\n` sequences in the verbatim prompt. During PDF generation, Pandoc/LaTeX interpreted `\n` as an undefined command.

Fix: move the verbatim prompt into a fenced `text` code block with actual line breaks.

### What I learned

For reMarkable/Pandoc delivery, long verbatim prompts are safer in fenced code blocks than inline quoted strings, especially if they contain backslash sequences.

### What was tricky to build

The upload pipeline validates Markdown through a PDF renderer, so Markdown that is acceptable in GitHub-style rendering can still fail in LaTeX. The symptom pointed to a generated line number in a temporary bundle, so I searched the ticket docs for literal `\n` and fixed the diary source.

### What warrants a second pair of eyes

- Confirm that adding `cleanup` and `code-review` to the repository vocabulary is acceptable for future tickets.

### What should be done in the future

- Prefer fenced blocks for verbatim user prompts in diaries that may be rendered to PDF.

### Code review instructions

Validate with:

```bash
cd sessionstream
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
remarquee cloud ls /ai/2026/04/29/SESSIONSTREAM-003 --long --non-interactive
```

### Technical details

Commands:

```bash
cd sessionstream
docmgr --root ttmp vocab add --category topics --slug cleanup --description "Code cleanup, refactoring, maintainability, and simplification work."
docmgr --root ttmp vocab add --category topics --slug code-review --description "Evidence-backed source code review, architecture audit, and quality assessment."
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
remarquee status
remarquee cloud account --non-interactive
remarquee upload bundle --dry-run ... --name "SESSIONSTREAM-003 Code Review and Architecture Audit" --remote-dir "/ai/2026/04/29/SESSIONSTREAM-003" --toc-depth 2
remarquee upload bundle ... --name "SESSIONSTREAM-003 Code Review and Architecture Audit" --remote-dir "/ai/2026/04/29/SESSIONSTREAM-003" --toc-depth 2
remarquee cloud ls /ai/2026/04/29/SESSIONSTREAM-003 --long --non-interactive
```

## Step 5: Expanded remediation plan and started the first implementation slice

I expanded the ticket from audit-only into an implementation plan and started work on the no-backwards-compatibility cleanup items that were safe to do before the full replay-store rewrite. This first implementation slice focused on removing misleading API surface and making the existing runtime safer without yet introducing the full event log schema.

The code now uses `sessionstream` runtime names outside `ttmp`, treats the websocket adapter as fanout/subscription-only, seeds local ordinals from the store cursor, defaults projection failures to fail-closed behavior, splits UI and timeline projection policy options, adds an error observer hook, and clones schema prototypes defensively.

### Prompt Context

**User prompt (verbatim):**

```text
ok, create a detailed remediation plan.

 reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the     
 system needed to understand what it is, with prose paragraphs and bullet                                                              
 point sand pseudocode and diagrams and api references and file                                                                        
   references.                                                                                                                         
   It should be very clear and detailed. Store in the ticket and the nupload to remarkable.   

And then add tasks, in phases, and get to work.
```

Follow-up prompt:

```text
continue
```

**Assistant interpretation:** Turn the accepted audit findings into a detailed implementation guide, organize follow-up tasks into phases, upload the refreshed ticket bundle, and begin implementing the first safe remediation slice.

**Inferred user intent:** The user wants movement from analysis to execution while keeping the docmgr ticket detailed and useful for interns.

### What I did

- Added `design-doc/02-remediation-plan-for-replay-store-and-api-cleanup.md`.
- Rewrote `tasks.md` into phased tasks.
- Renamed runtime `evtstream` constants/SQLite tables outside `ttmp` to `sessionstream` names.
- Removed the unused generic `pkg/sessionstream/transport/transport.go` command-ingress abstraction.
- Removed `Command.ConnectionId` while websocket command ingress is explicitly out of scope.
- Updated websocket comments to describe the adapter as fanout/subscription-only.
- Changed local ordinal assignment to seed from `HydrationStore.Cursor`.
- Made fail-closed projection policy the default.
- Added split UI and timeline projection policy options.
- Added `ErrorObserver`, `ErrorObserverFunc`, `ErrorRecord`, and error kinds for projection/fanout errors.
- Changed `SchemaRegistry` to clone prototypes on register and lookup.
- Added tests for local ordinal seeding, split policy behavior, error observer records, and schema cloning.
- Ran `go test ./...` and `make check`.
- Verified no `evtstream` string remains outside `ttmp` and `dist`.

### Why

These changes are prerequisites for the larger replay-store work. They remove stale names, clarify scope, and make current behavior safer before adding event-log tables, projection cursors, and durable error records.

### What worked

- The existing tests were easy to update around the new fail-default behavior.
- Local ordinal seeding could be implemented without the full replay store by using the existing `HydrationStore.Cursor` method.
- Schema cloning was a small, isolated hardening change.
- `go test ./...` passed.
- `make check` passed.

### What didn't work

The combined command `make check && rg -n 'evtstream' ...` exited with status 1 because `rg` found no matches. That is expected for `rg`; the absence of matches is the desired result. I reran the search with `|| true` to confirm there was no output.

### What I learned

A useful implementation order is to first remove misleading surface area and make existing paths fail loudly. The full replay store can then be introduced against a cleaner API surface.

### What was tricky to build

The projection policy change required preserving existing opt-in advance behavior while changing the default to fail. I kept `WithProjectionErrorPolicy` as a convenience that sets both UI and timeline policies, and added `WithUIProjectionErrorPolicy` / `WithTimelineProjectionErrorPolicy` for the new split behavior.

### What warrants a second pair of eyes

- Whether `Command.ConnectionId` removal is acceptable now, or whether it should wait until a larger public API cleanup commit.
- Whether `ConnectionId` should remain in the core package or move under websocket/fanout transport later.
- Whether keeping the convenience `WithProjectionErrorPolicy` conflicts with the no-backwards-compatibility cleanup direction.

### What should be done in the future

- Implement the actual replay-capable SQLite store.
- Add durable error/DLQ records.
- Add generated protobuf chat example.
- Refactor systemlab shared trace/runtime helpers.

### Code review instructions

Start with:

- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/hub_test.go`
- `pkg/sessionstream/schema.go`
- `pkg/sessionstream/schema_test.go`
- `pkg/sessionstream/bus.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/types.go`

Validate with:

```bash
cd sessionstream
go test ./...
make check
rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w pkg/sessionstream/bus.go pkg/sessionstream/hydration/sqlite/store.go pkg/sessionstream/schema.go pkg/sessionstream/types.go pkg/sessionstream/transport/ws/server.go pkg/sessionstream/fanout.go pkg/sessionstream/hub.go pkg/sessionstream/hub_test.go pkg/sessionstream/schema_test.go pkg/sessionstream/bus_test.go
go test ./...
make check
rg -n 'evtstream' . --glob '!ttmp/**' --glob '!dist/**'
```

## Step 6: Added the first replay-store primitives to SQLite

I continued the remediation by adding concrete replay-store primitives to the SQLite backend. This does not finish the full replay architecture yet, but it establishes the core durable structures: append-only backend events, event cursor lookup, event queries, historical timeline entity versions, `Snapshot(asOf)` support, an in-memory SQLite constructor, and a durable errors table.

The hub now appends events before projection when the configured store implements `EventStore`, and local/bus ordinal assignment prefers the durable event cursor when available. This is the important transition from "current-state-only hydration" toward "event log plus projection state."

### Prompt Context

**User prompt (verbatim):**

```text
ok, continue
```

**Assistant interpretation:** Continue executing the remediation plan from the previous turn.

**Inferred user intent:** Keep making implementation progress, especially on the replay store direction.

### What I did

- Added `EventStore` and `ErrorStore` interfaces in `pkg/sessionstream/hydration.go`.
- Added `Hub.EventCursor` and internal `eventCursor` fallback logic.
- Changed local and bus ordinal cursor sources to prefer `EventStore.EventCursor` when available.
- Changed `projectAndApply` to append events before projection when the store implements `EventStore`.
- Changed `reportError` to persist error records when the store implements `ErrorStore`.
- Added SQLite tables:
  - `sessionstream_events`
  - `sessionstream_entity_versions`
  - `sessionstream_errors`
- Added SQLite methods:
  - `AppendEvent`
  - `Events`
  - `EventCursor`
  - `RecordError`
  - `NewInMemory`
  - `MemoryDSN`
- Changed SQLite `Apply` to record entity versions in addition to current state.
- Changed SQLite `Snapshot(ctx, sid, asOf)` to use version history when `asOf > 0`.
- Added SQLite tests for event append/query, historical snapshots, tombstones, reset, and in-memory SQLite.
- Added hub test proving local ordinals prefer the event cursor when an event store exists.

### Why

The previous API exposed replay-shaped concepts without durable replay storage. These changes start making replay real while keeping existing local users working through the current `HydrationStore` path.

### What worked

- Existing store behavior remained intact for `Snapshot(..., 0)`.
- The new `Snapshot(asOf)` behavior can reconstruct older entity state from `sessionstream_entity_versions`.
- Local ordinal assignment now uses event cursor when possible, which is the correct cursor once events are appended before projection.
- `go test ./...` passed.

### What didn't work

Nothing failed in the final validation. I limited this slice to SQLite and hub integration rather than attempting projection cursors and replay/rebuild helpers at the same time.

### What I learned

The clean incremental path is to make the SQLite backend support replay semantics first, then introduce projection cursors and rebuild commands. Trying to design every replay API at once would make the diff much harder to review.

### What was tricky to build

`Snapshot(asOf)` needed separate behavior for current state and historical state. Current snapshots can read `sessionstream_entities`; historical snapshots need the latest entity version at or before the requested ordinal and must skip tombstones.

The query shape is:

```sql
SELECT v.kind, v.entity_id, v.tombstone, v.payload_json
FROM sessionstream_entity_versions v
JOIN (
  SELECT kind, entity_id, MAX(ordinal) AS ordinal
  FROM sessionstream_entity_versions
  WHERE session_id = ? AND ordinal <= ?
  GROUP BY kind, entity_id
) latest
ON latest.kind = v.kind
AND latest.entity_id = v.entity_id
AND latest.ordinal = v.ordinal
WHERE v.session_id = ?
ORDER BY v.kind ASC, v.entity_id ASC
```

### What warrants a second pair of eyes

- Whether `AppendEvent` should be idempotent with update-on-conflict, error-on-conflict, or compare-and-error semantics.
- Whether `RecordError` needs richer fields immediately, such as raw payload, command name, retryability, and metadata.
- Whether `NewInMemory` should use unique names by default to avoid accidental sharing between stores in one process.

### What should be done in the future

- Add projection cursor table and API.
- Add replay/rebuild helper that reprojects stored events.
- Add decode and ordinal errors to durable error reporting.
- Decide whether to remove the map-backed memory store or keep it as a tiny non-replay test store.

### Code review instructions

Start with:

- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/consumer.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`
- `pkg/sessionstream/hub_test.go`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w pkg/sessionstream/hydration.go pkg/sessionstream/hub.go pkg/sessionstream/consumer.go pkg/sessionstream/hydration/sqlite/store.go pkg/sessionstream/hydration/sqlite/store_test.go pkg/sessionstream/hub_test.go
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

## Step 7: Added projection cursors and an initial timeline rebuild helper

I added the next replay-store slice: projection cursor tracking and an initial timeline rebuild path. This lets the store distinguish the backend event cursor from the timeline projection cursor. That distinction is central to the fail-closed replay model: an event can exist durably while the timeline projector remains behind until a rebuild/retry succeeds.

The new `Hub.RebuildTimeline` helper replays stored backend events through the timeline projection only. It intentionally does not call the UI projection and does not publish to fanout, so rebuilds repair materialized timeline state without re-notifying live clients.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue implementing the remediation plan, focusing on the next open replay-store pieces.

**Inferred user intent:** Keep executing the phased plan rather than stopping at the initial event-log primitives.

### What I did

- Added `ProjectionCursorStore` to `pkg/sessionstream/hydration.go`.
- Added `TimelineProjectorName` as the default timeline projector cursor key.
- Added `Hub.ProjectionCursor`.
- Added projection cursor advancement after successful timeline application in the normal event path.
- Added `Hub.RebuildTimeline(ctx, sid, from)`.
- Added internal `rebuildTimelineEvent` helper.
- Added SQLite table `sessionstream_projection_cursors`.
- Added SQLite methods:
  - `ProjectionCursor`
  - `AdvanceProjectionCursor`
- Updated reset to clear projection cursors.
- Added tests for monotonic projection cursor advancement.
- Added hub tests for normal cursor advancement and rebuild-without-fanout behavior.

### Why

A replayable store needs separate progress markers. The event cursor answers "what is the last event we have?" The timeline projection cursor answers "what is the last event the timeline projector successfully materialized?" Conflating the two would recreate the original bug where cursor advancement could hide missing projection state.

### What worked

- Projection cursor advancement could be added incrementally without changing the existing `HydrationStore.Cursor` API.
- `Hub.RebuildTimeline` can operate on the new `EventStore` interface and existing timeline projection interface.
- Tests passed for the hub package and SQLite store.
- Full `go test ./...` passed.

### What didn't work

No validation failure occurred in this step. The implementation deliberately avoids a full operator-facing retry API for now; it only adds the lower-level rebuild helper.

### What I learned

The clean rebuild behavior is timeline-only by default. If UI events are replayed during rebuild, clients may see duplicate live events. Keeping rebuild fanout-free makes it safe as a repair operation.

### What was tricky to build

The test store had to implement both `EventStore` and `ProjectionCursorStore` to exercise the hub's cursor behavior. I kept the test implementation simple by storing the projection cursor in the same snapshot ordinal field, which is fine for tests but not the production model.

### What warrants a second pair of eyes

- Whether `TimelineProjectorName = "timeline"` should be exported as a constant or kept internal.
- Whether `Hub.RebuildTimeline` should start from `ProjectionCursor(ctx, "timeline", sid)` by default when `from == 0`, or whether callers should always pass the exact starting point.
- Whether rebuild should clear current timeline state before replaying from zero. The current implementation applies events over existing state.

### What should be done in the future

- Add a higher-level retry helper that starts from the stored projection cursor.
- Add an option to clear/rebuild timeline state from scratch if replaying from zero.
- Record decode and ordinal errors into durable error storage.
- Add operator-facing docs for replay/rebuild safety.

### Code review instructions

Start with:

- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`
- `pkg/sessionstream/hub_test.go`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w pkg/sessionstream/hydration.go pkg/sessionstream/hub.go pkg/sessionstream/hydration/sqlite/store.go pkg/sessionstream/hydration/sqlite/store_test.go pkg/sessionstream/hub_test.go
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

## Step 8: Added retry helpers, scratch rebuild, and decode/ordinal error reporting

I implemented the three next remediation items: a retry helper that resumes from the stored timeline projection cursor, durable decode/ordinal error reporting in the bus consumer path, and a scratch rebuild path for stores that can clear materialized timeline state before replaying events.

This completes the first practical loop for replay repair: events can be stored, projection cursors track materialization progress, decode/ordinal errors are reported, failed timelines can be retried from the projection cursor, and a full materialized timeline can be cleared and rebuilt from the event log.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue implementing the next open remediation tasks: retry helper, durable decode/ordinal error reporting, and clear/rebuild support.

**Inferred user intent:** The user wants the replay-store repair path to become usable, not just partially modeled in storage.

### What I did

- Added `TimelineResetStore` to `pkg/sessionstream/hydration.go`.
- Added `ErrorKindDecode` and `ErrorKindOrdinal`.
- Extended `ErrorRecord` with raw message bytes and metadata.
- Added `Hub.RetryTimeline(ctx, sid)`.
- Added `Hub.RebuildTimelineFromScratch(ctx, sid)`.
- Changed `eventConsumer.handleMessage` to record decode errors before acking malformed messages.
- Changed `eventConsumer.handleMessage` to record ordinal errors before returning the error for nack/stop behavior.
- Added SQLite `ClearTimeline(ctx, sid)`.
- Added tests for:
  - bus decode error recording;
  - bus ordinal error recording;
  - retry from projection cursor;
  - scratch rebuild clearing stale timeline state;
  - SQLite clear timeline preserving event log but clearing materialized state and projection cursor.

### Why

Projection cursors are only useful if callers have a simple way to resume from them. `RetryTimeline` gives that path. Scratch rebuild is a separate operation for cases where materialized state may be corrupt and should be rebuilt from the event log from zero.

Decode and ordinal failures are different from projection failures because they happen before `projectAndApply`. They needed explicit reporting in the consumer rather than relying on projection error handling.

### What worked

- The retry helper could be implemented as a small wrapper around `ProjectionCursor` and `RebuildTimeline`.
- Scratch rebuild could be implemented generically via a `TimelineResetStore` interface.
- The bus consumer tests can call `handleMessage` directly because the tests live in the same package.
- Full `go test ./...` passed.

### What didn't work

No validation failure occurred in this step.

### What I learned

It is useful to keep three operations distinct:

1. `RebuildTimeline(ctx, sid, from)` — low-level replay from a caller-provided ordinal.
2. `RetryTimeline(ctx, sid)` — repair from the stored projection cursor.
3. `RebuildTimelineFromScratch(ctx, sid)` — clear materialized timeline state and replay from zero.

They serve different operator needs and should not be overloaded into one ambiguous method.

### What was tricky to build

`ClearTimeline` must clear current entities, historical entity versions, and the timeline projection cursor, but it must not delete the event log. If it deleted events, scratch rebuild would have nothing to replay. The SQLite test now asserts this behavior.

### What warrants a second pair of eyes

- Whether malformed decode errors should continue to be acked after recording, or whether some deployments should nack/dead-letter externally.
- Whether ordinal errors should stop the consumer, as they do now, or be configurable.
- Whether `ErrorRecord.RawMessage` should be persisted in SQLite immediately; the current SQLite `RecordError` stores metadata and error text but not raw payload bytes yet.

### What should be done in the future

- Add operator-facing command/API around `RetryTimeline` and `RebuildTimelineFromScratch`.
- Persist raw error payloads in SQLite if needed for debugging malformed bus messages.
- Add documentation describing when to use retry versus scratch rebuild.

### Code review instructions

Start with:

- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/consumer.go`
- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hub_test.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w pkg/sessionstream/hydration.go pkg/sessionstream/hub.go pkg/sessionstream/consumer.go pkg/sessionstream/hydration/sqlite/store.go pkg/sessionstream/hydration/sqlite/store_test.go pkg/sessionstream/hub_test.go
go test ./pkg/sessionstream ./pkg/sessionstream/hydration/sqlite
go test ./...
```

## Step 9: Persisted raw error payloads and added error record queries

I continued with the error/DLQ part of the remediation plan by making SQLite error records more useful for operator tooling. Error records now persist raw message bytes and metadata JSON, and the store exposes a query method for reading persisted errors back.

This matters most for decode errors: if a Watermill payload cannot be decoded, the framework can now keep the raw payload bytes and metadata needed for debugging instead of only keeping the error string.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue the remediation work, focusing on the next open implementation gap.

**Inferred user intent:** Keep making the replay/error reporting path operational and testable.

### What I did

- Added `ErrorRecordStore` to `pkg/sessionstream/hydration.go`.
- Added `raw_message BLOB` to the SQLite `sessionstream_errors` table.
- Changed SQLite `RecordError` to persist:
  - kind;
  - session id;
  - ordinal;
  - event name;
  - error string;
  - raw message bytes;
  - metadata JSON.
- Added SQLite `ErrorRecords(ctx, sid, limit)` query API.
- Added a SQLite test proving error records round-trip raw payload bytes and metadata.

### Why

A durable error table is only useful if operators can inspect enough context to understand the failure. Decode failures often happen before there is a valid event name or session id, so raw bytes and transport metadata are critical.

### What worked

- The new error query API is small and does not affect the existing hub flow.
- Full test suite passed after adding the new SQLite persistence fields.

### What didn't work

Nothing failed in validation.

### What I learned

The next useful step is probably a configurable decode policy. Right now decode errors are recorded and acked, matching the previous behavior's ack semantics but making it observable. Some deployments may want nack/dead-letter behavior instead.

### What was tricky to build

SQLite rows use nullable columns for session id, ordinal, and event name because decode errors may not have those fields. The query code has to preserve that optionality when rebuilding `ErrorRecord` values.

### What warrants a second pair of eyes

- Whether `ErrorRecord.Err` should remain an `error` in persisted query results or whether persisted records should have a separate string field.
- Whether `ErrorRecords(ctx, sid, limit)` should support filtering by kind and time.

### What should be done in the future

- Add configurable bus decode ack/nack/dead-letter behavior.
- Add operator-facing commands/API for listing error records.

### Code review instructions

Start with:

- `pkg/sessionstream/hydration.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream/hydration/sqlite
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w pkg/sessionstream/hydration.go pkg/sessionstream/hydration/sqlite/store.go pkg/sessionstream/hydration/sqlite/store_test.go
go test ./pkg/sessionstream/hydration/sqlite
go test ./...
```

## Step 10: Removed the map-backed memory store

I removed the old map-backed `hydration/memory` package and ported local users to the SQLite store's in-memory mode. This gives the framework one store implementation path for local examples, websocket tests, systemlab phases, and persistent SQL mode.

The important semantic change is that "memory mode" now means "SQLite database running with a named in-memory DSN," not a separate Go map implementation. That reduces drift between local and persistent behavior.

### Prompt Context

**User prompt (verbatim):**

```text
3. remove map backed store,
```

**Assistant interpretation:** Remove the separate map-backed memory hydration store and migrate users to the in-memory SQLite backend.

**Inferred user intent:** Consolidate storage semantics around SQLite so replay/event/projection behavior is not split across two implementations.

### What I did

- Deleted `pkg/sessionstream/hydration/memory`.
- Updated systemlab Phase 1, 2, 3, 4, and memory-mode Phase 5 to use `storesqlite.NewInMemory(reg)`.
- Updated chat demo tests to use in-memory SQLite.
- Updated websocket transport tests to use in-memory SQLite.
- Updated systemlab chapter prose/file references that pointed at `hydration/memory`.
- Verified there are no remaining `hydration/memory`, `memory.New`, `storememory`, or map-backed memory references outside `ttmp`/`dist`.
- Ran the full Go test suite.

### Why

The replay-store work is now SQLite-first. Keeping a separate map-backed store would require maintaining event logs, entity versions, projection cursors, and error storage twice. Removing it early keeps local tests and examples exercising the same semantics as the durable backend.

### What worked

- The in-memory SQLite constructor was already available from the prior replay-store slice.
- Tests and systemlab code only needed local construction changes.
- Full `go test ./...` passed.

### What didn't work

No validation failure occurred. The main mechanical issue was that `NewInMemory(reg)` returns `(*Store, error)`, so call sites that previously used `memory.New()` had to move store construction after registry setup and handle errors.

### What I learned

Systemlab docs were the most likely place to leave stale references. Searching prose files as well as Go files is necessary when removing a package.

### What was tricky to build

Some phase setup functions previously created the store before the schema registry. In-memory SQLite requires the registry so it can decode event and entity payloads, so those functions now create/register schemas first and then construct the store.

### What warrants a second pair of eyes

- Whether Phase 5 should rename user-facing `mode=memory` to `mode=ephemeral-sql` eventually. For now, keeping `memory` as an ergonomic label is probably fine.
- Whether the core package should offer a top-level helper for local in-memory store setup, or whether keeping it in `hydration/sqlite` is cleaner.

### What should be done in the future

- Watch CI for any hidden CGO/SQLite environment assumptions now that all local tests use SQLite.
- Continue with generated protobuf chat example or systemlab refactor.

### Code review instructions

Start with:

- deleted `pkg/sessionstream/hydration/memory`
- `cmd/sessionstream-systemlab/lab_environment.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `examples/chatdemo/chat_test.go`
- `pkg/sessionstream/transport/ws/server_test.go`

Validate with:

```bash
cd sessionstream
rg -n 'hydration/memory|memory.New|memory store|map-backed|map backed|storememory' . --glob '!ttmp/**' --glob '!dist/**'
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
rm -rf pkg/sessionstream/hydration/memory
gofmt -w cmd/sessionstream-systemlab/lab_environment.go cmd/sessionstream-systemlab/phase2_lab.go cmd/sessionstream-systemlab/phase3_lab.go cmd/sessionstream-systemlab/phase4_lab.go cmd/sessionstream-systemlab/phase5_lab.go examples/chatdemo/chat_test.go pkg/sessionstream/transport/ws/server_test.go
rg -n 'hydration/memory|memory.New|memory store|map-backed|map backed|storememory' . --glob '!ttmp/**' --glob '!dist/**'
go test ./...
```

## Step 11: Added a self-contained protobuf chat example

I continued with the schema-first example work by adding a namespaced protobuf contract for `examples/chatdemo` and migrating the chat demo away from `structpb.Struct` payloads. The example now demonstrates the intended pattern: register generated protobuf command, event, UI event, and timeline entity message types in `SchemaRegistry`, then pass those generated messages through the hub.

This keeps the demo self-contained under `examples/chatdemo` while making it much clearer how downstream users should model their own payloads.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue executing the remediation plan, moving to the next open item after store consolidation.

**Inferred user intent:** Keep reducing the remaining open remediation tasks and make the examples match the new architecture direction.

### What I did

- Added `examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto`.
- Generated Go protobuf code under `examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat.pb.go`.
- Added `examples/chatdemo/generate.go` with a `go:generate` command for the proto.
- Updated `examples/chatdemo/chat.go` to use generated protobuf messages for:
  - commands;
  - backend events;
  - UI events;
  - timeline entities.
- Updated `examples/chatdemo/chat_test.go` to submit generated command messages and assert generated `ChatMessageEntity` payloads.
- Updated the systemlab `protoStructMap` helper so Phase 4 can still render generated protobuf UI payloads in trace details.
- Ran chat demo, systemlab, and full repository tests.

### Why

The audit called out that using only `structpb.Struct` made the framework look stringly typed despite having a protobuf schema registry. This change gives new users a concrete schema-first example without forcing generated protos into every systemlab phase.

### What worked

- `protoc` and `protoc-gen-go` were already available.
- The existing `SchemaRegistry` and payload validation worked with generated messages.
- The SQLite store could decode generated timeline entities from proto JSON via the registered schema.
- Full `go test ./...` passed.

### What didn't work

No final validation failure occurred. One important compatibility adjustment was needed: systemlab Phase 4 had been using a helper named `protoStructMap`, which only understood `structpb.Struct`. I extended it to marshal arbitrary protobuf messages to JSON maps.

### What I learned

A self-contained generated-protobuf example is much clearer than explaining schema-first usage abstractly. The chat demo now shows the complete loop from typed command to typed event to typed UI update and typed timeline entity.

### What was tricky to build

The chat demo still exposes trace hooks as `map[string]any` because systemlab wants generic JSON-ish trace details. I kept hooks map-based but made the actual framework payloads generated protobuf messages. That preserves teaching ergonomics without weakening the framework example.

### What warrants a second pair of eyes

- Whether the generated code path should be shortened. The current path mirrors the protobuf package namespace and is explicit but long.
- Whether `go generate ./examples/chatdemo` should be added to CI or documented more prominently.
- Whether additional systemlab phases should eventually migrate from `structpb.Struct` to generated protos.

### What should be done in the future

- Consider adding a short README section explaining the generated chat proto.
- Decide whether descriptor-oriented schema APIs are still needed after this example.
- Continue with systemlab refactor or operator-facing retry/error APIs.

### Code review instructions

Start with:

- `examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto`
- `examples/chatdemo/generate.go`
- `examples/chatdemo/chat.go`
- `examples/chatdemo/chat_test.go`
- `cmd/sessionstream-systemlab/phase2_lab.go` (`protoStructMap` helper)

Validate with:

```bash
cd sessionstream
go generate ./examples/chatdemo
go test ./examples/chatdemo ./cmd/sessionstream-systemlab
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
mkdir -p examples/chatdemo/gen
protoc -I examples/chatdemo/proto --go_out=examples/chatdemo/gen --go_opt=paths=source_relative examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto
gofmt -w examples/chatdemo/chat.go examples/chatdemo/chat_test.go examples/chatdemo/generate.go examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat.pb.go cmd/sessionstream-systemlab/phase2_lab.go
go test ./examples/chatdemo ./cmd/sessionstream-systemlab
go test ./...
```

## Step 12: Added systemlab replay inspection for cursors and errors

I added a systemlab-facing replay inspection endpoint for Phase 5. This is intentionally read-only: it exposes event cursor, timeline projection cursor, and persisted error records, but it does not expose retry or rebuild operations through HTTP.

This matches the revised direction: errors and cursors are useful for systemlab visualization and teaching, while retry endpoints are not needed right now.

### Prompt Context

**User prompt (verbatim):**

```text
no retry endpoints, let's do the systemlab refactor.
```

Follow-up prompt:

```text
exposing errors and cursors is good for system lab, add tasks
```

**Assistant interpretation:** Do not add retry/rebuild endpoints. Add systemlab visibility into replay cursors and persisted errors, update the task plan, and keep refactor work oriented around systemlab readability/teaching.

**Inferred user intent:** The user wants systemlab to demonstrate the replay/error model without becoming an operator control plane.

### What I did

- Added `cmd/sessionstream-systemlab/replay_api.go`.
- Added `GET /api/phase5/replay?sessionId=...&limit=...`.
- Added `replayInspectResponse` with:
  - phase;
  - session id;
  - event cursor;
  - timeline cursor;
  - persisted errors.
- Added encoding for persisted error records, including base64 raw payloads.
- Added `labEnvironment.inspectPhase5Replay`.
- Added a systemlab test for replay inspection after a Phase 5 seed operation.
- Added `trace_helpers.go` as an initial shared helper file for the systemlab refactor.
- Updated `tasks.md` to mark replay inspection done and add a remaining UI visualization task.

### Why

Systemlab should teach the difference between event cursor and projection cursor. Exposing those values read-only lets the frontend explain replay state without introducing dangerous mutating admin endpoints.

### What worked

- Phase 5 already has a runtime store and hub, so read-only inspection was straightforward.
- The existing SQLite error query API provided the persisted error records.
- Systemlab tests passed.
- Full repository tests passed.

### What didn't work

The first build failed because I added a helper named `cloneStringMap` in `replay_api.go`, but `phase2_lab.go` already had a function with that name in package `main`.

Exact error:

```text
cmd/sessionstream-systemlab/replay_api.go:118:6: cloneStringMap redeclared in this block
	cmd/sessionstream-systemlab/phase2_lab.go:674:6: other declaration of cloneStringMap
```

Fix: renamed the helper in `replay_api.go` to `cloneReplayStringMap`.

### What I learned

Because all systemlab files are in package `main`, small helper names collide across files. The systemlab refactor should either centralize helpers or use more specific names until common helpers are consolidated.

### What was tricky to build

The response needs to expose raw error payloads safely as JSON. I encoded raw bytes as base64 so arbitrary malformed payloads can be represented without corrupting the response.

### What warrants a second pair of eyes

- Whether Phase 5 is the right endpoint namespace for replay inspection, or whether this should become a cross-phase `/api/replay` endpoint later.
- Whether the frontend should display raw error payloads by default or hide them behind an expand/copy action.

### What should be done in the future

- Add frontend UI for `/api/phase5/replay`.
- Add a small explanation in the Phase 5 chapter for event cursor vs timeline cursor.
- Continue extracting shared trace/clone/snapshot helpers.

### Code review instructions

Start with:

- `cmd/sessionstream-systemlab/replay_api.go`
- `cmd/sessionstream-systemlab/server.go`
- `cmd/sessionstream-systemlab/lab_environment_test.go`
- `cmd/sessionstream-systemlab/trace_helpers.go`

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab
go test ./...
```

### Technical details

Commands:

```bash
cd sessionstream
gofmt -w cmd/sessionstream-systemlab/replay_api.go cmd/sessionstream-systemlab/server.go cmd/sessionstream-systemlab/lab_environment_test.go cmd/sessionstream-systemlab/trace_helpers.go
go test ./cmd/sessionstream-systemlab
go test ./...
```

## Step 13: Added Phase 5 replay cursor/error UI

I added a small read-only replay inspection panel to the Phase 5 systemlab UI. It displays the event cursor, timeline projection cursor, and persisted error records for the currently selected session. The panel refreshes after Phase 5 actions and can also be refreshed manually.

This makes the replay model visible in the teaching app without adding retry or rebuild endpoints.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue the systemlab refactor/visibility work after adding the read-only replay endpoint.

**Inferred user intent:** Show the newly exposed cursor/error data in systemlab, not just via raw API calls.

### What I did

- Added `fetchPhase5Replay(sessionId, limit)` to `static/js/api.js`.
- Added a `Replay Cursors and Errors` panel to `static/partials/phase5.html`.
- Added a `Refresh Replay State` button.
- Updated `static/js/pages/phase5.js` to fetch and render replay inspection data.
- Refreshed replay state after Phase 5 run/refresh actions.
- Marked the systemlab replay visualization task complete.

### Why

The replay/error model is easier to understand when users can see event cursor and timeline cursor side by side. Phase 5 is the natural place for this because it already teaches persistence and restart correctness.

### What worked

- The existing `setJSON` helper was enough for a first-pass visualization.
- No backend changes were needed beyond the read-only endpoint from Step 12.
- Go tests still passed.

### What didn't work

No test/build failure occurred in this step. There is no browser automation test for the new panel yet.

### What I learned

For systemlab, a raw JSON panel is often enough for the first teaching iteration. A richer UI can come after the backend contracts settle.

### What was tricky to build

The page refresh flow needed to avoid introducing extra control-plane behavior. The frontend only fetches read-only inspection data; it does not call retry/rebuild operations.

### What warrants a second pair of eyes

- Whether the replay panel should be visible only in SQL mode or also in memory/ephemeral SQLite mode. It is currently visible for both because both use SQLite.
- Whether raw error payloads should be decoded from base64 in the UI later.

### What should be done in the future

- Add prose to the Phase 5 chapter explaining event cursor vs timeline cursor.
- Consider a nicer visual presentation than raw JSON after the systemlab refactor settles.

### Code review instructions

Start with:

- `cmd/sessionstream-systemlab/static/partials/phase5.html`
- `cmd/sessionstream-systemlab/static/js/pages/phase5.js`
- `cmd/sessionstream-systemlab/static/js/api.js`

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab
go test ./...
```

## Step 14: Extracted shared trace helpers and documented replay cursors in Phase 5

I continued the systemlab refactor by putting the first shared trace helpers to use across phases and adding Phase 5 chapter prose for the replay inspection panel. This is a small but useful cleanup: trace append and trace clone behavior now has one implementation instead of being copied in each phase file.

I also updated the Phase 5 educational chapter to explain event cursor vs timeline cursor, including the healthy case and the failed-projection gap case.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue the systemlab refactor and teaching documentation work after adding replay cursor/error visibility.

**Inferred user intent:** Keep making systemlab clearer and less duplicated while preserving the teaching flow.

### What I did

- Added and used shared helpers in `trace_helpers.go`:
  - `appendTraceEntry`
  - `cloneTraceEntries`
  - `cloneNamedPayloads`
- Replaced duplicated trace append logic in:
  - Phase 1
  - Phase 2
  - Phase 3
  - Phase 4
  - Phase 5
- Replaced duplicated trace clone loops in phase response clone helpers.
- Updated `phase-5-persistence-and-restart.md` with a new replay inspection section explaining:
  - event cursor;
  - timeline cursor;
  - healthy matching cursor state;
  - failed projection cursor gap state;
  - persisted errors as explanation for cursor gaps.
- Updated tasks to mark initial trace helper extraction and Phase 5 cursor prose complete.

### Why

Systemlab had repeated trace bookkeeping in every phase. That repetition makes the teaching code longer and increases the chance of subtle divergence. Extracting the common trace mechanics keeps phase files focused on the lesson-specific behavior.

### What worked

- The helper extraction was mechanical and low risk.
- Systemlab tests passed after the refactor.

### What didn't work

No validation failure occurred.

### What I learned

A small helper extraction is a good systemlab refactor unit. It improves readability without forcing a large directory split before the code is ready.

### What was tricky to build

Phase 1 stores traces in a map keyed by session id, while later phases store traces directly on phase state. The helper therefore works on a `*[]traceEntry`, which supports both shapes.

### What warrants a second pair of eyes

- Whether `trace_helpers.go` should later grow into a `traceLog` struct with locking built in.
- Whether response clone helpers should be consolidated further.

### What should be done in the future

- Extract shared snapshot encoding helpers.
- Extract shared websocket hook builders.
- Consider splitting Phase 2 and Phase 5 after the smaller helper extractions land.

### Code review instructions

Start with:

- `cmd/sessionstream-systemlab/trace_helpers.go`
- `cmd/sessionstream-systemlab/lab_environment.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `cmd/sessionstream-systemlab/chapters/phase-5-persistence-and-restart.md`

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab
go test ./...
```

## Step 15: Extracted snapshot encoding and websocket trace hook helpers

I continued the Systemlab refactor by extracting the duplicated snapshot/payload encoding and websocket trace hook wiring into shared helpers. This keeps the phase files focused on their lesson-specific behavior while preserving the same trace messages and JSON response shapes.

This step also creates the first code commit for the remediation implementation, so the ticket now has a concrete review checkpoint: `aaac81d34cadad21820f68b7d335db701c0fc8b8`.

### Prompt Context

**User prompt (verbatim):**

```text
Next steps: Systemlab still has duplicated snapshot encoding and websocket hook setup.

Extract shared snapshot encoding helpers or websocket hook builders.

Check tasks, keep a detailed diary (see skills), commit at appropriate intervals.
```

**Assistant interpretation:** Continue Phase 6 by checking the task list, extracting shared Systemlab helpers for snapshots and websocket hooks, keeping a detailed diary, and committing the implementation at a sensible checkpoint.

**Inferred user intent:** Reduce Systemlab duplication in a reviewable way while keeping ticket bookkeeping and git history current.

**Commit (code):** `aaac81d34cadad21820f68b7d335db701c0fc8b8` — "Implement replay store remediation"

### What I did

- Checked `tasks.md` and confirmed the remaining Phase 6 helper extraction items.
- Added `cmd/sessionstream-systemlab/snapshot_helpers.go` with shared helpers:
  - `encodeSnapshot`
  - `currentEntityMap`
  - `currentEntityMapForKind`
  - `protoStructMap`
- Moved protobuf payload encoding out of `phase2_lab.go` so snapshot encoding, timeline entity lookup, UI trace payloads, and generated protobuf payloads share the same conversion path.
- Added `cmd/sessionstream-systemlab/ws_hooks.go` with `newWebsocketTraceHooks` and `websocketTraceOptions`.
- Replaced duplicated websocket hook setup in phases 3, 4, and 5.
- Kept Phase 3 client-frame tracing configurable because Phase 3 teaches websocket frame handling and phases 4/5 do not currently need that extra trace hook.
- Kept Phase 4/5 UI payload tracing configurable because those phases use UI payload fields in trace checks and teaching output.
- Ran formatting and tests.
- Committed the accumulated remediation implementation in commit `aaac81d34cadad21820f68b7d335db701c0fc8b8`.
- Marked the snapshot helper and websocket hook builder tasks complete.

### Why

Systemlab had repeated low-level mechanics in every phase. Snapshot encoding and websocket hook wiring are infrastructure concerns, not lesson-specific logic. Pulling them into helpers makes Phase 3/4/5 easier to compare and reduces future drift when transport trace semantics change.

### What worked

- `go test ./cmd/sessionstream-systemlab` passed after extracting both helpers.
- The websocket hook helper preserved the original trace message strings, which kept the existing check helpers valid.
- Moving `protoStructMap` to the shared snapshot helper file worked cleanly because it was already package-level and used by several phases.

### What didn't work

No compile or test failure occurred during this extraction. The main risk was import churn and preserving exact trace message strings; `gofmt` and the package test caught the obvious issues.

### What I learned

A small options struct is enough for the websocket helper right now. It avoids over-abstracting phase-specific behavior while still removing the duplicated connect/disconnect/subscribe/snapshot/live-event tracing boilerplate.

### What was tricky to build

Phase 3, Phase 4, and Phase 5 have nearly identical websocket hooks, but not identical teaching needs. Phase 3 records client frames; Phase 4 and Phase 5 include UI payload details in live event traces. The helper therefore takes explicit booleans instead of assuming all phases need the same trace detail.

Snapshot encoding had a second wrinkle: systemlab now uses both `structpb.Struct` and generated protobuf messages. The shared `protoStructMap` helper keeps the optimized `structpb.AsMap()` path and falls back to `protojson` for arbitrary protobuf messages.

### What warrants a second pair of eyes

- Confirm that the websocket trace helper has not hidden too much phase-specific teaching behavior.
- Confirm that `protojson` field-name settings match the desired frontend JSON shape for generated protobuf payloads.
- Review whether the helper should copy the client frame map before adding `connectionId`; it currently preserves the previous behavior of mutating the trace-local frame map.

### What should be done in the future

- Split Phase 2 and Phase 5 into smaller runtime/actions/projections/checks/render files after the helper extraction settles.
- Consider adding a focused browser smoke test for the Phase 5 replay panel.
- Consider a richer trace abstraction only if another repeated pattern emerges.

### Code review instructions

Start with:

- `cmd/sessionstream-systemlab/snapshot_helpers.go`
- `cmd/sessionstream-systemlab/ws_hooks.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `cmd/sessionstream-systemlab/phase2_lab.go`

Then review the broader commit:

```bash
git show --stat aaac81d34cadad21820f68b7d335db701c0fc8b8
git show aaac81d34cadad21820f68b7d335db701c0fc8b8 -- cmd/sessionstream-systemlab/snapshot_helpers.go cmd/sessionstream-systemlab/ws_hooks.go
```

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab
go test ./...
make check
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
```

### Technical details

The websocket helper intentionally preserves these trace message formats:

- `phase N websocket connected`
- `phase N websocket disconnected`
- `phase N subscribed`
- `phase N unsubscribed`
- `phase N snapshot sent`
- `phase N ui event sent`
- optional: `phase N client frame received`

The snapshot helper encodes snapshots as:

```json
{
  "sessionId": "...",
  "ordinal": 1,
  "entities": [
    { "kind": "...", "id": "...", "payload": {} }
  ]
}
```

## Step 16: Documented websocket fanout-only boundaries and tested unsupported command frames

I continued the cleanup by closing out the websocket fanout-only semantics tasks. The adapter now documents that `sinceOrdinal` is advisory, that subscribe sends a current snapshot plus future live UI events, and that event replay is deliberately not hidden inside websocket subscribe.

I also added a transport test proving websocket command frames are rejected as unsupported. This keeps the command ingress boundary explicit while preserving the reference adapter's role as a snapshot/fanout transport.

### Prompt Context

**User prompt (verbatim):**

```text
continue
```

**Assistant interpretation:** Continue working through the remaining SESSIONSTREAM-003 tasks after the Systemlab helper extraction and commits.

**Inferred user intent:** Keep reducing ambiguity in the remaining API/transport cleanup items and maintain the diary/task workflow.

### What I did

- Checked the open task list and selected the websocket fanout-only cleanup items.
- Updated `pkg/sessionstream/transport/ws/server.go` comments to document:
  - websocket command ingress is out of scope;
  - subscribe sends a current snapshot followed by future live UI events;
  - `sinceOrdinal` is parsed, stored, echoed, and traced, but does not trigger UI replay;
  - replay should stay behind explicit replay APIs for now;
  - the default adapter is reference/local-lab oriented and needs auth/origin/rate-limit wrappers in production.
- Added `TestServerRejectsCommandFramesAsUnsupported` in `pkg/sessionstream/transport/ws/server_test.go`.
- Updated `cmd/sessionstream-systemlab/chapters/phase-3-hydration-and-reconnect.md` to teach the same boundary:
  - accepted frames are subscribe/unsubscribe/ping;
  - command frames are unsupported;
  - `sinceOrdinal` is advisory;
  - websocket subscribe does not replay missed UI events.
- Marked the corresponding Phase 1 and Phase 4 tasks complete.

### Why

The remediation plan removed command ingress from the websocket adapter, but the remaining docs/tests still needed to make that boundary explicit. Without a test, a future change could accidentally turn the reference fanout transport into an implicit command transport.

### What worked

- The existing unknown-frame behavior already rejected `type: "command"`; the new test codifies that behavior.
- The chapter prose was the right place to teach the `sinceOrdinal` distinction because Phase 3 is where reconnect and subscriptions are introduced.

### What didn't work

No test failure occurred. The only design choice was whether to rename `sinceOrdinal` or document it. I documented it instead because the field remains useful as a client hint/diagnostic hook even though replay is not implemented in subscribe.

### What I learned

The cleanest boundary is: websocket subscribe provides convergence through snapshots plus future fanout; event-log replay is explicit and inspectable elsewhere. That preserves the teaching distinction between hydration, fanout, and replay.

### What was tricky to build

The subtle part is avoiding a half-replay contract. If `sinceOrdinal` looked authoritative but only affected hooks, users might assume missed UI events are replayed. The comments and chapter now explicitly say it is advisory until a real replay protocol exists.

### What warrants a second pair of eyes

- Whether `sinceOrdinal` should be renamed in a future breaking cleanup despite the new documentation.
- Whether unknown frame errors should close the websocket connection instead of leaving it open after an error frame.
- Whether production-readiness guidance belongs in package docs as well as the server type comment.

### What should be done in the future

- If websocket replay is ever implemented, add a new explicit protocol contract and tests for replay boundaries.
- Consider adding package-level transport documentation with a frame table.

### Code review instructions

Start with:

- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/transport/ws/server_test.go`
- `cmd/sessionstream-systemlab/chapters/phase-3-hydration-and-reconnect.md`
- `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/tasks.md`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream/transport/ws ./cmd/sessionstream-systemlab
make check
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
```

### Technical details

The new command-frame test sends:

```json
{
  "type": "command",
  "sessionId": "s-command",
  "name": "TestStart",
  "payload": { "text": "should not enter through websocket" }
}
```

The expected response is an error frame containing `unknown frame type` and no subscription side effect.

## Step 17: Deferred configurable bus decode ack/nack policy to GitHub issue

I left the flexible bus decode ack/nack implementation out of SESSIONSTREAM-003 and opened a dedicated GitHub issue with the detailed background, desired policy semantics, tests, non-goals, and default-behavior constraints.

This keeps SESSIONSTREAM-003 focused on the replay/store/systemlab remediation while preserving the production-hardening work as a concrete follow-up.

### Prompt Context

**User prompt (verbatim):**

```text
let's leave the flexible ack nack and open a gh issue for it, with all the detail needed
```

**Assistant interpretation:** Do not implement configurable ack/nack behavior now; create a detailed GitHub issue and update the ticket bookkeeping to show the item was deferred intentionally.

**Inferred user intent:** Avoid expanding the current remediation scope while ensuring the deferred production-hardening work is not lost.

### What I did

- Created GitHub issue #1: https://github.com/go-go-golems/sessionstream/issues/1
- Wrote issue details covering:
  - current record-and-ack behavior;
  - decode failure causes;
  - desired configurable policy API shape;
  - `RecordAndAck`, `RecordAndNack`, and `AckIfRecorded` semantics;
  - error record expectations;
  - tests to add;
  - non-goals;
  - SESSIONSTREAM-003 default-behavior decision.
- Updated `tasks.md` to mark the bus decode policy item as deferred to GitHub issue #1.

### Why

The current default is safe for local/dev/systemlab because it avoids poison-message retry loops. Production deployments may still want broker-level retry/DLQ behavior, but that deserves its own focused implementation and tests rather than being added late to this remediation slice.

### What worked

- `gh issue create` succeeded and returned issue #1.
- The issue body captures enough implementation detail to resume without relying on this chat context.

### What didn't work

No command failure occurred.

### What I learned

The deferred work is mostly about explicit operational policy, not core replay semantics. Keeping it separate helps avoid mixing production broker behavior with the current replay-store remediation.

### What was tricky to build

The important distinction is preserving the current poison-message-safe default while still allowing production users to opt into broker-managed nack/DLQ behavior. The issue therefore makes default preservation a requirement rather than an implementation detail.

### What warrants a second pair of eyes

- Whether the proposed `AckIfRecorded` policy is worth implementing or should be omitted for a smaller API.
- Whether decode and ordinal errors should share one policy path or have separate policy knobs.
- Whether the names should use `Ack`/`Nack` terminology or more domain-specific names.

### What should be done in the future

- Implement GitHub issue #1 after SESSIONSTREAM-003 is closed or split.
- Add policy-specific bus tests with a controllable message/pubsub test double.

### Code review instructions

Review:

- https://github.com/go-go-golems/sessionstream/issues/1
- `ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/tasks.md`

Validate ticket docs with:

```bash
cd sessionstream
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
```

## Step 18: Split large Phase 2 and Phase 5 Systemlab files

I cleaned up the remaining large Systemlab phase files by splitting Phase 2 and Phase 5 into small, lesson-oriented files. The split is mechanical: it does not change scenario behavior, API responses, projection behavior, or frontend contracts.

This finishes the Phase 6 structural cleanup items while preserving the educational flow. The phase entry files now hold constants and DTOs; runtime setup, actions, projections, checks, cloning, and rendering live in separate files with predictable names.

### Prompt Context

**User prompt (verbatim):**

```text
cleanup phase 2/5 large files
```

**Assistant interpretation:** Complete the remaining Systemlab Phase 6 cleanup by splitting the large Phase 2 and Phase 5 Go files into smaller, readable files.

**Inferred user intent:** Reduce file-size hotspots without over-abstracting the Systemlab teaching code.

### What I did

- Checked the working tree and current file sizes:
  - `phase2_lab.go`: 792 lines before split.
  - `phase5_lab.go`: 466 lines before split.
- Split Phase 2 into:
  - `phase2_lab.go` — constants, request/response DTOs, and state types.
  - `phase2_runtime.go` — hub/store/bus setup and lifecycle.
  - `phase2_actions.go` — scenario actions, waits, response assembly, export.
  - `phase2_projections.go` — command handler, projections, bus hooks, trace append.
  - `phase2_checks.go` — ordering and isolation checks.
  - `phase2_clone.go` — clone helpers.
  - `phase2_render.go` — transcript rendering and small render helpers.
- Split Phase 5 into:
  - `phase5_lab.go` — constants, request/response DTOs, and state/runtime types.
  - `phase5_runtime.go` — persistent/in-memory SQLite runtime setup and shutdown.
  - `phase5_actions.go` — scenario actions, snapshot/wait helpers, response assembly.
  - `phase5_projections.go` — command handler, UI/timeline projections, trace append.
  - `phase5_checks.go` — restart/persistence correctness checks.
  - `phase5_clone.go` — response clone helper.
- Ran `goimports` over the new split files.
- Updated `phase-2-ordering-and-ordinals.md` file references to point at the new Phase 2 split files.
- Marked Phase 6 split tasks complete.
- Ran tests.

### Why

Phase 2 and Phase 5 had become file-size hotspots because each mixed setup, action dispatch, projections, checks, clone helpers, and rendering in one file. Splitting by teaching role makes the code easier to review without introducing a new framework or hiding lesson-specific behavior behind abstractions.

### What worked

- The split was mechanical and `goimports` cleaned imports correctly.
- `go test ./cmd/sessionstream-systemlab` and `go test ./...` passed after the split.
- The resulting files are all substantially smaller and have clearer responsibility boundaries.

### What didn't work

No test or compile failure occurred. The main risk was accidentally dropping a function during the file move, so I split by function names and validated with package and full tests.

### What I learned

The earlier shared helper extraction made this split easier: trace helpers, snapshot helpers, and websocket hook helpers were already out of the phase files, so the remaining functions grouped naturally into runtime/actions/projections/checks/render files.

### What was tricky to build

The tricky part was keeping the split mechanical rather than redesigning the phases. Phase 2 especially has coupled bus hooks and response-building state, so I kept the state types in `phase2_lab.go` and moved behavior around those types into role-specific files.

Phase 5 has two runtime modes and restart semantics, so I kept all runtime construction/shutdown in one file and the scenario flow in another. That keeps persistence setup separate from the user-facing action sequence.

### What warrants a second pair of eyes

- Whether `phase2_clone.go` should remain separate or merge into `phase2_render.go` later.
- Whether Phase 5 should gain a render file if exports are added later.
- Whether any external documentation should point at the new split files beyond the Phase 2 chapter reference update.

### What should be done in the future

- Consider adding short package/file comments if new contributors need a roadmap.
- If Phase 2 grows again, consider extracting bus hook tests around `phase2Published`/`phase2Consumed`.
- If Phase 5 grows again, keep replay inspection UI/backend code separate from restart scenario code.

### Code review instructions

Start with the file layout:

```bash
ls cmd/sessionstream-systemlab/phase2*.go cmd/sessionstream-systemlab/phase5*.go
wc -l cmd/sessionstream-systemlab/phase2*.go cmd/sessionstream-systemlab/phase5*.go
```

Then review:

- `cmd/sessionstream-systemlab/phase2_lab.go`
- `cmd/sessionstream-systemlab/phase2_runtime.go`
- `cmd/sessionstream-systemlab/phase2_actions.go`
- `cmd/sessionstream-systemlab/phase2_projections.go`
- `cmd/sessionstream-systemlab/phase2_checks.go`
- `cmd/sessionstream-systemlab/phase2_render.go`
- `cmd/sessionstream-systemlab/phase5_lab.go`
- `cmd/sessionstream-systemlab/phase5_runtime.go`
- `cmd/sessionstream-systemlab/phase5_actions.go`
- `cmd/sessionstream-systemlab/phase5_projections.go`
- `cmd/sessionstream-systemlab/phase5_checks.go`

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab
go test ./...
make check
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
```

### Technical details

Post-split line counts:

```text
phase2_actions.go      ~210 lines
phase2_runtime.go      ~194 lines
phase2_projections.go  ~136 lines
phase2_render.go        ~92 lines
phase2_clone.go         ~73 lines
phase2_lab.go           ~66 lines
phase2_checks.go        ~62 lines
phase5_actions.go      ~167 lines
phase5_runtime.go      ~154 lines
phase5_checks.go        ~57 lines
phase5_projections.go   ~56 lines
phase5_lab.go           ~55 lines
phase5_clone.go         ~15 lines
```

## Step 19: Fixed lint findings after the Phase 2/5 split

I addressed the `make lint` findings reported after the Systemlab file split. The fixes are small hygiene changes: check deferred close errors in tests, remove an unused Phase 3 render helper, and fix an ineffectual `err` assignment in SQLite snapshot history lookup.

The SQLite fix is the only runtime-adjacent change: the historical snapshot query path had a shadowed `err` in the `asOf > 0` branch. Tests already covered historical snapshots, but lint correctly identified that the query assignment was writing to the inner scoped `err`. I changed the version cursor declaration to avoid shadowing.

### Prompt Context

**User prompt (verbatim):**

```text
make lint :

s, cgo: 782ns, max_per_file_from_linter: 755ns, invalid_issue: 617ns, filename_unadjuster: 566ns, fixer: 361ns, path_prettifier: 292ns, exclusion_paths: 286ns, diff: 267ns, severity-rules: 152ns 
INFO [runner] linters took 4.348203939s with stages: goanalysis_metalinter: 4.34645355s 
pkg/sessionstream/hydration/sqlite/store_test.go:43:22: Error return value of `reopened.Close` is not checked (errcheck)
    defer reopened.Close()
                       ^
pkg/sessionstream/hydration/sqlite/store_test.go:123:19: Error return value of `store.Close` is not checked (errcheck)
    defer store.Close()
                    ^
pkg/sessionstream/transport/ws/server_test.go:31:18: Error return value of `conn.Close` is not checked (errcheck)
    defer conn.Close()
                   ^
pkg/sessionstream/transport/ws/server_test.go:81:25: Error return value of `reconnected.Close` is not checked (errcheck)
    defer reconnected.Close()
                          ^
pkg/sessionstream/transport/ws/server_test.go:104:18: Error return value of `conn.Close` is not checked (errcheck)
    defer conn.Close()
                   ^
pkg/sessionstream/transport/ws/server_test.go:121:18: Error return value of `conn.Close` is not checked (errcheck)
    defer conn.Close()
                   ^
pkg/sessionstream/hydration/sqlite/store.go:174:9: ineffectual assignment to err (ineffassign)
        rows, err = s.db.QueryContext(ctx, `
             ^
cmd/sessionstream-systemlab/phase3_lab.go:390:6: func renderPhase3Markdown is unused (unused)
func renderPhase3Markdown(resp phase3RunResponse) string {
```

**Assistant interpretation:** Fix the reported lint failures and re-run validation.

**Inferred user intent:** Keep the cleanup branch green under the project's lint target, not just tests/build.

### What I did

- Replaced unchecked `defer Close()` calls in tests with deferred `require.NoError` wrappers:
  - `pkg/sessionstream/hydration/sqlite/store_test.go`
  - `pkg/sessionstream/transport/ws/server_test.go`
- Fixed the SQLite `Snapshot(asOf)` historical query branch by avoiding `err` shadowing around `uint64ToInt64(asOf)`.
- Removed unused `renderPhase3Markdown` from `cmd/sessionstream-systemlab/phase3_lab.go`.
- Removed the now-unused `encoding/json` import from `phase3_lab.go`.
- Ran `gofmt`.
- Ran `make lint`, `make check`, and `docmgr doctor`.

### Why

The branch should pass the project's lint target before further commits or final delivery. The lint findings were all concrete and low-risk to fix immediately.

### What worked

- `make lint` passed with `0 issues` after the fixes.
- `make check` passed after the lint fixes.
- `docmgr doctor` passed.

### What didn't work

No new failure occurred while fixing lint. The original lint output is preserved above as the trigger for this step.

### What I learned

`ineffassign` caught a subtle shadowing bug in the historical snapshot query path. The code was easy to read as if the outer `err` was updated, but the short declaration for `versionCursor` introduced an inner `err` scope.

### What was tricky to build

The websocket test file had several identical `defer conn.Close()` snippets. I updated all of them consistently to keep `errcheck` satisfied. For the SQLite bug, the fix was to declare `versionCursor` separately and assign both `versionCursor` and the existing outer `err` explicitly.

### What warrants a second pair of eyes

- The `Snapshot(asOf)` branch should be reviewed because the lint fix revealed real shadowing risk in replay/history logic.
- The test deferred close assertions are acceptable because these tests do not otherwise expect close errors.

### What should be done in the future

- Run `make lint` as part of the normal validation sequence whenever code moves across files.
- Consider adding `make check` to include lint if the team wants a single strict validation target.

### Code review instructions

Review:

- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`
- `pkg/sessionstream/transport/ws/server_test.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`

Validate with:

```bash
cd sessionstream
make lint
make check
docmgr --root ttmp doctor --ticket SESSIONSTREAM-003 --stale-after 30
```
