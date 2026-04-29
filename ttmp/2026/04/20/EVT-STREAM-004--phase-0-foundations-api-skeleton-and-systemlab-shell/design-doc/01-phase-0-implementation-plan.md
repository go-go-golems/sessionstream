---
Title: Phase 0 Foundations, API Skeleton, and Systemlab Shell
Ticket: EVT-STREAM-004
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - backend
    - implementation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md
      Note: Source-of-truth architecture and API contract.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md
      Note: Reuse matrix and donor-code guidance.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Cross-phase implementation guide used as the parent execution plan.
ExternalSources: []
Summary: "Phase 0 creates the two-repo/two-app shape, freezes the initial API surface, and builds a minimal separate Systemlab shell that consumes only public framework seams."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 0 Foundations, API Skeleton, and Systemlab Shell, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 0 Foundations, API Skeleton, and Systemlab Shell

## Executive Summary

Create the foundation for all later work: clear package boundaries, compile-only framework interfaces, a separate Systemlab app shell, and a disciplined rule that Systemlab may only touch public framework APIs.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- Choose and scaffold the implementation home for the framework.
- Create the separate Systemlab app/repo/package shell.
- Define compile-only substrate interfaces and types.
- Document the allowed API boundaries between framework and Systemlab.
- Add minimal landing pages in Systemlab with no deep feature logic yet.

### Out of scope
- No real event bus consumption yet.
- No websocket transport yet.
- No live projections yet.
- No SQL persistence yet.

## Deliverables

- `evtstream` package skeleton with compile-only interfaces.
- `systemlab` shell app with navigation and phase pages.
- One contract note describing what the lab may call.
- CI/test commands that prove both projects compile independently.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Create the package tree for `evtstream` and keep names aligned with EVT-STREAM-002 (`SessionId`, `ConnectionId`, `Command`, `Event`, `Session`).
- Implement compile-only files: `types.go`, `handler.go`, `projection.go`, `schema.go`, `hub.go`, `transport/transport.go`, `hydration/hydration.go`.
- Keep all functions shallow or stubbed; the goal is stable names and imports, not behavior.
- Document package ownership so no webchat-specific or Systemlab-specific types leak into the substrate.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

## Systemlab Workstream

- Create a separate Systemlab app with a feature index and placeholder phase pages.
- Define a thin client layer for whatever public seam will be used later (likely HTTP/WS, optionally in-process for local dev).
- Implement a “status” page showing framework version/build info and available labs.
- Add one static essay panel explaining the architecture and the purpose of Systemlab.

### ASCII reference: Systemlab page

```text
+--------------------------------------------------------------------------------+
| Systemlab                                                                      |
+----------------------+---------------------------------------------------------+
| Labs                 | Phase 0 — Foundations                                   |
|  - Overview          |---------------------------------------------------------|
|  - Phase 0           | [Essay] Why two apps? Why strict API boundaries?       |
|  - Phase 1           |                                                         |
|  - Phase 2           | [Framework Status]                                     |
|  - Phase 3           |  - build: ok                                           |
|  - Phase 4           |  - public API packages discovered: evtstream/*         |
|  - Phase 5           |  - transport plugins: (none yet)                       |
|  - Phase 6           |                                                         |
|                      | [Boundary Contract]                                     |
|                      |  Systemlab may call: public HTTP/WS or public Hub APIs |
|                      |  Systemlab may NOT import: internal webchat internals   |
+----------------------+---------------------------------------------------------+
```

### Why this page matters

The Systemlab page for this phase is not just a demo. It is a combined:

- onboarding artifact,
- debugging surface,
- validation tool,
- and regression reference.

For this phase, the page should make it obvious:

1. what input is being exercised,
2. what internal steps happened,
3. what state changed,
4. what invariants passed or failed.

## API Boundaries to Enforce

- Systemlab must not depend on `pinocchio/pkg/webchat` internals.
- No SEM envelope types in the substrate core.
- No package-global registries copied from legacy code.
- The only shared language between framework and lab should be public API contracts.

## Testing and Validation

- `go test` or compile checks for the framework package tree.
- Independent build of Systemlab.
- Manual review that the lab imports only allowed packages.
- One screenshot/export of the Systemlab landing page for future docs.

## Risks and Open Questions

- If the framework and lab start as one app, all later API boundary discipline becomes ambiguous.
- If naming is changed after Phase 0, later labs and examples will need churny rewrites.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Create `evtstream` package root and compile-only files listed in EVT-STREAM-002.
2. Create separate `systemlab` app/package/repo shell with its own README and dev entrypoint.
3. Choose public integration seam for Systemlab (HTTP/WS preferred; in-process allowed only if it mirrors the future public API).
4. Add architecture overview page in Systemlab.
5. Add placeholder navigation for Phase 1 through Phase 6 labs.
6. Write boundary contract document inside the phase design doc and copy key rules into Systemlab README.
7. Add compile/build commands for both codebases to CI or local Make targets.
8. Verify that no code in Systemlab imports legacy webchat internals.
9. Record follow-up tasks for Phase 1 implementation handoff.

## Definition of Done

A reviewer should be able to say this phase is complete when:

- the framework capability works through the intended public seam,
- the Systemlab page can demonstrate and inspect the feature,
- automated tests cover the main invariants,
- the next phase can start without rewriting this phase's architecture,
- and at least one artifact (trace, screenshot, transcript, or exported state) shows the feature in action.

## References

- `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md`
- `le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`
- `le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md`
- Relevant donor code in `pinocchio/pkg/webchat/*` and `pinocchio/pkg/persistence/chatstore/*` as called out in EVT-STREAM-002 and EVT-STREAM-003.
