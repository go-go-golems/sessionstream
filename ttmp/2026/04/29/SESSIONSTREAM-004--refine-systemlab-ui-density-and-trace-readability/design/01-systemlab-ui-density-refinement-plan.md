---
Title: Systemlab UI density refinement plan
Ticket: SESSIONSTREAM-004
Status: draft
Topics:
    - systemlab
    - frontend
    - ux
    - css
DocType: design
Intent: implementation-plan
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/static/app.css
      Note: |-
        Global Systemlab CSS including trace/session rendered panels
        Global CSS for rendered trace/session widgets and panel layout
    - Path: cmd/sessionstream-systemlab/static/js/pages/phase1.js
      Note: |-
        Phase 1 trace/session/snapshot renderers that currently produce sparse cards
        Phase 1 rendered trace/session/snapshot widgets
    - Path: cmd/sessionstream-systemlab/static/partials/phase1.html
      Note: Phase 1 panel layout and rendered/json toggles
ExternalSources:
    - /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-004--refine-systemlab-ui-density-and-trace-readability/sources/phase1-whitespace-before.png
Summary: Refine Systemlab rendered trace/event widgets to use vertical space efficiently while preserving readability and JSON escape hatches.
LastUpdated: 2026-04-29T17:05:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Systemlab UI Density Refinement Plan

## Problem statement

The Phase 1 Systemlab page wastes too much vertical space in its rendered trace and session/UI event widgets. In the provided screenshot, the Trace panel displays ten short entries across a very tall black area, with each row consuming far more height than its content needs. The Session + UI Events panel has the same issue: each UI event is rendered as a large card with substantial empty vertical space.

This makes the teaching UI harder to scan. Users should be able to see more of the causal chain at once: command, handler, UI projection, timeline projection, and resulting UI events.

## Evidence

Source screenshot:

- `sources/phase1-whitespace-before.png`
- Original Path: `/tmp/pi-clipboard-1a3bb7fa-8f10-4bf5-a514-733b6f5170b3.png`
- Observed route: `http://localhost:8091/#phase1`

Visible issues:

1. **Trace rows are too tall.** The rows themselves are simple: step number, kind badge, message. They should fit in compact rows.
2. **Trace panel uses a large dark block with sparse content.** Useful information is separated by excessive vertical gaps.
3. **Session + UI Events cards are oversized.** `Lab Message Started` and `Lab Message Appended` have almost no details but consume large cards.
4. **Finished event details are pushed far down.** The important final text appears in a large card instead of an inline or compact detail row.
5. **Checks panel has unused height.** This is secondary but contributes to the sense that Phase 1 uses large fixed areas rather than content-driven layout.

## Relevant code

Likely primary files:

- `cmd/sessionstream-systemlab/static/app.css`
  - `.trace-rendered`
  - `.trace-step`
  - `.trace-step-num`
  - `.trace-step-kind`
  - `.trace-step-message`
  - `.session-rendered`
  - `.session-header`
  - `.ui-events-list`
  - `.ui-event-item`
  - `.ui-event-icon`
  - `.ui-event-name`
  - `.ui-event-detail`

- `cmd/sessionstream-systemlab/static/js/pages/phase1.js`
  - `renderTraceRendered`
  - `renderSessionRendered`
  - `formatEventName`
  - `formatEventDetail`

- `cmd/sessionstream-systemlab/static/partials/phase1.html`
  - rendered/json toggles
  - Trace panel
  - Session + UI Events panel

## Design goals

1. **Compact scanability.** The user should see the full Phase 1 trace and all UI events without scrolling through empty card space.
2. **Preserve teaching clarity.** Compact does not mean cryptic; causal steps, event kind, ordinal/status, and text should remain clear.
3. **Keep JSON escape hatch.** The existing `[Rendered]` / `[JSON]` toggles should remain available.
4. **Prefer content-driven height.** Avoid fixed/min heights for rendered event cards unless needed for a specific visual reason.
5. **Use consistent density across phases.** Phase 1 can be the first pass, but the shared CSS should not make Phase 2–5 worse.
6. **Accessible contrast and hit targets.** Reduce whitespace while preserving readable typography and sufficient contrast.

## Proposed UI changes

### Trace panel

Change rendered trace from sparse rows to a compact event log:

- Reduce row padding from large vertical spacing to approximately `4px 0` or `6px 8px`.
- Align columns with CSS grid:
  - step number column;
  - kind badge column;
  - message column;
  - optional details column in the future.
- Use a tighter line-height, e.g. `1.25` to `1.35`.
- Make kind badges compact but readable.
- Keep subtle separators, but avoid large black gutters.

Potential shape:

```text
01  command              submit command
02  handler              handler invoked
03  ui-projection        ui projection emitted event
04  timeline-projection  timeline projection upserted entity
```

### Session + UI Events panel

Change UI event rendering from oversized cards to compact timeline rows:

- Render as a vertical list with small icons and compact content.
- Remove fixed/min-height card behavior if present through inherited styles.
- Use inline details for small payloads:
  - `Lab Message Appended — "hello"`
  - `Lab Message Finished — "hello from systemlab"`
- Keep event icons, but avoid dedicating a full card row to them.
- Use a `compact-list` or `event-log` class so future widgets can opt in.

Potential shape:

```text
● Lab Message Started
→ Lab Message Appended  "hello"
→ Lab Message Appended  " from systemlab"
✓ Lab Message Finished  "hello from systemlab"
```

### Panel layout

Optional improvements after the row/card fixes:

- Let the Checks panel shrink to content height rather than visually occupying a large blank area.
- Consider a denser `.grid.compact` variant for Phase 1.
- Consider making Trace and Session panels scroll independently only after a reasonable max height, e.g. `max-height: 60vh`.

## Acceptance criteria

- On `/#phase1`, after running the default prompt, the Trace panel displays all ten trace rows in a compact block without large vertical gaps.
- The Session + UI Events panel displays all events in compact rows, not large empty cards.
- `[JSON]` view still works for Trace, Session + UI Events, and Snapshot.
- Rendered Snapshot remains readable and is not regressed by shared CSS changes.
- Phase 2–5 smoke views still render without obvious layout breakage.
- `make lint` and `make check` pass.
- A before/after screenshot is saved under this ticket's `sources/` directory.

## Suggested implementation steps

1. Add a shared compact log CSS pattern to `app.css`.
2. Update Phase 1 trace renderer to emit the compact log structure.
3. Update Phase 1 session/UI event renderer to emit compact timeline rows.
4. Review rendered Snapshot for accidental regressions.
5. Manually smoke test phases 1–5.
6. Capture after screenshot and attach it to this ticket.
7. Update the diary/changelog with exact CSS/JS changes and validation commands.

## Non-goals

- Do not redesign the full Systemlab visual language in this ticket.
- Do not remove JSON views.
- Do not change backend APIs or trace payload semantics.
- Do not turn this into a component framework migration.

## Open questions

1. Should compact trace/event rendering be shared across Phase 2–5 immediately, or start with Phase 1 only?
2. Should trace rows expose details inline or through expandable rows later?
3. Should dense mode be the default, or should there be a user-facing density toggle?
4. Should Checks panels be restyled as compact header/status bars?
