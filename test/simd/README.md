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
  byte/word permutations and negative-index zeroing. Constant permutations
  fold to vector shuffles at O2; dynamic indices use a per-lane fallback.

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

## CPU-guarded SIMD128 specialization

On amd64, functions calling the official `archsimd.X86.AVX2` query get an
AVX2 version when their LLVM signature contains only scalars, pointers, and
vectors up to 128 bits. The original entry checks the effective runtime query
and tail-forwards to that version only when enabled. Function addresses retain
the original entry. The AVX2 version folds that query and removes dead branches
even at O0, before aggregate ABI lowering and target optimization.

Direct calls from specialized code to eligible SIMD128 functions use matching
AVX2 versions, including across packages without LTO. Only functions with Go
bodies or compiler-generated SIMD intrinsic bodies promise those entries;
bodyless assembly declarations retain their original calls. Ordinary calls
and indirect calls retain the baseline entry. Aggregate signatures and
arbitrary feature combinations are not yet specialized.

The transform is compiled into LLGo through a small C interface to LLVM C++.
It runs independently of the optimization level and LTO plugin. LLVM itself
uses the existing build's library linkage. Specialized functions retain their
Go source identity and get distinct runtime PC-line records.

`GOAMD64` still controls the compilation baseline. The dispatcher observes
the post-`GODEBUG=cpu.*` AVX2 query; instruction capability does not imply
other observable query results. In particular, the AVX and FMA queries remain
dynamic when AVX2 is enabled. This does not implement portable `simd` width
selection or 256/512-bit vector calling conventions.

Run the native matrix with:

```sh
LLGO=/path/to/llgo bash dev/test_native_simd.sh
```

On amd64 it runs O0 without LTO and O2 without LTO, with ThinLTO, and with
Full LTO. Each binary runs the complete SIMD suite, then the FMV tests with
AVX2, AVX, FMA, and all optional CPU features disabled in separate processes.
The FMV tests cover initialization, direct and indirect entry calls,
cross-package calls, independent CPU queries, and source-level stack traces.
On other native SIMD targets the script retains the O0/O2 suite.

## Running

From the repository root, using the built LLGo binary on `PATH`:

```sh
GOEXPERIMENT=simd go test -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O0 -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -count=1 ./test/simd/...
GOEXPERIMENT=simd GODEBUG=cpu.all=off llgo test -O2 -lto=full -count=1 ./test/simd/...

GOEXPERIMENT=simd GOOS=wasip1 GOARCH=wasm go test -exec=wasmtime -count=1 ./test/simd/...
GOEXPERIMENT=simd llgo test -O0 -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -lto=thin -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -lto=full -target wasi -emulator -count=1 -timeout=2m ./test/simd/...
GOEXPERIMENT=simd llgo test -O2 -target emscripten -emulator -count=1 -timeout=2m ./test/simd/...
```

LLGo WASI execution uses the threaded W32 profile and Wasmer 7.5.0, installed
with `bash dev/install_wasmer.sh`. The runner supports SIMD, shared-memory
threads, and standard Wasm exception handling; Wasmer selects an available
backend automatically. The official Go WASI comparison uses Wasmtime.
Emscripten execution requires a compatible SDK and Node.js. The local
qualification used Go 1.27.0, LLVM 22.1.8, Emscripten 6.0.8, and Node 24.19.0.
CI runs native amd64/arm64 and WASI at O0/O2. The JavaScript matrix runs
GoJS (`GOOS=js GOARCH=wasm`), Emscripten, and Emscripten Memory64 with the
default JavaScript SjLj/Asyncify configuration:

```sh
LLGO=/path/to/llgo dev/test_wasm_simd.sh
```

Each profile runs the O0 boundary executable, the complete O2 test suite
with LTO disabled, ThinLTO, and Full LTO, and the O3 boundary executable with
ThinLTO and Full LTO. The O3 runs guard against late argument promotion
replacing a bridge pointer with a vector parameter. The LTO modes pass `-flto=thin` or
`-flto=full` to both compilation and final linking. The requested `-O` level
also reaches compilation and emcc's post-link pipeline. LTO additionally gets
an explicit `--lto-O0` through `--lto-O3`; `-Os`/`-Oz` use `--lto-O2` and retain
their size attributes and post-link size optimizations. Earlier browser builds
accepted `-lto` but did not forward these driver flags, so their successful
runs did not qualify actual link-time optimization. Archives use the SDK's
`emar` to index bitcode produced by its Clang; `LLGO_AR` remains an explicit
override.

The complete O0 Emscripten test executable exceeds Node's local-variable
limit. A small executable covers SIMD initialization, cross-package calls,
recovery, and scheduling at O0 without importing the testing framework:

```sh
GOEXPERIMENT=simd llgo run -O0 -target wasi -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O0 -target emscripten -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O0 -target emscripten-memory64 -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd GOOS=js GOARCH=wasm llgo run -O0 -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O2 -lto=thin -target emscripten -emulator ./test/simd/testdata/boundary
GOEXPERIMENT=simd llgo run -O2 -lto=full -target emscripten -emulator ./test/simd/testdata/boundary
```

The default GoJS and Emscripten JavaScript SjLj wrappers cannot carry `v128`. Calls in functions
containing `setjmp` use a memory bridge for vector arguments/results; ordinary
Wasm vector calls retain their vector ABI. These bridges cannot be inlined,
and bodies retaining vector calls cannot be moved into recovery functions by
a later backend/LTO inliner. Volatile vector loads in the bridge also prevent
LLVM 22's O3 argument promotion from replacing its pointer parameters with
vectors, even though the bridge is `optnone`. The earlier LLVM optimization
still runs at the requested level. When compilation and linking select native Wasm SjLj (for
example, `EMCC_CFLAGS='-fwasm-exceptions -sSUPPORT_LONGJMP=wasm'`), these
bridges and late-inlining restrictions are unnecessary and omitted. Memory64
IR still uses explicit JS SjLj codegen and retains the bridge.

WASI SIMD execution is also qualified at O2 with Thin and Full LTO using the
commands above and Wasmer 7.5.0.

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

- Official Go 1.27.0 initializes Windows CPU flags before reading `GODEBUG`.
  The CPU startup test compares default hardware detection with Go and checks
  LLGo overrides against explicit expectations; other native hosts also retain
  the Go comparison for each override. Both amd64 compilers use `GOAMD64=v1`.

- Official Go 1.27.0 on WASI does not panic for nil SIMD array loads/stores.
  `memory_nil_test.go` therefore checks the nil-panic contract under LLGo on
  every target and under official native Go. Shared WASI tests still cover
  short-slice bounds; LLGo WASI retains both nil assertions.
- Running the full standard-library `internal/cpu` test package under LLGo
  currently hits duplicate symbols from the original/test package archives.
  This was reproduced without the CPU-initialization change. The dedicated
  CPU startup executables and target-hook tests pass independently.
