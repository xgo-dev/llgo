#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $0 <compiler-source-root> <llgo-output> <result-directory> [fixture-source-root]" >&2
  exit 2
fi

harness_root="$(cd "$(dirname "$0")/../.." && pwd)"
source_root="$(cd "$1" && pwd)"
fixture_root="${4:-$source_root}"
fixture_root="$(cd "$fixture_root" && pwd)"
mkdir -p "$(dirname "$2")" "$3"
llgo_output="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
result_directory="$(cd "$3" && pwd)"

size_args=()
(
  cd "$source_root"
  LLGO_ROOT="$source_root" go build -p=1 -o "$llgo_output" ./cmd/llgo
)
if [[ "${LLGO_WASM_BENCH_SIZE_MATRIX:-0}" == 1 ]]; then
  (
    cd "$source_root"
    LLGO_ROOT="$source_root" go build -p=1 -tags="${LLGO_WASM_BENCH_DEV_TAGS:-dev}" -o "$llgo_output-dev" ./cmd/llgo
  )
  size_args=(-size-llgo "$llgo_output-dev" -tinygo "${TINYGO:-tinygo}")
fi

(
  cd "$harness_root"
  LLGO_ROOT="$source_root" go run ./benchmark/wasm \
    -root "$source_root" \
    -fixture-root "$fixture_root" \
    -llgo "$llgo_output" \
    -out "$result_directory" \
    -build-runs "${LLGO_WASM_BENCH_BUILD_RUNS:-3}" "${size_args[@]}"
)
