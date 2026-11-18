#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

: "${LLGO:=llgo}"
export GOEXPERIMENT=simd

for profile in gojs emscripten emscripten-memory64; do
  (
    # Avoid inheriting a host target from the caller for raw GoJS builds.
    export GOOS=js GOARCH=wasm CGO_ENABLED=0
    target=(-emulator)
    if [[ "$profile" != gojs ]]; then
      target+=(-target "$profile")
    fi
    echo "SIMD: $profile O0 boundary"
    # The full O0 test binary exceeds Node's per-function local-variable limit.
    "$LLGO" run -O0 "${target[@]}" ./test/simd/testdata/boundary
    for lto in off thin full; do
      echo "SIMD: $profile O2 LTO=$lto"
      flags=(-O2)
      if [[ "$lto" != off ]]; then
        flags+=("-lto=$lto")
      fi
      "$LLGO" test "${flags[@]}" "${target[@]}" \
        -v -count=1 -timeout=2m -pclntab=none ./test/simd/... ./test/std/simd
    done
    for lto in thin full; do
      echo "SIMD: $profile O3 LTO=$lto boundary"
      # O3 argument promotion must preserve the JS SjLj memory bridge.
      "$LLGO" run -O3 "-lto=$lto" "${target[@]}" ./test/simd/testdata/boundary
    done
    GODEBUG=simd=0 "$LLGO" test -O2 "${target[@]}" \
      -v -count=1 -timeout=2m -pclntab=none ./test/simd/portable ./test/std/simd
  )
done
