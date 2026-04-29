---
Title: Contributor playbook — adding preview and committed custom chat events to evtstream apps
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
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Frontend snapshot and live UI-event mapping reference
    - Path: ../../../../../../../pinocchio/pkg/evtstream/apps/chat/chat.go
      Note: Canonical translation and projection reference implementation
    - Path: ../../../../../../../pinocchio/pkg/inference/runtime/composer.go
      Note: Final runtime contract uses WrapSink without a concrete legacy sink field
    - Path: ../../../../../../../pinocchio/pkg/webchat/conversation.go
      Note: Legacy webchat now assembles concrete conversation sinks via an explicit sink-builder dependency
    - Path: ../../../../../../../pinocchio/pkg/webchat/router.go
      Note: Router now assembles the base Watermill sink and applies runtime plus router wrappers
ExternalSources: []
Summary: Short contributor playbook for adding a runtime-driven preview event, a committed backend event, durable projection, and frontend widget rendering to an evtstream-backed chat app.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Give contributors a short, implementation-oriented checklist derived from the longer EVT-STREAM-013 design and implementation work.
WhenToUse: Use when extending cmd/web-chat or another evtstream chat app with a new custom backend event and matching UI widget.
---


# Contributor playbook — adding preview and committed custom chat events to evtstream apps

This is the short version of EVT-STREAM-013. The long guide explains the architecture in depth; this playbook is the practical sequence to follow when you want to add a new custom event and widget to an `evtstream` chat application.

The core rule is simple:

> **Preview is provisional and live-only. Commit is authoritative and durable.**

In other words, if a middleware or parser can infer something while tokens are still streaming, surface that as a preview. Once the model run completes and the middleware emits the final authoritative signal, translate that into a committed backend event, project durable state, and let hydration restore it on reload.

---

## 1. Keep the ownership model straight

The final wiring should look like this:

```text
runtime composition
  owns middleware chain
  owns sink decoration (WrapSink)
        |
        v
app-owned base sink
  translates runtime events into canonical app backend events
        |
        v
evtstream backend events
  -> UI projection
  -> timeline projection
  -> hydration snapshot
  -> websocket fanout
  -> frontend widget
```

Two rules fall out of that diagram:

1. **`ComposedRuntime` owns sink decoration, not the chat app.**
2. **The chat app owns canonical event translation and durable projection, not the middleware package.**

Do not put app-specific websocket or projection logic into `pkg/middlewares/...`. Do not put app-specific compatibility behavior into `pkg/evtstream` core.

---

## 2. Decide what is preview vs. what is commit

Before writing code, classify your signal.

### Preview candidates
Use preview events for information that is useful while the model is still speaking but may still change.

Examples:
- candidate agent mode
- candidate tool selection summary
- candidate structured form values
- partially parsed YAML/JSON selections

### Commit candidates
Use committed events for information that should survive reload, restart, and hydration.

Examples:
- final mode switch
- final tool execution selection
- final structured output payload
- final authoritative workflow state

If you cannot explain why the preview may differ from the final answer, you probably do not need a preview path.

---

## 3. Add or reuse a runtime event at the middleware layer

If the middleware already emits an authoritative runtime event, keep that event as the source of truth.

For a preview path, add a **transient runtime event** that can be emitted during streaming. In the `agentmode` example this is:

- `pinocchio/pkg/middlewares/agentmode/preview_event.go`

The preview event should be small and explicit. Keep fields focused on what the UI needs to render a useful preview card.

Recommended fields:
- candidate value or mode
- short analysis/reason text
- parse state such as `analysis-only`, `candidate`, or `complete`
- enough correlation metadata to tie it to the current message/run if needed

---

## 4. Emit preview events from the structured sink path

For streaming previews, the right place is usually the structured sink wrapper or extractor session, not the final middleware completion hook.

In the `agentmode` example, the progressive logic lives here:

- `pinocchio/pkg/middlewares/agentmode/structured_extractor.go`

Use `parsehelpers.YAMLController` or the equivalent incremental parser so you can inspect partial content as chunks arrive.

