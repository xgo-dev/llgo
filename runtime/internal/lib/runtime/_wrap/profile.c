/* CPU profiling for native LLGo executables.
 *
 * SIGPROF interrupts arbitrary code, so the handler only snapshots register
 * state and frame-pointer slots into a fixed ring. It does not allocate,
 * acquire a blocking lock, or call into Go. Ordinary Go code drains the ring
 * later and converts it to runtime/pprof's raw record stream. */
#define _XOPEN_SOURCE 700
#define _DARWIN_C_SOURCE 1
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif

#include <errno.h>
#include <pthread.h>
#include <sched.h>
#include <setjmp.h>
#include <signal.h>
#include <stdint.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/time.h>
#include <unistd.h>

#if defined(__APPLE__) || defined(__linux__)
#include <ucontext.h>
#endif

#define LLGO_PROF_STACK 64
#define LLGO_PROF_SAMPLES 2048
#define LLGO_PROF_MAX_FP_STRIDE (1u << 20)

/* Raw ABI shared with LLGO_SIGNAL_* in signal.c and signalMode* in
 * signal_llgo.go. Keep the numeric values synchronized. */
enum {
    LLGO_PROF_SIGNAL_DEFAULT = 0,
    LLGO_PROF_SIGNAL_WANTED = 1,
    LLGO_PROF_SIGNAL_IGNORED = 2,
};

struct llgo_prof_sample {
    uint32_t n;
    uintptr_t pc[LLGO_PROF_STACK];
};

static struct llgo_prof_sample llgo_prof_ring[LLGO_PROF_SAMPLES];
static unsigned int llgo_prof_read_index;
static unsigned int llgo_prof_write_index;
static volatile int llgo_prof_ring_lock;
static volatile int llgo_prof_active;
static volatile uint64_t llgo_prof_lost;
#if defined(__APPLE__) || defined(__linux__)
static struct sigaction llgo_prof_previous_action;
static int llgo_prof_previous_valid;
static sigjmp_buf llgo_prof_fault_jmp;
static volatile uintptr_t llgo_prof_fault_owner;
static volatile int llgo_prof_fault_active;
/* The Go cpuProfileStateMu serializes these control-plane fields. They are
 * never read from the asynchronous profiling handler. */
static int llgo_prof_previous_signal_mode;
static int llgo_prof_desired_signal_mode;
static int llgo_prof_signal_mode_updated;

/* begin reports a previous mode or negative errno. The remaining helpers
 * report zero or positive errno, matching the ordinary signal helpers. */
extern int llgo_signal_profile_begin(int signum);
extern int llgo_signal_profile_abort(int signum, int mode);
extern int llgo_signal_profile_finish(int signum, int mode);
extern int llgo_signal_profile_restore(int signum, int mode);
#endif

static int llgo_prof_ring_try_lock(void)
{
    return __atomic_exchange_n(&llgo_prof_ring_lock, 1, __ATOMIC_ACQUIRE) == 0;
}

static void llgo_prof_ring_lock_wait(void)
{
    unsigned int spins = 0;
    while (!llgo_prof_ring_try_lock()) {
        if (++spins == 64) {
            sched_yield();
            spins = 0;
        }
    }
}

static void llgo_prof_ring_unlock(void)
{
    __atomic_store_n(&llgo_prof_ring_lock, 0, __ATOMIC_RELEASE);
}

static void llgo_prof_drop(void)
{
    __atomic_fetch_add(&llgo_prof_lost, 1, __ATOMIC_RELAXED);
}

#if defined(__APPLE__) || defined(__linux__)
static void llgo_prof_signal(int sig, siginfo_t *info, void *uctx);

/* Called first by the runtime fault handler. The global jump buffer is safe
 * because the ring lock permits only one active profile walk; the owner check
 * prevents a simultaneous fault on another pthread from jumping across
 * threads. */
int llgo_cpu_profile_fault_recover(void)
{
    int expected = 1;

    if (!__atomic_load_n(&llgo_prof_fault_active, __ATOMIC_ACQUIRE) ||
        __atomic_load_n(&llgo_prof_fault_owner, __ATOMIC_RELAXED) !=
            (uintptr_t)pthread_self())
        return 0;
    if (!__atomic_compare_exchange_n(&llgo_prof_fault_active, &expected, 0,
                                     0, __ATOMIC_ACQ_REL,
                                     __ATOMIC_ACQUIRE))
        return 0;
    siglongjmp(llgo_prof_fault_jmp, 1);
    return 1;
}

