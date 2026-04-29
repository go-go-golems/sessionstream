---
Title: Investigation Diary
Ticket: EVT-STREAM-003
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - llm
    - agents
    - backend
    - implementation
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md
      Note: Primary deliverable authored during this ticket.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/changelog.md
      Note: Ticket history updated during authoring.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-003 covering ticket creation, evidence gathering, documentation authoring, validation with docmgr doctor, and reMarkable delivery."
LastUpdated: 2026-04-19T21:08:00-04:00
WhatFor: "Record the concrete steps, commands, and decisions taken while creating the implementation ticket and intern-facing onboarding guide."
WhenToUse: "When reviewing how the deliverable was produced, continuing the ticket later, or verifying validation/upload steps."
---

# Investigation Diary

## Goal

Capture how EVT-STREAM-003 was created, what evidence was gathered, what documentation was written, how it was validated, and how it was delivered to reMarkable.

## Step 1: Create the implementation ticket and author the guide

The main goal of this step was to turn the already-existing design work into a new implementation ticket with a concrete, intern-friendly deliverable. The earlier EVT-STREAM material already contained a clean-room architecture and a reuse analysis, so the work here was not to rediscover the design. It was to synthesize it into a practical build manual: where the code should likely live, what files should be created first, which donor code is safe to adapt, and what invariants a new engineer must preserve.

A second goal was to make the deliverable self-contained. A new intern should not need to piece the system together from five unrelated markdown files and a large existing webchat package. The new guide therefore emphasizes reading order, runtime explanation, file-level implementation guidance, phased milestones, and explicit “do not repeat this mistake” warnings sourced from the current `pinocchio/pkg/webchat` implementation.

### Prompt Context

**User prompt (verbatim):** "ok, now make a new ticket for the actual implementation. then reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a follow-up ticket specifically for implementation, then write a long-form onboarding and implementation blueprint detailed enough for a new engineer to start building the framework correctly.

**Inferred user intent:** The user wants the design work converted into an execution-ready package of documentation that can onboard implementation contributors quickly and reduce architecture drift.

**Commit (code):** N/A — documentation-only work in the ticket workspace.

### What I did

- Created the new ticket workspace and seeded the primary design doc plus diary doc:

```bash
docmgr ticket create-ticket --ticket EVT-STREAM-003 --title "Event Streaming LLM Framework Implementation Plan and Intern Onboarding Guide" --topics architecture,framework,event-streaming,llm,agents,backend,implementation,onboarding
docmgr doc add --ticket EVT-STREAM-003 --doc-type design-doc --title "Implementation Plan and Intern Onboarding Guide"
docmgr doc add --ticket EVT-STREAM-003 --doc-type reference --title "Investigation Diary"
```

- Inspected the created ticket files.
- Gathered evidence from:
  - EVT-STREAM-002 technical architecture doc,
  - EVT-STREAM-002 webchat reuse analysis,
  - `pinocchio/pkg/webchat` donor files,
  - `pinocchio/pkg/persistence/chatstore`,
  - `pinocchio/go.mod`.
- Wrote the primary guide at:
  - `design-doc/01-implementation-plan-and-intern-onboarding-guide.md`
- Updated:
  - `index.md`
  - `tasks.md`
  - `changelog.md`
- Related key source/design/code files to the design doc using `docmgr doc relate`.

### Why

- The technical design already existed, but there was no implementation-facing document telling a new engineer what to build first.
- The codebase contains strong donor code in `pinocchio/pkg/webchat`, but also traps that would produce the wrong substrate if copied wholesale.
- A separate implementation ticket reduces confusion between:
  - architecture design,
  - reuse analysis,
  - and actual execution planning.

### What worked

- The clean-room technical architecture doc was detailed enough to serve as the primary API source of truth.
- The existing webchat package provided concrete donor patterns for:
  - Watermill seams,
  - consumption-time ordinal logic,
  - websocket backpressure handling,
  - idle eviction.
- The ticket workspace and docs were created cleanly with `docmgr`.

### What didn't work

- Nothing failed during ticket creation or document authoring.
- One validation issue appeared later during `docmgr doctor` because the vocabulary did not yet include the `implementation` and `onboarding` topic slugs.

