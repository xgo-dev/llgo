#if defined(__linux__)
#define UNW_LOCAL_ONLY
#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif
#include <features.h>
#endif

#include <dlfcn.h>
#include <errno.h>
#include <libunwind.h>

void *llgo_address() {
    return __builtin_return_address(0);
}

int llgo_addrinfo(void *addr, Dl_info *info) {
    int saved_errno = errno;
    int ret = dladdr(addr, info);
    errno = saved_errno;
    return ret;
}

void llgo_stacktrace(int skip, void *ctx, int (*fn)(void *ctx, void *pc, void *offset, void *sp, char *name)) {
    int saved_errno = errno;
    unw_cursor_t cursor;
    unw_context_t context;
    unw_word_t offset, pc, sp;
    char fname[256];
    unw_getcontext(&context);
    unw_init_local(&cursor, &context);
    int depth = 0;
    while (unw_step(&cursor) > 0) {
        if (depth < skip) {
            depth++;
            continue;
        }
        if (unw_get_reg(&cursor, UNW_REG_IP, &pc) == 0) {
            fname[0] = 0;
            offset = 0;
            if (unw_get_proc_name(&cursor, fname, sizeof(fname), &offset) == 0) {
                fname[sizeof(fname) - 1] = 0;
            }
            unw_get_reg(&cursor, UNW_REG_SP, &sp);
            if (fn(ctx, (void*)pc, (void*)offset, (void*)sp, fname) == 0) {
                errno = saved_errno;
                return;
            }
        }
    }
    errno = saved_errno;
}
