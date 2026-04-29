# Changelog

## 2026-04-20

- Created ticket EVT-STREAM-005 for Phase 1 In-Memory Core and Command-to-Projection Lab.
- Added the primary phase implementation plan in `design-doc/01-phase-1-implementation-plan.md`.
- Included a phase-specific ASCII Systemlab page sketch so implementation and documentation can move in lockstep.
- Seeded detailed task checklists for framework work and Systemlab work.
- Implemented the in-memory submit path, memory hydration store, happy-path/error-path tests, Systemlab Lab 01 export flow, and captured run/transcript artifacts (`commit 142d77bc3376e4cd5de946314304764bb093a064`).
- Split the Phase 1 browser UI into a dedicated HTML partial and per-page JavaScript module so later Systemlab labs can scale without one oversized frontend file (`commit ef3165eb85bdb736bd95d37bcfc90cb45059a869`).
