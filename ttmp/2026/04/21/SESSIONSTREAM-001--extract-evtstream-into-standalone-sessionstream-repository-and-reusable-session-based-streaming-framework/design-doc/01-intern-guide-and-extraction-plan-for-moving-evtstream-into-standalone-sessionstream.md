---
Title: Intern guide and extraction plan for moving evtstream into standalone sessionstream
Ticket: SESSIONSTREAM-001
Status: active
Topics:
    - architecture
    - backend
    - framework
    - event-streaming
    - migration
    - extraction
    - onboarding
    - systemlab
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/evtstream-systemlab/README.md
      Note: Systemlab boundary contract and rationale for moving the app with the framework
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/server.go
      Note: Current canonical downstream consumer assembly of the substrate
    - Path: ../../../../../../../pinocchio/pkg/evtstream/apps/chat/chat.go
      Note: Main app-grade chat package and the key pinocchio coupling seam that must be refactored
    - Path: ../../../../../../../pinocchio/pkg/evtstream/doc.go
      Note: Core statement of intent for the generic substrate
    - Path: ../../../../../../../pinocchio/pkg/evtstream/hub.go
      Note: Main orchestration API and public boundary surface
    - Path: .ttmp.yaml
      Note: Repo-local docmgr root was corrected to sessionstream/ttmp before ticket creation
    - Path: go.mod
      Note: Destination module skeleton currently still has the template module path and must be cleaned during extraction bootstrap
ExternalSources: []
Summary: Detailed analysis and implementation guide for extracting pinocchio/pkg/evtstream and cmd/evtstream-systemlab into the standalone sessionstream repository while preserving a clean generic substrate and leaving pinocchio-specific adapters behind.
LastUpdated: 2026-04-21T15:40:00-04:00
WhatFor: Give a new intern enough architectural context and a concrete migration plan to move evtstream into sessionstream without dragging pinocchio-specific runtime concerns into the new module.
WhenToUse: Use when planning, reviewing, or executing the standalone sessionstream extraction and when onboarding new contributors to the session-based streaming substrate.
---


# Intern guide and extraction plan for moving evtstream into standalone sessionstream

## Executive summary

`evtstream` is already very close to being a standalone library. The package root explicitly describes itself as a reusable substrate for event-streaming LLM and agent applications, centered on a single canonical routing key (`SessionId`), typed commands and backend events, sibling UI and timeline projections, and small storage/transport interfaces (`pinocchio/pkg/evtstream/doc.go:1-10`). The core `Hub` type is already the right kind of boundary object for an extracted module: it owns schema registration, hydration, sessions, command dispatch, projections, fanout, and optional bus consumption without knowing anything about webchat routes or browser-specific compatibility concerns (`pinocchio/pkg/evtstream/hub.go:21-41`, `pinocchio/pkg/evtstream/hub.go:90-107`, `pinocchio/pkg/evtstream/hub.go:145-167`).

The extraction is therefore **not** a rewrite project. It is primarily a boundary-hardening and packaging project. Most of the substrate can move into `sessionstream` with light import-path rewrites. The main work is isolating the places where `evtstream` still knows about `pinocchio`: specifically the app-grade chat package (`pinocchio/pkg/evtstream/apps/chat`) and the current canonical runtime integration used by `cmd/web-chat`. Systemlab should move with the substrate as a separate app because it was explicitly designed to exercise only the public seams of `evtstream` and not legacy `pkg/webchat` internals (`pinocchio/cmd/evtstream-systemlab/README.md:3-24`).

The recommended outcome is:

```text
sessionstream/
  go.mod                     module github.com/go-go-golems/sessionstream
  <root package files>       generic substrate
  hydration/...              optional stores
  transport/...              optional transports
  apps/chat                  generic chat app package (after de-pinocchio refactor)
  examples/...               examples that teach consumption patterns
  cmd/evtstream-systemlab    boundary exerciser and textbook app

pinocchio/
  cmd/web-chat               product app consuming sessionstream
  pkg/inference/runtime      pinocchio-specific runtime composition
  pkg/middlewares/agentmode  pinocchio-specific middleware and structured parsing
  package(s) that adapt pinocchio runtime events into sessionstream chat contracts
```

