---
Title: Add hub and websocket observers for sessionstream diagnostics
Ticket: SS-OBSERVERS
Status: active
Topics:
    - sessionstream
    - observability
    - websocket
    - hydration
    - debugging
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-06T21:18:59.225742625-04:00
WhatFor: ""
WhenToUse: ""
---

# Add hub and websocket observers for sessionstream diagnostics

## Overview

This ticket designs and tracks generic observability hooks for `sessionstream`: a Hub pipeline observer and a WebSocket transport observer. The goal is to let applications such as Pinocchio record exact event/projection/hydration/fanout evidence without putting app-specific debug storage or HTTP endpoints into the reusable framework.

The detailed intern-oriented implementation guide is in [design-doc/01-observer-implementation-guide.md](./design-doc/01-observer-implementation-guide.md).

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- sessionstream
- observability
- websocket
- hydration
- debugging

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
