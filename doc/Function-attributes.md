# Function attributes

This implements the function attributes in [proposal #2590](https://github.com/xgo-dev/llgo/issues/2590).

Write each attribute on its own line immediately before a named function or method declaration. Both `//llgo:` and `// llgo:` are accepted. Attributes can be combined and repeated.

```go
//llgo:cold
//llgo:noreturn
func Fatal(message string) {
    panic(message)
}
```

| Attribute | Meaning |
| --- | --- |
| `cold` | Calls are expected to be uncommon; this is an optimization hint. |
| `noreturn` | The function never returns normally. It may panic or run indefinitely. |

`noreturn` is a programmer promise. LLGo checks the declaration and directive syntax, but does not prove the function's behavior or add runtime checks. A function whose deferred recovery permits a normal return must not declare `noreturn`. Callers may treat the normal continuation after a `noreturn` call as unreachable; panic unwinding and recovery by the caller remain valid.

The properties apply to definitions and imported declarations, including methods, generic instances and declarations connected by linkname. They are preserved by ABI conversion and cached builds. The runtime uses the same source directives as ordinary packages.

## Parameter and single-result attributes

Use `//llgo:param(name|index)`, `//llgo:receiver`, or `//llgo:result(name|index)` to select a whole Go value. Indices start at zero; the receiver is separate. A function with exactly one result also permits `//llgo:result`. Both comment spacings accepted for function attributes are accepted here.

| Attribute | Applicable value | Meaning |
| --- | --- | --- |
| `nonnull` | Pointer | The value is not nil. |
| `range(lo, hi)` | Integer | `lo <= value && value < hi`, with literal bounds within the source type's domain. |
| `nonnegative` | Integer | The value is at least zero; this adds no restriction to unsigned integers. |

Multiple attributes are separated by whitespace and multiple lines are combined. `range` and `nonnegative` use their intersection, which must be nonempty. Parameter guarantees apply at entry; result guarantees apply on normal return, including deferred changes. These are programmer promises, not runtime checks.

```go
//llgo:result nonnull
func Checked(p *int) *int {
    if p == nil { panic("nil pointer") }
    return p
}
```

Source selectors are resolved before exported signatures can lose parameter names. Imports, linkname declarations and generic instances preserve the guarantees; concrete generic types are checked at instantiation. ABI conversion preserves attributes on the corresponding scalar values when other arguments are packed or passed indirectly.

Result attributes currently require exactly one result. Multiple-result guarantees, `sameas`, `access` and `noalias` are subsequent steps of #2590.

