---
Title: Intern guide to streaming custom events, progressive widgets, and authoritative commit in evtstream chat apps
Ticket: EVT-STREAM-013
Status: active
Topics:
    - agents
    - architecture
    - backend
    - chat
    - event-streaming
    - framework
    - implementation
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/canonical_runtime_resolver.go
      Note: App-edge runtime resolution and sink wrapper injection
    - Path: ../../../../../../../pinocchio/cmd/web-chat/runtime_composer.go
      Note: Current runtime composer that should start returning sink-decoration behavior
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts
      Note: Focused frontend tests for snapshot and UI-event mapping
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Frontend snapshot and live UI event consumption path
    - Path: ../../../../../../../pinocchio/pkg/evtstream/apps/chat/chat.go
      Note: Canonical chat event translation
    - Path: ../../../../../../../pinocchio/pkg/evtstream/hub.go
      Note: Core command->projection->hydration pipeline and projection ordering
    - Path: ../../../../../../../pinocchio/pkg/inference/runtime/composer.go
      Note: |-
        ComposedRuntime contract that should own sink decoration after cleanup
        Legacy Sink field removed after cleanup; WrapSink is now the only runtime sink contract
    - Path: ../../../../../../../pinocchio/pkg/middlewares/agentmode/middleware.go
      Note: Authoritative final parse and agent mode switch event emission
    - Path: ../../../../../../../pinocchio/pkg/middlewares/agentmode/preview_event.go
      Note: Transient preview event type emitted during progressive structured parsing
    - Path: ../../../../../../../pinocchio/pkg/middlewares/agentmode/structured_extractor.go
      Note: Streaming extractor hook that needs progressive preview support
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: Legacy webchat cleanup completed by moving concrete sink assembly into app-level code
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: |-
        Original webchat example of wrapping sink behavior during runtime assembly
        Legacy router applies runtime and router sink wrapping via a dedicated sink builder
ExternalSources: []
Summary: Detailed intern-facing design and implementation guide for adding progressive streaming custom events and authoritative committed custom events to evtstream-backed chat apps, using agentmode in cmd/web-chat as the exemplar.
LastUpdated: 2026-04-20T20:23:32.396132048-04:00
WhatFor: Orient new contributors on how commands, backend events, projections, hydration, websocket fanout, runtime sink wrappers, agentmode middleware, and frontend entity rendering fit together when adding custom chat events.
WhenToUse: Use when implementing agentmode progressive previews, designing custom backend events for evtstream chat apps, or teaching engineers how to add new projected frontend widgets on top of pinocchio and evtstream.
---





# Intern guide to streaming custom events, progressive widgets, and authoritative commit in evtstream chat apps

## Executive Summary

This ticket exists to document, before implementation, a subtle but important extension to the `evtstream` architecture: allowing a chat application to surface **custom backend events** during inference, render a **progressive live widget** while tokens are still flowing, and then **commit authoritative durable state** only after the final inference result has been validated.

The concrete motivating example is `agentmode` in `cmd/web-chat`. Today the canonical `evtstream` path already supports ordinary chat deltas well: command submission becomes backend events, backend events become UI events plus timeline entity upserts, hydration persists those entities, and the frontend renders them from snapshots and websocket frames. What is missing is the same end-to-end story for a non-message event family such as “the model is preparing an agent-mode switch” and “the application has now authoritatively committed that mode switch.” This is important not only for the product feature itself, but because it demonstrates that the stack is extensible beyond assistant text streaming.

The recommended design in this document is a **two-phase pattern**:

1. **Preview / speculative phase**: a runtime sink wrapper progressively parses structured output while inference is streaming and emits preview events for a live widget.
2. **Commit / authoritative phase**: the existing middleware performs the final parse after inference completes, emits the authoritative mode-switch event, and only that committed event is treated as durable product state.

This pattern keeps the system honest. The preview path proves flexibility and creates a rich UI, but the middleware retains the authority to decide what is true enough to hydrate, replay, and carry across reconnects.

---

## Problem Statement and Scope

The current canonical `cmd/web-chat` migration already proves that `evtstream` can support chat commands, streaming assistant output, hydration, websocket delivery, and a role-aware canonical message model. The remaining gap is not about basic transport anymore. It is about **custom application semantics** layered on top of chat.

In legacy `pkg/webchat`, `agentmode` could emit a dedicated `EventAgentModeSwitch` event and the SEM translator would convert it into a frontend-consumable `agent.mode` frame (`pinocchio/pkg/webchat/sem_translator.go:522-538`). In the canonical `evtstream` path, the chat runtime sink currently only translates four Geppetto event types: partial completions, finals, errors, and interrupts (`pinocchio/pkg/evtstream/apps/chat/chat.go:339-399`). That means custom runtime-side events are not yet flowing through the same architecture.

The user intent for this ticket is broader than just fixing one missing event. The real goal is to produce a reusable playbook for future custom chat features:

- “tool is running” cards,
- planning panels,
- debugger prompts,
- analyst/reviewer mode widgets,
- review status badges,
- or any future custom entity that should appear in the timeline and frontend.

### Scope

This document covers:

- how `evtstream` currently handles commands, backend events, projections, hydration, and websocket delivery,
- how canonical chat currently translates LLM inference deltas,
- how `agentmode` currently injects instructions, parses structured payloads, and emits the final custom event,
- how to add a progressive preview path without making streaming speculation authoritative,
- a recommended file-by-file implementation plan for `cmd/web-chat`, `pkg/evtstream/apps/chat`, `pkg/middlewares/agentmode`, and the frontend.

### Out of Scope

This ticket does **not** implement the feature yet. It is a documentation and design ticket. It also does not attempt to redesign all frontend rendering at once. The recommended implementation should land incrementally and preserve the existing message UX.

---

## Current-State Architecture, with Evidence

### 1. `evtstream` already has the right substrate shape

