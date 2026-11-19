# WebAssembly build and size benchmarks

The harness builds the existing `benchmark/binary_size` examples `cprintf`, `println`, and `fmtprintf`, plus the dynamic `reflectcall` fixture, with LLGo for all five supported execution entries: J32/GoJS, W32/raw `wasip1`, J32/Emscripten, J64/Emscripten Memory64, and W32/WASI. Each example/profile records the Wasm module size, generated JavaScript glue size (zero when absent), and selected build times. These are build/size measurements, not runtime or official-Go ABI acceptance.

Official Go size references cover `println`, `fmtprintf`, and `reflectcall` on `js/wasm` and `wasip1/wasm`. There is no official-Go `cprintf` reference: that example calls C `printf` through LLGo's C interop, which official Go does not provide on wasm.

The existing unqualified metric names continue to mean `println`, preserving its benchmark history. The `cprintf/`, `fmtprintf/`, and `reflectcall/` prefixes identify the other examples. Artifacts are stored separately for every example, profile, and compiler.

From the repository root, with the usual LLVM, Emscripten and Binaryen tools:

```sh
benchmark/wasm/run.sh "$PWD" /tmp/llgo-wasm-bench /tmp/llgo-wasm-bench-results
```

An optional fourth argument selects a different repository checkout for benchmark fixtures. CI uses this when measuring a pull request base, so both compilers see the current benchmark sources even when the pull request adds a new fixture:

```sh
benchmark/wasm/run.sh /path/to/base /tmp/llgo-wasm-base /tmp/llgo-wasm-base-results "$PWD"
```

The result directory is replaced on each run; use a dedicated output directory.
To reuse a compiler already built for the source revision:

```sh
go run ./benchmark/wasm -root "$PWD" -llgo /path/to/llgo \
  -out /tmp/llgo-wasm-bench-results -build-runs 1
```

The `println` profile builds have one warm-up followed by measured builds; the CLI defaults to three measurements and reports their median. `cprintf` and `fmtprintf` are size-only cases and build once per profile. Both the named W32/WASI target and raw `GOOS=wasip1 GOARCH=wasm` entry use the typed-bridge fallback, and the harness records both module sizes. To expose its compile cost without duplicating an equivalent timed build, `reflectcall` times only the named W32/WASI entry; JavaScript profiles remain size-only. With `LLGO_WASM_BENCH_BUILD_RUNS=1`, CI performs 26 LLGo builds and six official-Go reference builds per revision. A single CI timing sample is noisier than the default median, but base and head run sequentially with the same fixtures and settings on one runner.

The existing WebAssembly benchmark job tests this harness and measures both PR base and head, publishing their results through `.github/llgo-wasm-benchmark.yml`.

## Release size and formatting dependencies

`testdata/fmtrich/main.go` reproduces the complex-formatting workload from #2480. It exercises compound values, reflection, Unicode, numeric formatting, and a user-defined Stringer. Run it separately from the per-PR timing matrix:

```sh
src=./benchmark/wasm/testdata/fmtrich/main.go
mkdir -p out/llgo out/go out/tinygo
llgo build -target=wasi -Oz -debug-artifact=none -ldflags=-s \
  -size -size-level=package -size-format=json -o out/llgo/app.wasm "$src" > out/llgo-size.json
GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -ldflags=-s -o out/go/app.wasm "$src"
tinygo build -target=wasip1 -opt=z -no-debug -o out/tinygo/app.wasm "$src"
```

Use the pinned Binaryen tools described in [Wasm size optimization](../../doc/wasm-size.md). Validate modules and compare their output in the same Wasmer version with exceptions and SIMD enabled. Repeat with `-target=emscripten` and `-target=emscripten-memory64` for LLGo browser deployment sizes, using the corresponding Node runners. Measure Wasm, JS, workers and manifests separately and sum the independently compressed files for transport size. Record the compiler revisions, feature flags, artifact sizes and runner versions with the results. Keep debug files in a separate total. Go and TinyGo use different runtime and host contracts.

The LLGo report attributes final function bodies using any names that survive linking and optimization, including names LLGo removes during final stripping. Use `-size-level=full` to investigate individual symbols. Body sizes include inline code attributed to the containing function. Unowned data and anonymous bodies remain in separate unknown buckets. For example, optimized Emscripten builds can omit names before LLGo's final strip. The JSON `wasm.imports` and `wasm.exports` lists show the final module boundary for these builds as well, with internal names for function exports where available. Compare those lists alongside JS and worker bytes when changing host roots.

For explicit final optimization, metadata stripping and complete deployment accounting, see [Final WebAssembly size optimization](../../doc/wasm-size.md). The existing benchmark defaults remain unchanged.

## WASI size matrix

The existing WebAssembly benchmark job also records the final stripped modules
for `cprintf`, `println`, and `fmtprintf`. LLGo uses a dev compiler at
`-Oz` with default reachability, explicit `-deadcodedrop`, and
`-deadcodedrop -globaldce=false -lto=full`. Go and TinyGo build the same
`println` and `fmtprintf` sources for WASI with debug information stripped.
The LLGo C FFI fixture has no same-source Go or TinyGo reference.

For a local size-only run with LLVM 22:

```sh
go build -tags=dev,llvm22 -o llgo-dev ./cmd/llgo
go run ./benchmark/wasm -size-only -size-llgo ./llgo-dev -tinygo tinygo -out /tmp/llgo-wasi-size
```

To include the matrix in `run.sh`, set `LLGO_WASM_BENCH_SIZE_MATRIX=1`.
Use `LLGO_WASM_BENCH_DEV_TAGS` to select tags for the dev compiler and
`TINYGO` to select the TinyGo executable.

PR comparisons use the same fixture paths and toolchains for base and head.
An older base that lacks the size flags still records the historical series.
The matrix enables the ordinary LLGo package cache for all size modes. The
historical build-time series still uses its existing cache settings and
metric names.
