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
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-004--phase-0-foundations-api-skeleton-and-systemlab-shell/design-doc/01-phase-0-implementation-plan.md
      Note: Primary phase implementation plan and Systemlab reference.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Parent cross-phase implementation guide.
ExternalSources: []
Summary: "Phase 0 creates the two-repo/two-app shape, freezes the initial API surface, and builds a minimal separate Systemlab shell that consumes only public framework seams."
LastUpdated: 2026-04-20T00:20:00-04:00
WhatFor: "Track one implementation phase of the event-streaming framework and its matching Systemlab page."
WhenToUse: "When implementing or reviewing Phase 0 Foundations, API Skeleton, and Systemlab Shell."
---

# Phase 0 Foundations, API Skeleton, and Systemlab Shell

## Overview

Create the foundation for all later work: clear package boundaries, compile-only framework interfaces, a separate Systemlab app shell, and a disciplined rule that Systemlab may only touch public framework APIs.

## Documents

- **`design-doc/01-phase-0-implementation-plan.md`** — detailed implementation plan, Systemlab page sketch, risks, and validation strategy.
- **`reference/01-investigation-diary.md`** — implementation diary covering the Phase 0 scaffold, boundary decisions, and validation commands.
- **`reference/02-systemlab-status.json`** — captured Phase 0 Systemlab status artifact from the running shell.

## Status

Current status: **implemented**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
