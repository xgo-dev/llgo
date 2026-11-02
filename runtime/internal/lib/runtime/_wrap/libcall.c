#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE 1
#endif
#include <stdint.h>

#if (defined(__linux__) || defined(__APPLE__)) && !defined(__wasm__)
int32_t llgo_libc_call(void *fn, void *arg)
{
    return ((int32_t (*)(void *))fn)(arg);
}
#endif

#if defined(__linux__) && !defined(__wasm__)
#include <errno.h>
#include <grp.h>
#include <sys/types.h>
#include <unistd.h>

/* Go's runtime/cgocall.go argset ABI. The wrapper writes errno into retval;
 * the direct libc-call return value is deliberately a different contract. */
typedef struct {
    uintptr_t *args;
    uintptr_t retval;
} llgo_argset;

extern void llgo_allthreads_credential_lock(void);
extern void llgo_allthreads_credential_unlock(void);

void llgo_cgo_argset_call(void *fn, llgo_argset *arg)
{
    ((void (*)(llgo_argset *))fn)(arg);
}

#define LLGO_SETTER(name, expression) \
    void llgo_cgo_##name(llgo_argset *arg) { \
        llgo_allthreads_credential_lock(); \
        int result = (expression); \
        int saved = errno; \
        llgo_allthreads_credential_unlock(); \
        arg->retval = result == -1 ? (uintptr_t)saved : (uintptr_t)result; \
    }

/* Use libc's process-wide pthread/setxid protocol, never a raw syscall on the
 * calling thread alone. Foreign C threads are included by that libc protocol. */
LLGO_SETTER(setuid, setuid((uid_t)arg->args[0]))
LLGO_SETTER(seteuid, seteuid((uid_t)arg->args[0]))
LLGO_SETTER(setgid, setgid((gid_t)arg->args[0]))
LLGO_SETTER(setegid, setegid((gid_t)arg->args[0]))
LLGO_SETTER(setreuid, setreuid((uid_t)arg->args[0], (uid_t)arg->args[1]))
LLGO_SETTER(setregid, setregid((gid_t)arg->args[0], (gid_t)arg->args[1]))
LLGO_SETTER(setresuid, setresuid((uid_t)arg->args[0], (uid_t)arg->args[1], (uid_t)arg->args[2]))
LLGO_SETTER(setresgid, setresgid((gid_t)arg->args[0], (gid_t)arg->args[1], (gid_t)arg->args[2]))
LLGO_SETTER(setgroups, setgroups((size_t)arg->args[0], (const gid_t *)arg->args[1]))
#endif
