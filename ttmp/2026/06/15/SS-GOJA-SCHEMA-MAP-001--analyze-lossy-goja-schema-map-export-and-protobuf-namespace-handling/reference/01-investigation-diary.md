---
Title: Investigation diary
Ticket: SS-GOJA-SCHEMA-MAP-001
Status: active
Topics:
    - sessionstream
    - goja
    - protobuf
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/js/modules/sessionstream/api_schemas.go
      Note: Implemented direct Goja bulk schema traversal and removed lossy map schema resolution
    - Path: pkg/js/modules/sessionstream/module_test.go
      Note: Regression tests for bulk namespace/string schemas and descriptor rejection
    - Path: ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/scripts/01-search-lossy-js-boundaries.sh
      Note: Reproducible inventory script for lossy JS/Go conversion boundaries
    - Path: ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/scripts/02-reproduce-bulk-schema-failure.sh
      Note: Temporary-test reproduction of current bulk schema failures
    - Path: ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/scripts/03-inspect-goja-export-loss.sh
      Note: Temporary Go program generator proving ExportTo strips hidden namespace refs
ExternalSources: []
Summary: Chronological investigation diary for the lossy Goja schema map export analysis.
LastUpdated: 2026-06-15T19:40:00-04:00
WhatFor: Records what was inspected, reproduced, and documented for SS-GOJA-SCHEMA-MAP-001.
WhenToUse: Read before implementing the schema bulk registration fix or resuming the ticket.
---



# Diary

## Goal

This diary records the investigation into lossy Goja schema map handling in `sessionstream`, including the accidental speculative edit that was reverted, the reproduction scripts that were retained in the ticket workspace, and the analysis/design guide produced for the follow-up implementation.

## Step 1: Initial schema map investigation and reverted speculative patch

I first inspected the `sessionstream` JavaScript schema registration path to understand the review comment around `map[string]any` and generated protobuf namespaces. The key finding was that `ss.schemas({...})` uses `goja.ExportTo` before resolving schema leaves, while the fluent `schemas.registerCommand(...)` path receives the original `goja.Value` and can still recover the hidden `protogoja` prototype token.

I briefly drafted a direct-Goja walker implementation in `pkg/js/modules/sessionstream/api_schemas.go`, then stopped when the user asked to study the system first and create a ticketed analysis/design guide instead. I reverted the speculative code change with `git checkout -- pkg/js/modules/sessionstream/api_schemas.go`, leaving the repository source unchanged before creating the ticket.

### Prompt Context

**User prompt (verbatim):** "Are there other lossy map strings any to be found? we should do the proper fix."

**Assistant interpretation:** Search for other lossy `map[string]any` or `ExportTo` boundaries and consider a more principled fix than only adding `typeName` to the existing map branch.

**Inferred user intent:** Avoid a narrow patch that misses related schema/protobuf object handling problems.

**Commit (code):** N/A — no code commit; speculative edit was reverted.

### What I did
- Read `pkg/js/modules/sessionstream/api_schemas.go`.
- Searched for `ExportTo`, `map[string]any`, and related conversion boundaries.
- Inspected generated protobuf namespace code in `examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat_goja.pb.go`.
- Inspected `go-go-goja/pkg/protogoja/prototype.go` and `ref.go` to understand hidden namespace and payload references.
- Drafted and then reverted a direct-Goja walker patch in `api_schemas.go`.

### Why
- The review comment suggested that the current map branch loses hidden generated-protobuf prototype metadata.
- A proper fix needed to distinguish schema namespace objects from built protobuf payload objects.

### What worked
- The source inspection confirmed that fluent registration already takes the correct `goja.Value` path.
- The initial patch sketch clarified the shape of the likely implementation: walk the original schema input object directly and reuse `mustResolveSchemaValue`.

### What didn't work
- I made a speculative edit before creating the requested analysis ticket. That edit was reverted with:

```bash
git checkout -- pkg/js/modules/sessionstream/api_schemas.go
```

