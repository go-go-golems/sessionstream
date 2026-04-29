# Changelog

## 2026-04-29

- Initial workspace created.
- Added the primary `Sessionstream code review and architecture audit` design document with architecture map, API analysis, concrete findings, cleanup sketches, and validation guidance.
- Added the `Investigation diary` reference document with commands, evidence collection notes, and review continuation instructions.
- Updated tasks to reflect completed review work and open follow-up decisions.

## 2026-04-29

Completed evidence-backed architecture audit and intern-oriented code review deliverables.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/01-sessionstream-code-review-and-architecture-audit.md — Primary architecture audit
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md — Chronological investigation diary


## 2026-04-29

Validated SESSIONSTREAM-003 and uploaded the document bundle to reMarkable at /ai/2026/04/29/SESSIONSTREAM-003.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md — Upload validation and troubleshooting diary step


## 2026-04-29

Uploaded a final refreshed reMarkable bundle after appending the upload diary step.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md — Final upload diary state included in refreshed PDF


## 2026-04-29

Added remediation plan covering replay store, ordinal seeding, split projection policies, error/DLQ reporting, evtstream cleanup, websocket fanout-only scope, schema cloning, protobuf chat example, and systemlab refactor direction.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/02-remediation-plan-for-replay-store-and-api-cleanup.md — Remediation design document


## 2026-04-29

Expanded phased remediation tasks and completed the first implementation slice: sessionstream naming cleanup, fanout-only websocket scope, cursor-seeded local ordinals, fail-default split projection policies, error observer hook, and defensive schema cloning.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go — Projection policy
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/schema.go — Schema prototype cloning changes
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/tasks.md — Phased implementation task list


## 2026-04-29

Uploaded refreshed SESSIONSTREAM-003 remediation bundle to reMarkable at /ai/2026/04/29/SESSIONSTREAM-003 as 'SESSIONSTREAM-003 Remediation Plan and Audit'.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/design-doc/02-remediation-plan-for-replay-store-and-api-cleanup.md — Remediation plan included in uploaded bundle


## 2026-04-29

Added initial SQLite replay-store primitives: event log, event cursor/query API, entity versions with Snapshot(asOf), in-memory SQLite constructor, durable errors table, and hub integration for event append/error persistence.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go — Event append
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration.go — EventStore and ErrorStore interfaces
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store.go — SQLite replay-store primitives
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store_test.go — Replay-store tests


## 2026-04-29

Uploaded refreshed v2 remediation bundle after adding SQLite replay-store primitives.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/ttmp/2026/04/29/SESSIONSTREAM-003--code-review-and-architecture-audit-for-sessionstream/reference/01-investigation-diary.md — Step 6 included in refreshed v2 upload


## 2026-04-29

Added projection cursor storage and initial timeline rebuild support for replayed events.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go — Projection cursor advancement and RebuildTimeline
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration.go — ProjectionCursorStore interface
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store.go — SQLite projection cursor table and methods


## 2026-04-29

Added RetryTimeline, RebuildTimelineFromScratch, TimelineResetStore, decode/ordinal error reporting, and SQLite ClearTimeline support.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/consumer.go — Decode and ordinal error reporting
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hub.go — Retry and scratch rebuild helpers
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store.go — ClearTimeline support


## 2026-04-29

Persisted raw error payloads and metadata in SQLite error records and added an ErrorRecords query API.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration.go — ErrorRecordStore interface
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store.go — SQLite raw error payload persistence and query API
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/sqlite/store_test.go — Error record round-trip test


## 2026-04-29

Removed the map-backed memory hydration store and ported local users/tests/systemlab phases to in-memory SQLite.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/lab_environment.go — Phase 1 now uses in-memory SQLite
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/phase5_lab.go — Memory mode now uses in-memory SQLite
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat_test.go — Chat demo tests now use in-memory SQLite
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/pkg/sessionstream/hydration/memory — Deleted map-backed store package


## 2026-04-29

Added a self-contained generated protobuf chat example and migrated chatdemo payloads from structpb to generated command/event/UI/entity messages.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat.go — Chat demo migrated to generated protobuf payloads
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/chat_test.go — Tests assert generated timeline entity payloads
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/examples/chatdemo/proto/sessionstream/examples/chatdemo/v1/chat.proto — Chat demo protobuf schema


## 2026-04-29

Added read-only Phase 5 systemlab replay inspection for event cursor, timeline cursor, and persisted errors.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/lab_environment_test.go — Replay inspection test
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/replay_api.go — Read-only replay inspection endpoint
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/server.go — Routes replay inspection endpoint


## 2026-04-29

Added Phase 5 systemlab UI panel for read-only replay cursors and persisted errors.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/js/api.js — Replay inspection API client
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/js/pages/phase5.js — Replay inspection frontend refresh
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/static/partials/phase5.html — Replay inspection panel


## 2026-04-29

Extracted initial shared systemlab trace helpers and documented Phase 5 replay cursor inspection.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/chapters/phase-5-persistence-and-restart.md — Replay cursor explanation
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/phase5_lab.go — Uses shared trace helper
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/trace_helpers.go — Shared trace append/clone helpers


## 2026-04-29

Step 15: Extracted shared Systemlab snapshot encoding and websocket hook helpers (commit aaac81d34cadad21820f68b7d335db701c0fc8b8).

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/phase3_lab.go — Uses shared websocket trace hooks
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/phase4_lab.go — Uses shared websocket trace hooks
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/phase5_lab.go — Uses shared websocket trace hooks
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/snapshot_helpers.go — Shared snapshot/protobuf payload encoding
- /home/manuel/workspaces/2026-04-07/extract-webchat/sessionstream/cmd/sessionstream-systemlab/ws_hooks.go — Shared websocket trace hook builder

