---
Title: Sessionstream schema vet analyzer migration plan
Ticket: SS-SCHEMA-VET
Status: active
Topics:
    - sessionstream
    - go-vet
    - protobuf
    - chatapp
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../pinocchio/pkg/analysis/sessionstreamschema/analyzer.go
    - Path: ../pinocchio/cmd/tools/pinocchio-lint/main.go
    - Path: ../pinocchio/Makefile
    - Path: ../2026-03-16--gec-rag/Makefile
    - Path: pkg/sessionstream/schema.go
ExternalSources: []
Summary: Plan to move the schema registration vet analyzer from Pinocchio into Sessionstream and wire it into Pinocchio, CoinVault, and future chatapp projects.
LastUpdated: 2026-05-06T17:15:39.31526679-04:00
WhatFor: "Guide implementation of a shared go vet analyzer that enforces concrete protobuf payloads for sessionstream schema registrations."
WhenToUse: "Use when moving the analyzer out of Pinocchio, adding schema-vet targets to sessionstream consumers, or extending schema policy enforcement."
---

# Sessionstream schema vet analyzer migration plan

## Goal

Move the current Pinocchio-local schema vet analyzer into `sessionstream`, then wire it into Pinocchio, CoinVault, and future chatapp/sessionstream-based projects.

The policy being enforced is not Pinocchio-specific:

> Top-level payloads registered with `sessionstream.SchemaRegistry` must be concrete feature-owned protobuf messages, not `*structpb.Struct`.

This belongs in `sessionstream` because `SchemaRegistry` is defined there and all downstream applications use the same registration contract.

## Current state

Pinocchio currently owns the analyzer:

- `../pinocchio/pkg/analysis/sessionstreamschema/analyzer.go`
- `../pinocchio/cmd/tools/pinocchio-lint/main.go`
- `../pinocchio/Makefile` target: `schema-vet`

The analyzer is type-aware. It scans calls whose receiver is:

```go
*github.com/go-go-golems/sessionstream/pkg/sessionstream.SchemaRegistry
```

and rejects registrations like:

```go
reg.RegisterCommand("SomeCommand", &structpb.Struct{})
reg.RegisterEvent("SomeEvent", &structpb.Struct{})
reg.RegisterUIEvent("SomeUIEvent", &structpb.Struct{})
reg.RegisterTimelineEntity("SomeEntity", &structpb.Struct{})
```

The rule already targets `sessionstream.SchemaRegistry`, which is the main signal that the analyzer should live with Sessionstream.

## Target ownership

Create a shared analyzer and vettool in `sessionstream`:

```text
sessionstream/pkg/analysis/sessionstreamschema/analyzer.go
sessionstream/cmd/sessionstream-lint/main.go
```

The command should be intentionally small:

```go
package main

import (
    "github.com/go-go-golems/sessionstream/pkg/analysis/sessionstreamschema"
    "golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
    singlechecker.Main(sessionstreamschema.Analyzer)
}
```

Downstream projects should then run:

```bash
go build -o /tmp/sessionstream-lint ../sessionstream/cmd/sessionstream-lint
go vet -vettool=/tmp/sessionstream-lint ./...
```

or, outside a workspace:

```bash
go install github.com/go-go-golems/sessionstream/cmd/sessionstream-lint@latest
go vet -vettool="$(go env GOPATH)/bin/sessionstream-lint" ./...
```

## Migration phases

### Phase 1 — Move analyzer into sessionstream

1. Copy `pinocchio/pkg/analysis/sessionstreamschema/analyzer.go` to:
   - `sessionstream/pkg/analysis/sessionstreamschema/analyzer.go`
2. Add command:
   - `sessionstream/cmd/sessionstream-lint/main.go`
3. Add the required `golang.org/x/tools` dependency to the Sessionstream module.
4. Add basic validation:
   - `go test ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint`
   - `go build -o /tmp/sessionstream-lint ./cmd/sessionstream-lint`

### Phase 2 — Add a Sessionstream Makefile target

Add a target similar to:

```make
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint

schema-vet:
	go build -o $(SESSIONSTREAM_LINT) ./cmd/sessionstream-lint
	go vet -vettool=$(SESSIONSTREAM_LINT) ./pkg/... ./cmd/...
```

If Sessionstream does not yet have many command packages, keep the package list narrow enough to avoid unrelated tool packages.

