# Native representation and defined types

This migration addresses [#2927](https://github.com/xgo-dev/llgo/issues/2927).
The first stage implements definition provenance and inherited native storage.
It is not a complete resolution of native layout, conversion, or reflection
consistency. The remaining stages below are required before closing the issue.

## Separate identity, representation, and invocation

`type Copy Native` creates a new Go type identity and method set, but must retain
the representation selected by `Native`. Otherwise an accepted pointer
conversion accesses the same object using different field offsets.

`ssa.TypeRepresentation` records the storage family, native invocation ABI, and
the declaration that selected it. The invocation ABI applies to native function
values; it does not turn an ordinary Go function accepting `Copy` into a native
function declaration. A native callback remains one code pointer, while an
unrelated ordinary Go callback parameter retains its closure environment.

The existing `SetTypeCallConv` API supplies native roots during this migration.
Derived declarations do not acquire additional calling-convention registrations.
Representation consumers use `RepresentationOf` instead of independently
looking up a new name in the calling-convention registry.

## Definition provenance

The source loader supplies its live `types.Info` to declaration collection before
checking. A representation query follows the actual bound RHS type, including
qualified imports, aliases, and generic instances. It does not cache an
unavailable RHS. Instances query their origin declaration's representation.
Local declarations are identified within their original package and lexical
scope, so a local type that shadows a native package type remains ordinary Go.

One-shot frontend clients that have discarded `types.Info` resolve named RHS
expressions in the completed package's original scope using `types.CheckExpr`.
That fallback is synchronized for backend readers, and a resolution error must
not silently select Go storage. Programmatic clients and future importers can
provide an explicit edge with `SetTypeDefinition`.

Backend Programs share this Go-owned provenance while retaining independent
LLVM contexts and lowering caches. Source information must finish changing
before backend lowering starts.

Neither shared `Underlying()` objects nor shared fields establish a definition
edge. An independently declared type stays independent even if its structure is
identical. Declaration positions in this implementation locate an AST RHS; they
do not identify an enclosing struct from its fields.

## First-stage coverage

- Closure conversion and native LLVM struct classification use inherited
  representation. Native callback aliases/definitions preserve their invocation
  ABI, including stdcall.
- The source-size closure adjustment uses the same representation query. This
  does not replace the source layout algorithm or correct all size/offset cases.
- Source tests cover transitive and cross-package definitions, aliases,
  parenthesized RHS types, generic templates/instances, recursion, local scopes,
  method-set independence, and concurrent backend queries.
- Executable tests in `test/llgoext/nativerepr` preserve an adjacent sentinel
  through pointer reads/writes and exercise ordinary Go closures accepting a
  derived native value. The same tests run under the official Go compiler,
  which ignores the native directives.
- Target-layout tests compare inherited storage on amd64, arm64, Windows/386,
  and profiled wasm32. These are LLVM layout tests, not execution on all targets.

## Remaining migration stages

1. Add context-preserving layout queries for the full owning type and selection
   path. Standard `types.Sizes.Offsetsof` receives fields only. The current field
   registry is not an adequate substitute; reverse native derivation and shared
   underlying objects can still make its answer ambiguous. Source constants
   must be correct while checking dependent declarations, not patched after
   type checking.
2. Introduce representation-aware shared-storage checks. Independent same-shape
   types, anonymous structs, and native function-field addresses must not be
   reinterpreted through incompatible Go views. Reject incompatible pointer
   operations; copying to a temporary would lose aliasing. A conflict on an
   alias must not create a second representation for the same identity.
3. Make source layout, LLVM storage, reflection metadata, pointer scanning,
   regular-memory classification, and map bucket/fast-path choices consume one
   layout description. This includes containing arrays/structs, trailing empty
   fields, and the map 128-byte storage boundary tracked in #2873.
4. Preserve callback representation in reflection and FFI descriptions (#2872).
   `abi.Func` alone cannot distinguish a code pointer from a Go closure. Keep
   unsupported native vector reflection calls explicitly rejected.
5. Define versioned package representation metadata and cache fingerprints.
   The current source loader can reconstruct declaration edges from source;
   standard Go export data does not carry this LLGo protocol. Do not infer it
   from export-data object sharing or silently assume Go representation when
   required native metadata is missing. Include generic origins and reachable
   private declarations, and separate target-independent policy from computed
   target layouts.

## Focused validation

Use matching LLVM, Clang, and LLD versions on `PATH`.

```sh
go test ./ssa ./cl -run 'TestNativeRepresentation|TestRepresentation|TestDerivedNative'
go test -race ./ssa ./cl -run 'TestNativeRepresentation|TestRepresentation'
go test ./test/llgoext/nativerepr/...
llgo test -O0 ./test/llgoext/nativerepr
llgo test -O2 ./test/llgoext/nativerepr
```

The main-line baseline fails the executable sentinel cases. Passing these cases
establishes definition inheritance only; it does not establish the remaining
layout and reflection properties above.
