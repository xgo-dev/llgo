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

- Horizontal pairwise addition on amd64/arm64 and subtraction on amd64,
  including wrapping and saturating integer pairs. Grouped AMD64 operations
  pair adjacent lanes separately in each 128-bit half.
- AMD64 sign and high-half products, widening even-lane products, pair dot
  products with optional saturation, sums of eight absolute byte differences,
  and alternating add/subtract and fused multiply-add/subtract.
- Array/slice loads and stores, broadcast, and lane extraction/insertion.
  AMD64 masked stores preserve disabled lanes without accessing their memory.
- Add/subtract/multiply, floating divide, negation, absolute value, square root,
  rounding, fused native multiply-add, and target-specific floating min/max semantics.
- Integer bitwise operations, saturated add/subtract, min/max, uniform shifts,
  the applicable variable shifts, rounded averages, uniform rotates, widening
  low- and high-half multiplication, and carryless multiplication. ARM64 also
  supports leading-zero and leading-sign-bit counts.
- Integer low/high-lane extension, truncating and saturating narrowing with zero-filled upper
  lanes, and AMD64 concatenating saturation. Sign/zero extension and signed or
  unsigned saturation follow each architecture's official declarations. The
  current WASM API provides extension methods but no saturating narrowing methods.
- Numeric bitcasts, signedness conversions, and SIMD128 numeric conversions,
  with defined results for NaN and out-of-range floating inputs.
- Four mask shapes, comparisons, mask bitmaps where provided, and the official
  Go helpers for masking and conditional selection.
- Native integer interleave-low/high operations, ARM64 even/odd concatenation
  and interleave, and AMD64 SIMD256 interleave within each 128-bit group.
- AMD64 128/256-bit selection from two vectors, including independent 128-bit
  groups and 256-bit selection of whole 128-bit halves. Constant and variable
  selectors preserve lane bits; invalid half selectors retain a recoverable panic.
- AMD64 128/256-bit `PermuteScalars*` dword and high/low word shuffles,
  including all 256 packed selectors, public selector masking, preserved
  word halves, method values, callable intrinsics, and AVX2 FMV.
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

Go helper bodies are compiled normally. The declaration test loads the selected
Go toolchain's `simd/archsimd` source for each target and checks every bodyless
declaration against the compiler's lowering table.

The integer resize suite covers every applicable SIMD128 `ExtendLo`, `ExtendHi`,
`SaturateTo`, and `SaturateTo*Concat` declaration. The conversion/resize suite
extends coverage to 128/256-bit `ExtendTo`, cross-width `ExtendLo`, `TruncTo`,
saturation and grouped concatenation, plus ARM64/WASM low-lane conversions.
Shared cases run under official Go and LLGo. AMD64 unary saturation, truncation,
and several 64-bit/unsigned conversions require AVX512 in the official compiler,
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

## AMD64 SIMD256

The ten numeric SIMD256 types and four mask shapes use native vector storage
alignment (32 bytes on amd64), including arrays and nested structs, following
the storage rules above. Computation uses LLVM vectors. Operations cover loads/stores, broadcasting,
arithmetic and bitwise operations, comparisons and masks, same-width bitcasts,
uniform integer shifts, and 128-bit half extraction/insertion. Official Go helper
bodies are reused, including byte multiplication and partial memory operations.
The portable256 bridge's intrinsic dependencies are also implemented: integer
and floating min/max, byte/word saturating add/subtract, rounded unsigned
averages, square root, fused multiply-add, lane-preserving numeric/signedness
conversions, and carryless multiplication independently within each 128-bit
group. Horizontal pairwise addition and subtraction also operate independently
within each 128-bit group. `InterleaveLoGrouped` and `InterleaveHiGrouped` likewise
interleave integer lanes independently within each 128-bit group.
The bridge call-graph test covers all 350 entries in Go 1.27.0, including
their private intrinsic dependencies. The applicable 128/256-bit numeric
conversions, integer extension/truncation/saturation, grouped concatenating
saturation and 256-bit floating rounding are also implemented. Shape checks
preserve selected lanes, zero-filled upper lanes and 128-bit group order;
invalid floating-to-integer inputs retain the architecture's defined results.
The same group boundaries apply to SIMD512.

