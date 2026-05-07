---
Title: Investigation diary
Ticket: SS-CODE-REVIEW-2026-05-07
Status: active
Topics:
    - sessionstream
    - code-review
    - cleanup
    - architecture
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/phase4_lab.go
      Note: Phase 4 now uses PipelineObserver for backend-event trace rows
    - Path: cmd/sessionstream-systemlab/ws_observer.go
      Note: Systemlab observer adapter replacing ws_hooks.go
    - Path: examples/chatdemo/chat.go
      Note: Removed demo engine Hooks API after Phase 4 moved to PipelineObserver
    - Path: pkg/sessionstream/hub.go
      Note: Made ErrorObserver panic-safe and surfaced error persistence failures
    - Path: pkg/sessionstream/hub_test.go
      Note: Added error observer panic and persistence-failure tests
    - Path: pkg/sessionstream/hydration/sqlite/store.go
      Note: Implemented additive migration behavior
    - Path: pkg/sessionstream/hydration/sqlite/store_test.go
      Note: Added SQLite tests for migration preservation
    - Path: pkg/sessionstream/transport/ws/observer.go
      Note: Extended TransportObserver stages and records so Systemlab can replace websocket hooks
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: |-
        Removed websocket Hooks dispatch and emitted observer records for semantic transport events
        Implemented hydration late-buffer filtering and aggregate fanout error returns
    - Path: pkg/sessionstream/transport/ws/server_test.go
      Note: Added/updated websocket tests for late-buffer filtering and fanout error propagation
    - Path: ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/design-doc/01-whole-package-code-review-and-intern-implementation-guide.md
      Note: Primary review deliverable whose investigation and validation are recorded by this diary
    - Path: ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/sources/01-inventory-output.txt
      Note: Captured inventory
    - Path: ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/sources/04-validation-output.txt
      Note: Captured test
ExternalSources: []
Summary: Chronological diary for the SS-CODE-REVIEW-2026-05-07 whole-package review and reMarkable delivery.
LastUpdated: 2026-05-07T17:02:00-04:00
WhatFor: Use this to understand how the package review was performed, what evidence was gathered, what failed, and how the final bundle was validated and delivered.
WhenToUse: Read before continuing this ticket, implementing findings, or reviewing the uploaded report.
---




# Diary

## Goal

This diary records the whole-package `sessionstream` review requested on 2026-05-07. It captures ticket setup, inventory, recent diary review, architecture/code-quality findings, validation commands, and reMarkable delivery steps.

## Step 1: Created the docmgr ticket and scoped the review

I created a new docmgr ticket directly under `sessionstream/ttmp` so the review has a durable home beside the repository's existing implementation history. The requested scope is broader than a defect hunt: it includes stale code, long files, unclear APIs, package organization, overengineering, and intern onboarding.

The ticket now has a primary design document for the long-form review and this diary for chronological investigation notes. I also added task items covering inventory, architecture mapping, cleanup inspection, writing, validation, and reMarkable upload.

### Prompt Context

**User prompt (verbatim):**

```text
Create a new docmgr ticket in sessionstream/ttmp to do a code review of the whole package, and especially the lately add code / improvements (see diaries too). 

We are looking for not just issues, but also deprecated code, unclear code, too long files, code that could be organized better, too large packages, overengineered functionality, etc... 

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable.


Keep a diary as you work.
```

**Assistant interpretation:** Create a new docmgr ticket in the `sessionstream` repository, review the whole package with special attention to recent observer/race/replay improvements and existing diaries, write a detailed intern-oriented architecture and cleanup guide, validate it, and upload it to reMarkable.

**Inferred user intent:** The user wants a durable, evidence-backed onboarding and cleanup artifact rather than a short chat answer.

**Commit (code):** N/A — documentation-only review ticket.

### What I did

- Loaded the relevant `docmgr`, `code-quality-review-cleanup`, `diary`, `ticket-research-docmgr-remarkable`, and `remarkable-upload` skills.
- Ran `docmgr status --summary-only` inside `sessionstream`.
- Created ticket `SS-CODE-REVIEW-2026-05-07` with topics `sessionstream,code-review,cleanup,architecture,onboarding`.
- Added the primary design document `design-doc/01-whole-package-code-review-and-intern-implementation-guide.md`.
- Added this diary document `reference/01-investigation-diary.md`.
- Added task items for inventory, architecture mapping, code-quality review, writing, validation, and upload.

