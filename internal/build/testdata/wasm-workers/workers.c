#include <emscripten/threading.h>
#include <pthread.h>
#include <stdatomic.h>
#include <stdint.h>

#ifndef LLGO_WASM_WORKERS
#define LLGO_WASM_WORKERS 1
#endif

static _Atomic uint32_t llgo_test_parallel_workers;
static uintptr_t llgo_test_parallel_threads[2];

int32_t llgo_test_parallel_worker_barrier(void) {
  uint32_t slot = atomic_fetch_add_explicit(
      &llgo_test_parallel_workers, 1, memory_order_acq_rel);
  if (slot >= 2) {
    return -1;
  }
  llgo_test_parallel_threads[slot] = (uintptr_t)pthread_self();
  if (slot == 0) {
    uint32_t attempts = 0;
    while (atomic_load_explicit(
               &llgo_test_parallel_workers, memory_order_acquire) != 2) {
      if (attempts++ == 10) {
        return -1;
      }
      emscripten_futex_wait(
          (volatile void *)&llgo_test_parallel_workers, 1, 100.0);
    }
  } else {
    emscripten_futex_wake(
        (volatile void *)&llgo_test_parallel_workers, 1);
  }
  return (int32_t)slot;
}

uintptr_t llgo_test_parallel_worker_thread(int32_t slot) {
  if (slot < 0 || slot >= 2) {
    return 0;
  }
  return llgo_test_parallel_threads[slot];
}

uintptr_t llgo_test_current_worker_thread(void) {
  return (uintptr_t)pthread_self();
}

int32_t llgo_test_worker_count(void) {
  return LLGO_WASM_WORKERS;
}
