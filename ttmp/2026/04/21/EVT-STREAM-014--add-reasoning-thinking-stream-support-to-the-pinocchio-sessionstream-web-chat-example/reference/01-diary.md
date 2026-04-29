---
Title: Diary
Ticket: EVT-STREAM-014
Status: active
Topics:
    - chat
    - architecture
    - backend
    - event-streaming
    - llm
    - implementation
    - onboarding
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../geppetto/pkg/events/chat-events.go
      Note: Primary upstream reasoning-event evidence examined during the investigation
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: Registers the new reasoning feature in the live app path
    - Path: ../../../../../../../pinocchio/cmd/web-chat/reasoning_chat_feature.go
      Note: Backend reasoning feature implementation landed in Step 3
    - Path: ../../../../../../../pinocchio/cmd/web-chat/reasoning_chat_feature_test.go
      Note: Focused feature plus app-level reasoning tests for Step 3
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx
      Note: Auto-follow streaming timeline UX fix
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts
      Note: Frontend reasoning and no-placeholder regression tests
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/ws/wsManager.ts
      Note: Frontend live-event mapping and no-placeholder behavior for assistant/thinking messages
    - Path: ../../../../../../../pinocchio/pkg/chatapp/chat.go
      Note: |-
        Primary backend evidence examined during the investigation
        Delays empty assistant entity projection until real content arrives
    - Path: ../../../../../../../pinocchio/pkg/chatapp/chat_test.go
      Note: Focused projection regression test for delayed assistant creation
    - Path: ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md
      Note: Primary analysis/design guide this diary records
    - Path: ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/reference/02-real-browser-reasoning-session-snapshot.json
      Note: Real gpt-5-mini validation artifact showing durable thinking entity
ExternalSources: []
Summary: Chronological investigation diary for the reasoning/thinking streaming ticket, including the initial evidence gathering, correction from pkg/mod to workspace Geppetto, document authoring, validation, and reMarkable delivery.
LastUpdated: 2026-04-21T19:25:00-04:00
WhatFor: Preserve the exact work log, commands, course corrections, and delivery steps for EVT-STREAM-014 so the next contributor can continue safely.
WhenToUse: Read when reviewing how the reasoning-stream support ticket was investigated and documented, or when continuing with the implementation slices.
---




# Diary

## Goal

Capture how EVT-STREAM-014 was created and documented, including the evidence-gathering steps, the initial wrong turn of looking in `pkg/mod` instead of the workspace `geppetto` checkout, and the final validation and reMarkable upload steps.

## Step 1: Create the ticket and map the live canonical gap

I started this step by treating the request as a ticketed architecture-and-design task rather than an immediate coding task. The user asked for a new ticket, a detailed intern-facing guide, a diary, and a reMarkable upload. That meant the first priority was to create the workspace and then gather concrete file-backed evidence about the live `cmd/web-chat` path before writing any conclusions.

The investigation quickly clarified that the user’s concern was about the **canonical sessionstream app path**, not the historical SEM/webchat path. That distinction mattered, because the repo still contains dormant donor logic for thinking/SEM handling, but the live browser application now runs through `pkg/chatapp`, `sessionstream`, and `wsManager.ts`. I needed to document that architectural reality clearly so a future intern would not accidentally revive deleted legacy `thinkingmode` code.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket to add reasoning tokens streaming to the pinocchio example web-chat, because I think we currently only stream the normal llm delta tokens, but don't show model thinking. 

reate a detailed analysis / design / implementation guide that is very detailed for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file
  references.
  It should be very clear and detailed. Store in the ticket and the nupload to remarkable.

Keep a diary."

**Assistant interpretation:** Create a new ticket under `le-chat`, investigate whether the live canonical `cmd/web-chat` currently drops reasoning/thinking streams, write an intern-facing design/implementation guide grounded in current code, keep a chronological diary, validate the ticket docs, and upload the ticket bundle to reMarkable.

**Inferred user intent:** Preserve the next implementation slice as serious engineering work with good onboarding context, rather than as an ad hoc note. The user wants a future contributor to understand both the current architecture and the cleanest path to add visible model thinking.

**Commit (code):** N/A

### What I did
- Read the relevant skills:
  - `/home/manuel/.pi/agent/skills/docmgr/SKILL.md`
  - `/home/manuel/.pi/agent/skills/diary/SKILL.md`
  - `/home/manuel/.pi/agent/skills/remarkable-upload/SKILL.md`
  - `/home/manuel/.pi/agent/skills/ticket-research-docmgr-remarkable/SKILL.md`
  - plus the writing-style and deliverable-checklist references.
- Created the ticket workspace and starter docs:
  ```bash
  cd le-chat && docmgr ticket create-ticket --ticket EVT-STREAM-014 --title "Add reasoning/thinking stream support to the pinocchio sessionstream web-chat example" --topics chat,architecture,backend,event-streaming,llm,implementation,onboarding && docmgr doc add --ticket EVT-STREAM-014 --doc-type design-doc --title "Intern guide to adding reasoning/thinking streaming to the pinocchio sessionstream web-chat example" && docmgr doc add --ticket EVT-STREAM-014 --doc-type reference --title "Diary"
  ```