Checklist:
- accumulate chunks incrementally
- try parsing on reasonable boundaries
- emit preview events only when the parsed preview meaningfully changes
- ignore incomplete or temporarily invalid intermediate states
- avoid duplicate preview spam

### Anti-pattern
Do **not** persist preview state directly from the extractor. The extractor should only emit runtime events.

---

## 5. Keep runtime sink wrapping in runtime composition

If your middleware needs sink decoration to emit preview events, wire that in the runtime composer.

Relevant files in the current `cmd/web-chat` example:
- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `pinocchio/cmd/web-chat/agentmode_sink.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/conversation.go`

The important contract is:

```go
type EventSinkWrapper func(events.EventSink) (events.EventSink, error)

type ComposedRuntime struct {
    Engine   engine.Engine
    WrapSink EventSinkWrapper
    ...
}
```

That means the runtime returns an engine plus an optional decorator for the app-owned base sink. The app provides the concrete sink, then applies `WrapSink` before inference starts.

### Why this matters
This keeps middleware and its matching structured-output parsing behavior together. If a runtime enables `agentmode`, it also brings the matching sink behavior with it.

---

## 6. Translate runtime events into canonical app backend events

The runtime event is not yet the canonical evtstream contract. The app needs to translate it.

In the canonical chat app, that translation happens in:

- `pinocchio/pkg/evtstream/apps/chat/chat.go`

For each runtime event type:

1. map it into a canonical backend event name,
2. normalize the payload shape,
3. include the chat message correlation id where needed.

For the current `agentmode` slice the canonical backend events are:
- `ChatAgentModePreviewUpdated`
- `ChatAgentModeCommitted`

That translation should happen in the runtime sink implementation that already handles:
- partial completion
- final completion
- interrupts
- errors

### Anti-pattern
Do **not** send raw middleware-specific runtime event objects directly to the browser.

---

## 7. Project only committed state into durable timeline entities

Once the canonical committed event exists, project it to timeline state.

Current example:
- committed `ChatAgentModeCommitted`
- projected to timeline entity kind `AgentMode`

Keep preview events **live-only** unless you have a very strong reason to hydrate them later.

Why:
- previews are speculative
- hydration should restore stable truth
- reload should not resurrect stale provisional UI

A good default pattern is:

```text
preview event
  -> UI event only

commit event
  -> UI event
  -> timeline entity upsert
  -> hydration snapshot
```

---

## 8. Clear preview state when the run ends or commits

If you add a preview widget, you also need explicit preview clearing.

Clear preview UI state when:
- the committed event arrives
- inference finishes normally
- inference stops/interruption occurs
- an error ends the run

In the canonical chat app this is handled with a dedicated UI clear event emitted by the projection layer.

If you skip this, the frontend will often look correct in the happy path but leave stale preview cards behind after edge-case exits.

---

## 9. Add frontend mapping in two places

For `cmd/web-chat`, there are two frontend responsibilities.

### A. Map snapshot entities and UI events into store mutations
Relevant file:
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`

You need to map:
- hydrated committed entities from snapshots
- live preview UI events
- preview-clear UI events

### B. Render the widget/card
Relevant files:
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/rendererRegistry.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`

Keep the rendering component intentionally dumb. By the time data reaches the card renderer, it should already be normalized into a frontend-friendly shape.

---

## 10. Add three layers of tests

Do not stop at one test layer.

### Layer 1 — extractor/runtime tests
Example area:
- `pinocchio/pkg/middlewares/agentmode/structured_extractor_test.go`

Test:
- incomplete intermediate input
- preview emission after useful partial parse
- deduplication
- final authoritative parse still working

### Layer 2 — canonical chat backend tests
Example area:
- `pinocchio/pkg/evtstream/apps/chat/chat_test.go`

Test:
- preview backend event publication
- committed backend event publication
- preview-clear behavior
- durable projection for committed state

