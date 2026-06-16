#!/usr/bin/env bash
set -euo pipefail

# Inspect how goja.ExportTo converts generated protobuf namespace objects when
# exporting the bulk schema input into schemaInput. The program is generated in
# a temp file so the repository source tree stays clean.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
cd "$REPO_ROOT"

TMP_GO="$(mktemp /tmp/sessionstream-inspect-export-XXXXXX.go)"
cleanup() { rm -f "$TMP_GO"; }
trap cleanup EXIT

cat > "$TMP_GO" <<'GOEOF'
package main

import (
	"fmt"
	"sort"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	ssm "github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream"
)

type schemaInput struct {
	Commands map[string]any `json:"commands"`
	Events   map[string]any `json:"events"`
	UIEvents map[string]any `json:"uiEvents"`
	Entities map[string]any `json:"entities"`
}

func inspect(js string) {
	vm := goja.New()
	reg := noderequire.NewRegistry()
	ssm.Register(reg, ssm.Options{})
	if err := chatdemov1.RegisterGojaBuilderFileChatProtoModule(reg, ""); err != nil {
		panic(err)
	}
	reg.Enable(vm)
	value, err := vm.RunString(js)
	if err != nil {
		panic(err)
	}
	var input schemaInput
	err = vm.ExportTo(value, &input)
	fmt.Printf("ExportTo error: %v\n", err)
	fmt.Printf("commands nil: %v\n", input.Commands == nil)
	for name, spec := range input.Commands {
		fmt.Printf("command %q exported as %T\n", name, spec)
		if m, ok := spec.(map[string]any); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			fmt.Printf("  keys: %v\n", keys)
			fmt.Printf("  typeName: %v (%T)\n", m["typeName"], m["typeName"])
			fmt.Printf("  type: %v (%T)\n", m["type"], m["type"])
		}
	}
}

func main() {
	fmt.Println("CASE 1: lower-case SchemaMap key used by TypeScript contract")
	inspect(`const pb = require("sessionstream.examples.chatdemo.v1"); ({ commands: { ChatStartInference: pb.StartInferenceCommand } })`)
	fmt.Println()
	fmt.Println("CASE 2: capitalized Go struct field name")
	inspect(`const pb = require("sessionstream.examples.chatdemo.v1"); ({ Commands: { ChatStartInference: pb.StartInferenceCommand } })`)
}
GOEOF

go run "$TMP_GO"
