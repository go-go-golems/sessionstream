#!/usr/bin/env bash
set -euo pipefail

# Inventory possible lossy JavaScript/Go conversion boundaries in sessionstream.
# This focuses on ExportTo, map[string]any, map[string]interface{}, and Value.Export()
# usages in JavaScript-facing packages/examples.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
cd "$REPO_ROOT"

rg -n \
  "ExportTo\\(|map\\[string\\]any|map\\[string\\]interface\\{|\\.Export\\(\\)" \
  pkg/js/modules/sessionstream \
  pkg/js/modules/chatdemo \
  examples/goja-chatdemo \
  examples/goja-chatdemo-server \
  examples/goja-redis-chatdemo-server \
  -S
