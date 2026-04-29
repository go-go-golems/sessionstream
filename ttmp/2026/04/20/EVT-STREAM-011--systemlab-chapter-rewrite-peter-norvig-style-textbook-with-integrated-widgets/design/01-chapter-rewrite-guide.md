---
Title: "Chapter Rewrite Guide: Systemlab Textbook in Peter Norvig Style"
Ticket: EVT-STREAM-011
Status: active
Topics:
    - documentation
    - systemlab
    - teaching
    - textbook-writing
    - onboarding
    - widgets
    - react
DocType: design-doc
Intent: medium-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-0-foundations.md
      Note: Current Phase 0 (690 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-1-command-to-projection.md
      Note: Current Phase 1 (698 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-2-ordering-and-ordinals.md
      Note: Current Phase 2 (582 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-3-hydration-and-reconnect.md
      Note: Current Phase 3 (410 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-4-chat-example.md
      Note: Current Phase 4 (447 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-5-persistence-and-restart.md
      Note: Current Phase 5 (388 lines)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-010--systemlab-textbook-rewrite-interactive-phase-pages-with-sidepanes-and-reusable-widgets/design/01-systemlab-textbook-design-guide.md
      Note: Widget taxonomy and layout design from EVT-STREAM-010
    - Path: /home/manuel/.pi/agent/skills/textbook-authoring/SKILL.md
      Note: Peter Norvig style guide with examples of good/bad writing
ExternalSources: []
Summary: "Detailed guide for rewriting Systemlab chapters in Peter Norvig style with integrated interactive widgets. Covers chapter structure, prose style, widget integration points, and phase-specific rewrite plans for all 6 phases."
LastUpdated: 2026-04-20T10:00:00-04:00
WhatFor: "Guide the full rewrite of Systemlab chapter content from AI-slop to Peter Norvig style textbook, with embedded widget integration."
WhenToUse: "When executing the chapter rewrite for EVT-STREAM-011."
---

# Chapter Rewrite Guide: Systemlab Textbook in Peter Norvig Style

## Overview

This guide describes how to rewrite all 6 Systemlab chapters from their current state (AI-slop prose, vertical stacking, disconnected controls) into a proper textbook in Peter Norvig style with integrated interactive widgets.

The goal is a textbook that:
1. Builds foundational understanding before implementation
2. Explains **why** things are, not just what or how
3. Uses concrete examples, pseudocode, and diagrams to ground abstract concepts
4. Balances prose paragraphs with bullet points for rhythm and emphasis
5. Integrates widgets that both illustrate the text and enable freeform exploration

---

## Part I: Chapter Structure Template

### The Universal Chapter Template

Each chapter follows this structure:

```
## N. [Section Title]

[Opening paragraph - state purpose, what reader should understand]

[Conceptual foundation - why this design exists, what problem it solves]

[Code or pseudocode - grounded in actual implementation]

[Tables or diagrams - comparisons, sequences, architectures]

[Try section with inline controls - targeted widget integration]

[Key points bullet list - surface important takeaways]

### N.1 [Subsection]

[Continue pattern...]

### N.2 [Subsection]

[Continue pattern...]

---

## N+1. [Next Section]

[Continue...]

---

## Key Points

- [Bullet 1: complete sentence]
- [Bullet 2: complete sentence]
- [Bullet 3: complete sentence]
```

### Why This Structure Works

The opening paragraph sets intent, not a table of contents. The reader knows what they will understand by the end.

The conceptual foundation builds the mental model before showing code. The reader grasps the "why" before the "how."

Code is grounded in actual implementation, not abstract sketches. The reader can run the examples.

Tables and diagrams do work that prose cannot—comparing approaches, showing sequences, visualizing architecture.

Try sections integrate controls near the explanation. The reader doesn't scroll from concept to action.

Key points surface the most important takeaways without introducing new information.

### Widget Integration Points

There are three integration patterns:

**1. Inline in Try Sections (Targeted)**
```
### 1.2 What handlers publish and why

[Prose explanation...]

┌─ Try: Submit a command ─────────────────────────────────────────────┐
│  Session: [__________]   Prompt: [________________________]          │
│  [Submit]                                    [Reset]                │
└──────────────────────────────────────────────────────────────────────┘

[What to look for...]
```

The widget is directly adjacent to the explanation. The reader acts immediately after understanding.

**2. Evidence Panel (Targeted + Freeform)**
```
┌─ Trace Timeline ─────────────────────────────────────────────────┐
│  ① Command received: LabStart          [JSON] [Rendered]          │
│  ② Session created or retrieved                                 │
│  ③ Handler invoked                                               │
│  ...                                                             │
└──────────────────────────────────────────────────────────────────┘
```

Evidence is visible without scrolling. JSON/Rendered toggle available.

**3. Sidepane (Freeform)**
```
┌──────────────────────────────────────────────────────────────────┐
│  Chapter prose...                                                │
│                                                                   │
├────────────────────────────────┬──────────────────────────────────┤
│                                │  SIDE PANE                       │
│  Main content                  │  ┌─ Freeform ─────────────────┐ │
│                                │  │  Session Explorer           │ │
│                                │  │  [Create arbitrary session] │ │
│                                │  └─────────────────────────────┘ │
│                                │  ┌─ Evidence Log ──────────────┐ │
│                                │  │  [Full event capture]       │ │
│                                │  └─────────────────────────────┘ │
└────────────────────────────────┴──────────────────────────────────┘
```

Freeform widgets are accessible but don't compete with the textbook flow.

---

## Part II: Prose Style Guide

### The Peter Norvig Standard

**Opening paragraphs**: State the purpose and what the reader should understand. Not a summary—a statement of intent.

**Good:**
> "When you click **Submit** on the Phase 1 page, a series of things happen in sequence. Understanding that sequence—not just what occurs, but why each step exists and how it connects to the next—is the purpose of this chapter. The goal is not to make you memorize an API, but to help you see the architecture as a coherent system of responsibilities, each playing a specific role."

**Bad:**
> "In this chapter, we will explore the command path in detail, looking at various aspects of handlers, events, and projections."

The good version states the goal. The bad version summarizes contents.

---

**Conceptual explanations**: Build the mental model. Explain why a design exists before showing how it works.

**Good:**
> "Why does the handler receive a publisher rather than returning a result? This is the most important design decision in the framework, and it pays to sit with it for a moment. If the handler returned a final value—say, a `UIState` or a `ChatMessage`—then the framework would have to know what to do with that value. It would have to route it somewhere, decide how to transform it, decide whether to store it. That knowledge would live in the handler, or in the framework, or in both in ways that would be hard to untangle. Instead, the handler describes what happened by publishing events."

**Bad:**
> "The handler publishes events rather than returning values. This is because events are the primary mechanism in the framework."

The good version explains the reasoning. The bad version states a fact without justification.

---

**Bullet lists**: Each bullet is a complete sentence that could stand alone. Bullets summarize, they don't introduce new information.

**Good:**
> The key points to internalize:
> - Sessions are created lazily. If the `SessionId` has never been seen before, a fresh session is instantiated.
> - Handlers publish events, not return values. The handler describes what happened; the framework decides what to do next.
> - Events are canonical. They describe backend state, not frontend state.

**Bad:**
> Key concepts:
> - Sessions
> - Events
> - Projections
> - Handlers

The good version is informative. The bad version is a glossary.

---

**Code blocks**: Show real implementation, not abstract sketches. Use meaningful variable names. Include comments that explain the why, not the what.

---

### Anti-Patterns to Eliminate

| Pattern | Problem | Fix |
|---------|---------|-----|
| "In this chapter, we will..." | Wastes time | State the purpose directly |
| "It is worth noting that..." | Hedged, says nothing | Be direct |
| "One might observe that..." | Passive, weak | "The handler does..." |
| "This is one of those patterns..." | Philosophical filler | Show the code |
| "Clearly, of course, it goes without saying" | Unnecessary qualifiers | Just state it |
| Fragment sentences in bullets | Incomplete thoughts | Write complete sentences |
| "As you can see..." | Condescending | Just show it |
| Walls of text with no breaks | Exhausts reader | Use code, tables, bullets |

---

## Part III: Phase-Specific Rewrite Plans

### Phase 0: Foundations (690 lines → ~500 lines)

**Current State**: Explains architecture, package structure, vocabulary. Dense with bullet lists. Prose is functional but flat.

**Target Style**: Peter Norvig - foundational, explains why architecture decisions were made.

**Core Teaching Goal**: Help the reader understand what evtstream is trying to become and why Systemlab is a separate app.

**New Structure**:

```
## 1. What this framework is trying to do

[Opening: state the goal - realtime event-streaming substrate]

The story this framework is trying to tell:
- A client sends a command
- Backend work happens
- Canonical events describe what happened
- Those events drive both live UI updates AND durable state
- Reconnecting clients can recover cleanly

[Diagram: Command → Handler → Events → UIProjection + TimelineProjection]

## 2. Why we created pkg/evtstream instead of renaming pkg/webchat

[Explain donor code vs clean-room abstraction]

The healthy mental model:
  pkg/webchat → donor and later consumer/example
  pkg/evtstream → reusable substrate

[Explain why this distinction matters for architecture]

## 3. Why Systemlab is a separate app

[Explain boundary rule - framework boundary is not proven until something outside uses it]

Systemlab may:
- Import public evtstream APIs
- Expose its own HTTP endpoints
- Render labs and explanations
- Simulate consumers

Systemlab may not:
- Import pkg/webchat internals
- Bypass public seams
- Redefine framework ideas in lab-specific ways

## 4. The vocabulary Phase 0 teaches

[For each term: explain what it is, why it exists, what problem it solves]

- SessionId: the universal routing key
- ConnectionId: transport-level identity (separate from SessionId)
- Command: typed request entering the framework
- Event: canonical backend event
- UIProjection: transforms events into client-facing messages
- TimelineProjection: transforms events into persistent state
- HydrationStore: the persistence seam

## 5. The import-cycle lesson

[Real example of why concrete implementations must depend on interfaces, not vice versa]

Correct dependency shape:
  evtstream defines HydrationStore interface
  evtstream/hydration/memory implements it
  callers inject implementations with options

## 6. The directory map and what it means

[Show the tree, explain what each part owns]

## 7. Validation commands and why they matter

[make systemlab-run, make evtstream-test, etc.]

## Key Points

- Canonical events sit in the middle of the architecture
- UI output and durable state are both derived from the same event stream
- Systemlab exists to prove the framework boundary is real
- Concrete implementations depend on interfaces; interfaces do not depend on implementations
```

**Widget Integration**:
- **StatusIndicator** (Core): Shows which phases are implemented - illustrates the progression
- **No freeform widgets**: Phase 0 is foundations only; no exploration needed yet

**Estimated Output**: ~500 lines (30% reduction from 690)

---

### Phase 1: Command → Event → Projection (698 lines → ~600 lines)

**Current State**: Explains Hub, handlers, projections, hydration store. Good structure but prose is flat. Tries to be comprehensive.

**Target Style**: Peter Norvig - grounded in code, explains the path through the system.

**Core Teaching Goal**: Show the full path from command to events to projections. Make canonical events feel real.

**New Structure**:

```
## 1. The Command Path

When you click **Submit** on the Phase 1 page, a series of things happen in sequence. Understanding that sequence—not just what occurs, but why each step exists and how it connects to the next—is the purpose of this chapter.

The entry point is the `Hub`. It sits at the center of the framework, and its job is deceptively simple: receive a command, find the right handler, give the handler what it needs, and let the rest unfold.

[Show Hub.Submit() pseudocode]

The Hub does not know what commands do. It does not format output. It does not decide what state should look like. It routes. Everything else flows from that decision.

### 1.1 Submit hits the Hub

[Explain the three things Hub does - lookup, get/create session, call handler]

Key points:
- Handlers are registered by name, not discovered at runtime
- Sessions are created lazily - no separate "create session" step
- Handler receives command, session, AND a publisher

### 1.2 Why handlers publish events, not return results

[This is the key conceptual paragraph - explain why]

If handlers returned values, the framework would have to know what to do with them. That knowledge would become tangled. Instead, the handler describes what happened by publishing events, and the framework asks: what should happen next?

### 1.3 What LabHandler publishes

[Show real handler code]

Three events, not one:
- LabStarted: work has begun
- LabChunk: (multiple) intermediate results
- LabFinished: completion

[Explain why this mirrors real inference behavior]

## 2. Ordinals: why order matters

Between the handler and the projections sits the publisher. Its job: give each event a sequential ordinal.

[Show the flow: Handler → Publisher → Projections → Store]

Why here, not in the handler?
- Handler shouldn't have to track state to know what number to use
- Publisher ensures consistent numbering
- Ordinals define order, enable reconnect, support hydration

Three purposes of ordinals:
1. Define event order
2. Enable clients to resume ("I've seen up to ordinal 5")
3. Support the hydration store (tracks latest ordinal per session)

## 3. Two projections, one source, different jobs

[This is the key conceptual section]

UIProjection asks: what should a live client see right now?
TimelineProjection asks: what persistent state should the system remember?

[Show both projection interfaces]

[Table: UIProjection vs TimelineProjection]

The key insight: both consume the same canonical event stream. Neither is more important. They answer different questions.

### 3.1 What UIProjection produces

[Explain UI events - transient, client-facing, optimized for live delivery]

LabStarted → LabMessageStarted
LabChunk → LabMessageAppended
LabFinished → LabMessageFinished

### 3.2 What TimelineProjection produces

[Explain timeline entities - store-facing, stateful, support hydration]

LabMessage entity (updated incrementally, finalized on LabFinished)

## 4. The in-memory hydration store

Even though Phase 1 only uses an in-memory store, it teaches the durable-state shape.

[Show HydrationStore interface: Apply, Snapshot, View, Cursor]

How to think about each:
- Apply: update the store and advance the cursor
- Snapshot: show current session state in serialized form
- View: give projections a read-only view
- Cursor: tell me the latest applied ordinal

This is the earliest visible form of the hydration story.

## 5. Reading the page

[Explain the page layout - not the controls, the reading flow]

Controls → Checks → Trace → Session + UI Events → Snapshot

When you use the page properly, you mentally travel from left to right and top to bottom:
input → checks → internal story → live-facing output → durable-ish state

That is a very good way to debug event-streaming systems generally.

## 6. Things to try

┌─ Try 1: the happy path ───────────────────────────────────────────┐
│  Session: [lab-session-1]   Prompt: [hello from systemlab]        │
│  [Submit]                                              [Reset]     │
└────────────────────────────────────────────────────────────────────┘

What should happen:
- Session created
- One handler invocation
- Multiple trace entries
- Multiple UI events
- Final snapshot with finished LabMessage

What to pay attention to:
- The command is singular; the event story is plural. This is one of the first truly important lessons.

┌─ Try 2: run again with the same session ──────────────────────────┐
│  [Change the prompt, keep Session: lab-session-1]                 │
└────────────────────────────────────────────────────────────────────┘

What to pay attention to:
- Session metadata remains stable
- Event and timeline outputs evolve
- SessionId is routing/stability, not decoration

┌─ Try 3: different session id ───────────────────────────────────────┐
│  [Use lab-session-2]                                                │
└────────────────────────────────────────────────────────────────────┘

What to pay attention to:
- Fresh session path created
- SessionId is what tells the framework where the work belongs

┌─ Try 4: longer prompt ──────────────────────────────────────────────┐
│  [Use: explain why projections should consume canonical events]     │
└────────────────────────────────────────────────────────────────────┘

What to pay attention to:
- Longer accumulated text
- More meaningful chunk progression
- Final snapshot reflects accumulated result

┌─ Try 5: click Reset ────────────────────────────────────────────────┐
│  [After a run, click Reset]                                         │
└────────────────────────────────────────────────────────────────────┘

What to pay attention to:
- Outputs clear, environment returns to fresh state
- Reset isolates one scenario from another

┌─ Try 6: export the transcript ─────────────────────────────────────┐
│  [Export JSON] [Export Markdown]                                    │
└────────────────────────────────────────────────────────────────────┘

What to pay attention to:
- Exports are teaching material
- System is designed to make its own behavior portable and inspectable

## 7. Reading the trace

[Walk through actual trace output from a run]

[Show real JSON trace]

Step by step:
1. command submitted
2. session created
3. handler invoked
4. UI projection emitted event
5. timeline projection updated entity
... (repeat for chunks)
... (finish)

The trace shows that the framework is not a glorified reducer that jumps straight from command to final snapshot. It shows the path from command to event stream to views. The trace is where the architecture becomes visible.

## 8. What the checks are trying to summarize

[Explain each check - what it validates, why it matters]

- sessionExists: session was found or created
- cursorAdvanced: hydration store cursor moved forward
- timelineProduced: timeline projection emitted durable state
- uiEventsProduced: UI projection emitted live-facing events

Each check points at a different part of the architecture. If one goes red later, you immediately know which subsystem to investigate first.

## Key Points

- The Hub routes; it does not know what commands do
- Handlers publish events, not return results
- One command produces multiple events (beginning, intermediate, end)
- Ordinals define order, enable reconnect, support hydration
- Both projections consume the same canonical event stream
- The trace shows the architecture; read it carefully

## API Reference

[Show the key APIs with brief explanations - not exhaustive, just the ones that matter for Phase 1]

- Hub.Submit(...)
- RegisterCommand(...)
- RegisterUIProjection(...)
- RegisterTimelineProjection(...)
- HydrationStore.Apply(...)
- HydrationStore.Snapshot(...)
- HydrationStore.View(...)
- HydrationStore.Cursor(...)

You do not need to memorize their signatures. You do need to understand their roles.
```

**Widget Integration**:
- **ScenarioControls** (Targeted): Inline in Try sections
- **TraceTimeline** (Targeted): Shows trace with step numbers
- **ToggledViewer** (Core): JSON/Rendered for UI Events and Snapshot
- **CheckList** (Targeted): Validates invariants
- **SessionExplorer** (Freeform): Create/inspect arbitrary sessions
- **EvidenceLog** (Freeform): Full event capture for exploration

**Estimated Output**: ~600 lines (15% reduction from 698)

---

### Phase 2: Ordering and Ordinals (582 lines → ~500 lines)

**Current State**: Explains Watermill bus, consumer-side ordinal assignment, ordering experiments. Good content, flat prose.

**Target Style**: Peter Norvig - grounded in the bus/consumer model, explains why ordering is non-negotiable.

**Core Teaching Goal**: Show that ordering is foundation for everything that follows (reconnect, hydration, durability).

**New Structure**:

```
## 1. Why ordering is a framework problem, not just an implementation detail

[Explain - if events arrive out of order, state becomes incoherent]

When a client subscribes and receives events, they expect them in order. If a client receives a "message finished" event before seeing the start, the state is wrong. If the hydration store applies events out of order, the snapshot is wrong.

This means ordering cannot be an afterthought. It must be built into the architecture from the start.

## 2. The Watermill bus boundary

[Show where the bus fits in the architecture]

Commands → Handler → Publisher → Watermill Bus → Consumer → Projections → Store

The bus is the boundary where events enter the durable stream. Before the bus, events are local to the handler's call stack. After the bus, events are in the system's canonical stream.

[Explain why this boundary matters - separates "event published" from "event consumed"]

## 3. Consumer-side ordinal assignment

[This is the key conceptual section]

The consumer, not the handler, assigns ordinals. Why?

1. The handler publishes events as they occur, without knowing what came before
2. The consumer sees the full stream and can assign sequential numbers
3. If the handler assigned ordinals, it would need to query the store first - coupling, complexity

[Show the flow: Handler publishes → Bus delivers → Consumer assigns ordinal → Consumer dispatches to projections]

The consumer is the authoritative source for ordinals because it sees the full sequence.

## 4. What the consumer does

[Show consumer code/structure]

For each event in the stream:
1. Assign next ordinal
2. Load current hydration view
3. Run UI projection → emit UI events
4. Run timeline projection → update entities
5. Apply to store with ordinal

[Explain why each step matters]

## 5. Session isolation and ordinal spaces

[Explain - each session has its own ordinal sequence]

s-a/1, s-a/2, s-a/3 (session A ordinals)
s-b/1, s-b/2 (session B ordinals)

Sessions don't share ordinals. Each session's ordinal space is independent. This matters for subscription and reconnection.

## 6. Things to try

┌─ Session A Controls ──────────────────────────────────────────────┐
│  Session: [s-a]   Burst: [4]                                        │
│  [Publish A] [Burst A]                                              │
└────────────────────────────────────────────────────────────────────┘

┌─ Session B Controls ──────────────────────────────────────────────┐
│  Session: [s-b]                                                     │
│  [Publish B]                                                        │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Ordinals increment per-session, not globally
- Bursts show interleaving of sessions
- Each session's stream is independent

## 7. The stream mode experiment

[Show the three modes: derived, missing, invalid]

This experiment shows what happens when the bus stream identifier is wrong.

┌─ Stream Mode ──────────────────────────────────────────────────────┐
│  [Derived] [Missing stream] [Invalid stream id]                     │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Derived: works normally, stream id from session
- Missing: what happens when no stream id?
- Invalid: what happens with malformed stream id?

## 8. Reading the ordinal timeline

[Show the timeline visualization]

Ordinal Timeline:
  s-a/1: LabCommand
  s-a/2: LabCommand
  s-b/1: LabCommand
  s-b/2: LabCommand

What this shows:
- Session prefix keeps ordinals isolated
- Ordinals increment within each session
- No global sequence; local sequences only

## Key Points

- Ordering is foundational; if events arrive out of order, state becomes incoherent
- The bus separates "published" from "consumed"
- Consumer assigns ordinals (not handler) because it sees the full stream
- Each session has its own ordinal space - sessions are isolated

## API Reference

- Bus publisher configuration
- Consumer runner
- Ordinal assignment strategy
```

**Widget Integration**:
- **ScenarioControls** (Targeted): Session A/B controls inline
- **OrdinalTimeline** (Targeted): Visual ordinal sequence
- **ConsumerTrace** (Targeted): Consumer processing steps
- **MessageHistory** (Targeted): Raw message log
- **EventPublisher** (Freeform): Publish raw events
- **BusInspector** (Freeform): Inspect bus state

**Estimated Output**: ~500 lines (15% reduction from 582)

---

### Phase 3: Hydration and Reconnect (410 lines → ~450 lines)

**Current State**: Explains websocket transport, snapshot-before-live, reconnect. Good concept, prose is okay. Layout already has side-by-side clients.

**Target Style**: Peter Norvig - grounded in the reconnect scenario, explains why snapshot-before-live is non-negotiable.

**Core Teaching Goal**: Show that reconnect correctness depends on framework sequencing, not frontend tricks.

**New Structure**:

```
## 1. Why reconnect is a framework problem, not just a UI problem

Many systems treat reconnect as a frontend concern. The browser dropped, so the frontend reconnects. That is true at a superficial level, but reconnect is fundamentally a framework problem because the framework owns the relationship between current durable state, live event delivery, session routing, and transport identity.

If the framework has no coherent answer to those relationships, the frontend can reconnect all it wants and still end up with duplicated updates, missing updates, inconsistent state, or live events arriving before the client has been hydrated.

[Show the four things the framework must answer:]
1. What is the current durable state?
2. What is the current live event stream?
3. Which connection is which?
4. How do they fit together on reconnect?

Phase 3 comes after Phase 2 (ordering) because reconnect only makes sense when the system already knows what state is current and how event order is defined.

## 2. The central lesson: snapshot before live

A reconnecting client should receive a coherent snapshot first, and only then continue with live UI events.

[This is the most important sentence in the phase]

If live events arrive before the client has been hydrated, the client can momentarily observe a world that does not line up with the snapshot it later receives. That leads to duplicated state, visual jumps, confusing debugging, and loss of trust in the system.

If, on the other hand, the framework ensures snapshot-before-live, then the reconnect story becomes much easier to reason about:
1. Client subscribes
2. Framework loads snapshot from HydrationStore
3. Framework sends snapshot message first
4. Framework begins sending live UI events after snapshot

That is the narrative the transport must preserve.

## 3. Why ConnectionId matters more here than it did before

In earlier phases, ConnectionId was mostly a vocabulary concept. In Phase 3 it becomes operational. This is where you really feel why the framework separated ConnectionId from SessionId.

[Table: SessionId vs ConnectionId]

SessionId: business-level routing identity
ConnectionId: transport-level connection (one websocket)

A single session may later have:
- Multiple browser tabs
- A reconnecting tab replacing an older connection
- Several observers attached to the same session
- One client disconnecting while another remains subscribed

If the framework had collapsed these concepts, transport logic would become awkward and brittle.

## 4. What the websocket transport is supposed to do

[Explain - transport is not the source of truth, it's the delivery mechanism]

The transport should:
- Accept live connections
- Assign ConnectionIds
- Track subscriptions by session
- Deliver snapshots on subscribe
- Deliver live UI events after snapshot
- Accept unsubscribe/disconnect cleanly
- Stay unaware of application-specific business logic

The transport should not:
- Invent application semantics
- Assign ordinals
- Decide what the canonical event model means
- Become the place where commands are secretly interpreted

[Explain why this distinction matters - transport code is one of the easiest places for architectural leakage]

## 5. Conceptual sequence

Client connects → receives ConnectionId
Client subscribes to SessionId → framework loads snapshot from HydrationStore
Framework sends snapshot message first
Framework begins sending live UI events after snapshot

[Show this as a diagram]

Why the order matters:
- If reversed: client sees live append events, then receives stale snapshot, then needs complex client logic to reconcile
- If correct: client receives current state, then live events continue naturally

## 6. The controls and what they teach

┌─ Scenario Setup ──────────────────────────────────────────────────┐
│  Session: [reconnect-demo]   Prompt: [watch reconnect preserve]   │
│  [Seed Session] [Refresh State] [Reset]                           │
└────────────────────────────────────────────────────────────────────┘

Seed Session: creates activity in the session, establishing state the clients will later reconnect to.

---

┌─ CLIENT A ─────────────────────────┐ ┌─ CLIENT B ───────────────┐
│ ○ Disconnected                     │ │ ○ Disconnected           │
│ ○ Not subscribed                   │ │ ○ Not subscribed         │
│ ○ No snapshot                      │ │ ○ No snapshot            │
│                                     │ │                          │
│ sinceOrdinal: [0]                   │ │ sinceOrdinal: [0]        │
│ [Connect] [Subscribe] [Disconnect]  │ │ [Connect] [Subscribe]     │
│                                     │ │ [Disconnect]              │
│ "Client A idle."                    │ │ "Client B idle."          │
└─────────────────────────────────────┘ └──────────────────────────┘

What each state means:
- ○ Disconnected: websocket not open
- ○ Not subscribed: connected but not watching session
- ○ No snapshot: subscribed but hasn't received snapshot yet
- ● (any state): that state is active

## 7. Things to try

┌─ Try 1: connect Client A and subscribe to empty session ─────────────┐
│                                                                       │
│  1. Click Connect for Client A                                        │
│  2. Click Subscribe for Client A                                      │
│  3. Observe: empty snapshot, then live state begins                   │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Even an empty session teaches whether snapshot-before-live is followed
- The absence of state should still arrive in the correct shape

┌─ Try 2: generate activity, then connect Client B later ──────────────┐
│                                                                       │
│  1. Seed Session (creates state)                                      │
│  2. Connect + Subscribe Client B                                      │
│  3. Observe: current snapshot, then only subsequent live events        │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Client B receives current state first
- Client B sees only events AFTER its subscription
- No duplication, no catching up required

┌─ Try 3: disconnect Client A, keep activity, then reconnect ──────────┐
│                                                                       │
│  1. Seed Session                                                      │
│  2. Connect + Subscribe Client A                                      │
│  3. Click Disconnect for Client A                                     │
│  4. Seed Session again (more activity)                                │
│  5. Reconnect Client A                                                │
│  6. Observe: snapshot includes ALL activity, then live continues       │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- On reconnect, Client A receives current snapshot (not just new events)
- Live stream continues naturally from snapshot
- No race condition between snapshot and live events

┌─ Try 4: compare Client A and Client B final state ────────────────────┐
│                                                                       │
│  1. Set up both clients subscribed to same session                    │
│  2. Generate activity                                                │
│  3. Observe: even with different connect/subscribe timings, clients   │
│     converge to the same final understanding of session              │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- This is the strongest transport invariant: clients with different live histories should still end at the same session truth

## 8. The kinds of bugs this phase prevents

[Explain each bug class and why it matters]

Bug class 1: live before snapshot
- Classic reconnect bug
- Client receives live data before coherent base state

Bug class 2: transport owning business semantics
- If websocket code interprets application meaning, framework boundary pollutes
- Transport becomes hidden business logic

Bug class 3: confusing connections with sessions
- Leads to brittle subscription and reconnect behavior
- One connection does not equal one session

Bug class 4: duplicated or skipped state after reconnect
- Store is correct but subscribe sequence is bad
- Client ends up wrong despite store being right

Bug class 5: hidden coupling between transport and example app
- If transport only works because it secretly knows chat message shapes, framework is trapped

## Key Points

- Reconnect correctness depends on framework sequencing, not frontend tricks
- Snapshot-before-live is non-negotiable
- ConnectionId and SessionId are separate concepts serving different purposes
- Transport is downstream of projections; it is not the source of truth
- Clients with different live histories should still converge to same session truth

## API Reference

- WebSocket transport configuration
- Connection lifecycle management
- Subscription management
- Snapshot delivery on subscribe
```

**Widget Integration**:
- **ScenarioControls** (Targeted): Seed/Refresh/Reset inline
- **WebSocketClient** (Targeted): Client A and Client B with state indicators
- **ClientComparison** (Targeted): Side-by-side state display
- **BackendTrace** (Targeted): Backend event trace with toggle
- **ClientExplorer** (Freeform): Create arbitrary clients
- **SubscriptionManager** (Freeform): Manage subscription state

**Estimated Output**: ~450 lines (slight increase from 410 to accommodate better prose)

---

### Phase 4: Chat Example (447 lines → ~500 lines)

**Current State**: Explains chat built ON evtstream, not in it. Stop behavior. Good content, flat prose.

**Target Style**: Peter Norvig - grounded in the distinction between framework and application, shows real chat behavior.

**Core Teaching Goal**: Make clear that chat is a consumer of evtstream, not part of it. The same framework supports any application.

**New Structure**:

```
## 1. What this phase is about

Phase 4 is the first time the framework does something that looks like a real application. It is not the first time the framework demonstrates its architecture—that was Phase 1. It is the first time the architecture runs a workload that feels familiar: chat.

This distinction matters. If we had put chat-like behavior into the framework itself, the framework would become specialized. It would carry chat assumptions into every consumer. That would defeat the purpose of building a reusable substrate.

Instead, chat lives in an example package that consumes the framework. The framework stays clean. The example stays honest. And you can see how the same framework patterns support other applications later.

## 2. How chat builds on evtstream

[Show the hierarchy: chat.go → evtstream APIs → transport]

Chat is not in evtstream. Chat imports evtstream. This is the key relationship.

[Show the import structure]
```go
package chat // pinocchio/pkg/evtstream/examples/chat
import "pinocchio/pkg/evtstream"
```

The chat handler registers with evtstream. It uses Hub.Submit, event publishing, projections. It lives in the same process as the framework but conceptually outside it.

This means:
- The framework has no chat-specific code
- The chat example proves the framework is usable
- Other examples could exist alongside chat

## 3. The chat command flow

When you send a prompt, this happens:

User → HTTP → ChatHandler → evtstream.Hub.Submit(ChatPrompt)

The Hub routes to ChatHandler. ChatHandler publishes events:
- ChatPromptSent: the prompt entered the system
- ChatChunk: (multiple) tokens streaming in
- ChatFinished: stream complete
- ChatStopped: stream interrupted by user

The UIProjection transforms these into client-facing messages. The TimelineProjection builds the persistent chat history.

[Show the flow with pseudocode]

## 4. The stop behavior

Chat supports stopping mid-stream. This is harder than it sounds.

The problem: once a handler has started producing chunks, stopping it cleanly requires coordination. The handler must check for cancellation. The transport must not deliver partial states. The hydration store must not record incomplete work as permanent.

[Show the stop flow]

1. User clicks Stop
2. Backend receives stop signal
3. Handler checks context and stops publishing
4. Transport delivers ChatStopped event
5. Hydration store records ChatStopped, not partial completion

What "correct" stop looks like:
- UI shows "stopped" not "finished"
- Hydration shows ChatStopped status
- No further chunks arrive
- Client state is coherent

What "incorrect" stop looks like:
- UI shows partial text as if finished
- Hydration records partial state
- Stop signal ignored, chunks keep coming
- Client state is wrong and confusing

## 5. The controls and what they teach

┌─ Interactive Demo ─────────────────────────────────────────────────┐
│  Session: [chat-demo]                                                │
│  Prompt: [Explain ordinals in plain language]                       │
│  [Send] [Stop] [Refresh] [Reset]                                   │
└─────────────────────────────────────────────────────────────────────┘

Send: submits ChatPrompt command
Stop: signals cancellation mid-stream
Refresh: reloads current session state
Reset: clears session and starts fresh

## 6. Things to try

┌─ Try 1: send a prompt ──────────────────────────────────────────────┐
│  [Send] an ordinary prompt                                          │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Prompt appears in UI
- Chunks stream in
- Final message appears
- Hydration shows ChatMessage with finished status

┌─ Try 2: stop mid-stream ────────────────────────────────────────────┐
│  [Send] a long prompt, click [Stop] while chunks are arriving        │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Chunks stop arriving
- UI shows "stopped" not "finished"
- Hydration shows ChatStopped status
- No partial completion recorded

What to look for:
- Does the framework deliver ChatStopped event?
- Does the hydration store record "stopped" or "finished"?
- Does the UI show coherent state or a mix of states?

┌─ Try 3: compare stop vs finish ─────────────────────────────────────┐
│  [Send] two identical prompts, stop one, finish one                  │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Finished: status = "finished", full text
- Stopped: status = "stopped", partial text
- Both should be coherent; neither should look broken

┌─ Try 4: refresh after stop ─────────────────────────────────────────┐
│  [Stop] a prompt, then [Refresh]                                     │
└────────────────────────────────────────────────────────────────────┘

What to observe:
- Session state includes the stopped message
- Hydration reflects "stopped" status
- Reconnect would see the same state

## 7. What the live stream preview shows

[Explain - this is what the client sees in real-time]

The live stream preview shows the UI events as they arrive. It demonstrates:
- Chunks arriving incrementally
- Status changing from streaming to finished or stopped
- The event stream becoming client-visible

[Show what the trace shows vs what the preview shows]

The preview is the rendered view of the trace. It shows what the user sees. The trace shows what the system did.

## 8. Why this matters for future applications

[Explain - chat is one example, the framework supports any event-streaming workload]

The patterns in chat:
- Command triggers work
- Events describe what happened
- Projections derive views
- Hydration stores state

These patterns apply to:
- Code generation agents
- Data processing pipelines
- Real-time analytics
- Any workload that fits the command → event → projection model

Chat proves the pattern works. Future applications will use it differently.

## Key Points

- Chat is a consumer of evtstream, not part of it
- The framework has no chat-specific code
- Stop behavior requires coordination: handler checks context, transport delivers event, store records status
- "Stopped" and "finished" are both valid terminal states
- The same patterns that support chat support any event-streaming workload

## API Reference

- ChatHandler registration
- ChatPrompt command structure
- Chat events: ChatPromptSent, ChatChunk, ChatFinished, ChatStopped
- UI events: ChatMessageStarted, ChatMessageAppended, ChatMessageFinished, ChatMessageStopped
- Timeline entity: ChatMessage with status field
```

**Widget Integration**:
- **ScenarioControls** (Targeted): Send/Stop/Refresh/Reset inline
- **LiveStreamPreview** (Targeted): Real-time UI event display
- **BackendTrace** (Targeted): Backend event trace
- **SessionViewer** (Targeted): Session state display
- **SnapshotViewer** (Targeted): Hydration state
- **CommandBuilder** (Freeform): Build custom chat commands
- **StreamMonitor** (Freeform): Monitor live streams

**Estimated Output**: ~500 lines (slight increase from 447 to add foundational content)

---

### Phase 5: Persistence and Restart (388 lines → ~450 lines)

**Current State**: Explains memory vs SQL mode, restart correctness, cursor/entity preservation. Good content, flat prose.

**Target Style**: Peter Norvig - grounded in what happens to state across restart, explains why durability is the final piece.

**Core Teaching Goal**: Show that SQL durability completes the framework's promise—state survives restart, ordinals continue without gaps.

**New Structure**:

```
## 1. Why persistence changes the emotional feel of the system

Up to Phase 4, the system is warm. Events flow, projections run, state exists in memory. Everything works. But when you restart the backend, everything disappears. The sessions are gone. The hydration state is gone. The ordinals start over.

This is fine for development. It is not fine for production.

Once the system uses SQL hydration, the emotional feel changes. You stop worrying about restart. You stop checking whether state survived. You trust the system to be there when you return. That trust is the promise of persistence.

Phase 5 is about making that promise real.

## 2. The central lesson: cursor state and timeline state must survive together

The hydration store tracks two things:
1. The latest ordinal per session (cursor)
2. The accumulated entities per session (timeline)

If only the cursor survives restart, the system knows where it left off but not what happened. If only the timeline survives, the system has state but not context for new events. Both must survive together.

Why does this matter for ordinals? Because if the cursor resets on restart, new events will reuse ordinals 1, 2, 3. That creates a gap: old events with ordinals 1-10, new events with ordinals 1-5. Clients reconnecting will see events out of order with no way to know the gap existed.

SQL persistence ensures:
- Cursor survives: ordinals continue from where they left off
- Timeline survives: entity state is preserved
- No gaps: new events have ordinals that continue the sequence

## 3. Memory vs SQL mode

[Show the toggle]

┌─ Mode ─────────────────────────────────────────────────────────────┐
│  ○ In-Memory    ● SQL                                               │
└────────────────────────────────────────────────────────────────────┘

In-Memory mode:
- State lives in process memory
- Fast, no disk I/O
- Lost on restart
- Good for development

SQL mode:
- State lives in SQLite database
- Persisted to disk
- Survives restart
- Good for production

The toggle lets you observe the difference. Switch to Memory, restart, see state disappear. Switch to SQL, restart, see state preserved.

## 4. What SQL persistence looks like

[Show the store interface - Apply, Snapshot, View, Cursor - with SQL backing]

The SQL store implements the same interface as the memory store. That is the point: consumers do not know whether the backing is memory or SQL.

```go
type HydrationStore interface {
    Apply(ctx context.Context, sessionId SessionId, ordinal uint64, entities []TimelineEntity) error
    Snapshot(ctx context.Context, sessionId SessionId) (*Snapshot, error)
    View(ctx context.Context, sessionId SessionId) (TimelineView, error)
    Cursor(ctx context.Context, sessionId SessionId) (uint64, error)
}
```

The interface is the contract. The implementation is an internal detail.

## 5. What survives restart

[Show the before/after comparison]

┌─ Session: persist-demo ────────────────────────────────────────────┐
│  BEFORE RESTART                    │  AFTER RESTART                 │
│  ──────────────────────────────     │  ─────────────────────────────  │
│  sessionId: persist-demo            │  sessionId: persist-demo       │
│  ordinal: 5                        │  ordinal: 5  ← preserved        │
│  entities: [ChatMessage]           │  entities: [ChatMessage]        │
│                                    │                                │
│                                    │  [SQL: preserved ✓]            │
└────────────────────────────────────┘                                │
```

In SQL mode:
- Ordinal continues from 5 (not reset to 0)
- Entity state is intact
- No gap in the sequence

In Memory mode:
- Ordinal resets to 0
- Entity state is lost
- Gap appears in sequence

## 6. The restart simulation

The controls let you simulate a backend restart without actually stopping the server. This is useful for observing the behavior in a controlled way.

┌─ Session Setup ─────────────────────────────────────────────────────┐
│  Session: [persist-demo]   Text: [persist this record]              │
│  [Seed Session]                                                            │
└────────────────────────────────────────────────────────────────────┘

┌─ Restart Test ──────────────────────────────────────────────────────┐
│  [Simulate Backend Restart]                                          │
│  [Reconnect Client] [Reset]                                          │
└────────────────────────────────────────────────────────────────────┘

Simulate Backend Restart: tells the hydration store to reload from disk, as if the backend had restarted.

## 7. Things to try

┌─ Try 1: SQL mode, seed, restart, observe ───────────────────────────┐
│                                                                       │
│  1. Ensure Mode is SQL                                                │
│  2. Seed Session                                                      │
│  3. Click Simulate Backend Restart                                    │
│  4. Observe: ordinal preserved, entities intact                      │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Ordinal counter continues from where it was
- ChatMessage entity is still in the store
- No duplication, no gap

┌─ Try 2: Memory mode, seed, restart, observe ───────────────────────┐
│                                                                       │
│  1. Switch Mode to In-Memory                                          │
│  2. Seed Session                                                      │
│  3. Click Simulate Backend Restart                                     │
│  4. Observe: ordinal resets, entities lost                           │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Ordinal counter resets to 0
- ChatMessage entity is gone
- State is fresh, as if no prior work happened

┌─ Try 3: reconnect after restart in SQL mode ─────────────────────────┐
│                                                                       │
│  1. Seed Session in SQL mode                                          │
│  2. Click Simulate Backend Restart                                     │
│  3. Connect + Subscribe Client                                        │
│  4. Observe: client receives current snapshot, then live events      │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Client does not receive the pre-restart events (it wasn't subscribed)
- Client receives current snapshot as its starting point
- Client receives live events after snapshot

┌─ Try 4: compare pre and post restart snapshots ───────────────────────┐
│                                                                       │
│  1. Seed Session in SQL mode                                          │
│  2. Refresh to see pre-restart snapshot                               │
│  3. Click Simulate Backend Restart                                     │
│  4. Refresh again                                                     │
│  5. Observe: snapshots are identical                                 │
└───────────────────────────────────────────────────────────────────────┘

What to observe:
- Pre-restart and post-restart snapshots show the same state
- The SQL store preserves everything

## 8. What the pre/post comparison shows

[Walk through the comparison widget]

The comparison shows:
- Ordinal: before and after (should be identical in SQL mode)
- Entities: before and after (should be identical in SQL mode)
- Mode label: [SQL: preserved] vs [Memory: forgotten]

The comparison makes durability visible. In SQL mode, before and after are identical. In Memory mode, after is fresh.

## Key Points

- Persistence changes the emotional feel: you trust the system to survive restart
- Cursor state (ordinal) and timeline state (entities) must both survive
- If ordinals reset on restart, new events create gaps in the sequence
- SQL persistence ensures ordinals continue without gaps
- The HydrationStore interface is the contract; memory/SQL are implementation details

## API Reference

- SQL store initialization
- Apply with ordinal advancement
- Snapshot serialization
- Cursor recovery on startup
```

**Widget Integration**:
- **ModeToggle** (Targeted): Memory vs SQL switch
- **ScenarioControls** (Targeted): Seed, Restart, Reconnect inline
- **PrePostComparison** (Targeted): Before/after snapshot comparison
- **SnapshotViewer** (Targeted): Current state display
- **StoreInspector** (Freeform): Raw SQL state inspection
- **QueryBuilder** (Freeform): Run custom queries on the store

**Estimated Output**: ~450 lines (increase from 388 to add foundational content about why persistence matters)

---

## Part IV: Widget Integration Summary

### Per-Phase Widget Inventory

| Phase | Targeted Widgets | Freeform Widgets |
|-------|-----------------|-----------------|
| Phase 0 | ChapterViewer, StatusIndicator | — |
| Phase 1 | ScenarioControls, TraceTimeline, ToggledViewer (×2), CheckList | SessionExplorer, EvidenceLog |
| Phase 2 | ScenarioControls, OrdinalTimeline, ConsumerTrace, MessageHistory, CheckList | EventPublisher, BusInspector |
| Phase 3 | ScenarioControls, WebSocketClient (×2), ClientComparison, BackendTrace, CheckList | ClientExplorer, SubscriptionManager |
| Phase 4 | ScenarioControls, LiveStreamPreview, BackendTrace, SessionViewer, SnapshotViewer, CheckList | CommandBuilder, StreamMonitor |
| Phase 5 | ModeToggle, ScenarioControls, PrePostComparison, SnapshotViewer, CheckList | StoreInspector, QueryBuilder |

### Widget Placement Principles

1. **Inline controls** go directly below the prose that explains them
2. **Evidence panels** go below inline controls, visible without scrolling
3. **Freeform widgets** go in the sidepane, accessible for exploration
4. **Try sections** use inline ScenarioControls tied to specific textbook concepts
5. **Freeform sections** use sidepane widgets for open experimentation

---

## Part V: Implementation Order

### Phase-by-Phase Rewrite Order

| Order | Phase | Estimated Lines | Key Widgets |
|-------|-------|----------------|------------|
| 1 | Phase 0 | 500 | ChapterViewer, StatusIndicator |
| 2 | Phase 1 | 600 | ScenarioControls, TraceTimeline, ToggledViewer |
| 3 | Phase 2 | 500 | OrdinalTimeline, ConsumerTrace |
| 4 | Phase 3 | 450 | WebSocketClient, ClientComparison |
| 5 | Phase 4 | 500 | LiveStreamPreview, SessionViewer |
| 6 | Phase 5 | 450 | ModeToggle, PrePostComparison |

Total: ~3,000 lines (reduced from 3,215 current lines)

### Writing Process

For each phase:
1. Read current chapter
2. Read design guide section
3. Read textbook-authoring skill
4. Write new chapter following template
5. Integrate targeted widgets inline
6. Add freeform widgets to sidepane design
7. Validate against Peter Norvig style guide
8. Upload to reMarkable for review

---

## Part VI: Quality Checklist

Before any chapter is considered complete:

- [ ] Opening paragraph states purpose (not summary)
- [ ] Conceptual foundation explains why, not just what
- [ ] Code is real implementation, not abstract sketch
- [ ] Tables/diagrams do work prose cannot
- [ ] Try sections have inline controls adjacent to explanation
- [ ] Bullet lists are complete sentences
- [ ] No AI-slop patterns (wandering preamble, hedged claims, qualifiers)
- [ ] Targeted widgets illustrate specific prose sections
- [ ] Freeform widgets enable exploration beyond the text
- [ ] Key points surface important takeaways (not new information)
- [ ] Closing connects to next phase
- [ ] Length is appropriate (Phase 0 shortest, Phases 1-2 longest)