The most important design rule is simple:

> `sessionstream` must own the generic session-streaming substrate; `pinocchio` must own the pinocchio-specific runtime, middleware, and application edge behavior.

## Problem statement

The current architecture proved that `evtstream` should exist as a reusable substrate, but it still lives under the `pinocchio` module path. That means the code is architecturally cleaner than old `pkg/webchat`, yet the module boundary is still soft. The extraction needs to solve four problems.

### Problem 1: the current module path still hides the true architectural seam

Today the package imports look like this:

- `github.com/go-go-golems/pinocchio/pkg/evtstream`
- `github.com/go-go-golems/pinocchio/pkg/evtstream/hydration/sqlite`
- `github.com/go-go-golems/pinocchio/pkg/evtstream/transport/ws`

That import path says “this belongs to pinocchio,” even though the package documentation says the opposite (`pinocchio/pkg/evtstream/doc.go:1-10`). A dedicated repository and module will make the public boundary real instead of merely aspirational.

### Problem 2: the substrate is generic, but one important app package is not yet generic enough

The extraction candidate is not just the root package. It also includes app-grade support code. The difficulty is that the current chat app package reaches into pinocchio runtime and middleware packages:

- `pinocchio/pkg/evtstream/apps/chat/service.go:8-18` imports `pkg/inference/runtime`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:14-17` imports both `pkg/inference/runtime` and `pkg/middlewares/agentmode`

That means `apps/chat` is only *partially* standalone today. The extraction plan must either:

1. make `apps/chat` generic before or during the move, or
2. leave the pinocchio-specific runtime integration in `pinocchio` and keep only the generic chat domain logic in `sessionstream`.

### Problem 3: Systemlab conceptually belongs with the substrate, not with pinocchio

The README for `cmd/evtstream-systemlab` is explicit: Systemlab exists to explain, exercise, and validate the public API boundaries of `pkg/evtstream` (`pinocchio/cmd/evtstream-systemlab/README.md:3-10`). It may import public `evtstream` packages and may not reach into `pkg/webchat` or inject SEM-specific substrate concepts (`pinocchio/cmd/evtstream-systemlab/README.md:14-24`).

That is exactly the behavior of a companion app for a framework repository. It should move with the framework.

### Problem 4: pinocchio still needs product-specific adapters even after extraction

`cmd/web-chat` currently constructs a canonical app server by composing:

- a schema registry,
- a hydration store,
- the websocket transport,
- the chat engine,
- and the chat service (`pinocchio/cmd/web-chat/app/server.go:88-121`).

That is good news because it shows `cmd/web-chat` is already a consumer. But it also still injects pinocchio-specific runtime resolution via `RuntimeResolver` and `*infruntime.ComposedRuntime` (`pinocchio/cmd/web-chat/app/server.go:20`, `pinocchio/cmd/web-chat/app/server.go:27-35`, `pinocchio/cmd/web-chat/app/runtime.go:8-12`). The extraction must preserve this arrangement rather than dragging runtime policy into the new module.

## Scope and non-goals

### In scope

This ticket is about planning and documenting how to:

1. move the generic `evtstream` substrate into the standalone `sessionstream` repository,
2. move Systemlab with it as a separate app,
3. refactor the chat app package so the extracted repository does not depend on `pinocchio/pkg/inference/runtime` or `pinocchio/pkg/middlewares/agentmode`,
4. switch `pinocchio` to consume `sessionstream` as a downstream dependency,
5. provide onboarding-quality documentation for a new engineer executing the move.

### Explicitly out of scope

This ticket does **not** require us to:

- rewrite the whole substrate from first principles,
- move `cmd/web-chat` into `sessionstream`,
- move `pkg/webchat` into `sessionstream`,
- preserve legacy `/chat` or `/api/timeline` compatibility routes,
- redesign the frontend UX,
- replace geppetto, Watermill, or protobuf.

Those would all be larger and less focused projects.

## Current-state system map

The best way to understand what should move is to break the current system into architectural layers.

### Layer 1: generic session-streaming substrate (`pkg/evtstream`)

This is the real core.

The package header already states the intended abstraction:

- one canonical routing key: `SessionId`
- typed commands in
- typed backend events out
- sibling UI and timeline projections
- storage and transport behind small public interfaces
- application-specific ideas belong in consumers (`pinocchio/pkg/evtstream/doc.go:3-10`)

The `Hub` type is the center of the runtime model:

```text
Hub
  -> SchemaRegistry
  -> HydrationStore
  -> Session registry
  -> Command registry
  -> UI projection
  -> Timeline projection
  -> UIFanout
  -> optional event bus consumer
