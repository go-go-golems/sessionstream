---
Title: Phase 6 cmd/web-chat Port to evtstream and Migration Playbook
Ticket: EVT-STREAM-010
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - backend
    - implementation
    - migration
    - webchat
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/02-technical-architecture-event-streaming-llm-framework.md
      Note: Source-of-truth clean-room substrate architecture.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md
      Note: Donor-code and do-not-copy analysis for the legacy webchat stack.
    - Path: le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Cross-phase implementation bridge and onboarding guide.
    - Path: pinocchio/cmd/web-chat/main.go
      Note: Current application entrypoint and route composition for the existing web-chat command.
    - Path: pinocchio/pkg/webchat/http/api.go
      Note: Current app-facing HTTP and websocket handler seam that should be replaced rather than pulled back into evtstream core.
    - Path: pinocchio/pkg/webchat/conversation_service.go
      Note: Current mixed lifecycle/service surface whose responsibilities need to be split across app shell, chat app package, and evtstream runtime.
    - Path: pinocchio/pkg/webchat/stream_hub.go
      Note: Current websocket attachment shape and legacy hello/ping/pong semantics.
    - Path: pinocchio/pkg/evtstream/transport/ws/server.go
      Note: New substrate websocket transport to be used as the target live delivery mechanism.
    - Path: pinocchio/pkg/evtstream/examples/chat/chat.go
      Note: Existing chat example that proves the substrate can already host a chat app outside core.
    - Path: pinocchio/pkg/evtstream/hydration/sqlite/store.go
      Note: Durable hydration store that should become the cmd/web-chat persistence substrate.
ExternalSources: []
Summary: "Replace the old Phase 6 idea with a concrete cmd/web-chat port onto evtstream. The deliverable is a detailed migration playbook for severing pkg/webchat from the runtime path while preserving important user-visible behavior and using transcripts/regression evidence to guide the cutover."
LastUpdated: 2026-04-20T11:40:00-04:00
WhatFor: "Detailed analysis, design, and implementation guide for porting cmd/web-chat to evtstream in a way a new intern can understand and execute."
WhenToUse: "When planning or implementing the cmd/web-chat cutover off pkg/webchat and onto evtstream, or when onboarding a new contributor to the migration effort."
---

# Phase 6 cmd/web-chat Port to evtstream and Migration Playbook

## Executive Summary

This document replaces the earlier, more generic idea for Phase 6. Instead of treating the phase as a vague “webchat migration / regression lab,” the phase should now be understood much more concretely:

> **Port `cmd/web-chat` onto `evtstream`, sever `pkg/webchat` from the runtime path, and use transcripts plus regression evidence to build a disciplined migration playbook.**

That change matters because the project is no longer in the stage where we need another abstract compatibility discussion. We already have the substrate pieces we need:

- `evtstream` core types and hub,
- websocket transport,
- chat example package,
- hydration store abstraction,
- SQLite hydration implementation,
- Systemlab pages that demonstrate ordering, reconnect, chat, and restart behavior.

What we need now is a focused final integration phase that answers one hard question clearly:

> Can the real `cmd/web-chat` application become an `evtstream` app without dragging `pkg/webchat` internals back into the new framework?

The answer this phase is designed to produce should be **yes**. But it will only stay yes if we preserve one critical architectural rule from EVT-STREAM-002 and EVT-STREAM-003:

> `pkg/evtstream` is the substrate, `cmd/web-chat` is the application, and the cutover should end with one new canonical app contract rather than a permanent compatibility layer.

This document is written as a **build manual for a new intern**. It explains the current system, the target architecture, the migration boundary, the file layout that should exist after the port, the sequence of implementation slices, the regression strategy, the role of transcripts, and the exact pitfalls to avoid.

---

## The Core Recommendation

The new Phase 6 should not be “make old webchat and new evtstream coexist forever.”

It should be:

