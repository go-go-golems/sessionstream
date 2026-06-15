.DEFAULT_GOAL := smoke

WORKSPACE_ROOT := $(abspath ../../..)
GO_GO_GOJA := $(WORKSPACE_ROOT)/go-go-goja
SESSIONSTREAM := $(WORKSPACE_ROOT)/sessionstream
XGOJA := cd $(GO_GO_GOJA) && go run ./cmd/xgoja
HTTP_ADDR ?= 127.0.0.1:18889
REDIS_ADDR ?= 127.0.0.1:6379
BUS_TOPIC ?= sessionstream.chatdemo.redis

.PHONY: doctor generate build compose-up compose-down serve smoke

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

generate:
	$(XGOJA) generate -f $(CURDIR)/xgoja.yaml --output $(CURDIR)/internal/xgojaruntime --clean

build: generate
	cd $(SESSIONSTREAM) && go build ./examples/goja-redis-chatdemo-server/cmd/redis-host

compose-up:
	docker compose -f $(CURDIR)/docker-compose.yml up -d redis

compose-down:
	docker compose -f $(CURDIR)/docker-compose.yml down

serve: generate
	cd $(SESSIONSTREAM) && go run ./examples/goja-redis-chatdemo-server/cmd/redis-host serve chatbot serve --http-listen $(HTTP_ADDR)

smoke: doctor compose-up generate
	@set -eu; \
	cd $(SESSIONSTREAM); \
	log1=$$(mktemp); log2=$$(mktemp); \
	go run ./examples/goja-redis-chatdemo-server/cmd/redis-host serve chatbot serve --http-listen 127.0.0.1:18889 >$$log1 2>&1 & \
	pid1=$$!; \
	go run ./examples/goja-redis-chatdemo-server/cmd/redis-host serve chatbot serve --http-listen 127.0.0.1:18890 >$$log2 2>&1 & \
	pid2=$$!; \
	trap 'kill $$pid1 $$pid2 >/dev/null 2>&1 || true; wait $$pid1 $$pid2 >/dev/null 2>&1 || true; rm -f $$log1 $$log2' EXIT; \
	sleep 2; \
	go run ./examples/goja-chatdemo-server/cmd/smoke-client --post-addr 127.0.0.1:18889 --ws-addr 127.0.0.1:18890 || { cat $$log1; cat $$log2; exit 1; }; \
	curl -fsS http://127.0.0.1:18890/healthz | grep -q '"ok":true'; \
	kill $$pid1 $$pid2 >/dev/null 2>&1 || true; \
	wait $$pid1 $$pid2 >/dev/null 2>&1 || true