### What I learned
- `map[string]any` is not the root concept; the root problem is crossing a lossy JS-to-Go export boundary before consuming hidden `protogoja` metadata.
- `MessagePrototypeFromValue` and `MessageFromValue` are intentionally separate APIs: schema namespaces use prototype refs; built payload messages use message refs.

### What was tricky to build
- The tricky part was recognizing that adding `typeName` to the existing map branch is only a compatibility fallback, not the proper fix. Once `ExportTo` has run, the authoritative hidden token is already gone.
- The other tricky point was not confusing JSON payload fallback in `codec.go` with schema-token handling in `api_schemas.go`; payload fallback is intentionally JSON-shaped after checking `protogoja.MessageFromValue`.

### What warrants a second pair of eyes
- Confirm whether any consumers intentionally pass capitalized `Commands`/`Events` section names today. The public TypeScript contract uses lower-case names.
- Confirm whether object descriptors with `{ type: ... }` should remain supported when refactoring away from `mustResolveSchemaSpec`.

### What should be done in the future
- Implement the direct-Goja schema input walker described in the design doc.
- Add failing tests before changing production code.

### Code review instructions
- Start at `pkg/js/modules/sessionstream/api_schemas.go` and compare the bulk `schemas(input)` path with the fluent `registerCommand` path.
- Validate with package-level tests once implementation starts: `go test ./pkg/js/modules/sessionstream -count=1`.

### Technical details
- Existing failing area: `pkg/js/modules/sessionstream/api_schemas.go:11-78`.
- Existing working fluent path: `pkg/js/modules/sessionstream/api_schemas.go:80-128`.
- Generated namespace hidden token API: `go-go-goja/pkg/protogoja/prototype.go:50-91`.

## Step 2: Ticket creation, reproducible scripts, and intern-ready design guide

After the user asked to pause implementation, I created ticket `SS-GOJA-SCHEMA-MAP-001` and moved the investigation into a docmgr workspace. I also retroactively saved the temporary reproduction and inspection programs as durable scripts under the ticket `scripts/` directory so future readers can rerun the exact evidence-gathering steps.

The main deliverable is a detailed design and implementation guide explaining the sessionstream schema registry, Goja generated protobuf namespace objects, hidden `protogoja` references, current failure modes, proposed direct-Goja walker fix, testing strategy, and whether `go-go-goja` itself should change.

### Prompt Context

**User prompt (verbatim):** "hold on, let's study things first and create a new ticket and an analysis + implemengation guide. 

We also want to consider if changing ./go-go-goja/ and its protobuf handling itself makes sense to be upgraded? 

Create a new ticket and then:

Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Stop implementation, create a docmgr ticket, investigate evidence first, write a comprehensive intern-ready analysis/design/implementation guide, consider upstream `go-go-goja` changes, store everything in the ticket, and upload the result to reMarkable.

**Inferred user intent:** Produce a durable design artifact before coding so the eventual fix is evidence-based and reviewable.

**Commit (code):** N/A — documentation and ticket artifacts only.

### What I did
- Created ticket `SS-GOJA-SCHEMA-MAP-001` with docmgr.
- Added design doc `design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md`.
- Added diary doc `reference/01-investigation-diary.md`.
- Added ticket tasks for inventory, reproduction, guide writing, validation, and upload.
- Wrote scripts:
  - `scripts/01-search-lossy-js-boundaries.sh`
  - `scripts/02-reproduce-bulk-schema-failure.sh`
  - `scripts/03-inspect-goja-export-loss.sh`
- Captured script outputs under `sources/`.
- Wrote the intern-ready design guide.

### Why
- The task changed from implementation to research/design.
- The reproduction scripts preserve exact commands and prevent future investigators from relying on vague memory of temporary experiments.

### What worked
- `02-reproduce-bulk-schema-failure.sh` reproduced two failures:
  - lower-case `commands` led to `unknown command "ChatStartInference"`;
  - capitalized `Commands` led to `TypeError: schema values must be protobuf full-name strings or generated message namespace objects`.
- `03-inspect-goja-export-loss.sh` showed that lower-case `commands` did not populate the Go struct in the reproduction, while capitalized `Commands` exported the namespace as `map[string]interface {}` with public keys `[builder clone from is typeName]` and no `type`.

