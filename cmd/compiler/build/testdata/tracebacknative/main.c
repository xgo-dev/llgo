/* White-box transport test, without Go's own signal handlers. */
#include "traceback_unix.c"
#include <assert.h>

static unsigned ready, stop, blocked, request_block, faults;
static void fault_handler(int sig)
{
    if (!llgo_traceback_fault_recover()) _exit(128+sig);
}
static size_t bad_unwind(void *ctx, uintptr_t *pcs, size_t cap, uintptr_t *fp)
{
    (void)ctx; (void)pcs; (void)cap; (void)fp;
    __atomic_add_fetch(&faults, 1, __ATOMIC_RELAXED);
    return *(volatile unsigned *)(uintptr_t)1;
}
static void *worker(void *node)
{
    llgo_traceback_attach(node);
    __atomic_store_n(&ready, 1, __ATOMIC_RELEASE);
    while (!__atomic_load_n(&stop, __ATOMIC_ACQUIRE)) {
        unsigned want = __atomic_load_n(&request_block, __ATOMIC_ACQUIRE);
        if (want != __atomic_load_n(&blocked, __ATOMIC_ACQUIRE)) {
            sigset_t mask;
            sigemptyset(&mask); sigaddset(&mask, SIGURG);
            pthread_sigmask(want ? SIG_BLOCK : SIG_UNBLOCK, &mask, 0);
            __atomic_store_n(&blocked, want, __ATOMIC_RELEASE);
        }
        sched_yield();
    }
    llgo_traceback_unregister(node);
    return 0;
}
int main(void)
{
    struct sigaction action;
    memset(&action, 0, sizeof(action));
    action.sa_handler = fault_handler;
    sigemptyset(&action.sa_mask);
    assert(sigaction(SIGSEGV, &action, 0) == 0);
    assert(sigaction(SIGBUS, &action, 0) == 0);
    llgo_traceback_set_unwinder(bad_unwind, 0);
    uintptr_t creators[LLGO_TRACEBACK_CREATOR_MAX + 1];
    for (unsigned i = 0; i <= LLGO_TRACEBACK_CREATOR_MAX; ++i) creators[i] = 0x1234 + i;
    void *node = llgo_traceback_register(2, 1, creators, LLGO_TRACEBACK_CREATOR_MAX + 1);
    /* The parent stack can disappear immediately after registration. */
    memset(creators, 0, sizeof(creators));
    uintptr_t creator_count;
    uintptr_t *retained = llgo_traceback_thread_creator(node, &creator_count);
    assert(creator_count == LLGO_TRACEBACK_CREATOR_MAX);
    for (unsigned i = 0; i < creator_count; ++i) assert(retained[i] == 0x1234 + i);
    assert(node);
    pthread_t thread;
    assert(pthread_create(&thread, 0, worker, node) == 0);
    while (!__atomic_load_n(&ready, __ATOMIC_ACQUIRE)) sched_yield();
    for (unsigned i=0; i<32; ++i) {
        llgo_traceback_snapshot *s = llgo_traceback_capture(1);
        assert(s && s->id == 2 && s->parent == 1 && s->count > 0 && !s->next);
        assert(s->creator_count == LLGO_TRACEBACK_CREATOR_MAX);
        for (unsigned j = 0; j < s->creator_count; ++j) assert(s->creator_pcs[j] == 0x1234 + j);
        llgo_traceback_free(s);
    }
    assert(__atomic_load_n(&faults, __ATOMIC_ACQUIRE) == 32);
    __atomic_store_n(&request_block, 1, __ATOMIC_RELEASE);
    while (!__atomic_load_n(&blocked, __ATOMIC_ACQUIRE)) sched_yield();
    llgo_traceback_snapshot *s = llgo_traceback_capture(1);
    assert(s && s->count == 0); /* timed out without freeing a live buffer */
    llgo_traceback_free(s);
    __atomic_store_n(&request_block, 0, __ATOMIC_RELEASE);
    while (__atomic_load_n(&blocked, __ATOMIC_ACQUIRE)) sched_yield();
    s = llgo_traceback_capture(1);
    assert(s && s->count > 0);
    llgo_traceback_free(s);
    __atomic_store_n(&stop, 1, __ATOMIC_RELEASE);
    assert(pthread_join(thread, 0) == 0);
    assert(llgo_traceback_capture(1) == 0);
    return 0;
}
