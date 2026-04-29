---
Title: Colleague Brief — Rewrite Systemlab Chapters and Redesign Interactive Phase Pages
Ticket: EVT-STREAM-009
Status: active
Topics:
    - documentation
    - systemlab
    - frontend
    - teaching
    - onboarding
DocType: reference
Intent: medium-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md
      Note: Source-of-truth clean-room architecture.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Cross-phase implementation and teaching intent.
    - Path: pinocchio/cmd/evtstream-systemlab/README.md
      Note: Current Systemlab structure and chapter serving layout.
ExternalSources: []
Summary: "Short handoff brief for a colleague tasked with rewriting the Systemlab chapters and redesigning the interactive teaching pages for phases 3, 4, and 5."
LastUpdated: 2026-04-20T06:48:00-04:00
WhatFor: "Provide focused reading order, constraints, and an implementation approach for chapter and interactive-page redesign work."
WhenToUse: "When handing off Systemlab pedagogy and UI/UX cleanup to another contributor."
---

# Brief: Rewrite Systemlab Chapters and Redesign Interactive Phase Pages

## Goal

Improve the **Systemlab teaching experience** for phases 3, 4, and 5 by:

- rewriting the chapter markdown in a stronger teaching style,
- redesigning the interactive page layouts and flows,
- and making the pages clearer, more consistent, and more intern-friendly.

This is a **presentation / pedagogy / UX cleanup task**, not an architecture redesign task.

---

## What to read first

### Core architecture and implementation intent
Read these first, in order:

1. `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md`
2. `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`
3. `le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md`

### Phase-specific design docs
4. `le-chat/ttmp/2026/04/20/EVT-STREAM-007--phase-3-websocket-transport-and-hydration-reconnect-lab/design-doc/01-phase-3-implementation-plan.md`
5. `le-chat/ttmp/2026/04/20/EVT-STREAM-008--phase-4-chat-example-backend-and-systemlab-integration/design-doc/01-phase-4-implementation-plan.md`
6. `le-chat/ttmp/2026/04/20/EVT-STREAM-009--phase-5-sql-hydration-store-and-restart-correctness/design-doc/01-phase-5-implementation-plan.md`

### Current chapter content
7. `pinocchio/cmd/evtstream-systemlab/chapters/phase-0-foundations.md`
8. `pinocchio/cmd/evtstream-systemlab/chapters/phase-1-command-to-projection.md`
9. `pinocchio/cmd/evtstream-systemlab/chapters/phase-2-ordering-and-ordinals.md`
10. `pinocchio/cmd/evtstream-systemlab/chapters/phase-3-hydration-and-reconnect.md`
11. `pinocchio/cmd/evtstream-systemlab/chapters/phase-4-chat-example.md`
12. `pinocchio/cmd/evtstream-systemlab/chapters/phase-5-persistence-and-restart.md`

### Current Systemlab page structure
13. `pinocchio/cmd/evtstream-systemlab/README.md`
14. `pinocchio/cmd/evtstream-systemlab/server.go`
15. `pinocchio/cmd/evtstream-systemlab/chapter_api.go`

### Current frontend files
16. `pinocchio/cmd/evtstream-systemlab/static/index.html`
17. `pinocchio/cmd/evtstream-systemlab/static/app.css`
18. `pinocchio/cmd/evtstream-systemlab/static/partials/phase3.html`
19. `pinocchio/cmd/evtstream-systemlab/static/partials/phase4.html`
20. `pinocchio/cmd/evtstream-systemlab/static/partials/phase5.html`
21. `pinocchio/cmd/evtstream-systemlab/static/js/pages/phase3.js`
22. `pinocchio/cmd/evtstream-systemlab/static/js/pages/phase4.js`
23. `pinocchio/cmd/evtstream-systemlab/static/js/pages/phase5.js`
24. `pinocchio/cmd/evtstream-systemlab/static/js/api.js`
25. `pinocchio/cmd/evtstream-systemlab/static/js/dom.js`
26. `pinocchio/cmd/evtstream-systemlab/static/js/main.js`

### Actual code the pages are teaching
27. `pinocchio/pkg/evtstream/transport/ws/server.go`
28. `pinocchio/pkg/evtstream/examples/chat/chat.go`
29. `pinocchio/pkg/evtstream/hydration/sqlite/store.go`
30. `pinocchio/cmd/evtstream-systemlab/phase3_lab.go`
31. `pinocchio/cmd/evtstream-systemlab/phase4_lab.go`
32. `pinocchio/cmd/evtstream-systemlab/phase5_lab.go`

