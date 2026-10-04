#define _POSIX_C_SOURCE 200809L
#include <errno.h>
#include <pthread.h>
#include <stdint.h>
#include <stdatomic.h>
#include <time.h>

// Count each entering thread before it registers Go roots and keep it counted
// until its roots are gone. The collector can then wait or skip a cycle without
// holding this mutex across Go allocation or TLS setup.
static pthread_mutex_t world_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t world_changed = PTHREAD_COND_INITIALIZER;
static _Atomic uint32_t world_epoch;
// A C call that cannot rendezvous aborts this collection instead of sweeping
// a running stack. This is a deadline, not a periodic mutator wakeup.
#define LLGO_WASI_GC_STOP_TIMEOUT_NS 500000000L
static uint32_t world_registered;
static uint32_t world_stopped;
static uint32_t world_blocked;
static uint32_t world_resuming;
static pthread_t world_owner;
static _Thread_local uint32_t world_seen_epoch;
static _Thread_local int world_is_registered;
static _Thread_local int world_is_blocked;

extern void llgo_gcroot_publish_thread(uintptr_t chain, uintptr_t bottom,
                                       uintptr_t top);
extern void llgo_gcroot_reset_thread_chain(void);
extern void *llgo_wasi_gc_mstart(void *arg);

void *llgo_wasi_gc_thread_start(void *arg) {
  llgo_gcroot_reset_thread_chain();
  return llgo_wasi_gc_mstart(arg);
}

static void world_lock(void) {
  if (pthread_mutex_lock(&world_mutex) != 0)
    __builtin_trap();
}

static void world_unlock(void) {
  if (pthread_mutex_unlock(&world_mutex) != 0)
    __builtin_trap();
}

static void world_wait(void) {
  if (pthread_cond_wait(&world_changed, &world_mutex) != 0)
    __builtin_trap();
}

// These waits only access pthread state. Publish the suspended Go caller's
// roots before entering C and prevent its return to Go during collection.
// In particular, cond_wait may block while reacquiring a mutex whose owner
// has already stopped at an allocation safepoint.
static void world_begin_wait(uintptr_t chain, uintptr_t bottom, uintptr_t top) {
  if (!world_is_registered)
    return;
  llgo_gcroot_publish_thread(chain, bottom, top);
  world_lock();
  if (world_is_blocked)
    __builtin_trap();
  world_is_blocked = 1;
  world_blocked++;
  if (pthread_cond_broadcast(&world_changed) != 0)
    __builtin_trap();
  world_unlock();
}

static void world_end_wait(void) {
  if (!world_is_registered)
    return;
  world_lock();
  world_resuming++;
  while (world_epoch & 1)
    world_wait();
  if (!world_is_blocked || world_blocked == 0)
    __builtin_trap();
  world_blocked--;
  world_is_blocked = 0;
  world_resuming--;
  if (pthread_cond_broadcast(&world_changed) != 0)
    __builtin_trap();
  world_unlock();
}

#if defined(__wasm__)
// TinyGC has one allocator mutex. Let an explicit collector hand it to a
// pending allocator before collecting again, without imposing FIFO handoff
// costs on every allocation.
static _Atomic uint32_t allocator_waiters;
static _Atomic uint32_t allocator_generation;
static _Atomic uint32_t allocator_yielders;

void llgo_wasi_gc_allocator_lock(pthread_mutex_t *mutex, uintptr_t chain,
                                 uintptr_t bottom, uintptr_t top) {
  atomic_fetch_add(&allocator_waiters, 1);
  world_begin_wait(chain, bottom, top);
  if (pthread_mutex_lock(mutex) != 0)
    __builtin_trap();
  atomic_fetch_sub(&allocator_waiters, 1);
  atomic_fetch_add(&allocator_generation, 1);
  if (atomic_load(&allocator_yielders) != 0)
    __builtin_wasm_memory_atomic_notify((int *)&allocator_generation, INT32_MAX);
  world_end_wait();
}

void llgo_wasi_gc_allocator_finish(pthread_mutex_t *mutex, uintptr_t chain,
                                   uintptr_t bottom, uintptr_t top) {
  if (atomic_load(&allocator_waiters) == 0) {
    if (pthread_mutex_unlock(mutex) != 0)
      __builtin_trap();
    return;
  }
  // Publish roots before unlocking: the next owner may itself collect while
  // this caller waits for the handoff in C.
  world_begin_wait(chain, bottom, top);
  atomic_fetch_add(&allocator_yielders, 1);
  uint32_t generation = atomic_load(&allocator_generation);
  if (pthread_mutex_unlock(mutex) != 0)
    __builtin_trap();
  while (atomic_load(&allocator_generation) == generation)
    __builtin_wasm_memory_atomic_wait32((int *)&allocator_generation,
                                       generation, -1);
  atomic_fetch_sub(&allocator_yielders, 1);
  world_end_wait();
}
#endif

void llgo_wasi_gc_mutex_lock(pthread_mutex_t *mutex, uintptr_t chain,
                             uintptr_t bottom, uintptr_t top) {
  world_begin_wait(chain, bottom, top);
  if (pthread_mutex_lock(mutex) != 0)
    __builtin_trap();
  world_end_wait();
}

