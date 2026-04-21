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

## Future implementation follow-ups (not done in this ticket)

- [ ] Replace the template `sessionstream/go.mod` module path with `github.com/go-go-golems/sessionstream` and clean the template placeholders in repo metadata.
- [ ] Move the pure substrate packages (`evtstream` root, hydration, transport) into the new repo with import-path rewrites.
- [ ] Move `cmd/evtstream-systemlab` into the new repo as a companion app.
- [ ] Refactor `apps/chat` so it no longer imports `pinocchio/pkg/inference/runtime` or `pinocchio/pkg/middlewares/agentmode` directly.
- [ ] Switch `pinocchio` to consume the external `sessionstream` module and retire the old in-tree `pkg/evtstream` copy after stabilization.
