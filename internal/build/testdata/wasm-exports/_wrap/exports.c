#include <emscripten.h>
#include <stdint.h>

extern int32_t llgo_go_add(int32_t, int32_t);

EMSCRIPTEN_KEEPALIVE int32_t llgo_public_add(int32_t a, int32_t b) {
  return llgo_go_add(a, b);
}
