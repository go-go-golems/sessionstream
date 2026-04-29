---
Title: Canonical Web Contract and Target cmd/web-chat Package Layout
Ticket: EVT-STREAM-010
Status: active
Topics:
    - architecture
    - backend
    - implementation
    - migration
    - webchat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md
      Note: Main Phase 6 execution guide and slice order.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/03-current-cmd-web-chat-behavior-inventory.md
      Note: Current-state inventory that this target layout/contract is replacing.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/04-preserve-change-drop-matrix.md
      Note: Preserve/change/drop decisions that constrain this target contract.
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Current app shell and route wiring that will eventually be reworked to this target shape.
    - Path: pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Frontend surface that will consume the new canonical contract.
ExternalSources: []
Summary: "Concrete target package layout and canonical web contract for the cmd/web-chat cutover, including the intended app-owned package seams and the HTTP/websocket routes the frontend should migrate to."
LastUpdated: 2026-04-20T12:35:00-04:00
WhatFor: "Give the implementation work a concrete backend/frontend contract and package layout so the next code slices can be implemented directly instead of inferred from broader architecture prose."
WhenToUse: "When implementing the evtstream-backed cmd/web-chat path and deciding exactly which packages, handlers, and route contracts to introduce first."
---

# Canonical Web Contract and Target cmd/web-chat Package Layout

## Executive Summary

This document makes the next coding slices concrete. It answers two questions directly:

1. **What package shape should the new `cmd/web-chat` implementation have?**
2. **What canonical HTTP/websocket contract should the frontend migrate to?**

The key principle is still the same:

> `cmd/web-chat` owns the web contract and app-specific request handling. `evtstream` owns the substrate and app-facing chat runtime seams. The frontend and backend migrate together to one new contract.

## 1. Target package layout inside cmd/web-chat

The current `cmd/web-chat` shell should move toward this shape:

```text
pinocchio/cmd/web-chat/
  main.go
  runtime_composer.go
  profile_policy.go
  app/
    app.go
    server.go
    services.go
    handlers_submit.go
    handlers_snapshot.go
    handlers_ws.go
    handlers_profile.go
    contracts.go
```

### Ownership model

- `main.go`
  - command wiring, flags, root mounting, app bootstrap
- `runtime_composer.go`
  - model/tool/middleware composition policy
- `profile_policy.go`
  - request-level profile/runtime selection policy
- `app/services.go`
  - app-owned evtstream service wiring
- `app/contracts.go`
  - canonical request/response/websocket frame shapes
- `app/handlers_*.go`
  - thin HTTP/websocket handlers that call app services

## 2. Target package layout outside cmd/web-chat

The `evtstream` side should be consumed through packages like:

```text
pinocchio/pkg/evtstream/apps/chat/
  chat.go
  service.go
  ...
```

The rule is:

- `cmd/web-chat` may depend on `pkg/evtstream/apps/chat`
- `pkg/evtstream/apps/chat` may depend on `pkg/evtstream`
- `pkg/evtstream` must not depend on `cmd/web-chat`

## 3. Canonical route contract

This is the proposed canonical route family for the cutover.

### 3.1 Session bootstrap / creation

```http
POST /api/chat/sessions
Content-Type: application/json

{
  "profile": "gpt-5-nano-low"
}
```

Response:

```json
{
  "sessionId": "sess-123",
  "profile": "gpt-5-nano-low"
}
```

Notes:

- This route is optional if the frontend is happy creating a client-side session id first.
- If we want the frontend to create the session id directly, we can skip this route and go straight to the message submit path.

### 3.2 Submit prompt

```http
POST /api/chat/sessions/:sessionId/messages
Content-Type: application/json

{
  "prompt": "Explain ordinals in plain language",
  "profile": "gpt-5-nano-low",
  "idempotencyKey": "optional-client-key"
}
```

Response:

