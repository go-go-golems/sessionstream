---
Title: Phase 1 Transcript
Ticket: EVT-STREAM-005
Status: active
Topics:
    - framework
    - event-streaming
    - transcript
    - validation
DocType: reference
Intent: short-term
Owners: []
RelatedFiles:
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/lab_environment.go
      Note: Source of the exported transcript content.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-005--phase-1-in-memory-core-and-command-to-projection-lab/reference/02-phase-1-run-response.json
      Note: Raw JSON response for the same captured run.
ExternalSources: []
Summary: "Captured happy-path Markdown transcript exported from Systemlab Lab 01 after submitting a LabStart command for session lab-session-1."
LastUpdated: 2026-04-19T23:45:00-04:00
WhatFor: "Provide a human-readable Phase 1 artifact showing trace rows, checks, UI events, and hydration snapshot output."
WhenToUse: "When reviewing the happy-path lab output, comparing later regressions, or sharing a concise run artifact."
---

# Phase 1 Transcript

- Session ID: `lab-session-1`

## Checks

- cursorAdvanced: PASS
- timelineProduced: PASS
- uiEventsProduced: PASS
- sessionExists: PASS

## Trace

1. **command** — submit command
   - details: `{"commandName":"LabStart","prompt":"hello from systemlab"}`
2. **session** — session created
   - details: `{"createdBy":"evtstream-systemlab","lab":"phase1","sessionId":"lab-session-1"}`
3. **handler** — handler invoked
   - details: `{"hasSession":true,"messageId":"msg-1","sessionId":"lab-session-1"}`
4. **ui-projection** — ui projection emitted event
   - details: `{"ordinal":1,"sourceEvent":"LabStarted","uiEvent":"LabMessageStarted"}`
5. **timeline-projection** — timeline projection upserted entity
   - details: `{"entityId":"msg-1","ordinal":1,"sourceEvent":"LabStarted"}`
6. **ui-projection** — ui projection emitted event
   - details: `{"ordinal":2,"sourceEvent":"LabChunk","uiEvent":"LabMessageAppended"}`
7. **timeline-projection** — timeline projection upserted entity
   - details: `{"entityId":"msg-1","ordinal":2,"sourceEvent":"LabChunk"}`
8. **ui-projection** — ui projection emitted event
   - details: `{"ordinal":3,"sourceEvent":"LabChunk","uiEvent":"LabMessageAppended"}`
9. **timeline-projection** — timeline projection upserted entity
   - details: `{"entityId":"msg-1","ordinal":3,"sourceEvent":"LabChunk"}`
10. **ui-projection** — ui projection emitted event
   - details: `{"ordinal":4,"sourceEvent":"LabFinished","uiEvent":"LabMessageFinished"}`
11. **timeline-projection** — timeline projection upserted entity
   - details: `{"entityId":"msg-1","ordinal":4,"sourceEvent":"LabFinished"}`

## UI Events

- `LabMessageStarted` — `{"messageId":"msg-1","prompt":"hello from systemlab"}`
- `LabMessageAppended` — `{"chunk":"hello from","messageId":"msg-1"}`
- `LabMessageAppended` — `{"chunk":" systemlab","messageId":"msg-1"}`
- `LabMessageFinished` — `{"messageId":"msg-1","text":"hello from systemlab"}`

## Snapshot

```json
{
  "entities": [
    {
      "id": "msg-1",
      "kind": "LabMessage",
      "payload": {
        "messageId": "msg-1",
        "status": "finished",
        "text": "hello from systemlab"
      }
    }
  ],
  "ordinal": 4,
  "sessionId": "lab-session-1"
}
```
