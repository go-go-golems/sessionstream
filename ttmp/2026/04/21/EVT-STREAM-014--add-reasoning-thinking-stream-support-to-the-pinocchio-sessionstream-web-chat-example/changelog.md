# Changelog

## 2026-04-21

- Initial workspace created.
- Investigated the live canonical `cmd/web-chat` path and confirmed that Geppetto already emits reasoning/thinking events while the current sessionstream-backed app only projects normal chat deltas plus `agentmode`.
- Wrote the main intern-facing analysis / design / implementation guide for reasoning/thinking support.
- Added the ticket diary with the investigation steps and the correction from `pkg/mod` lookup to the workspace `geppetto` checkout.
- Validated the ticket with `docmgr doctor` and uploaded the bundle to reMarkable.
- Implemented the backend reasoning slice in `pinocchio`: added `cmd/web-chat/reasoning_chat_feature.go`, registered it in `main.go`, translated `EventThinkingPartial` plus `thinking-started` / `thinking-ended` / `reasoning-summary` info events into canonical `ChatReasoning*` events, projected them into durable `ChatMessage` entities with `role: thinking`, and added focused Go tests covering direct feature behavior plus app-level snapshot/websocket flows (code commit `2c7676b2910ebba0baa17626e31073b74af5f502`).
- Decided that the first shipped slice will expose visible reasoning text only and defer final `reasoning_tokens` UI work to a follow-up slice unless browser validation reveals a stronger immediate need.
- Implemented the frontend/live-state slice in `pinocchio`: extended `cmd/web-chat/web/src/ws/wsManager.ts` to consume `ChatReasoning*` UI events, added focused frontend tests, and then fixed the placeholder timing bug by delaying assistant/thinking entity creation until actual visible bytes arrive instead of on `ChatMessageStarted` / empty finish events (code commits `f1def467790b0f2c27f7d6615b4bf42d1c8c2105` and `8edf6d8461c97b917503d37e84aa8cd7496be442`).
- Browser-validated the live canonical app against a real reasoning-capable profile (`gpt-5-mini`): the UI rendered separate `user`, `thinking`, and `assistant` messages, the persisted snapshot included `chat-msg-1:thinking`, and the validation artifact was saved as `reference/02-real-browser-reasoning-session-snapshot.json`.
- Tightened streaming UX by changing `cmd/web-chat/web/src/webchat/ChatWidget.tsx` so the timeline auto-follows incremental streaming updates instead of lagging behind during long responses (code commit `61b8b9a06cb2051e0d6684f6d5d70fa800da267c`).
- Refined the auto-follow behavior after additional user feedback that it was still reacting more to whole-message changes than text growth: `ChatWidget.tsx` now also watches live DOM text mutations in the timeline subtree and re-pins the scroll container to the bottom while streaming (code commit `8a51780bae74b5b9f30c1f4b0a2379763d0d88e8`).
- Added manual scroll-intent handling to the chat UI: `ChatWidget.tsx` now uses a small follow/detached state machine with bottom-distance thresholds, ignores programmatic scroll events, pauses auto-follow when the user scrolls away from the tail, and exposes a sticky "Jump to latest" control that reattaches follow mode (code commit `69075a32b17b9f0a1ac6ec181524c225c6316d4d`).
- Refactored the scroll-follow behavior into a reusable frontend hook at `cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`, so `ChatWidget.tsx` now consumes a packaged sticky-follow state machine instead of carrying the logic inline (code commit `887215fb9c5669058b87806c157f6c552b0de040`).
- Quieted the new sticky-scroll hook under Biome by removing the remaining exhaustive-deps warnings: reset-key and content-version change detection now use explicit previous-value refs, and the mutation-observer effect no longer carries an unnecessary `contentVersion` dependency (code commit `2d3c25a53668bf38f461d10b01d1210d97407c3a`).

## 2026-04-21

Created EVT-STREAM-014, documented the live canonical reasoning/thinking gap in cmd/web-chat, wrote a detailed intern-facing design/implementation guide plus diary, and prepared the ticket for validation and reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md — Main analysis and implementation guide
- /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/reference/01-diary.md — Chronological investigation record

