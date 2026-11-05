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
calls precede allocation snapshots. The reported `Mallocs` field in `defer`
is informational only: #2923 affects this counter on native BDWGC. Use
`TotalAlloc`/allocated bytes for allocation comparisons.

Timings are isolated kernels, not a VM/application throughput claim. Compare
both affected and control cases, native binary size, and compiler time/memory.
