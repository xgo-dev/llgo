#include <stddef.h>
#include <stdint.h>
#include <stdlib.h>

#if defined(__EMSCRIPTEN__)
#include <emscripten/heap.h>
#endif

extern unsigned char __global_base;

static volatile unsigned char c_root_storage[16] __attribute__((aligned(8)));

static void **llgo_test_gc_c_root_slot(void) {
	uintptr_t base = (uintptr_t)&__global_base;
	uintptr_t offset = sizeof(void *) == 4 && (base & 7) == 0 ? 4 : 0;
	return (void **)((uintptr_t)c_root_storage + offset);
}

void llgo_test_gc_set_c_root(void) {
	uint64_t *value = malloc(sizeof(*value));
	if (value == NULL) {
		abort();
	}
	*value = UINT64_C(0x123456789abcdef0);
	*llgo_test_gc_c_root_slot() = value;
}

uint64_t llgo_test_gc_read_c_root(void) {
	void *pointer = *llgo_test_gc_c_root_slot();
	return pointer == NULL ? 0 : *(uint64_t *)pointer;
}

void llgo_test_gc_clear_c_root(void) {
	*llgo_test_gc_c_root_slot() = NULL;
}

uint32_t llgo_test_gc_c_root_word_offset(void) {
	return (uint32_t)(((uintptr_t)llgo_test_gc_c_root_slot() -
		(uintptr_t)&__global_base) & 7);
}

uint32_t llgo_test_gc_c_pointer_size(void) {
	return sizeof(void *);
}

int llgo_test_gc_realloc(void) {
	unsigned char *from_zero = realloc(malloc(0), 8);
	if (from_zero == NULL) {
		return 0;
	}
	from_zero[0] = 0x5a;
	if (from_zero[0] != 0x5a) {
		return 0;
	}
	free(from_zero);

	unsigned char *bytes = malloc(17);
	if (bytes == NULL) {
		return 0;
	}
	for (unsigned i = 0; i < 17; i++) {
		bytes[i] = (unsigned char)(i + 1);
	}
	unsigned char *grown = realloc(bytes, 257);
	if (grown == NULL) {
		return 0;
	}
	for (unsigned i = 0; i < 17; i++) {
		if (grown[i] != (unsigned char)(i + 1)) {
			return 0;
		}
	}
	unsigned char *shrunk = realloc(grown, 8);
	if (shrunk == NULL) {
		return 0;
	}
	for (unsigned i = 0; i < 8; i++) {
		if (shrunk[i] != (unsigned char)(i + 1)) {
			return 0;
		}
	}
	free(shrunk);
	return 1;
}

__attribute__((noinline)) void llgo_test_gc_clobber_c_stack(unsigned depth) {
	volatile uintptr_t words[64];
	for (unsigned i = 0; i < sizeof(words) / sizeof(words[0]); i++) {
		words[i] = 0;
	}
	if (depth != 0) {
		llgo_test_gc_clobber_c_stack(depth - 1);
	}
}

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
