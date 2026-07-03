#if defined(__linux__)
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <features.h>
#endif

#include <dlfcn.h>
#include <errno.h>
#include <stdint.h>

void *llgo_address() {
    return __builtin_return_address(0);
}

int llgo_addrinfo(void *addr, Dl_info *info) {
    int saved_errno = errno;
    int ret = dladdr(addr, info);
    errno = saved_errno;
    return ret;
}

void *llgo_symbol(char *name) {
    int saved_errno = errno;
    void *ret = dlsym(RTLD_DEFAULT, name);
    errno = saved_errno;
    return ret;
}

void llgo_stacktrace(int skip, void *ctx, int (*fn)(void *ctx, void *pc, void *offset, void *sp, char *name)) {
    /* Frame-pointer chain walk. LLGo compiles every Go function with
     * "frame-pointer"="non-leaf", so [fp] is the previous frame pointer and
     * [fp+1] the return address on both arm64 and x86-64. This replaces the
     * libunwind cursor: no unwind tables, no -lunwind, and it keeps working
     * through any frame that maintains the chain (C code compiled with
     * frame pointers included). The walk stops at the first frame that
     * breaks chain discipline.
     *
     * The Go-side walker (runtime/internal/lib/runtime/unwind_llgo.go
     * fpCallers) implements the same discipline plus a text-range bound the
     * frame tables provide; keep the chain guards below in sync with it. */
    int saved_errno = errno;
    uintptr_t fp = (uintptr_t)__builtin_frame_address(0);
    int depth = 0;
    while (fp) {
        uintptr_t prev = *(uintptr_t *)fp;
        uintptr_t pc = *((uintptr_t *)fp + 1);
        if (pc < 4096)
            break;
        if (depth >= skip) {
            Dl_info info;
            const char *name = "";
            uintptr_t offset = 0;
            if (dladdr((void *)pc, &info) && info.dli_sname) {
                name = info.dli_sname;
                offset = pc - (uintptr_t)info.dli_saddr;
            }
            if (fn(ctx, (void *)pc, (void *)offset, (void *)fp, (char *)name) == 0)
                break;
        }
        depth++;
        if (prev <= fp || prev - fp > (uintptr_t)1 << 20 || (prev & (sizeof(uintptr_t) - 1)))
            break;
        fp = prev;
    }
    errno = saved_errno;
}
