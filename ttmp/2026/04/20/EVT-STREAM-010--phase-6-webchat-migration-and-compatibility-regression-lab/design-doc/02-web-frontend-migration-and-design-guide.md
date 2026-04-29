---
Title: Web Frontend Migration and Design Guide for cmd/web-chat on evtstream
Ticket: EVT-STREAM-010
Status: active
Topics:
    - architecture
    - frontend
    - webchat
    - migration
    - implementation
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md
      Note: Parent Phase 6 migration playbook focused on the backend cutover and overall app/runtime architecture.
    - Path: pinocchio/cmd/web-chat/web/src/App.tsx
      Note: Current frontend app entrypoint that switches between chat UI and debug UI.
    - Path: pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Current main chat widget, route/query bootstrap logic, and prompt submission flow.
    - Path: pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Current websocket connection, hydration bootstrap, buffering, and SEM-dispatch logic.
    - Path: pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts
      Note: Current normalized timeline entity store used by the chat UI.
    - Path: pinocchio/cmd/web-chat/web/src/store/profileApi.ts
      Note: Current RTK Query profile endpoints and current-profile selection behavior.
    - Path: pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx
      Note: Current rendering path for timeline entities and error panel behavior.
    - Path: pinocchio/cmd/web-chat/web/src/webchat/components/Composer.tsx
      Note: Current send/new-conversation interaction surface.
    - Path: pinocchio/cmd/web-chat/web/src/config/runtimeConfig.ts
      Note: Runtime config bootstrap used to derive base prefix and debug-API behavior.
    - Path: pinocchio/pkg/evtstream/transport/ws/server.go
      Note: New websocket transport that the frontend should target after the Phase 6 port.
ExternalSources: []
Summary: "Detailed intern-facing guide for the cmd/web-chat frontend migration during the evtstream cutover. Explains the existing frontend architecture, which UX/functionality must stay stable, what contracts may change freely, and how to redesign the frontend integration around evtstream-native APIs without changing the product feel."
LastUpdated: 2026-04-20T08:02:00-04:00
WhatFor: "Guide the frontend work required for Phase 6 so the web UI keeps the same design and functionality while the backend runtime moves to evtstream."
WhenToUse: "When planning or implementing the web frontend side of the cmd/web-chat migration onto evtstream, or when onboarding a contributor onto the frontend half of the cutover."
---

# Web Frontend Migration and Design Guide for cmd/web-chat on evtstream

## Executive Summary

This document is the **frontend companion** to the main Phase 6 migration playbook. The Phase 6 backend document explains how `cmd/web-chat` should move off `pkg/webchat` and onto `evtstream`. This document explains what that means for the **web frontend**.

The most important rule for the frontend migration is this one:

> **The visible UI design and user-facing functionality should remain substantially the same, even if the HTTP APIs, websocket message formats, and hydration payloads change completely underneath it.**

That sentence captures the migration strategy precisely.

We are **not** aiming for wire-level backwards compatibility. We are **not** trying to preserve old `/chat`, `/ws`, and `/api/timeline` payload shapes simply because they already exist. The frontend is allowed to change its API client code, websocket handling, and state-normalization logic as needed.

The intended end state is a **clean cut**, not a compatibility bridge: frontend and backend should migrate together onto one new canonical `cmd/web-chat` web contract, and the runtime path should not carry temporary compatibility adapters unless a concrete operational need is identified.

But we **are** aiming for product-level continuity:

- the chat UI should still look like the current chat UI,
- the composer should still work like the current composer,
- live streaming should still feel like live streaming,
- reconnect and hydration should still feel coherent,
- profile selection should still be present,
- timeline/history behavior should still exist,
- and any user-visible debug affordances we care about should still work.

In other words:

```text
What may change freely:
- route names
- JSON payloads
- websocket frame schemas
- internal frontend state shapes
- bootstrap sequence details

What should remain stable first:
- visible product design
- interaction flow
- user-facing functionality
- mental model of the app
```

This document is written as a detailed guide for a **new intern**. It explains:

