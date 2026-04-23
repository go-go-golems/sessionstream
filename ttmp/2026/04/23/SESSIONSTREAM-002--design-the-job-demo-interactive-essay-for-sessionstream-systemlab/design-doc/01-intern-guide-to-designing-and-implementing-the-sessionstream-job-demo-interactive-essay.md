---
Title: Intern guide to designing and implementing the sessionstream job demo interactive essay
Ticket: SESSIONSTREAM-002
Status: active
Topics:
    - architecture
    - backend
    - framework
    - event-streaming
    - onboarding
    - systemlab
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: Repository purpose and ownership boundaries for the extracted framework
    - Path: cmd/sessionstream-systemlab/README.md
      Note: Current teaching-app contract
    - Path: cmd/sessionstream-systemlab/lab_environment.go
      Note: Current lab orchestration pattern and substrate-consumer wiring
    - Path: cmd/sessionstream-systemlab/server.go
      Note: Current route/page registration model that the new essay page must extend
    - Path: examples/chatdemo/chat.go
      Note: Existing example-package structure used as the comparison point for a future examples/jobdemo package
    - Path: pkg/sessionstream/hub.go
      Note: Core Hub API and projection/publisher seams referenced throughout the design
    - Path: pkg/sessionstream/hydration.go
      Note: HydrationStore and Snapshot seams needed for the reconnect/hydration chapter
    - Path: pkg/sessionstream/projection.go
      Note: UIProjection and TimelineProjection contracts that shape the essay's visible architecture
    - Path: pkg/sessionstream/transport/ws/server.go
      Note: Existing websocket transport behavior used for the reconnect and live-event teaching slice
    - Path: pkg/sessionstream/types.go
      Note: Core SessionId
ExternalSources: []
Summary: Detailed intern-facing analysis and implementation guide for building the next sessionstream demo as a textbook-style interactive essay about long-running job sessions, using a reusable examples/jobdemo package and a dedicated sessionstream-systemlab page.
LastUpdated: 2026-04-23T10:55:00-04:00
WhatFor: Explain why the job demo should be an explorable essay instead of a small dashboard app, and give a new contributor enough architectural and implementation context to build it confidently.
WhenToUse: Use when designing, reviewing, or implementing the sessionstream job demo interactive essay and its supporting jobdemo example package.
---


# Intern guide to designing and implementing the sessionstream job demo interactive essay

## Executive Summary

This ticket proposes that the next `sessionstream` demo should not be another conventional application shell. Instead, it should be a **textbook-style interactive essay** that uses a long-running job as the example domain. The essay should teach the core ideas of the framework by letting the reader create a job session, issue commands, watch canonical backend events stream in, inspect projected state, disconnect and reconnect, and see why the same session can drive multiple views.

That recommendation matters because the framework is not trying to sell a single product workflow. It is trying to teach a reusable mental model: sessions are the routing identity, commands initiate work, events describe what happened, projections derive views, and hydration allows recovery. A polished dashboard can showcase outcomes, but it often hides the architecture that produced them. The interactive essay should do the opposite. It should make the architecture visible and pleasant to explore.

The implementation should be split in two layers:

- `examples/jobdemo` should own the reusable job-domain example package: schemas, commands, backend events, projections, and a small service/engine pair.
- `cmd/sessionstream-systemlab` should own the essay page: chapter prose, live controls, raw event/state inspectors, and websocket/hydration teaching surfaces.

This document is intentionally detailed for a new intern. It explains the current repository structure, the system boundaries, the recommended domain model, the proposed chapter structure, the backend and frontend seams, the file-by-file implementation plan, the risks, the alternatives, and the validation path.

---

## Problem Statement

The repository already has two useful assets:

1. a generic substrate under `pkg/sessionstream`, and
2. a framework-owned teaching app under `cmd/sessionstream-systemlab`.

It also has a small `examples/chatdemo` package. That chat example is helpful because it proves that the substrate can support a concrete application. But it is not the best flagship teaching surface for the extracted framework. Chat is too easy to mentally classify as “LLM infrastructure,” and a normal app demo is too easy to treat as “a nice UI that happens to use the framework somehow.”

