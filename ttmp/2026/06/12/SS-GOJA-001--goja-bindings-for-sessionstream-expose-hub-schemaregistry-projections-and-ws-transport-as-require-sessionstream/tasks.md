# Tasks

## TODO

- [x] Add tasks here

- [x] Create ticket and initial docs
- [x] Investigate sessionstream, go-go-goja EventEmitter, Geppetto bindings, and express/gojahttp patterns
- [x] Write intern-facing analysis/design/implementation guide
- [ ] Validate ticket docs and upload bundle to reMarkable
- [x] Add baseline sessionstream Goja/protobuf integration demo target: decide package path, confirm chatdemo proto generation, and document whether this starts as examples/goja-chatdemo or pkg/js/modules/sessionstream tests
- [x] Generate Goja protobuf builder companions for examples/chatdemo proto schemas using protoc-gen-goja-builder and commit the generated *_goja.pb.go outputs beside existing chatdemo pb.go files
- [x] Add a raw Goja require.Registry test that registers the generated chatdemo protobuf module, executes JavaScript building StartInferenceCommand, and extracts *chatdemov1.StartInferenceCommand via protogoja.MessageFromValue
- [x] Route the JavaScript-built StartInferenceCommand through the existing chatdemo Engine/Service or Hub and assert user prompt submission, emitted UI events, and snapshot entities
- [x] Add an xgoja provider example for sessionstream chatdemo protobuf builders, including provider registration, xgoja.yaml selection, generated TypeScript declarations, and a smoke test
- [x] Implement pkg/js/modules/sessionstream module skeleton with Options, NewLoader, Register, hidden refs, TypeScript descriptor, and goja-repl/xgoja-compatible module registration
- [x] Implement protobuf-aware codec helpers for sessionstream JS bindings, preferring protogoja.MessageFromValue for generated builder values and protojson/schema-registry conversion for plain JS objects
- [x] Implement SchemaRegistry JavaScript wrapper using host/global protobuf prototypes and generated prototype tokens where available; keep arbitrary top-level Struct schemas out of phase 1
- [x] Implement Hub JavaScript wrapper with hub({schemas, fanout}), submit, snapshot, run/shutdown lifecycle, projection policy options, and typed error reporting
- [x] Implement JavaScript command handler adapter using runtimeowner.RuntimeOwner.Call, EventPublisher wrapper, typed event publishing, reentrant submit safety, and explicit Promise-support decision
- [x] Implement JavaScript UIProjection and TimelineProjection adapters plus read-only TimelineView wrapper with get/list/ordinal and typed UI/entity payload conversion
- [x] Implement EventEmitter-backed UIFanout adapter using jsevents.Manager/EmitterRef so Go-side fanout safely emits UI batches onto the Goja owner thread
- [x] Add WebSocket server wrapper around existing transport/ws.Server and decide first mount path: provider-level gojahttp host mount versus narrow JavaScript mount helper
- [x] Add sessionstream xgoja provider package mirroring Geppetto provider style, including package id, module registration, config schema, TypeScript descriptor, host services, and optional WebSocket config
- [x] Write TypeScript declarations for require('sessionstream') covering schemas, Hub, Publisher, projections, TimelineView, fanout, WebSocket helpers, and protobuf-builder input shapes
- [x] Add end-to-end JS chatdemo integration test recreating a minimal chat flow: generated protobuf command, JS handler, event publish, UI projection, timeline projection, EventEmitter fanout, and snapshot
- [x] Update README/help docs with the sessionstream Goja workflow, including how it composes with go-go-goja protobuf builders and how to run the demo/smoke tests
- [x] Run validation for sessionstream integration: go test ./..., make schema-vet, targeted pkg/js/modules/sessionstream tests, and any required go-go-goja xgoja/protogoja compatibility tests
