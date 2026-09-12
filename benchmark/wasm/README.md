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

The `println` profile builds have one warm-up followed by measured builds; the CLI defaults to three measurements and reports their median. `cprintf` and `fmtprintf` are size-only cases and build once per profile. `reflectcall` is also size-only on the JavaScript profiles, while W32/WASI gets one warm-up and measured builds to expose typed-bridge compile cost. With `LLGO_WASM_BENCH_BUILD_RUNS=1`, CI performs 26 LLGo builds and six official-Go reference builds per revision. A single CI timing sample is noisier than the default median, but base and head run sequentially with the same fixtures and settings on one runner.

The existing WebAssembly benchmark job tests this harness and measures both PR base and head, publishing their results through `.github/llgo-wasm-benchmark.yml`.
