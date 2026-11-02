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
  AMD64 masked stores preserve disabled lanes without accessing their memory.
- Add/subtract/multiply, floating divide, negation, absolute value, square root,
  rounding, fused native multiply-add, and target-specific floating min/max semantics.
- Integer bitwise operations, saturated add/subtract, min/max, uniform shifts,
  the applicable variable shifts, rounded averages, uniform rotates, widening
  low-half multiplication, and carryless multiplication.
- Integer low/high-lane extension, saturating narrowing with zero-filled upper
  lanes, and AMD64 concatenating saturation. Sign/zero extension and signed or
  unsigned saturation follow each architecture's official declarations. The
  current WASM API provides extension methods but no saturating narrowing methods.
- Numeric bitcasts, signedness conversions, and SIMD128 numeric conversions,
  with defined results for NaN and out-of-range floating inputs.
- Four mask shapes, comparisons, mask bitmaps where provided, and the official
  Go helpers for masking and conditional selection.
- Byte lookup with out-of-range zeroing on arm64/wasm, and baseline-safe amd64
  byte/word permutations and negative-index zeroing. Constant permutations
  fold to vector shuffles at O2; dynamic indices use a per-lane fallback.

Tests cover NaNs, signed zero, wrapping/saturation boundaries, large shift
counts, every byte lookup index, element-aligned addresses, short slices,
nil arrays, exhaustive byte/word resize inputs, saturation boundaries at every
integer width, and values crossing interfaces, tuples, closures, package calls,
defer/recover, goroutines, and scheduling. The hex-encoding workload checks
all tail lengths, unaligned input, and output sentinels against `encoding/hex`.
It is a correctness workload, not a performance benchmark.

Go helper bodies are compiled normally. Remaining bodyless intrinsic
implementations panic with their symbol name; `unimplemented_llgo_test.go`
checks direct, indirect, deferred, and linkname calls. SIMD reflection,
general FMV, and 256/512-bit vectors remain
outside this implemented stage.

The integer resize suite covers every applicable SIMD128 `ExtendLo`, `ExtendHi`,
`SaturateTo`, and `SaturateTo*Concat` declaration. Shared cases run under official
Go and LLGo; AMD64 unary saturation requires AVX512 in the official compiler,
so those cases use scalar references under LLGo at the v1 baseline. LLVM
verification and O0/O2 code generation cover Linux/Windows amd64, ARM64, and
WASM, including callable intrinsic definitions. This does not add wider-vector
ABIs or instructions absent from an architecture's Go API.

## Native archsimd storage alignment

LLGo gives `archsimd` vector types native vector alignment, including ordinary Go
variables, defined types, arrays and containing structs. SIMD128 is 16-byte
aligned on the supported targets. This intentionally differs from the official
Go implementation: a byte followed by `archsimd.Float32x4` has the vector at
offset 16 and total size 32. Source `unsafe.Sizeof`/`Alignof`/`Offsetof`, LLVM
storage and reflection agree. Stack/global storage, heap allocation, slice
growth, channel buffers and reflection allocations honor this alignment.
Zero-sized SIMD arrays retain their alignment; a final zero-sized field still
receives Go's in-object-address padding.

Array Load/Store APIs retain the scalar array's alignment requirements; an
aligned vector result does not require aligned input. This storage rule does
not change the Go/C calling convention or add support for new native aggregate
signatures.

Vector storage cannot appear by value in map keys or values, including inside
arrays or structs. Use an array of scalars or a pointer instead. Layout is
recognized directly from type identity, without import-order prerequisites.

On native targets, `reflect.Call`, `CallSlice`, and `MakeFunc` reject signatures
that pass or return vector storage by value through libffi. This includes arrays,
structs, and zero-length vector fields. Pointers to vectors and slices of vectors
use their ordinary scalar ABI and remain supported. WASM reflection bridges
continue to support SIMD128 values.

## Portable SIMD Midway

With `GOEXPERIMENT=simd`, LLGo specializes portable `simd` source on amd64,
arm64, and wasm. The first type check identifies dependent declarations;
Midway generates emulated and 128-bit variants, rewrites their references to
the official `simd/internal/bridge`, and type-checks the result before Go SSA.
Importers see both the original declarations and the specialized exports.
Ordinary scalar entry points select their variant through `simd.Emulated()`.
The retained unspecialized vector signatures are metadata and trap if called.

