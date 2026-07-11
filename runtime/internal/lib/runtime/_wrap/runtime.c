#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif

#include <stdint.h>
#include <pthread.h>
#include <unistd.h>

int llgo_maxprocs()
{
#ifdef _SC_NPROCESSORS_ONLN
    return (int)sysconf(_SC_NPROCESSORS_ONLN);
#else
    return 1;
#endif
}

void llgo_clobber_pointer_regs(uintptr_t a0, uintptr_t a1, uintptr_t a2, uintptr_t a3,
    uintptr_t a4, uintptr_t a5, uintptr_t a6, uintptr_t a7)
{
    volatile uintptr_t sink = a0 | a1 | a2 | a3 | a4 | a5 | a6 | a7;
    (void)sink;
}

void llgo_clear_stack_ptr(uintptr_t target)
{
    if (target == 0) {
        return;
    }

    volatile uintptr_t marker = 0;
    uintptr_t *cur = 0;
    uintptr_t *end = 0;

#if defined(__APPLE__)
    void *stackaddr = pthread_get_stackaddr_np(pthread_self());
    size_t stacksize = pthread_get_stacksize_np(pthread_self());
    if (stackaddr != 0 && stacksize != 0) {
        uintptr_t *mark = (uintptr_t *)&marker;
        uintptr_t *lo = (uintptr_t *)((char *)stackaddr - stacksize);
        uintptr_t *hi = (uintptr_t *)stackaddr;
        if (mark >= lo && mark < hi) {
            cur = lo;
            end = hi;
        } else {
            lo = (uintptr_t *)stackaddr;
            hi = (uintptr_t *)((char *)stackaddr + stacksize);
            if (mark >= lo && mark < hi) {
                cur = lo;
                end = hi;
            }
        }
    }
#elif defined(__linux__)
    pthread_attr_t attr;
    void *stackaddr = 0;
    size_t stacksize = 0;
    if (pthread_getattr_np(pthread_self(), &attr) == 0) {
        if (pthread_attr_getstack(&attr, &stackaddr, &stacksize) == 0) {
            cur = (uintptr_t *)stackaddr;
            end = (uintptr_t *)((char *)stackaddr + stacksize);
        }
        pthread_attr_destroy(&attr);
    }
#endif

    if (cur == 0 || end == 0 || end <= cur) {
        return;
    }
    if ((uintptr_t *)target >= cur && (uintptr_t *)target < end) {
        return;
    }
    for (; cur < end; cur++) {
        if (*cur == target) {
            *cur = 0;
        }
    }
}
