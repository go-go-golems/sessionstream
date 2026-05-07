---
Title: Fix websocket subscribe snapshot race during streaming reconnect
Ticket: SS-WS-RACE
Status: active
Topics:
    - sessionstream
    - websocket
    - hydration
    - reconnect
    - streaming
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-06T21:18:59.278592785-04:00
WhatFor: ""
WhenToUse: ""
---

# Fix websocket subscribe snapshot race during streaming reconnect

## Overview

This ticket designs and tracks a fix for the WebSocket subscribe race where a reloaded browser can miss live UI events emitted after snapshot load but before subscription registration. The proposed implementation registers subscriptions as hydrating before snapshot load, buffers live UI events for that connection, sends the snapshot first, then flushes buffered events with ordinals greater than the snapshot ordinal.

The detailed intern-oriented implementation guide is in [design-doc/01-websocket-subscribe-race-fix-guide.md](./design-doc/01-websocket-subscribe-race-fix-guide.md).

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- sessionstream
- websocket
- hydration
- reconnect
- streaming

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
