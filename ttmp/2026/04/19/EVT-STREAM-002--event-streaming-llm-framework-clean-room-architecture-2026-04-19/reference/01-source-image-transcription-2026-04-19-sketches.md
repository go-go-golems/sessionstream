---
Title: Source Image Transcription (2026-04-19 sketches)
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
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: sources/diagram-page-1.png
      Note: "Page 1 — dataflow sketch with SRC/PKG, SRV/conv/PROC, multiplexed clients, hydration."
    - Path: sources/building-blocks-page-2.png
      Note: "Page 2 — BUILDING BLOCKS list (backend layer, generic layer, client layer, open questions)."
ExternalSources: []
Summary: "Verbatim transcription of two hand-drawn whiteboard pages dated 2026-04-19 sketching a reusable framework for event-streaming LLM applications."
LastUpdated: 2026-04-19T13:30:00-04:00
WhatFor: "Preserve the original sketch content as a faithful textual record so downstream design docs can reference it without re-interpreting handwriting."
WhenToUse: "When you need the literal source words/labels from the 2026-04-19 sketches; for paraphrased interpretation see the architecture analysis."
---

# Source Image Transcription (2026-04-19 sketches)

## Goal

A faithful, clean-room transcription of two hand-drawn pages dated **2026-04-19**, treated as the only source of truth for the EVT-STREAM-002 ticket. No other documents in the repository were consulted.

Where handwriting is ambiguous, the transcription uses `[?word?]` to mark a best guess and `[unclear]` where the text cannot be reasonably resolved.

## Context

- Two physical pages, captured as `sources/diagram-page-1.png` and `sources/building-blocks-page-2.png`.
- Page 1 mixes prose annotations with a pipeline-style block diagram.
- Page 2 is an outline titled **BUILDING BLOCKS** that decomposes the proposed framework into three layers plus open questions.
- The author appears to be sketching a reusable substrate for realtime, websocket-based LLM/agent applications (chat, scraper-with-replay, etc.) where one side emits structured events keyed by a conversation id and an intermediate layer carries all state, multiplexing and hydration concerns.

---

## Page 1 — Dataflow sketch (`sources/diagram-page-1.png`)

### Header
- Boxed date: `2026.04.19`

### Prose blocks (top to bottom)

> **REPLAY** will allow me to trace and reproduce individual scraper steps and refine the program, and then re-run [?scrape usemint?] and [?the labeled data?] for real.

> Then the cleaned up **webchat package** and maybe build a couple different backends.

> It's basically realtime websocket structured widget handling + hydration with one side just emitting events with a conversation id, and the rest managed by this intermediate layer.

### Top sub-diagram (left → right pipeline)

```
                 do something
                 (conv-id,
                  profile-id)
                       |
                       v
                  +-------+    [///]-[///]-[///]-->  +-------+
                  |  SRC  |--->  (3 stacked         |  PKG  |
                  +-------+       rectangles /      +-------+
                                   queue cells)
                                                          (small circle / "donut"
                                                           shape floats nearby)

           (label below the pipeline:) "no need for persistent connection"
           (label crossing the pipeline:) "CMDs carried through"
```

Verbatim labels:
- Box: `SRC`
- Box: `PKG`
- Arrow into `SRC` from text: `do something (conv-id) profile-id`
- Mid-pipeline annotation: `no need for persistent connection`
- Crossing label: `CMDs [?carried?] through`
- Standalone shape: small circle / donut to the right of the queue cells (no label)

### Bottom sub-diagram (left → right, multi-row)

```
   [////]-->                         /-- conv --> ( PROC )
   [////]--> +-----+   [conv]--DD--> ( PROC )
   [////]--> | SRV |---broadcast---> ( PROC )                   (Common? Cannot?) CLIENTS
             +-----+   |                                        (also general)
                       \-- conv --DD--> ( PROC )                    CMDS
                       \-- conv --DO--> ( PROC )                    /
                                                                   WC -->( • )  - comm multi-
                                                                   WS -->( • )    plexing
                                                                   WS -->( • )  - conv creation
                                                                                - conv hydration
   (annotation at bottom:)                                                      - cancellation
       "+ state machine                                                           multiplex
        (up to PROC, guess)"
   (label "map to conv" with arrow)
   (arrow labeled HYDRATION sweeping toward stacked DB cylinders)
   (X mark next to one cylinder)
```

