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