- what the current frontend looks like,
- how it is wired today,
- what functionality must be preserved,
- what code should be audited first,
- how to redesign the integration around `evtstream`,
- what frontend architecture should exist after the cutover,
- and how to validate that the migration preserved the product rather than accidentally redesigning it.

---

## 1. Frontend Migration Goal

A backend migration often fails because the frontend is treated as a passive consumer that will "just adapt later." That is not safe here.

The current `cmd/web-chat` frontend is not merely a thin shell. It already embeds assumptions about:

- conversation identity,
- websocket timing,
- hydration behavior,
- profile selection,
- timeline entity rendering,
- and the ordering in which state becomes visible.

So the frontend work in Phase 6 is not an afterthought. It is one of the two major halves of the cutover.

### The actual frontend goal

The goal is **not**:

- preserve all old API contracts,
- preserve all old websocket frame shapes,
- preserve all old internal Redux state conventions,
- add a long-lived backend compatibility layer just to protect the old frontend.

The goal **is**:

- preserve the **same frontend product feel**,
- preserve the **same visible features**,
- refactor the frontend to target the new `evtstream`-native backend,
- and keep the result understandable enough that future backend evolution does not require another architecture reset.

### A useful way to think about the work

Think of the Phase 6 frontend work as a **transport and data-contract migration under a stable product skin**.

That means:

- same visual design,
- same major component hierarchy,
- same main interactions,
- but new integration boundaries.

---

## 2. What the Current Frontend Looks Like

Before a contributor can redesign the frontend integration, they need to understand the current one.

### Entry point

Start here:

- `pinocchio/cmd/web-chat/web/src/App.tsx`

This file is small, but important. It shows that the frontend has two top-level modes:

- the normal chat UI (`ChatWidget`),
- the debug UI (`DebugUIApp`) selected by `?debug=1`.

So from the beginning, the frontend is already a **two-surface app**:

```text
App
  -> normal mode: ChatWidget
  -> debug mode: DebugUIApp
```

### Main chat surface

The main app surface is:

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`

This is the most important frontend file for Phase 6.

It currently owns:

- initial `conv_id` bootstrap from the URL,
- syncing that conversation id back into the URL,
- connecting the websocket manager,
- sending prompts via `fetch(.../chat)`,
- selecting and switching profiles,
- resetting the conversation state,
- composing the main widget layout,
- passing normalized timeline entities into the renderer pipeline.

### Main chat UI composition

At a conceptual level, the current chat UI is:

```text
ChatWidget
  -> Header / Statusbar
  -> Timeline
  -> Composer
```

Key component files:

- `src/webchat/components/Header.tsx`
- `src/webchat/components/Statusbar.tsx`
- `src/webchat/components/Timeline.tsx`
- `src/webchat/components/Composer.tsx`

These are not just implementation details. They define the visible product structure that should remain stable during the migration.

### Current state layers

The main state for the normal chat UI currently lives in:

- `src/store/appSlice.ts`
- `src/store/errorsSlice.ts`
- `src/store/timelineSlice.ts`
- `src/store/profileApi.ts`
- `src/store/store.ts`

Of these, the most important for migration are:

- `timelineSlice.ts`
- `profileApi.ts`
- the websocket manager

because those are the places where backend assumptions are encoded.

### Current websocket integration

The critical file here is:

- `src/ws/wsManager.ts`

This is where the current frontend assumes a lot about backend behavior:

- how to connect,
- how to hydrate,
- how to buffer websocket frames before hydration completes,
- how to parse SEM envelopes,
- how timeline entities get inserted into state.

This file is one of the most important places that will need redesign in Phase 6.

### Current runtime config

Read:

- `src/config/runtimeConfig.ts`
- `src/utils/basePrefix.ts`

These determine how the frontend discovers the backend mount prefix and whether debug API features are enabled. They are important because the cutover should preserve correct mounting behavior even if routes change.

---

## 3. Current Frontend Runtime Flow

A new intern should understand the current runtime behavior end to end before changing anything.

### Current normal flow

Today, the frontend roughly behaves like this:

```text
1. App loads.
2. ChatWidget reads conv_id from URL.
3. ChatWidget connects websocket if conv_id exists.
4. wsManager hydrates from /api/timeline?conv_id=...
5. Buffered websocket frames are replayed after hydration.
6. User types prompt.
7. ChatWidget POSTs /chat with { conv_id, prompt }.
8. Websocket frames continue updating live state.
9. Timeline renders normalized entities.
```

### Current pseudocode sketch

```ts
if (convId exists) {
  ws.connect(convId)
  hydrateTimeline(convId)
  replayBufferedFrames()
}

