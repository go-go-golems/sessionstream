# Changelog

## 2026-04-20 (continued)

- Added targeted vs freeform widget distinction to design guide
- **Targeted widgets**: tied to specific textbook concepts, placed near prose they illustrate, validate chapter claims
- **Freeform widgets**: enable experimentation, surface unanticipated behavior, live in sidepane
- Updated widget taxonomy with type column (Targeted, Freeform, Core)
- New freeform widgets: EvidenceLog, StoreInspector, ClientExplorer, SessionExplorer
- Updated Appendix B with split into Targeted vs Freeform widgets per phase
- Updated Section 4 implementation priorities to reflect widget type order
- Core principle: widgets both illustrate the textbook AND enable exploration beyond it

### Key Validation Findings

| Phase | Current State | Design Match |
|-------|---------------|-------------|
| Phase 1 | JSON well-formatted, syntax highlighted | Need rendered view for teaching |
| Phase 3 | Client A/B side-by-side ✓ | Keep layout, add state indicators |
| Phase 5 | Controls far from chapter (6+ viewport heights) | Must inline controls in chapter |

### Phase 0 Implementation Complete

**Chapter rewrite:**
- phase-0-foundations.md: 690 → ~480 lines
- Removed AI-slop, added foundational prose
- Structure: story, clean-room, systemlab, vocabulary, import-cycle, map, validation

**StatusIndicator widget:**
- overview.js: StatusIndicator showing phase progress
- Phase 0: status-active (blue) - current phase
- Phases 1-5: status-complete (green) - implemented
- Phase 6: status-pending (gray) - not implemented
- Phase section numbers fixed (0-based, displayed as Phase 0, Phase 1, etc.)

**CSS updates:**
- Added .phase-progress, .phase-item, .phase-icon, .phase-name
- Added status-active, status-complete, status-pending styling

**Design guide updates:**
- Added Section 3.0 for Phase 0 layout design
- Added StatusIndicator to phase-specific widgets table
- Documented chapter integration patterns

### Phase 1 Implementation Complete

**Chapter rewrite (EVT-STREAM-011):**
- phase-1-command-to-projection.md: 698 → ~480 lines
- Peter Norvig style: foundational prose, pseudocode, clear explanations
- Structure: Command Path → Handlers publish events → Ordinals → Projections → Store → Try sections → Trace → API Reference

**ToggledViewer widget pattern (EVT-STREAM-010):**
- phase1.js: JSON ↔ Rendered toggle for Trace, Session + UI Events, Snapshot
- Rendered views are much more interpretable:
  - Trace: step numbers, colored kind badges (command/session/handler/projection), messages
  - Session: session header, UI events with icons (●/→/✓), formatted details
  - Snapshot: session info, ordinal, entity with properties

**Partial HTML (phase1.html):**
- Added .evidence-panels container
- Added .panel-header with toggle buttons

**CSS (app.css):**
- Toggle button styles (.toggle-btn active/inactive)
- Rendered view styles for trace, session, snapshot panels

**Screenshot:** phase1-rendered-views.png shows working rendered views