What is missing is a demo shape that does three things at once:

- **teaches the architecture**, not just the outcome,
- **stays playful and explorable**, not just correct,
- and **uses a domain that is obviously not chat-specific**.

The user’s steering here is important. They did not merely ask for a “job demo.” They explicitly pushed toward a “better, more fun version of Systemlab” and then tightened the constraint further: the `jobdemo` itself should also take that interactive-essay form. In other words, the goal is not a tiny SaaS-style job runner. The goal is a piece of educational software that uses jobs as the example domain.

If we ignore that distinction and build a dashboard first, several problems appear quickly:

- the architecture becomes implicit rather than visible,
- readers focus on job-management surface details instead of session semantics,
- inspectors and explanatory prose look bolted on afterward,
- and the demo starts feeling like a product mockup rather than a framework explainer.

So the design problem is:

> How do we build a job-oriented demo that teaches session-based streaming through direct interaction, while remaining a clean consumer of public `sessionstream` APIs and a pleasant onboarding surface for future contributors?

---

## Current System Anatomy: What You Must Understand First

Before designing the new essay, a new intern needs a map of the system that already exists. The right design will feel obvious once the current ownership boundaries are clear.

### 1. The substrate layer lives under `pkg/sessionstream`

The core substrate is the reusable part of the repository. Read these files first:

- `sessionstream/pkg/sessionstream/types.go`
- `sessionstream/pkg/sessionstream/hub.go`
- `sessionstream/pkg/sessionstream/projection.go`
- `sessionstream/pkg/sessionstream/hydration.go`
- `sessionstream/pkg/sessionstream/transport/ws/server.go`

These files define the nouns and seams the framework cares about.

#### The most important substrate vocabulary

- `SessionId` — the universal routing key for commands, events, cursors, snapshots, and subscriptions.
- `Command` — the typed request entering the substrate.
- `Event` — the canonical backend event emitted by command handlers or engines.
- `UIProjection` — a projection that turns backend events into transient UI-facing events.
- `TimelineProjection` — a projection that turns backend events into persistent timeline entities.
- `HydrationStore` — the persistence seam that stores current timeline state and cursors.
- `Snapshot` — the recoverable reconnect payload returned by the store.

A useful mental diagram is:

```text
Client action
   -> Command
      -> CommandHandler / Engine
         -> canonical backend Events
            -> UIProjection           -> websocket/UI fanout
            -> TimelineProjection     -> hydration store
                                          -> Snapshot / reconnect
```

If you understand that diagram, you understand the framework well enough to design the demo.

### 2. The current teaching app lives under `cmd/sessionstream-systemlab`

The current framework teaching surface is a separate application. Read these files next:

- `sessionstream/cmd/sessionstream-systemlab/README.md`
- `sessionstream/cmd/sessionstream-systemlab/server.go`
- `sessionstream/cmd/sessionstream-systemlab/lab_environment.go`
- `sessionstream/cmd/sessionstream-systemlab/static/index.html`
- `sessionstream/cmd/sessionstream-systemlab/static/js/main.js`
- `sessionstream/cmd/sessionstream-systemlab/chapters/phase-0-foundations.md`

The key lesson is architectural, not stylistic:

- Systemlab is a **separate consumer app**.
- It is allowed to use only public `sessionstream` APIs.
- It owns its own routes, static shell, page modules, chapter markdown, and evidence rendering.

That means the new job essay should not be implemented by reaching around the public API boundary. If a page needs data or controls, those should come through systemlab-owned routes and public example-package seams.

### 3. The current example package is `examples/chatdemo`

Read:

- `sessionstream/examples/chatdemo/chat.go`

This file is useful for two reasons.

First, it shows the current pattern for a reusable example package:

- register schemas,
- install command handlers and projections into a `Hub`,
- expose a small service/engine layer,
- and keep the app domain out of the substrate itself.

Second, it shows exactly why the next flagship demo should not merely repeat the same shape with different nouns. `chatdemo` proves the framework can support a concrete app. It does not yet teach the architecture as directly as an interactive essay could.

---

## The Main Design Decision

The most important design decision in this ticket is simple to state and worth sitting with for a moment:

> The job demo should be an interactive essay first and a conventional app second.

That means the page should be organized around **conceptual revelations**, not around generic product screens.

### Not this

```text
Runs list
Run detail
Step detail
Settings
```

### But this

```text
1. A job begins with a command
2. Progress arrives as events
3. State is projected, not pushed directly
4. Failure becomes history
5. Retry preserves the story
6. Hydration survives disconnect
7. One stream can power many views
8. The API stays surprisingly small
```

This is not a cosmetic distinction. It changes the entire implementation.

- The UI must expose raw events and projected state as first-class citizens.
- The page must make it easy to intentionally cause failure, retry, cancellation, and reconnect.
- The prose must frame each interaction in causal terms: what just happened, why it happened, and what to notice.
- The backend domain model must be small and legible enough that the user can hold it in their head while interacting.

The right reference style here is not “small admin app.” It is “explorable explanation.” The textbook-writing guidance matters for that reason. Good sections begin by explaining why the design exists, then give the reader one thing to try, and then expose concrete evidence directly under the claim.

---

## Why Jobs Are the Right Example Domain

Jobs are a better demo domain than chat for this teaching goal because they make the generic nature of the framework obvious.

A job session gives us all the behaviors we want to teach:

- long-running work,
- visible progress,
- multiple steps,
- logs or status updates,
- failure,
- retry,
- cancellation,
- completion,
- and reconnect/hydration.

Those behaviors map beautifully to `sessionstream` concepts.

### The mapping

| Job concept | Sessionstream concept |
|---|---|
| one job run | one `SessionId` |
| start / retry / cancel actions | commands |
| step state changes, logs, progress, completion | canonical backend events |
| job summary card | projection |
| step list | projection |
| logs panel | projection or derived event view |
| reconnect after reload | hydration snapshot + websocket resume |

The job domain also gives us a pleasant narrative arc. A reader understands what it means for a job to begin, stream progress, fail, be retried, or finish. That makes the conceptual teaching work easier.

---

## Proposed Solution

### High-level solution

Build the feature in two layers:

```text
examples/jobdemo/
   -> reusable example domain package
   -> command/event schemas
   -> command handlers / engine
   -> UI + timeline projections
   -> tests

cmd/sessionstream-systemlab/
   -> interactive essay page and chapter markdown
   -> page-specific API routes
   -> session header, live controls, inspectors
   -> websocket/reconnect teaching surface
```

This split is the cleanest one because it keeps the framework example reusable and keeps the essay presentation app-owned.

### Why not put everything in systemlab?

Because then the example ceases to be a reusable example and becomes page-specific logic. We would lose the ability to point to a single small package and say:

> Here is the smallest useful job-domain app built on the public substrate.

### Why not put the essay concerns in `examples/jobdemo`?

Because prose chapters, lab-only controls, disconnect simulation widgets, inspector tabs, and section-by-section page flow are not the example domain. They are teaching-app concerns. They belong in systemlab.

---

## Recommended Package Boundaries

### `examples/jobdemo`

This package should own:

- domain command names,
- domain backend event names,
- UI-event and timeline-entity names,
- schema registration,
- command handler / engine behavior,
- projections,
- a small service API,
- and testable semantics like failure, retry, and cancellation.

A suggested file layout:

```text
sessionstream/examples/jobdemo/
  job.go                # constants, install/register entrypoints, service
  engine.go             # long-running job execution simulation
  projections.go        # UIProjection + TimelineProjection
  types.go              # small domain DTO helpers if needed
  hooks.go              # optional observation hooks for systemlab/tests
  job_test.go           # command + event flow tests
  projections_test.go   # state/projection tests
```

### `cmd/sessionstream-systemlab`

This app should own:

- the job essay page route and page partial,
- the chapter markdown,
- page-specific HTTP endpoints,
- inspector formatting,
- disconnect / reconnect simulation controls,
- and any teaching-only UI affordances.

A suggested file layout extension:

```text
sessionstream/cmd/sessionstream-systemlab/
  jobdemo_page.go                      # page-specific HTTP handlers
  jobdemo_state.go                     # page-local orchestration / view helpers
  chapters/jobdemo-the-life-of-a-job.md
  static/partials/jobdemo.html
  static/js/pages/jobdemo.js
```