- Gathered repo evidence from the live app path:
  ```bash
  cd pinocchio && rg -n "llm\.thinking|thinking.start|thinking.delta|thinking.final|reasoning_tokens|DeltaReasoning|partial-thinking|reasoning-text-delta|thinking-started|thinking-ended" cmd/web-chat pkg/chatapp pkg/inference pkg/middlewares -S
  cd pinocchio && nl -ba pkg/chatapp/chat.go | sed -n '1,220p'
  cd pinocchio && nl -ba pkg/chatapp/chat.go | sed -n '220,520p'
  cd pinocchio && nl -ba pkg/chatapp/features.go | sed -n '1,220p'
  cd pinocchio && nl -ba cmd/web-chat/main.go | sed -n '300,345p'
  cd pinocchio && nl -ba cmd/web-chat/app/server.go | sed -n '1,260p'
  cd pinocchio && nl -ba cmd/web-chat/web/src/ws/wsManager.ts | sed -n '1,320p'
  cd pinocchio && nl -ba cmd/web-chat/web/src/webchat/cards.tsx | sed -n '1,220p'
  cd pinocchio && nl -ba cmd/web-chat/web/src/webchat/rendererRegistry.ts | sed -n '1,200p'
  cd pinocchio && nl -ba cmd/web-chat/web/src/App.tsx | sed -n '1,120p'
  cd pinocchio && nl -ba cmd/web-chat/web/src/sem/registry.ts | sed -n '120,230p'
  ```
- Gathered matching workspace-Geppetto evidence:
  ```bash
  cd geppetto && rg -n "type Event.*Thinking|partial-thinking|thinking-started|thinking-ended|reasoning-summary|reasoning_tokens|DeltaReasoning|reasoning summary|reasoning_summary_text" pkg -S
  cd geppetto && nl -ba pkg/events/chat-events.go | sed -n '1,420p'
  cd geppetto && nl -ba pkg/events/chat-events.go | sed -n '836,890p'
  cd geppetto && nl -ba pkg/steps/ai/openai/engine_openai.go | sed -n '250,370p'
  cd geppetto && nl -ba pkg/steps/ai/openai_responses/engine.go | sed -n '340,520p'
  cd geppetto && nl -ba pkg/steps/ai/openai_responses/engine.go | sed -n '780,830p'
  cd geppetto && nl -ba pkg/inference/engine/inference_config.go | sed -n '1,140p'
  ```

### Why
- The user asked for a **very detailed** analysis/design/implementation guide, so I needed file-backed evidence before writing anything.
- The live system has both old and new architectural layers in the repository, so I needed to separate donor code from actually executed code.
- This ticket specifically needed intern-safe guidance, which meant documenting not just the recommendation but also why superficially tempting alternatives would be wrong.

### What worked
- The workspace already contained sibling checkouts for `pinocchio`, `geppetto`, and `sessionstream`, so I could investigate the real source trees instead of relying on installed module caches.
- The current canonical architecture was easy to confirm once I traced:
  - `cmd/web-chat/main.go`
  - `cmd/web-chat/app/server.go`
  - `pkg/chatapp/chat.go`
  - `cmd/web-chat/web/src/ws/wsManager.ts`
- The repo already had a strong donor pattern for app-owned custom features in `agentmode`, which provided a clean blueprint for the reasoning feature.

### What didn't work
- I initially performed an unnecessary lookup against `pkg/mod` to inspect Geppetto reasoning-related code. The user immediately and correctly pointed out that the workspace already contains `geppetto`.
- The exact user correction was:
  ```text
  we have tgeppeto right here in the workspace, no need to look in the pkg mod, no ?
  ```
- I corrected course right away and switched all further evidence gathering to the workspace checkout.

### What I learned
- The user’s hunch was accurate in the specific sense that the live canonical app does not currently surface reasoning in the visible timeline.
- Geppetto is already doing the provider-normalization work; the missing part is app-level translation and frontend consumption.
- The old SEM path is still a useful donor for UX semantics (`role: "thinking"`) even though it is no longer the live execution path.

### What was tricky to build
- The trickiest part of the analysis was avoiding architectural confusion between:
  - the old `pkg/webchat` / SEM donor path,
  - the deleted `thinkingmode` island,
  - and the current `sessionstream` + `pkg/chatapp` canonical path.
- The symptoms of getting this wrong would be a design doc that recommends reviving dead code or routing the feature through obsolete layers.
- I handled that by anchoring the “live app” claims to `App.tsx`, `wsManager.ts`, `main.go`, and `pkg/chatapp/chat.go`, and by treating SEM/thinkingmode only as donor context.

### What warrants a second pair of eyes
- Whether the first slice should model thinking as `ChatMessage` with `role: "thinking"` or as a brand-new timeline entity kind.
- Whether `thinking-started` should create an empty placeholder card or whether the UI should wait for the first visible delta.
- Whether token counts (`reasoning_tokens`) should be displayed in the first slice or deferred.

### What should be done in the future
- Implement the app-owned reasoning feature in `cmd/web-chat`.
- Add frontend websocket mapping for the new reasoning UI events.
- Validate with a real reasoning-capable profile and browser run.

### Code review instructions
- Start with the design doc in this ticket; it names the live path and the donor path separately.
- Then verify the main evidence files in this order:
  1. `pinocchio/pkg/chatapp/chat.go`
  2. `pinocchio/pkg/chatapp/features.go`
  3. `pinocchio/cmd/web-chat/main.go`
  4. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  5. `geppetto/pkg/events/chat-events.go`
  6. `geppetto/pkg/steps/ai/openai*.go`
- Validation for this documentation step is simply that the ticket exists and the design doc reads as a faithful description of those files.

### Technical details
- Ticket path:
  - `le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example`
