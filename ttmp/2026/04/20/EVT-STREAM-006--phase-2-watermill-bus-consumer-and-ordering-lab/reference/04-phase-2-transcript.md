---
Title: Phase 2 Transcript
Ticket: EVT-STREAM-006
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
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/pinocchio/cmd/evtstream-systemlab/phase2_lab.go
      Note: Source of the exported transcript content and ordering-lab behavior.
    - Path: /home/manuel/workspaces/2026-04-07/extract-webchat/le-chat/ttmp/2026/04/20/EVT-STREAM-006--phase-2-watermill-bus-consumer-and-ordering-lab/reference/02-phase-2-run-response.json
      Note: Raw JSON response for the same captured run.
ExternalSources: []
Summary: "Captured Markdown transcript exported from Systemlab Lab 02 after publishing ordered events for sessions s-a and s-b over the Phase 2 Watermill gochannel bus."
LastUpdated: 2026-04-20T01:35:00-04:00
WhatFor: "Provide a human-readable Phase 2 artifact showing publish/consume metadata, per-session ordinals, and trace output."
WhenToUse: "When reviewing the Ordering and Ordinals lab output, sharing a concise artifact, or comparing later regressions."
---

# Phase 2 Transcript

- Action: `burst-a`
- Stream mode: `derived`

## Checks

- messagesConsumed: PASS
- monotonicPerSession: PASS
- publishOrdinalZero: PASS
- sessionIsolation: PASS

## Message History

- `{"assignedOrdinal":"1713560000123000001","consumeMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-1"},"eventName":"OrderedEvent","label":"s-a-01","messageId":"c3e90c2e-fcc7-43b5-9f76-57ff69579581","publishMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-1"},"publishedOrdinal":"0","sessionId":"s-a","topic":"evtstream.phase2"}`
- `{"assignedOrdinal":"1713560000123000002","consumeMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-b","evtstream_published_ordinal":"0","evtstream_session_id":"s-b","evtstream_stream_id":"1713560000123-2"},"eventName":"OrderedEvent","label":"s-b-01","messageId":"476e8754-0dc4-4ae6-aca5-52b179b89aa7","publishMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-b","evtstream_published_ordinal":"0","evtstream_session_id":"s-b","evtstream_stream_id":"1713560000123-2"},"publishedOrdinal":"0","sessionId":"s-b","topic":"evtstream.phase2"}`
- `{"assignedOrdinal":"1713560000123000006","consumeMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-3"},"eventName":"OrderedEvent","label":"s-a-02","messageId":"f5431995-9cfd-4320-ac05-e1c411a9879e","publishMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-3"},"publishedOrdinal":"0","sessionId":"s-a","topic":"evtstream.phase2"}`
- `{"assignedOrdinal":"1713560000123000007","consumeMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-4"},"eventName":"OrderedEvent","label":"s-a-03","messageId":"971c27ac-b206-4138-bbd9-d3d85ff76af9","publishMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-4"},"publishedOrdinal":"0","sessionId":"s-a","topic":"evtstream.phase2"}`
- `{"assignedOrdinal":"1713560000123000005","consumeMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-5"},"eventName":"OrderedEvent","label":"s-a-04","messageId":"57287cd2-406b-494e-a184-54d363bab1e9","publishMetadata":{"evtstream_event_name":"OrderedEvent","evtstream_partition_key":"s-a","evtstream_published_ordinal":"0","evtstream_session_id":"s-a","evtstream_stream_id":"1713560000123-5"},"publishedOrdinal":"0","sessionId":"s-a","topic":"evtstream.phase2"}`

## Per-Session Ordinals

```json
{
  "s-a": [
    "1713560000123000001",
    "1713560000123000005",
    "1713560000123000006",
    "1713560000123000007"
  ],
  "s-b": [
    "1713560000123000002"
  ]
}
```

## Trace

1. **consumer** — phase 2 consumer started
   - details: `{"topic":"evtstream.phase2"}`
2. **control** — phase 2 action requested
   - details: `{"action":"publish-a","burstCount":3,"sessionA":"s-a","sessionB":"s-b","streamMode":"derived"}`
3. **session** — phase 2 session created
   - details: `{"createdBy":"evtstream-systemlab","lab":"phase2","sessionId":"s-a"}`
4. **handler** — phase 2 handler invoked
   - details: `{"hasSession":true,"label":"s-a-01","sessionId":"s-a"}`
5. **publish** — phase 2 event published
   - details: `{"messageId":"c3e90c2e-fcc7-43b5-9f76-57ff69579581","sessionId":"s-a","streamId":"1713560000123-1"}`
6. **consume** — phase 2 event consumed
   - details: `{"messageId":"c3e90c2e-fcc7-43b5-9f76-57ff69579581","ordinal":"1713560000123000001","sessionId":"s-a","streamId":"1713560000123-1"}`
7. **control** — phase 2 action requested
   - details: `{"action":"publish-b","burstCount":3,"sessionA":"s-a","sessionB":"s-b","streamMode":"derived"}`
8. **session** — phase 2 session created
   - details: `{"createdBy":"evtstream-systemlab","lab":"phase2","sessionId":"s-b"}`
9. **handler** — phase 2 handler invoked
   - details: `{"hasSession":true,"label":"s-b-01","sessionId":"s-b"}`
10. **publish** — phase 2 event published
   - details: `{"messageId":"476e8754-0dc4-4ae6-aca5-52b179b89aa7","sessionId":"s-b","streamId":"1713560000123-2"}`
11. **consume** — phase 2 event consumed
   - details: `{"messageId":"476e8754-0dc4-4ae6-aca5-52b179b89aa7","ordinal":"1713560000123000002","sessionId":"s-b","streamId":"1713560000123-2"}`
12. **control** — phase 2 action requested
   - details: `{"action":"burst-a","burstCount":3,"sessionA":"s-a","sessionB":"s-b","streamMode":"derived"}`
13. **handler** — phase 2 handler invoked
   - details: `{"hasSession":true,"label":"s-a-02","sessionId":"s-a"}`
14. **publish** — phase 2 event published
   - details: `{"messageId":"f5431995-9cfd-4320-ac05-e1c411a9879e","sessionId":"s-a","streamId":"1713560000123-3"}`
15. **handler** — phase 2 handler invoked
   - details: `{"hasSession":true,"label":"s-a-03","sessionId":"s-a"}`
16. **publish** — phase 2 event published
   - details: `{"messageId":"971c27ac-b206-4138-bbd9-d3d85ff76af9","sessionId":"s-a","streamId":"1713560000123-4"}`
17. **handler** — phase 2 handler invoked
   - details: `{"hasSession":true,"label":"s-a-04","sessionId":"s-a"}`
18. **publish** — phase 2 event published
   - details: `{"messageId":"57287cd2-406b-494e-a184-54d363bab1e9","sessionId":"s-a","streamId":"1713560000123-5"}`
19. **consume** — phase 2 event consumed
   - details: `{"messageId":"57287cd2-406b-494e-a184-54d363bab1e9","ordinal":"1713560000123000005","sessionId":"s-a","streamId":"1713560000123-5"}`
20. **consume** — phase 2 event consumed
   - details: `{"messageId":"f5431995-9cfd-4320-ac05-e1c411a9879e","ordinal":"1713560000123000006","sessionId":"s-a","streamId":"1713560000123-3"}`
21. **consume** — phase 2 event consumed
   - details: `{"messageId":"971c27ac-b206-4138-bbd9-d3d85ff76af9","ordinal":"1713560000123000007","sessionId":"s-a","streamId":"1713560000123-4"}`