The `evtstream` substrate already separates the core moving pieces that matter for custom event work:

- a **command** enters through the hub,
- a command handler emits **backend events**,
- a **timeline projection** derives durable entity state,
- a **UI projection** derives live client events,
- the **hydration store** persists entity updates and cursor movement,
- the **UI fanout** publishes live UI events to the transport.

You can see the split explicitly in:

- `pinocchio/pkg/evtstream/hub.go:117-142` — the hub registers exactly one UI projection and one timeline projection,
- `pinocchio/pkg/evtstream/projection.go:9-52` — `UIEvent`, `TimelineEntity`, `UIProjection`, `TimelineProjection`, and `TimelineView`,
- `pinocchio/pkg/evtstream/hydration.go:5-18` — `HydrationStore.Apply`, `Snapshot`, `View`, and `Cursor`.

The most important method to understand is `Hub.projectAndApply(...)` in `pinocchio/pkg/evtstream/hub.go:253-319`. That method:

1. loads the current session and `TimelineView`,
2. runs the UI projection,
3. runs the timeline projection,
4. persists timeline entities via `store.Apply(...)`,
5. publishes UI events through `UIFanout`.

That means `evtstream` already has the exact hook points a custom feature needs. The missing work is not new substrate primitives; it is correct use of the existing ones.

### 2. Canonical chat already demonstrates the full pattern for ordinary message deltas

The chat app under `pkg/evtstream/apps/chat` is currently the best reference implementation for “how a command becomes projected chat behavior.”

Relevant files:

