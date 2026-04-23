# Tasks

## Completed setup/documentation work

- [x] Create a new repo-local sessionstream ticket for the job demo interactive essay.
- [x] Write a detailed intern-facing analysis / design / implementation guide.
- [x] Create a diary for the ticket.
- [x] Relate the main framework and systemlab files to the ticket docs.
- [x] Validate the ticket with `docmgr doctor`.
- [x] Upload the guide bundle to reMarkable.

## Planned implementation work

### Phase 0 — Concept freeze and example boundaries
- [ ] Freeze the conceptual direction: job demo as a textbook-style interactive essay, not a generic dashboard app.
- [ ] Decide the exact ownership split between `examples/jobdemo` and `cmd/sessionstream-systemlab`.
- [ ] Freeze the minimal job domain vocabulary (`StartJob`, `RetryFailedStep`, `CancelJob`, etc.).
- [ ] Freeze the essay chapter list and the persistent inspector surfaces.

### Phase 1 — `examples/jobdemo` reusable example package
- [ ] Add `sessionstream/examples/jobdemo` as a small reusable example package.
- [ ] Register jobdemo command, event, UI-event, and timeline-entity schemas through the schema registry.
- [ ] Implement a small engine/service pair that can simulate long-running work, failure, retry, and cancellation.
- [ ] Add focused tests covering command flow, event emission, retry history, and cancellation.

### Phase 2 — systemlab essay page shell
- [ ] Add a new page route/partial/module in `cmd/sessionstream-systemlab` for the job demo essay.
- [ ] Add chapter markdown for the essay page using the textbook-style framing defined in the design doc.
- [ ] Add a global session/job status rail with session identity, connection state, and last ordinal.
- [ ] Add the first interactive section proving command -> event -> projection flow.

### Phase 3 — progressive interactions and inspectors
- [ ] Add sections for streaming progress, one-stream-many-views, failure, retry, and cancellation.
- [ ] Add raw event inspector and projected-state inspector panes.
- [ ] Add user actions that intentionally perturb the job session (`fail`, `retry`, `cancel`, `auto-run`).
- [ ] Keep all interactions routed through public `sessionstream` APIs rather than page-local state hacks.

### Phase 4 — hydration/reconnect teaching slice
- [ ] Add disconnect / reconnect / hydration demonstrations to the essay.
- [ ] Show how the websocket transport and snapshot store interact during recovery.
- [ ] Add a visible explanation of what was persisted versus what was only transient.
- [ ] Validate at least one reload/reconnect scenario end to end.

### Phase 5 — polish, teaching quality, and delivery
- [ ] Review the prose and section flow against the textbook-writing guidance.
- [ ] Add review-friendly screenshots / artifacts if the implementation warrants them.
- [ ] Add a short contributor playbook for extending the essay with future interactive chapters.
- [ ] Re-upload the refreshed bundle to reMarkable after implementation lands.

## Exit criteria

- [ ] `examples/jobdemo` exists as a clean reusable example package.
- [ ] `cmd/sessionstream-systemlab` includes a flagship job-demo interactive essay page.
- [ ] The page teaches commands, events, projections, and hydration by direct interaction.
- [ ] The page exposes raw events and projected state side by side.
- [ ] The implementation is documented well enough for a new intern to extend it confidently.
