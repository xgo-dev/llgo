# Source function attributes

Proposal: [#2518](https://github.com/xgo-dev/llgo/issues/2518).
Implementation: [#2572](https://github.com/xgo-dev/llgo/pull/2572).

## 1. Which attributes are proposed?

Only three kinds of objects can be annotated: **the function, a whole parameter,
or one whole Go result**.

| Object | Attributes |
| --- | --- |
| Pointer parameter/result | `nonnull` |
| Integer parameter/result | `range(lo, hi)`, `nonnegative` |
| Pointer/integer result | `same_as(param(...))` |
| Pointer parameter | `access(...)`, `capture(...)` |
| Function | `memory(...)`, `noreturn`, `cold` |

Select parameters and results with `param(name|index)` and
`result(name|index)`. Indices start at zero in the Go signature and exclude
compiler-added parameters and the receiver. `receiver` selects a method's
receiving parameter; it uses the same input attributes. Attributes without a
selector apply to the function.

There is no `field` or `element` syntax. For example, `nonnull` can describe
a pointer result, but cannot describe a pointer member inside a struct result.
Attributes retain their type requirements; a whole struct is not a pointer.

`align` is deferred. `nofree`, `nosync`, `nounwind`, and `willreturn` are
not accepted as source annotations. The compiler handles these internally.
They are not universally true defaults: LLVM may infer them from a function
body, but cannot assume them across arbitrary external calls or infinite loops.

## 2. What does each attribute mean?

| Attribute | Meaning |
| --- | --- |
| `nonnull` | The pointer is not nil. It does not by itself guarantee valid dereferencing. |
| `range(lo, hi)` | `lo <= value && value < hi`. Bounds must fit the Go integer type, without wrapping. |
| `nonnegative` | `value >= 0`; the width of int follows the target. |
| `same_as(param(p))` | The result equals the value of p when the function was entered, before any reassignment. Integer types must match; pointer conversions must preserve the pointer. |
| `memory(...)` | Restricts this call and its callees' access through input pointers, to global memory and to external state. Local variables and ABI argument/result copies are handled separately. |
| `access(none/read/write/readwrite)` | Permits no reads/writes, only reads, only writes, or both through the selected pointer parameter. It does not restrict saving that pointer. |
| `capture(none)` | Does not save the pointer or its address anywhere visible outside the call, even temporarily, or return it. |
| `capture(results)` | May save the pointer only by returning it, including within a returned struct. |
| `capture(any)` | No restriction on saving the pointer. |
| `noreturn` | Never returns normally; it may panic or run indefinitely. |
| `cold` | Calls are expected to be uncommon; an optimization hint. |

Memory modes are none/read/write/readwrite. Examples:

- `memory(none)`: no access through input pointers, to globals or to external state.
- `memory(read)`: reads are allowed, writes are not.
- `memory(args: read)`: only reads through pointers obtained from input parameters.
- `memory(args: readwrite, other: read)`: reads/writes through input pointers;
  other memory is only read.

A default mode applies to both args and other; an explicit location overrides
it. Without a default, unspecified locations are none. Access through a global
is other even if its pointer happens to equal an input. A second pointer loaded
from input-addressed memory is not automatically part of args.

Clocks, entropy, I/O and volatile/device observations require
`other: readwrite`, so repeated observations are not merged merely because
there is no intervening ordinary memory write. Reading ordinary globals only
needs `other: read`.

For example:

```go
//llgo:attribute result(out) nonnull same_as(param(p))
//llgo:attribute result(count) range(0, 64)
func Make(p *int, n uint32) (out *int, count uint32) {
    if p == nil {
        panic("nil pointer")
    }
    return p, n & 63
}
```

These describe successful returns: out is non-nil and equals the input p, and
count is below 64. They do not prohibit calling Make(nil, n), which still panics.

Input attributes describe entry values. Result attributes hold after normal
return, including defer changes. They are programmer promises, not runtime
checks. LLGo checks syntax and types but does not prove arbitrary bodies;
incorrect promises can cause incorrect optimization.

## 3. How are the attributes implemented under different ABIs?

An ABI can pass values directly, pack/split them into registers, or pass an
argument-copy address (including LLVM byval). It can also place multiple Go
results in caller-provided memory, using sret.

Even when several results share registers or sret storage, each `result(i)`
still refers to the corresponding complete Go result. No source field syntax
is needed to locate it.

`llvm.assume(condition)` tells LLVM a condition is guaranteed; it does not
perform a runtime check.

| Attribute | Direct matching LLVM value | After packing/splitting or indirect return |
| --- | --- | --- |
| `nonnull` | LLVM nonnull and a non-null assumption. | Extract/load the selected Go pointer result and assume it is non-null. Do not annotate the sret storage address instead. |
| `range` / `nonnegative` | LLVM range and the corresponding integer condition. | Extract/load the selected Go integer at its original width and apply the condition. Do not constrain unrelated bits in the same register. |
| `same_as` | LLVM returned when suitable; use the entry input value in place of result uses. | After the call, matching reads of the selected Go result can use the entry input value. Other results keep their actual returned values. |
| `noreturn` | LLVM noreturn. | Keep it if the generated function still cannot return normally. |
| `cold` | LLVM cold. | Independent of how values are passed. |

Result assumptions belong after a normal return, not before a possibly panicking
call. With LLVM invoke, they go only on its normal path. same_as uses the input
value from entry, not a later reload of changed input memory.

The compiler inserts these operations before ABI signature conversion.
Existing ABI conversion adjusts the parameters/results they refer to; no
additional wrapper call is required.

Memory and pointer-saving attributes need different treatment:

| Attribute | Direct pointer parameters | ABI adjustments |
| --- | --- | --- |
| `memory` | Use the corresponding LLVM memory attribute. | Include indirect argument reads and sret writes. If input pointers are inside an aggregate and LLVM cannot classify their accesses as argument memory, permit the corresponding other-memory accesses too. |
| `access` | Use LLVM parameter readnone/readonly/writeonly; readwrite adds no restriction. | Preserve only on a matching pointer parameter. An aggregate storage address must not stand in for a different pointer value. |
| `capture(none)` | Use LLVM captures(none). | Keep it on a matching pointer input. Returning that pointer via sret would violate the source promise. |
| `capture(results)` | Use LLVM return-only capture. | Remove that native restriction when results use sret: writing through sret is a memory store, not an LLVM return value. |
| `capture(any)` | No restrictive LLVM attribute. | No additional restriction. |

For a source function with memory(none), ABI copies alone may require:

| Added operations | LLVM attribute |
| --- | --- |
| Direct values/register packing only | `memory(none)` |
| Read an indirect argument copy | `memory(argmem: read)` |
| Write result storage | `memory(argmem: write)` |
| Both | `memory(argmem: readwrite)` |

Where LLVM cannot express a restriction correctly, the implementation permits
more operations and loses some optimization. It does not attach an incorrect
attribute to another value. These choices are recorded for inspection.

Compiler-added GC pointer registration or scheduling checks can perform further
operations. The current implementation removes incompatible native restrictions
consistently from both definitions and imported declarations. Other unmodelled
runtime work, such as implicit allocation or defer support, is diagnosed if it
would conflict with a retained source restriction.

Current limits: pointer same_as assumes the existing non-moving collectors.
Unsupported invoke signature changes are diagnosed. Unknown function-value or
interface calls do not inherit a particular callee's attributes.
