# Function attributes

LLGo supports source-level function contracts through `//llgo:attribute`.
They are available to every package, including the runtime. The design is
described in [proposal #2518](https://github.com/xgo-dev/llgo/issues/2518).

```go
//llgo:attribute param(p) returned
//llgo:attribute result(0) nonnull
func Checked(p unsafe.Pointer) unsafe.Pointer {
    if p == nil {
        panic("nil pointer")
    }
    return p
}

//llgo:attribute cold noreturn
func Fail(message string) {
    panic(message)
}
```

Attributes on results describe normal returns. `Checked` accepts nil and panics;
only its result is non-null. Annotating the input as `nonnull` would introduce a
different precondition and would not preserve that nil-input behavior.

## Targets

An annotation without a selector applies to the function. Use `param(name)` or
`param(index)` for parameters, `result(name)` or `result(index)` for results, and
`receiver` for a method receiver. Indices start at zero in the **source signature**.
A receiver and compiler-generated arguments do not occupy parameter indices.
Grouped names such as `a, b *T` occupy two positions. Blank and unnamed values can
only be selected by index.

Names are normalized to source positions before ABI lowering. Multiple lines are
combined, and attributes on a line are separated by whitespace. Attribute
arguments are parenthesized, for example `memory(read, argmem: readwrite)`.
Both `//llgo:` and `// llgo:` are accepted in a function's declaration comments.
Attributes elsewhere are rejected.

## Supported contracts

| Scope | Attributes |
| --- | --- |
| Function | `cold`, `noreturn`, `nounwind`, `willreturn`, `nofree`, `nosync`, `memory(...)` |
| Pointer parameter or receiver | `nonnull`, `readonly`, `writeonly`, `captures(none)`, `captures(ret: address, provenance)` |
| Scalar parameter or receiver matching the unique scalar result | `returned` |
| Single pointer result | `nonnull` |
| Single integer result | `range(lo, hi)`, `nonnegative` |

The initial memory forms are `memory(read)`, `memory(argmem: read)`,
`memory(argmem: readwrite)`, and `memory(read, argmem: readwrite)`.
Range bounds are integer literals describing a nonempty, non-wrapping half-open
interval within the source integer type's domain. `nonnegative` uses the target
width of the result type, so it also works for `int`. A full-domain range, including
`nonnegative` on an unsigned result, requires no LLVM range attribute.

Unknown attributes, conflicting values and unsupported target/type combinations
are errors. Identical repeated attributes are accepted. Existing `go:` directives
retain their behavior; there is no arbitrary LLVM string-attribute escape hatch.

Contracts must describe the implementation, including its implicit operations.
They do not generate runtime checks, and the compiler does not prove arbitrary
function bodies correct. `memory(read)` does not imply termination, absence of
unwinding, or absence of synchronization. `captures(none)` does not add a Go
escape-analysis `noescape` promise. Incorrect contracts can invalidate compiler
optimizations.

## ABI and compilation boundaries

Contracts are collected during package syntax preloading and shared with backend
programs. Caller-side declarations and definitions receive the same available
contracts, including resolved linkname aliases. Generic instances use the source
declaration's selectors and validate concrete types. Contracts are not inherited
by unknown indirect calls through ordinary Go function values or interfaces.

Build-specific contracts can be attached through typed `go:linkname` declarations
in files with ordinary Go build constraints. The runtime uses this for helpers
whose native contracts do not include the extra root publication in linear-memory
wasm GC. That mode keeps conservative declarations for those helpers; checked
pointer returns, lengths and panic contracts that remain valid stay annotated.

The initial implementation supports direct scalar/pointer values. If other
aggregate arguments cause a signature to be recreated, unchanged scalar result
and parameter attributes are remapped. Hidden closure environment arguments are
excluded from source numbering.

Contracts on components of multiple results, aggregate fields, packed values or
indirect result fields are not yet supported. In particular, a non-null pointer
stored inside an sret object cannot be described by marking the sret buffer
address non-null. Unsupported transformations are diagnosed. Compiler-generated
GC root publication and cooperative safepoints are also diagnosed when their
effects cannot be reconciled with the annotated contracts in this version.

LLGo retains the logical contracts separately from ABI indices. The internal
representation can describe fragments, field paths and indirect storage for
future extensions. Retaining the contract does not, by itself, make LLVM consume
it: each supported lowering must generate the appropriate optimizer-visible fact.

Additional attributes, aggregate/indirect result contracts, and the proposed
`//llgo:cdecl` / `//llgo:stdcall` spellings are follow-up work. Existing
`llgo:type C`, `llgo:type stdcall`, and native linking conventions remain in use.
