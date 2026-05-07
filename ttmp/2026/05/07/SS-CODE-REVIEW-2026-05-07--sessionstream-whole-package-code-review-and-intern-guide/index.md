---
Title: Sessionstream whole-package code review and intern guide
Ticket: SS-CODE-REVIEW-2026-05-07
Status: active
Topics:
    - sessionstream
    - code-review
    - cleanup
    - architecture
    - onboarding
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Whole-package Sessionstream code review, recent-change audit, intern guide, cleanup plan, diary, and reMarkable delivery record.
LastUpdated: 2026-05-07T17:02:00-04:00
WhatFor: Use this ticket to orient reviewers and interns before changing Sessionstream core, SQLite replay storage, websocket transport, diagnostics, examples, or Systemlab.
WhenToUse: Read when planning cleanup work from the 2026-05-07 whole-package review.
---

# Sessionstream whole-package code review and intern guide

## Overview

This ticket contains a whole-package review of `sessionstream` with special attention to the recent observer, replay-store, schema-vet, Systemlab cleanup, and websocket reconnect-race work recorded in recent diaries.

The deliverable is intentionally both a code review and an intern guide. It explains the system model, public APIs, runtime flows, package layout, current architecture, code-quality findings, stale/deprecated-code review, and phased cleanup plan.

## Key documents

- [Whole-package code review and intern implementation guide](design-doc/01-whole-package-code-review-and-intern-implementation-guide.md)
- [Investigation diary](reference/01-investigation-diary.md)
- [Tasks](tasks.md)
- [Changelog](changelog.md)

## Evidence artifacts

- `scripts/01-inventory.sh` — reproducible inventory script.
- `sources/01-inventory-output.txt` — package list, file inventory, line counts, marker search, recent commits.
- `sources/02-key-files-lines-1.txt` — line-numbered key file excerpts.
- `sources/03-issue-snippets.txt` — line-numbered snippets used in findings.
- `sources/04-validation-output.txt` — passing tests, websocket race test, and lint output.
- `sources/05-coverage-output.txt` — coverage result after Go cache cleanup.

## Top findings

1. Late websocket hydration-buffer batches should be filtered by `ordinal > snapshotOrdinal` just like the main drain path.
2. Websocket fanout delivery errors are currently hidden from Hub-level fanout error records.
3. SQLite migration currently drops all store tables for older `user_version` values and needs additive migration tests.
4. SQLite `AppendEvent` silently overwrites session/ordinal conflicts and should distinguish idempotent retry from conflicting event data.
5. `sqlite.NewInMemory` uses a fixed shared in-memory DSN name and should be isolated by default.
6. `ErrorObserver` should match the panic-safety of pipeline and transport observers.
7. `hub.go`, `transport/ws/server.go`, `hydration/sqlite/store.go`, and `examples/chatdemo/chat.go` should be split after correctness fixes.
8. Observer APIs are useful but fragmented; documentation should explain Bus vs Pipeline vs Transport observations and the role of websocket `Hooks`.

## Validation status

Recorded in `sources/04-validation-output.txt` and `sources/05-coverage-output.txt`:

- `go test ./...` passed.
- `go test ./pkg/sessionstream/transport/ws -race -count=1` passed.
- `make lint` passed with 0 issues.
- Coverage succeeded after `go clean -cache`; total statement coverage is about 66.9%.

## reMarkable delivery

The final bundle was uploaded to:

```text
/ai/2026/05/07/SS-CODE-REVIEW-2026-05-07
```

Bundle names:

```text
SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review
SS-CODE-REVIEW-2026-05-07 Sessionstream Code Review Final
```

The `Final` bundle was uploaded after the delivery diary step was added, without force-overwriting the first upload. See the diary and final chat response for the verified listing.

## Status

Current status: **active**. The review deliverable is complete; follow-up implementation tasks remain open for a later coding ticket or continuation.

## Topics

- sessionstream
- code-review
- cleanup
- architecture
- onboarding

## Structure

- `design-doc/` — Primary architecture/review/implementation guide.
- `reference/` — Chronological investigation diary.
- `scripts/` — Reproducible investigation commands.
- `sources/` — Captured evidence and validation outputs.
- `tasks.md` — Completed review work and recommended follow-up implementation tasks.
- `changelog.md` — Ticket changes.
