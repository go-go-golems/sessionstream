---
Title: 'Systemlab Chapter Rewrite: Peter Norvig Style Textbook with Integrated Widgets'
Ticket: EVT-STREAM-011
Status: active
Topics:
    - documentation
    - systemlab
    - teaching
    - textbook-writing
    - onboarding
    - widgets
DocType: index
Intent: medium-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-010--systemlab-textbook-rewrite-interactive-phase-pages-with-sidepanes-and-reusable-widgets/design/01-systemlab-textbook-design-guide.md
      Note: Widget taxonomy and layout design from EVT-STREAM-010
    - Path: /home/manuel/.pi/agent/skills/textbook-authoring/SKILL.md
      Note: Peter Norvig style guide with examples
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/chapters/phase-0-foundations.md
      Note: Current Phase 0 chapter (690 lines)
ExternalSources: []
Summary: "Rewrite all 6 Systemlab chapters from AI-slop prose to Peter Norvig style textbook with integrated interactive widgets. Based on EVT-STREAM-010 widget design and textbook-authoring skill."
LastUpdated: 2026-04-20T10:05:00-04:00
WhatFor: "Create a high-quality textbook that teaches evtstream architecture through foundational prose, concrete examples, and integrated interactive widgets."
WhenToUse: "When executing the chapter rewrite for EVT-STREAM-011."
---

# Systemlab Chapter Rewrite: Peter Norvig Style Textbook with Integrated Widgets

## Overview

Rewrite all 6 Systemlab chapters (3,215 current lines → ~3,000 target lines) from AI-slop prose to Peter Norvig style textbook with integrated interactive widgets.

**Core principles:**
1. Build foundational understanding before implementation
2. Explain **why** things are, not just what or how
3. Use concrete examples, pseudocode, diagrams to ground abstract concepts
4. Integrate widgets that both illustrate the text AND enable exploration

## Key Links

- **Design Guide**: [design/01-chapter-rewrite-guide.md](./design/01-chapter-rewrite-guide.md)
- **Widget Design**: [EVT-STREAM-010 design guide](../EVT-STREAM-010--systemlab-textbook-rewrite-interactive-phase-pages-with-sidepanes-and-reusable-widgets/design/01-systemlab-textbook-design-guide.md)
- **Writing Style**: [textbook-authoring skill](../../../../.pi/agent/skills/textbook-authoring/SKILL.md)

## Status

Current status: **active** - Design complete, ready for execution

## Phase Rewrite Order

| Order | Phase | Current | Target | Key Widgets |
|-------|-------|---------|--------|-------------|
| 1 | Phase 0: Foundations | 690 | 500 | ChapterViewer, StatusIndicator |
| 2 | Phase 1: Command → Event → Projection | 698 | 600 | ScenarioControls, TraceTimeline, ToggledViewer |
| 3 | Phase 2: Ordering and Ordinals | 582 | 500 | OrdinalTimeline, ConsumerTrace |
| 4 | Phase 3: Hydration and Reconnect | 410 | 450 | WebSocketClient, ClientComparison |
| 5 | Phase 4: Chat Example | 447 | 500 | LiveStreamPreview, SessionViewer |
| 6 | Phase 5: Persistence and Restart | 388 | 450 | ModeToggle, PrePostComparison |

**Total**: 3,215 → ~3,000 lines (7% reduction, but more substance per line)

## Chapter Structure Template

Each chapter follows this structure:

```
## N. [Section Title]

[Opening paragraph - state purpose, what reader should understand]

[Conceptual foundation - why this design exists]

[Code or pseudocode - grounded in actual implementation]

[Tables or diagrams - comparisons, sequences]

[Try section with inline controls - targeted widget integration]

[Key points bullet list - surface important takeaways]
```

## Widget Integration Patterns

**1. Inline in Try Sections (Targeted)**
- Controls directly adjacent to explanation
- Reader acts immediately after understanding

**2. Evidence Panel (Targeted + Core)**
- Visible without scrolling
- JSON/Rendered toggle on each panel

**3. Sidepane (Freeform)**
- Accessible for exploration
- Doesn't compete with textbook flow

## Writing Style

**Opening paragraphs**: State the purpose and what the reader should understand. Not a summary—a statement of intent.

**Conceptual explanations**: Build the mental model. Explain why a design exists before showing how it works.

**Bullet lists**: Each bullet is a complete sentence that could stand alone.

**Anti-patterns to eliminate**:
- "In this chapter, we will..."
- "It is worth noting that..."
- "This is one of those patterns..."
- Vague bullet lists
- Overused qualifiers

## Next Steps

1. [ ] Rewrite Phase 0 chapter (foundations)
2. [ ] Rewrite Phase 1 chapter (most important - command path)
3. [ ] Rewrite Phase 2 chapter (ordering - medium complexity)
4. [ ] Rewrite Phase 3 chapter (reconnect - already has good layout)
5. [ ] Rewrite Phase 4 chapter (chat - application layer)
6. [ ] Rewrite Phase 5 chapter (persistence - completion)