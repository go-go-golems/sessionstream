---
Title: Refine Systemlab UI density and trace readability
Ticket: SESSIONSTREAM-004
Status: active
Topics:
    - systemlab
    - frontend
    - ux
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/app.css
      Note: Global CSS for rendered trace/session widgets and panel layout
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/js/pages/phase1.js
      Note: Phase 1 rendered Trace and Session + UI Events widgets
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/partials/phase1.html
      Note: Phase 1 panel layout and rendered/json toggles
ExternalSources:
  - /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-004--refine-systemlab-ui-density-and-trace-readability/sources/phase1-whitespace-before.png
Summary: Improve Systemlab rendered trace and session/UI event widgets so Phase 1 uses vertical space efficiently while preserving readability and JSON views.
LastUpdated: 2026-04-29T17:05:00-04:00
WhatFor: Track and implement frontend/CSS refinements for Systemlab information density.
WhenToUse: Use when working on Phase 1 trace/session UI whitespace, compact rendered widgets, or Systemlab visual polish.
---

# Refine Systemlab UI density and trace readability

## Overview

SESSIONSTREAM-004 tracks a focused frontend refinement for Systemlab. The immediate problem is visible on `/#phase1`: rendered Trace and Session + UI Events panels waste significant vertical space for short rows and small UI events.

The goal is not a full redesign. The goal is to make the existing teaching UI denser and easier to scan while keeping the dark theme, rendered/json toggles, and backend payload contracts intact.

## Evidence

- Before screenshot: [sources/phase1-whitespace-before.png](./sources/phase1-whitespace-before.png)
- Route observed by user: `http://localhost:8091/#phase1`

## Primary files

- `cmd/sessionstream-systemlab/static/app.css`
- `cmd/sessionstream-systemlab/static/js/pages/phase1.js`
- `cmd/sessionstream-systemlab/static/partials/phase1.html`

## Design plan

See [design/01-systemlab-ui-density-refinement-plan.md](./design/01-systemlab-ui-density-refinement-plan.md).

## Diary

See [reference/01-investigation-diary.md](./reference/01-investigation-diary.md).

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).

## Status

Current status: **active**.

Initial ticket setup is complete. Implementation remains TODO.
