---
Title: Diary
Ticket: SESSIONSTREAM-002
Status: active
Topics:
    - architecture
    - backend
    - framework
    - event-streaming
    - onboarding
    - systemlab
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/sessionstream-systemlab/README.md
      Note: Source of the current teaching-app framing referenced in the diary
    - Path: examples/chatdemo/chat.go
      Note: Evidence file used while deciding the new jobdemo/example-package split
    - Path: ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md
      Note: Primary design document created and updated during this ticket
ExternalSources: []
Summary: Chronological diary of creating the job-demo interactive-essay ticket, reviewing the current sessionstream/systemlab/example code, writing the intern-facing guide, validating the ticket, and publishing the bundle to reMarkable.
LastUpdated: 2026-04-23T10:55:00-04:00
WhatFor: Preserve the exact commands, decisions, validation steps, and delivery record for the job-demo interactive-essay design ticket.
WhenToUse: Use when resuming work on the ticket, reviewing how the design docs were prepared, or checking the exact evidence and upload steps behind the guide.
---


# Diary

## Goal

This diary records the setup and documentation work for `SESSIONSTREAM-002`: the ticket that defines the next `sessionstream` demo direction as a job-oriented interactive essay for `sessionstream-systemlab`.

It captures the evidence-gathering pass over the current substrate, the existing systemlab app, the current chat example, and the resulting design-doc / task / changelog / upload steps so a future contributor can see exactly how the initial planning package was assembled.

## Step 1: Create the ticket and write the interactive-essay design package

This step turned a brainstorming direction into a concrete repository-local ticket. The user had already steered the concept clearly: Systemlab’s overall direction was right, but they wanted the job demo itself to take the same “interactive essay” form rather than becoming a conventional jobs dashboard. So I treated this as a planning-and-framing task first, not an implementation task.

The most important thing I did here was to preserve the distinction between the **example domain** and the **teaching surface**. That gave the design doc its central split: `examples/jobdemo` should own reusable job-domain semantics, while `cmd/sessionstream-systemlab` should own the essay page and its explanatory UI. Without that split, the page would either become a page-specific one-off or the example package would accumulate presentation concerns that do not belong there.

### Prompt Context

**User prompt (verbatim):** "Ok, create a new ticket for the job demo interactive essay, reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.

Also, we want it to be textbook style, so while you shouldn't write it out, look at the writing guidelines for textbooks to keep it properly framed."

**Assistant interpretation:** Create a new repo-local sessionstream ticket for the job-demo interactive essay, study the existing substrate/systemlab/example code, write a very detailed intern-facing analysis/design/implementation guide plus ticket bookkeeping, keep the framing aligned with the textbook-writing guidance, and upload the resulting bundle to reMarkable.

**Inferred user intent:** Freeze the job-demo direction before implementation begins so the future work has a strong teaching shape, a clear architecture split, and a high-quality onboarding document that future interns can build from.

**Commit (code):** N/A (documentation-only planning step at the time of writing)

### What I did
- Read the textbook-authoring guidance from:
  - `/home/manuel/.pi/agent/skills/textbook-authoring/SKILL.md`
- Read the ticket/doc/upload/diary skills to keep the workflow aligned with the local conventions.
- Inspected the current repository structure and vocabulary with:

```bash
cd sessionstream && docmgr status --summary-only && docmgr ticket list
cd sessionstream && docmgr vocab list
```

- Read the current framework and teaching surfaces:
  - `sessionstream/README.md`
  - `sessionstream/cmd/sessionstream-systemlab/README.md`
  - `sessionstream/cmd/sessionstream-systemlab/server.go`
  - `sessionstream/cmd/sessionstream-systemlab/lab_environment.go`
  - `sessionstream/cmd/sessionstream-systemlab/static/index.html`
  - `sessionstream/cmd/sessionstream-systemlab/static/js/main.js`
  - `sessionstream/cmd/sessionstream-systemlab/chapters/phase-0-foundations.md`
  - `sessionstream/examples/chatdemo/chat.go`
  - `sessionstream/pkg/sessionstream/types.go`
  - `sessionstream/pkg/sessionstream/hub.go`
  - `sessionstream/pkg/sessionstream/projection.go`
  - `sessionstream/pkg/sessionstream/hydration.go`
  - `sessionstream/pkg/sessionstream/transport/ws/server.go`
- Created the new ticket and primary documents with:

```bash
cd sessionstream && docmgr ticket create-ticket \
  --ticket SESSIONSTREAM-002 \
  --title "Design the job demo interactive essay for sessionstream-systemlab" \
  --topics architecture,backend,framework,event-streaming,onboarding,systemlab

cd sessionstream && docmgr doc add \
  --ticket SESSIONSTREAM-002 \
  --doc-type design-doc \
  --title "Intern guide to designing and implementing the sessionstream job demo interactive essay"

cd sessionstream && docmgr doc add \
  --ticket SESSIONSTREAM-002 \
  --doc-type reference \
  --title "Diary"
```

- Wrote:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
  - `design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md`
  - `reference/01-diary.md`
