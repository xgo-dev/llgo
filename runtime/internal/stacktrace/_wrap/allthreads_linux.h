/* Private implementation included by traceback_unix.c. Production entry
 * modules must not enable this transport until creation exclusion covers
 * foreign C thread creators as well as the runtime's registered cohort. */
#if defined(__linux__)
static pthread_mutex_t credential_lock = PTHREAD_MUTEX_INITIALIZER;
void llgo_allthreads_credential_lock(void) { pthread_mutex_lock(&credential_lock); }
void llgo_allthreads_credential_unlock(void) { pthread_mutex_unlock(&credential_lock); }

void llgo_traceback_mark_owned(void *raw)
{
#if defined(__x86_64__) || defined(__aarch64__)
    pthread_mutex_lock(&registry_lock);
    if (raw) ((thread_node *)raw)->owned = 1;
    pthread_mutex_unlock(&registry_lock);
#else
    (void)raw;
#endif
}

#if defined(__x86_64__) || defined(__aarch64__)
__attribute__((weak, visibility("hidden"))) int __llgo_allthreads_executable;

enum { ALL_IDLE, ALL_ARMED, ALL_CLAIMED, ALL_READY, ALL_DONE, ALL_RELEASED, ALL_FAILED };
enum { PHASE_IDLE, PHASE_PREPARE, PHASE_RUN, PHASE_RELEASE, PHASE_CANCEL };
static unsigned allcall_phase;
static uintptr_t allcall_trap, allcall_args[6];

typedef struct { uintptr_t r1, r2, error; } kernel_result;

static kernel_result kernel_syscall(uintptr_t trap, const uintptr_t *args)
{
    uintptr_t r1, r2;
#if defined(__x86_64__)
    register uintptr_t ax __asm__("rax") = trap;
    register uintptr_t dx __asm__("rdx") = args[2];
    register uintptr_t r10 __asm__("r10") = args[3];
    register uintptr_t r8 __asm__("r8") = args[4];
    register uintptr_t r9 __asm__("r9") = args[5];
    __asm__ volatile("syscall" : "+a"(ax), "+d"(dx)
                     : "D"(args[0]), "S"(args[1]), "r"(r10), "r"(r8), "r"(r9)
                     : "rcx", "r11", "cc", "memory");
    r1 = ax; r2 = dx;
#else
    register uintptr_t x0 __asm__("x0") = args[0];
    register uintptr_t x1 __asm__("x1") = args[1];
    register uintptr_t x2 __asm__("x2") = args[2];
    register uintptr_t x3 __asm__("x3") = args[3];
    register uintptr_t x4 __asm__("x4") = args[4];
    register uintptr_t x5 __asm__("x5") = args[5];
    register uintptr_t x8 __asm__("x8") = trap;
    __asm__ volatile("svc 0" : "+r"(x0), "+r"(x1)
                     : "r"(x2), "r"(x3), "r"(x4), "r"(x5), "r"(x8)
                     : "cc", "memory");
    r1 = x0; r2 = x1;
#endif
    if (r1 >= (uintptr_t)-4095)
        return (kernel_result){(uintptr_t)-1, 0, (uintptr_t)-(intptr_t)r1};
    return (kernel_result){r1, r2, 0};
}

/* The raw mask includes libc's private signals, unlike pthread_sigmask.
 * Once READY is published, GC or setxid handlers cannot park a participant
 * behind another participant that this barrier has already stopped. */
static int allcall_mask(const uint64_t *set, uint64_t *old)
{
    uintptr_t args[6] = {SIG_SETMASK, (uintptr_t)set, (uintptr_t)old, sizeof(uint64_t)};
    return (int)kernel_syscall(SYS_rt_sigprocmask, args).error;
}

static void allcall_wake(void)
{
    uintptr_t args[6] = {(uintptr_t)&allcall_phase, FUTEX_WAKE_PRIVATE, 0x7fffffff};
    kernel_syscall(SYS_futex, args);
}

static void allcall_wait(unsigned phase)
{
    while (__atomic_load_n(&allcall_phase, __ATOMIC_ACQUIRE) == phase) {
        uintptr_t args[6] = {(uintptr_t)&allcall_phase, FUTEX_WAIT_PRIVATE, phase};
        kernel_syscall(SYS_futex, args);
    }
}

static uint64_t allcall_time(void)
{
    struct timespec now;
    uintptr_t args[6] = {CLOCK_MONOTONIC, (uintptr_t)&now};
    if (kernel_syscall(SYS_clock_gettime, args).error) return UINT64_MAX;
    return (uint64_t)now.tv_sec * 1000000000u + now.tv_nsec;
}

static void allcall_pause(void)
{
    const struct timespec pause = {0, 100000};
    uintptr_t args[6] = {(uintptr_t)&pause};
    kernel_syscall(SYS_nanosleep, args);
}

