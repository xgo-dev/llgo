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

Wide Go parameters and results use an explicit memory carrier on amd64,
including indirect calls, cross-package calls and multiple-result tuples.
This carrier is fixed before FMV creates baseline/AVX2 bodies and tail-forwarding
entries. Both implementations use the same pointer ABI; the selected body can
use AVX2 internally without requiring AVX in its caller. Carrier loads/stores
remain volatile so later argument/return promotion, including LTO, cannot
silently recreate a feature-dependent YMM calling convention. This first stage
does not introduce a separate register-vector ABI for calls between fast bodies.

Runtime tests cover scalar-reference arithmetic, distinct upper/lower halves,
mask bitmaps, partial stores at a guard page, initialization, many arguments,
multiple results, source mutation during calls, interfaces, closures, recovery
and goroutine scheduling. The native amd64 matrix includes these tests at O0/O2,
Thin/Full LTO, and with optional CPU features disabled. Compiler tests check the
pointer carriers and baseline instruction safety on Linux and Windows.
Bridge tests additionally cover every byte pair for saturation/average,
word saturation boundaries, NaNs and signed zeros, invalid floating conversions,
single-rounding/subnormal FMA cases, and all four carryless lane selectors.

Official Go controls require AVX2. Its mask bitmap constructors and some other
256-bit operations require AVX512; LLGo's baseline implementations use scalar
references where the control hardware cannot execute those instructions.
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
Their Go size, alignment and indirect call carriers follow the same rules as
the 256-bit family. Grouped permutations and saturating packing operate within
128-bit groups. Mask bitsets can carry up to 64 active lanes.

The AMD64 implementation includes compression and expansion, vector and scalar
funnel shifts, truth tables, widening products, cryptographic rounds and scaled
floating-point operations. Instruction-specific leaves check the effective CPU
predicate before loading their vector operands from caller-owned storage.
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
The AVX2 function multiversioning policy is described below.

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

On amd64, a direct `archsimd.X86.AVX2()` guard over vector work or a call
to a promised AVX2 entry can select an AVX2 version. A function that only
returns the query result keeps its ordinary entry. Supported signatures include
scalars, pointers, slices, strings, aggregates and multiple results. The
original entry loads a cached implementation and tail-forwards to it. A separate resolver selects the baseline or AVX2 body from
the immutable, post-`GODEBUG` CPU snapshot on first use. Before CPU initialization
it tail-forwards to the baseline without caching. Function addresses retain the
original entry. Both bodies fold the selected AVX2 query and remove dead branches
even at O0, after call-boundary ABI lowering and before target optimization.
GOAMD64=v3/v4 already require AVX2, so an AVX2 query alone needs no dispatcher. Optional
predicates can still generate versions and a slot on these baselines.

Direct calls from specialized code to eligible SIMD128 functions use matching
AVX2 versions, including across packages without LTO. Only functions with Go
bodies or compiler-generated SIMD intrinsic bodies promise those entries;
bodyless assembly declarations retain their original calls. Ordinary calls
and indirect calls retain the original entry. AVX2 guards over scalar-signature local helpers and other guarded
feature combinations use [local guarded specializations](#guarded-x86-feature-combinations).

Midway's 128-bit bridge types also qualify for these direct-call variants,
including when nested in arrays and structs. The compiler validates the bridge
identity and storage against the corresponding official archsimd type while
retaining its aggregate ABI. This lets an AVX2 implementation call another
package's specialized portable function without LTO. Emulated bridge types do
not qualify as vector callees. The emulated path keeps ordinary CPU queries when it performs only scalar
work. The selected portable width and the effective AVX2 query are independent.

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
ABI before creating dispatchers. Tail transfers forward the caller's parameter
storage, avoiding vector temporaries in a resolver frame that the jump releases.
Aggregate ABI lowering also runs before specialization on every target, so
forwarding preserves byval/sret attributes and caller-owned result storage.
Vector results keep their native register ABI.

`GOAMD64` still controls the compilation baseline. The dispatcher observes
the post-`GODEBUG=cpu.*` AVX2 query; instruction capability does not imply
other observable query results. On v1/v2 the AVX and FMA queries remain
dynamic when AVX2 is enabled. At v3/v4 both are guaranteed. A guarded implementation is selected by the effective
Go predicates in its runtime mask. For example, VAES uses the VAES bit even
when AVX has been independently disabled. AVXAES uses the combined AVX and
AES result published by the runtime. Midway selects portable width independently.

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

## Dynamic half-selection benchmark

On native amd64, compare predictable and deterministic random selectors with
the same binary. The noinline kernel retains dynamic selectors; its AVX2 guard
exercises FMV, while `cpu.all=off` selects the baseline implementation.

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

Generic LLVM legalization handles baseline operations. CPU-guarded leaves carry the features required by instruction-specific operations. The amd64 assembly check explicitly uses
`GOAMD64=v1` and rejects AVX instructions. Tests needing official AVX512 APIs
use separate LLGo-only scalar-reference cases when that hardware is not
available. On an AVX512 host the native gate exercises the enabled paths.
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

On AMD64 the compiler also recognizes positive branches on the official
`archsimd.X86` predicates. At control-flow joins it retains the predicates
proven on every incoming path. Guarded vector operations, LLVM intrinsics
returning vectors, and direct calls to local SIMD helpers can select an
implementation with those features. A call to an opaque function that returns
a vector provides no local work to specialize. Nested
AVX512 and FMA guards can therefore produce a 512-bit fused implementation.
GOAMD64 v3 guarantees AVX, AVX2 and FMA. The v4 baseline also guarantees
the AVX512 predicate. LLGo folds these queries before generating guarded
versions. Optional predicates continue to read the effective CPU state.

On v1/v2, the public resolver for a mixed AVX2 and AVX512 guard selects
the physical body from both predicates and caches it in one slot. Separately
compiled AVX2 callers can still use the canonical AVX2 entry. An AVX2-only
guard reuses that entry when available. The wider
clone specializes direct calls to the local AVX2 helper body.
For independent guarded regions, the resolver selects the union of matching
masks when the function has up to 16 distinct unions (including the empty
mask). Above that limit it retains at most 14 observed masks and their full
union. Each root can create at most 64 new bodies including helpers, its
baseline and resolver. Wider masks receive the helper budget first. Further
callees keep their ordinary entry. Regions outside the selected mask retain
their runtime checks. Local helpers share a copy for each distinct proven mask.

Specializations follow direct calls through local helpers, including kernels
that take array pointers and use vectors internally. Recursive helper calls
reuse the same copy for a given feature combination. Indirect calls keep
the ordinary entry.
Calls to separately compiled functions can use their ordinary Go entries.
The public function value and its wide argument/return carriers stay valid
through dispatch. The baseline implementation serves calls before CPU feature
publication. `TestLazyDispatcherPolicy` checks that this call does not cache
the baseline. `TestSIMD512ScalarSignatureHelpers` exercises the scalar fallback
when AVX512 is unavailable. `TestSIMD512Guarded` checks the wide call on
supported CPUs, and
`TestGuardedFeatureCombinations` checks positive guards, a negated early exit,
local helper calls and repeated execution of the LLVM transformation.

FMV processes each amd64 package module once, before target optimization.
Guard specialization uses CPU queries visible directly in the branch condition.
Saved booleans and predicate helpers follow ordinary Go control flow.
`TestSIMDSavedGuard` and `TestSIMDPredicateHelper` check the results and
helper side effects independently of specialization.
