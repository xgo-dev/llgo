# Automatic cross-package inlining

LLGo makes small Go function and method bodies available to LLVM in importing
packages, including when the defining package's backend archive is cached. This
follows Go's cross-package inline model: the caller needs the body as well as the
declaration. Cold builds construct dependency Go SSA. With frontend caching
enabled (`LLGO_ESCAPE_CACHE=1`, with the build cache enabled and a supported
build mode; see [escape analysis](escape-analysis.md)), the driver restores
validated frontend facts and materializes eligible dependency bodies on demand
before lowering callers. Frontend preloading and backend import share the same
replacement-target resolver and reuse the existing frontend exports and SSA
loading path.

No new directive is introduced. `//go:noinline` prevents import and marks both
the defining function and importing declarations with LLVM `noinline`, including
when LTO later makes the definition visible. Functions and methods remain
defined in their original package for function values and calls that LLVM does
not inline. Importing a body does not force inlining; LLVM applies its normal
optimization-level, target and call-site cost policy.
Cross-package inlining therefore does not require LTO.

With Full LTO, the frontend skips optimization-body imports and leaves
cross-package inlining to the final linker, which sees the defining packages'
real bodies. This avoids changing optimization order through early inline
copies. Without LTO and with ThinLTO, the import policy below applies.

For Linux amd64 executables built with `-O3 -lto=full` through ELF LLD, calls
in unprofiled cyclic switch regions can receive LLVM 22's ordinary O3 inline
budget (250). Packages that receive hints request complete inline-cost analysis
at final link, including after an archive-cache hit. The hint decision participates
in the package fingerprint. Profile summaries, branch profiles and explicit
inline attributes remain authoritative. The same link profile aligns loops to
64 bytes and pads selected branches at 32-byte boundaries, excluding calls and
returns. This trades code size for dispatch performance.

To select an explicit inline budget, use, for example,
`LDFLAGS=-Wl,-mllvm,-inline-threshold=20`; this disables LLGo's switch-call hints
and their automatic complete-cost flag. An explicit LLVM layout option, such as
`LDFLAGS=-Wl,-mllvm,-align-loops=16`, replaces the automatic loop/branch layout
settings. Ordinary C flags and library paths do not disable these policies.
Other optimization levels, Thin/off LTO and explicitly selected alternate
linkers retain their existing behavior.

## Candidate and import limits

The initial frontend policy uses a budget of 80 SSA cost units per body,
charging a non-importable call an additional 57 units. A call to another
candidate includes its body cost. These are frontend costs, not source-line or
LLVM instruction counts, and are deliberately separate from LLVM's inliner.
Each candidate's decision is cached once per compilation. Nested depth and
cycle analysis remains local to that candidate, so cache warmup order does not
change import decisions. Different candidates can be analyzed concurrently;
the shared cache lock only covers entry lookup and publication.

Each module imports at most 1024 cost units, with a discovery depth of five.
Value and pointer receiver methods use the same budget and lowering as ordinary
functions. The policy excludes loops, recursion, synthetic method wrappers,
generic declarations and instances, closure environments, goroutines,
defer/recover, and functions with pragmas that depend on retaining their call
frame. Functions that need runtime caller frames or access package-owned
locality storage are also excluded. Missing bodies remain ordinary calls.
Declaration-only packages (`LLGoPackage = "decl"`) are excluded even when their
source contains a body: the build driver does not emit that body, and the real
implementation may be supplied by another module.
Imports are disabled at `-O0` and for shadow-stack and debugger-frame
instrumented builds.

Source-patch imports require matching symbols and mapped parameter/result types.
Methods are resolved among the receiver's declared methods in the replacement
SSA package and must also match the mapped receiver type, including pointer
versus value receivers. Missing or incompatible replacement methods remain
ordinary calls; their original bodies are never used as a fallback.

## LLVM representation

Eligible direct cross-package calls queue normal function lowering into the
caller's module. The copy has `available_externally` linkage under the canonical
function name: it supplies optimization information without emitting another
real function definition. The defining package continues to own that symbol.
Imported copies do not duplicate function-info metadata. Function values keep
the canonical function address, whether or not a direct call is inlined.

## Validation

`benchmark/crossinline` compares value and pointer receiver calls with an
ordinary function control using the same computation. Run it with the same
optimization level and LTO setting on both compiler revisions; the ordinary
function already benefits from cross-package body import.
The microbenchmarks pass the final result to `runtime.KeepAlive`, retaining the
computation when LTO could otherwise eliminate an unread global sink.

`benchmark/crossinline/json` measures decoding, updating 64 records and encoding
the result. `benchmark/crossinline/template` renders 64 records into a
reused buffer, with parsing and initial buffer growth outside the timer. Both
programs also run independently and emit their complete output for comparison
with native Go. They use the built-in `print`/`println`; native Go writes this
output to stderr. Capture both streams, and check that the reference is nonempty:

```sh
go run ./benchmark/crossinline/json > go.out 2>&1 &&
    llgo run -Os ./benchmark/crossinline/json > llgo.out 2>&1 &&
    test -s go.out &&
    cmp go.out llgo.out
```

Use the same commands with `template` in place of `json`. These are library
workloads, not application-level benchmarks.
For compiler comparisons, alternate runs of prebuilt test binaries with the
same inputs and flags. Measure empty-cache and warmed-cache builds separately
with isolated LLGo backend caches; disabling the cache is not an empty-cache
build. Report executable size and code size alongside runtime and build time.
