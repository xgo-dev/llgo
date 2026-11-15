#include <stdint.h>

static volatile intptr_t observation;

__attribute__((noinline)) intptr_t llgo_debug_inline_opaque(intptr_t value) {
    observation = value;
    return value;
}
