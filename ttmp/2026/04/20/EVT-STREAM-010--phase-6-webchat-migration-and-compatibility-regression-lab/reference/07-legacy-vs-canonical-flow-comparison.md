---
Title: Legacy vs Canonical Flow Comparison
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
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/reference/05-legacy-flow-transcript.json
      Note: Historical expectation capture for the legacy `/chat` + `/ws` + `/api/timeline` route family using a deterministic harness.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/reference/06-canonical-flow-transcript.json
      Note: Equivalent deterministic capture for the canonical `/api/chat/...` route family.
    - Path: le-chat/ttmp/2026/04/20/EVT-STREAM-010--phase-6-webchat-migration-and-compatibility-regression-lab/design-doc/04-preserve-change-drop-matrix.md
      Note: Original preserve/change/drop decisions used to classify the observed differences.
ExternalSources: []
Summary: "Comparison of legacy and canonical cmd/web-chat flow captures, tagging concrete route/protocol differences as preserve/change/drop and intentional/follow-up."
LastUpdated: 2026-04-20T16:20:00-04:00
WhatFor: "Give reviewers one place to compare the replaced legacy flow with the canonical evtstream-backed flow using captured evidence rather than memory."
WhenToUse: "When reviewing the Phase 6 cutover, deciding whether an observed difference is intentional, and checking which follow-ups still remain after the live runtime cutover."
---

# Legacy vs Canonical Flow Comparison

## Scope

This note compares two deterministic harness captures:

- `reference/05-legacy-flow-transcript.json`
- `reference/06-canonical-flow-transcript.json`

These are **migration evidence artifacts**, not real-model transcripts. They intentionally use deterministic synthetic outputs so the comparison stays stable and reviewable. The purpose here is to compare route ownership, websocket contract shape, hydration contract shape, and final transcript visibility.

## Executive summary

The migration is now doing what Phase 6 said it should do:

- the **user-visible outcome** is preserved: submit a prompt, observe assistant output, and recover transcript state from a reloadable identifier;
- the **live transport contract** has intentionally changed from legacy webchat mechanics to the canonical session-based evtstream contract;
- the most important remaining work is now **documentation polish + Systemlab regression console**, not proving that the basic cutover is technically viable.

## Side-by-side summary

| Area | Legacy evidence | Canonical evidence | Classification | Intentional? | Notes |
|---|---|---|---|---|---|
| Submit route | `/chat/default` | `/api/chat/sessions/:sessionId/messages` | Change | Yes | Expected hard cut from legacy route family to session-based canonical family |
| Identity key | `conv_id=legacy-conv-1` | `sessionId=9ed83944-...` | Change | Yes | Matches Phase 6 decision to move to session-centered identity |
| Websocket attach model | `GET /ws?conv_id=...` | `GET /api/chat/ws` then client `subscribe` frame | Change | Yes | Explicit subscribe model is the canonical evtstream contract |
| Hydration endpoint | `GET /api/timeline?conv_id=...` | `GET /api/chat/sessions/:sessionId` | Change | Yes | Snapshot is now session-native rather than timeline-native |
| Initial websocket frame | `ws.hello` SEM envelope | `hello` transport frame | Change | Yes | Legacy SEM transport details were not preserved |
| Websocket progression | URL attach, then live SEM stream | `hello` → `snapshot` → `subscribed` → `ui-event*` | Change | Yes | This is the most visible protocol replacement |
| Final transcript visibility | Legacy assistant entity contains `Answer: Explain ordinals in plain language` | Canonical final snapshot contains `Answer: Explain ordinals in plain language` | Preserve | Yes | Core user-visible result survived the cutover |
| Prompt-submit success | `submitStatus: 200` | `createStatus: 200`, `submitStatus: 200` | Preserve | Yes | Happy-path request lifecycle remains healthy |
| Profile API family | Legacy app-owned `/api/chat/profile*` | Same app-owned `/api/chat/profile*` family remains | Preserve | Yes | No evidence of required product change here |
| Old `/chat` + `/api/timeline` live availability | Historically available | Live runtime now returns `404` for these paths | Drop | Yes | Verified separately in `reference/04-live-route-cutover-checks.json` |

## Preserve / change / drop calls

### Preserve

These behaviors are visibly preserved:

1. **Submit prompt and get assistant text back**
   - Legacy artifact assistant entity content: `Answer: Explain ordinals in plain language`
   - Canonical artifact final snapshot payload text: `Answer: Explain ordinals in plain language`
2. **A stable identifier supports transcript recovery**
   - Legacy: `conv_id`
   - Canonical: `sessionId`
   - The name changed, but the product-level behavior survived.
3. **Profile APIs remain app-owned and still fit the frontend shell**
   - The cutover did not require a product-facing profile UX rewrite.

### Change

These differences are intentional and aligned with the ticket plan:

1. **Route family and identity model changed**
   - This is the core architecture move, not a regression.
2. **Hydration moved from timeline-specific to session-specific**
   - The canonical app no longer exposes the old timeline API as the live user path.
3. **Websocket contract became explicit**
   - Query-string attachment was replaced by `subscribe` frames.
4. **Transport frames changed shape**
   - Legacy SEM-ish hello/live behavior was replaced by transport-native `hello/snapshot/subscribed/ui-event` frames.

### Drop

These legacy behaviors are intentionally not part of the live path anymore:

1. **`/chat` as a live submit route**
2. **`/ws?conv_id=...` as the live attach contract**
3. **`/api/timeline` as the live hydration contract**
4. **Implicit dependence on legacy SEM transport details in the normal frontend path**

## Intentional vs bug vs follow-up

### Intentional

- Session-based identity replacing `conv_id`
- Explicit websocket subscribe flow
- Canonical session snapshot endpoint replacing `/api/timeline`
- Legacy live routes returning `404` after cutover

### Not classified as bugs from this evidence

Nothing in these two captures looks like a migration bug for the basic happy path. The canonical path appears to preserve the product workflow while changing the runtime contract as designed.

### Follow-up

1. **Real inference comparison is still separate work**
   - These captures use deterministic synthetic outputs, not live provider calls.
2. **Phase 6 Systemlab migration console still needs to be built**
   - That is the right place to surface these comparisons interactively.
3. **Final cutover/removal playbook still needs one explicit write-up**
   - The runtime has cut over, but the final review note should state what else can now be deleted or archived.

## Recommended interpretation for reviewers

If a reviewer sees differences between old and new flow artifacts, the default assumption should now be:

- **different route/protocol shape** → probably intentional;
- **different user-visible outcome** → investigate as possible regression;
- **legacy path now 404s in the live runtime** → intentional and desired after the cutover slice.

## Evidence references

- `reference/04-live-route-cutover-checks.json`
- `reference/05-legacy-flow-transcript.json`
- `reference/06-canonical-flow-transcript.json`
- `reference/03-live-runtime-session-snapshot.json`
