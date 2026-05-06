---
Title: "Playbook: Use the Sessionstream Schema Vet Tool"
Slug: "sessionstream-schema-vet-playbook"
Short: "Build and run sessionstream-lint to reject top-level Struct payload registrations in sessionstream applications."
Topics:
  - sessionstream
  - protobuf
  - schema-registry
  - go-vet
  - static-analysis
Commands:
  - sessionstream-lint
  - go vet
  - make schema-vet
Flags:
  - -vettool
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

The schema vet tool enforces one sessionstream contract: top-level payloads registered with `sessionstream.SchemaRegistry` must be concrete protobuf messages, not `*structpb.Struct`. This matters because registered payloads cross runtime code, projections, hydration, websocket transport, and frontend rendering. If the top-level payload is an arbitrary JSON object, each layer has to guess the contract.

Use this playbook when you add a new sessionstream command, backend event, UI event, or timeline entity, or when you migrate an older feature away from `google.protobuf.Struct`.

## Prerequisites

You need a Go module that imports `github.com/go-go-golems/sessionstream/pkg/sessionstream` and registers schemas through `*sessionstream.SchemaRegistry`.

In a local multi-repo workspace, the expected layout is usually:

```text
workspace/
├── sessionstream/
├── pinocchio/
└── 2026-03-16--gec-rag/
```

The shared vettool source lives in:

```text
sessionstream/cmd/sessionstream-lint
```

The analyzer implementation lives in:

```text
sessionstream/pkg/analysis/sessionstreamschema/analyzer.go
```

## Step 1: Register concrete protobuf payloads

Before running the tool, make sure new registrations use concrete generated message types.

Do this:

```go
reg.RegisterCommand("StartInference", &chatv1.StartInferenceCommand{})
reg.RegisterEvent("InferenceStarted", &chatv1.InferenceStartedEvent{})
reg.RegisterUIEvent("MessageStarted", &chatv1.ChatMessageUpdate{})
reg.RegisterTimelineEntity("ChatMessage", &chatv1.ChatMessageEntity{})
```

Do not do this:

```go
reg.RegisterEvent("InferenceStarted", &structpb.Struct{})
```

The analyzer only rejects top-level schema registrations. It does not reject nested `google.protobuf.Struct` fields inside concrete messages, because those can be valid when a single field is intentionally open-ended metadata.

## Step 2: Build the vettool from Sessionstream

From the `sessionstream` repository, build the tool directly:

```bash
cd sessionstream
go build -o /tmp/sessionstream-lint ./cmd/sessionstream-lint
```

From a downstream repository in the same workspace, build the same tool using a relative path:

```bash
cd pinocchio
go build -o /tmp/sessionstream-lint ../sessionstream/cmd/sessionstream-lint
```

The output path does not matter. `/tmp/sessionstream-lint` is convenient because the binary is build output, not source.

## Step 3: Run go vet with the custom vettool

Run the analyzer through `go vet`:

```bash
go vet -vettool=/tmp/sessionstream-lint ./cmd/... ./pkg/...
```

For a repository that keeps application code under `internal/`, use that package set instead:

```bash
go vet -vettool=/tmp/sessionstream-lint ./cmd/... ./internal/...
```

The tool reports diagnostics at the payload argument position:

```text
pkg/chatapp/plugins/example.go:42:51: sessionstream schema registrations must use concrete protobuf messages, not *structpb.Struct
```

Read that as: the registration itself is fine, but the second argument is too generic.

## Step 4: Add a Makefile target

A Makefile target makes the check easy to run locally and in CI.

For a downstream app with `cmd/` and `pkg/` packages:

```make
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint
SESSIONSTREAM_LINT_PKG ?= ../sessionstream/cmd/sessionstream-lint

schema-vet:
	go build -o $(SESSIONSTREAM_LINT) $(SESSIONSTREAM_LINT_PKG)
	go vet -vettool=$(SESSIONSTREAM_LINT) ./cmd/... ./pkg/...
```

For an app with `cmd/` and `internal/` packages:

