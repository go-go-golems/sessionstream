---
Title: Final cutover recommendation and remaining legacy removal plan
Ticket: EVT-STREAM-010
Status: active
Topics:
    - architecture
    - backend
    - implementation
    - migration
    - webchat
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../pinocchio/cmd/web-chat/app/server.go
      Note: Canonical session/create/submit/websocket server path used for the acceptance recommendation
    - Path: ../../../../../../../pinocchio/cmd/web-chat/main.go
      Note: Live cmd/web-chat route wiring now serves only the canonical evtstream-backed app path
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/debug-ui/api/debugApi.ts
      Note: Legacy thinking-mode flattening references were removed during final cmd/web-chat cleanup
    - Path: ../../../../../../../pinocchio/cmd/web-chat/web/src/webchat/ChatWidget.stories.tsx
      Note: Storybook references to the deleted thinking-mode feature were removed during final cmd/web-chat cleanup
ExternalSources: []
Summary: Final recommendation for the Phase 6 cmd/web-chat cutover, including current readiness assessment, explicit acceptance call, and a concrete plan for removing or deferring remaining pkg/webchat-era artifacts.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: Give reviewers one final note that says whether the canonical evtstream cutover should be accepted, what evidence supports that call, and which remaining legacy files should be deleted, retained temporarily, or deferred.
WhenToUse: Use when deciding whether cmd/web-chat should remain on the canonical evtstream path, whether legacy route families should stay gone, and what cleanup remains for pkg/webchat-era artifacts.
---



# Final cutover recommendation and remaining legacy removal plan

This note is the final recommendation for Phase 6.

The short version is:

> **Accept the cutover. Keep the canonical `evtstream` path as the only live `cmd/web-chat` product path. Do not reintroduce the legacy `/chat`, `/ws?conv_id=...`, or `/api/timeline` route family.**

The migration has now crossed the threshold from “architecturally plausible” to “operationally acceptable.” The canonical path is live, browser-validated, provider-validated, transcript-backed, and inspected through the Systemlab Phase 6 console. The remaining work is no longer “finish the cutover.” The remaining work is historical-ticket hygiene: the live app is already cut over, and the old `pinocchio/pkg/webchat` package has now been deleted entirely.

## Recommendation

### Acceptance call

**Recommendation:** accept the current canonical `evtstream` implementation as the product path for `cmd/web-chat`.

### Rationale

The current canonical path now satisfies the requirements that actually matter for Phase 6:

1. **The live app runs on the canonical evtstream-backed server path**
   - `cmd/web-chat/main.go` mounts:
     - `POST /api/chat/sessions`
     - `GET|POST /api/chat/sessions/:sessionId...`
     - `GET /api/chat/ws`
   - the legacy `/chat`, `/ws?conv_id=...`, and `/api/timeline` routes are not the live product path anymore.

2. **Real inference works through the canonical path**
   - the canonical app resolves runtime/profile settings at the app edge,
   - the canonical chat engine translates runtime events into canonical evtstream chat events,
   - real provider-backed browser validation succeeded with `gpt-5-nano-low`.

3. **The browser-visible product behavior survives the cutover**
   - users can create a session,
   - submit a prompt,
   - see their own message,
   - see the assistant response,
   - reload and recover state from the session snapshot.

4. **The intentional divergences are documented rather than accidental**
   - route family change,
   - session-based identity,
   - explicit websocket subscribe flow,
   - canonical snapshot contract replacing legacy timeline hydration.

5. **The cutover is evidenced, not just asserted**
   - deterministic legacy-vs-canonical comparison artifacts exist,
   - real provider-backed snapshot evidence exists,
   - the Systemlab Phase 6 console probes the public canonical routes and confirms the expected conditions.

## Evidence summary

The strongest acceptance evidence is already captured in the ticket workspace:

- `reference/04-live-route-cutover-checks.json`
  - confirms canonical routes succeed while legacy `/chat` and `/api/timeline` return `404` in the cutover build.
- `reference/07-legacy-vs-canonical-flow-comparison.md`
  - explains which differences are preserve / change / drop and why.
- `reference/09-real-provider-backed-session-snapshot.json`
  - shows a real provider-backed canonical session with user and assistant entities and a finished non-echo assistant response.
- `reference/10-systemlab-phase-6-run.json`
  - verifies canonical routes, legacy route removal, snapshot presence, and non-echo assistant output through the public Phase 6 inspection console.

Taken together, those artifacts are enough to support a final acceptance decision for the cutover itself.

## Product behavior judgment

### Preserved well enough to accept

These user-visible/product-level behaviors are preserved well enough that the cutover should be accepted:

- prompt submission still works,
- assistant responses still render in the browser,
- the user’s own message now appears authoritatively in the canonical timeline,
- session reload/reconnect behavior works through canonical snapshot hydration,
- profile selection remains available through app-owned profile APIs.

### Intentionally changed and should stay changed

These are not regressions. They are the whole point of the migration and should remain as-is:

- `conv_id` -> `sessionId`
- implicit websocket URL attachment -> explicit subscribe protocol
- legacy timeline hydration endpoint -> canonical session snapshot endpoint
- legacy SEM-ish transport details -> canonical transport frames
- removal of the live `/chat` route family

### Not worth preserving

These legacy details should **not** be restored merely for compatibility nostalgia:

- `/chat/default` as the submit route shape
- `/ws?conv_id=...` as the attach mechanism
- `/api/timeline` as the primary browser hydration path
- legacy SEM envelope assumptions in the normal frontend path

