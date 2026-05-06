---
Title: Move sessionstream schema vet analyzer into sessionstream
Ticket: SS-SCHEMA-VET
Status: active
Topics:
    - sessionstream
    - go-vet
    - protobuf
    - chatapp
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/sessionstream/schema.go
      Note: SchemaRegistry API whose registration contract the analyzer enforces.
    - Path: ttmp/2026/05/06/2026-03-16--gec-rag/Makefile
      Note: CoinVault Makefile should gain a schema-vet target using sessionstream-lint.
    - Path: ttmp/2026/05/06/pinocchio/Makefile
      Note: Current Pinocchio schema-vet target to rewire to the shared tool.
    - Path: ttmp/2026/05/06/pinocchio/cmd/tools/pinocchio-lint/main.go
      Note: Current vettool wrapper to replace with sessionstream-lint usage.
    - Path: ttmp/2026/05/06/pinocchio/pkg/analysis/sessionstreamschema/analyzer.go
      Note: Current Pinocchio-local analyzer implementation to migrate into Sessionstream.
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-06T17:15:11.544805823-04:00
WhatFor: ""
WhenToUse: ""
---


# Move sessionstream schema vet analyzer into sessionstream

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- sessionstream
- go-vet
- protobuf
- chatapp

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
