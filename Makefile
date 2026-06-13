.PHONY: all fmt fmt-check lint lintmax docker-lint golangci-lint-install gosec govulncheck test build build-bin boundary-check schema-vet systemlab-build systemlab-run check ci-check goreleaser ensure-svu tag-major tag-minor tag-patch release bump-go-go-golems install logcopter-generate logcopter-check glazed-lint-build glazed-lint

all: check

BINARY ?= sessionstream-systemlab
MODULE ?= github.com/go-go-golems/sessionstream
CMD_DIR ?= ./cmd/$(BINARY)

GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
SVU ?= svu
SESSIONSTREAM_LINT ?= /tmp/sessionstream-lint

boundary-check:
	@! rg -n 'github.com/go-go-golems/pinocchio/' . --glob '*.go' --glob '!ttmp/**' >/dev/null || (echo 'sessionstream must not import pinocchio packages' && exit 1)

fmt:
	GOWORK=off go fmt ./...

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	GOWORK=off GOBIN=$(dir $(GOLANGCI_LINT_BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v

lintmax: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v --max-same-issues=100

fmt-check: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) fmt --diff

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

build-bin:
	@mkdir -p ./dist
	GOWORK=off go build -o ./dist/$(BINARY) $(CMD_DIR)

schema-vet:
	GOWORK=off go build -o $(SESSIONSTREAM_LINT) ./cmd/sessionstream-lint
	GOWORK=off go vet -vettool=$(SESSIONSTREAM_LINT) ./pkg/analysis/sessionstreamschema ./cmd/sessionstream-lint

systemlab-build:
	@mkdir -p .bin
	GOWORK=off go build -o ./.bin/$(BINARY) $(CMD_DIR)

systemlab-run:
	GOWORK=off go run $(CMD_DIR) serve

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

ensure-svu:
	@command -v $(SVU) >/dev/null || (echo 'svu is required for tag/release targets: go install github.com/caarlos0/svu/v3@latest' && exit 1)

tag-major: ensure-svu
	git tag $$($(SVU) major)

tag-minor: ensure-svu
	git tag $$($(SVU) minor)

tag-patch: ensure-svu
	git tag $$($(SVU) patch)

release: ensure-svu
	git push origin --tags
	GOWORK=off GOPROXY=proxy.golang.org go list -m $(MODULE)@$$($(SVU) current)

install:
	GOWORK=off go install $(CMD_DIR)

check: boundary-check schema-vet logcopter-check glazed-lint test build

ci-check: fmt-check lint check

logcopter-generate:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.sessionstream -strip-prefix github.com/go-go-golems/sessionstream ./cmd/... ./pkg/...

logcopter-check:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.sessionstream -strip-prefix github.com/go-go-golems/sessionstream -check ./cmd/... ./pkg/...

GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_VERSION ?= latest
GLAZED_LINT_FLAGS ?=

glazed-lint-build:
	@if [ -n "$(GLAZED_VERSION)" ] && [ "$(GLAZED_VERSION)" != "(devel)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION) || { \
			echo "Falling back to $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION)"; \
			GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION); \
		}; \
	else \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_VERSION); \
	fi

glazed-lint: glazed-lint-build
	GOWORK=off $(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) ./...

bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest"; done; \
	fi
	GOWORK=off go mod tidy
