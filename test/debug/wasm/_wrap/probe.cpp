#include <stdint.h>

extern "C" __attribute__((noinline)) int32_t llgo_debug_cpp_probe(int32_t input) {
  int32_t cpp_local = input + 7;
  return cpp_local;
}

#include <emscripten/heap.h>

extern "C" int32_t llgo_debug_grow_memory() {
  const size_t before = emscripten_get_heap_size();
  return emscripten_resize_heap(before + 65536) && emscripten_get_heap_size() > before;
}
