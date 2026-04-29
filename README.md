# sessionstream

`sessionstream` is the standalone home for a generic session-based streaming substrate.

The goal of this repository is to host the reusable parts of that architecture:

- session-centered routing (`SessionId` as the primary key),
- typed commands and backend events,
- sibling UI and timeline projections,
- hydration stores,
- transport adapters,
- framework-oriented examples and labs.

## What belongs here

This repository is intended to own the framework layer:

- the core sessionstream package under `pkg/sessionstream`,
- generic stores and transports under `pkg/sessionstream/...`,
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

## Development

Useful commands:

```bash
make test
make build
make lint
make goreleaser   # local single-target snapshot release into dist/
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
