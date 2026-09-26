/* The bridge deliberately has no Windows SDK header dependency, matching the
 * runtime's existing cross-compilable Windows exception shims. */
#include "traceback.h"
#include <stdlib.h>
#include <string.h>

#if defined(_WIN64)
#define WINAPI
#else
#define WINAPI __attribute__((stdcall))
#endif
typedef unsigned long DWORD;
typedef void *HANDLE;
__declspec(dllimport) void WINAPI AcquireSRWLockExclusive(void *);
__declspec(dllimport) void WINAPI ReleaseSRWLockExclusive(void *);
__declspec(dllimport) DWORD WINAPI GetCurrentThreadId(void);
__declspec(dllimport) HANDLE WINAPI OpenThread(DWORD, int, DWORD);
__declspec(dllimport) int WINAPI CloseHandle(HANDLE);
__declspec(dllimport) DWORD WINAPI SuspendThread(HANDLE);
__declspec(dllimport) DWORD WINAPI ResumeThread(HANDLE);
__declspec(dllimport) int WINAPI GetThreadContext(HANDLE, void *);
__declspec(dllimport) int WINAPI ReadProcessMemory(HANDLE, const void *, void *, size_t, size_t *);
__declspec(dllimport) void *WINAPI VirtualAlloc(void *, size_t, DWORD, DWORD);
__declspec(dllimport) int WINAPI VirtualFree(void *, size_t, DWORD);
__declspec(dllimport) size_t WINAPI VirtualQuery(void *, void *, size_t);
#if defined(_WIN64)
__declspec(dllimport) void *WINAPI RtlLookupFunctionEntry(uint64_t, uint64_t *, void *);
__declspec(dllimport) void *WINAPI RtlVirtualUnwind(DWORD, uint64_t, uint64_t, void *,
                                                  void *, void **, uint64_t *, void *);
#endif

typedef struct thread_node {
    struct thread_node *next, *prev;
    uint64_t id, parent;
    uintptr_t created;
    HANDLE thread;
    DWORD thread_id;
    uint32_t state;
} thread_node;
static void *registry_lock;
static thread_node *threads;
static _Thread_local thread_node *current;
static _Thread_local uintptr_t *fault_buffer;

/* Preserve the old stack-copy limit independently of the PC-buffer limit. */
#define LLGO_TRACEBACK_COPY_MAX (1u << 23)

void llgo_traceback_set_unwinder(llgo_traceback_unwinder walk,
                                llgo_traceback_thread_init init)
{
    (void)walk; (void)init;
}

uintptr_t *llgo_traceback_fault_buffer(void)
{
    if (!fault_buffer)
        return 0;
    return VirtualAlloc(fault_buffer, LLGO_TRACEBACK_MAX*sizeof(uintptr_t),
                        0x1000, 4); /* commit the reserved pages, PAGE_READWRITE */
}

void llgo_traceback_release_fault_buffer(void)
{
    if (fault_buffer)
        VirtualFree(fault_buffer, LLGO_TRACEBACK_MAX*sizeof(uintptr_t), 0x4000);
}

void *llgo_traceback_register(uint64_t id, uint64_t parent, uintptr_t created)
{
    thread_node *n = calloc(1, sizeof(*n));
    if (!n) return 0;
    n->id = id; n->parent = parent; n->created = created; n->state = 1;
    AcquireSRWLockExclusive(&registry_lock);
    n->next = threads;
    if (threads) threads->prev = n;
    threads = n;
    ReleaseSRWLockExclusive(&registry_lock);
    return n;
}

void llgo_traceback_attach(void *raw)
{
    thread_node *n = raw;
    if (!n || current == n) return;
    if (!fault_buffer)
        fault_buffer = VirtualAlloc(0, LLGO_TRACEBACK_MAX*sizeof(uintptr_t), 0x2000, 4);
    DWORD tid = GetCurrentThreadId();
    HANDLE thread = OpenThread(0x0002 | 0x0008 | 0x0040, 0, tid);
    AcquireSRWLockExclusive(&registry_lock);
    n->thread_id = tid; n->thread = thread; n->state = 2;
    current = n;
    ReleaseSRWLockExclusive(&registry_lock);
}

void llgo_traceback_unregister(void *raw)
{
    thread_node *n = raw;
    if (!n) return;
    __atomic_store_n(&n->state, 6, __ATOMIC_RELEASE);
    AcquireSRWLockExclusive(&registry_lock);
    if (n->prev) n->prev->next = n->next;
    else threads = n->next;
    if (n->next) n->next->prev = n->prev;
    ReleaseSRWLockExclusive(&registry_lock);
    if (n->thread) CloseHandle(n->thread);
    if (current == n) {
        current = 0;
        if (fault_buffer) VirtualFree(fault_buffer, 0, 0x8000);
        fault_buffer = 0;
    }
    free(n);
}

