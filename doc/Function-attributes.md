# Source function attributes

Design: [#2518](https://github.com/xgo-dev/llgo/issues/2518).

## Summary

Use `//llgo:attr` to describe function properties and parameter and result guarantees in Go source code. Ordinary packages and the LLGo runtime use the same mechanism. Attributes describe the Go declaration consistently across targets, regardless of how the ABI passes arguments or returns results.

## Background and goals

The runtime work in [#2510](https://github.com/xgo-dev/llgo/pull/2510) identified properties that implementations already guarantee but that callers cannot always use for optimization. Examples include a checked pointer being non-nil on normal return, a copy helper returning its destination, and map lengths being nonnegative.

Placing this information next to a declaration makes it available to ordinary packages and to callers compiled separately.

This proposal aims to:

- Provide attributes with explicit meanings, applicable types and validation rules.
- Use one `//llgo:attr` syntax for function, parameter and result attributes.
- Preserve attribute meanings across ABI conversion, imports, generic instantiation and cached builds.
- Express runtime properties through the same source annotations available to ordinary packages.

## Specification

### Syntax and applicable declarations

```go
//llgo:attr result(out) nonnull same_as(param(p))
//llgo:attr result(count) range(0, 64)
func Make(p *int, n uint32) (out *int, count uint32) {
    if p == nil {
        panic("nil pointer")
    }
    return p, n & 63
}
```

On normal return, out is non-nil and equals the entry value of p, and count is below 64. Calling Make(nil, n) still panics.

Attributes apply to named function and method declarations, including external declarations. A selector identifies the whole source parameter or one whole Go result:

| Selector | Meaning |
| --- | --- |
| No selector | The function. |
| `param(name)` / `param(index)` | A parameter in the Go signature. |
| `result(name)` / `result(index)` | One result in the Go signature. |
| `receiver` | The method's receiving parameter, using the same input attributes. |

Indices start at zero and count source parameters or results. The receiver has its own selector. Unnamed parameters/results and those named `_` are selected by index. ABI changes preserve this numbering.

Multiple attributes on a line are separated by whitespace. Multiple annotation lines are combined independently of order. Both `//llgo:attr` and `// llgo:attr` are accepted.

### Attributes

| Attribute | Applicable object | Meaning |
| --- | --- | --- |
| `nonnull` | Pointer parameter/result | The pointer is not nil. Valid dereferencing still depends on the referenced memory. |
| `range(lo, hi)` | Integer parameter/result | `lo <= value && value < hi`, using integer-literal bounds and a nonempty, non-wrapping interval within the Go type's domain. |
| `nonnegative` | Integer parameter/result | `value >= 0`. The width of int follows the target. On unsigned integers this adds no restriction. |
| `same_as(param(p))` / `same_as(receiver)` | Pointer/integer result | The result equals the selected input's entry value. Integer types must match; pointer conversions must preserve the pointer. |
| `memory(...)` | Function | Restricts the whole call and its callees' access through input pointers, to globals and to external state. Local variables and ABI copies are accounted for separately. |
| `access(none/read/write/readwrite)` | Pointer parameter | Allows no reads/writes, reads, writes, or both through this particular pointer and pointers derived from it. |
| `capture(none)` | Pointer parameter | The pointer or its address is never saved anywhere visible outside the call, even temporarily or through a result. |
| `capture(results)` | Pointer parameter | Allows saving the pointer only through returned values, including within a returned struct. |
| `capture(any)` | Pointer parameter | Allows unrestricted saving of the pointer. |
| `noalias` | Pointer parameter | During the call, memory accessed through this pointer that is modified by any means must be accessed only through this pointer or pointers derived from it. Shared reads of unmodified memory remain allowed. |
| `noreturn` | Function | The function never returns normally; it may panic or run indefinitely. |
| `cold` | Function | Calls are expected to be uncommon; an optimization hint. |

The modes for `memory` are none/read/write/readwrite:

- `memory(none)`: no access through input pointers, to globals or to external state.
- `memory(read)`: reads are allowed, writes are not.
- `memory(args: read)`: only reads through input-derived pointers are allowed.
- `memory(args: readwrite, other: read)`: input-derived accesses may read or write; other accesses may only read.

A default mode applies to both args and other; an explicit location overrides it. Without a default, unspecified locations are none.

The classification follows how an access obtains its pointer. Access through a global is other even if the pointer equals an input. A pointer loaded from input-addressed memory is not automatically part of args.

Clocks, entropy, I/O and volatile/device observations require `other: readwrite`. Ordinary global reads require `other: read`.

`memory` and `access` apply together: `memory` limits the whole call, while `access` limits the accesses through a selected pointer parameter. An access must satisfy both. For example:

```go
//llgo:attr memory(args: readwrite)
//llgo:attr param(dst) access(write)
//llgo:attr param(src) access(read)
func CopyOne(dst, src *int) {
    *dst = *src
}
```

The function may read through src and write through dst; its memory attribute excludes global and other-memory accesses. Overlap remains permitted. `access(read)` alone would allow the function to write through other parameters or to globals. Conversely, `memory(read)` would prohibit writes even with `param(dst) access(write)`: the latter grants no exception to the function-wide restriction.

The `noalias` rule follows [LLVM's pointer-parameter semantics](https://releases.llvm.org/22.1.0/docs/LangRef.html#parameter-attributes). For `param(p) noalias`, independent accesses through another parameter or a global violate the promise when they access modified memory also accessed through p. The rule concerns accessed memory, rather than pointer-address inequality alone, and lasts for this call. Nullability and saving the pointer are described separately by `nonnull` and `capture`.

### Validity and combinations

Parameter value attributes apply to entry values. Result attributes hold after normal return, including changes made by deferred functions. `memory`, `access`, `capture` and `noalias` cover the whole call, including functions it calls.

Value and behavior attributes are programmer promises. LLGo checks syntax, types and known conflicts; it relies on the author for properties of an arbitrary implementation. Incorrect promises can cause incorrect optimization. The annotations themselves generate no runtime checks.

Attributes compose according to their individual meanings. In particular, `range` and `nonnegative` apply their intersection, which must be nonempty. `access` restricts reads/writes, `capture` restricts saving a pointer, and `noalias` restricts independent accesses to modified memory. These are separate guarantees. A function that permits overlapping copies must retain valid overlapping calls.

## ABI implementation

An ABI can pass values directly, pack or split them into registers, or pass an argument-copy address, including LLVM byval. Results can also be written into caller-provided storage through sret. In every case, `param(i)` and `result(i)` refer to the corresponding whole Go values.

`llvm.assume(condition)` communicates a guaranteed condition to LLVM without adding a runtime check.

| Attribute | Direct matching LLVM value | Packing, splitting or indirect passing |
| --- | --- | --- |
| `nonnull` | LLVM nonnull and a non-null assumption. | Recover the selected Go pointer and apply the non-null condition to that value. |
| `range` / `nonnegative` | LLVM range and the corresponding integer condition. | Recover the selected integer at its Go width and apply the condition to it. |
| `same_as` | LLVM returned when suitable; result uses can reuse the entry input value. | Reads of the selected Go result can reuse the entry input value after the call; other results retain their actual values. |
| `noreturn` | LLVM noreturn. | Preserve it on the generated function that never returns normally. |
| `cold` | LLVM cold. | Preserve the hint independently of value representation. |

Parameter value assumptions belong at entry; result assumptions belong on the normal return path. `same_as` uses the entry input value even if memory supplying that value changes during the call.

Memory-related attributes require the following adjustments:

| Attribute | LLVM implementation | ABI adjustments |
| --- | --- | --- |
| `memory` | Corresponding LLVM memory attribute. | Include indirect argument reads and sret writes. Allow corresponding other-memory accesses when pointers inside an aggregate cannot be classified as LLVM argument memory. |
| `noalias` | LLVM parameter noalias. | Preserve it on the corresponding pointer parameter when other arguments or results change. |
| `access` | LLVM parameter readnone/readonly/writeonly; readwrite is unrestricted. | Preserve the restriction on the corresponding pointer parameter. |
| `capture(none)` | LLVM captures(none). | Preserve it on the corresponding pointer parameter. |
| `capture(results)` | LLVM return-only capture. | Allow capture through memory when sret implements the source return. |
| `capture(any)` | Unrestricted capture. | Retain unrestricted capture. |

A pointer's attributes describe that pointer, including when it is stored inside ABI transfer storage. The transfer-storage address represents a different value. Where an ABI representation prevents LLVM from expressing a restriction precisely, use weaker LLVM restrictions and retain correct program behavior.

For a source `memory(none)` function, ABI copies alone may require:

| ABI operation | LLVM memory attribute |
| --- | --- |
| Direct values or register packing | `memory(none)` |
| Read an indirect argument copy | `memory(argmem: read)` |
| Write result storage | `memory(argmem: write)` |
| Both | `memory(argmem: readwrite)` |

Compiler-added GC and scheduling operations must also be accounted for consistently in definitions and imported declarations. Required operations that cannot be reconciled with retained restrictions produce a diagnostic.

## Validation and compatibility

The compiler reports invalid attributes, selectors, types, arguments, bounds and conflicting declarations at the source annotation. Identical repeated annotations may be combined. Results must be independent of annotation order and package loading order.

Definitions and imported declarations share applicable attributes. Known declarations of the same function, including those connected by `go:linkname`, must agree. Generic attributes are validated against concrete types. Cached builds must reflect annotation changes.

A call may use a known callee's value and behavior guarantees.

Existing Go directives, native bindings, export declarations and linkname behavior remain compatible.

## Implementation and acceptance

Implementation comprises declaration parsing and validation, propagation across packages and caches, ABI handling, and migration of applicable runtime properties to source annotations. Related implementation work is tracked in [PR #2572](https://github.com/xgo-dev/llgo/pull/2572).

Acceptance requires:

1. Syntax, type and conflict checks for the supported declaration and attribute combinations.
2. Consistent behavior across definitions, imports, methods, generic instances, linkname declarations and cached builds.
3. Correct optimization across direct, packed, split and indirect arguments/results, including multiple Go results.
4. Preservation of nil-triggered panic, results after defer, valid overlapping copies and reads separated by relevant writes.
5. `noalias` optimization on valid calls, acceptance of shared reads of unmodified memory, and correct parameter mapping after ABI conversion.
6. Runtime tests in the applicable GC modes, together with target-specific IR and assembly checks.
