---
Title: Investigation Diary
Ticket: EVT-STREAM-008
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - chat
    - implementation
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/examples/chat/chat.go
      Note: Chat example package with schemas, command handlers, streaming runner, and projections.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/examples/chat/chat_test.go
      Note: Happy-path and stop-path tests for the example package.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase4_lab.go
      Note: Systemlab Phase 4 backend integration between the chat example, websocket transport, and teaching page.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-008 covering the reusable chat example package, its streaming and stop behavior, and the matching Systemlab chat page that exercises the example through public seams."
LastUpdated: 2026-04-20T06:42:00-04:00
WhatFor: "Record how Phase 4 was implemented, validated, and framed as both a teaching example and a reusable substrate consumer."
WhenToUse: "When reviewing the chat example, understanding where example code ends and substrate code begins, or continuing chat-oriented example work later."
---

# Diary

## Goal

Capture the implementation work for EVT-STREAM-008 so a reviewer can see how a real application-level example was added on top of `evtstream`, how start/stop chat flows are represented without leaking back into the substrate core, and how the Systemlab chat page now exercises those public seams directly.

## Step 1: Add the reusable chat example package and the Phase 4 page

This step introduced the first concrete application backend on top of the substrate. I added a dedicated `pkg/evtstream/examples/chat` package that owns chat-specific command names, backend events, UI events, timeline entities, and a simple streaming runner with cancellable generation.

I then wired the example into Systemlab rather than treating it as a unit-test-only package. The Phase 4 page now uses the websocket transport from Phase 3, submits `StartInference` / `StopInference` style actions through public hub registration, and shows backend-event, live UI, and final timeline-state evidence together.

### Prompt Context

**User prompt (verbatim):** "implement phase 3-4-5, commit at appropriate intervals, keep detailed diaries in their respective tickets."

**Assistant interpretation:** Continue the implementation plan by finishing the remaining runtime phases, committing the code, and recording detailed implementation diaries inside the matching ticket workspaces.

**Inferred user intent:** The user wants the planned framework phases to move from documentation into working code, with careful ticket bookkeeping and continuation-friendly diaries rather than only code changes.

**Commit (code):** `8941f33bb752fda7572fa3daf3a9f7315bf0a171` — `feat(evtstream): implement phases 3-5 transport persistence and chat labs`

### What I did
- Added the example package:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/examples/chat/chat.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/examples/chat/chat_test.go`
- Implemented:
  - chat command schemas for start/stop,
  - canonical backend events for started / delta / finished / stopped,
  - live UI events for started / appended / finished / stopped,
  - timeline reduction into `ChatMessage`,
  - a small cancellable runner that simulates streaming output over time.
- Added the Phase 4 Systemlab backend in:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase4_lab.go`
- Added the matching Systemlab page wiring:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase4.html`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/phase4.js`
  - plus shared `server.go`, `static/js/api.js`, `lab_environment.go`, and `lab_environment_test.go` updates.
- Validated stop/cancel coherence through the example tests and the Systemlab lab state.
- Captured a Phase 4 state artifact:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-008--phase-4-chat-example-backend-and-systemlab-integration/reference/02-phase-4-state.json`

### Why
- Phase 4 exists to prove the substrate is useful for a real application shape without letting that example redefine the substrate itself.
- Chat is a strong teaching example because it naturally exposes start, stream, finish, and stop behavior while also producing a meaningful durable timeline entity.
- The Systemlab page needed to show more than a prompt box; it needed to make backend events, live websocket UI frames, and final hydrated state observable together.

### What worked
- The chat example stayed package-local to `pkg/evtstream/examples/chat` instead of leaking chat-specific helpers into `pkg/evtstream`.
- The example tests cover both happy-path completion and the stop path.
- The Systemlab Phase 4 page exercises the public websocket transport instead of bypassing it.
- The final snapshot and the last live UI evidence agree, which means the example is teaching the right projection story instead of only producing nice-looking output.

### What didn't work
- There was no major bug in this slice, but the most important non-obvious constraint was package ownership. It was tempting to add chat-specific conveniences in the generic substrate, and I intentionally kept those inside the example package instead.

### What I learned
- The split between UI projection and timeline projection becomes much easier to explain once a streaming chat message exists; one side is about what the user watches, the other is about what the system remembers.
- A tiny cancellable runner is enough to make stop/cancel behavior meaningful in a lab without dragging in product-specific LLM runtime details.

### What was tricky to build
- The main tricky part was ensuring the example felt realistic enough to teach from while still staying clearly outside the substrate core. I treated the example package as a consumer of hub registration, projections, and the websocket transport rather than a place to sneak substrate changes through a concrete use case.
- The second tricky part was making stop behavior coherent. The runner must surface interruption as an event in the model, not as an abrupt disappearance of activity.

### What warrants a second pair of eyes
- Whether the current simulated runner API is the right long-term shape for later real model integrations, or whether it should stay intentionally toy-like to keep the example pedagogical.
- Whether the current Systemlab checks are sufficient for stopped runs, or whether a later transcript export should distinguish more clearly between completed and interrupted flows.

### What should be done in the future
- Add richer example scenarios only if they continue to clarify the substrate rather than turning the example package into a second core.
- Decide later whether a real model-backed adapter should live next to this example or remain a separate app-level concern entirely.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/examples/chat/chat.go` and `chat_test.go`.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase4_lab.go` and the page pair `static/partials/phase4.html` + `static/js/pages/phase4.js`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/examples/chat ./cmd/evtstream-systemlab -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/... -count=1`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && make evtstream-check`

### Technical details
- Focused validation commands used while finishing the phase:
  - `cd pinocchio && gofmt -w pkg/evtstream/examples/chat/*.go cmd/evtstream-systemlab/*.go`
  - `cd pinocchio && go test ./pkg/evtstream/examples/chat ./cmd/evtstream-systemlab -count=1`
- Manual API validation:
  - `curl -sf -X POST http://127.0.0.1:18105/api/phase4/run -H 'Content-Type: application/json' -d '{"action":"send","sessionId":"chat-demo","prompt":"Explain ordinals in plain language"}'`
  - `curl -sf http://127.0.0.1:18105/api/phase4/state?sessionId=chat-demo`
- Browser validation:
  - loaded `http://127.0.0.1:18105/#phase4` in Playwright and confirmed the page renders with live controls and chapter content.