```

Evidence:

- fields: `pinocchio/pkg/evtstream/hub.go:21-41`
- constructor defaults: `pinocchio/pkg/evtstream/hub.go:90-107`
- command submission + snapshot access: `pinocchio/pkg/evtstream/hub.go:145-167`

This layer is the strongest extraction candidate because it already has the right shape for a public library.

### Layer 2: optional persistence packages (`hydration/memory`, `hydration/sqlite`)

These are good fits for `sessionstream` because they implement a substrate interface rather than product logic.

For example, the SQLite hydration store only depends on:

- the generic `evtstream.SchemaRegistry`,
- the generic `evtstream.HydrationStore` contract,
- protobuf JSON encoding,
- SQLite itself (`pinocchio/pkg/evtstream/hydration/sqlite/store.go:10-18`).

It stores:

- session cursor in `evtstream_sessions`
- entity payloads in `evtstream_entities`
- generic `(session_id, kind, entity_id)` keys (`pinocchio/pkg/evtstream/hydration/sqlite/store.go:163-175`)

That is exactly the kind of persistence package a standalone module should own.

### Layer 3: optional transport packages (`transport/ws`)

The websocket transport is also a strong extraction candidate.

`transport/ws.Server` is both:

- an `http.Handler`
- an `evtstream.UIFanout`

It accepts a generic `SnapshotProvider`, manages connection subscriptions, emits snapshot and UI-event envelopes, and does not know about webchat-specific URLs or browser compatibility hacks (`pinocchio/pkg/evtstream/transport/ws/server.go:17-36`, `pinocchio/pkg/evtstream/transport/ws/server.go:57-98`, `pinocchio/pkg/evtstream/transport/ws/server.go:149-171`).

This package belongs in `sessionstream/transport/ws` with only an import-path rewrite.

### Layer 4: app-grade chat package (`pkg/evtstream/apps/chat`)

This package is the most interesting and the most delicate.

It does three distinct jobs today:

1. defines chat-level commands, backend events, UI events, and timeline entities (`pinocchio/pkg/evtstream/apps/chat/chat.go:20-43`),
2. installs chat-specific command handlers and projections into a generic hub (`pinocchio/pkg/evtstream/apps/chat/chat.go:102-150`),
3. runs either a demo inference flow or a pinocchio runtime-backed inference flow (`pinocchio/pkg/evtstream/apps/chat/chat.go:190-220` and later sections of the same file).

The service layer also embeds `*infruntime.ComposedRuntime` directly in `PromptRequest` (`pinocchio/pkg/evtstream/apps/chat/service.go:13-18`). That is a pinocchio-specific contract, not a reusable sessionstream contract.

So the package is architecturally valuable, but not extraction-ready in its current exact form.

### Layer 5: Systemlab (`cmd/evtstream-systemlab`)

Systemlab is already behaving like a companion application for the substrate repo.

The README says it is a separate app used to explain, exercise, and validate the public API boundaries of `pkg/evtstream` (`pinocchio/cmd/evtstream-systemlab/README.md:3-10`). It is intentionally kept separate from substrate code and is allowed to consume only public `evtstream` APIs (`pinocchio/cmd/evtstream-systemlab/README.md:14-18`).

In other words, Systemlab is exactly what we would expect to live under:

```text
sessionstream/cmd/evtstream-systemlab
```

### Layer 6: product consumer (`cmd/web-chat`)

`cmd/web-chat` is already the first serious downstream consumer of the substrate.

The app server constructs the canonical runtime like this:

```text
SchemaRegistry
  -> chatapp.RegisterSchemas
