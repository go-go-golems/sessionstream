---
Title: Webchat Reuse Analysis — Independent Read (vs pinocchio/pkg/webchat)
Ticket: EVT-STREAM-002
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - chat
    - websocket
    - backend
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: design/02-technical-architecture-event-streaming-llm-framework.md
      Note: "The new framework's technical architecture — the target of this reuse analysis."
    - Path: design/01-architecture-analysis-event-streaming-llm-framework.md
      Note: "The clean-room analysis the new framework derives from."
    - Path: ../../pinocchio/pkg/webchat/conversation.go
      Note: "Existing per-conversation state object (the largest delta vs. the new Session)."
    - Path: ../../pinocchio/pkg/webchat/stream_coordinator.go
      Note: "Existing per-stream Seq stamping at consumption (matches new design's Ordinal idea)."
    - Path: ../../pinocchio/pkg/webchat/timeline_projector.go
      Note: "Existing SEM-frame -> timeline-entity reduce loop with throttling and id correlation."
    - Path: ../../pinocchio/pkg/webchat/connection_pool.go
      Note: "Existing websocket pool with idle reaper — seed of the new design's liveness contract."
ExternalSources: []
Summary: "Independent reuse analysis comparing the existing pinocchio/pkg/webchat implementation against the new EVT-STREAM framework. Identifies what is reusable verbatim, what is reusable as pattern with surgery, what should be rewritten, and what the existing code teaches the new design (rearchitecting suggestions). Written without consulting any prior reuse analysis."
LastUpdated: 2026-04-19T16:00:00-04:00
WhatFor: "Decide concretely how to migrate from the current webchat implementation to the new framework: which files lift directly, which need surgery, which get discarded, and which gaps in the new design the existing code fills in."
WhenToUse: "When planning the implementation roadmap, when scoping the chat-example application, or when arguing about whether a given existing concern belongs in the substrate or the application layer."
---

# Webchat Reuse Analysis — Independent Read (vs `pinocchio/pkg/webchat`)

> **Method note.** This document is an independent read of `pinocchio/pkg/webchat/` against the new framework specified in `design/02-technical-architecture-event-streaming-llm-framework.md`. A colleague has produced a parallel analysis (`design/03-…`); per directive I have not read it. Any agreement with that document is convergent, not derivative; any disagreement is honest. Files cited are direct reads (`conversation.go`, `stream_coordinator.go`, `chat_service.go`, `timeline_projector.go`, plus the package summary `doc.go` / `types.go`) plus a directed mapping pass over the rest of the package.

## 0. Scope and method

I read the package's central objects first-hand and used a directed exploration pass over the remaining files to build a subsystem map. The questions I asked of each subsystem were:

1. What does it own? What does it depend on?
2. Where would it live in the new framework's three-layer model (Backend / Generic / Client)?
3. Does it embody a contract the new design names, or a contract the new design omits?
4. If we were extracting today, would it lift verbatim, lift with surgery, or get rewritten?

The output is this document — organised as bottom-line, side-by-side comparison, then a four-bucket audit (verbatim reuse, pattern reuse, rewrite, lessons-for-the-design), then an extraction roadmap.

## 1. Bottom line, up front

The existing webchat is a working, opinionated implementation of roughly **70% of what the new framework calls "the substrate"** — but it does not draw the substrate/application line in the same place. The existing package is a vertical, chat-shaped slice in which session lifecycle, websocket transport, watermill wiring, SEM event translation, timeline projection, idempotency, eviction, and chat-specific inference orchestration are all intertwined inside one `Router` aggregator.

The new framework is a horizontal substrate with three pluggable application slots (`CommandHandler`, `UIProjection`, `TimelineProjection`) and an application-owned event bus. The code shapes overlap heavily; the **ownership boundaries differ at almost every seam**.

The right migration path is therefore neither "rewrite from scratch" nor "rename in place" — it is a **directed extraction**: pull the substrate out of `webchat`, leaving a much smaller `chat-backend` example behind that registers itself with the substrate via the three slots. Roughly:

| Existing webchat code         | Destination in new world                              | Effort                |
|-------------------------------|-------------------------------------------------------|-----------------------|
| `~70%`                        | `evtstream/*` substrate                               | extract + reshape API |
| `~20%`                        | `examples/chat/*` backend application                 | shed dependencies     |
| `~10%`                        | discarded (pinocchio runtime coupling, dead seams)    | delete                |

The remainder of this document justifies that table at the file/type level and pulls out specific design lessons.

## 2. The two designs, side by side

