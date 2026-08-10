---
Title: Timed Failure Detector and WebSocket Heartbeat State Machine
Ticket: SESSIONSTREAM-005
Status: complete
Topics:
    - websocket
    - architecture
    - event-streaming
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources:
    - https://github.com/go-go-golems/sessionstream/pull/10
Summary: Build a deterministic timed failure-detector state machine and per-connection supervisor for Sessionstream application-level WebSocket heartbeat handling.
LastUpdated: 2026-08-10T18:13:33.405605492-04:00
WhatFor: Tracking design and implementation of explicit, testable WebSocket heartbeat semantics.
WhenToUse: Use this ticket for all work that extracts heartbeat state from ad hoc timers and pong channels into a pure reducer and supervisor.
---


# Timed Failure Detector and WebSocket Heartbeat State Machine

## Overview

SESSIONSTREAM-005 turns Sessionstream's application-level WebSocket ping/pong handling into an explicit timed failure detector. The target architecture uses a pure state/event/action reducer, one per-connection supervisor, generation-safe timers, actual-write-based deadlines, and observer-independent control processing.

The ticket follows PR #10's lifecycle hardening. It preserves the current protobuf-JSON wire frames and public connection configuration while replacing implicit coordination among ticker, timer, pong channel, writer acknowledgements, and close state.

## Key links

- [Intern design and implementation guide](design-doc/01-intern-guide-to-the-timed-failure-detector-and-websocket-heartbeat-state-machine.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Implementation tasks](tasks.md)
- [Changelog](changelog.md)
- [Sessionstream PR #10](https://github.com/go-go-golems/sessionstream/pull/10)

## Status

Current status: **active**

Research, architecture, implementation guidance, and delivery are complete. Code implementation remains intentionally open and is divided into phases in `tasks.md`.

## Architectural direction

```text
socket reader/writer events
          |
          v
per-connection supervisor
          |
          v
pure timed failure-detector reducer
          |
          v
ordered actions: write, timer, observe, close
```

## Topics

- websocket
- architecture
- event-streaming
- onboarding