- `pinocchio/pkg/evtstream/apps/chat/chat.go`
- `pinocchio/pkg/evtstream/apps/chat/service.go`
- `pinocchio/cmd/web-chat/app/server.go`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`

#### Write path today

`chat.Service.SubmitPromptRequest(...)` stores a pending `PromptRequest` in the engine and submits `ChatStartInference` through the hub (`pinocchio/pkg/evtstream/apps/chat/service.go:53-72`).

`Engine.handleStartInference(...)` then:

- normalizes the prompt,
- emits `ChatUserMessageAccepted`,
- starts an active inference run,
- invokes `runPrompt(...)` in a goroutine (`pinocchio/pkg/evtstream/apps/chat/chat.go:143-188`).

#### Runtime delta translation today

Inside `runRuntimeInference(...)`, the engine creates a `runtimeEventSink` and passes it to Geppetto’s inference session builder (`pinocchio/pkg/evtstream/apps/chat/chat.go:214-236`).

That sink currently translates Geppetto runtime events as follows (`pinocchio/pkg/evtstream/apps/chat/chat.go:339-399`):

- `EventPartialCompletion` → `ChatTokensDelta`
- `EventFinal` → `ChatInferenceFinished`
- `EventError` → `ChatInferenceStopped`
- `EventInterrupt` → `ChatInferenceStopped`

This is the canonical example of turning low-level inference activity into canonical app-level backend events.

#### Projection split today

The chat UI projection maps backend chat events to websocket UI events (`pinocchio/pkg/evtstream/apps/chat/chat.go:420-443`):

- `ChatUserMessageAccepted` → `ChatMessageAccepted`
- `ChatInferenceStarted` → `ChatMessageStarted`
- `ChatTokensDelta` → `ChatMessageAppended`
- `ChatInferenceFinished` → `ChatMessageFinished`
- `ChatInferenceStopped` → `ChatMessageStopped`

The chat timeline projection maps the same backend events to a durable `ChatMessage` entity (`pinocchio/pkg/evtstream/apps/chat/chat.go:445-495`). Message content, role, status, and streaming state are upserted into the hydration store by `messageId`.

This is the pattern future custom event families should follow.

### 3. The current canonical app experimented with a per-request sink-wrapper seam, but the cleaner target is runtime-owned sink composition

The current canonical chat path introduced a `chat.ResolvedRuntime` wrapper that carries a `SinkWrapper` alongside `ComposedRuntime` (`pinocchio/pkg/evtstream/apps/chat/service.go:13-29`). `runRuntimeInference(...)` creates the canonical chat sink and then applies that wrapper before starting inference (`pinocchio/pkg/evtstream/apps/chat/chat.go:214-236`).

That experiment was useful because it proved that the canonical chat sink can be wrapped at inference time. However, it is probably not the best long-term abstraction. The wrapper does not really represent “resolved runtime data”; it represents app behavior. That makes `ResolvedRuntime` feel thinner and more awkward than the original webchat layering.

The original webchat design used a cleaner ownership model. It defined an app-owned `EventSinkWrapper` hook (`pinocchio/pkg/webchat/types.go:23-24`), exposed router wiring through `WithEventSinkWrapper(...)` (`pinocchio/pkg/webchat/router_options.go:119-127`), and applied the wrapper while assembling the conversation runtime artifacts (`pinocchio/pkg/webchat/router.go:320-344`). That older design followed a good rule: whoever composes the runtime and enables a middleware feature should also own the matching sink behavior.

`infruntime.ComposedRuntime` already has the right conceptual home for runtime-owned sink behavior (`pinocchio/pkg/inference/runtime/composer.go:21-30`). The current type still exposes `Sink events.EventSink`, but for the canonical chat path the more useful contract is a **sink decorator** rather than a side sink. In other words, the target design should move away from `chat.ResolvedRuntime + SinkWrapper` and back toward a runtime-composition-owned wrapper contract.

### 4. `agentmode` already has a complete final parse / commit path

The `agentmode` middleware is not speculative. It already contains the correct final-parse logic.

Relevant files:

- `pinocchio/pkg/middlewares/agentmode/middleware.go`
- `pinocchio/pkg/middlewares/agentmode/protocol.go`
- `pinocchio/pkg/middlewares/agentmode/parser.go`

#### Prompt injection

`agentmode.NewMiddleware(...)` determines the current mode, inserts the mode instructions, and tells the model to emit a structured block if it wants to recommend a switch (`pinocchio/pkg/middlewares/agentmode/middleware.go:81-191`).

The protocol is defined in `pinocchio/pkg/middlewares/agentmode/protocol.go:8-68`:

```text
<pinocchio:agent_mode_switch:v1>
```yaml
mode_switch:
  analysis: |
    ...
  new_mode: reviewer
```
</pinocchio:agent_mode_switch:v1>
```

#### Final parse and authoritative event emission

After downstream inference completes, the middleware examines only newly added assistant blocks and parses the structured payload (`pinocchio/pkg/middlewares/agentmode/middleware.go:194-230`).

The final parser (`pinocchio/pkg/middlewares/agentmode/parser.go:18-109`) already does the heavy lifting:

- strips code fences via `parsehelpers.StripCodeFenceBytes(...)`,
- optionally sanitizes YAML via `github.com/go-go-golems/sanitize/pkg/yaml`,
- unmarshals into `ModeSwitchPayload`,
- extracts `analysis` and `new_mode`.

When a final mode switch is detected, the middleware:

1. updates turn state,
2. optionally records the change in the service,
3. appends a system block announcing the switch,
4. emits `events.NewAgentModeSwitchEvent(...)` (`pinocchio/pkg/middlewares/agentmode/middleware.go:217-229`).

This is already the correct authoritative boundary. The final middleware pass sees the entire assistant output and can safely decide what to persist.

### 5. There is already helper infrastructure for progressive structured parsing

The stack already contains the primitives needed for progressive YAML parsing.

#### Streaming structured capture

Geppetto’s filtering sink is designed to:

1. remove structured side-channel blocks from user-visible completion text, and
2. emit extractor callbacks during streaming.

This is documented explicitly in `github.com/go-go-golems/geppetto/pkg/events/structuredsink/filtering_sink.go:63-77`.

That same comment also contains an important warning: progressive extraction is for **UX/telemetry**, and durable domain state should still be committed only at a clear final boundary.

#### Incremental YAML controller with sanitization

The parse helper package already provides a debounced incremental YAML parser:

- `StripCodeFenceBytes(...)` (`.../parsehelpers/helpers.go:18-37`)
- `DebounceConfig` (`.../parsehelpers/helpers.go:40-65`)
- `YAMLController[T]` (`.../parsehelpers/helpers.go:67-162`)

The `YAMLController` can:

- accumulate bytes as they stream in,
- attempt parses on newline or byte cadence,
- sanitize YAML before each parse,
- perform a final parse at completion.

This is almost tailor-made for progressive agentmode previews.

### 6. The current extractor is intentionally conservative

`pinocchio/pkg/middlewares/agentmode/structured_extractor.go:61-75` shows that the current extractor session is minimal:

- `OnStart(...)` returns nothing,
- `OnRaw(...)` returns nothing,
- `OnCompleted(...)` only validates the final payload and returns nothing.

In other words, the extractor is wired into the streaming path, but it is not yet taking advantage of the incremental YAML helpers. This is exactly why a dedicated ticket is warranted: the seam exists, but the richer progressive behavior has not been designed and documented yet.

### 7. The frontend already has two consumption paths: snapshot and live UI events

The frontend currently handles both hydration and live websocket updates.

In `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`:

- `applySnapshot(...)` clears the client timeline store and rebuilds it from `SessionSnapshotResponse.entities` (`wsManager.ts:93-107`),
- `applyUIEvent(...)` maps live canonical UI events into Redux entity upserts (`wsManager.ts:109-164`).

For ordinary chat messages, `timelineEntityFromSnapshotEntity(...)` gives special treatment to `ChatMessage` entities (`wsManager.ts:66-91`). Unknown kinds are still preserved as generic entities, which is useful for future custom widgets.

The Redux timeline store itself is generic and supports repeated upserts of the same entity ID (`pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts:18-104`). That means the frontend is already capable of supporting additional kinds such as `agentMode`, `agentModePreview`, `toolCall`, or similar, so long as the websocket layer maps them consistently.

---

## Gap Analysis

The current architecture is close, but not complete, for custom events like agentmode.

### What is already solved

- The hub already supports backend events, UI projections, timeline projections, and hydration.
- Canonical chat already demonstrates the message-delta pattern end to end.
- `agentmode` already has a final parser and authoritative event emission path.
- The stack already contains both a current canonical sink-wrapper experiment and an older webchat precedent for composing sink behavior next to runtime assembly.
- The frontend already knows how to hydrate snapshots and apply live UI updates.
- The Geppetto stack already has incremental structured parsing helpers with YAML sanitization.

### What is missing

1. **Progressive preview events are not emitted.**
   The current agentmode extractor ignores `OnRaw(...)` entirely (`pinocchio/pkg/middlewares/agentmode/structured_extractor.go:65-67`).

2. **The canonical runtime sink ignores custom runtime-side events.**
   `runtimeEventSink.PublishEvent(...)` has no case for `EventAgentModeSwitch` or any custom preview event (`pinocchio/pkg/evtstream/apps/chat/chat.go:343-399`).

3. **There is no canonical event contract for non-message widgets.**
   The registered schemas currently only cover chat message commands, events, UI events, and the `ChatMessage` timeline entity (`pinocchio/pkg/evtstream/apps/chat/chat.go:98-118`).

4. **There is no documented pattern separating speculative preview from authoritative commit.**
   The comment in `FilteringSink` says this separation matters (`.../filtering_sink.go:70-77`), but the chat app has not yet expressed that boundary as a first-class design.

5. **The frontend does not yet render a dedicated widget for agentmode preview or commit.**
   Unknown entity kinds will survive in the Redux store, but there is no explicit rendering contract yet.

6. **The current `cmd/web-chat` tree contains both old and new sink-wrapper experiments.**
   `cmd/web-chat/agentmode_sink_wrapper.go` still references legacy `pkg/webchat` wrapper shape, while `cmd/web-chat/agentmode_sink.go` is the new canonical shape. This ongoing refactor state is another reason the ticket should define the intended target before implementation continues.

---

## Proposed Architecture

### Design Goals

The proposed design should satisfy all of the following:

1. Preserve the canonical `evtstream` architecture rather than bypass it.
2. Keep app-specific behavior in `cmd/web-chat` and app-grade chat code, not in substrate core.
3. Allow a live progressive widget while tokens stream.
4. Treat streaming parse results as provisional, never authoritative.
5. Use the final middleware pass to commit durable mode state.
6. Produce a reusable playbook for future custom events in chat applications.

### Composition ownership rule and cleanup target

The design has to satisfy one additional rule beyond preview-versus-commit semantics:

> whoever decides that a runtime includes `agentmode` middleware should also decide the matching sink behavior.

That rule pushes sink ownership toward runtime composition rather than toward a thin app-facing wrapper type. Concretely, the cleanup target for this ticket is:

1. move sink-decoration ownership into `infruntime.ComposedRuntime`,
2. remove `chat.ResolvedRuntime` if it no longer carries any meaningful data beyond composed artifacts,
3. make the chat app consume `*infruntime.ComposedRuntime` directly,
4. have the runtime composer return a wrapper/decorator contract rather than relying on an extra app-layer `SinkWrapper` field.

This is both conceptually cleaner and closer to the original webchat design, where sink composition was attached to runtime assembly rather than stuffed into a second wrapper object.

### The key idea: two classes of custom events

The clean solution is to explicitly distinguish **preview events** from **committed events**.

#### Preview events

Preview events are emitted by the sink wrapper while tokens stream.

Properties:

- derived from partial structured output,
- useful for live UI,
- may be incomplete or wrong,
- must not be treated as the final source of truth.

Examples:

- `ChatAgentModePreviewStarted`
- `ChatAgentModePreviewUpdated`
- `ChatAgentModePreviewCleared`

#### Committed events

Committed events are emitted only after final middleware parsing.

Properties:

- derived from full assistant output,
- durable enough for timeline projection and hydration,
- the correct source of truth for later replays, reloads, and downstream behavior.

Examples:

- `ChatAgentModeCommitted`
- or reuse/bridge the final `EventAgentModeSwitch` into a canonical evtstream event name.

### Recommended event model for the first implementation

For the first version, I recommend this split:

#### Preview path: live-only

- Preview events become canonical backend events inside the chat app.
- `UIProjection` emits preview UI events for the frontend.
- `TimelineProjection` does **not** persist them yet.

Rationale:

- it respects the `FilteringSink` warning that streaming extraction is primarily for UX,
- it avoids stale preview state surviving server restarts,
- it keeps the first implementation smaller and safer.

#### Commit path: durable

- The final middleware event becomes a canonical backend event.
- `TimelineProjection` upserts an `AgentMode` entity.
- `UIProjection` emits a committed UI event.
- The committed event can also clear any preview widget state in the client.

Rationale:

- the final middleware pass sees the full assistant output,
- the committed entity becomes the durable, hydratable truth,
- the frontend still gets a live update when the commit happens.

### Why not persist the preview immediately?

Because preview parsing is speculative.

The incremental YAML controller may briefly expose states like:

- partial `analysis` text,
- a candidate `new_mode` that later disappears,
- syntactically invalid snapshots that become valid after more tokens,
- or a stream that is interrupted before finalization.

Persisting that preview into hydrated state would blur a very important boundary: “this is the best current guess” versus “this is the application’s final conclusion.” For a future advanced version, provisional hydrated preview entities may be reasonable, but they should be added only after the simpler live-only preview model proves useful.

---

## Sequence Diagrams and Data Flow

### A. Ordinary message delta flow today

```text
HTTP POST /api/chat/sessions/:id/messages
    -> chat.Service.SubmitPromptRequest(...)
    -> Hub.Submit(ChatStartInference)
    -> Engine.handleStartInference(...)
    -> runtimeEventSink receives EventPartialCompletion
    -> publish ChatTokensDelta backend event
    -> UIProjection => ChatMessageAppended
    -> TimelineProjection => upsert ChatMessage entity
    -> HydrationStore.Apply(...)
    -> UIFanout.PublishUI(...)
    -> websocket frame
    -> frontend upserts message entity
