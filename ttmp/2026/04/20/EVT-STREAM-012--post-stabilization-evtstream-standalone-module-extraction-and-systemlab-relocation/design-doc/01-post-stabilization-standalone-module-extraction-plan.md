---
Title: Post-stabilization standalone module extraction plan
Ticket: EVT-STREAM-012
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - backend
    - implementation
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pinocchio/pkg/evtstream/doc.go
      Note: Current package root and statement of intent for the substrate.
    - Path: pinocchio/pkg/evtstream/hub.go
      Note: Central orchestration surface that defines much of the public substrate API.
    - Path: pinocchio/pkg/evtstream/transport/ws/server.go
      Note: Extractable websocket transport that should move with the standalone module.
    - Path: pinocchio/pkg/evtstream/hydration/sqlite/store.go
      Note: Optional persistence subpackage that should move with the standalone module.
    - Path: pinocchio/pkg/evtstream/examples/chat/chat.go
      Note: Example application layer that may move with the module or remain an optional later follow-up.
    - Path: pinocchio/cmd/evtstream-systemlab/README.md
      Note: Current Systemlab boundary contract and rationale for separate-app ownership.
    - Path: pinocchio/go.mod
      Note: Current module boundary that still bundles evtstream with the rest of pinocchio.
ExternalSources: []
Summary: "Detailed later-phase plan for extracting evtstream from pinocchio into its own standalone Go module/package, moving Systemlab alongside it as a separate app, and keeping the extraction deferred until the substrate and its first real consumers are polished and tested."
LastUpdated: 2026-04-20T11:12:00-04:00
WhatFor: "Capture the intended standalone-module split so the current implementation can stay extraction-friendly and the future move can be executed deliberately later."
WhenToUse: "When reviewing whether evtstream is sufficiently decoupled for extraction, or when eventually executing the move into its own module/repo."
---

# Post-stabilization standalone module extraction plan

## Executive Summary

This ticket proposes a **later** architectural split: once `evtstream`, Systemlab, and the first real consuming application path are polished and tested, move `evtstream` out of `pinocchio` and into its own standalone Go module/package.

The extraction should move not only the substrate packages themselves, but also the surrounding `evtstream`-native support code that belongs with the substrate ecosystem:

- core `evtstream` packages,
- hydration subpackages,
- transport subpackages,
- likely example packages,
- and `cmd/evtstream-systemlab` as a separate consumer app.

The move should **not** happen immediately. Right now, co-location inside `pinocchio` is still useful while the architecture is being proven. But the long-term architecture should not stop at "clean code inside one repo." It should end with a real module seam that enforces the distinction between:

- substrate,
- teaching/demo/validation app,
- and consuming applications such as `cmd/web-chat`.

The target outcome is:

```text
standalone evtstream module/repo
  -> substrate core
  -> optional stores/transports/examples
  -> Systemlab as separate app

pinocchio
  -> consumes evtstream
  -> owns cmd/web-chat and other applications
  -> no longer owns evtstream internals
```

## Problem Statement

`pkg/evtstream` was intentionally designed as a reusable substrate rather than a renamed `pkg/webchat`. Architecturally, that has gone well: the package already appears substantially less entangled with the rest of `pinocchio` than `pkg/webchat`.

But module co-location still weakens the boundary in several ways.

### Problem 1: the boundary is currently enforced mostly by discipline

Today, `evtstream` lives under the `pinocchio` module path, so application and substrate code still inhabit one broader import namespace. That makes it too easy for future changes to blur the line between:

- generic substrate APIs,
- `evtstream`-specific examples and tools,
- and `pinocchio` application internals.

### Problem 2: Systemlab conceptually belongs to evtstream more than to pinocchio

Systemlab is not a `pinocchio` product surface. It is an `evtstream` teaching, validation, and inspection app. Keeping it under `pinocchio/cmd/...` was pragmatic while the framework was young, but conceptually it belongs beside the extracted module rather than inside a consuming application repo.

### Problem 3: future publishing and versioning are awkward if the module never splits

If `evtstream` remains embedded inside `pinocchio`, then:

