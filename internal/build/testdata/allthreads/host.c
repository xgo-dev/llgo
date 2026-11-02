#include <assert.h>
#include <dlfcn.h>
#include <pthread.h>
#include <sched.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>

typedef int32_t (*check_func)(void);
static _Atomic(check_func) check;
static atomic_int ready, adopted, stop;
static pthread_t initial;

static void *foreign(void *raw)
{
    (void)raw;
    atomic_fetch_add(&ready, 1);
    int entered = 0;
    while (!atomic_load(&stop)) {
        check_func fn = atomic_load(&check);
        if (fn && !entered) {
            assert(fn() == 0);
            entered = 1;
            atomic_fetch_add(&adopted, 1);
        }
        sched_yield();
    }
    return 0;
}

/* For archives this thread predates the LLGo runtime constructor. For shared
 * libraries main creates it before dlopen initializes the runtime. */
static void start_initial(void)
{
    assert(!pthread_create(&initial, 0, foreign, 0));
    while (atomic_load(&ready) != 1) sched_yield();
}

#ifndef SHARED
extern int32_t AllThreadsUnsupported(void);
__attribute__((constructor(101))) static void before_runtime(void)
{
    start_initial();
}
#endif

int main(int argc, char **argv)
{
    (void)argc;
    (void)argv;
#ifdef SHARED
    assert(argc == 2);
    start_initial();
    void *library = dlopen(argv[1], RTLD_NOW | RTLD_LOCAL);
    if (!library) { fprintf(stderr, "%s\n", dlerror()); return 1; }
    check_func fn = (check_func)dlsym(library, "AllThreadsUnsupported");
#else
    check_func fn = AllThreadsUnsupported;
#endif
    assert(fn && fn() == 0);
    atomic_store(&check, fn);
    while (atomic_load(&adopted) != 1) sched_yield();
    assert(fn() == 0);
    pthread_t later;
    assert(!pthread_create(&later, 0, foreign, 0));
    while (atomic_load(&adopted) != 2) sched_yield();
    assert(fn() == 0);
    atomic_store(&stop, 1);
    assert(!pthread_join(later, 0));
    assert(!pthread_join(initial, 0));
    assert(fn() == 0);
    /* Unloading an LLGo runtime with TLS destructors is outside this test. */
    puts("allthreads-library-ok");
    return 0;
}
