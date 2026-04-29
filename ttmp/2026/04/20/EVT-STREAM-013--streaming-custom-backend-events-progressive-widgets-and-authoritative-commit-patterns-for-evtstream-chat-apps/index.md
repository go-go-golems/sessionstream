---
Title: Streaming custom backend events, progressive widgets, and authoritative commit patterns for evtstream chat apps
Ticket: EVT-STREAM-013
Status: active
Topics:
    - agents
    - architecture
    - backend
    - chat
    - event-streaming
    - framework
    - implementation
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Dedicated design ticket for adding progressive custom event previews and authoritative committed custom events to evtstream-backed chat applications, using agentmode in cmd/web-chat as the exemplar."
LastUpdated: 2026-04-20T20:23:27.369049947-04:00
WhatFor: "Collect the design, diary, and delivery artifacts needed to implement and later teach the preview-versus-commit pattern for custom chat widgets on top of evtstream and pinocchio."
WhenToUse: "Use when implementing agentmode previews, designing future custom backend events for evtstream chat apps, or onboarding engineers to the custom-event extension model."
---

# Streaming custom backend events, progressive widgets, and authoritative commit patterns for evtstream chat apps

## Overview

This ticket captures the architecture and implementation plan for a two-phase custom-event model in `evtstream` chat applications:

1. a **progressive preview path** driven by streaming structured parsing in a runtime sink wrapper, and
2. an **authoritative commit path** driven by the final middleware pass after inference completes.

The motivating example is `agentmode` in `cmd/web-chat`, but the deeper purpose is broader: this ticket is intended to become the long-form reference for future contributors who need to add custom backend events, custom projections, custom hydrated entities, and custom frontend widgets to `evtstream`-backed chat applications.

## Key Links

- Design guide: [design-doc/01-intern-guide-to-streaming-custom-events-progressive-widgets-and-authoritative-commit-in-evtstream-chat-apps.md](./design-doc/01-intern-guide-to-streaming-custom-events-progressive-widgets-and-authoritative-commit-in-evtstream-chat-apps.md)
- Contributor playbook: [playbooks/01-contributor-playbook-adding-preview-and-committed-custom-chat-events.md](./playbooks/01-contributor-playbook-adding-preview-and-committed-custom-chat-events.md)
- Diary: [reference/01-diary.md](./reference/01-diary.md)
- Tasks: [tasks.md](./tasks.md)
- Changelog: [changelog.md](./changelog.md)

## Status

Current status: **active**

Completed in this documentation phase:

- dedicated ticket workspace created,
- architecture evidence gathered,
- intern-facing design guide written,
- diary recorded,
- validation and reMarkable delivery completed.

Remaining work is mostly follow-up hardening and any later extraction/reuse work; the implementation slice and short contributor playbook now exist.

## Topics

- agents
- architecture
- backend
- chat
- event-streaming
- framework
- implementation
- onboarding

## Structure

- `design-doc/` — long-form design and implementation guide
- `reference/` — diary and later quick-reference material
- `playbooks/` — future shorter contributor playbooks derived from implementation
- `scripts/` — ticket-scoped helper scripts if needed later
- `sources/`, `various/`, `archive/` — supporting workspace areas retained by docmgr