static void llgo_prof_walk_frames(struct llgo_prof_sample *sample, uintptr_t fp)
{
    uintptr_t word = sizeof(uintptr_t);

    __atomic_store_n(&llgo_prof_fault_owner, (uintptr_t)pthread_self(),
                     __ATOMIC_RELAXED);
    if (sigsetjmp(llgo_prof_fault_jmp, 1) != 0)
        return;
    __atomic_store_n(&llgo_prof_fault_active, 1, __ATOMIC_RELEASE);
    while (fp != 0 && sample->n < LLGO_PROF_STACK) {
        uintptr_t prev, ret;
        if ((fp & (word - 1)) != 0)
            break;
        prev = *(uintptr_t *)fp;
        ret = *(uintptr_t *)(fp + word);
        if (ret < 4096)
            break;
        sample->pc[sample->n++] = ret;
        if (prev <= fp || prev - fp > LLGO_PROF_MAX_FP_STRIDE ||
            (prev & (word - 1)) != 0)
            break;
        fp = prev;
    }
    __atomic_store_n(&llgo_prof_fault_active, 0, __ATOMIC_RELEASE);
}

static int llgo_prof_action_is_ours(const struct sigaction *sa)
{
    return (sa->sa_flags & SA_SIGINFO) != 0 &&
           sa->sa_sigaction == llgo_prof_signal;
}

static int llgo_prof_install_signal(void)
{
    struct sigaction current;
    struct sigaction sa;

    if (sigaction(SIGPROF, 0, &current) != 0)
        return -1;
    if (llgo_prof_action_is_ours(&current))
        return 0;

    /* Keep the disposition that was current immediately before this install.
     * A default disposition is normalized to ignore, as the Go runtime does:
     * a final pending timer signal must not terminate the process after Stop. */
    llgo_prof_previous_action = current;
    /* exec can leave SA_SIGINFO set after resetting a caught disposition to
     * SIG_DFL. Test the sentinel in sa_handler even with SA_SIGINFO set: that
     * flag selects how an actual callback is invoked, not a special disposition. */
    if (current.sa_handler == SIG_DFL || current.sa_handler == SIG_IGN) {
        llgo_prof_previous_action.sa_handler = SIG_IGN;
        llgo_prof_previous_action.sa_flags &= ~SA_SIGINFO;
    }
    llgo_prof_previous_valid = 1;

    memset(&sa, 0, sizeof(sa));
    sa.sa_sigaction = llgo_prof_signal;
    sa.sa_mask = current.sa_mask;
    sa.sa_flags = SA_SIGINFO | SA_RESTART;
#ifdef SA_ONSTACK
    sa.sa_flags |= current.sa_flags & SA_ONSTACK;
#endif
    return sigaction(SIGPROF, &sa, 0);
}

static int llgo_prof_restore_signal(void)
{
    struct sigaction current;
    int code;
    int mode;
    int updated = llgo_prof_signal_mode_updated;

    mode = updated ? llgo_prof_desired_signal_mode
                   : llgo_prof_previous_signal_mode;
    code = llgo_signal_profile_finish(SIGPROF, mode);
    llgo_prof_signal_mode_updated = 0;
    if (code != 0)
        return code;
    if (sigaction(SIGPROF, 0, &current) != 0)
        return errno;
    /* Do not overwrite a handler installed by foreign code that did not use
     * the coordinated os/signal path. */
    if (!llgo_prof_action_is_ours(&current))
        return 0;
    if (updated)
        return llgo_signal_profile_restore(SIGPROF, mode);
    if (llgo_prof_previous_valid &&
        sigaction(SIGPROF, &llgo_prof_previous_action, 0) != 0)
        return errno;
    return 0;
}

