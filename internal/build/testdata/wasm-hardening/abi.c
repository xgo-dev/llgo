#include <stdint.h>

#if defined(__EMSCRIPTEN__) && defined(__EMSCRIPTEN_PTHREADS__)
#include <emscripten/emscripten.h>
#include <stdatomic.h>

static _Atomic int llgo_hardening_entered;
#endif

uint64_t llgo_hardening_hold_payload(const uint64_t *value) {
#if defined(__EMSCRIPTEN__) && defined(__EMSCRIPTEN_PTHREADS__)
  atomic_store_explicit(&llgo_hardening_entered, 1, memory_order_release);
  double deadline = emscripten_get_now() + 25.0;
  while (emscripten_get_now() < deadline) {
  }
#endif
  return *value;
}

int llgo_hardening_hold_entered(void) {
#if defined(__EMSCRIPTEN__) && defined(__EMSCRIPTEN_PTHREADS__)
  return atomic_load_explicit(&llgo_hardening_entered, memory_order_acquire);
#else
  return 0;
#endif
}