| Concern                     | Existing webchat                                                                                  | New EVT-STREAM framework                                                          |
|-----------------------------|---------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------|
| **Top-level aggregator**    | `Router` (holds bus, stores, services, conv manager, mux, …)                                      | `Hub` (holds bus consumer, store, transport, slot registry — no HTTP mux)         |
| **Session object**          | `Conversation{ID, SessionID, Pool, Stream, llm, requests, semBuf, timelineProj, …}` (~25 fields) | `Session{Id, Metadata}` (2 fields)                                                |
| **Session id**              | `ConvID` (string) + `SessionID` (UUID); both used                                                 | `SessionId` (string) — single key                                                 |
| **Connection id**           | `wsConn` interface as the map key; no opaque id                                                   | `ConnectionId` (typed string), substrate-only                                     |
| **Event bus**               | `StreamBackend` owns watermill `EventRouter`; per-conv subscribers built on demand                | Application owns watermill pub/sub; substrate is a consumer                       |
| **Event ordinal**           | `StreamCursor{StreamID, Seq}` stamped per-stream by `StreamCoordinator` at consumption            | `Event.Ordinal` stamped per-`SessionId` by substrate at consumption               |
| **Timeline entity type**    | `TimelineEntityV2{Id, Kind, Props *structpb.Struct}` (proto)                                      | `TimelineEntity{Kind, Id, Payload proto.Message, Tombstone}`                      |
| **Wire payload**            | `semEnvelope{Sem bool, Event{Type, ID, Seq, StreamID, Data}}` (JSON wrapping geppetto events)     | Typed `UIEvent{Name, Payload proto.Message}`                                      |
| **Application slots**       | One: an `LLMLoopRunner` (specialised for chat inference)                                          | Three: `CommandHandler`, `UIProjection`, `TimelineProjection`                     |
| **Command dispatch**        | HTTP handler → `ChatService.SubmitPrompt` → `LLMLoopRunner.Start`                                 | Wire frame or `Hub.Submit` → registered handler keyed by command Name             |
| **Hydration story**         | `TimelineService.Snapshot(convID, sinceVersion, limit)` + `semFrameBuffer` (recent frame tail)   | `HydrationStore.Snapshot(sid, asOf)` (snapshot only, no frame tail in spec)       |
| **Connection lifecycle**    | `ConnectionPool` with idle timer + reaper                                                         | "Open question" — substrate spec defers to §17.3                                  |
| **Idempotency**             | `SendQueue` deduplicates submissions by `IdempotencyKey`                                          | Not in the spec                                                                   |
| **Application extension**   | Global `RegisterTimelineHandler` + optional JS runtime via `SetTimelineRuntime`                   | Per-Hub `RegisterUIProjection` / `RegisterTimelineProjection` (one each)         |
| **HTTP integration**        | Mounted by Router (`/api/timeline`, `/api/debug`, optional UI)                                    | Out of scope; substrate exposes a `Transport` interface, ws is one impl          |

The two share the same skeleton; they disagree on **where each bone connects**.

## 3. Reusable verbatim (or near-verbatim)

For each item: source file, the new home, and the shape of the surgery. These are pieces where the existing code already does what the substrate needs.

### 3.1 SEM-frame-to-timeline-entity projection logic

**Source.** `timeline_projector.go` — `TimelineProjector` plus its caches (`msgRoles`, `msgContents`, `lastMsgWrite`, `toolNames`, `toolInputs`).

**Why it's gold.** This is the working implementation of the "first event creates the entity, subsequent events reduce into it" pattern that the new design's §7 worked example calls out. It already handles cross-event correlation by id (delta events match their start events via `msg.Id`), per-id throttling on high-frequency events (`llm.delta` skips persistence if < 250 ms since the last write — see §6.4), and explicit projection to typed proto messages (`MessageSnapshotV1`, `ToolCallSnapshotV1`).

**New home.** The chat application's `TimelineProjection` in `examples/chat/timeline_projection.go`.

**Surgery.** ~150 lines changed: drop the direct `store.Upsert` call, add the `view TimelineView` parameter, change the return type from `error` to `([]evtstream.TimelineEntity, error)`. The cache fields become projection state held by the projection object. The global `RegisterTimelineHandler` registry goes away — the projection is now a single object with a `switch` on `ev.Name`, identical in shape to the worked example.

### 3.2 Per-stream sequence stamping at the subscriber

**Source.** `stream_coordinator.go` — `StreamCursor{StreamID, Seq}` plus `seq atomic.Uint64` and `nextSeq(streamID)`. Stamp happens inside `consume(...)` after `Subscribe(ctx, topicForConv(...))`.

**Why it's gold.** This is *exactly* the new design's "ordinal stamped at consumption (subscriber-side), monotonic per partition" pattern. The existing code stamps at the per-stream level; the new design stamps at the per-`SessionId` level — but the mechanism is identical, and the existing code is the working reference.

**New home.** `evtstream/bus/watermill.go`. The atomic counter logic lifts directly. The partitioning key changes from `StreamID` to `SessionId` (whether the new granularity is correct is itself a design question — see §6.1).

