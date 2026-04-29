---
Title: Intern guide to adding reasoning/thinking streaming to the pinocchio sessionstream web-chat example
Ticket: EVT-STREAM-014
Status: active
Topics:
    - chat
    - architecture
    - backend
    - event-streaming
    - llm
    - implementation
    - onboarding
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../geppetto/pkg/events/chat-events.go
      Note: Upstream Geppetto event taxonomy including partial-thinking and info events
    - Path: ../../../../../../../geppetto/pkg/steps/ai/openai_responses/engine.go
      Note: Provider stream normalization and reasoning-summary publication path
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: Live app wiring currently registers only the agentmode feature
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/cards.tsx
      Note: Existing MessageCard already supports role=thinking for a minimal UI slice
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Frontend snapshot and UI-event mapping that currently lacks reasoning cases
    - Path: ../../../../../../../pinocchio/pkg/chatapp/chat.go
      Note: Current canonical runtime sink
    - Path: ../../../../../../../pinocchio/pkg/chatapp/features.go
      Note: Feature extension seam used for app-owned runtime event translation and projection
ExternalSources: []
Summary: Investigates why the live canonical sessionstream-based cmd/web-chat currently streams assistant text but not model thinking, maps the relevant Geppetto and Pinocchio seams, and proposes an app-owned feature-based implementation plan for durable reasoning/thinking UI support.
LastUpdated: 2026-04-21T19:25:00-04:00
WhatFor: Give a new intern enough architectural context to safely add reasoning/thinking streaming to the live canonical pinocchio web-chat without reviving deleted legacy thinkingmode code or polluting sessionstream core.
WhenToUse: Read before implementing visible model thinking/reasoning in cmd/web-chat, especially when deciding where Geppetto reasoning events should be translated into sessionstream timeline entities and websocket UI events.
---


# Intern guide to adding reasoning/thinking streaming to the pinocchio sessionstream web-chat example

## Executive Summary

The user’s intuition is directionally correct for the **live canonical** `cmd/web-chat`: the current sessionstream-backed app streams normal assistant text deltas, but it does **not** currently surface model thinking/reasoning in the visible chat timeline. That gap is not because Geppetto lacks thinking events. In fact, the upstream Geppetto engines already emit reasoning-related events such as `partial-thinking`, `reasoning-text-delta`, `thinking-started`, `thinking-ended`, and final reasoning metadata including `reasoning_tokens` (`geppetto/pkg/events/chat-events.go:13-20,350-365`; `geppetto/pkg/steps/ai/openai/engine_openai.go:273-280,324-352`; `geppetto/pkg/steps/ai/openai_responses/engine.go:462-490,798-806`).

The real gap is in the canonical Pinocchio app layer. The current chat app schema, runtime sink, and frontend websocket mapping only have first-class handling for normal assistant/user chat messages plus the app-owned `agentmode` feature. The live app registers only the `agentmode` feature (`pinocchio/cmd/web-chat/main.go:326-331`), the canonical runtime sink directly converts only normal text completion events into `ChatTokensDelta` / `ChatInferenceFinished` (`pinocchio/pkg/chatapp/chat.go:352-412`), and the frontend websocket manager only understands `ChatMessage*` plus `ChatAgentMode*` UI events (`pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:138-226`).

The recommended implementation is therefore **not** to change `sessionstream` core and **not** to resurrect the deleted legacy `thinkingmode` island. Instead, we should add a new **app-owned chat feature** under `cmd/web-chat`, following the same pattern already used successfully for `agentmode` (`pinocchio/pkg/chatapp/features.go:10-21,46-103`; `pinocchio/cmd/web-chat/agentmode_chat_feature.go:29-124`). That feature should translate Geppetto reasoning/thinking events into canonical backend events, project them into durable timeline entities, fan them out as websocket UI events, and let the existing React `MessageCard` render them using `role: "thinking"`, which it already supports visually (`pinocchio/cmd/web-chat/web/src/webchat/cards.tsx:14-33`).

A second important clarification is terminology. “Reasoning tokens” can mean two different things in conversation, but they are not the same implementation problem:

1. **Visible streamed thinking text** — what the user wants to see in the UI while the model is reasoning.
2. **Final reasoning token counts** — numerical usage metadata such as `reasoning_tokens` in final provider metadata.

The first slice should ship visible thinking text. Token-count display can follow as a smaller, optional enhancement once the visible reasoning stream is wired end-to-end.

## Problem Statement and Scope

### The practical problem

