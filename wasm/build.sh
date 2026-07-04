#!/usr/bin/env bash
# Build the chalk WebAssembly adapter.
set -euo pipefail
cd "$(dirname "$0")"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" ./wasm_exec.js
GOOS=js GOARCH=wasm go build -o chalk.wasm .
echo "built chalk.wasm ($(du -h chalk.wasm | cut -f1))"
