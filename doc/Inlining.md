# Automatic cross-package inlining

LLGo makes small Go function bodies available to LLVM in importing packages,
including when the defining package's backend archive is cached. This follows
Go's cross-package inline model: the caller needs the body as well as the
declaration. The build driver already constructs dependency Go SSA before
looking up backend archives, so no additional source loading or export format
is required.

No new directive is introduced. `//go:noinline` prevents import. Ordinary
functions remain defined in their original package for function values and
calls that LLVM does not inline. Importing a body does not force inlining;
LLVM applies its normal optimization-level, target and call-site cost policy.
Cross-package inlining therefore does not require LTO.

## Candidate and import limits

The initial frontend policy uses a budget of 80 SSA cost units per body,
charging a non-importable call an additional 57 units. A call to another
candidate includes its body cost. These are frontend costs, not source-line or
LLVM instruction counts, and are deliberately separate from LLVM's inliner.
Accepted and rejected analyses are cached once per compilation and shared
safely by parallel package backends.

Each module imports at most 1024 cost units, with a discovery depth of five.
The initial policy excludes loops, recursion, methods, generic instances,
closure environments, goroutines, defer/recover, and functions with pragmas
that depend on retaining their call frame. Functions that need runtime caller
frames or access package-owned locality storage are also excluded. Missing
bodies and incompatible source-patch signatures remain ordinary calls.
Imports are disabled for shadow-stack and debugger-frame instrumented builds.

## LLVM representation

Eligible direct cross-package calls queue normal function lowering into the
caller's module. The copy has `available_externally` linkage under the canonical
function name: it supplies optimization information without emitting another
real function definition. The defining package continues to own that symbol.
Imported copies do not duplicate function-info metadata. Function values keep
the canonical function address, whether or not a direct call is inlined.