On amd64, 256-bit and 512-bit Go vector parameters and results use the native
vector types through package boundaries and function values. YMM arguments
and results require AVX, and ZMM arguments and results require AVX-512F.
Win64 uses vectorcall for these signatures. Ordinary aggregate and stack
argument rules still apply.

A Go guard such as `if archsimd.X86.AVX512()` skips the guarded call when
the effective query is false. If the program reaches an unconditional wide
call, its caller must have hardware and OS support for the register width.
The runtime records the OS-enabled YMM and ZMM widths before applying Go's
feature overrides. For example, with `GOAMD64=v1` and `GODEBUG=cpu.all=off`,
a caller can still use the fixed ZMM ABI on an AVX-512F machine with ZMM
state enabled. On a narrower machine the caller can take its scalar paths.
Reaching a call that requires an unavailable register width faults at that site. Each explicit SIMD
operation in the callee retains its own feature requirements.
`TestSIMD512FixedRegisterCall` exercises both branches. The Sapphire Rapids
SDE run also checks all 16 transferred lanes with Go's optional CPU features
disabled.

Runtime tests cover scalar-reference arithmetic, distinct upper/lower halves,
mask bitmaps, partial stores at a guard page, initialization, many arguments,
multiple results, source mutation during calls, interfaces, closures, recovery
and goroutine scheduling. The native amd64 matrix runs wide calls at O0/O2 and with
Thin/Full LTO when the required predicates are enabled. With the predicates
disabled it checks initialization and the scalar branches of guarded callers.
Compiler tests check the register arguments, results and baseline instruction
safety on Linux and Windows.
Bridge tests additionally cover every byte pair for saturation/average,
word saturation boundaries, NaNs and signed zeros, invalid floating conversions,
single-rounding/subnormal FMA cases, and all four carryless lane selectors.

Official Go controls require AVX2. Its mask bitmap constructors and some other
256-bit operations require AVX512. The LLGo-only bridge and integer multiply
tests exercise their 256-bit implementations on AVX2 hosts.
Go 1.27.0 also restores only XMM0 on a recovered SIMD256 return, losing the
upper half. LLGo keeps an executable recovery regression; the official failing
control is retained separately (run on an AVX2-capable amd64 target):

```sh
GOEXPERIMENT=simd go run ./test/simd/testdata/upstream-simd256-recover
GOEXPERIMENT=simd llgo run ./test/simd/testdata/upstream-simd256-recover
```

Portable `simd` also generates and selects a 256-bit variant on amd64, using
these bridge operations. By-value native C bindings using these wide `archsimd`
types receive a diagnostic; explicit memory interfaces remain available.
By-value `archsimd` values across native C boundaries require a separate ABI contract.

## AMD64 SIMD512

The compiler recognizes all ten numeric 512-bit types and four mask shapes.
Their Go size, alignment and vector call ABI follow the same rules as
the 256-bit family. Grouped permutations and saturating packing operate within
128-bit groups. Mask bitsets can carry up to 64 active lanes.

The AMD64 implementation includes compression and expansion, vector and scalar
funnel shifts, truth tables, widening products, cryptographic rounds and scaled
floating-point operations. Before calling a private x86 instruction leaf, the
caller checks the effective CPU features published by the runtime. A missing
feature takes the recoverable panic path. The instruction function takes and returns vectors
by value using XMM, YMM or ZMM registers, including on Win64. LLVM can inline
it into a caller with compatible target features. Go functions use the
platform ABI for their declared signatures.

For SHA-1 rounds and scaled rounding, a constant selector produces a leaf
with the immediate encoded in its instruction. A runtime selector uses a
shared leaf that switches on the selector bits. Both paths use the same CPU
feature check.
`GODEBUG=simd=512` selects the 512-bit portable representation when eligible.

On Linux/amd64, run the Sapphire Rapids instruction tests with Intel SDE:

```sh
LLGO=/path/to/llgo SDE=/path/to/sde64 bash dev/test_simd_amd64_sde.sh
```

The script compares the complete upstream instruction case list against each
compiler's passing tests and checks the selected portable width. It runs on
a host that can execute the built Linux binaries and find their runtime
libraries. Use hardware runs for throughput measurements.

## Portable SIMD Midway

