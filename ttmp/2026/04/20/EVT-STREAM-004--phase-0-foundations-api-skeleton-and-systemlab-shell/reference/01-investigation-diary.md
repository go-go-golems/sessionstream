---
Title: Investigation Diary
Ticket: EVT-STREAM-004
Status: active
Topics:
    - architecture
    - framework
    - event-streaming
    - implementation
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/doc.go
      Note: Public package entrypoint for the new substrate.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/server.go
      Note: HTTP shell used to keep Systemlab separate from the substrate package.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/Makefile
      Note: Phase 0 validation targets and boundary-check commands.
ExternalSources: []
Summary: "Chronological diary for EVT-STREAM-004 covering the new evtstream package scaffold, separate Systemlab shell, boundary checks, and Phase 0 validation artifacts."
LastUpdated: 2026-04-19T23:45:00-04:00
WhatFor: "Record how the Phase 0 substrate boundary and shell app were implemented, validated, and prepared for follow-on work."
WhenToUse: "When reviewing the Phase 0 implementation, understanding the import-cycle fix, or continuing the shell and boundary work later."
---

# Diary

## Goal

Capture the implementation work for EVT-STREAM-004 so a reviewer can see exactly how the Phase 0 scaffold was built, why the public boundary was chosen, what validation passed, and what failures happened while getting the shell into a reviewable state.

## Step 1: Scaffold the public evtstream API and separate Systemlab shell

I continued the previously requested Phase 0 and Phase 1 build-out by first locking in the Phase 0 shape: a new `pkg/evtstream` package for the substrate contract and a separate `cmd/evtstream-systemlab` application that only talks to public `evtstream` seams. The goal for this step was not feature depth yet; it was to make the architectural boundary real in code.

The important constraint from the ticket set was that Systemlab must remain a separate app and must not quietly reuse `pkg/webchat` internals. I therefore treated the shell, README, validation targets, and captured status artifact as Phase 0 deliverables in their own right, even though the implementation later grew Phase 1 behavior in the same code path.

### Prompt Context

**User prompt (verbatim):** "continue"

**Assistant interpretation:** Continue the already-started Phase 0 and Phase 1 implementation work, keep the work split cleanly by ticket intent, commit the code, and record the result in detailed diary entries.

**Inferred user intent:** The user wants the planning tickets turned into real code and wants enough execution history captured that someone else can review or continue the work without re-discovering the implementation path.

**Commit (code):** 142d77bc3376e4cd5de946314304764bb093a064 — "feat(evtstream): build phase 0 shell and phase 1 in-memory lab"

### What I did
- Created the new public substrate package under `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream` with the initial contract files:
  - `doc.go`
  - `types.go`
  - `handler.go`
  - `projection.go`
  - `schema.go`
  - `hydration.go`
  - `hub.go`
  - `transport/transport.go`
- Added root-local fallback store scaffolding in `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/noop_store.go` so `evtstream` does not import the memory hydration implementation directly.
- Added the separate Systemlab app shell under `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab`:
  - `README.md`
  - `main.go`
  - `server.go`
  - `static/index.html`
- Chose an HTTP seam for the shell and exposed `/api/status` plus placeholder navigation for Phase 0 through Phase 6.
- Added explicit Phase 0 validation targets to `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/Makefile`:
  - `evtstream-test`
  - `systemlab-build`
  - `systemlab-run`
  - `evtstream-boundary-check`
  - `evtstream-check`
- Copied the key boundary rules into `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/README.md`.
- Ran the shell locally and captured the status payload as `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-004--phase-0-foundations-api-skeleton-and-systemlab-shell/reference/02-systemlab-status.json`.

### Why
- The ticket explicitly required the new framework to be its own substrate instead of a renamed `pkg/webchat` surface.
- The separate Systemlab app is the mechanism that makes the API boundary testable instead of aspirational.
- A root-local noop store is the clean dependency-inversion fix for the `evtstream -> memory -> evtstream` import-cycle trap.
- Make targets make the boundary reviewable by command, not just by prose.

