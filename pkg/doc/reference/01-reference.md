---
Title: "Sessionstream Reference"
Slug: "sessionstream-reference"
Short: "Reference for Sessionstream packages, core types, schema rules, transport semantics, and development commands."
Topics:
  - sessionstream
  - reference
  - api
  - packages
  - protobuf
  - transport
Commands:
  - sessionstream-lint
  - sessionstream-systemlab
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This reference summarizes the public pieces of the `sessionstream` repository. Use it when you know the concept and need names, packages, commands, or contracts.

## Module

```text
github.com/go-go-golems/sessionstream
```

Core import path:

```go
import sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
```

Embedded documentation import path:

```go
import sessionstreamdoc "github.com/go-go-golems/sessionstream/pkg/doc"
```

## Repository layout

| Path | Purpose |
|---|---|
| `pkg/sessionstream` | Core framework APIs: hub, commands, events, projections, hydration interfaces. |
| `pkg/sessionstream/hydration/sqlite` | SQLite hydration store implementation. |
| `pkg/sessionstream/transport/ws` | WebSocket transport adapter. |
| `pkg/sessionstream/pb/proto/sessionstream/v1` | Generated Go bindings for transport frames. |
| `proto/sessionstream/v1/transport.proto` | WebSocket transport protobuf schema. |
| `pkg/analysis/sessionstreamschema` | Go analyzer that rejects top-level `Struct` schema registrations. |
| `cmd/sessionstream-lint` | Vettool command for the schema analyzer. |
| `pkg/doc` | Embedded Glazed help entries for downstream CLIs. |
| `cmd/sessionstream-systemlab` | Browser-based lab/reference application. |
| `examples/chatdemo` | Small runnable chat-style example. |
| `ttmp` | Design docs, tickets, and implementation diaries. |

## Core types

### Hub

The `Hub` is the command/event/projection coordinator.

Important functions and options:

```go
func NewHub(opts ...HubOption) (*Hub, error)
func WithSchemaRegistry(r *SchemaRegistry) HubOption
func WithHydrationStore(s HydrationStore) HubOption
func WithUIFanout(f UIFanout) HubOption
func WithProjectionPolicies(p ProjectionPolicies) HubOption

func (h *Hub) RegisterCommand(name string, handler CommandHandler) error
func (h *Hub) RegisterUIProjection(p UIProjection) error
func (h *Hub) RegisterTimelineProjection(p TimelineProjection) error
func (h *Hub) Submit(ctx context.Context, sid SessionId, name string, payload proto.Message) error
func (h *Hub) Snapshot(ctx context.Context, sid SessionId) (Snapshot, error)
```

### CommandHandler

```go
type CommandHandler func(
    ctx context.Context,
    cmd Command,
    sess *Session,
    pub EventPublisher,
) error
```

Handlers publish backend events through `EventPublisher`. They should not return UI state directly.

### EventPublisher

```go
type EventPublisher interface {
    Publish(ctx context.Context, ev Event) error
}
```

The publisher validates event payload types, assigns ordinals, applies projections, stores timeline state, and fans out UI events according to hub configuration.

### Event

```go
type Event struct {
    Name      string
    Payload   proto.Message
    SessionId SessionId
    Ordinal   uint64
}
```

Backend events are canonical records. Projections derive other views from them.

### UIEvent

```go
type UIEvent struct {
    Name    string
    Payload proto.Message
}
```

UI events are live client-facing projected events.

### TimelineEntity

```go
type TimelineEntity struct {
    Kind             string
    Id               string
    CreatedOrdinal   uint64
    LastEventOrdinal uint64
    Payload          proto.Message
    Tombstone        bool
}
```

Timeline entities are durable projected state. Hydration stores use them to build snapshots.

### TimelineView

```go
type TimelineView interface {
    Get(kind, id string) (TimelineEntity, bool)
    List(kind string) []TimelineEntity
    Ordinal() uint64
}
```

Projections receive a read-only timeline view so they can update existing entities instead of blindly replacing state.

## Schema registry

```go
func NewSchemaRegistry() *SchemaRegistry
func (r *SchemaRegistry) RegisterCommand(name string, msg proto.Message) error
func (r *SchemaRegistry) RegisterEvent(name string, msg proto.Message) error
func (r *SchemaRegistry) RegisterUIEvent(name string, msg proto.Message) error
func (r *SchemaRegistry) RegisterTimelineEntity(kind string, msg proto.Message) error
```

