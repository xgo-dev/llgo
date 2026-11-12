# Go-layer escape analysis comparison

Reference: Go **1.27.3**, the installed toolchain used for the comparison. This is a comparison of source-level storage decisions and emitted/runtime allocations, not a claim that LLGo implements every cmd/compile optimization. LLVM heap-to-stack is a subsequent, independent optimization.

The [unified escape-analysis proposal](https://github.com/xgo-dev/llgo/issues/3012) describes the complete design and implementation status.

## Implemented coverage

| Area | Go 1.27.3 | LLGo |
| --- | --- | --- |
| Core model | Weighted location graph, parameter/result leaks, recursive batches | Adapted graph and leak encoding, SSA lifetime scopes and call SCC fixed point |
| Local/parameter/result addresses | Taking an address alone does not require heap storage | Same principle; target layout and GC constraints feed back before final summaries |
| Cross-package calls | Exported parameter effects | Opt-in checked-type/frontend exports and complete SCC facts; source/body import on misses; immutable worker input |
| Interface boxes/closures | Separate implicit storage, capture and call effects | Box/payload separation, closure environments, captures and concrete interface-call effects |
| Known function/interface values | Frontend devirtualization and inlining improve escape precision | Bounded direct/phi target resolution for the analysis; #3003 supplies frontend cross-package body import for LLVM inlining |
| Callback invocation | Callee effects; additional precision after inlining | Known callback capture effects, including retained callback results; unknown callees remain conservative |
| Fixed slice backing arrays | Stack when local and within size/alignment limits | Stack with the same lifetime proof, including constant-capacity makes with runtime length checks |
| Dynamic slice capacity | Small fixed caller buffer with heap fallback | 32-byte caller buffer with heap fallback; pointer elements retain heap lifetime |
| String/byte/rune conversion and concatenation | Small caller buffers, heap fallback, copy-elision rules | Explicit implicit-storage nodes, small caller buffers and original large-result helpers |
| Temporary byte/string views | No-copy conversion for supported read-only/temporary uses | Read-only `[]byte(string)` and bounded mutation-free `string([]byte)` regions spanning multiple consumers, branches and inspectable read-only helpers; solve again with the actual source alias |
| Local maps | Caller-owned header and small initial backing storage | Caller-owned header and typed initial bucket, including explicit-root targets; existing growth/overflow allocator |
| Map key arguments | Caller-owned synchronous temporary | Entry-frame key copies, typed field roots and shared stack budget; runtime-owned copies for inserted keys |
| Map/string range iterators | Caller-owned iteration state | Invocation-owned entry slots with typed field roots and shared budget; heap fallback for oversized frames |
| Channels | Runtime allocation | Runtime allocation; this is not an outstanding heap-to-stack optimization in cmd/compile |
| defer/go | Invocation and asynchronous lifetimes differ | Non-loop native defer storage is invocation-local; loop/foreign/asynchronous ownership is retained |
| Trusted runtime/foreign contracts | Runtime pointer-hiding helpers and assembly noescape annotations | Shared `llgo.noescape` for runtime/ABI helpers, bodyless `//go:noescape` contracts; ordinary pointer-to-uintptr conversion remains conservative |
| KeepAlive | Dedicated SSA liveness operation with GC stack maps | An intrinsic borrowing/no-mutation/no-callback contract feeds the unified graph; box placement shares invocation/iteration lifetimes and the 64 KiB budget. Type-metadata omission is independent of placement. |
| Safety fallbacks | Unknown calls and unsupported proofs are conservative | Unknown calls/SSA, irreducible CFG and solver-budget fallback retain heap storage |

## Remaining differences

1. **Bounded call-site context.** Known callback and concrete interface arguments specialize completed callee summaries without rewriting Go SSA or changing the generic body's placement. This includes forwarding helpers. Recursive groups, unavailable bodies, unknown alternatives and exhausted work limits retain generic effects. Contexts are limited to four levels, 512 instructions per body and 64 solves per generic body. LLVM inlining remains independent (PR [#3003](https://github.com/xgo-dev/llgo/pull/3003)).
2. **Explicit GC-root targets.** Eligible pointer-bearing locals, interface boxes, closure environments, slice/map buffers and non-loop defer storage use typed field-address roots. The collector reads current fields, including writes in borrowing callees. Physical LLVM offsets/widths cover Go's wider Wasm32 words and native C pointers. Root slots share the function budget; unknown ownership and oversized storage retain heap placement.
3. **Additional copy elision.** Temporary byte-to-string views support multiple comparisons/lookups and read-only helpers across acyclic control flow. Every path to the last consumer must be free of writes and unknown calls; checking parameter mutation alone would miss global aliases. Cycles, retained results and unsupported consumers retain a copy. Region and helper inspection have bounded work.
4. **Receiver-word and defer-record lowering.** Synchronous receiver adapters use caller storage on native and explicit-root targets. Invocation-owned defer frames and non-loop records have typed roots when needed. Loop/foreign-owner records and asynchronous receiver adapters retain heap ownership. Argument boxes follow the same lifetime, GC and budget decisions.
5. **Target heuristics and build cost.** Stack size/alignment limits, native fixed stacks, vector layouts and conservative versus explicit GC roots differ. Opt-in frontend exports omit eligible dependency checking/SSA/lowering, while complete SCC facts omit repeated escape solving for source-built functions. Required generic/inline bodies and changed Wasm program contexts retain source analysis. Cache validation still has a cost.
6. **By-value lexical closure capture.** Small lexical variables with one dominating initialization, no later assignment, no observable address and read-only capture access are copied into closure environments. This avoids separate captured-variable heap storage for eligible escaping closures. Shared writable variables, forwarded captures, large values and method wrappers retain reference capture. Analysis and lowering consume the same immutable capture plan.

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

The measured contracts remove the former 80 B measurement overhead (five heap output words in native `ReadMemStats`) and the string-map key-header allocation: in the same batch, lookup changes from 3,198,400 to 0 B and 21.97 to 7.19 ns/call. Mutation through `NoEscape` still requires an owned byte copy.

Both compilers heap-allocate the 128 B dynamic slice. This host's BDWGC allocation class is 144 B versus Go's 128 B, explaining the byte difference. Its time is 2.1% faster than baseline in this batch; the previous batch was 6.9% slower, so no stable heap-path speedup is established. Some short controls vary (read-only bytes: 1.93 → 2.14 ns/call versus the preceding implementation); temporary string comparison remains much slower than Go.

Executable: 1,368,032 → 1,384,944 B (+16,912 B, +1.24%). These are isolated probes; they do not establish application-wide performance.

The executable size and timing series above supersede the previous-main comparison; earlier issue-specific figures retain their original revision labels in [`benchmark/escape/README.md`](../benchmark/escape/README.md).

Validation at the measured revision includes race-tested engine proofs (94.5% statement coverage), compiler/runtime IR and metadata suites, recursive/cross-package cold/warm cache tests with 1/8 build workers, native public language tests, native default/full-LTO escape tests, and actual WASI/Emscripten memory32/memory64 runs. The same new semantic cases also pass cmd/compile. Ordinary pointer-bearing explicit-root storage deliberately keeps its conservative placement; the KeepAlive-only immutable-box exception is validated separately below.

## KeepAlive integration validation (2026-11-08)

Implementation snapshot: `2f17a50cb` (#2997), including foundation `d5eca1d9e` (#3016). The numbered escape benchmark table above still refers to `ccda44f8f` and has not been rerun as a complete batch for this integration.

Focused KeepAlive tests pass with Go on macOS/arm64, native LLGo on macOS/arm64 and Linux/amd64, and LLGo on actual Wasm J32/J64/W32 runtimes. Cases include boxing before GC, separate loop-defer snapshots, panic-unwind GC before a deferred KeepAlive, and indirect/reflected function calls. IR tests verify the borrowing contract and shared budget: three 32 KiB boxes permit only two stack slots, and later helper buffers respect the remaining capacity. Full compiler/escape/C ABI suites and focused cache/runtime-contract checks pass.

The aggregate microprobe remains **0 B/op** and has the same allocation-free assembly as standalone #3016. This focused result does not update the historical timings or executable-size comparison above and does not establish application-wide performance.

## Reproduce

[`benchmark/escape/coverage`](../benchmark/escape/coverage/main.go) runs the same source with both compilers. Small probes use `//go:noinline` to preserve call boundaries. The mutated-byte probe requires a real copy; the read-only long-string probe measures copy elision. Dynamic 128-byte slices are a heap-fallback control. Checksums must agree.

```sh
go build -gcflags=-m=2 -o /tmp/go-escape ./benchmark/escape/coverage
llgo build -O3 -lto=full -o /tmp/llgo-escape ./benchmark/escape/coverage
/tmp/go-escape -n=100000
/tmp/llgo-escape -n=100000
llgo build -gcflags=-m=2 ./benchmark/escape/coverage
```

Measure `MemStats.TotalAlloc` separately from elapsed time. Native BDWGC `Mallocs` now reports recorded allocation events; historical runs before the #2923 fix did not use it. Compare allocation bytes within each compiler; BDWGC rounding and Go allocation classes differ. Use the shared probe source from this PR for both worktrees. Existing baseline/follow-up LLGo binaries must use their matching runtime source roots, so runtime helper changes do not contaminate the baseline.

Go references: [graph and solver](https://github.com/golang/go/tree/go1.27.3/src/cmd/compile/internal/escape), [frontend pass order](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/gc/main.go), [make lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/builtin.go), [conversion lowering](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/convert.go), [string concatenation](https://github.com/golang/go/blob/go1.27.3/src/cmd/compile/internal/walk/expr.go). Semantic regressions are exercised by [`test/escape`](../test/escape), engine proofs by `internal/goescape`, and target IR by `cl/escape_test.go` and `ssa/make_32bit_test.go`.

### Frontend-export cache measurement (2026-11-09)

On macOS/arm64 with Go 1.27.3, build `benchmark/escape/coverage` using the same compiler, `-p=8`, warm archive namespaces, and five alternating runs of `LLGO_ESCAPE_CACHE=0` / `1`. Disabling persistence gave a 3.9685 s median; enabling frontend/SCC persistence gave 4.2925 s (8.2% slower). SSA construction fell from 92 packages to 24; 67 packages used validated opaque contracts. This fixture required no checked-type exports because its eligible optimization bodies retained source checking. The result supports actual SSA omission, not a general end-to-end speedup; persistence remains opt-in.

The shared Go-compatible semantic fixture separately loaded checked private-type exports. Actual J32/J64 reuse runs built SSA for 22 packages instead of 82 (59 opaque archive packages); W32 built 22 instead of 77 (54 opaque packages). Removing the W32 reflection call recomputed the program's feature decisions with reflection bridges disabled, then reused the new context. All seeded/reused artifacts executed successfully. These runs deliberately used 1/8 workers to verify cache portability, so their times are not a controlled speed comparison.
