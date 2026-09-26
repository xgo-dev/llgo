/* Fault handler with fault-site context: see the comment on
 * llgo_install_fault_handler below. Separate file because darwin's
 * ucontext.h demands _XOPEN_SOURCE before any system header. */
#define _XOPEN_SOURCE 700
#define _DARWIN_C_SOURCE 1
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif

/* Fault handler with fault-site context. The plain signal() handler the
 * runtime core installs cannot see the interrupted registers, and walking
 * from the handler's own frame stops at the signal trampoline — so the
 * panic pc snapshot for hardware faults (nil derefs compiled with
 * null_pointer_is_valid, faults inside C code, integer division on x86)
 * starts from the ucontext's pc/fp instead. */
#include <errno.h>
#include <signal.h>
#include <stdint.h>
#include <sys/mman.h>
#include <unistd.h>

static long llgo_pagesz; /* primed at handler install, out of signal context */

#if defined(__APPLE__) || defined(__linux__)
#include <ucontext.h>

static void (*llgo_fault_go)(uintptr_t pc, uintptr_t fp, int sig);

/* profile.c arms this recovery point only while dereferencing a sampled
 * frame-pointer chain. A bad frame truncates that sample instead of entering
 * the Go fault path while the profiler ring lock is held. */
extern int llgo_cpu_profile_fault_recover(void);
extern int llgo_traceback_fault_recover(void);

/* Dynamic-libunwind fault unwinding (dynunwind.c); no-ops when disabled
 * (LLGO_DYNUNWIND=0) or no libunwind flavor resolved. */
extern void llgo_dynunwind_init(void);
extern void llgo_dynunwind_capture(void *uctx);

static _Thread_local volatile int llgo_in_fault;

static void llgo_fault_trampoline(int sig, siginfo_t *info, void *uctx)
{
    uintptr_t pc = 0, fp = 0;
    ucontext_t *uc = (ucontext_t *)uctx;
    (void)info;
    if (llgo_traceback_fault_recover() || llgo_cpu_profile_fault_recover())
        return;
    if (llgo_in_fault) {
        signal(sig, SIG_DFL);
        raise(sig);
        return;
    }
    llgo_in_fault = 1;
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
    llgo_dynunwind_capture(uctx);
    llgo_fault_go(pc, fp, sig);
}

void llgo_fault_capture_done(void)
{
    /* Belt and braces alongside SA_NODEFER: make sure no fault signal
     * stays blocked once this fault becomes an ordinary panic. */
    sigset_t set;
    sigemptyset(&set);
    sigaddset(&set, SIGSEGV);
    sigaddset(&set, SIGBUS);
    sigaddset(&set, SIGFPE);
    sigprocmask(SIG_UNBLOCK, &set, 0);
    llgo_in_fault = 0;
}

int llgo_mem_readable(void *p);

void llgo_install_fault_handler(void (*cb)(uintptr_t, uintptr_t, int))
{
    struct sigaction sa;
    int sigs[3] = {SIGSEGV, SIGBUS, SIGFPE};
    int i;
    /* Installation runs at startup, before user code. dlopen probes for
     * libunwind (and sysconf/sigaction) may set errno; a leaked errno is
     * visible to the first two-result cgo call ("v, err := C.f()"), which
     * turns it into a spurious non-nil err. */
    int saved_errno = errno;
    llgo_fault_go = cb;
    llgo_pagesz = sysconf(_SC_PAGESIZE);
    llgo_dynunwind_init();
    for (i = 0; i < 3; i++) {
        sa.sa_sigaction = llgo_fault_trampoline;
        sigemptyset(&sa.sa_mask);
        /* SA_NODEFER: the handler converts the fault to a Go panic that
         * longjmps out through jmpbufs saved with savemask=0, so an
         * auto-blocked signal would stay blocked forever — the next
         * genuine fault would then be force-delivered with the default
         * action (core) instead of reaching us. Recursion is guarded by
         * llgo_in_fault above. */
        sa.sa_flags = SA_SIGINFO | SA_NODEFER;
        sigaction(sigs[i], &sa, 0);
    }
    errno = saved_errno;
}
#endif

/* Probe whether one byte at p is mapped-readable: msync on the containing
 * page returns ENOMEM for unmapped ranges. Used by the fault-context
 * walks — an arithmetic-valid frame pointer can still point into a hole,
 * and faulting inside the fault path recurses. */
int llgo_mem_readable(void *p)
{
    int saved_errno = errno;
    char *page;
    long sz = llgo_pagesz;
    if (sz <= 0)
        sz = 4096;
    page = (char *)((uintptr_t)p & ~(uintptr_t)(sz - 1));
    if (msync(page, 1, MS_ASYNC) == 0)
        { int r = 1; errno = saved_errno; return r; }
    { int r = errno != ENOMEM; errno = saved_errno; return r; }
}
