#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 3 && $# -ne 6 ]]; then
  echo "usage: $0 <source-root> <llgo-output> <result-directory> [<current-source-root> <current-llgo-output> <current-result-directory>]" >&2
  exit 2
fi

harness_root="$(cd "$(dirname "$0")/../.." && pwd)"

benchmark_names=(
  MergeCompilerFlags
  MergeLinkerFlags
  LookupPCRandom
  RuntimeGetG
  GlobalRead
  GlobalWrite
  DirectCall
  InterfaceCall
  Defer
  ChannelBuffered
  ChannelHandoff
  Goroutine
)
benchmark_binaries=(
  clang
  clang
  funcinfo
  llgoext
  llgoext
  llgoext
  llgoext
  llgoext
  llgoext
  llgoext
  llgoext
  llgoext
)

absolute_output() {
  mkdir -p "$(dirname "$1")"
  echo "$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
}

absolute_directory() {
  mkdir -p "$1"
  (cd "$1" && pwd)
}

build_llgo() {
  local source_root="$1"
  local llgo_output="$2"
  (
    cd "$source_root"
    LLGO_ROOT="$source_root" go build -p=1 -o "$llgo_output" ./cmd/llgo
  )
}

export_result() {
  local result_directory="$1"
  (
    cd "$harness_root"
    go run ./benchmark/baseline \
      -mode export \
      -out "$result_directory" \
      -benchmark-output "$result_directory/benchmark.txt"
  )
}

run_single() {
  local source_root="$1"
  local llgo_output="$2"
  local result_directory="$3"

  build_llgo "$source_root" "$llgo_output"
  (
    cd "$harness_root"
    LLGO_ROOT="$source_root" go run ./benchmark/baseline \
      -root "$source_root" \
      -harness-root "$harness_root" \
      -llgo "$llgo_output" \
      -out "$result_directory"
  )

  local go_results="$result_directory/go.txt"
  : > "$go_results"

  local binaries="$result_directory/tests"
  build_benchmark_binaries "$source_root" "$llgo_output" "$binaries"
  local index
  for index in "${!benchmark_names[@]}"; do
    local benchtime=1s
    [[ "${benchmark_names[$index]}" != Goroutine ]] || benchtime=100x
    run_benchmark_series \
      "$source_root" \
      "$binaries/${benchmark_binaries[$index]}.test" \
      "$go_results" \
      "^Benchmark${benchmark_names[$index]}$" \
      "$benchtime"
  done

  export_result "$result_directory"
}

build_benchmark_binaries() {
  local source_root="$1"
  local llgo_output="$2"
  local binary_directory="$3"
  mkdir -p "$binary_directory"
  (
    cd "$source_root"
    GOMAXPROCS=1 LLGO_ROOT="$source_root" go test -c \
      -o "$binary_directory/clang.test" ./internal/clang
    GOMAXPROCS=1 LLGO_ROOT="$source_root" go test -c \
      -o "$binary_directory/funcinfo.test" ./internal/build/funcinfo
    GOMAXPROCS=1 LLGO_ROOT="$source_root" LLGO_FULL_RPATH=true \
      "$llgo_output" test -c -o "$binary_directory/llgoext.test" ./test/llgoext
  )
}

run_benchmark_sample() {
  local source_root="$1"
  local binary="$2"
  local pattern="$3"
  local benchtime="$4"
  local output="$5"
  (
    cd "$source_root"
    GOMAXPROCS=1 LLGO_ROOT="$source_root" "$binary" \
      -test.run '^$' \
      -test.bench "$pattern" \
      -test.benchtime "$benchtime" \
      -test.count=1 \
      -test.cpu=1
  ) | tee -a "$output"
}

run_benchmark_series() {
  local source_root="$1"
  local binary="$2"
  local output="$3"
  local pattern="$4"
  local benchtime="$5"

  run_benchmark_sample "$source_root" "$binary" "$pattern" "$benchtime" /dev/null
  local round
  for ((round = 0; round < 7; round++)); do
    run_benchmark_sample "$source_root" "$binary" "$pattern" "$benchtime" "$output"
  done
}