static int handle_allthreads(thread_node *node)
{
    unsigned expected = ALL_ARMED;
    if (!__atomic_compare_exchange_n(&node->allcall_state, &expected, ALL_CLAIMED,
                                    0, __ATOMIC_ACQ_REL, __ATOMIC_ACQUIRE))
        return 0;
    uint64_t mask = UINT64_MAX, previous;
    if (allcall_mask(&mask, &previous)) {
        __atomic_store_n(&node->allcall_state, ALL_FAILED, __ATOMIC_RELEASE);
        return 1;
    }
    __atomic_store_n(&node->allcall_state, ALL_READY, __ATOMIC_RELEASE);
    allcall_wait(PHASE_PREPARE);
    if (__atomic_load_n(&allcall_phase, __ATOMIC_ACQUIRE) == PHASE_RUN) {
        kernel_result result = kernel_syscall(allcall_trap, allcall_args);
        node->allcall_r1 = result.r1;
        node->allcall_r2 = result.r2;
        node->allcall_error = result.error;
        __atomic_store_n(&node->allcall_state, ALL_DONE, __ATOMIC_RELEASE);
        allcall_wait(PHASE_RUN);
    }
    /* No shared request reads after RELEASED. Pending GC signals may run as
     * soon as the old mask is restored, so acknowledge before restoring it. */
    __atomic_store_n(&node->allcall_state, ALL_RELEASED, __ATOMIC_RELEASE);
    allcall_mask(&previous, 0);
    return 1;
}

/* Enumeration is a completeness check on the registered, stopped cohort,
 * not a source of thread handles. Raw getdents avoids allocator/libc locks
 * that a parked thread might hold. It is repeated after every target parks. */
static int allcall_complete_cohort(void)
{
    uintptr_t args[6] = {(uintptr_t)AT_FDCWD, (uintptr_t)"/proc/self/task",
                         O_RDONLY | O_DIRECTORY | O_CLOEXEC};
    kernel_result opened = kernel_syscall(SYS_openat, args);
    if (opened.error) return ENOTSUP;
    uintptr_t fd = opened.r1;
    unsigned count = 0, registered = 0;
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) registered++;
    int error = 0;
    for (;;) {
        unsigned char buffer[1024];
        uintptr_t readargs[6] = {fd, (uintptr_t)buffer, sizeof(buffer)};
        kernel_result read = kernel_syscall(SYS_getdents64, readargs);
        if (read.error) { error = ENOTSUP; break; }
        if (!read.r1) break;
        for (size_t pos = 0; pos < read.r1;) {
            if (read.r1-pos < 20) { error = EAGAIN; break; }
            uint16_t len;
            memcpy(&len, buffer+pos+16, sizeof(uint16_t));
            if (len < 20 || len > read.r1-pos) { error = EAGAIN; break; }
            uintptr_t tid = 0;
            size_t i = pos+19;
            while (i < pos+len && buffer[i] >= '0' && buffer[i] <= '9') {
                tid = tid*10 + (buffer[i]-'0');
                i++;
            }
            if (tid && i < pos+len && buffer[i] == 0) {
                thread_node *n = llgo_debugger_threads_v1;
                while (n && n->debugger_thread_id != tid) n = n->next;
                if (!n) { error = ENOTSUP; break; }
                count++;
            }
            pos += len;
        }
        if (error) break;
    }
    uintptr_t closeargs[6] = {fd};
    kernel_syscall(SYS_close, closeargs);
    return error ? error : (count == registered ? 0 : EAGAIN);
}

static int allcall_allowed_trap(uintptr_t trap)
{
    switch (trap) {
    /* Kernel credentials are per-thread; these setters return zero on
     * success. Unlisted syscalls must not execute even on the caller. */
    case SYS_setuid: case SYS_setgid: case SYS_setreuid: case SYS_setregid:
    case SYS_setresuid: case SYS_setresgid: case SYS_setgroups:
        return 1;
    default: return 0;
    }
}

static void allcall_fatal(void)
{
    static const char message[] = "runtime: AllThreadsSyscall results differ or a running syscall blocked\n";
    uintptr_t writeargs[6] = {2, (uintptr_t)message, sizeof(message)-1};
    kernel_syscall(SYS_write, writeargs);
    uintptr_t exitargs[6] = {2};
    kernel_syscall(SYS_exit_group, exitargs);
    __builtin_trap();
}

static void allcall_cancel(void)
{
    __atomic_store_n(&allcall_phase, PHASE_CANCEL, __ATOMIC_RELEASE);
    allcall_wake();
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
        unsigned expected = ALL_ARMED;
        __atomic_compare_exchange_n(&n->allcall_state, &expected, ALL_IDLE, 0,
                                   __ATOMIC_ACQ_REL, __ATOMIC_ACQUIRE);
    }
    /* A claimed handler can be delayed by a previously pending GC signal.
     * Its per-node state stays valid; the next call rejects an active state
     * rather than reusing request arguments before that handler acknowledges. */
}

