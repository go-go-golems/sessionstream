# Agent Guidelines for sessionstream

## Repository purpose

`sessionstream` is a framework/library repository.

The main architectural rule is:

- keep the repository focused on the generic session-based streaming substrate,
- do not pull downstream product behavior into the framework just because it currently exists in the source repository we are extracting from.

If a feature is clearly pinocchio-specific, keep it downstream.

## Build commands

- Test: `go test ./...`
- Build: `go build ./...`
- Run a single test: `go test ./path/to/pkg -run TestName -count=1`
- Format: `go fmt ./...`
- Lint: `golangci-lint run -v` or `make lint`

## Repository structure

- root package: the generic `sessionstream` substrate
- `hydration/`: optional store implementations
- `transport/`: optional transport adapters
- `examples/`: small framework-owned examples only
- `cmd/`: framework-owned tools/apps only
- `ttmp/`: repo-local ticket documentation and diaries

## Documentation workflow

- Tickets for this repository live under `sessionstream/ttmp`.
- Keep design docs, tasks, changelogs, and diaries up to date as work progresses.
- Prefer focused, evidence-backed tickets over vague umbrella notes.

## Design guardrails

- Do not add backwards-compatibility layers unless explicitly requested.
- Do not import downstream product packages into the framework if a generic seam would do.
- Prefer small interfaces and clean ownership boundaries.
- If a feature cannot be made honestly generic, leave it in the consumer repo.

## Go guidelines

- Use contexts when appropriate.
- Use `var _ Interface = (*Type)(nil)` assertions for interface implementations where helpful.
- Use `github.com/pkg/errors` only if the repo already standardizes on it; otherwise prefer the standard library for new code unless consistency requires otherwise.
- When adding goroutines, prefer `errgroup` for coordinated lifecycle management.

## Debugging guideline

If the work starts turning into repeated patching without a clear design, stop and reassess the boundary first.
