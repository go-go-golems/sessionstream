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
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-005--phase-1-in-memory-core-and-command-to-projection-lab/design-doc/01-phase-1-implementation-plan.md
      Note: Primary phase implementation plan and Systemlab reference.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Parent cross-phase implementation guide.
ExternalSources: []
Summary: "Phase 1 delivers the first executable substrate path in memory: synchronous command dispatch, session creation, projection execution, and hydration snapshots. Systemlab gets the first true interactive lab: Command → Event → Projection."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Track one implementation phase of the event-streaming framework and its matching Systemlab page."
WhenToUse: "When implementing or reviewing Phase 1 In-Memory Core and Command-to-Projection Lab."
---

# Phase 1 In-Memory Core and Command-to-Projection Lab

## Overview

Make the framework do one real thing end to end, without websockets or distributed transport yet. The phase should prove that a command can create backend events, run projections, and update durable-ish in-memory state.

## Documents

- **`design-doc/01-phase-1-implementation-plan.md`** — detailed implementation plan, Systemlab page sketch, risks, and validation strategy.
- **`reference/01-investigation-diary.md`** — implementation diary covering the in-memory execution path, tests, and export flow.
- **`reference/02-phase-1-run-response.json`** — raw Systemlab Phase 1 run response captured from the HTTP lab endpoint.
- **`reference/03-phase-1-transcript.json`** — exported JSON transcript for the happy-path lab run.
- **`reference/04-phase-1-transcript.md`** — exported Markdown transcript for the same happy-path run.

## Status

Current status: **implemented**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
