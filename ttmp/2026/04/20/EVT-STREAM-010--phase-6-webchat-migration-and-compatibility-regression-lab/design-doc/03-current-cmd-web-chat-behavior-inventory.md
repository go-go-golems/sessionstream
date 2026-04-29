---
Title: Current cmd/web-chat Behavior Inventory
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
    - Path: pinocchio/cmd/web-chat/README.md
      Note: Current public route and runtime behavior summary for the application shell.
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Current route composition and dependency wiring for the live app.
    - Path: pinocchio/cmd/web-chat/profile_policy.go
      Note: Current app-owned request resolution and profile selection behavior.
    - Path: pinocchio/pkg/webchat/http/api.go
      Note: Current HTTP/websocket handler contracts and request body shapes.
    - Path: pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Current frontend submit/bootstrap flow.
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Current frontend websocket + hydration flow.
ExternalSources: []
Summary: "Inventory of the current public/runtime behavior of cmd/web-chat that matters for the evtstream port, including routes, frontend-visible flows, legacy request semantics, and the current implementation ownership split between cmd/web-chat and pkg/webchat."
LastUpdated: 2026-04-20T11:40:00-04:00
WhatFor: "Capture the behavior that must be consciously preserved, replaced, or dropped during the Phase 6 cutover."
WhenToUse: "When implementing the cmd/web-chat port and deciding what the new canonical evtstream-backed app contract must cover."
---

# Current cmd/web-chat Behavior Inventory

## Executive Summary

This document records the current `cmd/web-chat` behavior that matters for the Phase 6 cutover. It is not trying to preserve the old stack forever. It is trying to make the migration honest.

The practical question is:

> What does the current application actually do today that the new evtstream-backed application must either preserve as product behavior, replace with a clearer canonical behavior, or intentionally drop?

At the moment, `cmd/web-chat` is still a **handler-first shell over `pkg/webchat`**. The app owns route composition and request policy, but the live runtime path still depends heavily on `pkg/webchat` and `pkg/webchat/http`.

## 1. Current ownership split

### What `cmd/web-chat` already owns

From `pinocchio/cmd/web-chat/main.go` and `pinocchio/cmd/web-chat/README.md`, the app currently owns:

- route composition for `/chat` and `/ws`,
- profile/runtime request resolution,
- profile API mounting,
- root-prefix mounting behavior,
- static UI serving,
- `app-config.js` runtime config serving,
- registration of additional timeline/thinking-mode handlers.

### What `pkg/webchat` still owns in practice

The live runtime underneath still comes from `pkg/webchat`:

- server construction via `webchat.NewServer(...)`,
- chat submission service,
- websocket attachment semantics,
- timeline hydration service,
- internal conversation/runtime lifecycle.

This means the current app is **app-owned at the edge, but not yet substrate-owned at runtime**.

## 2. Current frontend-visible routes

Based on the current README and the route wiring in `cmd/web-chat/main.go`, the important frontend-visible routes are:

### Core routes

- `POST /chat`
- `POST /chat/{runtime}`
- `GET /ws?conv_id=<id>`
- `GET /api/timeline?conv_id=<id>&since_version=<n>&limit=<n>`

### Profile routes

- `GET /api/chat/profiles`
- `GET /api/chat/profile`
- `POST /api/chat/profile`

### Debug routes

Mounted only when `--debug-api` is enabled:

- `GET /api/debug/*`
- turn/timeline-debug routes documented in `cmd/web-chat/README.md`

### Static/config routes

- `/`
- `/app-config.js`
- root-prefixed variants when `--root` is set

## 3. Current request/bootstrap model

### 3.1 Initial frontend load

The current frontend boot path is:

```text
1. Browser loads UI shell.
2. Frontend reads runtime config from app-config.js.
3. ChatWidget reads conv_id from the URL.
4. If conv_id exists, frontend connects websocket.
5. Frontend hydrates via GET /api/timeline.
6. Buffered websocket frames replay after hydration.
```

### 3.2 Current chat submit model

The current frontend sends prompts using `POST /chat` with a body based on `pkg/webchat/http/api.go` and the current frontend code:

```json
{
  "prompt": "hello",
  "conv_id": "optional-conversation-id",
  "profile": "optional-profile-slug-or-runtime-key",
  "registry": "optional-registry-slug",
  "idempotency_key": "optional-client-idempotency-key"
}
```

Legacy body aliases also still exist in the current backend request types even though the README already treats them as deprecated or removed from the intended hard-cut contract:

- `runtime_key`
- `registry_slug`
- some older selector semantics handled in request policy code

### 3.3 Current websocket model

The current websocket path is:

```text
GET /ws?conv_id=<id>
  -> request resolver validates request
  -> websocket upgraded
  -> pkg/webchat stream service attaches socket
  -> hello/ping/pong + SEM/live frames handled through old hub path
```

The important point is not the exact frame shape. The important point is that the frontend currently expects:

- connection by URL + `conv_id`,
- live streaming updates after attach,
- reconnect to an existing conversation,
- some sequencing relationship between websocket frames and timeline hydration.

### 3.4 Current hydration model

The current hydration path is:

```text
GET /api/timeline?conv_id=...
  -> returns TimelineSnapshotV2 payload
  -> frontend clears normalized state
  -> frontend rehydrates from snapshot
  -> buffered websocket frames replay afterward
```

## 4. Current frontend behavior that matters

From `ChatWidget.tsx`, `wsManager.ts`, and `profileApi.ts`, the current user-visible behavior includes:

- submit prompt,
- generate or reuse a conversation id,
- write conversation id into the URL,
- connect websocket for an existing conversation,
- hydrate timeline state,
- buffer websocket frames before hydration completes,
- select active profile,
- start a new conversation from the UI,
- show connection/status/error state,
- render timeline entities as the main chat transcript.

These are product-level behaviors that the new path must account for.

## 5. Current app/runtime seams that matter for the port

### Request resolution seam

`cmd/web-chat/profile_policy.go` currently resolves:

- `conv_id`
- profile/registry/runtime selection
- normalized request policy for both HTTP and websocket flows

This is likely to survive conceptually, though the exact output type should become app-specific for the new evtstream path rather than `pkg/webchat/http` specific.

### Chat handler seam

`pkg/webchat/http/api.go` currently provides:

- `NewChatHandler(...)`
- `NewWSHandler(...)`
- `NewTimelineHandler(...)`

These are important donor seams, but they should be replaced by app-owned handlers that target evtstream-backed services instead of `pkg/webchat` services.

### Frontend state seam

The current frontend assumes a specific set of backend interactions:

- `/chat`
- `/ws`
- `/api/timeline`
- profile APIs

That means the port cannot only replace backend code. It must define and implement the new frontend-facing contract intentionally.

## 6. Current legacy assumptions worth calling out explicitly

These assumptions matter because they can quietly survive a migration if nobody names them.

### Legacy identity assumption

The current stack is still conversation-id centered (`conv_id`), while the new substrate is session-id centered.

### Legacy transport assumption

The current frontend still thinks in terms of a URL-attached websocket that begins streaming into a legacy frame model.

### Legacy hydration assumption

The current frontend fetches a timeline snapshot from `/api/timeline` and then replays buffered live frames.

### Legacy runtime-selection assumption

The current request resolution layer still reflects profile/runtime selection logic inherited from the webchat era.

## 7. What this inventory says about the port

The cutover must cover at least these concrete behaviors:

- canonical prompt submission,
- canonical session identification,
- live streaming over websocket,
- coherent snapshot hydration,
- profile selection,
- connection/error/status visibility,
- new-conversation reset,
- durable history/timeline visibility.

The old stack’s exact request and frame shapes do **not** need to survive. But the user-facing workflow definitely does.

## 8. Immediate implications for implementation

This inventory implies the first real migrated slice should include all of the following together:

1. backend submit path,
2. backend snapshot path,
3. backend websocket live path,
4. frontend client update for those three paths,
5. one browser-validated happy-path prompt flow.

That is the smallest slice that proves the migration is real rather than theoretical.

## References

- `pinocchio/cmd/web-chat/README.md`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/web-chat/profile_policy.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