### What didn't work
- The reproduction test intentionally fails under current code. Its output is saved in `sources/02-reproduce-bulk-schema-failure.txt` and should be interpreted as evidence, not as a validation pass.
- reMarkable upload has not yet been completed at the time of this diary step.

### What I learned
- The review comment was directionally correct but understated the lower-case key issue: the advertised TypeScript `SchemaMap` uses lower-case keys, and the reproduction showed those keys did not populate `schemaInput` via `ExportTo`.
- The `go-go-goja` hidden-token APIs are behaving as designed; sessionstream should avoid destroying the hidden token before asking `protogoja` to read it.

### What was tricky to build
- The reproduction needed to actually submit through the hub. A test that only constructs `ss.schemas({...})` can pass even if the registry is empty, because the error appears later when resolving a command payload.
- The temporary inspection program needed to compare lower-case `commands` with capitalized `Commands` to expose both distinct failure modes.

### What warrants a second pair of eyes
- Review the proposed fallback support for both `typeName` and `type` object descriptors.
- Review the recommendation not to change `go-go-goja.MessagePrototypeFromValue` semantics.
- Review whether namespace `typeName` should become read-only in generated `go-go-goja` code as a separate upstream hardening task.

### What should be done in the future
- Run `docmgr doctor` after file relations and changelog updates are complete.
- Upload the ticket bundle to reMarkable after validation.
- Implement the direct-Goja walker in a follow-up code change.

### Code review instructions
- Review the design doc first, especially "Proposed design", "Decision records", and "Implementation guide for a new intern".
- Rerun evidence scripts from the ticket root if needed:

```bash
./scripts/01-search-lossy-js-boundaries.sh
./scripts/02-reproduce-bulk-schema-failure.sh
./scripts/03-inspect-goja-export-loss.sh
```

### Technical details
- Ticket path: `ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling`.
- Design doc: `design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md`.
- Evidence outputs: `sources/01-lossy-js-boundaries.txt`, `sources/02-reproduce-bulk-schema-failure.txt`, `sources/03-inspect-goja-export-loss.txt`.

## Step 3: Retroactively preserve investigation scripts

The user explicitly asked that all scripts I write be saved in the ticket `scripts/` folder, including scripts already used. I confirmed the temporary scripts had been converted into tracked ticket scripts and that their outputs were stored in `sources/`.

### Prompt Context

**User prompt (verbatim):** "save all the scripts you write in the scripts/ folder of the ticket, retroactively too, so we can keep track of your work."

**Assistant interpretation:** Ensure every ad-hoc reproduction or inspection script from this investigation exists as a durable script in the ticket workspace.

**Inferred user intent:** Keep the investigation reproducible and auditable for future continuation or review.

**Commit (code):** N/A — ticket artifact update only.

### What I did
- Saved the lossy boundary search as `scripts/01-search-lossy-js-boundaries.sh`.
- Saved the temporary failing test reproduction as `scripts/02-reproduce-bulk-schema-failure.sh`.
- Saved the temporary `goja.ExportTo` inspection program generator as `scripts/03-inspect-goja-export-loss.sh`.
- Made scripts executable.
- Stored outputs in `sources/`.

### Why
- Reproducible scripts make the design guide evidence-backed and easy for an intern to verify.

### What worked
- All scripts are now in the ticket workspace and can be rerun from a clean checkout.

### What didn't work
- N/A

### What I learned
- The ticket workflow should save scripts before or immediately after first execution; doing so retroactively is possible but easier to miss.

### What was tricky to build
- The scripts are stored several directories below the repository root, so they use `git -C "$SCRIPT_DIR" rev-parse --show-toplevel` instead of fragile relative `../../..` paths.

### What warrants a second pair of eyes
- Confirm the script set is sufficient, or add a fourth script later for validating the final implementation once code changes begin.

### What should be done in the future
- Add `04-validate-implemented-fix.sh` after the actual production patch is written.

### Code review instructions
- Start with the scripts in numerical order.
- Compare each script output with the corresponding file under `sources/`.