### What I learned

- The architectural picture is now mature enough that the hard part is no longer design ambiguity; it is implementation ordering and scope discipline.
- The most practical recommendation for the first code drop is to implement in `pinocchio/pkg/evtstream`, not in `le-chat`, because `pinocchio` already has a Go module and Watermill dependency nearby.
- The cleanest way to teach the system is to organize it around a few core invariants:
  - one `SessionId`,
  - canonical backend events,
  - sibling UI/timeline projections,
  - consumption-time ordinals.

### What was tricky to build

The tricky part was not collecting evidence; it was choosing the right level of prescriptiveness. The guide needed to be detailed enough that a new intern can start coding from it, but not so rigid that it pretends every unresolved question has already been settled. The approach I used was:

1. treat EVT-STREAM-002 design/02 as the API source of truth,
2. treat EVT-STREAM-002 design/03 as the donor/rejection guide,
3. recommend concrete defaults where the repository state makes one choice clearly cheaper,
4. keep open questions visible only where they do not block the first implementation slices.

### What warrants a second pair of eyes

- The recommendation to implement first in `pinocchio/pkg/evtstream` rather than a new standalone module.
- The decision to defer TS-client implementation from the earliest milestone.
- The exact file split between `hub.go`, registries, and bus consumer.

### What should be done in the future

- Begin Phase 0 scaffolding in `pinocchio/pkg/evtstream`.
- Confirm the remaining open implementation policy questions:
  - `SessionId` allocation,
  - liveness/tick protocol,
  - first-milestone TS scope,
  - eventual module extraction strategy.

### Code review instructions

- Start with `design-doc/01-implementation-plan-and-intern-onboarding-guide.md`.
- Verify that every major recommendation is grounded either in EVT-STREAM-002's technical architecture or in concrete donor code.
- Check `tasks.md` to see the intended phase ordering.
- Cross-check the cited donor files if any recommendation seems too strong.

### Technical details

Evidence-gathering commands included:

```bash
docmgr status --summary-only
docmgr ticket list
docmgr doc list --ticket EVT-STREAM-003
find le-chat/ttmp/2026/04/19/EVT-STREAM-001--reusable-base-framework-for-event-streaming-llm-applications -maxdepth 2 -type f | sort
find le-chat/ttmp/2026/04/19/EVT-STREAM-002--event-streaming-llm-framework-clean-room-architecture-2026-04-19 -maxdepth 3 -type f | sort
find pinocchio/pkg/webchat -type f | sort
nl -ba <file> | sed -n '<range>'
```

Key files inspected:

- `.../EVT-STREAM-002/design/02-technical-architecture-event-streaming-llm-framework.md`
- `.../EVT-STREAM-002/design/03-webchat-reuse-analysis-vs-pinocchio-webchat.md`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/go.mod`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/webchat/doc.go`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/webchat/http/api.go`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/webchat/stream_backend.go`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/webchat/stream_coordinator.go`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/webchat/connection_pool.go`
- `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/persistence/chatstore/timeline_store.go`

## Step 2: Validate the ticket docs and upload the guide to reMarkable

Once the design doc existed, the next step was to make sure the ticket was structurally healthy and then deliver the guide to reMarkable. This step matters because the deliverable is supposed to be usable by another human, not just present in the filesystem. Validation ensured the ticket metadata and vocabulary were consistent; upload ensured the document was available outside the repo.

This step also surfaced the only actual issue encountered during the ticket work: two topic slugs used in the new ticket (`implementation` and `onboarding`) were not yet present in the local `docmgr` vocabulary. That was easy to fix, but it was worth recording because it is exactly the kind of small metadata issue that can make later ticket hygiene noisier than necessary.

### Prompt Context

**User prompt (verbatim):** (same as Step 1)

**Assistant interpretation:** Finish the ticket properly by validating it and delivering the resulting document to reMarkable.

**Inferred user intent:** The output should not only exist locally; it should be cleanly stored and already available on the user's reading device.

**Commit (code):** N/A — documentation-only work in the ticket workspace.

### What I did

