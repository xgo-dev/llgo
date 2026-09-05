#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#if defined(__EMSCRIPTEN__)
#include <emscripten/heap.h>
#endif

int llgo_test_gc_aligned_alloc(void) {
#if defined(__EMSCRIPTEN__)
	void *ptr = emscripten_builtin_memalign(65536, 257);
	if (ptr == NULL || (uintptr_t)ptr % 65536 != 0) {
		return 0;
	}
	unsigned char *bytes = ptr;
	bytes[0] = 0x5a;
	bytes[256] = 0xa5;
	if (bytes[0] != 0x5a || bytes[256] != 0xa5) {
		return 0;
	}
	emscripten_builtin_free(ptr);

	ptr = NULL;
	if (posix_memalign(&ptr, 65536, 257) != 0 || ptr == NULL ||
		(uintptr_t)ptr % 65536 != 0) {
		return 0;
	}
	free(ptr);
#endif
	return 1;
}