1. **keep `cmd/web-chat` as the application shell**,
2. **replace its runtime path with an `evtstream`-based chat app package**,
3. **preserve important frontend/user-visible behavior while cutting cleanly to one new canonical web contract**,
4. **use transcripts and Systemlab to decide what must match, what may diverge, and what should be intentionally dropped**,
5. **remove `pkg/webchat` from the live runtime path once the new path is proven**.

In short:

```text
current:
cmd/web-chat -> pkg/webchat -> everything

phase 6 target:
cmd/web-chat -> app-owned handlers + evtstream chat app + evtstream runtime
```

That is the concrete meaning of “sever `pkg/webchat` entirely.”

---

## 1. Why the Phase 6 Goal Changed

The original Phase 6 plan treated migration as a comparison problem: run something old-ish, run something new-ish, and diff them. That was useful when the substrate still lacked key runtime pieces. It is less useful now because the core runtime is no longer hypothetical.

Phases 0 through 5 already gave us the foundation:

- **Phase 0** gave us the shell and vocabulary.
- **Phase 1** gave us command → event → projection shape.
- **Phase 2** gave us bus-backed consumer-side ordinals.
- **Phase 3** gave us websocket transport and snapshot-before-live reconnect behavior.
- **Phase 4** gave us a real chat example on top of the substrate.
- **Phase 5** gave us durable hydration and restart correctness.

At that point, the most honest next step is no longer another synthetic compatibility page. The honest next step is:

- take the real app (`cmd/web-chat`),
- re-home it on the new substrate,
- and use regression evidence to control the migration.

This is better than the earlier Phase 6 framing for three reasons:

- it gives us a direct path to retiring `pkg/webchat`,
- it turns prior phase artifacts into a real migration playbook,
- and it prevents the codebase from accumulating a permanent “old app vs new framework” duality.

---

## 2. The Current System: What Exists Today

A new intern must understand the current shape before attempting the port. Right now, `cmd/web-chat` is already somewhat modern in one important way: **the app owns `/chat` and `/ws` route composition**. But the runtime underneath still lives largely in `pkg/webchat`.

### Key current app files

Read these first:

- `pinocchio/cmd/web-chat/README.md`
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `pinocchio/cmd/web-chat/profile_policy.go`
- `pinocchio/cmd/web-chat/timeline/*`

### Key current core files

Read these next:

- `pinocchio/pkg/webchat/doc.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/stream_hub.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/timeline_projector.go`
- `pinocchio/pkg/webchat/connection_pool.go`
- `pinocchio/pkg/webchat/stream_coordinator.go`

### What the current app actually does

The current `cmd/web-chat` flow, at a high level, is:

```text
Browser UI
  -> GET /ws?conv_id=...
  -> POST /chat
  -> GET /api/timeline

cmd/web-chat/main.go
  -> builds webchat.Server
  -> mounts app-owned /chat and /ws handlers
  -> mounts /api/timeline and optional debug APIs

pkg/webchat
  -> resolves/creates conversation
  -> starts runner / LLM loop
  -> emits SEM-like envelopes and timeline updates
  -> attaches websockets
  -> serves hydration/timeline reads
```

### Why this is not the final architecture

The current app is a useful stepping stone, but it still mixes too many concerns under `pkg/webchat`:

- conversation identity and runtime lifecycle,
- websocket attachment,
- transport-level hello/ping/pong,
- chat submission,
- timeline projection,
- persistence hooks,
- some chat-specific behavioral assumptions.

Those are exactly the kinds of mixed responsibilities EVT-STREAM-002 warned us not to preserve in the clean-room substrate.

---

## 3. The New System: What Should Exist After the Port

After the port, the application should still be `cmd/web-chat`, but the runtime underneath it should be completely different.

### Target layering

