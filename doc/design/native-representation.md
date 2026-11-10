# Native representation and defined types

This migration addresses [#2927](https://github.com/xgo-dev/llgo/issues/2927).
Definition provenance and inherited native storage are implemented together
with aggregate source layout, owning-type metadata, and versioned package
representation facts. The remaining limitations below still prevent treating
this as a complete migration of every native layout operation.

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

## Implemented coverage

- Closure conversion and native LLVM struct classification use inherited
  representation. Native callback aliases/definitions preserve their invocation
  ABI, including stdcall.
- Source checking computes native aggregate layouts without constructing LLVM
  named types while declarations are incomplete. Containing arrays and Go
  structs use the resulting element/field sizes and alignment. Native records
  omit Go trailing-zero-field protection; containing Go records retain it.
- Runtime aggregate sizes retain the owning named type. Reflected field tables
  use that owner's physical LLVM offsets; native pointer-prefix metadata follows
  physical child storage, including four-byte native fields in memory32.
- Explicit anonymous native RHS layouts have separate backend type/layout
  caches and descriptor identities. Their layout attribute participates in
  persistent structural type keys too.
- Typed pointer conversions reject incompatible size, alignment, field offsets,
  nested storage, or callback ABI. Compatible inherited views retain aliasing.
  An explicit unsafe.Pointer conversion remains the caller's responsibility.
- Source tests cover transitive and cross-package definitions, aliases,
  parenthesized RHS types, generic templates/instances, recursion, local scopes,
  method-set independence, and concurrent backend queries.
- Executable tests in `test/llgoext/nativerepr` preserve an adjacent sentinel
  through pointer reads/writes and exercise ordinary Go closures accepting a
  derived native value. The same tests run under the official Go compiler,
  which ignores the native directives.
- Target-layout tests compare inherited storage on amd64, arm64, Windows/386,
  and profiled wasm32. These are LLVM layout tests, not execution on all targets.

## Package persistence

Frontend export protocol version 4 retains each declaration's checked storage
policy, native ABI, and origin type, including private declarations and generic
origins. Explicit Go facts are persisted as well: an opaque package has no live
RHS bindings from which to reconstruct them. Source RHS objects are deliberately
not compared with backend closure-converted bodies. Anonymous native layout
attributes are serialized and validated independently of source positions.
Unknown ABIs, unsupported attributes, and invalid origin references select the
source-checking fallback before metadata is published. The existing archive
fingerprints also include compiler, target, source, and feature identities.

## Remaining limitations

1. Standard `types.Sizes.Offsetsof` receives fields rather than their owning
   type. The legacy source field registry still cannot distinguish a Go type
   and a new native override sharing its exact underlying fields, such as
   `type Plain struct { Fn func(); Tail uint64 }` followed by an annotated
   `type Native Plain`. A complete solution needs owner-aware checker queries.
2. Native function-field address escapes and all implicit shared-storage views
   need representation-aware checking. The implemented typed-pointer diagnostic
   does not establish compatibility for every assignment, slice, or map view.
3. Callback reflection and FFI descriptions (#2872) still need to distinguish a
   code pointer from a Go closure. Native anonymous function fields have a
   storage context not represented by their source `types.Signature` alone.
   Unsupported native vector reflection calls remain rejected.
4. The aggregate metadata fixes cover the tested inheritance paths; they do not
   close every native metadata case from #2873, especially nested ordinary Go
   types whose callback representation is changed by an outer native aggregate.

## Focused validation

Use matching LLVM, Clang, and LLD versions on `PATH`.

```sh
go test ./ssa ./cl -run 'TestNativeRepresentation|TestRepresentation|TestDerivedNative'
go test -race ./ssa ./cl -run 'TestNativeRepresentation|TestRepresentation'
go test ./test/llgoext/nativerepr/...
llgo test -O0 ./test/llgoext/nativerepr
llgo test -O2 ./test/llgoext/nativerepr
```

The main-line baseline fails the executable sentinel cases. These cases
establish the covered inheritance, aggregate layout, and cache contracts; they do not establish the remaining limitations above.
