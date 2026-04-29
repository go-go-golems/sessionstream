# Diary: Phase 0 Rewrite (EVT-STREAM-010 + EVT-STREAM-011)

## 2026-04-20T10:15 - Starting Phase 0 Rewrite

Working both tickets in lockstep. Phase 0 is foundations - shortest phase, good starting point.

**Plan:**
1. Update EVT-STREAM-010 with chapter integration patterns
2. Rewrite Phase 0 chapter
3. Update partial/layout for Phase 0
4. Commit

## 2026-04-20T10:20 - Analyzing Current Phase 0

**Current State:**
- 690 lines (target: ~500)
- Some good content, some AI-slop
- 21 numbered sections is too many
- "Things to try" are prose, not inline controls

**What's Good:**
- Opening paragraph is actually decent
- Section 3 (pkg/evtstream creation) is solid
- Section 4 (Systemlab separation) is good
- Import-cycle lesson (section 8) is well-done
- Invariants section is clear

**What Needs Rewriting:**
- AI-slop phrases: "It is tempting to look at..." "That is what Phase 0 is about"
- Too many sections (21)
- Bullet lists verbose
- Missing inline controls in Try sections
- Missing "why" explanations in some places

## 2026-04-20T10:25 - Updating EVT-STREAM-010

Adding chapter integration patterns to the design guide.

**Changes:**
- Add Section 2.1: "Core Principle: Widgets Illustrate the Textbook and Enable Experimentation" (already exists)
- Add Section 2.6: "Chapter Integration Patterns" - how widgets connect to prose
- Update composition patterns to show inline Try sections
- Add Phase 0 specific guidance

## 2026-04-20T10:30 - Writing New Phase 0

**Structure:**
1. What this framework is trying to do
2. Why we created pkg/evtstream instead of renaming pkg/webchat
3. Why Systemlab is a separate app
4. The vocabulary Phase 0 teaches
5. The import-cycle lesson
6. The directory map and what it means
7. Validation commands and why they matter

**Target: ~500 lines (from 690)**

## 2026-04-20T10:45 - Phase 0 Draft Complete

**Draft structure:**
- Opening (strong purpose statement)
- Section 1: The Story (why architecture matters)
- Section 2: Why pkg/evtstream (clean-room abstraction)
- Section 3: Why Systemlab is Separate (boundary proof)
- Section 4: Core Vocabulary (SessionId, ConnectionId, Command, Event, Projections, HydrationStore)
- Section 5: The Import-Cycle Lesson (real example)
- Section 6: The Directory Map (ownership)
- Section 7: Validation Commands (mechanized trust)
- Key Points
- File References

**Widgets for Phase 0:**
- ChapterViewer (Core)
- StatusIndicator (Targeted: shows phase progress)
- No freeform widgets needed (foundations only)

## 2026-04-20T11:00 - Updating Partial HTML

Phase 0 partial updated to show:
- Chapter with proper structure
- StatusIndicator widget showing phase progress
- Minimal controls (no rich interactions for foundations)

## 2026-04-20T11:15 - Phase 0 Rewrite Complete

**Phase 0 chapter rewritten and StatusIndicator widget implemented.**

### Chapter Rewrite (phase-0-foundations.md)
- Reduced from 690 lines to ~480 lines
- Removed AI-slop phrases and wandering preambles
- Added foundational prose explaining "why" before "what"
- Structure:
  1. What this framework is trying to do (story, not features)
  2. Why we created pkg/evtstream (clean-room vs donor code)
  3. Why Systemlab is separate (boundary proof)
  4. Core vocabulary (SessionId, ConnectionId, Command, Event, Projections)
  5. The import-cycle lesson (real architectural lesson)
  6. The directory map (ownership, not just listing)
  7. Validation commands (mechanized trust)
  8. Key Points (surface the takeaways)

### StatusIndicator Widget
- Implemented in overview.js
- Shows phase progress with ●/○ indicators
- Phase 0: status-active (blue border) - current phase
- Phases 1-5: status-complete (green border) - implemented
- Phase 6: status-pending (gray, dimmed) - not implemented
- CSS styling for status classes

### Design Guide Updates (EVT-STREAM-010)
- Added Section 3.0 for Phase 0 layout design
- Added StatusIndicator to phase-specific widgets table
- Documented chapter integration patterns