**Surgery.** ~30 lines. Drop `StreamID` from the cursor; key the counter map by `SessionId`. Persist the per-`SessionId` cursor through `HydrationStore.Cursor` so restarts resume cleanly (the existing code does *not* do this — its sequences reset on process restart).

### 3.3 Connection pool with idle timer and reaper

**Source.** `connection_pool.go` — `ConnectionPool{convID, conns, idleTimer, idleTimeout, onIdle}` with `Add/Remove/Broadcast/SendToOne` and per-conn buffered send channels (default 64 slots).

**Why it's gold.** The new design's §17.3 explicitly lists "stale-connection handling, ticks" as an open question. The existing code has a working answer: per-pool idle timer that fires `onIdle` on empty state after a configurable timeout. This is the seed of the substrate's liveness contract.

**New home.** `evtstream/transport/ws/connection_pool.go` (private to the websocket transport). The pool is a transport-internal data structure; nothing application-facing should reference it.

**Surgery.** Re-key from `convID`/`wsConn` to `SessionId`/`ConnectionId`. Add a per-connection liveness "last-tick" field so the reaper can target individual stale sockets, not just empty pools (see §6.5).

### 3.4 Recent-frame circular buffer (catch-up tail)

**Source.** `sem_buffer.go` — `semFrameBuffer{frames, max}` with `Snapshot() [][]byte`.

**Why it's gold.** Hydration in the new design is currently *snapshot-only*: a reconnecting client gets the timeline state at a point in time and then live UI events from that ordinal forward. But **the gap between "snapshot taken" and "live subscription started" is a race window** — events that fall in that gap are never delivered to the reconnecting client. The existing buffer is exactly the catch-up tail that closes the window.