```text
cmd/web-chat (application shell)
    |
    +-- canonical HTTP / websocket handlers
    +-- app-owned route composition
    +-- profile/runtime selection
    +-- static UI serving
    |
    v
chat app package on evtstream
    |
    +-- chat command schemas
    +-- backend event schemas
    +-- UI projections
    +-- timeline projections
    +-- model/runtime adapters
    |
    v
pkg/evtstream (generic substrate)
    |
    +-- Hub
    +-- command dispatch
    +-- event bus
    +-- consumer-side ordinals
    +-- websocket transport
    +-- hydration store
    +-- reconnect semantics
```

### The key architectural rule

The most important sentence for the intern is this one:

> **Do not port `pkg/webchat`; port `cmd/web-chat` onto `evtstream`.**

That distinction is everything.

If we “port webchat,” we risk dragging old abstractions into the new core. If we instead port the **application** onto the substrate, we preserve the architecture:

- app concerns stay app-level,
- chat concerns stay chat-level,
- substrate concerns stay generic.

---

## 4. What Gets Deleted, What Gets Kept, What Gets Rebuilt

This section is the migration map a new intern should keep nearby while working.

### 4.1 Keep as application-level concerns

These belong in `cmd/web-chat` or an app-specific package after the port:

- request parsing and validation,
- canonical HTTP / websocket handler logic,
- profile/runtime selection,
- route composition,
- app-specific API response shapes,
- static UI asset serving,
- feature flags / cutover routing.

### 4.2 Rebuild on top of `evtstream`

These should be rebuilt using the new substrate instead of preserved from `pkg/webchat`:

- chat submission path,
- websocket live event delivery,
- reconnect hydration behavior,
- timeline hydration state,
- durable restart behavior,
- ordering / ordinal behavior.

### 4.3 Reuse as donors, not as architecture

These older pieces are useful as donor patterns, but should not survive whole-cloth:

- `pinocchio/pkg/webchat/connection_pool.go`
  - useful for fanout/backpressure ideas,
  - but Phase 3 transport now owns the new shape.
- `pinocchio/pkg/webchat/stream_coordinator.go`
  - useful for consumption-time ordering ideas,
  - already reflected in `evtstream` ordinals.
- `pinocchio/pkg/webchat/http/api.go`
  - useful for thin handler shape,
  - but handlers should now talk to `evtstream` app code.

### 4.4 Intentionally leave behind

These should not be pulled back into the substrate:

- dual identity model centered on `conv_id` and `session_id` separately,
- SEM-first canonical internal stream,
- package-global timeline registries,
- chat-specific transport semantics inside core,
- large monolithic “conversation service” ownership of too many runtime concerns.

---

## 5. Proposed File and Package Layout After the Port

A concrete target layout helps prevent endless architectural drift.

### Application shell

```text
pinocchio/cmd/web-chat/
  main.go
  server.go
  routes.go
  app/
    app.go
    config.go
    handlers_chat.go
    handlers_ws.go
    handlers_timeline.go
    handlers_debug.go
```

### Chat app package on evtstream

```text
pinocchio/pkg/evtstream/apps/chat/
  schemas.go
  service.go
  projections.go
  runtime.go
  llm_adapter.go
```

If we want a stricter migration boundary, we can also split presentation-facing code from domain/runtime code:

```text
pinocchio/pkg/evtstream/apps/chat/
pinocchio/cmd/web-chat/app/
```

Where:

- `chat/` is the app-level domain/runtime model,
- `cmd/web-chat/app/` owns the web-facing handler layer.

### What should *not* happen

Do **not** create new substrate-core packages like:

- `pkg/evtstream/webchat`
- `pkg/evtstream/conversation`
- `pkg/evtstream/sem`

Those names are warning signs that old architecture is leaking back in.

---

## 6. How the Ported `cmd/web-chat` Should Work at Runtime

A new intern should understand the runtime in terms of **boot**, **submit**, **live UI**, **hydrate**, and **restart**.

### 6.1 Boot sequence

At startup, the new `cmd/web-chat` should do roughly this:

