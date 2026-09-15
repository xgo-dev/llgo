# WebAssembly build and size benchmarks

The harness builds the existing `benchmark/binary_size` examples `cprintf`,
`println`, and `fmtprintf` with LLGo for all five supported execution entries:
J32/GoJS, W32/raw `wasip1`, J32/Emscripten, J64/Emscripten Memory64, and
W32/WASI. Each example/profile records the Wasm module size, generated
JavaScript glue size (zero when absent), and build time.
These are build/size measurements, not runtime or official-Go ABI acceptance.

Official Go size references cover `println` and `fmtprintf` on `js/wasm` and
`wasip1/wasm`. There is no official-Go `cprintf` reference: that example calls
C `printf` through LLGo's C interop, which official Go does not provide on wasm.

The existing unqualified metric names continue to mean `println`, preserving
its benchmark history. New `cprintf/` and `fmtprintf/` metric prefixes identify
the other examples. Artifacts are stored separately for every example,
profile, and compiler.

From the repository root, with the usual LLVM, Emscripten and Binaryen tools:

```sh
benchmark/wasm/run.sh "$PWD" /tmp/llgo-wasm-bench /tmp/llgo-wasm-bench-results
```

The result directory is replaced on each run; use a dedicated output directory.
To reuse a compiler already built for the source revision:

```sh
go run ./benchmark/wasm -root "$PWD" -llgo /path/to/llgo \
  -out /tmp/llgo-wasm-bench-results -build-runs 1
```

The `println` profile builds have one warm-up followed by measured builds; the
CLI defaults to three measurements and reports their median. `cprintf` and
`fmtprintf` are size-only cases and build once per profile without redundant
warm-ups. The CI wrapper uses `LLGO_WASM_BENCH_BUILD_RUNS=1`, so each revision
still performs 20 LLGo builds: ten for timed `println` and ten for the two
additional size workloads. Exact size coverage expands without increasing the
previous build count, adding CI jobs, or repeating dependency installation. A
single CI timing sample is noisier than the default median; base and head use
the same harness and setting sequentially on the same runner. Go references
are built once each.

The existing WebAssembly benchmark job tests this harness and measures both PR
base and head, publishing their results through `.github/llgo-wasm-benchmark.yml`.
