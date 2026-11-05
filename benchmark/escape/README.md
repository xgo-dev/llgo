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
`d4b248e8` versus `83778ac00`, five alternating process pairs, medians.

| Case | Before | After |
| --- | ---: | ---: |
| Callback box, bytes/100k calls | 3,195,248 | 80 |
| Borrowed parameter, bytes/100k calls | 12,800,000 | 0 |
| Borrowed parameter, ns/op | 57.80 | 9.16 |
| Deferred unlock, bytes/op | 96 | 0 |
| Deferred unlock, ns/op | 51.86 | 12.03 |
| Value-call control, ns/op | 5.99 | 6.09 |
| Explicit-unlock control, ns/op | 7.84 | 7.86 |
| Boxing executable, bytes | 2,186,048 | 2,168,880 |
| Borrow executable, bytes | 2,185,312 | 2,168,080 |
| Defer executable, bytes | 2,487,136 | 2,469,632 |

All control allocation counts were unchanged. The 80-byte measurement floor
is reported as observed, not rounded to zero.

For the borrow program, five warm build pairs took 9.92 → 9.54 s with
614 → 632 MiB peak RSS; three cold pairs (`LLGO_BUILD_CACHE=0`) took
13.37 → 13.09 s with 747 → 747 MiB peak RSS. These are whole-process
`/usr/bin/time -l` measurements; the new analysis phase was about 0.25 s in
`-debug-trace` output. Warm cache still loads SSA and runs the analysis.
The small wall-time difference does not establish a compile-speed gain,
and the warm peak memory increase is about 2.9%.