### What worked
- The new package tree compiled cleanly and gave the framework a stable home under `pkg/evtstream`.
- The Systemlab shell stayed isolated from `pkg/webchat` and rendered a real overview page with placeholder lab navigation.
- The boundary-check target worked once it was narrowed to scan `*.go` files only.
- The captured status artifact demonstrates the shell boundary and available labs without requiring a reviewer to run the app first.

### What didn't work
- My first attempt to default `NewHub(...)` to the in-memory store caused an import cycle. The failing command was:
  - `cd pinocchio && go test ./pkg/evtstream/...`
- The exact failure was:

```text
package github.com/go-go-golems/pinocchio/pkg/evtstream
	imports github.com/go-go-golems/pinocchio/pkg/evtstream/hydration/memory from hub.go
	imports github.com/go-go-golems/pinocchio/pkg/evtstream from store.go: import cycle not allowed
```

- My first version of `evtstream-boundary-check` searched all files under `cmd/evtstream-systemlab`, which caused a false failure because the README intentionally mentioned `pkg/webchat` in prose. I fixed that by restricting the search to `--glob '*.go'`.

### What I learned
- The right place to solve the memory-store cycle is architecture, not syntax: `evtstream` owns interfaces, concrete stores live in leaf subpackages, and the app injects them.
- Even a small shell app benefits from a first-class README and make targets because those become the executable form of the boundary contract.
- Capturing a tiny JSON status artifact is a low-cost way to satisfy the phase’s “show me it exists” requirement.

### What was tricky to build
- The tricky part was preserving the one-way dependency graph while still making `NewHub(...)` usable by default. The symptom was the import-cycle error above. The fix was to move the default behavior to a root-local noop store (`noop_store.go`) and reserve `/hydration/memory` for opt-in injection from callers.
- The second sharp edge was that `go build ./cmd/evtstream-systemlab` in a Make target emitted a binary into the repo root. I changed the target to `go build -o /tmp/evtstream-systemlab ./cmd/evtstream-systemlab` so Phase 0 validation would not create accidental repo noise.

### What warrants a second pair of eyes
- Whether the current root package file split is the exact public API shape we want before Watermill is introduced.
- Whether the HTTP shell should stay in-process for now or gain a thinner abstraction before Phase 2.
- Whether the Make target names fit the repo’s existing conventions closely enough.

### What should be done in the future
- Add a small screenshot artifact for the shell if we want a visual Phase 0 record in addition to the JSON status payload.
- Revisit whether some of the current public types should be grouped into subpackages before external consumers appear.

### Code review instructions
- Start in `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/doc.go` and `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/types.go` to review the substrate vocabulary.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/README.md`, `server.go`, and `static/index.html` to confirm the shell only uses public seams.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && make evtstream-check`

### Technical details
- Public seam chosen for Systemlab: HTTP, with the shell exposing its own endpoints and consuming public `evtstream` contracts internally.
- Key files:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/noop_store.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/server.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/Makefile`
- Captured artifact:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-004--phase-0-foundations-api-skeleton-and-systemlab-shell/reference/02-systemlab-status.json`

## Step 2: Split the Systemlab frontend into modular assets before it grows

After the initial shell landed, I did a small but important maintainability pass on the browser UI. The immediate concern was that future Phase 2 through Phase 6 labs would quickly turn `static/index.html` into an oversized file with inline styles, inline JavaScript, and page-specific markup all mixed together.

I therefore moved the Systemlab shell to a modular static layout now, while it is still small. The result is that the app shell, shared styling, page partials, shared JavaScript helpers, and per-page logic each live in their own files, making it much easier to add new labs without producing one large unreviewable blob.

### Prompt Context

**User prompt (verbatim):** "Continue, make sure to split the js and potentially html into multiple files so that we don't end up with huge files for system lab?"