```go
func buildApp() {
    reg := evtstream.NewSchemaRegistry()
    chatapp.RegisterSchemas(reg)

    store := chooseHydrationStore(config)
    ws := ws.NewServer(snapshotProviderFromStore(store))
    bus := buildBus(config)

    hub := evtstream.NewHub(
        evtstream.WithSchemaRegistry(reg),
        evtstream.WithHydrationStore(store),
        evtstream.WithUIFanout(ws),
        evtstream.WithEventBus(bus.Publisher, bus.Subscriber, ...),
    )

    chatapp.Install(hub, runtimeDeps)
    hub.Run(ctx)

    mountHTTPHandlers(hub, ws, appHandlers)
}
```

### 6.2 Submit path

The new submit path should look like:

```text
POST /chat
  -> app handler parses request
  -> resolves / creates SessionId
  -> submits StartInference command to hub
  -> returns bootstrap/app response
```

### 6.3 Live websocket path

```text
client connects to /ws
  -> websocket transport allocates ConnectionId
client subscribes to SessionId
  -> snapshot is loaded from HydrationStore
  -> snapshot is sent first
  -> connection joins live fanout set
  -> UI events arrive live after snapshot
```

### 6.4 Timeline / hydration path

```text
consumer handles backend event
  -> assign ordinal
  -> run UI projection
  -> run timeline projection
  -> apply timeline entities to HydrationStore
  -> fan out UI events to websocket clients
```

### 6.5 Restart path

```text
process restarts
  -> rebuild store + bus + hub + transport
  -> store loads durable cursor + entities
  -> next consumed event resumes ordinal progression
  -> reconnecting client gets current snapshot first
```

These are already the semantics proved in Phases 3–5. The port should *reuse them*, not reimplement them in application-specific ways.

---

## 7. Clean-Cut Strategy: One New Canonical Web Contract

This is one of the most important practical decisions in the whole migration.

### Recommended strategy: no runtime backwards-compatibility layer

Do the cutover with one new canonical `cmd/web-chat` web contract and migrate the frontend in lockstep.

That means:

- the **backend internals** become `evtstream`-native,
- the **frontend** is updated to speak the new contract,
- the **runtime path** does not preserve old request/response or websocket shapes merely because they existed before.

### Why this is the right tradeoff

It gives us:

- a cleaner backend cutover,
- a cleaner frontend cutover,
- fewer hidden legacy assumptions,
- no temptation to recreate `pkg/webchat` behavior behind thin adapters.

### What still must be preserved

Preserve only the user-visible or product-visible behavior that matters:

- prompt submission as a visible workflow,
- live streaming behavior,
- timeline/history visibility,
- reconnect/hydration coherence,
- ordering guarantees that the UI relies on,
- profile selection,
- any debug routes we consciously decide still matter.

### What does not need preservation

Do not preserve any of the following merely because the old stack had them:

- old `/chat` payload shapes,
- old `/ws` handshake details,
- old websocket envelope names,
- old `/api/timeline` response schema,
- old `conv_id` naming,
- old hello/ping/pong conventions,
- old debug endpoints that only existed to support the previous runtime.

### How to classify differences

Each legacy behavior should be classified as:

- **must preserve as product behavior**,
- **replace with a clearer new canonical behavior**,
- **intentionally drop**.

This classification should be documented explicitly in the migration playbook artifacts.

---

## 8. Frontend Migration Strategy

The backend cutover will fail if the frontend is treated as a passive afterthought. `cmd/web-chat` already has a real product surface, and that product surface carries implicit assumptions about identity, bootstrap order, websocket timing, hydration, profile selection, and timeline rendering.

So the frontend strategy for Phase 6 should be explicit:

> **Preserve the visible UI/UX and the user-facing functionality, but allow the HTTP APIs, websocket frame schemas, hydration payloads, and frontend state integration code to change as needed.**

That is the right tradeoff for this phase because it protects the product while still allowing `cmd/web-chat` to become an `evtstream`-native application.

### 8.1 What should remain stable

The frontend migration should preserve, as closely as possible:

