# SIMD behavior tests

Add executable SIMD regressions as ordinary `Test*` functions in this package.
Use `goexperiment.simd` and architecture build tags to select applicable APIs.
Shared behavior tests run under official Go and LLGo. LLGo-specific behavior
uses an additional `llgo` build tag; LLVM IR/instruction assertions belong in
compiler tests instead.

## Implemented SIMD128 coverage

The table-driven lowering implements the applicable official declarations on
amd64, arm64, and wasm. The architecture's API determines which type/method
combinations exist; the families below do not imply identical APIs on every
architecture.

- Array/slice loads and stores, broadcast, and lane extraction/insertion.
- Add/subtract/multiply, floating divide, negation, absolute value, square root,
  rounding, and target-specific floating min/max semantics.
- Integer bitwise operations, saturated add/subtract, min/max, uniform shifts,
  and the applicable variable shifts.
- Numeric bitcasts, signedness conversions, and SIMD128 numeric conversions,
  with defined results for NaN and out-of-range floating inputs.
- Four mask shapes, comparisons, mask bitmaps where provided, and the official
  Go helpers for masking and conditional selection.
- Byte lookup with out-of-range zeroing on arm64/wasm, and baseline-safe amd64
  byte/word permutations and negative-index zeroing.

Tests cover NaNs, signed zero, wrapping/saturation boundaries, large shift
counts, every byte lookup index, element-aligned addresses, short slices,
nil arrays, and values crossing interfaces, tuples, closures, package calls,
defer/recover, goroutines, and scheduling. The hex-encoding workload checks
all tail lengths, unaligned input, and output sentinels against `encoding/hex`.
It is a correctness workload, not a performance benchmark.

Go helper bodies are compiled normally. Remaining bodyless intrinsic
implementations panic with their symbol name; `unimplemented_llgo_test.go`
checks direct, indirect, deferred, and linkname calls. SIMD reflection,
portable `simd` specialization, general FMV, and 256/512-bit vectors remain
outside this implemented stage.

## Running

From the repository root, using the built LLGo binary on `PATH`:

```sh
GOEXPERIMENT=simd go test -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O0 -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -count=1 ./test/simd/...
GOEXPERIMENT=simd GODEBUG=cpu.all=off llgo test -O2 -lto=full -count=1 ./test/simd/...

GOEXPERIMENT=simd GOOS=wasip1 GOARCH=wasm go test -exec=wasmtime -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -target emscripten -emulator -count=1 -timeout=2m ./test/simd/...
```

LLGo WASI execution uses the threaded W32 profile and WAMR (`iwasm`), built
with `bash dev/build_iwasm.sh`. The official Go WASI comparison uses Wasmtime.
Emscripten execution requires a compatible SDK and Node.js. The local
qualification used Go 1.27.0, LLVM 22.1.8, Emscripten 6.0.8, and Node 24.19.0.
Existing CI runs native amd64/arm64 at O0/O2 and WASI at O2.

The current WAMR 2.4.5 classic-interpreter profile rejects `v128` function
types with `unknown value type`, even though its build reports SIMD enabled.
The WASI suite is therefore blocked at module loading; the same failure is
reproducible on the main-branch baseline and an import-free `v128` identity
module. Emscripten provides executable Wasm SIMD coverage independently.

The complete O0 Emscripten test executable exceeds Node's local-variable
limit. A small executable covers SIMD initialization, cross-package calls,
recovery, and scheduling at O0 without importing the testing framework:

```sh
GOEXPERIMENT=simd llgo run -O0 -target wasi -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O0 -target emscripten -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O2 -lto=thin -target emscripten -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O2 -lto=full -target emscripten -emulator ./test/simd/testdata/boundary
```

Emscripten's JavaScript SjLj wrappers cannot carry `v128`. Calls in functions
containing `setjmp` use a memory bridge for vector arguments/results; ordinary
Wasm vector calls retain their vector ABI. These bridges cannot be inlined,
and bodies retaining vector calls cannot be moved into recovery functions by
a later backend/LTO inliner. The earlier LLVM optimization still runs at the
requested level.

## CPU initialization and reference boundaries

Effective CPU flags are initialized before `archsimd` and user initialization,
using the official `GODEBUG=cpu.*` policy. The compiler tests compare startup
queries with official Go and check the platform hooks:

```sh
go test ./cl ./ssa ./internal/build -run '^(TestSIMD|TestCPUInitialization|TestEmscriptenSIMDCallBridge)' -count=1
```

Generic LLVM legalization keeps the implemented native operations valid for
the compilation baseline. The amd64 assembly check explicitly uses
`GOAMD64=v1` and rejects AVX instructions. Tests needing official AVX512 APIs
use separate LLGo-only scalar-reference cases when that hardware is not
available; this does not qualify native AVX512 execution.

Reference boundaries:

- Official Go 1.27.0 on WASI does not panic for nil SIMD array loads/stores.
  `memory_nil_test.go` therefore checks the nil-panic contract under LLGo on
  every target and under official native Go. Shared WASI tests still cover
  short-slice bounds; LLGo WASI retains both nil assertions.
- Running the full standard-library `internal/cpu` test package under LLGo
  currently hits duplicate symbols from the original/test package archives.
  This was reproduced without the CPU-initialization change. The dedicated
  CPU startup executables and target-hook tests pass independently.