### Why

The repository already has many design tickets and diaries. A whole-package code review should be stored in the same docmgr structure so later implementation work can link back to concrete findings and evidence.

### What worked

- `docmgr` was already initialized for `sessionstream/ttmp`.
- The vocabulary already had the needed review topics.
- The ticket workspace was created cleanly under `ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide`.

### What didn't work

Nothing failed in this setup step.

### What I learned

The repo already had a prior code review ticket (`SESSIONSTREAM-003`) and fresh implementation diaries for observers and the websocket reconnect race. This ticket therefore needed to review both the current code and those recent implementation decisions, not repeat the old audit verbatim.

### What was tricky to build

The requested deliverable combines code review, architecture explanation, implementation planning, diary, docmgr validation, and reMarkable delivery. I treated the primary design doc as the canonical long-form artifact and kept evidence artifacts under `sources/`.

### What warrants a second pair of eyes

- Whether the ticket ID should be shortened in future reviews; it is descriptive but long.
- Whether findings from this review should become separate implementation tickets immediately or after human review.

### What should be done in the future

- Convert accepted high-priority findings into implementation tasks or tickets.

### Code review instructions

Start with:

- `ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/design-doc/01-whole-package-code-review-and-intern-implementation-guide.md`
- `ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/reference/01-investigation-diary.md`

### Technical details

Commands:

```bash
cd sessionstream
docmgr status --summary-only
TICKET=SS-CODE-REVIEW-2026-05-07
docmgr ticket create-ticket --ticket "$TICKET" --title "Sessionstream whole-package code review and intern guide" --topics sessionstream,code-review,cleanup,architecture,onboarding
docmgr doc add --ticket "$TICKET" --doc-type design-doc --title "Whole-package code review and intern implementation guide"
docmgr doc add --ticket "$TICKET" --doc-type reference --title "Investigation diary"
docmgr task add --ticket "$TICKET" --text "Inventory sessionstream source, docs, tests, and recent diaries"
docmgr task add --ticket "$TICKET" --text "Map runtime architecture and public APIs for intern onboarding"
docmgr task add --ticket "$TICKET" --text "Inspect code quality hot spots, deprecated code, package boundaries, and overengineering"
docmgr task add --ticket "$TICKET" --text "Write detailed analysis/design/implementation guide with diagrams and pseudocode"
docmgr task add --ticket "$TICKET" --text "Validate docmgr ticket and upload bundle to reMarkable"
```

## Step 2: Inventoried the repository and read recent diaries

I inventoried source files, packages, line-count hotspots, marker strings, recent commits, and existing diaries before writing findings. I also saved the inventory commands and output into the ticket so the review is reproducible.

The recent diaries show that the latest work was not random churn. It completed a replay-store cleanup from `SESSIONSTREAM-003`, added observer APIs, and fixed a websocket reconnect race through hydrating subscriptions and fanout buffering.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Gather evidence first and especially inspect recent diary context before making recommendations.

**Inferred user intent:** The final review should identify real current risks, not re-open issues already fixed by recent work.

**Commit (code):** N/A — documentation-only review ticket.

### What I did

- Created `scripts/01-inventory.sh` inside the ticket.
- Ran the script and stored output in `sources/01-inventory-output.txt`.
- Read recent observer and websocket race diaries:
  - `SS-OBSERVERS/reference/01-implementation-diary.md`
  - `SS-WS-RACE/reference/01-implementation-diary.md`
  - `SESSIONSTREAM-003/reference/01-investigation-diary.md`
- Read key package files in:
  - `pkg/sessionstream`
  - `pkg/sessionstream/hydration/sqlite`
  - `pkg/sessionstream/transport/ws`
  - `examples/chatdemo`
  - `cmd/sessionstream-systemlab`
  - `README.md` and embedded reference docs
- Generated line-numbered evidence files under `sources/`.

### Why

The code-quality review skill requires concrete file evidence for findings. The recent diaries were especially important because they describe why the latest observer, replay-store, and race-fix code exists.

### What worked

The inventory clearly separated:

- compact public API files;
- large implementation hotspots;
- generated protobuf files;
- Systemlab educational code;
- recently added observer/race code.

Useful counts:

