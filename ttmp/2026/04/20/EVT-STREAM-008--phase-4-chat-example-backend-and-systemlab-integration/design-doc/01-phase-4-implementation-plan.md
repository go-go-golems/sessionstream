---
Title: Phase 4 Chat Example Backend and Systemlab Integration
Ticket: EVT-STREAM-008
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
Summary: "Phase 4 proves the substrate with the first real application backend: chat. The framework stays generic, while Systemlab starts to feel like a product demo because it now exercises a real conversational flow."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Detailed phase-specific implementation plan covering framework work, Systemlab work, validation, risks, and a reference ASCII sketch of the intended Systemlab page."
WhenToUse: "When implementing Phase 4 Chat Example Backend and Systemlab Integration, planning the handoff into the next phase, or reviewing what the Systemlab companion page for this phase should expose."
---

# Phase 4 Chat Example Backend and Systemlab Integration

## Executive Summary

Use a concrete chat example to prove that the framework abstractions are neither too generic to be usable nor too chat-specific to be reusable. The example should become the canonical teaching backend for new contributors.

This phase is intentionally scoped so that both workstreams move together:

- the **framework** gains one real capability,
- the **Systemlab** gains one real page that explains and exercises that capability,
- the page is built only against the public API boundary the framework is trying to establish.

The phase is complete only when the feature works **and** Systemlab can demonstrate, inspect, and validate it.

## Problem Statement

The full event-streaming framework will only remain understandable if each major capability is implemented in a bounded phase with its own validation surface. Without that, the code may technically work while remaining difficult to explain, difficult to demo, and difficult to regression-test. This ticket solves that by scoping a single phase with paired framework and Systemlab deliverables.

## Scope

### In scope
- `examples/chat` schemas, handlers, and projections.
- Streaming chat-message reduction logic.
- Systemlab chat presets and timeline visualizers.
- Narrated explanation of chat flow as a first-class example.

### Out of scope
- No SQL persistence yet.
- No legacy webchat compatibility work yet.

## Deliverables

- Working chat example package built only on public substrate APIs.
- Start/stop inference command flow for the example.
- Timeline projection that builds `ChatMessage` from deltas.
- Systemlab page tailored to chat with prompt entry, token stream trace, and timeline entity view.

## Implementation Sequence

1. Finish the framework work for this phase behind the intended public boundary.
2. Freeze the smallest usable seam needed by Systemlab.
3. Build the Systemlab page against that seam only.
4. Add automated tests for the feature itself.
5. Add scenario presets and assertions in Systemlab so the feature can be explored and explained.
6. Capture at least one transcript or screenshot that demonstrates the phase goal.

## Framework Workstream

- Implement protobuf schemas and registration helpers for chat commands, backend events, UI events, and timeline entities.
- Implement `StartInference` and `StopInference` command handlers.
- Implement chat UI projection and timeline projection mirroring the worked example in EVT-STREAM-002.
- Keep all chat semantics in the example package, not in the core substrate.

### Design notes

- Keep the substrate centered on `SessionId` as the universal routing key.
- Keep backend events canonical; do not make UI envelopes the primary internal form.
- Preserve the public seams defined in EVT-STREAM-002 rather than inventing phase-local shortcuts.
- Use donor code from `pinocchio/pkg/webchat` only where EVT-STREAM-002 and EVT-STREAM-003 already marked it as safe to adapt.

## Systemlab Workstream

- Build a chat-focused Systemlab page with prompt box, session selector, and scenario presets.
- Show backend event stream, UI event stream, and timeline entity evolution in separate columns.
- Add presets for token streaming, completion, and interrupted generation.
- Provide a “print transcript/export” button for documentation and debugging.

### ASCII reference: Systemlab page

```text
+------------------------------------------------------------------------------------------------+
| Phase 4 — Chat Example                                                                        |
+--------------------------------------+---------------------------------------------------------+
| [Prompt]                             | [Backend Events]                                        |
| "Explain ordinals"                  | InferenceStarted                                         |
| Model: claude-sonnet                 | TokensDelta("Ordi")                                     |
| [Send] [Stop]                        | TokensDelta("nals...")                                  |
| Presets: happy / stream / cancel     | InferenceFinished                                        |
+--------------------------------------+---------------------------+-----------------------------+
| [UI Events]                                                      | [Timeline Entities]        |
| MessageStarted                                                  | ChatMessage{id=m1,...}     |
| MessageAppended("Ordi")                                         | text accumulates live       |
| MessageFinished                                                 | streaming=false at end      |
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

- The chat example is a consumer of `evtstream`, not part of `evtstream` itself.
- Systemlab should exercise the chat example via public registration and transport seams.
- Do not let chat-specific helper types leak back into generic package signatures.

## Testing and Validation

- Example-specific unit tests for chat projections.
- End-to-end example run through Systemlab or integration tests.
- Validation that timeline final state matches live streamed UI state.
- Happy-path and stop/cancel path coverage.

## Risks and Open Questions

- Example code may accidentally become substrate code if convenience helpers are placed in the wrong package.
- Systemlab may become too chat-specific if the generic lab pages are removed instead of supplemented.

Open questions should be handled conservatively in this phase: if a decision is not required to ship the scoped deliverable, prefer a narrow implementation that preserves future options instead of expanding the phase.

## Detailed Task Breakdown

1. Add `examples/chat` package tree.
2. Register chat command/event/UI/entity schemas.
3. Implement `StartInference` and `StopInference` handlers.
4. Implement timeline projection reducing token deltas into `ChatMessage`.
5. Implement UI projection emitting message append/finish UI events.
6. Add Systemlab Phase 4 page with prompt control, presets, event panels, and timeline panel.
7. Add a cancellation/stop scenario and verify final state is coherent.
8. Document which pieces of the example are teaching material vs production candidates.
9. Capture at least one exported transcript to use in later docs or regression fixtures.

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
