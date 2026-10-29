# Function directives

Function directives are prepared while loading each package, before its users
are compiled. `directive.Function` pairs a concrete Go SSA function with the directive
values used by analyses and lowering. Type directives retain their existing
preparation and consumption paths.

## Data and ownership

- `directive.FunctionDirectives` contains five flags (`NoInline`, `NoSplit`,
  `UintptrEscapes`, `ClosureEnv`, `NoInterface`), the Wasm module and import name,
  and the link and export names. Empty strings represent absent bindings.
- `directive.Function` contains only `SSA` and `Directives`. It keeps the instantiated
  SSA body and signature even when the directive values come from its generic
  origin. Values are immutable after the wrapper is published.
- `directive.FunctionSet` owns one wrapper per `*go/ssa.Function` for a compilation.
  The build coordinator, caller analysis and isolated backends share this set.
  Its mutex protects the pointer-keyed cache, including late SSA adapters.

`directive.FunctionSources`, shared through the Program-owned Go metadata,
has two preparation indexes. The AST
index identifies the exact source function body. The package/name index selects
a declaration within a concrete `*types.Package`, including a patch view. Both
indexes refer to the same immutable directive value. The second index also
serves consumers that have a `types.Func` or an imported SSA declaration without
syntax. Package paths alone do not identify a package view.

There is no common declaration base, stored source name/position/error, object
index, or general directive discovery store. Diagnostics are returned during
preparation. Source positions remain on the AST and SSA objects. Wasm imports
and explicit closure environments no longer have separate Program caches.

The existing symbol/export indexes remain for linker reachability, variables
and backend-created entries. They do not override an authoritative prepared
function declaration, including a declaration with no annotations.

## Preparation and use

1. `ParsePkgSyntaxWithOptions` delegates function annotation parsing and
   attachment validation to `directive.ParseFunction`. It
   incorporates package-scoped `go:linkname` annotations, including cross-file
   references, before publishing the function values.
2. Typed source rewriting creates a fresh checked package view. The existing
   loader callback prepares that view again before SSA is built.
3. Patch preparation selects the effective package bindings. Locality
   preparation explicitly registers the empty directive values of the AST
   helpers it generates.
4. After SSA construction and caller/method preparation,
   `CallerTracking.Precompute` calls `FunctionSet.Prepare` before directive-consuming
   analysis and before backend workers start.
   Later SSA adapters can be wrapped using these same prepared source values;
   this operation does not parse annotations.
5. Lowering, caller tracking, GC root planning and safepoint insertion read
   `Function.Directives`. Export preservation iterates prepared export names;
   it does not rediscover functions in the AST. An unknown source declaration
   is a preparation error, not an implicit claim that it has no directives.

The `internal/directive` package owns the attributes, source association and
wrapper construction, including AST access, generic origins and transparent
adapter inheritance. `cl.NewFunctionSet` only supplies patch package selection
and the explicit symbol-binding fallback. Function directive consumers in `cl`
pass `*directive.Function` through `compileFunction`, `compileFuncDecl`,
`funcOf`, `funcName` and closure-context construction. The current-function field
also holds that wrapper. These paths read attributes without looking the wrapper
up again. AST access remains
in source preparation and inside the directive constructors.

Bare functions enter through SSA members, operands, static callees, anonymous
functions and method/wrapper creation. These boundaries still use the shared
pointer-keyed cache once to obtain a wrapper. Structural SSA analyses retain raw
function identity; their directive-consuming steps receive prepared wrappers.
This keeps upstream SSA unchanged and reduces repeated lookups inside compiler
helpers, but does not eliminate the conversion cache.

Direct compiler entrypoints prepare their supplied root files and imported
symbol bindings before materializing wrappers. Callers that
supply SSA dependencies with source must prepare those dependencies with
`ParsePkgSyntax` first. Explicit bindings for export-object-only imports remain
available through the existing symbol registration API.

## Semantic boundaries

- A generic instance retains its concrete body and receives its origin's source
  properties.
- Patch selection changes the wrapper's link/export binding. It does not replace
  the source body's flags, closure environment ABI or Wasm import.
- Promoted methods, thunks and bound-method adapters do not inherit the adapted
  method's link name or body attributes. Only their existing transparent
  `UintptrEscapes` propagation is computed when constructing the wrapper.
- Ordinary callers and interface calls do not inherit a callee's pragma.
- `NoInterface` is still a function directive. Method metadata construction uses
  the prepared package/name bridge without constructing an SSA method merely
  to read the flag.
- Existing per-directive rules remain distinct, including the trailing-comment
  rule for `go:nointerface` and the last-occurrence rule for `go:wasmimport`.

Regression coverage includes comment and function-declaration removal from the
input files before lowering, prepared export preservation, package variants,
patch binding versus body attributes, imported declarations without syntax,
generic instances, transparent adapters, concurrent wrapper lookup, generated
locality initializers and the existing native/Wasm compiler fixtures.
