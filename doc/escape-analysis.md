# Go SSA escape analysis

LLGo analyzes Go storage lifetimes before LLVM lowering. One result controls locals, address-taken parameters, named results, interface boxes, closure environments and captured variables. It replaces the separate allocation walkers from #2963, #2970 and #2971, and addresses #2975, #2974 and #2932. Static interface constant pooling remains independent and runs before dynamic box allocation.

## Relation to the Go compiler and LLVM

`internal/goescape/graph.go` adapts the weighted location graph, worklist solver and leak encoding from Go 1.27.3's `cmd/compile/internal/escape/{graph,solve,leaks}.go`, with Go's BSD attribution. The SSA adapter, target policy and call-graph integration are LLGo-specific. Importing cmd/compile's implementation directly would require its internal IR, types and export format; LLGo uses go/types and x/tools/go/ssa instead.

The Go compiler reference also includes `call.go` (parameter/result effects and go/defer lifetimes), `expr.go` (interface storage versus payload), and `walk/convert.go` (allocation placement). The issue reproducers' `go build -gcflags=-m=2` output confirms borrowed parameter addresses and nonescaping callback boxes. Matching Go's diagnostic text or every optimization is not a goal.

This operates before inlining and LLVM optimization. PR #2985's allocator contracts and Attributor remain complementary for allocations exposed after lowering/inlining, including #2983's SIMD buffers. This implementation neither depends on nor incorporates #2247's separate LLVM-IR escape-analysis draft.

```text
patched sources + resolved declarations
                 |
           complete Go SSA
                 |
 internal/goescape: graph + call SCC summaries
                 |
  immutable, target-aware storage decisions
                 |
   cl placement + SSA defer ownership lowering
                 |
       LLVM optimization / inlining / LTO
```

## Graph and lifetimes

A location represents either storage or a value-flow node. Keeping these distinct lets an interface box stay on the stack while its extracted payload flows to a caller. Edges carry indirection depth: address-taking is -1, copying is 0 and dereferencing is +1. The solver clamps negative depths before following stored contents, so address cycles cannot accumulate invalid negative distances.

Each receiver, ordinary parameter and free variable has effects for heap retention, each result, mutation and indirect invocation. Static calls instantiate result effects at their uses: returning a borrowed pointer from a helper does not unconditionally escape the caller's object. Function strongly connected components reach a monotone fixed point before their callers are analyzed. Generic instances and method wrappers use their actual SSA bodies.

Invocation and nested iteration scopes describe storage lifetime. CFG strongly connected components include backward gotos; removing a dominating loop header exposes nested loops. A loop-header phi outlives either iteration, preserving distinct simultaneously observable objects. Irreducible regions retain heap storage. Analysis never mutates SSA `Heap` flags.

The solver limits work to one million location/edge visits per function solve. A partial solve discards all placement and parameter proofs for that function. Recursive call groups have a 64-iteration limit; exceeding it publishes all arguments conservatively. Loop discovery has a separate work limit. These bounds can lose optimizations but cannot turn incomplete information into a stack-allocation proof.

## Operations and resolved contracts

| Construct | Behavior |
| --- | --- |
| Local/parameter/result addresses | Taking an address is not itself an escape; follow stores, returned aliases and call effects. |
| Interfaces | Track indirect boxes separately. Concrete assertions copy the payload; interface-to-interface operations preserve box identity. Use the backend's actual interface representation. |
| Closures | Model each environment and its captured pointers. Compose free-variable effects for statically known calls; unknown callees conservatively retain reachable captured pointers. |
| Arrays/slices | Track local backing storage, views and copied elements. Dynamic backing storage retains the existing heap path. |
| append/copy | Borrow source backing storage; copied pointer elements may escape into a destination or newly allocated array. |
| Maps/channels/select | Retain inserted keys/values and sent values. Container allocation remains conservative. |
| defer/go | Non-loop defer arguments persist to invocation exit. Loop defers, goroutines and foreign-owner range-function defer stacks require longer lifetimes. Eventual callee retention is modeled separately. |
| panic/recover | Panic payloads escape. Existing recover-visible SSA storage remains heap allocated; new defer records preserve SJLJ ownership. |
| unsafe/native/indirect calls | Unknown publication is conservative. Pointer-to-uintptr conversion cannot hide an address from the analysis. Existing backend uintptr liveness handling remains in force. |
| Unknown SSA operation | Retain all allocation sites and operands instead of accepting an unrecognized operation as harmless. |

Bodies replaced by source patches, linknames, native declarations or intrinsics are not trusted as the called implementation. The adapter resolves declarations through the same `funcName`/`llgoInstrs` identifiers as lowering. Atomic instruction contracts borrow their address, return loaded contents at the proper depth, and retain any pointer being published by a store, exchange or compare-and-exchange. `llgo.skip` discards its operands. Bodyless `go:noescape` declarations promise no retention, but can mutate pointees and invoke callbacks, following cmd/compile's treatment. Unknown or malformed contracts publish their arguments.

The analysis is deliberately field-insensitive. Stores through unresolved aliases retain their value; interface dispatch and most native intrinsics remain conservative. Original bodies in patched packages are also conservative unless their actual replacement is analyzed. These are precision limitations, not alternate allocation heuristics.

