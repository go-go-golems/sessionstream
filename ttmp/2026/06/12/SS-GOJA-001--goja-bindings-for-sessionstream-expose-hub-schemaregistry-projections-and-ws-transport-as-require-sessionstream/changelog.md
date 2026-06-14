# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Created intern-facing Goja sessionstream bindings design guide and investigation diary covering Hub, SchemaRegistry, projections, EventEmitter fanout, xgoja provider, and WebSocket/express integration paths.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/design-doc/01-goja-sessionstream-bindings-design.md — Primary intern-facing design and implementation guide
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/reference/01-investigation-diary.md — Chronological investigation diary and continuation notes


## 2026-06-12

Validated SS-GOJA-001 docs with docmgr doctor and uploaded the design bundle to reMarkable at /ai/2026/06/12/SS-GOJA-001.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/design-doc/01-goja-sessionstream-bindings-design.md — Uploaded as part of reMarkable bundle
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/reference/01-investigation-diary.md — Updated with upload failure/retry details and uploaded as part of bundle


## 2026-06-12

Expanded SS-GOJA-001 with implementation tasks 6-23, updated for the merged go-go-goja protobuf builder workflow and a sessionstream chatdemo Goja/protobuf demo first milestone.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/reference/01-investigation-diary.md — Diary step recording the task expansion
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/12/SS-GOJA-001--goja-bindings-for-sessionstream-expose-hub-schemaregistry-projections-and-ws-transport-as-require-sessionstream/tasks.md — New implementation task sequence


## 2026-06-12

Completed tasks 6-10: generated chatdemo Goja protobuf builders, added compiled goja-chatdemo proof, raw require.Registry extraction test, chatdemo Hub routing assertions, and xgoja provider/DTS smoke tests.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1/chat_goja.pb.go — Generated Goja protobuf builder companion for chatdemo schema
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/chatdemo/generate.go — go generate now emits chatdemo Goja protobuf builders
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/goja-chatdemo/goja_chatdemo_test.go — Compiled proof from JavaScript builder to sessionstream Hub
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/goja-chatdemo/provider/provider.go — xgoja provider package for generated chatdemo protobuf builders


## 2026-06-12

Completed tasks 11-23: implemented phase-1 require("sessionstream") module, schema/protobuf codecs, Hub wrappers, JS callbacks/projections, EventEmitter fanout, WebSocket wrapper, xgoja provider, DTS, docs, and validation.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_callbacks.go — JS command/projection adapters
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_fanout.go — EventEmitter-backed UIFanout bridge
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_hub.go — Hub wrapper API
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module.go — Native CommonJS module skeleton and refs
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/provider/provider.go — xgoja provider registration


## 2026-06-12

Step 6: attached sessionstream WebSocket server objects to the merged gojahttp mountable handler ABI and verified Express app.mount composition (commit 8ab489f).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_websocket.go — WebSocket server objects now carry shared HTTP handler refs
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module_test.go — Coverage for app.mount('/ws'


## 2026-06-12

Step 7: aligned CI/CD plumbing with go-template, added ci-check and bump-go-go-golems, regenerated logcopter, and validated make ci-check plus goreleaser snapshot (commit 708f869).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/.github/workflows/push.yml — Hosted CI entrypoint
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/Makefile — CI/lint/logcopter/glazed-lint/release targets
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/go.mod — Standalone go-go-goja dependency


## 2026-06-12

Step 8: fixed hosted CI tool assumptions by removing rg usage and installing protoc-gen-go/protoc-gen-goja-builder before go generate.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/Makefile — Installs generation tools and avoids rg in CI


## 2026-06-14

Step 9: finished the real xgoja goja-chatdemo-server smoke against RuntimePlan v2; adapted provider IDs, HTTP serve flags, WebSocket fanout attachment, and runtime-owner context preservation (commit d852d8b).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/goja-chatdemo-server/xgoja.yaml — RuntimePlan v2 example spec
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_callbacks.go — Publisher preserves current owner context
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/sessionstream/hub.go — Dynamic SetUIFanout for xgoja WebSocket composition


## 2026-06-14

Step 10: split chatdemo browser UI into xgoja embedded assets, served CSS/JS through fs:assets/staticFromAssetsModule, and added /api/config plus ?sessionId= override (commit 5eba490).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/goja-chatdemo-server/assets/public/app.js — Browser session id selection and WebSocket client
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/examples/goja-chatdemo-server/xgoja.yaml — Embedded assets source and artifact

