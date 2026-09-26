#define _GNU_SOURCE 1
#define _XOPEN_SOURCE 700
#define _DARWIN_C_SOURCE 1
#include "traceback.h"
#include <errno.h>
#include <pthread.h>
#include <sched.h>
#include <setjmp.h>
#include <signal.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <time.h>
#include <ucontext.h>
#include <unistd.h>

typedef struct thread_node {
    struct thread_node *next, *prev;
    uint64_t id, parent;
    uintptr_t created, low, high;
    pthread_t thread;
    int attached;
    uint32_t state;
    /* 0 idle, 1 requested, 2 handler owns the buffer, 3 complete. */
    unsigned request;
    uintptr_t *pcs;
    size_t count;
} thread_node;

static pthread_mutex_t registry_lock = PTHREAD_MUTEX_INITIALIZER;
static pthread_once_t handler_once = PTHREAD_ONCE_INIT;
static thread_node *threads;
static _Thread_local thread_node *current;
static _Thread_local uintptr_t *fault_buffer;
static _Thread_local sigjmp_buf capture_recovery;
static _Thread_local volatile sig_atomic_t capture_armed;
static llgo_traceback_unwinder unwind_context;
static llgo_traceback_thread_init initialize_thread;
static struct sigaction previous_urgent;
/* Immutable original disposition plus one atomic function pointer. A handler
 * must never read a struct sigaction concurrently with os/signal changing it. */
static void original_urgent(int sig) { (void)sig; }
static void (*urgent_user)(int) = original_urgent;

static void *map_buffer(void)
{
    void *p = mmap(0, LLGO_TRACEBACK_MAX * sizeof(uintptr_t),
                   PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANON, -1, 0);
    return p == MAP_FAILED ? 0 : p;
}

uintptr_t *llgo_traceback_fault_buffer(void) { return fault_buffer; }

void llgo_traceback_bounds(uintptr_t *low, uintptr_t *high)
{
    *low = current ? current->low : 0;
    *high = current ? current->high : 0;
}

void llgo_traceback_release_fault_buffer(void)
{
    if (fault_buffer)
        madvise(fault_buffer, LLGO_TRACEBACK_MAX * sizeof(uintptr_t), MADV_DONTNEED);
}

/* A diagnostic walk must not turn bad unwind metadata into a Go panic, or
 * abandon a collector that is waiting for this handler to release scratch. */
int llgo_traceback_fault_recover(void)
{
    if (!capture_armed)
        return 0;
    capture_armed = 0;
    siglongjmp(capture_recovery, 1);
}

static size_t frame_records(uintptr_t fp, uintptr_t *pcs, size_t cap,
                             uintptr_t low, uintptr_t high)
{
    size_t n = 0;
    while (n < cap && fp >= low && high >= 2*sizeof(uintptr_t) &&
           fp <= high - 2*sizeof(uintptr_t) && !(fp & (sizeof(uintptr_t)-1))) {
        uintptr_t *record = (uintptr_t *)fp;
        uintptr_t prev = record[0], ret = record[1];
        if (ret < 4096)
            break;
        pcs[n++] = ret;
        if (prev <= fp || prev-fp > (1u << 20))
            break;
        fp = prev;
    }
    return n;
}

static size_t frame_chain(void *raw, uintptr_t *pcs, size_t cap,
                          uintptr_t low, uintptr_t high)
{
    ucontext_t *uc = raw;
    uintptr_t pc = 0, fp = 0;
#if defined(__APPLE__) && defined(__aarch64__)
    pc = uc->uc_mcontext->__ss.__pc; fp = uc->uc_mcontext->__ss.__fp;
#elif defined(__APPLE__) && defined(__x86_64__)
    pc = uc->uc_mcontext->__ss.__rip; fp = uc->uc_mcontext->__ss.__rbp;
#elif defined(__linux__) && defined(__aarch64__)
    pc = uc->uc_mcontext.pc; fp = uc->uc_mcontext.regs[29];
#elif defined(__linux__) && defined(__x86_64__)
    pc = uc->uc_mcontext.gregs[REG_RIP]; fp = uc->uc_mcontext.gregs[REG_RBP];
#endif
    size_t n = 0;
    if (pc && cap)
        pcs[n++] = pc + 1;
    return n + frame_records(fp, pcs+n, cap-n, low, high);
}

/* One recovery point on the successful path. A failed platform unwind retries
 * only the bounded frame chain; failure there makes this snapshot unavailable. */