```

### B. Proposed progressive agentmode flow

```text
assistant emits tagged structured block while streaming
    -> runtime composer returns a sink decorator in ComposedRuntime
    -> canonical chat runtime sink is wrapped at inference start
    -> FilteringSink captures <pinocchio:agent_mode_switch:v1> ...
    -> agentmode extractor OnRaw(...) feeds YAMLController
    -> successful parse snapshot => publish ChatAgentModePreviewUpdated backend event
    -> UIProjection => AgentModePreviewUpdated UI event
    -> websocket frame
    -> frontend renders provisional agent-mode preview widget
```

### C. Proposed authoritative commit flow

```text
inference completes
    -> agentmode middleware examines newly added assistant blocks
    -> ParseModeSwitchPayload(...) on final full payload
    -> emit EventAgentModeSwitch (authoritative)
    -> canonical chat sink/runtime bridge converts to ChatAgentModeCommitted
    -> TimelineProjection => upsert AgentMode entity
    -> UIProjection => AgentModeCommitted + optional preview clear
    -> HydrationStore.Apply(...)
    -> websocket frame + future snapshots include committed mode entity
```

### D. Conceptual separation

```text
[streaming sink preview] ------------------> [live UI only]
          provisional                                |
                                                      v
                                     transient widget in connected browser

[final middleware parse] -----------------> [timeline + hydration + UI]
          authoritative                              |
                                                      v
                                    durable application state and replayable truth
