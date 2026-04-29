---
Title: Code review and architecture audit for sessionstream
Ticket: SESSIONSTREAM-003
Status: active
Topics:
    - architecture
    - backend
    - event-streaming
    - framework
    - onboarding
    - code-review
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Ticket workspace for a detailed intern-oriented code review and architecture audit of sessionstream."
LastUpdated: 2026-04-29T14:44:36-04:00
WhatFor: "Use this ticket to find the sessionstream architecture audit, investigation diary, task list, and changelog."
WhenToUse: "Read before planning cleanup work in the core hub, hydration stores, websocket transport, examples, or systemlab."
---

# Code review and architecture audit for sessionstream

## Overview

This ticket contains a detailed code review and architecture audit of the `sessionstream` repository. The primary deliverable is written for a new intern: it explains what the system is, how commands/events/projections/stores/transports fit together, where the important APIs live, and which cleanup items should be prioritized.

The review focuses on unclear APIs, obtuse or misleading contracts, long files, extraction-era naming, potential correctness issues, and problematic architecture boundaries.

## Key Links

- [Sessionstream code review and architecture audit](./design-doc/01-sessionstream-code-review-and-architecture-audit.md)
- [Remediation plan for replay store and API cleanup](./design-doc/02-remediation-plan-for-replay-store-and-api-cleanup.md)
- [Investigation diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**.

The review deliverable is complete. A follow-up remediation design now captures the accepted direction: real replay store, cursor-seeded local ordinals, fail-closed projection defaults with error reporting, split UI/timeline policies, no-backcompat `evtstream` cleanup, fanout-only websocket scope, defensive schema cloning, and a self-contained protobuf chat example.

## Topics

- architecture
- backend
- event-streaming
- framework
- onboarding
- code-review
- cleanup

## Main findings

The highest-priority findings are:

1. `HydrationStore.Snapshot(ctx, sid, asOf)` exposes historical-looking semantics that stores do not implement.
2. Websocket `sinceOrdinal` is accepted but not used for replay or filtering.
3. Local ordinals are not seeded from persistent store cursors after restart.
4. Default projection error handling can advance cursors while dropping timeline state.
5. Malformed Watermill messages are acknowledged silently.
6. Old `evtstream` names remain in public constants and SQLite table names.
7. Systemlab phase files are useful but too large and repetitive for long-term maintenance.

## Structure

- `design-doc/` - primary long-form audit and cleanup plan
- `reference/` - investigation diary
- `playbooks/` - future command sequences and test procedures
- `scripts/` - temporary code and tooling
- `various/` - working notes and research
- `archive/` - deprecated or reference-only artifacts