- the current overall chat layout,
- the composer placement and send flow,
- the streamed-response reading experience,
- profile selection as a visible feature,
- conversation/session reset behavior,
- coherent reconnect/hydration behavior from the user perspective,
- the current general visual design and product feel.

### 8.2 What may change freely

The frontend migration may change:

- route names,
- request/response JSON shapes,
- websocket frame contracts,
- hydration snapshot schema,
- query-parameter naming,
- Redux/client state wiring,
- buffering/replay implementation details.

Those changes are acceptable as long as they define the new canonical application contract and do not leak webchat-specific legacy behavior into `pkg/evtstream` core.

### 8.3 Recommended frontend implementation approach

The safest frontend approach is:

1. keep the visible component tree largely stable,
2. replace the transport/client layer underneath it,
3. refactor websocket + hydration logic around `evtstream` transport semantics,
4. validate feature parity continuously with transcript and browser evidence.

In practice, this means the migration should focus first on files like:

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`

while trying to keep the visual component files as stable as possible:

- `pinocchio/cmd/web-chat/web/src/webchat/components/Composer.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Timeline.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Header.tsx`
- `pinocchio/cmd/web-chat/web/src/webchat/components/Statusbar.tsx`

### 8.4 Frontend work checklist

- [ ] Inventory the current user-visible frontend behavior that must survive the cutover.
- [ ] Document existing frontend assumptions about `/chat`, `/ws`, `/api/timeline`, `conv_id`, hydration, and reconnect.
- [ ] Define the new app-edge frontend contract for submit, snapshot load, websocket subscribe/live updates, and profile APIs.
- [ ] Refactor the frontend client/data layer before rewriting visual components.
- [ ] Port websocket live updates to the new `evtstream`-backed transport path.
- [ ] Port hydration/reconnect to snapshot-before-live semantics backed by the new runtime.
- [ ] Keep the existing UI/UX/functionality stable while swapping the plumbing underneath.
- [ ] Capture before/after screenshots, transcripts, and parity notes for at least one real `cmd/web-chat` flow.

### 8.5 Detailed companion guide

This section is intentionally condensed. The detailed frontend-specific execution guide for an intern lives in:

- `design-doc/02-web-frontend-migration-and-design-guide.md`

Use that document for:

- current frontend architecture walkthrough,
- file-by-file reading order,
- proposed frontend-facing `evtstream` API direction,
- deeper refactor strategy,
- parity checklist and frontend-specific risks.

---

## 9. Proposed Migration Phases Inside Phase 6

The port should not be attempted as one massive cutover. The right approach is a sequence of small, reviewable slices.

### Slice 0: Inventory and contract freeze

Goal: document the current `cmd/web-chat` behavior that actually matters.

Deliverables:
- list of critical routes,
- list of request/response contracts,
- list of websocket visible behaviors,
- list of debug/timeline behaviors,
- initial preserve/change/drop matrix.

Key files to inspect:
- `pinocchio/cmd/web-chat/main.go`
- `pinocchio/cmd/web-chat/README.md`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/stream_hub.go`

### Slice 1: Build the `evtstream`-backed chat app package for real cmd/web-chat use

Goal: evolve the current example package into a real app package suitable for `cmd/web-chat`.

Deliverables:
- chat schemas stabilized,
- runtime adapter boundary defined,
- stop/start behavior preserved,
- package no longer positioned as only a Systemlab demo.

Likely files:
- `pinocchio/pkg/evtstream/apps/chat/...`

### Slice 2: Add a parallel app-owned `evtstream` path inside `cmd/web-chat`

Goal: run the new path without deleting the old path immediately.

Techniques:
- feature flag,
- alternate route prefix,
- alternate handler wiring behind config.

This is where the migration becomes safe.

### Slice 3: Port websocket live path

Goal: replace old websocket live delivery with `evtstream/transport/ws`.

Requirements:
- canonical websocket contract owned by the new app,
- reconnect via snapshot-before-live,
- no legacy hello/ping/pong emulation unless consciously retained as part of the new contract.

