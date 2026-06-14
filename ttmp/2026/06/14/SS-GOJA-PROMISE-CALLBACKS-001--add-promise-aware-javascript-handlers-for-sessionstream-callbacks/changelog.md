# Changelog

## 2026-06-14

- Initial workspace created


## 2026-06-14

Created Promise-aware JS callback ticket with design, detailed task plan, and initial diary.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/14/SS-GOJA-PROMISE-CALLBACKS-001--add-promise-aware-javascript-handlers-for-sessionstream-callbacks/design-doc/01-promise-aware-sessionstream-js-callback-design.md — Initial design
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/14/SS-GOJA-PROMISE-CALLBACKS-001--add-promise-aware-javascript-handlers-for-sessionstream-callbacks/tasks.md — Detailed implementation task list


## 2026-06-14

Implemented Promise-aware JS command/projection callbacks plus submitAsync/publishAsync; focused, full, and chatdemo smoke validations pass.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_hub.go — submitAsync API
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_promises.go — Promise wait helper


## 2026-06-14

Replaced JS submitAsync/publishAsync with Promise-native submit/publish and added in-memory hub.enqueue receipts; focused/full tests and chatdemo smoke pass.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_callbacks.go — publish implementation
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_hub.go — submit/enqueue implementation


## 2026-06-14

Removed experimental local hub.enqueue API and kept Promise-native JS submit/publish only; focused/full tests and chatdemo smoke pass.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_hub.go — enqueue removal
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/typescript.go — enqueue removed from declarations


## 2026-06-14

Added rejected async UI/timeline projection regressions for Promise-native submit/publish error propagation.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module_test.go — Rejected async projection regressions

