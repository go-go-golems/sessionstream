---
Title: Phase 3 Websocket Transport and Hydration/Reconnect Lab
Ticket: EVT-STREAM-007
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
Summary: "Phase 3 adds the first real live-client transport: websocket connect, subscribe, snapshot-before-live delivery, and per-session routing by ConnectionId. Systemlab gains the Hydration/Reconnect lab."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 3 Websocket Transport and Hydration/Reconnect Lab, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 3 Websocket Transport and Hydration/Reconnect Lab

## Executive Summary

Make the framework feel like a realtime substrate rather than just an in-process event engine. A reconnecting client should receive a coherent snapshot first and then continue with live UI events for the same session.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- Websocket transport.
- Connection registry and subscription sets.
- Subscribe/unsubscribe flow.
- Snapshot-before-live sequencing.
- Systemlab Hydration/Reconnect lab.

### Out of scope
- No SQL store yet.
- No legacy webchat migration yet.

## Deliverables

- Functional websocket transport implementing the public transport seam.
- Connection tracking via `ConnectionId`.
- Snapshot-first subscribe flow.
- Systemlab page showing multiple simulated clients and reconnect behavior.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Implement `transport/ws` with connection bookkeeping, command intake, and outgoing UI delivery.
- Connect websocket subscriptions to `SessionRegistry` subscriber sets.
- Ensure the transport requests snapshot from the real hydration store before starting live delivery.
- Keep transport-specific concerns isolated from command handler logic.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

## Systemlab Workstream

- Build Lab 03 with simulated Client A / Client B panels.
- Allow connect, disconnect, reconnect, and subscribe with a `sinceOrdinal` field.
- Display snapshot payloads, live UI events, and the current store snapshot side by side.
- Add invariant checks for “snapshot ordinal < next live ordinal” and “reconnected client converges to same final state”.

### ASCII reference: Systemlab page

```text
+------------------------------------------------------------------------------------------------+
| Phase 3 — Hydration and Reconnect                                                             |
+--------------------------------------+---------------------------------------------------------+
| [Client A]                           | [Client B]                                              |
| status: connected                    | status: disconnected                                    |
| subscribed: s-123                    | [Connect] [Subscribe s-123 since=0]                     |
| live events:                         | snapshot: ordinal=4                                     |
|  - MessageStarted                    | entities=[ChatMessage(...)]                             |
|  - MessageAppended                   | live events after reconnect:                            |
|                                      |  - MessageFinished                                      |
+--------------------------------------+---------------------------+-----------------------------+
| [Hydration Store Snapshot]                                        | [Checks]                    |
| ordinal=4                                                         | ✓ snapshot before live      |
| ChatMessage(id=m1, text="hello world", streaming=false)           | ✓ final states converge     |
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

- Handlers remain unaware of websocket connections.
- Transport may route by `ConnectionId` but must not own business logic.
- Systemlab simulates clients using the same public websocket protocol later used by real apps.

## Testing and Validation

- Transport tests for subscribe/unsubscribe.
- Integration tests for snapshot-before-live semantics.
- Systemlab reconnect scenarios with at least two clients.
- Explicit tests for empty session, existing session, and reconnect after several events.

## Risks and Open Questions

- Sending live events before snapshot is emitted.
- Accidentally coupling transport to app-specific message shapes beyond public UI events.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Implement websocket transport package under `evtstream/transport/ws`.
2. Implement connection registry keyed by `ConnectionId`.
3. Implement subscribe/unsubscribe protocol and session subscription updates.
4. Ensure snapshot retrieval and delivery occurs before live stream fan-out.
5. Add tests for empty snapshot, populated snapshot, and reconnect after disconnection.
6. Build Systemlab Lab 03 with two simulated clients, connection controls, and live trace panels.
7. Show store snapshot and client-local received events in one screen.
8. Add invariant badges for snapshot-before-live and convergence of final state.
9. Prepare hooks for future liveness/tick support without blocking this phase.

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
