---
Title: Remove Systemlab and Downstream Diagnostic Complexity
Ticket: SESSIONSTREAM-006
Status: active
Topics:
    - architecture
    - onboarding
    - event-streaming
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://ttmp/2026/08/11/SESSIONSTREAM-006--remove-systemlab-and-downstream-diagnostic-complexity/design-doc/01-systemlab-removal-and-sessionstream-simplification-plan.md
      Note: Primary implementation plan
    - Path: repo://ttmp/2026/08/11/SESSIONSTREAM-006--remove-systemlab-and-downstream-diagnostic-complexity/reference/01-investigation-diary.md
      Note: Chronological investigation and future implementation record
ExternalSources: []
Summary: Remove the Sessionstream Systemlab application and every downstream API, observer, lifecycle, build, release, documentation, dependency, and test subsystem that lacks an independently retained consumer.
LastUpdated: 2026-08-11T06:58:02.655041636-04:00
WhatFor: Coordinate evidence-led repository simplification without deleting browser heartbeat, WebSocket lifecycle, or downstream product contracts required independently of Systemlab.
WhenToUse: Use as the entry point for Systemlab removal, observer API disposition, and before/after simplification validation.
---


# Remove Systemlab and Downstream Diagnostic Complexity

## Overview

SESSIONSTREAM-006 removes `cmd/sessionstream-systemlab` and audits every subsystem that exists downstream of the teaching application. The initial candidates include Bus, Pipeline, and Transport observers; WebSocket observer dispatch; Makefile and GoReleaser binary wiring; CI smoke tests; help publication; README and reference documentation; static browser assets; and dependencies made unreachable by deletion.

The ticket preserves independently required Sessionstream behavior. In particular, browser-compatible protobuf ping/pong, the WebSocket failure detector, snapshot-before-live hydration, ordered request processing, lifecycle APIs, stores, projections, and downstream rag-ttc contracts are not Systemlab features and remain unless separately approved.

Implementation begins only after PR #11 merges and this branch is rebased onto final main.

## Key links

- [Systemlab Removal and Sessionstream Simplification Plan](./design-doc/01-systemlab-removal-and-sessionstream-simplification-plan.md)
- [Investigation Diary](./reference/01-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Current status

- Ticket initialized.
- Initial direct and downstream inventory recorded.
- Twelve implementation tasks open.
- Code deletion not started.
- PR #11 merge/rebase is the first prerequisite.

## Decision boundary

Delete unsupported complexity before introducing a generic dispatcher. Audit each exported observer independently, document downstream usage and compatibility, and preserve only behavior with an identified retained consumer or protocol requirement.
