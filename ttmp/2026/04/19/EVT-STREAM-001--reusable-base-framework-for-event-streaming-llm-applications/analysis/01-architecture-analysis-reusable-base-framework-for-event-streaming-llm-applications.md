---
title: "Architecture Analysis: Reusable Base Framework for Event Streaming LLM Applications"
status: active
doc-type: analysis
intent: long-term
topics:
  - llm
  - event-streaming
  - agents
  - architecture
  - framework
  - websocket
  - session-management
owners: []
date: 2026-04-19
ticket: EVT-STREAM-001
---

# Architecture Analysis: Reusable Base Framework for Event Streaming LLM Applications

**Ticket:** EVT-STREAM-001  
**Date:** 2026-04-19  
**Status:** Active — Analysis Phase

---

## 1. Executive Summary

This document defines the architecture for a **reusable, layered framework** that handles real-time event streaming between LLM-powered backends and client applications (agents, chats, interactive UIs). The framework is decomposed into three distinct layers — **Backend**, **Generic**, and **Client** — with a clear separation of concerns that allows different backends and UIs to be plugged in without touching the core machinery.

> [!NOTE]
> **Claim:** "Three distinct layers — Backend, Generic, and Client."  
> **Reasoning:** I wrote this because in the "BUILDING BLOCKS" diagram, I see three top-level bullet points clearly labeled `• backend layer (for example llm chat)`, `• Generic Layer`, and `• client layer`. These are presented as sibling categories, not nested, so I inferred they are equal peers in the architecture. The document title "BUILDING BLOCKS" reinforced this as an architectural scaffolding composed of distinct parts.
> **Source:** BUILDING BLOCKS diagram, three main bullets under the header.

The core insight is that LLM applications share a common structure: they emit structured events over a session, support commands (start/stop inference), manage hydration and timeline state, and need to multiplex multiple conversations over shared connections. The Generic Layer is the reusable heart of this system.

> [!NOTE]
> **Claim:** "The Generic Layer is the reusable heart of this system."  
> **Reasoning:** I wrote this because the three-layer model implies that the Backend and Client layers are use-case specific (LLM backend vs. web/CLI client), while the middle layer is the only component that has no direct dependency on either. I also inferred it from the explicit statement in the second sketch: *"it's basically realtime websocket structured widget handling + hydration, with one side just emitting events with a conversation id and the rest managed by this intermediate layer."* — "this intermediate layer" is the Generic Layer, and "one side just emitting events" is the Backend Layer. The Generic Layer is what remains when you remove the LLM-specific and UI-specific parts.
> **Source:** Second sketch, second paragraph; BUILDING BLOCKS diagram, Generic Layer section.

---

## 2. Problem Statement and Scope

### 2.1 The Problem

Every new LLM chat or agent project reinvents the same plumbing:

- WebSocket connection management and lifecycle
- Session creation, tracking, and cleanup
- Event schema definitions for both backend and UI
- Hydration of UI state from persisted conversation data
- Timeline entity storage and event replay
- Command dispatch (start inference, stop inference, cancel)
- Stale connection handling and heartbeat/tick mechanisms

> [!NOTE]
> **Claim:** "Every new LLM chat or agent project reinvents the same plumbing."  
> **Reasoning:** I wrote this as a framing statement because the two sketches appear to be preparatory notes for designing a reusable framework. The date (2026.04.19) and the forward-looking language ("then the cleaned up webchat packages and maybe build a couple different backends") suggest the author has just done an LLM chat project and is now extracting the common patterns into a shared foundation. This is an inference about context and intent, not a direct transcription.
> **Source:** Second sketch, second paragraph ("then the cleaned up webchat packages and maybe build a couple different backends").

> [!NOTE]
> **Claim:** "Stale connection handling and heartbeat/tick mechanisms" as a shared problem.  
> **Reasoning:** I wrote this because at the bottom of the BUILDING BLOCKS diagram, the author explicitly wrote two questions: *"how do we handle stale connections?"* and *"tick messages from backend and frontend?"* These appear at the end as open questions — meaning the author identified these as unresolved concerns that the framework needs to address.
> **Source:** BUILDING BLOCKS diagram, bottom two lines.

### 2.2 Scope of This Framework

**In scope:**
- Three-layer architecture: Backend → Generic → Client
- WebSocket transport with connection pooling
- Session and conversation management
- Event schemas and typed event processing pipeline
- Pluggable command framework with `on_command(evt, session, connection)` semantics
- Hydration: restoring UI state from persisted conversation entities
- Timeline entity storage and event replay
- Conversation multiplexing over shared connections
- Stale connection detection and recovery
- Backend-to-frontend tick/heartbeat messages

**Out of scope:**
- Specific LLM provider integrations (those are backend implementations)
- UI component libraries (those are client implementations)
- Persistence technology choices (plug in your preferred store)
- Authentication and authorization (handled by the surrounding application)

> [!NOTE]
> **Claim:** "Conversation multiplexing over shared connections" is in scope.  
> **Reasoning:** I wrote this because the second sketch shows three shaded oval/cloud shapes labeled "ws" (websockets) with a feature list next to them that includes `conv multi-plexing` and `conv creation` and `conv hydration` and `cancellation multiplex`. The word "multiplex" appearing four times in the feature list, and the fact that the three "ws" shapes share the same PROC output, made it clear that multiplexing is a core, explicitly named capability.
> **Source:** Second sketch, system architecture diagram, feature list next to the three shaded oval ws shapes.

> [!NOTE]
> **Claim:** "Backend-to-frontend tick/heartbeat messages" is in scope.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram explicitly asks *"tick messages from backend and frontend?"* as an open question at the bottom. The word "tick" appears in this question, and "messages from backend and frontend" implies bidirectional communication. This is the only mention of a heartbeat-like mechanism in either sketch.
> **Source:** BUILDING BLOCKS diagram, bottom line: "tick messages from backend and frontend?"

---

## 3. Architectural Layers

### 3.1 Layer 1 — Backend Layer

The Backend Layer is the **LLM execution engine**. It owns the actual work: running inference, streaming tokens, managing model state. Multiple backends can coexist (e.g., OpenAI chat backend, Anthropic backend, local inference backend).

