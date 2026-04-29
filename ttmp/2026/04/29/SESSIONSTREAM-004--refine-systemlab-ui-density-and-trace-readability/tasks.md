# Tasks

## Intake

- [x] Create SESSIONSTREAM-004 ticket workspace.
- [x] Save user-provided Phase 1 whitespace screenshot under `sources/`.
- [x] Identify likely frontend files for trace/session widgets.
- [x] Write initial UI density refinement plan.

## Phase 1 — Trace density

- [ ] Design compact trace row layout for rendered view.
- [ ] Reduce vertical padding/gaps in `.trace-rendered` and `.trace-step`.
- [ ] Preserve readable step number, kind badge, and message hierarchy.
- [ ] Confirm long trace messages wrap cleanly without expanding rows excessively.
- [ ] Verify `[JSON]` trace view still works.

## Phase 2 — Session + UI Events density

- [ ] Replace oversized UI event cards with compact timeline/log rows.
- [ ] Render short event details inline when useful.
- [ ] Preserve icons/colors for started/appended/finished states.
- [ ] Ensure finished event text remains visible without large blank cards.
- [ ] Verify `[JSON]` session view still works.

## Phase 3 — Layout polish

- [ ] Review whether Checks panels should shrink to content height or move into a compact status strip.
- [ ] Review Snapshot rendered view for accidental regressions.
- [ ] Decide whether compact styles should apply to Phase 2–5 trace views.
- [ ] Add shared compact-list/log CSS only if it improves reuse without over-abstracting.

## Phase 4 — Validation

- [ ] Capture after screenshot for `/#phase1` and save under `sources/`.
- [ ] Compare before/after screenshots for information density.
- [ ] Smoke test phases 1–5 in browser.
- [ ] Run `make lint`.
- [ ] Run `make check`.
- [ ] Run `docmgr --root ttmp doctor --ticket SESSIONSTREAM-004 --stale-after 30`.

## Non-goals

- [ ] Do not remove rendered/json toggles.
- [ ] Do not change backend trace, session, UI event, or snapshot APIs.
- [ ] Do not start a broad component framework migration.