```text
745 pkg/sessionstream/hydration/sqlite/store.go
712 pkg/sessionstream/transport/ws/server.go
650 pkg/sessionstream/transport/ws/server_test.go
608 pkg/sessionstream/hub.go
575 pkg/sessionstream/hub_test.go
463 cmd/sessionstream-systemlab/lab_environment.go
440 examples/chatdemo/chat.go
395 cmd/sessionstream-systemlab/phase3_lab.go
```

### What didn't work

No command failed in this inventory step.

### What I learned

The `Deprecated:` markers reported by search are generated protobuf descriptor helper comments, not hand-written deprecated code. The active stale-code signal is mostly documentation/history: an old `evtstream` ticket link in `README.md`, legacy wording used as boundary documentation, and historical tickets in `ttmp`.

### What was tricky to build

The line-count hotspots include generated protobuf files, which should not be treated as cleanup targets. I filtered the recommendations toward hand-written files and treated generated files only as evidence of codegen size.

### What warrants a second pair of eyes

- Whether the old `evtstream` README link should be replaced or kept with a historical-name note.
- Whether generated protobuf files should be excluded from future line-count reports by default.

### What should be done in the future

- Keep `scripts/01-inventory.sh` updated if this ticket is resumed after further code changes.

### Code review instructions

Evidence files to inspect:

- `sources/01-inventory-output.txt`
- `sources/02-key-files-lines-1.txt`
- `sources/03-issue-snippets.txt`

### Technical details

Commands included:

```bash
cd sessionstream
rg --files -g '!dist/**' -g '!ttmp/**' | sort
go list ./...
find . -path './ttmp' -prune -o -path './dist' -prune -o -type f \( -name '*.go' -o -name '*.md' -o -name '*.proto' \) -print0 | xargs -0 wc -l | sort -nr | head -50
rg -n 'TODO|FIXME|Deprecated|deprecated|legacy|obsolete|HACK|XXX|panic\(' . --glob '!ttmp/**' --glob '!dist/**' || true
git log --oneline --decorate -30
```

## Step 3: Validated the codebase and captured a coverage hiccup

I ran normal tests, a websocket race test, lint, and coverage. Tests, race, and lint passed. The first coverage attempt failed due to a Go toolchain/cache mismatch after lint installed or used a different toolchain; I recorded the error and reran after `go clean -cache`, which succeeded.

This matters because it explains why the final coverage artifact contains both a failure record and a successful coverage result.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Validate the current repository state while reviewing it, and record failures honestly in the diary.

**Inferred user intent:** The review should distinguish code-quality recommendations from actual failing validation.

**Commit (code):** N/A — validation only.

### What I did

- Ran `go test ./...` and stored output in `sources/04-validation-output.txt`.
- Ran `go test ./pkg/sessionstream/transport/ws -race -count=1`.
- Ran `make lint`.
- Attempted `go test ./... -coverprofile=...` and captured a Go version mismatch failure.
- Ran `go clean -cache`.
- Reran coverage and stored the successful output in `sources/05-coverage-output.txt`.

### Why

The latest code includes concurrency-sensitive websocket changes. A race test is an important sanity check for the hydrating subscription and buffering code.

### What worked

Validation passed:

```text
go test ./...                                      PASS
go test ./pkg/sessionstream/transport/ws -race    PASS
make lint                                         0 issues
```

Coverage succeeded after cache cleanup and reported about 66.9% total statement coverage.

### What didn't work

The first coverage attempt failed with a Go toolchain/cache mismatch. Exact error pattern:

```text
compile: version "go1.26.1" does not match go tool version "go1.25.5"
```

Fix:

```bash
go clean -cache
go test ./... -coverprofile=...
go tool cover -func=...
```

### What I learned

The regular test/lint path was healthy, but coverage can be sensitive to mixed cached standard-library objects when toolchains change. Recording the failure prevents future readers from assuming there was a code failure.

Coverage also highlighted useful follow-up tests around noop store methods, SQLite view helpers, websocket ping/unsubscribe helpers, and frame helper branches.

### What was tricky to build

The coverage run initially produced partial package coverage mixed with many standard-library version mismatch errors. The correct action was to clear the Go cache and rerun, not to treat package failures as source-level errors.

### What warrants a second pair of eyes

