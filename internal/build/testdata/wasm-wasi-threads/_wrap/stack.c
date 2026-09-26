#define _GNU_SOURCE
#include <errno.h>
#include <pthread.h>
#include <stdint.h>
#include <errno.h>
#include <time.h>

extern int llgo_timer_cond_init(pthread_cond_t *condition);
extern int llgo_timer_cond_timedwait(pthread_cond_t *condition,
                                   pthread_mutex_t *mutex, int64_t wait_nanos);

int64_t llgo_probe_monotonic_clock(void) {
  struct timespec now;
  if (clock_gettime(CLOCK_MONOTONIC, &now) != 0)
    return -1;
  return (int64_t)now.tv_sec * INT64_C(1000000000) + now.tv_nsec;
}

int32_t llgo_probe_timer_clock(void) {
  pthread_cond_t condition;
  pthread_mutex_t mutex = PTHREAD_MUTEX_INITIALIZER;
  if (llgo_timer_cond_init(&condition) != 0)
    return 0;
  pthread_mutex_lock(&mutex);
  int64_t start = llgo_probe_monotonic_clock();
  int result = 0;
  while (result == 0)
    result = llgo_timer_cond_timedwait(&condition, &mutex, INT64_C(2000000));
  int64_t elapsed = llgo_probe_monotonic_clock() - start;
  pthread_mutex_unlock(&mutex);
  pthread_mutex_destroy(&mutex);
  pthread_cond_destroy(&condition);
  return start >= 0 && result == ETIMEDOUT && elapsed >= INT64_C(2000000);
}

// Compile the collector's stack-bound helper into this nogc probe so the
// pthread ABI is checked before threaded collection is enabled.
#include "../../../../../runtime/internal/runtime/tinygogc/_wrap/gc_wasm.c"

extern unsigned char __stack_high;

int32_t llgo_wasi_worker_stack_bounds(void) {
  pthread_attr_t attr;
  void *base = 0;
  size_t size = 0;
  int status = pthread_getattr_np(pthread_self(), &attr);
  uintptr_t sp = (uintptr_t)&attr;
  uintptr_t top = llgo_gc_stack_top();
  if (status == ENOSYS)
    return top == (uintptr_t)&__stack_high && sp < top;
  if (status != 0)
    return 0;
  status = pthread_attr_getstack(&attr, &base, &size);
  pthread_attr_destroy(&attr);
  uintptr_t start = (uintptr_t)base;
  return status == 0 && size > 0 && sp >= start && sp - start < size &&
         top == start + size;
}
