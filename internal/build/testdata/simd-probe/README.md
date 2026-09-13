# SIMD compiler probes

These opt-in probes inspect the current compiler pipeline with real Go 1.27
`simd/archsimd` types. They are excluded from the default test build with the
`llgo_simd_probe` build tag. There is no skipped or expected-failure test in the
normal suite.

Run from the repository root with the normal LLGo development dependencies:

```sh
LLGO_SIMD_PROBE_OUT=/absolute/path/to/probe-artifacts \
  go test -tags=llgo_simd_probe ./internal/build \
  -run '^TestSIMDCompilerProbe$' -count=1 -v
```

To inspect one target, use `-run '^TestSIMDCompilerProbe/wasm$'`, replacing `wasm`
with `amd64` or `arm64` as needed. The test itself selects `GOEXPERIMENT=simd` for
fixture compilation. It does not require enabling the experiment for the test
binary. The selected source toolchain must provide all three generated
`simd/archsimd/types_*.go` files.

Each target directory receives:

- `report.json`: compiler HEAD, source Go version/root, target triple/features,
  source file hashes, type layouts, ABI signatures, and unmet requirements.
- `native.s`: Go's assembly listing from compiling the same fixture with
  `GOEXPERIMENT=simd`. This is a native compiler control; LLGo has its own ABI.
- `before-abi.ll`: the fixture module at `Config.ModuleHook`, after Go SSA is
  compiled into LLGo/LLVM, before aggregate and C ABI transformations.
- `after-abi.ll`: the fixture module returned by `ModeGen`, after those ABI
  transformations. `ModeGen` does not run the ordinary LLVM optimization pipeline.
- `archsimd-init-functions.ll`: generated `archsimd` package initialization
  functions and one layer of their direct helpers, which can expose required
  operations beyond the fixture's `Add`.

The type report separates ordinary `[4]float32` storage, canonical SIMD Go
storage, loaded frontend sizes, LLGo sizes, and actual LLVM global storage.
`llvm_natural_vector_layout` measures a separately constructed LLVM vector for
comparison. It is **not evidence that LLGo emitted such a value**. The SIMD Go
storage contract requires alignment 8, including on wasm; LLVM's default vector
alignment is a separate quantity.

The fixture covers 128-bit vectors on all three targets and 256/512-bit vectors
on amd64: addition, direct parameters/results, pointer loads/stores, a call
chain, and structs with bytes before and after a vector. ABI signatures are
observations, without asserting that an existing aggregate carrier constitutes
SIMD support. The JSON also records direct initialization callees, their
immediate helper callees, and archsimd declarations for which no body was
observed among generated modules. This does not compute reachability or
establish final-link availability; native objects are outside `ModeGen`.

The probe exits with failure for unmet Go SIMD storage requirements, invalid
LLVM, unsuccessful compilation, or missing natural vector `fadd` in the
representative `Add` functions. These are pending implementation requirements,
not accepted baseline expectations. As operation lowering and FMV are added,
the checks can become focused regressions at the appropriate compiler stage.
The probes do not link or execute LLGo binaries and do not establish CPU guard
safety, runtime initialization, instruction selection, or full SIMD coverage.

## CPU initialization probe

The `cpu` program reads feature queries during global initialization, `init`,
and `main`. Run native controls and then build with LLGo:

```sh
GOEXPERIMENT=simd go run ./internal/build/testdata/simd-probe/cpu
GODEBUG=cpu.all=off GOEXPERIMENT=simd go run ./internal/build/testdata/simd-probe/cpu
GOEXPERIMENT=simd llgo build -o /tmp/simd-cpu ./internal/build/testdata/simd-probe/cpu
```

Only execute `/tmp/simd-cpu` after a successful build. A failed link is a missing
implementation result, not a measured CPU snapshot. Feature values depend on the
host; on arm64 use `GODEBUG=cpu.pmull=off` as an individual-feature control.
