# Changelog

## 2026-05-06

- Initial workspace created.
- Added detailed WebSocket subscribe race fix guide explaining the snapshot-before-subscribe race, subscribe-first hydration buffering, lock ordering, observer evidence, and test strategy.
- Replaced placeholder tasks with a phase-by-phase implementation checklist for the race fix and regression tests.

## 2026-05-07

- Implemented subscribe-first hydration buffering in commit `53ad2ff`.
- Added deterministic WebSocket regression tests for fanout during snapshot load, duplicate prevention, live-vs-hydrating multi-tab behavior, and buffer overflow.
- Added implementation diary for the race fix.
