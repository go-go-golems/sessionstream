# Changelog

## 2026-06-15

- Initial workspace created


## 2026-06-15

Created evidence-backed analysis and implementation guide for lossy Goja schema map handling; saved reproduction scripts and outputs for future implementation.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md — Primary deliverable
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/reference/01-investigation-diary.md — Chronological investigation record


## 2026-06-15

Validated ticket docs with docmgr doctor and uploaded the analysis bundle to reMarkable at /ai/2026/06/15/SS-GOJA-SCHEMA-MAP-001.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/sources/SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide.pdf — Rendered PDF bundle uploaded to reMarkable


## 2026-06-16

Tightened schema map design to support only generated MessageNamespace values and protobuf full-name strings; object descriptor fallback will be removed.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/design-doc/01-goja-schema-map-protobuf-namespace-analysis-and-implementation-guide.md — Updated API decision and implementation plan


## 2026-06-16

Implemented strict bulk schema registration by walking original Goja objects, preserving generated namespace prototypes, and rejecting plain descriptor objects.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_schemas.go — Removed ExportTo/map[string]any schema handling and added direct Goja section traversal
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module_test.go — Added bulk namespace


## 2026-06-16

Committed strict bulk schema registration implementation (298d0496b4a2661d44ca80f74d5538fe81e28e51).

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/api_schemas.go — Committed direct Goja schema traversal
- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/pkg/js/modules/sessionstream/module_test.go — Committed regression coverage


## 2026-06-16

Refreshed and re-uploaded reMarkable bundle after implementation diary/design updates.

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-sessionstream/sessionstream/ttmp/2026/06/15/SS-GOJA-SCHEMA-MAP-001--analyze-lossy-goja-schema-map-export-and-protobuf-namespace-handling/sources/SS-GOJA-SCHEMA-MAP-001 schema map protobuf namespace guide.pdf — Updated rendered bundle

