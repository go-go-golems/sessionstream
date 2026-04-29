# Changelog

## 2026-04-20

- Created ticket workspace EVT-STREAM-012 for the later extraction of `evtstream` into a standalone module/package.
- Defined the scope as **post-stabilization work**, not immediate implementation work.
- Added a design document focused on:
  - why the current `evtstream` package is a good extraction candidate,
  - why Systemlab should move with `evtstream` but remain a separate app/command,
  - what should move versus what should remain in `pinocchio`,
  - what readiness criteria must be met before starting the extraction.
- Added a task list that separates readiness review, extraction design, later execution, and validation/publication work.
