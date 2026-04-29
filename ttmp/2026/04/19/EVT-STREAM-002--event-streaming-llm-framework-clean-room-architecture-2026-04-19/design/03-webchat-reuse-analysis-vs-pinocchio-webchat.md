---
Title: Webchat Reuse Analysis — EVT-STREAM-002 vs pinocchio/pkg/webchat
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
    - reuse
    - pinocchio
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design/02-technical-architecture-event-streaming-llm-framework.md
      Note: Target framework architecture being evaluated.
    - Path: geppetto/ttmp/2026/02/02/PI-007-WEBCHAT-BACKEND-REFACTOR--webchat-backend-refactor/sources/local/webchat-backend-refactor.md
      Note: Older webchat backend refactor analysis; useful baseline, but now partially outdated.
    - Path: pinocchio/pkg/webchat/doc.go
      Note: Current package ownership model and public composition guidance.
    - Path: pinocchio/pkg/webchat/router.go
      Note: Current router/server composition and handler split.
    - Path: pinocchio/pkg/webchat/conversation.go
      Note: Conversation/ConvManager lifecycle and stream fan-out.
    - Path: pinocchio/pkg/webchat/stream_coordinator.go
      Note: Event consumption, sequence derivation, stream-id handling.
    - Path: pinocchio/pkg/webchat/timeline_projector.go
      Note: Current timeline projection architecture.
    - Path: pinocchio/pkg/persistence/chatstore/timeline_store.go
      Note: Current hydration store contract and durable store implementations.
ExternalSources: []
Summary: "Detailed comparison between EVT-STREAM-002's clean-room reusable framework design and the current pinocchio/pkg/webchat implementation. Identifies which subsystems can be reused directly, which should only inform the new design, and which should not be carried over. Main conclusion: use webchat as a donor of battle-tested transport/lifecycle/storage pieces, but do not turn it into the substrate verbatim because its identity model, event model, and projection pipeline are too chat/SEM-specific."
LastUpdated: 2026-04-19T18:30:00-04:00
WhatFor: "Guide implementation of the new reusable event-streaming framework by grounding design decisions in the existing webchat package's strengths, limitations, and already-landed refactors."
WhenToUse: "When deciding whether to extract code from pinocchio/pkg/webchat into EVT-STREAM-002, when planning the first implementation slices, or when debating whether the new framework should inherit webchat architecture wholesale."
---

# Webchat Reuse Analysis — EVT-STREAM-002 vs `pinocchio/pkg/webchat`

## Executive summary

**Short answer:** the existing `pinocchio/pkg/webchat` is a **strong donor codebase**, but a **bad substrate to rename in place**.

What is worth reusing almost directly:

- the **transport/lifecycle split** already achieved in the current package (`doc.go`, `router.go`, `http/api.go`)
- the **conversation/session manager shape** with idle eviction (`conversation.go`, `conv_manager_eviction.go`)
- the **application-owned pub/sub seam** (`stream_backend.go`)
- the **consumer-side sequence derivation from Redis stream IDs** (`stream_coordinator.go`)
- the **non-blocking websocket connection pool** (`connection_pool.go`)
- the **durable hydration store implementations** (`pkg/persistence/chatstore/*`), after adapting the interface
- the **request policy seam** (`http/api.go`'s `ConversationRequestResolver`)

What should only be reused conceptually, not copied as framework core:

- `ConvManager` / `ConversationService` / `StreamHub`
- the current runner/start pattern in `ChatService` + `LLMLoopRunner`
- timeline/debug indexing
- timeline custom-handler ideas (`timeline_registry.go`, `timeline_js_runtime.go`)

What should **not** become the core of EVT-STREAM-002:

- the **dual identity model** (`Conversation.ID` + `SessionID`)
- the **Geppetto-event → SEM-envelope → timeline** pipeline
- the **chat-specific command surface** (`SubmitPrompt`, `LLMLoopStartPayload`, queue/idempotency baked into prompt flow)
- package-global projection registries and hook sprawl
- `Router` as the main public abstraction

**Bottom line:** build EVT-STREAM-002 as a clean package (`evtstream` or equivalent), then port selected internals from webchat into it. After that, re-implement webchat as the **first backend/application** on top of the new substrate.

---

## 1. The old analysis is now partially outdated

The older analysis in `geppetto/.../webchat-backend-refactor.md` correctly identified the big problems of the then-current webchat package, but several of its recommendations have already landed in the current `pinocchio/pkg/webchat`.

### 1.1 What the old analysis got right

It argued for:

- splitting core runtime from HTTP transport and UI serving
- fixing subpath mounting by returning handlers instead of hiding route ownership
- extracting a conversation manager
- making websocket fan-out non-blocking
- deriving stream ordering from Redis stream IDs when available
- adding idle eviction

Those are still exactly the right architectural instincts for EVT-STREAM-002.

### 1.2 What is already true in current `pkg/webchat`

The package today already documents a much cleaner ownership model:

```go
// Package webchat provides conversation lifecycle primitives plus optional UI/API routing helpers.
//
// Ownership model:
//   - Applications own transport routes such as /chat and /ws.
//   - Package helpers (UIHandler/APIHandler) expose the embedded UI and core APIs (timeline/debug),
//     but applications still own HTTP route composition.
```

Where to look:
- `pinocchio/pkg/webchat/doc.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/http/api.go`

Why it matters:
- The old analysis described a router that was too monolithic.
- The current package has already moved toward the same separation EVT-STREAM-002 wants: **core lifecycle services + thin transport adapters**.

Verdict:
- Treat the old analysis as **historically useful**, not as a description of the current package.
- EVT-STREAM-002 should evaluate the **current code first**, and use the old analysis mainly as rationale for why certain refactors matter.

---

## 2. Reuse matrix

| Area | Current webchat status | Reuse verdict | Why |
|---|---|---|---|
| HTTP/UI ownership split | Implemented | **Reuse pattern directly** | Matches new framework layering well |
| Request resolver/policy seam | Implemented | **Reuse pattern directly** | Good transport/core separation |
| Pub/sub backend seam | Implemented | **Reuse/adapt** | Similar to EVT-STREAM-002 event bus ownership |
| Redis stream ID → seq | Implemented | **Reuse directly** | Strong fit for ordinal stamping |
| Websocket connection pool | Implemented | **Reuse internally** | Good transport primitive, but hide behind ConnectionId |
| Idle eviction | Implemented | **Reuse pattern directly** | Operationally important in long-lived servers |
| Hydration store impls | Implemented | **Adapt heavily** | Storage code is useful, interface is too narrow |
| Conversation manager | Implemented | **Adapt heavily** | Good lifecycle shape, wrong identity and app coupling |
| Timeline projection | Implemented | **Do not reuse directly** | Runs off SEM/UI frames, not canonical backend events |
| Event translator / SEM envelopes | Implemented | **Do not use as substrate core** | Too UI/webchat-specific |
| Chat prompt queue/idempotency | Implemented | **Extract as optional middleware idea** | Useful pattern, not generic core |
| Router options / hooks | Implemented | **Do not copy API** | Sign of missing first-class framework slots |
| Global timeline registries | Implemented | **Do not reuse** | Wrong scoping model for a reusable hub |

---

## 3. What is already strongly aligned with EVT-STREAM-002

## 3.1 Transport/core split is already the right shape

### Issue: current webchat has already moved away from “router owns everything”

Problem:
The clean-room framework wants a reusable Generic layer plus optional transport adapters. Current webchat is no longer a monolithic router in the old sense; it already exposes a split between lifecycle services and transport helpers.

Where to look:
- `pinocchio/pkg/webchat/doc.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/server.go`
- `pinocchio/pkg/webchat/http/api.go`

Example:

```go
func (r *Router) APIHandler() http.Handler {
    mux := http.NewServeMux()
    r.registerAPIHandlers(mux)
    return mux
}

func (r *Router) UIHandler() http.Handler {
    mux := http.NewServeMux()
    r.registerUIHandlers(mux)
    return mux
}
```

And the transport helper interfaces are narrow:

```go
type ChatService interface {
    SubmitPrompt(ctx context.Context, in root.SubmitPromptInput) (root.SubmitPromptResult, error)
}

type StreamService interface {
    ResolveAndEnsureConversation(ctx context.Context, req root.ConversationRuntimeRequest) (*root.ConversationHandle, error)
    AttachWebSocket(ctx context.Context, convID string, conn *websocket.Conn, opts root.WebSocketAttachOptions) error
}
```

Why it matters:
- EVT-STREAM-002 should **absolutely keep** this split.
- It validates the new design's instinct that the core should not own app route composition.
- It also means webchat is already closer to “first backend on top of a substrate” than the old analysis suggests.

Reuse verdict:
- **Yes, reuse the pattern.**
- Do **not** carry over `Router` as the main abstraction; instead map these ideas into `Hub`, `Transport`, and optional adapter packages.

Cleanup sketch:

```text
evtstream/
  hub.go              # Generic core
  transport/ws/...    # websocket transport
  adapters/http/...   # optional HTTP adapters (chat, timeline, profiles, etc.)

webchat/
  backend/...         # StartInference, StopInference, chat projections
  adapters/http/...   # app-specific REST/WS mounting if needed
```

---

## 3.2 Application-owned bus + subscriber construction already matches the new design

### Issue: `StreamBackend` is close to the new event-bus seam

Problem:
EVT-STREAM-002 wants the application to own the event bus while the substrate consumes it. Current webchat already has a clear seam for that.

Where to look:
- `pinocchio/pkg/webchat/stream_backend.go`
- `pinocchio/pkg/webchat/router.go`

Example:

```go
type StreamBackend interface {
    EventRouter() *events.EventRouter
    Publisher() message.Publisher
    BuildSubscriber(ctx context.Context, convID string) (message.Subscriber, bool, error)
    Close() error
}
```

Why it matters:
- This is very close to `WithEventBus(pub, sub)` in the clean-room design.
- The fact that subscriber construction is a seam already means webchat learned the right lesson: **the core runtime should not hardcode Redis vs in-memory**.

Reuse verdict:
- **Reuse conceptually, likely with a new API.**
- The specific `events.EventRouter` type is too tied to current Geppetto/webchat plumbing.
- The abstraction should become more generic: publisher/subscriber pairs plus topic/partitioning discipline.

Cleanup sketch:

```go
type EventBus struct {
    Publisher  message.Publisher
    Subscriber message.Subscriber
}

func WithEventBus(pub message.Publisher, sub message.Subscriber, opts ...BusOption) HubOption
```

Keep from webchat:
- in-memory vs Redis wiring helpers
- subscriber creation strategy
- “shared subscriber vs dedicated subscriber” operational knowledge

---

## 3.3 Consumer-side ordinal derivation from stream IDs should be carried over almost unchanged

### Issue: `StreamCoordinator` already implements the right ordering idea

Problem:
The clean-room technical design says ordinals should be stamped at consumption time, ideally derived from the underlying stream ordering when possible. Current webchat already has this logic.

Where to look:
- `pinocchio/pkg/webchat/stream_coordinator.go`
- older rationale in `geppetto/.../webchat-backend-refactor.md` §6.2

Example:

```go
func (sc *StreamCoordinator) nextSeq(streamID string) uint64 {
    if streamID != "" {
        if derived, ok := deriveSeqFromStreamID(streamID); ok {
            for {
                current := sc.seq.Load()
                next := derived
                if next <= current {
                    next = current + 1
                }
                if sc.seq.CompareAndSwap(current, next) {
                    return next
                }
            }
        }
    }
    // fallback atomic++
}
```

Why it matters:
- This is one of the best pieces of existing architecture in webchat.
- It directly supports EVT-STREAM-002's `Event.Ordinal` model.
- It also encodes a practical, battle-tested fallback when the bus lacks a stable stream ID.

Reuse verdict:
- **Reuse directly** in spirit and probably almost in code.

Recommended adjustment for EVT-STREAM-002:
- Make this logic live in the bus-consumer layer.
- Persist cursor/ordinal state through the new `HydrationStore.Apply` + `Cursor` contract, not via conversation-local bookkeeping.

---

## 3.4 Non-blocking websocket fan-out is worth keeping

### Issue: `ConnectionPool` already embodies a good transport-internal backpressure policy

Problem:
A reusable realtime framework needs sane backpressure behavior. Current webchat already moved to per-client buffered queues and drops slow clients instead of stalling broadcasts.

Where to look:
- `pinocchio/pkg/webchat/connection_pool.go`

Example:

```go
type poolClient struct {
    conn      wsConn
    send      chan []byte
    closeOnce sync.Once
}

func (cp *ConnectionPool) Broadcast(data []byte) {
    // snapshot clients under lock
    // enqueue without blocking
    // drop client if buffer is full
}
```

Why it matters:
- This is exactly the sort of boring-but-hard code the new framework should reuse.
- It belongs inside the websocket transport, not inside backend application code.

Reuse verdict:
- **Reuse internally**, but do not expose this API as part of the substrate.

Important rearchitecture:
- Replace raw `*websocket.Conn` / `wsConn` exposure with substrate-owned `ConnectionId`.
- Let the transport map `ConnectionId -> conn`, while the Hub works only with ids.

---

## 3.5 Idle eviction is operationally important and should be part of the first implementation wave

### Issue: webchat proves idle reaping is not optional in a long-lived session server

Problem:
The clean-room design still leaves liveness/staleness partly open. Existing webchat already contains real idle eviction logic for conversations and stream shutdown.

Where to look:
- `pinocchio/pkg/webchat/conv_manager_eviction.go`
- `pinocchio/pkg/webchat/conversation.go`

Example:

```go
func (cm *ConvManager) shouldEvictConversation(now time.Time, idle time.Duration, conv *Conversation) bool {
    if conv.pool != nil && !conv.pool.IsEmpty() {
        return false
    }
    if conv.stream != nil && conv.stream.IsRunning() {
        return false
    }
    // do not evict while busy / queued
    // evict once idle long enough
}
```

Why it matters:
- Without this, the new substrate risks becoming fine in tests but leaky in production.
- Webchat shows the real conditions that matter: no connections, no running stream, no active work, idle long enough.

Reuse verdict:
- **Reuse the policy shape**.
- The exact implementation will need new types (`Session`, `ConnectionId`, command in-flight tracking), but the operational rules are already validated.

Recommended design improvement to EVT-STREAM-002:
- Move liveness/eviction from “open question” toward an early concrete subsystem.
- Keep protocol heartbeat questions open, but do not defer all lifecycle cleanup questions.

---

## 4. Where existing webchat conflicts with EVT-STREAM-002

## 4.1 The identity model is the biggest mismatch: `convID` and `SessionID` are separate

### Issue: current webchat routes by conversation id but filters events by a second session id

Problem:
EVT-STREAM-002 intentionally centers everything on one canonical `SessionId`. Current webchat has two first-class identities inside `Conversation`:

```go
type Conversation struct {
    ID        string
    SessionID string
    ...
}
```

And its stream callback uses both:

```go
func(e events.Event, _ StreamCursor, frame []byte) {
    if e != nil {
        eventSessionID := e.Metadata().SessionID
        if eventSessionID != "" && eventSessionID != c.SessionID {
            return
        }
    }
    if c.pool != nil { c.pool.Broadcast(frame) }
    if c.timelineProj != nil { _ = c.timelineProj.ApplySemFrame(c.baseCtx, frame) }
}
```

Where to look:
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store.go` (`ConversationRecord` also persists both ids)

Why it matters:
- This is not just naming. It means current webchat has:
  - a **routing id** (`convID` / topic key / hydration key)
  - a **backend event/session id** (`SessionID` in event metadata)
- EVT-STREAM-002 deliberately removes this split by making `SessionId` the universal key.
- If you carry webchat over unchanged, the new framework will immediately leak dual identity semantics into every API.

Reuse verdict:
- **Do not reuse this identity model.**

Recommended rearchitecture:
- In EVT-STREAM-002, the substrate should have only one routing identity: `SessionId`.
- If applications need a stable user-facing “conversation id”, put that in:
  - `Session.Metadata`, or
  - application-defined timeline entities, or
  - application command payloads.

Do not let the substrate itself own both `ConversationId` and `SessionId`.

---

## 4.2 The current projection pipeline is chained through SEM/UI frames, not parallel from backend events

### Issue: timeline projection currently consumes UI-facing SEM frames, not canonical backend events

Problem:
The clean-room design's key architectural claim is:

> one backend-event stream -> **two sibling projections**
> - backend event -> UI events
> - backend event -> timeline entities

Current webchat does something different. In `ConvManager.GetOrCreate`, stream consumption ends in this callback:

```go
func(e events.Event, _ StreamCursor, frame []byte) {
    if c.pool != nil {
        c.pool.Broadcast(frame)
    }
    if c.semBuf != nil {
        c.semBuf.Add(frame)
    }
    if c.timelineProj != nil {
        _ = c.timelineProj.ApplySemFrame(c.baseCtx, frame)
    }
}
```

And `TimelineProjector` parses SEM envelopes and switches on UI-ish semantic event names:

```go
func (p *TimelineProjector) ApplySemFrame(ctx context.Context, frame []byte) error {
    var env semEnvelope
    if err := json.Unmarshal(frame, &env); err != nil { return nil }
    ...
    switch env.Event.Type {
    case "llm.start", "llm.thinking.start":
    case "llm.delta", "llm.thinking.delta":
    case "llm.final":
    ...
    }
}
```

Where to look:
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/stream_coordinator.go`
- `pinocchio/pkg/webchat/timeline_projector.go`
- `pinocchio/pkg/webchat/sem_translator.go`

Why it matters:
- Timeline projection is downstream of UI translation.
- That means the timeline store is not built from the most canonical event form.
- Any information lost, coalesced, renamed, or dropped by SEM translation is already gone before hydration sees it.
- It also duplicates reduction logic in two places: one for UI/SEM translation and another for timeline reconstruction.

This is the **single biggest reason not to turn current webchat directly into the new substrate**.

Reuse verdict:
- **Do not reuse this pipeline shape.**
- Reuse only the high-level lesson that “one consumed stream fans out to both websocket output and hydration persistence”.

Cleanup sketch:

```text
current webchat:
  Geppetto event -> SEM frame -> websocket
                       \
                        -> TimelineProjector parses SEM frame -> store

EVT-STREAM-002 target:
  Backend event -> UIProjection -> UIEvent -> websocket
               \
                -> TimelineProjection -> TimelineEntity batch -> store
```

---

## 4.3 Current command handling is chat-first, not framework-first

### Issue: command admission exists, but it is specialized to prompt submission and runner start

Problem:
The new framework wants generic synchronous command dispatch of `(Name, Payload, ConnectionId, SessionId)` to registered handlers. Current webchat has a useful admission/start pattern, but it is very chat-specific.

Where to look:
- `pinocchio/pkg/webchat/chat_service.go`
- `pinocchio/pkg/webchat/runner.go`
- `pinocchio/pkg/webchat/llm_loop_runner.go`
- `pinocchio/pkg/webchat/send_queue.go`

Example:

```go
type StartRequest struct {
    ConvID             string
    SessionID          string
    RuntimeKey         string
    RuntimeFingerprint string
    Sink               events.EventSink
    Timeline           TimelineEmitter
    Payload            any
    Metadata           map[string]any
}
```

And the concrete runner only accepts `LLMLoopStartPayload`:

```go
payload, ok := req.Payload.(LLMLoopStartPayload)
if !ok {
    return StartResult{}, errors.New("llm loop runner payload has unexpected type")
}
```

Why it matters:
- The overall pattern is useful:
  - synchronously validate/prepare work
  - start async processing
  - return immediate response / handle
- But the surface is not a generic framework contract:
  - `Payload any`
  - `Metadata map[string]any`
  - direct exposure of `events.EventSink` and `TimelineEmitter`
  - chat prompt queue/idempotency baked in

Reuse verdict:
- **Reuse the pattern, not the API.**
- `ChatService` + `LLMLoopRunner` should become an **example backend** on top of EVT-STREAM-002, not the substrate's handler model.

Recommended design improvement to EVT-STREAM-002:
- Consider a middleware/decorator layer around `CommandHandler` for common behaviors webchat already needed:
  - idempotency
  - queueing / concurrency policy
  - logging / tracing
  - auth context injection

That lets chat-specific prompt queueing survive as a reusable add-on instead of framework core.

---

## 4.4 Hydration storage is promising, but the contract is too narrow for the new framework

### Issue: the current `TimelineStore` interface is useful, but it is not yet the `HydrationStore` EVT-STREAM-002 needs

Problem:
Current store contract:

```go
type TimelineStore interface {
    Upsert(ctx context.Context, convID string, version uint64, entity *timelinepb.TimelineEntityV2) error
    GetSnapshot(ctx context.Context, convID string, sinceVersion uint64, limit int) (*timelinepb.TimelineSnapshotV2, error)
    UpsertConversation(ctx context.Context, record ConversationRecord) error
    GetConversation(ctx context.Context, convID string) (ConversationRecord, bool, error)
    ListConversations(ctx context.Context, limit int, sinceMs int64) ([]ConversationRecord, error)
    Close() error
}
```

Target design contract:
- atomic `Apply(session, ordinal, []entities)`
- `Snapshot(session, asOf)`
- `View(session)`
- `Cursor(session)`

Where to look:
- `pinocchio/pkg/persistence/chatstore/timeline_store.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_memory.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go`
- target: `design/02-technical-architecture-event-streaming-llm-framework.md` §9

Why it matters:
- Current implementations are valuable and should not be thrown away.
- But the interface is missing two critical things from EVT-STREAM-002:
  1. **atomic batch apply + cursor advance**
  2. a **projection-time read view** (`TimelineView`) for stateful reducers
- Right now versioning is per upsert call, not a formal substrate cursor.

Reuse verdict:
- **Reuse the implementations after refactoring the interface.**
- Do not freeze the current interface into EVT-STREAM-002.

Cleanup sketch:

```go
type HydrationStore interface {
    Apply(ctx context.Context, sid SessionId, ord uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sid SessionId, asOf uint64) (Snapshot, error)
    View(ctx context.Context, sid SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sid SessionId) (uint64, error)
}
```

Migration note:
- `chatstore` can likely become the seed for `hydration/memory` and `hydration/sql`.
- The conversation index/debug listing should be kept as a separate concern or optional extension.

---

## 4.5 Webchat still has ad-hoc hooks where EVT-STREAM-002 wants first-class slots

### Issue: hook accumulation is evidence that the generic seams were never named cleanly enough

Problem:
Current webchat exposes a growing list of router options and overrides:

- `WithBuildSubscriber`
- `WithTimelineUpsertHook`
- `WithEventSinkWrapper`
- `WithRuntimeComposer`
- `WithConvManager`
- `WithStepController`
- `WithTimelineStore`
- `WithTurnStore`
- ...

Where to look:
- `pinocchio/pkg/webchat/router_options.go`
- `pinocchio/pkg/webchat/types.go`

Why it matters:
- This is a classic sign that reusable extension points exist, but are not yet formalized.
- EVT-STREAM-002's cleaner model of exactly three application slots is a genuine simplification over webchat's accumulated override surface.

Reuse verdict:
- **Do not copy this option surface.**
- Use it as evidence for which seams are real.

Recommended reframing:
- Replace many current hooks with explicit framework roles:
  - `CommandHandler`
  - `UIProjection`
  - `TimelineProjection`
  - `SessionMetadataFactory`
  - optional transport/bus/storage options

That gives applications stronger extension points and the substrate a clearer contract.

---

## 4.6 Global projection registries should not survive into the substrate

### Issue: current timeline extensibility uses package-global state

Problem:
`timeline_registry.go` keeps projection handlers and runtime bridges in package globals:

```go
var (
    timelineHandlersMu sync.RWMutex
    timelineHandlers   = map[string][]TimelineSemHandler{}
    timelineRuntime    TimelineSemRuntime
)
```

Where to look:
- `pinocchio/pkg/webchat/timeline_registry.go`
- `pinocchio/pkg/webchat/timeline_js_runtime.go`

Why it matters:
- This is fine for one embedded webchat package in one process.
- It is a poor fit for a reusable framework where multiple hubs/applications/tests may coexist.
- EVT-STREAM-002 correctly scopes projections per `Hub`, not globally.

Reuse verdict:
- **Do not reuse this scoping model.**
- The idea of pluggable runtimes/reducers is useful, but the registry must be instance-scoped.

Potential future use:
- If EVT-STREAM-002 later wants composable projections or a scripting bridge, build it per-Hub, not as package-global mutable state.

---

## 4.7 Current manual timeline versioning is not strong enough for the new framework

### Issue: some timeline writes still derive versions from in-memory conversation state

Problem:
One path in `LLMLoopRunner` computes timeline version like this:

```go
version := uint64(1)
if conv != nil {
    conv.mu.Lock()
    if conv.lastSeenVersion > 0 {
        version = conv.lastSeenVersion + 1
    }
    conv.mu.Unlock()
}
return timeline.Upsert(ctx, ..., version)
```

Where to look:
- `pinocchio/pkg/webchat/llm_loop_runner.go`

Why it matters:
- This is okay for webchat's current single-process assumptions.
- It is not the robust cursor model EVT-STREAM-002 is aiming for.
- The new framework should avoid any version allocation based on ad-hoc mutable session state when the canonical event stream already exists.

Reuse verdict:
- **Do not carry this version allocation behavior into the framework.**

Better approach:
- all ordinals come from consumed backend events
- projection persistence advances cursor atomically with entity apply
- synthetic events (like seeding a user message) should themselves be emitted onto the backend event stream rather than sneaking around it via direct timeline writes

This is an important architectural refinement the new framework should make over current webchat.

---

## 5. What the new framework should explicitly learn from webchat

## 5.1 Keep route composition app-owned

Webchat's newer docs are right: let applications own `/chat`, `/ws`, auth middleware, and mount points. The substrate should expose handlers/transports, not hardcode server composition.

## 5.2 Keep policy/request resolution outside the core

`ConversationRequestResolver` is a good lesson. Transport adapters should accept a policy/resolver seam instead of embedding auth/profile logic into the substrate.

## 5.3 Make slow-client handling a transport concern

Webchat's `ConnectionPool` shows that backpressure handling is a transport responsibility, not a backend/application responsibility.

## 5.4 Ship idle eviction early

Webchat already proves this matters. EVT-STREAM-002 should not postpone lifecycle cleanup until “after the core works”.

## 5.5 Separate stable substrate slots from app-specific behaviors

Webchat accumulated hooks because it mixed generic concerns with chat specifics. EVT-STREAM-002 should preserve a small public contract and keep app-specific concerns in backend packages.

## 5.6 Make synthetic events first-class backend events

Webchat currently sometimes publishes SEM frames or direct timeline writes for convenience. EVT-STREAM-002 should route everything meaningful through canonical backend events so UI and hydration stay in sync by construction.

## 5.7 Consider optional command middleware from day one

Webchat's queue/idempotency logic is application-specific, but the pattern is common enough that the new framework may benefit from optional command middleware.

Possible future middleware set:
- idempotency
- per-session serialization / queueing
- tracing
- metrics
- auth context binding
- retry / dead-letter policy

This is not required for v1, but webchat is evidence that the need is real.

---

## 6. Suggested extraction strategy

## 6.1 Build the new substrate cleanly; do not start by renaming `pkg/webchat`

Recommended rule:
- `pkg/webchat` is a **reference implementation and donor**, not the core package to evolve in place.

Reason:
- too much current meaning is chat-specific
- dual-id semantics are baked in
- the projection chain is wrong for the new architecture
- public names (`Conversation`, `/chat`, `SubmitPrompt`, `LLMLoopRunner`) bias the design toward “webchat with more knobs” instead of a generic LLM/event substrate

## 6.2 Port donor pieces in this order

### First wave: reuse almost as-is

1. **sequence derivation logic** from `stream_coordinator.go`
2. **websocket per-client send-queue pattern** from `connection_pool.go`
3. **in-memory + SQLite hydration store implementations** from `chatstore`, behind the new `HydrationStore` interface
4. **idle eviction logic shape** from `conv_manager_eviction.go`

### Second wave: adapt structurally

5. `ConvManager` -> `SessionRegistry` / session lifecycle manager
6. `StreamBackend` -> event-bus wiring helpers
7. `ConversationRequestResolver` -> transport adapter policy seam
8. request/attach helpers from `http/api.go` -> optional adapters on top of `Transport`

### Third wave: keep only as backend/application code

9. `ChatService`
10. `LLMLoopRunner`
11. `TimelineProjector`
12. `EventTranslator`

These should inform the first **chat backend example**, not the substrate.

---

## 7. Recommended changes to EVT-STREAM-002 itself, based on webchat evidence

## 7.1 Keep `SessionId` as the only substrate id, but explicitly allow app-level conversation ids in metadata

Current webchat is evidence that applications often want a long-lived human-facing conversation id and a lower-level execution/session id. EVT-STREAM-002 should still keep only one substrate id, but it should explicitly say:

- if you need extra identity, keep it in application metadata/entity state
- do not make it a second substrate routing key

## 7.2 Add a note that transport adapters should own request/policy resolution

The technical design already describes transports, but it would benefit from explicitly calling out the `ConversationRequestResolver`-style seam webchat has discovered.

## 7.3 Consider a small command middleware story

The current design's `CommandHandler` API is good, but webchat shows repeated need for behaviors around the handler. Even if not implemented in v1, the design doc should leave room for middleware/decorators.

## 7.4 Clarify that projection input must be canonical backend events, never UI envelopes

The design already implies this, but webchat shows why it matters. Make it explicit in the doc to prevent accidental “UI projection first, hydration second” implementations.

## 7.5 Consider a separate optional session index/debug capability

Webchat's `ConversationRecord` persistence and debug APIs are not core substrate concerns, but they are operationally useful. EVT-STREAM-002 may eventually want an optional `SessionIndexStore` or a small debug/inspection extension package.

This should remain optional so the core stays small.

---

## 8. Final recommendation

If the goal is:

> create a reusable generic framework that makes it easy to build LLM applications (chat, agent orchestration, etc.) while reusing the hard parts

then the right move is:

1. **Do not treat `pinocchio/pkg/webchat` as the framework.**
2. **Do treat it as proof that several hard parts are already understood.**
3. **Extract and adapt the generic parts into a new clean package.**
4. **Rebuild webchat on top of that package as the first concrete backend.**

The existing webchat is most valuable as a source of:
- operational transport behavior
- stream ordering behavior
- hydration-store implementation knowledge
- lifecycle/eviction lessons
- API ownership lessons

It is least suitable as the direct basis of the new substrate in:
- identity model
- canonical event model
- projection architecture
- public API shape

So the answer is:

- **yes**, significant parts can be reused
- **no**, the package should not be reused wholesale
- **yes**, the new design should be adjusted slightly based on webchat's lived lessons: explicit lifecycle cleanup, policy seams, transport backpressure, optional middleware, and clear separation between canonical backend events and downstream projections

---

## Appendix — most reusable existing files

### Best direct donors
- `pinocchio/pkg/webchat/stream_coordinator.go`
- `pinocchio/pkg/webchat/connection_pool.go`
- `pinocchio/pkg/webchat/conv_manager_eviction.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_memory.go`
- `pinocchio/pkg/persistence/chatstore/timeline_store_sqlite.go`
- `pinocchio/pkg/webchat/http/api.go` (transport-adapter patterns only)

### Best conceptual donors
- `pinocchio/pkg/webchat/conversation.go`
- `pinocchio/pkg/webchat/conversation_service.go`
- `pinocchio/pkg/webchat/stream_backend.go`
- `pinocchio/pkg/webchat/timeline_registry.go`

### Do not promote into substrate core
- `pinocchio/pkg/webchat/chat_service.go`
- `pinocchio/pkg/webchat/llm_loop_runner.go`
- `pinocchio/pkg/webchat/timeline_projector.go`
- `pinocchio/pkg/webchat/sem_translator.go`
- `pinocchio/pkg/webchat/router.go` (as the central public concept)