```

---

## API Sketches and Pseudocode

The following sketches are illustrative, not exact final code.

### 0. Runtime composition cleanup (Option A)

The cleanup target is to remove the thin `chat.ResolvedRuntime` wrapper and let the runtime resolver return `*infruntime.ComposedRuntime` directly.

#### Before

```go
// pinocchio/pkg/inference/runtime/composer.go
type ComposedRuntime struct {
    Engine             engine.Engine
    Sink               events.EventSink
    RuntimeFingerprint string
    RuntimeKey         string
    SeedSystemPrompt   string
}

// pinocchio/pkg/evtstream/apps/chat/service.go
type ResolvedRuntime struct {
    ComposedRuntime infruntime.ComposedRuntime
    SinkWrapper     SinkWrapper
}

type PromptRequest struct {
    Prompt         string
    IdempotencyKey string
    Runtime        *ResolvedRuntime
}
```

#### After

```go
// pinocchio/pkg/inference/runtime/composer.go
type EventSinkWrapper func(events.EventSink) (events.EventSink, error)

type ComposedRuntime struct {
    Engine             engine.Engine
    WrapSink           EventSinkWrapper
    RuntimeFingerprint string
    RuntimeKey         string
    SeedSystemPrompt   string
}

// pinocchio/pkg/evtstream/apps/chat/service.go
// ResolvedRuntime deleted

type PromptRequest struct {
    Prompt         string
    IdempotencyKey string
    Runtime        *infruntime.ComposedRuntime
}
```

#### Why this is the right cleanup

- `ComposedRuntime` is the natural home for runtime-owned sink behavior.
- `ResolvedRuntime` no longer carries distinct resolved-policy data in the canonical path.
- The chat app still owns the canonical chat sink, but the composed runtime owns the decision to wrap it.
- `agentmode` middleware and sink behavior stay coupled.

### 1. Preview event payload

```go
// canonical evtstream backend event payload
{
  "messageId": "chat-msg-17",
  "candidateMode": "reviewer",
  "analysisPartial": "The user is asking for critique...",
  "parseState": "candidate",
  "provisional": true
}
```

Suggested parse states:

- `capturing`
- `analysis-only`
- `candidate`
- `invalid`
- `cleared`

### 2. Committed event payload

```go
{
  "messageId": "chat-msg-17",
  "from": "analyst",
  "to": "reviewer",
  "analysis": "The user asked for a critical evaluation...",
  "provisional": false
}
```

### 3. Committed timeline entity

```go
TimelineEntity{
    Kind: "AgentMode",
    Id:   "session",
    Payload: {
        "mode": "reviewer",
        "from": "analyst",
        "analysis": "...",
        "sourceMessageId": "chat-msg-17",
        "committed": true,
    },
}
```

### 4. Preview extractor sketch using existing YAML helpers

```go
func (s *modeSwitchSession) OnRaw(ctx context.Context, chunk []byte) []events.Event {
    snapshot, err := s.ctrl.FeedBytes(chunk)
    if err != nil {
        return nil // tolerate parse churn; wait for later success
    }
    if snapshot == nil {
        return nil
    }

    preview := buildPreview(snapshot)
    if preview == nil {
        return nil
    }

    return []events.Event{
        NewAgentModePreviewEvent(metadata, preview.CandidateMode, preview.AnalysisPartial),
    }
}
```

### 5. Canonical sink translation sketch

```go
func (s *runtimeEventSink) PublishEvent(event gepevents.Event) error {
    switch ev := event.(type) {
    case *gepevents.EventPartialCompletion:
        return publishChatTokensDelta(...)
    case *gepevents.EventAgentModePreview:
        return publishChatAgentModePreviewUpdated(...)
    case *gepevents.EventAgentModeSwitch:
        return publishChatAgentModeCommitted(...)
    default:
        return nil
    }
}
```

### 6. Projection split sketch

```go
func uiProjection(ev evtstream.Event) []evtstream.UIEvent {
    switch ev.Name {
    case "ChatAgentModePreviewUpdated":
        return []evtstream.UIEvent{{Name: "AgentModePreviewUpdated", Payload: ...}}
    case "ChatAgentModeCommitted":
        return []evtstream.UIEvent{
            {Name: "AgentModeCommitted", Payload: ...},
            {Name: "AgentModePreviewCleared", Payload: ...},
        }
    }
}

