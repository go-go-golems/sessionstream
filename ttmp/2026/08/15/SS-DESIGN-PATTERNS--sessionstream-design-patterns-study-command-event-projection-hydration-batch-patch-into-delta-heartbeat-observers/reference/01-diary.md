---
Title: Diary
Ticket: SS-DESIGN-PATTERNS
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - sessionstream
    - onboarding
DocType: reference
Intent: long-term
Summary: "Diary for the sessionstream design patterns study."
LastUpdated: 2026-08-15T00:00:00-04:00
WhatFor: "Records the in-depth study of sessionstream design patterns."
WhenToUse: "Read before resuming SS-DESIGN-PATTERNS work."
---

# Diary

## Goal

Produce an intern-ready, in-depth study of the design patterns in the
`sessionstream` framework, with the batch-patch-into-delta pattern included,
and store it in a docmgr ticket.

## Step 1: In-depth code study, then write the patterns study

The user asked for the same kind of study as the MZ1-004 intern guide, but for
`~/code/wesen/go-go-golems/sessionstream`, with the "batch patch into delta"
pattern included, and an overview of findings first. I read the framework
end-to-end (core substrate, hydration, transport, heartbeat, observers,
schema-vet) and the EVT-STREAM-013 design doc, synthesized ten recurring
patterns, created a docmgr ticket, and wrote a 16-section design doc.

### Prompt Context

**User prompt (verbatim):** "Do the same kind of study of design patterns (first giving me an overview of what you find after in depth study) of ~/code/wesen/go-go-golems/sessionstream, and i want the batch patch into delta pattern in there"

**Assistant interpretation:** Study the sessionstream framework's design patterns in depth, give an overview of findings, and write an intern-style design-patterns study that explicitly includes the batch-patch-into-delta pattern.

**Inferred user intent:** A reusable, onboarding-grade catalog of the framework's design patterns, grounded in the actual code, with the streaming delta/preview/commit pattern called out.

### What I did
- Read AGENT.md (framework-purpose guardrails), the README (the full model),
  and the repo layout; mapped every Go file under pkg/cmd/examples/proto.
- Read the core substrate: `types.go`, `handler.go`, `projection.go`, `hub.go`
  (incl. `projectAndApply`, `RebuildTimeline`, projection error policies),
  `schema.go`, `command_registry.go`, `session_registry.go`, `ordinals.go`,
  `bus.go`, `consumer.go`, `fanout.go`, `hydration.go`.
- Read the SQLite store fully (`hydration/sqlite/store.go`) — the append-log +
  materialized-view + version-history + per-projector-cursor realization.
- Read the chatdemo (`examples/chatdemo/chat.go` + `chat.proto` + `chat_test.go`)
  — the canonical batch-patch-into-delta reference (`TokensDeltaEvent` carries
  `chunk` + accumulated `text`; timeline projection upserts the *whole*
  `ChatMessage` each tick).
- Read the EVT-STREAM-013 design doc — the authoritative preview-vs-commit
  discipline (preview live-only, commit durable).
- Read the heartbeat pure state machine (`transport/ws/internal/heartbeat/machine.go`)
  + supervisor (`heartbeat.go`) + observer dispatcher (`observer.go`).
- Ran `docmgr status`/`vocab list`/`ticket list` to learn the repo's docmgr
  conventions (27 tickets, rich vocabulary, ttmp root = `ttmp/`).
- Created ticket `SS-DESIGN-PATTERNS`, added a design doc, wrote 42 KB / 16
  sections covering 10 patterns with prose, bullets, pseudocode, ASCII
  diagrams, API references, and absolute file references; related it to 17
  files; updated the changelog; `docmgr doctor` ✅ all checks passed.

### Why
A patterns vocabulary is the fastest onboarding path for a framework whose
value is in its seams (command/event/projection, UI vs timeline, preview vs
commit, snapshot vs live). The user asked for the overview first and the
delta pattern explicitly, so the doc foregrounds Pattern 2.

### What worked
- The code reads confirmed every pattern with exact file/symbol anchors; the
  chatdemo is an unusually clean reference for the delta pattern (chunk +
  accumulated text in one event, whole-state upsert in the projection).
- EVT-STREAM-013 already documents the preview/commit discipline in depth, so
  Pattern 2 could cite it rather than reinvent it.
- docmgr doctor passed from the repo root on the first try (unlike MZ1-004,
  this repo's ttmp root IS the repo root, so `repo://` paths resolve cleanly).

### What didn't work
- Nothing failed. The repo is well-documented; the main effort was synthesis,
  not discovery.

### What I learned
- The "batch patch into delta" name is the user's phrasing for two coupled
  ideas the code expresses separately: (a) the delta event shape (chunk +
  accumulated state on the wire) and (b) the timeline projection patching the
  durable entity with the *whole* accumulated state each tick (a batch upsert,
  not a patch log). Together they're what make snapshot-before-live work for
  streaming content.
- The heartbeat being a *pure* state machine + a supervisor is a deliberate
  testability pattern (the recent commits 957c906/229a47e/4754f78 hardened
  exactly that seam), worth naming as its own pattern.
- Schema-vet is the framework making "contracts stay typed" a *build-time*
  guarantee via a Go analyzer — a pattern distinct from "registration is data".

### What was tricky to build
- Naming the patterns at the right altitude: too generic ("event sourcing")
  loses the framework-specific value; too specific ("the chatdemo") loses
  reusability. Settled on 10 patterns, each tied to concrete files and to a
  "why it exists" rationale.

### What warrants a second pair of eyes
- The Pattern 2 claim that "the timeline projection upserts the whole entity
  each tick" — verified in `chatdemo/chat.go` `timelineProjection`, but a
  reviewer should confirm the SQLite `Apply` upsert semantics match (it does:
  `ON CONFLICT DO UPDATE` keyed by `(session, kind, entity_id)`).
- The cross-cutting Section 13 walkthrough — every pattern fires in one
  request; a reviewer should sanity-check the ordering.

### What should be done in the future
- Add a short "how to add a new event family" playbook that uses these
  patterns in order (register schema -> handler -> UI projection -> timeline
  projection -> snapshot test), with the preview/commit decision tree.
- Consider reMarkable upload if the user wants it offline (not requested this
  time).

### Code review instructions
- Read `design-doc/01-sessionstream-design-patterns-intern-study.md`;
  spot-check Pattern 2 (Section 4) against `examples/chatdemo/chat.go` and
  Pattern 4 (Section 6) against `hydration/sqlite/store.go`.
- Validate: `docmgr doctor --ticket SS-DESIGN-PATTERNS --stale-after 30` → all
  checks passed.

### Technical details
- Ticket: `SS-DESIGN-PATTERNS` at `ttmp/2026/08/15/SS-DESIGN-PATTERNS--...`.
- Design doc: `design-doc/01-sessionstream-design-patterns-intern-study.md` (42,860 bytes, 16 sections).
- 17 file relations; doctor clean.
- Repo HEAD: `d62dca9` (branch `task/experiment-verification`).
