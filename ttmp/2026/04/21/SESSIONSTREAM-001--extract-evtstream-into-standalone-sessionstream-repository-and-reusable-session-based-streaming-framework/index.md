---
Title: Extract evtstream into standalone sessionstream repository and reusable session-based streaming framework
Ticket: SESSIONSTREAM-001
Status: active
Topics:
    - architecture
    - backend
    - framework
    - event-streaming
    - migration
    - extraction
    - onboarding
    - systemlab
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: First repository-local planning ticket in sessionstream for extracting the generic evtstream substrate into a standalone sessionstream module while keeping the real pinocchio chat app downstream, moving agentmode ownership outward to pinocchio-owned adapters, and carrying only framework-oriented Systemlab/example material into the new repo.
LastUpdated: 2026-04-21T15:40:00-04:00
WhatFor: Track the standalone sessionstream extraction plan and the supporting intern-facing analysis documents in the destination repository itself.
WhenToUse: Use when orienting contributors to the sessionstream extraction effort, reviewing the plan, or locating the canonical repo-local design docs for the move.
---

# Extract evtstream into standalone sessionstream repository and reusable session-based streaming framework

## Overview

`SESSIONSTREAM-001` is the first planning ticket in the new `sessionstream` repository. Its purpose is to turn the old deferred idea of “extract evtstream later” into a repository-local plan with a real destination, a chronological diary, and a detailed intern-facing design/implementation guide.

The ticket’s core recommendation is:

- move the generic `evtstream` substrate into `sessionstream`,
- move the framework-oriented Systemlab material with it as a companion app,
- keep pinocchio-specific runtime, middleware, and the real chat product in `pinocchio`,
- move `agentmode` ownership out to `cmd/web-chat` or another pinocchio-owned adapter layer,
- and provide only a small generic demo/example chat app in `sessionstream` rather than trying to extract the whole real chat stack.

## Current status

Current status: **active**

This ticket currently contains:

- a detailed design doc / intern guide,
- a chronological diary,
- ticket tasks and changelog,
- repo-local docmgr setup aligned to `sessionstream/ttmp`.

## Primary documents

- [Design doc](./design-doc/01-intern-guide-and-extraction-plan-for-moving-evtstream-into-standalone-sessionstream.md)
- [Diary](./reference/01-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Why this ticket exists in sessionstream instead of le-chat

The earlier extraction plan lived in the old workspace as a deferred follow-up note. That was useful while `sessionstream` did not yet exist. Now that the destination repository exists, the canonical extraction planning ticket should live here instead so that:

- future contributors can find the plan where the work will happen,
- the repository can build its own documentation history,
- and the new repo does not depend on external ticket workspaces for foundational context.

## Planned outcome

Target relationship after the extraction program:

```text
sessionstream/
  -> generic session-streaming substrate
  -> optional stores/transports/examples
  -> framework-oriented Systemlab companion app

pinocchio/
  -> downstream consumer of sessionstream
  -> real chat app built on sessionstream
  -> product-specific cmd/web-chat app
  -> pinocchio runtime and middleware adapters
```

## Structure

- `design-doc/` — long-form architecture and implementation guidance
- `reference/` — diary and future quick-reference material
- `playbooks/` — future execution runbooks and smoke-test procedures
- `scripts/` — temporary scripts used by this ticket
- `various/` — scratch/supporting notes if needed later
- `archive/` — retired material if the plan evolves