**Responsibilities:**
- Define backend event schemas (e.g., `InferenceStarted`, `TokenStream`, `InferenceComplete`, `Error`)
- Expose a set of commands: `start_inference`, `stop_inference`
- Maintain session metadata: profile, token count, model configuration, cost tracking
- Run event processors that transform internal LLM events into the framework's event stream

**Key constraints:**
- Backends emit events only; they do not know about connections, sessions, or multiplexing
- Backends receive commands via the Generic Layer's pluggable command framework
- Session metadata flows upward into the Generic Layer for sharing with clients

```
Backend Layer
├── backend_event_schemas       (InferenceStarted, TokenStream, ...)
├── command_definitions          (start_inference, stop_inference)
├── session_metadata             (profile, token_count, model, cost)
└── event_processors             (backend → generic event transformation)
```

> [!NOTE]
> **Claim:** The Backend Layer "owns the actual work: running inference, streaming tokens, managing model state."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram explicitly labels this layer `• backend layer (for example llm chat)`. The phrase "for example llm chat" tells me this layer is the LLM-specific component — the thing that actually runs the model. The second sketch also shows "SRC." (the base layer) sending `do something (conv-id) profile-id` to PKG, implying the base layer receives an instruction and does work. "SRC." is the same as the backend layer.
> **Source:** BUILDING BLOCKS diagram, first bullet: "backend layer (for example llm chat)"; second sketch, top diagram: "SRC." box with arrow labeled "do something (conv-id) profile-id."

> [!NOTE]
> **Claim:** "Backend event schemas" are a responsibility of the Backend Layer.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ backend event schemas` as a direct sub-bullet under `• backend layer`. The word "schemas" implies typed event definitions. I inferred `InferenceStarted`, `TokenStream`, `InferenceComplete`, `Error` as examples because these are the natural events an LLM backend would emit.
> **Source:** BUILDING BLOCKS diagram, second bullet under backend layer: "backend event schemas."

> [!NOTE]
> **Claim:** The Backend Layer exposes commands `start_inference` and `stop_inference`.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ set of commands (start inference, stop inference)` as a sub-bullet under the backend layer. These are the only two commands explicitly named, suggesting they are the primary command interface for controlling LLM execution.
> **Source:** BUILDING BLOCKS diagram, third bullet under backend layer: "set of commands (start inference, stop inference)."

> [!NOTE]
> **Claim:** The Backend Layer maintains session metadata: "profile, token count, model configuration, cost tracking."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ session metadata` with an arrow pointing to `(profile, token count, etc.)`. The "(etc.)" implies there are additional metadata fields beyond those explicitly written. I added "model configuration" and "cost tracking" as plausible LLM-specific metadata fields that would logically accompany profile and token count.
> **Source:** BUILDING BLOCKS diagram, fourth bullet under backend layer: "session metadata" with annotation "(profile, token count, etc.)."

> [!NOTE]
> **Claim:** The Backend Layer runs "event processors" that transform internal LLM events into the framework's event stream.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ event processors` as a sub-bullet under the backend layer, and separately lists `◦ generic event processing pipeline + projection` under the Generic Layer. This implies the backend has its own processing step that feeds into the generic pipeline. The arrow direction (backend → generic) is implied by the layered structure.
> **Source:** BUILDING BLOCKS diagram, fifth bullet under backend layer: "event processors."

> [!NOTE]
> **Claim:** "Backends emit events only; they do not know about connections, sessions, or multiplexing."  
> **Reasoning:** I wrote this as a constraint because the second sketch shows the Base Layer (SRC.) receiving only `do something (conv-id) profile-id` — a single trigger with an ID — and the annotation "no need for persistent connection." This suggests the Base Layer is stateless relative to the connection layer. It receives a conversation ID and emits events into the system; it doesn't manage connections. The absence of connection or multiplexing terms under the Backend Layer bullets reinforced this.
> **Source:** Second sketch, top diagram annotation: "no need for persistent connection."

---

### 3.2 Layer 2 — Generic Layer (The Reusable Core)

The Generic Layer is **the framework proper** — the part that is shared across all backends and all clients. It handles the transport, session lifecycle, event routing, and hydration. No LLM-specific logic lives here.

**Responsibilities:**

1. **WebSocket handling**  
   - Establish and teardown connections
   - Produce connection objects with metadata
   - Manage connection lifecycle events (connect, disconnect, reconnect)

2. **Session objects**  
   - Created per conversation or per user session
   - Allow backend metadata to be attached (from Layer 1)
   - Serve as the primary routing key for events

3. **Generic event processing pipeline**  
   - Receives typed events from any backend
   - Runs a projection step: transforms backend events into UI-facing events
   - Handles back-pressure and buffering

4. **Pluggable command framework**  
   - Signature: `on_command(evt, session, connection)`
   - Backends register command handlers
   - Commands flow from clients → Generic Layer → Backend Layer
   - Examples: `start_inference`, `stop_inference`, `cancel`, `reset`

5. **Generic hydration / timeline entity storage**  
   - Store conversation entities (messages, tool calls, results)
   - Provide replay capability for restoring UI state
   - Decouple persistence from the event stream

6. **Connection state management from the backend side**  
   - The Generic Layer can push state changes back to backends (e.g., "client disconnected, pause streaming")
   - Enables backpressure and graceful degradation

7. **Stale connection handling**  
   - Detect inactive connections
   - Optionally close, pool, or notify backends

8. **Tick / heartbeat messages**  
   - Bidirectional: backend → frontend and frontend → backend
   - Used for keep-alive, progress updates, and latency probing

```
Generic Layer
├── websocket_handling           (connect, disconnect, reconnect)
├── session_objects              (+ backend metadata attachment)
├── event_processing_pipeline    (+ projection step)
├── pluggable_command_framework  (on_command(evt, session, connection))
├── hydration_storage             (timeline entities, replay)
├── connection_state_management   (backend-side control)
├── stale_connection_detection
└── tick/heartbeat_protocol
```