onSend(prompt) {
  ensureConvId()
  ensureWsConnected()
  POST /chat { conv_id, prompt }
}
```

### Why this matters

The frontend migration must preserve the **effect** of this flow, but it does not need to preserve the exact implementation.

That means:

- the frontend still needs a stable notion of session/chat identity,
- the frontend still needs a coherent bootstrap + live-update sequence,
- the frontend still needs a send path,
- the frontend still needs rendered timeline/history entities.

But it no longer needs to depend on:

- old SEM envelope shapes,
- old timeline snapshot schema,
- old websocket frame names,
- or old `conv_id`-centric contracts if the app chooses to modernize those.

---

## 4. What Must Stay the Same

This section is the most important product-level constraint in the document.

### The visible UI should ideally stay the same

That means the Phase 6 port should not become a design rewrite. The following should remain stable first:

- overall chat layout,
- composer placement and behavior,
- streamed-response reading experience,
- header/status/profile selection concepts,
- conversation-reset/new-conversation flow,
- timeline rendering style and content model,
- debug mode entrypoint behavior (`?debug=1`) unless intentionally redesigned later.

### The functionality that must survive

At minimum, the migrated frontend must preserve:

- submit prompt,
- create a new conversation/session,
- live stream assistant output,
- show durable history / timeline state,
- reconnect to an existing running or completed session coherently,
- switch profile/runtime,
- surface runtime/connectivity state,
- keep error handling visible.

### Product-level parity, not contract-level parity

This distinction should be explicit for every reviewer:

```text
Required:
- same user-visible features
- same user-visible workflow
- same overall look and feel

Not required:
- same HTTP payloads
- same websocket envelopes
- same bootstrap JSON schema
- same internal Redux shapes
```

---

## 5. What May Change Freely

Because the backend cutover is real, the frontend must be allowed to change the integration layer decisively.

### Allowed changes

The frontend may change:

- route names,
- request body shapes,
- query parameter names,
- websocket subscribe protocol,
- hydration payload schema,
- entity normalization logic,
- buffering strategy,
- state slices and selectors,
- error classification and reporting details.

The backend should change with it. The target is one shared new contract, not a frontend rewrite layered on top of backend compatibility shims.

### Why that freedom matters

If the frontend is forced to preserve legacy backend contracts exactly, then the backend migration becomes artificially constrained and we risk re-encoding `pkg/webchat` assumptions into the new app.

That would defeat the purpose of the cutover.

So the frontend should be free to become an **`evtstream`-native web client** while still keeping the same product behavior.

---

## 6. The New Frontend Target: evtstream-Native App, Same Product

After the migration, the frontend should think in `evtstream` terms, even if the app shell still uses `cmd/web-chat` naming on the surface.

### The new mental model

The current frontend has a lot of legacy assumptions centered on `conv_id`, old websocket frames, and SEM/timeline bootstrap details.

The new frontend target should instead think in terms of:

- session identity,
- submit command,
- snapshot hydrate,
- live UI events,
- durable timeline entities,
- reconnect via snapshot-before-live.

### Proposed frontend runtime model

```text
Page load
  -> resolve session id from route / app state
  -> load initial snapshot from evtstream-backed API
  -> connect websocket transport
  -> subscribe to session
  -> receive snapshot first if using websocket bootstrap path
  -> process live UI events after snapshot
