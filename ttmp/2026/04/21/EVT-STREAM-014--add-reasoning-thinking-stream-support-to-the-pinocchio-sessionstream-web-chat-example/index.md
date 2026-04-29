---
Title: Add reasoning/thinking stream support to the pinocchio sessionstream web-chat example
Ticket: EVT-STREAM-014
Status: active
Topics:
    - chat
    - architecture
    - backend
    - event-streaming
    - llm
    - implementation
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Ticket workspace for investigating and planning visible reasoning/thinking support in the live canonical cmd/web-chat application, including a detailed intern guide, diary, validation, and reMarkable delivery.
LastUpdated: 2026-04-21T19:25:00-04:00
WhatFor: Track the analysis and future implementation work required to surface Geppetto reasoning/thinking streams in the sessionstream-based example web-chat.
WhenToUse: Use this ticket when implementing or reviewing reasoning/thinking UI support for the canonical cmd/web-chat path.
---

# Add reasoning/thinking stream support to the pinocchio sessionstream web-chat example

## Overview

This ticket captures the analysis and implementation plan for adding visible model reasoning/thinking support to the live canonical `pinocchio/cmd/web-chat` application.

The core finding is that Geppetto already emits reasoning-related events, but the current sessionstream-backed app only projects normal assistant/user chat messages plus `agentmode`. The recommended approach is to add reasoning support as a new **app-owned feature** in `cmd/web-chat`, not as a `sessionstream` core change and not as a revival of deleted legacy `thinkingmode` code.

## Key Links

- Main design doc:
  - `design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md`
- Diary:
  - `reference/01-diary.md`
- Tasks:
  - `tasks.md`
- Changelog:
  - `changelog.md`

## Status

Current status: **active**

This ticket is now implemented for the first visible-reasoning slice. The canonical web-chat path now persists and renders `thinking` messages, delays empty assistant/thinking placeholders until real bytes arrive, auto-follows long streaming responses, and respects manual scroll detachment until the user returns near bottom or clicks "Jump to latest". The only explicitly deferred item from the original planning list is whether to add separate `reasoning_tokens` UI treatment later.

## Topics

- chat
- architecture
- backend
- event-streaming
- llm
- implementation
- onboarding

## Tasks

See [tasks.md](./tasks.md) for the detailed task list.

## Changelog

See [changelog.md](./changelog.md) for the running history.

## Structure

- `design-doc/` — primary intern guide and implementation plan
- `reference/` — investigation diary and supporting reference material
- `playbooks/` — future smoke/validation playbooks if implementation begins
- `scripts/` — temporary ticket-local tooling if needed later
- `sources/` — optional copied artifacts or notes if future validation generates them
