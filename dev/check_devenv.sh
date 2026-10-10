#!/usr/bin/env bash
set -euo pipefail

: "${LLGO_ROOT:?enter the development environment with 'mise -C dev en' or 'pixi shell --manifest-path dev/pixi.toml'}"
test "${GOFLAGS:-}" = '-tags=byollvm'
go_version="$(go env GOVERSION)"
required_go_version="go$(cat "$LLGO_ROOT/.go-version")"
[[ "${go_version%.*}" == "${required_go_version%.*}" ]]
[[ "$(llvm-config --version)" == 22.* ]]
[[ -f "$LLGO_ROOT/runtime/go.mod" ]]

cd "$LLGO_ROOT"
go run ./cmd/llgo version
go build ./cmd/llgo
./llgo version

smoke_dir="$(mktemp -d)"
trap 'rm -rf "$smoke_dir"' EXIT
cat > "$smoke_dir/main.go" <<'EOF'
package main

import "fmt"

func main() { fmt.Println("LLGo dev shell works") }
EOF
output=$(./llgo run "$smoke_dir/main.go")
[[ "$output" == 'LLGo dev shell works' ]]
output=$(./llgo run ./dev/_devenv_smoke)
[[ "$output" == 'LLGo native dependencies work' ]]
echo 'LLGo development environment checks passed'
