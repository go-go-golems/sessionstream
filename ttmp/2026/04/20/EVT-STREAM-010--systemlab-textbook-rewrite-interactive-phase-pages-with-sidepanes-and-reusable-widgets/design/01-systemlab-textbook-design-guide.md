---
Title: "Design Guide: Systemlab Textbook - Sidepanes, Widget Architecture, and Phase Layouts"
Ticket: EVT-STREAM-010
Status: active
Topics:
    - documentation
    - systemlab
    - frontend
    - teaching
    - onboarding
    - widgets
    - react
    - ux-design
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase1.html
      Note: Phase 1 current layout (vertical stack)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase3.html
      Note: Phase 3 current layout (horizontal grid for clients)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase4.html
      Note: Phase 4 current layout (vertical stack)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase5.html
      Note: Phase 5 current layout (to be analyzed)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/app.css
      Note: Current CSS (no responsive sidepane support)
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-0-foundations.md
      Note: Reference chapter structure
ExternalSources: []
Summary: "Design guide for restructuring Systemlab phase pages with interactive sidepanes, reusable React widgets, and minimized scrolling. Widgets serve two purposes: (1) illustrate textbook content by making invariants and behaviors visible, tied to specific prose sections; (2) enable freeform experimentation by giving readers room to explore behavior not covered in the text. Covers current state analysis, widget taxonomy (targeted vs freeform), phase-specific layouts, and ASCII wireframes for each phase."
LastUpdated: 2026-04-20T09:20:00-04:00
WhatFor: "Guide the rewrite of Systemlab phase pages to use sidepane layout with controls adjacent to explanatory text, and establish a widget system that can be implemented in React."
WhenToUse: "When implementing the Phase 1-5 page rewrites and planning the React widget migration."
---

# Design Guide: Systemlab Textbook - Sidepanes, Widget Architecture, and Phase Layouts

## Context

This document provides the design guide for rewriting Systemlab phase pages according to the brief in `EVT-STREAM-009`. The core constraint from the brief is: **minimize scrolling, interactive controls should be close to related text, or in a sidepane**.

This design guide addresses:
1. Current state analysis - what's broken in the vertical stack layout
2. Widget taxonomy - reusable components for the React migration
3. Phase-by-phase layout designs with ASCII wireframes
4. Implementation priorities and migration path

---

## 1. Current State Analysis

### 1.1 The Problem: Vertical Stacking Causes Distance

The current layout for Phase 1, 4, and 5 pages follows this order:

```
┌─────────────────────────────────────────────────────┐
│ Phase Title + Description                           │
├─────────────────────────────────────────────────────┤
│ Phase Chapter (collapsible, often very long)        │
├─────────────────────────────────────────────────────┤
│ [Controls]          │ [Checks]                       │
├─────────────────────────────────────────────────────┤
│ [Trace]             │ [Session + UI Events]         │
├─────────────────────────────────────────────────────┤
│ [Hydration Snapshot]                                │
└─────────────────────────────────────────────────────┘
```

**Problems with this layout:**

1. **Chapter scrolls away** - Once you submit and results appear, the chapter text is no longer visible. The learner must scroll up to re-read context.

2. **Controls are far from their explanation** - If the chapter section explains what "Seed Session" does, and then the user must scroll way down to find the button.

3. **Results split across panels** - Trace, UI Events, and Snapshot are in separate panels, requiring mental assembly of the story.

4. **No persistent context** - After interaction, you lose the teaching relationship between action and evidence.

### 1.2 What "Sidepane" Means Here

The brief says "interactive part should be close to the text that relates it, or in a sidepane." This design interprets that as:

- **Inline controls**: For simple controls that directly relate to a paragraph, put them right below or beside the explanation.
- **Dedicated sidepane**: For complex multi-control scenarios (like Phase 3's dual clients), use a collapsible right-side panel.
- **Sticky chapter header**: When scrolling through long chapter text, keep a minimal header visible.

### 1.3 Phase-Specific Patterns

| Phase | Current Pattern | Challenge |
|-------|----------------|-----------|
| Phase 1 | Sequential panels | Controls far from chapter "Things to try" sections |
| Phase 2 | Unknown | Need to check |
| Phase 3 | Two-column clients | Works well, but trace/state still below |
| Phase 4 | Sequential panels | Stop button context lost after scrolling |
| Phase 5 | Unknown | Need to check |

---

## 2. Widget Taxonomy for React Migration

### 2.1 Core Principle: Widgets Illustrate the Textbook and Enable Experimentation

The widgets serve two interconnected purposes:

**1. Illustrate the textbook.** Targeted widgets are tied to specific concepts in the chapter. They make invariants visible, show evidence of behaviors as they are explained, and let the reader verify claims in real-time. When the chapter says "the handler publishes three events," a targeted widget shows exactly those three events appearing in the trace. The widget is an extension of the prose, not a separate demo.

**2. Enable freeform experimentation.** Freeform widgets give the reader room to explore beyond the textbook. They are not tied to a specific section or concept—they exist to surface behavior that may not be covered in the text, or that will be covered later. When a reader tries something the author didn't anticipate and discovers an interesting pattern, that's experimentation working as intended. Freeform widgets are more open-ended; targeted widgets are more constrained.

Both kinds are necessary. Targeted widgets ensure the textbook can be verified and trusted. Freeform widgets ensure the reader can go beyond what was written. A good textbook system offers both, side by side.

### 2.2 Widget Types: Targeted vs Freeform

### 2.2 Widget Categories

#### A. Navigation & Structure

| Widget | Purpose | States |
|--------|---------|--------|
| `PhaseNavigation` | Top-level phase switcher | default, active, disabled |
| `ChapterViewer` | Renders chapter markdown | loading, ready, error |
| `StickyHeader` | Keeps phase context visible | collapsed, expanded |

#### B. Input & Controls

| Widget | Purpose | States |
|--------|---------|--------|
| `TextInput` | Session ID, prompts | default, focused, error |
| `ActionButton` | Primary actions (Submit, Send, Seed) | default, hover, loading, disabled |
| `SecondaryButton` | Supporting actions (Reset, Export) | default, hover, disabled |
| `ScenarioControls` | Grouped inputs + actions for a scenario | idle, running, complete |

#### C. Evidence & Display

| Widget | Purpose | States |
|--------|---------|--------|
| `JSONViewer` | Formatted JSON display | empty, populated, scrollable |
| `RenderedViewer` | Human-readable annotated view | empty, populated |
| `ToggledViewer` | Wraps JSONViewer with JSON/Rendered toggle | json-mode, rendered-mode |
| `TraceTimeline` | Sequential trace with step numbers | empty, running, complete |
| `CheckList` | Invariant badges | all-pass, partial, all-fail |
| `SnapshotViewer` | Hydration state display | empty, hydrated |
| `RenderedViewer` | Human-readable view of structured data | empty, populated |
| `ToggledViewer` | JSON ↔ Rendered toggle for complex data | json-mode, rendered-mode |

**ToggledViewer Pattern** (important UX improvement):
Raw JSON is hard to read. Each evidence panel should offer:
- **JSON mode**: Full structural data for debugging
- **Rendered mode**: Human-readable, annotated view

```
┌────────────────────────────────────────────┐
│ Session + UI Events         [JSON | Rendered]│
├────────────────────────────────────────────┤
│ Rendered Mode:                              │
│ ┌────────────────────────────────────────┐  │
│ │ ● LabMessageStarted                    │  │
│ │   session: lab-session-1               │  │
│ │   ordinal: 1                          │  │
│ │                                       │  │
│ │ ● LabMessageAppended                  │  │
│ │   chunk: "hello"                      │  │
│ │   ordinal: 2                          │  │
│ │                                       │  │
│ │ ✓ LabMessageFinished                  │  │
│ │   final text: "hello from systemlab"  │  │
│ │   ordinal: 3                          │  │
│ └────────────────────────────────────────┘  │
└────────────────────────────────────────────┘
```

Key improvements in rendered mode:
- Event names as readable labels (LabMessageStarted → "Message Started")
- Key-value pairs on separate lines
- Status indicators (● for in-progress, ✓ for complete)
- Ordinal shown but not overwhelming
- Entity types visually distinguished

#### D. Phase-Specific Widgets

| Widget | Phase | Type | Purpose |
|--------|-------|------|---------|
| `StatusIndicator` | 0 | Targeted | Shows phase progress and boundary rules |
| `WebSocketClient` | 3, 4, 5 | Targeted | Simulated client with connect/subscribe/disconnect |
| `ClientComparison` | 3 | Targeted | Side-by-side Client A/B state |
| `MemoryVsSQLToggle` | 5 | Targeted | Switch hydration mode |
| `SessionTimeline` | 4 | Targeted | Visual stream of events per session |
| `FreeformPanel` | All | Freeform | Open experimentation area |

### 2.4 Widget Type Summary

| Type | Purpose | Examples |
|------|---------|----------|
| **Targeted** | Illustrate specific textbook concepts; validate chapter claims | TraceTimeline, CheckList, WebSocketClient, ClientComparison |
| **Freeform** | Enable experimentation; surface unanticipated behavior | EvidenceLog, StoreInspector, ClientExplorer, SessionExplorer |
| **Core** | Infrastructure used by both targeted and freeform widgets | ChapterViewer, ToggledViewer, ActionButton |

### 2.5 Widget Composition Patterns

```
PhasePage
├── StickyHeader (phase title + minimal nav)
├── Layout
│   ├── MainContent
│   │   ├── ChapterViewer
│   │   │   └── InlineControl (Targeted: tied to "Try" sections)
│   │   └── EvidencePanel
│   │       ├── TraceTimeline (Targeted: validates chapter claims)
│   │       ├── EvidenceLog (Freeform: full event capture)
│   │       └── SnapshotViewer (Targeted: shows predicted state)
│   └── Sidepane (collapsible)
│       ├── ScenarioControls (Targeted: specific exercises)
│       ├── CheckList (Targeted: validates invariants)
│       ├── PhaseSpecificWidget (Targeted: e.g., ClientComparison)
│       └── FreeformPanel (Freeform: open experimentation)
```

**Layout principle**: Targeted widgets are placed near the prose they illustrate. Freeform widgets live in the sidepane, accessible for exploration but not competing with the textbook flow.

---

## 3. Phase-by-Phase Layout Designs

### 3.0 Phase 0 - Foundations

**Teaching Goal**: Help the reader understand what evtstream is trying to become and why Systemlab is a separate app.

**Layout**: Chapter-only with StatusIndicator. No rich controls (foundations phase).

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 0 — Foundations                               [≡ Evidence]│
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ## 1. What this framework is trying to do                       │
│                                                                  │
│  [Prose: explains the story]                                    │
│                                                                  │
│  ┌─ Diagram ──────────────────────────────────────────────────┐  │
│  │  Command → Handler → Events → UIProjection + Timeline       │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ## 2. Why we created pkg/evtstream                              │
│                                                                  │
│  [Prose: clean-room vs donor code]                              │
│                                                                  │
│  ┌─ Mental Model ──────────────────────────────────────────────┐  │
│  │  pkg/webchat  → donor and later consumer/example           │  │
│  │  pkg/evtstream → reusable substrate                         │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│ Status Indicator (Targeted: shows phase progress)                 │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ Phase 0: Foundations          ✓ [active]                   │   │
│ │ Phase 1: Command → Event       ○ [not visited]              │   │
│ │ Phase 2: Ordering             ○ [not visited]              │   │
│ │ Phase 3: Hydration            ○ [not visited]              │   │
│ │ Phase 4: Chat Example         ○ [not visited]              │   │
│ │ Phase 5: SQL / Restart        ○ [not visited]              │   │
│ └────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- No inline controls (Phase 0 is foundations only)
- StatusIndicator is the primary interactive widget - shows which phases exist
- Diagram and mental model boxes visualize architecture concepts
- Evidence panel shows framework status (phases implemented, boundary rules)
- This is the orientation phase - sets expectations for what comes next

**Chapter Content Structure:**
1. What this framework is trying to do (story, not features)
2. Why we created pkg/evtstream (clean-room vs donor code)
3. Why Systemlab is separate (boundary proof)
4. Core vocabulary (SessionId, ConnectionId, Command, Event, Projections)
5. The import-cycle lesson (real architectural lesson)
6. The directory map (ownership, not just listing)
7. Validation commands (mechanized trust)
8. Key Points (surface the takeaways)

**Widget Integration Points:**
- StatusIndicator: below chapter, shows phase progress
- No freeform widgets needed (Phase 0 is orientation only)

**Estimated Output**: ~500 lines (from 690)

---

### 3.1 Phase 1 - Command → Event → Projection

**Teaching Goal**: Show the path from command to events to projections.

**Current Layout Problem**: Chapter explains the trace, but trace panel is below the fold.

**Design: Inline Controls + Collapsible Evidence**

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 1 - Command → Event → Projection              [≡ Evidence]│
├──────────────────────────────────────────────────────────────────┤
│
│  ## The Story So Far
│  [Chapter text with inline "Try" controls]
│
│  ┌─ Try 1: Happy Path ─────────────────────────────────────┐
│  │  Session ID: [________]                                  │
│  │  Prompt:    [________]                                  │
│  │  [Submit] [Reset]                                       │
│  └────────────────────────────────────────────────────────┘
│
│  ## What the Trace Shows
│  [Short explanation]
│
├──────────────────────────────────────────────────────────────────┤
│ Evidence Panel (collapsible)
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ │ Trace [JSON|Rend]│ │ UI Events        │ │ Snapshot         │
│ │                 │ │ [JSON|Rendered]  │ │ [JSON|Rendered]  │
│ │ Step 1: cmd     │ │ ● Started        │ │ LabMessage       │
│ │ Step 2: handler │ │ ● Appended       │ │ └ text: "hello"  │
│ │ Step 3: events  │ │ ✓ Finished       │ │ └ status: ✓     │
│ └──────────────────┘ └──────────────────┘ └──────────────────┘
│ ┌──────────────────────────────────────────────────────────────┐
│ │ ✓ sessionExists  ✓ cursorAdvanced  ✓ timelineProduced       │
│ └──────────────────────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- Inline "Try" sections embed controls directly in the teaching text
- Evidence panel collapses by default, expands after first run
- Three-column evidence layout replaces vertical stacking
- "Export JSON/Markdown" buttons appear in evidence panel header

### 3.2 Phase 2 - Ordering and Ordinals

**Teaching Goal**: Show Watermill consumer and ordinal assignment.

**Current Layout Analysis**:
- Sequential panels: Controls → Checks → Bus/Consumer Trace → Message History → Per-Session Ordinals → Snapshots
- All six panels stacked, requiring significant scrolling
- Multiple text areas with no visual grouping
- Session A/B controls separated from their related output

**Design: Timeline Focus + Grouped Controls**

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 2 - Ordering and Ordinals                    [≡ Evidence]  │
├──────────────────────────────────────────────────────────────────┤
│
│  ## Why Ordering Matters
│  [Chapter text]
│
│  ┌─ Try: Submit Multiple Commands ──────────────────────────┐
│  │  Session: [________]                                     │
│  │  Commands:                                              │
│  │    [cmd1] [cmd2] [cmd3] [Run All]                       │
│  └────────────────────────────────────────────────────────┘
│
│  ## Expected Ordinal Sequence
│  [Diagram: cmd1 → ordinal 1, cmd2 → ordinal 2, etc.]
│
├──────────────────────────────────────────────────────────────────┤
│ Evidence Panel
│ ┌────────────────────────────┐ ┌────────────────────────────┐
│ │ Ordinal Assignment         │ │ Consumer Processing        │
│ │ cmd-1 → ordinal: 1          │ │ [step 1] consumed          │
│ │ cmd-2 → ordinal: 2          │ │ [step 2] consumed          │
│ │ cmd-3 → ordinal: 3          │ │ [step 3] consumed          │
│ └────────────────────────────┘ └────────────────────────────┘
│ ┌────────────────────────────┐
│ │ ✓ ordinalsSequential  ✓ consumerProcessingOrder            │
│ └────────────────────────────┘
└──────────────────────────────────────────────────────────────────┘
```

### 3.3 Phase 3 — Hydration and Reconnect

**Teaching Goal**: Demonstrate snapshot-before-live with two clients.

**Current Layout Analysis** (from live testing):
- Two-column grid for Client A and Client B (works well!)
- Each client has: sinceOrdinal input, Connect/Subscribe/Disconnect buttons, status output
- Below controls: Backend Trace (left) + Connections/Snapshot (right) in 2-column grid
- Checks panel at the top
- Chapter text scrolls away before controls visible

**Validated Layout Features**:
- Client A and Client B ARE side-by-side in a 2-column grid ✓
- Each has Connect/Subscribe/Disconnect buttons ✓
- Status shown in dark panel ("Client A idle.")
- Backend Trace shows step sequence ✓
- Connections/Snapshot shows entities and ordinal ✓

**Design: Keep Client Layout, Improve State Indicators and Evidence Visibility**

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 3 — Hydration and Reconnect                   [≡ Evidence] │
├──────────────────────────────────────────────────────────────────┤
│
│  ## The Snapshot-Before-Live Rule
│  [Chapter text explaining the sequence]
│
│  ┌─ Scenario Setup ──────────────────────────────────────────┐
│  │  Session: [________]  Prompt: [________]                   │
│  │  [Seed Session] [Refresh State] [Reset]                   │
│  └──────────────────────────────────────────────────────────┘
│
├──────────────────────────────────────────────────────────────────┤
│ Evidence Panel (collapsible)                                      │
│ ┌──────────────────────────────┐ ┌────────────────────────────┐  │
│ │ CLIENT A                     │ │ CLIENT B                   │  │
│ │ ○ Disconnected              │ │ ○ Disconnected            │  │
│ │ ○ Not subscribed            │ │ ○ Not subscribed          │  │
│ │ ○ No snapshot               │ │ ○ No snapshot             │  │
│ │                              │ │                            │  │
│ │ sinceOrdinal: [____]         │ │ sinceOrdinal: [____]       │  │
│ │ [Connect] [Subscribe]        │ │ [Connect] [Subscribe]      │  │
│ │ [Disconnect]                │ │ [Disconnect]               │  │
│ │                              │ │                            │  │
│ │ "Client A idle."            │ │ "Client B idle."          │  │
│ └──────────────────────────────┘ └────────────────────────────┘  │
│ ┌──────────────────────────────────────────────────────────────┐
│ │ Backend Trace                                                 │
│ │ [Collapsible timeline of backend events]                     │
│ └──────────────────────────────────────────────────────────────┘
│ ┌──────────────────────────────────────────────────────────────┐
│ │ ✓ clientConvergence  ✓ snapshotBeforeLive  ✓ connectionsOk  │
│ └──────────────────────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- Client panels show connection state visually (●/○) not just text
- Snapshot receipt is a distinct indicator, not buried in log
- Backend trace is collapsible to focus on client comparison
- Checks panel summarizes convergence behavior

### 3.4 Phase 4 - Chat Example

**Teaching Goal**: Show real chat on top of evtstream, including stop behavior.

**Design: Split View with Streaming Visualization**

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 4 - Chat Example                              [≡ Evidence] │
├──────────────────────────────────────────────────────────────────┤
│
│  ## How Chat Builds on evtstream
│  [Chapter text]
│
│  ┌─ Interactive Demo ────────────────────────────────────────┐
│  │  Session: [________]                                      │
│  │  Prompt:    [________________________________]              │
│  │  [Send] [Stop] [Refresh] [Reset]                           │
│  └──────────────────────────────────────────────────────────┘
│
│  ## Live Stream Preview
│  ┌──────────────────────────────────────────────────────────┐
│  │  > Explain ordinals in plain language                     │
│  │  ↓ streaming response appears here...                     │
│  │  ✓ finished                                              │
│  └──────────────────────────────────────────────────────────┘
│
├──────────────────────────────────────────────────────────────────┤
│ Evidence Panel
│ ┌────────────────────────────┐ ┌────────────────────────────┐
│ │ Backend Trace              │ │ Session + UI Events        │
│ │ [step 1] ChatPromptSent     │ │ sessionId: chat-demo       │
│ │ [step 2] ChatChunk          │ │ uiEvents: [...]            │
│ │ [step 3] ChatChunk          │ │                            │
│ │ [step 4] ChatFinished       │ │                            │
│ └────────────────────────────┘ └────────────────────────────┘
│ ┌──────────────────────────────────────────────────────────────┐
│ │ Hydration Snapshot                                           │
│ │ { entities: [ChatMessage: { status: "finished", text: "..."}]│
│ └──────────────────────────────────────────────────────────────┘
│ ┌──────────────────────────────────────────────────────────────┐
│ │ ✓ eventsConverge  ✓ stopWorks  ✓ snapshotCorrect            │
│ └──────────────────────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- "Live Stream Preview" widget shows what's happening in real-time
- Stop button is visually prominent, not secondary
- Backend trace has step numbers matching the chapter explanation
- Three evidence views are adjacent, not stacked

### 3.5 Phase 5 - Persistence and Restart

**Teaching Goal**: Show memory vs SQL mode, pre/post restart comparison.

**Current Layout Analysis**:
- Mode selector at top of controls (good - should be prominent)
- Seven action buttons: Connect, Subscribe, Seed, Restart, Reconnect, Refresh, Reset
- Three output panels in 2-column grid (Client Frames + Backend Trace), then Pre/Post below
- Many controls but not grouped by phase (setup, runtime, restart)

**Design: Mode Toggle + Side-by-side State Comparison**

```
┌──────────────────────────────────────────────────────────────────┐
│ Phase 5 - SQL / Restart                              [≡ Evidence]│
├──────────────────────────────────────────────────────────────────┤
│
│  ## Why Durability Changes Everything
│  [Chapter text]
│
│  ┌─ Mode Toggle ─────────────────────────────────────────────┐
│  │  ○ In-Memory    ● SQL                                      │
│  └──────────────────────────────────────────────────────────┘
│
│  ┌─ Scenario ────────────────────────────────────────────────┐
│  │  Session: [________]                                      │
│  │  Prompt:    [________]                                    │
│  │  [Run] [Simulate Restart] [Reset]                         │
│  └──────────────────────────────────────────────────────────┘
│
│  ## What Restart Should Preserve
│  [Explanation of cursor + entity preservation]
│
├──────────────────────────────────────────────────────────────────┤
│ Evidence Panel
│ ┌────────────────────────────┐ ┌────────────────────────────┐
│ │ Pre-Restart State           │ │ Post-Restart State           │
│ │ sessionId: session-1        │ │ sessionId: session-1         │
│ │ ordinal: 5                  │ │ ordinal: 5  ← preserved     │
│ │ entities: [msg-1, msg-2]    │ │ entities: [msg-1, msg-2]     │
│ └────────────────────────────┘ └────────────────────────────┘
│ ┌──────────────────────────────────────────────────────────────┐
│ │ ✓ restartCorrect  ✓ cursorPreserved  ✓ entitiesComplete      │
│ └──────────────────────────────────────────────────────────────┘
└──────────────────────────────────────────────────────────────────┘
```

**Key Design Decisions:**
- Mode toggle is prominent, not buried in controls
- Pre/Post comparison is side-by-side, showing preservation
- "Simulate Restart" is a clear action button
- Checks explicitly verify cursor + entity preservation

---

## 4. Implementation Priorities

### Migration Path: Vanilla JS → React Widgets

**Phase 1: Establish Widget Contracts (CSS + data attributes)**
- Keep current vanilla JS implementation
- Add CSS classes that map to future widget states
- Ensure data flows through structured JSON

**Phase 2: Implement React shell (no widget changes)**
- Set up React build pipeline (Vite)
- Create basic routing for phase pages
- Ensure existing CSS still works

**Phase 3: Extract targeted widgets first**
- Migrate evidence display components (CheckList, ToggledViewer)
- These are tied to specific prose sections
- Validate they illustrate the textbook correctly

**Phase 4: Extract freeform widgets**
- Add EvidenceLog, StoreInspector, ClientExplorer
- These enable experimentation beyond the text
- Place in sidepane, accessible but not competing with flow

**Phase 5: Full migration and cleanup**
- Remove vanilla JS files
- Optimize bundle
- Add Storybook for widget documentation

### Widget Type Priority

| Priority | Widget Type | Reason |
|----------|-------------|--------|
| 1 | Core (ChapterViewer, ToggledViewer, ActionButton) | Infrastructure for both |
| 2 | Targeted (TraceTimeline, CheckList, ScenarioControls) | Illustrate textbook |
| 3 | Freeform (EvidenceLog, StoreInspector) | Enable experimentation |
| 4 | Phase-specific (WebSocketClient, ClientComparison) | Complete coverage |

### 4.2 CSS Architecture for Sidepanes

```css
/* New layout system */
.phase-layout {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 0;
  height: calc(100vh - nav-height);
}

.phase-layout.with-collapsed-sidepane {
  grid-template-columns: 1fr 0;
}

.main-content {
  overflow-y: auto;
  padding: 24px;
}

.sidepane {
  background: var(--surface-secondary);
  border-left: 1px solid var(--border);
  overflow-y: auto;
  padding: 16px;
  transition: width 0.3s ease;
}

.sidepane.collapsed {
  width: 0;
  padding: 0;
  overflow: hidden;
}

/* Widget base styles */
.widget {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 12px;
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

/* State indicators */
.indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}

.indicator.on { background: var(--success); }
.indicator.off { background: var(--muted); }
.indicator.pending { background: var(--warning); }
```

### 4.3 Responsive Behavior

| Viewport | Layout |
|----------|--------|
| Desktop (>1200px) | Full two-column with sidepane |
| Tablet (768-1200px) | Stacked, sidepane as bottom drawer |
| Mobile (<768px) | Single column, evidence as modal |

---

## 5. ASCII Wireframe Summary

### Overall Page Structure

```
┌──────────────────────────────────────────────────────────────────────┐
│ [≡] Phase 1 - Command → Event → Projection         [Evidence ▶/▼]   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ╔════════════════════════════════════════════════════════════╗    │
│  ║  CHAPTER CONTENT WITH INLINE CONTROLS                      ║    │
│  ║  (scrollable, but evidence panel visible on right)         ║    │
│  ╚════════════════════════════════════════════════════════════╝    │
│                                                                      │
├────────────────────────────────┬─────────────────────────────────────┤
│                                │ ╔═══════════════════════════════╗   │
│                                │ ║  SIDE PANE                   ║   │
│                                │ ║  - Scenario Controls         ║   │
│                                │ ║  - CheckList                 ║   │
│                                │ ║  - Phase-specific widget     ║   │
│                                │ ╚═══════════════════════════════╝   │
└────────────────────────────────┴─────────────────────────────────────┘
```

### Evidence Panel (Expanded)

```
┌──────────────────────────────────────────────────────────────────────┐
│ Evidence                                                              │
├─────────────────────┬─────────────────────┬────────────────────────┤
│ Trace               │ Session + Events     │ Snapshot               │
│ ┌─────────────────┐ │ ┌─────────────────┐ │ ┌────────────────────┐ │
│ │ Step 1: cmd     │ │ │ session: {...} │ │ │ entities: [...]    │ │
│ │ Step 2: handler │ │ │ uiEvents: [...] │ │ │ ordinal: 4         │ │
│ │ Step 3: events  │ │ │                 │ │ │                    │ │
│ └─────────────────┘ │ └─────────────────┘ │ └────────────────────┘ │
├─────────────────────┴─────────────────────┴────────────────────────┤
│ ✓ sessionExists  ✓ cursorAdvanced  ✓ timelineProduced  ✓ uiEvents   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 6. Next Steps

1. **Validate this design** with the actual running Systemlab
2. **Check Phase 2 and Phase 5** current layouts
3. **Define widget API contracts** as TypeScript interfaces
4. **Create React project scaffold** with Vite
5. **Implement Phase 1 as first migrated page**

---

## Appendix A: Widget State Machine

```
Widget States:
  - idle: waiting for user input
  - loading: async operation in progress
  - success: operation completed, data displayed
  - error: operation failed, error displayed
  - disabled: widget not available in current context

Transitions:
  idle → loading (on action)
  loading → success (on success)
  loading → error (on failure)
  success → idle (on reset)
  error → idle (on reset/retry)
  any → disabled (on context change)
```

## Appendix B: Phase-Specific Widget Inventory

### B.1 Targeted Widgets (Illustrate Textbook)

| Phase | Widgets | Purpose |
|-------|---------|---------|
| Phase 0 | ChapterViewer, StatusIndicator | Display chapter and system status |
| Phase 1 | ScenarioControls, TraceTimeline, ToggledViewer (UI Events), ToggledViewer (Snapshot), CheckList | Validate command path and projections |
| Phase 2 | ScenarioControls, OrdinalTimeline, ConsumerTrace, MessageHistory, CheckList | Validate ordering and ordinal assignment |
| Phase 3 | ScenarioControls, WebSocketClient (×2), ClientComparison, BackendTrace, SnapshotViewer, CheckList | Validate snapshot-before-live |
| Phase 4 | ScenarioControls, LiveStreamPreview, BackendTrace, SessionViewer, SnapshotViewer, CheckList | Validate chat on evtstream |
| Phase 5 | ModeToggle, ScenarioControls, PrePostComparison, SnapshotViewer, CheckList | Validate restart correctness |

### B.2 Freeform Widgets (Enable Experimentation)

| Phase | Widgets | Purpose |
|-------|---------|---------|
| Phase 0 | — | No freeform needed (foundations only) |
| Phase 1 | SessionExplorer, EvidenceLog | Explore arbitrary sessions, see all events |
| Phase 2 | EventPublisher, BusInspector | Publish raw events, inspect bus state |
| Phase 3 | ClientExplorer, SubscriptionManager | Create arbitrary clients, manage subscriptions |
| Phase 4 | CommandBuilder, StreamMonitor | Build custom commands, monitor live streams |
| Phase 5 | StoreInspector, QueryBuilder | Inspect SQL store, run custom queries |

**Freeform widgets live in the sidepane**, accessible for exploration but not competing with the textbook flow. They enable users to surface behavior not covered in the text—or to discover patterns that will be covered later.

---

## Appendix C: Rendered Viewer UX Guidelines

The **ToggledViewer** is the most impactful UX improvement for learners. Raw JSON is readable but lacks semantic meaning for teaching.

**Key insight from testing**: The current JSON IS well-formatted with syntax highlighting. The problem isn't readability-it's **interpretability**. JSON shows structure but not meaning. A rendered view should bridge that gap.

### Data Transformation Rules

| JSON Structure | Rendered Display | Why |
|---------------|------------------|-----|
| `{"step": 1, "kind": "control", "message": "command submitted"}` | `1 Command submitted` | Step indicator + readable message |
| `{"event": "LabStarted", "ordinal": 1}` | `● Started (ordinal 1)` | Icon + readable name |
| `{"event": "LabFinished", "ordinal": 4}` | `✓ Finished (ordinal 4)` | Check icon for completion |
| `"LabMessage": {status: "finished", text: "hello"}` | `Message: "hello" ✓` | Human-readable content + status |
| `"ordinal": 3` | Dimmed, smaller | De-emphasize technical detail |

### Session + UI Events Transformation

Current JSON (readable but not interpretable):
```json
{"session": {...}, "uiEvents": [{"name": "LabMessageStarted", ...}]}
```

Rendered view (interpretable):
```
┌─ Session: lab-session-1 ────────────────────────┐
│  Created: 2026-04-20T13:30:44Z                 │
│  Message count: 1                               │
└────────────────────────────────────────────────┘
┌─ UI Events ────────────────────────────────────┐
│  1 ● Message started                            │
│     "hello from systemlab"                     │
│  2 ● Message appended                           │
│     chunk: "hello"                              │
│  3 ✓ Message finished                           │
│     final text: "hello from systemlab"          │
└────────────────────────────────────────────────┘
```

### Trace Transformation

Current JSON:
```json
[{"step": 1, "kind": "command", ...}, {"step": 2, "kind": "session", ...}]
```

Rendered view:
```
┌─ Backend Trace ────────────────────────────────┐
│  1 → Command received: LabStart                 │
│  2 → Session created: lab-session-1             │
│  3 → Handler invoked                            │
│  4 → UI projection: LabMessageStarted           │
│  5 → Timeline projection: LabMessage upserted   │
│  6 → UI projection: LabMessageAppended         │
│  7 → Timeline projection: LabMessage updated   │
│  8 ✓ UI projection: LabMessageFinished          │
│  9 ✓ Timeline projection: LabMessage finalized │
└────────────────────────────────────────────────┘
```

### Icon System for Events

```
● (filled circle) = in-progress event
○ (outline circle) = awaiting/not started
✓ (checkmark) = completed event
✗ (x mark) = error/failed event
▶ (play) = command/action received
→ (arrow) = step in sequence
```

### Panel Header Toggle Design

```
┌─ Session + UI Events ──────────────────┬────────────────┐
│                                        │ [JSON] [Rendered]│
│  ● MessageStarted                      │                 │
│  ✓ MessageFinished                     │                 │
└────────────────────────────────────────┴────────────────┘
```

- Toggle buttons are small, secondary styling
- Current mode is highlighted
- Both modes always accessible