> [!NOTE]
> **Claim:** The Generic Layer handles "WebSocket handling (establish, disconnect) → connection objects."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ websocket handling (establish, disconnect)` as a sub-bullet under Generic Layer, with an arrow pointing to `connection objects`. This tells me that establishing and disconnecting websockets produces connection objects as first-class entities in this layer.
> **Source:** BUILDING BLOCKS diagram, second bullet under Generic Layer: "websocket handling (establish, disconnect)" with annotation arrow to "connection objects."

> [!NOTE]
> **Claim:** Session objects "allow backend metadata to be attached (from Layer 1)."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram explicitly says `◦ session objects (+ allow backend metadata)`. The parentheses and "allow" strongly imply that session objects are designed to accept and carry backend-specific data — a form of composition or extension.
> **Source:** BUILDING BLOCKS diagram, third bullet under Generic Layer: "session objects (+ allow backend metadata)."

> [!NOTE]
> **Claim:** The Generic Layer has "generic event processing pipeline + projection."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ generic event processing pipeline + projection` as a sub-bullet under Generic Layer. The word "projection" is notable — it implies a transformation step where backend events are projected into a different shape (likely UI-facing events). This connects to the Client Layer bullet `processors: backend evt + session -> ui events`, which is the other side of the same projection.
> **Source:** BUILDING BLOCKS diagram, fourth bullet under Generic Layer: "generic event processing pipeline + projection."

> [!NOTE]
> **Claim:** The pluggable command framework has signature `on_command(evt, session, connection)`.  
> **Reasoning:** I wrote this exact signature because the BUILDING BLOCKS diagram explicitly says `on_command (evt, session, connection)` as an annotation next to "pluggable command framework." This three-parameter signature was written by the author, making it a first-class API contract.
> **Source:** BUILDING BLOCKS diagram, annotation for "pluggable command framework": "on_command (evt, session, connection)."

> [!NOTE]
> **Claim:** The Generic Layer supports "generic hydration / timeline entity storage" and "generic timeline events."  
> **Reasoning:** I wrote both because the BUILDING BLOCKS diagram lists `◦ generic hydration / timeline entity storage` and `◦ generic timeline events` as two separate sub-bullets under Generic Layer. These are distinct concepts: hydration is the act of restoring state from persisted entities; timeline events are the entities themselves. The second sketch reinforces this with a dedicated HYDRATION cylinder and "conv hydration" in the CMDS feature list.
> **Source:** BUILDING BLOCKS diagram, sixth and seventh bullets under Generic Layer: "generic hydration / timeline entity storage" and "generic timeline events."

> [!NOTE]
> **Claim:** "This also allows managing connection state from the backend side."  
> **Reasoning:** I wrote this as a specific capability of the Generic Layer because the BUILDING BLOCKS diagram has a left-side annotation block that reads: *"this also allows managing connection state from the backend side"*, with an arrow pointing to the Generic Layer bullets. This is a direct quote, not an inference — the author was explicitly calling out that the Generic Layer gives the backend side visibility and control over connection state.
> **Source:** BUILDING BLOCKS diagram, left annotation block next to Generic Layer section: "this also allows managing connection state from the backend side."

> [!NOTE]
> **Claim:** "Stale connection handling" is a responsibility of the Generic Layer.  
> **Reasoning:** I wrote this as a responsibility because the BUILDING BLOCKS diagram asks "how do we handle stale connections?" at the bottom. This question appears at the same level as the three layers, suggesting it's a cross-cutting concern handled by the framework — i.e., the Generic Layer. I placed it as a Generic Layer responsibility because stale connection detection operates at the transport/connection layer, not at the backend or client layer.
> **Source:** BUILDING BLOCKS diagram, bottom question: "how do we handle stale connections?"

---

### 3.3 Layer 3 — Client Layer

The Client Layer is the **UI or agent application**. It receives processed events from the Generic Layer and renders them. It also translates user actions into commands.

**Responsibilities:**
- Define UI event schemas (e.g., `MessageRendered`, `ToolCallInFlight`, `TypingIndicator`)
- Define UI commands and map them to backend commands
- Process backend events + session context → UI events
- Hydrate UI state from timeline entity storage
- Define timeline object schemas for rendering

```
Client Layer
├── ui_event_schemas             (MessageRendered, TypingIndicator, ...)
├── event_processors              (backend_evt + session → ui_evt)
├── command_mappings               (ui_command → backend_command)
├── hydration                      (timeline → UI state)
└── timeline_object_schemas
```

> [!NOTE]
> **Claim:** The Client Layer processes "backend evt + session → ui events."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram explicitly says `◦ processors: backend evt + session -> ui events` as a sub-bullet under the client layer. The arrow direction (`->`) and the two inputs (backend evt + session) are directly from the sketch. This is the client's projection step — the inverse of what the Generic Layer does.
> **Source:** BUILDING BLOCKS diagram, third bullet under client layer: "processors: backend evt + session -> ui events."

> [!NOTE]
> **Claim:** The Client Layer has "ui event schemas" as a responsibility.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ ui event schemas` as a sub-bullet under client layer. This mirrors the Backend Layer's `backend event schemas`, confirming that both ends of the pipeline define their own typed event sets.
> **Source:** BUILDING BLOCKS diagram, second bullet under client layer: "ui event schemas."

> [!NOTE]
> **Claim:** The Client Layer has "set of ui commands and how they are mapped to backend."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ set of ui commands and how they are mapped to backend.` as a sub-bullet under client layer. The word "mapped" is key — it means the client layer doesn't directly execute backend commands; it maps its own UI-level commands to the backend's command names. This connects to the Backend Layer's `start_inference, stop_inference` and the Generic Layer's `on_command` dispatcher.
> **Source:** BUILDING BLOCKS diagram, fourth bullet under client layer: "set of ui commands and how they are mapped to backend."

> [!NOTE]
> **Claim:** The Client Layer has "hydration" as a responsibility.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram lists `◦ hydration` as a sub-bullet under client layer. Combined with the Generic Layer's `generic hydration / timeline entity storage` and the second sketch's HYDRATION cylinder, hydration spans both Generic and Client layers — Generic stores/provides, Client consumes/restores.
> **Source:** BUILDING BLOCKS diagram, fifth bullet under client layer: "hydration."

