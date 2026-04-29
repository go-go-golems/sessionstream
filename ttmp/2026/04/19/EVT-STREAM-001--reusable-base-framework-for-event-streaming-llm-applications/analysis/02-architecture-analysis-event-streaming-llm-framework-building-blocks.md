---
title: "Architecture Analysis: Event-Streaming LLM Framework Building Blocks"
ticket: EVT-STREAM-001
doc-type: analysis
status: active
intent: long-term
topics:
  - llm
  - event-streaming
  - agents
  - architecture
  - framework
  - websocket
  - hydration
date: 2026-04-19
---

# Architecture Analysis: Event-Streaming LLM Framework Building Blocks

## Preface — Diagram Sources

This document is derived from two hand-drawn architectural sketches dated `2026.04.19`.

- **Diagram A** — "Building Blocks": a layered outline diagram describing three architectural tiers (Backend Layer, Generic Layer, Client Layer).
- **Diagram B** — "System Flow": a data-flow + topology diagram showing the runtime structure of conversations (`conv`), processes (`PROC`), a server (`SRV`), sources (`SRC`), packages (`PKG`), and WebSocket clients.

Every claim below is annotated with a citation tracing it to a specific element in these diagrams.

---

## 1. Overview

The sketches propose a **three-tier, reusable middleware framework** for building event-streaming LLM applications — covering chatbots, autonomous agents, and other real-time conversational workloads. The core insight is that the *generic plumbing* (WebSockets, session tracking, event pipelines, conversation hydration) can be extracted into a shared layer, leaving only domain logic to be written per application.

> **Citation 1.1** — I wrote "three-tier framework" because Diagram A's header explicitly reads **"BUILDING BLOCKS"** and the content is organized into exactly three major bullet sections labeled: *backend layer*, *Generic Layer*, and *client layer*.

> **Citation 1.2** — I wrote "LLM applications — covering chatbots, autonomous agents" because Diagram A's Backend Layer sub-bullet reads **"backend layer (for example llm chat)"**, giving the LLM chat as the canonical example. Diagram B's top explanatory text reads **"the cleaned up webchat packages and maybe build a couple different backends"**, indicating intent to support multiple backends beyond chat.

> **Citation 1.3** — I wrote "generic plumbing can be extracted into a shared layer" because Diagram A describes a **Generic Layer** sandwiched between the Backend Layer and Client Layer, containing items like *"generic event processing pipeline + projection"* and *"generic hydration / timeline entity storage"*. This is generic by name and by content — none of its sub-items are LLM- or chat-specific.

---

## 2. Diagram A — Building Blocks: Layer-by-Layer Analysis

### 2.1 Backend Layer

The Backend Layer is the domain-specific, application-facing tier. It is responsible for defining what events and commands are meaningful to a particular LLM workload.

**Sub-components identified in the diagram:**

| Sub-component | Diagram text |
|---|---|
| Event schemas | *"backend event schemas"* |
| Command set | *"set of commands (start inference, stop inference)"* |
| Session metadata | *"session metadata (profile, token count, etc.)"* |
| Event processors | *"event processors"* |

> **Citation 2.1.1** — I wrote "event schemas" as a sub-component because Diagram A, Backend Layer, lists **"backend event schemas"** as the first bullet under the backend layer.

> **Citation 2.1.2** — I wrote that commands include "start inference, stop inference" because the diagram explicitly names these two: **"set of commands (start inference, stop inference)"**. These are the archetypal lifecycle control signals for an LLM inference session.

> **Citation 2.1.3** — I wrote "session metadata (profile, token count)" because the diagram lists **"session metadata"** with an arrow pointing to the parenthetical *"(profile, token count, etc.)"*. The arrow indicates that profile and token count are concrete examples of what metadata the session tracks.

> **Citation 2.1.4** — I wrote "event processors" as a sub-component because the diagram ends the Backend Layer section with the bullet **"event processors"**, indicating domain-specific handlers that process backend events.

**Architectural interpretation:** The Backend Layer is intentionally thin — it provides schemas, a small command set, lightweight metadata, and processors. The heavy lifting (routing, storage, connection management) is delegated downward to the Generic Layer. This promotes reuse: a new LLM application only needs to define these four items.