### Layer 3 — frontend mapping tests
Example area:
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`

Test:
- snapshot hydration mapping for committed entities
- live preview event mapping
- preview-clear mapping

### Layer 4 — real browser validation
Run the real app with a real profile and observe:
- preview card appears during streaming
- committed card appears when final switch lands
- preview disappears after commit/finish

---

## 11. Validation commands

Backend-focused validation:

```bash
cd pinocchio && go test ./pkg/inference/runtime ./pkg/webchat/... ./cmd/web-chat/... ./pkg/evtstream/apps/chat ./pkg/middlewares/agentmode -count=1
```

Frontend-focused validation:

```bash
cd pinocchio/cmd/web-chat/web && npm run check
cd pinocchio/cmd/web-chat/web && npx vitest run src/ws/wsManager.test.ts
cd pinocchio/cmd/web-chat/web && npm run build
```

Real browser run:

```bash
cd pinocchio && go build -o /tmp/webchat-bin ./cmd/web-chat
/tmp/webchat-bin web-chat --addr 127.0.0.1:18120
```

Then open the app, select an agentmode-enabled profile such as `gpt-5-nano-low`, and use a prompt that is likely to trigger the structured switching behavior.

---

## 12. Common pitfalls

### Pitfall 1: using a side sink instead of wrapping the app sink
If the middleware needs to inspect the same event stream as the app runtime sink, it usually needs to **wrap** the sink, not sit beside it.

### Pitfall 2: persisting preview state
This makes hydration lie. If the preview was provisional, do not restore it as truth after reload.

### Pitfall 3: leaking middleware-specific payloads into the frontend contract
Normalize at the app layer. The browser should speak the app contract, not the middleware package’s raw internal types.

### Pitfall 4: forgetting preview-clear on stop/error
You will only notice this during real browser validation, not in the happy-path unit test.

### Pitfall 5: assuming malformed and incomplete streaming input behave the same
They do not. In progressive parsing, incomplete content can become valid later. Truly malformed buffered content may not “recover” the way you expect.

---

## 13. File map for the current agentmode example

### Runtime composition and sink ownership
- `pinocchio/pkg/inference/runtime/composer.go`
- `pinocchio/cmd/web-chat/runtime_composer.go`
- `pinocchio/cmd/web-chat/agentmode_sink.go`
- `pinocchio/pkg/webchat/router.go`
- `pinocchio/pkg/webchat/conversation.go`

### Middleware/runtime signal production
- `pinocchio/pkg/middlewares/agentmode/middleware.go`
- `pinocchio/pkg/middlewares/agentmode/preview_event.go`
- `pinocchio/pkg/middlewares/agentmode/structured_extractor.go`

### Canonical chat translation and projection
- `pinocchio/pkg/evtstream/apps/chat/chat.go`
- `pinocchio/pkg/evtstream/apps/chat/service.go`

### Frontend mapping and rendering
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
- `pinocchio/cmd/web-chat/web/src/store/timelineSlice.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/rendererRegistry.ts`
- `pinocchio/cmd/web-chat/web/src/webchat/cards.tsx`

### Tests
- `pinocchio/pkg/middlewares/agentmode/structured_extractor_test.go`
- `pinocchio/pkg/evtstream/apps/chat/chat_test.go`
- `pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`
- `pinocchio/pkg/webchat/conversation_service_test.go`
- `pinocchio/pkg/webchat/chat_service_test.go`
- `pinocchio/pkg/webchat/stream_hub_test.go`
- `pinocchio/pkg/webchat/llm_loop_runner_test.go`

---

## 14. Minimal implementation checklist

Use this as the short execution list:

- [ ] Decide preview vs commit semantics
- [ ] Add preview runtime event type if needed
- [ ] Emit preview events from structured sink/extractor
- [ ] Return sink decoration via `ComposedRuntime.WrapSink`
- [ ] Translate preview/commit runtime events in canonical chat sink
- [ ] Project committed state to durable timeline entities
- [ ] Emit preview-clear UI events on commit/finish/stop
- [ ] Map snapshot + UI events in frontend websocket layer
- [ ] Render preview and committed cards/widgets
- [ ] Add extractor, backend, frontend, and browser validation

If you follow that list in order, you will usually avoid the two biggest failure modes: putting app logic into middleware packages, and accidentally making preview state durable.