**Assistant interpretation:** Continue refining the Systemlab implementation, specifically by breaking the current browser UI into smaller frontend files so future lab work does not accumulate in one monolithic HTML file.

**Inferred user intent:** The user wants the Systemlab app to remain maintainable as more phases are implemented, so structural cleanup should happen now rather than after the UI becomes large and brittle.

**Commit (code):** ef3165eb85bdb736bd95d37bcfc90cb45059a869 — "refactor(systemlab): split frontend into modular assets"

### What I did
- Replaced the inline-style and inline-script version of `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/index.html` with a lightweight shell document.
- Added shared styling in `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/app.css`.
- Split page markup into partials:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/overview.html`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/partials/phase1.html`
- Split JavaScript into modules:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/main.js`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/api.js`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/dom.js`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/overview.js`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/pages/phase1.js`
- Updated `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/README.md` with the intended frontend file layout and the rule that new labs should add a new partial + page module instead of expanding one global file.
- Validated the split by running the server, fetching static assets directly with `curl`, and opening the page with Playwright to confirm that both the overview page and the Phase 1 page still render.

### Why
- Every additional lab would otherwise have appended more HTML, CSS, and JavaScript into one file.
- Per-page files are a better fit for the Systemlab concept because each phase page is meant to become its own feature exerciser.
- Doing the split early avoids churn later when more labs exist and the single-file structure is harder to unwind.

### What worked
- The embedded static server served the new CSS, partial HTML, and JavaScript module files without any backend changes.
- The browser UI still rendered correctly after the split.
- The Phase 1 page still loaded and displayed its controls and outputs after being moved into a partial plus page module.
- The structure now has a clear place for future pages to grow.

### What didn't work
- There were no code-level failures in this refactor after the split landed. The only browser console error observed during Playwright validation was the expected missing favicon request:

```text
[ERROR] Failed to load resource: the server responded with a status of 404 (Not Found) @ http://127.0.0.1:18093/favicon.ico:0
```

### What I learned
- The built-in static file server and embedded assets are enough for a clean modular frontend as long as the structure is kept simple.
- HTML partial loading is a good middle ground here: it avoids a frontend build tool while still preventing `index.html` from becoming a dumping ground.

### What was tricky to build
- The main subtlety was choosing a split that improves maintainability without inventing a heavier frontend stack. The solution was to keep everything static-file-based and browser-native: `type="module"` JavaScript, fetched partial HTML, and a shared CSS file. That gives us modularity without introducing npm, bundling, or a second build pipeline for the shell.

### What warrants a second pair of eyes
- Whether partial-per-page is the right long-term unit or whether later labs should move to a small client-side component library.
- Whether we want to add a trivial `/favicon.ico` to silence the harmless browser 404.

### What should be done in the future
- Add one partial + one page module per new phase page so the structure stays flat and reviewable.
- If multiple pages start sharing richer widgets, extract a `widgets/` or `components/` layer rather than pushing logic back into `main.js`.

### Code review instructions
- Start with `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/index.html` to verify it is now just the shell.
- Then review `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/static/js/main.js`, `static/js/api.js`, and `static/js/pages/phase1.js`.
- Validate with:
  - `cd /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio && go test ./pkg/evtstream/... ./cmd/evtstream-systemlab/...`
  - `go run ./cmd/evtstream-systemlab --addr :18093`
  - Open `http://127.0.0.1:18093/` and switch between Overview and Phase 1.

### Technical details
- Static asset layout introduced:
  - `static/index.html`
  - `static/app.css`
  - `static/partials/*.html`
  - `static/js/*.js`
  - `static/js/pages/*.js`
- Browser validation performed with:
  - `curl -sf http://127.0.0.1:18092/`
  - `curl -sf http://127.0.0.1:18092/partials/overview.html`
  - `curl -sf http://127.0.0.1:18092/js/main.js`
  - Playwright navigation to `http://127.0.0.1:18093/`