static size_t capture_context(void *raw, thread_node *node)
{
    size_t n = 0;
    if (sigsetjmp(capture_recovery, 1) == 0) {
        capture_armed = 1;
        llgo_traceback_unwinder walk = __atomic_load_n(&unwind_context, __ATOMIC_ACQUIRE);
        uintptr_t endfp = 0;
        n = walk ? walk(raw, node->pcs, LLGO_TRACEBACK_MAX, &endfp) : 0;
        if (n && endfp && n < LLGO_TRACEBACK_MAX) {
            size_t extra = frame_records(endfp, node->pcs+n, LLGO_TRACEBACK_MAX-n,
                                         node->low, node->high);
            if (extra && node->pcs[n] == node->pcs[n-1])
                memmove(node->pcs+n, node->pcs+n+1, (--extra)*sizeof(uintptr_t));
            n += extra;
        }
        if (!n)
            n = frame_chain(raw, node->pcs, LLGO_TRACEBACK_MAX, node->low, node->high);
        capture_armed = 0;
    } else {
        n = 0;
        if (sigsetjmp(capture_recovery, 1) == 0) {
            capture_armed = 1;
            n = frame_chain(raw, node->pcs, LLGO_TRACEBACK_MAX, node->low, node->high);
            capture_armed = 0;
        } else {
            n = 0;
        }
    }
    return n;
}

static void urgent_handler(int sig, siginfo_t *info, void *raw)
{
    int saved = errno;
    thread_node *node = current;
    unsigned expected = 1;
    if (node && __atomic_compare_exchange_n(&node->request, &expected, 2, 0,
                                            __ATOMIC_ACQUIRE, __ATOMIC_RELAXED)) {
        node->count = capture_context(raw, node);
        __atomic_store_n(&node->request, 3, __ATOMIC_RELEASE);
    } else {
        void (*user)(int) = __atomic_load_n(&urgent_user, __ATOMIC_ACQUIRE);
        if (user == original_urgent) {
            if (previous_urgent.sa_handler != SIG_IGN && previous_urgent.sa_handler != SIG_DFL) {
                if (previous_urgent.sa_flags & SA_SIGINFO)
                    previous_urgent.sa_sigaction(sig, info, raw);
                else
                    previous_urgent.sa_handler(sig);
            }
        } else if (user != SIG_IGN && user != SIG_DFL) {
            user(sig);
        }
    }
    errno = saved;
}

static void install_handler(void)
{
    struct sigaction action;
    memset(&action, 0, sizeof(action));
    action.sa_sigaction = urgent_handler;
    action.sa_flags = SA_RESTART | SA_SIGINFO;
    sigemptyset(&action.sa_mask);
    /* CPU profiling has its own nested-fault recovery point. It must not
     * interrupt this guard while holding its sample buffer lock. */
    sigaddset(&action.sa_mask, SIGPROF);
    sigaction(SIGURG, 0, &previous_urgent);
    sigaction(SIGURG, &action, 0);
}

/* os/signal keeps its requested SIGURG disposition while the runtime retains
 * the transport used for stack capture. Ordinary signals still reach it. */
void llgo_traceback_urgent_action(void (*handler)(int))
{
    pthread_once(&handler_once, install_handler);
    __atomic_store_n(&urgent_user, handler, __ATOMIC_RELEASE);
}

void llgo_traceback_set_unwinder(llgo_traceback_unwinder walk,
                                llgo_traceback_thread_init init)
{
    __atomic_store_n(&initialize_thread, init, __ATOMIC_RELEASE);
    if (init)
        init();
    __atomic_store_n(&unwind_context, walk, __ATOMIC_RELEASE);
}

void *llgo_traceback_register(uint64_t id, uint64_t parent, uintptr_t created)
{
    thread_node *node = calloc(1, sizeof(*node));
    if (!node)
        return 0;
    node->id = id; node->parent = parent; node->created = created;
    node->state = 1; /* runnable */
    pthread_mutex_lock(&registry_lock);
    node->next = threads;
    if (threads) threads->prev = node;
    threads = node;
    pthread_mutex_unlock(&registry_lock);
    return node;
}

