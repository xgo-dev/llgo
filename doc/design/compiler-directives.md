# Compiler directive records

`internal/directive` owns comment recognition and source records for Go and LLGo
compiler directives. A compilation owns one `directive.Store`; coordinator and
backend Programs share it through `packageSyntaxData`. Records contain Go data,
never LLVM values. Consumers must not modify published records.

## Preparation and consumption

1. Source patch discovery records load-time `llgo:skip` / `llgo:skipall` commands.
   The overlay builder consumes them before type checking. Directive neutralizing
   preserves byte offsets and line endings. Reparsed overlays have distinct AST
   identities and therefore distinct records.
2. The loader registers each selected AST with the Store. A File snapshot records
   normalized comment groups, function properties, package-wide links, internal
   directives, embed directives, cgo pragmas and C preambles. Existing syntax
   dialects remain separate where their accepted spelling or precedence differs.
3. `cl.ParsePkgSyntaxWithOptions` validates trust and declaration placement and
   creates a `directive.Package`. Declaration records belong to source nodes in
   a particular `types.Package` instance. Type-dependent locality and embed checks
   remain in their existing preparation phases and consume those records.
4. After type checking, `Package.Bind` associates records with checker objects.
   Patch views bind their effective declarations using both original and alternate
   type information. Replaced declarations cannot override the active name record.
   Standalone clients without `types.Info` use position-checked `BindScope`.
5. Caller analysis prepares its function records before backend workers start.
   The driver freezes the Store; attempting to discover an unprepared file,
   comment group, function or imported source after that boundary panics.
6. Lowering queries records. A backend-local `sourceFunction` embeds the x/tools
   SSA function and carries its source properties. Generic instances consult their
   origin. Synthetic wrappers retain explicit propagation rules (for example,
   uintptr escapes) instead of inheriting every property. LLVM attributes are
   applied after function creation using existing constructors.

## Ownership and compatibility

| Record | Properties and consumers |
| --- | --- |
| Function | Closure environment, noinline, nosplit, uintptr escapes, wasm imports, nointerface; call/declaration lowering, frame analysis, safepoints and GC roots |
| Declaration | Link/export names; symbol creation and export preservation |
| Type | Go/C/stdcall background; ABI and layout |
| Comment group | Locality/internal directive inputs and placement diagnostics |
| File/package | Embed patterns, cgo flags/imports/preambles, source patch commands and package skip decisions |

Source declarations that share a linker name remain distinct. Object keys retain
receiver aliases and generic origin identity; package keys distinguish test and
alternate views with the same import path. Effective patch selection precedes
linkname, type-background and nointerface lookup. Synthetic promoted-method
wrappers do not inherit their embedded method object's linkname. Legacy string-keyed maps remain derived compatibility indexes
for runtime names, link alias chains and existing Program APIs; source-backed
lookups use package records first.

The refactor preserves diagnostic timing: recognition can record a malformed
value early, while the existing owning phase reports it. It also preserves
source-order precedence, exact legacy spellings, trailing-comment rules,
internal-directive trust checks and existing invalid-directive behavior.

## Standalone entrypoints and remaining source use

Standalone compiler clients may provide imported type objects without dependency
ASTs. Dependency SSA declarations with available syntax are also snapshotted
during preparation. The preparation phase discovers and caches legacy imported link directives
through `Store.ReadLegacyLinks`, including failed reads. Lowering applies the
prepared records without opening those files. Independent helper APIs may create
a temporary Store; the normal build path passes its shared Store throughout.

This does not remove Go parsing, type checking, SSA construction, patch AST
transformation, embedded resource reads, debug source-line reads or syntax-based
nil-check analysis. Those operations have purposes beyond directive parsing.
Source-free cache hits and serialized declaration exports require separate work.

## Regression checks

- Record tests cover spelling/precedence, CRLF, receiver aliases, package variants,
  distinct declarations sharing a symbol, freeze enforcement and concurrent reads.
- A compiler test removes original comments after preparation and verifies the
  resulting LLVM module and attributes.
- Generic instance and patch replacement tests verify source/object association.
- Existing compiler, loader, locality, embed, cgo and source-patch suites preserve
  diagnostics and runtime behavior across the migrated consumers.