HydrationStore
  -> memory or sqlite implementation
Websocket transport
  -> transport/ws.Server
Hub
  -> schema + store + fanout
chatapp.Install
chatapp.NewService
HTTP handlers around the service
```

See `pinocchio/cmd/web-chat/app/server.go:88-121`.

This is a good sign for extraction because it means the product app is no longer the owner of substrate internals. It is a consumer that wires a product edge around generic services.

## Dependency and extraction readiness analysis

The extraction should be guided by a package classification, not by a giant blind move.

### Classification matrix

| Area | Current state | Extraction readiness | Recommended action |
|---|---|---:|---|
| `pkg/evtstream` root | generic substrate | high | move almost unchanged |
| `pkg/evtstream/hydration/memory` | generic store | high | move almost unchanged |
| `pkg/evtstream/hydration/sqlite` | generic store | high | move almost unchanged |
| `pkg/evtstream/transport/ws` | generic websocket transport | high | move almost unchanged |
| `pkg/evtstream/examples/chat` | example consumer | medium-high | move after import rewrites |
| `cmd/evtstream-systemlab` | companion app consuming public seams | high | move with module |
| `pkg/evtstream/apps/chat` | useful app package, but pinocchio-coupled | medium | refactor during move |
| `cmd/web-chat/app` | product-specific edge/server | not for move | keep in pinocchio |
| `pkg/inference/runtime` | pinocchio runtime composition | not for move | keep in pinocchio |
| `pkg/middlewares/agentmode` | pinocchio middleware/runtime behavior | not for move | keep in pinocchio |

### What is already clean enough to move with light touch

These packages currently depend only on the generic `evtstream` root and external libraries:

- `hydration/memory`
- `hydration/sqlite`
- `transport`
- `transport/ws`
- the root substrate package itself

These should be migrated early because they are low-risk and they establish the new module skeleton quickly.

### What needs deliberate refactoring first

The critical coupling is inside `apps/chat`.

The evidence is straightforward:

- `PromptRequest.Runtime *infruntime.ComposedRuntime` in `service.go` (`pinocchio/pkg/evtstream/apps/chat/service.go:13-18`)
- imports of `pkg/inference/runtime` and `pkg/middlewares/agentmode` in `chat.go` (`pinocchio/pkg/evtstream/apps/chat/chat.go:14-17`)

That means the extracted repo would currently have to depend on pinocchio, which defeats the point. So `apps/chat` must be de-pinocchioed.

### What should stay in pinocchio

The following responsibilities are correctly product-owned today and should remain there:

- profile/runtime policy resolution,
- pinocchio middleware definitions,
- sink decoration based on pinocchio runtime configuration,
- canonical web routes and request/response contracts for `cmd/web-chat`.

The comments in `cmd/web-chat/app/runtime.go` and `cmd/web-chat/agentmode_sink.go` are explicit that these behaviors are app-owned and should stay out of `pkg/evtstream` core (`pinocchio/cmd/web-chat/app/runtime.go:8-12`, `pinocchio/cmd/web-chat/agentmode_sink.go:11-17`). The extraction should preserve that principle.

## Recommended target repository layout

The standalone repository should not preserve `pkg/evtstream` as a nested path. Once the repo itself is named `sessionstream`, the ergonomic public API is the repo root.

Recommended target layout:

```text
sessionstream/
  go.mod                         module github.com/go-go-golems/sessionstream
  README.md
  pkg/doc.go                     or root doc.go, depending on repo conventions

  doc.go
  types.go
  schema.go
  hub.go
  bus.go
  consumer.go
  fanout.go
  hydration.go
  projection.go
  handler.go
  command_registry.go
  session_registry.go
  ordinals.go
  noop_store.go

  hydration/
    memory/
      store.go
    sqlite/
      store.go

  transport/
    transport.go
    ws/
      server.go

  apps/
    chat/
      service.go
      engine.go or chat.go

  examples/
    chat/
      chat.go

  cmd/
    evtstream-systemlab/
      ...

  ttmp/
    ... docmgr tickets for the repo itself ...