- Related the key code/docs files into the design doc and diary with:

```bash
cd sessionstream && docmgr doc relate --doc ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/README.md:Repository purpose and ownership boundaries for the extracted framework" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/README.md:Current teaching-app contract, goals, and page/layout conventions" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/server.go:Current route/page registration model that the new essay page must extend" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/lab_environment.go:Current lab orchestration pattern and substrate-consumer wiring" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat.go:Existing example-package structure used as the comparison point for a future examples/jobdemo package" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go:Core Hub API and projection/publisher seams referenced throughout the design" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/types.go:Core SessionId, Command, Event, and Session vocabulary used in the guide" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/projection.go:UIProjection and TimelineProjection contracts that shape the essay's visible architecture" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration.go:HydrationStore and Snapshot seams needed for the reconnect/hydration chapter" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/transport/ws/server.go:Existing websocket transport behavior used for the reconnect and live-event teaching slice"

cd sessionstream && docmgr doc relate --doc ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/reference/01-diary.md \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md:Primary design document created and updated during this ticket" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/README.md:Source of the current teaching-app framing referenced in the diary" \
  --file-note "/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat.go:Evidence file used while deciding the new jobdemo/example-package split"
```
- Validated and uploaded with:

```bash
cd sessionstream && docmgr doctor --ticket SESSIONSTREAM-002 --stale-after 30

remarquee status
cd sessionstream && remarquee upload bundle --dry-run \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/index.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/tasks.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/changelog.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/reference/01-diary.md \
  --name "SESSIONSTREAM-002 Job Demo Interactive Essay Guide" \
  --remote-dir "/ai/2026/04/23/SESSIONSTREAM-002" \
  --toc-depth 2

cd sessionstream && remarquee upload bundle \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/index.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/tasks.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/changelog.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md \
  /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/reference/01-diary.md \
  --name "SESSIONSTREAM-002 Job Demo Interactive Essay Guide" \
  --remote-dir "/ai/2026/04/23/SESSIONSTREAM-002" \
  --toc-depth 2

remarquee cloud ls /ai/2026/04/23/SESSIONSTREAM-002 --long --non-interactive
```

### Why
- The user asked for a **new ticket**, not just an ad hoc markdown note.
- The work is inherently architectural and educational, so it needed to live in the repo-local docmgr workspace where future contributors will actually look for it.
- The textbook-authoring guidance mattered because the goal is not only to explain what to build, but to explain *why this is the right demo shape* and *how each layer of the current repository supports it*.

### What worked
- The current repository already had enough structure to support a strong design document without any code changes.
- Existing vocabulary was sufficient for the ticket topics, so no vocabulary extension was required.
- The current `sessionstream-systemlab` and `examples/chatdemo` code made the recommended ownership split very easy to explain concretely.

### What didn't work
- N/A in the design-writing step itself.

### What I learned
- The current `sessionstream` repo is already structurally ready for this direction. The real missing piece was not a new subsystem; it was the design narrative that says how `examples/jobdemo` and the systemlab essay page should relate.
- The most important design move is not “jobs instead of chat.” It is “interactive essay instead of dashboard.” Jobs are the domain; the essay is the real product shape.

### What was tricky to build
- The tricky part was keeping the guide concrete without accidentally sliding into writing the final essay content itself. The user wanted implementation/design guidance in textbook framing, not the completed chapter prose. That required staying at the level of structure, flow, domain model, and teaching rhythm rather than drafting the final page text.
- Another subtle point was preserving the boundary between example-package concerns and systemlab concerns. If that line is fuzzy in the guide, the later implementation is likely to sprawl.

### What warrants a second pair of eyes
- Review the proposed ownership split (`examples/jobdemo` vs `cmd/sessionstream-systemlab`) to confirm it is the right long-term boundary.
- Review the proposed chapter structure to make sure it teaches the right conceptual sequence before code lands.
- Review the recommendation to treat raw event/state inspectors as persistent first-class UI rather than optional “debug” tabs.

### What should be done in the future
- Implement the ticket in focused slices, starting with the reusable `examples/jobdemo` package.
- Add the new essay page to `cmd/sessionstream-systemlab` after the example package exists and is testable.
- Re-upload the ticket bundle after implementation lands so the reMarkable copy reflects the built result as well as the design.

### Code review instructions
- Review in this order:
  1. `sessionstream/ttmp/2026/04/23/SESSIONSTREAM-002--design-the-job-demo-interactive-essay-for-sessionstream-systemlab/index.md`
  2. `.../tasks.md`
  3. `.../design-doc/01-intern-guide-to-designing-and-implementing-the-sessionstream-job-demo-interactive-essay.md`
  4. `.../reference/01-diary.md`
- Validate with:

```bash
cd sessionstream && docmgr doctor --ticket SESSIONSTREAM-002 --stale-after 30
```

### Technical details
- Key evidence files consulted while writing this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/README.md`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/README.md`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/server.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/lab_environment.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/types.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/projection.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/transport/ws/server.go`