### Slice 4: Port timeline hydration and persistence

Goal: replace old timeline hydration path with `evtstream` hydration store and read APIs.

Requirements:
- timeline snapshots come from `evtstream` hydration,
- SQL mode becomes the durable baseline,
- memory mode remains useful for dev/test.

### Slice 5: Port debug and operational routes

Goal: rebuild only the debug/API surfaces that still matter after the runtime cutover.

Do not blindly carry every old debug route forward.

### Slice 6: Switch default path and sever `pkg/webchat`

Goal: make the new `evtstream` path the real `cmd/web-chat` implementation.

This is the slice where:
- old path becomes fallback or is removed,
- `pkg/webchat` leaves the live runtime path,
- remaining legacy-only code is either archived or explicitly deprecated.

---

## 10. Systemlab’s New Role in Phase 6

Systemlab should not be merely a “legacy comparison museum.” It should become the **migration inspection console**.

### The new Phase 6 page should show

- the scenario being exercised,
- the old expectation transcript,
- the new `cmd/web-chat on evtstream` transcript,
- a structural diff,
- a decision classification for each difference,
- links to follow-up tickets if needed.

### Suggested page layout

```text
+------------------------------------------------------------------------------------------------+
| Phase 6 — cmd/web-chat Port / Migration Playbook                                              |
+--------------------------------------+---------------------------------------------------------+
| [Scenario] streaming chat            | [Preserve / Change / Drop Matrix]                       |
| [Run migrated path] [Load fixture]   | prompt workflow: preserve                               |
| [Export transcript]                  | ws protocol: replace with canonical evtstream app flow  |
|                                      | reconnect semantics: replace with snapshot-before-live  |
+--------------------------------------+---------------------------+-----------------------------+
| [Legacy / Historical Transcript]                                  | [New evtstream Path]       |
| ws.hello -> chat.message -> ...                                   | StartInference -> ...      |
+--------------------------------------+---------------------------+-----------------------------+
| [Diff / Decision]                                                 | [Reviewer Notes]           |
| changed: yes/no                                                   | intentional? yes/no        |
| ticket: EVT-STREAM-0xx                                             | future cleanup?            |
+------------------------------------------------------------------------------------------------+
```

### Why this matters

This makes the migration:

- teachable,
- reviewable,
- exportable,
- and less dependent on oral history.

---

## 11. Using Transcripts to Build the Migration Playbook

The user explicitly wants to use “the transcript of our work” to build a solid migration playbook. That is the right instinct.

### What transcript sources we already have

We already have good material from prior phases:

- EVT-STREAM-004 / 005 / 006 / 007 / 008 / 009 ticket docs and artifacts,
- Systemlab captured responses and transcripts,
- implementation diaries that explain why the current substrate is shaped the way it is.

### What additional transcripts Phase 6 should capture

For each migration scenario, capture:

1. **historical expectation transcript**
   - not necessarily a full live run if old path is being retired,
   - may be a curated golden fixture.
2. **new evtstream path transcript**
   - structured JSON preferred,
   - markdown export optional for review readability.
3. **decision record**
   - same / changed / intentional / bug / follow-up.

### Transcript schema suggestion

```json
{
  "scenario": "streaming-chat-basic",
  "request": {"prompt": "hello"},
  "legacyExpectation": {...},
  "newPath": {...},
  "diff": [...],
  "decision": [
    {
      "path": "ws.frames[0].type",
      "status": "changed-intentionally",
      "reason": "new transport uses snapshot-before-live contract",
      "followUp": null
    }
  ]
}
```

This should become the heart of the migration playbook.

---

## 12. Implementation Risks

### Risk 1: accidentally rebuilding pkg/webchat inside evtstream

This is the biggest risk.

Symptoms:
- new core types start looking like old conversation types,
- chat-specific names appear in `pkg/evtstream` signatures,
- compatibility envelopes become canonical internal events.

