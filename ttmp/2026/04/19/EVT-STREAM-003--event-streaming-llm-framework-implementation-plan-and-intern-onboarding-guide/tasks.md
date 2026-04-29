# Tasks

## TODO

- [x] Create implementation ticket workspace and primary docs
- [x] Write a detailed implementation/onboarding guide for a new engineer
- [x] Ground the implementation guide in EVT-STREAM-002 design docs and current `pinocchio/pkg/webchat` donor code
- [ ] Phase 0: Scaffold `pinocchio/pkg/evtstream` package skeleton and compile-only interfaces
- [ ] Phase 1: Implement in-memory Hub/Session/Command/Projection/Hydration path with `Hub.Submit`
- [ ] Phase 2: Implement Watermill-backed event publisher and consumer with consumption-time ordinal stamping
- [ ] Phase 3: Implement websocket transport with snapshot-before-live subscribe behavior
- [ ] Phase 4: Implement `examples/chat` as the first backend on top of the substrate
- [ ] Phase 5: Add durable SQL hydration store and restart/cursor tests
- [ ] Phase 6: Rebuild or bridge webchat on top of the new substrate
- [x] Upload the implementation guide bundle to reMarkable

## Open Questions

- [ ] Confirm final code home: `pinocchio/pkg/evtstream` vs standalone module extraction timeline
- [ ] Confirm initial `SessionId` allocation policy
- [ ] Confirm default projection error policy (`log + advance` vs stricter)
- [ ] Confirm whether TS client work is part of the first implementation milestone
- [ ] Confirm liveness/tick protocol timing for the first public transport release