The example `cmd/web-chat` now uses the canonical sessionstream architecture, but the user experience still only shows the assistant’s normal answer text. For reasoning-capable models and profiles, Geppetto can emit reasoning/thinking progress, yet the live chat UI does not render it.

That means the system currently hides a class of useful signals that already exist upstream:

- “the model has entered a thinking phase,”
- “the model is streaming reasoning summary or reasoning text,”
- “the model has ended its thinking phase,”
- “the final response included reasoning usage metadata.”

### Why this matters

This matters for three reasons.

- **Operator trust**: when a model pauses before speaking, users want to know whether it is actually reasoning or just stuck.
- **Debugging**: reasoning-capable model profiles are harder to validate if the app discards all thinking-related signals.
- **Architecture integrity**: once the canonical app supports durable custom event projection for `agentmode`, reasoning/thinking is the next obvious proof that the app-owned feature seam is real and reusable.

### In scope

This ticket is about the **live canonical** `pinocchio/cmd/web-chat` application path.

In scope:

- backend translation of Geppetto thinking events into canonical app events,
- timeline/hydration support so reasoning survives reconnects,
- websocket UI fanout for live reasoning updates,
- frontend mapping so reasoning appears in the normal chat timeline,
- tests and browser validation,
- detailed intern-facing documentation.

### Out of scope

Explicitly out of scope for the first slice:

- changing `sessionstream` substrate packages,
- reviving deleted `cmd/web-chat/thinkingmode/*` legacy code,
- restoring old `pkg/webchat` SEM wiring as the primary live path,
- designing a complex new visual widget if the existing message card is enough,
- exposing a full end-user reasoning configuration panel in the browser.

That last item can become a follow-up if needed. The first ticket should make already-emitted thinking visible.

## Important Terms: what exactly are we trying to surface?

Before changing code, an intern needs the vocabulary straight.

### 1. Assistant text delta

This is the normal answer text the user already sees today.

Examples in current code:

- Geppetto emits `EventPartialCompletion` and `EventFinal`.
- Pinocchio maps them to `ChatTokensDelta` and `ChatInferenceFinished` (`pinocchio/pkg/chatapp/chat.go:356-382`).
- The frontend maps them to `ChatMessageAppended` and `ChatMessageFinished` (`pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:164-183`).

### 2. Thinking / reasoning text

This is model-internal or model-adjacent reasoning text that some providers expose. Depending on the provider, Geppetto may surface it as:

- `EventThinkingPartial` (`partial-thinking`),
- `EventReasoningTextDelta`,
- `EventReasoningTextDone`,
- `EventInfo("thinking-started")`,
- `EventInfo("thinking-ended")`,
- `EventInfo("reasoning-summary")`.

### 3. Reasoning summary

Especially for OpenAI Responses-style models, the stream may expose a summarized public reasoning trace rather than raw chain-of-thought text. Geppetto already mirrors that summary into `partial-thinking` so existing UIs can render it live (`geppetto/pkg/steps/ai/openai_responses/engine.go:462-490`).

### 4. Reasoning tokens

This is **usage metadata**, not stream text. Geppetto stores it in event metadata extras such as `metadata.Extra["reasoning_tokens"]` (`geppetto/pkg/steps/ai/openai/engine_openai.go:324-328`; `geppetto/pkg/steps/ai/openai_responses/engine.go:798-800`).

This is useful, but it is a separate display problem from streaming visible thinking text.

## Current-State Architecture: where the gap actually is

## End-to-end flow today

The live app currently looks like this:

```text
React ChatWidget
  -> POST /api/chat/sessions/:sessionId/messages
  -> cmd/web-chat/app.Server
  -> pkg/chatapp.Service
  -> sessionstream Hub command
  -> pkg/chatapp.Engine
  -> runtimeEventSink
  -> backend Chat* events
  -> sessionstream UI/timeline projection
  -> websocket ui-event + hydrated snapshot
  -> wsManager.ts
  -> timeline Redux slice
  -> ChatWidget timeline
```

That is the correct modern architecture. The problem is simply that the canonical app’s Chat* event vocabulary currently covers assistant/user messages and stop/start state, but not reasoning/thinking.

### Evidence 1: the base chat schema only knows about normal chat messages

`pinocchio/pkg/chatapp/chat.go:19-35` defines the canonical chat constants:

- `ChatUserMessageAccepted`
- `ChatInferenceStarted`
- `ChatTokensDelta`
- `ChatInferenceFinished`
- `ChatInferenceStopped`
- `ChatMessageAccepted`
- `ChatMessageStarted`
- `ChatMessageAppended`
- `ChatMessageFinished`
- `ChatMessageStopped`
- `ChatMessage` timeline entity

