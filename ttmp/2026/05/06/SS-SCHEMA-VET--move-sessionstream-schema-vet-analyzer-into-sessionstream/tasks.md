---
Title: Move sessionstream schema vet analyzer into sessionstream tasks
Ticket: SS-SCHEMA-VET
Status: active
Topics:
    - sessionstream
    - go-vet
    - protobuf
    - chatapp
DocType: tasks
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Implementation checklist for moving the schema registration analyzer into Sessionstream and wiring downstream projects.
LastUpdated: 2026-05-06T17:16:00-04:00
WhatFor: "Track implementation work for the shared sessionstream schema vet analyzer."
WhenToUse: "Use while implementing SS-SCHEMA-VET or reviewing whether Pinocchio/CoinVault consume the shared analyzer."
---

# Tasks

## TODO

- [ ] Move `pinocchio/pkg/analysis/sessionstreamschema/analyzer.go` into `sessionstream/pkg/analysis/sessionstreamschema/analyzer.go`.
- [ ] Add `sessionstream/cmd/sessionstream-lint/main.go` using `singlechecker.Main(sessionstreamschema.Analyzer)`.
- [ ] Add/verify Sessionstream module dependencies for `golang.org/x/tools/go/analysis`.
- [ ] Add Sessionstream validation target for building and running `sessionstream-lint`.
- [ ] Update Pinocchio `Makefile` `schema-vet` to build/use `../sessionstream/cmd/sessionstream-lint`.
- [ ] Remove or wrap Pinocchio-local `pinocchio-lint` analyzer files so the rule is not duplicated.
- [ ] Add CoinVault `Makefile` `schema-vet` target that builds/uses `../sessionstream/cmd/sessionstream-lint`.
- [ ] Document `sessionstream-lint` usage in Sessionstream docs.
- [ ] Update Pinocchio/CoinVault docs to point to the Sessionstream-owned schema policy tool.
- [ ] Validate Sessionstream: `go test ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint -count=1` and `go vet -vettool=/tmp/sessionstream-lint ./pkg/... ./cmd/...`.
- [ ] Validate Pinocchio: `make schema-vet` and targeted chatapp tests.
- [ ] Validate CoinVault: `make schema-vet` and targeted webchat/projection tests.
- [ ] Decide separately whether to add stricter nested `google.protobuf.Struct` descriptor linting in a follow-up ticket.