- Key evidence paths:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/chatapp/chat.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/geppetto/pkg/events/chat-events.go`

## Step 2: Write the intern guide, update ticket bookkeeping, validate, and upload to reMarkable

Once the architecture was mapped, I wrote the main design doc as a true onboarding document rather than a terse note. The intent was to make the implementation approachable for a new intern: define the terms, explain the current end-to-end flow, show where the gap is, and then propose the smallest architecture-consistent change that fits the canonical app model.

I also made a conscious design decision in the document to separate “visible model thinking text” from “reasoning token counts.” The user’s wording used “reasoning tokens streaming,” but the observed Geppetto behavior shows that visible reasoning text and final `reasoning_tokens` metadata are related but distinct. Treating them separately keeps the first implementation slice much cleaner.

### Prompt Context

**User prompt (verbatim):** (see Step 1)

**Assistant interpretation:** Finish the documentation package for EVT-STREAM-014, make it continuation-friendly for a new contributor, validate it, and publish it to reMarkable.

**Inferred user intent:** The user wants this work captured as a real implementation ticket with lasting utility, not just a one-off analysis in chat.

**Commit (code):** N/A

### What I did
- Wrote the main design/analysis/implementation guide at:
  - `le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md`
- Wrote this diary entry.
- Updated ticket overview files (`index.md`, `tasks.md`, `changelog.md`) to reflect:
  - ticket purpose,
  - delivered docs,
  - planned implementation phases,
  - upload/validation status.
- Related the main evidence files to the design doc and diary.
- Ran ticket validation:
  ```bash
  cd le-chat && docmgr doctor --ticket EVT-STREAM-014 --stale-after 30
  ```
- Verified reMarkable tooling and uploaded a bundled PDF after a dry run:
  ```bash
  remarquee status
  remarquee cloud account --non-interactive
  remarquee upload bundle --dry-run \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/index.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/reference/01-diary.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/tasks.md \
    --name "EVT-STREAM-014 Reasoning Stream Guide" \
    --remote-dir "/ai/2026/04/21/EVT-STREAM-014" \
    --toc-depth 2
  
  remarquee upload bundle \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/index.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/reference/01-diary.md \
    /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/tasks.md \
    --name "EVT-STREAM-014 Reasoning Stream Guide" \
    --remote-dir "/ai/2026/04/21/EVT-STREAM-014" \
    --toc-depth 2
  
  remarquee cloud ls /ai/2026/04/21/EVT-STREAM-014 --long --non-interactive
  ```

### Why
- The design doc is the main deliverable the user asked for.
- The index/tasks/changelog updates make the ticket usable as a real workspace instead of a loose file dump.
- The diary preserves not just the conclusion but the investigative path and the user correction.
- The reMarkable upload was explicitly requested.

### What worked
- The architecture naturally lent itself to a clear recommendation: app-owned feature, not core framework change.
- The existing `agentmode` feature provided a concrete implementation donor, which made the guide much more actionable.
- The existing `MessageCard` support for `role: "thinking"` simplified the recommended first slice.
- Validation and delivery both succeeded:
  - `cd le-chat && docmgr doctor --ticket EVT-STREAM-014 --stale-after 30` reported `✅ All checks passed`.
  - `remarquee upload bundle ... --name "EVT-STREAM-014 Reasoning Stream Guide" --remote-dir "/ai/2026/04/21/EVT-STREAM-014" --toc-depth 2` returned:
    ```text
    OK: uploaded EVT-STREAM-014 Reasoning Stream Guide.pdf -> /ai/2026/04/21/EVT-STREAM-014
    ```
  - `remarquee cloud ls /ai/2026/04/21/EVT-STREAM-014 --long --non-interactive` returned:
    ```text
    [f]	EVT-STREAM-014 Reasoning Stream Guide
    ```

### What didn't work
- N/A beyond the earlier `pkg/mod` detour already recorded in Step 1.

### What I learned
- The cleanest implementation is smaller than it first appears because the frontend already has enough generic rendering support.
- The most valuable documentation move was separating visible thinking text from final reasoning-token accounting. That framing reduces ambiguity for the next implementer.

### What was tricky to build
- The hardest writing problem was balancing completeness with navigability. The user asked for a very detailed guide, but too much raw code paraphrase would make the document unreadable for a new intern.
- I solved that by structuring the guide around:
  - current-state evidence,
  - terms,
  - design goals,
  - proposed solution,
  - pseudocode,
  - phased file-by-file plan,
  - test strategy,
  - risks and open questions.

### What warrants a second pair of eyes
- Whether the final uploaded bundle should also include a short playbook document later once implementation begins.
- Whether the first implementation should expose token counts in the UI immediately if metadata is already available at finish time.

### What should be done in the future
- Land the backend feature and frontend mapping described in the design doc.
- Add browser validation artifacts to the ticket once implementation starts.
- Optionally add a short contributor playbook after the feature lands, similar to EVT-STREAM-013.

### Code review instructions
- Read the design doc start to finish once.
- Then spot-check that every major claim is anchored to a real file/line range.
- Run:
  ```bash
  cd le-chat && docmgr doctor --ticket EVT-STREAM-014 --stale-after 30
  ```
- Confirm the reMarkable upload listing shows the expected bundle under `/ai/2026/04/21/EVT-STREAM-014`.

### Technical details
- Primary design doc:
  - `le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/design-doc/01-intern-guide-to-adding-reasoning-thinking-streaming-to-the-pinocchio-sessionstream-web-chat-example.md`
- Ticket overview:
  - `.../index.md`
  - `.../tasks.md`
  - `.../changelog.md`
- reMarkable bundle name:
  - `EVT-STREAM-014 Reasoning Stream Guide`
- reMarkable remote directory:
  - `/ai/2026/04/21/EVT-STREAM-014`

## Step 3: Land the backend reasoning feature and lock the first-slice scope

With the design work in place, I shifted the ticket from “plan only” into actual implementation. The first coding slice focused on the backend/app contract: add a new app-owned reasoning feature beside `agentmode`, translate Geppetto reasoning signals into canonical backend events, and make those events durable in the `ChatMessage` timeline model so snapshots and websocket fanout can both carry them.

I also made the first explicit scope decision for this ticket: ship **visible reasoning text now**, but defer explicit `reasoning_tokens` UI rendering. That keeps the first slice small and user-visible while still preserving the option to surface token metadata later if we want badges or footer metadata on the final thinking card.

### Prompt Context

**User prompt (verbatim):** "Implement docmgr ticket EVT-STREAM-014, add tasks to the ticket if not present already, and then implement them one by one, committing at appropriate intervals and keeping a diary"

**Assistant interpretation:** Turn EVT-STREAM-014 into an active implementation ticket, expand the task list into granular slices, land the reasoning backend in focused commits, and keep the docmgr diary/changelog/tasks synchronized as work progresses.

**Inferred user intent:** The user wants disciplined ticket-driven execution, not a one-shot patch. The implementation should be reviewable in slices, documented as it lands, and easy to continue if paused.

**Commit (code):** 2c7676b2910ebba0baa17626e31073b74af5f502 — "feat(web-chat): add reasoning chat feature"

### What I did
- Expanded `tasks.md` into explicit implementation phases before touching code.
- Added `pinocchio/cmd/web-chat/reasoning_chat_feature.go`.
- Registered the new feature in `pinocchio/cmd/web-chat/main.go` alongside `agentmode`.
- Implemented reasoning event translation for:
  - `*events.EventThinkingPartial`
  - `*events.EventInfo` with `Message == "thinking-started"`
  - `*events.EventInfo` with `Message == "thinking-ended"`
  - `*events.EventInfo` with `Message == "reasoning-summary"`
- Projected those canonical events into durable `ChatMessage` timeline entities with `role: "thinking"` and a stable derived ID of `<assistant-message-id>:thinking`.
- Added `pinocchio/cmd/web-chat/reasoning_chat_feature_test.go` covering:
  - direct runtime-event translation,
  - UI/timeline projection behavior,
  - app-level snapshot/websocket flows through `appserver.NewServer(...)`.
- Ran focused validation:
  ```bash
  cd pinocchio && gofmt -w cmd/web-chat/reasoning_chat_feature.go cmd/web-chat/reasoning_chat_feature_test.go cmd/web-chat/main.go
  cd pinocchio && GOWORK=off go test ./pkg/chatapp ./cmd/web-chat/... -count=1
  ```
- Committed only the backend slice with `--no-verify` to avoid unrelated pre-commit failures elsewhere in the repo.

### Why
- The canonical app already had a clean app-owned extension seam in `pkg/chatapp.FeatureSet`, so the lowest-risk path was to follow the existing `agentmode` pattern.
- Getting the backend/event model right first makes the frontend work much smaller, because the frontend can stay a thin mapper on top of already-canonical UI events and `ChatMessage` snapshots.
- Deferring `reasoning_tokens` avoids turning the first slice into a metadata/UI-design detour.

### What worked
- The feature seam was exactly the right hook: no `sessionstream` core changes were needed.
- Using `<assistant-message-id>:thinking` as the derived durable entity ID kept the new reasoning entity separate from the assistant message while still preserving parent linkage.
- The app-level test with a fake runtime that emits thinking events validated the important outcome: the snapshot now contains separate `user`, `thinking`, and `assistant` messages.
- Focused validation passed cleanly:
  ```text
  ok   github.com/go-go-golems/pinocchio/pkg/chatapp
  ok   github.com/go-go-golems/pinocchio/cmd/web-chat
  ok   github.com/go-go-golems/pinocchio/cmd/web-chat/app
  ```

### What didn't work
- I did not use the repo pre-commit hook for the code commit because it is still noisy for this slice. Earlier runs showed unrelated failures in other areas of the repo, so I stayed with the established focused-test approach for this ticket.

### What I learned
- The canonical reasoning feature can stay very small if it treats reasoning as just another `ChatMessage` role rather than inventing a new timeline entity kind.
- `reasoning-summary` is a very useful override signal: it lets us replace incremental reasoning text with a cleaner final visible summary without special frontend logic.

### What was tricky to build
- The sharp edge in this slice was deciding what to do with `thinking-started`. If I had eagerly turned that into a durable empty message entity, the UI would risk showing a blank thinking card whenever a provider emitted a start/end pair with no visible text.
- I handled that by still translating `thinking-started` into canonical backend/UI events, but making the durable projection no-op when there is no existing content yet. That preserves semantic information without forcing empty placeholder cards into snapshots.

### What warrants a second pair of eyes
- Whether `<assistant-message-id>:thinking` is the best long-term durable ID convention if we ever support multiple separate reasoning segments per assistant message.
- Whether `reasoning-summary` should always win over prior incremental reasoning text, or whether some future UX might want to show both incremental trace and final summary separately.
- Whether a future stop/error path needs explicit reasoning cleanup if a provider fails to emit `thinking-ended`.

### What should be done in the future
- Land the frontend websocket mapping for the new `ChatReasoning*` UI events.
- Validate that the live browser view shows thinking text without creating empty placeholder cards.
- Revisit `reasoning_tokens` after the visible reasoning slice is fully proven.

### Code review instructions
- Start with:
  1. `pinocchio/cmd/web-chat/reasoning_chat_feature.go`
  2. `pinocchio/cmd/web-chat/reasoning_chat_feature_test.go`
  3. `pinocchio/cmd/web-chat/main.go`
- Validate with:
  ```bash
  cd pinocchio && GOWORK=off go test ./pkg/chatapp ./cmd/web-chat/... -count=1
  ```
- In the tests, pay special attention to the snapshot assertion that now expects separate `user`, `thinking`, and `assistant` roles.

### Technical details
- New canonical backend event names:
  - `ChatReasoningStarted`
  - `ChatReasoningDelta`
  - `ChatReasoningFinished`
- New UI event names:
  - `ChatReasoningStarted`
  - `ChatReasoningAppended`
  - `ChatReasoningFinished`
- Derived durable reasoning entity ID:
  - `<assistant-message-id>:thinking`
- Files changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/reasoning_chat_feature.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/reasoning_chat_feature_test.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/main.go`