**New home.** Add to the new design at `evtstream/buffer/` (or, better, fold into the substrate's per-session bookkeeping). This is a **gap in the new design that the existing code already fills**. See §6.3.

### 3.5 Idempotency-key tracking on submissions

**Source.** `send_queue.go` — `chatRequestRecord{IdempotencyKey, Status, Response}` plus the `requests map[string]*chatRequestRecord` on `Conversation`.

**Why it's gold.** A retried command should not double-execute. The existing code: if the same `IdempotencyKey` is seen twice within the conversation's request map, the second call returns the cached response. Real-world chat clients retry under network failures; without this, a single user prompt can run twice.

**New home.** Substrate, on `Hub.Submit` and on the inbound websocket-command path. Add `IdempotencyKey` as an optional field on `Command` and on `Hub.Submit`'s signature; store the active-request map on substrate-side per-session bookkeeping (not on `Session.Metadata`, which is for the application). See §6.6.

### 3.6 Per-conversation subscriber-from-global-publisher pattern

**Source.** `StreamBackend.BuildSubscriber(ctx, convID)` plus the `topicForConv(convID)` convention.

**Why it's gold.** The natural shape for the new design is: one application-owned bus, per-`SessionId` topics, per-`SessionId` subscribers built on demand. The existing code shows the watermill-API ergonomics that work — particularly the `BuildSubscriber` factory pattern that gives each subscription its own backpressure boundary.

**New home.** `evtstream/bus/watermill.go`. Lift the recipe; the topic naming convention (`topicForConv` → `topicForSession`) carries.

### 3.7 Proto schemas already exist

**Source.** `pinocchio/pkg/sem/pb/proto/sem/{base,timeline}/` — the `sem.base.*` event types (LlmStart, LlmDelta, LlmFinal, ToolStart, ToolDone, ToolResult) and `sem.timeline.*` snapshot types (MessageSnapshotV1, ToolCallSnapshotV1, TimelineEntityV2, TimelineSnapshotV2).

**Why it's gold.** The new design specifies "single source of truth via buf, codegen Go + TS". These protos are *already* the single source of truth in the existing system; they're already generated for Go and used by `protojson` / `structpb`. The chat example in the new design (`proto/api/v1/chat.proto`) can either reuse these as-is or reshape them slightly.

**Surgery.** Add `protoc-gen-es` to the `buf.gen.yaml` to also produce TS — which the existing setup does *not* do today (the existing TS client uses ad-hoc JSON shapes). Optionally rename `_v2` → no-suffix and provide a one-shot migration. No semantic changes.

### 3.8 Turn snapshot persistence pattern (the lesson, not the code)

**Source.** `turn_persister.go`, `turn_snapshot_hook.go`. The `SnapshotHook` plumbing inside the geppetto inference loop calls `TurnStore.Save` at "interim" and "final" phases.

**Why it's interesting.** The notion of *phased* snapshots (in-progress vs. terminal) is a useful idea the new design currently does not have. It maps cleanly onto timeline projection: an entity can be persisted with `Streaming: true` mid-stream and `Streaming: false` at finalisation — which is exactly what the `ChatMessage` example does. The existing turn store is pinocchio-specific (turns of an LLM conversation); the *pattern* is general.

**New home.** Don't lift the code. Lift the lesson: document the "interim" vs. "final" pattern in the new design's projection guidelines.

### 3.9 Optional — the JS timeline runtime

**Source.** `timeline_js_runtime.go` — `JSTimelineRuntime` with `goja`, exposing `require("pinocchio").timeline.registerSemReducer(eventType, fn)`.

**Why it's interesting.** This is the pinocchio answer to "let users add timeline reducers without forking the library". The new design doesn't have a runtime extension story (the `TimelineProjection` is registered at boot and is fixed for the process). For some applications — debugging, A/B-ing projection logic, customising views per tenant — runtime-loadable projection code is genuinely useful.

**New home.** Out of scope for v1. Worth noting in the open questions: a future `WithProjectionRuntime(r ProjectionRuntime)` option could let applications load JS or wasm reducers. Keep the existing code in `examples/extensions/js-timeline/` as a reference.

## 4. Reusable as pattern with surgery (medium effort)

The *idea* carries; the *shape* changes substantially.

### 4.1 `Conversation` → `Session` (lose 23 fields)

The existing `Conversation` struct (`conversation.go:24-58`) has ~25 fields: `ID`, `SessionID`, `Sink`, `sub`, `subClose`, `pool`, `stream`, `RuntimeKey`, `RuntimeFingerprint`, `ResolvedProfileMetadata`, `resolvedInferenceSettings`, `resolvedRuntime`, `profileVersion`, `llm`, `activeRequestKey`, `queue`, `requests`, `lastActivity`, `createdAt`, `semBuf`, `timelineProj`, `lastSeenVersion`, plus mutex and base context.

The new design's `Session` has two fields: `Id`, `Metadata`. **Twenty-three fields disappear** — but every one of them lands somewhere. The migration map:

| Existing `Conversation` field             | Destination                                                                                  |
|-------------------------------------------|----------------------------------------------------------------------------------------------|
| `ID`, `SessionID`                         | `Session.Id` (single field; pick one — almost certainly `SessionID`)                         |
| `Sink`                                    | `EventPublisher` injected into command handlers (substrate-managed)                          |
| `sub`, `subClose`, `stream`, `pool`       | Substrate-internal (`Hub` + `Transport` own these; never on the session)                     |
| `RuntimeKey`, `RuntimeFingerprint`, `ResolvedProfileMetadata`, `resolvedInferenceSettings`, `resolvedRuntime`, `profileVersion` | `Session.Metadata` (typed struct in the chat application) |
| `llm`                                     | Application-side state in the chat backend (not in the substrate)                            |
| `activeRequestKey`, `queue`, `requests`   | Substrate's idempotency table (see §3.5 / §6.6)                                              |
| `lastActivity`, `createdAt`               | Substrate's session bookkeeping (private; not exposed)                                       |
| `semBuf`                                  | Substrate's recent-frame buffer (see §3.4 / §6.3)                                            |
| `timelineProj`                            | The application-supplied `TimelineProjection`, registered once on the Hub (not per-session)  |
| `lastSeenVersion`                         | Substrate's per-session ordinal counter                                                      |

**Most of the fields that disappear from `Session` are state the substrate owns**, not state the application owns. The `Session.Metadata` slot is a much smaller bag.

### 4.2 `ConvManager.GetOrCreate` → substrate session lifecycle

The existing `GetOrCreate` is complex: it takes a `convID`, runtime key, fingerprint, and rebuilds the LLM engine + event sink if the fingerprint changed. The new design's session lifecycle is "lazy: create on first reference, factory builds metadata, store cursor consulted to resume ordinal" (§12.1).

**The runtime-rebuild logic does not migrate.** That entire concept — a session whose underlying inference engine can be swapped mid-conversation if the profile changes — is a *chat application* concern. In the new design, this becomes: the chat application's `SessionMetadataFactory` returns the typed metadata; if the application wants to swap the underlying engine, it does so internally (and the substrate doesn't care). The substrate has nothing to do with it.

What *does* migrate: the laziness, the per-session locking, the per-session subscriber wiring. ~80 lines, mostly slimmer.

### 4.3 `ChatService.SubmitPrompt` → `Hub.Submit` + `RegisterCommand("StartInference")`

The existing surface (`ChatService.SubmitPrompt`) takes a `SubmitPromptInput`, resolves a conversation, idempotency-checks, queues if busy, and starts a runner. In the new design, all that work splits:

- **The HTTP handler** (chat application) parses the incoming request and calls `hub.Submit(ctx, sid, "StartInference", &chatpb.StartInference{Prompt: …})`. ← The application owns the HTTP layer, not the substrate.
- **`Hub.Submit`** (substrate) handles idempotency dedup (§3.5 / §6.6), enqueues if the session is busy, and dispatches to the registered handler.
- **The `StartInference` `CommandHandler`** (chat application) does what `LLMLoopRunner.Start` does today: builds a turn, kicks off the inference loop, publishes events on the bus.