```

### Why root package instead of `pkg/sessionstream`

A standalone library repo should feel natural to import:

```go
import "github.com/go-go-golems/sessionstream"
```

That is simpler and more idiomatic than:

```go
import "github.com/go-go-golems/sessionstream/pkg/sessionstream"
```

The current `pkg/evtstream` location made sense inside the larger pinocchio repo. It is no longer necessary once `sessionstream` is its own repository.

## Proposed package strategy

### Strategy A: root substrate stays minimal and generic

The root package should continue to own only generic session-streaming concepts:

- `SessionId`
- commands/events/UI events/timeline entities as substrate contracts
- schema registry
- hub orchestration
- session registry
- command registry
- timeline/UI projection contracts
- hydration store interface
- fanout interface
- optional bus integration

Anything that smells like “this is specifically how pinocchio does inference” should not live here.

### Strategy B: `apps/chat` stays, but becomes generic

A generic chat package is still valuable. It proves the substrate can support a real application domain and gives downstream consumers a starting point.

However, it should stop depending directly on pinocchio runtime types.

Recommended shape:

```go
type PromptRequest struct {
    Prompt         string
    IdempotencyKey string
    Runtime        RuntimeHandle // generic interface, not pinocchio type
}

type RuntimeHandle interface {
    Run(ctx context.Context, req PromptExecutionRequest, sink AssistantStreamSink) error
}

type AssistantStreamSink interface {
    OnStarted(meta MessageMeta)
    OnDelta(text string)
    OnStopped(err error)
    OnFinished(text string)
    OnCustomEvent(name string, payload map[string]any)
}
```

Pinocchio can then provide an adapter implementation that wraps:

- `*infruntime.ComposedRuntime`
- agentmode structured sink wrappers
- pinocchio-specific middleware events

This splits the system correctly:

```text
sessionstream/apps/chat
  -> generic chat domain contracts and projections

pinocchio adapters
  -> translate pinocchio runtime behavior into the generic chat RuntimeHandle
```

### Strategy C: pinocchio custom middleware stays downstream

The current `agentmode` behavior is real and valuable, but it is not generic sessionstream core behavior. It is a pinocchio feature implemented through middleware plus sink decoration.

So the correct ownership is:

- generic preview/commit custom-event pattern can be documented in `sessionstream`
- actual `agentmode` middleware implementation stays in `pinocchio`
- pinocchio emits sessionstream-compatible custom events through adapters

## Detailed migration plan

This plan is intentionally phased so a new intern can execute it without losing track of boundaries.

### Phase 0: bootstrap the sessionstream repository correctly

Goal: make the new repository structurally ready before any code move.

Tasks:

1. change `sessionstream/go.mod` from the template module path to `github.com/go-go-golems/sessionstream`,
2. replace template placeholders in `README.md`, `Makefile`, and any CI/release config,
3. keep `sessionstream/ttmp` as the docmgr home for all new extraction tickets,
4. add CI targets that mirror the substrate needs:
   - `go test ./...`
   - lint
   - optional boundary checks for Systemlab

Why first:

- it gives the extraction a real destination,
- it prevents “temporary template repo” drift,
- it makes later import-path rewrites less confusing.

### Phase 1: move the pure substrate packages

Goal: move the packages that are already clearly generic.

Move first:

- root `evtstream` files,
- `hydration/memory`,
- `hydration/sqlite`,
- `transport`,
- `transport/ws`.

Mechanical steps:

```text
copy files
  -> rewrite import path from pinocchio/pkg/evtstream to sessionstream
  -> run gofmt
  -> run go test ./...