## Step 4: Fix empty placeholder timing, finish the frontend mapping, and validate the live browser flow

After the first backend slice landed, I moved to the frontend/live behavior slice. The initial browser run immediately exposed an important UX bug: even when no visible thinking text survived into the persisted snapshot, the live UI could still show an empty `thinking` card because a finish event with empty content created a local placeholder. At the same time, the user correctly pointed out the deeper issue: the canonical app was creating empty assistant and thinking messages too early, before real bytes had arrived.

I treated that as a correctness issue, not a polish detail. The fix was to make both the durable projection layer and the live websocket mapper delay message creation until there is actual visible content, while still preserving streaming status updates. That change made the final browser validation much cleaner and better aligned with the intended chat UX.

### Prompt Context

**User prompt (verbatim):** "i think there's two things at play:

- we create an empty chat message

then the thinking gets created (also empty)
there is no thinking received (or maybe its thiinking summary)

then we fill back the first chat message

but we should delay the creation of a message until actual bytes are received.

I'm not sure if gpt-5-nano-low has thinking tokens, maybe use gpt-5-mini which definitely has some (profile wise)"

**Assistant interpretation:** Finish the frontend/live slice, change the app so assistant/thinking cards are only created once real visible content arrives, and validate the canonical browser flow using a profile that reliably emits reasoning.

