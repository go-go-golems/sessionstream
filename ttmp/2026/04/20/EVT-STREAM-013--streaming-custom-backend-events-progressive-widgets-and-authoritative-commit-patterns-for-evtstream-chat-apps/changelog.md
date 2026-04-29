# Changelog

## 2026-04-20

- Created dedicated ticket `EVT-STREAM-013` for progressive custom-event previews and authoritative committed custom events in `evtstream` chat applications.
- Added a long-form intern-facing design guide covering the current substrate, canonical chat flow, `agentmode`, progressive parsing helpers, proposed preview-versus-commit architecture, file-by-file implementation plan, and validation strategy.
- Added a diary entry recording the ticket-setup workflow, architecture evidence gathering, and the currently observed duplicate-wrapper build issue in `cmd/web-chat`.
- Validated the ticket with `docmgr doctor` and uploaded the document bundle to reMarkable.
- Updated the design and task plan to adopt Option A cleanup: move sink ownership into `ComposedRuntime`, remove the thin `ResolvedRuntime` wrapper, and make runtime contract cleanup the first implementation phase.
- Implemented Phase 0 cleanup in `pinocchio` (`1ae6834f91f78e818f8413d43087eb814f49a193`): added runtime-owned sink decoration via `WrapSink`, removed canonical `ResolvedRuntime`, updated runtime resolver plumbing, and retired the old canonical wrapper path.
- Implemented progressive preview and committed agentmode event flow in `pinocchio` (`c59e33812b7a3a5b085bf7a2d52d2a322246443c`): the structured extractor now emits preview events, canonical chat translates preview/commit events, committed state projects to an `AgentMode` entity, and the frontend renders preview and committed cards.
- Added focused frontend mapping tests and completed real browser validation with `gpt-5-nano-low` in `pinocchio` (`3bb714743078085fb76759f2bddb1c42b41fa40a`).
- Added an additional extractor test covering incomplete intermediate YAML that later yields a preview snapshot in `pinocchio` (`faacfc6aa10ebaf3fec10ba9a5272e5e19cb1336`).
- Removed the last legacy `ComposedRuntime.Sink` field in `pinocchio` (`ef8ccf8edb35bdb9cc51d214ec8c8f4d028e6a0d`): legacy `pkg/webchat` now assembles its concrete sink through an explicit sink-builder path while runtime composition owns only sink decoration via `WrapSink`.
- Extracted a short contributor playbook in `playbooks/01-contributor-playbook-adding-preview-and-committed-custom-chat-events.md` so future contributors have a concise execution checklist in addition to the long-form guide.

## 2026-04-20

Created the dedicated documentation ticket, gathered evidence across evtstream/chat/agentmode/frontend, wrote the intern guide and diary, and prepared the ticket for validation and reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/evtstream/apps/chat/chat.go — Core canonical chat event path described in the new guide
- /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/middlewares/agentmode/middleware.go — Authoritative mode-switch emission documented for the ticket


## 2026-04-20

Validated EVT-STREAM-013 with docmgr doctor, then uploaded the index/design/diary bundle to reMarkable under /ai/2026/04/20/EVT-STREAM-013.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-013--streaming-custom-backend-events-progressive-widgets-and-authoritative-commit-patterns-for-evtstream-chat-apps/design-doc/01-intern-guide-to-streaming-custom-events-progressive-widgets-and-authoritative-commit-in-evtstream-chat-apps.md — Primary uploaded architecture guide
- /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-013--streaming-custom-backend-events-progressive-widgets-and-authoritative-commit-patterns-for-evtstream-chat-apps/reference/01-diary.md — Uploaded diary and delivery record


## 2026-04-20

Revised the design to adopt Option A cleanup: runtime composition owns sink decoration, `ResolvedRuntime` is removed, and detailed cleanup tasks now precede preview-event implementation.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/runtime_composer.go — Composer should own agentmode middleware and sink behavior together
- /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/inference/runtime/composer.go — ComposedRuntime contract targeted by the cleanup

