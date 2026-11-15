# Automatic cross-package inlining

LLGo makes small Go function and method bodies available to LLVM in importing
packages, including when the defining package's backend archive is cached. This
follows Go's cross-package inline model: the caller needs the body as well as the
declaration. The build driver already constructs dependency Go SSA before
looking up backend archives, so no additional source loading or export format
is required.

No new directive is introduced. `//go:noinline` prevents import and marks both
the defining function and importing declarations with LLVM `noinline`, including
when LTO later makes the definition visible. Functions and methods remain
defined in their original package for function values and calls that LLVM does
not inline. Importing a body does not force inlining; LLVM applies its normal
optimization-level, target and call-site cost policy.
Cross-package inlining therefore does not require LTO.

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

`benchmark/crossinline/json` measures decoding, updating 64 records and encoding
the result. `benchmark/crossinline/template` renders 64 records into a
reused buffer, with parsing and initial buffer growth outside the timer. Both
programs also run independently and emit their complete output for comparison
with native Go. These are library workloads, not application-level benchmarks.
For compiler comparisons, alternate runs of prebuilt test binaries with the
same inputs and flags. Measure empty-cache and warmed-cache builds separately
with isolated LLGo backend caches; disabling the cache is not an empty-cache
build. Report executable size and code size alongside runtime and build time.