void llgo_traceback_attach(void *raw)
{
    thread_node *node = raw;
    if (!node || current == node)
        return;
    pthread_once(&handler_once, install_handler);
    if (!fault_buffer)
        fault_buffer = map_buffer();
    llgo_traceback_thread_init init = __atomic_load_n(&initialize_thread, __ATOMIC_ACQUIRE);
    if (init)
        init();
    uintptr_t low = 0, high = 0;
#if defined(__APPLE__)
    high = (uintptr_t)pthread_get_stackaddr_np(pthread_self());
    low = high - pthread_get_stacksize_np(pthread_self());
#elif defined(__linux__)
    pthread_attr_t attr;
    if (pthread_getattr_np(pthread_self(), &attr) == 0) {
        void *base; size_t size;
        if (pthread_attr_getstack(&attr, &base, &size) == 0) {
            low = (uintptr_t)base; high = low + size;
        }
        pthread_attr_destroy(&attr);
    }
#endif
    pthread_mutex_lock(&registry_lock);
    node->low = low; node->high = high;
    node->thread = pthread_self();
    node->attached = 1;
    node->state = 2;
    current = node;
    pthread_mutex_unlock(&registry_lock);
    sigset_t mask;
    sigemptyset(&mask); sigaddset(&mask, SIGURG);
    pthread_sigmask(SIG_UNBLOCK, &mask, 0);
}

void llgo_traceback_unregister(void *raw)
{
    thread_node *node = raw;
    if (!node)
        return;
    __atomic_store_n(&node->state, 6, __ATOMIC_RELEASE);
    pthread_mutex_lock(&registry_lock);
    if (node->prev) node->prev->next = node->next;
    else threads = node->next;
    if (node->next) node->next->prev = node->prev;
    pthread_mutex_unlock(&registry_lock);
    if (current == node) {
        current = 0;
        if (fault_buffer) {
            munmap(fault_buffer, LLGO_TRACEBACK_MAX * sizeof(uintptr_t));
            fault_buffer = 0;
        }
    }
    free(node);
}

void llgo_traceback_state(void *raw, uint32_t state)
{
    if (raw)
        __atomic_store_n(&((thread_node *)raw)->state, state, __ATOMIC_RELEASE);
}

static uint64_t nanotime(void)
{
    struct timespec now;
    clock_gettime(CLOCK_MONOTONIC, &now);
    return (uint64_t)now.tv_sec * 1000000000u + now.tv_nsec;
}

llgo_traceback_snapshot *llgo_traceback_capture(uint64_t except)
{
    uintptr_t *scratch = map_buffer();
    if (!scratch)
        return 0;
    llgo_traceback_snapshot *head = 0, **tail = &head;
    uint64_t deadline = nanotime() + 2000000000u;
    pthread_mutex_lock(&registry_lock);
    for (thread_node *node = threads; node; node = node->next) {
        unsigned state = __atomic_load_n(&node->state, __ATOMIC_ACQUIRE);
        if (node->id == except || state == 6)
            continue;
        node->count = 0;
        if (node->attached && nanotime() < deadline) {
            node->pcs = scratch;
            __atomic_store_n(&node->request, 1, __ATOMIC_RELEASE);
            int sent = pthread_kill(node->thread, SIGURG);
            uint64_t timeout = nanotime() + 250000000u;
            unsigned spins = 0;
            while (__atomic_load_n(&node->request, __ATOMIC_ACQUIRE) != 3) {
                if (sent || nanotime() > timeout || nanotime() > deadline) {
                    unsigned expected = 1;
                    if (__atomic_compare_exchange_n(&node->request, &expected, 0, 0,
                                                     __ATOMIC_ACQ_REL, __ATOMIC_ACQUIRE))
                        break;
                    /* Once the handler owns scratch, wait until it releases
                     * it; cancellation must never free a live handler buffer. */
                }
                if (spins++ < 32) {
                    sched_yield();
                } else {
                    /* A foreign call may mask SIGURG. Do not burn a CPU
                     * for its entire timeout while holding the registry. */
                    const struct timespec pause = {0, 100000};
                    nanosleep(&pause, 0);
                }
            }
            __atomic_store_n(&node->request, 0, __ATOMIC_RELEASE);
        }
        llgo_traceback_snapshot *s = calloc(1, sizeof(*s) + node->count*sizeof(uintptr_t));
        if (!s)
            break;
        s->id = node->id; s->parent = node->parent; s->created = node->created;
        s->state = state; s->count = node->count;
        s->pcs = (uintptr_t *)(s+1);
        memcpy(s->pcs, scratch, node->count*sizeof(uintptr_t));
        *tail = s; tail = &s->next;
    }
    pthread_mutex_unlock(&registry_lock);
    munmap(scratch, LLGO_TRACEBACK_MAX * sizeof(uintptr_t));
    return head;
}

void llgo_traceback_free(llgo_traceback_snapshot *head)
{
    while (head) {
        llgo_traceback_snapshot *next = head->next;
        free(head); head = next;
    }
}
