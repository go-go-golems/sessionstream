---
Title: Phase 6 cmd/web-chat Port and Migration Playbook
Ticket: EVT-STREAM-010
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
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md
      Note: Primary phase implementation plan and Systemlab reference.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Parent cross-phase implementation guide.
ExternalSources: []
Summary: "Phase 6 ports cmd/web-chat onto evtstream, severs pkg/webchat from the live runtime path, and uses transcript-backed regression evidence to produce a disciplined migration playbook."
LastUpdated: 2026-04-20T18:42:00-04:00
WhatFor: "Track the phase that ports cmd/web-chat onto evtstream and turns the resulting regression evidence into a migration playbook."
WhenToUse: "When implementing or reviewing the cmd/web-chat cutover off pkg/webchat and onto evtstream."
---

# Phase 6 cmd/web-chat Port and Migration Playbook

## Overview

Port the real `cmd/web-chat` application onto `evtstream`, remove `pkg/webchat` from the live runtime path, and use transcript-backed regression evidence to control the cutover. This phase is no longer a vague comparison exercise; it is the concrete severance and migration-playbook phase.

## Documents

- **`design-doc/01-phase-6-implementation-plan.md`** — detailed analysis, design, and implementation guide for the `cmd/web-chat` port, including file layout, runtime diagrams, migration slices, and regression-playbook strategy.
- **`design-doc/02-web-frontend-migration-and-design-guide.md`** — detailed frontend companion guide covering the current web architecture, UX/functionality continuity requirements, new evtstream-facing API direction, and the work needed to port the web client without changing the product feel.
- **`design-doc/03-current-cmd-web-chat-behavior-inventory.md`** — current-state inventory of the app’s public/runtime behavior, route surface, frontend flow, and legacy assumptions that matter for the cutover.
- **`design-doc/04-preserve-change-drop-matrix.md`** — explicit decision matrix naming which product behaviors must survive and which legacy runtime/wire behaviors should be replaced or dropped.
- **`design-doc/05-canonical-web-contract-and-package-layout.md`** — concrete target package seams and canonical HTTP/websocket contract for the next `cmd/web-chat` implementation slices.
- **`reference/01-investigation-diary.md`** — detailed implementation diary tracking the execution of the port slice by slice, including commands, failures, tests, and review notes.
- **`reference/02-migrated-session-snapshot.json`** — captured snapshot from the migrated canonical path after a real browser-driven prompt flow.
- **`reference/03-live-runtime-session-snapshot.json`** — captured snapshot from the post-cutover live runtime path after the legacy `/chat` and `/api/timeline` routes were removed from the default app wiring.
- **`reference/04-live-route-cutover-checks.json`** — machine-readable route check showing canonical routes returning success while legacy `/chat` and `/api/timeline` return `404` in the cutover build.
- **`reference/05-legacy-flow-transcript.json`** — deterministic harness capture of the legacy `/chat` + `/ws` + `/api/timeline` flow used as historical expectation evidence.
- **`reference/06-canonical-flow-transcript.json`** — deterministic harness capture of the canonical session-based `/api/chat/...` flow.
- **`reference/07-legacy-vs-canonical-flow-comparison.md`** — preserve/change/drop comparison that tags the observed flow differences as intentional, preserved, dropped, or still needing follow-up.
- **`reference/08-runtime-wired-user-message-snapshot.json`** — canonical snapshot captured after wiring runtime-resolved inference and user-message projection; on this machine the assistant stops with `no API key for openai`, but the artifact shows both user and assistant entities flowing through the new stack.
- **`reference/09-real-provider-backed-session-snapshot.json`** — successful canonical snapshot captured from a real provider-backed `gpt-5-nano-low` run using the normal pinocchio config/profile stack; it shows both user and assistant entities and a finished non-echo assistant response.
- **`reference/10-systemlab-phase-6-run.json`** — machine-readable result from the new Systemlab Phase 6 migration/regression console probing a live canonical `cmd/web-chat` server and verifying canonical routes, legacy-route removal, user+assistant snapshot contents, and non-echo assistant output.
- **`reference/11-final-cutover-recommendation-and-legacy-removal-plan.md`** — final acceptance note for the Phase 6 cutover, including the current readiness call and a concrete plan for deleting or deferring the remaining `pkg/webchat`-era artifacts.

## Status

Current status: **active**

The cutover itself is now recommended for acceptance, and the old `pinocchio/pkg/webchat` package has now been deleted entirely. The remaining work in this ticket is historical-ticket hygiene rather than any product-path cleanup.

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
