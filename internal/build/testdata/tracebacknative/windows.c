/* Exercise context recovery and copied-stack unwinding without Go handlers. */
#include "traceback_windows.c"
#include <stdio.h>
#define assert(test) do { if (!(test)) { fprintf(stderr, "failed at %d: %s\n", __LINE__, #test); exit(1); } } while (0)

typedef DWORD (WINAPI *thread_start)(void *);
__declspec(dllimport) HANDLE WINAPI CreateThread(void *, size_t, thread_start, void *, DWORD, DWORD *);
__declspec(dllimport) DWORD WINAPI WaitForSingleObject(HANDLE, DWORD);
__declspec(dllimport) int WINAPI SwitchToThread(void);
typedef struct { void *record, *context; } exception_pointers;
__declspec(dllimport) void *WINAPI AddVectoredExceptionHandler(unsigned long,
    long (WINAPI *)(exception_pointers *));

static long WINAPI fault_handler(exception_pointers *exception)
{
    (void)exception;
    return llgo_traceback_windows_fault_recover() ? -1 : 0;
}

static int guarded_fault(void)
{
    if (capture_point() != 0) return 1;
    recovery_armed = 1;
    *(volatile unsigned *)(uintptr_t)1 = 0;
    recovery_armed = 0;
    return 0;
}

static unsigned ready, stop;
static DWORD WINAPI worker(void *node)
{
    llgo_traceback_attach(node);
    __atomic_store_n(&ready, 1, __ATOMIC_RELEASE);
    while (!__atomic_load_n(&stop, __ATOMIC_ACQUIRE)) SwitchToThread();
    llgo_traceback_unregister(node);
    return 0;
}

int main(void)
{
    assert(AddVectoredExceptionHandler(1, fault_handler));
    for (unsigned i=0; i<32; ++i) assert(guarded_fault());
    uintptr_t creators[LLGO_TRACEBACK_CREATOR_MAX + 1];
    for (unsigned i = 0; i <= LLGO_TRACEBACK_CREATOR_MAX; ++i) creators[i] = 0x1234 + i;
    void *node = llgo_traceback_register(2, 1, 42, creators, LLGO_TRACEBACK_CREATOR_MAX + 1);
    /* The parent stack can disappear immediately after registration. */
    memset(creators, 0, sizeof(creators));
    uintptr_t creator_count;
    uintptr_t *retained = llgo_traceback_thread_creator(node, &creator_count);
    assert(creator_count == LLGO_TRACEBACK_CREATOR_MAX);
    for (unsigned i = 0; i < creator_count; ++i) assert(retained[i] == 0x1234 + i);
    assert(node);
    HANDLE thread = CreateThread(0, 0, worker, node, 0, 0);
    assert(thread);
    while (!__atomic_load_n(&ready, __ATOMIC_ACQUIRE)) SwitchToThread();
    for (unsigned i=0; i<32; ++i) {
        llgo_traceback_snapshot *s = llgo_traceback_capture(1);
        assert(s && s->id == 2 && s->parent == 1 && s->created == 42 && s->count > 1);
        assert(s->creator_count == LLGO_TRACEBACK_CREATOR_MAX);
        for (unsigned j = 0; j < s->creator_count; ++j) assert(s->creator_pcs[j] == 0x1234 + j);
        llgo_traceback_free(s);
    }
    __atomic_store_n(&stop, 1, __ATOMIC_RELEASE);
    assert(WaitForSingleObject(thread, 5000) == 0);
    CloseHandle(thread);
    assert(llgo_traceback_capture(1) == 0);
    return 0;
}
