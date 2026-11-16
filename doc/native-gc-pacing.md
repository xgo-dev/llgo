# Native collection pacing

Native LLGo programs require BDWGC (libgc) 8.2 or newer. Older releases are not
supported by this policy. Its scan-cost heuristic weights traced and
pointer-free allocations differently. Reducing temporary allocations can shrink
the initial heap and the next allocation budget, making an unchanged steady
workload collect more frequently (issue #3106).

LLGo adds an allocation-budget floor after initialization and each completed
collection:

```
p = last active non-negative GOGC percentage (initially 100)
heap budget = clamp(max(4 MiB, occupied heap blocks) * p / 100, 1, SIZE_MAX/2)
divisor = p == 0 ? UINTPTR_MAX : max(1, ceil(100 / p))
effective budget = max(BDWGC scan-work estimate / divisor, heap budget, preexisting minimum)
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
divisor, rounded toward more frequent collection. The divisor remains 1 for
percentages of 100 or greater; the occupied-heap floor continues to scale with
the full percentage. A zero percentage sets the heap floor to one byte.

The divisor also controls BDWGC's heap expansion step, approximately the current
heap size divided by the divisor before collector limits are applied. At the
default percentage this can grow the heap by about its current size, rather than
about one third. Both this expansion and the allocation budget can increase
managed and physical heap footprints; the percentage is a memory/time tradeoff.

`GOGC` defaults to 100. `GOGC=off` and negative `debug.SetGCPercent` values disable
automatic collection; explicit `runtime.GC` remains available. Changing the
percentage is synchronized and returns the previous value. BDWGC can retain its
current scan-cost estimate until the next collection, so a percentage change is
not an exact heap-size limit. Disabling GC preserves the active budget, so
restoring the same percentage does not force a collection. Small reductions can
retain the preceding estimate for one cycle. A substantially lower target or a
lowered target already exceeded by allocations refreshes the estimate with a
collection outside the policy lock, so finalizers can reenter the API.
Existing `GC_disable` nesting and an automatic-collection disable already set by
the host at policy initialization are preserved.

The collection callback performs only C arithmetic and uses the collector's
unsynchronized statistics API while holding its allocation lock. It neither
allocates nor calls Go code, and preserves a previously installed event callback.
No work is added to ordinary allocation calls.

This policy applies to native BDWGC builds. WebAssembly, bare-metal and `nogc`
builds keep their existing collector policies. `debug.SetMemoryLimit` is not
implemented by this change. The heap budget covers collector-managed blocks;
it does not bound process RSS, foreign allocations or JIT/linear memory.
