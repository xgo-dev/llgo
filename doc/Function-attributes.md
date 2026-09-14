# Function and parameter attributes

Design: [#2590](https://github.com/xgo-dev/llgo/issues/2590).

Introduce `//llgo:` source annotations to describe function properties and parameter and result guarantees in Go source code. Ordinary packages and the LLGo runtime use the same mechanism. Attributes describe the Go declaration consistently across targets, regardless of how the ABI passes arguments or returns results.

## Background and goals

The runtime work in [#2510](https://github.com/xgo-dev/llgo/pull/2510) identified properties that implementations already guarantee but that callers cannot always use for optimization — a checked pointer being non-nil on normal return, a copy helper returning its destination, map lengths being nonnegative. Placing this information next to a declaration makes it available to ordinary packages and to callers compiled separately.

This proposal aims to:

- Provide attributes with explicit meanings, applicable types and validation rules.
- Use one `//llgo:` syntax for function properties.
- Preserve attribute meanings across ABI conversion, imports, generic instantiation and cached builds.
- Express runtime properties through the same source annotations available to ordinary packages.

Attributes apply to named function and method declarations, including external declarations, and split into two groups: **function attributes**, written with no selector, and **parameter and result attributes**, written with a `param`, `result`, or `receiver` selector.

## Function attributes

These attributes take no selector — they describe the call as a whole, not a specific parameter or result. Each attribute below is independent and is written on its own `//llgo:` line; a declaration may combine several of them.

### Noreturn

`noreturn` states that the function never returns normally; it may panic or run indefinitely.

```go
//llgo:noreturn
func Fatal(msg string) {
    panic(msg)
}
```

Callers and the optimizer can treat any code path after a call to `Fatal` as unreachable.

### Cold

`cold` is an optimization hint: calls to the function are expected to be uncommon, so the compiler may deprioritize the call path for size or speed.

```go
//llgo:cold
func logRareEvent(err error) {
    ...
}
```

## Parameter and result attributes

These attributes use a `param`, `result`, or `receiver` selector to identify the single parameter, result, or receiver they describe:

```go
//llgo:result(out) nonnull sameas(p)
//llgo:result(count) range(0, 64)
func Make(p *int, n uint32) (out *int, count uint32) {
    if p == nil {
        panic("nil pointer")
    }
    return p, n & 63
}
```

On normal return, out is non-nil and equals the entry value of p, and count is below 64. Calling Make(nil, n) still panics.

Multiple attributes on a line are separated by whitespace. Multiple annotation lines are combined independently of order. Both the compact `//llgo:` and spaced `// llgo:` forms are accepted.

| Selector | Meaning |
| --- | --- |
| `param(name)` / `param(index)` | A parameter in the Go signature. |
| `result(name)` / `result(index)` | One result in the Go signature. |
| `receiver` | The method's receiving parameter, using the same attributes as a parameter selector. |

**Indexing.** `index` is **0-based**: index `0` is the first source parameter or result, index `1` is the second, and so on. Indices count declared parameters/results in source order, including unnamed ones and those named `_`. ABI changes (packing, splitting, indirect passing, sret, etc.) never renumber parameters or results — `param(i)` and `result(i)` always refer back to the corresponding whole Go value.

**Single-result shorthand.** When a function declares exactly one result, `result` may be written with no selector at all — `//llgo:result` — as shorthand for `result(0)`:

```go
//llgo:result nonnull
func Lookup(key string) *Entry {
    ...
}
```

This shorthand is only valid when the function has exactly one result; functions with two or more results must select each one explicitly with `result(name)` or `result(index)`.

Parameter and result attributes describe either a fact about the value itself, or an access rule for a pointer parameter:

| Attribute | Applicable object | Meaning |
| --- | --- | --- |
| `nonnull` | Pointer parameter/result | The pointer is not nil. Valid dereferencing still depends on the referenced memory. |
| `range(lo, hi)` | Integer parameter/result | `lo <= value && value < hi`, using integer-literal bounds and a nonempty, non-wrapping interval within the Go type's domain. |
| `nonnegative` | Integer parameter/result | `value >= 0`. The width of int follows the target. On unsigned integers this adds no restriction. |
| `sameas(name)` | Pointer/integer result | The result equals the entry value of the parameter named `name`. Integer types must match; pointer conversions must preserve the pointer. |
| `access(none/read/write/readwrite)` | Pointer parameter | Allows no reads/writes, reads, writes, or both through this particular pointer and pointers derived from it. |
| `noalias` | Pointer parameter | During the call, memory accessed through this pointer that is modified by any means must be accessed only through this pointer or pointers derived from it. Shared reads of unmodified memory remain allowed. |

Parameter value attributes (`nonnull`, `range`, `nonnegative`) apply to entry values; result forms, including `sameas`, hold after normal return, including changes made by deferred functions. `access` and `noalias` instead describe a pointer's access pattern during the call, and cover the whole call, including functions it calls.

```go
//llgo:param(dst) access(write)
//llgo:param(src) access(read)
func CopyOne(dst, src *int) {
    *dst = *src
}
```

`CopyOne` reads through src and writes through dst. These parameter attributes do not impose a function-wide memory restriction: `access(read)` alone would still permit writes through other parameters or to globals, and `access(write)` alone would still permit reads through other parameters.

The `noalias` rule follows [LLVM's pointer-parameter semantics](https://releases.llvm.org/22.1.0/docs/LangRef.html#parameter-attributes). For `param(p) noalias`, independent accesses through another parameter or a global violate the promise when they access modified memory also accessed through p. The rule concerns accessed memory, rather than pointer-address inequality alone, and lasts for this call. Nullability is described separately by `nonnull`.

## Validity and combinations

Value and behavior attributes are programmer promises. LLGo checks syntax, types and known conflicts; it relies on the author for properties of an arbitrary implementation. Incorrect promises can cause incorrect optimization. The annotations themselves generate no runtime checks.

Attributes compose according to their individual meanings. In particular, `range` and `nonnegative` apply their intersection, which must be nonempty. `access` restricts reads/writes and `noalias` restricts independent accesses to modified memory; these are separate guarantees. A function that permits overlapping copies must retain valid overlapping calls.

## ABI implementation

An ABI can pass values directly, pack or split them into registers, or pass an argument-copy address, including LLVM byval. Results can also be written into caller-provided storage through sret. In every case, `param(i)` and `result(i)` refer to the corresponding whole Go values.

`llvm.assume(condition)` communicates a guaranteed condition to LLVM without adding a runtime check.

### Function attributes

| Attribute | Direct matching LLVM value | Packing, splitting or indirect passing |
| --- | --- | --- |
| `noreturn` | LLVM noreturn. | Preserve it on the generated function that never returns normally. |
| `cold` | LLVM cold. | Preserve the hint independently of value representation. |

### Parameter and result attributes

| Attribute | Direct matching LLVM value | Packing, splitting or indirect passing |
| --- | --- | --- |
| `nonnull` | LLVM nonnull and a non-null assumption. | Recover the selected Go pointer and apply the non-null condition to that value. |
| `range` / `nonnegative` | LLVM range and the corresponding integer condition. | Recover the selected integer at its Go width and apply the condition to it. |
| `sameas` | LLVM returned when suitable; result uses can reuse the entry input value. | Reads of the selected Go result can reuse the entry input value after the call; other results retain their actual values. |
| `access` | LLVM parameter readnone/readonly/writeonly; readwrite is unrestricted. | Preserve the restriction on the corresponding pointer parameter. |
| `noalias` | LLVM parameter noalias. | Preserve it on the corresponding pointer parameter when other arguments or results change. |

Parameter value assumptions belong at entry; result assumptions belong on the normal return path. `sameas` uses the entry input value even if memory supplying that value changes during the call.

A pointer's attributes describe that pointer, including when it is stored inside ABI transfer storage. The transfer-storage address represents a different value. Where an ABI representation prevents LLVM from expressing a restriction precisely, use weaker LLVM restrictions and retain correct program behavior.

Compiler-added GC and scheduling operations must also be accounted for consistently in definitions and imported declarations. Required operations that cannot be reconciled with retained restrictions produce a diagnostic.

## Validation and compatibility

The compiler reports invalid attributes, selectors, types, arguments, bounds and conflicting declarations at the source annotation. Identical repeated annotations may be combined. Results must be independent of annotation order and package loading order.

Definitions and imported declarations share applicable attributes. Known declarations of the same function, including those connected by `go:linkname`, must agree. Generic attributes are validated against concrete types. Cached builds must reflect annotation changes.

A call may use a known callee's value and behavior guarantees. Existing Go directives, export declarations and linkname behavior remain compatible.

## Implementation and acceptance

Implementation comprises declaration parsing and validation, propagation across packages and caches, ABI handling, and migration of applicable runtime properties to source annotations. Related implementation work is tracked in [PR #2572](https://github.com/xgo-dev/llgo/pull/2572).

Acceptance requires:

1. Syntax, type and conflict checks for the supported declaration and attribute combinations.
2. Consistent behavior across definitions, imports, methods, generic instances, linkname declarations and cached builds.
3. Correct optimization across direct, packed, split and indirect arguments/results, including multiple Go results.
4. Preservation of nil-triggered panic, results after defer, and reads separated by relevant writes.
5. `noalias` optimization on valid calls, acceptance of shared reads of unmodified memory, and correct parameter mapping after ABI conversion.
6. Runtime tests in the applicable GC modes and native interoperability tests on the relevant platforms, together with target-specific IR and assembly checks.

## 中文说明

通过 `//llgo:` 声明函数、完整参数和单个完整返回值的属性。普通包与运行时使用同一套机制，参数和返回值的编号始终对应 Go 声明，即使 ABI 将它们打包、拆分或改为间接传递，也不会改变编号。

函数属性为 `//llgo:cold` 和 `//llgo:noreturn`，每个属性独占一行。参数和返回值使用 `//llgo:param(名称或下标)`、`//llgo:result(名称或下标)`，方法接收者使用 `//llgo:receiver`；下标从零开始。恰好有一个返回值时，可以简写为 `//llgo:result`。同一参数或返回值的多个属性以空格分隔，多行声明合并处理。

| 属性 | 适用对象 | 含义 |
| --- | --- | --- |
| `nonnull` | 指针参数或返回值 | 指针不为 nil；可解引用性仍取决于所指内存。 |
| `range(lo, hi)` | 整数参数或返回值 | 值位于左闭右开的非空区间，边界必须符合 Go 类型的取值范围。 |
| `nonnegative` | 整数参数或返回值 | 值大于等于零；对无符号整数不增加限制。 |
| `sameas(name)` | 指针或整数返回值 | 等于指定名称参数在函数入口处的值。 |
| `access(none/read/write/readwrite)` | 指针参数 | 限制通过该指针及其派生指针的读写，不限制其他参数或全局内存。 |
| `noalias` | 指针参数 | 调用期间，经该指针访问且被任何途径修改的内存，只能经该指针或其派生指针访问；允许共享读取未修改的内存。 |
| `noreturn` | 函数 | 不会正常返回，可以 panic 或一直运行。 |
| `cold` | 函数 | 调用预计较少，作为优化提示。 |

参数值属性在入口处成立，返回值属性在正常返回后成立，包括 defer 对结果的修改；`access` 和 `noalias` 约束整个调用。属性是程序员提供的保证，编译器检查语法、类型与已知冲突，不插入运行时检查。

编译器在跨包导入、泛型实例化、linkname 和缓存构建中保留属性，并在 ABI 转换后将其应用到对应的 Go 值。GC 插桩与隐式运行时操作也计入处理；无法精确表达的 LLVM 限制会保守放宽，无法兼容的情况给出诊断。