- Ran ticket validation:

```bash
docmgr doctor --ticket EVT-STREAM-003 --stale-after 30
```

- Got a vocabulary warning for missing topics `implementation` and `onboarding`.
- Added both vocabulary entries:

```bash
docmgr vocab add --category topics --slug implementation --description "Implementation planning and execution work"
docmgr vocab add --category topics --slug onboarding --description "Onboarding-oriented documentation for new contributors"
```

- Re-ran validation successfully:

```bash
docmgr doctor --ticket EVT-STREAM-003 --stale-after 30
```

- Performed a dry-run reMarkable upload of the primary doc:

```bash
remarquee upload md --dry-run /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md --remote-dir /ai/2026/04/19/EVT-STREAM-003 --name "EVT-STREAM-003 - Implementation Plan and Intern Onboarding Guide"
```

- Uploaded the primary doc and verified the remote listing:

```bash
remarquee upload md /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/19/EVT-STREAM-003--event-streaming-llm-framework-implementation-plan-and-intern-onboarding-guide/design-doc/01-implementation-plan-and-intern-onboarding-guide.md --remote-dir /ai/2026/04/19/EVT-STREAM-003 --name "EVT-STREAM-003 - Implementation Plan and Intern Onboarding Guide"
remarquee cloud ls /ai/2026/04/19/EVT-STREAM-003 --long --non-interactive
```

### Why

- `docmgr doctor` is the quickest way to catch metadata and vocabulary mistakes before the ticket grows more documents.
- Dry-run upload protects against remote-path or Pandoc rendering mistakes.
- Upload verification confirms the output is actually accessible on the device/cloud side.

### What worked

- `docmgr doctor` passed cleanly after adding the missing vocabulary entries.
- `remarquee upload md --dry-run` produced the expected PDF/rendering plan.
- The actual upload succeeded.
- The remote listing showed:

```text
[f]	EVT-STREAM-003 - Implementation Plan and Intern Onboarding Guide
```

### What didn't work

- Initial `docmgr doctor` warning:

```text
unknown topics: [implementation onboarding]
```

This was fixed by adding those topic slugs to the vocabulary.

### What I learned

- The ticket workflow is smoother if new topic vocabulary is added as soon as a new class of documentation appears.
- For this kind of deliverable, uploading the primary guide alone is sufficient and lower-friction than trying to bundle a diary that may continue evolving after delivery.

### What was tricky to build

The main subtlety here was deciding what exactly to upload. A bundle would have been possible, but the diary is a living record and is more likely to change after delivery than the primary design doc. Uploading the primary guide alone kept the deliverable stable while still satisfying the user's request to store the work in the ticket and deliver it to reMarkable.

### What warrants a second pair of eyes

- Whether future tickets of this kind should prefer single-doc upload or bundled upload by default.
- Whether the topic vocabulary should be expanded proactively for implementation/onboarding work elsewhere in the repo.

### What should be done in the future

- If the implementation ticket starts accumulating additional stable reference docs, consider a future bundled upload.
- If more EVT-STREAM tickets use `implementation` and `onboarding`, keep those vocabulary slugs standardized.

### Code review instructions

- Review `index.md`, `tasks.md`, `changelog.md`, and the primary design doc for ticket completeness.
- Re-run:

```bash
docmgr doctor --ticket EVT-STREAM-003 --stale-after 30
```

- Verify remote delivery with:

```bash
remarquee cloud ls /ai/2026/04/19/EVT-STREAM-003 --long --non-interactive
```

### Technical details

Validation result after vocabulary update:

```text
## Doctor Report (1 findings)

### EVT-STREAM-003

- ✅ All checks passed
```

Upload result:

```text
OK: uploaded EVT-STREAM-003 - Implementation Plan and Intern Onboarding Guide.pdf -> /ai/2026/04/19/EVT-STREAM-003
[f]	EVT-STREAM-003 - Implementation Plan and Intern Onboarding Guide
```

## Related

- Primary guide: `../design-doc/01-implementation-plan-and-intern-onboarding-guide.md`
- Ticket index: `../index.md`
- Ticket changelog: `../changelog.md`
