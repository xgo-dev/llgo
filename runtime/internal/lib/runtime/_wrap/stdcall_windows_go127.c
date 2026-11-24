#include <stdint.h>

struct llgo_windows_syscall_info {
    uintptr_t fn, n, args, r1, r2, err;
};

struct llgo_windows_stdcall_info {
    uintptr_t fn, n, args, r1, r2;
};

_Static_assert(sizeof(struct llgo_windows_stdcall_info) == 5 * sizeof(uintptr_t), "SDK five-word layout");
_Static_assert(sizeof(struct llgo_windows_syscall_info) == 6 * sizeof(uintptr_t), "native six-word layout");

extern void llgo_windows_syscall(struct llgo_windows_syscall_info *);

/* The Go1.27 SDK block has five words, unlike the existing six-word thunk.
 * Never cast between them or overwrite storage after the SDK's R2 field. */
uint32_t llgo_windows_stdcall_go127(struct llgo_windows_stdcall_info *info)
{
    if (info->n > (UINTPTR_MAX - 15) / sizeof(uintptr_t))
        __builtin_trap(); /* Cannot represent a valid outgoing argument object. */
    struct llgo_windows_syscall_info call = {
        info->fn, info->n, info->args, 0, 0, 0
    };
    llgo_windows_syscall(&call);
    info->r1 = call.r1;
#if !defined(__aarch64__)
    info->r2 = call.r2;
#endif
    /* ARM64's original entry leaves R2 untouched; last-error is uint32. */
    return (uint32_t)call.err;
}

uintptr_t llgo_windows_stdcall_go127_addr(void)
{
    return (uintptr_t)llgo_windows_stdcall_go127;
}
