# WASI threaded GC

The Wasmer WASI pthread backend runs one goroutine per pthread.
It supports collection with multiple application threads: one thread stops
the others and performs serial, non-moving, conservative mark-and-sweep.
Marking and sweeping do not run in parallel with application code or on
multiple collector threads. `-tags nogc` remains available.

Threads publish compiler root chains and native stack bounds before
acknowledging a stop. Collection also scans global roots and explicitly
registered host-TLS roots. Go allocations occupy separate libc arenas;
their capacity excludes metadata and alignment padding. Objects cannot cross
arenas. Freed object space is reusable, but empty arenas are not returned to
libc.

An uninstrumented C call may never acknowledge a stop. After a 500 ms deadline,
the collector resumes stopped threads and skips marking and sweeping. In
that case `runtime.GC()` returns without advancing `MemStats.NumGC`. If more
space is needed, allocation adds a disjoint arena under the allocator lock
without requesting another stop. Repeated allocations can still trigger
subsequent 500 ms collection attempts. A permanently blocked C call can
prevent all reclamation and exhaust the module's memory limit.

This also applies to foreign C threads that explicitly enter Go through
`EnterForeignThread`: returning through `ExitForeignThread` currently retains
their registration until pthread destruction. A long-lived C thread pool
idling after a callback can therefore prevent collection. This is a known
progress limitation, not a guarantee that arbitrary C blocking is GC-safe.
Automatically detaching those threads requires a separate protocol for any
Go roots retained by C or host TLS.

`dev/test_wasm_wasi_threads.py` exercises collection with worker-private roots,
thread entry/exit, an idle foreign thread, blocked C calls, timers and panic
recovery. It also allocates just below, at and above 32 MiB in separate Wasmer
processes so an earlier large arena cannot hide a sizing regression.
