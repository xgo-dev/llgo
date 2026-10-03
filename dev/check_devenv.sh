#!/usr/bin/env bash
set -euo pipefail

test "${GOFLAGS:-}" = '-tags=byollvm'
[[ "$(go env GOVERSION)" == go1.27.* ]]
[[ "$(llvm-config --version)" == 22.* ]]
[[ -f "$LLGO_ROOT/runtime/go.mod" ]]

cd "$LLGO_ROOT"
go build ./cmd/llgo
./llgo version

smoke_dir="$(mktemp -d)"
trap 'rm -r "$smoke_dir"' EXIT
cat > "$smoke_dir/main.go" <<'EOF'
package main

import "fmt"

func main() { fmt.Println("LLGo dev shell works") }
EOF
[[ "$(./llgo run "$smoke_dir/main.go")" == 'LLGo dev shell works' ]]