- Whether the repo's Makefile should include a coverage target that avoids this toolchain-cache trap or runs `go clean -cache` conditionally.
- Whether the uncovered helper branches are worth prioritizing after the correctness fixes.

### What should be done in the future

- Add a stable coverage target if coverage will become part of routine review.
- Add focused tests for uncovered websocket and SQLite edge paths.

### Code review instructions

Review validation artifacts:

- `sources/04-validation-output.txt`
- `sources/05-coverage-output.txt`

### Technical details

Commands:

```bash
cd sessionstream
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
go test ./... -coverprofile=ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/sources/05-cover.out
go clean -cache
go test ./... -coverprofile=ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/sources/05-cover.out
go tool cover -func=ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/sources/05-cover.out
```

## Step 4: Wrote the primary architecture and cleanup guide

I wrote the primary design document as a long-form intern guide and evidence-backed code review. It explains the system model, package map, public APIs, runtime flows, recent changes, quality findings, deprecated/stale-code review, file-size assessment, and phased implementation plan.

The most important findings are narrow but real: a late websocket hydration-buffer filtering gap, hidden websocket fanout failures, destructive SQLite migrations, SQLite event conflict overwrites, shared in-memory SQLite DSNs, observer panic-safety mismatch, and large files that should be split after correctness fixes.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Produce the requested detailed analysis/design/implementation guide in the ticket.

**Inferred user intent:** The final document should be understandable by a new intern and actionable by maintainers.

**Commit (code):** N/A — documentation-only review ticket.

### What I did

- Rewrote `design-doc/01-whole-package-code-review-and-intern-implementation-guide.md`.
- Included diagrams, pseudocode, API references, package map, current-state architecture, detailed findings, cleanup sketches, and implementation phases.
- Anchored findings to concrete file references and line ranges.
- Distinguished generated protobuf `Deprecated` comments from real deprecated/stale code.
- Added a review checklist for future PRs.

### Why

A short summary would not satisfy the request. The repo is small enough for a whole-package review, but dense enough that an intern needs an explicit mental model and reading path.

### What worked

The recent diaries made it possible to tell which issues were already remediated and which remain. For example, the previous map-backed memory store and top-level `Struct` chat demo concerns are now mostly resolved.

### What didn't work

No write failed. The main challenge was keeping the report detailed without turning it into an unprioritized list of every minor style preference.

### What I learned

The strongest remaining cleanup opportunities are at boundaries:

- websocket snapshot/live ordering;
- websocket fanout error policy;
- SQLite durable-store guarantees;
- diagnostics API clarity;
- file organization around Hub, websocket, SQLite, and chat demo.

### What was tricky to build

Some findings are subtle concurrency or semantics issues rather than obvious lint failures. The late hydration-buffer duplicate concern depends on an interleaving between snapshot loading, buffer draining, and live transition. I documented it as a high-priority test-and-fix target rather than claiming it is easily reproducible.

### What warrants a second pair of eyes

- Finding 1's late-buffer duplicate hypothesis should be validated with a focused deterministic test before changing behavior.
- Finding 2's websocket fanout error policy needs an owner decision because returning errors may alter command success semantics.
- Finding 3's migration recommendation should be prioritized before durable external use.

### What should be done in the future

- Turn accepted findings into implementation tickets or phased tasks.
- Keep behavior changes separate from file-split commits.

### Code review instructions

Read the primary document sections in this order:

1. Executive summary.
2. Mental model for interns.
3. Current-state architecture by subsystem.
4. Detailed findings and cleanup sketches.
5. Phased implementation plan.

### Technical details

Primary document path:

```text
ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide/design-doc/01-whole-package-code-review-and-intern-implementation-guide.md
```

## Step 5: Updated ticket bookkeeping and prepared validation/upload

I updated ticket tasks, index metadata, related files, and changelog entries after writing the report. The ticket now links the main implementation files that shaped the review as well as the local evidence artifacts.

This step prepares the ticket for `docmgr doctor` and reMarkable upload.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Keep docmgr bookkeeping consistent with the analysis work.

**Inferred user intent:** The ticket should be searchable and reviewable later, not just contain a standalone Markdown file.

**Commit (code):** N/A — documentation-only review ticket.

### What I did

- Rewrote `tasks.md` to remove the placeholder and mark review tasks complete.
- Rewrote `index.md` with an overview, document links, status, and top findings.
- Planned file relationships for the primary design doc and diary.
- Prepared changelog entries summarizing setup, evidence gathering, report writing, validation, and upload.

