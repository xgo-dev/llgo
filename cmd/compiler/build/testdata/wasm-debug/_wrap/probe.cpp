#include <stdint.h>

extern "C" __attribute__((noinline)) int32_t llgo_debug_cpp_probe(int32_t input) {
  int32_t cpp_local = input + 7;
  return cpp_local;
}
