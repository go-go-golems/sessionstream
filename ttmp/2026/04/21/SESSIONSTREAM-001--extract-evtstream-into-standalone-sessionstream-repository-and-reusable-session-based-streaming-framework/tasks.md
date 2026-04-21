# Tasks

## Repo bootstrap and ticket setup

- [x] Correct `sessionstream/.ttmp.yaml` so docmgr uses `sessionstream/ttmp` instead of the template's stale `hair-booking/ttmp` root.
- [x] Initialize repo-local docmgr vocabulary and add the topic slugs needed for sessionstream extraction work.
- [x] Create `SESSIONSTREAM-001` in `sessionstream/ttmp` with index/tasks/changelog, a design doc, and a diary.

## Analysis and design

- [x] Audit the current `pinocchio/pkg/evtstream` package layout and classify what is already extraction-ready versus what is still pinocchio-coupled.
- [x] Audit `pinocchio/cmd/evtstream-systemlab` and confirm whether it should move with the standalone framework.
- [x] Audit the canonical `pinocchio/cmd/web-chat` consumer path and document which pieces should stay downstream after extraction.
- [x] Write a detailed intern-facing analysis/design/implementation guide with prose, diagrams, pseudocode, API references, and file references.
- [x] Keep a chronological implementation diary while doing the work.

## Ticket bookkeeping and delivery

- [x] Relate the key source files to the design doc and diary with `docmgr doc relate`.
- [x] Update the changelog with the major ticket-creation and design-documentation milestones.
- [x] Run `docmgr doctor --ticket SESSIONSTREAM-001 --stale-after 30` and resolve any warnings.
- [x] Upload the final bundle to reMarkable under `/ai/2026/04/21/SESSIONSTREAM-001`.

## Future implementation follow-ups

### Phase 0 — bootstrap the sessionstream repository

- [x] Replace the template `sessionstream/go.mod` module path with `github.com/go-go-golems/sessionstream`.
- [x] Replace the placeholder/template repository docs and agent guidance (`README.md`, `AGENT.md`, package doc stubs) with sessionstream-specific content.
- [x] Clean the template placeholders out of repo metadata and dev tooling (`Makefile`, release config, placeholder command stubs, CI references as needed).
- [x] Validate the bootstrapped repo with focused checks (`go test ./...`, any repo-local lint/build checks that still apply).

### Phase 1 — extract the pure substrate

- [x] Move the root `evtstream` substrate files into `sessionstream` with import-path rewrites.
- [x] Move `hydration/memory` into `sessionstream`.
- [x] Move `hydration/sqlite` into `sessionstream`.
- [x] Move `transport` and `transport/ws` into `sessionstream`.
- [x] Add a boundary check ensuring the moved substrate no longer imports `github.com/go-go-golems/pinocchio/...`.
- [x] Validate the moved substrate in `sessionstream` with `go test ./...`.

### Phase 2 — move framework-oriented examples and Systemlab

- [x] Add a small generic demo/example chat app to `sessionstream` if a framework-owned application example is still desired.
- [x] Move the framework-oriented Systemlab pieces into the new repo.
- [x] Decide whether the current Phase 6 pinocchio migration lab should stay downstream instead of moving with the framework-oriented Systemlab phases.
- [x] Validate the moved example/Systemlab code in `sessionstream`.

### Phase 3 — keep the real chat app downstream in pinocchio

- [x] Keep the real chat app in `pinocchio` and rebase it on `sessionstream` instead of extracting `pkg/evtstream/apps/chat` wholesale.
- [x] Move `agentmode` ownership out of the shared chat package and into `cmd/web-chat` or another pinocchio-owned adapter layer.
- [x] Update the downstream chat app so it publishes the needed sessionstream-compatible events without making the framework repo depend on `agentmode`.

### Phase 4 — switch pinocchio to consume sessionstream

- [x] Switch `pinocchio` to consume the external `sessionstream` module.
- [x] Update downstream imports/tests in `pinocchio`.
- [x] Retire the old in-tree `pkg/evtstream` copy after stabilization.
- [x] Re-run focused `cmd/web-chat` backend/frontend validation after the consumer cutover.