### Why

Docmgr tickets are most useful when the index, tasks, changelog, and document frontmatter all tell a coherent story.

### What worked

The ticket structure is straightforward and matches prior sessionstream review tickets.

### What didn't work

N/A so far. Upload and doctor validation are recorded in the next step after they run.

### What I learned

For reviews with many source references, it is better to relate a tight set of core files via docmgr and keep exhaustive evidence in `sources/` rather than stuffing dozens of files into `RelatedFiles`.

### What was tricky to build

The report references many files. The docmgr skill recommends keeping related files tight, so I chose the most important architectural files instead of relating every file mentioned.

### What warrants a second pair of eyes

- Whether the related file list is too broad or too narrow for docmgr search.

### What should be done in the future

- If implementation begins, relate the specific files changed by each implementation task to the new implementation ticket or this ticket's diary.

### Code review instructions

Review:

- `index.md`
- `tasks.md`
- `changelog.md`
- primary design doc frontmatter after `docmgr doc relate`

### Technical details

Planned validation/upload commands:

```bash
cd sessionstream
docmgr doctor --ticket SS-CODE-REVIEW-2026-05-07 --stale-after 30
remarquee status
remarquee cloud account --non-interactive
remarquee upload bundle --dry-run ... --name "SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review" --remote-dir "/ai/2026/05/07/SS-CODE-REVIEW-2026-05-07" --toc-depth 2
remarquee upload bundle ... --name "SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review" --remote-dir "/ai/2026/05/07/SS-CODE-REVIEW-2026-05-07" --toc-depth 2
remarquee cloud ls /ai/2026/05/07/SS-CODE-REVIEW-2026-05-07 --long --non-interactive
```

## Step 6: Validated docmgr metadata and uploaded the bundle to reMarkable

I validated the completed ticket with `docmgr doctor`, confirmed `remarquee` and the cloud account, ran a dry-run bundle upload, uploaded the final PDF bundle, and verified the remote listing.

The uploaded bundle includes the ticket index, primary design/code-review guide, investigation diary, tasks, and changelog. The evidence source files remain in the ticket workspace for local review but were not included in the PDF bundle to keep the reMarkable document readable. After adding this final diary step, I uploaded a second no-overwrite bundle named `SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review Final` so the on-device copy includes the delivery evidence without forcing over the first upload.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Complete the requested docmgr and reMarkable delivery after writing and validating the review.

**Inferred user intent:** The user wants the final review available both in the repository ticket and as a readable PDF on the reMarkable.

**Commit (code):** N/A — documentation-only review ticket.

### What I did

- Ran `docmgr doctor --ticket SS-CODE-REVIEW-2026-05-07 --stale-after 30`.
- Confirmed `remarquee status`.
- Confirmed `remarquee cloud account --non-interactive`.
- Ran `remarquee upload bundle --dry-run` for the ticket documents.
- Ran the real `remarquee upload bundle`.
- Verified the remote folder with `remarquee cloud ls`.
- Updated the ticket changelog with delivery information.

### Why

Docmgr validation catches metadata and relationship issues before delivery. The dry-run confirms that the bundle inputs, generated PDF name, and remote folder are correct before uploading.

### What worked

Docmgr doctor passed:

```text
## Doctor Report (1 findings)

### SS-CODE-REVIEW-2026-05-07

- ✅ All checks passed
```

Dry-run passed and the real upload succeeded:

```text
OK: uploaded SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review.pdf -> /ai/2026/05/07/SS-CODE-REVIEW-2026-05-07
```

Remote verification succeeded:

```text
[f]	SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review
[f]	SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review Final
```

### What didn't work

No upload or doctor failure occurred. The earlier coverage hiccup is recorded in Step 3 and was resolved before upload.

### What I learned

The current Markdown rendered cleanly through the reMarkable upload path. Keeping the verbatim user prompt in a fenced `text` block avoided the LaTeX issue seen in older ticket diaries with literal backslash sequences.

### What was tricky to build

The ticket has many evidence artifacts, but bundling all of them into the PDF would make the reMarkable document noisy. I uploaded the human-readable documents and left the raw evidence under `sources/` in the repo.

### What warrants a second pair of eyes