### Verification
- Server running at http://127.0.0.1:18105/
- StatusIndicator visible and properly styled
- Chapter content loading correctly
- Screenshot phase0-status-scrolled.png shows working widget

## 2026-04-20T14:50 - Phase 0 Complete, No Git Repository

**Note:** No git repository found in workspace. Changes saved to disk but not committed.

**Phase 0 work complete:**
- Chapter rewritten in Peter Norvig style
- StatusIndicator widget implemented
- Design guide updated

**Proceeding to Phase 1...**

## 2026-04-20T15:00 - Phase 1 Rewrite Complete

### Chapter Rewrite (phase-1-command-to-projection.md)
- Reduced from 698 lines to ~480 lines
- Removed AI-slop, added Peter Norvig style foundational prose
- Structure:
  1. What this chapter is about (reading guide)
  2. The Command Path (Hub.Submit pseudocode)
  3. Why handlers publish events (key design decision)
  4. Ordinals: why order matters
  5. Two projections, one source, different jobs (with table)
  6. The in-memory hydration store
  7. Reading the page (controls → checks → trace → session/events → snapshot)
  8. Things to try (6 Try sections)
  9. Reading the trace (annotated pseudocode)
  10. What checks summarize
  11. The command registry: simple on purpose
  12. Key Points
  13. API Reference
  14. File References

### Widget Implementation (phase1.js)
**ToggledViewer pattern working!**
- Added JSON ↔ Rendered toggle buttons to Trace, Session + UI Events, and Snapshot panels
- Rendered views are much more interpretable than raw JSON:
  - Trace: step numbers, colored kind badges, messages
  - Session + UI Events: session header, event icons (●/→/✓), formatted details
  - Snapshot: session info, entity with properties

### Partial HTML Updates (phase1.html)
- Added `.evidence-panels` container for grouped output panels
- Added `.panel-header` with h3 + view-toggle div
- Added toggle buttons: [Rendered] [JSON]

### CSS Updates (app.css)
- `.evidence-panels`, `.panel-header`, `.view-toggle`
- `.toggle-btn` (active/inactive states)
- Rendered view styles:
  - `.trace-rendered`, `.trace-step`, `.trace-step-num`, `.trace-step-kind`, `.trace-step-message`
  - `.session-rendered`, `.session-header`, `.ui-events-list`, `.ui-event-item`
  - `.snapshot-rendered`, `.snapshot-session`, `.snapshot-entity`

### Verification
- Server running at http://127.0.0.1:18105/
- Phase 1 page renders correctly
- Chapter content shows in ChapterViewer
- Toggle buttons work (JSON ↔ Rendered)
- Rendered trace shows: step numbers, kind badges (command/session/handler/projection), messages
- Rendered session shows: session header, UI events with icons and text
- Rendered snapshot shows: session info, ordinal, LabMessage entity with properties
- Screenshot: phase1-rendered-views.png

### Next Steps
1. [ ] Phase 2 chapter rewrite (Ordering and Ordinals)
2. [ ] Phase 3 chapter rewrite (Hydration and Reconnect)
3. [ ] Phase 4 chapter rewrite (Chat Example)
4. [ ] Phase 5 chapter rewrite (SQL / Restart)

## 2026-04-20T15:20 - Starting Phase 2: Ordering and Ordinals

**Note:** Dynamic widgets deferred - implementing widgets later.

**Current Phase 2 chapter:** Read and assessed (560 lines, 19 sections)
**Target:** ~500 lines
**Result:** Rewrote to ~420 lines, 12 sections + Key Points + API Reference + File References

**Core teaching goal:** Why ordinals matter, how they're assigned, and what happens when events arrive out of order.

## 2026-04-20T15:40 - Phase 1 Restructured

**Restructured Phase 1 following first principles pedagogy (same as Phase 2 rewrite).**

**Old structure:**
1. What this chapter is about
2. The Command Path (Hub pseudocode)
3. Why handlers publish events
4. ...

**New structure:**
1. What this chapter is about (stated purpose)
2. The problem: how does a client ask the server to do something? (why commands exist)
3. The Hub: routing, not logic (Hub code, after establishing the problem)
4. Why handlers publish events instead of returning values (the critical design choice)
5. What happens when you click Submit (the full sequence diagram)
6. Ordinals
7. Two projections, one source
8. The hydration store
9. Reading the page
10. Things to try
11. What the checks prove