- its public API is harder to reason about,
- versioning is coupled to unrelated repository concerns,
- downstream reuse is less natural,
- and architectural drift can creep in gradually.

### Problem 4: extraction gets harder the longer it is deferred without a plan

Deferring the actual move is reasonable. Deferring the plan entirely is not. Without a written extraction plan, the code can slowly accumulate assumptions that make later separation much more painful.

## Current Assessment: How Decoupled evtstream Already Is

The current codebase suggests that `evtstream` is a strong extraction candidate.

### What is already going well

At the package level, `pkg/evtstream` is already shaped like a standalone library:

- core types are substrate-centric,
- session identity is explicit and generic,
- stores/transports are subpackages,
- projections and command handlers are generic seams,
- and example chat logic already lives outside the core package.

### What this means architecturally

That is exactly the structure we would want if we were designing a standalone module from the start. In other words, extraction is not a fantasy follow-up. It is a plausible next architecture step once the public surface is stable enough.

## Proposed Solution

After stabilization, extract `evtstream` into its own module/package and move Systemlab with it.

### 1. Create a standalone module boundary

The new home should be a dedicated Go module, with its own `go.mod`, versioning, CI, and public import path.

The precise hosting location can be decided later, but the architectural requirement is clear:

> `evtstream` should no longer live only as a subdirectory of the `pinocchio` module.

If the work is staged under `le-chat` first, it should still be done as a **dedicated Go module subtree**, not as an ad hoc package mixed into the ticket workspace root.

### 2. Move substrate-owned code with the module

These pieces should move with the extracted module:

```text
evtstream/
  go.mod
  core package files
  hydration/memory
  hydration/sqlite
  transport
  transport/ws
```

### 3. Move Systemlab with the module, but keep it separate from core

Systemlab should move into the standalone `evtstream` repo/module as a **consumer app**, not as part of the substrate core package.

Recommended shape:

```text
evtstream/
  cmd/
    evtstream-systemlab/
```

This keeps the ownership model clean:

- `evtstream` core stays reusable,
- Systemlab stays a separate app,
- but the app now lives in the same ecosystem as the module it teaches and validates.

### 4. Leave consuming applications in pinocchio

These should remain in `pinocchio`:

- `cmd/web-chat`
- any app-specific web-chat packages
- `pkg/webchat`
- other unrelated application/runtime code

That means `pinocchio` becomes a downstream consumer of the extracted module, which is exactly the kind of real-world boundary we want.

## Proposed Target Layout

One strong target shape is:

```text
evtstream/
  go.mod
  doc.go
  types.go
  hub.go
  schema.go
  handler.go
  projection.go
  hydration.go
  bus.go
  consumer.go
  ordinals.go
  fanout.go
  command_registry.go
  session_registry.go
  noop_store.go

  hydration/
    memory/
    sqlite/

  transport/
    transport.go
    ws/

  examples/
    chat/

  cmd/
    evtstream-systemlab/
```

And then:

```text
pinocchio/
  cmd/web-chat/
  pkg/webchat/
  ...
```

## Design Decisions

### Decision 1: do not extract until the public surface is stable enough

This ticket is explicitly about a **later** move.

We should not split the module while:

- public APIs are still changing rapidly,
- Systemlab is still proving core behavior,
- or the first real app consumer has not yet converged.

The right sequence is:

1. prove the design,
2. polish the design,
3. lock down the boundary,
4. then extract.

### Decision 2: Systemlab should move with evtstream

Systemlab conceptually belongs with `evtstream` because it:

- teaches the substrate,
- validates the substrate,
- and exercises only the substrate’s public seams.

That makes it a better fit in the `evtstream` ecosystem than in `pinocchio`.

### Decision 3: Systemlab should remain a separate app, not a core package

Systemlab includes:

- HTTP serving,
- embedded assets,
- lab-specific traces,
- chapters,
- and browser-side teaching UI.

Those are all appropriate for a companion app, but not for the substrate core package.

### Decision 4: examples may move with the module, but can be staged

`examples/chat` is probably a good fit for the standalone module, because it demonstrates how to consume the substrate without contaminating the core. But if timing or API stability makes that awkward, it can be moved in a second step after the core split.

