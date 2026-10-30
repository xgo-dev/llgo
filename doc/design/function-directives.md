# Function directives

Function directives are prepared during package loading and carried by
`directive.Function` through analysis and lowering. The public model consists
of the function wrapper, its `FunctionDirectives` value, and the compilation's
`Functions` registry. Type directives retain their existing paths.

## Ownership

`ssa.Program.Functions()` owns the registry. Isolated backend Programs share
that same instance. Compiler and build contexts do not store additional registry
fields, and the compiler does not obtain the registry through caller tracking.
Caller tracking borrows it only for its own analysis.

`Functions` privately separates source preparation from the wrapper cache:

- Source snapshots identify exact AST function bodies and declarations within
  concrete `types.Package` instances. Neither index is a public consumer API.
- The cache associates upstream Go SSA function pointers with immutable
  wrappers. Generic instances retain their concrete body and receive their
  source origin's attributes. Late SSA adapters use the same prepared source.
- `Program.BeginFunctionCompilation` creates a fresh cache and copies the
  selected patch map while sharing source snapshots. It runs before analysis or
  backend creation. Existing backends retain their previous registry, including
  its wrappers and patch selection.

No LLVM state lives in the registry. Source registration completes before
consumption; registered snapshots and published wrappers must not be mutated.
The source and wrapper locks protect their respective association maps.

## Preparation

`cl.ParsePkgSyntaxWithOptions` validates the package and prepares its type and
locality information, then calls `Functions.PreparePackage`. The latter owns
function annotation parsing, receiver aliases, declaration/name association,
package-wide and cross-file link annotations, and automatic C-package exports.
It publishes the package only after validation succeeds. The returned attribute
values feed the existing linker indexes and are copies, not mutable handles to
source snapshots. Shared link/export syntax is parsed by a private helper;
function parsing no longer has a variable/function mode flag.

Typed rewriting prepares a new checked package view. Source patches are also
prepared as distinct package views; preparing them does not redirect or mutate
another package's source index. `Register` supplies already prepared attributes
for generated AST helpers, including explicit empty attributes. An unregistered
source function is an error, not an implicit declaration without pragmas.

After Go SSA construction, the driver calls `Functions.Prepare` before caller
analysis and backend workers. SSA APIs may still create methods and intrinsic
wrappers later; their creation or consumption boundary converts the returned
raw pointer once. One-shot compiler entrypoints load imported symbol bindings
before preparing wrappers and caller analysis.

## Consumption

`Function` contains only the concrete `SSA` pointer and its `Directives` value.
`cl` passes it through function compilation, symbol resolution, closure-context
construction, GC planning and safepoint insertion. The current-function field
also holds the wrapper. Structural SSA analyses retain raw function identity;
their attribute-consuming steps receive wrappers.

The upstream SSA graph remains unchanged. Members, operands, static callees and
anonymous functions still return raw pointers, so conversion at these boundaries
uses `Functions.Function`. There is no lookup inside symbol resolution to
rediscover an already supplied wrapper.

Two consumers have no SSA body to wrap:

- `Functions.ForObject(types.Func)` supplies effective method attributes for
  `ssa` interface and method metadata. It handles generic origins and reconstructed interface receivers, and shares
  the same patch-selection rule as SSA wrapper construction. Its bool indicates
  prepared package state, including known absence. An authoritative empty value
  must not fall back to an older global symbol entry.
- `Functions.Exports` enumerates prepared export names for the effective package
  view, so export preservation does not traverse function declarations.

The build driver's runtime-free proof also needs original source-body binding
facts. `HasExternalBody` keeps that conservative AST check inside preparation
state, treating unknown declarations as unsafe. It does not use an effective
patch binding to hide a source body's foreign entry.

## Semantic boundaries

- Patch selection changes a wrapper's link/export binding, not its source-body
  flags, environment ABI or Wasm import. Type metadata instead reads the selected
  declaration: an original method's wrapper and the effective method metadata
  can legitimately have different `NoInterface` values.
- Promoted methods, thunks and bound-method adapters do not inherit their target's
  link name or body attributes. Only the existing transparent `UintptrEscapes`
  propagation is computed during wrapper construction. Ordinary callers and
  interface calls do not inherit that pragma.
- Export-object-only imports and explicitly registered backend symbols retain
  the existing linker-binding fallback when their package has no prepared source.
- Same-path package instances remain distinct. Function-only, nointerface,
  trailing-comment and last-wasmimport rules retain their existing semantics.

Regression coverage includes patch/cache isolation across backend Programs,
consistent effective method selection, atomic package preparation and value
ownership, known absence versus unprepared packages, source identity, generic
instances, transparent adapters, imported declarations without syntax, generated
locality helpers, concurrent wrapper lookup, and lowering after input comments
and declarations are removed.