The existing `SendQueue` queueing semantics survive — they just move into the substrate. Most of `LLMLoopRunner.Start` (the geppetto/turns/toolloop wiring) survives too — it just becomes the body of the registered `CommandHandler`.

### 4.4 `TimelineService.Snapshot` → `HydrationStore.Snapshot`

The existing `TimelineService.Snapshot(ctx, convID, sinceVersion, limit)` returns a `*timelinepb.TimelineSnapshotV2`. The new design's `HydrationStore.Snapshot(ctx, sid, asOf)` returns a `Snapshot{SessionId, Ordinal, Entities}`. Same shape, different types. Lifts directly with a rename and a slight return-type reshape; the underlying `chatstore.TimelineStore` SQLite implementation lifts with it (it becomes the `hydration/sql` package's content).

### 4.5 `LLMLoopRunner` → application command handler

The existing `LLMLoopRunner.Start(ctx, req StartRequest)` is the entire "what happens when a user clicks send" logic: append the user turn, publish a `chat.message` SEM event, project it, configure the toolloop step controller, attach a snapshot hook, kick off async inference, return a wait handle.

In the new design **all of this is application-layer** — the substrate doesn't know what an LLM is. The `LLMLoopRunner` becomes the body of a `CommandHandler` registered in `examples/chat/`:

```go
func startInference(
    ctx context.Context,
    cmd evtstream.Command,
    sess *evtstream.Session,
    pub evtstream.EventPublisher,
) error {
    // Everything LLMLoopRunner.Start does today, but using `pub` to publish
    // events instead of the geppetto event sink, and reading `sess.Metadata`
    // to get the runtime profile instead of looking it up by ConvID.
}
```

The actual *inference machinery* (geppetto engine, toolloop, turn snapshot hook) is unchanged. What changes is the wiring: events go through `EventPublisher.Publish` instead of `events.EventSink.Sink` — they end up in the same watermill bus underneath.

## 5. Rewritten or discarded

### 5.1 `Router` aggregator → `Hub`, but unbundled

The existing `Router` (`router.go`, ~70 fields visible in `types.go:27-72`) is the integration point that holds *everything*. The new `Hub` is much smaller — it holds only the bus consumer wiring, the slot registry, and the substrate-internal session/connection bookkeeping. Removed from the Hub-equivalent:

- HTTP `mux`, `staticFS`, route mounting (application owns HTTP).
- `db *sql.DB`, `timelineStore`, `turnStore` (the application configures the `HydrationStore`; the substrate consumes the interface).
- `toolFactories` (chat-application concern).
- `enableDebugRoutes` (application concern).
- `runtimeComposer`, `eventSinkWrapper`, `buildSubscriberOverride`, `timelineUpsertHookOverride` (these compose the existing system together; the new system has explicit pluggable interfaces instead of optional override hooks).

The existing `Router.BuildHTTPServer` method has no analogue in the new substrate — it's an application-layer helper. That is deliberate.

### 5.2 The split `ChatService` / `StreamService` / `ConversationService` / `TimelineService`

The existing service split (`chat_service.go`, `conversation_service.go`, `timeline_service.go`, plus `StreamHub`) was an attempt to factor the `Router`'s responsibilities into smaller objects. The new design dissolves the split entirely: there is just the `Hub` plus the three application slot interfaces. None of these service objects survive as such.

Their *responsibilities* still exist:

- `ChatService.SubmitPrompt` — moves to `Hub.Submit` + `CommandHandler`.
- `StreamHub.AttachWebSocket` — moves to `transport/ws.New(addr)` + `Transport.Start`.
- `ConversationService.ResolveAndEnsureConversation` — disappears entirely; sessions are lazy.
- `TimelineService.Snapshot` — moves to `HydrationStore.Snapshot`.

### 5.3 The `infruntime.RuntimeBuilder` / `ProfileRuntime` / runtime-fingerprint chain

This entire concept — that a conversation has a "runtime profile" with a fingerprint that, if changed, causes the engine + sink to be rebuilt mid-session — is **chat-application-specific** and does not belong in the substrate. The new framework should have no awareness of "profiles" or "engines".

The information in `RuntimeKey` / `RuntimeFingerprint` / `ResolvedProfileMetadata` / `resolvedInferenceSettings` / `resolvedRuntime` lives in the chat application's `Session.Metadata`. The "rebuild on change" logic, if still wanted, lives in the chat application's `CommandHandler` (which can decide to re-instantiate its inference engine when it sees metadata change).

This represents maybe ~20% of the existing webchat code by line count and is the largest lift-and-shift to the application side.

### 5.4 The HTTP layer (`http/`, `router_debug_*`, `router_handlers_*`)

The existing `http/` subdirectory and the various `router_handlers_*` files mount endpoints (`/api/timeline`, `/api/debug`, etc.) and handle the chat-submission HTTP shape. None of this belongs in the substrate. It moves wholesale to `examples/chat/http/`.

