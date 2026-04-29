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
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-006--phase-2-watermill-bus-consumer-and-ordering-lab/design-doc/01-phase-2-implementation-plan.md
      Note: Primary phase implementation plan and Systemlab reference.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Parent cross-phase implementation guide.
ExternalSources: []
Summary: "Phase 2 replaces the purely local in-memory event path with a real Watermill publisher/consumer pipeline and introduces consumption-time ordinal assignment. Systemlab adds the Ordering and Ordinals lab."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Track one implementation phase of the event-streaming framework and its matching Systemlab page."
WhenToUse: "When implementing or reviewing Phase 2 Watermill Bus, Consumer, and Ordering Lab."
---

# Phase 2 Watermill Bus, Consumer, and Ordering Lab

## Overview

Prove the event-streaming architecture under a real bus boundary. Events should be published with ordinal zero and stamped on consumption, preserving per-session ordering and laying the foundation for distributed workers.

## Documents

- **`design-doc/01-phase-2-implementation-plan.md`** — detailed implementation plan, Systemlab page sketch, risks, and validation strategy.
- **`reference/01-investigation-diary.md`** — implementation diary covering the Watermill adapter, consumer loop, Lab 02 UI, and bug fixes found while exercising the phase.
- **`reference/02-phase-2-run-response.json`** — captured Lab 02 run response showing publish/consume metadata and per-session ordinals.
- **`reference/03-phase-2-transcript.json`** — exported JSON transcript for the captured Lab 02 scenario.
- **`reference/04-phase-2-transcript.md`** — exported Markdown transcript for the same Phase 2 scenario.

## Status

Current status: **implemented**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