> **Citation 2.1.5** — I wrote the Backend Layer is "intentionally thin" because it has only four sub-bullets, whereas the Generic Layer has six, and the Generic Layer is explicitly positioned as the shared foundation. The note in Diagram A reads **"This also allows managing connection state from the backend side"**, confirming the Generic Layer handles connection state so the Backend Layer need not.

---

### 2.2 Generic Layer

The Generic Layer is the reusable core of the framework. It handles all infrastructure concerns — WebSocket lifecycle, session management, event processing, command dispatch, and timeline storage — in a domain-agnostic way.

**Sub-components identified in the diagram:**

| Sub-component | Diagram text |
|---|---|
| WebSocket handling | *"websocket handling (establish, disconnect)"* → *connection objects* |
| Session objects | *"session objects (+ allow backend metadata)"* |
| Event processing pipeline | *"generic event processing pipeline + projection"* |
| Command framework | *"pluggable command framework / on_command(evt, session, connection)"* |
| Hydration / storage | *"generic hydration / timeline entity storage"* |
| Timeline events | *"generic timeline events"* |

> **Citation 2.2.1** — I wrote "WebSocket handling" because Diagram A, Generic Layer, reads **"websocket handling (establish, disconnect)"** with a curved arrow pointing to the label *"connection objects"*, showing that websocket lifecycle events produce named connection objects consumed by the rest of the layer.

> **Citation 2.2.2** — I wrote "session objects allow backend metadata" because the diagram explicitly reads **"session objects (+ allow backend metadata)"**. The parenthetical indicates the Generic Layer's session concept is extensible — backends can inject their own metadata (matching the Backend Layer's "session metadata" bullet).

> **Citation 2.2.3** — I wrote "generic event processing pipeline + projection" because the diagram uses this exact phrase. The word "projection" is significant: it implies that raw events are projected (transformed, aggregated) into a derived view — a standard CQRS/event-sourcing pattern.

> **Citation 2.2.4** — I wrote "pluggable command framework with `on_command(evt, session, connection)` signature" because the diagram reads **"pluggable command framework"** followed by the indented entry **"on_command(evt, session, connection)"**, which is the handler signature exposed to command implementors. The three arguments (`evt`, `session`, `connection`) are the minimal context needed to handle any command.

> **Citation 2.2.5** — I wrote "generic hydration / timeline entity storage" because the diagram bullet reads exactly this. "Hydration" means reconstituting state from stored events — confirming an event-sourcing approach. "Timeline entity storage" implies entities (conversations, sessions) are stored as ordered sequences of events.

> **Citation 2.2.6** — I wrote "generic timeline events" as a sub-component because the diagram ends the Generic Layer section with this bullet, indicating a base event type or schema that all timeline entries conform to.

> **Citation 2.2.7** — I wrote "This also allows managing connection state from the backend side" as a key design note because these are the exact words written in a side annotation in Diagram A, with a long arrow pointing into the Generic Layer. This indicates that exposing connection objects to the backend side is a deliberate design affordance, not a side effect.

**Open questions from the diagram:**

The diagram asks two questions at the bottom:

> *"how do we handle stale connections?"*
> *"tick messages from backend and frontend?"*

> **Citation 2.2.8** — I included these as open questions because they appear verbatim at the bottom of Diagram A, below the three main layers, written without a bullet prefix — indicating they are active design questions not yet resolved in the sketches.

These questions point to two cross-cutting concerns: (a) connection liveness / heartbeating, and (b) a tick/clock mechanism to drive time-sensitive processing from either side.

---

### 2.3 Client Layer

The Client Layer sits atop the Generic Layer from the client's perspective, translating backend events into UI-consumable events and mapping UI commands back to the backend command set.

**Sub-components identified in the diagram:**

| Sub-component | Diagram text |
|---|---|
| UI event schemas | *"UI event schemas"* |
| Event processors | *"processors: backend evt + session -> UI events"* |
| UI command mapping | *"set of UI commands and how they are mapped to backend"* |
| Hydration | *"hydration"* |
| Timeline object schemas | *"timeline object schemas"* |

> **Citation 2.3.1** — I wrote "UI event schemas" as a Client Layer sub-component because the diagram lists **"UI event schemas"** as the first bullet under the client layer.

