---
Title: Investigation Diary
Ticket: EVT-STREAM-006
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - implementation
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/bus.go
      Note: Watermill publisher adapter and bus configuration surface added for Phase 2.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/consumer.go
      Note: Canonical consumer loop with decode, ordinal assignment, and projection application.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go
      Note: Systemlab Lab 02 backend, publish/consume tracing, restart handling, and transcript export.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-006 covering the Watermill-backed Phase 2 event path, Systemlab Ordering and Ordinals lab, manual validation, and fixes for an interaction deadlock plus browser ordinal precision." 
LastUpdated: 2026-04-20T01:35:00-04:00
WhatFor: "Record how Phase 2 was implemented, validated, debugged, and prepared for later websocket integration."
WhenToUse: "When reviewing the Phase 2 implementation, understanding why ordinals are assigned on consume, or continuing the bus/consumer work later."
---

# Diary

## Goal

Capture the implementation work for EVT-STREAM-006 so a reviewer can follow how the purely local Phase 1 path was replaced by a real Watermill-backed publish/consume boundary, how the Systemlab Ordering and Ordinals lab was built, and what issues were discovered when exercising the lab interactively.

## Step 1: Add the Watermill bus path, consumer, ordinal assigner, and Lab 02

The first Phase 2 step was to make the framework cross a real bus boundary instead of immediately projecting local events in the same call stack. That meant adding a Watermill-backed publisher, a separate bus consumer, consumption-time ordinal assignment, and a dedicated output seam for projected UI events.

At the same time, I built the matching Systemlab page that exercises those seams directly. The page now publishes to a real `gochannel` bus, displays publish and consume metadata side by side, supports multiple sessions, burst publishing, restart/reset controls, and exports captured transcripts for review.

### Prompt Context

**User prompt (verbatim):** "continue phase 2"

**Assistant interpretation:** Implement the next phase of the evtstream framework by replacing the local event path with a Watermill-backed consumer pipeline and building the matching Systemlab Ordering and Ordinals lab.

**Inferred user intent:** The user wants the implementation plan to continue in order, with the framework and Systemlab evolving together so the new bus/ordering behavior is both real and inspectable.

**Commit (code):** 8b14bb8fe963d91931b98f0e4fed04e24b3bde1c — "feat(evtstream): add phase 2 watermill consumer and ordering lab"

### What I did
- Added the Phase 2 framework files under `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream`:
  - `bus.go`
  - `consumer.go`
  - `ordinals.go`
  - `fanout.go`
- Extended `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hub.go` to support:
  - `WithEventBus(...)`
  - `WithUIFanout(...)`
  - `Run(...)`
  - `Shutdown(...)`
  - local-vs-bus publisher selection
- Implemented component and unit tests:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/bus_test.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/ordinals_test.go`
- Added the Systemlab Phase 2 backend in `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go`.
- Extended the Systemlab server and modular frontend to include Phase 2:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/server.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase2.html`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/phase2.js`
  - plus updates to `static/index.html`, `static/app.css`, `static/js/api.js`, and `static/js/main.js`
- Added a Systemlab Phase 2 test in `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/lab_environment_test.go`.
- Captured artifacts:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-006--phase-2-watermill-bus-consumer-and-ordering-lab/reference/02-phase-2-run-response.json`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-006--phase-2-watermill-bus-consumer-and-ordering-lab/reference/03-phase-2-transcript.json`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-006--phase-2-watermill-bus-consumer-and-ordering-lab/reference/04-phase-2-transcript.md`

### Why
- Phase 2 is the first phase where the architecture needs a real publisher/consumer separation.
- Consumption-time ordinal assignment is one of the core framework invariants, so it had to move into a dedicated consumer path before websocket work starts.
- The Systemlab page needs to make ordering visible rather than implied, which is why it shows message metadata, per-session ordinals, and assertion badges.

