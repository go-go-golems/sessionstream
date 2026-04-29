---
Title: Investigation Diary
Ticket: EVT-STREAM-009
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - persistence
    - sqlite
    - implementation
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hydration/sqlite/store.go
      Note: SQL hydration-store implementation with transactional apply plus cursor advancement.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hydration/sqlite/store_test.go
      Note: Persistence and reopen coverage for the SQL store.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase5_lab.go
      Note: Memory-vs-SQL restart lab, runtime rebuild logic, and restart-oriented checks.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-009 covering the SQLite hydration store, restart/resume behavior, and the Systemlab page that compares memory and SQL persistence modes."
LastUpdated: 2026-04-20T06:44:00-04:00
WhatFor: "Record how Phase 5 was implemented, validated, and shaped into a restart-correctness teaching lab."
WhenToUse: "When reviewing persistence semantics, understanding restart behavior, or continuing durable hydration-store work later."
---

# Diary

## Goal

Capture the implementation work for EVT-STREAM-009 so a reviewer can follow how the hydration store moved beyond memory-only semantics, how the SQL-backed store preserves both entity state and cursor state, and how the Systemlab page compares restart behavior across memory and SQL modes.

## Step 1: Add the SQLite hydration store and the restart-correctness lab

This step made persistence real. I added a SQLite-backed `HydrationStore` implementation that stores per-session cursor state and timeline entities transactionally, then wired a Systemlab lab that can switch between memory and SQL modes, restart the backend runtime, reconnect a websocket client, and compare pre/post restart state.

The key design constraint was keeping the restart path honest. The same event-consumer path still assigns ordinals, and on restart the SQL store becomes the source of truth for the next cursor value. That lets the lab prove resume-without-gaps behavior instead of simply persisting a snapshot for display.

### Prompt Context

**User prompt (verbatim):** "implement phase 3-4-5, commit at appropriate intervals, keep detailed diaries in their respective tickets."

**Assistant interpretation:** Continue the implementation plan by finishing the remaining runtime phases, committing the code, and recording detailed implementation diaries inside the matching ticket workspaces.

**Inferred user intent:** The user wants the planned framework phases to move from documentation into working code, with careful ticket bookkeeping and continuation-friendly diaries rather than only code changes.

**Commit (code):** `8941f33bb752fda7572fa3daf3a9f7315bf0a171` — `feat(evtstream): implement phases 3-5 transport persistence and chat labs`

### What I did
- Added the SQLite hydration store:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hydration/sqlite/store.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hydration/sqlite/store_test.go`
- Implemented:
  - transactional `Apply(...)` with cursor advancement,
  - snapshot reconstruction from stored timeline entities,
  - `View(...)` and `Cursor(...)` semantics,
  - store reset helper for dev/test scenarios,
  - file-DSN helper for SQLite.
- Added the Phase 5 Systemlab backend in:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase5_lab.go`
- Built a restart lab that:
  - switches between memory and SQL modes,
  - seeds session activity through the regular event path,
  - restarts the phase runtime,
  - reconnects through the websocket transport,
  - compares pre/post restart snapshots and cursor behavior.
- Extended Systemlab page wiring with:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase5.html`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/phase5.js`
  - plus shared `server.go`, `static/js/api.js`, `lab_environment.go`, and `lab_environment_test.go` updates.
- Captured a Phase 5 restart artifact:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-009--phase-5-sql-hydration-store-and-restart-correctness/reference/02-phase-5-restart-state.json`

### Why
- Phase 5 exists to prove that hydration state and cursor state survive restart together rather than drifting apart.
- A SQL store that merely remembers entity blobs would not be enough; the consumer also needs durable cursor state so post-restart ordinals continue coherently.
- The Systemlab page needed to compare memory and SQL modes in one place so the restart story becomes visible instead of buried in tests alone.

### What worked
- The SQLite store persists timeline entities and cursor state and reconstructs snapshots correctly after reopening.
- The Phase 5 tests cover store persistence, reset behavior, and runtime restart/resume behavior.
- The restart lab shows the intended distinction between memory mode and SQL mode: memory forgets state, SQL preserves it.
- In SQL mode, post-restart checks confirm cursor preservation, entity preservation, and resume-without-gaps behavior.

### What didn't work
- My first SQL runtime builder reused a fixed temp-path name, which meant state could leak across separate manual server runs and make the lab look less deterministic than intended. I corrected that by creating a fresh temp database path whenever a new SQL runtime is created without an existing restart path to reuse.

### What I learned
- Persistence semantics are easier to keep honest when the store implementation is forced to obey the same `HydrationStore` contract as memory mode rather than adding SQL-specific shortcuts.
- Restart correctness becomes much easier to explain once the lab shows both the pre-restart and post-restart snapshots side by side.

### What was tricky to build
- The main tricky part was restart scope. The HTTP server itself keeps running while the lab runtime is rebuilt, so the lab had to simulate backend restart by tearing down and recreating the phase-local hub, store, bus, and websocket transport while keeping the rest of Systemlab alive.
- The second tricky part was making SQL-mode restarts deterministic without accidentally persisting state across unrelated manual runs. The fix was to reuse the database only for an intentional restart path and otherwise allocate a fresh temp file.

### What warrants a second pair of eyes
- Whether the SQLite schema is the right long-term foundation for later production-like use, or whether it should remain lab-oriented and intentionally simple.
- Whether the current restart lab should later expose more explicit client-side reconnection transcripts instead of relying mainly on backend comparison output plus websocket frames.

### What should be done in the future
- Consider adding a later transcript export for Phase 5 once restart walkthroughs need to be embedded in longer-form docs.
- Revisit whether SQL migrations need a more explicit versioning surface before Phase 6 migration/regression work begins.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/hydration/sqlite/store.go` and `store_test.go`.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase5_lab.go` and the page pair `static/partials/phase5.html` + `static/js/pages/phase5.js`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/hydration/sqlite ./cmd/evtstream-systemlab -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/... -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && make evtstream-check`

### Technical details
- Focused validation commands used while finishing the phase:
  - `cd pinocchio && gofmt -w pkg/evtstream/hydration/sqlite/*.go cmd/evtstream-systemlab/*.go`
  - `cd pinocchio && go test ./pkg/evtstream/hydration/sqlite ./cmd/evtstream-systemlab -count=1`
- Manual API validation:
  - `curl -sf -X POST http://127.0.0.1:18105/api/phase5/run -H 'Content-Type: application/json' -d '{"action":"seed-session","mode":"sql","sessionId":"persist-demo","text":"persist this record"}'`
  - `curl -sf -X POST http://127.0.0.1:18105/api/phase5/run -H 'Content-Type: application/json' -d '{"action":"restart-backend","mode":"sql","sessionId":"persist-demo","text":"persist this record"}'`
- Browser validation:
  - loaded `http://127.0.0.1:18105/#phase5` in Playwright and confirmed the page renders with the storage-mode selector, restart controls, and chapter content.