static void llgo_prof_signal(int sig, siginfo_t *info, void *uctx)
{
    uintptr_t pc = 0, fp = 0;
    unsigned int next;
    struct llgo_prof_sample *sample;
    int active;
    ucontext_t *uc = (ucontext_t *)uctx;
    int saved_errno = errno;
    (void)sig;
    (void)info;
    (void)uc;

    active = __atomic_load_n(&llgo_prof_active, __ATOMIC_ACQUIRE);
    /* Like the Go runtime, the profiler owns SIGPROF while active. In
     * particular, timer samples are not forwarded to os/signal watchers. */
    if (!active) {
        errno = saved_errno;
        return;
    }
    if (!llgo_prof_ring_try_lock()) {
        llgo_prof_drop();
        errno = saved_errno;
        return;
    }
    active = __atomic_load_n(&llgo_prof_active, __ATOMIC_RELAXED);
    if (!active)
        goto done;
#if defined(__APPLE__) && defined(__aarch64__)
    pc = (uintptr_t)uc->uc_mcontext->__ss.__pc;
    fp = (uintptr_t)uc->uc_mcontext->__ss.__fp;
#elif defined(__APPLE__) && defined(__x86_64__)
    pc = (uintptr_t)uc->uc_mcontext->__ss.__rip;
    fp = (uintptr_t)uc->uc_mcontext->__ss.__rbp;
#elif defined(__linux__) && defined(__aarch64__)
    pc = (uintptr_t)uc->uc_mcontext.pc;
    fp = (uintptr_t)uc->uc_mcontext.regs[29];
#elif defined(__linux__) && defined(__x86_64__)
    pc = (uintptr_t)uc->uc_mcontext.gregs[16 /* REG_RIP */];
    fp = (uintptr_t)uc->uc_mcontext.gregs[10 /* REG_RBP */];
#endif
    if (pc == 0) {
        llgo_prof_drop();
        goto done;
    }

    next = llgo_prof_write_index + 1;
    if (next == LLGO_PROF_SAMPLES)
        next = 0;
    if (next == llgo_prof_read_index) {
        llgo_prof_drop();
        goto done;
    }

    sample = &llgo_prof_ring[llgo_prof_write_index];
    sample->n = 1;
    /* runtime.CallersFrames subtracts one from every sampled PC. */
    sample->pc[0] = pc + 1;
    llgo_prof_walk_frames(sample, fp);
    llgo_prof_write_index = next;
done:
    llgo_prof_ring_unlock();
    errno = saved_errno;
}
#endif

/* Returns 1 on success, 0 while an old profile is still draining, and -1
 * when the OS rejects SIGPROF or ITIMER_PROF setup. */
int llgo_cpu_profile_start(int hz)
{
#if defined(__APPLE__) || defined(__linux__)
    struct itimerval timer;
    uint64_t usec;
    int saved_errno = errno;

    if (hz <= 0) {
        errno = saved_errno;
        return -1;
    }
    llgo_prof_ring_lock_wait();
    if (__atomic_load_n(&llgo_prof_active, __ATOMIC_RELAXED) ||
        llgo_prof_read_index != llgo_prof_write_index) {
        llgo_prof_ring_unlock();
        errno = saved_errno;
        return 0;
    }
    llgo_prof_previous_signal_mode = llgo_signal_profile_begin(SIGPROF);
    if (llgo_prof_previous_signal_mode < 0) {
        llgo_prof_ring_unlock();
        errno = saved_errno;
        return -1;
    }
    if (llgo_prof_install_signal() != 0) {
        llgo_signal_profile_abort(SIGPROF,
                                  llgo_prof_previous_signal_mode);
        llgo_prof_ring_unlock();
        errno = saved_errno;
        return -1;
    }
    llgo_prof_read_index = 0;
    llgo_prof_write_index = 0;
    llgo_prof_signal_mode_updated = 0;
    __atomic_store_n(&llgo_prof_lost, 0, __ATOMIC_RELAXED);
    __atomic_store_n(&llgo_prof_active, 1, __ATOMIC_RELEASE);

    usec = 1000000u / (unsigned int)hz;
    if (usec == 0)
        usec = 1;
    memset(&timer, 0, sizeof(timer));
    timer.it_interval.tv_sec = (time_t)(usec / 1000000u);
    timer.it_interval.tv_usec = (suseconds_t)(usec % 1000000u);
    timer.it_value = timer.it_interval;
    if (setitimer(ITIMER_PROF, &timer, 0) != 0) {
        __atomic_store_n(&llgo_prof_active, 0, __ATOMIC_RELEASE);
        llgo_prof_restore_signal();
        llgo_prof_ring_unlock();
        errno = saved_errno;
        return -1;
    }
    llgo_prof_ring_unlock();
    errno = saved_errno;
    return 1;
#else
    (void)hz;
    return -1;
#endif
}

