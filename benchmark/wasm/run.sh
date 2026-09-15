#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <source-root> <llgo-output> <result-directory>" >&2
  exit 2
fi

harness_root="$(cd "$(dirname "$0")/../.." && pwd)"
source_root="$(cd "$1" && pwd)"
mkdir -p "$(dirname "$2")" "$3"
llgo_output="$(cd "$(dirname "$2")" && pwd)/$(basename "$2")"
result_directory="$(cd "$3" && pwd)"

(
  cd "$source_root"
  LLGO_ROOT="$source_root" go build -p=1 -o "$llgo_output" ./cmd/llgo
)

(
  cd "$harness_root"
  LLGO_ROOT="$source_root" go run ./benchmark/wasm \
    -root "$source_root" \
    -llgo "$llgo_output" \
    -out "$result_directory" \
    -build-runs "${LLGO_WASM_BENCH_BUILD_RUNS:-3}"
)
