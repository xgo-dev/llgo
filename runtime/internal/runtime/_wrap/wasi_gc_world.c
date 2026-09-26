#define _POSIX_C_SOURCE 200809L
#include <errno.h>
#include <pthread.h>
#include <stdint.h>
#include <time.h>

// Count each entering thread before it registers Go roots and keep it counted
// until its roots are gone. The collector can then wait or skip a cycle without
// holding this mutex across Go allocation or TLS setup.
static pthread_mutex_t world_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t world_changed = PTHREAD_COND_INITIALIZER;
static uint32_t world_epoch;
static uint32_t world_registered;
static uint32_t world_stopped;
static pthread_t world_owner;
static _Thread_local uint32_t world_seen_epoch;
static _Thread_local int world_is_registered;

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

// Wake hosted Go condition waiters often enough to publish their roots for a
// stop-the-world request, even when no application signal is forthcoming.
void llgo_wasi_gc_cond_timedwait(pthread_cond_t *condition,
                                pthread_mutex_t *mutex) {
  struct timespec deadline;
  if (clock_gettime(CLOCK_REALTIME, &deadline) != 0)
    __builtin_trap();
  deadline.tv_nsec += 20000000;
  if (deadline.tv_nsec >= 1000000000) {
    deadline.tv_sec++;
    deadline.tv_nsec -= 1000000000;
  }
  int status = pthread_cond_timedwait(condition, mutex, &deadline);
  if (status != 0 && status != ETIMEDOUT)
    __builtin_trap();
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
  world_unlock();
}

// A thread in an uninstrumented C call may not reach a Go safepoint. A
// bounded wait skips collection until it returns; collection must never
// inspect a running C stack.
int llgo_wasi_gc_stop(void) {
  world_lock();
  if ((world_epoch & 1) || !world_is_registered || world_registered == 0)
    __builtin_trap();
  world_owner = pthread_self();
  world_stopped = 0;
  world_epoch++;

  struct timespec deadline;
  if (clock_gettime(CLOCK_REALTIME, &deadline) != 0)
    __builtin_trap();
  deadline.tv_nsec += 500000000;
  if (deadline.tv_nsec >= 1000000000) {
    deadline.tv_sec++;
    deadline.tv_nsec -= 1000000000;
  }
  while (world_stopped < world_registered - (uint32_t)world_is_registered) {
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
