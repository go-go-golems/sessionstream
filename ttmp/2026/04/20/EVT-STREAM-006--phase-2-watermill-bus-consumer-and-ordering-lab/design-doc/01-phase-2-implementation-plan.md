---
Title: Phase 2 Watermill Bus, Consumer, and Ordering Lab
Ticket: EVT-STREAM-006
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
Summary: "Phase 2 replaces the purely local in-memory event path with a real Watermill publisher/consumer pipeline and introduces consumption-time ordinal assignment. Systemlab adds the Ordering and Ordinals lab."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 2 Watermill Bus, Consumer, and Ordering Lab, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 2 Watermill Bus, Consumer, and Ordering Lab

## Executive Summary

Prove the event-streaming architecture under a real bus boundary. Events should be published with ordinal zero and stamped on consumption, preserving per-session ordering and laying the foundation for distributed workers.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- Watermill-backed `EventPublisher`.
- Bus consumer loop.
- Event decoding and schema validation.
- Ordinal assignment on consume.
- Systemlab Ordering and Ordinals lab.

### Out of scope
- No websocket subscription yet.
- No SQL store yet.

## Deliverables

- Real Watermill pipeline using `gochannel` first and keeping other backends possible.
- Ordinal assigner with stream-id-aware logic.
- Systemlab page showing publish metadata, consume metadata, and assigned ordinals per session.
- Fault-injection controls for missing/odd stream ids.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Implement `NewEventPublisher` against Watermill message publisher.
- Implement the bus consumer that decodes events and invokes projections.
- Add an ordinal assigner that derives ordinals from stream metadata when possible and falls back safely otherwise.
- Keep topic/partitioning keyed by `SessionId`.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

### Implementation note: topic and partition rule

The current Phase 2 implementation uses a shared Watermill topic (`evtstream.phase2` in Systemlab) and carries the partition intent in message metadata under `evtstream_partition_key=<SessionId>`. That keeps the `gochannel` implementation simple while still documenting the invariant that real backends must preserve ordered consumption per `SessionId`.

### Implementation note: ordinal rendering in Systemlab

Consumer-side ordinals derived from Redis-style stream ids are intentionally large enough to exceed JavaScript's safe integer range. The Systemlab HTTP responses therefore render ordinals as strings so the lab can display exact values without browser rounding.

## Systemlab Workstream

- Build Lab 02 with controls for multiple sessions, burst publishing, missing stream id, and synthetic restart.
- Render a live ordered table of consumed events per session.
- Show both publish-time and consume-time views of each event so the zero-at-publish rule is obvious.
- Expose a reset button that clears the memory store and consumer state.

### ASCII reference: Systemlab page

```text
+------------------------------------------------------------------------------------------------+
| Phase 2 — Ordering and Ordinals                                                               |
+--------------------------------------+---------------------------------------------------------+
| [Scenario Controls]                  | [Bus / Consumer Trace]                                  |
| Session A: s-a                       | Published: TaskStepCompleted ord=0                       |
| Session B: s-b                       | Watermill msg id: 4f2...                                 |
| [Publish A] [Publish B]              | Stream ID: 1713560000123-4                               |
| [Burst A] [Restart Consumer]         | Assigned ordinal: 1713560000123000004                    |
| [Missing Stream ID] toggle           |                                                         |
+--------------------------------------+---------------------------+-----------------------------+
| [Per-Session Ordinal Table]                                      | [Invariant Checks]         |
| s-a : 1,2,3,4                                                    | ✓ monotonic per session    |
| s-b : 1,2                                                        | ✓ no regressions           |
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

- The consumer owns ordinal assignment; handlers must never assign final ordinals.
- Systemlab must inspect behavior through public traces/endpoints, not by reading consumer internals directly.
- Per-session ordering rules must remain explicit in code and docs.

## Testing and Validation

- Unit tests for ordinal derivation logic.
- Component tests for Watermill `gochannel` roundtrip.
- Regression tests for missing/invalid stream ids.
- Systemlab scenarios for two sessions and consumer restart.

## Risks and Open Questions

- Leaking ordering decisions into handlers or examples.
- Assuming a specific Watermill backend too early instead of keeping the seam generic.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Implement Watermill-backed `EventPublisher` with schema validation.
2. Implement bus consumer loop and event decode/dispatch pipeline.
3. Implement ordinal assigner with stream-id-aware and fallback modes.
4. Document required topic/partitioning rule keyed by `SessionId`.
5. Add tests for monotonic ordering under stream-id and no-stream-id cases.
6. Build Systemlab Lab 02 with publish controls, burst controls, and restart/reset controls.
7. Show raw message metadata and derived ordinal side by side in the lab UI.
8. Add assertion panel for monotonic ordering and per-session isolation.
9. Prepare the consumer output seam that websocket transport will subscribe to in Phase 3.

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
