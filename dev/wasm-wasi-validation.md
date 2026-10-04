# Wasmer threads, SIMD and exception validation

The W32 backend keeps one goroutine per pthread. The following checks qualify
that backend; they do not introduce an M:N scheduler on native or WASI targets.

## Fixed failure modes

- The default runner is the unmodified Wasmer 7.5.0 CLI with Cranelift on Unix
  and V8 on Windows,
  standard Wasm EH, SIMD and shared memory. LLVM emits standard EH directly
  with `-wasm-use-legacy-eh=false`, including the LTO link; no Binaryen
  translation or runtime patch is needed. WASI libc supplies pthread creation
  and TLS, and the runtime maps `pthread_exit` to `wasix_32v1.thread_exit`.
  This is a Preview 1 + WASI threads + WASIX thread-exit host contract, not a
  portable single-thread Preview 1 binary. The earlier WAMR exception failures
  are recorded in [wasm-eh-comparison.md](wasm-eh-comparison.md).
- A pthread condition wait can block while reacquiring a mutex held by a Go
  thread already stopped for collection. Merely polling the condition every
  20 ms did not resolve that cycle: finalizer tests repeatedly skipped GC and
  exceeded their three-second deadlines. Runtime mutex/condition waits now
  publish the suspended Go caller's roots before entering C and prevent a
  return to Go until collection finishes. This includes the timer scheduler.
  Blocked callers no longer wake every 20 ms: normal conditions wait for a
  signal, and timers wait for their actual deadline. Arbitrary user C calls
  retain the bounded-wait/skip-collection behavior.
- The initial thread parked after Goexit must unregister its roots, since it
  will never execute another Go safepoint. A worker now verifies that GC still
  advances after initial Goexit with the timer service active.
- Repeated explicit GC must allow waiting allocations and resumed threads to
  progress. The allocator uses a GC-safe pthread mutex instead of spinning
  through repeated host calls, and the next collection waits for the previous
  rendezvous to finish. After an explicit GC, a collector with pending allocator
  waiters waits in C for another lock acquisition before returning to Go. This
  prevents a tight GC loop from repeatedly taking the mutex ahead of woken
  allocators; ordinary allocations retain normal pthread mutex handoff. The
  collector publishes its roots while waiting because the next owner may GC. Compiler polls and runtime symbol-table
  initialization waiters acknowledge collection requests. Arbitrary user C
  calls can still prevent collection. The 32-thread/1000-round concurrent
  function-info test timed out at 100 seconds with the spin lock; the blocking
  allocator completed it in 10.72 seconds (nogc comparison: 7.26 seconds) on
  macOS arm64. These are local regression timings, not cross-host benchmarks.
- Small programs begin with a 1 MiB GC arena; subsequent arenas double up to
  32 MiB. This avoids sweeping a nearly empty 32 MiB arena on every explicit
  collection.
- Reflection's whole-program type-name lookup compares encoded names without
  allocating a temporary string for each extra-star type. This preserves the
  existing type identity and name rules while reducing GC contention.

Safepoint checks use an atomic epoch read before acquiring the rendezvous
mutex. Heap scans carry the known segment through block-state operations;
address and block lookup use binary search, including out-of-order libc arenas.
The host metadata regression covers all 128 segment slots, gaps, sentinels,
metadata exclusion and preservation of marked objects during sweep.

The focused `test/go` regression compiles once with a 300-second deadline.
It then runs three groups in fresh Wasmer invocations: finalizers/callback GC/
function-info, the 20 pointer-argument startup races, and the 20 zero-argument
startup races. Each invocation has a 300-second deadline and verbose output.
Separate deadlines retain every case and repetition and keep first-use checks
independent of earlier test initialization.

Wasmer's compiled-module cache remains enabled. Wasmer tracing and guest
stderr use the same host stream, so `llgo test` and these acceptance drivers set
`RUST_LOG=off` for the host process. Guest stdout/stderr and runner failures are
preserved without parsing or filtering their text. Ordinary `llgo run` retains
the caller's logging configuration; use `RUST_LOG=warn` when diagnosing the
engine. Wasmer 7.5.0 normally disables tracing when `RUST_LOG` is unset.
The acceptance driver uses a fresh cache to verify cold/warm execution, unchanged
cached artifacts, separate guest stdout/stderr, and a nonzero guest exit status.
Guest stderr deliberately resembles an engine log to guard against text filters.

The installer verifies pinned release archive SHA-256 digests. Prebuilt hosts
are macOS arm64, Linux amd64/aarch64/riscv64 and Windows amd64 (including use
from MinGW). Wasmer 7.5.0 does not publish a macOS Intel archive; that host
requires a source-built CLI on PATH. The Windows CLI is a standalone host
process and does not need to match the guest compiler's C ABI. Its official
archive only includes the V8 backend. All invocations leave backend selection to
Wasmer: the pinned Windows CLI selects V8, while supported Unix builds prefer
Cranelift for these modules. Backend selection also checks the module's required
features. Windows host directories still need explicit POSIX guest mappings.

## Reproducible checks

```sh
bash dev/install_wasmer.sh
go test ./internal/build -run '^TestWASIGCWaitsSuspendGoAcrossMutexReacquisition$'
LLGO="$PWD/.bin/llgo" WASMER=/path/to/wasmer python3 dev/test_wasm_wasi_threads.py
LLGO_WASI_THREADS=1 go run ./dev/wasmstdlib -full -profile W32-WASI \
  -shard 9 -shards 16 -llgo "$PWD/.bin/llgo" -report w32-shard9.json
```

The raw SIMD regression spawns a WASI thread and checks v128 calls and a
v128 standard-EH payload before an atomic wakeup. The pthread regression exercises both mutex waits and condition-variable
reacquisition, including a waiter woken while collection is still active.
The Wasmer acceptance driver covers GC/nogc exception isolation, main/init/worker
Goexit, finalizers, reflection with concurrent GC, retained roots, heap growth,
uncooperative C, timers, filesystems, selected standard-library tests and a
GOROOT sentinel. The full package audit remains a separate gate; passing this
focused suite is not a claim of complete standard-library compatibility.