The implementation handles recursive aggregates, aliases, embedded fields,
generic instantiations, methods, closures, interfaces, and cross-package calls.
Layout-dependent constants and array types are rechecked per variant. Global
initializers evaluate only the selected variant, retain their original Go
dependency order, and execute side effects once; multiple `init` functions
retain their order. The loader avoids requesting unused gc export data, which
would otherwise run the official compiler's own Midway before LLGo's checks.

This stage emits only emulation and SIMD128. On amd64 the official effective
AVX eligibility rule remains in force, with the maximum compiled width capped
at 128. `GODEBUG=simd=0` selects emulation; explicit unsupported widths fail
instead of selecting an absent variant. ARM64 and wasm retain their official
feature policy. CPU queries and LLVM's instruction-level FMV remain separate
from this representation choice.

`portable/` tests compare shared portable operations with the official Go
toolchain and exercise LLGo's initialization and embedded-field contracts in
LLGo-specific fixtures. Go 1.27.0's own Midway rejects keyed literals containing
embedded portable fields and does not implement the same global-initializer
handling; those fixtures are not included in the native Go control.
The shared tests cover partial memory operations, all numeric lane families,
masks, and architecture conversion as well as cross-package boundaries.
Emulated `ToArch` retains the software representation; `FromArch` accepts
architecture vector types and its hardware round trip is tested separately.

Remaining limits include reflection-driven calls with portable signatures,
bodyless user functions depending on portable types, and inferred global types
requiring another package's unexported type, field, or method names.
The latter two cases produce source diagnostics. Midway also diagnoses
`go:linkname`, `llgo:link`, and `export` bindings on SIMD-dependent globals,
or link directives targeting a known SIMD-dependent global: an external storage binding needs an ABI shared by both concrete
representations. Embedded named strings and byte slices retain their contents
in both variants. Wider variants and general FMV are follow-up work.

For the WASI portable matrix, including explicit guest emulation, run
`LLGO=/path/to/llgo bash dev/test_portable_simd_wasi.sh`. It passes `GODEBUG`
through Wasmer's `--env` option; setting the host variable alone does not
exercise WASI emulation.

## Portable SIMD128 bridge prerequisites

The compiler checks the static call closure of the official Go toolchain's
128-bit `simd/internal/bridge` entries on amd64, arm64, and wasm. Their bodyless
`archsimd` dependencies now have implementations. Midway reuses these entries
for its hardware representation and the official software bridge for emulation.
Go helper bodies retain the toolchain's semantics, including separate multiply
and add for wasm and fused rounding for native `MulAdd`. Windows/amd64 uses
software FMA helpers because MinGW's `fma`/`fmaf` can overflow before cancellation.
The float64 helper follows Go's integer-significand implementation; float32
retains a residual before conversion to prevent double rounding. Other native
targets use LLVM's FMA intrinsic.

Bridge-operation regressions cover exhaustive byte averages, signed and
unsigned overflow boundaries, modulo rotate counts, widening multiplication,
full-width polynomial products, fused cancellation, and partial-store tails.
AMD64 tests cover every 32-bit lane mask, sign-bit mask semantics, all six
masked-store element types, inactive nil pointers, active nil recovery under
LLGo, and stores at a Linux guard-page boundary. Compiler tests verify LLVM IR
and O0/O2 code generation for Linux/Windows amd64, ARM64, and WASM.

Go 1.27's wasm `CarrylessMultiplyOdd` helper reads the second operand from
`x` instead of `y`. Both official Go and LLGo reproduce the incorrect result;
the compiler does not replace this Go body. The failing reproducer is retained
separately from the intrinsic tests:

```sh
GOEXPERIMENT=simd GOOS=wasip1 GOARCH=wasm go run -exec=wasmtime ./test/simd/testdata/upstream-carrylessodd
GOEXPERIMENT=simd llgo run -target wasi -emulator ./test/simd/testdata/upstream-carrylessodd
```

The shared suite checks even-lane carryless multiplication on all three
architectures; native tests also check odd-lane and AMD64 mixed-lane selection.
The closure check establishes implementation coverage, not correctness of every
upstream Go helper.

## CPU-guarded SIMD128 specialization