func timelineProjection(ev evtstream.Event) []evtstream.TimelineEntity {
    switch ev.Name {
    case "ChatAgentModeCommitted":
        return []evtstream.TimelineEntity{{Kind: "AgentMode", Id: "session", Payload: ...}}
    default:
        return nil // preview remains live-only in v1
    }
}
```

---

## File-by-File Implementation Guide

This section is written for a new intern. If you implement the feature, work roughly in this order.

### 1. `pinocchio/pkg/inference/runtime/composer.go`

**Purpose:** move sink-decoration ownership into runtime composition.

Current state:

- `ComposedRuntime` carries `Sink events.EventSink`.
- that shape works well for a runtime-produced sink artifact, but it is not the best fit for “wrap the canonical chat sink before inference starts.”

Recommended changes:

1. Introduce a runtime-level wrapper type, for example:

```go
type EventSinkWrapper func(events.EventSink) (events.EventSink, error)
```

2. Replace `ComposedRuntime.Sink` with `ComposedRuntime.WrapSink`.
3. Update comments so the type makes the ownership rule explicit: the runtime composer owns middleware selection and the matching sink-decoration behavior.

Key caution:

- keep the wrapper contract generic; it should know about `events.EventSink`, not about `cmd/web-chat` or `agentmode` specifically.

### 2. `pinocchio/cmd/web-chat/runtime_composer.go`

**Purpose:** make the runtime composer return the sink decorator when `agentmode` is enabled.

Current state:

- the composer already resolves middleware and builds the engine,
- but it does not currently populate `ComposedRuntime.Sink` or any wrapping contract.

Recommended changes:

1. Add helper logic that inspects `req.ResolvedProfileRuntime` and decides whether a sink decorator is needed.
2. Return that decorator as part of `ComposedRuntime`.
3. Keep the coupling intentional: if the composer decides to include `agentmode` middleware, it should also include the corresponding sink-decoration behavior.

Suggested shape:

```go
return infruntime.ComposedRuntime{
    Engine:             eng,
    WrapSink:           runtimeSinkWrapperFromProfile(req.ResolvedProfileRuntime),
    RuntimeKey:         runtimeKey,
    RuntimeFingerprint: runtimeFingerprint,
    SeedSystemPrompt:   systemPrompt,
}, nil
```

### 3. `pinocchio/cmd/web-chat/agentmode_sink.go`

**Purpose:** build the runtime-owned sink decorator from the resolved middleware configuration.

Current state:

- this file currently returns the newer app-level `chatapp.SinkWrapper` shape.

Recommended changes:

1. Change it to return the runtime-level wrapper type from `pkg/inference/runtime/composer.go`.
2. Keep the middleware-config decoding here, because this file is still the right cmd-web-chat-specific place to interpret `sanitize_yaml` and future agentmode sink config.
3. Make the returned wrapper simply wrap the provided sink with `agentmode.WrapStructuredSink(next, cfg)`.

Example target signature:

```go
func runtimeSinkWrapperFromProfile(runtime *infruntime.ProfileRuntime) infruntime.EventSinkWrapper
```

### 4. `pinocchio/pkg/evtstream/apps/chat/service.go`

**Purpose:** remove the thin wrapper type and pass composed runtime artifacts directly.

Current state:

- `PromptRequest.Runtime` points at `*ResolvedRuntime`.
- `ResolvedRuntime` now mostly wraps `ComposedRuntime` and the experimental sink wrapper.

Recommended changes:

1. Delete `type ResolvedRuntime` entirely.
2. Change `PromptRequest.Runtime` to `*infruntime.ComposedRuntime`.
3. Update comments to explain that the request now carries already-composed runtime artifacts directly.

This is one of the central cleanup steps in Option A.

### 5. `pinocchio/pkg/evtstream/apps/chat/chat.go`

**Purpose:** wrap the canonical chat sink using the runtime-owned decorator, then translate custom runtime events into canonical chat backend events.

Current state:

- `runRuntimeInference(...)` creates `runtimeEventSink`.
- it currently expects an app-layer `ResolvedRuntime` and experimental `SinkWrapper` field.

Recommended changes:

1. Change `runRuntimeInference(...)` to accept `*infruntime.ComposedRuntime`.
2. Create the canonical `runtimeEventSink` as before.
3. If `runtime.WrapSink != nil`, wrap the canonical sink before passing it into the Geppetto engine.
4. Remove all references to `ResolvedRuntime` and the app-level `SinkWrapper`.
5. Keep extending `runtimeEventSink.PublishEvent(...)` to translate preview and committed agentmode events once those event types exist.

Suggested control flow:

```go
baseSink := &runtimeEventSink{...}
eventSink := gepevents.EventSink(baseSink)
if runtime.WrapSink != nil {
    wrapped, err := runtime.WrapSink(baseSink)
    if err != nil { ... }
    eventSink = wrapped
}

sess.Builder = &enginebuilder.Builder{
    Base:       runtime.Engine,
    EventSinks: []gepevents.EventSink{eventSink},
}
```

### 6. `pinocchio/cmd/web-chat/app/server.go` and `contracts.go`

**Purpose:** make the app server and runtime resolver consume `*infruntime.ComposedRuntime` directly.

Current state:

- the app server asks its runtime resolver for a chat-layer resolved runtime wrapper.

Recommended changes:

1. Change the `RuntimeResolver` interface to return `*infruntime.ComposedRuntime`.
2. Update `handleSubmitMessage(...)` so `PromptRequest.Runtime` receives that composed runtime directly.
3. Confirm that the snapshot and websocket contracts remain unchanged; this cleanup is internal runtime plumbing, not an HTTP contract redesign.

### 7. `pinocchio/cmd/web-chat/canonical_runtime_resolver.go`

**Purpose:** stop constructing a chat-owned wrapper object and return the composed runtime directly.

Current state:

- the resolver composes a runtime,
- converts the resolved profile runtime,
- injects `newAgentModeSinkWrapper(profileRuntime)` into `chatapp.ResolvedRuntime`.

Recommended changes:

1. Change the return type to `*infruntime.ComposedRuntime`.
2. Remove `chatapp.ResolvedRuntime` from this file entirely.
3. Return the composed runtime from the runtime composer without adding another wrapper layer.

The conceptual step “resolve policy, then compose runtime” still exists; only the extra transport envelope disappears.

### 8. `pinocchio/cmd/web-chat/agentmode_sink_wrapper.go`

**Purpose:** legacy reference only.

Current state:

- this file still imports `pkg/webchat` and uses the old wrapper signature.

Recommended changes:

- remove it once the canonical implementation has fully moved to runtime-owned sink decoration,
- update or delete any tests that still refer to the old wrapper symbol.

Do not let old webchat-style and new canonical-style wrapper APIs coexist longer than necessary.

### 9. `pinocchio/pkg/middlewares/agentmode/structured_extractor.go`

**Purpose:** teach the streaming extractor to emit preview snapshots while tokens are still arriving.

Current state:

- `OnRaw(...)` returns nothing.
- `OnCompleted(...)` only validates the final payload.

Recommended changes:

1. Add a `parsehelpers.YAMLController[ModeSwitchPayload]` field to `modeSwitchSession`.
2. Initialize it with a conservative `DebounceConfig`:
   - `SnapshotOnNewline = true`
   - optional `SnapshotEveryBytes`
   - `SanitizeYAML = true`
3. In `OnRaw(...)`, feed bytes into the controller.
4. When a parse succeeds, emit a **preview event**, not a final switch event.
5. Leave `OnCompleted(...)` as validation-oriented; it may emit a final preview snapshot if useful, but the authoritative event should still come from middleware.

Key caution:

- Do not publish final committed state here.
- Do not mutate mode storage here.
- Treat parsing errors during streaming as normal churn, not fatal failures.

### 10. `pinocchio/pkg/middlewares/agentmode/parser.go` and `protocol.go`

**Purpose:** keep the payload contract centralized and reusable.

Recommended changes:

- If preview payload shaping needs helper functions, add them here rather than duplicating YAML field knowledge elsewhere.
- Keep `ModeSwitchPayload` as the single typed YAML shape.
- Add small helpers like `PreviewFromPayload(...)` only if they genuinely reduce duplication.

Do **not** split parsing rules across the extractor, middleware, and frontend.

### 11. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`