You do not have to use these exact filenames, but the ownership model should look like this.

---

## The User Experience: What the Essay Should Feel Like

The page should feel like a long-form chapter with live evidence embedded directly under each important claim.

### The emotional goal

The reader should feel:

- “I understand what a session is now,”
- “I can see the event stream, not just the UI result,”
- “I understand why hydration matters,”
- and “the API is smaller and cleaner than I expected.”

### The page should not feel like

- a generic enterprise dashboard,
- a pile of tabs with unexplained JSON,
- or an internal test harness dressed up as documentation.

### The page should feel like

- a guided essay,
- with a sticky instrument panel,
- a few clear buttons,
- visible raw evidence,
- and a consistent “try this / notice this” rhythm.

---

## Proposed Essay Structure

This is the most important part of the page design. The chapter order should follow conceptual insight, not implementation chronology.

### Section 0 — Hero / setup: “A Job Is a Session”

Purpose:
- establish the framing,
- create a new job session,
- show the `SessionId`, connection state, and current summary strip.

Suggested interaction:
- `Create job session`
- optional template selector (`build-and-deploy`, `generate-report`, etc.)

What should be visible immediately:
- session id,
- job status,
- connection state,
- last ordinal,
- snapshot ordinal or version.

### Section 1 — “A Job Begins with a Command”

Purpose:
- show that the job does not start because the UI mutated,
- it starts because the user submitted a command.

Suggested interaction:
- `Start job`

Visible evidence:
- command inspector,
- raw events (`JobCreated`, `JobStarted`, `JobStepStarted`),
- projected state summary.

What the prose should emphasize:

- commands are requests,
- events are the canonical record of what happened,
- projections, not handlers, decide what the UI looks like.

### Section 2 — “Progress Arrives as Events”

Purpose:
- show that long-running work is incremental.

Suggested controls:
- `Emit next progress update`
- `Append log burst`
- `Auto-run`

Visible evidence:
- streaming event list,
- progress bar,
- step state list,
- log tail.

Core insight:

> Many events may collapse into one current state. The stream contains the story; the projection contains the current truth.

### Section 3 — “One Stream, Many Views”

Purpose:
- show that the same session can drive multiple useful surfaces.

Suggested views:
- summary view,
- step list,
- logs,
- raw events,
- projected JSON state.

Core insight:

> A reusable framework stabilizes the stream and lets consumers build different views from it.

### Section 4 — “Failure Becomes History”

Purpose:
- make failure teachable instead of exceptional.

Suggested controls:
- `Fail current step`
- optional `Inject warning`

Visible evidence:
- failed step card,
- failure events,
- summary state moving to failed.

Core insight:

> The session does not break. It records what happened and projects a new state.

### Section 5 — “Retry Preserves the Story”

Purpose:
- teach append-only history versus authoritative current state.

Suggested control:
- `Retry failed step`

Visible evidence:
- attempt history for the failed step,
- new success events,
- final summary state.

Core insight:

> Retry should not erase prior attempts. It should outgrow them.

### Section 6 — “Cancellation Is a Command Too”

Purpose:
- show that stopping work belongs inside the same session model.

Suggested control:
- `Cancel job`

Visible evidence:
- `JobCancelRequested`, `JobCancelled` events,
- status flip,
- halted progress.

### Section 7 — “Hydration Survives Your Tab”

Purpose:
- teach why sessions are bigger than live websocket connections.

Suggested controls:
- `Disconnect`
- `Reconnect`
- `Hydrate snapshot`
- optional `Reload view`

Visible evidence:
- connection-state indicator,
- snapshot viewer,
- resumed event flow.

Core insight:

> The browser did not own the job. It rejoined the job.

### Section 8 — “The Smallest Useful API”

Purpose:
- reassure the reader that the implementation surface is not enormous.

Suggested contents:
- one small code block for command submission,
- one for event publication,
- one for projections,
- one for websocket/snapshot integration.

This section should connect the visible system back to the concrete API surface.

---