- Confirm whether the evidence files should also be uploaded as a separate source bundle if a reviewer wants line-numbered snippets on-device.
- If only one reMarkable document should remain, remove or archive the first non-final upload manually; I did not force-overwrite it because force can delete annotations.

### What should be done in the future

- If follow-up implementation begins, either continue this ticket's diary or create focused implementation tickets for the high-priority findings.

### Code review instructions

Validate delivery with:

```bash
cd sessionstream
docmgr doctor --ticket SS-CODE-REVIEW-2026-05-07 --stale-after 30
remarquee cloud ls /ai/2026/05/07/SS-CODE-REVIEW-2026-05-07 --long --non-interactive
```

### Technical details

Commands:

```bash
cd sessionstream
docmgr doctor --ticket SS-CODE-REVIEW-2026-05-07 --stale-after 30
remarquee status
remarquee cloud account --non-interactive
TICKET_DIR=$(pwd)/ttmp/2026/05/07/SS-CODE-REVIEW-2026-05-07--sessionstream-whole-package-code-review-and-intern-guide
remarquee upload bundle --dry-run \
  "$TICKET_DIR/index.md" \
  "$TICKET_DIR/design-doc/01-whole-package-code-review-and-intern-implementation-guide.md" \
  "$TICKET_DIR/reference/01-investigation-diary.md" \
  "$TICKET_DIR/tasks.md" \
  "$TICKET_DIR/changelog.md" \
  --name "SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review" \
  --remote-dir "/ai/2026/05/07/SS-CODE-REVIEW-2026-05-07" \
  --toc-depth 2
remarquee upload bundle \
  "$TICKET_DIR/index.md" \
  "$TICKET_DIR/design-doc/01-whole-package-code-review-and-intern-implementation-guide.md" \
  "$TICKET_DIR/reference/01-investigation-diary.md" \
  "$TICKET_DIR/tasks.md" \
  "$TICKET_DIR/changelog.md" \
  --name "SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review" \
  --remote-dir "/ai/2026/05/07/SS-CODE-REVIEW-2026-05-07" \
  --toc-depth 2
remarquee cloud ls /ai/2026/05/07/SS-CODE-REVIEW-2026-05-07 --long --non-interactive
```

## Step 7: Replaced lab websocket hooks with observers and removed hook APIs

I implemented the follow-up cleanup discussed after the review: Systemlab now consumes the websocket transport through `TransportObserver` instead of the old websocket `Hooks` callbacks. I also removed the websocket `Hooks` type and `WithHooks` option from the transport package.

While doing that, I also removed the chat demo's example-specific `Hooks` API. Phase 4 now uses the Hub `PipelineObserver` to record backend event trace rows, which keeps tracing at the framework observer layer instead of adding a side channel to the demo engine.

### Prompt Context

**User prompt (verbatim):**

```text
can we rewrite the lab code to use the new observers, so that we can delete hooks?
```

**Assistant interpretation:** Replace Systemlab's use of websocket hook callbacks with the new observer APIs, then delete the hook surfaces that only existed for lab tracing.

**Inferred user intent:** The user wants the observer model to become the single diagnostics path and avoid carrying redundant hook-style APIs.

**Commit (code):** pending — working tree contains implementation changes but no commit was created in this step.

### What I did

- Replaced `cmd/sessionstream-systemlab/ws_hooks.go` with `ws_observer.go`.
- Changed Phase 3, 4, and 5 websocket setup from `wstransport.WithHooks(...)` to `wstransport.WithTransportObserver(...)`.
- Added higher-level transport observer stages for the lab-friendly events that hooks previously represented:
  - `snapshot_sent`
  - `subscribed`
  - `unsubscribed`
  - `ui_event_sent`
  - `unsubscribe_received`
- Extended `TransportRecord` with a cloned `UIEvent` for `TransportStageUIEventSent` so Phase 4/5 can still render UI payload details.
- Removed `Hooks` and `WithHooks` from `pkg/sessionstream/transport/ws`.
- Removed hook callback dispatch from `server.go`.
- Removed chat demo `Hooks` / `WithHooks` and made Phase 4 use `sessionstream.WithPipelineObserver` for backend event trace rows.
- Ran formatting, tests, websocket race test, and lint.

### Why

The new observer APIs are the intended diagnostics path. Keeping a second hook system in the websocket transport made the API surface larger and forced Systemlab to depend on a lab-only callback shape instead of the public observer records.

