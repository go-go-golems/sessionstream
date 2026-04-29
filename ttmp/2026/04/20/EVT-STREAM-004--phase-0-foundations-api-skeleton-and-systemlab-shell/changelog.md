# Changelog

## 2026-04-20

- Created ticket EVT-STREAM-004 for Phase 0 Foundations, API Skeleton, and Systemlab Shell.
- Added the primary phase implementation plan in `design-doc/01-phase-0-implementation-plan.md`.
- Included a phase-specific ASCII Systemlab page sketch so implementation and documentation can move in lockstep.
- Seeded detailed task checklists for framework work and Systemlab work.
- Implemented the initial `pkg/evtstream` public API skeleton, separate `cmd/evtstream-systemlab` shell, Makefile validation targets, and a captured Systemlab status artifact (`commit 142d77bc3376e4cd5de946314304764bb093a064`).
- Split the Systemlab frontend shell into modular static assets (`index.html`, `app.css`, `partials/`, and `js/`) so later labs do not accumulate in one monolithic file (`commit ef3165eb85bdb736bd95d37bcfc90cb45059a869`).