int llgo_cpu_profile_stop(void)
{
#if defined(__APPLE__) || defined(__linux__)
    struct itimerval timer;
    int code;
    int saved_errno = errno;
    llgo_prof_ring_lock_wait();
    memset(&timer, 0, sizeof(timer));
    setitimer(ITIMER_PROF, &timer, 0);
    __atomic_store_n(&llgo_prof_active, 0, __ATOMIC_RELEASE);
    code = llgo_prof_restore_signal();
    llgo_prof_ring_unlock();
    errno = saved_errno;
    return code;
#else
    return 0;
#endif
}

/* os/signal must record its desired SIGPROF state without replacing the
 * active profiler handler or changing the live handler mode. Installing
 * SIG_DFL even briefly can terminate the process when a timer signal arrives
 * between that install and the profiler refresh. Exposing DEFAULT to an old
 * os/signal handler that the kernel selected before profiling began has the
 * same problem. The Go control plane serializes this with start and stop. */
int llgo_cpu_profile_update_signal(int mode)
{
#if defined(__APPLE__) || defined(__linux__)
    if (!__atomic_load_n(&llgo_prof_active, __ATOMIC_RELAXED))
        return 0;
    if (mode < LLGO_PROF_SIGNAL_DEFAULT ||
        mode > LLGO_PROF_SIGNAL_IGNORED)
        return -EINVAL;
    llgo_prof_desired_signal_mode = mode;
    llgo_prof_signal_mode_updated = 1;
    return 1;
#else
    (void)mode;
    return 0;
#endif
}

/* An os/signal watcher update may replace the process disposition. The Go
 * control plane serializes that update with profile start/stop, then asks the
 * native sampler to take ownership back while profiling is active. */
int llgo_cpu_profile_refresh_signal(void)
{
#if defined(__APPLE__) || defined(__linux__)
    if (__atomic_load_n(&llgo_prof_active, __ATOMIC_RELAXED))
        return llgo_prof_install_signal();
    return 0;
#else
    return 0;
#endif
}

int llgo_cpu_profile_drain(uintptr_t *pc, uint32_t *lengths,
                           int max_records, int max_stack,
                           uint64_t *lost, int *empty)
{
    struct llgo_prof_sample *sample;
    unsigned int i, n;
    int records = 0;

    if (pc == 0 || lengths == 0 || max_records <= 0 || max_stack <= 0 ||
        lost == 0 || empty == 0)
        return 0;
    llgo_prof_ring_lock_wait();
    *lost = __atomic_exchange_n(&llgo_prof_lost, 0, __ATOMIC_RELAXED);
    while (llgo_prof_read_index != llgo_prof_write_index &&
           records < max_records) {
        sample = &llgo_prof_ring[llgo_prof_read_index];
        n = sample->n;
        if (n > (unsigned int)max_stack)
            n = (unsigned int)max_stack;
        lengths[records] = n;
        for (i = 0; i < n; i++)
            pc[(size_t)records * (size_t)max_stack + i] = sample->pc[i];
        records++;
        llgo_prof_read_index++;
        if (llgo_prof_read_index == LLGO_PROF_SAMPLES)
            llgo_prof_read_index = 0;
    }
    *empty = llgo_prof_read_index == llgo_prof_write_index;
    llgo_prof_ring_unlock();
    return records;
}

/* Exercise the guarded frame walk against a mapped but unreadable page. This
 * is kept as a narrow runtime test hook so a regression fails deterministically
 * instead of depending on a corrupt frame pointer appearing in a sample. */
int llgo_cpu_profile_test_fault_recovery(void)
{
#if defined(__APPLE__) || defined(__linux__)
    struct llgo_prof_sample sample;
    long page_size = sysconf(_SC_PAGESIZE);
    void *page;
    int n;

    if (page_size <= 0)
        page_size = 4096;
    page = mmap(0, (size_t)page_size, PROT_NONE,
                MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (page == MAP_FAILED)
        return -1;
    sample.n = 1;
    sample.pc[0] = 1;
    llgo_prof_ring_lock_wait();
    llgo_prof_walk_frames(&sample, (uintptr_t)page);
    llgo_prof_ring_unlock();
    n = (int)sample.n;
    munmap(page, (size_t)page_size);
    return n;
#else
    return -1;
#endif
}