### What worked

Validation passed:

```text
go test ./cmd/sessionstream-systemlab ./pkg/sessionstream/transport/ws ./examples/chatdemo
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
```

The search for old hook APIs now only finds intentional `BusObserverHooks` and `PipelineObserverHooks` adapters, plus prose references to persistence hooks:

```bash
rg -n 'Hooks|WithHooks|hook|hooks' . --glob '!ttmp/**' --glob '!dist/**'
```

### What didn't work

The first lint run failed because the curated Systemlab observer switch intentionally handles only teaching-relevant transport stages. The exhaustive linter reported all unhandled `TransportStage` constants.

Exact failure:

```text
cmd/sessionstream-systemlab/ws_observer.go:25:3: missing cases in switch of type ws.TransportStage ... (exhaustive)
```

Fix: added a targeted `//nolint:exhaustive` comment explaining that Systemlab renders a curated teaching trace, not every low-level transport observation.

### What I learned

Deleting hooks exposed which observer records were not quite high-level enough for Systemlab. The transport observer had low-level queued/written frame records, but the lab needed semantic events like "snapshot sent" and "UI event sent with payload." Adding those stages made the observer API a better replacement rather than forcing the lab to reverse-engineer low-level frame records.

### What was tricky to build

The websocket hooks used to expose full UI payloads. `TransportRecord` previously carried summaries and counts, not a cloned `UIEvent`. To keep the Phase 4/5 checks and trace details intact, I added a cloned `UIEvent` only for `TransportStageUIEventSent`.

Phase 4 also used `chatdemo.WithHooks` to record backend events before publish. Replacing that with `PipelineObserver` changes the trace source from a demo-engine side channel to the actual Hub pipeline, which is cleaner and more faithful to the framework.

### What warrants a second pair of eyes

- Confirm that adding `UIEvent` to `TransportRecord` is acceptable for observer payload retention and API shape.
- Confirm that the new high-level transport stages are the right stable names.
- Confirm that removing `chatdemo.WithHooks` is acceptable as an example API break.

### What should be done in the future

- Consider updating the reference docs to mention `TransportStageUIEventSent`, `TransportStageSnapshotSent`, and `TransportStageSubscribed` as the preferred lab/diagnostic stages.
- If external code depended on websocket `Hooks`, mention the removal in release notes.

### Code review instructions

Start with:

- `pkg/sessionstream/transport/ws/observer.go`
- `pkg/sessionstream/transport/ws/server.go`
- `cmd/sessionstream-systemlab/ws_observer.go`
- `cmd/sessionstream-systemlab/phase3_lab.go`
- `cmd/sessionstream-systemlab/phase4_lab.go`
- `cmd/sessionstream-systemlab/phase5_runtime.go`
- `examples/chatdemo/chat.go`

Validate with:

```bash
cd sessionstream
go test ./cmd/sessionstream-systemlab ./pkg/sessionstream/transport/ws ./examples/chatdemo
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
```

## Step 8: Implemented review findings 1 through 6

I addressed the six highest-priority findings from the review in one focused implementation slice. The changes cover websocket hydration ordering, websocket fanout error propagation, SQLite migration safety, event-log conflict semantics, in-memory SQLite isolation, and error observer robustness.

This is the first direct remediation pass after the review. I kept the changes targeted and backed each behavior change with tests before committing.

### Prompt Context

**User prompt (verbatim):**

```text
address finding 1 -6 , commit at appropriate intervals, keep a diary
```

**Assistant interpretation:** Implement the first six findings from the code review, keep the docmgr diary updated, validate, and create focused git commits.

**Inferred user intent:** The user wants the highest-value correctness and API cleanup recommendations from the review to move from documentation into code while preserving a reviewable history.

**Commit (code):** `c23c9e98b581e633ec8dc3dd40ab41b1f93ef5fd` — "fix(sessionstream): address review findings 1-6"

### What I did