With `GOEXPERIMENT=simd`, LLGo specializes portable `simd` source on amd64,
arm64, and wasm. The first type check identifies dependent declarations;
Midway generates emulated and 128-bit variants, plus 256-bit and 512-bit variants on
amd64, rewrites their references to the official `simd/internal/bridge`, and
type-checks the result before Go SSA.
Importers see both the original declarations and the specialized exports.
Ordinary scalar entry points select their variant through `simd.Emulated()`
and, on amd64, `simd.VectorBitSize()`.
The retained unspecialized vector signatures are metadata and trap if called.

The implementation handles recursive aggregates, aliases, embedded fields,
generic instantiations, methods, closures, interfaces, and cross-package calls.
Layout-dependent constants and array types are rechecked per variant. Global
initializers evaluate only the selected variant, retain their original Go
dependency order, and execute side effects once; multiple `init` functions
retain their order. The loader avoids requesting unused gc export data, which
would otherwise run the official compiler's own Midway before LLGo's checks.

On amd64 the upstream width selector uses effective AVX, AVX2, AVX512 and
carryless-multiply flags. An explicit `+` bypasses the all-feature downgrade
within the available hardware width. `GODEBUG=simd=0` selects emulation.
ARM64 and Wasm retain their 128-bit feature policy.

The native amd64 matrix exercises default, emulated and eligible explicit
widths at O0/O2 and Thin/Full LTO, including CPU overrides and rejected width
requests. Cross-package calls, indirect calls, global
initialization and deferred recovery use the same selected representation.
LLVM tests verify that ARM64 and Wasm omit both the 256-bit and 512-bit variants.

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
or link directives targeting a known SIMD-dependent global: an external storage
binding needs an ABI shared by all concrete representations. Embedded named
strings and byte slices retain their contents in every generated variant.
The function-local multiversioning policy is described below.

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

## Function-local SIMD multiversioning

FMV runs once per amd64 module, after calling conventions are fixed and
before target optimization. Each function is planned from its own vector
operations, the machine signatures at its call sites, its compilation floor
and direct CPU guards. A pointer-only forwarder or a function returning a
CPU query keeps a single body. A function doing wide arithmetic through a
pointer can choose a wider implementation even when it has no wide arguments.

When a function has multiple implementations, its original symbol is their
common entry. Direct calls, calls through function values, recursion and
separately compiled callers all use that symbol. The first call after CPU
initialization caches a physical implementation in the entry's atomic slot.
Earlier calls execute the baseline without caching. The resolver preserves
the fixed argument and return convention on every path.

Midway selects the portable representation first. For example, its 512-bit
branch calls `kernel@simd512`. This symbol is the FMV entry when the kernel
has an additional local requirement such as FMA. A compatible same-width
Midway reference can use this symbol directly. Both the ordinary body and
the FMA body inherit the machine features guaranteed by this representation.
A caller's independent FMA check still calls the ordinary Midway entry.
`GODEBUG=simd=0` selects software portable operations while explicit
`archsimd` queries in the same function still observe the effective CPU state.

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
frame. The physical implementations keep their own debug subprograms and
the original Go display name. These are FMV-specific identity rules; LLGo's
general runtime inline-frame representation remains separate from GoALLC's
GoObj inline-tree format.

The selected body calls wide Go functions and function values at their source
call sites. Two calls to the same callback from different lines retain their
distinct caller PCs under `runtime.CallersFrames`.

Package archives contain a dedicated metadata object with post-backend function
identities, wrapper flags, and PC-line records. Both fresh and cached archives
supply these mappings to the linker; reconstructed frontend modules and frontend
cache exports are not sources for the archived machine code's traceback data.
The final exported C archive omits these compiler-private members; runtime
traceback tables are generated before that merge, and cached package archives
retain their mappings.
The metadata object is native even with LTO and has no code-address relocations.
`TestSIMDFMVArchiveCache` checks cold and warm builds through isolated and
coordinator backends with off/Thin/Full LTO, plus frontend-cache reuse, baseline
CPU overrides, and source-frame identity.

On Windows/amd64, the compiler exposes the native indirect SIMD128 parameter
ABI before creating dispatchers. SIMD128 tail transfers forward the caller's
parameter storage. When a wide vector occupies the seventh or a later parameter
position, the vectorcall forwarder uses an ordinary call so its temporary
storage for overflow arguments remains live until the callee returns.
Aggregate ABI lowering also runs before specialization on every target, so
forwarding preserves byval/sret attributes and caller-owned result storage.
Vector results keep their native register ABI.