### Phase 3 — Update Pinocchio to consume the shared tool

Replace Pinocchio's local analyzer command with the Sessionstream vettool.

Recommended local-workspace target:

```make
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint
SESSIONSTREAM_LINT_PKG ?= ../sessionstream/cmd/sessionstream-lint

schema-vet:
	go build -o $(SESSIONSTREAM_LINT) $(SESSIONSTREAM_LINT_PKG)
	go vet -vettool=$(SESSIONSTREAM_LINT) ./cmd/... ./pkg/...
```

Then remove the Pinocchio-local copies:

```text
pinocchio/pkg/analysis/sessionstreamschema/analyzer.go
pinocchio/cmd/tools/pinocchio-lint/main.go
```

If `pinocchio-lint` is expected to grow with Pinocchio-specific analyzers later, keep the command but make it compose the Sessionstream analyzer rather than owning this rule. For now, the simplest path is deletion and replacement with `sessionstream-lint`.

### Phase 4 — Add CoinVault schema-vet

Add a CoinVault `Makefile` target:

```make
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint
SESSIONSTREAM_LINT_PKG ?= ../sessionstream/cmd/sessionstream-lint

schema-vet:
	go build -o $(SESSIONSTREAM_LINT) $(SESSIONSTREAM_LINT_PKG)
	go vet -vettool=$(SESSIONSTREAM_LINT) ./cmd/... ./internal/...
```

Run it as part of local validation for schema migrations:

```bash
cd 2026-03-16--gec-rag
make schema-vet
```

Do not wire it into strict pre-commit until the existing `GOWORK=off` local dependency mismatch issues are resolved, unless the hook is allowed to run in workspace mode.

### Phase 5 — Document downstream usage

Add short docs to Sessionstream explaining:

- why top-level `google.protobuf.Struct` registrations are rejected;
- how `Struct` may still appear inside a concrete message for intentionally open-ended metadata;
- how to run `sessionstream-lint` locally and in CI;
- how to add a `schema-vet` target in a downstream app.

Pinocchio and CoinVault docs should point to Sessionstream as the source of truth.

## Analyzer scope

### In scope now

Reject direct registrations of top-level generic Struct payloads:

```go
reg.RegisterEvent("X", &structpb.Struct{})
reg.RegisterUIEvent("X", &structpb.Struct{})
reg.RegisterTimelineEntity("X", &structpb.Struct{})
reg.RegisterCommand("X", &structpb.Struct{})
```

The analyzer should continue to be receiver-type-aware, not text-only. It should only fire for calls on `sessionstream.SchemaRegistry`.

### Out of scope for the first migration

Do not initially reject all nested `google.protobuf.Struct` fields. A typed message like this can be valid when it is intentionally open-ended metadata:

```proto
message ToolResultEntity {
  string tool_call_id = 1;
  google.protobuf.Struct metadata = 2;
}
```

Nested Struct policy needs a separate protobuf descriptor lint rule with allowlists/naming conventions. Keep this ticket focused on moving and wiring the existing top-level registration rule.

## Validation checklist

Run these checks after implementation:

```bash
cd sessionstream
go test ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint -count=1
go build -o /tmp/sessionstream-lint ./cmd/sessionstream-lint
go vet -vettool=/tmp/sessionstream-lint ./pkg/... ./cmd/...
```

```bash
cd pinocchio
make schema-vet
go test ./pkg/chatapp ./pkg/chatapp/plugins ./cmd/web-chat -count=1
```

```bash
cd 2026-03-16--gec-rag
make schema-vet
go test ./internal/webchat ./internal/projectionlookup ./internal/projectionblocks -count=1
```

Expected result:

- no `&structpb.Struct{}` top-level schema registrations pass in Pinocchio or CoinVault;
- the former Pinocchio analyzer files are removed or reduced to wrappers;
- all downstream `schema-vet` targets use `sessionstream-lint`.

## Done definition

- `sessionstream-lint` exists in the Sessionstream repo.
- Pinocchio no longer owns the Sessionstream schema-registration analyzer.
- Pinocchio `make schema-vet` builds/runs the shared Sessionstream vettool.
- CoinVault has a `schema-vet` target using the shared tool.
- Documentation identifies Sessionstream as the source of truth for schema registration policy.
- CI/pre-commit integration is documented; actual strict hook wiring can be a follow-up if local workspace dependency issues still block it.
