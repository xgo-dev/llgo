#!/usr/bin/env bash
set -euxo pipefail

# Run outside the checkout without LLGO_ROOT or a prepopulated build cache.
unset LLGO_ROOT LLVM_CONFIG
export LLGO_BUILD_CACHE=0
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$work"

root=$(llgo env LLGO_ROOT)
test "$root" = /usr/lib/llgo
test "$(readlink /usr/bin/llgo)" = /usr/lib/llgo/bin/llgo
[[ "$(llgo env LLGO_LLVM_VERSION)" == 23.* ]]
test -f "$root/runtime/go.mod"
test -f "$root/targets/esp32.json"
test -x "$root/crosscompile/clang/bin/clang"
test -f "$root/crosscompile/clang/LICENSE-LLVM.txt"
llgo version

cat > go.mod <<'MODULE'
module example.com/llgo-release-smoke

go 1.20
MODULE
cat > main.go <<'GO'
package main

import (
	"fmt"
	_ "unsafe"
)

//go:linkname abs C.abs
func abs(value int32) int32

func main() {
	done := make(chan string)
	go func() { done <- fmt.Sprintf("LLGo release: %d", abs(-42)) }()
	fmt.Println(<-done)
}
GO
llgo build -o hello .
test "$(./hello)" = 'LLGo release: 42'