## Persistent Page Elements

The page should have a few persistent elements that remain visible while the reader scrolls.

### 1. Sticky session header

Always show:

- session id,
- job status,
- connection state,
- last ordinal,
- snapshot ordinal,
- active step.

This is the reader’s stable anchor.

### 2. Raw event inspector

The event inspector should never be hard to reach. It can be sticky, collapsible, or docked, but it should be treated as first-class evidence.

Columns to show:

- ordinal,
- event type,
- timestamp,
- payload preview,
- expandable JSON.

### 3. Projected state inspector

This should show the current derived state in a compact and legible form. It is especially useful when paired with the raw event stream.

### 4. “Try this / Notice this” callouts

Each major section should include one or two short prompts:

- “Start the job and watch the first three events.”
- “Fail the current step. What changed in the event stream? What changed in the projected state?”
- “Disconnect, then hydrate. What survived?”

These callouts are not decorative. They are the bridge between prose and action.

---

## Backend Domain Model Recommendation

Keep the domain model small. A teaching demo becomes harder to follow when the domain vocabulary sprawls.

### Recommended commands

```text
StartJob
AdvanceJob
AppendLogBurst
FailCurrentStep
RetryFailedStep
CancelJob
ReconnectJobView      # optional teaching-only control, maybe app-side not domain-side
```

Not all of these must ship as domain commands. Some can remain page-local control helpers that translate into one or more domain commands. But `StartJob`, `RetryFailedStep`, and `CancelJob` should definitely exist as real commands.

### Recommended backend events

```text
JobCreated
JobStarted
JobProgressUpdated
JobStepStarted
JobStepProgressUpdated
JobLogAppended
JobStepFailed
JobRetryRequested
JobStepRetried
JobStepSucceeded
JobCancelled
JobFinished
ArtifactProduced      # optional for one-stream-many-views section
```

### Recommended UI events

These should be derived from backend events rather than invented ad hoc. Keep them close to the backend semantics so the essay can explain the mapping honestly.

Example families:

```text
JobSummaryUpdated
JobStepViewUpdated
JobLogViewAppended
JobStatusChanged
```

### Recommended timeline entities

Keep the persistent entities focused and easy to inspect:

- `JobSummary`
- `JobStep`
- `JobLogChunk` or aggregated log state
- `JobArtifact` (optional)

The summary projection should answer “what is true now?” The event inspector should answer “what happened?”

---

## Backend Architecture Recommendation

### Core recommendation

Model the job execution engine as a small stateful simulator that publishes canonical events over time. It should support:

- normal forward progress,
- injected failure,
- retry,
- cancellation,
- and optional auto-run timing.

### Suggested engine responsibilities

- track active runs by `SessionId`,
- assign stable step ids / attempt ids,
- emit progress and log events in sequence,
- cancel cleanly,
- retry failed steps without deleting prior attempts,
- expose hooks for systemlab observation if needed.

### Pseudocode sketch

```go
func (e *Engine) handleStartJob(ctx, cmd, session, pub) error {
    state := e.ensureRun(cmd.SessionId)
    if state.alreadyStarted() {
        return nil
    }

    publish(JobCreated)
    publish(JobStarted)
    publish(JobStepStarted("prepare", attempt=1))
    return nil
}

func (e *Engine) advance(ctx, sid, pub) error {
    run := e.currentRun(sid)
    switch run.activeStep {
    case "prepare":
        publish(JobStepSucceeded("prepare", attempt=1))
        publish(JobStepStarted("validate", attempt=1))
    case "validate":
        publish(JobProgressUpdated(...))
        publish(JobLogAppended(...))
    }
}

func (e *Engine) failCurrentStep(ctx, sid, pub) error {
    run := e.currentRun(sid)
    publish(JobStepFailed(step=run.activeStep, attempt=run.attempt))
    return nil
}

func (e *Engine) retryFailedStep(ctx, sid, pub) error {
    run := e.currentRun(sid)
    run.attempt++
    publish(JobRetryRequested(...))
    publish(JobStepStarted(step=run.failedStep, attempt=run.attempt))
    return nil
}
```

The exact implementation can differ, but the visible semantics should remain that clear.

