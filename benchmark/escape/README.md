# Escape-analysis probes

These programs reproduce the allocation patterns in #2975 (boxing), #2974
(borrowed by-value parameters) and #2932 (deferred mutex unlock). They include
control cases and verify the results, so deleting the measured work is not a
valid optimization.

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

[`coverage/main.go`](coverage/main.go) compares fixed/dynamic slices, strings and rune conversions, temporary byte/string views, small maps and known interface calls against the Go compiler. See the [Go compiler comparison](../../doc/escape-analysis-comparison.md) for all 14 probes, previous-PR versus follow-up measurements, the heap-fallback regression, and remaining analysis/lowering gaps.
