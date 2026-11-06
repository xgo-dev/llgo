# Go-layer escape analysis comparison

Reference: Go **1.27.3**, the installed toolchain used for the comparison. This is a comparison of source-level storage decisions and emitted/runtime allocations, not a claim that LLGo implements every cmd/compile optimization. LLVM heap-to-stack is a subsequent, independent optimization.

## Implemented coverage

| Area | Go 1.27.3 | LLGo |
| --- | --- | --- |
| Core model | Weighted location graph, parameter/result leaks, recursive batches | Adapted graph and leak encoding, SSA lifetime scopes and call SCC fixed point |
| Local/parameter/result addresses | Taking an address alone does not require heap storage | Same principle; target layout and GC constraints feed back before final summaries |
| Cross-package calls | Exported parameter effects | Effects derived from the exact dependency SSA, including archive-cache hits; immutable worker input |
| Interface boxes/closures | Separate implicit storage, capture and call effects | Box/payload separation, closure environments, captures and concrete interface-call effects |
| Known function/interface values | Frontend devirtualization and inlining improve escape precision | Bounded direct/phi target resolution for the analysis, preserving original lowering and ABI |
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

1. **General inlining and context specialization before escape analysis.** LLGo's Go SSA analysis precedes LLVM inlining. The bounded value/capture resolution does not replace arbitrary frontend inlining. For example, a helper `apply(fn func(*int), p *int) { fn(p) }` still exports conservative publication for `p`, even when a caller supplies a known borrowing function. Go can inline the helper and connect those particular arguments. PR #3003 imports bodies for LLVM inlining; it does not by itself rerun this Go analysis on an inlined SSA graph.
2. **Explicit GC-root targets.** Newly promoted pointer-bearing objects and map storage stay in scanned heap memory until the collector can scan typed stack fields for the full lifetime, including writes by borrowing callees. Pointer-free character/slice buffers can use the stack. Removing this restriction requires root-registration/lifetime work, not a weaker escape predicate.
3. **Additional copy elision.** Read-only `[]byte(string)` and immediately consumed temporary `string([]byte)` are covered. LLGo's temporary-string proof requires one use in the same block, with no intervening instruction other than debug information. More general use patterns, or cases exposed only after inlining (for example a string comparison hidden inside a helper), can retain a copy. Source mutation between conversion and use must preserve the original copied string.
4. **Receiver-word and defer lowering.** Native synchronous interface receiver words use caller storage. Deferred/asynchronous invocations and explicit-root targets retain the original allocation path. Extending those cases requires storage owned by the defer/goroutine or a collector-aware frame.
5. **Runtime ABI temporaries.** Eliminating a source conversion does not eliminate every allocation inside a runtime helper. The string-map lookup probe still allocates approximately 32 bytes/call after removing its 128-byte string copy; cmd/compile allocates no bytes for this probe. This needs separate runtime/lowering lifetime work.
6. **Target heuristics and build cost.** Stack size/alignment limits, native fixed stacks, vector layouts and conservative versus explicit GC roots differ. Cross-package effects are recomputed from available SSA instead of using Go's serialized escape tags. This affects optimization opportunities and build cost, independently of correctness.

Field-insensitivity, unknown indirect calls, and heap retention for map/channel contents also occur in Go's analyzer. They are conservative design boundaries, not evidence that an entire Go feature is unsupported. Future precision work should be justified by a reproducer that cmd/compile improves and LLGo does not, rather than treating every remaining allocation as an analysis bug.

## Follow-up measurements

Apple M4 Max, macOS/arm64, Go 1.27.3, LLVM 22.1.8. Compare the previous PR implementation (`2508246b0`, rebased to `1be30bc85` on upstream `5448d82c1`) with this follow-up, both using `-O3 -lto=full` and matching runtime source roots. Five sequential alternating process pairs, 100,000 calls/probe; medians below. Go builds the same source with its default optimizations.