### What worked
- The bus-backed `Hub.Submit(...)` path now publishes backend events with ordinal zero and assigns final ordinals only in the consumer.
- The ordinal assigner correctly derives ordinals from Redis-style stream ids and falls back safely when metadata is missing or invalid.
- The new `UIFanout` seam gives the consumer a transport-neutral place to send projected UI events, which prepares Phase 3 without coupling the consumer to websocket code.
- The Systemlab Lab 02 page demonstrates two-session ordering, burst publishing, stream-id fault injection, restart/reset behavior, and transcript export.

### What didn't work
- My first broad `go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...` command appeared to hang during the Systemlab package tests while I was still iterating on Phase 2. That turned out to be a real interaction problem, not a flaky test harness.
- While manually exercising the page in the browser, the first Phase 2 implementation also appeared to "hang" on interaction, which matched the unit-test symptom and confirmed the bug was in the implementation.

### What I learned
- The Watermill split is still manageable as long as the core keeps small seams: publisher adapter, consumer loop, ordinal assigner, and UI fanout.
- Large consumption-time ordinals are fine on the backend but need explicit string rendering once they cross into browser-facing JSON.

### What was tricky to build
- The central tricky part was keeping the architectural rules intact while still making Systemlab useful. The handler must publish canonical backend events and must never assign ordinals or write UI/timeline state directly. That required discipline in both the framework code and the lab code.
- The second tricky part was that `gochannel` is asynchronous enough to need explicit wait helpers in the lab code so the HTTP responses include the consumed state rather than just the publish request.

### What warrants a second pair of eyes
- Whether the current `evtstream_partition_key` metadata approach is the right cross-backend expression of the per-session ordering rule.
- Whether `UIFanout` should remain a minimal push interface or grow richer metadata before Phase 3.

### What should be done in the future
- Phase 3 should plug websocket fanout into the `UIFanout` seam instead of teaching the consumer about websocket specifics.
- If Redis or another backend is introduced next, confirm the partition/ordering strategy maps cleanly from the current metadata convention.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/bus.go`, `consumer.go`, and `ordinals.go`.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go` and the new `static/partials/phase2.html` + `static/js/pages/phase2.js` pair.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && ./.bin/golangci-lint run ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && make evtstream-check`

### Technical details
- Happy-path validation after the implementation stabilized:
  - `cd pinocchio && ./.bin/golangci-lint run ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd pinocchio && make evtstream-check`
- Manual Phase 2 capture flow:
  - `go run ./cmd/evtstream-systemlab --addr :18097`
  - `curl -sf -X POST http://127.0.0.1:18097/api/phase2/run -H 'Content-Type: application/json' -d '{"action":"publish-a","sessionA":"s-a","sessionB":"s-b","burstCount":3,"streamMode":"derived"}'`
  - `curl -sf -X POST http://127.0.0.1:18097/api/phase2/run -H 'Content-Type: application/json' -d '{"action":"publish-b","sessionA":"s-a","sessionB":"s-b","burstCount":3,"streamMode":"derived"}'`
  - `curl -sf -X POST http://127.0.0.1:18097/api/phase2/run -H 'Content-Type: application/json' -d '{"action":"burst-a","sessionA":"s-a","sessionB":"s-b","burstCount":3,"streamMode":"derived"}'`

## Step 2: Fix the interaction deadlock and browser ordinal precision issue

While exercising the new lab, the user pointed out that Phase 2 seemed to hang on interaction. That matched the earlier test timeout, so I treated it as a real correctness issue in the new code rather than a tooling problem.

The root cause was a lock-ordering bug in the Phase 2 command handler: it held `labEnvironment.mu` while calling `pub.Publish(...)`, and the publish path immediately re-entered the lab through the message mutator and publish observer, which also attempted to lock `labEnvironment.mu`. I fixed the deadlock by limiting the lock to the trace write and releasing it before the publish call.

### Prompt Context

**User prompt (verbatim):** "systemlab phase 2 seems to hang when interacting."

**Assistant interpretation:** Investigate the new Phase 2 interaction path, find the reason the UI freezes or blocks, and fix it before continuing.