**Purpose:** map snapshot entities and live UI events into Redux timeline state.

Recommended changes:

1. Add explicit handling for committed `AgentMode` entities in `timelineEntityFromSnapshotEntity(...)`.
2. Add explicit handling for preview UI events in `applyUIEvent(...)`.
3. Decide whether preview state should live in the same timeline slice or in a specialized slice.

Recommended initial frontend approach:

- keep committed mode state as a generic timeline entity,
- keep preview state in timeline as a specialized entity kind only on the client side,
- or use a dedicated small Redux sub-state if that proves cleaner.

The important part is that the widget should be rendered from canonical state, not by reparsing assistant text in the browser.

### 12. `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`

**Purpose:** entity upsert behavior already exists; reuse it.

Recommended changes:

- likely none for basic support,
- but confirm whether preview clear behavior should use explicit deletion/tombstoning semantics or a separate reducer.

---

## Phased Implementation Plan

### Phase 0 — Contract cleanup before feature work

1. Replace `ComposedRuntime.Sink` with a runtime-owned sink decorator contract such as `WrapSink`.
2. Move agentmode sink selection into `cmd/web-chat/runtime_composer.go` and `cmd/web-chat/agentmode_sink.go`.
3. Remove `chat.ResolvedRuntime`.
4. Change `PromptRequest.Runtime`, `RuntimeResolver`, and `canonicalRuntimeResolver` to use `*infruntime.ComposedRuntime` directly.
5. Delete the obsolete `cmd/web-chat/agentmode_sink_wrapper.go` path once callers and tests are updated.

This phase should leave the runtime composition story internally consistent before adding new preview behavior.

### Phase 1 — Progressive parser in `agentmode`

1. Add `YAMLController`-based `OnRaw(...)` parsing.
2. Emit preview events from the extractor.
3. Add unit tests for partial analysis, partial `new_mode`, invalid YAML churn, and final parse success.

### Phase 2 — Canonical event translation

1. Extend `runtimeEventSink.PublishEvent(...)` to translate preview and commit events.
2. Register schemas for the new canonical event/UI/entity names.
3. Add tests proving preview events are no longer silently dropped.

### Phase 3 — Projection policy

1. Add UI projection for preview and commit events.
2. Add committed timeline projection for `AgentMode` entities.
3. Keep preview live-only in v1.
4. Add snapshot tests proving committed mode survives reload.

### Phase 4 — Frontend widget

1. Add preview widget rendering.
2. Add committed mode rendering.
3. Clear preview state on commit or stop.
4. Add browser validation with real inference using `gpt-5-nano-low`.

### Phase 5 — Playbook extraction

1. Convert the final implementation lessons into a shorter operational playbook.
2. Point future contributors to this ticket for the deep explanation.
3. Use the resulting pattern for the next custom widget/event family.

---

## Testing and Validation Strategy

### Unit tests

#### Runtime composition cleanup tests

Add or update tests for:

- `ComposedRuntime` carrying a sink decorator contract instead of a side sink,
- `cmd/web-chat/runtime_composer.go` returning a decorator when `agentmode` middleware is present,
- `canonicalRuntimeResolver` returning `*infruntime.ComposedRuntime` directly,
- `pkg/evtstream/apps/chat` consuming `*infruntime.ComposedRuntime` without `ResolvedRuntime`,
- failure behavior when `WrapSink(...)` returns an error.

#### `agentmode` parser/extractor tests

Add tests for:

- `YAMLController` preview parse snapshots at newline cadence,
- malformed intermediate YAML that later becomes valid,
- payloads with `analysis` only,
- payloads with both `analysis` and `new_mode`,
- sanitize on/off behavior.

#### Chat runtime sink tests

Add tests proving:

- preview Geppetto events become canonical backend preview events,
- committed agentmode events become canonical committed backend events,
- unrecognized events are still ignored safely.

### Projection tests

Add tests for:

- committed event → `AgentMode` entity upsert,
- preview event → UI event only,
- committed event clearing preview UI state if that behavior is chosen.

### Frontend tests

Add tests for:

- snapshot hydration of `AgentMode` entities,
- live preview widget updates from preview UI events,
- widget replacement/clear on committed event.

### Browser validation

Run a real end-to-end flow:

1. start `cmd/web-chat`,
2. select a profile with agentmode enabled,
3. submit a prompt likely to cause a mode switch,
4. verify the preview widget appears while streaming,
5. verify the committed mode widget/state appears after completion,
6. refresh the page and confirm the committed state hydrates,
7. confirm the preview state does not linger incorrectly after completion or interruption.

### Suggested commands

```bash
cd pinocchio && go test ./pkg/middlewares/agentmode ./pkg/evtstream/apps/chat ./cmd/web-chat/app ./cmd/web-chat/... -count=1
cd pinocchio/cmd/web-chat/web && npm run check && npm run build
```