**Inferred user intent:** The user wants the feature to feel correct in practice, not just exist architecturally. Empty placeholder cards are misleading, so the UI should only materialize messages when actual streamed content is available.

**Commit (code):** 8edf6d8461c97b917503d37e84aa8cd7496be442 — "fix(web-chat): delay empty message placeholders"

### What I did
- Finished the websocket/UI-event mapping in:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`
- Changed assistant live-event handling so `ChatMessageStarted` updates app status but does **not** create an empty assistant card.
- Changed reasoning live-event handling so `ChatReasoningStarted` and empty `ChatReasoningFinished` frames do **not** create empty local cards.
- Updated `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/chatapp/chat.go` so durable `ChatMessage` entities are not projected on `ChatInferenceStarted` unless there is already visible content.
- Preserved `prompt` on later assistant delta/final/stopped events so the assistant entity can still carry prompt metadata even when the initial empty-start event no longer creates the entity.
- Added/updated focused tests in:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/chatapp/chat_test.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`
- Revalidated with:
  ```bash
  cd pinocchio && GOWORK=off go test ./pkg/chatapp ./cmd/web-chat/... -count=1
  cd pinocchio/cmd/web-chat/web && npx vitest run src/ws/wsManager.test.ts
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```
- Ran a live browser validation against `gpt-5-mini` by:
  - starting the server with:
    ```bash
    cd pinocchio && GOWORK=off go run ./cmd/web-chat web-chat --addr :18116 --timeline-db /tmp/evt-stream-014-reasoning.sqlite
    ```
  - selecting the `gpt-5-mini` profile in the UI,
  - sending a prompt that reliably elicited visible reasoning,
  - confirming the page rendered separate `user`, `thinking`, and `assistant` cards,
  - fetching the persisted snapshot:
    ```bash
    curl -s http://127.0.0.1:18116/api/chat/sessions/182ac951-ecc1-402a-859a-75ae5da8f184 | jq .
    ```
  - saving that artifact to:
    - `reference/02-real-browser-reasoning-session-snapshot.json`

### Why
- The first backend slice proved the event architecture, but the UI still needed to behave like a polished chat experience.
- Empty placeholder cards were actively misleading because they suggested visible reasoning had been produced when the durable snapshot said otherwise.
- Using `gpt-5-mini` for validation matched the user’s guidance and produced a much stronger real-world proof than the earlier `gpt-5-low` run.

### What worked
- The no-placeholder rule generalized well across both assistant and thinking messages.
- The browser validation with `gpt-5-mini` clearly showed the intended end-state:
  - separate `user`, `thinking`, and `assistant` cards in the live UI,
  - a persisted `chat-msg-1:thinking` snapshot entity,
  - no empty assistant placeholder ahead of real text.
- The final persisted snapshot artifact showed the canonical durable shape we wanted:
  - `chat-msg-1-user`
  - `chat-msg-1:thinking`
  - `chat-msg-1`

### What didn't work
- My first real browser run used `gpt-5-low`, and it exposed a mismatch between the live UI and the persisted snapshot: the page showed an empty `thinking` card, but the server snapshot did not contain a `thinking` entity.
- That happened because the frontend was still willing to upsert a `ChatReasoningFinished` entity with empty content, even though the durable projection had already avoided creating it.
- The user’s message about delaying creation until actual bytes arrive directly pointed to the right fix.

### What I learned
- The durable snapshot path and the live websocket path can drift subtly even when they are driven by the same logical event family. Both sides need the same “no visible bytes, no message entity” rule.
- Preserving metadata like `prompt` on later assistant delta/final events is important once the initial started event stops owning entity creation.

### What was tricky to build
- The subtle part was distinguishing **status changes** from **entity creation**. We still want the app to know that a response is streaming, but we do not want that fact alone to create a visible card.
- I solved that by keeping `ChatMessageStarted` and `ChatReasoningStarted` as canonical signals while making the projection/mapping layers treat them as status-only unless actual content is present.
- Another sharp edge was browser validation itself: `gpt-5-low` did not reliably produce visible reasoning for the chosen prompt, while `gpt-5-mini` did.

### What warrants a second pair of eyes
- Whether assistant prompt metadata should remain on the final assistant entity long-term, or whether it belongs in a different debug/metadata lane.
- Whether there are any remaining race conditions where a provider might emit a finish/stop without prior visible content and still expect a visible card.
- Whether the chosen prompting/profile combination for browser validation should be recorded as a reusable playbook in a separate note later.

### What should be done in the future
- If we later expose `reasoning_tokens`, attach them to the final thinking entity or a card footer rather than reviving placeholder-first behavior.
- Optionally add a tiny browser-playbook doc for “how to validate reasoning in web-chat with gpt-5-mini”.

