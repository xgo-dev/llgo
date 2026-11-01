# LLGo type layout and go/types

`runtime.Compiler` remains `"llgo"`. In a program built by LLGo,
`go/types.SizesFor("llgo", arch)` and `go/types.SizesFor("gc", arch)` both return
LLGo's layout. The latter supports clients that hard-code the Go toolchain
family. It does not select cmd/compile's object representation. Unknown
compiler names and architectures still return nil; `gccgo` retains its standard
library implementation. Build constraints are unchanged.

The canonical parser, layout model, metadata encoding, and validation live in
`runtime/_patch/go/types/llgo_*.go`. The compiler uses the generated adapter in
`internal/typesizes`; reflection's tag-display helper is generated from the
same source. Neither layout implementation depends on LLVM.

The model includes Go closures (code and environment words), native function
pointers, pointers and aggregate fields, arrays including zero-length arrays,
aliases and defined types, instantiated generic structs, native calling
conventions, SIMD alignment markers, ordinary trailing zero-sized field
padding, and the layout directives below. Nested aggregates are laid out
recursively instead of adjusting offsets computed using gc sizes.

## Maintaining one implementation

The source patch compiles inside standard `go/types`. The compiler adapter
imports `go/types` to work with the host toolchain's types. Importing that adapter
back into the patched package would create an import cycle. The two packages
therefore need separate source files, but the layout rules have one maintained
implementation.

Edit the shared `llgo_*.go` files under `runtime/_patch/go/types`, then run from
the repository root:

```sh
go generate ./internal/typesizes
go test ./internal/typesizes -run '^TestGeneratedAdapterMatchesPatch$' -count=1
```

Commit the canonical source and generated files together. Do not edit the files
marked `Code generated` directly. When adding or removing a shared source file,
update the file list in `internal/typesizes/generate.go` and its generated output.
The package-specific entry points `check.go` and `sizes.go` are not copied.

Generation changes the package declaration and qualifies standard `go/types`
identifiers; it does not rewrite layout rules. Reflection's generated helper
also changes the exported helper name to its package-local name.
`TestGeneratedAdapterMatchesPatch` runs the generator in `-check` mode, computes
expected output without writing files, and compares the complete generated
contents. Only CRLF/LF differences are ignored. A stale or manually edited copy
fails this test, which is included in the regular Go CI suite.

Text consistency alone does not prove correct layouts. The source-versus-LLVM
checks, export-data tests, C++ comparisons, and Go/LLGo execution tests described
below separately check that the shared rules produce the expected storage and
reflection behavior.

## Source attributes

Parse source with `parser.ParseComments`. The patched `types.Checker.Files`
registers declaration attributes before folding `unsafe.Sizeof`,
`unsafe.Alignof`, and `unsafe.Offsetof`. It also supplies LLGo sizes when
`types.Config.Sizes` is nil. Explicit custom `types.Sizes` implementations are
preserved.

Both `//llgo:cdecl` and `//llgo:stdcall` select native type representation;
legacy `//llgo:type C` and `//llgo:type stdcall` remain recognized. Attributes
propagate through ordinary aliases and defined types. Applying a native
annotation directly to an alias creates a distinct defined native view; an
unannotated alias shares its target's identity. A function declaration linked
using `//go:linkname local C.symbol` or `stdcall.symbol` uses the native call
boundary. Its ordinary Go function values still have code and environment
words. Explicit native function types and native aggregate fields instead
store a single native pointer. Linkage must not turn a structural Go signature
into a different storage type: generic instantiation can share that signature
with unrelated Go functions.

A C-linked **variable** holding an anonymous callback is different: its slot
holds a native function pointer. Such variables and anonymous callbacks in
native structs are normalized to canonical native function types before
checking. Explicitly named Go field types retain their declared representation,
including their closure environment; declaring an outer native struct does not
change those named types. Declare C-linked callback variables separately rather
than sharing a multi-name variable declaration.

A new native struct must have an explicitly annotated struct declaration, or
derive from an existing native struct. Applying a native annotation to a type
reference whose underlying struct is still Go storage is rejected: reusing the
same field identities would otherwise make `Offsetsof` ambiguous.

Layout attributes participate in LLGo type identity. Before checking, the
shared parser inserts reserved `llgo.layout` field metadata. Structural type
comparison and generic instantiation then distinguish otherwise equal field
lists with different layouts. Anonymous native callbacks use canonical named
function types, distinguished by calling convention and signature. They do not
tag or mutate shared Go signatures. These rules can require explicit value
conversions where plain Go would permit assignment.

The field metadata survives standard Go export data. Native named types also
receive generated `LLGoTypeLayout_*` metadata constants in the checked package
scope. The patched importer restores attributes from those records, including
derived function types. Export data produced by an ordinary Go checker that
never processed LLGo annotations cannot recover the omitted information.

Reflection retains field metadata internally for identity checks, but its public
`StructField.Tag` preserves the user's original tag. `types.LLGoFieldTag` provides
the same filtering for tools inspecting `go/types.Struct.Tag`. Metadata names
are reserved for the compiler. `LLGoLayout.RecordPackage` and `RecordType` restore
attributes for custom importers and synthetic types carrying that metadata;
plain `types.NewStruct` cannot infer attributes from fields alone. These extension
APIs are available in LLGo's patched `go/types`; portable source referring to
them should use the `llgo` build constraint.

