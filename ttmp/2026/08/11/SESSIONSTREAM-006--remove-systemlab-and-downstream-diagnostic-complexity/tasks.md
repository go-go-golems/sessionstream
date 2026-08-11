# Tasks

## TODO

- [ ] Rebase onto main after PR #11 merges and establish clean baseline validation <!-- t:4a4t -->
- [ ] Inventory all Systemlab code, build, CI, documentation, release, dependency, and generated-file edges <!-- t:ldu1 -->
- [ ] Search local and remote downstream consumers of BusObserver, PipelineObserver, TransportObserver, and ErrorObserver <!-- t:ybz9 -->
- [ ] Decide the retained observer APIs and document public compatibility/versioning policy <!-- t:zzdr -->
- [ ] Delete cmd/sessionstream-systemlab and direct Makefile, CI, release, and documentation wiring <!-- t:a5gd -->
- [ ] Preserve browser heartbeat protocol with a minimal retained conformance reference <!-- t:t6w6 -->
- [ ] Remove approved orphan BusObserver and its callback, record, option, and test surface <!-- t:lflb -->
- [ ] Remove approved orphan PipelineObserver and its callback, record, option, and test surface <!-- t:yvxc -->
- [ ] Remove approved orphan TransportObserver and its record, cloning, dispatcher, lifecycle, and test surface <!-- t:8dep -->
- [ ] Decide and implement the independent ErrorObserver disposition <!-- t:fgl3 -->
- [ ] Tidy dependencies, regenerate metadata, and record before/after simplification metrics <!-- t:vr2f -->
- [ ] Run full workspace, GOWORK=off, race, lint, vet, vulnerability, release, and rag-ttc downstream validation <!-- t:3bnu -->