## Target placement and GC

`cl/escape.go` combines lifetime proofs with LLGo layout. Queries use a disposable backend context so original types cannot pollute the named-type cache before source replacements are lowered (including shared/archive builds). Existing explicit locals retain the 128 KiB stack-object limit; newly promoted allocations and implicit boxes/environments use 64 KiB. Backend allocation helpers retain alignment, initialization and zero-sized identity rules. Fixed stack reservation is hoisted, while initialization stays at the original statement.

Forced heap placement feeds back into the graph **before** summaries are finalized. Otherwise a large or GC-constrained container could retain a pointer to an object incorrectly moved to the stack. The diagnostic records the constraint, and retention propagates to its contents.

Native conservative GC can scan stack fields. Explicit-root targets currently cannot keep managed fields alive merely by rooting their stack address, so newly promoted pointer-bearing objects remain in scanned heap storage. Existing SSA-local allocations retain their root plan: their addresses never leave the function and loads publish individual pointer values. A newly promoted object can instead pass its address to a borrowing callee whose writes are not covered by that plan. Extending promotion on those targets requires field roots with correct mutation and safepoint lifetimes. Pointer-free objects can still use the stack.

`ssa/eh.go` owns compiler-generated defer storage. Native frames live until the invocation unlinks them. Each non-loop argument record up to 64 KiB uses a dedicated entry slot, with arguments captured at registration. Loop/foreign-owner/oversized records and explicit-root frames/records stay on the heap. Pop code uses the same ownership policy: stack nodes never reach `FreeDeferNode`. Mixed lists retain LIFO order, panic/recover behavior and Goexit handling.

## Packages, caches and diagnostics

The build coordinator constructs dependency SSA before package-cache lookup (`registerSSAPkgs`, `buildSSAPkgs`). It derives summaries from those exact bodies on every invocation, including archive cache hits, then shares immutable decisions with isolated backend workers. No new serialized summary format is needed. A future loader that skips dependency SSA will need versioned exported effects before reusing these proofs.

Package fingerprints include `GO_ESCAPE_VERSION` as well as existing compiler/source/dependency/target inputs. This also invalidates old archives for compiler API callers that omit a development compiler hash. Missing analysis facts never promote an existing heap object. Compiled bodies excluded from Go summaries (for example, C ABI functions) preserve existing SSA-local allocations, avoiding new runtime dependencies in freestanding code. Direct API calls without inspectable dependency bodies use conservative call effects.

For placement diagnostics, use a function/package-name prefix, or `1` for all functions:

```sh
LLGO_ESCAPE=example.com/app llgo build .
LLGO_ESCAPE=1 llgo build .
```

Output includes source position when available, function, allocation kind, placement and a reason such as returned address, call retention, cross-iteration lifetime, work limit or GC restriction. Reports are sorted and generated even on cache hits. They do not affect code or cache identity.

## Validation and measurements

- `internal/goescape`: LLVM-independent graph, scope and SSA tests; recursive/result effects, box/payload separation, aliases, forced storage, malformed contracts and work-limit fallback. Run with `go test -race ./internal/goescape`.
- `cl`: existing named-result, append and closure regressions now use the unified analysis. Interface tests cover native 64-bit, Windows 386 and Wasm layouts; explicit-root tests retain heap storage. Atomic tests verify actual lowered instructions with deliberately misleading Go stub bodies.
- `internal/build/escape_test.go`: cold/warm archives and 1/8 workers in fresh processes, with allocation checks and diagnostic proofs for cross-package borrows and callback boxes. A caller rebuild reuses the analyzed dependency's archive.
- `test/escape`: forced GC and finalizers, stack-held fields, boxed closures, escaped aliases, loop identity, atomic publication, goroutines, recovery and mixed defer/range-function ownership. Run natively, with `-O3 -lto=full`, and with `-target wasi|emscripten|emscripten-memory64 -emulator`. Multiworker Emscripten uses `LLGO_WASM_WORKERS=2`.
- [benchmark/escape](../benchmark/escape/README.md): reproducible issue probes and measured allocation, timing and build-size comparisons. Allocation bytes are reported separately from timing; native `MemStats.Mallocs` is not used because of #2923.

The broad native unwind suite has two full-LTO source-line expectation failures (`TestDeferredCallerClosingLine`, `TestRuntimeDeferredPanicLine`) that reproduce unchanged on the baseline. The new lifetime tests pass under full LTO; the existing line failures must not be represented as a clean full-LTO suite.

The broad local build-system suite also reproduces two baseline environment failures on macOS: external archive member naming, and the Go 1.21-to-1.27 toolchain fixture with `GOSUMDB=off`. The SIMD alignment expectation was updated to check the actual 16-byte stack storage on native targets and aligned heap storage on Wasm.

The analysis caches each function's CFG scopes and retains allocation decisions from the converged summary pass. Only the bounded non-convergence fallback solves again after replacing summaries with conservative effects. Closure lifetimes use call/free-variable summaries; the graph contains one function and no unused cross-function shortcuts.