**Inferred user intent:** The user wants the new phase to be genuinely usable, not just compiling; the interactive lab must work without hanging.

**Commit (code):** 8b14bb8fe963d91931b98f0e4fed04e24b3bde1c — "feat(evtstream): add phase 2 watermill consumer and ordering lab"

### What I did
- Inspected the Phase 2 interaction path and noticed that the command handler held the environment mutex across `pub.Publish(...)`.
- Fixed `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go` so the lock is released before publishing.
- Re-ran the focused test:
  - `cd pinocchio && go test ./cmd/evtstream-systemlab/... -run TestLabEnvironmentRunAndExportPhase2 -count=1 -v`
- Re-ran full targeted validation:
  - `cd pinocchio && ./.bin/golangci-lint run ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd pinocchio && make evtstream-check`
- Used both `curl` and Playwright against a live server to verify that clicking `Publish A` now updates the page instead of hanging.
- While doing that browser validation, I found a second issue: large consumer-side ordinals were being rounded in JavaScript because they exceeded the browser's safe integer range. I changed the Phase 2 HTTP responses to render ordinals as strings in:
  - trace details
  - message history
  - per-session ordinal lists
  - snapshots
  - fanout payloads

### Why
- A hanging Systemlab page would make the phase effectively unusable and would also hide a real locking bug in the new bus path.
- Rendering ordinals as strings is necessary for a lab whose whole purpose is to make ordering visible and exact.

### What worked
- The deadlock disappeared once the mutex was no longer held across the publish call.
- The focused Phase 2 test now completes quickly instead of stalling.
- The browser page now updates correctly on interaction.
- The displayed ordinals now match the backend values exactly because they are serialized as strings.

### What didn't work
- The original buggy path effectively deadlocked on interaction. The problematic shape was: hold `labEnvironment.mu` in `handlePhase2Command(...)` -> call `pub.Publish(...)` -> re-enter `phase2MessageMutator(...)` / `phase2Published(...)` -> attempt to take the same mutex.

### What I learned
- Any Systemlab callback path that re-enters the lab from framework hooks has to be treated like concurrent code, even when the original call starts from a single HTTP request.
- Browser numeric precision matters for exactly the kind of large ordinals this phase produces, so string rendering is the safer default for inspection-oriented APIs.

### What was tricky to build
- The tricky part was that the same underlying bug explained both symptoms: the apparent test hang and the apparent UI hang. Once I treated the test timeout as a concurrency clue instead of a flaky harness, the lock-ordering problem became obvious.

### What warrants a second pair of eyes
- Whether any other future observer or fanout callbacks accidentally hold `labEnvironment.mu` across framework calls.
- Whether later client-facing APIs should standardize on string ordinals anywhere `uint64` values may exceed JavaScript precision.

### What should be done in the future
- Keep observer and mutator callbacks small and avoid holding app-level locks across framework or bus calls.
- Reuse the string-ordinal rule in later browser-facing transcript endpoints if those ordinals can also exceed safe integer range.

### Code review instructions
- Review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go` around `handlePhase2Command`, `phase2MessageMutator`, `phase2Published`, and `phase2Consumed`.
- Then confirm the string conversion path in `buildPhase2Response(...)` and `phase2UIProjection(...)`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./cmd/evtstream-systemlab/... -run TestLabEnvironmentRunAndExportPhase2 -count=1 -v`
  - run the server and click `Publish A` on the Phase 2 page

### Technical details
- Focused debug/validation commands used while fixing the bug:
  - `cd pinocchio && go test ./cmd/evtstream-systemlab/... -run TestLabEnvironmentRunAndExportPhase2 -count=1 -v`
  - `curl -sf -X POST http://127.0.0.1:18096/api/phase2/run -H 'Content-Type: application/json' -d '{"action":"publish-a","sessionA":"s-a","sessionB":"s-b","burstCount":4,"streamMode":"derived"}'`
  - Playwright navigation to `http://127.0.0.1:18095/#phase2` followed by clicking `Publish A`