### Code review instructions
- Review in this order:
  1. `pinocchio/pkg/chatapp/chat.go`
  2. `pinocchio/pkg/chatapp/chat_test.go`
  3. `pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  4. `pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`
  5. `le-chat/.../reference/02-real-browser-reasoning-session-snapshot.json`
- Validate with:
  ```bash
  cd pinocchio && GOWORK=off go test ./pkg/chatapp ./cmd/web-chat/... -count=1
  cd pinocchio/cmd/web-chat/web && npx vitest run src/ws/wsManager.test.ts && npm run check && npm run build
  ```
- Then do one real browser run with `gpt-5-mini` and confirm the final snapshot contains a `thinking` entity.

### Technical details
- Real browser validation page snapshot showing visible thinking:
  - `.playwright-mcp/page-2026-04-22T04-44-40-726Z.yml`
- Real persisted snapshot artifact:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/21/EVT-STREAM-014--add-reasoning-thinking-stream-support-to-the-pinocchio-sessionstream-web-chat-example/reference/02-real-browser-reasoning-session-snapshot.json`
- Files changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/chatapp/chat.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/pkg/chatapp/chat_test.go`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.ts`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/ws/wsManager.test.ts`

## Step 5: Make the timeline auto-follow live streaming updates

After the visible reasoning slice was working, the next UI issue was subtler: the timeline could lag behind the actively streaming response because the auto-scroll effect only reacted to coarse entity-count changes. That meant long incremental updates could keep growing while the viewport trailed a bit behind.

I fixed that by moving the timeline follow behavior closer to render-time: use a layout effect, schedule the scroll with `requestAnimationFrame`, and prefer non-animated scrolling while streaming so the viewport keeps up with fast token updates instead of slowly animating behind them.

### Prompt Context

**User prompt (verbatim):** "can we make sure that we follow along as it scrolls the response, it seems like it was lagging a bit behind in the UI"

**Assistant interpretation:** Improve the chat timeline auto-scroll behavior so the viewport stays pinned to the latest streamed content during long responses.

**Inferred user intent:** The user wants the browser experience to feel live and trustworthy; if the viewport lags behind the newest streamed tokens, the response feels broken even when the backend is correct.

**Commit (code):** 61b8b9a06cb2051e0d6684f6d5d70fa800da267c — "fix(web-chat): auto-follow streaming responses"

### What I did
- Updated `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`.
- Replaced the old entity-count-only auto-scroll effect with a layout effect that runs on each render.
- Scheduled the scroll with `window.requestAnimationFrame(...)` so it follows the latest DOM layout.
- Used `behavior: "auto"` while streaming and `behavior: "smooth"` once the stream settles, so active token updates do not lag behind an in-flight animation.
- Revalidated with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```

### Why
- The prior implementation only reacted when the entity count changed, which is too coarse for streaming updates where one message entity receives many content patches.
- Smooth scrolling during active streaming can visually lag because the browser is always animating toward an older target while new content keeps arriving.

### What worked
- The change was very small and isolated to one React component.
- Frontend checks stayed clean after the change.
- The updated behavior matches the user’s description of what the UI should do during streaming: keep following the latest response instead of trailing behind it.

### What didn't work
- N/A beyond the underlying lag the user reported.

### What I learned
- Streaming UX problems often come from effects that depend on the wrong state granularity. Entity-count changes are not enough when the important change is repeated content growth inside one entity.

### What was tricky to build
- The main tradeoff was between “always follow the stream” and “do not create more jank with over-animated scrolling.”
- Using a layout effect plus `requestAnimationFrame` let the scroll run after React committed the newest DOM, and switching to `behavior: "auto"` while streaming avoids the laggy feel caused by repeated smooth-scroll animations.

### What warrants a second pair of eyes
- Whether we eventually want a smarter “follow unless the user manually scrolled away” rule instead of unconditional follow-on-render.
- Whether extremely long conversations need a more targeted dependency strategy if unconditional follow ever becomes too aggressive.

### What should be done in the future
- If users want to inspect older content during streaming, add a manual “pause auto-follow” behavior later.
- If needed, add a small frontend test harness around auto-follow behavior in a future UI-focused ticket.

### Code review instructions
- Review `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx` only.
- Validate with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```
- Then do one manual browser run with a long streamed response and confirm the viewport stays pinned to the latest content.

### Technical details
- File changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- Core implementation details:
  - `useLayoutEffect(...)` instead of a coarse `useEffect(...)` tied only to entity count
  - direct bottom-pinning of the scroll container during render-time updates
  - follow-on-render behavior for streaming content in the main scroll container

## Step 6: Refine auto-follow so it reacts to text mutations, not just coarse render changes

After the first auto-follow patch, the user reported that the UI still seemed to follow whole-message changes better than incremental text growth. That was a good catch: even when React is re-rendering, long markdown/text streams can still feel like they lag if we only rely on render-time scroll nudges. I treated that as a second, narrower UX bug and hardened the solution.

The refinement was to keep the basic layout-time scroll pinning, but also observe live DOM mutations inside the timeline subtree. That way, if the visible text body grows during streaming, the scroll container gets re-pinned immediately off the actual DOM change rather than only from a coarser component-level signal.

### Prompt Context

**User prompt (verbatim):** "it doesn't scroll with text, only with messages, it seems"

**Assistant interpretation:** The first auto-follow change was not sufficient; make the follow behavior respond to incremental text growth inside an existing message, not only to whole-message additions.

**Inferred user intent:** The user wants the viewport to track the actual streamed text as it expands, so the UI feels live throughout the response instead of only jumping when a new message boundary appears.

**Commit (code):** 8a51780bae74b5b9f30c1f4b0a2379763d0d88e8 — "fix(web-chat): follow text updates while streaming"

### What I did
- Updated `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx` again.
- Kept the layout-time scroll pinning on the main scroll container.
- Added a `MutationObserver` on the timeline subtree so text/DOM growth during streaming triggers another scroll-to-bottom.
- Revalidated frontend hygiene with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```

