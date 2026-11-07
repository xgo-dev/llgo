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

Apple M4 Max, macOS/arm64, Go 1.27.3, LLVM 22.1.8. Compare the previous PR implementation (`2508246b0`, rebased to `ccfc1061d`) with this follow-up (`4d0e32f9b`), both on upstream `9a9cf779b`, including merged #3003. Both use `-O3 -lto=full` and matching runtime source roots. Five sequential alternating process pairs, 100,000 calls/probe; medians below. Go builds the same source with its default optimizations.

| Probe | LLGo before bytes | LLGo after bytes | Go bytes | LLGo before ns/call | LLGo after ns/call | Go ns/call |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| fixed-slice | 3,197,616 | 80 | 0 | 17.00 | 1.29 | 1.72 |
| dynamic-slice-small | 3,201,264 | 80 | 0 | 15.80 | 5.57 | 2.08 |
| dynamic-slice-large | 14,400,080 | 14,395,760 | 12,838,032 | 56.17 | 60.03 | 25.80 |
| fixed-capacity | 3,198,288 | 80 | 0 | 15.74 | 1.43 | 1.98 |
| string-to-bytes | 1,600,576 | 80 | 0 | 12.73 | 2.98 | 3.65 |
| readonly-string-bytes | 14,400,000 | 80 | 0 | 59.37 | 1.87 | 1.95 |
| temporary-string-compare | 14,398,848 | 80 | 0 | 95.56 | 37.00 | 3.74 |
| temporary-map-lookup | 17,599,488 | 3,198,400 | 0 | 75.94 | 22.96 | 7.62 |
| bytes-to-string | 1,602,752 | 0 | 0 | 11.92 | 2.94 | 3.55 |
| string-to-runes | 4,799,600 | 0 | 0 | 26.43 | 5.21 | 10.20 |
| runes-to-string | 3,197,568 | 0 | 0 | 19.12 | 6.78 | 19.43 |
| string-concat | 3,203,808 | 0 | 0 | 18.03 | 4.61 | 8.58 |
| small-map | 22,402,432 | 0 | 0 | 102.94 | 8.76 | 9.14 |
| known-interface | 3,200,480 | 0 | 0 | 19.69 | 0.76 | 0.81 |

All 14 checksums agree across all runs. Allocated bytes are totals for 100,000 calls, not per-call counts; the observed 80-byte floor is retained. The 128-byte dynamic-slice heap control retains its allocations and slows from 56.17 to 60.03 ns/call (**6.9%**). An earlier sample on a previous main showed a 9.9% slowdown, while another showed a small improvement. The buffer/fallback path has a cost; these samples do not establish a universal slowdown estimate. Temporary map lookup still has the runtime allocation described above. String equality remains slower than cmd/compile despite eliminating copied storage.

The probe executable grows from 1,368,032 to 1,384,992 bytes (+16,960 bytes, 1.24%). The previous-main follow-up measured +448 bytes (0.033%); the current measurements supersede those size figures. These are isolated probes, not application throughput or compilation-speed claims. Earlier issue-specific measurements remain labeled with their original revisions in [`benchmark/escape/README.md`](../benchmark/escape/README.md).

Validation includes race-tested engine proofs (94.5% statement coverage), compiler/runtime IR and metadata suites, recursive/cross-package cold/warm cache tests with 1/8 build workers, native public language tests, native default/full-LTO escape tests, and actual WASI/Emscripten memory32/memory64 runs. The same new semantic cases also pass cmd/compile. Pointer-bearing explicit-root storage deliberately keeps its conservative placement.

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
