#include <emscripten/emscripten.h>
#include <emscripten/fiber.h>

_Static_assert(
    sizeof(emscripten_fiber_t) == 8 * sizeof(void *),
    "LLGo Fiber storage does not match emscripten_fiber_t");

EM_JS(int, llgo_emscripten_asyncify_state_js, (), {
    return Asyncify.state;
});

int llgo_emscripten_fiber_rewinding(void) {
    return llgo_emscripten_asyncify_state_js() == 2;
}

int llgo_emscripten_asyncify_state(void) {
    return llgo_emscripten_asyncify_state_js();
}
