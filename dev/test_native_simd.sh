#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

: "${LLGO:=llgo}"
# Preserve paths relative to the repository when entering the runtime module.
if [[ "$LLGO" == */* && "$LLGO" != /* ]]; then
  LLGO="$PWD/$LLGO"
fi
export GOEXPERIMENT=simd

# The runtime is a separate module. Exercise the software FMA implementation
# against the host Go and high-precision references through LLGo as well.
(cd runtime && "$LLGO" test -O2 -count=1 -timeout=2m ./internal/runtime/math)

if [[ "$(go env GOARCH)" != amd64 ]]; then
  "$LLGO" test -O0 -count=1 -timeout=2m ./test/simd/...
  "$LLGO" test -O2 -count=1 -timeout=2m ./test/simd/...
  GODEBUG=simd=0 "$LLGO" test -O2 -count=1 -timeout=2m ./test/simd/portable
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
  for suite in arch portable; do
    package=./test/simd
    filter='^TestSIMD(FMV|Bridge|Extend|Narrow|Resize)'
    settings=(cpu.avx2=off cpu.avx=off cpu.fma=off cpu.all=off)
    if [[ "$suite" == portable ]]; then
      package+=/portable
      filter='^TestPortable'
      settings+=(simd=0 simd=128 simd=+128)
      for width in 0 128; do
        for feature in avx2 avx fma all; do
          settings+=("simd=$width,cpu.$feature=off")
        done
      done
    fi
    binary="$output/simd-$suite-$profile.exe"
    "$LLGO" test "${flags[@]}" -c -o "$binary" "$package"
    "$binary" -test.v -test.count=1 -test.timeout=2m
    for debug in "${settings[@]}"; do
      echo "SIMD $suite: $profile GODEBUG=$debug"
      GODEBUG="$debug" "$binary" -test.v -test.count=1 -test.timeout=2m -test.run="$filter"
    done
  done
done