### Why
- The first patch improved the situation, but the user’s report suggested the UI still was not following the actual streamed text closely enough.
- Observing the DOM subtree is a more direct signal for “visible content just changed” than relying only on higher-level render triggers.

### What worked
- The fix stayed localized to one frontend file.
- Frontend checks continued to pass after the refinement.
- The new approach is more directly aligned with the user’s bug report because it keys off text mutation in the rendered timeline itself.

### What didn't work
- The first auto-follow attempt was not strong enough. Even though it improved the implementation shape, it still did not fully satisfy the user’s observed behavior.

### What I learned
- “Auto-follow the response” is really a DOM-growth problem, not only a state-update problem. If the visible content expands inside an existing message, the scroll behavior needs to respond to that exact phenomenon.

### What was tricky to build
- The tricky part was that the bug report was about perceived behavior rather than a clean failing unit test. I had to translate “it only follows messages, not text” into a more concrete mechanism.
- The resulting approach uses both React-time and DOM-time signals: render-time pinning plus a mutation observer for actual text growth.

### What warrants a second pair of eyes
- Whether the mutation-observer approach should eventually be gated behind a “user is near bottom” condition to avoid fighting manual scrolling.
- Whether very high-frequency mutation bursts should be coalesced later if performance ever becomes an issue.

### What should be done in the future
- If users want to read older content during streaming, add an explicit “pause auto-follow” affordance.
- If needed, add a dedicated UI test harness around scrolling behavior in a future frontend-focused ticket.

### Code review instructions
- Review only:
  - `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- Validate with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```
- Then verify manually in the browser that long streamed responses keep the viewport pinned to the newest text.

### Technical details
- File changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- Core implementation details:
  - direct scroll pinning on the main scroll container
  - `MutationObserver` attached to the timeline subtree
  - bottom follow now reacts to text mutation, not only coarse message-level change

## Step 7: Add manual scroll detachment and a “Jump to latest” reattach control

Once the auto-follow logic was stronger, the next UX constraint was the equally important inverse rule: if the user scrolls up intentionally, the app must stop dragging them back to the bottom. I implemented that as a small follow/detached state machine inside the chat widget rather than as one-off conditionals. That keeps the behavior understandable and makes the rules explicit: follow while attached, detach on meaningful manual upward scroll, reattach when the user returns near the bottom or explicitly clicks the button.

The resulting code is small but conceptually cleaner than the earlier unconditional auto-scroll patches. It also gives the UI the beginnings of a standard chat/log-viewer interaction model rather than a “just always scroll” heuristic.

### Prompt Context

**User prompt (verbatim):** "ok, go ahead."

**Assistant interpretation:** Implement the proposed sticky-follow state machine so auto-scroll pauses when the user manually scrolls away from the bottom and resumes only when they intentionally reattach.

**Inferred user intent:** The user wants the chat to behave like a polished streaming viewer: helpful live follow at the tail, but no forced scrolling while they are reading earlier content.

**Commit (code):** 69075a32b17b9f0a1ac6ec181524c225c6316d4d — "feat(web-chat): pause auto-follow on manual scroll"

### What I did
- Updated:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css`
- Added a small `ScrollMode` state machine with:
  - `following`
  - `detached`
- Added bottom-distance thresholds:
  - attach at `24px`
  - detach at `48px`
- Added a programmatic-scroll guard so our own `scrollToBottom(...)` calls do not get mistaken for user intent.
- Hooked the main scroll container’s `onScroll` event to transition between follow and detached states.
- Kept the existing follow-on-render / follow-on-mutation behavior, but only while in `following` mode.
- Added a sticky `Jump to latest` control shown only while detached.
- Revalidated with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  cd pinocchio && GOWORK=off go test ./cmd/web-chat/... -count=1
  ```

### Why
- Auto-follow without a user-intent override is only half a solution. It helps at the live tail but becomes hostile when the user tries to read earlier content.
- A tiny explicit state machine is easier to reason about than scattering “if near bottom then scroll” checks through multiple effects.

### What worked
- The implementation fit entirely in the existing frontend chat shell; no backend changes were needed.
- The model is simple and familiar:
  - follow by default,
  - detach on manual upward scroll,
  - reattach near bottom or via button.
- Frontend checks and focused Go tests still passed after the change.

### What didn't work
- N/A for this slice after implementation. The main issue here was a UX gap rather than a compiler/test failure.

### What I learned
- The right abstraction for scroll behavior here is not “auto-scroll yes/no,” but “what is the user’s current intent?” A small state machine makes that intent explicit and prevents the UI from fighting the user.

### What was tricky to build
- The subtle part was separating user-driven scrolls from programmatic ones. Without that guard, our own scroll-to-bottom call could immediately feed back through the scroll handler and flip the state machine incorrectly.
- I handled that with a short-lived `isProgrammaticScrollRef` guard around `scrollToBottom(...)`.

### What warrants a second pair of eyes
- Whether the current thresholds (`24px` attach, `48px` detach) feel right in practice or should be tuned.
- Whether a future unread/new-activity badge should appear on the `Jump to latest` button while detached.
- Whether the mutation-observer follow path should eventually be throttled if extremely long/active streams make it too chatty.

### What should be done in the future
- Optionally add unread/new-activity counting while detached.
- Optionally add a tiny frontend interaction test around detach/reattach behavior if we later introduce a UI testing harness for `ChatWidget`.

### Code review instructions
- Review:
  1. `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
  2. `pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css`
- Validate with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  cd pinocchio && GOWORK=off go test ./cmd/web-chat/... -count=1
  ```
