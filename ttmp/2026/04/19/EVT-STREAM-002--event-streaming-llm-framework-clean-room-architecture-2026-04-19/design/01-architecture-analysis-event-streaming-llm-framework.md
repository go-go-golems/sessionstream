---
Title: Architecture Analysis — Event Streaming LLM Framework
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
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: reference/01-source-image-transcription-2026-04-19-sketches.md
      Note: "Verbatim transcription this analysis is built on (clean-room source of truth)."
    - Path: sources/diagram-page-1.png
      Note: "Page 1 — dataflow sketch."
    - Path: sources/building-blocks-page-2.png
      Note: "Page 2 — building blocks outline."
ExternalSources: []
Summary: "Clean-room architectural reading of the 2026-04-19 sketches plus four post-sketch clarifications from the author: a three-layer (backend / generic / client) framework for realtime websocket-driven LLM applications, organised around SessionId-keyed event streams, a multiplexing edge, and a single backend-event stream fanning out to two pluggable user-supplied projections (UI + Timeline). Every substantive claim is annotated with a `> **Trace:**` blockquote citing its source so the author can audit each interpretation."
LastUpdated: 2026-04-19T15:00:00-04:00
WhatFor: "Translate the hand-drawn sketches into a usable architectural narrative — naming the layers, the seams, the responsibilities, and the open questions — so implementation can begin without re-deriving intent from the whiteboard."
WhenToUse: "When scoping or implementing the framework's first concrete pieces, when arguing about layer boundaries, or when onboarding a new contributor to the EVT-STREAM line of work."
---

# Architecture Analysis — Event Streaming LLM Framework

> **Clean-room note.** This document is derived **only** from the two 2026-04-19 sketches transcribed in `reference/01-source-image-transcription-2026-04-19-sketches.md`. No other repository documents, prior tickets, or existing implementations were consulted. Anywhere this doc names a concrete API, file, or behaviour beyond what the sketches say, it is an *inference* and is marked as such.

> **How to read the traces.** After every substantive claim in this document there is a `> **Trace:**` blockquote of the form *"I wrote X because I found Y in the source and it says Z."* The traces are the audit trail back to the sketches; if a claim has no trace, treat it as unsupported and challenge it.

> **Terminology and post-sketch clarifications.** The author has since clarified four points that supersede the raw-sketch reading. They are applied throughout this document and called out below.
>
> 1. **Single object, canonical name `Session`.** The page-1 `conv` glyph and the page-2 *session objects* bullet name the same object. The canonical name is **`Session`** (Go: `SessionId`). The page-1 token `conv-id` is preserved only when quoting the source; everywhere the framework's *types* are named, this document and the framework use `Session` / `SessionId` exclusively.
> 2. **The connection is opaque to the backend.** The backend does *not* read connection objects. The substrate owns connection state; a `ConnectionId` is the only identifier the backend may see, and it is metadata, not something the backend inspects or stores.
> 3. **Two user-supplied projections, not one.** A single backend-event stream fans out to two pluggable, user-supplied projections: `backend events → frontend events` (delivered to the UI) and `backend events → timeline entities` (persisted into the hydration store).
> 4. **Concrete command shape.** An incoming command is `(Name, Payload, ConnectionId, SessionId)`. Commands are **synchronous** and substrate-dispatched against a registered handler keyed by `Name`. The page-1 `profile-id` field is *not* a command field; it is connection-bound user/identity context that will live on the connection object as future work.

## 1. One-paragraph summary

The sketches describe a **reusable substrate for realtime, websocket-driven LLM/agent applications**. The substrate's job is to absorb everything that every such application has to do anyway — websocket lifecycle, session/connection state, session routing, event projection, hydration, command dispatch, multiplexing across many clients, cancellation — so that an individual application (a chat backend, an agent, a scraper-with-replay) only has to declare its **event schemas**, its **commands**, and its **per-event processors**. The architecture is split into three layers (Backend / Generic / Client), with the Generic layer doing the heavy lifting at the network edge and the two end layers being thin, schema-driven adapters.

> **Trace — "reusable substrate for realtime, websocket-driven LLM/agent applications".** I wrote this because page 1 prose says *"It's basically realtime websocket structured widget handling + hydration with one side just emitting events with a [session] id, and the rest managed by this intermediate layer"* (source word in brackets per the canonical-name decision; verbatim original is in the transcription doc) and page 1 also says *"the cleaned up webchat package and maybe build a couple different backends"* — i.e., one substrate, multiple backends.
>
> **Trace — list of substrate responsibilities (websocket lifecycle, session/connection state, session routing, event projection, hydration, command dispatch, multiplexing, cancellation).** I assembled this list directly from page 2's "Generic layer" bullets (*websocket handling (establish, disconnect)*, *session objects*, *generic event processing pipeline + projection*, *generic hydration / timeline entity storage*, *pluggable command framework*) plus the page 1 right-column bullet list (*comm multiplexing*, *conv creation*, *conv hydration*, *cancellation multiplex* — `conv` is the source token for `Session`).
>
> **Trace — "only has to declare its event schemas, its commands, and its per-event processors".** Page 2's "Backend layer" bullets are exactly: *backend event schemas*, *set of commands (start inference, stop inference)*, *event processors* (plus session metadata). I summarised those four bullets as the per-app contribution.
>
> **Trace — "three layers (Backend / Generic / Client)".** Page 2 is literally titled "BUILDING BLOCKS" and is structured as three top-level bullets: *Backend layer*, *Generic Layer*, *Client layer*. I am using its exact decomposition.
>
> **Trace — "the Generic layer doing the heavy lifting at the network edge".** This is a paraphrase: page 2's Generic layer has the most bullets (websocket, sessions, pipeline, commands, hydration, timeline events, connection objects), and the page-1 diagram puts everything between SRV and the websocket endpoints (multiplexing, broadcast, hydration arrow) at this seam.