Then `RegisterSchemas(...)` registers only those plus any additional feature schemas (`pinocchio/pkg/chatapp/chat.go:96-124`). There is no built-in reasoning event family.

### Evidence 2: the canonical runtime sink directly handles only normal completion/error/interrupt events

The runtime sink is the component that receives Geppetto runtime events during real inference. In `pinocchio/pkg/chatapp/chat.go:352-412`, `runtimeEventSink.PublishEvent(...)` has hard-coded handling for:

- `*gepevents.EventPartialCompletion`
- `*gepevents.EventFinal`
- `*gepevents.EventError`
- `*gepevents.EventInterrupt`

Everything else falls through to `handleFeatureRuntimeEvent(...)` (`pinocchio/pkg/chatapp/chat.go:410-411`). That is an important architectural clue: **thinking support belongs in an app-owned feature**, not in a one-off fork of the base sink.

### Evidence 3: the app feature seam already exists for exactly this class of extension

`pinocchio/pkg/chatapp/features.go:10-21` defines a feature interface with four responsibilities:

- schema registration,
- runtime-event translation,
- UI projection,
- timeline projection.

The flow is already wired:

- `handleFeatureRuntimeEvent(...)` dispatches unknown runtime events to active features (`pinocchio/pkg/chatapp/features.go:46-63`),
- `uiProjection(...)` lets a feature generate UI events (`pinocchio/pkg/chatapp/features.go:65-83`),
- `timelineProjection(...)` lets a feature generate durable entities (`pinocchio/pkg/chatapp/features.go:85-103`).

This is the seam reasoning support should use.

### Evidence 4: the live app currently registers only one feature: agentmode

In `pinocchio/cmd/web-chat/main.go:326-331`, the canonical app server is created with:

```go
appserver.WithChatFeatureSets(newAgentModeChatFeature())
```

That means only `agentmode` participates in the app-owned extension seam today.

### Evidence 5: the frontend websocket mapping only knows ChatMessage and AgentMode

`pinocchio/cmd/web-chat/web/src/ws/wsManager.ts` shows the current browser contract.

Snapshot mapping:

- `ChatMessage` -> local `message` entity (`wsManager.ts:80-96`)
- `AgentMode` -> local `agent_mode` entity (`wsManager.ts:98-105`)

Live UI event mapping:

- `ChatMessageAccepted`
- `ChatMessageStarted`
- `ChatMessageAppended`
- `ChatMessageFinished`
- `ChatMessageStopped`
- `ChatAgentModePreviewUpdated`
- `ChatAgentModeCommitted`
- `ChatAgentModePreviewCleared`

See `wsManager.ts:138-226`.

There is no reasoning-specific case.

### Evidence 6: the renderer layer is already flexible enough

The renderer registry supports multiple entity kinds (`pinocchio/cmd/web-chat/web/src/webchat/rendererRegistry.ts:14-21`), but the existing `MessageCard` already supports a `thinking` role without needing a brand-new component (`pinocchio/cmd/web-chat/web/src/webchat/cards.tsx:14-19`).

That matters because it means the first reasoning slice can be small:

- keep using the existing message card,
- add `role: "thinking"`,
- project reasoning as a second chat-like message entity.

### Evidence 7: there is dormant thinking support in the old SEM path, but it is not the live app path

The repo still contains thinking mappings in `pinocchio/cmd/web-chat/web/src/sem/registry.ts:159-196`:

- `llm.thinking.start`
- `llm.thinking.delta`
- `llm.thinking.final`
- `llm.thinking.summary`

Those handlers upsert timeline entities with `role: "thinking"`.

However, this is not the live canonical path anymore.

- The live `App.tsx` renders `ChatWidget` directly (`pinocchio/cmd/web-chat/web/src/App.tsx:1-20`).
- The search results show `sem/registry` is imported in `ChatWidget.stories.tsx`, not in the actual runtime app.

So the repo already contains a **good donor idea**—“thinking should look like a message with role `thinking`”—but the live app no longer consumes that SEM pipeline.

## What Geppetto already gives us upstream

This ticket is not blocked on upstream inference engines. Geppetto is already doing the hard provider normalization work.

## Event definitions

`geppetto/pkg/events/chat-events.go:13-20` defines:

- `partial` for normal assistant text,
- `partial-thinking` for reasoning/summary text.

`geppetto/pkg/events/chat-events.go:350-365` defines `EventThinkingPartial`, which mirrors the shape of normal text partials:

- `Delta`
- `Completion`

That shape is ideal for websocket UI updates because the frontend can remain idempotent by preferring cumulative content.