The x86 table records which effective Go predicates authorize each register
width and instruction extension. For example, an effective VBMI2 query can
authorize ZMM registers while the composite AVX512 query remains false.
The baseline, a fixed vector signature and a selected Midway representation
provide machine capabilities. Only the compile-time Go baseline or a query
tested by the resolver licenses folding that query to true.

Run the native matrix with:

```sh
LLGO=/path/to/llgo bash dev/test_native_simd.sh
```

On amd64 it runs O0 without LTO and O2 without LTO, with ThinLTO, and with
Full LTO. Each binary runs the complete SIMD suite, then the FMV and bridge-operation tests with
AVX2, AVX, FMA, and all optional CPU features disabled in separate processes.
The same overrides also exercise integer extension and narrowing, horizontal
pair operations, integer interleaving, pair selection and half shuffles, scalar
permutations, numeric conversions, float widening/narrowing, and 256-bit rounding.
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

Run from the repository root with the built LLGo binary on `PATH`. Wide
archsimd tests compare results with scalar references on CPUs that satisfy
their feature checks.

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

## Dynamic half-selection benchmark

On native amd64, compare predictable and deterministic random selectors with
the same binary. The noinline kernel retains dynamic selectors; its AVX2 guard
exercises FMV. With `cpu.all=off`, the AVX2 query is false.

```sh
GOAMD64=v1 GOEXPERIMENT=simd llgo test -O2 -c -o /tmp/simd-half.test ./test/simd
/tmp/simd-half.test -test.run='^$' -test.bench='^BenchmarkSIMDPairShuffleHalves$' -test.count=6
GODEBUG=cpu.all=off /tmp/simd-half.test -test.run='^$' -test.bench='^BenchmarkSIMDPairShuffleHalves$' -test.count=6
```

The result includes call/dispatch overhead and a dependent sequence of shuffles;
it is not an application-level speedup. Inspect both selector distributions,
since whole-half selection may lower to branches.

## CPU initialization and reference boundaries

Effective CPU flags are initialized before `archsimd` and user initialization,
using the official `GODEBUG=cpu.*` policy. The compiler tests compare startup
queries with official Go and check the platform hooks:

```sh
go test ./cl ./ssa ./internal/build -run '^(TestSIMD|TestCPUInitialization|TestEmscriptenSIMDCallBridge)' -count=1
```

Generic LLVM legalization handles baseline operations. CPU-guarded leaves
carry the features required by instruction-specific operations. The amd64
assembly check uses `GOAMD64=v1` and rejects AVX instructions in baseline
functions. The Linux O2 portable integer-add test also checks for YMM and ZMM
adds in the respective Midway implementations. The native gate exercises
AVX512 paths on capable hosts and scalar branches on lower baselines.
For emulated AVX512 execution, run `dev/test_simd_amd64_sde.sh` on Linux/amd64.

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

### Guarded x86 feature combinations

The planner considers the YMM and ZMM widths needed by the function. A single
guarded instruction family, such as FMA in a hot loop, can select a whole
function body. For mixed families, the x86 policy projects its common levels
(AVX2, AVX2 with FMA, and AVX2 with FMA and AVX512) onto the actual local
requirements. Whole-function version counts are bounded by these levels and
register widths, independently of how many source checks occur.

Independent extensions keep their effective Go branches. Where a selected
body needs an instruction beyond its target features, the guarded operation
can execute in a private outlined leaf. For example, `SHA256TwoRounds`
already runs in a private function compiled with SHA support. Its caller
keeps the Go guard and runtime safety check. These checks alone create no
SHA version of the caller. A guarded FMA lowered to an instruction in the
caller can still specialize that function.

A function with a scalar signature can select a body with enough register
width for a wide call. Its source guards and scalar exits keep their original
order. An outlined instruction also uses a register signature compatible
with its calling body.

Guard recognition uses direct official CPU queries, including negated early
exits. Saved booleans and predicate-returning helpers retain ordinary Go
control flow. `TestSIMDSavedGuard` and `TestSIMDPredicateHelper` check the
observable results and helper side effects. `TestSIMD512VBMI2WideCall`
exercises a cross-package wide call with the AVX512 bundle disabled.
