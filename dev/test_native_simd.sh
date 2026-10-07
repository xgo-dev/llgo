#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

: "${LLGO:=llgo}"
export GOEXPERIMENT=simd

if [[ "$(go env GOARCH)" != amd64 ]]; then
  "$LLGO" test -O0 -count=1 -timeout=2m ./test/simd/...
  "$LLGO" test -O2 -count=1 -timeout=2m ./test/simd/...
  exit
fi

export GOAMD64=v1
if [[ -n "${LLGO_SIMD_ARTIFACT_DIR:-}" ]]; then
  output=$LLGO_SIMD_ARTIFACT_DIR
  mkdir -p "$output"
else
  output=$(mktemp -d)
  trap 'rm -rf "$output"' EXIT
fi

for profile in O0-off O2-off O2-thin O2-full; do
  opt=${profile%%-*}
  lto=${profile#*-}
  flags=("-$opt")
  if [[ "$lto" != off ]]; then
    flags+=("-lto=$lto")
  fi
  echo "SIMD: amd64 $profile"
  binary="$output/simd-$profile.exe"
  "$LLGO" test "${flags[@]}" -c -o "$binary" ./test/simd
  "$binary" -test.v -test.count=1 -test.timeout=2m
  for debug in cpu.avx2=off cpu.avx=off cpu.fma=off cpu.all=off; do
    echo "SIMD FMV: $profile GODEBUG=$debug"
    GODEBUG="$debug" "$binary" -test.v -test.count=1 -test.timeout=2m -test.run='^TestSIMDFMV'
  done
done