void llgo_traceback_state(void *raw, uint32_t state)
{
    if (raw) __atomic_store_n(&((thread_node *)raw)->state, state, __ATOMIC_RELEASE);
}

typedef struct {
    void *base, *allocation_base;
    DWORD allocation_protect;
#if defined(_WIN64)
    unsigned short partition_id;
#endif
    size_t region_size;
    DWORD state, protect, type;
} memory_info;

static uintptr_t word(void *ctx, unsigned offset)
{
    return *(uintptr_t *)((char *)ctx+offset);
}
static void setword(void *ctx, unsigned offset, uintptr_t value)
{
    *(uintptr_t *)((char *)ctx+offset) = value;
}

#if defined(__aarch64__)
#define CTX_SIZE 912
#define CTX_FLAGS_OFFSET 0
#define CTX_FLAGS 0x00400007
#define PC_OFFSET 264
#define SP_OFFSET 256
#define FP_OFFSET 240
#define LR_OFFSET 248
#elif defined(__x86_64__)
#define CTX_SIZE 1232
#define CTX_FLAGS_OFFSET 48
#define CTX_FLAGS 0x0010000b
#define PC_OFFSET 248
#define SP_OFFSET 152
#define FP_OFFSET 160
#else
#define CTX_SIZE 716
#define CTX_FLAGS_OFFSET 0
#define CTX_FLAGS 0x00010007
#define PC_OFFSET 184
#define SP_OFFSET 196
#define FP_OFFSET 180
#endif

/* RtlVirtualUnwind reads stack slots directly. Unwind a private copy after
 * resuming the target: neither unwinder/loader locks nor fault recovery may
 * run while another thread that could own those locks is suspended. */
static void relocate_context(void *ctx, uintptr_t low, uintptr_t high,
                             uintptr_t copy)
{
#if defined(__aarch64__)
    unsigned first = 8, last = 256; /* X0..X30, SP */
#elif defined(__x86_64__)
    unsigned first = 120, last = 240; /* RAX..R15, including RSP */
#else
    unsigned first = FP_OFFSET, last = SP_OFFSET;
#endif
    for (unsigned offset = first; offset <= last; offset += sizeof(uintptr_t)) {
        uintptr_t value = word(ctx, offset);
        if (value >= low && value < high)
            setword(ctx, offset, copy + value-low);
    }
}

static _Thread_local _Alignas(16) unsigned char recovery_env[256];
static _Thread_local volatile int recovery_armed;

/* Use the runtime's ABI-only jump path: CRT longjmp on Win64 may try to
 * unwind a vectored exception frame that has no ordinary caller chain. */
#if defined(__aarch64__)
extern __attribute__((returns_twice)) int llgo_setjmp(void *);
extern __attribute__((noreturn)) void llgo_longjmp(void *, int);
#define capture_point() llgo_setjmp(recovery_env)
#define recover_capture() llgo_longjmp(recovery_env, 1)
#elif defined(__x86_64__)
extern __attribute__((returns_twice)) int _setjmpex(void *, void *);
extern __attribute__((noreturn)) void llgo_longjmp(void *, int);
#define capture_point() _setjmpex(recovery_env, 0)
#define recover_capture() llgo_longjmp(recovery_env, 1)
#else
extern __attribute__((returns_twice)) int _setjmp3(void *, int, ...);
extern __attribute__((noreturn)) void longjmp(void *, int);
#define capture_point() _setjmp3(recovery_env, 0)
#define recover_capture() longjmp(recovery_env, 1)
#endif

/* Called before the Go exception callback. Only the guarded C activation
 * is abandoned; the target thread has already resumed. */
int llgo_traceback_windows_fault_recover(void)
{
    if (!recovery_armed) return 0;
    recovery_armed = 0;
    recover_capture();
}

