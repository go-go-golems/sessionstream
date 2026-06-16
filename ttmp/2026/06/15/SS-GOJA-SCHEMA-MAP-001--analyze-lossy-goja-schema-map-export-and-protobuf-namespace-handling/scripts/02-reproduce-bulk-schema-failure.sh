#!/usr/bin/env bash
set -euo pipefail

# Reproduce the bulk ss.schemas({...}) failure without modifying repository files.
# The temporary test proves two related issues:
#   1. lower-case SchemaMap keys (commands/events/uiEvents/entities) do not populate schemaInput via goja.ExportTo;
#   2. even if callers use capitalized Commands to match Go struct fields, generated namespace objects are exported to map[string]any and lose the hidden protogoja prototype token.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
cd "$REPO_ROOT"

TMP_TEST="pkg/js/modules/sessionstream/schema_bulk_tmp_test.go"
cleanup() { rm -f "$TMP_TEST"; }
trap cleanup EXIT

cat > "$TMP_TEST" <<'GOEOF'
package sessionstream

import (
	"testing"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	"github.com/stretchr/testify/require"
)

func TestTmpBulkSchemaLowercaseNamespaceFails(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	_, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		const schemas = ss.schemas({
		  commands: { ChatStartInference: pb.StartInferenceCommand },
		});
		const hub = ss.hub({ schemas });
		hub.command("ChatStartInference", () => {});
		hub.submit("s-1", "ChatStartInference", pb.StartInferenceCommand.builder().prompt("typed").build());
	`)
	require.NoError(t, err)
}

func TestTmpBulkSchemaCapitalizedNamespaceFailsAfterExportToMap(t *testing.T) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	Register(reg, Options{})
	require.NoError(t, chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""))
	reg.Enable(vm)
	_, err := vm.RunString(`
		const ss = require("sessionstream");
		const pb = require("sessionstream.examples.chatdemo.v1");
		ss.schemas({
		  Commands: { ChatStartInference: pb.StartInferenceCommand },
		});
	`)
	require.NoError(t, err)
}
GOEOF

go test ./pkg/js/modules/sessionstream \
  -run 'TestTmpBulkSchema(LowercaseNamespaceFails|CapitalizedNamespaceFailsAfterExportToMap)' \
  -count=1