---

## Constraints

Please preserve these architectural boundaries:

- **Systemlab must remain a separate app** using only public `evtstream` seams.
- **Do not couple anything to `pkg/webchat` internals.**
- **Chapter content must remain editable markdown files** under:
  - `pinocchio/cmd/evtstream-systemlab/chapters/`
- **Chapters should continue to render above the interactive controls.**
- **Frontend must remain modular**:
  - HTML in `static/partials/`
  - page logic in `static/js/pages/`
  - shared helpers in shared JS files
- Avoid turning the pages into fake demos; they should remain **real runtime exercisers**.
- Do not change the clean-room architectural direction of `evtstream`.

---

## What “good” should look like

### For the chapters
The writing should be:

- intern-friendly,
- prose-driven,
- concrete,
- explanatory,
- and readable like a small textbook chapter rather than terse notes.

Each chapter should help the reader understand:

- why the phase exists,
- what architectural problem it solves,
- what invariants matter,
- what the page demonstrates,
- what to try,
- what to expect,
- and what files/APIs to read.

Good ingredients:
- prose first, bullets second,
- small diagrams,
- pseudocode where useful,
- explicit references to real files/symbols,
- “things to try” sections,
- “what to pay attention to” sections.

### For the interactive pages
Each page should do two things well:

1. **Teach the concept before interaction**
2. **Show evidence after interaction**

A good page should make it easy to answer:
- What did I do?
- What happened internally?
- What state changed?
- Which invariant passed or failed?
- Why does that matter?

---

## Phase-specific design direction

### Phase 3 — Hydration and Reconnect
Focus on:
- client lifecycle,
- session subscription,
- snapshot-before-live sequencing,
- reconnect behavior,
- comparison between Client A and Client B,
- backend truth vs client-local experience.

The page should make reconnect behavior easy to reason about.

### Phase 4 — Chat Example
Focus on:
- prompt / send / stop flow,
- backend events vs UI events vs timeline entity,
- live streaming behavior,
- interrupted/stopped behavior,
- making the example feel real without losing architectural clarity.

The page should clearly show how chat is built **on top of** `evtstream`, not baked into it.

### Phase 5 — Persistence and Restart
Focus on:
- memory vs SQL mode,
- pre-restart vs post-restart comparison,
- cursor preservation,
- entity preservation,
- resume-without-gaps behavior,
- reconnect after restart.

The page should make durability visible, not abstract.

---

## Recommended approach

### Step 1
Read the architecture docs and phase plans.

### Step 2
Read the current chapters and the current phase page implementations.

### Step 3
For each of phases 3, 4, and 5, write a short working note:
- what the phase is trying to teach,
- what is currently confusing or weak,
- what runtime evidence the page should foreground,
- what a new intern should understand after using the page.

### Step 4
Sketch the redesigned page structure before editing code:
- chapter structure,
- controls,
- evidence panels,
- checks/invariants,
- guided interaction flow.

### Step 5
Rewrite the chapter markdown.

### Step 6
Refactor the page partial and page JS to match the improved teaching structure.

### Step 7
Validate by using the page like a new learner:
- Is the lesson clear?
- Are the controls aligned with the lesson?
- Do the evidence panels actually prove the architectural point?
- Can the page be read and used without already knowing the codebase deeply?

---

## Things to avoid

Please avoid:

- redesigning the architecture itself,
- moving example-specific logic into `pkg/evtstream`,
- over-optimizing for visual polish at the expense of clarity,
- collapsing the modular frontend into large monolithic files,
- replacing real evidence with fake demo output,
- writing chapters that become only API reference and lose the teaching voice.

---

## Definition of done

A page/chapter rewrite is successful when:

- the chapter is easier and more enjoyable to read,
- the interactive flow matches the lesson,
- the evidence panels show the real mechanics clearly,
- the checks make sense to a learner,
- and a new intern could use the page to understand the phase without a guided walkthrough.

---

## Deliverable expectation

Please update:

- the phase chapter markdown,
- the corresponding phase partial HTML,
- the corresponding phase page JS,
- and any minimal shared frontend helpers needed to support the improved teaching structure.

Please keep changes focused on:
- pedagogy,
- clarity,
- structure,
- interaction design,
- and consistency.
