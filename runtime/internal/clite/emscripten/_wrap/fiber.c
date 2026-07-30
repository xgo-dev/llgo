#include <emscripten/fiber.h>

_Static_assert(
    sizeof(emscripten_fiber_t) == 8 * sizeof(void *),
    "LLGo Fiber storage does not match emscripten_fiber_t");