```json
{
  "sessionId": "sess-123",
  "accepted": true,
  "status": "running",
  "profile": "gpt-5-nano-low"
}
```

### 3.3 Session snapshot

```http
GET /api/chat/sessions/:sessionId
```

Response:

```json
{
  "sessionId": "sess-123",
  "ordinal": "42",
  "status": "running",
  "entities": [
    {
      "kind": "ChatMessage",
      "id": "chat-msg-1",
      "tombstone": false,
      "payload": {
        "messageId": "chat-msg-1",
        "text": "Answer: Explain ordinals in plain language",
        "status": "finished",
        "streaming": false
      }
    }
  ]
}
```

### 3.4 Websocket endpoint

```http
GET /api/chat/ws
```

Client frame:

```json
{
  "type": "subscribe",
  "sessionId": "sess-123",
  "sinceOrdinal": "0"
}
```

Server frames:

```json
{ "type": "subscribed", "sessionId": "sess-123", "sinceOrdinal": "0" }
```

```json
{
  "type": "snapshot",
  "sessionId": "sess-123",
  "ordinal": "42",
  "entities": [...]
}
```

```json
{
  "type": "ui-event",
  "sessionId": "sess-123",
  "ordinal": "43",
  "name": "ChatMessageAppended",
  "payload": { ... }
}
```

```json
{ "type": "error", "error": "..." }
```

### 3.5 Profile endpoints

Keep these routes conceptually, even if implementation changes underneath:

- `GET /api/chat/profiles`
- `GET /api/chat/profile`
- `POST /api/chat/profile`

Why keep them?

- The frontend already has a profile-selection surface.
- Profile selection is product behavior we want to preserve.
- The backend implementation can change without redesigning the visible feature.

## 4. Contract decisions that are now explicit

### Replace `conv_id` with `sessionId`

The new canonical contract should be session-centered. The frontend can still keep a stable identifier in the URL or local state, but the canonical field name should move to `sessionId`.

### Stop using `/chat` and `/ws?conv_id=...` as the canonical API

Those were practical shapes for the old runtime. The new runtime should expose a cleaner, explicit app-owned contract.

### Snapshot-before-live is canonical

The websocket contract and snapshot endpoint should align around the same session identity and the same ordinal model.

### Browser-facing ordinals should remain strings

The Phase 2 work already showed why this matters. Browser-facing ordinals should stay stringified to avoid JavaScript precision problems.

## 5. First code slice that should implement this contract

The next meaningful backend/frontend slice should implement all of these together:

1. `pkg/evtstream/apps/chat` as the app-facing runtime package,
2. app-owned `cmd/web-chat` handlers for:
   - submit,
   - snapshot,
   - websocket,
3. frontend client updates that target those exact routes,
4. one browser-validated send → stream → finish flow.

That should be the first slice where the port feels real.

## 6. Deliberate non-goals for this phase

This document is **not** proposing:

- a compatibility layer for the old wire format,
- permanent dual-path backend operation,
- moving `evtstream` into its own repository yet,
- preserving `pkg/webchat/http` handler contracts.

Those would slow down the cutover or belong to later work.

## 7. Open questions

- Should the frontend create `sessionId` client-side, or should the backend own initial session creation via `POST /api/chat/sessions`?
- Do we want a stop endpoint in the canonical HTTP contract, or only a websocket/UI action later?
- Do we keep `app-config.js` as the final bootstrap mechanism, or replace it with a different runtime bootstrap document?

## Final Recommendation

Use this document as the concrete coding target for the next `cmd/web-chat` slices.

The main Phase 6 design doc explains the architecture. This document explains what should actually get coded next:

- package seams,
- handler seams,
- and the canonical route/frame contract the frontend should adopt.

## References

- `design-doc/01-phase-6-implementation-plan.md`
- `design-doc/03-current-cmd-web-chat-behavior-inventory.md`
- `design-doc/04-preserve-change-drop-matrix.md`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