On amd64, functions calling the official `archsimd.X86.AVX2` query get an
AVX2 version for signatures containing scalars, pointers, slices, strings,
arrays, structs, and multiple results, including nested SIMD128 values. The
original entry loads a cached implementation and tail-forwards to it. A separate resolver selects the baseline or AVX2 body from
the immutable, post-`GODEBUG` CPU snapshot on first use. Before CPU initialization
it tail-forwards to the baseline without caching. Function addresses retain the
original entry. Both bodies fold the selected AVX2 query and remove dead branches
even at O0, after call-boundary ABI lowering and before target optimization.
GOAMD64=v3/v4 already require AVX2, so they fold the query without generating extra versions,
a resolver, or a cache slot.

Direct calls from specialized code to eligible SIMD128 functions use matching
AVX2 versions, including across packages without LTO. Only functions with Go
bodies or compiler-generated SIMD intrinsic bodies promise those entries;
bodyless assembly declarations retain their original calls. Ordinary calls
and indirect calls retain the original entry. Arbitrary feature
combinations are not yet specialized.

Midway's 128-bit bridge types also qualify for these direct-call variants,
including when nested in arrays and structs. The compiler validates the bridge
identity and storage against the corresponding official archsimd type while
retaining its aggregate ABI. This lets an AVX2 implementation call another
package's specialized portable function without LTO. Emulated bridge types do
not qualify as vector callees. A CPU-querying function can still specialize in
either representation: `simd=0` does not imply that AVX2 is disabled, and a
128-bit representation does not imply that AVX2 is enabled.

The transform is implemented in Go using the LLVM Go API. It discovers CPU
queries, clones eligible functions, rewrites direct calls and creates dispatch
blocks without registering a custom LLVM pass. Generic LLVM cloning and local
CFG utilities preserve instruction and debug metadata and remove dead branches
even at O0. The transform runs independently of the optimization level and LTO
plugin. Specialized functions retain their Go source identity and get distinct
runtime PC-line records.

Inlining follows the same FMV policy as GoALLC: the public dispatcher and the
physical implementations remain eligible for normal, feature-compatible LLVM
inlining. The lazy resolver is noinline, and its pre-initialization baseline
edge has a call-site noinline attribute so it stays tail-only. Explicit source
noinline directives and disabled inlining remain honored. LLGo's existing
runtime-frame requirements remain on the physical body, not the synthetic
dispatcher. LLVM lowers a cloned musttail call to an ordinary call when the
dispatcher is inlined into a caller with a continuation.

The dispatcher retains its runtime function identity but has no synthetic debug
subprogram. LLVM therefore attributes an inlined dispatcher to its real call
site, preserving outer inline chains without adding a second source execution
frame. The baseline/AVX2 implementations keep their own debug subprograms and
the original Go display name. These are FMV-specific identity rules; LLGo's
general runtime inline-frame representation remains separate from GoALLC's
GoObj inline-tree format.

On Windows/amd64, the compiler exposes the native indirect SIMD128 parameter
ABI before creating dispatchers. Tail transfers forward the caller's parameter
storage, avoiding vector temporaries in a resolver frame that the jump releases.
Aggregate ABI lowering also runs before specialization on every target, so
forwarding preserves byval/sret attributes and caller-owned result storage.
Vector results keep their native register ABI.

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
Full LTO. Each binary runs the complete SIMD suite, then the FMV and bridge-operation tests with
AVX2, AVX, FMA, and all optional CPU features disabled in separate processes.
The same overrides also exercise integer extension and narrowing.
The FMV tests cover initialization, direct and indirect entry calls,
cross-package calls, aggregate value copies and multiple results, closures,
independent CPU queries, and source-level stack traces.
The portable suite also runs each amd64 profile with the same CPU overrides
and explicit emulated/128-bit requests, including these requests crossed
with AVX2, AVX, FMA, and all optional CPU features disabled. The composition
fixtures cover cross-package aggregates and arrays, function values, methods,
interfaces, closures, generics, and initialization. On other native SIMD targets the script
retains the O0/O2 suite and an additional portable emulation run.

## Running

From the repository root, using the built LLGo binary on `PATH`. The official
Go amd64 comparison excludes helpers that require AVX512; LLGo tests their
baseline legalization against scalar references at GOAMD64=v1:

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
