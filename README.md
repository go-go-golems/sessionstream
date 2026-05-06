# sessionstream

`sessionstream` is the standalone home for a generic session-based streaming substrate.

The goal of this repository is to host the reusable parts of that architecture:

- session-centered routing (`SessionId` as the primary key),
- typed commands and backend events,
- sibling UI and timeline projections,
- hydration stores,
- transport adapters,
- protobuf transport schemas for websocket clients,
- framework-oriented examples and labs.

## What belongs here

This repository is intended to own the framework layer:

- the core sessionstream package under `pkg/sessionstream`,
- generic stores and transports under `pkg/sessionstream/...`,
- protobuf schemas under `proto/sessionstream/v1/` and generated bindings under `pkg/sessionstream/pb/proto/sessionstream/v1/`,
- small demo/example applications that prove the substrate API,
- framework-oriented Systemlab content,
- repository-local design and implementation tickets under `ttmp/`.

## What does not belong here

This repository is **not** intended to own product-specific application behavior such as:

- pinocchio profile/runtime policy,
- `cmd/web-chat` HTTP edge behavior,
- pinocchio middleware implementations like `agentmode`,
- legacy webchat compatibility routes.

Those stay in downstream consumer repositories such as `pinocchio`.

## Current status

The repository is currently in bootstrap mode.

The initial repository planning tickets live under `ttmp/` and capture the extraction plan, intern-facing architecture guides, and implementation diaries.

## Transport and ordinal contract

The websocket adapter uses protobuf-defined frames from `proto/sessionstream/v1/transport.proto`. The wire format is protobuf JSON so browsers can consume it over normal websocket text messages while still getting a typed contract:

- clients send `ClientFrame` oneofs such as `subscribe`, `unsubscribe`, `ping`, and `pong`;
- servers send `ServerFrame` oneofs such as `hello`, `snapshot`, `subscribed`, `uiEvent`, and `error`;
- UI event and snapshot payloads are `google.protobuf.Any`, so consumers must register application payload schemas before unpacking;
- `uint64` ordinals appear as JSON strings under protojson and should not be coerced through JavaScript `number` if precision matters.

Ordinals have distinct meanings at different layers:

- backend events carry an event ordinal assigned by the consumer or hub path;
- `Snapshot.snapshotOrdinal` / `SnapshotFrame.snapshotOrdinal` is the highest timeline ordinal materialized in the hydration store;
- each snapshot entity carries `createdOrdinal` and `lastEventOrdinal` so hydrated timelines can preserve display order instead of sorting only by kind/id;
- live `UiEventFrame.eventOrdinal` identifies the backend event that produced that UI event.

`SubscribeRequest.sinceSnapshotOrdinal` is currently advisory. The reference websocket server still sends the current snapshot first and then future live UI events; it does not silently replay missed UI events.

## Development

Useful commands:

```bash
make test
make build
make lint
make schema-vet
make goreleaser   # local single-target snapshot release into dist/
```

## Schema vet analyzer

`sessionstream` owns the shared `sessionstream-lint` vettool for schema registration policy. It rejects top-level `*structpb.Struct` payloads registered through `sessionstream.SchemaRegistry`; use concrete, feature-owned protobuf messages for `RegisterCommand`, `RegisterEvent`, `RegisterUIEvent`, and `RegisterTimelineEntity` payloads. Nested `google.protobuf.Struct` fields are still allowed inside concrete messages when intentionally used for open-ended metadata.

From this repository:

```bash
go build -o /tmp/sessionstream-lint ./cmd/sessionstream-lint
go vet -vettool=/tmp/sessionstream-lint ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint
```

Downstream workspace consumers can build the shared tool and run it locally:

```bash
go build -o /tmp/sessionstream-lint ../sessionstream/cmd/sessionstream-lint
go vet -vettool=/tmp/sessionstream-lint ./cmd/... ./pkg/...
```

Outside a workspace, install the command from the module and use it as a vettool:

```bash
go install github.com/go-go-golems/sessionstream/cmd/sessionstream-lint@latest
go vet -vettool="$(go env GOPATH)/bin/sessionstream-lint" ./...
```

Release tags are cut with [`svu`](https://github.com/caarlos0/svu) and published by the `release` GitHub Actions workflow:

```bash
make tag-patch    # or tag-minor / tag-major
make release      # pushes tags and primes the Go module proxy
```

If you are working on repo-local planning/docs:

```bash
docmgr status --summary-only
```

## Roadmap

1. bootstrap the repository cleanly,
2. move the generic substrate out of `pinocchio`,
3. move framework-oriented examples and labs,
4. keep downstream consumers importing the library from `github.com/go-go-golems/sessionstream/pkg/sessionstream` and sibling `pkg/sessionstream/...` subpackages.
