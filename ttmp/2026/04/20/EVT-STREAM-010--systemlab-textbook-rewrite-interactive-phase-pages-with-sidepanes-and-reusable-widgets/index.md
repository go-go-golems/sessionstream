---
Title: 'Systemlab Textbook Rewrite: Interactive Phase Pages with Sidepanes and Reusable Widgets'
Ticket: EVT-STREAM-010
Status: active
Topics:
    - documentation
    - systemlab
    - frontend
    - teaching
    - onboarding
    - widgets
    - react
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase1.html
      Note: Phase 1 current layout (vertical stack)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase3.html
      Note: Phase 3 current layout (horizontal grid for clients)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-010--systemlab-textbook-rewrite-interactive-phase-pages-with-sidepanes-and-reusable-widgets/design/01-systemlab-textbook-design-guide.md
      Note: Design guide with ASCII wireframes and widget taxonomy
ExternalSources: []
Summary: "Rewrite Systemlab phase pages to minimize scrolling. Interactive controls should be close to related text, or in a sidepane. Add ToggledViewer widget with JSON/Rendered toggle for better interpretability. Migrate to React widgets."
LastUpdated: 2026-04-20T09:35:00-04:00
WhatFor: "Improve Systemlab teaching experience by redesigning phase pages with sidepanes, inline controls, and reusable React widgets."
WhenToUse: "When implementing the Phase 1-5 page rewrites and planning the React widget migration."
---

# Systemlab Textbook Rewrite: Interactive Phase Pages with Sidepanes and Reusable Widgets

## Overview

Rewrite Systemlab phase pages to minimize scrolling. Interactive controls should be close to related text, or in a sidepane. Add **ToggledViewer** widget with JSON/Rendered toggle for better interpretability. Migrate to React widgets.

## Key Links

- **Design Guide**: [design/01-systemlab-textbook-design-guide.md](./design/01-systemlab-textbook-design-guide.md)
- **Source Brief**: [EVT-STREAM-009 reference/03-colleague-brief-systemlab-chapter-and-ui-rewrite.md](../EVT-STREAM-009--phase-5-sql-hydration-store-and-restart-correctness/reference/03-colleague-brief-systemlab-chapter-and-ui-rewrite.md)

## Status

Current status: **active** - Design validated against live Systemlab

## Validation Results (Live Testing)

| Phase | Layout | Issue |
|-------|--------|-------|
| Phase 1 | JSON well-formatted, syntax highlighted | Need rendered view for teaching |
| Phase 3 | Client A/B side-by-side ✓ | Add state indicators, JSON/Rendered toggle |
| Phase 5 | Controls 6+ viewport heights away | Must inline controls in chapter |

## Key Design Decisions

### 1. Minimize Scrolling
- Inline controls in chapter "Try" sections
- Collapsible evidence panel (sidepane)
- Controls close to related teaching text

### 2. Widget Dual Purpose: Illustrate AND Experiment

**Targeted widgets** illustrate specific textbook concepts:
- Tied to specific prose sections and chapter claims
- Validate invariants and show evidence of behaviors as explained
- Placed near the text they illustrate
- Examples: TraceTimeline (validates trace explanation), CheckList (validates invariants)

**Freeform widgets** enable experimentation beyond the text:
- Not tied to specific sections; surface behavior not covered (or covered later)
- Give readers room to explore and discover
- Live in the sidepane, accessible but not competing with textbook flow
- Examples: EvidenceLog (full event capture), StoreInspector (raw hydration state)

Both are necessary. Targeted widgets ensure the textbook can be trusted. Freeform widgets ensure readers can go beyond what was written.

## Next Steps

1. Define ToggledViewer API - how JSON transforms to rendered view
2. Define widget API contracts as TypeScript interfaces
3. Create React project scaffold with Vite
4. Implement Phase 1 as first migrated page