### Risk 2: preserving too many quirks

Some historical behavior will not be worth preserving.

If we keep every quirk, we slow down the cutover and muddy the architecture.

### Risk 3: frontend dependence hidden inside implicit behavior

Even without a compatibility layer, the frontend can still depend on subtle old assumptions about timing, identity, hydration order, and event visibility.

If we do not document those assumptions explicitly, the port will become fragile and surprising.

### Risk 4: split-brain runtime during migration

If both old and new paths stay live too long without a clear cutover plan, Phase 6 can become a permanent dual stack.

That would be a failure.

---

## 13. What a New Intern Should Actually Do First

This section is deliberately practical.

### Reading order

Read in this order:

1. `EVT-STREAM-002 design/02`
2. `EVT-STREAM-002 design/03`
3. `EVT-STREAM-003 implementation guide`
4. `pinocchio/cmd/web-chat/README.md`
5. `pinocchio/cmd/web-chat/main.go`
6. `pinocchio/pkg/webchat/http/api.go`
7. `pinocchio/pkg/webchat/conversation_service.go`
8. `pinocchio/pkg/webchat/stream_hub.go`
9. `pinocchio/pkg/evtstream/transport/ws/server.go`
10. `pinocchio/pkg/evtstream/examples/chat/chat.go`
11. `pinocchio/pkg/evtstream/hydration/sqlite/store.go`
12. Phase 3 / 4 / 5 Systemlab lab backends:
    - `cmd/evtstream-systemlab/phase3_lab.go`
    - `cmd/evtstream-systemlab/phase4_lab.go`
    - `cmd/evtstream-systemlab/phase5_lab.go`

### First implementation tasks

Start with these concrete steps:

- inventory the current `cmd/web-chat` public/runtime behavior,
- write preserve/change/drop matrix,
- define target `evtstream` chat app package layout,
- define the new canonical contract for each route and websocket flow,
- implement one migrated happy-path scenario,
- capture transcripts,
- build Phase 6 Systemlab comparison page around those artifacts.

### Concrete implementation guide and slice order

The safest way to execute Phase 6 is as a sequence of narrow vertical slices. Each slice should end with code that runs, focused validation, and a small commit. Do not wait until the whole backend is rewritten before updating the frontend. Do not attempt the standalone-module extraction during this phase.

#### Slice A — Analysis freeze and contract inventory

Deliverables:

- current behavior inventory,
- preserve/change/drop matrix,
- canonical route + websocket contract sketch,
- detailed task ordering.

Validation:

- docs reviewed,
- ticket tasks updated,
- diary started.

#### Slice B — App-grade evtstream chat package

Goal: evolve the current example chat package into an app-grade package intended to be consumed by `cmd/web-chat`.

Likely code areas:

- `pinocchio/pkg/evtstream/apps/chat/...`
- `pinocchio/pkg/evtstream/examples/chat/...` (if staged through a move or wrapper)
- tests for the app-grade package

Validation:

- `go test ./pkg/evtstream/...`

Commit point:

- commit once the new package shape exists, tests pass, and Systemlab Phase 4 can consume it or is ready to be switched.

#### Slice C — Canonical backend handlers inside cmd/web-chat

Goal: create app-owned handlers/services that target the new evtstream-backed package rather than `pkg/webchat`.

Likely code areas:

- `pinocchio/cmd/web-chat/app/...` or equivalent app-owned handler layer
- `cmd/web-chat/main.go`
- app-owned profile/request resolution glue

Validation:

- focused Go tests for handler/service seams,
- one server startup path that mounts the new contract.

Commit point:

- commit once submit, snapshot, and websocket seams exist in code, even if the frontend is not yet wired.

#### Slice D — Frontend client-layer cutover

Goal: update the frontend to use the new canonical contract.

Likely code areas:

- `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/store/profileApi.ts`

Validation:

- frontend typecheck/lint,
- browser load without console/runtime failures,
- one real prompt flow hitting the new backend path.

