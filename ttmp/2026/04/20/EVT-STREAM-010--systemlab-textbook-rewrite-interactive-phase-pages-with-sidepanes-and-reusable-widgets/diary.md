# Diary: Systemlab Textbook Rewrite (EVT-STREAM-010)

## 2026-04-20T09:15 - Ticket Created

**What happened:**
- Read source brief from `EVT-STREAM-009` about rewriting Systemlab chapters and phase pages
- Created ticket workspace EVT-STREAM-010
- Started analysis of current layouts

**Key insight from brief:**
> "minimize scrolling, the interactive part should be close to the text that relates it, or in a sidepane"

## 2026-04-20T09:20 - Design Guide Created

**What happened:**
- Created comprehensive design guide with ASCII wireframes for all 6 phases
- Documented widget taxonomy for React migration
- Defined layout patterns per phase

**Files created:**
- `design/01-systemlab-textbook-design-guide.md` (25KB design doc)

## 2026-04-20T09:25 - Live Testing Started

**What happened:**
- Opened Systemlab at http://127.0.0.1:18105/
- Tested Phase 5 layout - had to scroll 6+ viewport heights to reach controls
- Confirmed the core problem: chapter text is very long, controls are at the bottom

**Screenshot analysis:**
- Phase 5 chapter text spans ~4000px+
- Controls only visible after significant scrolling

## 2026-04-20T09:30 - User Feedback: ToggledViewer

**User said:** "the json view only is quite hard to read. It should probably have a widget view, and toggle between json and rendered view"

**What I did:**
- Added `ToggledViewer` widget to taxonomy
- Added `RenderedViewer` for human-readable annotated view
- Documented icon system (● ✓ ✗ ○ ▶ →)
- Created Appendix C with UX guidelines

## 2026-04-20T09:32 - Phase 1 Live Testing

**What happened:**
- Navigated to Phase 1
- Submitted a command
- Analyzed output

**Finding:** JSON is actually well-formatted with syntax highlighting. The problem isn't readability—it's **interpretability**. JSON shows structure but not meaning.

**Examples of transformation:**
- `{"step": 1, "kind": "command"}` → `① Command received`
- `{"event": "LabFinished"}` → `✓ Finished`
- Ordinals should be dimmed/less prominent

## 2026-04-20T09:34 - Phase 3 Live Testing

**What happened:**
- Navigated to Phase 3
- Scrolled to find client controls
- Analyzed Client A/B layout

**Validated findings:**
- Client A and Client B ARE side-by-side in a 2-column grid ✓
- Each has Connect/Subscribe/Disconnect buttons ✓
- Status shown in dark panel ("Client A idle.") ✓
- Backend Trace below in 2-column layout ✓

**Design improvements for Phase 3:**
- State indicators (○ Disconnected, ● Connected) at top of each client panel
- JSON/Rendered toggle for Backend Trace
- Collapsible evidence panel so chapter stays accessible

## User Feedback: Widget Purpose Clarified

**User said:** "for the rewrite guide, the core principle of the widgets is to illustrate the content of the textbook, and give the freedom to the user to experiment on their own (this doesn't need to be within the same widget, since targeted widgets are more useful for explaining, but widgets that allow and encourage experimentation will be more freeform. that will also allow the user to surface behavior that is not necessarily covered in the textbook or will be covered later). the goal is both pedagogical and covering the system."

**What this means:**
- Widgets serve TWO purposes, not one
- **Illustrate** = targeted widgets tied to specific prose sections, validate chapter claims, make invariants visible
- **Enable exploration** = freeform widgets for experimentation, surface unanticipated behavior, go beyond the text
- These can be separate widgets - targeted is better for explaining, freeform is better for exploration
- Both are necessary for a complete learning system

**Updated design guide:**
- Added section 2.1: "Core Principle: Widgets Illustrate the Textbook and Enable Experimentation"
- Added distinction: Targeted vs Freeform widget types
- New widget taxonomy with Type column (Targeted, Freeform, Core)
- New freeform widgets: EvidenceLog, StoreInspector, ClientExplorer, SessionExplorer
- Updated composition patterns to show widget types

## Files Modified (Update)

1. `index.md` - Updated Key Design Decisions section with widget dual purpose
2. `tasks.md` - Updated
3. `changelog.md` - Added entry for targeted vs freeform distinction
4. `design/01-systemlab-textbook-design-guide.md` - Added section 2.1, updated widget taxonomy, updated composition patterns

**Design validated against live Systemlab:**
- Phase 1: Need rendered view for teaching (JSON readable but not interpretable)
- Phase 3: Keep 2-column client layout, add state indicators
- Phase 5: Must inline controls in chapter to eliminate scrolling

**Next steps:**
1. Define ToggledViewer API - how JSON transforms to rendered view
2. Define widget API contracts as TypeScript interfaces
3. Create React project scaffold with Vite
4. Implement Phase 1 as first migrated page

## Challenges Encountered

1. **docmgr ticket creation**: Had to use full path due to duplicate EVT-STREAM-010 tickets
2. **Large ASCII wireframes**: Matching exact whitespace was difficult - ended up rewriting sections
3. **Scrolling distance validation**: Had to take multiple screenshots at different scroll positions to understand full page height

## Files Modified

1. `index.md` - Updated with validation results and key design decisions
2. `tasks.md` - Updated with completed/in-progress items
3. `changelog.md` - Updated with detailed progress
4. `design/01-systemlab-textbook-design-guide.md` - Added ToggledViewer pattern and UX guidelines

## Screenshots Taken

- `phase5-page-initial.png` - Phase 5 top
- `phase5-page-scroll1.png` - Phase 5 after 500px scroll
- `phase5-page-scroll2.png` - Phase 5 after 1500px scroll
- `phase5-page-scroll3.png` - Phase 5 Try sections
- `phase5-page-scroll4.png` - Phase 5 near end of chapter
- `phase5-page-scroll5.png` - Phase 5 more chapter
- `phase5-page-scroll6.png` - Phase 5 controls visible ✓
- `phase1-page-initial.png` - Phase 1 top
- `phase1-page-after-submit.png` - Phase 1 with results
- `phase1-page-snapshot.png` - Phase 1 snapshot panel
- `phase3-page-initial.png` - Phase 3 top
- `phase3-page-controls.png` - Phase 3 chapter
- `phase3-page-clients.png` - Phase 3 more chapter
- `phase3-page-bottom.png` - Phase 3 clients visible ✓