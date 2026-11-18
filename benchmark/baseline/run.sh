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

# A cross-architecture Windows lane builds the host compiler before activating
# the target SDK, then measures target programs in the activated environment.
if [[ -n "${LLGO_BENCHMARK_PREBUILT_LLGO:-}" ]]; then
  llgo_output="$(cd "$(dirname "$LLGO_BENCHMARK_PREBUILT_LLGO")" && pwd)/$(basename "$LLGO_BENCHMARK_PREBUILT_LLGO")"
  if [[ ! -f "$llgo_output" ]]; then
    echo "prebuilt LLGo compiler not found: $llgo_output" >&2
    exit 1
  fi
fi

# An explicit output path is not given the platform suffix by go build. Keep
# the compiler discoverable by both Bash and native Windows subprocesses.
host_goos="$(go env GOOS)"
if [[ "$host_goos" == windows && "$llgo_output" != *.exe ]]; then
  llgo_output+=.exe
fi

if [[ -z "${LLGO_BENCHMARK_PREBUILT_LLGO:-}" ]]; then
  (
    cd "$source_root"
    LLGO_ROOT="$source_root" go build -p=1 -o "$llgo_output" ./cmd/llgo
  )
fi

(
  cd "$harness_root"
  # Keep one current-checkout harness for both source revisions. Benchmark
  # suite changes must therefore remain executable against the PR base.
  LLGO_ROOT="$source_root" go run ./benchmark/baseline \
    -root "$source_root" \
    -llgo "$llgo_output" \
    -out "$result_directory"
)

go_results="$result_directory/go.txt"
: > "$go_results"

timer_test_suffix=
timer_stdio_nobuf=
if [[ "$host_goos" == windows ]]; then
  timer_test_suffix=.exe
  timer_stdio_nobuf=1
fi
timer_llgo_test="$(dirname "$llgo_output")/$(basename "${llgo_output%.exe}")-timer.test${timer_test_suffix}"

(
  cd "$harness_root"
  LLGO_FULL_RPATH=true \
    LLGO_ROOT="$source_root" \
    LLGO_STDIO_NOBUF="$timer_stdio_nobuf" \
    "$llgo_output" test \
    -c \
    -o "$timer_llgo_test" \
    ./benchmark/timer
)

run_llgo_timer_samples() {
  local benchmarks="$1"
  local benchtime="$2"
  for _ in 1 2 3 4 5; do
    "$timer_llgo_test" \
      -test.run='^$' \
      -test.bench="$benchmarks" \
      -test.benchtime="$benchtime" \
      -test.count=1
  done
}

(
  cd "$source_root"
  GOMAXPROCS=1 LLGO_ROOT="$source_root" go test \
    -run '^$' \
    -bench '^(BenchmarkMergeCompilerFlags|BenchmarkMergeLinkerFlags|BenchmarkLookupPCRandom)$' \
    -benchtime=250ms \
    -count=5 \
    -cpu=1 \
    ./internal/clang ./internal/build/funcinfo
) | tee -a "$go_results"

# Run one benchmark source through native Go and the compiler revision under
# test. Native Go can calibrate normally. Each LLGo sample gets a fresh process
# because the old libuv baseline retains native timer handles after Stop; fixed
# iteration counts keep the within-sample workload bounded too.
(
  cd "$harness_root"
  GOMAXPROCS=1 go test \
    -run '^$' \
    -bench '^BenchmarkTimer' \
    -benchtime=250ms \
    -count=5 \
    -cpu=1 \
    ./benchmark/timer
) | tee -a "$go_results"

run_llgo_timer_samples '^BenchmarkTimerCreateStop$' 10000x | tee -a "$go_results"
run_llgo_timer_samples '^BenchmarkTimerAfterFuncZeroDelivery$' 10000x | tee -a "$go_results"
run_llgo_timer_samples '^BenchmarkTimer(RearmStopped|ResetActive|ResetHeap1024)$' 20000x | tee -a "$go_results"

# As with timer benchmarks, use one current-checkout source for both compiler
# revisions. Keep the data checksum and fixed iteration count comparable.
(
  cd "$harness_root"
  GOMAXPROCS=1 go test -run '^$' -bench '^BenchmarkStringEquality$' \
    -benchtime=1000000x -count=5 -cpu=1 ./benchmark/stringequal
  GOMAXPROCS=1 LLGO_ROOT="$source_root" "$llgo_output" test -O3 -lto=full \
    -run '^$' -bench '^BenchmarkStringEquality$' \
    -benchtime=1000000x -count=5 ./benchmark/stringequal
) | tee -a "$go_results"

(
  cd "$source_root"
  GOMAXPROCS=1 LLGO_ROOT="$source_root" "$llgo_output" test \
    -run '^$' \
    -bench '^(BenchmarkRuntimeGetG|BenchmarkGlobal(Read|Write)|Benchmark(DirectCall|InterfaceCall|Defer|ChannelBuffered|ChannelHandoff))$' \
    -benchtime=250ms \
    -count=5 \
    ./test/llgoext
) | tee -a "$go_results"

# The current native backend creates one pthread per goroutine and intentionally
# has a bounded lifecycle stress limit. Keep creation monitoring deterministic
# instead of letting testing auto-calibrate to millions of host threads.
(
  cd "$source_root"
  GOMAXPROCS=1 LLGO_ROOT="$source_root" "$llgo_output" test \
    -run '^$' \
    -bench '^BenchmarkGoroutine$' \
    -benchtime=100x \
    -count=5 \
    ./test/llgoext
) | tee -a "$go_results"

(
  cd "$harness_root"
  go run ./benchmark/baseline \
    -mode export \
    -out "$result_directory" \
    -benchmark-output "$result_directory/benchmark.txt"
)