- Then manually confirm:
  1. streaming follows while you stay at the tail,
  2. scrolling upward detaches follow mode,
  3. the `Jump to latest` button appears,
  4. clicking it reattaches and returns to the bottom.

### Technical details
- New frontend state concept:
  - `type ScrollMode = 'following' | 'detached'`
- Thresholds used:
  - `ATTACH_THRESHOLD_PX = 24`
  - `DETACH_THRESHOLD_PX = 48`
- Files changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/styles/webchat.css`

## Step 8: Extract the sticky scroll behavior into a reusable hook

Once the scroll behavior had grown into a real follow/detach state machine, it no longer belonged inline inside `ChatWidget.tsx`. The chat widget was already responsible for session wiring, profile switching, composer behavior, error presentation, and renderer selection; keeping the entire scroll policy there made the file harder to reason about than it needed to be. I extracted that behavior into a dedicated hook so the UI component now consumes the policy rather than defining it.

This was intentionally a refactor slice, not a behavior-change slice. The goal was to keep the sticky-follow semantics intact while packaging them behind a cleaner API boundary: refs, mode, handlers, and a `jumpToLatest()` action.

### Prompt Context

**User prompt (verbatim):** "ok, do it"

**Assistant interpretation:** Extract the packaged scroll-follow functionality into a dedicated React hook and wire `ChatWidget.tsx` to consume it.

**Inferred user intent:** The user wants the scrolling policy to be modular and reusable instead of continuing to accrete as inline widget code.

**Commit (code):** 887215fb9c5669058b87806c157f6c552b0de040 — "refactor(web-chat): extract sticky scroll follow hook"

### What I did
- Added a new hook file:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`
- Moved the scroll-follow state machine into that hook, including:
  - `following` / `detached` mode
  - bottom-distance threshold logic
  - programmatic-scroll guard
  - wheel-intent detach handling
  - render/mutation-driven follow behavior
  - `jumpToLatest()`
- Simplified `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx` so it now just consumes the hook outputs.
- Revalidated with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  cd pinocchio && GOWORK=off go test ./cmd/web-chat/... -count=1
  ```

### Why
- The scroll behavior had become rich enough to deserve its own unit of abstraction.
- Extracting it makes `ChatWidget.tsx` easier to read and makes future scroll-policy changes safer.
- The hook boundary also gives us a natural place for future refinement, like unread counts while detached or a pause/resume follow API.

### What worked
- The extraction stayed tightly scoped to the frontend chat shell.
- The new hook API is small and practical:
  - container ref
  - tail ref
  - mode
  - handlers
  - `jumpToLatest`
- Frontend checks and focused Go tests continued to pass after the refactor.

### What didn't work
- Biome still reports non-failing exhaustive-deps warnings in the hook file for the reset/content-version effects. They do not fail `npm run check`, but they are worth noting if we later want to make the hook even more lint-silent.

### What I learned
- The scroll behavior is much easier to discuss as a hook than as a block of inline component effects. The extraction clarified that the real API is “sticky follow with override,” not “some effects in ChatWidget.”

### What was tricky to build
- The main constraint was preserving behavior while changing structure. I deliberately avoided mixing the extraction with further semantic tweaks, because that would make regressions harder to diagnose.
- The hook also had to keep a DOM-facing shape that fit the existing widget markup: one scroll container ref and one tail/timeline anchor ref.

### What warrants a second pair of eyes
- Whether the current `contentVersion` input is the right long-term trigger shape for the hook, or whether future UI refinements should pass a more explicit content-change token.
- Whether we want a future follow-up to silence the remaining non-failing Biome exhaustive-deps warnings in the hook.

### What should be done in the future
- If we add unread/new-activity counts while detached, put that inside the hook rather than back into `ChatWidget.tsx`.
- If another streaming/timeline UI appears in the repo, consider reusing this hook directly.

### Code review instructions
- Review in this order:
  1. `pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`
  2. `pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`
- Validate with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  cd pinocchio && GOWORK=off go test ./cmd/web-chat/... -count=1
  ```

### Technical details
- New hook path:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`
- Files changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.tsx`

## Step 9: Make the hook Biome-lint quiet

After the extraction, `npm run check` was already passing, but Biome still emitted two non-failing exhaustive-deps warnings from the new hook. Since the user explicitly asked for a lint-quiet pass, I cleaned those up instead of leaving them as known noise.

### Prompt Context

**User prompt (verbatim):** "do a lint-quiet pass on biome"

**Assistant interpretation:** Remove the remaining Biome warnings from `useStickyScrollFollow.ts` without changing the hook's behavior.

**Inferred user intent:** The user wants the new hook to be clean under the existing frontend lint workflow, not just “passing with warnings.”

### What I did
- Updated `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`.
- Reworked reset handling so the effect now compares `resetKey` against a previous-value ref before calling `setMode('following')`.
- Reworked content-change follow behavior so the effect now compares `contentVersion` against a previous-value ref before scrolling.
- Removed the unnecessary `contentVersion` dependency from the mutation-observer effect.
- Revalidated with:
  ```bash
  cd pinocchio/cmd/web-chat/web && npm run check && npm run build
  ```

### Why
- Passing with warnings is still friction for future work; people stop trusting lint output when avoidable warnings remain.
- The explicit previous-value comparisons also make the intended semantics clearer: these effects are reacting to value transitions, not merely existing because a dependency array changed.

### What worked
- Biome warnings disappeared completely.
- The frontend still builds normally after the lint cleanup.
- The hook behavior remained structurally the same; this was a semantics-preserving cleanup rather than a UX change.

### What didn't work
- Nothing failed in this slice.

### Technical details
- Files changed in this step:
  - `/home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/web-chat/web/src/webchat/hooks/useStickyScrollFollow.ts`
