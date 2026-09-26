#define _GNU_SOURCE
#include <stddef.h>
#include <stdint.h>

#if defined(__EMSCRIPTEN__)
#include <emscripten/heap.h>
#include <emscripten/stack.h>
#elif defined(__wasi__) && defined(_REENTRANT)
#include <errno.h>
#include <pthread.h>
extern unsigned char __stack_high;
#else
extern unsigned char __stack_high;
#endif

extern unsigned char __data_end;
extern unsigned char __global_base;
extern unsigned char __heap_base;

#define LLGO_WASM_PAGE_SIZE 65536

uintptr_t llgo_gc_globals_start(void) {
	return (uintptr_t)&__global_base;
}

uintptr_t llgo_gc_globals_end(void) {
	return (uintptr_t)&__data_end;
}

uintptr_t llgo_gc_heap_base(void) {
	return (uintptr_t)&__heap_base;
}

uintptr_t llgo_gc_stack_top(void) {
#if defined(__EMSCRIPTEN__)
	return (uintptr_t)emscripten_stack_get_base();
#elif defined(__wasi__) && defined(_REENTRANT)
	// __stack_high belongs to the main thread. Every WASI pthread has its own
	// stack, so the collector must obtain the current thread's upper bound.
	pthread_attr_t attr;
	void *base = NULL;
	size_t size = 0;
	int status = pthread_getattr_np(pthread_self(), &attr);
	// wasi-libc has no pthread stack descriptor for the process main thread.
	if (status == ENOSYS)
		return (uintptr_t)&__stack_high;
	if (status != 0)
		__builtin_trap();
	status = pthread_attr_getstack(&attr, &base, &size);
	pthread_attr_destroy(&attr);
	if (status != 0 || base == NULL || size == 0 ||
	    size > UINTPTR_MAX - (uintptr_t)base)
		__builtin_trap();
	return (uintptr_t)base + size;
#else
	return (uintptr_t)&__stack_high;
#endif
}

uintptr_t llgo_gc_memory_size(void) {
	return (uintptr_t)__builtin_wasm_memory_size(0) * LLGO_WASM_PAGE_SIZE;
}

int llgo_gc_grow_memory(uintptr_t required) {
#if defined(__EMSCRIPTEN__)
	return emscripten_resize_heap(required);
#else
	uintptr_t current = llgo_gc_memory_size();
	if (required <= current) {
		return 1;
	}
	uintptr_t pages = (required - current + LLGO_WASM_PAGE_SIZE - 1) /
		LLGO_WASM_PAGE_SIZE;
	return __builtin_wasm_memory_grow(0, pages) != (size_t)-1;
#endif
}
