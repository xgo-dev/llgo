# Native representation and defined types

This migration addresses [#2927](https://github.com/xgo-dev/llgo/issues/2927).
The first stage implements frontend type attributes and inherited native storage.
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
Backend representation consumers use `RepresentationOf` instead of independently
looking up a new name in the calling-convention registry.

## Frontend attributes and checking lifecycle

Definition provenance is frontend input, not a backend query mechanism. The
frontend resolves native storage, callback ABI, and diagnostic origin into a
`TypeRepresentation` for each declared type. `RepresentationOf` reads those
attributes without walking named definition edges or accessing source bindings.
Generic instances read the attribute of their origin declaration. Independent
same-shape declarations remain independent; Go identity and method sets do not
change.

SSA instantiation of function-local types creates fresh `TypeName` objects, even
for `Named.Origin()`. The snapshot therefore also indexes checked local
attributes by package identity, declaration position, and name. These stable
declaration coordinates transfer attributes to SSA's scope-less copies,
including ordinary Go attributes for local shadows of native package types.
This indexes a known declaration; it does not infer representation from shared
underlying structures or recover source bindings in the backend. When lowering
renames these types to positionless ABI names, it explicitly copies the resolved
attribute into Program-local adapter metadata; the shared snapshot stays immutable.

The lifecycle has two phases:

1. During source checking, a frontend resolver uses the checker's live
   `types.Info` for source size/alignment queries. This is needed for constants
   and array bounds containing `unsafe.Sizeof`. An unavailable RHS is never
   memoized as a completed attribute. Only source layout can use this resolver;
   attempting backend lookup or creating a backend Program before completion
   fails explicitly.
2. After each successful check, including a recheck after source rewriting, the
   frontend resolves all package and local type declarations and atomically
   publishes their attributes. It then releases the provisional callback and
   source references. Dependencies complete this phase before their importers
   are checked. Backend Programs share only the published Go-owned metadata,
   keeping independent LLVM contexts and layout caches.

One-shot clients with already checked syntax resolve and publish attributes
when collecting directives. If `types.Info` is unavailable, the frontend may
use `types.CheckExpr` in the original lexical scope during this preparation
step. Binding failures prevent publication; backend queries never retry source
resolution. Clients collecting directives before checking should supply live
`types.Info` and call `cl.CompleteTypeRepresentations` after a successful check.

Programmatic SSA clients supply resolved attributes with
`SetTypeRepresentation`; `SetTypeCallConv` remains the legacy API for native
roots. Published attributes cannot be changed to invalidate an existing layout.
Neither shared `Underlying()` objects nor shared fields establish inheritance.

Package export serialization is a later stage. Until it carries these attributes
(including reachable private and generic origins), native packages retain the
existing source-checking fallback; this change does not disable archive or SSA
caching. Validated opaque imports publish a completed default-Go snapshot;
partial binding maps reconstructed from exports are never treated as live
checker information.

## First-stage coverage

- Closure conversion and native LLVM struct classification use inherited
  representation. Native callback aliases/definitions preserve their invocation
  ABI, including stdcall.
- The source-size closure adjustment uses the same representation query. This
  does not replace the source layout algorithm or correct all size/offset cases.
- HostLayout field queries also follow definition provenance for native
  callbacks and callback arrays. HostLayout itself continues to preserve the
  environment of ordinary Go closures.
- Source tests cover transitive and cross-package definitions, aliases,
  parenthesized RHS types, generic templates/instances, recursion, local scopes,
  method-set independence, and concurrent backend queries after source bindings are discarded.
- Executable tests in `test/llgoext/nativerepr` preserve an adjacent sentinel
  through pointer reads/writes and exercise ordinary Go closures accepting a
  derived native value, inherited callback invocation and clearing, and array
  element storage. The same tests run under the official Go compiler, which
  ignores the native directives.
- Wasm32 executable tests cover derived native slice callbacks through multiple
  definitions and aliases alongside the original callback ABI cases.
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
   The current source loader can resolve type attributes from source;
   standard Go export data does not carry this LLGo protocol. Do not infer it
   from export-data object sharing or silently assume Go representation when
   required native metadata is missing. Include generic origins and reachable
   private declarations, and separate target-independent policy from computed
   target layouts. Until that protocol exists, packages declaring native types
   remain source-checked instead of using opaque frontend type exports. Archive
   and SSA caching remain enabled. Cold/warm build regressions cover native
   definitions across packages while an unrelated dependency still uses its
   opaque type export.

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