void llgo_traceback_all_threads_syscall(llgo_allthreads_call *call)
{
    call->r1 = (uintptr_t)-1; call->r2 = 0; call->error = ENOTSUP;
    if (!__llgo_allthreads_executable || !current || !allcall_allowed_trap(call->trap)) return;
    pthread_mutex_lock(&credential_lock);
    pthread_mutex_lock(&registry_lock);
    /* Remain suspendible while acquiring runtime locks. A GC may have
     * suspended their owner and must be allowed to finish before we park it. */
    uint64_t full = UINT64_MAX, previous;
    if (allcall_mask(&full, &previous)) {
        pthread_mutex_unlock(&registry_lock);
        pthread_mutex_unlock(&credential_lock);
        return;
    }
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
        if (!n->owned) goto finish;
        unsigned state = __atomic_load_n(&n->allcall_state, __ATOMIC_ACQUIRE);
        /* Teardown publishes dead before waiting for registry_lock. Do not
         * repeatedly interrupt an exiting thread while it tries to acquire
         * this lock; let unregister complete before starting a new cohort. */
        if (!n->attached || __atomic_load_n(&n->state, __ATOMIC_ACQUIRE) == 6 ||
            (state != ALL_IDLE && state != ALL_RELEASED && state != ALL_FAILED)) {
            call->error = EAGAIN; goto finish;
        }
    }
    call->error = allcall_complete_cohort();
    if (call->error) goto finish;
    allcall_trap = call->trap;
    memcpy(allcall_args, call->args, sizeof(allcall_args));
    __atomic_store_n(&allcall_phase, PHASE_PREPARE, __ATOMIC_RELEASE);
    uintptr_t pidargs[6] = {0};
    uintptr_t pid = kernel_syscall(SYS_getpid, pidargs).r1;
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
        __atomic_store_n(&n->allcall_state, n == current ? ALL_IDLE : ALL_ARMED, __ATOMIC_RELEASE);
        /* pthread_kill can take libc's target exit lock, which a parked
         * participant may hold. The locked registry already owns each TID;
         * use the raw transport without acquiring another thread's locks. */
        uintptr_t signalargs[6] = {pid, n->debugger_thread_id, SIGURG};
        if (n != current && kernel_syscall(SYS_tgkill, signalargs).error) {
            call->error = EAGAIN; allcall_cancel(); goto finish;
        }
    }
    uint64_t deadline = allcall_time()+1000000000u;
    for (;;) {
        int ready = 1;
        for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
            if (n == current) continue;
            unsigned state = __atomic_load_n(&n->allcall_state, __ATOMIC_ACQUIRE);
            if (state == ALL_FAILED) { call->error = ENOTSUP; allcall_cancel(); goto finish; }
            if (state != ALL_READY) ready = 0;
        }
        if (ready) break;
        if (allcall_time() > deadline) { call->error = EAGAIN; allcall_cancel(); goto finish; }
        allcall_pause();
    }
    call->error = allcall_complete_cohort();
    if (call->error) { allcall_cancel(); goto finish; }
    kernel_result result = kernel_syscall(allcall_trap, allcall_args);
    if (result.error) {
        /* Match syscall.AllThreadsSyscall: failure on the calling thread
         * returns immediately without applying the call to other threads. */
        call->r1 = result.r1; call->r2 = result.r2; call->error = result.error;
        allcall_cancel();
        goto finish;
    }
    __atomic_store_n(&allcall_phase, PHASE_RUN, __ATOMIC_RELEASE);
    allcall_wake();
    deadline = allcall_time()+5000000000u;
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
        if (n == current) continue;
        while (__atomic_load_n(&n->allcall_state, __ATOMIC_ACQUIRE) != ALL_DONE) {
            if (allcall_time() > deadline) allcall_fatal();
            allcall_pause();
        }
        if (n->allcall_r1 != result.r1 || n->allcall_r2 != result.r2 || n->allcall_error != result.error)
            allcall_fatal();
    }
    call->r1 = result.r1; call->r2 = result.r2; call->error = result.error;
    __atomic_store_n(&allcall_phase, PHASE_RELEASE, __ATOMIC_RELEASE);
    allcall_wake();
    deadline = allcall_time()+5000000000u;
    for (thread_node *n = llgo_debugger_threads_v1; n; n = n->next) {
        if (n == current) continue;
        while (__atomic_load_n(&n->allcall_state, __ATOMIC_ACQUIRE) != ALL_RELEASED) {
            if (allcall_time() > deadline) allcall_fatal();
            allcall_pause();
        }
    }
finish:
    pthread_mutex_unlock(&registry_lock);
    pthread_mutex_unlock(&credential_lock);
    allcall_mask(&previous, 0);
}
#else
void llgo_traceback_all_threads_syscall(llgo_allthreads_call *call)
{
    call->r1 = (uintptr_t)-1; call->r2 = 0; call->error = ENOTSUP;
}
#endif
#endif
