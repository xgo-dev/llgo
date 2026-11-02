#define _GNU_SOURCE 1
#include <stdint.h>

/* Exclude module globals from WaitGroup association. This is not a GC heap
 * membership test: dynamic objects may be on stacks, in C allocations, or in
 * later disjoint WASI GC arenas. __heap_base excludes Wasm module data without
 * coupling this predicate to an allocator or to the first GC arena. */
#if defined(__wasm__)
extern unsigned char __heap_base;
_Bool llgo_synctest_can_associate(void *p) {
    return (uintptr_t)p >= (uintptr_t)&__heap_base;
}
#elif defined(_WIN32)
#include <windows.h>
_Bool llgo_synctest_can_associate(void *p) {
    HMODULE module;
    return p && !GetModuleHandleExW(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS |
        GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, (LPCWSTR)p, &module);
}
#elif defined(__linux__) || defined(__APPLE__)
#include <dlfcn.h>
_Bool llgo_synctest_can_associate(void *p) {
    Dl_info info;
    return p && !dladdr(p, &info);
}
#else
_Bool llgo_synctest_can_associate(void *p) { return p != 0; }
#endif
