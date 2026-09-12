# Source function attributes

Proposal: [#2518](https://github.com/xgo-dev/llgo/issues/2518).
Implementation for review: [#2572](https://github.com/xgo-dev/llgo/pull/2572).

## 1. Which attributes are proposed?

Use `//llgo:attribute` on a function or method declaration:

- Parameters and results: `nonnull`, `range`, `nonnegative`, `align`.
- A result equal to an input: `same_as`.
- Memory access and pointer retention: `memory`, `access`, `capture`.
- Function execution: `nofree`, `nosync`, `nounwind`, `willreturn`, `noreturn`.
- Optimization hint: `cold`.

These apply to ordinary Go packages and the runtime. This proposal does not add
allocation-size, no-alias or lifetime attributes.

## 2. What does each attribute mean?

### Parameters and results

| Attribute | Applies to | Meaning |
| --- | --- | --- |
| `nonnull` | Pointer parameter or result | The pointer is not nil. |
| `range(lo, hi)` | Integer parameter or result | `lo <= value && value < hi`. Bounds must fit the Go type; the interval cannot wrap around. |
| `nonnegative` | Integer parameter or result | `value >= 0`. The width of `int` follows the target platform. |
| `align(N)` | Pointer parameter or result | The pointer address is a multiple of N, a positive power of two. This alone permits nil and does not promise that dereferencing is valid. |
| `same_as(param(p))` | Pointer or integer result | The result equals the value of parameter p at function entry. It does not mean the value of p after reassignment. `same_as(receiver)` also works. |

Integer `same_as` requires the same Go type, with type aliases resolved.
Pointer types may differ if the conversion preserves the pointer, including
conversion to or from `unsafe.Pointer`.

Choose a parameter, result or receiver with `param(name|index)`,
`result(name|index)` or `receiver`. Indices start at zero in the Go signature,
excluding the receiver and compiler-added parameters. Use `.field(Name)` for a
struct field and `.element(index)` for a fixed-array element. These do not
automatically dereference a pointer.

For example:

```go
type Result struct {
    P *int
    N uint32
}

//llgo:attribute result(out).field(P) nonnull same_as(param(p))
//llgo:attribute result(out).field(N) range(0, 64)
func Make(p *int, n uint32) (out Result) {
    if p == nil {
        panic("nil pointer")
    }
    return Result{P: p, N: n & 63}
}
```

The two lines say: when Make returns normally, out.P is non-nil and equals the
input p, and out.N is below 64. They do **not** say that passing nil is forbidden:
Make(nil, n) still panics.

Parameter attributes describe entry values. Result attributes apply after a
normal return, including any defer that changes a result. These are promises
from the programmer, not inserted runtime checks. LLGo checks syntax and types
but does not prove every function body. Incorrect promises can produce
incorrect optimized code.

### Memory access and pointer retention

| Attribute | Applies to | Meaning |
| --- | --- | --- |
| `memory(...)` | Function | Restricts access through input pointers, to global memory and to external state during this call and the calls it makes. Local variables and ABI copies are handled separately below. |
| `access(none/read/write/readwrite)` | Pointer parameter, including a pointer field | Permits neither reading nor writing, only reading, only writing, or both through that pointer. This does not by itself prohibit retaining the pointer. |
| `capture(none)` | Pointer parameter, including a pointer field | Does not save the pointer or its address anywhere visible outside the call, or return it. Even temporary publication in a global is prohibited. |
| `capture(results)` | Same | May retain the pointer only by returning it, including inside a returned struct. |
| `capture(any)` | Same | No restriction on retaining the pointer. |

For `memory`, the modes are `none`, `read`, `write` and `readwrite`:

- `memory(none)`: no access through input pointers, to globals or to external state.
- `memory(read)`: reads are permitted, writes are not.
- `memory(args: read)`: only reads through pointers obtained from input parameters.
- `memory(args: readwrite, other: read)`: reads/writes through input pointers,
  but only reads of other memory.

A default mode applies to both `args` and `other`; an explicit location
overrides it. With no default, unspecified locations are `none`.

For `args`, the way the pointer is obtained matters. Access through a global
is `other`, even if that global happens to equal an input pointer. Loading a
second pointer from the memory addressed by an input does not automatically
make accesses through that second pointer part of `args`.

Clock/entropy observations, I/O and volatile/device interactions require
`other: readwrite`, so the optimizer cannot merge repeated observations merely
because the program made no intervening memory write. Reading ordinary program
globals only needs `other: read`.

### Function execution

| Attribute | Meaning |
| --- | --- |
| `nofree` | Does not free storage that already existed before the call, including through another function. |
| `nosync` | Does not synchronize with other threads, including through another function. |
| `nounwind` | Does not propagate a panic or other stack unwinding to its caller. A panic recovered internally is not excluded. |
| `willreturn` | The call eventually finishes, normally or by unwinding; it cannot run indefinitely. |
| `noreturn` | The call never returns normally. It can panic or run indefinitely. |
| `cold` | Calls are expected to be uncommon; this is an optimization hint. |

These are independent. For example, `memory(read)` does not imply `nosync` or
`willreturn`. A function that always propagates a panic can have both
`willreturn` and `noreturn`.

## 3. How is each attribute implemented under different ABIs?

An ABI determines how arguments and results are passed. The cases that matter
here are:

1. **Direct values:** a Go pointer or integer is also an LLVM pointer or integer
   parameter/result.
2. **Packed or split values:** a struct is passed as one integer or several
   registers. Its fields must be extracted before applying field attributes.
3. **Indirect input / byval:** the caller passes the address of an argument
   copy. This includes LLVM byval; not every target uses that exact attribute.
4. **Indirect result / sret:** the caller supplies storage, and the callee writes
   its result there. Multiple Go results may be combined into this storage.

These are passing forms, not four disjoint architectures. One function can use
several forms. For example, the tests exercise LLVM byval on amd64 and indirect
input passing without that attribute on Darwin arm64.

### nonnull, range, nonnegative, align and same_as

`llvm.assume(condition)` tells LLVM that a condition is guaranteed at that point.
It does not check the condition at runtime.

| Attribute | Direct value | Packed/split fields | Indirect input / byval | Indirect result / sret |
| --- | --- | --- | --- | --- |
| `nonnull` | LLVM `nonnull`; also emit an assumption where the value is used by the implementation. | Extract the pointer field, then assume it is non-nil. | Load the pointer field, then assume it is non-nil. | After the call returns normally, load the pointer field and assume it is non-nil. |
| `range` | LLVM `range` plus an assumption for the interval. | Extract the integer at its Go width, then assume the interval. | Load the integer field, then assume the interval. | Load the returned integer after the call, then assume the interval. |
| `nonnegative` | Use the corresponding LLVM range and `value >= 0` assumption. | Extract the Go integer, then assume it is nonnegative. | Load the integer field, then assume it is nonnegative. | Load the returned integer after the call, then assume it is nonnegative. |
| `align` | LLVM `align` when supported, plus an assumption about the pointer address. | Extract the pointer, then assume the address is aligned. | Load the pointer field, then assume its address is aligned. | Load the returned pointer after the call, then assume its address is aligned. |
| `same_as` | Use LLVM `returned` when parameter/result representations match; replace uses of the result with the entry parameter value. | For matching result-field extractions, use the corresponding entry input value. | Save the input field value before the call; use that value for matching results. | For matching result-field extractions, use the saved input value after the call. Other fields retain their actual returned values. |

Four details prevent wrong implementations:

- A non-null byval/sret **storage address** says nothing about a pointer stored
  **inside** it. In the example, the assumption concerns out.P.
- An `int8` range applies to the extracted 8-bit field, not the whole register
  that also contains other fields.
- Result assumptions belong after a successful return. They must not remove a
  call that may panic. With LLVM invoke, they are placed only on its normal path.
- `same_as` uses the entry input value, not a later load of changed input memory.
  Whole struct copies keep the actual returned struct; the implementation need
  not rebuild a large struct just to replace one field.

The current compiler inserts these operations before changing LLVM function
signatures for the ABI. Existing ABI conversion then replaces the arguments and
results they refer to. No extra wrapper call is required.

### memory, access and capture

These cannot be implemented by assuming a condition about one value. The LLVM
attributes must describe the memory operations that the generated code performs.

| Attribute | Direct pointer parameters/results | Packed or indirect input containing pointers | Indirect result / sret |
| --- | --- | --- | --- |
| `memory` | Use LLVM `memory` with the corresponding permitted reads/writes. | Include reads needed for indirect argument copies. If a pointer inside a struct cannot be described as LLVM argument memory, allow the corresponding accesses to other memory too. | Include writes to result storage. |
| `access` | Use LLVM parameter `readnone`, `readonly` or `writeonly`; readwrite needs no restriction. | The address of the struct is not the pointer field. Where LLVM cannot express the field restriction, omit that optimization. | A direct input pointer keeps its restriction; result-storage writes are accounted for separately. |
| `capture(none)` | Use LLVM `captures(none)` on the input pointer. | If no LLVM pointer parameter represents the selected field, omit that optimization. | Keep it on a direct input pointer. Returning that pointer through sret would violate the programmer's promise. |
| `capture(results)` | Use LLVM's return-only capture attribute on the input pointer. This can include fields of a directly returned struct. | If no LLVM pointer parameter represents the selected field, omit that optimization. | Do not keep LLVM return-only capture: storing a pointer through sret is not LLVM's return-value mechanism. Allow broader capture. |
| `capture(any)` | No restrictive LLVM attribute. | Same. | Same. |

For example, a source function with `memory(none)` may still need these LLVM
attributes solely because of its calling convention:

| Generated operations | LLVM memory attribute |
| --- | --- |
| Direct values or register packing only | `memory(none)` |
| Read an indirect argument copy | `memory(argmem: read)` |
| Write an indirect result | `memory(argmem: write)` |
| Both of the above | `memory(argmem: readwrite)` |

Omitting a field-specific LLVM restriction loses some optimization; it does not
change what the source annotation means. The current implementation records this
choice instead of attaching an incorrect attribute to the struct address.

### nofree, nosync, nounwind, willreturn, noreturn and cold

| Attribute | LLVM implementation | Effect of ordinary packing, byval and sret |
| --- | --- | --- |
| `nofree` | LLVM `nofree` | Keep it when the added operations only copy data and free nothing. |
| `nosync` | LLVM `nosync` | Keep it when the added operations perform no synchronization. |
| `nounwind` | LLVM `nounwind` | Keep it when the added operations cannot unwind. |
| `willreturn` | LLVM `willreturn` | Keep it for finite packing/copying operations. |
| `noreturn` | LLVM `noreturn` | Keep it if the generated function still cannot return normally. |
| `cold` | LLVM `cold` | Independent of how parameters and results are passed. |

If the compiler adds runtime work, simple copying is no longer the whole story.
The current implementation handles this as follows:

- In GC-root-publication or cooperative-safepoint modes, omit restrictive LLVM
  memory/access/capture and nofree/nosync/nounwind/willreturn attributes on both
  definitions and imported declarations. Keep the value attributes, cold and
  noreturn.
- For other runtime operations not yet accounted for, such as implicit heap
  allocation, defer support, per-thread/per-goroutine storage or call logging,
  report an error if the memory, synchronization or execution restrictions
  would otherwise be incorrect.

Current limits: pointer forwarding assumes the existing non-moving collectors;
unusual pointer representations may not support the alignment implementation.
Invoke calls that require unsupported ABI signature changes are diagnosed.
Unknown function-value/interface calls do not inherit a specific callee's
attributes.
