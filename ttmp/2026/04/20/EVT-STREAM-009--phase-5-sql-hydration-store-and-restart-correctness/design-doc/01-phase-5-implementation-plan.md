---
Title: Phase 5 SQL Hydration Store and Restart Correctness
Ticket: EVT-STREAM-009
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
Summary: "Phase 5 makes durability real. The hydration store moves beyond in-memory-only semantics and must preserve cursor plus timeline state across restart. Systemlab gains a store-compare and restart-correctness page."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 5 SQL Hydration Store and Restart Correctness, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 5 SQL Hydration Store and Restart Correctness

## Executive Summary

Guarantee that reconnect and replay behavior survive process restart. The same session should resume from durable cursor state without duplicate or skipped ordinals, and snapshots from the SQL store should match expectations.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- SQL hydration store.
- Atomic apply plus cursor advance.
- Restart/resume behavior.
- Systemlab memory-vs-SQL comparison mode.
- Failure/restart simulation.

### Out of scope
- No legacy migration yet.

## Deliverables

- Durable SQL store implementation.
- Restart correctness tests.
- Systemlab page that can switch between memory and SQL modes and show before/after restart snapshots.
- Operational guidance for local dev database reset.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Implement SQL schema and migrations for entity state plus per-session cursor.
- Implement `Apply`, `Snapshot`, `View`, and `Cursor` semantics transactionally.
- Ensure the consumer resumes ordinal assignment from durable cursor state on startup.
- Add utilities to reset the SQL store in test/dev scenarios.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

## Systemlab Workstream

- Build a page with a storage-mode toggle (memory vs SQL).
- Add buttons for seed session, stop backend, restart backend, reconnect client, and compare snapshots.
- Display both pre-restart and post-restart entity/cursor state.
- Highlight divergence if memory and SQL modes behave differently.

### ASCII reference: Systemlab page

```text
+------------------------------------------------------------------------------------------------+
| Phase 5 — Persistence and Restart Correctness                                                 |
+--------------------------------------+---------------------------------------------------------+
| [Storage Mode]                       | [Pre-Restart State]                                     |
| ( ) Memory   (*) SQL                 | cursor=8                                                 |
| [Seed Session] [Restart Backend]     | entities=[ChatMessage(...)]                              |
| [Reconnect Client]                   |                                                         |
+--------------------------------------+---------------------------+-----------------------------+
| [Post-Restart State]                                              | [Comparison]               |
| cursor=8                                                         | ✓ cursor preserved         |
| entities=[ChatMessage(...)]                                       | ✓ entities preserved       |
| next live event ordinal=9                                         | ✓ resume without gaps      |
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

- Store semantics must remain behind `HydrationStore`; Systemlab should not query the SQL schema directly except through explicit debug endpoints if added.
- Restart correctness must be validated via the same consumer path used in production, not a special-case shortcut.

## Testing and Validation

- Unit tests for SQL store transaction behavior.
- Integration tests for process restart and cursor resume.
- Systemlab scenarios comparing memory and SQL modes.
- Regression tests for duplicate or skipped ordinal after restart.

## Risks and Open Questions

- Implementing SQL semantics that differ subtly from memory semantics.
- Testing restart logic only manually instead of encoding it in automated integration tests.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Design SQL schema for timeline entities and per-session cursor rows.
2. Implement SQL `HydrationStore` with atomic `Apply` + cursor advancement.
3. Add migration/bootstrap path for the SQL store.
4. Implement store reset helper for local lab/testing use.
5. Add integration tests for restart/resume correctness.
6. Build Systemlab Phase 5 page with memory/SQL mode toggle and restart controls.
7. Show pre/post restart cursor and snapshot state side by side.
8. Add comparison badges highlighting any divergence between backends.
9. Document dev setup for local SQL store and cleanup procedure.

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
