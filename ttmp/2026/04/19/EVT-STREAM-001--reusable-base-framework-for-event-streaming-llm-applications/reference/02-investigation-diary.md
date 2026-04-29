---
title: "Investigation Diary"
status: active
doc-type: reference
intent: long-term
topics:
  - llm
  - event-streaming
  - agents
  - architecture
  - framework
owners: []
date: 2026-04-19
ticket: EVT-STREAM-001
---

# Investigation Diary

**Ticket:** EVT-STREAM-001  
**Started:** 2026-04-19

---

## Entry 1 — 2026-04-19: Initial Research from Handwritten Sketches

### What

Reviewed two handwritten architecture sketches dated 2026-04-19:

1. **BUILDING BLOCKS** (`/tmp/pi-clipboard-30e8760e-3990-4838-be7f-61b95b345d2b.png`)  
   A three-layer architecture defining the structure of an event-streaming LLM framework. Three top-level bullets: `backend layer (for example llm chat)`, `Generic Layer`, `client layer`. Each layer has 4–7 sub-bullets describing responsibilities. Bottom has two open questions: *"how do we handle stale connections?"* and *"tick messages from backend and frontend?"*

2. **System topology diagram** (`/tmp/pi-clipboard-48ef5d9f-b33a-4eaf-8593-3f6db8a28dce.png`)  
   A more detailed flow: Base Layer (labeled "SRC.") → PKG (labeled "PKG.") → SRV (dispatch) → CONV queues → PROC (state machines) → CMDS (multiplexing) → WS (client connections) + HYDRATION (persistence cylinder). Key insight: annotation between SRC. and PKG says "no need for persistent connection."

### What I transcribed from the sketches

**BUILDING BLOCKS — verbatim items:**

- `• backend layer (for example llm chat)`
  - `◦ backend event schemas`
  - `◦ set of commands (start inference, stop inference)`
  - `◦ session metadata` → `(profile, token count, etc.)`
  - `◦ event processors`
- `• Generic Layer`
  - `◦ websocket handling (establish, disconnect)` → `connection objects`
  - `◦ session objects (+ allow backend metadata)`
  - `◦ generic event processing pipeline + projection`
  - `◦ pluggable command framework` → `on_command (evt, session, connection)`
  - `◦ generic hydration / timeline entity storage`
  - `◦ generic timeline events`
  - *(left annotation)* "this also allows managing connection state from the backend side"
- `• client layer`
  - `◦ ui event schemas`
  - `◦ processors: backend evt + session -> ui events`
  - `◦ set of ui commands and how they are mapped to backend.`
  - `◦ hydration`
  - `◦ timeline object schemas`
- Bottom questions: `how do we handle stale connections?` and `tick messages from backend and frontend?`

**System topology — verbatim items:**

- Top diagram: SRC. ← arrow labeled `do something (conv-id) profile-id` → PKG. Three shaded squares in between, annotation below: `no need for persistent connection`
- Bottom diagram left: three queue icons → SRV (internal "dispatch" column), upward arrow labeled `CMDS carried through`
- From SRV dispatch: three stacked `conv` boxes → PROC clouds
- Annotation near PROC: `+ state machine (up to PROC I guess)` and `(conv but CLIENTS also backend)`
- Below `conv` boxes: `map to conv` → database cylinder (marked with "x")
- From PROC: three arrows labeled `ws` → three shaded ovals
- CMDS feature list: `• CMDS`, `• conv multi-plexing`, `• conv creation`, `• conv hydration`, `• cancellation multiplex`
- Bottom-right: cylinder labeled `HYDRATION`
- Second paragraph: *"it's basically realtime websocket structured widget handling + hydration, with one side just emitting events with a conversation id and the rest managed by this intermediate layer."*

### What worked

- The three-layer model maps cleanly to a reusable framework design. Backend = LLM-specific, Generic = transport/session/command machinery, Client = UI-specific. Each layer's bullet list is internally coherent.
- The CMDS component (conv multiplexing, creation, hydration, cancellation multiplex) is a well-scoped named unit. Four bullet points = four responsibilities. Easy to scope as a module.
- Treating HYDRATION as a dedicated cylinder icon signals it deserves its own component, not just a sub-feature. The second sketch gives it a distinct visual identity.

### What didn't work

- The second sketch initially appeared to have "SRC" as a scraper. The user clarified it is the base LLM backend. This changed my interpretation of the top diagram from "scraper → webchat" to "LLM backend → package module". Both are fire-and-forget patterns but the meaning of "do something" changes.

### What was tricky

- Distinguishing between "session" and "conversation": the sketches use `conv` and `session` in overlapping ways. Session objects appear in the Generic Layer bullets; `conv` appears in the topology diagram and CMDS feature list. I inferred that `session` is the Generic Layer's term (a transport-level object keyed by connection), while `conv` is the Backend/Client term (a logical LLM interaction). The distinction matters because one session can multiplex multiple conversations.
- The parenthetical "(I guess)" in "+ state machine (up to PROC I guess)" signals the author was uncertain. I treated this as an open design decision rather than a confirmed fact.
- The two sketches are at different levels of abstraction: BUILDING BLOCKS is a component taxonomy; the topology diagram is a runtime data flow. Reconciling them required mapping bullets to boxes and arrows, which was not always 1:1.

### Decisions made

- **SRC → Base Layer / LLM Backend**: The user corrected "scraper source" to "the base layer backend that does the actual llm work." Applied throughout.
- **Three-layer model**: Confirmed from three top-level bullets in BUILDING BLOCKS. Generic Layer is the reusable core.
- **`on_command(evt, session, connection)`**: Taken verbatim from the sketch annotation. This is the command handler signature.
- **"no need for persistent connection"**: Taken verbatim. Base Layer operates fire-and-forget.
- **CMDS as named component**: Four capabilities = four module responsibilities.
- **HYDRATION as first-class component**: Cylinder icon with distinct label in topology diagram.
- **"this also allows managing connection state from the backend side"**: Taken verbatim as a left annotation. Confirms bidirectional state management.
- **Six-phase plan**: Derived from layer structure + CMDS + HYDRATION + stale-connection and tick questions.
- **Derived capabilities** (not verbatim): `cancellation multiplex` → cancel one conversation without affecting others; `(etc.)` in metadata → model config and cost tracking; `conv multi-plexing` → single connection carries multiple conversations.

### Code review instructions

1. After initial implementation, verify the command handler signature `on_command(evt, session, connection)` is consistent across all backends.
2. Check that the event processing pipeline validates schemas at input boundaries (to mitigate event schema drift risk).
3. Confirm hydration replay is idempotent: replaying the same event log must produce identical UI state.
4. Validate that "cancellation multiplex" actually isolates: cancel conv A, verify conv B on same connection is unaffected.
5. Confirm the `no need for persistent connection` pattern holds: the base layer should be stateless with respect to connections.

### Verification and delivery

- Architecture analysis doc created with reasoning traces for every claim
- All verbatim items marked as `[VERBATIM]` citations with exact source text
- All derived items marked as `[DERIVED]` with reasoning chain
- Ticket: `EVT-STREAM-001`
- Doc: `analysis/01-architecture-analysis-reusable-base-framework-for-event-streaming-llm-applications.md`
- Uploaded to reMarkable at `/ai/2026/04/19/EVT-STREAM-001/`
- `docmgr doctor --ticket EVT-STREAM-001` → ✅ All checks passed
