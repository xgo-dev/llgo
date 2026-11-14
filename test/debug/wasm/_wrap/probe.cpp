#include <stdint.h>
#include <stddef.h>

extern "C" __attribute__((noinline)) int32_t llgo_debug_cpp_probe(int32_t input) {
  volatile int32_t cpp_local = input + 7;
  return cpp_local;
}

#ifdef __EMSCRIPTEN__
#include <emscripten/heap.h>
#endif

extern "C" int32_t llgo_debug_grow_memory() {
#ifdef __EMSCRIPTEN__
  const size_t before = emscripten_get_heap_size();
  return emscripten_resize_heap(before + 65536) && emscripten_get_heap_size() > before;
#else
  return __builtin_wasm_memory_grow(0, 1) != (size_t)-1;
#endif
}