run_benchmark_pair() {
  local base_root="$1"
  local base_binary="$2"
  local base_output="$3"
  local current_root="$4"
  local current_binary="$5"
  local current_output="$6"
  local pattern="$7"
  local benchtime="$8"
  local pair_index="$9"

  # Warm both binaries without recording the result. Alternate the leading
  # revision between groups, then alternate it again for every measured sample.
  if (( pair_index % 2 == 0 )); then
    run_benchmark_sample "$base_root" "$base_binary" "$pattern" "$benchtime" /dev/null
    run_benchmark_sample "$current_root" "$current_binary" "$pattern" "$benchtime" /dev/null
  else
    run_benchmark_sample "$current_root" "$current_binary" "$pattern" "$benchtime" /dev/null
    run_benchmark_sample "$base_root" "$base_binary" "$pattern" "$benchtime" /dev/null
  fi

  local round
  for ((round = 0; round < 7; round++)); do
    if (( (round + pair_index) % 2 == 0 )); then
      run_benchmark_sample "$base_root" "$base_binary" "$pattern" "$benchtime" "$base_output"
      run_benchmark_sample "$current_root" "$current_binary" "$pattern" "$benchtime" "$current_output"
    else
      run_benchmark_sample "$current_root" "$current_binary" "$pattern" "$benchtime" "$current_output"
      run_benchmark_sample "$base_root" "$base_binary" "$pattern" "$benchtime" "$base_output"
    fi
  done
}

run_paired() {
  local base_root="$1"
  local base_llgo="$2"
  local base_result="$3"
  local current_root="$4"
  local current_llgo="$5"
  local current_result="$6"

  build_llgo "$base_root" "$base_llgo"
  build_llgo "$current_root" "$current_llgo"

  # Keep one current-checkout harness for both revisions. Suite changes must
  # therefore remain executable against the pull request base.
  (
    cd "$harness_root"
    go run ./benchmark/baseline \
      -mode collect-paired \
      -base-root "$base_root" \
      -base-llgo "$base_llgo" \
      -base-out "$base_result" \
      -root "$current_root" \
      -harness-root "$harness_root" \
      -llgo "$current_llgo" \
      -out "$current_result"
  )

  local base_binaries="$base_result/tests"
  local current_binaries="$current_result/tests"
  build_benchmark_binaries "$base_root" "$base_llgo" "$base_binaries"
  build_benchmark_binaries "$current_root" "$current_llgo" "$current_binaries"

  local base_go="$base_result/go.txt"
  local current_go="$current_result/go.txt"
  : > "$base_go"
  : > "$current_go"

  local index
  for index in "${!benchmark_names[@]}"; do
    local benchtime=1s
    [[ "${benchmark_names[$index]}" != Goroutine ]] || benchtime=100x
    local binary="${benchmark_binaries[$index]}.test"
    run_benchmark_pair \
      "$base_root" "$base_binaries/$binary" "$base_go" \
      "$current_root" "$current_binaries/$binary" "$current_go" \
      "^Benchmark${benchmark_names[$index]}$" "$benchtime" "$index"
  done

  export_result "$base_result"
  export_result "$current_result"
}

source_root="$(cd "$1" && pwd)"
llgo_output="$(absolute_output "$2")"
result_directory="$(absolute_directory "$3")"

if [[ $# -eq 3 ]]; then
  run_single "$source_root" "$llgo_output" "$result_directory"
else
  current_source_root="$(cd "$4" && pwd)"
  current_llgo_output="$(absolute_output "$5")"
  current_result_directory="$(absolute_directory "$6")"
  run_paired \
    "$source_root" "$llgo_output" "$result_directory" \
    "$current_source_root" "$current_llgo_output" "$current_result_directory"
fi
