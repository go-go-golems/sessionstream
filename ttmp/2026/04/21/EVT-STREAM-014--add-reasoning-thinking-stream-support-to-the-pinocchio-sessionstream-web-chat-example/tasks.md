# Tasks

## Delivered in this ticket setup/documentation slice

- [x] Create ticket workspace `EVT-STREAM-014`
- [x] Investigate the live canonical `cmd/web-chat` reasoning/thinking gap with file-backed evidence
- [x] Write a detailed intern-facing analysis / design / implementation guide
- [x] Create and update the ticket diary
- [x] Relate key code files to the main design doc and diary
- [x] Validate the ticket with `docmgr doctor --ticket EVT-STREAM-014 --stale-after 30`
- [x] Upload the ticket bundle to reMarkable

## Planned implementation work

### Phase 0 — execution setup and scope lock

- [x] Expand this ticket from design-only status into an implementation work log with granular slices and diary steps
- [x] Decide whether the first shipped slice will show only visible reasoning text or also final `reasoning_tokens`

### Phase 1 — backend reasoning feature and canonical event translation

- [x] Add an app-owned `reasoning` / `thinking` chat feature under `pinocchio/cmd/web-chat`
- [x] Register the new feature in `pinocchio/cmd/web-chat/main.go` alongside `agentmode`
- [x] Translate Geppetto `EventThinkingPartial` into canonical backend reasoning events
- [x] Translate thinking-related `EventInfo` events (`thinking-started`, `thinking-ended`, `reasoning-summary`) into canonical backend reasoning events
- [x] Project reasoning events into durable `ChatMessage` timeline entities with `role: thinking`
- [x] Add websocket UI-event projection for the new reasoning event family
- [x] Add/update backend Go tests for reasoning snapshot + live-event behavior

### Phase 2 — frontend websocket consumption and rendering

- [x] Extend `cmd/web-chat/web/src/ws/wsManager.ts` to consume reasoning UI events
- [x] Preserve snapshot rendering for `ChatMessage` entities with `role: thinking`
- [x] Add/update frontend tests for reasoning live-event mapping and no-placeholder behavior
- [x] Keep the timeline auto-following incremental streaming updates instead of lagging behind long responses
- [x] Respect manual user scroll detachment so auto-follow pauses until the user returns near bottom or clicks "Jump to latest"

### Phase 3 — validation and ticket closeout

- [x] Validate focused Go tests for the reasoning slices
- [x] Validate frontend checks/build for the reasoning slices
- [x] Validate real browser behavior with a reasoning-capable profile
- [x] Update the EVT-STREAM-014 diary, tasks, index, and changelog with commit-by-commit implementation evidence
