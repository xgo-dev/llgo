#include <limits.h>
#include <stdint.h>
#include <time.h>

#if defined(__EMSCRIPTEN__)
#include <emscripten.h>
#elif defined(__wasi__)
#include <poll.h>
#else
#error "unsupported WebAssembly host"
#endif

#define LLGO_NANOSECONDS_PER_MILLISECOND UINT64_C(1000000)

int64_t llgo_wasm_event_now(void) {
	struct timespec now;
	if (clock_gettime(CLOCK_MONOTONIC, &now) != 0) {
		return 0;
	}
	return (int64_t)now.tv_sec * INT64_C(1000000000) + now.tv_nsec;
}

void llgo_wasm_event_wait(uint64_t nanoseconds) {
	uint64_t milliseconds = nanoseconds / LLGO_NANOSECONDS_PER_MILLISECOND;
	if (nanoseconds % LLGO_NANOSECONDS_PER_MILLISECOND != 0) {
		milliseconds++;
	}
#if defined(__EMSCRIPTEN__)
	if (milliseconds > UINT32_MAX) {
		milliseconds = UINT32_MAX;
	}
	/*
	 * emscripten_sleep returns to JavaScript and resumes this Asyncify context
	 * later. The host callback does not call back into Go synchronously.
	 */
	emscripten_sleep((unsigned int)milliseconds);
#else
	if (milliseconds > INT_MAX) {
		milliseconds = INT_MAX;
	}
	(void)poll(NULL, 0, (int)milliseconds);
#endif
}
