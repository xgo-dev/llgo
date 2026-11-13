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
test -f "$root/runtime/go.mod"
test -f "$root/targets/esp32.json"
payloads=("$root"/crosscompile/clang-esp-*.tar.xz)
test "${#payloads[@]}" = 1
test -s "${payloads[0]}"
test -f "$root/LICENSES/XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt"
ldd "$root/bin/llgo" > loader.txt
cat loader.txt
if grep -q 'not found' loader.txt; then exit 1; fi
grep -E 'libLLVM\.so\.22\.1 => /' loader.txt
# Debian installs versioned LLVM tools; Fedora provides unversioned commands.
if command -v llvm-config-22 >/dev/null; then
    LLVM_CONFIG=$(command -v llvm-config-22)
    export LLVM_CONFIG
fi
llvm_config=${LLVM_CONFIG:-llvm-config}
[[ "$("$llvm_config" --version)" == 22.* ]]
[[ "$(llgo env LLGO_LLVM_VERSION)" == 22.* ]]
llvm_bin=$("$llvm_config" --bindir)
"$llvm_bin/llvm-nm" --undefined-only --dynamic "$root/bin/llgo" > imports.txt
if grep -q LLVMInitializeXtensa imports.txt; then exit 1; fi
"$llvm_bin/clang++" --version
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
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/std"
)

//go:linkname abs C.abs
func abs(value int32) int32

func main() {
	done := make(chan string)
	go func() { done <- fmt.Sprintf("LLGo release: %d", abs(-42)) }()
	fmt.Println(<-done)
	c.Printf(std.Str("C++ dependency works\n").CStr())
}
GO
go mod edit -require=github.com/goplus/lib@v0.6.3
go mod tidy
llgo build -o hello .
test "$(./hello)" = $'LLGo release: 42\nC++ dependency works'

# The native compiler continues to use the system LLVM. Embedded builds unpack
# the pinned archive shipped by this DEB/RPM into a fresh writable user cache.
unset LLVM_CONFIG CC CXX CGO_CPPFLAGS CGO_CFLAGS CGO_CXXFLAGS CGO_LDFLAGS
export XDG_CACHE_HOME="$work/empty cache"
host=$(uname -m)
case "$host" in
    x86_64) target=x86_64-linux-gnu ;;
    aarch64) target=aarch64-linux-gnu ;;
    *) echo "Unsupported package host: $host" >&2; exit 1 ;;
esac
asset=$(basename "${payloads[0]}")
version=${asset#clang-esp-}
version=${version%-"$target".tar.xz}
cached="$XDG_CACHE_HOME/llgo/crosscompile/esp-clang-$version-$target"
mkdir firmware
printf 'package main\nfunc main() {}\n' > firmware/main.go
count=0
for board in esp32-coreboard-v2 esp32c3; do
    for mode in default full; do
        flags=()
        if [[ "$mode" == full ]]; then flags+=('-lto=full'); fi
        output="$work/firmware/$board-$mode.out"
        log="$work/$board-$mode.log"
        if ! llgo build -x "${flags[@]}" -target "$board" -o "$output" ./firmware > "$log" 2>&1; then
            cat "$log" >&2
            exit 1
        fi
        if grep -E 'ESP Clang not found|Downloading ESP Clang' "$log"; then exit 1; fi
        grep -F "$cached/bin/" "$log" > /dev/null
        if [[ "$count" == 0 ]]; then
            grep -F "Using bundled ESP Clang archive: ${payloads[0]}" "$log"
        else
            if grep -F 'Using bundled ESP Clang archive:' "$log"; then exit 1; fi
        fi
        "$llvm_bin/llvm-readobj" --file-headers "$output.elf" > firmware-header.txt
        if [[ "$board" == esp32c3 ]]; then
            grep -F EM_RISCV firmware-header.txt
        else
            grep -F EM_XTENSA firmware-header.txt
        fi
        count=$((count + 1))
        echo "Passed installed package ESP firmware: $board/$mode"
    done
done
grep -Fx "payload_version=$version" "$cached/LLGO-LLVM-MANIFEST.txt"
grep -Fx "host_target=$target" "$cached/LLGO-LLVM-MANIFEST.txt"
