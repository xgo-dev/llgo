# LLGo baseline benchmarks

This suite is the lightweight performance gate for ordinary LLGo changes. It
uses fixed workloads and short calibrated benchmarks on Linux, macOS, and the
MSVC and MinGW Windows profiles, so it can run on every `main` push and pull
request. Branch-only series can be run explicitly with `workflow_dispatch`,
avoiding duplicate push and pull-request jobs for the same commit. The native
jobs record normalized artifacts; a trusted `workflow_run` publisher validates
and merges all four platform profiles into one commit, branch, or pull-request
series.

The program workloads reuse:

- `benchmark/binary_size/cprintf`: only `lib/c.Printf` (default and `-lto=full`);
- `benchmark/binary_size/println`: only the built-in `println` (default and `-lto=full`);
- `benchmark/binary_size/fmtprintf`: `fmt.Printf` (default and `-lto=full`).

For each workload, the collector performs an unmeasured warm build, then records
median build time, median process time, file size, executable-code bytes,
allocated non-executable data, and zero-filled data. On ELF, read-only constants
are included in the data bucket; on Mach-O, `__TEXT` constants are included in
the text bucket. The Go benchmark stream records five samples of selected
compiler helpers and LLGo-generated core-language operations: direct/interface
calls, defer, goroutine creation, channels, `getg`, and global access. It also
runs the same `benchmark/timer` source with native Go and LLGo. The timer suite
measures `AfterFunc` create/stop, active reset, a stopped-timer rearm/stop cycle,
reset with 1,024 live timers, and zero-duration callback delivery through one
reused callback and channel. The first four avoid timer-channel implementation
details; the delivery case intentionally includes scheduling and callback
handoff so timer-management improvements are not mistaken for end-to-end
latency improvements. Fixed iteration budgets bound native-handle accumulation
when the same harness runs against the former libuv implementation. Each timer
path warms its native runtime before recording samples so initialization is
excluded consistently.

The same `benchmark/stringequal` source also runs with Go and each LLGo revision.
It compares distinct 8-, 128- and 1,024-byte strings, using ordinary string
equality and temporary byte-to-string conversion. Equal inputs and first/last
byte mismatches use fixed one-million-call samples with checked results. LLGo
uses `-O3 -lto=full`; Go uses its defaults. The remaining conversion cost is
included when measuring a compiler that has not eliminated that conversion.

The `benchmark/goroutine` suite measures two workloads with native Go and LLGo.
In `create-and-wait/goroutines=N`, each timed iteration creates N goroutines,
each performing one atomic increment, and waits for the whole batch to finish.
In `serial-dispatch/tasks=N`, each iteration sends one batch to a single
pre-existing goroutine, which performs N atomic increments serially and sends a
completion notification. The worker reuses its request and completion channels.
The sizes are 1, 16, 128, and 1,024. The reported `ns/op` covers one complete batch
in each workload. Compare each case against the same case in the PR base to
measure compiler and runtime changes.

Each case completes one warmup batch before resetting the timer. Creation uses
ten fixed iterations per sample to bound native thread creation. Serial dispatch
uses calibrated 250 ms samples. Both workloads collect five samples. Native Go
uses one P. LLGo retains its native execution model, which currently does not
limit goroutine execution through `GOMAXPROCS`.

Once the timer selectors are present on the target branch, a paired pull-request
result compiles `/LLGo` once from the merge base and once from the head, while
`/Go` is repeated as a same-runner control. A timer-runtime pull request whose
base still uses libuv then shows native Go, libuv LLGo, and current LLGo in one
report. Pull-request artifacts deliberately use the target branch's trusted
configuration. A pull request that first introduces selectors still runs and
validates their samples, but its new table begins publishing after merge.

For pull requests, each platform job checks out the recorded base and current
commits into the same source path, then runs both suites sequentially on the same
runner. The pull request comment compares that pair, avoiding differences from
runner machines and embedded source paths. Dependency setup is shared, and Go's
build cache can be reused by unchanged packages; main pushes still run the suite
only once. Very small changes can still be scheduler, frequency, or thermal
noise and should be confirmed by repeated workflow runs. If a workflow does not
provide a paired result, the publisher falls back to the latest matching `main`
data.

