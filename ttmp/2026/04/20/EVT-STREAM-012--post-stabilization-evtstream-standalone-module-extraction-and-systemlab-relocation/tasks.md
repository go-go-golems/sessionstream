# Tasks

## TODO

### Readiness and boundary review
- [ ] Confirm `pkg/evtstream` remains free of non-evtstream `pinocchio` package imports.
- [ ] Confirm `cmd/evtstream-systemlab` only depends on public `evtstream` APIs and its own assets/app code.
- [ ] Inventory what `cmd/web-chat` will consume from `evtstream` after the Phase 6 cutover.
- [ ] Decide the threshold for “stable enough to extract” (tests green, API shape stable, docs updated, Systemlab stable, at least one real app consumer working).

### Extraction design
- [ ] Freeze the intended public package surface for the first standalone `evtstream` release.
- [ ] Decide the standalone module location and import path strategy.
- [ ] Define the target repository/module layout for core, hydration stores, transports, examples, and Systemlab.
- [ ] Decide whether `examples/chat` ships in the standalone module from day one or follows later.
- [ ] Specify what stays in `pinocchio` permanently versus what moves with the extracted module.

### Later execution plan
- [ ] Move `evtstream` core packages into the standalone module.
- [ ] Move `cmd/evtstream-systemlab` into the standalone module as a separate app/command, not into substrate core.
- [ ] Update `pinocchio` imports so `cmd/web-chat` and any other consumers depend on the standalone module.
- [ ] Update workspace/CI/tooling so tests can run both in the standalone module and in `pinocchio` as a downstream consumer.
- [ ] Add or strengthen boundary checks that forbid the extracted module from importing `pinocchio` internals.

### Validation and publication
- [ ] Validate that Systemlab still exercises only public `evtstream` seams after the move.
- [ ] Validate that `pinocchio` can consume the extracted module without local-path hacks beyond normal workspace setup.
- [ ] Document versioning/release expectations for the standalone module.
- [ ] Capture the final cutover notes and any follow-up cleanup work in both repos.

## Exit Criteria

- [ ] A detailed extraction plan exists and is stable enough to execute later without rediscovering the architecture from scratch.
- [ ] The preconditions for extraction are explicit and measurable.
- [ ] The intended standalone module boundary is clearly documented.
- [ ] Systemlab’s future home is defined as part of the `evtstream` module/repo, while remaining separate from substrate core.
- [ ] The ticket can be picked up later as an execution plan once stabilization is complete.