Browser test target:

- when running real inference flows during this phase, prefer the `gpt-5-nano-low` inference profile unless a test-specific fake/stub path is being used.

Commit point:

- commit once one browser-validated prompt → stream → finish path works on the new contract.

#### Slice E — Hydration/reconnect and timeline correctness

Goal: replace the old `/api/timeline` and websocket sequencing assumptions with evtstream-native snapshot-before-live semantics.

Validation:

- reload/reconnect scenario passes,
- transcript snapshot matches expected timeline state,
- focused browser test for reconnect.

Commit point:

- commit once one full reload/reconnect scenario works against the new stack.

#### Slice F — Profile integration, cleanup, and pkg/webchat severance

Goal: finish profile/runtime integration, remove the live dependency on `pkg/webchat`, and document the cutover.

Validation:

- end-to-end browser flow with `gpt-5-nano-low`,
- focused tests in `cmd/web-chat`,
- transcript artifacts captured.

Commit point:

- commit when the live runtime path no longer depends on `pkg/webchat`.

### Working loop for this ticket

Use this loop repeatedly:

1. implement one narrow slice,
2. run focused tests,
3. validate manually in the browser when the slice affects UI/runtime behavior,
4. commit code,
5. update tasks,
6. update the diary with exact commands, failures, and review instructions,
7. commit docs.

### First validation commands

At minimum, keep these in your loop:

```bash
cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio

go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...
make evtstream-check
```

And later for the real app port, add focused `cmd/web-chat` tests plus transcript fixtures.

---

## 14. Definition of Done for the New Phase 6

Phase 6 should be considered complete when a reviewer can say all of the following are true:

- `cmd/web-chat` runs on top of `evtstream` for at least one end-to-end real scenario,
- websocket live delivery uses the new transport path,
- hydration and restart use the new store path,
- no runtime compatibility layer is required for the live path,
- the new canonical web contract is explicit and documented,
- `pkg/webchat` is no longer required in the live runtime path,
- transcripts exist that explain the migration decisions,
- Systemlab Phase 6 makes the migration understandable and inspectable.

That is a much more concrete and useful finish line than the earlier Phase 6 wording.

---

## 15. Final Recommendation

The final recommendation is simple:

> **Treat Phase 6 as the `cmd/web-chat` cutover phase, not as an indefinite compatibility phase.**

Build one clean path:

- `cmd/web-chat` as the app shell,
- `evtstream` as the runtime substrate,
- app-level chat package for domain logic,
- transcript-backed migration artifacts for confidence,
- Systemlab Phase 6 as the human-readable regression console.

If we do that, we get three benefits at once:

- a real severance from `pkg/webchat`,
- a reusable migration playbook for later apps,
- and a cleaner long-term architecture than keeping the old stack half-alive indefinitely.

---

## References at a Glance

### Current app
- `pinocchio/cmd/web-chat/README.md`
- `pinocchio/cmd/web-chat/main.go`

### Current legacy runtime
- `pinocchio/pkg/webchat/doc.go`
- `pinocchio/pkg/webchat/http/api.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/stream_hub.go`
- `pinocchio/pkg/webchat/router.go`

### New substrate/runtime
- `pinocchio/pkg/evtstream/transport/ws/server.go`
- `pinocchio/pkg/evtstream/examples/chat/chat.go`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go`

### Prior phase companions
- `pinocchio/cmd/evtstream-systemlab/phase3_lab.go`
- `pinocchio/cmd/evtstream-systemlab/phase4_lab.go`
- `pinocchio/cmd/evtstream-systemlab/phase5_lab.go`

### Design source of truth
- `le-chat/.../EVT-STREAM-002/design/02-technical-architecture-event-streaming-llm-framework.md`
- `le-chat/.../EVT-STREAM-002/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`
- `le-chat/.../EVT-STREAM-003/design-doc/01-implementation-plan-and-intern-onboarding-guide.md`