## OpenAI chat-completions engine

In `geppetto/pkg/steps/ai/openai/engine_openai.go:273-280`, when `response.DeltaReasoning` is present, Geppetto does all of these:

1. emits `thinking-started` the first time,
2. normalizes the reasoning delta into a cumulative buffer,
3. emits `reasoning-text-delta`,
4. emits `partial-thinking`.

Then, at the end (`engine_openai.go:324-352`), it stores:

- `thinking_text`,
- `saying_text`,
- `reasoning_tokens`,

and emits `reasoning-text-done` plus `thinking-ended`.

## OpenAI Responses engine

The Responses engine is even richer.

In `geppetto/pkg/steps/ai/openai_responses/engine.go:340-347`, when a reasoning item starts, Geppetto emits `thinking-started`.

Then:

- `response.reasoning_summary_text.delta` -> `partial-thinking` (`engine.go:462-475`)
- `response.reasoning_text.delta` -> `reasoning-text-delta` and also `partial-thinking` (`engine.go:479-490`)
- `response.output_item.done` for a reasoning item -> `thinking-ended` (`engine.go:507-513`)

At the end of the response, Geppetto stores:

- `reasoning_tokens`,
- `thinking_text`,
- `saying_text`,
- `reasoning_summary_text`,

and emits `reasoning-summary` info with the full summary (`engine.go:798-806`).

### Architectural conclusion from the Geppetto evidence

For the canonical web-chat, the best compatibility event to consume is:

- `*events.EventThinkingPartial`

because Geppetto already uses it as the cross-provider “something visible and reasoning-like is streaming” event.

Raw `EventReasoningTextDelta` still exists, but handling it in the app would likely duplicate what `EventThinkingPartial` already mirrors, especially on the Responses path.

That exact duplication concern already existed in older webchat code: `pkg/webchat/sem_translator.go:245-251` explicitly treats `EventReasoningTextDelta` and `EventReasoningTextDone` as redundant drop candidates, while `partial-thinking` remains the UI-facing donor signal. The old SEM translator test also verifies `reasoning-summary` -> `llm.thinking.summary` (`pinocchio/pkg/webchat/sem_translator_test.go:147-166`).

The new canonical app should copy that **idea**, not the old stack.

## Gap Analysis

The end-to-end mismatch is now very clear.

### Upstream can already emit visible thinking

Geppetto:

- can normalize provider reasoning streams,
- can emit streaming partial-thinking updates,
- can emit thinking start/end markers,
- can attach final `reasoning_tokens` and summary text.

### Canonical backend currently does not translate those signals

The canonical `pkg/chatapp` path only translates normal assistant text and a few terminal events. Thinking events are neither part of the base schema nor implemented as a feature.

### Canonical frontend currently cannot consume them even if they existed

The browser websocket manager currently has no case labels for reasoning event names. So even if the backend started emitting them tomorrow, the current frontend would drop them.

### There is already a dormant donor idea in the repo

The old SEM registry already shows a reasonable user model: thinking can be rendered as another message-like entity with role `thinking`. That is a useful reuse hint, but the live app needs the canonical sessionstream version of that idea.

## Design Goals

The implementation should obey these goals.

1. **Stay app-owned.** Do not teach `sessionstream` what reasoning is.
2. **Stay canonical.** Do not revive deleted legacy `thinkingmode` machinery.
3. **Durable first.** Reasoning should appear in snapshots and survive reconnects, not only as ephemeral UI events.
4. **Provider-tolerant.** The app should work whether the provider emits raw reasoning text, reasoning summary text, or no thinking at all.
5. **Minimal first slice.** Reuse the existing message renderer when possible.
6. **Extensible second slice.** Preserve room to later show reasoning token counts, badges, or richer widgets.

## Proposed Solution

## High-level idea

Add a new app-owned feature under `cmd/web-chat` that translates Geppetto thinking events into canonical sessionstream chat events and projects them into the existing chat timeline as an additional `ChatMessage` entity whose `role` is `thinking`.

That means the frontend can reuse the current message renderer and simply display a second card associated with the assistant reply.

## Why use a feature instead of changing base chatapp?

Because reasoning/thinking is not a universally required property of every app that might reuse `pkg/chatapp`. The current architecture already separates:

- base chat behavior in `pkg/chatapp`, and
- product- or runtime-specific behavior in app-owned features.

Reasoning/thinking display belongs with the second category.

## Why use `ChatMessage` instead of inventing `ChatReasoningCard` immediately?