## 2. The diagram, read out loud

Page 1 sketches two complementary slices of the same system.

> **Trace — "two complementary slices of the same system".** Page 1 has visually two distinct diagram clusters: an upper pipeline (SRC → queue cells → PKG) and a lower fan-out (SRV → many conv/PROC rows → websocket endpoints). I am calling those the "two slices" and treating them as parts of one system because they share the `conv-id` key (top slice arrow says `do something (conv-id, profile-id)`; bottom slice rows are labelled `conv`).

### 2.1 Top sub-diagram — the "fire and forget" slice (`SRC → … → PKG`)

A `SRC` (source) emits a request — annotated `do something (conv-id, profile-id)` — into a short pipeline of staged buffers (`[///]-[///]-[///]`) which terminates in `PKG` (a packaged result). Two annotations frame this slice:

- **`CMDs carried through`** — the units flowing through the pipeline are *commands*, not raw transport frames.
- **`no need for persistent connection`** — this slice is request/response-shaped: the caller does not have to hold a socket open for the work to progress.

> **Trace — "fire and forget".** This is my label, not the author's. I chose it because the annotation under the pipeline literally says *"no need for persistent connection"*, which is the operational definition of fire-and-forget for this architecture.
>
> **Trace — `SRC` and `PKG` boxes and the `[///]-[///]-[///]` queue cells.** All three glyphs are drawn as boxes on page 1, top sub-diagram. I read `SRC` as "source" and `PKG` as "packaged output" — these are interpretations of the abbreviations, captured as the glyph key in the transcription.
>
> **Trace — `do something (conv-id, profile-id)` annotation.** This is verbatim text from page 1: an arrow points into `SRC` with the words "do something (conv-id) profile-id" written next to it.
>
> **Trace — "the units flowing through the pipeline are commands, not raw transport frames".** Page 1 has the literal label *"CMDs carried through"* across the queue cells. I am interpreting `CMDs` as "commands" and using that label to assert the type of payload in flight.

> **Post-sketch update on the inbound annotation.** Per clarification #4, the post-sketch command shape is `(Name, Payload, ConnectionId, SessionId)`. The page-1 `conv-id` becomes `SessionId`; the page-1 `profile-id` is *not* a command field — it migrates onto the connection object (future work) as user/identity context. The annotation captures what was scribbled at the time, not the settled type.

**Reading.** This is the "issue a job, come back later" path. It is interesting precisely because it sits *in the same framework* as the realtime path below: the same conv-id / profile-id keys, the same command vocabulary, but a different transport posture. The sketch is staking a claim that **command issuance and event streaming are decoupled** — you can hand a command in over a non-persistent channel and pick the resulting event stream up later (this is what "REPLAY" gestures at in the prose at the top of the page).

> **Trace — "command issuance and event streaming are decoupled".** I inferred this from the conjunction of two source facts: (a) page 1 says the top slice has *"no need for persistent connection"* yet still moves `CMDs`; and (b) the bottom slice is where realtime event streaming over websockets happens. If a command can enter without a socket, but the event stream lives on the socket side, the two must be decoupled.
>
> **Trace — REPLAY reference.** Page 1's top prose block says: *"REPLAY will allow me to trace and reproduce individual scraper steps and refine the program, and then re-run scrape [usemint] and the labeled data for real."* I am using REPLAY as evidence that the author already wants to issue commands without a live UI subscriber — exactly the use case the decoupling enables.

### 2.2 Bottom sub-diagram — the "live multiplexed" slice (`clients ↔ SRV ↔ conv ↔ PROC`)

The bottom half is the realtime path:

- On the **left**, three queue arrays feed into a single `SRV` (server). These are inbound channels carrying commands from clients.
- The `SRV` **broadcasts** (annotated literally on the arrow) into a fan of per-session rows. Each row pairs a `conv` object (the source's name for `Session`) with a `PROC` cloud — i.e., one logical session has one processor doing the work.
- On the **right**, multiple websocket endpoints (`WC`, `WS`, `WS`, drawn as hatched circles) attach to a `[?Common?] CLIENTS` aggregator and a `CMDS` lane. From here, four capabilities are listed: **comm multiplexing, conv creation, conv hydration, cancellation multiplex**.
- Underneath, a `+ state machine (up to PROC, guess)` annotation parks the question of *where* the per-session state machine lives — the author's tentative answer is "as far up as `PROC`".
- A `map to conv` arrow points from the inbound side at the conv objects: **the routing job is "which conv does this message belong to?"**.
- A curving `HYDRATION` arrow sweeps from the right edge down toward two stacked DB cylinders. One cylinder has an `X` next to it — read here as "**don't put hydration state in this store**" or "this persistence path is not the one we want".

> **Trace — "three queue arrays feed into a single SRV".** Page 1, bottom sub-diagram, left edge: three stacked `////` rectangles each have an arrow into the `SRV` box. I am calling them "queue arrays" because the hatched-rectangle glyph reads as a multi-cell queue.
>
> **Trace — "broadcasts (annotated literally on the arrow)".** Page 1 has the word *"Broadcast"* written along an arrow leaving `SRV` (it sits to the left of the conv rows). I am quoting the author's word.
>
> **Trace — "one logical session has one processor doing the work".** I read this from the visual structure: each row of the fan-out shows exactly one `conv` paired with exactly one `PROC` cloud. The repetition (multiple rows) is the multi-session case; within a row it is 1:1.
>
> **Post-sketch update on `conv` rows.** Per clarification #1, each `conv` row is a `Session`. The page-2 *session objects* bullet and the page-1 `conv` glyph are the same object; I use `Session` / `SessionId` for the framework type and keep `conv` only when quoting the source.
>
> **Trace — `WC` / `WS` are websocket endpoints.** Page 1 right column has letters `WC`, `WS`, `WS` next to hatched circle glyphs. I am interpreting these as websocket-client and websocket-server (or just websocket) endpoints — this is an interpretation captured in the transcription glyph key.
>
> **Trace — the four capabilities ("comm multiplexing, conv creation, conv hydration, cancellation multiplex").** These are a verbatim bullet list on page 1 right column. I quoted them as-is.
>
> **Trace — "the author's tentative answer is 'as far up as PROC'".** Page 1 has the literal annotation *"+ state machine (up to PROC, guess)"*. The word "guess" in the source is what makes me call it "tentative".
>
> **Trace — "the routing job is 'which conv does this message belong to?'".** Page 1 has a label *"map to conv"* with an arrow pointing from the inbound side at the conv objects. I am paraphrasing the arrow's intent: mapping = routing to a conv.
>
> **Trace — `X` cylinder reading.** Page 1 shows two stacked DB cylinders at the bottom; one has an X next to it. The author drew an X next to a storage glyph — the most natural reading of "X next to a thing" in a sketch is "not this". I flag this is a reading, not a transcription.

**Reading.** This slice answers: *"how do many websocket clients, possibly attached to the same sessions, share a single live processor cleanly?"* The substrate's load-bearing responsibilities here are:

1. **Multiplexing** — many client sockets ↔ one session ↔ one processor.
2. **Routing** — `map to conv` (page-1 source label): turn an incoming websocket frame into a `(SessionId, command)` pair.
3. **Broadcast** — turn a single processor's emitted events into a fan-out to every client subscribed to that session.
4. **Lifecycle** — session creation and cancellation across the multiplexed surface.
5. **Hydration** — when a (re)connecting client subscribes to an existing session, replay enough state for it to render.

> **Trace — the five responsibilities.** Each item maps directly to a label on page 1: *Multiplexing* ← bullet "comm multiplexing"; *Routing* ← label "map to conv"; *Broadcast* ← annotation "Broadcast" on the arrow leaving SRV; *Lifecycle* ← bullets "conv creation" and "cancellation multiplex"; *Hydration* ← bullet "conv hydration" plus the curving HYDRATION arrow at the bottom. I deliberately re-stated each bullet in operational terms rather than just listing the labels.

### 2.3 How the two slices fit together

The two diagrams share the **conv-id** key. The top slice issues commands without holding a socket; the bottom slice exposes the resulting event stream over websockets. The substrate's contract is: **wherever a command came in, the events for its conv-id are available on the realtime side**. That is the whole reason "REPLAY" works as a feature — it can re-issue commands into the top slice and the realtime side will surface the events as if they had been live.

> **Trace — "the two diagrams share the conv-id key".** Top slice has the explicit annotation *"do something (conv-id, profile-id)"* on the inbound arrow into `SRC`. Bottom slice has rows literally labelled `conv` (and the routing arrow says *"map to conv"*). The shared token across both clusters is `conv` / `conv-id`.
>
> **Trace — "wherever a command came in, the events for its SessionId are available on the realtime side".** This is my synthesis. I am inferring it from: (a) the prose statement *"one side just emitting events with a [session] id, and the rest managed by this intermediate layer"* (source word in brackets per the canonical-name decision), and (b) the existence of two distinct command entry shapes (top non-persistent slice vs. bottom websocket inbound queues) feeding the same Session model. Marked as inference because the sketch never literally writes a contract sentence — but it is the only consistent reading.
>
> **Trace — "REPLAY can re-issue commands into the top slice".** Page 1 prose: *"REPLAY will allow me to trace and reproduce individual scraper steps and refine the program, and then re-run scrape …"*. "Re-run" is what I mean by "re-issue commands". The top slice is the natural home for this because it is the non-persistent path.

## 3. The building blocks (page 2), interpreted

Page 2 names the substrate's parts. Reading it as an architecture, three layers fall out cleanly:

### 3.1 Backend layer — *application-specific*

> "for example llm chat"

What an *individual* application contributes:

- **Backend event schemas** — the typed events this app emits. Examples span chat *and* agentic / document workflows: `InferenceStarted`, `TokensDelta`, `ToolCallRequested`, `TaskStarted`, `TaskCompleted`, `DocumentWritten`, `ScraperStepCompleted`, `PlanRevised`, `ArtifactPublished`. The substrate is application-agnostic — chat is one shape, an autonomous agent emitting task/document/artifact events is another.
- **Set of commands** — the typed commands this app accepts. Examples again span shapes: `StartInference`, `StopInference`, `RunAgent`, `CancelTask`, `RetryStep`, `WriteDocument`, `ReplayScrape`. Each one is a registered handler keyed by name.
- **Session metadata** — application-specific session shape (profile, token count, …).
- **Event processors** — the functions that actually do the work in response to commands and emit events. Note: per clarification #2, the substrate also expects two *projection* processors per app (UI projection, timeline projection) which are sketched in §3.3 and §6 — those are distinct from the command-handling "event processors" page 2 lists here. To avoid term collision below, this document calls the page-2 "event processors" **command handlers** when speaking of the framework's API.

> **Trace — the four bullets.** Verbatim from page 2 under "Backend layer (for example llm chat)": *backend event schemas*; *set of commands (start inference, stop inference)*; *session metadata → (profile, token count, etc.)*; *event processors*. My example event names (`inference.started`, etc.) are illustrative, marked with "e.g.", and are not in the source.
>
> **Trace — "(for example llm chat)" parenthetical.** Page 2 has this exact parenthetical next to the "Backend layer" heading. I quoted it as the section's epigraph.

**Reading.** The Backend layer is *just declarations + handlers*. It doesn't know about websockets, sessions on the wire, multiplexing, or hydration. That is by design: the entire point of the substrate is that swapping `chat` for `agent` for `scraper` should require only this layer to change.

> **Trace — "doesn't know about websockets, sessions on the wire, multiplexing, or hydration".** Page 2 puts each of those concerns under "Generic layer", not under "Backend layer". The absence is the evidence: those bullets are not in the Backend layer's list.
>
> **Trace — "swapping chat for agent for scraper".** Page 1 prose names *webchat package* and *scraper steps* (REPLAY block); the user's question that opened this work also named *agents, chats, etc.*. So the variants under discussion in the source are exactly chat / agent / scraper.

### 3.2 Generic layer — *the substrate itself*

> Margin annotation on the page: "**connection objects**", and on the side: "**this also allows managing connection state from the backend side**".

This is where the architecture lives:

- **Websocket handling** — establish/disconnect, the wire-level lifecycle.
- **Session objects** *(+ allow backend metadata)* — a generic session container that *also* carries the backend layer's session metadata. The seam is: generic owns the object, backend owns the payload schema.
- **Generic event processing pipeline + projection** — the substrate carries a single backend-event stream (per clarification #3) and fans it out to **two pluggable, user-supplied projections**: a `UIProjection` (`backend event → UI events`, sent to the wire) and a `TimelineProjection` (`backend event → timeline upserts`, persisted into the hydration store). The "+ projection" half of the page-2 bullet is exactly these two.
- **Pluggable command framework** — incoming commands have shape `(Name, Payload, ConnectionId, SessionId)` (clarification #4). The substrate dispatches synchronously to a registered command handler keyed by `Name`. Handlers receive `(ctx, Command, *Session, EmitFunc)` — note the absence of any connection object (clarification #2). The page-2 *"on command (evt, session, connection)"* triple is preserved as the substrate's *internal* call chain shape; the *backend handler* only sees `Session`.
- **Generic hydration / timeline entity storage** — a substrate-owned store of timeline entities, fed by the `TimelineProjection`, queryable so a (re)connecting client can be brought back to current state.
- **Generic timeline events** — a base event vocabulary the substrate itself emits (likely lifecycle / system events).
- **Connection objects** — first-class, *substrate-owned* handles for individual client connections (clarification #2). The substrate uses them for routing UI events back to subscribers; the backend layer does not read or mutate them. Future work places user/identity context (today's `profile-id`) on the connection.

> **Trace — every bullet in this list.** All seven are verbatim (or near-verbatim) page-2 bullets under "Generic Layer": *websocket handling (establish, disconnect)*; *session objects (+ allow backend metadata)*; *generic event processing pipeline + projection*; *pluggable command framework / on command (evt, session, connection)*; *generic hydration / timeline entity storage*; *generic timeline events*; and the marginal *connection objects* annotation. My right-hand explanations after the em-dashes are paraphrase / interpretation.
>
> **Trace — `(evt, session, connection)` handler signature.** Page 2 has this triple in parentheses on the "pluggable command framework" line: *"on command (evt, session, connection)"*. I am quoting that signature directly.
>
> **Trace — "the seam is: generic owns the object, backend owns the payload schema".** This is a paraphrase of the page-2 bullet *"session objects (+ allow backend metadata)"*. The "+" notation says the session object itself is generic, while the metadata slot is for the backend — that's the seam I am naming.

**Reading.** Two design choices stand out:

1. **Connections are substrate-owned and opaque to the backend.** The page-2 marginal note about "managing connection state from the backend side" is narrower than it looks: per clarification #2, the backend does *not* read connection objects. It addresses connections through the substrate (e.g., the substrate routes UI events to the right subscribers automatically), but it does not inspect or mutate connection state directly. This keeps backend logic untangled from transport identity.
2. **Hydration is a first-class storage concern owned by the substrate, not the app.** Apps describe their timeline entity schemas and a `TimelineProjection`; the substrate is responsible for storing, projecting, and replaying them on reconnect.

> **Trace — "Connections are substrate-owned and opaque to the backend".** Page 2 left-margin note: *"this also allows managing connection state from the backend side"* — but per clarification #2, "managing" here means substrate-mediated effects, not direct read/write access. The backend never receives a connection object in its handler signature.
>
> **Trace — "Hydration is a first-class storage concern owned by the substrate".** Page 2 lists *"generic hydration / timeline entity storage"* under the Generic layer, not under the Backend layer. Ownership = which layer the bullet appears under.

### 3.3 Client layer — *application-supplied projections + UI command mapping*

- **UI event schemas** — what the UI's view-layer reasons about.
- **`UIProjection` (backend evt → UI events, +session)** — translation from substrate-carried backend events into UI-shaped events. **User-supplied** but registered on the substrate side per clarification #3, so it can run wherever the substrate decides (typically server-side, just before the websocket fan-out).
- **`TimelineProjection` (backend evt → timeline entities)** — the *second* user-supplied projection, sibling to the UI projection. It populates the substrate's hydration / timeline-entity store. Page 2 names "hydration" and "timeline object schemas" on the client side; the projection that *produces* the entities is the part this section names explicitly.
- **Set of UI commands and how they are mapped to backend** — the UI's outbound vocabulary, declared, and the mapping from each UI command to one or more backend commands.
- **Hydration consumer** — how the client takes a hydration snapshot (timeline-entity list + ordinal cursor) and reconciles its local view-state.
- **Timeline object schemas** — the typed shape of timeline entities as the UI wants to consume them.

> **Trace — every bullet in this list.** Page 2 Client layer bullets are: *UI event schemas*; *processors: backend evt → ui events + session*; *set of UI commands and how they are mapped to backend*; *hydration*; *timeline object schemas*. Per clarification #3 I am splitting the single page-2 *processors* bullet into two — UI projection and timeline projection — because the same backend-event stream feeds both, and both are user-supplied.

**Reading.** "Mirror image" is too strong; the more accurate framing is **two user-supplied projection slots that the framework calls in parallel for every backend event**. The substrate owns the event bus, the fan-out, the storage, and the wire; the application contributes the command handlers (Backend layer) plus the two projections (Client layer). UI commands flow the other direction, mapped at the client and dispatched into the substrate's command framework.

> **Trace — "two user-supplied projection slots called in parallel".** This wording is the structural consequence of clarification #3: *"backend events to frontend events, backend events to timeline entities … coming from both sides"*. Both processors share an input (backend event) and have distinct outputs; running them on the same event is the substrate's job.

### 3.4 Open questions captured in the sketch

> How do we handle stale connections?
> Tick messages from backend and frontend?

These are first-class architectural questions, not implementation TODOs. The natural answer they hint at is a **bidirectional liveness "tick"** at the substrate level (heartbeats both ways, with a configurable stale threshold that promotes a connection to a "to be reaped" state). Resolving this almost certainly belongs in the Generic layer, not in any specific Backend.

> **Trace — the two questions.** Verbatim from the bottom of page 2: *"how do we handle stale connections?"* and *"tick messages from backend and frontend?"*. The author wrote them as questions; I am preserving them as questions.
>
> **Trace — "bidirectional liveness 'tick'" inference.** This is my reading. The author uses the words *"tick messages"* and *"from backend and frontend"* in the same line — both directions are explicitly named. So a bidirectional liveness mechanism is the most direct reading of those two phrases together.
>
> **Trace — "belongs in the Generic layer".** Inferred: stale connections and ticks are about the wire / connection lifecycle, which page 2 already places in the Generic layer (*websocket handling (establish, disconnect)*).

## 4. Cross-cutting architectural themes

Several themes recur across both pages and deserve to be named.

### 4.1 `SessionId` is the universal key

(Per clarification #1, `SessionId` is the canonical name for the page-1 `conv-id` token; the framework exposes only `Session` / `SessionId`.)

Every interesting routing decision in the diagram is a function of `SessionId`:
- which `PROC` does this command go to,
- which sockets get this event broadcast,
- which timeline entities does a hydrating client need,
- which command stream does a "REPLAY" run target.

**Implication.** `SessionId` is part of the substrate's core type, not a backend-defined field. It must be created, validated, and routed by the Generic layer.

> **Trace — "universal key".** The source token `conv` / `conv-id` (= `SessionId` in the framework) appears in: page 1 inbound annotation *"do something (conv-id, profile-id)"*; the per-row labels `conv` in the bottom fan-out; the *"map to conv"* routing label; the bullets *"conv creation"* and *"conv hydration"*; and the page 1 prose statement that one side just emits events keyed by a `conv-id` (verbatim wording in the transcription doc). That recurrence across both prose and diagram is the basis for "universal".
>
> **Trace — the four "function of SessionId" bullets.** Each is sourced: routing-to-PROC ← *"map to conv"* arrow; broadcast ← *Broadcast* arrow + multiple WS endpoints; hydration ← *"conv hydration"* + HYDRATION arrow; REPLAY ← top prose block.
>
> **Trace — "must be created, validated, and routed by the Generic layer".** Inference from page 2's "Generic Layer" bullet *"session objects"* and *"pluggable command framework"* both receiving `(evt, session, connection)` — the substrate is the layer that already touches every command and session, so it is the natural owner of the `SessionId` lifecycle.

### 4.2 Schemas at every seam

The sketches name "event schemas", "timeline object schemas", and "UI event schemas" — three distinct schema vocabularies at three distinct seams (backend↔substrate, substrate↔storage, substrate↔UI). The framework's value proposition is that these schemas are *declared* and the substrate generates / validates / projects against them.

**Implication.** Pick a schema technology early (JSON Schema, protobuf, ts/zod-equivalent) and standardise on it across the three seams. Don't mix.

> **Trace — three schema vocabularies.** Each one is a verbatim page-2 bullet: *"backend event schemas"* (Backend layer), *"timeline object schemas"* (Client layer — and matched on the substrate side by *"timeline entity storage"*), *"UI event schemas"* (Client layer).
>
> **Trace — "JSON Schema, protobuf, ts/zod-equivalent" parenthetical.** This is an *example*, marked as such, and is not in the source. The source only says "schemas" — it does not pick a serialisation.

### 4.3 One backend-event stream, two pluggable projections

Per clarification #3, the substrate exposes **two projection slots**, both user-supplied, that share the backend-event stream as their input:

```
                                ┌── UIProjection       ──► UI events  (to the wire)
backend events  ────►  fan-out ─┤
                                └── TimelineProjection ──► timeline upserts  (to hydration store)
```

The page-2 Backend-layer bullet *"event processors"* is the **command-handling** end of the pipe (commands → backend events). The page-2 Client-layer bullet *"processors: backend evt → ui events"* is *one* of the two projection slots — the substrate adds a sibling slot for timeline upserts on the same axis.

**Implication.** The substrate has three pluggable application-supplied slots, not two:

1. **Command handler** — `(ctx, Command, *Session, EmitFunc) → error` (Backend layer).
2. **`UIProjection`** — `(Event) → []UIEvent` (Client layer; runs substrate-side, emits over wire).
3. **`TimelineProjection`** — `(Event) → []TimelineUpsert` (Client layer; runs substrate-side, persists for hydration).

Each slot's input/output type, scheduling, and error policy is a first-class substrate concept and should be identical regardless of which application supplies the slot.

> **Trace — three slots, not two.** Composing clarification #3 (two projections from a single event stream) with the page-2 Backend-layer bullet *"event processors"* (which is the command-handling slot, distinct from the projection slots). The number "three" is therefore the consequence of the corrected reading, not of the raw sketch.
>
> **Trace — "runs substrate-side".** Inferred. The user said projections come "from both sides" (i.e., user-supplied) but the substrate has to *invoke* them on every backend event before either persistence or wire-fan-out — which is only possible if they execute under the substrate's control. They are *contributed by* the application but *executed within* the Generic layer.

### 4.4 Decoupling command issuance from event consumption

The top sub-diagram's "no need for persistent connection" is the architectural prerequisite for REPLAY, batch jobs, scheduled work, and any non-interactive use case. The substrate must let a caller post a command without subscribing, and let a (different) client subscribe to the resulting events without ever having posted.

**Implication.** Don't model commands as websocket frames. Model them as substrate-level objects with a transport-neutral entry point. The websocket is just one way in.

> **Trace — "no need for persistent connection" prerequisite.** That phrase is verbatim on page 1 under the top sub-diagram. I am calling it a *prerequisite* because if you required a persistent connection, you could not run REPLAY (page 1 prose) as a non-interactive job.
>
> **Trace — "batch jobs, scheduled work, and any non-interactive use case".** *Inference* — these examples are not in the source. They are illustrative extensions of the same "no persistent connection" property. Marked as such.

### 4.5 Hydration and storage are part of the framework

Hydration shows up in both pages, as a substrate responsibility on page 2 and as a curving arrow on page 1. The `X`-marked DB cylinder on page 1 is a **negative architectural assertion**: the author has at least one storage shape they explicitly do not want hydration to use.

**Implication.** Define the hydration store interface inside the Generic layer; let backends supply storage implementations but not bypass the projection model. Capture *what the X store is* as a follow-up question (see open questions below).

> **Trace — "hydration shows up in both pages".** Page 1: bullet *"conv hydration"* in the right column, plus the curving arrow labelled *HYDRATION* sweeping toward DB cylinders at the bottom. Page 2: Generic-layer bullet *"generic hydration / timeline entity storage"* and Client-layer bullet *"hydration"*. Three separate occurrences across the two pages.
>
> **Trace — "negative architectural assertion".** This is my interpretation of the X mark next to one DB cylinder on page 1. The author drew an X on a storage glyph; the most natural reading is "do not put state here". I am explicitly flagging this as a reading, and I have also captured "what is the X store" as Open Question §7.4 below to avoid over-committing.

## 5. Layered model (synthesis, post-clarifications)

```
+--------------------------------------------------------------------------+
| Backend layer (per-app: chat, agent, scraper, document writer, …)        |
|   - event schemas        (e.g. TokensDelta, TaskCompleted, DocumentWritten)
|   - command set          (e.g. StartInference, RunAgent, ReplayScrape)   |
|   - session metadata     (profile, token count, task id, …)              |
|   - command handlers     (cmd, *Session, EmitFunc) -> error              |
|       NOTE: handler signature receives Session only, NOT Connection      |
+-----------------------------^--------------------------------------------+
                              |  registers handlers + projections,
                              |  declares schemas, attaches Session metadata
+-----------------------------v--------------------------------------------+
| Generic layer (the substrate)                                            |
|   transport:    websocket lifecycle (establish, disconnect, ticks*)      |
|   identity:     ConnectionId (substrate-owned, opaque to backend),       |
|                 Session objects (+ backend metadata slot)                |
|   routing:      SessionId-keyed dispatch, broadcast, multiplexing        |
|   commands:     synchronous dispatch of (Name, Payload, ConnId, SessId)  |
|                 to a registered command handler keyed by Name            |
|   pipeline:     single backend-event stream, fanned out to TWO           |
|                 user-supplied projections (UI + Timeline)                |
|   storage:      hydration / timeline-entity store, fed by                |
|                 TimelineProjection                                       |
|   events:       generic timeline events (system/lifecycle)               |
|   liveness:     stale-connection handling, ticks (OPEN QUESTION)         |
+-----------------------------^--------------------------------------------+
                              |  UI events out, UI commands in
+-----------------------------v--------------------------------------------+
| Client layer (per-app — but slots execute substrate-side)                |
|   - UIProjection        (Event)  -> []UIEvent          [substrate-side]  |
|   - TimelineProjection  (Event)  -> []TimelineUpsert   [substrate-side]  |
|   - UI event schemas    (consumed by the front-end UI)                   |
|   - UI command set + mapping to backend commands                         |
|   - hydration consumer  (snapshot -> view-state reconciliation)          |
|   - timeline object schemas                                              |
+--------------------------------------------------------------------------+
```

Two perpendicular flows cross every layer:

- **Commands flow down** (UI → Generic → Backend handler), entering either through a websocket frame *or* through the `SRC → PKG` non-persistent path. The backend handler receives `(ctx, Command, *Session, EmitFunc)` — no connection.
- **Backend events flow up** (Backend → Generic), where the substrate runs the two registered projections **in parallel for every event**: `UIProjection` produces UI events that fan out to all subscribed connections for that `SessionId`; `TimelineProjection` produces timeline upserts that the substrate writes into the hydration store.

> **Trace — "Commands flow down".** Page 2 Backend layer says *"set of commands (start inference, stop inference)"* — these are commands the backend *receives*. Page 2 Client layer says *"set of UI commands and how they are mapped to backend"* — UI commands are commands the UI *sends*. So commands originate at the UI and arrive at the Backend; the Generic layer is the only thing in between (page 2's three-layer outline). Hence "flow down".
>
> **Trace — "Events flow up".** Symmetric: Backend layer has *"event processors"* (which emit events); Client layer has *"processors: backend evt → ui events"* (which consume them). So events originate at the Backend and arrive at the UI.
>
> **Trace — "broadcast across all connections subscribed to the originating conv-id".** Justified by page 1's *Broadcast* arrow leaving SRV plus the multi-WS-endpoint cluster on the right (multiple sockets, one event source).
>
> **Trace — "persisted into the hydration store on the way through".** Page 1 has the curving HYDRATION arrow going from the right-side cluster down toward DB cylinders; page 2 has *"generic hydration / timeline entity storage"*. Combined: events are projected into the timeline-entity store as they pass.

## 6. Implementation seams (post-clarifications)

Eight seams suggest themselves as the first concrete interfaces to nail down. None of these are in the sketch verbatim; they are the natural API shape that follows from the corrected layer model. Names use Go camelCase/PascalCase per author preference.

1. **`SessionId`** — typed identifier; the universal routing key. Owned by the substrate.
2. **`ConnectionId`** — typed identifier for a wire-level connection. Substrate-owned and opaque to backend handlers.
3. **`Command`** — `{ Name string; Payload proto.Message; ConnectionId; SessionId }`. Transport-agnostic; **synchronously** dispatched against a handler registered under `Name`.
4. **`Event`** — `{ Name string; Payload proto.Message; SessionId; Ordinal uint64 }`. Monotonic per-`SessionId` ordinal so projections and hydration can resume.
5. **`Session`** — substrate-owned container `{ Id SessionId; Metadata any }`. The `Metadata` slot is the seam for backend-defined typed session shape (profile, tokens, task id, …).
6. **`CommandHandler`** — `func(ctx, Command, *Session, EmitFunc) error`. **No connection in the signature** (clarification #2). `EmitFunc` is the only outbound channel: the handler emits backend events, never UI events directly.
7. **`UIProjection`** + **`TimelineProjection`** — the two pluggable projection slots (clarification #3). Roughly:
    - `UIProjection.Project(Event) ([]UIEvent, error)`
    - `TimelineProjection.Project(Event) ([]TimelineUpsert, error)`
8. **`HydrationStore`** — `Apply(SessionId, ord, []TimelineUpsert)` on the write side; `Snapshot(SessionId, asOf) Snapshot` on the read side. Snapshots carry a list of typed `TimelineEntity` plus the ordinal cursor a client should resume from.

> **Trace — all eight seams are not verbatim.** Stated explicitly in the section preamble.
>
> **Trace — `Command` shape `{ Name, Payload, ConnectionId, SessionId }`.** Verbatim from clarification #4: *"incoming commands will be cmd, args, connection-id, session-id"*. `cmd+args` becomes `Name+Payload` per the author's later note that "cmd, args can say Payload" (consolidating the two fields under one typed message).
>
> **Trace — `Session.Metadata` slot.** Page 2 Generic layer: *"session objects (+ allow backend metadata)"*. The `+` is the typed slot.
>
> **Trace — `CommandHandler` excludes connection.** Per clarification #2 the backend does not read connections; therefore the handler signature does not receive one. The page-2 *"on command (evt, session, connection)"* triple is the substrate's *internal* dispatch context, not the handler's surface.
>
> **Trace — two projection slots.** Verbatim per clarification #3: *"backend events to frontend events, backend events to timeline entities"*. Each becomes one slot.
>
> **Trace — `Event.Ordinal` for resume.** Inference; the sketches do not name an ordinal. Justified because hydration must say "bring me up to current from where I left off", which requires a totally-ordered cursor per session.

## 7. Open questions (pulled forward from the sketches)

Not just the two written ones — every place a sketch left intent ambiguous becomes a deliberate open question to resolve before coding.

1. **Stale connections** *(written on page 2)* — what is the substrate's liveness contract? Heartbeats, timeouts, reaper?
2. **Tick messages** *(written on page 2)* — bidirectional? Substrate-driven or backend-driven? Same channel as events or separate?
3. **Where does the per-conv state machine live?** *(page 1, "+ state machine (up to PROC, guess)")* — at `PROC`, at `conv`, or distributed?
4. **What is the `X`-marked storage path on page 1?** — the author rejected one DB store for hydration; the rejection is informative and should be captured before it is forgotten.
5. **What do `DD` and `DO` mean on the per-conv arrows?** — directionality? Document id? Capture the original intent now.
6. **What is the small donut/circle near the top pipeline?** — token/ack/cache marker? Decide whether it is a substrate concept or a sketch artifact.
7. **Is the `[?Common?] CLIENTS (also general)` aggregator a substrate concept or a per-app one?** — i.e., does the substrate ship a generic client-fanout object, or does each app build its own?
8. **REPLAY semantics** *(prose at the top of page 1)* — does REPLAY re-issue commands, replay events, or both? What is replayed against — historical storage or a deterministic processor?
9. **`SessionId` allocation** — substrate-generated, client-supplied, or backend-supplied? (Affects idempotency of `do something` / command-issuance calls.) *(Sharpened post-clarification: the field is renamed but the allocation question is unchanged.)*
10. ~~Multi-tenancy via `profile-id`~~ → **resolved per clarification #4**: `profile-id` is *not* a command-level field; it is connection-bound user/identity context that will live on the connection object as future work. The remaining narrower question is: **when does the connection's user-info population happen** — on connect, on first authenticated command, or lazily on demand? And: **what hooks does the substrate expose to the application for that population step**, given that the backend cannot read the connection directly?

> **Trace — each of the ten questions.** Q1 and Q2 are verbatim from page 2 bottom (*"how do we handle stale connections?"*, *"tick messages from backend and frontend?"*). Q3 sources directly the page 1 annotation *"+ state machine (up to PROC, guess)"* — the word "guess" is what makes it an open question. Q4 sources the X mark on the page 1 DB cylinder. Q5 sources the unresolved `DD`/`DO` two-letter labels in the page 1 conv rows (also flagged as unresolved in the transcription glyph key). Q6 sources the unlabelled donut/circle near the top SRC→PKG pipeline. Q7 sources the partially-illegible *"[?Common?] CLIENTS (also general)"* label on page 1 right column. Q8 sources the REPLAY prose at the top of page 1. Q9 sources the `conv-id` token in *"do something (conv-id, profile-id)"* — the author writes the field but never says who allocates it. Q10 sources `profile-id` from the same annotation — appears once, role unspecified.

## 8. Recommended first cuts

Read of the sketches if implementation were to start tomorrow:

1. **Pin the type model first.** `Command`, `Event`, `Session`, `Connection`, `conv-id`, `profile-id`. Nothing else can be designed until these exist.
2. **Build the Generic layer against an in-memory transport before introducing websockets.** The whole multiplexing / broadcast / hydration story should work with a fake transport so it can be tested deterministically — websockets are then a thin adapter.
3. **Make the non-persistent command path (top sub-diagram) the *first* transport, not the second.** It is the simpler shape, it forces the substrate to truly decouple commands from event delivery, and it unlocks REPLAY immediately.
4. **Defer storage choice; commit to the `HydrationStore` interface.** Ship an in-memory implementation first; capture the `X`-marked rejection as a constraint on the eventual real implementation.
5. **Resolve the liveness question (stale connections + ticks) before exposing the websocket transport publicly** — it is a contract, not an optimisation, and changing it later is expensive.

> **Trace — these are recommendations, not source content.** I am explicitly framing this section as "if implementation were to start tomorrow". Each item nonetheless leans on a sketch fact:
> - #1 lists the six types from §6, which were derived from sketch labels (see §6 traces).
> - #2 leans on the page-2 Generic-layer bullets that are all about *what* happens, not *where over the wire* — fits an in-memory test substrate.
> - #3 leans on the *"no need for persistent connection"* annotation on page 1 — it is the simpler transport because the sketch already says so.
> - #4 leans on the X cylinder (Open Question §7.4) — until that is resolved, only the interface should be committed to.
> - #5 leans on the bottom-of-page-2 questions (Q1/Q2) — calling these *contracts* rather than *optimisations* is my framing.

## 9. What this analysis deliberately does not do

- It does not prescribe a language, a runtime, or specific libraries. The sketches do not commit to any.
- It does not adopt or contradict any existing implementation in the repository — by directive, none was read.
- It does not flesh out wire protocols, on-the-wire framing, or schema serialisation choices. Those depend on decisions in §7 that the sketches leave open.
- It does not name files to create or refactor. That is downstream of resolving the open questions and choosing the type model.

> **Trace — "the sketches do not commit to any" language/runtime.** Verified: no language name appears anywhere on either page.
> **Trace — "by directive, none was read".** Sources the user-stated clean-room directive that scoped this work to the two images only.

## 10. Related

- Source transcription (verbatim): `../reference/01-source-image-transcription-2026-04-19-sketches.md`
- Source images: `../sources/diagram-page-1.png`, `../sources/building-blocks-page-2.png`