Verbatim labels (row-/element-wise):
- Box: `SRV`
- Three queue arrays (`////` shapes) on the far left feeding into `SRV`.
- Label between `SRV` and the conv rows: `Broadcast` (written vertically along an arrow).
- Per-row labels: `conv`, `DD` / `DO` (small two-letter tags), cloud label `PROC`.
- Right column header: `[?Cannot?] CLIENTS (also general)` followed by `CMDS`. The first word is hard to read — most plausible readings are `Connect`, `Common`, or `Cannot`. (Best guess: `Common` / `Connected`.)
- Right column ports: `WC`, `WS`, `WS` each pointing at a hatched circle (likely a websocket endpoint glyph).
- Bullet list to the right:
  - `comm multiplexing`
  - `conv creation`
  - `conv hydration`
  - `cancellation multiplex`
- Bottom-left annotation: `+ state machine (up to PROC, guess)`
- Bottom-left label with arrow: `map to conv`
- Bottom-right arrow: `HYDRATION` (curving toward stacked DB cylinders)
- Two stacked database cylinders at the bottom; one has an `X` mark next to it (meaning unclear — possibly "do not persist here" or "wrong path").

---

## Page 2 — BUILDING BLOCKS (`sources/building-blocks-page-2.png`)

### Header
- Boxed: `BUILDING BLOCKS`

### Outline (verbatim, lightly normalised punctuation)

- **Backend layer** (for example llm chat)
  - Backend event schemas
  - Set of commands (start inference, stop inference)
  - Session metadata
    - ↳ (profile, token count, etc.)
  - Event processors

- **Generic layer**   *(margin annotation: `connection objects`)*
  - Websocket handling (establish, disconnect)
  - Session objects (+ allow backend metadata)
  - Generic event processing pipeline + projection
  - Pluggable command framework
    - on command (evt, session, connection)
  - Generic hydration / timeline entity storage
  - Generic timeline events
  - *(left-margin annotation pointing up at the bullets:)*
    > "this also allows managing connection state from the backend side"

- **Client layer**
  - UI event schemas
  - Processors: backend evt → UI events (+ session)
  - Set of UI commands and how they are mapped to backend
  - Hydration
  - Timeline object schemas

### Open questions (bottom of page, verbatim)

> How do we handle stale connections?
> Tick messages from backend and frontend?

---

## Glyph / abbreviation key (interpretation, not transcription)

| Glyph / token         | Most likely meaning                                                  |
|-----------------------|----------------------------------------------------------------------|
| `SRC`                 | Source — emitter of work (e.g., scraper step or "do something" call) |
| `PKG`                 | Package — final/packaged output of the pipeline                      |
| `[///]` cells         | Queue / buffer / pipeline stages between SRC and PKG                 |
| `SRV`                 | Server — long-running process owning conversations                   |
| `conv`                | Conversation (keyed by `conv-id`)                                    |
| `PROC` (cloud)        | Processor — does the actual LLM/agent work for one conversation     |
| `DD` / `DO`           | Per-conv tag, possibly "data direction" or doc id (unresolved)       |
| `WC` / `WS`           | Websocket Client / Websocket Server endpoints                        |
| Hatched circle        | Websocket connection (endpoint)                                      |
| Donut at top          | Unlabeled — possibly "ack" / "token" / cache marker                  |
| `X` near DB cylinder  | "do not persist" / "wrong path" — author's mark                      |

These interpretations are the transcriber's best guess; the design doc is responsible for any further extrapolation.

## Usage Examples

- Quoting source intent in a design discussion:
  > Per the 2026-04-19 sketch (page 1): *"one side just emitting events with a conversation id, and the rest managed by this intermediate layer."*
- Resolving "what was actually written" disputes during implementation: open `sources/diagram-page-1.png` / `sources/building-blocks-page-2.png` alongside this transcription.

## Related

- Architecture analysis: `../design/01-architecture-analysis-event-streaming-llm-framework.md`
- Source images: `../sources/diagram-page-1.png`, `../sources/building-blocks-page-2.png`
