# Go SSA escape analysis

LLGo analyzes Go storage lifetimes before LLVM lowering. One result controls locals, address-taken parameters, named results, interface boxes, closure environments, captured variables, slice/map storage and string conversion/concatenation temporaries. It replaces the separate allocation walkers from #2963, #2970 and #2971, and addresses #2975, #2974 and #2932. Static interface constant pooling remains independent and runs before dynamic box allocation. [Comparison with Go 1.27.3](escape-analysis-comparison.md) separates implemented behavior from remaining precision and target limitations.

The [unified escape-analysis proposal](https://github.com/xgo-dev/llgo/issues/3012) records the complete design, implementation status, and remaining work. The KeepAlive work originally reviewed in [#3016](https://github.com/xgo-dev/llgo/pull/3016) is consolidated into [#2997](https://github.com/xgo-dev/llgo/pull/2997), which replaces the dedicated allocation decision with this shared analysis. #3016 was closed after consolidation on 2026-11-08; #2997 is the sole active delivery PR and remains pending merge.

## Relation to the Go compiler and LLVM

`internal/goescape/graph.go` adapts the weighted location graph, worklist solver and leak encoding from Go 1.27.3's `cmd/compile/internal/escape/{graph,solve,leaks}.go`, with Go's BSD attribution. The SSA adapter, target policy and call-graph integration are LLGo-specific. Importing cmd/compile's implementation directly would require its internal IR, types and export format; LLGo uses go/types and x/tools/go/ssa instead.

The Go compiler reference also includes `call.go` (parameter/result effects and go/defer lifetimes), `expr.go` (interface storage versus payload), and `walk/convert.go` (allocation placement). The issue reproducers' `go build -gcflags=-m=2` output confirms borrowed parameter addresses and nonescaping callback boxes. Matching Go's diagnostic text or every optimization is not a goal.

This operates before inlining and LLVM optimization. The allocator contracts and Attributor merged in #2985 remain complementary for allocations exposed after lowering/inlining, including #2983's SIMD buffers. This implementation neither depends on nor incorporates #2247's separate LLVM-IR escape-analysis draft.

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
| Arrays/slices | Track local backing storage, views and copied elements. Nonescaping constant backing arrays use stack storage. Dynamic capacities try a fixed 32-byte caller buffer and retain the original heap fallback; pointer elements keep heap lifetime because that fallback can be taken. Synthetic constant-capacity makes retain their runtime length checks with a bounded caller array. |
| String conversions/concatenation | Model copied character storage independently from its source. Nonescaping results use caller buffers (32 bytes or 32 rune slots), with original heap helpers for larger results. Read-only `[]byte(string)` and immediately compared/queried `string([]byte)` can borrow data after a second solve propagates the actual alias lifetime. Mutable or escaping results preserve copies. |
| append/copy | Borrow source backing storage; copied pointer elements may escape into a destination or newly allocated array. |
| Maps/channels/select | Retain inserted keys/values and sent values. Native nonescaping maps use a caller-owned header and typed initial bucket; growth, overflow and oversized/over-aligned buckets retain runtime allocation. Channels retain runtime storage. Explicit-root targets retain heap maps. |
| defer/go | Non-loop defer arguments persist to invocation exit. Loop defers, goroutines and foreign-owner range-function defer stacks require longer lifetimes. Eventual callee retention is modeled separately. |
| panic/recover | Panic payloads escape. Existing recover-visible SSA storage remains heap allocated; new defer records preserve SJLJ ownership. |
| unsafe/native/indirect calls | Unknown publication is conservative. Pointer-to-uintptr conversion cannot hide an address from the analysis. Existing backend uintptr liveness handling remains in force. |
| Unknown SSA operation | Retain all allocation sites and operands instead of accepting an unrecognized operation as harmless. |

Bodies replaced by source patches, linknames, native declarations or intrinsics are not trusted as the called implementation. The adapter resolves declarations through the same `funcName`/`llgoInstrs` identifiers as lowering. Atomic instruction contracts borrow their address, return loaded contents at the proper depth, and retain any pointer being published by a store, exchange or compare-and-exchange. `llgo.skip` discards its operands. Bodyless `go:noescape` declarations promise no retention, but can mutate pointees and invoke callbacks, following cmd/compile's treatment. Unknown or malformed contracts publish their arguments.

Calls through proven interface values and bounded phis of known function values compose the actual targets' summaries. Unknown alternatives keep conservative publication. Invoking a known callback composes its capture effects, including callback results that an unknown invoker could retain. Only invocation effects, rather than fabricated heap publication, are exported for direct calls of function parameters. All candidate callees also participate in call-graph discovery and SCC ordering.

The analysis is deliberately field-insensitive, like cmd/compile. Stores through unresolved aliases retain their value; unresolved interface dispatch and most native intrinsics remain conservative. Original bodies in patched packages are also conservative unless their actual replacement is analyzed. These are precision limitations, not alternate allocation heuristics.

## Target placement and GC

`cl/escape.go` combines lifetime proofs with LLGo layout. Queries use a disposable backend context so original types cannot pollute the named-type cache before source replacements are lowered (including shared/archive builds). Existing explicit locals retain the 128 KiB stack-object limit; newly promoted allocations and implicit boxes/environments use 64 KiB. Backend allocation helpers retain alignment, initialization and zero-sized identity rules. Fixed stack reservation is hoisted, while initialization stays at the original statement.

Forced heap placement feeds back into the graph **before** summaries are finalized. Otherwise a large or GC-constrained container could retain a pointer to an object incorrectly moved to the stack. The diagnostic records the constraint, and retention propagates to its contents.

Newly promoted objects and temporary buffers share a 64 KiB budget per function. Target layouts account for slice capacity, rune buffers, map headers/buckets, and alignment; borrowed views consume no storage. Selection follows SSA order and over-budget heap constraints are solved before final parameter effects. Receiver adapters and defer frames/records reserve remaining capacity during lowering. Defer registration records ownership for cleanup. Existing SSA locals, later LLVM promotion, ABI spills, and call-chain depth are outside this incremental budget.

`llgo.keepAlive` has an explicit borrowing contract: it does not retain the argument beyond the call, mutate its contents or invoke a callback. Keeping references live until the call remains its purpose. Its interface boxes use these same lifetime and budget decisions, including deferred, loop and goroutine ownership. Omitting an unobservable static type descriptor remains a separate lowering decision; it does not authorize stack placement. On explicit-root targets, boxes used only by KeepAlive may use stack storage because the intrinsic cannot mutate them and their original SSA values already publish the pointer fields through return. This narrow GC exception does not bypass escape or budget checks; ordinary borrowing calls retain the conservative stack-field policy.

Temporary buffers are fixed entry allocations. Runtime helpers initialize only the storage they use after ordinary bounds checks; copies initialize all observable bytes. Large fallbacks avoid clearing an unused stack buffer. The compiler and runtime share their size contract in `runtime/abi/stack.go`. A synchronous native interface invocation also uses a caller-owned receiver word instead of `IfacePtrData`'s per-call allocation; receiver words for deferred/asynchronous invocations and explicit-root targets keep the longer-lived heap path. Receiver wrappers and nil-interface checks are preserved.

Native conservative GC can scan stack fields. Explicit-root targets currently cannot keep managed fields alive merely by rooting their stack address, so ordinary newly promoted pointer-bearing objects remain in scanned heap storage. The immutable KeepAlive-only box described above is a limited exception. Existing SSA-local allocations retain their root plan: their addresses never leave the function and loads publish individual pointer values. A newly promoted object can instead pass its address to a borrowing callee whose writes are not covered by that plan. Extending promotion on those targets requires field roots with correct mutation and safepoint lifetimes. Pointer-free objects can still use the stack.

`ssa/eh.go` owns compiler-generated defer storage. Native frames live until the invocation unlinks them. Each non-loop argument record up to 64 KiB uses a dedicated entry slot, with arguments captured at registration. Loop/foreign-owner/oversized records and explicit-root frames/records stay on the heap. Pop code uses the same ownership policy: stack nodes never reach `FreeDeferNode`. Mixed lists retain LIFO order, panic/recover behavior and Goexit handling.

## Packages, caches and diagnostics

The build coordinator constructs dependency SSA before package-cache lookup (`registerSSAPkgs`, `buildSSAPkgs`). It derives summaries from those exact bodies on every invocation, including archive cache hits, then shares immutable decisions with isolated backend workers. No new serialized summary format is needed. A future loader that skips dependency SSA will need versioned exported effects before reusing these proofs.

Package fingerprints include `GO_ESCAPE_VERSION` as well as existing compiler/source/dependency/target inputs. This also invalidates old archives for compiler API callers that omit a development compiler hash. Missing analysis facts never promote an existing heap object. Compiled bodies excluded from Go summaries (for example, C ABI functions) preserve existing SSA-local allocations, avoiding new runtime dependencies in freestanding code. Direct API calls without inspectable dependency bodies use conservative call effects.

Use Go-compatible compiler flags for allocation diagnostics:

```sh
llgo build -gcflags=-m .
llgo build -gcflags=-m=2 .
llgo build -gcflags=all=-m=2 .
llgo tool compile -m=2 file.go
```

`-m` reports frontend storage plans; `-m=2` adds reasons such as retention, iteration lifetime, analysis limits, stack budget, or GC restrictions. Output distinguishes `stack`, `stack-buffer` with heap fallback, `borrowed` views, and `heap`; it does not claim final machine allocation counts. Unpatterned flags select command-line packages, while `all=` includes dependencies, using Go's count and last-matching-list rules. Other package patterns retain the CLI's existing rejection. LLGo consumes diagnostic flags before host metadata loading. Reports remain sorted and available on cache hits, without changing generated code or LLGo cache identity. Go's full inlining diagnostics and exact message wording are not implemented.

## Validation and measurements

- `internal/goescape`: LLVM-independent graph, scope and SSA tests; recursive/result effects, box/payload separation, aliases, forced storage, malformed contracts and work-limit fallback. Run with `go test -race ./internal/goescape`.
- `cl`: existing named-result, append and closure regressions now use the unified analysis. Interface tests cover native 64-bit, Windows 386 and Wasm layouts; ordinary explicit-root cases retain heap storage. KeepAlive tests cover its borrowing contract, immutable-box exception, and shared object/helper budgets. Atomic tests verify actual lowered instructions with deliberately misleading Go stub bodies.
- `test/go/keep_alive_stack_test.go`: common Go/LLGo tests cover reachable stack fields, boxing before GC, deferred iteration snapshots, panic-unwind GC and callable function values. Focused tests pass natively and on actual Wasm J32/J64/W32 runtimes. The aggregate benchmark covers the allocation-free path; its measurements are separate from the earlier full escape benchmark batch.
- `internal/build/escape_test.go`: cold/warm archives and 1/8 workers in fresh processes, with allocation checks and diagnostic proofs for cross-package borrows and callback boxes. A caller rebuild reuses the analyzed dependency's archive.
- `test/escape`: forced GC and finalizers, stack-held fields, boxed closures, escaped aliases, loop identity, atomic publication, goroutines, recovery and mixed defer/range-function ownership. Run natively, with `-O3 -lto=full`, and with `-target wasi|emscripten|emscripten-memory64 -emulator`. Multiworker Emscripten uses `LLGO_WASM_WORKERS=2`.
- [benchmark/escape](../benchmark/escape/README.md): reproducible issue probes and measured allocation, timing and build-size comparisons. Allocation bytes are reported separately from timing; native `MemStats.Mallocs` is not used because of #2923.

The broad native unwind suite has two full-LTO source-line expectation failures (`TestDeferredCallerClosingLine`, `TestRuntimeDeferredPanicLine`) that reproduce unchanged on the baseline. The new lifetime tests pass under full LTO; the existing line failures must not be represented as a clean full-LTO suite.

The broad local build-system suite also reproduces two baseline environment failures on macOS: external archive member naming, and the Go 1.21-to-1.27 toolchain fixture with `GOSUMDB=off`. The SIMD alignment expectation was updated to check the actual 16-byte stack storage on native targets and aligned heap storage on Wasm.

The analysis caches each function's CFG scopes and retains allocation decisions from the converged summary pass. Only the bounded non-convergence fallback solves again after replacing summaries with conservative effects. Closure lifetimes use call/free-variable summaries; the graph contains one function and no unused cross-function shortcuts.

## Bounded precision

Known function and concrete interface arguments can specialize a completed callee's effects at a call site. The generic summary and emitted callee storage remain authoritative: a specialized caller cannot turn storage in that body from heap into stack. Recursive or unknown contexts and work limits fall back conservatively. No shared Go SSA is rewritten.

An immutable lexical capture may be stored by value in its environment when its small variable has one dominating initialization and no observable address or later assignment. The analysis publishes the same capture plan that environment construction and free-variable lowering use.

Temporary `string([]byte)` views may span multiple consumers and branches. All paths to the final consumer are checked for writes or unknown effects. Inspectable helpers must be read-only throughout their bodies, including global accesses; a parameter-only mutation summary cannot rule out a global alias of the source. Cyclic or over-budget regions preserve the copy.
