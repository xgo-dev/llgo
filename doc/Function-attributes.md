# Source function contracts

LLGo defines `//llgo:attribute` in terms of Go source values and observable
behavior. LLVM attributes are an implementation mechanism, not the language of
the contracts. Every package, including the runtime, uses the same mechanism.
See [proposal #2518](https://github.com/xgo-dev/llgo/issues/2518).

This revision replaces the original scalar-only prototype and the proposed
general ABI reconstruction framework. Value facts are materialized before ABI
lowering; behavior effects are composed with physical ABI operations.

## Meaning and syntax

```go
//llgo:attribute result(p) nonnull same_as(param(input))
//llgo:attribute result(n) range(0, 64)
func Checked(input *Node) (p *Node, n uint32) {
    if input == nil {
        panic("nil pointer")
    }
    return input, 16
}

//llgo:attribute param(arg).field(P) nonnull
//llgo:attribute result(0).field(Count) range(0, 64)
func Transform(arg Pair) Pair
```

Contracts are unchecked promises. LLGo checks syntax, source types, consistency
and supported representations; it does not prove arbitrary bodies or insert
runtime guards. An invocation violating a semantic contract is outside the
contract's defined behavior. `cold` is only an optimization hint.

Input contracts hold for the logical **entry value**. Result contracts hold only
after **normal completion**, including deferred work. A recovered panic followed
by a return must satisfy result contracts. An unwinding or nonreturning call
provides no result facts. The example accepts nil and panics; its result
postcondition must not become an input precondition.

Both `//llgo:` and `// llgo:` are accepted in declaration comments. Multiple
lines combine; whitespace separates attributes on a line. Selectors are:

* `param(name)` / `param(index)`: zero-based source parameter position.
* `result(name)` / `result(index)`: zero-based source result position.
* `receiver`: the method receiver, separate from parameter numbering.
* `.field(name)` / `.element(index)`: append an explicitly named struct field
  or a fixed-array element selected by a nonnegative integer literal; nested
  paths are supported.

Names resolve during syntax preloading, before import data may lose them.
Grouped parameters occupy separate positions; unnamed and blank values use
indices. Hidden closure environments, sret pointers and ABI fragments do not
change these positions. Paths never implicitly dereference pointers, search
promoted fields, or expose slice/string/interface implementation fields.
Generic paths and types are validated on concrete instantiation.

## Value contracts

| Contract | Subject | Source meaning |
| --- | --- | --- |
| `nonnull` | Pointer input/result leaf | Not nil at the boundary. |
| `range(lo, hi)` | Integer input/result leaf | Mathematical, nonempty, non-wrapping half-open interval within the source type's domain. |
| `nonnegative` | Integer input/result leaf | At least zero, using target width for `int`, `uint` and `uintptr`. |
| `align(N)` | Pointer input/result leaf | Address is a multiple of positive power-of-two N; nil is permitted. |
| `same_as(param(...))` / `same_as(receiver...)` | Integer/pointer result leaf | Same logical value as the selected input's entry snapshot. |

Integer `same_as` requires identical source types after aliases are resolved.
Pointer relations permit identity-preserving pointer-type conversions,
including `unsafe.Pointer`. The relation preserves pointer identity and access
capability; an integer address comparison alone is insufficient.

Bounds are integer literals, not target bit-pattern syntax. Full-domain ranges
need no optimizer fact. `range` and `nonnegative` combine by intersection;
an empty intersection is rejected. Different repeated values of an attribute
are rejected; identical declarations are accepted. Alignment alone implies no
allocation, initialized storage, dereferenceability or non-nullness.
N must fit the target pointer width. Alignments above the native LLVM attribute
limit use an address predicate instead.

## Behavior contracts

These constrain a complete invocation, including calls and callbacks. They are
not predicates on one SSA value and cannot be implemented by `llvm.assume`.

| Contract | Meaning |
| --- | --- |
| `memory(mode, args: mode, other: mode)` | Permitted observable-state accesses: `none`, `read`, `write`, `readwrite`. |
| `access(mode)` on pointer input leaf | Permitted reads/writes through that pointer's derived access paths. |
| `capture(none)` on pointer input leaf | No externally retained address or access capability, including temporary publication and result capture. |
| `capture(results)` | Capture only through logical normal results, including pointer fields. |
| `capture(any)` | No capture restriction. |
| `nofree` | Does not invalidate existing storage by deallocation, directly or transitively. |
| `nosync` | Performs no synchronization with other threads. |
| `nounwind` | Does not unwind out of the invocation. |
| `willreturn` | Does not diverge indefinitely; control returns to an existing caller frame normally or by unwinding. |
| `noreturn` | Never completes normally; unwinding is allowed. |
| `cold` | Invocation is expected to be uncommon; a hint only. |

A default memory mode applies to both locations; explicit locations override
it. An omitted default is `none`: `memory(args: read)` permits only
input-derived reads. Duplicate locations/defaults and unknown modes are errors.

`args` classifies an **access origin**, not a partition of memory objects.
It covers accesses derived from pointer leaves of logical entry inputs. An
access through a global remains `other` even when that global aliases an input.
A pointer loaded from pointed-to memory is a new root, not automatically part
of a recursively reachable input graph. This admits conservative native
lowering without whole-program alias analysis.

`other` covers remaining program and external observable state. I/O,
volatile/device interactions, clock or entropy observations, and events whose
result or effect can change without an intervening program write require
`other: readwrite`. Thus `memory(none)` excludes these events, and readonly
does not incorrectly permit their calls to be merged. Reading ordinary stable
program globals needs `other: read`. There is no separate `noexternal`
contract in this revision.

`access(read)` permits capture unless separately restricted.
`capture(none)` does not add Go escape analysis's `noescape` promise.
`memory(read)` does not imply termination, absence of synchronization or
unwinding. `noreturn` and `willreturn` can coexist for an always-unwinding
invocation.

Prototype spellings `readonly`/`writeonly`,
`captures(none)`/`captures(ret: address, provenance)`, `argmem`, and
single-result input `returned` normalize to `access`, `capture`, `args`,
and result `same_as`. New code should use the source vocabulary. Arbitrary LLVM
string attributes are not accepted.

## Value lowering across ABI changes

The frontend retains a typed, versioned source model: selectors, paths,
semantic kinds, typed operands and positions. LLVM indices and encodings are
separate backend data.

1. Resolve each source leaf to a logical LLVM value while Go types and target
   layout are available. Account for target wrappers/padding and multiple
   results; source field numbers need not equal LLVM field numbers.
2. Before large-aggregate and C ABI conversion, emit input facts at function
   entry and normal-result facts at known direct calls. Install native
   attributes as well when a logical leaf is a matching scalar.
3. Existing ABI conversion replaces old parameters/call results with their
   reconstructed values. The fact instructions follow those replacements.
4. Ordinary LLVM optimization consumes the facts. Remove the temporary value
   plan after materialization; retain the source contract separately.

| Source fact | Direct scalar | Packed/split value | Byval input | Sret/multiple results |
| --- | --- | --- | --- | --- |
| `nonnull` | Native attribute plus fact | Fact on reconstructed pointer | Extract pointer leaf, then assume non-null | After normal call, extract returned leaf and assume non-null |
| `range` / `nonnegative` | Native range plus integer fact | Original width/sign, not carrier high bits | Fact on extracted integer | Fact on loaded/reconstructed result leaf |
| `align` | Native alignment when representable, plus address fact | Fact on reconstructed pointer | Contained pointer, not container alignment | Returned pointer value, not sret storage |
| `same_as` | Native `returned` where valid, plus forwarding | Forward source-width input | Forward entry leaf snapshot | Forward selected existing result projections; retain bulk result transport |

Alignment facts currently require integral address-space-zero pointers;
unsupported representations are diagnosed. Current collectors do not relocate
objects or native stacks. Pointer `same_as` forwarding relies on that stable
representation; a future moving collector must forward the relocated input and
disable native `returned` when machine bits can change.

`same_as` preserves the call, its effects, exceptional behavior and all other
result components. For a scalar result, forward the input snapshot directly.
For an aggregate result, forward existing projections of the selected leaf
to that snapshot, while retaining the actual returned aggregate for bulk
transfers. The contract already guarantees that its selected leaf has the same
identity. Rebuilding the whole aggregate with `insertvalue` solely to carry a
relation could turn an 80 KB indirect copy back into a large SSA value and is
unnecessary. An equality assumption relates the actual leaf and the snapshot
as an optimization aid for later projections; it does not create or replace
pointer access provenance.

The input snapshot is never reconstructed by reloading a mutated input
variable or byval buffer after the call. Result facts never precede a possibly
panicking call. The production path uses direct LLVM calls. For `invoke`, facts
belong on a split normal edge, with successor PHIs repaired. An invoke that
also requires an unsupported signature-changing ABI conversion is diagnosed.
Unknown function-value/interface calls inherit no callee contract.

Large aggregate facts must not force whole-object loads merely to inspect a
leaf. The large-aggregate pass scalarizes extract-only loads at the original
snapshot point and keeps bulk transport as copies. Moving a scalar load to a
later user could observe an intervening write and is invalid.

## Behavior lowering and generated operations

Native effects summarize **physical** operations. Compose source restrictions
with ABI transport rather than copying them unchanged:

| Conversion | Physical adjustment |
| --- | --- |
| Byval transport | Include reads of input storage. |
| Sret transport | Include writes of result storage. |
| Pointer leaf inside aggregate | Widen source `args` into other native locations when no pointer parameter represents the root. |
| Hidden pointer `access`/`capture` | Retain source meaning and report conservative lowering; never annotate the container instead. |
| `capture(results)` via sret | Widen native capture: a store through sret is not LLVM return capture. |
| Pure finite packing/copying | Preserve control-flow properties when generated operations satisfy them. |

The backend records whether an effect has a native consumer or was lowered
conservatively, with its reason. Metadata alone does not demonstrate optimizer
consumption. This revision accepts reduced effect precision on hidden pointer
leaves. It adds no permanent adapter calls, custom effect optimizer or second
ABI reconstruction framework.

An assumption expressing an already-promised value does not make a pure source
function impure: LLVM retains the intrinsic's own control-dependence behavior.
Actual transport reads/writes and runtime effects still require composition.

GC root publication and cooperative safepoints may add memory, capture,
synchronization, allocation or unwinding behavior. In either mode, apply a
uniform conservative policy to **definitions and imported declarations**:
remove native memory/access/capture and
`nofree`/`nosync`/`nounwind`/`willreturn` restrictions; keep value facts,
`cold` and `noreturn`. A declaration must not remain stronger than its
compiled implementation.

Unexpected compiler-generated heap allocation is not a harmless ABI copy.
Implicit allocation with surviving strong behavior attributes must be diagnosed
rather than silently producing a false physical contract. Value-only contracts
remain usable. Explicit source allocation and transitive source calls remain
the annotation author's responsibility. A precise allocator/instrumentation
effect model is separate work. The same diagnostic policy covers unmodelled
compiler runtime protocols, including closure/defer state, recover frames,
local-context entry/exit and lazy package storage, shadow-stack updates and
function tracing. These operations must not leave a stronger caller declaration
than the generated implementation.

## Import, cache and runtime integration

Syntax preloading shares contracts with backend programs and imported caller
declarations. Linkname aliases merge deterministically with conflict and type
checks. Generic instances bind to their declaration and validate their concrete
signatures. Build constraints select declarations normally. Annotation-only
changes must invalidate dependent artifacts; cache hits must preserve the
same contracts and optimization behavior.

Runtime annotations live on actual declarations. Allocator non-null results,
checked-pointer results, length ranges, memory helpers and panic hints use
public machinery, without a runtime function-name whitelist. Target-mode
instrumentation policy applies uniformly.

## Implementation plan and acceptance checks

1. **Source model:** typed operands, selectors, generic/type validation,
   deterministic merging and serialization round trips. Cover invalid paths,
   target-width bounds, conflicts, aliases and source numbering.
2. **Value materialization:** logical boundary facts and forwarding before ABI
   conversion, target layouts and removal of temporary plans. Verify IR before
   and after lowering. Compare optimized branches against controls, including
   nil panic paths and unrelated carrier bits.
3. **Effect composition:** scalar native effects, hidden roots, byval/sret
   transport, capture channels and uniform instrumentation policy. Check both
   declarations and definitions, including conservative outcomes.
4. **Integration:** imports/linknames/generics, runtime migration, large
   aggregate snapshots, annotation-only cache invalidation and reuse. Run the
   CLI and runtime tests under normal GC and `nogc`.
5. **Qualification:** amd64, arm64, 386 and wasm layout/IR checks; distinguish
   object generation from execution. Record exact source head, LLVM payload
   and native environment in the delivery report.

Expected benefits include eliminating redundant tests, result comparisons and
eligible repeated calls. Work scales with annotated leaves and known calls,
not every aggregate field. Unoptimized builds may retain extract/compare/assume
instructions; optimized code should remove redundant facts. Broad performance
improvement requires measurements and is not asserted here.

Deferred vocabulary includes allocation size/family, fresh/noalias results,
dereferenceable or initialized extents, lifetimes, slice projections and
arbitrary relations. Zero-sized allocator results do not imply freshness.
Each extension needs useful consumers and a separate semantic design.
