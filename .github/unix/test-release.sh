#!/usr/bin/env bash
set -euo pipefail

unset LLGO_ROOT LLVM_CONFIG
root=$(cd "${1:?installed release directory required}" && pwd -P)
test "$(cd "$(llgo env LLGO_ROOT)" && pwd -P)" = "$root"
clang="$root/crosscompile/clang"
test -x "$clang/bin/clang"
test -x "$clang/bin/clang++"
test -x "$clang/bin/ld.lld"
test -x "$clang/bin/llvm-config"
test -f "$clang/LICENSE-LLVM.txt"
test -d "$clang/lib/clang/22/include"
for directory in llvm llvm-c clang clang-c; do
    test ! -e "$clang/include/$directory"
done
test ! -e "$clang/bin/llvm-exegesis"
test -z "$(find "$clang/lib" -maxdepth 1 \( -name 'libLLVM*.a' -o -name 'libclang*.a' \) -print)"
test -z "$(find "$clang" -type l ! -exec test -e {} \; -print)"
llgo version
"$clang/bin/clang" --version
echo 'Verified Unix runtime payload and compiler startup without LLVM SDK headers/static libraries'