Rules:

- Names must be stable logical names.
- Payload prototypes must be concrete protobuf messages.
- Do not use top-level `*structpb.Struct` registrations.
- Nested `google.protobuf.Struct` fields are allowed when intentionally scoped inside a concrete message.

## Projection interfaces

```go
type UIProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]UIEvent, error)
}

type TimelineProjection interface {
    Project(ctx context.Context, ev Event, sess *Session, view TimelineView) ([]TimelineEntity, error)
}
```

Function adapters are available:

```go
sessionstream.UIProjectionFunc(func(ctx context.Context, ev sessionstream.Event, sess *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
    return nil, nil
})

sessionstream.TimelineProjectionFunc(func(ctx context.Context, ev sessionstream.Event, sess *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
    return nil, nil
})
```

## WebSocket transport

Transport schemas live in:

```text
proto/sessionstream/v1/transport.proto
```

Important frame types:

| Frame | Direction | Purpose |
|---|---|---|
| `ClientFrame.subscribe` | client -> server | Subscribe to a session. |
| `ClientFrame.unsubscribe` | client -> server | Stop receiving a session. |
| `ClientFrame.ping` | client -> server | Keepalive. |
| `ServerFrame.hello` | server -> client | Connection greeting. |
| `ServerFrame.snapshot` | server -> client | Current hydrated timeline state. |
| `ServerFrame.uiEvent` | server -> client | Live UI event derived from a backend event. |
| `ServerFrame.error` | server -> client | Transport or subscription error. |

Payloads are encoded as protobuf JSON. Snapshot and UI payloads use `google.protobuf.Any`; clients need matching application schema knowledge to unpack or interpret them.

## Ordinal semantics

| Field | Meaning |
|---|---|
| Backend event `Ordinal` | Order assigned to a canonical backend event. |
| `Snapshot.snapshotOrdinal` | Highest timeline ordinal represented by the snapshot. |
| `SnapshotEntity.createdOrdinal` | Ordinal that created the entity. |
| `SnapshotEntity.lastEventOrdinal` | Latest event ordinal that updated the entity. |
| `UiEventFrame.eventOrdinal` | Backend event ordinal that produced the live UI event. |

Protojson renders `uint64` values as strings. Browser code should not force them through JavaScript `number` if precision matters.

## Commands

### sessionstream-lint

Build the schema vettool:

```bash
go build -o /tmp/sessionstream-lint ./cmd/sessionstream-lint
```

Run it through `go vet`:

```bash
go vet -vettool=/tmp/sessionstream-lint ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint
```

Downstream workspace usage:

```bash
go build -o /tmp/sessionstream-lint ../sessionstream/cmd/sessionstream-lint
go vet -vettool=/tmp/sessionstream-lint ./cmd/... ./pkg/...
```

### sessionstream-systemlab

Run the browser-based lab app:

```bash
make systemlab-run
# or directly:
go run ./cmd/sessionstream-systemlab serve --addr :8091
```

Browse embedded CLI help entries:

```bash
go run ./cmd/sessionstream-systemlab help sessionstream-user-guide
go run ./cmd/sessionstream-systemlab help sessionstream-reference
```

Default URL:

```text
http://localhost:8091/
```

Embedded help docs are served by Systemlab under:

```text
http://localhost:8091/docs/
```

## Development commands

```bash
make test
make build
make lint
make schema-vet
make systemlab-build
make systemlab-run
make boundary-check
```

Most repository Makefile targets use `GOWORK=off` so the module is tested as an external consumer would see it.

## Embedded docs package

Downstream CLIs can load Sessionstream help entries into a Glazed help system:

```go
helpSystem := help.NewHelpSystem()
if err := sessionstreamdoc.AddDocToHelpSystem(helpSystem); err != nil {
    return err
}
```

The package also exposes the embedded filesystem:

```go
docsFS := sessionstreamdoc.FS()
```

## See Also

- `sessionstream-user-guide` for the architecture narrative.
- `sessionstream-getting-started` for a first app walkthrough.
- `sessionstream-schema-vet-playbook` for vettool operation.
- `examples/chatdemo/chat.go` for runnable application code.
- `proto/sessionstream/v1/transport.proto` for websocket schema details.
