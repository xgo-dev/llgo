# Native libc calls and all-thread syscalls

LLGo's native pthread runtime runs a Go goroutine on its OS thread's stack.
There is no separate Go g0 stack or migrating M/P transition to emulate.
Synchronous C calls preserve the current G, its bubble, and the physical unwind
chain. A callback on the same thread reuses that context; a newly adopted C
thread starts outside any bubble.

These call ABIs are explicit:

| Entry | Target signature | Result |
| --- | --- | --- |
| Existing syscall/x/sys trampoline | C function with actual syscall arguments | Existing syscall return/errno rules |
| `runtime.libcCall` / `runtime.asmcgocall` | `int32_t (*)(void *argblock)` | Target's int32 return |
| Linux `syscall.cgocaller` | `void (*)(struct { uintptr_t *args; uintptr_t retval; } *)` | Wrapper writes errno into retval |

An argument block remains live through the synchronous call. The target owns
its errno convention; the direct bridge does not read or reinterpret errno.
The Linux credential wrappers save errno immediately, then release their gate.
Nil direct-call pointers panic; an unavailable packed cgo bridge returns
`ENOTSUP`. Arbitrary upstream Go assembly trampolines still require an actual
C ABI implementation; a symbol's name never selects an ABI.

The nine `syscall.cgo_libc_set*` pointers are initialized to typed C wrappers.
They use libc's pthread/setxid implementation, which includes libc-managed C
threads. A shared gate serializes these wrappers with LLGo all-thread calls.
Calling arbitrary credential setters directly from foreign C code bypasses
that gate and is outside the all-thread contract.

Native stack capture and profiling retain their existing signal handlers and
physical unwinder. A C callback can allocate, collect, or make a nested call.
The asynchronous all-thread handler itself runs only raw kernel operations and
atomics; it never calls Go, allocates, or acquires application locks.

## Linux ownership and barrier

Public `AllThreadsSyscall` and `AllThreadsSyscall6` retain Go's libc/cgo guard
and return `ENOTSUP` in ordinary LLGo native Linux executables and all C
shared/archive libraries. Every such executable links pthread/libc code; an import graph or
an executable build mode cannot prove exclusion of all foreign thread
creators. The `syscall` credential setters use the libc wrappers above and
remain supported independently of the raw all-thread API.

Standalone Linux amd64/arm64 executables can explicitly select a controlled
thread set:

```sh
llgo build -tags=llgo.allthreads ./app
```

This is an application contract: **every thread creator must use the runtime's
registration and creation/exit barrier**. Unknown C-library worker pools,
direct `pthread_create`/`clone`, hidden native helpers, and foreign credential
setters are outside this mode. A successful call never grants permission to
start an uncoordinated thread later. In particular, `/proc/self/task` checks
can detect a visible violation but cannot prove that arbitrary native code
will not create a thread after the check. Programs that cannot establish this
boundary must keep the default `ENOTSUP` behavior.

The executable entry configures the collector before its first runtime/context
allocation. It fixes `GC_MARKERS=1` and calls `GC_set_markers_count(1)`; BDWGC's
environment otherwise overrides that setter. This intentionally disables
parallel native marker threads for the opt-in mode. A collector already
initialized by an earlier C constructor, or missing configuration APIs, leaves
the public API disabled. `nogc` executables do not acquire a GC dependency.
C shared/archive modes never emit this activation, even with the tag. Other
Linux architectures retain `ENOTSUP`.

A runtime-created pthread has a registry entry before `pthread_create` runs.
The main thread is explicitly marked owned. Adopted foreign threads are
registered for callbacks and stack capture but remain unowned. Parallel GC
marker threads, foreign C threads, and emulator helper threads are outside the
owned cohort; their presence returns `ENOTSUP`. The collector configuration
does not override ownership checks. A C callback on an owned runtime thread
reuses its context; an adopted host thread remains unowned through teardown.

The registry lock excludes registration, attachment, and exit. An entry whose
thread is not yet attached returns `EAGAIN` before executing the syscall. All
other threads park in the existing SIGURG transport, with kernel signal masks
blocking GC/libc-private signals while parked. `/proc/self/task` validates
completeness both before and after the registered cohort is stopped; it does
not supply the cohort or replace the creation/exit barrier.

A masked transport or incomplete preparation returns `EAGAIN` without applying
the syscall. Cancellation leaves per-thread storage alive until every delayed
handler acknowledges it; a retry can also return `EAGAIN` during this interval.
The calling thread executes first. Its error returns without invoking other
threads. On success, every other participant executes once; all result words
and errno must agree. Disagreement, or a peer blocked after execution starts,
terminates the process because partial state cannot be rolled back.

The bridge uses an explicit allowlist: `getpid` for harmless functional probes,
and raw `setuid`, `setgid`, `setreuid`, `setregid`, `setresuid`, `setresgid`, and
`setgroups`. The setters change per-thread kernel credentials and return zero
on success. Every other trap
returns `ENOTSUP` before executing the caller syscall, including process-wide
operations such as `umask`, `chdir`, and `setsid`, and thread-dependent probes
such as `gettid`. This is initially Linux-only; it does not introduce a host
cooperation API for C-library modes or claim full arbitrary-C process support.

The tests cover executables and both C library modes on Linux/macOS, packed
calls with GC, nested callbacks, signals and unwind, credential errors, and the
Linux native protocol under creation/exit stress plus ASan/UBSan. Ordinary
executables and library modes require `ENOTSUP`. Controlled GC and `nogc`
executables exercise both public signatures with `getpid`, no-op credentials,
locked threads, GC and concurrent thread creation/exit. Earlier C constructors
that initialize the collector or start foreign threads are rejected. A linked C shared-library integration
test checks rejection before pthread creation, while its thread is live, and
after joining it, including the direct runtime hook. A C-only native test
supplies a controlled owned cohort for no-op `setresuid(-1, -1, -1)` calls,
rejected process-wide calls with an unchanged umask, and barrier cancellation.
Isolated seccomp subprocesses verify fatal result disagreement and prove that
a caller syscall error does not execute the syscall on its peers. Those cases
report an explicit unavailable result on kernels/emulators without the native
seccomp facility. Both C-library modes are also tested with the opt-in tag.