---

## Projection Design Recommendation

This is where the framework teaching value really emerges.

### Summary projection

The summary projection should derive:

- overall job status,
- active step,
- overall progress,
- started/ended timestamps,
- retry count,
- cancellation status.

### Step projection

The step projection should derive:

- ordered steps,
- current status per step,
- attempt history,
- timestamps and durations,
- failure detail if present.

### Log projection

The log view can be designed in one of two ways.

#### Option A: persistent log chunks
Persist log chunks as timeline entities.

Pros:
- easiest to hydrate honestly,
- easiest to inspect in snapshots.

Cons:
- snapshot can grow noisier.

#### Option B: aggregate logs into job summary state
Store only aggregated log state.

Pros:
- simpler snapshot shape.

Cons:
- less explicit event-to-entity relationship.

For this teaching surface, I would bias toward **Option A** unless performance gets silly. It is more legible for an explorable essay.

### UI projection

The UI projection should preserve a live-feeling stream, but the essay must make clear that the live stream is not the only truth. The durable timeline entities are what power hydration.

---

## Websocket and Hydration Teaching Slice

The reconnect/hydration section should make visible use of the existing websocket transport rather than simulating recovery entirely in page-local code.

Relevant file to read carefully:

- `sessionstream/pkg/sessionstream/transport/ws/server.go`

Important transport ideas already available there:

- connection ids,
- subscribe flows,
- snapshot send on subscribe,
- UI-event fanout,
- hooks for lab/debug observation.

### Teaching recommendation

Show three things at once during the reconnect section:

1. **connection state** — connected vs disconnected,
2. **snapshot state** — what the hydration store knows,
3. **live event continuation** — what arrives after reconnect.

That lets the reader distinguish:

- durable projected state,
- transport subscription state,
- and live post-hydration updates.

---

## Frontend / Page Layout Recommendation

I recommend a hybrid page layout:

```text
Left column: essay prose + inline controls
Right rail: sticky session header + event inspector + state inspector
Main embedded sections: focused live widgets for each chapter
```

### Why this layout works

A fully dashboard-like page loses the narrative. A fully linear page hides too much context between sections. The hybrid gives you both:

- prose and local interaction where the reader is thinking,
- persistent truth panels where the reader can keep an eye on the system.

### Stable UI regions

- **Header strip** — session identity and status
- **Control strip** — current section’s main actions
- **Raw events panel** — append-only evidence
- **Projected state panel** — current truth
- **Rendered human view** — the summary / steps / logs that feel like an app

---

## File-by-File Implementation Plan

This is the section a new intern should use when transitioning from design reading to implementation.

### Phase A — create `examples/jobdemo`

Create a new example package modeled loosely after `examples/chatdemo`, but simpler and more obviously generic.

Start with:

- `sessionstream/examples/jobdemo/job.go`
- `sessionstream/examples/jobdemo/engine.go`
- `sessionstream/examples/jobdemo/projections.go`
- `sessionstream/examples/jobdemo/job_test.go`

What to implement first:

1. schema registration,
2. `Install(...)` equivalent to register commands and projections,
3. minimal service layer for start/cancel/retry,
4. deterministic tests for event and snapshot semantics.

### Phase B — add a new essay page in systemlab

Touch:

- `sessionstream/cmd/sessionstream-systemlab/server.go`
- `sessionstream/cmd/sessionstream-systemlab/static/index.html`
- `sessionstream/cmd/sessionstream-systemlab/static/js/main.js`
- `sessionstream/cmd/sessionstream-systemlab/static/partials/jobdemo.html`
- `sessionstream/cmd/sessionstream-systemlab/static/js/pages/jobdemo.js`
- `sessionstream/cmd/sessionstream-systemlab/chapters/jobdemo-the-life-of-a-job.md`

You will probably also need a small page-local orchestration file in Go to expose routes that drive the live demo.

### Phase C — connect example package to page orchestration

This is where systemlab becomes a real consumer.

Possible new file:

- `sessionstream/cmd/sessionstream-systemlab/jobdemo_page.go`

Responsibilities:

