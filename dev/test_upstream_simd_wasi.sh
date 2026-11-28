#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
: "${LLGO:=llgo}"
export GOOS=wasip1 GOARCH=wasm GOEXPERIMENT=simd CGO_ENABLED=0 LC_ALL=C
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
package=simd/archsimd/internal/simd_test
awk '/^Test/ {print $1}' dev/simd/upstream-archsimd-wasm.txt | sort >"$work/expected"

go test -c -o "$work/gc.wasm" "$package"
"$LLGO" test -target wasi -O2 -c -pclntab=none -o "$work/llgo.wasm" "$package"
for compiler in gc llgo; do
  if [[ $compiler == gc ]]; then
    runner=(wasmtime "$work/gc.wasm")
  else
    runner=(wasmer run --enable-exceptions --enable-simd --stack-size=1048576 "$work/llgo.wasm" --)
  fi
  "${runner[@]}" -test.v -test.count=1 -test.timeout=5m | tee "$work/$compiler.log"
  awk '/^--- PASS: Test/ {print $3}' "$work/$compiler.log" | sort >"$work/actual"
  diff -u "$work/expected" "$work/actual"
done
