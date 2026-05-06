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

- [x] Move `pinocchio/pkg/analysis/sessionstreamschema/analyzer.go` into `sessionstream/pkg/analysis/sessionstreamschema/analyzer.go`.
- [x] Add `sessionstream/cmd/sessionstream-lint/main.go` using `singlechecker.Main(sessionstreamschema.Analyzer)`.
- [x] Add/verify Sessionstream module dependencies for `golang.org/x/tools/go/analysis`.
- [x] Add Sessionstream validation target for building and running `sessionstream-lint`.
- [x] Update Pinocchio `Makefile` `schema-vet` to build/use `../sessionstream/cmd/sessionstream-lint`.
- [x] Remove or wrap Pinocchio-local `pinocchio-lint` analyzer files so the rule is not duplicated.
- [x] Add CoinVault `Makefile` `schema-vet` target that builds/uses `../sessionstream/cmd/sessionstream-lint`.
- [x] Document `sessionstream-lint` usage in Sessionstream docs.
- [x] Update Pinocchio/CoinVault docs to point to the Sessionstream-owned schema policy tool.
- [x] Validate Sessionstream: `go test ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint -count=1`; `make schema-vet` builds/runs the tool on the analyzer package and lint command.
- [x] Validate Pinocchio: `make schema-vet` and targeted chatapp tests.
- [x] Validate CoinVault: `make schema-vet` and targeted webchat/projection tests.
- [ ] Decide separately whether to add stricter nested `google.protobuf.Struct` descriptor linting in a follow-up ticket.
- [ ] Follow up: convert existing Sessionstream tests/systemlab fixtures off top-level `*structpb.Struct` registrations, then broaden Sessionstream `schema-vet` to `./pkg/... ./cmd/...`.