void llgo_wasi_gc_cond_timedwait(pthread_cond_t *condition,
                                pthread_mutex_t *mutex, int64_t wait_nanos,
                                int monotonic, uintptr_t chain,
                                uintptr_t bottom, uintptr_t top) {
  world_begin_wait(chain, bottom, top);
  if (wait_nanos < 0) {
    if (pthread_cond_wait(condition, mutex) != 0)
      __builtin_trap();
    world_end_wait();
    return;
  }
  struct timespec deadline;
  if (clock_gettime(monotonic ? CLOCK_MONOTONIC : CLOCK_REALTIME, &deadline) != 0)
    __builtin_trap();
  deadline.tv_sec += (time_t)(wait_nanos / 1000000000);
  deadline.tv_nsec += (long)(wait_nanos % 1000000000);
  if (deadline.tv_nsec >= 1000000000) {
    deadline.tv_sec++;
    deadline.tv_nsec -= 1000000000;
  }
  int status = pthread_cond_timedwait(condition, mutex, &deadline);
  if (status != 0 && status != ETIMEDOUT)
    __builtin_trap();
  world_end_wait();
}

void llgo_wasi_gc_enter_begin(void) {
  world_lock();
  while (world_epoch & 1)
    world_wait();
  if (world_is_registered)
    __builtin_trap();
  world_is_registered = 1;
  world_registered++;
  world_unlock();
}

void llgo_wasi_gc_enter_end(void) {
  if (!world_is_registered)
    __builtin_trap();
}

void llgo_wasi_gc_leave_begin(void) {
  if (!world_is_registered)
    __builtin_trap();
}

void llgo_wasi_gc_leave_end(void) {
  world_lock();
  if (world_registered == 0 || !world_is_registered)
    __builtin_trap();
  world_is_registered = 0;
  world_registered--;
  if (pthread_cond_broadcast(&world_changed) != 0)
    __builtin_trap();
  world_unlock();
}

int llgo_wasi_gc_pending(void) {
  // The overwhelmingly common case needs no process-wide pthread lock.
  // Only inspect world_owner while holding the lock after observing a stop.
  if (!(atomic_load_explicit(&world_epoch, memory_order_acquire) & 1))
    return 0;
  world_lock();
  int pending = (world_epoch & 1) &&
      !pthread_equal(world_owner, pthread_self());
  world_unlock();
  return pending;
}

int llgo_wasi_gc_registered(void) {
  world_lock();
  int count = (int)world_registered;
  world_unlock();
  return count;
}

void llgo_wasi_gc_park(uintptr_t chain, uintptr_t bottom, uintptr_t top) {
  llgo_gcroot_publish_thread(chain, bottom, top);
  world_lock();
  world_resuming++;
  while ((world_epoch & 1) && !pthread_equal(world_owner, pthread_self())) {
    uint32_t epoch = world_epoch;
    if (world_seen_epoch != epoch) {
      world_seen_epoch = epoch;
      world_stopped++;
      if (pthread_cond_broadcast(&world_changed) != 0)
        __builtin_trap();
    }
    while (world_epoch == epoch)
      world_wait();
  }
  world_resuming--;
  if (pthread_cond_broadcast(&world_changed) != 0)
    __builtin_trap();
  world_unlock();
}

// A thread in an uninstrumented C call may not reach a Go safepoint. A
// bounded wait skips collection until it returns; collection must never
// inspect a running C stack.
int llgo_wasi_gc_stop(void) {
  world_lock();
  if ((world_epoch & 1) || !world_is_registered || world_registered == 0)
    __builtin_trap();
  // A new collection must let waiters leave the previous rendezvous first.
  // Otherwise repeated runtime.GC calls can re-park them without Go progress.
  while (world_resuming != 0)
    world_wait();
  world_owner = pthread_self();
  world_stopped = 0;
  world_epoch++;

  struct timespec deadline;
  if (clock_gettime(CLOCK_REALTIME, &deadline) != 0)
    __builtin_trap();
  deadline.tv_nsec += LLGO_WASI_GC_STOP_TIMEOUT_NS;
  if (deadline.tv_nsec >= 1000000000) {
    deadline.tv_sec++;
    deadline.tv_nsec -= 1000000000;
  }
  while (world_stopped + world_blocked <
         world_registered - (uint32_t)world_is_registered) {
    int status = pthread_cond_timedwait(&world_changed, &world_mutex,
                                        &deadline);
    if (status == ETIMEDOUT) {
      world_epoch++;
      if (pthread_cond_broadcast(&world_changed) != 0)
        __builtin_trap();
      world_unlock();
      return 0;
    }
    if (status != 0)
      __builtin_trap();
  }
  world_unlock();
  return 1;
}

void llgo_wasi_gc_resume(void) {
  world_lock();
  if (!(world_epoch & 1) || !pthread_equal(world_owner, pthread_self()))
    __builtin_trap();
  world_epoch++;
  if (pthread_cond_broadcast(&world_changed) != 0)
    __builtin_trap();
  world_unlock();
}
