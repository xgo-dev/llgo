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
| Trusted runtime/foreign contracts | Runtime pointer-hiding helpers and assembly noescape annotations | Shared `llgo.noescape` for runtime/ABI helpers, bodyless `//go:noescape` contracts; ordinary pointer-to-uintptr conversion remains conservative |
| Safety fallbacks | Unknown calls and unsupported proofs are conservative | Unknown calls/SSA, irreducible CFG and solver-budget fallback retain heap storage |

## Remaining differences

1. **Using inlined call-site context in escape analysis.** Merged PR [#3003](https://github.com/xgo-dev/llgo/pull/3003) implements frontend cross-package Go SSA body import and LLVM inlining; that capability is covered there. At its merged revision `d510ec8b5`, imported functions are lowered as `available_externally` LLVM definitions without cloning/inlining their Go SSA into the caller or rerunning this analysis on an inlined Go graph. For example, `apply(fn func(*int), p *int) { fn(p) }` still exports conservative publication for `p`, even when a caller supplies a known borrowing function. Go can inline the helper before escape analysis and connect those particular arguments. The remaining gap is this context-sensitive proof, not cross-package inlining itself. LLVM inlining and heap-to-stack may still remove the resulting allocation later, so this is not evidence of an unavoidable runtime allocation.
2. **Explicit GC-root targets.** Newly promoted pointer-bearing objects and map storage stay in scanned heap memory until the collector can scan typed stack fields for the full lifetime, including writes by borrowing callees. Pointer-free character/slice buffers can use the stack. Removing this restriction requires root-registration/lifetime work, not a weaker escape predicate.
3. **Additional copy elision.** Read-only `[]byte(string)` and immediately consumed temporary `string([]byte)` are covered. LLGo's temporary-string proof requires one use in the same block, with no intervening instruction other than debug information. More general use patterns, or cases exposed only after inlining (for example a string comparison hidden inside a helper), can retain a copy. Source mutation between conversion and use must preserve the original copied string.
4. **Receiver-word and defer lowering.** Native synchronous interface receiver words use caller storage. Deferred/asynchronous invocations and explicit-root targets retain the original allocation path. Extending those cases requires storage owned by the defer/goroutine or a collector-aware frame.
5. **Target heuristics and build cost.** Stack size/alignment limits, native fixed stacks, vector layouts and conservative versus explicit GC roots differ. Cross-package effects are recomputed from available SSA instead of using Go's serialized escape tags. This affects optimization opportunities and build cost, independently of correctness.
6. **By-value lexical closure capture.** Go's escape pass can copy a small, unmodified variable with no observable address use directly into a closure environment. LLGo currently retains the Go SSA lexical binding representation, which uses pointers to captured variables, except for its existing zero-sized environment elimination. An escaping closure can therefore retain separate captured-variable storage that Go avoids. General by-value capture requires coordinated capture proofs and environment layout changes; LLVM may still remove some resulting storage later.

Field-insensitivity, unknown indirect calls, and heap retention for map/channel contents also occur in Go's analyzer. They are conservative design boundaries, not evidence that an entire Go feature is unsupported. Future precision work should be justified by a reproducer that cmd/compile improves and LLGo does not, rather than treating every remaining allocation as an analysis bug.

## Follow-up measurements

Previous #2997 implementation `2508246b0`, rebased to `9ee3ba5e2`, versus measured implementation `ccda44f8f`, both on main `be65dbdeb` (including #3010). The preceding implementation is `05b4ab0d3`, on that same main.

M4 Max, macOS/arm64, LLVM 22.1.8, Go 1.27.3. LLGo: `-O3 -lto=full`; Go: default optimization. Same source and matching checksums; medians of five sequential rounds (baseline, preceding implementation, measured implementation, Go). Allocation bytes are totals for 100,000 calls; time is ns/call.

| Probe | LLGo before bytes | LLGo after bytes | Go bytes | LLGo before ns/call | LLGo after ns/call | Go ns/call |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| fixed-slice | 3,197,616 | 0 | 0 | 17.80 | 1.29 | 1.73 |
| dynamic-slice-small | 3,201,264 | 0 | 0 | 15.99 | 5.85 | 2.14 |
| dynamic-slice-large | 14,400,080 | 14,396,112 | 12,837,920 | 57.49 | 56.28 | 26.60 |
| fixed-capacity | 3,200,656 | 0 | 0 | 17.90 | 1.54 | 2.07 |
| string-to-bytes | 1,600,896 | 0 | 0 | 10.89 | 3.30 | 3.78 |
| readonly-string-bytes | 14,399,568 | 0 | 0 | 62.17 | 2.14 | 1.93 |
| temporary-string-compare | 14,398,848 | 0 | 0 | 102.48 | 42.08 | 3.84 |
| temporary-map-lookup | 17,597,168 | 0 | 0 | 81.14 | 7.19 | 8.22 |
| bytes-to-string | 1,602,944 | 0 | 0 | 12.89 | 3.31 | 3.55 |
| string-to-runes | 4,799,600 | 0 | 0 | 26.42 | 5.95 | 10.32 |
| runes-to-string | 3,197,856 | 0 | 0 | 19.85 | 8.20 | 20.85 |
| string-concat | 3,202,560 | 0 | 0 | 20.93 | 5.22 | 8.47 |
| small-map | 22,400,416 | 0 | 0 | 104.78 | 9.70 | 9.88 |
| known-interface | 3,200,032 | 0 | 0 | 21.71 | 0.77 | 0.89 |

The latest contracts remove the former 80 B measurement overhead (five heap output words in native `ReadMemStats`) and the string-map key-header allocation: in the same batch, lookup changes from 3,198,400 to 0 B and 21.97 to 7.19 ns/call. Mutation through `NoEscape` still requires an owned byte copy.

Both compilers heap-allocate the 128 B dynamic slice. This host's BDWGC allocation class is 144 B versus Go's 128 B, explaining the byte difference. Its time is 2.1% faster than baseline in this batch; the previous batch was 6.9% slower, so no stable heap-path speedup is established. Some short controls vary (read-only bytes: 1.93 → 2.14 ns/call versus the preceding implementation); temporary string comparison remains much slower than Go.

Executable: 1,368,032 → 1,384,944 B (+16,912 B, +1.24%). These are isolated probes; they do not establish application-wide performance.

The executable size and timing series above supersede the previous-main comparison; earlier issue-specific figures retain their original revision labels in [`benchmark/escape/README.md`](../benchmark/escape/README.md).

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