| Probe | LLGo before bytes | LLGo after bytes | Go bytes | LLGo before ns/call | LLGo after ns/call | Go ns/call |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| fixed-slice | 3,197,616 | 80 | 0 | 17.11 | 1.29 | 1.82 |
| dynamic-slice-small | 3,201,264 | 80 | 0 | 15.91 | 5.91 | 2.15 |
| dynamic-slice-large | 14,400,080 | 14,395,760 | 12,832,728 | 55.62 | 61.10 | 26.73 |
| fixed-capacity | 3,198,576 | 80 | 0 | 17.64 | 1.63 | 2.06 |
| string-to-bytes | 1,600,576 | 80 | 0 | 13.40 | 3.35 | 3.82 |
| readonly-string-bytes | 14,400,000 | 80 | 0 | 63.54 | 2.22 | 1.94 |
| temporary-string-compare | 14,398,848 | 80 | 0 | 97.18 | 38.45 | 3.88 |
| temporary-map-lookup | 17,599,552 | 3,198,400 | 0 | 80.54 | 23.49 | 8.28 |
| bytes-to-string | 1,602,752 | 0 | 0 | 12.29 | 3.15 | 3.62 |
| string-to-runes | 4,799,600 | 0 | 0 | 26.02 | 4.79 | 11.02 |
| runes-to-string | 3,198,112 | 0 | 0 | 20.07 | 8.38 | 20.29 |
| string-concat | 3,203,136 | 0 | 0 | 20.32 | 4.86 | 8.29 |
| small-map | 22,402,432 | 0 | 0 | 106.74 | 9.17 | 9.22 |
| known-interface | 3,200,672 | 0 | 0 | 20.65 | 0.78 | 0.86 |

All 14 checksums agree across all runs. Allocated bytes are totals for 100,000 calls, not per-call counts; the observed 80-byte floor is retained. The 128-byte dynamic-slice heap control keeps its allocations and is **9.9% slower** in this run (55.62 → 61.10 ns/call), reflecting the extra buffer/fallback path in this kernel; it is not a universal slowdown estimate. Temporary map lookup still has the runtime allocation described above. String equality also remains slower than cmd/compile despite eliminating the copied storage.

The probe executable grows from 1,367,440 to 1,367,904 bytes (+464 bytes, 0.034%). These are isolated probes, not application throughput or compilation-speed claims. Earlier issue-specific and build-cost measurements remain labeled with their original revisions in [`benchmark/escape/README.md`](../benchmark/escape/README.md).

Validation includes race-tested engine proofs (94.4% statement coverage), compiler/runtime IR and metadata suites, recursive/cross-package cold/warm cache tests with 1/8 build workers, native public language tests, native default/full-LTO escape tests, and actual WASI/Emscripten memory32/memory64 runs. The same new semantic cases also pass cmd/compile. Pointer-bearing explicit-root storage deliberately keeps its conservative placement.

## Reproduce

[`benchmark/escape/coverage`](../benchmark/escape/coverage/main.go) runs the same source with both compilers. Small probes use `//go:noinline` to preserve call boundaries. The mutated-byte probe requires a real copy; the read-only long-string probe measures copy elision. Dynamic 128-byte slices are a heap-fallback control. Checksums must agree.

```sh
go build -gcflags=-m=2 -o /tmp/go-escape ./benchmark/escape/coverage
llgo build -O3 -lto=full -o /tmp/llgo-escape ./benchmark/escape/coverage
/tmp/go-escape -n=100000
/tmp/llgo-escape -n=100000
LLGO_ESCAPE=github.com/xgo-dev/llgo/benchmark/escape/coverage llgo build ./benchmark/escape/coverage
```

Measure `MemStats.TotalAlloc` separately from elapsed time: LLGo's native `Mallocs` accounting remains tracked by #2923. Compare allocation bytes within each compiler; BDWGC rounding and Go allocation classes differ. Existing baseline/follow-up LLGo binaries must use their matching runtime source roots, so runtime helper changes do not contaminate the baseline.

Go references: [graph and solver](https://github.com/golang/go/tree/go1.27.3/src/cmd/compile/internal/escape), [frontend pass order](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/gc/main.go), [make lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/builtin.go), [conversion lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/convert.go), [string concatenation](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/expr.go). Semantic regressions are exercised by [`test/escape`](../test/escape), engine proofs by `internal/goescape`, and target IR by `cl/escape_test.go` and `ssa/make_32bit_test.go`.
