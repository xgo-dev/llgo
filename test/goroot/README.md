# GOROOT Runner

This package runs selected upstream `GOROOT/test` cases against `llgo` without
copying the upstream source files into this repository.

The source of truth is an external `GOROOT`. Upstream files stay read-only;
each case runs in a temporary workspace.

The runner's own unit tests each have a three-minute watchdog, including their subtests and cleanup. Each case logs `goroot case START` and `goroot case END` with its name and elapsed time to stderr; external GOROOT cases also log their path and position in the suite. Helper subprocesses keep their original output contracts. A timeout fails the test process and reports the test name; it is not retried or ignored. This limit does not apply to the external `TestGoRootRunCases` suite, whose individual build/run steps use the budgets below. Child output pipes have a five-second drain limit after exit, and termination is followed by at most ten seconds of waiting before failing the runner. These guards depend on the runner's runtime timers working; they do not replace an external process/job timeout for runtime-wide hangs.

Directive modes:

- `legacy`: `run`
- `ci`: `run`, `runoutput`, and `buildrun`
- `runlike`: all runnable single-file and directory directives
- `coverage`: recursively discovers and executes `compile`, `errorcheck`,
  `errorcheckandrundir`, `run`, `runoutput`, `rundir`, `runindir`, `buildrun`,
  and `buildrundir`

`coverage` follows the upstream test runner's directive and compiler flag
conventions. Compile-only and diagnostic cases are executed, not omitted from
the coverage total. Known failures remain executed and are classified through
`xfail.yaml`. Cases that exercise gc-specific or otherwise inapplicable
behavior are skipped globally through `notapplicable.yaml`, so they cannot be
mistaken for version-specific compatibility work. Explicitly host-unsafe cases
are also skipped. Each not-applicable entry documents both the
toolchain-specific mechanism under test and why the corresponding behavior is
not an LLGo compatibility goal.

The GOROOT workflow builds LLGo and this runner with the repository toolchain,
then runs the `ci` directive set from the two most recent Go releases. Every run
publishes the expectation-mismatch table in its Actions summary. The scheduled
run in `xgo-dev/llgo` also replaces the previous `[GOROOT daily] YYYY-MM-DD`
issue. A manual dispatch or adding the existing `go-test-compat` label to a
pull request runs the same matrix without modifying issues.

Basic usage:

```bash
go test ./test/goroot -count=1 -args \
  -goroot "$(go env GOROOT)" \
  -dirs . \
  -case '^helloworld\.go$'
```

Resource-constrained full coverage:

```bash
GOMAXPROCS=2 go test -p=1 ./test/goroot -count=1 -timeout=180m -args \
  -goroot "$(go env GOROOT)" \
  -directive-mode coverage \
  -max-rss-mib 1536 \
  -rss-warn-mib 768 \
  -min-memory-free-percent 20 \
  -min-swap-free-mib 1024
```

On Unix, the RSS limit covers the complete process group and terminates the
group when exceeded. System memory and swap are checked before and during every
child command. Resource guard failures cannot be converted into expected
failures by `xfail.yaml`.

Multiple toolchains:

```bash
bash ./dev/test_goroot.sh /path/to/go1.23 /path/to/go1.24 -- -dirs . -case '^helloworld\.go$'
```

Host-driven WebAssembly execution keeps discovery and result comparison in the native runner while compiling both the official Go baseline and LLGo output for WebAssembly. The four W3 paths are `J32-GoJS` (wasm32 with Go-compatible js/wasm source/API), `J32-Emscripten` (wasm32 with the Emscripten JS provider), `J64-Emscripten` (wasm64 with the Emscripten JS provider), and `W32-WASI` (wasm32 with WASI Preview 1). Generated programs run through the profile's Node or Wasmtime adapter.

```bash
go test ./test/goroot -count=1 -args \
  -goroot "$(go env GOROOT)" \
  -wasm-profile J32-Emscripten \
  -directive-mode ci \
  -case '^helloworld\.go$'
```

Useful flags:

- `-goroot`: upstream Go toolchain root to read tests from
- `-go`: baseline `go` binary; defaults to `<goroot>/bin/go`
- `-llgo`: existing `llgo` binary to use; otherwise one is built from the current checkout
- `-wasm-profile`: host-driven target execution (`J32-GoJS`, `J32-Emscripten`, `J64-Emscripten`, or `W32-WASI`)
- `-dirs`: comma-separated `GOROOT/test` subdirectories to scan
- `-case`: regexp filter on the relative case path
- `-directive-mode`: case discovery mode: `legacy`, `ci`, `runlike`, or `coverage`
- `-directives`: comma-separated directive filter within the selected mode
- `-limit`: stop after N matching cases
- `-shard-index`: 0-based shard index used to partition matching cases
- `-shard-total`: total number of shards used to partition matching cases
- `-keepwork`: keep the temporary symlink work tree for debugging
- `-build-timeout`: timeout for each build or compile step
- `-run-timeout`: timeout for each compiled program run
- `-list-cases`: print selected totals without building or running cases
- `-list-case-paths`: print selected directives and paths without running cases
- `-max-rss-mib`: hard process-group RSS limit; `0` disables it
- `-rss-warn-mib`: log commands whose observed peak RSS reaches this value
- `-rss-poll`: process-group RSS sampling interval
- `-min-memory-free-percent`: minimum system free-memory percentage
- `-min-swap-free-mib`: minimum free swap in MiB
- `-memory-pressure-poll`: system memory and swap sampling interval
- `-xfail`: xfail YAML file, relative to repo root by default
- `-not-applicable`: not-applicable YAML file, relative to repo root by default
