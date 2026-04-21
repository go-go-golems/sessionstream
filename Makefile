.PHONY: fmt lint lintmax gosec govulncheck test build boundary-check check

boundary-check:
	@! rg -n 'github.com/go-go-golems/pinocchio/' . --glob '*.go' --glob '!ttmp/**' >/dev/null || (echo 'sessionstream must not import pinocchio packages' && exit 1)

fmt:
	GOWORK=off go fmt ./...

lint:
	GOWORK=off golangci-lint run -v

lintmax:
	GOWORK=off golangci-lint run -v --max-same-issues=100

gosec:
	GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306 -exclude-dir=.history ./...

govulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	GOWORK=off go test ./...

build:
	GOWORK=off go generate ./...
	GOWORK=off go build ./...

check: boundary-check test build