The trusted publisher commits the current result history and generated site to
the `pages` branch of the configured data repository. Every LLGo repository
defaults to `<owner>/llgo-benchmark-data`:

```text
llgo/baseline/series/main/main
llgo/baseline/series/branch/<safe branch identifier>
llgo/baseline/series/pull/<number>
```

The publisher never executes code from the measured revision and pull request
jobs never receive the benchmark repository token. Pull requests receive one
updated summary comment linking to their long-term trend page. If no matching
`main` history exists yet, the pull-request report is still published and
marks every metric as `new`.

Local collection:

```sh
GOMAXPROCS=2 go build -o .benchmark/llgo ./cmd/llgo
go run ./benchmark/baseline \
  -llgo .benchmark/llgo \
  -out .benchmark/results
```

Write the selected Go benchmark output to `.benchmark/results/go.txt`:

```bash
results=.benchmark/results/go.txt
GOMAXPROCS=1 go test \
  -run '^$' \
  -bench '^(BenchmarkMergeCompilerFlags|BenchmarkMergeLinkerFlags|BenchmarkLookupPCRandom)$' \
  -benchtime=250ms -count=5 -cpu=1 \
  ./internal/clang ./internal/build/funcinfo | tee "$results"
GOMAXPROCS=1 go test \
  -run '^$' -bench '^BenchmarkTimer' -benchtime=250ms -count=5 -cpu=1 \
  ./benchmark/timer | tee -a "$results"
timer_test=.benchmark/llgo-timer.test
timer_stdio_nobuf=
if [[ "$(go env GOOS)" == windows ]]; then
  timer_test+=.exe
  timer_stdio_nobuf=1
fi
LLGO_FULL_RPATH=true LLGO_ROOT="$PWD" LLGO_STDIO_NOBUF="$timer_stdio_nobuf" \
  .benchmark/llgo test -c -o "$timer_test" ./benchmark/timer
run_llgo_timer_samples() {
  benchmarks="$1"
  benchtime="$2"
  for _ in 1 2 3 4 5; do
    "$timer_test" \
      -test.run='^$' -test.bench="$benchmarks" \
      -test.benchtime="$benchtime" -test.count=1
  done
}
run_llgo_timer_samples '^BenchmarkTimerCreateStop$' 10000x | tee -a "$results"
run_llgo_timer_samples '^BenchmarkTimerAfterFuncZeroDelivery$' 10000x | tee -a "$results"
GOMAXPROCS=1 go test \
  -run '^$' -bench '^BenchmarkStringEquality$' \
  -benchtime=1000000x -count=5 -cpu=1 \
  ./benchmark/stringequal | tee -a "$results"
GOMAXPROCS=1 LLGO_ROOT="$PWD" .benchmark/llgo test -O3 -lto=full \
  -run '^$' -bench '^BenchmarkStringEquality$' \
  -benchtime=1000000x -count=5 \
  ./benchmark/stringequal | tee -a "$results"
run_llgo_timer_samples \
  '^BenchmarkTimer(RearmStopped|ResetActive|ResetHeap1024)$' 20000x \
  | tee -a "$results"
GOMAXPROCS=1 .benchmark/llgo test \
  -run '^$' \
  -bench '^(BenchmarkRuntimeGetG|BenchmarkGlobal(Read|Write)|Benchmark(DirectCall|InterfaceCall|Defer|ChannelBuffered|ChannelHandoff))$' \
  -benchtime=250ms -count=5 \
  ./test/llgoext | tee -a "$results"
GOMAXPROCS=1 .benchmark/llgo test \
  -run '^$' -bench '^BenchmarkGoroutine$' -benchtime=100x -count=5 \
  ./test/llgoext | tee -a "$results"
```

Native Go timer samples use `-cpu=1`. LLGo timer samples intentionally retain
the current native runtime's 1:1 goroutine-to-thread execution model instead of
claiming an unsupported `GOMAXPROCS` limit; base and head are measured
sequentially on the same runner.

Then validate and export the complete artifact in standard Go benchmark format:

```sh
go run ./benchmark/baseline \
  -mode export \
  -out .benchmark/results \
  -benchmark-output .benchmark/results/benchmark.txt
```
