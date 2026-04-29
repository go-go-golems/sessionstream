---
Title: Investigation Diary
Ticket: EVT-STREAM-007
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - websocket
    - implementation
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/transport/ws/server.go
      Note: Public websocket transport implementation with connection registry, subscribe/unsubscribe flow, snapshot-before-live behavior, and UI fanout.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/transport/ws/server_test.go
      Note: Transport tests for empty snapshot, reconnect, and subscription tracking.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase3_lab.go
      Note: Systemlab Lab 03 backend wiring, trace generation, and reconnect-oriented checks.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-007 covering the websocket transport, snapshot-before-live sequencing, Lab 03 backend/page wiring, and validation of reconnect-oriented invariants."
LastUpdated: 2026-04-20T06:40:00-04:00
WhatFor: "Record how Phase 3 was implemented, validated, and prepared for later example and persistence phases."
WhenToUse: "When reviewing the Phase 3 websocket transport, understanding why snapshot-before-live is enforced, or continuing reconnect-oriented work later."
---

# Diary

## Goal

Capture the implementation work for EVT-STREAM-007 so a reviewer can see how the framework gained its first real websocket transport, how `ConnectionId` tracking and subscription routing were implemented, and how the matching Systemlab reconnect lab demonstrates snapshot-before-live behavior instead of merely describing it.

## Step 1: Build the websocket transport and the Hydration/Reconnect lab

This step turned the transport seam from a planned abstraction into a working public package. I added a dedicated websocket server under `evtstream/transport/ws`, made it implement `evtstream.UIFanout`, and wired it so a client receives a snapshot before entering the live subscription set for a session.

I then built the matching Systemlab Lab 03 around that transport instead of mocking the behavior. The page now opens real websocket clients, allows connect/disconnect/subscribe flows, seeds session activity through public `evtstream` commands, and displays backend trace, active connections, and current store snapshot together.

### Prompt Context

**User prompt (verbatim):** "implement phase 3-4-5, commit at appropriate intervals, keep detailed diaries in their respective tickets."

**Assistant interpretation:** Continue the implementation plan by finishing the remaining runtime phases, committing the code, and recording detailed implementation diaries inside the matching ticket workspaces.

**Inferred user intent:** The user wants the planned framework phases to move from documentation into working code, with careful ticket bookkeeping and continuation-friendly diaries rather than only code changes.

**Commit (code):** `8941f33bb752fda7572fa3daf3a9f7315bf0a171` — `feat(evtstream): implement phases 3-5 transport persistence and chat labs`

### What I did
- Added the public websocket transport package:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/transport/ws/server.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/transport/ws/server_test.go`
- Implemented connection bookkeeping keyed by `ConnectionId`, per-session subscription sets, client frame handling, snapshot delivery, and downstream UI-event fanout.
- Kept snapshot-before-live sequencing explicit by sending the snapshot before inserting the connection into the session subscription set.
- Added Phase 3 Systemlab backend wiring in:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase3_lab.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/snapshot_provider.go`
- Extended the Systemlab server with real Phase 3 HTTP and websocket endpoints:
  - `/api/phase3/run`
  - `/api/phase3/state`
  - `/api/phase3/ws`
- Replaced the Phase 3 placeholder page with a real two-client lab surface:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase3.html`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/phase3.js`
  - plus `static/js/api.js`, `server.go`, `lab_environment.go`, and `lab_environment_test.go` updates.
- Captured a Phase 3 lab artifact:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-007--phase-3-websocket-transport-and-hydration-reconnect-lab/reference/02-phase-3-run-response.json`

### Why
- Phase 3 is the first point where transport lifecycle matters enough to deserve its own public package rather than app-local glue.
- The framework needed a clean answer to: how do clients connect, subscribe, hydrate, and then receive live UI events without the consumer knowing anything about websocket details?
- The Systemlab page needed to exercise the same websocket protocol future apps will use, otherwise the lab would only be documentation theater.

### What worked
- The websocket transport cleanly owns `ConnectionId` allocation, subscription bookkeeping, and session-scoped routing without leaking transport details into handlers or projections.
- `snapshot-before-live` now holds as an explicit transport rule and is covered in both package tests and Systemlab traces.
- The transport tests cover empty snapshot, populated snapshot on reconnect, and subscription tracking.
- The Systemlab page now allows two independent clients to connect, subscribe, disconnect, reconnect, and inspect the backend trace and current snapshot.

### What didn't work
- There was no major runtime failure in this slice once the transport shape was chosen, but the first design challenge was a circular dependency between the websocket transport's need for snapshots and the hub's need for a fanout target. I solved that by introducing a tiny hydration-store snapshot adapter rather than forcing the transport to depend directly on the hub.

### What I learned
- The websocket transport can remain small and understandable when it is treated strictly as a downstream delivery mechanism for projected UI events.
- `ConnectionId` and `SessionId` separation becomes operationally valuable as soon as a reconnect lab exists; it stops feeling like theoretical vocabulary.
- A chapter-first page is useful, but the real learning starts once the page also shows actual protocol traffic and trace evidence.

### What was tricky to build
- The main tricky part was preserving the architectural rule that snapshots must come before live UI fanout without inventing hidden buffering or app-specific replay tricks. The chosen implementation keeps that rule simple: load snapshot, send snapshot, then add the connection to the session subscription set.
- That sequencing does mean the current implementation is conservative: it prioritizes clarity over more advanced replay windows. That is acceptable for this phase and keeps future `sinceOrdinal` work optional rather than baked in prematurely.

### What warrants a second pair of eyes
- Whether the current conservative subscribe sequence is the right long-term default once `sinceOrdinal` becomes more than a placeholder field.
- Whether the transport hooks should remain debugging-oriented or grow into a more formal observable transport instrumentation surface.

### What should be done in the future
- Add richer reconnect semantics only if Phase 4/5 or later product needs prove them necessary.
- Consider adding explicit server-side unsubscribe-on-close reporting to later exported transcripts if reconnect debugging becomes more involved.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/transport/ws/server.go` and the tests in `server_test.go`.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase3_lab.go` and the browser page pair `static/partials/phase3.html` + `static/js/pages/phase3.js`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/transport/ws ./cmd/evtstream-systemlab -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/... -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && make evtstream-check`

### Technical details
- Focused validation commands used while finishing the phase:
  - `cd pinocchio && gofmt -w pkg/evtstream/transport/ws/*.go cmd/evtstream-systemlab/*.go`
  - `cd pinocchio && go test ./pkg/evtstream/transport/ws ./cmd/evtstream-systemlab -count=1`
- Manual API validation:
  - `curl -sf -X POST http://127.0.0.1:18105/api/phase3/run -H 'Content-Type: application/json' -d '{"action":"seed-session","sessionId":"reconnect-demo","prompt":"watch reconnect preserve a coherent snapshot"}'`
  - `curl -sf http://127.0.0.1:18105/api/phase3/state?sessionId=reconnect-demo`
- Browser validation:
  - loaded `http://127.0.0.1:18105/#phase3` in Playwright and confirmed the page renders with the live client controls and chapter content.