Because the repo already contains a strong hint that “thinking” is message-like, and the current `MessageCard` already supports `role: "thinking"` (`pinocchio/cmd/web-chat/web/src/webchat/cards.tsx:14-19`).

Reusing `ChatMessage` for the first slice buys us:

- fewer frontend files to change,
- less snapshot/UI event branching,
- easier hydration/reconnect correctness,
- lower review surface.

A dedicated `ChatReasoning` entity kind remains a valid future refinement if the UI needs richer styling or badges later.

## Canonical event model for the first slice

I recommend these app-owned backend/UI events:

```text
Backend events
- ChatReasoningStarted
- ChatReasoningDelta
- ChatReasoningFinished

UI events
- ChatReasoningStarted
- ChatReasoningAppended
- ChatReasoningFinished
```

And I recommend projecting them into the existing timeline entity kind:

```text
Timeline entity kind
- ChatMessage
```

with payload fields such as:

```json
{
  "messageId": "chat-msg-17:thinking",
  "parentMessageId": "chat-msg-17",
  "role": "thinking",
  "content": "I should first compute...",
  "status": "streaming",
  "streaming": true,
  "source": "summary",
  "reasoningTokens": 64
}
```

## Message identity strategy

The thinking entity should not reuse the assistant message ID. It needs its own stable identity so it can exist alongside the final assistant answer.

Recommended helper:

```go
func reasoningMessageID(parentMessageID string) string {
    return parentMessageID + ":thinking"
}
```

Why this is good:

- deterministic,
- easy to derive on backend and frontend,
- easy to relate to the assistant answer,
- no extra persistent ID generator required.

## Runtime-event translation rules

The feature should handle at least these Geppetto events:

### `*events.EventThinkingPartial`

This is the main live-streaming event.

Publish:

- `ChatReasoningDelta`

Payload sketch:

```go
{
  "messageId": reasoningMessageID(runtime.MessageID),
  "parentMessageId": runtime.MessageID,
  "role": "thinking",
  "chunk": ev.Delta,
  "content": ev.Completion,
  "status": "streaming",
  "streaming": true,
}
```

### `*events.EventInfo` with `Message == "thinking-started"`

Optional but useful for a placeholder/spinner.

Publish:

- `ChatReasoningStarted`

### `*events.EventInfo` with `Message == "thinking-ended"`

Mark the reasoning entity as no longer streaming.

Publish:

- `ChatReasoningFinished`

This event can carry:

- `streaming: false`
- `status: "finished"`
- optionally `reasoningTokens` if found in `event.Metadata().Extra`

### `*events.EventInfo` with `Message == "reasoning-summary"`

If Geppetto provides a final summarized text payload, prefer it as the finishing payload.

Publish:

- `ChatReasoningFinished`

with:

- `content: ev.Data["text"]`
- `source: "summary"`
- `reasoningTokens` when present in metadata extras

### Events to intentionally ignore in the first slice

- `*events.EventReasoningTextDelta`
- `*events.EventReasoningTextDone`

Reason: on the providers we care about, Geppetto already mirrors them into `EventThinkingPartial`, and consuming both paths risks duplicate UI.

## Projection rules

## Timeline projection

Project all reasoning backend events into `TimelineEntityChatMessage`.

Rules:

- `role = "thinking"`
- `messageId = reasoningMessageID(parentMessageId)`
- `parentMessageId = assistantMessageId`
- `content` uses cumulative text when available
- `streaming` flips to `false` on finish
- `status` transitions `streaming -> finished`

Because snapshot mapping already understands `ChatMessage` payloads generically (`wsManager.ts:80-96`), this gives reconnect correctness almost for free.

## UI projection

Emit dedicated UI event names rather than overloading `ChatMessageAppended`, because explicit names make backend logs and frontend tests easier to read.

But the projected payload can still be message-shaped.

## Frontend mapping rules

In `wsManager.ts`, add support for:

- `ChatReasoningStarted`
- `ChatReasoningAppended`
- `ChatReasoningFinished`

Each should upsert a local `message` entity with `role: "thinking"`.

Since `MessageCard` already accepts `role === "thinking"`, no new renderer is required for the first slice.

## Proposed end-to-end flow

```text
Geppetto runtime
  -> EventThinkingPartial / EventInfo(thinking-started|thinking-ended|reasoning-summary)
      -> cmd/web-chat reasoning feature HandleRuntimeEvent(...)
          -> publish ChatReasoning* backend events
              -> timeline projection -> ChatMessage(kind, role=thinking)
              -> UI projection -> ChatReasoning* ui-events
                  -> websocket fanout
                      -> wsManager.ts
                          -> Redux timeline entity kind=message role=thinking
                              -> existing MessageCard renderer
```