## Current legacy classification inside `cmd/web-chat`

The important question is no longer “is `pkg/webchat` still around somewhere in the repo?” The important question is “what is still left in the `cmd/web-chat` area, and does it affect the live cutover?”

As of the current state, the remaining `pkg/webchat`-era traces inside `cmd/web-chat` have now been removed, and the broader legacy package itself has also been deleted. The earlier categories below are retained mainly as historical bookkeeping so reviewers can see what was cleaned up.

### Category A — live runtime path

**Recommendation:** treat as fully cut over.

The live `cmd/web-chat` product path no longer depends on `pkg/webchat` route ownership. The canonical app server under `cmd/web-chat/app` is the live backend. This is the key acceptance criterion, and it is satisfied.

### Category B — legacy compatibility shim in the `cmd/web-chat` build graph

File:
- `pinocchio/cmd/web-chat/profile_policy.go`

Current status:
- **now removed**,
- its old request-resolution wrapper logic was copied into the historical migration harness test instead,
- the live `cmd/web-chat` main package no longer imports `pkg/webchat/http` through this shim.

**Recommendation:** treat this cleanup as complete.

This was the cleanest immediate deletion target because it kept a `pkg/webchat/http` dependency in the `cmd/web-chat` main package even though the live app no longer used it.

### Category C — test-only historical regression harness

File:
- `pinocchio/cmd/web-chat/migration_comparison_test.go`

Current status:
- **now removed**,
- the historical comparison artifacts remain in the ticket workspace,
- the repo no longer keeps a live `cmd/web-chat` test that imports `pkg/webchat` only to recreate the old world.

**Recommendation:** treat this cleanup as complete.

The saved artifacts are enough historical evidence for the cutover review. Keeping the legacy harness in-tree was no longer worth the continued `pkg/webchat` reference inside `cmd/web-chat`.

### Category D — dead or tutorial-island artifact

Files:
- `pinocchio/cmd/web-chat/thinkingmode/*`
- `pinocchio/cmd/web-chat/web/src/features/thinkingMode/*`

Current status:
- **now removed**,
- the old backend/webchat thinking-mode island and its frontend feature module are gone,
- no `thinkingmode` or `thinking_mode` feature path remains in `cmd/web-chat`.

**Recommendation:** treat this cleanup as complete.

Removing it reduced confusion because the feature was not live, not canonical, and not aligned with the current `agentmode` architecture.

## Removal plan

The recommended cleanup order is intentionally narrow and low-risk.

### Phase R1 — remove the legacy profile-policy shim from `cmd/web-chat`

Target:
- `cmd/web-chat/profile_policy.go`

Status:
- **completed**

What happened:
1. `migration_comparison_test.go` gained a local legacy request-resolver adapter,
2. `profile_policy.go` was deleted,
3. focused `cmd/web-chat` tests were re-run successfully.

### Phase R2 — remove the legacy migration comparison harness from `cmd/web-chat`

Target:
- `cmd/web-chat/migration_comparison_test.go`

Status:
- **completed**

What happened:
1. the historical comparison test was deleted,
2. the ticket artifacts remain as the preserved review evidence,
3. `cmd/web-chat` no longer imports `pkg/webchat` through this harness.

### Phase R3 — remove the old `thinkingmode` island

Target:
- `cmd/web-chat/thinkingmode/*`
- `cmd/web-chat/web/src/features/thinkingMode/*`

Status:
- **completed**

What happened:
1. the unused backend/tutorial island was deleted,
2. the unused frontend feature module and story/test references were removed,
3. the `cmd/web-chat` tree no longer contains `thinkingmode` feature references.

### Phase R4 — broader `pkg/webchat` retirement

Target:
- `pinocchio/pkg/webchat/*`

Status:
- **completed**

What happened:
1. the legacy `pinocchio/pkg/webchat` package was deleted entirely,
2. the remaining live documentation pages under `pinocchio/pkg/doc` that still taught the deleted package API were removed,
3. the surviving tutorial/reference pages that still mentioned `pkg/webchat` were updated to point at the post-sessionstream `cmd/web-chat` / `pkg/chatapp` architecture,
4. focused validation passed on the live `cmd/web-chat`, `pkg/chatapp`, and help/doc surfaces.

This means the old package is no longer merely severed from the live runtime path; it is gone from the repo’s live code and live doc surface.

## Risk assessment

### Risks low enough to accept now

- canonical route family is established,
- real provider-backed flow works,
- user and assistant messages hydrate correctly,
- explicit divergences are documented,
- live regression inspection exists in Systemlab.

### Risks that remain but do not block acceptance

- duplicate EVT-STREAM-010 ticket workspaces still make some `docmgr` ticket-scoped commands ambiguous.

These are all real cleanup items, but none of them justify reopening the route/runtime cutover decision.

## Final recommendation to reviewers

If the question is:

> “Should we keep `cmd/web-chat` on the canonical evtstream path and treat the old webchat route family as removed?”

then the answer is:

> **Yes.**

If the question is:

> “Do we need to restore legacy compatibility routes before calling the migration successful?”

then the answer is:

> **No.**

If the question is:

> “Is there still cleanup left?”

then the answer is:

> **Yes, but it is now legacy-deletion and repo-hygiene work, not cutover work.**

## Concrete next actions

Recommended next actions after accepting this cutover note:

1. keep any further cleanup scoped to historical archive material and duplicate ticket hygiene rather than product-path code.
2. optionally trim non-product historical docs/tests elsewhere in the repo that still talk about the removed legacy path.
