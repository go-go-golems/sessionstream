---
Title: Implementation Diary
Ticket: SS-WS-RACE
Status: active
Topics:
    - sessionstream
    - websocket
    - hydration
    - reconnect
    - streaming
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Implemented hydrating/live subscription state and fanout buffering.
    - Path: pkg/sessionstream/transport/ws/server_test.go
      Note: Added deterministic regression tests for subscribe hydration race.
ExternalSources: []
Summary: Chronological implementation diary for the WebSocket subscribe race fix.
LastUpdated: 2026-05-07T00:00:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Implementation Diary

## Goal

This diary records the implementation of the WebSocket subscribe race fix: a reconnecting browser should receive a snapshot first, then any live UI events emitted after that snapshot ordinal, without losing events during snapshot loading.

## Step 1: Subscribe-first hydration buffering

The first implementation step replaced the snapshot-before-subscribe gap with a hydrating subscription state. A connection is now registered as subscribed before snapshot loading begins, but it is marked `hydrating`. Live fanout sees the connection and buffers UI event batches instead of sending them before the snapshot.

After the snapshot is queued, the server drains buffered batches with ordinals greater than the snapshot ordinal, sends them after the snapshot, marks the subscription live, and finally sends the subscribed frame. This gives the browser a coherent story: durable state up to N, then live UI events after N.

### Prompt Context

**User prompt (verbatim):** "Alright, do it all, task by task, committing at appropriate inter vvals, and keeping a detailed diary"

**Assistant interpretation:** Implement the race-fix ticket incrementally after adding observers, with tests and a detailed diary.

**Inferred user intent:** The user wants the documented reconnect race fixed in Sessionstream itself, not merely diagnosed from Pinocchio.

**Commit (code):** pending — first race-fix commit will include hydrating subscription state, buffering, observer stages, and regression tests.

### What I did

- Added `subscriptionStateHydrating` and `subscriptionStateLive`.
- Extended `subscription` with `state`, `snapshotOrdinal`, and a per-subscription buffer.
- Added `bufferedUIBatch` for ordinal-preserving UI event batches.
- Added `WithHydrationBufferLimit` and a default buffer limit.
- Added helper methods: `registerHydrating`, `subscriptionState`, `bufferHydrationEvents`, `drainHydrationBuffer`, `markLive`, and `sendUIBatch`.
- Modified subscribe handling to register hydrating before snapshot load.
- Modified `PublishUI` to buffer for hydrating targets and send directly for live targets.
- Filtered buffered batches with `ordinal > snapshotOrdinal` to prevent duplicates.
- Added transport observer stages for buffered events, buffer flush, subscription live, and overflow.
- Added deterministic tests for buffering during snapshot load, duplicate prevention, live/hydrating multi-tab behavior, and buffer overflow.
- Ran `go test ./pkg/sessionstream/... -count=1` successfully.

### Why

The old flow loaded and sent the snapshot before adding the connection to `bySession`. A live UI event emitted in that window had no target and could be missed by a reloaded browser. Registering the connection as hydrating closes that target-selection gap while preserving snapshot-first ordering.

### What worked

- The race could be tested deterministically with a snapshot provider blocked on channels.
- Per-connection buffering lets a live tab receive events immediately while a second tab is still hydrating.
- The observer stages from `SS-OBSERVERS` make the new sequence visible in tests and future Pinocchio traces.

### What didn't work

- N/A for this step. The implementation compiled and the targeted package tests passed.

### What I learned

The safe fix is not simply “subscribe before snapshot.” That would deliver live UI events before the snapshot and risk the frontend clearing them during hydration. The subscription has to be visible to fanout but not live to the client until the snapshot is sent.

### What was tricky to build

The tricky part was lock and send ordering. The implementation copies buffered batches under the connection lock, then sends frames outside the lock. It also avoids holding the server map lock while queueing WebSocket frames. This preserves the existing non-blocking fanout shape.

### What warrants a second pair of eyes

- Whether `subscribed` should be queued before or after `markLive`. The current implementation queues snapshot, buffered UI events, marks live, then queues subscribed.
- Whether overflow should close the connection immediately. The current implementation sends an error frame if possible and closes, rather than silently dropping events.
- Whether only `maxBatches` is sufficient, or whether a separate max-event/max-byte limit should be added later.

### What should be done in the future

- Run race detector for the WebSocket package when feasible.
- Use the new observer traces from Pinocchio to validate reload-during-streaming in a real browser.

### Code review instructions

Start with `pkg/sessionstream/transport/ws/server.go`, especially the subscribe branch, `PublishUI`, and the new subscription helpers. Then review the new tests in `pkg/sessionstream/transport/ws/server_test.go`.

Validate with:

```bash
go test ./pkg/sessionstream/... -count=1
go test ./pkg/sessionstream/transport/ws -race -count=1
```

### Technical details

The key invariant is:

```text
snapshot ordinal N is queued first;
only buffered UI batches with ordinal > N are flushed after it;
then the subscription becomes live.
```