### Technical details
- Scripts are under `ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/scripts/`.

## Step 4: Validation and reMarkable delivery

I validated the ticket with `docmgr doctor`, rendered a local PDF bundle, fixed a Mermaid parse issue in the design document, and uploaded the corrected bundle to reMarkable. The final remote listing confirms the bundle exists at the requested date/ticket destination.

The Mermaid issue was caused by a flowchart node label containing `builder().build()` punctuation. I changed that label to plain quoted text, regenerated the PDF successfully, and re-uploaded with `--force` so the reMarkable copy reflects the corrected diagram.

### Prompt Context

**User prompt (verbatim):** (same as Step 2 and Step 3)

**Assistant interpretation:** Complete ticket validation and upload the generated analysis bundle to reMarkable.

**Inferred user intent:** Make the design guide available both in the repository ticket workspace and on the reMarkable device.

**Commit (code):** N/A — documentation and generated PDF only.

### What I did
- Ran `docmgr doctor --ticket SS-GOJA-SCHEMA-MAP-001 --stale-after 30`.
- Ran `remarquee status` and `remarquee cloud account --non-interactive`.
- Ran a dry-run bundle upload.
- Ran the real bundle upload.
- Observed Mermaid render warnings, fixed the flowchart label, generated a local PDF with `--pdf-only`, and re-uploaded with `--force`.
- Verified remote listing with `remarquee cloud ls /ai/2026/06/15/SS-GOJA-SCHEMA-MAP-001 --long --non-interactive`.

### Why
- The ticket-research workflow requires validation before delivery and a dry-run before upload.
- The first upload succeeded but had Mermaid render warnings, so the bundle needed a corrected re-upload.

### What worked
- `docmgr doctor` passed with all checks.
- `remarquee upload bundle --pdf-only` generated `sources/SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide.pdf` without warnings after the Mermaid fix.
- The final remote listing showed:

```text
[f]	SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide
```

### What didn't work
- First real upload produced Mermaid warnings:

```text
WARNING: failed to render Mermaid block 1: mmdc failed: Generating single mermaid chart
Error: Parse error on line 3:
...renceCommand.builder().build() payload]
-----------------------^
```

### What I learned
- Mermaid flowchart labels that include method-call punctuation should be quoted or simplified for reliable PDF rendering.

### What was tricky to build
- The upload command succeeded even while Mermaid rendering warned. I treated the warning as a quality issue, fixed the source document, verified local PDF rendering, and then overwrote the remote copy intentionally with `--force`.

### What warrants a second pair of eyes
- Review the rendered PDF if diagram fidelity matters; the corrected flowchart uses a simplified label instead of the exact JavaScript method-call string.

### What should be done in the future
- Add a fourth script for validating the implementation after the production code patch is written.

### Code review instructions
- Check `design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md` for the final corrected Mermaid flowchart.
- Check `sources/SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide.pdf` if reviewing the uploaded PDF contents locally.

### Technical details
- reMarkable destination: `/ai/2026/06/15/SS-GOJA-SCHEMA-MAP-001`.
- Uploaded document name: `SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide`.

## Step 5: Tighten the design to remove object descriptor fallback

The design was updated to reflect the final API choice: bulk schemas should support only generated `MessageNamespace` objects and primitive protobuf full-name strings. Plain descriptor objects such as `{ type: "..." }` and `{ typeName: "..." }` should be rejected rather than documented or kept for compatibility.

This aligns the implementation plan with the fact that the code is still unpublished. The `map[string]any` branch is an artifact of `ExportTo`; removing `ExportTo` also removes the need for descriptor fallback support.

### Prompt Context

**User prompt (verbatim):** "ok, keep it. Otherwise, we can do a hard cutover and not worry about backwards support, since this is still unpublished code. Update your design documentation to make this tighter API, and add a detailed list of tasks to the ticket. 

THen implement them one by one, committing at appropriate intervals, keeping a detailed adiary."

**Assistant interpretation:** Keep string support, remove object descriptor fallback, update the ticket design and tasks, then implement the fix incrementally with commits and diary updates.

