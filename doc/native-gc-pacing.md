# Native collection pacing

Native LLGo programs use BDWGC. Its scan-cost heuristic weights traced and
pointer-free allocations differently. Reducing temporary allocations can shrink
the initial heap and the next allocation budget, making an unchanged steady
workload collect more frequently (issue #3106).

LLGo adds an allocation-budget floor after initialization and each completed
collection:

```
budget = max(4 MiB, occupied heap blocks) * GOGC / 100
effective budget = max(BDWGC scan-cost budget, LLGo budget, preexisting minimum)
```

Occupied blocks are a conservative estimate from BDWGC's heap statistics, not
an exact measurement of live Go objects. The floor includes pointer-free storage
and the minimum heap budget avoids excessive collection in small heaps. It can
increase physical memory use in exchange for less collection work. It does not
allocate an artificial warmup heap or change the root set.

The collector's scan-cost budget also participates: at GOGC=100 it permits its
full scan-work estimate between collections, rather than BDWGC's default one
third. This includes the fixed cost of scanning roots, which a heap-only floor
cannot account for. Other percentages map to the collector's integer free-space
divisor, rounded toward more frequent collection; the occupied-heap floor
continues to scale with the full percentage.

`GOGC` defaults to 100. `GOGC=off` and negative `debug.SetGCPercent` values disable
automatic collection; explicit `runtime.GC` remains available. Changing the
percentage is synchronized and returns the previous value. BDWGC can retain its
current scan-cost estimate until the next collection, so a percentage change is
not an exact heap-size limit. Existing BDWGC process-wide disablement is preserved.

The collection callback performs only C arithmetic and uses the collector's
unsynchronized statistics API while holding its allocation lock. It neither
allocates nor calls Go code, and preserves a previously installed event callback.
No work is added to ordinary allocation calls.

This policy applies to native BDWGC builds. WebAssembly, bare-metal and `nogc`
builds keep their existing collector policies. `debug.SetMemoryLimit` is not
implemented by this change. The allocation-only and frozen application
comparisons must report collection counts and heap footprints alongside time;
passing semantic tests does not establish a workload speedup.
