#define _GNU_SOURCE
#include <stddef.h>
#include <stdint.h>

#if defined(__EMSCRIPTEN__)
#include <emscripten/heap.h>
#include <emscripten/stack.h>
#elif defined(__wasi__) && defined(_REENTRANT)
#include <errno.h>
#include <pthread.h>
#include <stdlib.h>
extern unsigned char __stack_high;
#else
extern unsigned char __stack_high;
#endif

extern unsigned char __data_end;
extern unsigned char __global_base;
extern unsigned char __heap_base;

#define LLGO_WASM_PAGE_SIZE 65536

#if defined(__wasi__) && defined(_REENTRANT)
// wasi-libc allocates pthread stacks and TLS from its own heap. Reserve a
// disjoint region before tinygogc starts, rather than treating all remaining
// linear memory as a Go heap and corrupting later libc allocations. Further
// disjoint regions can be allocated as the Go heap grows.
#define LLGO_WASI_GC_ARENA_SIZE (32u << 20)
static uintptr_t llgo_wasi_gc_arena_start;
static uintptr_t llgo_wasi_gc_arena_end;

static void llgo_wasi_gc_init_arena(void) {
  if (llgo_wasi_gc_arena_start != 0)
    return;
  void *arena = malloc(LLGO_WASI_GC_ARENA_SIZE);
  if (arena == NULL)
    __builtin_trap();
  llgo_wasi_gc_arena_start = (uintptr_t)arena;
  llgo_wasi_gc_arena_end = (uintptr_t)arena + LLGO_WASI_GC_ARENA_SIZE;
}

uintptr_t llgo_gc_new_arena(uintptr_t size) {
  return (uintptr_t)malloc(size);
}
#endif

uintptr_t llgo_gc_globals_start(void) {
	return (uintptr_t)&__global_base;
}

uintptr_t llgo_gc_globals_end(void) {
	return (uintptr_t)&__data_end;
}

uintptr_t llgo_gc_heap_base(void) {
#if defined(__wasi__) && defined(_REENTRANT)
  llgo_wasi_gc_init_arena();
  return llgo_wasi_gc_arena_start;
#else
	return (uintptr_t)&__heap_base;
#endif
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
	// Older wasi-libc may not expose the process main thread's stack.
	if (status == ENOSYS)
		return (uintptr_t)&__stack_high;
	if (status != 0)
		__builtin_trap();
	status = pthread_attr_getstack(&attr, &base, &size);
	pthread_attr_destroy(&attr);
	// A stack-first module legitimately gives the main thread base address 0.
	if (status != 0 || size == 0 ||
	    size > UINTPTR_MAX - (uintptr_t)base)
		__builtin_trap();
	return (uintptr_t)base + size;
#else
	return (uintptr_t)&__stack_high;
#endif
}

uintptr_t llgo_gc_memory_size(void) {
#if defined(__wasi__) && defined(_REENTRANT)
  llgo_wasi_gc_init_arena();
  return llgo_wasi_gc_arena_end;
#else
	return (uintptr_t)__builtin_wasm_memory_size(0) * LLGO_WASM_PAGE_SIZE;
#endif
}

int llgo_gc_grow_memory(uintptr_t required) {
#if defined(__EMSCRIPTEN__)
	return emscripten_resize_heap(required);
#elif defined(__wasi__) && defined(_REENTRANT)
  return required <= llgo_gc_memory_size();
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
