# Tasks

## Intake

- [x] Create SESSIONSTREAM-004 ticket workspace.
- [x] Save user-provided Phase 1 whitespace screenshot under `sources/`.
- [x] Identify likely frontend files for trace/session widgets.
- [x] Write initial UI density refinement plan.

## Phase 1 — Trace density

- [x] Design compact trace row layout for rendered view.
- [x] Reduce vertical padding/gaps in `.trace-rendered` and `.trace-step`.
- [x] Preserve readable step number, kind badge, and message hierarchy.
- [x] Confirm long trace messages wrap cleanly without expanding rows excessively.
- [x] Verify `[JSON]` trace view still works.

## Phase 2 — Session + UI Events density

- [x] Replace oversized UI event cards with compact timeline/log rows.
- [x] Render short event details inline when useful.
- [x] Preserve icons/colors for started/appended/finished states.
- [x] Ensure finished event text remains visible without large blank cards.
- [x] Verify `[JSON]` session view still works.

## Phase 3 — Layout polish

- [x] Review whether Checks panels should shrink to content height or move into a compact status strip.
- [x] Review Snapshot rendered view for accidental regressions.
- [x] Decide whether compact styles should apply to Phase 2–5 trace views.
- [x] Add shared compact-list/log CSS only if it improves reuse without over-abstracting.

## Phase 4 — Validation

- [x] Capture after screenshot for `/#phase1` and save under `sources/`.
- [x] Compare before/after screenshots for information density.
- [ ] Smoke test phases 1–5 in browser.
- [x] Render Phase 2 bus/consumer trace as compact rows instead of raw JSON.
- [x] Render Phase 2 message history as a compact table instead of raw JSON.
- [x] Render Phase 2 per-session ordinals as compact chips/table instead of raw JSON.
- [x] Render Phase 2 snapshots as compact cards/tables instead of raw JSON.
- [x] Run `make lint`.
- [x] Run `make check`.
- [ ] Run `docmgr --root ttmp doctor --ticket SESSIONSTREAM-004 --stale-after 30`.

## Non-goals

- [ ] Do not remove rendered/json toggles.
- [ ] Do not change backend trace, session, UI event, or snapshot APIs.
- [ ] Do not start a broad component framework migration.
