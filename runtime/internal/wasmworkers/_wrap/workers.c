#include <emscripten/atomic.h>
#include <emscripten/eventloop.h>
#include <emscripten.h>
#include <math.h>
#include <stdint.h>

#ifndef LLGO_WASM_WORKERS
#define LLGO_WASM_WORKERS 1
#endif

static _Thread_local void *llgo_wasm_current_worker;

extern void llgo_wasm_worker_resume(void *worker);

EM_JS(void, llgo_wasm_worker_install_host_wake, (uint32_t *address), {
  const index = Math.trunc(Number(address) / 4);
  const state = Module['llgoWasmHostWait'] ||
      (Module['llgoWasmHostWait'] = {});
  state.workerWakeIndex = index;
  state.wake = function() {
    Atomics.add(HEAPU32, index, 1);
    Atomics.notify(HEAP32, index);
  };
});

EM_JS(void, llgo_wasm_worker_clear_host_wake, (uint32_t *address), {
  const state = Module['llgoWasmHostWait'];
  const index = Math.trunc(Number(address) / 4);
  if (state !== undefined && state.workerWakeIndex === index) {
    delete state.workerWakeIndex;
    delete state.wake;
  }
});

static void llgo_wasm_worker_wait_finished(
    int32_t *address, uint32_t expected,
    ATOMICS_WAIT_RESULT_T result, void *worker) {
  (void)expected;
  (void)result;
  llgo_wasm_worker_clear_host_wake((uint32_t *)address);
  // Let the atomic-wait callback return before entering the Go scheduler.
  // Emscripten fibers use Asyncify underneath; switching fibers while this
  // host callback is still active can make its callback frame part of the
  // saved scheduler continuation.
  emscripten_async_call(llgo_wasm_worker_resume, worker, 0);
}

void llgo_wasm_worker_resume_soon(void *worker) {
  emscripten_async_call(llgo_wasm_worker_resume, worker, 0);
}

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
  // PROXY_TO_PTHREAD keeps every LLGo scheduler worker off the browser main
  // thread. Use the raw Wasm wait instruction here instead of
  // emscripten_futex_wait: the latter may call emscripten_yield on a browser
  // main thread, which makes Binaryen conservatively Asyncify allocator and
  // STW lock paths that never suspend a fiber in LLGo's worker model.
  return emscripten_atomic_wait_u32(
      address, expected, timeout_nanoseconds);
}

int llgo_wasm_worker_arm_wait(
    uint32_t *address, uint32_t expected, int64_t timeout_nanoseconds,
    void *worker) {
  double timeout_milliseconds = INFINITY;
  if (timeout_nanoseconds >= 0) {
    timeout_milliseconds = (double)timeout_nanoseconds / 1000000.0;
  }
  llgo_wasm_worker_install_host_wake(address);
  ATOMICS_WAIT_TOKEN_T token = emscripten_atomic_wait_async(
      (volatile void *)address, expected, llgo_wasm_worker_wait_finished,
      worker, timeout_milliseconds);
  if (!EMSCRIPTEN_IS_VALID_WAIT_TOKEN(token)) {
    llgo_wasm_worker_clear_host_wake(address);
    return 0;
  }
  return 1;
}

__attribute__((noreturn)) void llgo_wasm_worker_suspend(void) {
  // This is used only to retire the pthread's initial native entry while
  // keeping its JavaScript worker alive. Async-wait resume callbacks return
  // normally, so their function epilogues restore the C stack themselves.
  emscripten_unwind_to_js_event_loop();
  __builtin_unreachable();
}

int llgo_wasm_worker_wake(uint32_t *address) {
  return (int)emscripten_atomic_notify(
      address, EMSCRIPTEN_NOTIFY_ALL_WAITERS);
}