- create or reset job sessions,
- expose endpoints for start / advance / fail / retry / cancel,
- expose current snapshot and any teaching-only metadata,
- wire websocket transport if the page needs a live subscription surface separate from the current phase labs.

### Phase D — implement reconnect/hydration section

Touch:

- websocket transport setup in the page orchestration,
- snapshot fetch route,
- frontend page module handling disconnect / reconnect / hydrate flows.

Do not fake this entirely in the browser. The entire point is to show the real framework seams.

### Phase E — polish prose and inspectors

Touch:

- chapter markdown,
- rendered evidence widgets,
- state/event JSON viewers,
- any sticky status UI.

Review this phase against the textbook-writing guidance:

- foundations before mechanics,
- prose paragraphs that develop one idea at a time,
- bullet lists used for emphasis rather than filler,
- concrete examples and traces near every important claim.

---

## API References You Should Keep Open While Building

### Substrate APIs

- `sessionstream/pkg/sessionstream/hub.go`
  - `NewHub(...)`
  - `RegisterCommand(...)`
  - `RegisterUIProjection(...)`
  - `RegisterTimelineProjection(...)`
  - `Submit(...)`
  - `Snapshot(...)`
  - `Session(...)`
  - `Cursor(...)`

- `sessionstream/pkg/sessionstream/projection.go`
  - `UIProjection`
  - `UIProjectionFunc`
  - `TimelineProjection`
  - `TimelineProjectionFunc`

- `sessionstream/pkg/sessionstream/hydration.go`
  - `HydrationStore`
  - `Snapshot`

- `sessionstream/pkg/sessionstream/types.go`
  - `SessionId`
  - `Command`
  - `Event`
  - `Session`

### Transport APIs

- `sessionstream/pkg/sessionstream/transport/ws/server.go`
  - `NewServer(...)`
  - `ServeHTTP(...)`
  - `PublishUI(...)`
  - transport hooks and frame behavior

### Current teaching app references

- `sessionstream/cmd/sessionstream-systemlab/server.go`
- `sessionstream/cmd/sessionstream-systemlab/lab_environment.go`
- `sessionstream/cmd/sessionstream-systemlab/chapters/phase-0-foundations.md`

### Example-package reference

- `sessionstream/examples/chatdemo/chat.go`

Read `chatdemo` for structure, but do not let it dictate the page design. The job essay must teach more explicitly than the chat example does.

---

## Design Decisions and Rationale

### Decision 1 — Build the demo as an interactive essay rather than a dashboard

**Rationale:** The framework needs a teaching surface more than it needs another normal-looking app. The essay format keeps the architecture visible.

### Decision 2 — Use jobs as the example domain

**Rationale:** Jobs naturally demonstrate streaming progress, failure, retry, cancel, and reconnect without feeling chat-specific.

### Decision 3 — Split ownership between `examples/jobdemo` and `cmd/sessionstream-systemlab`

**Rationale:** The example logic should stay reusable and testable. The essay presentation should stay app-owned.

### Decision 4 — Keep raw events and projected state visible at all times

**Rationale:** Without visible evidence, the reader sees only outcomes and misses the architecture.

### Decision 5 — Use the existing websocket/hydration seams rather than a browser-only simulation

**Rationale:** The page should exercise the real framework seams honestly, the same way systemlab was intended to.

---

## Alternatives Considered

### Alternative A — Build a normal run-list + run-detail jobs app

This would be easier to explain to someone who only wants a product demo, but it weakens the framework-teaching goal. The architecture would retreat behind polished UI, and the raw stream/state relationships would likely become hidden or secondary.

### Alternative B — Keep everything in a single systemlab page-local implementation with no reusable `jobdemo` package

This would be faster initially, but it would weaken the repository’s example story. Future readers would have a teaching page but no small reusable example package to study in isolation.

### Alternative C — Expand `chatdemo` instead of adding a new domain

This was rejected because it continues to tie the framework’s public image to chat. The extracted repo needs at least one clearly non-chat example domain.

### Alternative D — Build multiple mini-demos at once

This was rejected for sequencing reasons. One excellent flagship explorable essay is more valuable than three shallow demos that each split attention and implementation energy.

---

