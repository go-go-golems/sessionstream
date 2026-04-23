---
Title: Design the job demo interactive essay for sessionstream-systemlab
Ticket: SESSIONSTREAM-002
Status: active
Topics:
    - architecture
    - backend
    - framework
    - event-streaming
    - onboarding
    - systemlab
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Plan and intern-facing implementation guide for turning the next sessionstream demo into a textbook-style interactive essay about long-running job sessions, rather than a conventional dashboard app.
LastUpdated: 2026-04-23T10:55:00-04:00
WhatFor: Track the analysis, design, implementation plan, diary, and delivery steps for the sessionstream job demo interactive essay.
WhenToUse: Use when orienting contributors to the job demo effort, reviewing the architecture, or implementing the next generation of sessionstream-systemlab teaching surfaces.
---

# Design the job demo interactive essay for sessionstream-systemlab

## Overview

`SESSIONSTREAM-002` defines the next framework-facing demo direction for `sessionstream`: a job-oriented interactive essay that explains session-based streaming by letting the reader create, inspect, interrupt, retry, reconnect to, and reinterpret one long-running job session.

The central recommendation is deliberately narrow and opinionated:

- do **not** build a small SaaS-style “jobs dashboard” first,
- do **not** hide the framework behind a polished but opaque demo shell,
- instead build a **textbook-style explorable explanation** that teaches the reader what a session is, how commands become events, how projections derive state, and why hydration/reconnect matter,
- and use a small `jobdemo` example package plus a dedicated `sessionstream-systemlab` essay page to make those ideas visible.

## Current status

Current status: **active**

This ticket currently contains:

- a detailed intern-facing analysis / design / implementation guide,
- a diary recording ticket creation and document preparation,
- ticket task and changelog bookkeeping,
- and the reMarkable delivery step for the guide bundle.

## Primary documents

- [Design doc](./design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md)
- [Diary](./reference/01-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Why this ticket exists

The extracted `sessionstream` repository already has a useful `chatdemo` and a framework-oriented `sessionstream-systemlab`. What it does not yet have is a flagship teaching surface that feels more like an explorable essay than a phase-by-phase internal lab. The user explicitly steered toward a “better, more fun version of Systemlab” and then narrowed that idea further: the `jobdemo` itself should be expressed in that same interactive-essay style.

This ticket exists to pin down that direction before implementation starts. It makes the conceptual shape explicit, names the domain model, recommends the package boundaries, describes the user journey section by section, and gives a new intern enough system context to build it without cargo-culting the existing chat demo.

## Planned outcome

Target outcome after implementation:

```text
sessionstream/
  examples/jobdemo/
    -> reusable job-domain example package showing commands/events/projections

  cmd/sessionstream-systemlab/
    -> interactive essay page that exercises jobdemo through public APIs
    -> chapter prose + live controls + raw-event/state inspectors
    -> reconnect/hydration and retry/cancel demonstrations
```

The essay should teach the framework, not merely showcase one app.

## Structure

- `design-doc/` — long-form architecture, teaching strategy, and implementation guidance
- `reference/` — chronological diary and future quick-reference artifacts
- `playbooks/` — future smoke-test or review runbooks if needed later
- `scripts/` — temporary ticket-specific scripts if implementation creates any
- `archive/` — retired design material if the direction evolves