---

## 4. System Topology

### 4.1 High-Level Data Flow

```
┌──────────────┐     do something (conv-id, profile-id)     ┌──────────────┐
│   BASE LAYER │  ─────────────────────────────────────────▶ │    PKG       │
│  (LLM Backend)│                                        │  (Package/   │
│              │◀─────────────────────────────────────────│   Module)    │
└──────────────┘           events + commands              └──────────────┘
                                                                │
                      ┌─────────────────────────────────────────┘
                      │ events / commands
                      ▼
               ┌──────────────┐
               │     SRV      │  (Dispatch: map conv → storage)
               │  (Service)   │
               └──────┬───────┘
                      │ map to conv
                      ▼
               ┌──────────────┐     ┌──────────────┐
               │  CONV queues │────▶│   PROC       │  (Processors +
               └──────────────┘     │  (clouds)    │   State Machines)
                                    └──────┬───────┘
                                           │
                         ┌─────────────────┼─────────────────┐
                         │                 │                 │
                         ▼                 ▼                 ▼
                    ┌─────────┐       ┌─────────┐       ┌─────────┐
                    │   WS    │       │   WS    │       │   WS    │
                    │ (Client)│       │ (Client)│       │ (Backend│
                    └─────────┘       └─────────┘       │ or more)│
                                                        └─────────┘
                          CMDS (conv multiplexing, creation,
                               hydration, cancellation multiplex)
                          HYDRATION (cylinder — timeline storage)
```

> [!NOTE]
> **Claim:** The top-level flow is: Base Layer (SRC.) → PKG → SRV (Dispatch) → PROC → WS clients.  
> **Reasoning:** I reconstructed this flow from the second sketch's system architecture diagram. The left side shows three queue icons with arrows pointing into "SRV", which has an internal "dispatch" column. SRV's output maps to three stacked "conv" boxes, which feed into three cloud shapes labeled "PROC". PROC outputs via three arrows labeled "ws" to three shaded oval ws shapes. The "CMDS carried through" arrow from SRV to the top SRC./PKG diagram confirms bidirectional flow.
> **Source:** Second sketch, bottom system architecture diagram — spatial reading left-to-right and top-to-bottom.

> [!NOTE]
> **Claim:** "CMDS carried through" SRV.  
> **Reasoning:** I wrote this because the second sketch shows an upward-pointing arrow from the top of the "SRV" box going toward the "SRC."/PKG diagram, labeled `CMDS carried through`. This tells me that the CMDS (multiplexing) component is threaded through SRV — it spans the top diagram (SRC. → PKG) and the bottom diagram (SRV → PROC → WS). CMDS is the unifying concept across both views.
> **Source:** Second sketch, upward arrow from SRV box labeled "CMDS carried through."

> [!NOTE]
> **Claim:** PROC has state machines.  
> **Reasoning:** I wrote this because the second sketch has text next to the PROC clouds: `+ state machine (up to PROC I guess)`. The parenthetical "I guess" tells me the author was uncertain but believed state machines live in or up to PROC. I treated this as a design decision to be confirmed, not a settled fact.
> **Source:** Second sketch, annotation next to PROC clouds: "+ state machine (up to PROC I guess)."

> [!NOTE]
> **Claim:** "(conv but CLIENTS also backend)" — clients and backends are both connected to PROC.  
> **Reasoning:** I wrote this because the second sketch has a parenthetical annotation near the PROC clouds: `(conv but CLIENTS also backend)`. This tells me that the PROC layer's inputs/outputs are not exclusively clients — backends also connect to PROC via websockets. This is a symmetry that the architecture supports.
> **Source:** Second sketch, annotation near PROC: "(conv but CLIENTS also backend)."

> [!NOTE]
> **Claim:** HYDRATION is a cylinder/storage icon with arrows from the PROC area.  
> **Reasoning:** I wrote this because the second sketch shows a cylinder icon (standard database symbol) labeled "HYDRATION" with arrows pointing into it from the PROC area and a curved arrow from the bottom-center pointing toward it. This is the persistence layer — it receives data from PROC and is called HYDRATION, confirming it's the storage mechanism for the hydration feature.
> **Source:** Second sketch, bottom-right cylinder icon labeled "HYDRATION" with arrows from PROC area.

> [!NOTE]
> **Claim:** CMDS feature list: conv multi-plexing, conv creation, conv hydration, cancellation multiplex.  
> **Reasoning:** I wrote this list because the second sketch shows a feature list next to the three shaded oval ws shapes: `• CMDS`, `• conv multi-plexing`, `• conv creation`, `• conv hydration`, `• cancellation multiplex`. Note: the sketch writes "cancellation" on one line and "multiplex" on the next, suggesting they are separate but related concepts — "cancellation multiplex" meaning multiplexed cancellation of specific conversations.
> **Source:** Second sketch, feature list next to shaded oval ws shapes.

---

### 4.2 Key Design Decisions

#### 4.2.1 No Persistent Connection Required for Base Layer → PKG

The base LLM backend does not need a persistent WebSocket to the Generic Layer. It receives a `do something (conv-id, profile-id)` trigger and operates fire-and-forget, emitting events asynchronously. This allows:
- LLM backends to be stateless workers
- Independent scaling of inference vs. transport
- Easy swap of LLM providers

> [!NOTE]
> **Claim:** "No persistent connection required" between Base Layer and PKG.  
> **Reasoning:** I wrote this because the second sketch's top diagram has an annotation written directly below the three shaded squares between SRC. and PKG: `no need for persistent connection`. This is an explicit statement by the author, not an inference. Combined with the fact that the trigger is just `do something (conv-id) profile-id` — a single stateless call — it strongly suggests a fire-and-forget pattern where the base layer sends an event and the PKG/Generic layer handles delivery.
> **Source:** Second sketch, top diagram, annotation below sequence of squares: "no need for persistent connection."

#### 4.2.2 Conversation Multiplexing