## Risks and How to Think About Them

### Risk 1 — The page becomes a dashboard with prose glued on afterward

This is the most likely failure mode.

Mitigation:

- freeze the section structure before implementation,
- keep every section organized around one conceptual claim,
- require a live artifact directly under each important claim.

### Risk 2 — The example domain gets too rich

If the job model grows too many step types, settings, artifacts, and branching rules, the teaching value drops.

Mitigation:

- keep the job domain tiny,
- prefer four simple steps over a realistic workflow matrix,
- optimize for clarity rather than realism.

### Risk 3 — The essay cheats by bypassing public seams

This would damage the whole point of systemlab.

Mitigation:

- keep orchestration in systemlab,
- keep job semantics in `examples/jobdemo`,
- use public hub, store, transport, and snapshot APIs end to end.

### Risk 4 — Inspectors become noisy and unreadable

Raw JSON alone is not pedagogy.

Mitigation:

- provide rendered views next to raw inspectors,
- emphasize important fields visually,
- use collapsible JSON rather than only raw dumps.

### Risk 5 — Reconnect/hydration remains conceptually muddy

This is an area many new contributors misunderstand.

Mitigation:

- explicitly distinguish transport connection, snapshot state, and live post-hydration events,
- show them side by side on the page.

---

## Validation Strategy

The implementation should be validated at three layers.

### 1. Example package tests

Run focused tests on `examples/jobdemo` covering:

- command handling,
- event emission,
- retry history,
- cancellation,
- summary and step projections.

### 2. Systemlab page tests or smoke checks

Validate:

- page loads,
- chapter markdown renders,
- controls hit the correct routes,
- websocket path connects,
- snapshot route hydrates current state.

### 3. Human walkthrough validation

A reviewer should be able to perform this path:

1. create session,
2. start job,
3. watch progress,
4. fail current step,
5. retry,
6. disconnect,
7. reconnect/hydrate,
8. inspect raw events and current state,
9. finish job.

If the page teaches those moments clearly, the design is working.

---

## Recommended Reading Order for a New Intern

Do not start coding from the prose chapter alone. Read in this order:

1. `sessionstream/README.md`
2. `sessionstream/pkg/sessionstream/types.go`
3. `sessionstream/pkg/sessionstream/hub.go`
4. `sessionstream/pkg/sessionstream/projection.go`
5. `sessionstream/pkg/sessionstream/hydration.go`
6. `sessionstream/pkg/sessionstream/transport/ws/server.go`
7. `sessionstream/examples/chatdemo/chat.go`
8. `sessionstream/cmd/sessionstream-systemlab/README.md`
9. `sessionstream/cmd/sessionstream-systemlab/server.go`
10. `sessionstream/cmd/sessionstream-systemlab/lab_environment.go`
11. `sessionstream/cmd/sessionstream-systemlab/chapters/phase-0-foundations.md`
12. this ticket’s design doc again, now with the codebase in your head

That second reading is important. The document is much easier to implement once the existing seams feel familiar.

---

## Open Questions

These are worth keeping visible until implementation starts.

1. Should the essay be a new first-class top-level page, or should it replace one of the current phase-oriented pages over time?
2. Should `examples/jobdemo` support both step-by-step manual advancement and timed auto-run from day one, or should auto-run be a later polish slice?
3. Should log data be persisted as first-class timeline entities or aggregated into a summary payload?
4. Should the reconnect section use one dedicated websocket per page or reuse the same transport pattern already used in Phase 3/4 labs?
5. Should artifacts be part of MVP, or should the first version stop at summary + steps + logs + reconnect?

None of these block the central direction. They are sequencing questions, not architectural doubts.

---

## Final Recommendation

Build the next `sessionstream` demo as **The Life of a Job**: a textbook-style interactive essay implemented on top of a small reusable `examples/jobdemo` package and presented through a dedicated `sessionstream-systemlab` page.

The page should teach by making the architecture visible:

- commands,
- canonical events,
- projections,
- snapshots,
- reconnect,
- and multiple views over one stream.

That is the demo shape most likely to make `sessionstream` feel like a framework with a coherent model rather than just a renamed chat substrate.