## Preconditions Before Execution

The extraction should not begin until the following are true.

### API stability

- `evtstream` core public types are no longer changing every phase.
- Store and transport seams are stable enough for downstream consumption.
- The Phase 6 `cmd/web-chat` port has clarified what a real consumer needs.

### Test and validation stability

- `evtstream` package tests pass reliably.
- Systemlab remains a trustworthy boundary exerciser.
- At least one real consuming app path is working against the public substrate seams.

### Documentation stability

- The package purpose and boundaries are documented clearly.
- The split between substrate, examples, Systemlab, and consuming apps is explicit.
- The intended standalone public surface is named and reviewable.

## Implementation Plan

This section describes the later execution sequence.

### Step 1: boundary audit before the move

Confirm what currently imports `evtstream`, and verify that `evtstream` itself is still free of non-substrate `pinocchio` dependencies.

### Step 2: freeze the intended public API

Before moving files, explicitly decide:

- what packages are public,
- what types/functions define the supported surface,
- what is still internal or provisional.

### Step 3: create the standalone module skeleton

Create the new module with:

- `go.mod`
- package layout
- baseline README/docs
- test entrypoints
- CI/test commands

### Step 4: move substrate code

Move the core `evtstream` packages first, then stores/transports.

### Step 5: move Systemlab into the new module

Move `cmd/evtstream-systemlab` so it now imports the extracted module through its new public path, not via local `pinocchio/pkg/...` paths.

### Step 6: update pinocchio as downstream consumer

Change `pinocchio` to consume the standalone module in:

- `cmd/web-chat`
- any app-specific `evtstream` usage
- any future internal tooling

### Step 7: add stronger boundary enforcement

After extraction, add checks that make architectural backsliding harder, for example:

- the standalone module may not import `pinocchio` packages,
- Systemlab may only import public `evtstream` packages,
- `pinocchio` consumes only the published/exposed `evtstream` surface.

## Alternatives Considered

### Alternative 1: keep evtstream inside pinocchio permanently

This is the easiest short-term path, but it weakens long-term architectural clarity. The boundary stays mostly social rather than structural.

### Alternative 2: move only the core package, leave Systemlab behind

This is possible, but it misses a good opportunity. Systemlab is conceptually an `evtstream` companion app, and moving it later would just create a second extraction task.

### Alternative 3: move Systemlab into the core package itself

This should be rejected. Systemlab is not substrate core. It is a separate application that teaches and validates the substrate.

## Risks

### Risk 1: extracting too early

If we split before the public API is mature, we create churn twice: once during phase development, and again during module cleanup.

### Risk 2: moving examples and apps without a clean ownership model

If the extraction does not clearly separate core from examples and companion apps, the new repo can become a second mixed-ownership monolith.

### Risk 3: pinocchio accidentally keeps substrate-private assumptions

If `cmd/web-chat` comes to depend on implementation details rather than public `evtstream` seams, extraction will expose those assumptions painfully. That is useful, but it must be planned for.

## Open Questions

- What should the final module path be?
- Should `examples/chat` move in the first extraction or a follow-up slice?
- Should the standalone repo own its own chapter/docs publishing workflow for Systemlab?
- What exact readiness checklist will we require before starting the move?

## Final Recommendation

Treat this as a **planned follow-up architecture ticket**, not as immediate implementation work.

The present job is to keep `evtstream` extraction-friendly while Phase 6 and the first real app consumer settle. The later job, tracked by this ticket, is to finish the separation properly:

- extract `evtstream` into its own standalone module/package,
- move Systemlab with it as a separate command/app,
- leave `pinocchio` as a downstream consumer,
- and let the module boundary enforce the clean design we currently have to enforce mostly by convention.

## References

- `pinocchio/pkg/evtstream/doc.go`
- `pinocchio/pkg/evtstream/hub.go`
- `pinocchio/pkg/evtstream/transport/ws/server.go`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go`
- `pinocchio/pkg/evtstream/examples/chat/chat.go`
- `pinocchio/cmd/evtstream-systemlab/README.md`
- `le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/01-phase-6-implementation-plan.md`
