---
Title: Preserve Change Drop Matrix for cmd/web-chat Port
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
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/03-current-cmd-web-chat-behavior-inventory.md
      Note: Current behavior inventory used as the source for this matrix.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md
      Note: Main cutover design and canonical contract direction.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/02-web-frontend-migration-and-design-guide.md
      Note: Frontend counterpart explaining product-level continuity requirements.
ExternalSources: []
Summary: "Explicit preserve/change/drop matrix for the cmd/web-chat cutover so the team can distinguish product behaviors that must survive from legacy wire/runtime details that should be replaced or intentionally removed."
LastUpdated: 2026-04-20T11:40:00-04:00
WhatFor: "Provide a reviewable decision matrix for the Phase 6 cutover before implementation proceeds too far."
WhenToUse: "When deciding what the new canonical cmd/web-chat contract must preserve, what it should replace, and what legacy behavior should be dropped entirely."
---

# Preserve Change Drop Matrix for cmd/web-chat Port

## Executive Summary

This matrix turns the Phase 6 migration into explicit decisions instead of vague instincts. The point is not to preserve everything old. The point is to preserve the product where it matters and to stop preserving old runtime details that no longer serve the architecture.

The working rule is:

> Preserve product behavior. Replace legacy runtime/wire behavior with a cleaner canonical evtstream-backed contract. Drop historical details that only existed because of the old `pkg/webchat` implementation.

## Decision Table

| Area | Current Behavior | Decision | Rationale |
|---|---|---|---|
| Prompt submission | User types prompt and gets a streamed assistant response | **Preserve** | Core product workflow |
| New conversation | User can reset and start a new conversation/session | **Preserve** | Core product workflow |
| Profile selection | User can inspect and switch profile/runtime before subsequent sends | **Preserve** | Core product workflow |
| Timeline/history visibility | User can see current and completed transcript/timeline state | **Preserve** | Core product workflow |
| Reload/reconnect coherence | Reloading an existing conversation/session yields a coherent snapshot and continued live updates | **Preserve** | Core product workflow |
| Header/status/error visibility | User can observe connection/runtime/error state | **Preserve** | Core product workflow |
| `conv_id` naming | Conversation identity is represented and routed as `conv_id` | **Replace** | New substrate should be session-centered; frontend can migrate in lockstep |
| `GET /ws?conv_id=...` attach model | Websocket identity/subscription is conveyed only through URL query | **Replace** | New app should own a canonical websocket contract |
| Legacy websocket hello/ping/pong semantics | Old `pkg/webchat` websocket lifecycle details | **Drop unless consciously reintroduced as canonical** | Old transport details should not control the new architecture |
| `/api/timeline` snapshot schema | TimelineSnapshotV2 + current hydration contract | **Replace** | New snapshot endpoint/schema should match evtstream-native session/snapshot semantics |
| Old websocket frame schema | SEM-ish / legacy live frame model | **Replace** | Frontend and backend migrate together to a new canonical event model |
| Legacy request body aliases | `runtime_key`, `registry_slug`, other historical selectors | **Drop** | These are implementation leftovers, not product behavior |
| `POST /chat/{runtime}` route shape | Runtime selected by route segment | **Drop or consciously redesign later** | Cleaner explicit request/body/profile model is preferable |
| Debug routes tied to old runtime | Old `/api/debug/*` routes that only exist to inspect webchat internals | **Review individually** | Some may remain useful, but only if they still make sense after the cutover |
| Root-prefix serving | `--root` mounting behavior and runtime config prefix support | **Preserve** | Deployment/product integration behavior |
| `app-config.js` bootstrap concept | Frontend receives runtime mount/debug config from backend | **Preserve or replace with equivalent canonical bootstrap** | App still needs a runtime bootstrap mechanism |
| Frontend buffering before hydration completes | Frames are buffered until snapshot load finishes | **Replace with equivalent coherent snapshot-before-live logic** | Product coherence matters more than exact buffering implementation |

## Canonical interpretation

### Preserve as product behavior

These must survive in some recognizable form:

- send prompt,
- stream assistant output,
- switch profiles,
- start a new session,
- reload/reconnect coherently,
- see transcript/timeline state,
- see connection/error state,
- support root-prefix deployment.

### Replace with a new canonical contract

These should change rather than be carried forward:

- session identity naming and routing,
- snapshot endpoint shape,
- websocket subscribe/live protocol,
- frontend state normalization logic,
- request/response payload shapes.

### Drop unless a new reason appears

These should not be preserved just because they existed:

- legacy request aliases,
- legacy hello/ping/pong behavior,
- runtime-selection leftovers tied to the old stack,
- old debug surfaces that only expose `pkg/webchat` internals.

## Immediate implementation consequence

The first end-to-end migrated slice must preserve the product behaviors in the first category while intentionally replacing the runtime details in the second category.

That means the slice should prove:

- send prompt,
- receive streaming output,
- reconnect via snapshot-before-live,
- select profile,
- observe status/error state,
- all using the new canonical backend/frontend contract.

## Open items requiring later confirmation

- Which debug routes still matter after the cutover?
- Whether root-prefix bootstrap stays as `app-config.js` specifically or becomes a different canonical bootstrap mechanism.
- Whether `POST /chat` remains the final canonical submit route or is replaced by a more explicit session-based route family.

## References

- `design-doc/03-current-cmd-web-chat-behavior-inventory.md`
- `design-doc/01-phase-6-implementation-plan.md`
- `design-doc/02-web-frontend-migration-and-design-guide.md`