### 5.5 Tool registration (`RegisterTool`, `toolFactories`)

`Router.RegisterTool` and the `toolFactories map[string]infruntime.ToolRegistrar` are pure chat-application concerns (LLM tool calling). Out of the substrate.

### 5.6 The SEM envelope wrapper (partial)

The existing wire payload is a JSON envelope:

```go
type semEnvelope struct {
    Sem   bool `json:"sem"`
    Event struct {
        Type     string          `json:"type"`
        ID       string          `json:"id"`
        Seq      uint64          `json:"seq"`
        StreamID string          `json:"stream_id"`
        Data     json.RawMessage `json:"data"`
    } `json:"event"`
}
```

The new design's wire payload is a typed `UIEvent{Name, Payload proto.Message}`. The envelope concept (a discriminated union of "this is a substrate frame" vs. "this is something else on the same socket") is *less* useful in the new design because the wire is intentionally homogeneous. Drop the envelope.

But: the *fields* of the envelope (Type/Id/Seq/StreamID/Data) are essentially the same as the new design's `(Name, Id, Ordinal, SessionId, Payload)`. The data model lives on; the JSON wrapping doesn't.

## 6. What the existing code teaches the new design

This is the most important section. Reading the existing webchat surfaced a number of design points that the new framework spec either gets wrong, gets too-coarse, or omits entirely. I recommend folding all of these into the next iteration of `design/02`.

### 6.1 Per-stream ordinals vs. per-session ordinals

The existing system stamps `Seq` *per stream*, not per conversation: a single `SessionID` can have many `StreamID`s (typically one per inference turn). The new design stamps `Ordinal` *per `SessionId`*. There is a real question whether the new granularity is right:

- **Argument for per-session ordinals** (current design): simpler hydration, single cursor, single resume point.
- **Argument for per-stream ordinals**: parallelism. If two distributed workers contribute events for the same session under different conceptual sub-streams (e.g., one for inference, one for tool-execution telemetry), per-stream ordinals avoid contention on a single counter.

**Recommendation:** keep per-session as the default for v1 (simpler), but add an optional `Stream string` field to `Event` that, when set, partitions ordinal assignment within a session. The substrate's bookkeeping then becomes `(SessionId, Stream) -> ordinal`; clients can either ignore the stream dimension or filter by it. This is a small addition that costs almost nothing and unlocks the existing system's parallelism story.

### 6.2 Server-side projection of UI events vs. raw event passthrough

The existing system broadcasts the SEM-translated frame *directly* to websocket clients — there is no separate `UIProjection` step that produces a different shape for the wire. The client receives the same payload that the timeline projector consumes, and does its own client-side rendering decisions.

The new design separates `UIProjection` (events to wire) from `TimelineProjection` (events to store). This is cleaner, but carries a real cost: **the application now writes the same logic twice** for any decision that affects both UI and timeline (e.g., "skip empty deltas"). The existing single-payload approach is more compact.

**Recommendation:** the new design is correct to separate them, but the doc should explicitly show **two patterns** for `UIProjection`:

- **Pure passthrough** (return the backend event verbatim as a UIEvent) — the existing webchat behaviour.
- **Derived UI events** (transform shapes for UI ergonomics) — the case the new design is optimised for.

Either is valid; making the choice explicit prevents accidental divergence between `TimelineProjection` and `UIProjection` logic.

### 6.3 Hydration is not just a snapshot

The existing `semFrameBuffer` (see §3.4) exists because **a snapshot taken at ordinal N and a live subscription started at ordinal N + k** has a race window of `k` events that no client ever sees. The buffer holds the recent frame tail so a reconnecting client can stitch snapshot + tail + live without gaps.

The new design's `HydrationStore.Snapshot` is currently snapshot-only. **This is a bug-shaped omission.**

**Recommendation:** add to the new design either

- a `Tail(ctx, sid, fromOrdinal) ([]Event, error)` method on `HydrationStore`, *or*
- a substrate-internal `RecentFrames(sid)` buffer that the websocket transport reads on subscribe.

The latter is better — it puts the catch-up tail at the same layer that does the live fan-out, so subscription is one atomic "snapshot + tail-from-snapshot-ordinal + live-from-tail-end" operation.

### 6.4 High-frequency event throttling

The existing `TimelineProjector` (`timeline_projector.go`) has an explicit throttle: for `llm.delta` events, it skips persistence if less than 250 ms have passed since the last write for the same message id. Without this, a single token-stream session writes thousands of entities per second to the store and the SQLite store can't keep up.

The new design has no notion of throttling. Projections are called for every event. **This will break the moment a real LLM session attaches.**

**Recommendation:** the new design needs one of

