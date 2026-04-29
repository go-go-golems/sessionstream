# Changelog

## 2026-04-19

- Initial workspace created


## 2026-04-19

Created clean-room ticket from 2026-04-19 sketches: added verbatim transcription (reference) and architecture analysis (design) covering the three-layer Backend/Generic/Client model, multiplexed websocket fan-out, conv-id routing, hydration, and ten captured open questions.


## 2026-04-19

Annotated architecture analysis with citation blockquotes (> Trace: ...) explaining the source-image evidence behind every substantive claim.


## 2026-04-19

Applied four author clarifications: Session ≡ Conversation (canonical SessionId); backend cannot read connection objects; single backend-event stream with two user-supplied projections (UI + Timeline); Command shape is (Name, Payload, ConnectionId, SessionId), synchronous. Broadened example commands/events to cover agentic/document workflows (TaskCompleted, DocumentWritten, RunAgent, ReplayScrape) — not just chat.


## 2026-04-19

Removed all 'conversation' wording per author preference; framework exposes Session/SessionId only. Source-quoting traces preserve the original conv-id token via bracket-substitution; verbatim source remains in the transcription doc.


## 2026-04-19

Added technical architecture doc (design/02): full Go + TypeScript API specification, application-owned watermill bus with substrate-side ordinal stamping, defensive-copy TimelineView passed to both projections, single TimelineEntity with Tombstone, Hub.Submit collapsed from the 'direct' transport, ChatMessage as the canonical worked example, 10 carried open questions.


## 2026-04-19

Added a detailed comparison against the current `pinocchio/pkg/webchat` implementation (design/03): identified direct donors (transport split, stream-id-derived ordinals, non-blocking websocket fan-out, hydration-store implementations, idle eviction), key mismatches (dual identity model, SEM-first projection chain, chat-specific command API, global timeline registries), and a recommended extraction strategy for building EVT-STREAM-002 as a clean substrate rather than renaming webchat in place.


## 2026-04-19

Added independent reuse analysis (design/04): file-level audit of pinocchio/pkg/webchat against the new framework, identifying 70/20/10 split (substrate / chat-app / discard), 7 missing-feature recommendations (idempotency, hydration tail, throttling, liveness, optional stream partitioning, UI projection patterns, interim/final phases), and a 9-step extraction roadmap. Independent of design/03 by directive.

