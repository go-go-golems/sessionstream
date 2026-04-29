---
Title: Event Streaming LLM Framework — Clean Room Architecture (2026-04-19)
Ticket: EVT-STREAM-002
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - chat
    - websocket
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/01-architecture-analysis-event-streaming-llm-framework.md
      Note: Architecture analysis derived solely from the two sketches.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md
      Note: Technical architecture document — full Go + TS API specification with the ChatMessage worked example.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md
      Note: Detailed reuse analysis comparing the new framework design against the current pinocchio/pkg/webchat implementation and the older webchat refactor notes.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/04-webchat-reuse-analysis-independent-read-vs-pinocchio-pkg-webchat.md
      Note: Independent reuse analysis (not consulting the colleague's design/03) — file-level reuse audit + 7 design-doc recommendations + 9-step extraction roadmap.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/reference/01-source-image-transcription-2026-04-19-sketches.md
      Note: Verbatim transcription used as the clean-room source of truth.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/sources/building-blocks-page-2.png
      Note: Page 2 of the 2026-04-19 sketch — BUILDING BLOCKS outline + open questions.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/sources/diagram-page-1.png
      Note: Page 1 of the 2026-04-19 sketch — dataflow diagram (SRC/PKG
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-19T13:27:38.251011874-04:00
WhatFor: ""
WhenToUse: ""
---




# Event Streaming LLM Framework — Clean Room Architecture (2026-04-19)

## Overview

Clean-room architectural workup of two hand-drawn sketches dated **2026-04-19** that propose a reusable substrate for realtime, websocket-driven LLM/agent applications (chat, agents, scraper-with-replay, …). The substrate is organised as three layers — **Backend / Generic / Client** — with the Generic layer owning websocket lifecycle, session routing, multiplexing, command dispatch, hydration, and timeline storage so that a concrete application contributes only schemas, commands, and processors.

This ticket exists as a **clean-room** read of the sketches alone; no other repository documents informed the analysis. Per directive, treat it as independent of `EVT-STREAM-001`.

### Documents

- **`design/01-architecture-analysis-event-streaming-llm-framework.md`** — primary deliverable: layer model, themes, seams, ten open questions, recommended first cuts.
- **`design/02-technical-architecture-event-streaming-llm-framework.md`** — public Go + TypeScript API specification for the substrate and its three application slots.
- **`design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`** — comparison against the current `pinocchio/pkg/webchat`, including direct-reuse candidates, mismatches, and recommended extraction strategy.
- **`reference/01-source-image-transcription-2026-04-19-sketches.md`** — verbatim transcription of both pages, with handwriting-uncertainty markers.
- **`sources/diagram-page-1.png`**, **`sources/building-blocks-page-2.png`** — original images.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- architecture
- framework
- event-streaming
- llm
- agents
- chat
- websocket
- backend

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
