---
Title: Phase 1 In-Memory Core and Command-to-Projection Lab
Ticket: EVT-STREAM-005
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
Summary: "Phase 1 delivers the first executable substrate path in memory: synchronous command dispatch, session creation, projection execution, and hydration snapshots. Systemlab gets the first true interactive lab: Command → Event → Projection."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 1 In-Memory Core and Command-to-Projection Lab, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 1 In-Memory Core and Command-to-Projection Lab

## Executive Summary

Make the framework do one real thing end to end, without websockets or distributed transport yet. The phase should prove that a command can create backend events, run projections, and update durable-ish in-memory state.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- Session registry and lazy session creation.
- Command registry and dispatch.
- In-memory hydration store.
- In-memory execution path for `Hub.Submit`.
- Systemlab Lab 01 with real traces and state inspection.

### Out of scope
- No Watermill backend yet.
- No websocket subscription flow yet.
- No SQL store yet.

## Deliverables

- Real `Hub.Submit(...)` path.
- One example command handler and projections running against real substrate logic.
- Systemlab page that shows before/after command execution, event trace, UI projection output, and timeline snapshot.
- Exportable transcript for lab runs.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Implement `SessionRegistry` and `CommandRegistry` as testable internal helpers.
- Add in-memory hydration store with `Apply`, `Snapshot`, `View`, and `Cursor` semantics.
- Wire `Hub.Submit(...)` through the same dispatch code later transports will use.
- Add a tiny test backend/example solely to prove the flow.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

### Phase 2 seam replacement map

Phase 1 should be written so Phase 2 can swap transport and event delivery without rewriting the public API. The intended replacements are:

- `Hub.Submit(...)` keeps its public shape, but the current in-process `localEventPublisher` path will be replaced by publish/consume flow backed by Watermill.
- `nextLocalOrdinal(...)` is phase-local scaffolding only. Phase 2 should move ordinal assignment to the single consumer-side ordering point.
- The current in-memory `HydrationStore` implementation remains a valid store implementation, but Phase 2 should run it behind the bus consumer rather than treating direct in-process publish as the permanent architecture.
- The Systemlab Lab 01 HTTP surface should remain stable; only the framework internals behind it should change when Watermill is introduced.

## Systemlab Workstream

- Build Lab 01 around a real command form: session id, command name, payload editor.
- Display trace rows for dispatch, handler invocation, emitted events, projection outputs, and store mutations.
- Show current session metadata and current timeline snapshot in side panels.
- Add pass/fail badges for invariants like “timeline projection ran” and “cursor advanced”.

### ASCII reference: Systemlab page

```text
+------------------------------------------------------------------------------------------------+
| Phase 1 — Command → Event → Projection                                                         |
+--------------------------------------+---------------------------------------------------------+
| [Controls]                           | [Trace]                                                 |
| Session: s-123                       | 1. Submit(StartInference)                               |
| Command: StartInference              | 2. Session created                                       |
| Payload: {prompt:"hello"}           | 3. Handler invoked                                       |
| [Submit] [Reset]                    | 4. Event published: InferenceStarted                     |
|                                      | 5. UIProjection -> MessageStarted                        |
| [Assertions]                         | 6. TimelineProjection -> ChatMessage(streaming=true)     |
|  ✓ session exists                    | 7. Hydration cursor = 1                                  |
|  ✓ cursor advanced                   |                                                         |
+--------------------------------------+---------------------------+-----------------------------+
| [Session State]                                                  | [Hydration Snapshot]       |
| id=s-123                                                         | ordinal=1                  |
| metadata={...}                                                   | entities=[ChatMessage...]  |
+------------------------------------------------------------------------------------------------+
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

- The lab must invoke real command submission logic, not a fake reducer.
- Handlers may publish backend events but may not write UI output directly.
- Projections consume canonical backend events, not UI envelopes.

## Testing and Validation

- Unit tests for session and command registries.
- Unit tests for memory hydration store semantics.
- Component test for `Hub.Submit -> handler -> projection -> store`.
- Systemlab transcript export for the happy path and one error path.

## Risks and Open Questions

- Temptation to bypass the event model and mutate store state directly from handlers.
- Overloading the lab with websocket concerns before the in-memory path is stable.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Implement `SessionRegistry` with lazy `SessionMetadataFactory` integration.
2. Implement `CommandRegistry` with duplicate-registration checks.
3. Implement in-memory `HydrationStore` including cursor and defensive-copy `View` semantics.
4. Wire `Hub.Submit(...)` to the real dispatch path.
5. Create one minimal example handler plus UI/timeline projections for lab use.
6. Build Systemlab Lab 01 page with command form, trace panel, session panel, and hydration snapshot panel.
7. Add transcript export (JSON or markdown) for a lab run.
8. Add automated tests for happy path, unknown command, and projection error handling policy.
9. Document exactly which seams Phase 2 will replace with Watermill-backed behavior.

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