- Finding 1: Added shared buffered-batch filtering and made `markLive` return only late batches with `ordinal > snapshotOrdinal`.
- Finding 1: Added `TestMarkLiveFiltersLateBufferedBatchesAlreadyCoveredBySnapshot`.
- Finding 2: Changed websocket `PublishUI` to aggregate delivery failures with `errors.Join` and return them to the Hub.
- Finding 2: Updated hydration-buffer overflow test to assert the returned fanout error.
- Finding 3: Removed destructive SQLite migration table drops and made migration additive.
- Finding 3: Added compatibility handling for older `sessionstream_errors` tables missing `raw_message` / `metadata_json` columns.
- Finding 3: Added `TestMigratePreservesExistingRowsAndAddsErrorColumns`.
- Finding 4: Changed `AppendEvent` from overwrite-upsert to `INSERT OR IGNORE` plus identity check: exact duplicate is idempotent, conflicting duplicate returns an error.
- Finding 4: Added `TestStoreAppendEventAllowsOnlyIdenticalDuplicate`.
- Finding 5: Changed `NewInMemory` to use a unique UUID-backed memory DSN by default.
- Finding 5: Added `TestNewInMemoryStoresAreIsolated`.
- Finding 6: Made `ErrorObserver` delivery panic-safe and added `cloneErrorRecord` for raw bytes and metadata.
- Finding 6: Surfaced `ErrorStore.RecordError` failures as `ErrorKindStore` observer records without recursive error persistence.
- Finding 6: Added tests for panic recovery and error persistence failure visibility.
- Updated `tasks.md` to mark findings 1-6 remediation tasks complete.

### Why

These findings were the highest-value review outcomes because they affect correctness and operational trust more than style:

- reconnect hydration should not duplicate snapshot-covered live events;
- fanout failures should be visible to Hub-level error reporting;
- durable stores should not drop data during migration;
- event logs should not silently mutate canonical events;
- in-memory stores should not share hidden state by default;
- diagnostic observers must not be able to crash the pipeline.

### What worked

All targeted and full validation passed:

```text
go test ./pkg/sessionstream/transport/ws ./pkg/sessionstream/hydration/sqlite ./pkg/sessionstream
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
```

### What didn't work

No validation failure occurred during this implementation slice. The main risk was deciding the fanout error policy for Finding 2; I chose to return aggregate delivery errors to the Hub so `ErrorKindFanout` can be emitted by the existing Hub path.

### What I learned

The late-buffer duplicate fix was small once the buffer filtering was extracted into a helper. The bigger semantic change is actually Finding 2: returning fanout errors means callers may now see command failures when websocket delivery fails. That is stricter and more observable, but it is an API behavior change worth reviewing.

SQLite migration was also less invasive than expected because the current schema can be created additively with `CREATE TABLE IF NOT EXISTS`; only older `sessionstream_errors` columns needed explicit `ALTER TABLE` support.

### What was tricky to build

`reportError` needed to surface error persistence failures without recursively trying to persist that new store-error record. I split observer delivery into `observeError`, which is panic-safe and clone-based, and kept persistence only in `reportError` for the original record.

`AppendEvent` needed to preserve idempotent retry semantics while rejecting mismatched duplicate ordinals. I avoided driver-specific conflict error handling by using `INSERT OR IGNORE`, checking `RowsAffected`, and comparing the existing event row when the insert was ignored.

### What warrants a second pair of eyes

- Returning websocket fanout errors may alter command success semantics for applications with flaky clients.
- The additive migration helper currently adds known missing columns; future migrations should become explicitly versioned functions if schema changes continue.
- The event conflict error currently reports event names but not both payloads; that is safer but less diagnostic.

### What should be done in the future

- Add release notes for the stricter fanout error behavior.
- Consider a configurable websocket fanout failure policy only if callers need close-and-continue semantics.
- Convert future SQLite schema changes into named `migrateFromNToM` functions.

### Code review instructions

Start with:

- `pkg/sessionstream/transport/ws/server.go`
- `pkg/sessionstream/transport/ws/server_test.go`
- `pkg/sessionstream/hydration/sqlite/store.go`
- `pkg/sessionstream/hydration/sqlite/store_test.go`
- `pkg/sessionstream/hub.go`
- `pkg/sessionstream/hub_test.go`

Validate with:

```bash
cd sessionstream
go test ./pkg/sessionstream/transport/ws ./pkg/sessionstream/hydration/sqlite ./pkg/sessionstream
go test ./...
go test ./pkg/sessionstream/transport/ws -race -count=1
make lint
docmgr doctor --ticket SS-CODE-REVIEW-2026-05-07 --stale-after 30
```