- **A min-write-interval declared per `(Kind, Id)`** at projection registration time, enforced by the substrate (the substrate batches `Apply` calls within the interval).
- **A per-projection batching window** (the substrate calls `Apply` with the *aggregated* result of all events in the last N ms).
- **A per-projection "is hot path" tag** that makes the substrate skip persistence entirely on streaming entities and only persist on a "settle" event (e.g., `InferenceFinished`).

The cleanest fit with the new design's `TimelineEntity` shape is the second option — the substrate naturally aggregates the last-write-wins for `(Kind, Id)` within a window. Add a `WithTimelineBatchWindow(d time.Duration)` `HubOption`.

### 6.5 Liveness has a working starting point

The existing `ConnectionPool` has a per-pool idle timer and an `onIdle` callback. This is not a complete liveness contract (no per-connection ping/pong, no client-side tick), but it's a working starting answer to the new design's open question §17.3.

**Recommendation:** the new design's liveness section should be promoted from "open question" to "v1 contract" with the following minimum:

- Per-connection client-side ping every `clientTickInterval` (default 30s).
- Per-connection server-side pong with `lastSeen` updated.
- Substrate-side reaper running every `reaperInterval` (default 60s) that closes connections whose `lastSeen` is older than `staleThreshold` (default 90s).
- `WithLiveness(opts LivenessOptions)` Hub option to override the defaults.

This is essentially what the existing `ConnectionPool.idleTimer` does, sharpened to the per-connection level instead of per-pool.

### 6.6 Idempotency belongs in the substrate

Per §3.5: the existing `SendQueue` deduplicates by `IdempotencyKey`. The new design's spec does not mention this. If the substrate doesn't handle it, every chat application has to re-implement it.

**Recommendation:** add `IdempotencyKey string` (optional) to `Command` and to `Hub.Submit`'s signature. The substrate keeps a per-session idempotency table; if a duplicate key arrives within a configurable TTL, the substrate returns the cached result without invoking the handler. The TTL plus the table size are configurable via `WithIdempotency(opts)`.

### 6.7 Session "phases" and projection of in-progress entities

The existing `TurnPersister` writes turns at "interim" and "final" phases. The new design's `TimelineProjection` worked example (the `ChatMessage` from §7) implicitly has the same notion via the `Streaming bool` field on `ChatMessage` — but it isn't promoted to a substrate-level concept.

**Recommendation:** keep this at the application level (the `Streaming` field works), but document the pattern explicitly in the `TimelineProjection` doc section so authors know to use it. The lesson: hydration consumers (UIs) need to be able to render in-progress entities differently from finalised ones, and the easiest way to do that is for the projection to mark them.

### 6.8 The projection-runtime extension story is real

The existing JS timeline runtime (§3.9) exists because someone needed to extend timeline behaviour without recompiling. Whether you ship JS or wasm is bikeshed; the *architectural* question is whether the substrate exposes a hook for runtime-loaded projections.

**Recommendation:** out of scope for v1, but explicitly note in §17 (open questions) that "extensible projection runtime (JS/wasm)" is a deferred item, not an oversight.

## 7. Recommended extraction strategy

If the goal is to land the new framework with a chat-example backend that demonstrates feature parity with the current webchat, here is the suggested ordering. Each step is independently shippable and reversible.

### Step 1 — Extract proto schemas (no behaviour change)

Move `pinocchio/pkg/sem/pb/proto/sem/{base,timeline}/` into `evtstream/proto/api/v1/` (or import from there). Add `protoc-gen-es` to `buf.gen.yaml` so TS types are produced. No code change yet; both old and new framework consume the same generated Go. **Day 1.**

### Step 2 — Build the substrate skeleton

Create `evtstream/` with the types, the `Hub`, the empty slot registrations, and an in-memory `HydrationStore`. No transport yet, no real bus. The Hub can be `Run`-ed and `Submit`-ed but does nothing useful. Cover with unit tests that exercise the slot registration and the dispatch path. **Week 1.**

### Step 3 — Extract the SEM-frame projector into a real `TimelineProjection`

Lift `TimelineProjector` from `pinocchio/pkg/webchat/timeline_projector.go` into `examples/chat/timeline_projection.go`, change the signature to return `[]TimelineEntity`, drop the direct `store.Upsert` call. Cover with the existing `timeline_projector_test.go` cases (renamed / reshaped). **Week 1-2.**

### Step 4 — Build the websocket transport

Lift `connection_pool.go` and the websocket attach logic into `evtstream/transport/ws/`. Add per-connection liveness per §6.5. **Week 2.**

### Step 5 — Build the application-owned bus consumer

Add `evtstream/bus/watermill.go` with the per-session subscriber pattern (lifted from `StreamBackend` / `BuildSubscriber`) and per-session ordinal stamping (lifted from `StreamCoordinator`). Add the recent-frame tail per §6.3. Add throttling per §6.4. **Week 2-3.**

### Step 6 — Build the chat application's command handler