**Inferred user intent:** Prefer a clean unpublished API over backwards-compatible support for accidental descriptor objects.

**Commit (code):** pending — documentation/task update before implementation.

### What I did
- Added detailed follow-up tasks to the ticket for design tightening, tests, implementation, validation, and diary/changelog updates.
- Updated the design guide to remove `{ type: ... }` / `{ typeName: ... }` support from the proposed implementation.
- Updated decision records to make object descriptor removal an accepted decision.

### Why
- The public TypeScript contract is `string | MessageNamespace`.
- Object descriptors are redundant and spoofable, and were only present because the old implementation exported schema values into maps.

### What worked
- The design now states that `schemaInput`, `mustResolveSchemaSpec`, and the `map[string]any` branch should be removed entirely.
- The test plan now includes descriptor rejection tests instead of descriptor acceptance tests.

### What didn't work
- N/A

### What I learned
- Keeping string support is useful because it exercises `resolvePrototype`, `Options.Prototypes`, and protobuf global type lookup without requiring generated JS namespace modules.
- Keeping object descriptor support is unnecessary for unpublished code and would turn an implementation artifact into public API.

### What was tricky to build
- The wording had to distinguish primitive string support from object descriptor fallback. Both contain type names, but only primitive strings are part of the tightened API.

### What warrants a second pair of eyes
- Confirm that no downstream unpublished examples rely on `{ type: ... }`. If they do, update them to a string full name or generated namespace.

### What should be done in the future
- Implement the stricter resolver and descriptor rejection tests.

### Code review instructions
- Review the design guide sections: Executive summary, Gap 3, Proposed design, Decision records, Phase 1/2 implementation guide, and Test strategy.

### Technical details
- Updated design doc: `ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md`.

## Step 6: Implement strict bulk schema registration

I implemented the stricter schema registration path described by the updated design. Bulk `ss.schemas({...})` now walks the original Goja input object, reads lower-case schema sections directly, and resolves each schema leaf while it is still a `goja.Value`. This preserves generated protobuf namespace hidden prototype tokens and removes the old `ExportTo`/`map[string]any` schema path entirely.

The resulting API supports exactly generated `MessageNamespace` values and primitive protobuf full-name strings. Plain object descriptors such as `{ type: "..." }` and `{ typeName: "..." }` now fail immediately with the same schema type error as any other unsupported object.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Implement the tightened API and keep the diary current while validating each task.

**Inferred user intent:** Move from design to code with a clean API cutover and strong regression coverage.

**Commit (code):** 298d0496b4a2661d44ca80f74d5538fe81e28e51 — "Fix bulk schema registration for Goja protobuf namespaces"

### What I did
- Added regression tests in `pkg/js/modules/sessionstream/module_test.go`:
  - `TestSchemasBulkRegisterGeneratedPrototypeNamespaces`
  - `TestSchemasBulkRegisterStringFullNames`
  - `TestSchemasRejectPlainObjectDescriptors`
- Verified the new tests fail against the old implementation before changing production code.
- Refactored `pkg/js/modules/sessionstream/api_schemas.go`:
  - removed `schemaInput`;
  - removed `mustResolveSchemaSpec(spec any)`;
  - removed the `map[string]any` branch;
  - removed `goja.Runtime.ExportTo` from `schemasBuilder`;
  - added `registerSchemaSection` over original `goja.Value` objects;
  - tightened `mustResolveSchemaValue` to accept generated namespaces or primitive strings only.
- Ran `gofmt`.

### Why
- `ExportTo` stripped hidden generated-protobuf prototype tokens and also failed to populate lower-case TypeScript section keys in the reproduction.
- Removing the exported-map path makes bulk registration behave like fluent registration.

### What worked
- The focused tests now pass:

```bash
go test ./pkg/js/modules/sessionstream -run 'TestSchemas(BulkRegisterGeneratedPrototypeNamespaces|BulkRegisterStringFullNames|RejectPlainObjectDescriptors)' -count=1
# ok github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream 0.005s
```

- Package tests now pass:

```bash
go test ./pkg/js/modules/sessionstream -count=1
# ok github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream 0.072s
```

- Full repository tests now pass:

```bash
go test ./... -count=1
# ok github.com/go-go-golems/sessionstream/... (all packages with tests passed)
```

- The pre-commit hook also passed `GOWORK=off go test ./...` and `golangci-lint run -v` before creating commit `298d0496b4a2661d44ca80f74d5538fe81e28e51`.

### What didn't work
- The new tests failed as expected before implementation:

```text
--- FAIL: TestSchemasBulkRegisterGeneratedPrototypeNamespaces
Error: GoError: unknown command "ChatStartInference"

--- FAIL: TestSchemasBulkRegisterStringFullNames
Error: GoError: unknown command "ChatStartInference"

--- FAIL: TestSchemasRejectPlainObjectDescriptors/typeName
Error: An error is expected but got nil.

--- FAIL: TestSchemasRejectPlainObjectDescriptors/type
Error: An error is expected but got nil.
```

### What I learned
- Lower-case bulk sections must be read from the original JavaScript object. Relying on Go struct export tags here is both unnecessary and fragile.
- Checking `value.Export().(string)` is stricter than `value.String()` and avoids accidentally treating arbitrary objects as strings.

### What was tricky to build
- Promise-returning hub operations can hide asynchronous handler/projection failures if a test only checks `RunString` errors. The bulk generated namespace test now checks that the returned Promise is fulfilled, which exercises command, event, UI event, and timeline entity schemas.
- Descriptor rejection needed to call `ss.schemas(...)` directly after the refactor because schema values are resolved during registration rather than later at submit time.

### What warrants a second pair of eyes
- Review `registerSchemaSection` for Goja object semantics, especially whether `obj.Keys()` is the right key set for this API.
- Review the hard cutover from `{ type: "..." }` to primitive string full names.

### What should be done in the future
- Consider a small go-go-goja documentation update warning consumers not to call `ExportTo` before `protogoja.MessagePrototypeFromValue` / `MessageFromValue`.

### Code review instructions
- Start with `pkg/js/modules/sessionstream/api_schemas.go` and confirm `ExportTo` is gone from schema registration.
- Then review `pkg/js/modules/sessionstream/module_test.go` for the three new regression tests.
- Validate with:

```bash
go test ./pkg/js/modules/sessionstream -count=1
go test ./... -count=1
```

### Technical details
- Modified code file: `/home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_schemas.go`.
- Modified test file: `/home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module_test.go`.

## Step 7: Refresh delivered PDF after implementation updates

After committing the implementation and diary updates, I regenerated the ticket PDF bundle and re-uploaded it to reMarkable so the delivered copy matches the final design, tasks, changelog, and diary.

### Prompt Context

**User prompt (verbatim):** (same as Step 5)

**Assistant interpretation:** Keep ticket documentation and external delivery synchronized after implementation.

**Inferred user intent:** Ensure the ticket remains the durable source of truth for future review.

**Commit (code):** N/A — rendered documentation refresh only.

### What I did
- Regenerated the PDF bundle with `remarquee upload bundle --pdf-only`.
- Re-uploaded the bundle with `remarquee upload bundle --force`.
- Verified the remote listing under `/ai/2026/06/15/SS-GOJA-SCHEMA-MAP-001`.

### Why
- The design and diary changed after the initial upload, so the reMarkable copy needed to be refreshed.

### What worked
- The PDF regenerated successfully.
- The reMarkable upload succeeded and the remote listing shows the bundle.

### What didn't work
- N/A

### What I learned
- Implementation updates can make previously uploaded ticket bundles stale; refresh them when the ticket document is part of the deliverable.

### What was tricky to build
- N/A

### What warrants a second pair of eyes
- N/A

### What should be done in the future
- N/A

### Code review instructions
- Check the rendered PDF under `sources/` only if reviewing the reMarkable artifact.

### Technical details
- Remote path: `/ai/2026/06/15/SS-GOJA-SCHEMA-MAP-001`.
- Bundle name: `SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide`.