---

## Risks, Alternatives, and Open Questions

### Risk 1: preview semantics become too durable

If preview data leaks into hydrated state too early, reconnects may show stale speculative UI after interrupted runs.

Mitigation:

- keep preview live-only in v1,
- or store preview in a clearly provisional entity with explicit cleanup semantics if reconnect support becomes necessary later.

### Risk 2: progressive YAML parsing becomes noisy

Parsing every chunk can create a lot of failed parse attempts.

Mitigation:

- use `YAMLController` debounce settings,
- prefer newline-triggered parsing,
- ignore intermediate parse errors until a useful snapshot appears.

### Risk 3: cleanup leaves the runtime contract in a half-updated state

The most likely cleanup failure is leaving a mixture of old and new contracts in place: `ResolvedRuntime` still referenced in some call sites, `ComposedRuntime.Sink` partially replaced, or both the legacy and new wrapper files present simultaneously.

Mitigation:

- land the contract cleanup as one focused slice,
- update resolver, prompt request, chat engine, and runtime composer together,
- remove the old legacy variant promptly.

### Alternative A: final-only agentmode support

This would be simpler, but it would miss the main UX and architecture lesson this ticket is meant to demonstrate.

### Alternative B: sink directly commits durable state

Rejected. It violates the `FilteringSink` guidance (`.../filtering_sink.go:70-77`) and weakens the final authoritative boundary.

### Alternative C: preview entities are fully hydrated in v1

Possible later, but likely too much complexity for the first implementation.

### Open Questions

1. Should preview events become generic “entity upsert” UI events, or feature-specific names?
2. Should the committed `AgentMode` entity use a stable session-level ID such as `session`, or a more explicit ID like `agent-mode:<sessionId>`?
3. Should the frontend model preview state in the same timeline slice or a dedicated reducer?
4. Do we want a generic “custom event playbook” doc after implementation, or should this ticket remain the deep source of truth?

---

## Intern Checklist

If you are new to the codebase, read in this order:

1. `pinocchio/pkg/inference/runtime/composer.go:11-34`
2. `pinocchio/pkg/evtstream/projection.go:9-52`
3. `pinocchio/pkg/evtstream/hydration.go:5-18`
4. `pinocchio/pkg/evtstream/hub.go:117-142,253-319`
5. `pinocchio/pkg/evtstream/apps/chat/service.go:13-29,53-72` (read this as the cleanup target; the type described here is expected to shrink or disappear)
6. `pinocchio/pkg/evtstream/apps/chat/chat.go:98-140,214-261,339-495`
7. `pinocchio/pkg/webchat/types.go:23-24`
8. `pinocchio/pkg/webchat/router_options.go:119-127`
9. `pinocchio/pkg/webchat/router.go:320-344`
10. `pinocchio/pkg/middlewares/agentmode/protocol.go:8-68`
11. `pinocchio/pkg/middlewares/agentmode/parser.go:18-109`
12. `pinocchio/pkg/middlewares/agentmode/middleware.go:81-230`
13. `pinocchio/pkg/middlewares/agentmode/structured_extractor.go:20-82`
14. `pinocchio/cmd/web-chat/runtime_composer.go:33-93`
15. `pinocchio/cmd/web-chat/canonical_runtime_resolver.go:28-69`
16. `pinocchio/cmd/web-chat/agentmode_sink.go:16-43`
17. `pinocchio/cmd/web-chat/app/server.go:87-120,222-258`
18. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:66-164`
19. `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts:18-104`
20. `github.com/go-go-golems/geppetto/pkg/events/structuredsink/filtering_sink.go:63-77`
21. `github.com/go-go-golems/geppetto/pkg/events/structuredsink/parsehelpers/helpers.go:40-162`

After that, come back to this ticket and reread the phased plan.

---

## References

### Primary code references

- `pinocchio/pkg/inference/runtime/composer.go:11-34`
- `pinocchio/pkg/evtstream/hub.go:117-142,253-319`
- `pinocchio/pkg/evtstream/projection.go:9-52`
- `pinocchio/pkg/evtstream/hydration.go:5-18`
- `pinocchio/pkg/evtstream/schema.go:11-128`
- `pinocchio/pkg/evtstream/apps/chat/service.go:13-29,53-72`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:98-140,214-261,339-495`
- `pinocchio/pkg/webchat/types.go:23-24`
- `pinocchio/pkg/webchat/router_options.go:119-127`
- `pinocchio/pkg/webchat/router.go:320-344`
- `pinocchio/pkg/middlewares/agentmode/protocol.go:8-68`
- `pinocchio/pkg/middlewares/agentmode/parser.go:18-109`
- `pinocchio/pkg/middlewares/agentmode/middleware.go:81-230`
- `pinocchio/pkg/middlewares/agentmode/structured_extractor.go:20-82`
- `pinocchio/cmd/web-chat/runtime_composer.go:33-93`
- `pinocchio/cmd/web-chat/canonical_runtime_resolver.go:28-69`
- `pinocchio/cmd/web-chat/agentmode_sink.go:16-43`
- `pinocchio/cmd/web-chat/app/contracts.go:27-39`
- `pinocchio/cmd/web-chat/app/server.go:87-120,222-258`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:66-164`
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts:18-104`
- `pinocchio/pkg/webchat/sem_translator.go:522-538`

### External helper references

- `/home/manuel/go/pkg/mod/github.com/go-go-golems/geppetto@v0.11.14/pkg/events/structuredsink/filtering_sink.go:63-77`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/geppetto@v0.11.14/pkg/events/structuredsink/parsehelpers/helpers.go:18-162`
- `/home/manuel/go/pkg/mod/github.com/go-go-golems/geppetto@v0.11.14/pkg/events/chat-events.go:872-890`

### Related ticket docs

- `le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md`
- `le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/reference/01-investigation-diary.md`

