# Changelog

## 2026-04-21

- Created the first repository-local ticket in `sessionstream/ttmp`: `SESSIONSTREAM-001`, covering extraction of `pinocchio/pkg/evtstream` into standalone `sessionstream` plus Systemlab relocation.
- Corrected `sessionstream/.ttmp.yaml` so docmgr now points at `sessionstream/ttmp` instead of the template's stale `hair-booking/ttmp` root.
- Seeded repo-local docmgr vocabulary and added topic slugs for architecture, framework, event-streaming, migration, extraction, onboarding, and systemlab.
- Audited the current `pinocchio/pkg/evtstream`, `cmd/evtstream-systemlab`, and `cmd/web-chat` boundaries to distinguish extraction-ready packages from pinocchio-specific runtime integration.
- Wrote the primary design doc: `design-doc/01-intern-guide-and-extraction-plan-for-moving-evtstream-into-standalone-sessionstream.md`.
- Wrote the chronological diary: `reference/01-diary.md`.
- Related the key source files to the design doc and diary with `docmgr doc relate`.
- Validated the ticket successfully with `docmgr doctor --ticket SESSIONSTREAM-001 --stale-after 30`.
- Uploaded the bundled ticket deliverable to reMarkable as `SESSIONSTREAM-001 Sessionstream Extraction Plan.pdf` under `/ai/2026/04/21/SESSIONSTREAM-001`.
- Revised the extraction recommendation: keep the real chat app in `pinocchio`, move `agentmode` ownership outward to `cmd/web-chat` or another pinocchio-owned adapter layer, treat `sessionstream` as the home of the generic substrate plus a small demo/example chat app, and split framework-oriented Systemlab work from pinocchio-specific migration labs when useful.
- Started Phase 0 implementation: updated `go.mod` to `github.com/go-go-golems/sessionstream`, replaced the template README/agent guidance/doc stub with sessionstream-specific content, and verified the repo still passes `go test ./...`.
- Finished Phase 0 bootstrap cleanup: replaced the placeholder `Makefile`, removed the template `cmd/XXX` stub, removed the unreconciled placeholder release config/workflow, and verified the cleaned repo still passes `go test ./...` and `go build ./...`.
- Completed the first pure-substrate extraction slice: copied the `evtstream` root package, hydration stores, and websocket transport out of `pinocchio` into `sessionstream`, renamed the root package to `sessionstream`, added a boundary check preventing `pinocchio/...` imports, and validated the moved code with `go test ./...` and `make check`.
- Added `examples/chatdemo`, a small framework-owned demo chat app built directly on `sessionstream` without pinocchio runtime or `agentmode` dependencies, and validated it with `go test ./...`.