## Pseudocode: backend feature

```go
type reasoningChatFeature struct{}

func (reasoningChatFeature) RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
    register event ChatReasoningStarted
    register event ChatReasoningDelta
    register event ChatReasoningFinished
    register ui-event ChatReasoningStarted
    register ui-event ChatReasoningAppended
    register ui-event ChatReasoningFinished
}

func reasoningID(parent string) string {
    return parent + ":thinking"
}

func (reasoningChatFeature) HandleRuntimeEvent(ctx context.Context, runtime chatapp.RuntimeEventContext, event gepevents.Event) (bool, error) {
    switch ev := event.(type) {
    case *gepevents.EventThinkingPartial:
        return true, runtime.Publish(ctx, "ChatReasoningDelta", map[string]any{
            "messageId": reasoningID(runtime.MessageID),
            "parentMessageId": runtime.MessageID,
            "role": "thinking",
            "chunk": ev.Delta,
            "content": ev.Completion,
            "status": "streaming",
            "streaming": true,
        })

    case *gepevents.EventInfo:
        switch ev.Message {
        case "thinking-started":
            return true, runtime.Publish(ctx, "ChatReasoningStarted", map[string]any{ ... })
        case "reasoning-summary":
            return true, runtime.Publish(ctx, "ChatReasoningFinished", map[string]any{ ...final summary text... })
        case "thinking-ended":
            return true, runtime.Publish(ctx, "ChatReasoningFinished", map[string]any{ ...streaming:false... })
        }
    }
    return false, nil
}
```

## Pseudocode: timeline projection

```go
func (reasoningChatFeature) ProjectTimeline(ctx context.Context, ev sessionstream.Event, _ *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, bool, error) {
    if ev.Name not in ChatReasoning* {
        return nil, false, nil
    }

    payload := toMap(ev.Payload)
    id := payload["messageId"]
    entity := currentKindEntity(view, chatapp.TimelineEntityChatMessage, id)
    entity["messageId"] = id
    entity["parentMessageId"] = payload["parentMessageId"]
    entity["role"] = "thinking"

    if payload has content {
        entity["content"] = payload["content"]
        entity["text"] = payload["content"]
    }
    if payload has reasoningTokens {
        entity["reasoningTokens"] = payload["reasoningTokens"]
    }
    entity["status"] = payload["status"]
    entity["streaming"] = payload["streaming"]

    return []sessionstream.TimelineEntity{{Kind: chatapp.TimelineEntityChatMessage, Id: id, Payload: structpb(entity)}}, true, nil
}
```

## Pseudocode: frontend mapping

```ts
case 'ChatReasoningStarted':
  return {
    upsert: messageEntity(messageId, {
      role: 'thinking',
      content: asString(payload.content) || '',
      status: 'streaming',
      streaming: true,
      parentMessageId: asString(payload.parentMessageId),
    }),
  }

case 'ChatReasoningAppended':
  return {
    upsert: messageEntity(messageId, {
      role: 'thinking',
      content: asString(payload.content) || asString(payload.text) || asString(payload.chunk),
      status: 'streaming',
      streaming: true,
      parentMessageId: asString(payload.parentMessageId),
      reasoningTokens: payload.reasoningTokens,
    }),
  }

case 'ChatReasoningFinished':
  return {
    upsert: messageEntity(messageId, {
      role: 'thinking',
      content: asString(payload.content),
      status: 'finished',
      streaming: false,
      parentMessageId: asString(payload.parentMessageId),
      reasoningTokens: payload.reasoningTokens,
    }),
  }
```

## Design Decisions and Rationale

## Decision 1: keep the feature in `cmd/web-chat`, not `sessionstream`

Reasoning is product/runtime semantics, not framework semantics. `sessionstream` should remain generic.

## Decision 2: use the `pkg/chatapp.FeatureSet` seam

This reuses the same extension model already proven by `agentmode`. It avoids growing a giant switch inside the base sink and keeps app-specific behavior explicit.

## Decision 3: model thinking as a second chat message first

The shortest path to visible value is to reuse the existing message renderer with `role: "thinking"`. The current UI already visually tolerates that role.

## Decision 4: consume `EventThinkingPartial`, not raw reasoning deltas, as the primary live signal

That is Geppetto’s compatibility event across engines. It avoids provider-specific duplication and matches older webchat donor behavior.

## Decision 5: separate “visible reasoning stream” from “reasoning token counts”

The first slice should make thinking visible. Counts can ride along when present, but the project should not stall on building token badges first.

