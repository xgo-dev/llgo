# Go-layer escape analysis comparison

Reference: Go **1.27.3**, the installed toolchain used for the comparison. This is a comparison of source-level storage decisions and emitted/runtime allocations, not a claim that LLGo implements every cmd/compile optimization. LLVM heap-to-stack is a subsequent, independent optimization.

The [unified escape-analysis proposal](https://github.com/xgo-dev/llgo/issues/3012) describes the complete design and status.

## Implemented coverage

| Area | Go 1.27.3 | LLGo |
| --- | --- | --- |
| Core model | Weighted location graph, parameter/result leaks, recursive batches | Adapted graph and leak encoding, SSA lifetime scopes and call SCC fixed point |
| Local/parameter/result addresses | Taking an address alone does not require heap storage | Same principle; target layout and GC constraints feed back before final summaries |
| Cross-package calls | Exported parameter effects | Effects derived from the exact dependency SSA, including archive-cache hits; immutable worker input |
| Interface boxes/closures | Separate implicit storage, capture and call effects | Box/payload separation, closure environments, captures and concrete interface-call effects |
| Known function/interface values | Frontend devirtualization and inlining improve escape precision | Bounded direct/phi target resolution for the analysis; #3003 supplies frontend cross-package body import for LLVM inlining |
| Callback invocation | Callee effects; additional precision after inlining | Known callback capture effects, including retained callback results; unknown callees remain conservative |
| Fixed slice backing arrays | Stack when local and within size/alignment limits | Stack with the same lifetime proof, including constant-capacity makes with runtime length checks |
| Dynamic slice capacity | Small fixed caller buffer with heap fallback | 32-byte caller buffer with heap fallback; pointer elements retain heap lifetime |
| String/byte/rune conversion and concatenation | Small caller buffers, heap fallback, copy-elision rules | Explicit implicit-storage nodes, small caller buffers and original large-result helpers |
| Temporary byte/string views | No-copy conversion for supported read-only/temporary uses | Read-only `[]byte(string)`, and single-use `string([]byte)` immediately consumed by comparison, length or map lookup; solve again with the actual source alias |
| Local maps | Caller-owned header and small initial backing storage | Native caller-owned header and typed initial bucket; existing growth/overflow allocator |
| Channels | Runtime allocation | Runtime allocation; this is not an outstanding heap-to-stack optimization in cmd/compile |
| defer/go | Invocation and asynchronous lifetimes differ | Non-loop native defer storage is invocation-local; loop/foreign/asynchronous ownership is retained |
| Safety fallbacks | Unknown calls and unsupported proofs are conservative | Unknown calls/SSA, irreducible CFG and solver-budget fallback retain heap storage |

## Remaining differences

1. **Using inlined call-site context in escape analysis.** Merged PR [#3003](https://github.com/xgo-dev/llgo/pull/3003) implements frontend cross-package Go SSA body import and LLVM inlining; that capability is covered there. At its merged revision `d510ec8b5`, imported functions are lowered as `available_externally` LLVM definitions without cloning/inlining their Go SSA into the caller or rerunning this analysis on an inlined Go graph. For example, `apply(fn func(*int), p *int) { fn(p) }` still exports conservative publication for `p`, even when a caller supplies a known borrowing function. Go can inline the helper before escape analysis and connect those particular arguments. The remaining gap is this context-sensitive proof, not cross-package inlining itself. LLVM inlining and heap-to-stack may still remove the resulting allocation later, so this is not evidence of an unavoidable runtime allocation.
2. **Explicit GC-root targets.** Newly promoted pointer-bearing objects and map storage stay in scanned heap memory until the collector can scan typed stack fields for the full lifetime, including writes by borrowing callees. Pointer-free character/slice buffers can use the stack. Removing this restriction requires root-registration/lifetime work, not a weaker escape predicate.
3. **Additional copy elision.** Read-only `[]byte(string)` and immediately consumed temporary `string([]byte)` are covered. LLGo's temporary-string proof requires one use in the same block, with no intervening instruction other than debug information. More general use patterns, or cases exposed only after inlining (for example a string comparison hidden inside a helper), can retain a copy. Source mutation between conversion and use must preserve the original copied string.
4. **Receiver-word and defer lowering.** Native synchronous interface receiver words use caller storage. Deferred/asynchronous invocations and explicit-root targets retain the original allocation path. Extending those cases requires storage owned by the defer/goroutine or a collector-aware frame.
5. **Runtime ABI temporaries.** Eliminating a source conversion does not eliminate every allocation inside a runtime helper. The string-map lookup probe still allocates approximately 32 bytes/call after removing its 128-byte string copy; cmd/compile allocates no bytes for this probe. This needs separate runtime/lowering lifetime work.
6. **Target heuristics and build cost.** Stack size/alignment limits, native fixed stacks, vector layouts and conservative versus explicit GC roots differ. Cross-package effects are recomputed from available SSA instead of using Go's serialized escape tags. This affects optimization opportunities and build cost, independently of correctness.
7. **By-value lexical closure capture.** Go's escape pass can copy a small, unmodified variable with no observable address use directly into a closure environment. LLGo currently retains the Go SSA lexical binding representation, which uses pointers to captured variables, except for its existing zero-sized environment elimination. An escaping closure can therefore retain separate captured-variable storage that Go avoids. General by-value capture requires coordinated capture proofs and environment layout changes; LLVM may still remove some resulting storage later.

Field-insensitivity, unknown indirect calls, and heap retention for map/channel contents also occur in Go's analyzer. They are conservative design boundaries, not evidence that an entire Go feature is unsupported. Future precision work should be justified by a reproducer that cmd/compile improves and LLGo does not, rather than treating every remaining allocation as an analysis bug.

## Follow-up measurements

Apple M4 Max, macOS/arm64, Go 1.27.3, LLVM 22.1.8. Compare the previous PR implementation (`2508246b0`, rebased to `4f39b7d5b`) with this follow-up (`9ee0d75aa`), both on upstream `92a62c9e9`, including merged #3003. Both use `-O3 -lto=full` and matching runtime source roots. Five sequential alternating process pairs, 100,000 calls/probe; medians below. Go builds the same source with its default optimizations.

| Probe | LLGo before bytes | LLGo after bytes | Go bytes | LLGo before ns/call | LLGo after ns/call | Go ns/call |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| fixed-slice | 3,197,616 | 80 | 0 | 18.48 | 1.31 | 1.73 |
| dynamic-slice-small | 3,200,784 | 80 | 0 | 16.95 | 5.62 | 2.12 |
| dynamic-slice-large | 14,400,080 | 14,395,904 | 12,838,032 | 60.34 | 59.35 | 26.79 |
| fixed-capacity | 3,200,656 | 80 | 0 | 17.19 | 1.54 | 2.04 |
| string-to-bytes | 1,600,896 | 80 | 0 | 11.26 | 3.35 | 3.76 |
| readonly-string-bytes | 14,399,568 | 80 | 0 | 66.61 | 2.06 | 1.94 |
| temporary-string-compare | 14,398,848 | 80 | 0 | 101.52 | 38.54 | 3.72 |
| temporary-map-lookup | 17,599,152 | 3,198,400 | 0 | 82.02 | 21.13 | 7.65 |
| bytes-to-string | 1,602,944 | 0 | 0 | 12.57 | 3.00 | 3.57 |
| string-to-runes | 4,799,600 | 0 | 0 | 26.52 | 4.81 | 9.79 |
| runes-to-string | 3,198,336 | 0 | 0 | 20.09 | 7.62 | 19.92 |
| string-concat | 3,202,560 | 0 | 0 | 19.66 | 4.76 | 8.78 |
| small-map | 22,400,416 | 0 | 0 | 113.73 | 9.38 | 9.22 |
| known-interface | 3,200,160 | 0 | 0 | 21.82 | 0.78 | 0.90 |

All 14 checksums agree across all runs. Allocated bytes are totals for 100,000 calls, not per-call counts; the observed 80-byte floor is retained. The 128-byte dynamic-slice heap control keeps its allocations: 60.34 → 59.35 ns/call in this run. An earlier sample on the previous main showed a **9.9% slowdown** (55.62 → 61.10 ns/call). The extra buffer/fallback path remains a cost to monitor; the timing variation does not establish a heap-path speedup or a universal slowdown estimate. Temporary map lookup still has the runtime allocation described above. String equality also remains slower than cmd/compile despite eliminating the copied storage.

The probe executable grows from 1,368,032 to 1,368,480 bytes (+448 bytes, 0.033%). These are isolated probes, not application throughput or compilation-speed claims. Earlier issue-specific and build-cost measurements remain labeled with their original revisions in [`benchmark/escape/README.md`](../benchmark/escape/README.md).

Validation includes race-tested engine proofs (94.4% statement coverage), compiler/runtime IR and metadata suites, recursive/cross-package cold/warm cache tests with 1/8 build workers, native public language tests, native default/full-LTO escape tests, and actual WASI/Emscripten memory32/memory64 runs. The same new semantic cases also pass cmd/compile. Pointer-bearing explicit-root storage deliberately keeps its conservative placement.

## Reproduce

[`benchmark/escape/coverage`](../benchmark/escape/coverage/main.go) runs the same source with both compilers. Small probes use `//go:noinline` to preserve call boundaries. The mutated-byte probe requires a real copy; the read-only long-string probe measures copy elision. Dynamic 128-byte slices are a heap-fallback control. Checksums must agree.

```sh
go build -gcflags=-m=2 -o /tmp/go-escape ./benchmark/escape/coverage
llgo build -O3 -lto=full -o /tmp/llgo-escape ./benchmark/escape/coverage
/tmp/go-escape -n=100000
/tmp/llgo-escape -n=100000
llgo build -gcflags=-m=2 ./benchmark/escape/coverage
```

Measure `MemStats.TotalAlloc` separately from elapsed time: LLGo's native `Mallocs` accounting remains tracked by #2923. Compare allocation bytes within each compiler; BDWGC rounding and Go allocation classes differ. Use the shared probe source from this PR for both worktrees. Existing baseline/follow-up LLGo binaries must use their matching runtime source roots, so runtime helper changes do not contaminate the baseline.

Go references: [graph and solver](https://github.com/golang/go/tree/go1.27.3/src/cmd/compile/internal/escape), [frontend pass order](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/gc/main.go), [make lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/builtin.go), [conversion lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/convert.go), [string concatenation](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/expr.go). Semantic regressions are exercised by [`test/escape`](../test/escape), engine proofs by `internal/goescape`, and target IR by `cl/escape_test.go` and `ssa/make_32bit_test.go`.