> **Citation 2.3.2** — I wrote that processors transform `backend evt + session → UI events` because the diagram shows exactly this mapping: **"processors: backend evt + session -> UI events"**. The `->` arrow notation makes the transformation direction explicit. The inclusion of `session` as input means processors can enrich UI events with session state (e.g., user profile, accumulated token count).

> **Citation 2.3.3** — I wrote "set of UI commands and how they are mapped to backend" because the diagram reads **"set of UI commands and how they are mapped to backend"**. This confirms the client layer is not just consuming events — it also originates commands, and the mapping from UI intent to backend command is an explicit Client Layer concern.

> **Citation 2.3.4** — I wrote "hydration" as a Client Layer concern because the diagram lists it as a separate bullet under the client layer. This is the client-side reconstitution of timeline state into display objects, distinct from the Generic Layer's server-side timeline entity storage.

> **Citation 2.3.5** — I wrote "timeline object schemas" because the diagram ends the Client Layer with **"timeline object schemas"**, indicating that the client has its own schema for timeline entries (separate from the Generic Layer's generic timeline events). This separation allows the client to have a richer, UI-specific representation.

---

## 3. Diagram B — System Flow: Runtime Topology

Diagram B shows how the building blocks from Diagram A manifest at runtime. It is a data-flow / topology diagram showing sources, a server, conversation instances, processors, storage, and WebSocket clients.

### 3.1 Source and Package Ingestion (Top Diagram)

A `SRC.` (source) box emits data — tagged with `(conv-id) profile-id` — into a pipeline of hatched-rectangle "packages" that are forwarded to a `PKG.` (package) box. A note reads *"no need for persistent connection"*.

> **Citation 3.1.1** — I wrote that sources tag data with `(conv-id) profile-id` because the diagram shows an arrow leaving the `SRC.` box labeled **"do something (conv-id) profile-id."** — indicating that every unit of work from a source is attributed to a conversation ID and a profile ID.

> **Citation 3.1.2** — I wrote "no need for persistent connection" as a design property of the source path because this exact phrase appears below the hatched-rectangle packages in the top diagram. This implies sources can be stateless or short-lived (e.g., HTTP POST, batch job) rather than requiring a long-lived WebSocket.

> **Citation 3.1.3** — I wrote the top diagram represents "source and package ingestion" because it contains the boxes `SRC.` and `PKG.` with a flow between them. The explanatory text in Diagram B reads **"REPLAPI will allow me to trace and reproduce individual scraper steps and refine the program, and then run scraper using all the collected data for real"**, suggesting that sources may include scrapers or external data feeds, not just LLM inference streams.

### 3.2 Server and Dispatch (Bottom Diagram)

A central `SRV` (server) box receives input from multiple upstream queues (shown as three small queue icons with arrows pointing in). From the server, a vertical `dispatch` bracket fans out to three stacked `conv` (conversation) boxes.

> **Citation 3.2.1** — I wrote "server receives input from multiple upstream queues" because the diagram shows three small rectangular icons with vertical lines (queue symbols) each sending an arrow into the `SRV` box.

> **Citation 3.2.2** — I wrote "dispatch fans out to three stacked `conv` boxes" because immediately to the right of `SRV` is a vertical bracket labeled **"dispatch"**, with three boxes labeled **"conv"** stacked below it. The bracket represents the server routing incoming events to the correct conversation handler.

> **Citation 3.2.3** — I wrote "commands are carried through" as a bidirectional flow because the diagram shows an arrow pointing from `SRV` back toward the top `SRC.` diagram, labeled **"CMDS carried through"**. This indicates commands originating from the server (or from clients via the server) propagate back to the source level.

### 3.3 Conversation Processors and State Machines

Each `conv` box connects via an arrow labeled `ID` to a cloud-shaped `PROC` (processor) node. A note below the bottom processor reads *"+ state machine (up to PROC I guess)"*.

> **Citation 3.3.1** — I wrote "each conversation connects to a processor" because the diagram shows horizontal arrows labeled `ID` going from each `conv` box to a corresponding `PROC` cloud. The `ID` label indicates the conversation ID is passed to the processor, keeping the processor stateless by identity-keying.

> **Citation 3.3.2** — I wrote "processors implement state machines" because the diagram note reads **"+ state machine (up to PROC I guess)"**, indicating that each PROC is responsible for maintaining the state machine for its conversation. The qualifier "up to PROC I guess" suggests this is still a design decision in flux.

### 3.4 Storage: Conversation Map and Hydration

Two cylinder (database) shapes appear: one associated with a `map to conv` arrow from the `conv` area, and one labeled `HYDRATION` on the far right. A large curved arrow connects the first cylinder toward the HYDRATION cylinder (with an "X" mark near its base, possibly indicating a planned but not-yet-decided path).

> **Citation 3.4.1** — I wrote "a conversation map store" because the diagram shows a cylinder with the label **"map to conv"** pointing to it, indicating a lookup table that maps conversation IDs to their state or handlers.

> **Citation 3.4.2** — I wrote "a separate HYDRATION store" because the diagram labels the right cylinder **"HYDRATION"**. The explanatory text in Diagram B reads **"it's basically realtime websocket structured widget handling + hydration with one side just emitting events with a conversation id and the rest managed by this intermediate layer"** — confirming hydration is a first-class concept, where the server reconstructs conversation state from stored events before dispatching to clients.

> **Citation 3.4.3** — I noted the large curved arrow has an "X" near its start because this visual element is explicitly present in the diagram and suggests uncertainty: either the direct path between stores is unresolved, or it represents an error case.

### 3.5 WebSocket Client Multiplexing

On the right side, three dark circles represent client endpoints. Three arrows labeled `WS` point from a large circular area labeled `(conv but CLIENTS also backend)` toward these clients. A `CMDS` box also feeds into the clients.

A four-item bullet list describes the responsibilities of this section:

1. `conn multi-plexing`
2. `conv creation`
3. `conv hydration`
4. `cancellat. multiplex`

> **Citation 3.5.1** — I wrote "WebSocket multiplexing" as a core responsibility because the diagram bullet reads **"* conn multi-plexing"** and shows three `WS` arrows fanning out to three client endpoints from a single conversation area.

> **Citation 3.5.2** — I wrote "conversation creation and hydration at the client boundary" because the diagram explicitly lists **"* conv creation"** and **"* conv hydration"** as responsibilities of the WebSocket layer. Hydration here means providing a new connecting client with the accumulated conversation state.

> **Citation 3.5.3** — I wrote "cancellation multiplexing" because the diagram lists **"* cancellat. multiplex"** — indicating that when an inference or operation is cancelled, the cancellation signal must be fanned out to all multiplexed connections (or processes).

> **Citation 3.5.4** — I wrote "clients also includes backends" because the large circle area in the diagram is labeled **"(conv but CLIENTS also backend)"**. This is a significant design note: the WebSocket layer is not exclusively for UI clients — backend services can also connect as "clients" of a conversation stream, enabling pipeline chaining.

> **Citation 3.5.5** — I wrote that commands feed into the client boundary because the diagram shows a `CMDS` box with an arrow pointing into the same group of client circles. This corresponds to the Client Layer's *"set of UI commands and how they are mapped to backend"* from Diagram A.

---

## 4. Cross-Cutting Concerns

### 4.1 Event Sourcing and Hydration as Core Patterns

Both diagrams converge on **event sourcing** as the fundamental storage and recovery pattern. Events are the source of truth; state is derived by replaying or projecting them.

> **Citation 4.1.1** — I wrote "event sourcing as the fundamental pattern" because Diagram A names **"generic hydration / timeline entity storage"** and **"generic timeline events"** as core Generic Layer primitives. Diagram B names a `HYDRATION` cylinder as the final storage endpoint. Both "hydration" (state reconstitution from events) and "timeline" (ordered event sequence) are hallmarks of event sourcing.

### 4.2 Conversation as the Unit of Identity

The `conv-id` appears throughout Diagram B as the primary key that threads through all layers: from source tagging, through server dispatch, into processor state machines, into the conversation store, and out to hydrated clients.

> **Citation 4.2.1** — I wrote "conversation is the unit of identity" because `conv-id` appears in four distinct places in Diagram B: in the source label **"do something (conv-id) profile-id."**, in the `conv` boxes in the dispatch section, in the `map to conv` storage label, and implicitly in the `ID` labels on the `conv → PROC` arrows.

### 4.3 The Intermediate Layer Principle

The explanatory text in Diagram B makes the core architectural philosophy explicit:

> *"it's basically realtime websocket structured widget handling + hydration with one side just emitting events with a conversation id and the rest managed by this intermediate layer."*

> **Citation 4.3.1** — I wrote "intermediate layer principle" and quoted the above because this sentence appears verbatim in the top explanatory text of Diagram B. It is the clearest statement of intent in either diagram: producers (LLM backends, scrapers) only need to emit events with a `conv-id`; all the complexity of routing, storing, hydrating, and multiplexing is absorbed by the intermediate (Generic) layer.

### 4.4 Separation of Source Connection from Client Connection

A key design property is that sources (`SRC.`) do not need a persistent WebSocket connection, while clients do. This decouples ingestion latency from client connectivity.

> **Citation 4.4.1** — I wrote "sources do not need a persistent connection" because the top diagram in Diagram B labels the source-to-package path with **"no need for persistent connection."** In contrast, the bottom diagram shows three persistent `WS` (WebSocket) arrows to clients, confirming that only the client side is persistently connected.

---

## 5. Proposed Package Structure

Based on the three-layer architecture, the framework naturally maps to three Go packages (or TypeScript modules):

| Package | Corresponds to | Key exports |
|---|---|---|
| `pkg/generic` | Generic Layer | `Session`, `Connection`, `Pipeline`, `CommandRouter`, `Timeline`, `HydrationStore` |
| `pkg/backend/<name>` | Backend Layer | `EventSchema`, `Commands`, `SessionMetadata`, `EventProcessor` |
| `pkg/client` | Client Layer | `UIEventSchema`, `UICommandMapper`, `ClientHydration`, `TimelineObjectSchema` |

> **Citation 5.1** — I proposed this package split because Diagram A's three layers (Backend, Generic, Client) are described as self-contained with explicit interfaces between them (`on_command(evt, session, connection)` bridging Generic↔Backend; `backend evt + session -> UI events` bridging Generic↔Client). These interface signatures make natural package boundaries.

---

## 6. Open Questions and Next Steps

From both diagrams, the following questions remain unresolved:

1. **Stale connections** — How does the framework detect and clean up stale WebSocket connections? (Diagram A, bottom note: *"how do we handle stale connections?"*)
2. **Tick messages** — Is there a heartbeat or clock tick from backend and/or frontend to drive time-sensitive operations? (Diagram A, bottom note: *"tick messages from backend and frontend?"*)
3. **State machine ownership** — Should the PROC layer fully own conversation state machines, or is there shared responsibility with the `conv` dispatcher? (Diagram B note: *"+ state machine (up to PROC I guess)"*)
4. **Cancellation multiplexing** — How is a cancellation (inference stop, session abort) fanned out to multiple multiplexed connections? (Diagram B bullet: *"* cancellat. multiplex"*)
5. **The X-marked path** — The large curved arrow with an "X" between the conversation store and the HYDRATION cylinder in Diagram B is unresolved. Does this represent an error path, an alternative path, or a rejected design?

> **Citation 6.1** — I listed questions 1 and 2 because they appear verbatim as open questions at the bottom of Diagram A.
> **Citation 6.2** — I listed question 3 because Diagram B notes **"+ state machine (up to PROC I guess)"** — the qualifier *"I guess"* signals uncertainty.
> **Citation 6.3** — I listed question 4 because **"* cancellat. multiplex"** appears in Diagram B's WS client responsibility list with no further elaboration.
> **Citation 6.4** — I listed question 5 because the "X" on the large curved arrow in Diagram B is an unexplained visual element that may indicate a design decision still pending.

---

## 7. Summary

The two diagrams together sketch a clean, layered, event-sourced framework for real-time LLM applications:

- **Backend Layer** = domain-specific schemas, commands, metadata, processors.
- **Generic Layer** = all infrastructure: WebSockets, sessions, event pipelines, command routing, hydration storage, timeline events.
- **Client Layer** = UI translation: schema adaptation, command mapping, client-side hydration.
- **Runtime Topology** = sources emit tagged events (no persistent connection needed), a server dispatches by `conv-id`, processors run state machines, a hydration store reconstitutes state, and a WebSocket multiplex layer delivers events to any number of clients or backend subscribers.

The central innovation is the **Generic Layer as a conversation-aware event bus with built-in hydration** — making it trivial to attach new LLM backends or new client UIs without rewriting the plumbing.
