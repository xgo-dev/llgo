# All-thread syscall ownership contract

This document records the thread-ownership decision for
[issue #2130](https://github.com/xgo-dev/llgo/issues/2130). It complements the
[native-call ABI](NATIVE_CALLS.md). It distinguishes the private Linux protocol
that can be tested today from the creation boundary required for production
support. It does not enable that protocol in application binaries.

## Build modes and platforms

| Configuration | Public syscall APIs | Direct runtime hook |
| --- | --- | --- |
| Native Linux executable | `ENOTSUP` | `ENOTSUP` |
| Linux `c-shared` or `c-archive` | `ENOTSUP` | `ENOTSUP` |
| Controlled native Linux amd64/arm64 C harness | Not linked | Private protocol |
| Other platforms | No new API | No Linux protocol |

An executable is not proof of exclusive ownership: libc, the collector, a
loaded library, or an emulator can create pthreads without entering LLGo.
Neither an import-graph scan nor two snapshots of `/proc/self/task` exclude a
thread creation between the last check and execution. The private weak enable
symbol belongs to the C harness; generated application entry modules must
never define it. Credential setters continue through libc's setxid wrappers.

Initial protocol qualification is Linux amd64 and arm64 only. Darwin,
Windows, other Linux architectures, bare metal, and WebAssembly do not acquire
support through pthread compatibility or this design. Each would need an
equivalent raw syscall ABI, signal/interruption mechanism, and creation barrier.

## Thread classes and membership

* **Owned:** the initial standalone runtime thread, explicitly marked owned,
  and threads created by `NewProc`. A child's registry entry is published
  before its native thread is created; attachment publishes its actual TID.
  Runtime service goroutines participate even when excluded from the user
  goroutine count. Ownership follows the native thread's lifetime.
* **Adopted:** an externally created thread that enters an exported Go callback.
  LLGo installs a G, traceback registration, and collector registration as
  needed, retaining them until its TLS destructor runs. This makes the thread
  visible; it does not transfer control over its creation or native lifetime.
  Its registry entry remains unowned. Nested callbacks on an owned thread reuse
  that thread's membership. A Go goroutine started by an adopted thread has a
  new runtime-created, owned native thread.
* **Foreign:** a thread without an LLGo runtime context, including collector
  helper threads and arbitrary native library threads. It has no participating
  registry entry. Registering an unwinder or disabling parallel GC markers
  does not establish ownership of such threads.

An owned subset is insufficient for credential consistency. The private
protocol rejects any adopted entry or foreign TID. Adopting an existing foreign
thread cannot repair its prior credential state by replaying an old request:
requests can contain expired pointers and need not be safe to replay. The main
thread of a C host also remains adopted; library initialization cannot promote
the host process to an LLGo-owned executable.

## Creation, exit, and the operation boundary

The registry mutex serializes registration, attachment, detachment, and each
private all-thread operation. The credential mutex is acquired first and also
serializes libc credential wrappers. Both locks are acquired while the caller
can still receive GC signals, before it masks signals or parks any peer.

For runtime-created threads, publishing an unattached entry before
`pthread_create` closes the creation gap: a request that observes that entry
returns `EAGAIN` without executing the syscall. If registration is blocked by
an operation, native creation has not started yet. If creation has started,
the entry already exists and cannot be ignored. A failed creation must remove
the entry. After attachment, the thread participates in the next cohort.

Exit marks a node dead before waiting for the registry mutex. An operation
that sees dead state returns `EAGAIN`, allowing teardown to proceed instead
of continually interrupting the exiting thread. Once a cohort is prepared,
its entries remain allocated until the operation releases the mutex. Native
teardown after removal can briefly remain visible in procfs; that is grounds
for rejection or retry, never permission to omit a live participant.

This is a registry barrier, not a barrier over arbitrary process thread
creators. Production activation additionally requires a creation gate held
from before native creation through attachment or failure for **every** thread
creator, including the collector and libraries, with these guarantees:

1. No thread can appear without a reserved entry or cross the gate while an
   operation is running.
2. Every reservation either attaches or is removed on failure; the coordinator
   cannot park a creator and then wait for that creator's unfinished child.
3. Threads created after release inherit the completed state from a
   participating creator. Foreign-thread adoption cannot bypass this rule.
4. Teardown cannot reuse a TID or free participant storage while a request or
   delayed handler still refers to it.

An interposed `pthread_create` alone is insufficient: direct `clone`, other
namespaces, and internal library thread APIs can bypass it. No runtime flag,
environment variable, or public opt-in is provided until a production backend
can enforce this boundary.

## Prepare, execute, and release

The private transport uses the existing SIGURG handler and per-thread registry
storage. It does not call Go, allocate, or take application/libc locks in its
asynchronous handler. Raw kernel operations implement masks, futexes, timeouts,
TID signaling, and the procfs completeness check.

1. Validate the enable symbol, calling context, syscall allowlist, ownership,
   attachment, and absence of outstanding handlers. Verify procfs contains
   exactly the registered cohort while holding the registry mutex.
2. Publish the request, arm each peer, and send SIGURG with its registered TID.
   Each handler claims its own node, blocks all maskable signals using the raw
   kernel mask (including libc-private GC/setxid signals), and publishes READY.
3. Wait for all peers to become READY, then repeat the completeness check.
   A masked or unavailable transport times out before execution. This checks
   the harness's controlled cohort; it does not make foreign creation safe.
4. Execute on the caller first. If it fails, return its raw result and errno
   and cancel preparation without executing on any peer.
5. On caller success, publish RUN. Each peer executes exactly once and
   publishes both result words and errno. All three must match the caller.
   Disagreement or a timeout after execution begins terminates the process
   with exit status 2; returning success or a recoverable error could expose
   partially changed credentials.
6. Publish RELEASE and wait for every handler to acknowledge that it will no
   longer read the request. Release the registry and credential locks. Each
   thread restores its previous signal mask. The caller's libc errno is
   preserved independently of the returned raw syscall errno.

Preparation has a one-second timeout; execution and release each have a
five-second timeout. Cancellation publishes CANCEL, wakes parked peers, and
disarms unclaimed nodes. A claimed handler can still be delayed: its node stays
live and its state prevents the next request from reusing shared arguments
until it acknowledges release. A retry during that interval returns `EAGAIN`.

The pthread backend does not migrate a G between OS threads, so locked-thread
code participates on its current native thread without a scheduler transition.
A thread blocked in C or a kernel wait must still receive SIGURG; otherwise
preparation fails without changing credentials. Synchronous and nested C
callbacks remain on their original participant. Calling the protocol from an
asynchronous user signal handler, or bypassing the credential gate with a
foreign credential setter, is outside the contract.

The private allowlist contains the per-thread credential setters `setuid`,
`setgid`, `setreuid`, `setregid`, `setresuid`, `setresgid`, and `setgroups`, plus
the harmless, process-identical `getpid` probe. Process-wide changes such as
`umask` or `chdir`, thread-dependent results such as `gettid`, signal/lifecycle
changes, blocking calls, and all unlisted traps return `ENOTSUP` before even
the caller executes. Argument memory must remain valid until return.

## C host cooperation

Both C library modes keep `ENOTSUP` before foreign threads exist, while they
run or call back into Go, and after they exit. An optional host API is deferred:
registering the host's current TIDs is insufficient. A usable API would need
the same creation gate, attachment/failure handshake, inheritance guarantee,
and teardown exclusion across every host dependency for the runtime's entire
lifetime. A declaration of cooperation without enforcement cannot enable the
public syscall APIs safely.

## Executable checks and remaining work

`go test ./internal/build -run TestAllThreads -count=1 -v` exercises:

* Multiple controlled pthreads running `getpid` and no-op credential syscalls,
  concurrent callers, a reserved-but-unattached child, creation/exit stress,
  foreign/adopted rejection, masked SIGURG, cancellation, and recovery.
* Thread-local seccomp filters proving caller failure executes no peer syscall,
  a peer error is fatal, and differing successful result values are fatal.
  These cases explicitly skip when the host cannot install seccomp filters;
  the cohort tests still run.
* The same protocol scenarios under ASan/UBSan.
* Executable rejection, a linked foreign creator, and actual `c-shared` and
  `c-archive` C hosts with threads created before initialization and afterward,
  including adopted callbacks and checks after joining those threads. Both
  public syscall forms and the direct runtime hook are checked.

The native harness qualifies synchronization within its controlled cohort;
it is not evidence of production creation exclusion. Issue #2130's production
activation remains dependent on an enforceable backend creation boundary and
tests of that boundary. The public APIs must retain `ENOTSUP` until then.