**Key improvements:**
- Starts with WHY commands exist (vs direct function calls)
- Explains WHY handlers publish events (before showing the Hub code)
- Shows the full sequence diagram (what happens when you click Submit)
- Concrete code and real examples
- Clear "Key Points" closing

**Result:** ~310 lines, 10 sections + Key Points + API Reference + File References

## 2026-04-20T15:50 - Phase 3 Rewrite Complete

**Restructured Phase 3 following first principles pedagogy.**

**Old chapter:** 680 lines, 16 sections, verbose with AI slop
**New chapter:** ~280 lines, 10 sections + Key Points + API Reference + File References

**Structure:**
1. What this chapter is about
2. Why reconnect is a framework problem (not just UI)
3. The central rule: snapshot before live (with consequence comparison)
4. What a subscribe looks like (sequence diagram)
5. ConnectionId vs SessionId
6. The transport architecture
7. Why this matters for correctness
8. The Phase 3 page
9. Things to try
10. What the checks prove
11. Common mistakes

**Removed:**
- "Up to this point, you have seen..."
- "At the time of writing, this chapter is ahead of implementation..."
- Philosophical throat-clearing

## 2026-04-20T15:55 - Phase 4 and 5 Rewrites Complete

**Both phases rewritten following first principles pedagogy.**

### Phase 4: Chat Example
**Old:** 560 lines, verbose with AI slop
**New:** ~180 lines, 7 sections + Key Points + API Reference + File References

**Structure:**
1. What this chapter is about
2. Why chat as the first example?
3. The framework and the example are separate
4. The chat flow
5. Why two projections makes sense in chat
6. Stop behavior
7. The Phase 4 page
8. Things to try

**Key teaching:**
- Chat is a consumer of evtstream, not part of it
- Same patterns as Phase 1 (command → handler → events → projections)
- Stop is normal, not exceptional

### Phase 5: Persistence and Restart
**Old:** 600+ lines, verbose with AI slop
**New:** ~230 lines, 10 sections + Key Points + API Reference + File References

**Structure:**
1. What this chapter is about
2. Why restart changes everything
3. The central rule
4. What the hydration store promises
5. Why transactional Apply matters
6. What restart looks like
7. What the cursor does
8. Memory vs SQL: semantic equivalence
9. How restart connects to reconnect
10. Common mistakes

**Key teaching:**
- Cursor and timeline must survive restart together
- Apply must be atomic (state + cursor)
- Memory and SQL must have same semantics
- Persistence and reconnect are the same story

**View at:**
- Phase 4: http://127.0.0.1:18105/#phase4
- Phase 5: http://127.0.0.1:18105/#phase5

### All Phases Complete!
| Phase | Lines | Status |
|-------|-------|--------|
| Phase 0 | ~480 | ✅ |
| Phase 1 | ~310 | ✅ (restructured) |
| Phase 2 | ~300 | ✅ (restructured) |
| Phase 3 | ~280 | ✅ |
| Phase 4 | ~180 | ✅ |
| Phase 5 | ~230 | ✅ |

## 2026-04-20T14:50 - Starting Phase 1: Command → Event → Projection

Phase 1 is the most important phase - it shows the full path through the Hub. This is where readers first see the framework "feel alive."

**Current Phase 1 chapter:** 698 lines
**Target:** ~600 lines

**Core teaching goal:** Show the command → event → projection flow. Make canonical events feel real.

**Key content from chapter rewrite guide:**
1. The Command Path (Hub.Submit, handlers publish events not return values)
2. Why handlers publish events (the key design decision)
3. Ordinals: why order matters
4. Two projections, one source, different jobs
5. The in-memory hydration store (teaches durable-state shape)
6. Reading the page (controls → checks → trace → session/events → snapshot)
7. Things to try (6 Try sections with inline controls)
8. Reading the trace
9. What checks summarize
10. API Reference

**Widget integration for Phase 1:**
- Targeted: ScenarioControls, TraceTimeline, ToggledViewer (×2), CheckList
- Freeform: SessionExplorer, EvidenceLog