A single WebSocket connection can carry multiple conversations simultaneously. The Generic Layer's **CMDS** (Commands/Multiplexing) component handles:
- Conversation creation
- Conversation multiplexing over shared connections
- Conversation hydration (restoring a conversation's state on reconnect)
- Cancellation multiplexing (canceling specific conversations without affecting others)

> [!NOTE]
> **Claim:** A single WebSocket connection can carry multiple conversations simultaneously.  
> **Reasoning:** I wrote this because the second sketch shows three "ws" (websocket) shapes sharing the same output from PROC, and the CMDS feature list explicitly mentions `conv multi-plexing`. If multiplexing is a named feature, it implies that a single connection handles multiple conversations — otherwise multiplexing would not be needed.
> **Source:** Second sketch, three shaded ovals labeled "ws" sharing PROC output; CMDS feature list: "conv multi-plexing."

> [!NOTE]
> **Claim:** CMDS handles "conv creation, conv hydration, cancellation multiplex."  
> **Reasoning:** I wrote these four capabilities because the second sketch's CMDS feature list explicitly lists them as bullet points. These are the four named responsibilities of CMDS, making it the composition operator for all conversation-level operations.
> **Source:** Second sketch, CMDS feature list: "conv creation," "conv hydration," "cancellation multiplex."

#### 4.2.3 State Machines Up to PROC

The processor layer (PROC) maintains per-conversation state machines. This enables:
- Typed progression through inference phases (Idle → Starting → Streaming → Complete → Error)
- Graceful cancellation at any phase
- UI-side animation and progress indicators driven by state transitions

> [!NOTE]
> **Claim:** State machines live "up to PROC."  
> **Reasoning:** I wrote this because the second sketch explicitly says `+ state machine (up to PROC I guess)` next to the PROC clouds. The parenthetical "I guess" signals the author's uncertainty. I preserved the uncertainty by framing it as a design decision to be validated. The state machine concept is new — it doesn't appear in the BUILDING BLOCKS diagram, so I derived it from the second sketch's annotation.
> **Source:** Second sketch, annotation next to PROC: "+ state machine (up to PROC I guess)."

#### 4.2.4 Hydration as a First-Class Concept

Hydration is not an afterthought — it's a dedicated component (HYDRATION) that:
- Reconstructs UI state from persisted timeline entities
- Feeds into the processor layer for state machine restoration
- Supports incremental hydration (load recent, lazy-load history)

> [!NOTE]
> **Claim:** Hydration is a first-class, named concept (HYDRATION cylinder).  
> **Reasoning:** I wrote this because the second sketch gives hydration its own distinct icon — a cylinder (database symbol) with the label "HYDRATION" — positioned separately from PROC and WS. Giving a concept its own icon in an architecture sketch signals that it is a first-class, named component with its own responsibilities, not just a sub-feature of something else.
> **Source:** Second sketch, cylinder icon labeled "HYDRATION" in the bottom-right of the system architecture diagram.

#### 4.2.5 Stale Connections

Open questions flagged during design:
- How to detect and handle stale connections from both client and backend sides?
- Tick/heartbeat protocol design: frequency, timeout, backoff strategy
- Whether to auto-reconnect, pool, or close stale connections

> [!NOTE]
> **Claim:** These are open questions, not settled design decisions.  
> **Reasoning:** I wrote these as open questions because they appear at the bottom of the BUILDING BLOCKS diagram with question marks, in the same style as the three layers. Questions with question marks in an architecture sketch signal unresolved concerns — they are flags for future work, not confirmed design choices.
> **Source:** BUILDING BLOCKS diagram, bottom two lines: "how do we handle stale connections?" and "tick messages from backend and frontend?"

---

## 5. Proposed API Sketches

> [!NOTE]
> **Claim:** The API sketches are creative proposals, not direct transcriptions.  
> **Reasoning:** The second sketch and BUILDING BLOCKS diagram describe components and their responsibilities but do not include TypeScript/interface-level code. I wrote the API sketches as my own interpretation of the sketched components, applying standard software engineering patterns (typed interfaces, pipeline registration, store patterns) to the described concepts. Every interface name (Session, Connection, CommandHandler, etc.) is grounded in the BUILDING BLOCKS bullets, but the specific property names and method signatures are my invention to make the design concrete and actionable. The one exception is `on_command(evt, session, connection)`, which appears verbatim in the sketch.
> **Source:** BUILDING BLOCKS diagram: `on_command (evt, session, connection)`; BUILDING BLOCKS diagram: `session objects`, `connection objects`, `generic hydration / timeline entity storage`, `generic event processing pipeline + projection`; second sketch: HYDRATION cylinder, CMDS feature list.

### 5.1 Generic Layer — Session Object

```typescript
interface Session {
  id: string;                        // conversation or session ID
  profile: ProfileMetadata;           // from backend layer
  connection: Connection;
  backendMetadata: BackendMetadata;   // opaque, backend-defined
  state: SessionState;               // managed by generic layer
}

interface Connection {
  id: string;
  socket: WebSocket;                 // transport handle
  multiplexedConversations: Set<string>;
  lastTick: number;
}
```

> [!NOTE]
> **Claim:** `backendMetadata` field on Session.  
> **Reasoning:** I added this field because the BUILDING BLOCKS diagram says `session objects (+ allow backend metadata)` — meaning session objects must carry backend metadata. The `+` implies aggregation, not substitution of existing fields.
> **Source:** BUILDING BLOCKS diagram: "session objects (+ allow backend metadata)."

> [!NOTE]
> **Claim:** `multiplexedConversations` on Connection.  
> **Reasoning:** I added this field because the CMDS feature list in the second sketch says `conv multi-plexing` and `cancellation multiplex`, and the top diagram shows a single connection (PKG) handling multiple conversations. A Connection object in a multiplexing system must track which conversations it is carrying.
> **Source:** Second sketch, CMDS feature list: "conv multi-plexing," "cancellation multiplex."

> [!NOTE]
> **Claim:** `lastTick` on Connection.  
> **Reasoning:** I added this field because the BUILDING BLOCKS diagram asks "tick messages from backend and frontend?" as an open question. If tick/heartbeat messages are part of the protocol, a Connection object naturally tracks the last tick time for stale connection detection.
> **Source:** BUILDING BLOCKS diagram: "tick messages from backend and frontend?"; second sketch: "no need for persistent connection."

### 5.2 Generic Layer — Pluggable Command Framework

```typescript
type CommandHandler = (
  event: BackendCommandEvent,
  session: Session,
  connection: Connection
) => Promise<void>;

// Registration
genericLayer.registerCommand('start_inference', async (evt, session, conn) => {
  // route to appropriate backend
});

genericLayer.registerCommand('stop_inference', async (evt, session, conn) => {
  // signal backend to cancel
});

genericLayer.registerCommand('cancel', async (evt, session, conn) => {
  // cancellation multiplex: cancel specific conv on shared connection
});
```

> [!NOTE]
> **Claim:** The `cancel` command is a separate capability from `stop_inference`.  
> **Reasoning:** I wrote `cancel` as a distinct command because the CMDS feature list in the second sketch has `cancellation multiplex` as a standalone bullet point. This implies cancellation is not just "stop inference" but a multiplexing concern — the ability to cancel one specific conversation without affecting others sharing the same connection.
> **Source:** Second sketch, CMDS feature list: "cancellation multiplex."

### 5.3 Generic Layer — Event Processing Pipeline

```typescript
type EventProcessor = (backendEvent: BackendEvent, session: Session) => UiEvent | null;

const pipeline = new EventProcessingPipeline();

pipeline.register('token_stream', (evt, session) => ({
  type: 'TokenDelta',
  sessionId: session.id,
  delta: evt.tokens,
  timestamp: Date.now(),
}));

pipeline.register('inference_complete', (evt, session) => ({
  type: 'MessageRendered',
  sessionId: session.id,
  message: evt.fullContent,
  metadata: evt.usage,
}));
```

> [!NOTE]
> **Claim:** "projection" in the event processing pipeline.  
> **Reasoning:** I wrote the word "projection" into the pipeline description because the BUILDING BLOCKS diagram explicitly says `generic event processing pipeline + projection`. Projection means transforming events from one schema to another — in this case, from backend event schemas to UI event schemas. This is confirmed by the Client Layer bullet: `processors: backend evt + session -> ui events` — the same transformation but described from the client side.
> **Source:** BUILDING BLOCKS diagram: "generic event processing pipeline + projection" and "processors: backend evt + session -> ui events."

### 5.4 Hydration API

```typescript
interface HydrationStore {
  // Persist a timeline entity
  persist(entity: TimelineEntity): Promise<void>;

  // Load all entities for a conversation (for full hydration)
  loadConversation(convId: string): Promise<TimelineEntity[]>;

  // Load recent entities (for incremental hydration)
  loadRecent(convId: string, limit: number): Promise<TimelineEntity[]>;

  // Restore a session's state machine from stored entities
  restoreSessionState(convId: string): Promise<SessionState>;
}
```

> [!NOTE]
> **Claim:** "conv hydration" in CMDS implies a hydration API exists.  
> **Reasoning:** I wrote this API sketch because the CMDS feature list in the second sketch explicitly includes `conv hydration` as one of the four CMDS responsibilities. This implies CMDS must have a way to fetch and restore a conversation's state — which requires a hydration store interface. The "generic timeline events" and "generic hydration / timeline entity storage" bullets in BUILDING BLOCKS also support the existence of a timeline entity type and a store.
> **Source:** Second sketch, CMDS feature list: "conv hydration"; BUILDING BLOCKS diagram: "generic hydration / timeline entity storage" and "generic timeline events."

---

## 6. Implementation Phases

> [!NOTE]
> **Claim:** Six-phase implementation plan.  
> **Reasoning:** I derived the six phases from analyzing what components were explicitly named in the sketches and what dependencies exist between them. The BUILDING BLOCKS diagram shows three layers, and the second sketch shows CMDS and HYDRATION as distinct components. I ordered the phases by dependency: you cannot do multiplexing (Phase 4) without sessions and commands (Phase 1); you cannot do full hydration (Phase 5) without the event pipeline (Phase 2/3). This ordering is my inference, not from the sketches.
> **Source:** BUILDING BLOCKS diagram (three layers = ~3 phases); second sketch (CMDS and HYDRATION as named components = 2 more phases); "stale connections" and "tick messages" open questions = hardening phase.

### Phase 1 — Core Generic Layer Skeleton

**Goal:** Establish the minimum viable Generic Layer.

1. Define core interfaces: `Session`, `Connection`, `Event`, `Command`
2. Implement basic WebSocket server (connect/disconnect)
3. Implement session creation and routing by `conv-id`
4. Implement the command registry and `on_command` dispatcher
5. Add a tick/heartbeat loop

**Deliverable:** A working echo pipeline (client sends command → generic processes → emits event → client receives).

> [!NOTE]
> **Claim:** Phase 1 deliverable is an "echo pipeline."  
> **Reasoning:** I wrote this as the Phase 1 deliverable because an echo pipeline tests the full round-trip of the Generic Layer in isolation — command in, event out — without requiring a real LLM backend or a real UI client. It validates that the WebSocket handling, session creation, and command dispatch work together before adding the complexity of actual LLM or UI integration.
> **Source:** BUILDING BLOCKS diagram: "websocket handling (establish, disconnect)", "session objects", "on_command (evt, session, connection)"; second sketch: PKG receives a trigger and routes it.

### Phase 2 — Backend Layer Integration

**Goal:** Wire in the first LLM backend (e.g., the existing pinocchio webchat backend).

1. Define backend event schemas
2. Implement backend event → Generic Layer event projection
3. Implement `start_inference` / `stop_inference` command handlers
4. Attach session metadata (token count, model) from backend to session
5. Verify: real inference runs, events stream to client

> [!NOTE]
> **Claim:** "Profile, token count" as session metadata that flows from backend to session.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram says under Backend Layer: "session metadata" with annotation "(profile, token count, etc.)", and under Generic Layer: "session objects (+ allow backend metadata)". The parenthetical is the backend's contribution to session metadata; the "+ allow" is the generic layer's acceptance of it.
> **Source:** BUILDING BLOCKS diagram: "session metadata" annotation "(profile, token count, etc.)"; "session objects (+ allow backend metadata)."

### Phase 3 — Client Layer Integration

**Goal:** Wire in a client (web UI or CLI).

1. Define UI event schemas
2. Implement backend event + session → UI event processor
3. Implement command mapping (UI action → backend command)
4. Implement hydration: restore UI from timeline storage on reconnect
5. Implement timeline object schemas and renderer integration

> [!NOTE]
> **Claim:** Command mapping is bidirectional: UI action → backend command.  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram says under Client Layer: "set of ui commands and how they are mapped to backend." The phrase "mapped to" implies translation from one namespace (UI commands) to another (backend commands). The Generic Layer's `on_command` is the handler that performs or delegates this mapping.
> **Source:** BUILDING BLOCKS diagram: "set of ui commands and how they are mapped to backend."

### Phase 4 — Conversation Multiplexing

**Goal:** Support multiple simultaneous conversations over one connection.

1. Implement CMDS component: create, multiplex, hydrate, cancel
2. Test cancellation isolation (cancel conv A without affecting conv B)
3. Implement per-conversation state machines in PROC

> [!NOTE]
> **Claim:** Cancellation isolation — canceling one conversation must not affect others.  
> **Reasoning:** I wrote this because the CMDS feature list in the second sketch says both `conv multi-plexing` and `cancellation multiplex` as separate bullets. If cancellation were global (affecting the whole connection), there would be no need for the word "multiplex" after it. The word "multiplex" after "cancellation" implies the ability to target cancellation at a specific conversation within the multiplexed set.
> **Source:** Second sketch, CMDS feature list: "conv multi-plexing," "cancellation multiplex."

### Phase 5 — Persistence and Hydration

**Goal:** Make conversations survive reconnects and restarts.

1. Implement HydrationStore interface (swap in a concrete store)
2. Implement timeline entity persistence
3. Implement incremental hydration (lazy load history)
4. Test full round-trip: create conv → disconnect → reconnect → hydrate → resume

> [!NOTE]
> **Claim:** "Incremental hydration" (lazy load history) is a goal.  
> **Reasoning:** I added "incremental hydration" as a sub-goal because the second sketch shows HYDRATION as a separate cylinder receiving arrows from multiple sources. If full hydration were required on every reconnect, the HYDRATION cylinder would be a bottleneck. The ability to load recent entities (a subset) and lazy-load older history follows from the "timeline entity storage" and "conv hydration" bullets — it's a natural optimization of those concepts.
> **Source:** BUILDING BLOCKS diagram: "generic hydration / timeline entity storage"; second sketch: "conv hydration" in CMDS feature list; HYDRATION cylinder.

### Phase 6 — Hardening

**Goal:** Production readiness.

1. Stale connection detection: configurable timeouts, backoff, reconnect policy
2. Bidirectional tick/heartbeat protocol
3. Error propagation: backend errors → generic layer → UI notifications
4. Back-pressure: pause backend when client connection is slow
5. Load testing: multiple clients, multiple backends

> [!NOTE]
> **Claim:** "Stale connection detection" and "tick/heartbeat" belong in the hardening phase, not Phase 1.  
> **Reasoning:** I placed these in Phase 6 because the BUILDING BLOCKS diagram marks both "how do we handle stale connections?" and "tick messages from backend and frontend?" as open questions (bottom of the diagram, with question marks). Open questions belong in a hardening phase after the core loop is proven. The "no need for persistent connection" annotation in the second sketch suggests the initial version may not need persistent connections at all — tick/heartbeat are optional enhancements.
> **Source:** BUILDING BLOCKS diagram, bottom questions: "how do we handle stale connections?" and "tick messages from backend and frontend?"; second sketch: "no need for persistent connection."

> [!NOTE]
> **Claim:** Back-pressure is a concern.  
> **Reasoning:** I added back-pressure to Phase 6 because the BUILDING BLOCKS diagram mentions "generic event processing pipeline + projection" and the second sketch shows "(conv but CLIENTS also backend)" feeding into PROC. When clients and backends both connect to PROC, a slow client could cause back-pressure on the backend. The Generic Layer's annotation "this also allows managing connection state from the backend side" implies the ability to signal back-pressure to the backend.
> **Source:** BUILDING BLOCKS diagram, left annotation: "this also allows managing connection state from the backend side"; second sketch: "(conv but CLIENTS also backend)."

---

## 7. Testing Strategy

| Phase | Test Type | Scope |
|-------|-----------|-------|
| Phase 1 | Unit | Command registry, session creation, routing |
| Phase 2 | Integration | Backend → Generic Layer event flow |
| Phase 3 | Integration | Generic Layer → Client event rendering |
| Phase 4 | Unit + Integration | Multiplexing, cancellation isolation |
| Phase 5 | Integration | Hydration round-trip, reconnect/resume |
| Phase 6 | Load + Chaos | Stale connections, back-pressure, multi-client |

> [!NOTE]
> **Claim:** "Cancellation isolation" as a specific test case for Phase 4.  
> **Reasoning:** I wrote "cancellation isolation" as a specific Phase 4 test case because the CMDS feature list in the second sketch explicitly names `cancellation multiplex` as a standalone capability. Testing cancellation isolation means: start conversations A and B over the same connection, cancel A, verify B continues uninterrupted. This is the only way to validate that "cancellation multiplex" actually works.
> **Source:** Second sketch, CMDS feature list: "cancellation multiplex."

**Test fixtures to build:**
- Mock LLM backend that emits configurable event sequences
- Mock client that asserts on expected UI events
- Replay fixture: serialize event log, replay through pipeline, assert idempotency

> [!NOTE]
> **Claim:** "Replay fixture: serialize event log, replay through pipeline, assert idempotency."  
> **Reasoning:** I wrote this because the BUILDING BLOCKS diagram mentions "generic timeline events" and "generic hydration / timeline entity storage". A replay fixture validates that the same event log replayed through the pipeline produces the same UI state — which is the foundation of the hydration feature. If replay is not idempotent, hydration will produce different results on each reconnect, breaking the user experience.
> **Source:** BUILDING BLOCKS diagram: "generic timeline events" and "generic hydration / timeline entity storage."

---

## 8. Risks, Alternatives, and Open Questions

### 8.1 Open Questions

1. **Stale connections** — What is the timeout? What happens on timeout (reconnect, pool, close)? Should the backend be notified?
2. **Tick protocol** — Fixed interval or adaptive? Should ticks carry payload (e.g., token count progress)?
3. **Backpressure** — When a client is slow, should the Generic Layer buffer events, drop events, or signal the backend to pause?
4. **Persistence boundary** — Who owns the timeline store? Can multiple Generic Layer instances share one store?
5. **Backend multiplexing** — Can one Generic Layer instance route to multiple backend instances? (For HA/scaling inference workers.)

> [!NOTE]
> **Claim:** Questions 1 and 2 are directly transcribed from the sketches as open questions.  
> **Reasoning:** I wrote questions 1 and 2 verbatim from the BUILDING BLOCKS diagram's bottom two lines. These appear as open questions, not confirmed answers, and belong in the open questions section.
> **Source:** BUILDING BLOCKS diagram, bottom two lines: "how do we handle stale connections?" and "tick messages from backend and frontend?"

> [!NOTE]
> **Claim:** Questions 3, 4, and 5 are my inferences — plausible concerns the framework must address.  
> **Reasoning:** I added questions 3, 4, and 5 because they are natural follow-on concerns that arise from the sketched architecture. Question 3 (backpressure) follows from the "(conv but CLIENTS also backend)" annotation and the Generic Layer's connection state management. Question 4 (persistence boundary) follows from the HYDRATION cylinder and the "generic hydration / timeline entity storage" bullet. Question 5 (backend multiplexing) follows from the phrase "build a couple different backends" in the second sketch and the possibility of scaling inference workers.
> **Source:** Question 3: second sketch "(conv but CLIENTS also backend)"; BUILDING BLOCKS diagram: "allows managing connection state from the backend side." Question 4: second sketch, HYDRATION cylinder; BUILDING BLOCKS diagram: "generic hydration / timeline entity storage." Question 5: second sketch: "maybe build a couple different backends."

### 8.2 Risks

- **Over-engineering the Generic Layer.** The danger of building too much into the reusable core. Mitigation: ship Phase 1 with minimal wiring and add features only when a concrete use case demands them.
- **Event schema drift.** As backends proliferate, event schemas may diverge. Mitigation: define a stable core schema early, use typed schema validation at the pipeline boundary.
- **Hydration complexity.** Replaying a full conversation timeline on reconnect can be slow. Mitigation: incremental hydration + pagination.

> [!NOTE]
> **Claim:** "Event schema drift" is a risk.  
> **Reasoning:** I wrote this as a risk because the BUILDING BLOCKS diagram has `backend event schemas` under Backend Layer and `ui event schemas` under Client Layer — two separate schema spaces. If multiple backends (OpenAI, Anthropic, local) each define their own event schemas, they may diverge over time, making the generic pipeline's projection step increasingly complex. The mitigation (typed schema validation at boundaries) is a standard pattern for this problem.
> **Source:** BUILDING BLOCKS diagram: "backend event schemas" and "ui event schemas" as separate concerns.

> [!NOTE]
> **Claim:** "Hydration complexity" is a risk.  
> **Reasoning:** I wrote this as a risk because the second sketch gives HYDRATION its own dedicated cylinder icon, and the CMDS feature list includes `conv hydration` as a named capability. Hydration that works well is simple; hydration that handles large conversation histories, partial loads, and state machine restoration is complex. I flagged this explicitly so it doesn't become an afterthought.
> **Source:** Second sketch: HYDRATION cylinder; CMDS feature list: "conv hydration."

### 8.3 Alternatives Considered

- **Single-layer approach (everything in one service):** Simpler initially but leads to tight coupling between LLM logic and transport. Rejected in favor of the three-layer model for reusability.
- **REST polling instead of WebSockets:** Lower complexity for clients, but loses real-time streaming and introduces polling lag. Rejected for real-time LLM use cases.
- **GraphQL subscriptions:** A viable transport alternative but adds GraphQL dependency. Could be a future pluggable transport option.

> [!NOTE]
> **Claim:** "Single-layer approach" rejected because the sketches show three distinct layers.  
> **Reasoning:** I wrote this as an alternative because the BUILDING BLOCKS diagram explicitly partitions the system into three layers with distinct responsibilities. A single-layer approach would collapse Backend and Client into the Generic Layer, losing the pluggability that the three-layer model enables. The phrase "build a couple different backends" in the second sketch reinforces that pluggable backends are a goal — which requires layer separation.
> **Source:** BUILDING BLOCKS diagram: three separate layers with different bullet lists; second sketch: "maybe build a couple different backends."

> [!NOTE]
> **Claim:** "REST polling" rejected for real-time LLM use cases.  
> **Reasoning:** I wrote this as an alternative because the second sketch explicitly mentions "real-time websocket" as a defining characteristic ("it's basically realtime websocket structured widget handling + hydration"). The choice of WebSockets over REST is a first-class concern in the sketches. REST polling was rejected implicitly by the WebSocket-first design.
> **Source:** Second sketch, second paragraph: "it's basically realtime websocket structured widget handling + hydration."

---

## 9. References

- Source sketch 1: **"BUILDING BLOCKS"** — three-layer architecture overview. Dated 2026.04.19. Transcribed from: `/tmp/pi-clipboard-30e8760e-3990-4838-be7f-61b95b345d2b.png`
- Source sketch 2: **System topology** — SRC/PKG → queues → SRV → PROC → CMDS → HYDRATION. Dated 2026.04.19. Transcribed from: `/tmp/pi-clipboard-48ef5d9f-b33a-4eaf-8593-3f6db8a28dce.png`
- Existing work: **WEBCHAT-001** — "Refactor pinocchio webchat into a reusable package" (provides the first concrete backend to integrate)