## Padding directives

Place these directives in a struct field's documentation comment:

| Directive | Placement | Effect for a zero-sized field |
| --- | --- | --- |
| `//llgo:padding` | Generic or non-generic field | Adds one byte after a non-final field. On the final field, warns and keeps ordinary layout. |
| `//llgo:no-padding` | Last field of a non-generic struct | Suppresses the ordinary Go trailing padding byte. |
| `//llgo:padding-if` | Field of a generic struct | Suppresses trailing padding unless a same-type direct member or embedded base occupies the same address; a conflict moves the field to the next available aligned address and reserves one byte. |

For nonzero-sized fields these directives do not add padding. Conflicting
directives, arguments, or invalid placement are errors. Padding bytes do not
become Go fields: selectors, positional composite literals, reflection field
indexes, comparisons, and copies retain their original field lists.

These rules implement explicit layout controls, not the entire C++ object
model. A bare empty Go struct still has size zero. `no-padding` suppresses
trailing storage without moving the final field into an earlier member's
storage; its address can therefore be at the end of the object. Conditional
conflicts compare direct members and embedded bases, not recursively nested
empty subobjects. No virtual base, union, or bit-field layout is inferred.
An address of a suppressed trailing empty field must not be the only GC root
for its owner: retain the owner separately, for example with `runtime.KeepAlive`.

## Packing

Put `//llgo:packed` before a package-level struct type declaration to cap field alignment at
one byte. `//llgo:pack N` caps it at `N`, where `N` is 1, 2, 4, 8, or 16.
The struct's alignment is the largest capped field alignment, and its size
includes tail rounding to that alignment. Packing does not recursively change
the internal layout of a separately declared field type. Arrays use the packed
element's size as their stride. Native representation and packing can be
combined on the same declaration.

LLVM storage wrappers preserve field indexes while adding inaccessible
padding and controlling field alignment. Ordinary memory accesses in a source
graph containing packed declarations use conservative alignment, including
escaped field pointers; LLVM may infer stronger alignment from allocations.
The graph-wide packing mode participates in every package cache fingerprint,
so a packed field pointer cannot reach a cached callee compiled with stronger
alignment assumptions. Packing does not relax the alignment requirements of
atomic operations.

The conservative collectors scan pointer-aligned words. Ordinary packed Go
types that contain managed references below native pointer alignment are
rejected, including pointers, `unsafe.Pointer`, closures, strings, slices, maps, channels
and interfaces.
Numerical fields and native callback pointers do not have this restriction.
Explicit native C storage may be packed, but any Go-owned objects referenced
through unaligned C storage must be retained separately.

Value conversions between compatible layouts copy fields, including nested
arrays, instead of reinterpreting storage. Public reflection reports both Go
funcval words, while native callback sizes remain one native pointer. Ordinary pointer conversions between
different layouts are rejected; `unsafe.Pointer` remains an explicit escape
hatch. Reflection follows the same distinction. Compiler-generated call bridges
handle packed parameters and results that libffi cannot describe. Dynamically
constructed reflection-only signatures can use `MakeFunc` and `Call` without a
compiled call site. Creating a native callback with `MakeFunc` requires a native
closure trampoline; targets/signatures using compiler bridges report this
unsupported operation rather than dropping the callback's environment.

## Targets and validation

For the running target, `SizesFor` obtains Go word width, native pointer width,
and native double alignment from compiler-folded probes. This distinguishes
wasm Memory32 and Memory64 even though both report `GOARCH=wasm`. For another
architecture, `LLGoSizesFor` supplies the architecture defaults. Cross-target
C ABI users must additionally configure platform-specific properties such as
`PointerSize` and `NativeAlign64`; GOARCH alone cannot express every OS ABI.

`test/go/type_sizes_gc_test.go` checks gc-specific sizes and ignored LLGo
annotations; `type_sizes_llgo_test.go` checks LLGo-specific sizes and native
representation. Both supply `types.Sizes` implementations to the shared suite
in `type_sizes_test.go`, which has no compiler-identity branches.
The common tests compare each compiler's sizes with its own unsafe constants,
generic builtins, real addresses and array strides, aggregate reflection metadata, packed
loads/stores, maps, copying, and closure environments. LLGo-specific tests also
check both compiler names, native function declarations/variables, anonymous
type identity, native callback fields, and named Go fields in native structs.
Standard Go ignores
LLGo directives and verifies its normal layout.

SSA tests compare source and physical LLVM layouts across native 32/64-bit
targets and wasm32/64. `internal/build/testdata/layoutcpp` compares sizes,
alignments and offsets with compiled C++ and passes packed aggregates by value
in both directions. The source patch also has overflow, diagnostic, generated
adapter consistency, repeated-AST checking, export-data round trips, managed
packing diagnostics (including local generic instances), pointer-conversion
checks, and concurrent-checker tests. Cache tests ensure packed programs do not
reuse callees compiled with natural-alignment assumptions.
