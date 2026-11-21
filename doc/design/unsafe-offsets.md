# Owner-aware offset queries

The source loader wraps `types.Sizes` for each checker. Before an
`unsafe.Offsetof` callback, `go/types` records the selector and its actual
receiver in the live `types.Info`. The wrapper associates that selection with
its field list, including promoted paths and repeated constant initializers,
and calls `Program.SourceOffsets` with the owning type. Typed source rewrites
receive a fresh wrapper and `types.Info` rather than retaining stale selections.
The AST, Go type identity and shared field objects are unchanged.

The offset calculation uses source metadata without creating LLVM types while
named declarations are incomplete. It shares aggregate offset arithmetic,
struct reconciliation policy and field alignment policy with storage lowering.
Tests compare constants with LLVM offsets across 13 target configurations and
with actual native/Wasm addresses, including array bounds and cold/warm caches.
`Sizeof`, `Alignof`, reflection/GC/map metadata and physical storage are outside
this change.

This fixes the named-owner ambiguity in #2796, but does not close the broader
representation migration. In particular, a native outer record can suppress
closure conversion inside an ordinary Go nested record. Then
`unsafe.Offsetof(outer.Inner.Tail)` queries the declared Go type while the
nested storage has a raw callback: on arm64 the query is 16 and the address
difference is 8 both before and after this change. Mixing the two views can
also depend on LLVM named-type construction order. Repairing that existing
shared-storage inconsistency requires the representation compatibility work
in #2927; it must not be disguised by inferring a new type identity from fields.

Validation commands:

```sh
go test ./internal/build -run 'TestUnsafeOffsetOwners|TestFrontend'
llgo test -O0 ./test/llgoext/nativeoffset
llgo test -O2 ./test/llgoext/nativeoffset
llgo test -O2 -emulator -target emscripten ./test/llgoext/nativeoffset
```