## Alternatives Considered

## Alternative A: modify `pkg/chatapp` base sink directly

Rejected.

Why:

- it would hard-code reasoning into the shared chat package,
- it breaks the clean distinction between base chat behavior and app-owned extensions,
- it makes future feature growth harder to reason about.

## Alternative B: resurrect deleted `cmd/web-chat/thinkingmode/*`

Rejected.

Why:

- that was a deleted legacy/tutorial island, not part of the canonical app,
- reviving it would conflict with the clean-cut migration work already completed,
- it would move the codebase backwards architecturally.

## Alternative C: route live app back through the old SEM translator

Rejected.

Why:

- the canonical app now uses sessionstream snapshots and UI events,
- going back to the SEM path would reintroduce the old architectural split,
- we only need the **idea** from the old SEM mapping, not the old pipeline itself.

## Alternative D: build a completely custom reasoning widget first

Deferred.

Why:

- not required to prove the architecture,
- higher UI complexity,
- reusing `MessageCard` is enough to validate backend and hydration correctness.

## File-by-File Implementation Plan

## Phase 0 — documentary groundwork (this ticket)

Files:

- `le-chat/ttmp/.../EVT-STREAM-014/index.md`
- `le-chat/ttmp/.../EVT-STREAM-014/tasks.md`
- `le-chat/ttmp/.../EVT-STREAM-014/changelog.md`
- this design doc
- the diary

Goal:

- make the architecture and scope explicit before implementation.

## Phase 1 — backend feature scaffolding

Create:

- `pinocchio/cmd/web-chat/reasoning_chat_feature.go`
- `pinocchio/cmd/web-chat/reasoning_chat_feature_test.go`

Implement:

- schema registration,
- reasoning message id helper,
- runtime-event translation for `EventThinkingPartial` and `EventInfo` boundaries,
- timeline projection into `ChatMessage`,
- UI projection for dedicated reasoning UI events.

## Phase 2 — app wiring

Update:

- `pinocchio/cmd/web-chat/main.go`

Change:

- register `newReasoningChatFeature()` alongside `newAgentModeChatFeature()`.

## Phase 3 — frontend websocket mapping

Update:

- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`

Implement:

- `ChatReasoningStarted`
- `ChatReasoningAppended`
- `ChatReasoningFinished`

Confirm that snapshot mapping already renders reasoning through `ChatMessage` payload + `role: "thinking"`.

## Phase 4 — visual polish (minimal)

Likely no functional change needed in:

- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`

But verify:

- thinking messages are visually distinguishable enough,
- streaming dot behavior is acceptable,
- markdown rendering is safe for the reasoning content the model emits.

## Phase 5 — validation

Add or update tests in:

- `pinocchio/cmd/web-chat/reasoning_chat_feature_test.go`
- `pinocchio/cmd/web-chat/app/server_test.go`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`

Then do a real browser run with a reasoning-capable profile.

## Test Strategy

## Backend unit tests

At minimum, cover:

1. `EventThinkingPartial` -> `ChatReasoningDelta`
2. `EventInfo("thinking-started")` -> `ChatReasoningStarted`
3. `EventInfo("thinking-ended")` -> `ChatReasoningFinished`
4. `EventInfo("reasoning-summary")` -> final content override/update
5. projection produces `ChatMessage` entity with `role: "thinking"`
6. reasoning message ID derivation stays stable
7. token count extraction from metadata extras when present

## Frontend unit tests

At minimum, cover:

1. reasoning UI event -> local `message` entity with `role: "thinking"`
2. delta events prefer cumulative `content`
3. finish event preserves existing content when payload omits it
4. snapshot path already renders `ChatMessage` with `role: "thinking"`

## App/server tests

If feasible, add a fake runtime event stream that emits:

```text
thinking-started
partial-thinking("Let me think")
partial-thinking("Let me think this through")
reasoning-summary("Short summary")
thinking-ended
partial assistant text
final assistant text
```

Then assert that:

- snapshot contains a thinking message entity,
- websocket stream sends reasoning UI events in order,
- assistant answer still appears normally,
- reconnect shows the thinking entity durably.

## Browser validation

Use a real profile that actually emits thinking. Do not assume every configured profile will do so.

Validation checklist:

- create a new session,
- submit a prompt that gives the model time to think,
- confirm a visible thinking card appears before or during answer generation,
- confirm the thinking card stops streaming when reasoning ends,
- confirm reload preserves the card if the feature is projected durably,
- confirm the assistant answer still renders normally.

## Risks and Sharp Edges

## Risk 1: confusing “reasoning tokens” with “reasoning text”

These are related but different. If the ticket tries to solve both as one opaque problem, the implementation may become muddled.

Mitigation:

- ship visible thinking first,
- treat `reasoning_tokens` as metadata attached to the final state.

## Risk 2: duplicate reasoning content

If the feature consumes both `EventThinkingPartial` and `EventReasoningTextDelta`, the UI may display duplicate text.

Mitigation:

- use `EventThinkingPartial` as the canonical live signal,
- ignore raw reasoning text delta events in the first slice.

## Risk 3: models/providers that never emit thinking

Some profiles simply will not surface reasoning text.

Mitigation:

- degrade gracefully,
- do not create empty placeholder entities unless `thinking-started` or a real delta was observed,
- validate with a known reasoning-capable profile.

## Risk 4: finish events with no final content

`thinking-ended` may not carry content. If the frontend blindly overwrites content with empty text, the card will blank out.

Mitigation:

- on finish, preserve prior content when the payload does not include a non-empty final string.

## Risk 5: accidental resurrection of legacy architecture

Contributors may be tempted to reuse old SEM or `thinkingmode` code paths directly.

Mitigation:

- keep implementation narrowly inside the current feature seam,
- treat old SEM code only as a donor for naming/UX ideas.

## Open Questions

1. Should the first slice display a distinct badge for `reasoningTokens` when present, or leave that for a follow-up?
2. Should `thinking-started` create an empty placeholder card immediately, or should the UI wait for the first visible delta?
3. If both `reasoning-summary` and `thinking-ended` arrive, should `reasoning-summary` always win as the final visible content?
4. Should thinking cards be persisted indefinitely in history, or eventually collapsed/hidden by UX policy?

My recommendation:

- do **not** block on question 1,
- for question 2, prefer “first visible delta creates the card” unless UX explicitly wants a placeholder,
- for question 3, yes: let explicit summary text win,
- for question 4, persist it for now because durability is part of the architecture this app is trying to demonstrate.

## Recommended Reading Order for a New Intern

1. `pinocchio/pkg/chatapp/chat.go`
   - understand the base runtime sink and base chat event vocabulary.
2. `pinocchio/pkg/chatapp/features.go`
   - understand how app-owned features plug into runtime translation and projections.
3. `pinocchio/cmd/web-chat/agentmode_chat_feature.go`
   - use it as the immediate implementation pattern donor.
4. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
   - understand the browser contract and how snapshots differ from live UI events.
5. `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`
   - confirm the existing renderer can already show `role: "thinking"`.
6. `geppetto/pkg/events/chat-events.go`
   - understand upstream event taxonomy.
7. `geppetto/pkg/steps/ai/openai/engine_openai.go`
   - understand chat-completions reasoning flow.
8. `geppetto/pkg/steps/ai/openai_responses/engine.go`
   - understand reasoning summary + partial-thinking behavior.

## Final Recommendation

Implement reasoning/thinking streaming as a **new app-owned `cmd/web-chat` feature** that consumes Geppetto `EventThinkingPartial` and thinking-related `EventInfo` signals, publishes canonical `ChatReasoning*` events, projects them into durable `ChatMessage` entities with `role: "thinking"`, and lets the existing React message renderer display them.

That approach is the smallest architecture-consistent slice. It preserves the clean `sessionstream` / `pinocchio` split, reuses the feature seam already proven by `agentmode`, avoids resurrecting legacy `thinkingmode`, and gives the example web-chat the missing capability the user is asking for.

## References

### Pinocchio

- `pinocchio/pkg/chatapp/chat.go:19-35,96-124,220-275,352-506`
- `pinocchio/pkg/chatapp/features.go:10-21,46-103`
- `pinocchio/cmd/web-chat/main.go:323-331`
- `pinocchio/cmd/web-chat/app/server.go:94-135,229-260`
- `pinocchio/cmd/web-chat/agentmode_chat_feature.go:29-124`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts:80-226`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx:14-33`
- `pinocchio/cmd/web-chat/web/src/sem/registry.ts:159-196`
- `pinocchio/pkg/webchat/sem_translator.go:245-251`
- `pinocchio/pkg/webchat/sem_translator_test.go:147-166`

### Geppetto

- `geppetto/pkg/events/chat-events.go:13-20,350-365,843-862`
- `geppetto/pkg/steps/ai/openai/engine_openai.go:273-280,324-352`
- `geppetto/pkg/steps/ai/openai_responses/engine.go:340-347,462-490,507-513,798-806`
- `geppetto/pkg/inference/engine/inference_config.go:15-25`