```make
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint
SESSIONSTREAM_LINT_PKG ?= ../sessionstream/cmd/sessionstream-lint

schema-vet:
	go build -o $(SESSIONSTREAM_LINT) $(SESSIONSTREAM_LINT_PKG)
	go vet -vettool=$(SESSIONSTREAM_LINT) ./cmd/... ./internal/...
```

Run it with:

```bash
make schema-vet
```

## Step 5: Fix failures by naming the payload

When the analyzer flags a registration, resist the urge to add an allowlist. The useful fix is usually to name the concept.

Start from the registration:

```go
reg.RegisterUIEvent("ChatAgentModePreviewUpdated", &structpb.Struct{})
```

Ask what the payload represents. In this case it is not arbitrary JSON. It is a preview of an agent mode switch, so it deserves a message:

```proto
message AgentModePreviewUpdate {
  string message_id = 1;
  string candidate_mode = 2;
  string analysis = 3;
  string parse_state = 4;
  bool preview = 5;
}
```

Regenerate code, then replace the registration:

```go
reg.RegisterUIEvent("ChatAgentModePreviewUpdated", &chatappv1.AgentModePreviewUpdate{})
```

Finally replace map construction with generated message construction:

```go
return runtime.Publish(ctx, agentModePreviewEventName, &chatappv1.AgentModePreviewUpdate{
    MessageId:     runtime.MessageID,
    CandidateMode: ev.CandidateMode,
    Analysis:      ev.Analysis,
    ParseState:    ev.ParseState,
    Preview:       true,
})
```

## Complete example

A complete local validation flow for Pinocchio looks like this:

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/pinocchio
make schema-vet
go test ./pkg/chatapp ./pkg/chatapp/plugins ./cmd/web-chat -count=1
```

A complete local validation flow for CoinVault looks like this:

```bash
cd /home/manuel/workspaces/2026-05-02/use-sessionstream-coinvault/2026-03-16--gec-rag
make schema-vet
go test ./internal/webchat ./internal/projectionlookup ./internal/projectionblocks -count=1
```

The vet check prevents generic top-level payload contracts from returning. The tests then verify that the typed payloads still behave correctly in the application.

## Running outside a workspace

If the downstream repository is not next to a local `sessionstream` checkout, install the vettool from the module:

```bash
go install github.com/go-go-golems/sessionstream/cmd/sessionstream-lint@latest
go vet -vettool="$(go env GOPATH)/bin/sessionstream-lint" ./...
```

Use this pattern in CI when the build should consume a released version of `sessionstream-lint` instead of a sibling checkout.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| `stat ../sessionstream/cmd/sessionstream-lint: directory not found` | The downstream repo is not next to a local Sessionstream checkout. | Install the tool with `go install github.com/go-go-golems/sessionstream/cmd/sessionstream-lint@latest` or adjust `SESSIONSTREAM_LINT_PKG`. |
| `go vet` reports `*structpb.Struct` in a schema registration. | A command, backend event, UI event, or timeline entity is using an arbitrary top-level JSON payload. | Define a concrete protobuf message, regenerate code, and register that message instead. |
| The analyzer does not flag a `Struct` field inside a concrete message. | Nested `Struct` fields are intentionally out of scope for this analyzer. | Keep nested `Struct` only when the field is deliberately open-ended metadata; use a separate descriptor lint if nested policy is needed. |
| `go vet` fails in unrelated packages. | The package set is too broad for the repository's current state. | Narrow the Makefile target to application packages first, then broaden after cleanup. |
| CI cannot build the sibling workspace path. | CI checks out only one repository. | Use `go install ...@<version>` or configure CI to check out Sessionstream as a sibling. |

## See Also

- `README.md` — the Sessionstream landing page and core model.
- `pkg/analysis/sessionstreamschema/analyzer.go` — the analyzer implementation.
- `cmd/sessionstream-lint/main.go` — the vettool entry point.
- `ttmp/2026/05/06/SS-SCHEMA-VET--move-sessionstream-schema-vet-analyzer-into-sessionstream/design/01-sessionstream-schema-vet-analyzer-migration-plan.md` — design notes for moving the analyzer into Sessionstream.
- `proto/sessionstream/v1/transport.proto` — websocket transport frame schema.
