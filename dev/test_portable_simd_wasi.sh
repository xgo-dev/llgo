#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

: "${LLGO:=llgo}"
export GOEXPERIMENT=simd GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0
output=$(mktemp -d)
trap 'rm -rf "$output"' EXIT

for profile in O0-off O2-off O2-thin O2-full; do
  flags=("-${profile%%-*}")
  lto=${profile#*-}
  if [[ "$lto" != off ]]; then flags+=("-lto=$lto"); fi
  binary="$output/portable-$profile.wasm"
  echo "Portable SIMD: WASI $profile"
  "$LLGO" test "${flags[@]}" -c -o "$binary" ./test/simd/portable
  for debug in simd=128 simd=0; do
    # WASI does not inherit host environment variables into the guest.
    wasmer run --enable-exceptions --enable-simd --stack-size=1048576 \
      --env="GODEBUG=$debug" "$binary" -- -test.v -test.count=1 -test.timeout=2m \
      -test.bench='^BenchmarkSIMDDot$' -test.benchtime=1x
  done
done
