#include <emscripten/threading.h>
#include <limits.h>
#include <math.h>
#include <stdint.h>

#ifndef LLGO_WASM_WORKERS
#define LLGO_WASM_WORKERS 1
#endif

static _Thread_local void *llgo_wasm_current_worker;

int llgo_wasm_worker_count(void) {
  return LLGO_WASM_WORKERS;
}

void *llgo_wasm_worker_current(void) {
  return llgo_wasm_current_worker;
}

void llgo_wasm_worker_set_current(void *worker) {
  llgo_wasm_current_worker = worker;
}

int llgo_wasm_worker_wait(
    uint32_t *address, uint32_t expected, int64_t timeout_nanoseconds) {
  double timeout_milliseconds = INFINITY;
  if (timeout_nanoseconds >= 0) {
    timeout_milliseconds = (double)timeout_nanoseconds / 1000000.0;
  }
  return emscripten_futex_wait(
      (volatile void *)address, expected, timeout_milliseconds);
}

int llgo_wasm_worker_wake(uint32_t *address) {
  return emscripten_futex_wake((volatile void *)address, INT_MAX);
}