Lift `LLMLoopRunner` machinery into `examples/chat/handlers.go`, change the entry signature to the `CommandHandler` shape, route events through `EventPublisher` instead of through `events.EventSink`. **Week 3.**

### Step 7 — Add idempotency and stable session lifecycle

Lift the `SendQueue` logic into substrate-level `Hub.Submit` plumbing per §6.6. Reduce `Conversation` to `Session{Id, Metadata}`. **Week 3-4.**

### Step 8 — Wire the chat application end-to-end

Mount the substrate behind a chat HTTP handler in `examples/chat/cmd/`. Validate against the existing chat UI. **Week 4.**

### Step 9 — Decommission `pinocchio/pkg/webchat/`

Once parity is reached, the existing package is deleted in favour of `evtstream/` + `examples/chat/`. Pinocchio depends on `evtstream/` directly. **Week 4-5.**

This is roughly a one-month effort for one engineer, assuming the new framework's open questions (§17 in design/02) settle as the work goes.

## 8. Risks and open questions surfaced by this analysis

A few items the reuse exercise made vivid that deserve their own treatment in the design doc:

1. **The runtime profile / fingerprint concept needs a clear home.** The existing system's "swap engines mid-session if the profile changes" pattern is genuinely useful. In the new design it lives in the chat application's `Session.Metadata` + `CommandHandler` — but this doc should be explicit that *the substrate provides no help* for this concern. If multiple applications need it, factor it into a shared library at the application layer, not the substrate.

2. **Per-stream vs. per-session ordinals (§6.1).** The choice should be settled before the ordinal-stamping code is written.

3. **Hydration tail (§6.3).** This is currently a missing piece, not an open question — it should be promoted to a v1 substrate feature.

4. **Throttling (§6.4).** Same — currently absent from the spec, will block adoption the moment LLMs attach.

5. **Liveness contract (§6.5).** The new design should commit to a default rather than leaving it open; the existing `ConnectionPool` is the seed of the answer.

6. **Idempotency (§6.6).** Same — should be promoted from "not in spec" to "in v1 substrate".

7. **The `_v2` suffix in the existing protos** is evidence of past schema migration. If the new framework is a clean break, it can drop the suffix; if backward compatibility matters at all, the migration story needs a paragraph.

8. **Tests are the under-appreciated reuse.** The existing package has substantial test coverage (`*_test.go` files for nearly every subsystem). These should migrate alongside the code lifts and continue to pass — they are the regression net for the extraction.

## 9. What I would change in the new design before implementation starts

To make the above extraction possible without painful late-stage rework, I'd land these edits in `design/02` first:

1. Add `IdempotencyKey string` to `Command` and to `Hub.Submit` (§6.6).
2. Add a tail/catch-up mechanism to hydration (§6.3).
3. Add a throttling/batching hook for `TimelineProjection` (§6.4).
4. Promote liveness from open-question to v1 contract with a default (§6.5).
5. Add an optional `Stream string` to `Event` for sub-session ordinal partitioning (§6.1).
6. Document the "pure passthrough" vs. "derived" `UIProjection` patterns (§6.2).
7. Document the "interim/final" phase pattern in `TimelineProjection` (§6.7).

None of these are large; together they are perhaps a day of doc work. They turn the existing webchat from "a thing we have to rewrite" into "a thing we can extract from".

## 10. Summary

The existing `pinocchio/pkg/webchat/` is a mature implementation of an opinionated, chat-shaped slice of what the new framework wants to be a horizontal substrate. The two share most of the underlying machinery; they disagree on where the substrate ends and the application begins.

Roughly **70%** of the existing code is substrate-shaped and lifts (with surgery) into `evtstream/`. About **20%** is chat-application-shaped and lifts into `examples/chat/`. The remaining **10%** — the runtime-profile fingerprint chain, the HTTP/debug routes, the tool registry, the service-object aggregations — is shed.

The most valuable thing the existing code teaches the new design is **what's missing from the new design**: hydration tail, throttling, idempotency, and a real liveness contract. None of these are mentioned in `design/02` today, and all four will be required from day one. They should be added before code begins.

The most valuable thing the new design teaches the existing code is **where the seams should be drawn**: a `Session` is a small object the application annotates, not a giant struct the substrate populates with twenty-three fields; an event has one `Ordinal` (or `Ordinal + Stream`) that's the universal cursor; projections are *contracts* registered as plain functions, not handler tables managed by globals.

The right next step is to fold the seven recommendations from §9 into the design doc, and then begin the extraction work in §7's order.

## 11. Related

- `design/02-technical-architecture-event-streaming-llm-framework.md` — the new framework's API specification.
- `design/01-architecture-analysis-event-streaming-llm-framework.md` — the clean-room analysis that the new framework derives from.
- `design/03-…` — colleague's parallel reuse analysis (intentionally not consulted; this document is independent).
- `pinocchio/pkg/webchat/` — the existing implementation under analysis.
