# Go SSA escape analysis

LLGo analyzes Go storage lifetimes before LLVM lowering. One result controls locals, address-taken parameters, named results, interface boxes, closure environments, captured variables, slice/map storage and string conversion/concatenation temporaries. It replaces the separate allocation walkers from #2963, #2970 and #2971, and addresses #2975, #2974 and #2932. Static interface constant pooling remains independent and runs before dynamic box allocation. [Comparison with Go 1.27.3](escape-analysis-comparison.md) separates implemented behavior from remaining precision and target limitations.

The [unified escape-analysis proposal](https://github.com/xgo-dev/llgo/issues/3012) records the complete design, implementation status and remaining work.

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
| String conversions/concatenation | Model copied character storage independently from its source. Nonescaping results use caller buffers (32 bytes or 32 rune slots), with original heap helpers for larger results. Read-only `[]byte(string)` and proven temporary `string([]byte)` read regions can borrow data after a second solve propagates the actual alias lifetime. Mutable or escaping results preserve copies. |
| append/copy | Borrow source backing storage; copied pointer elements may escape into a destination or newly allocated array. |
| Maps/channels/select | Retain inserted keys/values and sent values. Nonescaping maps use a caller-owned header and typed initial bucket; growth, overflow and oversized/over-aligned buckets retain runtime allocation. Channels retain runtime storage. Explicit-root targets register managed fields with typed stack roots. |
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

Newly promoted objects and temporary buffers share a 64 KiB budget per function. Target layouts account for slice capacity, rune buffers, map headers/buckets, and alignment; borrowed views consume no storage. Selection follows SSA order and over-budget heap constraints are solved before final parameter effects. Receiver adapters and defer frames/records reserve remaining capacity during lowering. Defer registration records ownership for cleanup. The original bytes of SSA locals, later LLVM promotion, ABI spills, and call-chain depth are outside this incremental budget. Newly required field-root slots for existing locals are included; an over-budget local falls back to the heap before summaries are finalized.

`llgo.keepAlive` has an explicit borrowing contract: it does not retain the argument beyond the call, mutate its contents or invoke a callback. Keeping references live until the call remains its purpose. Its interface boxes use these same lifetime and budget decisions, including deferred, loop and goroutine ownership. Omitting an unobservable static type descriptor remains a separate lowering decision; it does not authorize stack placement. On explicit-root targets, managed stack fields use the same typed root registration as ordinary objects; KeepAlive needs no GC-specific placement exception.

Temporary buffers are fixed entry allocations. Runtime helpers initialize only the storage they use after ordinary bounds checks; copies initialize all observable bytes. Large fallbacks avoid clearing an unused stack buffer. The compiler and runtime share their size contract in `runtime/abi/stack.go`. A synchronous interface invocation also uses a caller-owned receiver word instead of `IfacePtrData`'s per-call allocation; receiver words for deferred/asynchronous invocations keep the longer-lived heap path. Receiver wrappers and nil-interface checks are preserved.

Generic map lookup, update and delete helpers borrow their key address synchronously. Their lowering-generated key copies use fixed entry slots, including inside loops, and share the 64 KiB budget with other promoted storage. Explicit-root targets charge and register the key's managed fields. Map insertion copies both direct and indirect keys into map-owned storage; the temporary address is never retained. Oversized or over-budget copies keep the heap fallback, and generic keys retain the larger physical representation required by the runtime.

Go's opaque map/string range iterators also use invocation-owned entry slots, with the same budget and typed field roots. Buffer constructors reset all fields each time a range restarts, including for nil or empty inputs; nested ranges have separate slots. General iterator-construction APIs retain heap ownership, and over-budget source ranges use those same constructors.

Native conservative GC can scan stack fields. Explicit-root targets register the physical addresses of managed pointer fields, including interface data and closure environment words, arrays and nested structures. Collection reads each field's current value, so writes in borrowing callees are covered. Offsets come from LLVM storage layouts, and metadata records the physical pointer width: Wasm32 C fields must not be read using a wider Go word.

One root frame is finalized after storage lowering, before returns are completed. Static allocas remain in LLVM's entry block. Conditional and loop objects are initialized before frame publication, and declaration-time initialization still runs at its original statement. The existing head/replay guards preserve the single frame and its fields through Asyncify/SjLj replay, context switches, defer cleanup and exceptional exits. Frames containing field roots retain inline initialization; ordinary value-root frames can still use the size-oriented outlined helpers.

`ssa/eh.go` owns compiler-generated defer storage. Native frames live until the invocation unlinks them. Each non-loop argument record up to 64 KiB uses a dedicated entry slot, with arguments captured at registration. Eligible explicit-root frames and records register their pointer fields too. Loop/foreign-owner/oversized records stay on the heap. Pop code uses the same ownership policy: stack nodes never reach `FreeDeferNode`. Mixed lists retain LIFO order, panic/recover behavior and Goexit handling.

## Packages, caches and diagnostics

With `LLGO_ESCAPE_CACHE=1` and the ordinary build cache enabled, the coordinator first loads checked type exports and frontend contracts for eligible dependencies. A contract is usable only with its exact validated archive. Opaque dependencies register declarations and link snapshots without rebuilding or lowering their bodies. Initial packages, cache misses, ambiguous inventories and required optimization bodies use source SSA. Functions with inline candidates and generic declarations keep their source type information; the coordinator imports eligible inline bodies before backend workers start. Current-invocation generic instances remain source facts rather than being certified by the defining library's source key.

The persistent call-graph SCC records contain generic parameter effects, every object's placement, stack reservations and capture representations. An SCC (strongly connected component) is a mutually reachable group of functions; recursive groups are validated together. A hit restores complete facts onto source-built SSA and skips scope construction/solving for that component. An opaque dependency instead supplies generic effects owned by its matching archive; stable component keys preserve its callers' cache identities. Caller/recover facts and LLVM-free ABI, export and PC-line snapshots retain the behavior previously obtained by lowering dependencies again.

Keys cover analysis/schema versions, compiler and source toolchain, language version, target/layout/feature policies, source transforms and overlays, and complete dependency source closures, including versioned modules. Sorted closure identities handle cycles introduced by runtime replacements. Full SCC records additionally validate queried type properties, object inventories and dependency keys; ambiguous same-name wrappers disable affected reuse. Checksummed size-bounded files are published atomically. Missing, corrupt, incompatible or incomplete records fall back to source/checking/analysis. If an export-only package later needs a body, loading restarts from source before publishing backend work so existing Go type identities are never replaced in place.

Wasm reflection bridges, function-entry metadata, runtime-free startup and global-initializer removal are **whole-program** decisions. Their separate export is keyed by all package inputs, roots and the actual SSA builder mode. Initializer removals rebind and validate exact instruction positions/text in source-built packages; opaque packages use the plan already included in their archives. A changed program context recomputes the ordinary complete-source analyses. These decisions are never treated as generic package effects.

Persistence remains opt-in. Type exports currently cover ordinary dependency packages without generic or inline-body requirements; syntax is still parsed for compiler directives. Runtime patches and test variants keep independently checked scopes. Libraries, generation, deadcode-drop, module-inspection hooks and dependency-wide diagnostic paths retain source loading. Cache validation can exceed solving costs for small programs, so hit counts alone do not establish a speedup. `certify frontend exports` trace events report actual `skipped_ssa` and `type_exports` counts.

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
- `cl`: existing named-result, append and closure regressions now use the unified analysis. Interface tests cover native 64-bit, Windows 386 and Wasm layouts, including typed stack roots. KeepAlive tests cover its borrowing contract and shared object/helper budgets. Atomic tests verify actual lowered instructions with deliberately misleading Go stub bodies.
- `test/go/keep_alive_stack_test.go`: common Go/LLGo tests cover reachable stack fields, boxing before GC, deferred iteration snapshots, panic-unwind GC and callable function values. Focused tests pass natively and on actual Wasm J32/J64/W32 runtimes. The aggregate benchmark covers the allocation-free path; its measurements are separate from the earlier full escape benchmark batch.
- `internal/build/escape_test.go`: cold/warm archives and 1/8 workers in fresh processes, with allocation checks and diagnostic proofs for cross-package borrows and callback boxes. A caller rebuild reuses the analyzed dependency's archive.
- `test/escape`: forced GC and finalizers, stack-held fields, boxed closures, escaped aliases, loop identity, atomic publication, goroutines, recovery and mixed defer/range-function ownership. Run natively, with `-O3 -lto=full`, and with `-target wasi|emscripten|emscripten-memory64 -emulator`. Multiworker Emscripten uses `LLGO_WASM_WORKERS=2`.
- [benchmark/escape](../benchmark/escape/README.md): reproducible issue probes and measured allocation, timing and build-size comparisons. Report allocation events, bytes and timing separately. Native BDWGC `MemStats.Mallocs` uses the existing allocator event counters; historical measurements predating the #2923 fix did not use it.

The broad native unwind suite has two full-LTO source-line expectation failures (`TestDeferredCallerClosingLine`, `TestRuntimeDeferredPanicLine`) that reproduce unchanged on the baseline. The new lifetime tests pass under full LTO; the existing line failures must not be represented as a clean full-LTO suite.

The broad local build-system suite also reproduces two baseline environment failures on macOS: external archive member naming, and the Go 1.21-to-1.27 toolchain fixture with `GOSUMDB=off`. The SIMD alignment expectation checks the actual 16-byte stack storage on native and Wasm targets.

Within a solve, the analysis memoizes each function's CFG scopes and retains allocation decisions from the converged summary pass. Persistent hits do not construct those scopes unless a later contextual solve needs them. Only the bounded non-convergence fallback solves again after replacing summaries with conservative effects. Closure lifetimes use call/free-variable summaries; the graph contains one function and no unused cross-function shortcuts.

## Bounded precision

Known function and concrete interface arguments can specialize a completed callee's effects at a call site. The generic summary and emitted callee storage remain authoritative: a specialized caller cannot turn storage in that body from heap into stack. Recursive or unknown contexts and work limits fall back conservatively. No shared Go SSA is rewritten.

An immutable lexical capture may be stored by value in its environment when its small variable has one dominating initialization and no observable address or later assignment. The analysis publishes the same capture plan that environment construction and free-variable lowering use.

Temporary `string([]byte)` views may span multiple consumers and branches. All paths to the final consumer are checked for writes or unknown effects. Inspectable helpers must be read-only throughout their bodies, including global accesses; a parameter-only mutation summary cannot rule out a global alias of the source. Cyclic or over-budget regions preserve the copy.

## Frontend cache regression probes

`TestEscapeCrossPackageCacheAndWorkers` uses fresh processes and 1/8 workers to check cold/hot builds, caller changes, corrupt frontend/type records, missing archives, repair, stack-placement diagnostics and runtime results. `TestFrontendExportsMethodsCallerRecoverAndGenericBodies` also runs the fixture with Go, checks imported private/recursive/alias types and constants, and preserves methods, reflection, caller tracking, recover, generic bodies and cross-package inlining. Type-graph and initializer-plan tests reject incompatible records.

Actual hosted Wasm probes use the same fixture:

```sh
LLGO_ESCAPE_CACHE=1 python3 test/wasm/frontendcache/test.py --llgo /path/to/llgo --output /tmp/frontend-cache-results
```

The probe executes J32/J64 under Node and W32 under Wasmer, verifies dependency SSA omission and required inline-body reconstruction, and retains traces and logs. Its seed/reuse worker counts differ to test cache portability across worker counts; those timings are not a controlled speed comparison.
