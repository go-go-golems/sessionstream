---
Title: Post-stabilization evtstream standalone module extraction and Systemlab relocation
Ticket: EVT-STREAM-012
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - backend
    - implementation
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-012--post-stabilization-evtstream-standalone-module-extraction-and-systemlab-relocation/design-doc/01-post-stabilization-standalone-module-extraction-plan.md
      Note: Primary design and execution plan for extracting evtstream into its own module and moving Systemlab with it.
    - Path: pinocchio/pkg/evtstream/doc.go
      Note: Current substrate package root that is a strong candidate for extraction.
    - Path: pinocchio/cmd/evtstream-systemlab/README.md
      Note: Current Systemlab boundary contract and rationale for keeping the lab separate from substrate core.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md
      Note: Phase 6 cutover plan that should stabilize before the extraction is attempted.
ExternalSources: []
Summary: "Plan a later, post-stabilization extraction of evtstream into its own standalone Go module/package, with Systemlab moving alongside it as a separate consumer app, so future architectural boundaries are enforced by real module seams rather than convention alone."
LastUpdated: 2026-04-20T11:12:00-04:00
WhatFor: "Track the later architectural split that should happen after evtstream and cmd/web-chat are polished, tested, and stable enough to freeze a clean public boundary."
WhenToUse: "When planning or eventually executing the extraction of evtstream out of pinocchio into a standalone module and relocating Systemlab with it."
---

# Post-stabilization evtstream standalone module extraction and Systemlab relocation

## Overview

This ticket is intentionally **deferred work**. It exists to capture the architectural move we want later: once `evtstream`, Systemlab, and the `cmd/web-chat` port are polished and tested, extract `evtstream` into its own standalone Go module/package and move Systemlab into that module as a separate app.

The goal is not to do the move immediately. The goal is to define it early enough that current implementation choices keep the extraction path open.

## Why this ticket exists

The current `pkg/evtstream` code is already much cleaner and less interwoven with the rest of `pinocchio` than `pkg/webchat`. Keeping it inside `pinocchio` for initial development was pragmatic, but it weakens boundary enforcement because the substrate, example code, Systemlab, and consuming applications all still live under the same module umbrella.

A later extraction should give us:

- a real module boundary between substrate and applications,
- cleaner publishing/versioning for `evtstream`,
- stronger protection against `webchat`-style leakage back into the substrate,
- a more honest ownership model for Systemlab as an `evtstream` teaching and validation app.

## Documents

- **`design-doc/01-post-stabilization-standalone-module-extraction-plan.md`** — detailed design, rationale, target layout, preconditions, and later execution plan for the extraction.

## Status

Current status: **active**

Execution status: **planned for later, after stabilization**

## Tasks

See [tasks.md](./tasks.md).

## Changelog

See [changelog.md](./changelog.md).
