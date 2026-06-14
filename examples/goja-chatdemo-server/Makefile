.DEFAULT_GOAL := smoke

WORKSPACE_ROOT := $(abspath ../../..)
GO_GO_GOJA := $(WORKSPACE_ROOT)/go-go-goja
SESSIONSTREAM := $(WORKSPACE_ROOT)/sessionstream
BIN := $(CURDIR)/dist/goja-chatdemo-server
XGOJA := cd $(GO_GO_GOJA) && go run ./cmd/xgoja
HTTP_ADDR := 127.0.0.1:18789

.PHONY: smoke doctor build serve-smoke clean

smoke: doctor build serve-smoke

doctor:
	$(XGOJA) doctor -f $(CURDIR)/xgoja.yaml

build:
	$(XGOJA) build -f $(CURDIR)/xgoja.yaml --output $(BIN) --keep-work

serve-smoke: build
	@set -eu; \
	log=$$(mktemp); \
	$(BIN) serve chatbot serve --http-listen $(HTTP_ADDR) >$$log 2>&1 & \
	pid=$$!; \
	trap 'kill $$pid >/dev/null 2>&1 || true; wait $$pid >/dev/null 2>&1 || true; rm -f $$log' EXIT; \
	cd $(SESSIONSTREAM) && go run ./examples/goja-chatdemo-server/cmd/smoke-client --addr $(HTTP_ADDR) || { cat $$log; exit 1; }; \
	curl -fsS http://$(HTTP_ADDR)/healthz | grep -q '"ok":true'; \
	curl -fsS http://$(HTTP_ADDR)/ | grep -q 'xgoja sessionstream chatbot'; \
	kill $$pid >/dev/null 2>&1 || true; \
	wait $$pid >/dev/null 2>&1 || true

clean:
	rm -rf $(CURDIR)/dist
