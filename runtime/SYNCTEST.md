# Synctest on LLGo

The private hooks and compatibility tests are pinned to Go 1.27. The scheduler
owns bubble membership, runnable accounting, and durable wait transitions;
public runtime hooks forward to it. Native pthreads and the WebAssembly fiber
and worker backends use the same accounting with their existing wait/wake
primitives.

A bubble's clock starts at 2000-01-01 UTC. Time advances to its next timer only
when all its goroutines are durably blocked and no captured asynchronous
reference remains active. `Wait` returns at a stable blocked state without
advancing future time. Due timers run before Wait returns; the main bubble
function's exit stops future clock advancement. Remaining blocked goroutines
produce a deadlock panic.

Bubble `time.Time` values have no monotonic reading, matching Go. Arithmetic
between an outside real timestamp and an inside fake timestamp uses wall time;
the fake nanosecond clock remains available to timer scheduling.

Channels and timers created inside a bubble retain its identity. Sends,
receives, close/select operations, and timer mutations from another bubble or
from outside it are rejected. A wait on an outside channel, a mutex/RWMutex,
I/O, or a syscall remains active. A channel wait is durable only for an inside
channel (or a nil channel); a select is durable only when every non-nil case is
inside. Wakes restore runnable accounting before publishing readiness.

WaitGroups use the standard library's association flag and a runtime side
table for escaped objects. Global variables remain unbubbled. Associations are
removed on WaitGroup reset and bubble teardown. Cond waiters carry their own
bubble identity; an outside wake is rejected. Mutexes keep ordinary scheduler
behavior and do not advance fake time.

Captured references keep a bubble active through `acquire`/`release`.
`inBubble` temporarily enters that identity on an outside callback and restores
the previous context even after panic. New goroutines inherit the identity
before they become runnable. Context teardown releases membership, including
Goexit and recovered-panic paths. Runtime timer, cleanup, finalizer, and unique
services initialize outside the bubble; their private channels do not acquire
test ownership.

No-bubble execution keeps the ordinary clock and wait paths. Tests exercise
fake time, timers/tickers/AfterFunc, WaitGroup/Cond, outside waits, independent
bubbles, callback references, panic cleanup, and invalid boundaries. Run the
same tests when changing the supported Go version or a scheduler backend:

```sh
llgo test ./test/std/testing/synctest ./test/std/runtime ./test/std/sync ./test/std/time
llgo test -target emscripten -emulator ./test/std/testing/synctest
llgo test -target emscripten-memory64 -emulator ./test/std/testing/synctest
llgo test -target wasip1 -emulator ./test/std/testing/synctest
dev/test_std_buildmodes.sh ./test/std/runtime/abicall ./test/std/testing/synctest
```
