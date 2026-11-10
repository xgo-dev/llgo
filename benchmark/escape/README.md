# Escape-analysis probes

These programs reproduce the allocation patterns in #2975 (boxing), #2974
(borrowed by-value parameters) and #2932 (deferred mutex unlock). They include
control cases and verify the results, so deleting the measured work is not a
valid optimization.

`mapkeys` covers #3012's synchronous map-key temporaries: struct lookup,
update, missing-key deletion, indirect-key lookup, map/string range and a scalar-key control.
It reports collector bytes over 100,000 calls, ns/call and checked results.
Build the same source with Go and both LLGo compilers:

```sh
go build -o mapkeys-go ./benchmark/escape/mapkeys
llgo build -O3 -lto=full -o mapkeys-llgo ./benchmark/escape/mapkeys
```

Use five alternating baseline/candidate runs and report median timings.
LLGo's native bytes use BDWGC accounting; Go reports managed allocation bytes.
Each probe has its own function budget; these measurements do not model
contention between cold and hot sites for the shared stack budget.

Build the same source with baseline and candidate compilers, then run the two
binaries in alternating order on an otherwise idle machine:

```sh
llgo build -O3 -lto=full -o boxing ./benchmark/escape/boxing
llgo build -O3 -lto=full -o borrow ./benchmark/escape/borrow
llgo build -O3 -lto=full -o defer ./benchmark/escape/defer
./boxing
./borrow
./defer
```

`boxing` reports bytes for 100,000 calls; `borrow` reports seven timed samples
of 100,000 calls; `defer` uses testing.Benchmark with a 500 ms target. Warm-up
calls precede allocation snapshots. The reported `reportedMallocsPerOp` field in `defer`
is informational only: #2923 affects this counter on native BDWGC. Use
`TotalAlloc`/allocated bytes for allocation comparisons.

Timings are isolated kernels, not a VM/application throughput claim. Compare
both affected and control cases, native binary size, and compiler time/memory.

## Reference measurements

Apple M4 Max, macOS/arm64, Go 1.27.3, LLVM 22, `-O3 -lto=full`:
`d4b248e8` versus `9cf504cd1`, five alternating process pairs, medians.

| Case | Before | After |
| --- | ---: | ---: |
| Callback box, bytes/100k calls | 3,195,248 | 80 |
| Borrowed parameter, bytes/100k calls | 12,800,000 | 0 |
| Borrowed parameter, ns/op | 57.37 | 9.24 |
| Deferred unlock, bytes/op | 95 | 0 |
| Deferred unlock, ns/op | 52.55 | 12.12 |
| Value-call control, ns/op | 6.01 | 6.13 |
| Explicit-unlock control, ns/op | 7.89 | 7.92 |
| Boxing executable, bytes | 2,186,048 | 2,168,880 |
| Borrow executable, bytes | 2,185,312 | 2,168,080 |
| Defer executable, bytes | 2,487,136 | 2,469,632 |

All control allocation counts were unchanged. The 80-byte measurement floor
is reported as observed, not rounded to zero.

For the borrow program, five warm build pairs took 9.86 → 9.49 s with
608 → 609 MiB peak RSS; three cold pairs (`LLGO_BUILD_CACHE=0`) took
13.46 → 13.10 s with 732 → 767 MiB peak RSS. These are whole-process
`/usr/bin/time -l` measurements; the new analysis phase was about 0.175 s in
`-debug-trace` output. Warm cache still loads SSA and runs the analysis.
The small wall-time difference does not establish a compile-speed gain,
and the cold peak memory increase is about 4.8%.

## Additional storage and conversion coverage

[`coverage/main.go`](coverage/main.go) compares fixed/dynamic slices, strings and rune conversions, temporary byte/string views, small maps and known interface calls against the Go compiler. See the [Go compiler comparison](../../doc/escape-analysis-comparison.md) for all 14 probes, previous-PR versus follow-up measurements, the heap-fallback control, and remaining analysis/lowering gaps.

## Runtime temporary measurements (2026-11-10)

M4 Max, macOS/arm64, LLVM 22.1.8, Go 1.27.3, GOMAXPROCS=4; five alternating
rounds of 100,000 calls. Main `4a7f42648` includes #3021–#3023; measured fix
`c70305d7f` uses the same source. LLGo uses `-O3 -lto=full`, Go defaults.
Each cell gives allocation bytes per call / ns per call.

| Case | Go (B / ns per call) | LLGo main (B / ns per call) | This PR (B / ns per call) |
| --- | ---: | ---: | ---: |
| Struct key lookup | 0.00 / 6.91 | 32.00 / 32.99 | 0.00 / 18.91 |
| Struct key update | 0.00 / 7.07 | 32.02 / 35.91 | 0.00 / 18.51 |
| Struct key delete (missing) | 0.00 / 6.29 | 31.99 / 21.12 | 0.00 / 7.53 |
| Indirect key lookup | 0.00 / 17.57 | 272.01 / 144.54 | 0.00 / 48.06 |
| Map range | 0.00 / 25.83 | 112.01 / 69.08 | 0.00 / 24.49 |
| String range | 0.00 / 3.86 | 32.02 / 18.32 | 0.00 / 3.73 |
| Scalar key control | 0.00 / 2.87 | 0.00 / 2.90 | 0.00 / 2.80 |

Native probe size: 2,138,144 → 2,137,760 bytes. These isolated timings do not
establish application-wide speedups.
