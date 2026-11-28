# Defer Locality and GC Integration

## Background

`defer` chains belong to the current goroutine, and unwind paths must locate the
active `*runtime.Defer`. A raw pointer stored only in pthread TLS is invisible to
the Boehm collector. In stress scenarios—e.g. `TestDeferLoopStress` with
1,000,000 defers—the collector could reclaim defer nodes, leaving dangling
pointers and causing crashes inside deferred closures.

Prior experiments (`test-defer-dont-free` branch) confirmed the crash disappeared when allocations bypassed GC (plain `malloc` without `free`), pointing to a root-registration gap rather than logical corruption.

## Current Design

1. **Runtime-owned goroutine state**
   - Each `runtimeContext` owns its `g`, `m`, and `p`; `g.defer_` is the current
     goroutine's defer-chain head.
   - The context is allocated with `AllocRoot`, so pointers reachable through
     `g` remain visible to the collector.
   - A pointer-free `//llgointernal:tls` `uintptr` slot locates the current `g`. The
     slot is only an address cache; the `runtimeContext` allocation is the GC
     root.
   - Bare-metal targets keep the same state in ordinary globals because they
     have one execution context and their LLVM backends do not support native
     TLS relocations.
   - Runtime-created threads release that root in `mexit`. A pthread key remains
     only as a thread-exit destructor sidecar for main and foreign threads; the
     current-G read path does not call `pthread_getspecific`.

2. **SSA codegen synchronization**
   - `ssa/eh.go` calls `runtime.ThreadDeferSlot` once per defer activation to
     obtain `&getg().defer_`, then loads the previous head, installs the new
     frame, and restores the previous link on exit/unwind through that cached
     slot pointer.
   - Reusing the slot pointer across the activation relies on the invariant
     that the `*g` observed at frame install remains the current goroutine's
     `*g` until the frame is unlinked (i.e. `setg` only transitions at thread
     startup/exit or scheduler boundaries outside active Go defer frames).
     `llgo.deferData()` still calls `runtime.GetThreadDefer` to read the head
     value directly.
   - Defer argument nodes and the `runtime.Defer` struct itself are allocated with `aggregateAllocU`, ensuring new memory comes from GC-managed heaps, and nodes are released via `runtime.FreeDeferNode`.

3. **Locality directives**
   - Runtime state that belongs to a logical goroutine uses `//llgointernal:gls`.
   - Physical scheduler and execution-resource slots that must be available
     before goroutine-local context setup use `//llgointernal:tls`.
   - TLS and GLS currently share one physical owner because LLGo binds one
     goroutine/P/M to one OS thread. The directive still records the intended
     lifetime so a future scheduler can keep GLS with a migrating goroutine
     while TLS remains with the thread.
   - Pointer-bearing locality variables are kept in the GC-rooted locality
     package payload rather than bespoke pthread-key allocations.
   - Bare-metal builds use ordinary globals for the caller and FIPS state:
     they have one logical context. This is an implementation fallback because
     the current GLS package-block accessor still uses a native TLS address
     cache, which those LLVM targets cannot lower.

4. **Dynamic `sync.Pool` state**
   - `sync.Pool` retains the GC-aware `clite/tls` handle. Each `Pool` needs a
     dynamically allocated per-thread slot, while locality directives declare
     a fixed set of static package variables.
   - Keeping the handle preserves the existing atomic hot path, thread-exit
     victim handoff, and cross-thread access to that victim. A static TLS map
     would add an explicit never-pruned `*Pool` key set and a map lookup to
     every `Get` and `Put`.
   - The existing dynamic handle can itself retain state until thread exit;
     matching upstream's GC-cycle pool cleanup would require a separate runtime
     integration. This migration deliberately preserves the baseline behavior
     instead of introducing a new retention mechanism.
   - This cache has per-P/execution-resource lifetime, not goroutine lifetime;
     dynamic TLS is correct for the current 1:1 P/M/thread model. Fully modeling
     a scheduler that can move P between M would require a dynamic P-local
     facility rather than GLS.

5. **Non-GC builds**
   - `FreeDeferNode` continues to release nodes via `c.Free` when building with
     `-tags nogc`.

## Testing

Run the stress and regression suites to validate the integration:

```sh
./dev/llgo.sh test ./test -run TestDeferLoopStress
./dev/llgo.sh test ./test
```

The updated `TestDeferLoopStress` now asserts 1,000,000 loop defers execute without failure, catching regressions in GC root tracking.