```

Validation target:

- the new `sessionstream` repo builds and tests for the moved packages,
- no moved package imports `github.com/go-go-golems/pinocchio/...`.

### Phase 2: move Systemlab with import rewrites only

Goal: relocate the teaching and validation app into the same ecosystem as the substrate.

Move:

- `cmd/evtstream-systemlab/*`
- chapters
- static assets
- README

Important note:

Most of Systemlab is already a clean consumer of public `evtstream` seams. That is why it is a high-confidence move. The one caveat is Phase 6, which intentionally probes a live `cmd/web-chat` server over HTTP (`pinocchio/cmd/evtstream-systemlab/phase6_lab.go:112-196`). That is okay. It is still an external probe, not an import-time dependency on pinocchio internals.

### Phase 3: refactor `apps/chat` into a truly standalone app package

Goal: remove direct pinocchio dependencies from the moved chat package.

This is the highest-leverage design step.

Recommended sub-steps:

1. replace `*infruntime.ComposedRuntime` in `PromptRequest` with a generic execution interface,
2. move pinocchio-specific runtime execution into an adapter package in `pinocchio`,
3. preserve the existing command/event/UI/timeline schema names if possible,
4. keep the preview/commit custom-event pattern, but do not hard-code pinocchio middleware packages into `sessionstream`.

Pseudocode target:

```text
sessionstream/apps/chat
  handleStartInference()
    -> accept prompt
    -> publish user accepted event
    -> if request has RuntimeHandle
         RuntimeHandle.Run(ctx, prompt, sink)
       else
         run demo inference

pinocchio runtime adapter
  Run(ctx, prompt, sink)
    -> build geppetto session from pinocchio runtime
    -> feed deltas into sink
    -> translate agentmode custom signals into sink.OnCustomEvent(...)
```

### Phase 4: switch pinocchio to consume sessionstream

Goal: make pinocchio a real downstream consumer.

Tasks:

1. add the new module to `pinocchio/go.mod` or `go.work`,
2. rewrite imports in:
   - `cmd/web-chat/app/server.go`
   - `cmd/evtstream-systemlab` if it still exists temporarily during transition
   - any examples/tests still using `pinocchio/pkg/evtstream`
3. add temporary local `replace` directives while the new repo is under active development,
4. ensure `cmd/web-chat` still passes focused tests and browser validation.

### Phase 5: remove the old in-tree `pkg/evtstream` copy from pinocchio

Goal: finish the cutover and avoid dual ownership.

This should happen only after:

- `pinocchio` imports the external module cleanly,
- tests pass using the external module,
- no important package still imports `pinocchio/pkg/evtstream`.

## API and compatibility guidance

### Preserve the core mental model

The extraction should preserve the architectural identity that already proved itself:

```text
SessionId
  -> Command
  -> backend Event
  -> TimelineProjection + UIProjection
  -> HydrationStore + UIFanout
  -> Snapshot + UI event delivery
```

This is the conceptual API. The whole point of extraction is to make that model reusable without exposing webchat legacy baggage.

### Avoid carrying webchat compatibility into sessionstream

Do not move any of these into the new repo:

- `/chat` legacy route shapes
- `/api/timeline` compatibility behavior
- browser edge compatibility shims
- profile cookie policy
- legacy websocket attach semantics

Those belong in application repos, not in a reusable sessionstream substrate.

### Prefer extension points over pinocchio-specific special cases

When you encounter a feature that is “really useful, but currently pinocchio-shaped,” ask this question:

> Is there a generic extension point here, or is this truly product behavior?

Examples:

- generic custom event publishing pattern -> belongs in `sessionstream`
- `agentmode` middleware implementation -> belongs in `pinocchio`
- generic websocket fanout transport -> belongs in `sessionstream`
- `cmd/web-chat` profile resolution -> belongs in `pinocchio`

## Testing and validation plan

A standalone extraction is successful only if the same behavior can be proved on both sides of the seam.

### Repo-local validation inside `sessionstream`

Minimum:

```bash
cd sessionstream
GOWORK=off go test ./...
```

Add repo-specific targets later, but keep this baseline simple and reliable.

### Consumer validation inside `pinocchio`

After rewiring imports:

```bash
cd pinocchio
go test ./cmd/web-chat/... -count=1
cd cmd/web-chat/web && npm run check
```

If the extraction touches Systemlab during a staged move, also validate the relocated app in whichever repo currently owns it.

### Behavior validation checklist

1. generic hub tests still pass,
2. hydration store tests still pass,
3. websocket transport tests still pass,
4. Systemlab still runs and teaches the same phases,
5. `cmd/web-chat` still creates sessions, submits prompts, hydrates snapshots, and receives websocket updates,
6. runtime-backed inference still works through the pinocchio adapter,
7. custom preview/commit events (such as agentmode) still render correctly in the consumer app.

## Risks and mitigations

### Risk 1: dragging pinocchio into the new module through `apps/chat`

Mitigation:

- treat `apps/chat` as the main refactor seam,
- add an explicit “no `pinocchio/...` imports in sessionstream” check,
- move runtime adapters downstream.

### Risk 2: dual ownership during a long transition

Mitigation:

- use a short-lived local `replace` period,
- switch imports deliberately,
- delete the old in-tree package after the new module is proven.

### Risk 3: Systemlab accidentally becoming product-specific

Mitigation:

- keep the README boundary contract intact,
- preserve the rule that Systemlab only uses public seams,
- allow HTTP probes against consumers, but not direct consumer-package imports.

### Risk 4: over-generalizing too early

Mitigation:

- move pure substrate first,
- only generalize what blocks extraction,
- keep the initial public API close to what already works.

## Alternatives considered

### Alternative A: leave `evtstream` inside pinocchio forever

Rejected because it keeps the module boundary soft and makes downstream reuse feel incidental rather than intentional.

### Alternative B: move everything, including web-chat app code, into sessionstream

Rejected because `cmd/web-chat` is a product application with pinocchio-specific runtime policy and application-edge behavior. Moving it would blur the framework boundary.

### Alternative C: extract only the root substrate and leave Systemlab behind

Rejected because Systemlab was designed specifically to exercise the public substrate boundary. It belongs beside the framework as a consumer app.

### Alternative D: move `apps/chat` unchanged and let sessionstream depend on pinocchio

Rejected because that would invert the dependency direction and make the new repo less reusable than the current intent demands.

## Open questions

1. Should the extracted repo use root package files directly, or keep a small `pkg/` directory to match go-go-golems template conventions?
   - Recommendation: root package for ergonomics, but this is a repo-level style call.
2. Should `apps/chat` remain in the first extraction cut, or should it be split into “generic chat” and “pinocchio runtime adapter” as two sequential tickets?
   - Recommendation: do the split during the extraction program, but allow two commits/phases.
3. Should Systemlab keep the Phase 6 web-chat migration lab after the move?
   - Recommendation: yes, because it is a valuable external-consumer regression lab, and it reaches the consumer over HTTP rather than by private imports.
4. Should examples move in the first cut or after the core stabilizes?
   - Recommendation: move them with the substrate if they remain lightweight and generic; otherwise stage them one phase later.

## Concrete implementation checklist for a new intern

### First read these files

Core substrate:

- `pinocchio/pkg/evtstream/doc.go`
- `pinocchio/pkg/evtstream/hub.go`
- `pinocchio/pkg/evtstream/schema.go`
- `pinocchio/pkg/evtstream/types.go`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go`
- `pinocchio/pkg/evtstream/transport/ws/server.go`

App-grade chat and consumer wiring:

- `pinocchio/pkg/evtstream/apps/chat/service.go`
- `pinocchio/pkg/evtstream/apps/chat/chat.go`
- `pinocchio/cmd/web-chat/app/server.go`
- `pinocchio/cmd/web-chat/app/runtime.go`
- `pinocchio/cmd/web-chat/agentmode_sink.go`

Systemlab:

- `pinocchio/cmd/evtstream-systemlab/README.md`
- `pinocchio/cmd/evtstream-systemlab/lab_environment.go`
- `pinocchio/cmd/evtstream-systemlab/phase4_lab.go`
- `pinocchio/cmd/evtstream-systemlab/phase5_lab.go`
- `pinocchio/cmd/evtstream-systemlab/phase6_lab.go`

Destination repo bootstrap:

- `sessionstream/go.mod`
- `sessionstream/Makefile`
- `sessionstream/.ttmp.yaml`

### Then execute in this order

1. bootstrap the destination repo,
2. move pure substrate packages,
3. move Systemlab,
4. refactor `apps/chat` to remove pinocchio imports,
5. switch pinocchio to consume the external module,
6. delete the old in-tree substrate copy.

### Do not do these things

- do not move `cmd/web-chat` into the framework repo,
- do not preserve legacy webchat compatibility routes in the new module,
- do not import `pinocchio/pkg/inference/runtime` from `sessionstream`,
- do not let Systemlab import consumer internals just because they are convenient during the move.

## Reference snippets

### Current canonical consumer assembly

From `cmd/web-chat/app/server.go` the current pattern is effectively:

```text
reg := sessionstream.NewSchemaRegistry()
chatapp.RegisterSchemas(reg)
store := memory or sqlite hydration store
ws := transport/ws.NewServer(snapshotProvider)
hub := sessionstream.NewHub(...)
chatapp.Install(hub, engine)
service := chatapp.NewService(hub, engine)
```

That pattern should survive the extraction almost unchanged; only the import paths and the runtime adapter boundary should differ.

### Desired downstream relationship

```text
pinocchio runtime + middleware
  -> pinocchio adapter
      -> sessionstream/apps/chat
          -> sessionstream Hub / store / transport
```

This is the key mental model for the whole project.

## References

### Key evidence files

- `pinocchio/pkg/evtstream/doc.go:1-10`
- `pinocchio/pkg/evtstream/hub.go:21-41`
- `pinocchio/pkg/evtstream/hub.go:90-107`
- `pinocchio/pkg/evtstream/hub.go:145-167`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go:10-18`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go:61-104`
- `pinocchio/pkg/evtstream/hydration/sqlite/store.go:163-175`
- `pinocchio/pkg/evtstream/transport/ws/server.go:17-36`
- `pinocchio/pkg/evtstream/transport/ws/server.go:57-98`
- `pinocchio/pkg/evtstream/transport/ws/server.go:149-171`
- `pinocchio/pkg/evtstream/apps/chat/service.go:13-18`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:14-17`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:20-43`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:102-150`
- `pinocchio/pkg/evtstream/apps/chat/chat.go:190-220`
- `pinocchio/cmd/evtstream-systemlab/README.md:3-24`
- `pinocchio/cmd/evtstream-systemlab/phase6_lab.go:112-196`
- `pinocchio/cmd/web-chat/app/server.go:88-121`
- `pinocchio/cmd/web-chat/app/runtime.go:8-12`
- `pinocchio/cmd/web-chat/agentmode_sink.go:11-17`
- `sessionstream/go.mod:1-2`
- `sessionstream/.ttmp.yaml:1-5`

### Related ticket donor material

- `le-chat/ttmp/2026/04/20/EVT-STREAM-012--post-stabilization-evtstream-standalone-module-extraction-and-systemlab-relocation/design-doc/01-post-stabilization-standalone-module-extraction-plan.md`

This earlier ticket is useful donor material, but the current ticket is the canonical planning ticket because it lives in the destination repository (`sessionstream`) and is written specifically around the new repository name and ownership model.