```

### Proposed frontend pseudocode

```ts
async function openSession(sessionId: string) {
  setActiveSession(sessionId)
  const snapshot = await api.getSessionSnapshot(sessionId)
  timeline.replaceFromSnapshot(snapshot)

  await ws.connect()
  await ws.subscribe({ sessionId })
}

async function sendPrompt(prompt: string) {
  const sessionId = ensureSessionId()
  await api.submitChat({ sessionId, prompt, profile })
}

ws.onUIEvent((event) => {
  timeline.applyUIEvent(event)
})
```

Notice the difference: the frontend is no longer structured around "parse miscellaneous SEM frames and project them locally." Instead, it should consume clearer app-facing APIs that are already shaped around the new runtime.

---

## 7. Proposed Frontend Architecture After the Cutover

The current frontend can keep much of its visual structure while changing its integration layer.

### Suggested layering

```text
UI components
  -> ChatWidget / Timeline / Composer / Header

Feature/controller layer
  -> session controller
  -> chat submit controller
  -> profile controller
  -> websocket live-sync controller

Data/client layer
  -> chat API client
  -> snapshot API client
  -> websocket client

State layer
  -> app/ui state
  -> timeline entity state
  -> profile state
  -> error state
```

### What should change most

The biggest rewrite targets are probably:

- `src/ws/wsManager.ts`
- parts of `src/webchat/ChatWidget.tsx`
- `src/store/timelineSlice.ts` normalization assumptions
- the API client layer for submit/hydrate/profile interaction

### What should change least

These should ideally stay closest to current shape:

- `src/webchat/components/Composer.tsx`
- `src/webchat/components/Timeline.tsx`
- `src/webchat/components/Header.tsx`
- `src/webchat/components/Statusbar.tsx`
- `src/webchat/styles/*`

Because those largely define the product design the user wants to preserve.

---

## 8. Current Key Files to Audit Before Changing Anything

A new intern should read the following files carefully.

### Entry and app shell

- `pinocchio/cmd/web-chat/web/src/App.tsx`
- `pinocchio/cmd/web-chat/web/src/main.tsx`

### Main chat widget

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/types.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/rendererRegistry.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/profileSelection.ts`

### Main components

- `pinocchio/cmd/web-chat/web/src/webchat/components/Composer.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Header.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Statusbar.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx`

### State and data clients

- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/appSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/errorsSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`

### Runtime bootstrap/config

- `pinocchio/cmd/web-chat/web/src/config/runtimeConfig.ts`
- `pinocchio/cmd/web-chat/web/src/utils/basePrefix.ts`

### Debug UI (separate but relevant)

The debug UI is not the main product surface, but it should still be inventoried because backend API changes may affect it later.

Read at least:

- `pinocchio/cmd/web-chat/web/src/debug-ui/DebugUIApp.tsx`
- `pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts`
- `pinocchio/cmd/web-chat/web/src/debug-ui/ws/debugTimelineWsManager.ts`

---

## 9. Proposed New Frontend-Facing Backend Contract

This document does not lock the final API, but it should still offer a target shape so the frontend migration has something concrete to build against.

### 9.1 Chat submit endpoint

Example target:

```http
POST /api/chat/sessions/:sessionId/messages
Content-Type: application/json

{
  "prompt": "Explain ordinals in plain language",
  "profile": "default"
}
```

Response could be minimal:

```json
{
  "sessionId": "sess-123",
  "accepted": true
}
```

Or include useful bootstrap data:

```json
{
  "sessionId": "sess-123",
  "accepted": true,
  "profile": "default",
  "status": "running"
}
```

### 9.2 Session snapshot endpoint

Example target:

```http
GET /api/chat/sessions/:sessionId
```

Response:

```json
{
  "sessionId": "sess-123",
  "ordinal": "42",
  "entities": [...],
  "status": "running"
}
```

### 9.3 Websocket protocol

Instead of the old implicit connect-by-URL-only behavior, the frontend can use the new explicit subscribe model.

Client:

```json
{ "type": "subscribe", "sessionId": "sess-123", "sinceOrdinal": "0" }
```

Server first sends snapshot:

```json
{
  "type": "snapshot",
  "sessionId": "sess-123",
  "ordinal": "42",
  "entities": [...]
}
```

Then live UI events:

```json
{
  "type": "ui-event",
  "sessionId": "sess-123",
  "ordinal": "43",
  "name": "ChatMessageAppended",
  "payload": { ... }
}
```

This is the kind of contract that matches the actual Phase 3 transport rather than the older SEM assumptions.

### 9.4 Profile API

If profile selection remains a visible feature, the frontend still needs:

- list profiles,
- get current profile,
- set current profile.

These routes can change, but the feature should remain.

---

## 10. Recommended Frontend Refactor Strategy

A safe frontend migration should happen in deliberate steps.

### Step 1: Inventory current user-visible functionality

Before changing code, document:

- what the user can do,
- what the UI displays,
- what state transitions are visible,
- what failures the UI surfaces,
- what profile-related behavior exists.

This is not busywork. It protects the product while the transport changes.

### Step 2: Isolate backend assumptions

Audit and mark the places where the current frontend assumes:

- `/chat` payload shape,
- `/api/timeline` payload shape,
- websocket SEM frame structure,
- `conv_id` naming,
- buffering/replay assumptions.

These assumptions should be pulled behind cleaner adapter helpers.

### Step 3: Build the new canonical frontend client layer

Create a clear app-facing client layer for:

- submit,
- snapshot load,
- profile APIs,
- websocket subscribe/live flow.

This is where the new API shape should live.

### Step 4: Refactor the websocket manager

The current `wsManager.ts` is one of the biggest migration hotspots.

It should be rewritten so it no longer depends on old SEM envelopes and timeline bootstrap assumptions.

A likely better shape is:

```text
ws client
  -> connect()
  -> subscribe(sessionId)
  -> receive snapshot frame
  -> receive ui-event frames
  -> dispatch to timeline reducer / session controller
```

### Step 5: Update state normalization

`timelineSlice.ts` should be reviewed so the frontend state model is driven by the new `evtstream`-based app contract, not by the historical shape of SEM-to-timeline conversion.

### Step 6: Keep the visible component tree stable

Try not to rewrite `Composer`, `Timeline`, `Header`, and `Statusbar` unless the migration reveals a real deficiency.

The product should look familiar while the data/control layers change underneath.

### Step 7: Validate feature parity continuously

After each slice, check:

- can I send a prompt?
- do I see streaming output?
- can I start a new conversation/session?
- does reconnect still feel coherent?
- does history/timeline still render correctly?
- does profile selection still work?

---

## 11. Functional Parity Checklist

This checklist should live next to the migration work and be used continuously.

### Core chat behavior
- [ ] User can type and send a prompt.
- [ ] A new session/conversation can be created from the UI.
- [ ] Assistant output streams live.
- [ ] Final output remains visible after streaming completes.
- [ ] Stopped/interrupted runs surface coherently if stop remains a visible feature.

### Session / reconnect behavior
- [ ] Session identity persists in the UI.
- [ ] Reloading or reconnecting restores coherent state.
- [ ] Snapshot-before-live behavior is preserved from the user perspective.
- [ ] Historical timeline entities render correctly after reconnect.

### Profile/runtime behavior
- [ ] Available profiles are listed.
- [ ] Current profile is visible.
- [ ] Profile can be switched.
- [ ] The selected profile affects subsequent chat runs.

### Error handling
- [ ] Connection failures surface clearly.
- [ ] Submit failures surface clearly.
- [ ] Hydration failures surface clearly.
- [ ] Error panel/toggles remain understandable.

### Visual/product continuity
- [ ] Header still looks like the same product.
- [ ] Timeline still looks like the same product.
- [ ] Composer still looks like the same product.
- [ ] Status indicators still communicate the same ideas.

---

## 12. Risks on the Frontend Side

### Risk 1: Accidental product redesign

This is the biggest frontend risk.

If the migration becomes an excuse to rethink the UI structure, spacing, renderer model, and interaction flow all at once, the team will lose the ability to tell whether the backend cutover preserved the product.

### Risk 2: Legacy assumptions remain hidden in the frontend

If old contracts are not explicitly removed, the frontend can silently continue depending on backend quirks even after the backend is supposedly modernized.

### Risk 3: wsManager becomes a second backend

The frontend websocket manager should not contain too much business logic. If it starts re-projecting backend meaning instead of consuming a clearer app-facing contract, the frontend will remain tightly coupled to transport internals.

### Risk 4: Debug UI is forgotten until too late

The debug UI is not the primary migration surface, but it still depends on backend APIs. If the main chat UI migrates and the debug UI is ignored, the team may discover hidden route assumptions too late.

---

## 13. Recommended Reading Order for a New Intern

Read in this order:

1. `le-chat/.../EVT-STREAM-010/design-doc/01-phase-6-implementation-plan.md`
2. `pinocchio/cmd/web-chat/web/src/App.tsx`
3. `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
4. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
5. `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
6. `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`
7. `pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx`
8. `pinocchio/cmd/web-chat/web/src/webchat/components/Composer.tsx`
9. `pinocchio/cmd/web-chat/web/src/config/runtimeConfig.ts`
10. `pinocchio/pkg/evtstream/transport/ws/server.go`
11. `pinocchio/pkg/evtstream/examples/chat/chat.go`
12. `pinocchio/pkg/evtstream/hydration/sqlite/store.go`

Optional but useful after that:

13. `pinocchio/cmd/web-chat/web/src/debug-ui/DebugUIApp.tsx`
14. `pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts`
15. `pinocchio/cmd/web-chat/web/src/debug-ui/ws/debugTimelineWsManager.ts`

---

## 14. First Concrete Frontend Tasks

A new intern should begin with these tasks, in this order.

### Task 1: Write a frontend behavior inventory

Document:

- current visible screens,
- current chat flow,
- current reconnect behavior,
- current profile behavior,
- current error behavior.

### Task 2: Mark current backend-coupled points in code

Specifically identify:

- fetch submit logic,
- timeline hydration logic,
- websocket connection URL logic,
- frame parsing assumptions,
- conv/session identity assumptions.

### Task 3: Propose a new frontend-facing API contract

Before rewriting code deeply, write down:

- submit endpoint,
- snapshot endpoint,
- websocket subscribe/live event schema,
- profile endpoints.

### Task 4: Refactor the data/client layer first

Do not start with visual components.

Start with:

- API helpers,
- websocket manager,
- normalization layer,
- session controller behavior.

### Task 5: Reconnect the existing components to the new client layer

Keep the UI surface stable while swapping the plumbing.

### Task 6: Validate with transcript and screenshot evidence

For every major slice, capture:

- before/after screenshot,
- before/after transcript,
- feature parity notes,
- intentional differences.

---

## 15. Final Recommendation

The frontend side of Phase 6 should be treated with the same discipline as the backend side:

> **Preserve the product, not the old wire format.**

That means the team should feel comfortable changing:

- routes,
- payloads,
- websocket frames,
- state integration code,

as long as the result still feels like the same `cmd/web-chat` application to the user.

The fastest way to fail this migration would be to preserve backend contract details that no longer make sense. The second-fastest way would be to redesign the UI while the backend is also changing.

The right path is the middle one:

- keep the UI steady,
- modernize the integration,
- and let the frontend become a first-class `evtstream` client.

---

## References at a Glance

### Main chat frontend
- `pinocchio/cmd/web-chat/web/src/App.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Composer.tsx`

### State and transport
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`
- `pinocchio/cmd/web-chat/web/src/config/runtimeConfig.ts`

### Debug UI (secondary)
- `pinocchio/cmd/web-chat/web/src/debug-ui/DebugUIApp.tsx`
- `pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts`
- `pinocchio/cmd/web-chat/web/src/debug-ui/ws/debugTimelineWsManager.ts`

### Target backend/runtime references
- `pinocchio/pkg/evtstream/transport/ws/server.go`
- `pinocchio/pkg/evtstream/examples/chat/chat.go`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go`

### Parent migration doc
- `le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md`