static size_t unwind_copy(void *ctx, uintptr_t *pcs, uintptr_t low,
                          uintptr_t high, uintptr_t copy, size_t size)
{
    if (capture_point() != 0) return 0;
    recovery_armed = 1;
    size_t count = 0;
    uintptr_t pc = word(ctx, PC_OFFSET);
    if (pc) pcs[count++] = pc+1;
    while (count < LLGO_TRACEBACK_MAX) {
        relocate_context(ctx, low, high, copy);
        uintptr_t sp = word(ctx, SP_OFFSET);
        if (sp < copy || sp-copy > size-sizeof(uintptr_t)) break;
#if defined(_WIN64)
        uint64_t image = 0;
        void *entry = RtlLookupFunctionEntry(pc, &image, 0);
        if (entry) {
            void *data = 0;
            uint64_t frame = 0;
            RtlVirtualUnwind(0, image, pc, entry, ctx, &data, &frame, 0);
        } else {
#if defined(__aarch64__)
            uintptr_t lr = word(ctx, LR_OFFSET);
            if (!lr) break;
            setword(ctx, PC_OFFSET, lr);
            setword(ctx, LR_OFFSET, 0);
#else
            setword(ctx, PC_OFFSET, *(uintptr_t *)sp);
            setword(ctx, SP_OFFSET, sp+sizeof(uintptr_t));
#endif
        }
#else
        uintptr_t fp = word(ctx, FP_OFFSET);
        if (fp < copy || fp-copy > size-2*sizeof(uintptr_t)) break;
        uintptr_t *record = (uintptr_t *)fp;
        setword(ctx, FP_OFFSET, record[0]);
        setword(ctx, PC_OFFSET, record[1]);
        setword(ctx, SP_OFFSET, fp+2*sizeof(uintptr_t));
#endif
        uintptr_t nextpc = word(ctx, PC_OFFSET), nextsp = word(ctx, SP_OFFSET);
        if (nextpc < 4096 || (nextpc == pc && nextsp == sp)) break;
        pc = nextpc;
        pcs[count++] = pc;
    }
    recovery_armed = 0;
    return count;
}

static size_t capture_thread(HANDLE thread, uintptr_t *pcs)
{
    _Alignas(16) unsigned char ctx[CTX_SIZE];
    memset(ctx, 0, sizeof(ctx));
    *(DWORD *)(ctx+CTX_FLAGS_OFFSET) = CTX_FLAGS;
    if (SuspendThread(thread) == (DWORD)-1) return 0;
    size_t size = 0;
    uintptr_t sp = 0;
    void *copy = 0;
    memory_info info;
    if (GetThreadContext(thread, ctx)) {
        sp = word(ctx, SP_OFFSET);
        if (VirtualQuery((void *)sp, &info, sizeof(info)) == sizeof(info) &&
            info.state == 0x1000 && !(info.protect & (0x100|1))) {
            size = (uintptr_t)info.base+info.region_size-sp;
            if (size > LLGO_TRACEBACK_COPY_MAX)
                size = LLGO_TRACEBACK_COPY_MAX;
            /* Allocate only the live portion, not an 8 MiB copy per capture. */
            copy = VirtualAlloc(0, size, 0x3000, 4);
            if (copy) {
                size_t read = 0;
                if (!ReadProcessMemory((HANDLE)(intptr_t)-1, (void *)sp, copy, size, &read) || read != size)
                    size = 0;
            }
        }
    }
    ResumeThread(thread);
    size_t count = size >= 2*sizeof(uintptr_t) && copy
                 ? unwind_copy(ctx, pcs, sp, sp+size, (uintptr_t)copy, size) : 0;
    if (copy) VirtualFree(copy, 0, 0x8000);
    return count;
}

llgo_traceback_snapshot *llgo_traceback_capture(uint64_t except)
{
    uintptr_t *scratch = VirtualAlloc(0, LLGO_TRACEBACK_MAX*sizeof(uintptr_t), 0x3000, 4);
    if (!scratch) return 0;
    llgo_traceback_snapshot *head = 0, **tail = &head;
    AcquireSRWLockExclusive(&registry_lock);
    for (thread_node *n = threads; n; n = n->next) {
        uint32_t state = __atomic_load_n(&n->state, __ATOMIC_ACQUIRE);
        if (n->id == except || state == 6 || n->thread_id == GetCurrentThreadId())
            continue;
        size_t count = n->thread ? capture_thread(n->thread, scratch) : 0;
        llgo_traceback_snapshot *s = calloc(1, sizeof(*s)+count*sizeof(uintptr_t));
        if (!s) break;
        s->id = n->id; s->parent = n->parent; s->created = n->created;
        s->state = state; s->count = count; s->pcs = (uintptr_t *)(s+1);
        memcpy(s->pcs, scratch, count*sizeof(uintptr_t));
        *tail = s; tail = &s->next;
    }
    ReleaseSRWLockExclusive(&registry_lock);
    VirtualFree(scratch, 0, 0x8000);
    return head;
}

void llgo_traceback_free(llgo_traceback_snapshot *head)
{
    while (head) {
        llgo_traceback_snapshot *next = head->next;
        free(head); head = next;
    }
}
