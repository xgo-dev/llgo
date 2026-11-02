#define _GNU_SOURCE 1
#include "../../../../runtime/internal/stacktrace/_wrap/traceback.h"
#include <assert.h>
#include <errno.h>
#include <linux/filter.h>
#include <linux/seccomp.h>
#include <pthread.h>
#include <sched.h>
#include <signal.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stddef.h>
#include <stdlib.h>
#include <string.h>
#include <sys/prctl.h>
#include <sys/syscall.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <time.h>
#include <unistd.h>

int __llgo_allthreads_executable = 1;
extern void llgo_traceback_mark_owned(void *);

typedef struct {
    void *node;
    atomic_int ready, stop, masked, unblock, calls_done;
    uintptr_t filtered_trap;
    unsigned filter_action, calls;
} worker;

static llgo_allthreads_call call(uintptr_t);
static void retry_pause(void);

/* A thread-local filter observes an actual kernel entry, without adding a
 * foreign observer thread or instrumenting the protocol under test. */
static void filter_syscall(uintptr_t trap, unsigned action)
{
    struct sock_filter instructions[] = {
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS, offsetof(struct seccomp_data, nr)),
        BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, trap, 0, 1),
        BPF_STMT(BPF_RET | BPF_K, action),
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ALLOW),
    };
    struct sock_fprog program = {
        .len = sizeof(instructions)/sizeof(instructions[0]),
        .filter = instructions,
    };
    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) ||
        prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &program)) {
        /* Some kernels, outer sandboxes, and user-mode emulators cannot
         * install a filter. Keep cohort tests usable in those environments. */
        if (errno == ENOSYS || errno == EINVAL || errno == EPERM || errno == EACCES) {
            fputs("allthreads: seccomp unavailable\n", stderr);
            _exit(77);
        }
        assert(!"failed to install seccomp filter");
    }
}

static void *work(void *raw)
{
    worker *w = raw;
    llgo_traceback_attach(w->node);
    if (w->filter_action) filter_syscall(w->filtered_trap, w->filter_action);
    if (atomic_load(&w->masked)) {
        sigset_t mask;
        sigemptyset(&mask); sigaddset(&mask, SIGURG);
        assert(!pthread_sigmask(SIG_BLOCK, &mask, 0));
    }
    atomic_store(&w->ready, 1);
    for (unsigned i = 0; i < w->calls;) {
        llgo_allthreads_call c = call(SYS_getpid);
        if (!c.error) {
            assert(c.r1 == (uintptr_t)getpid());
            i++;
        } else {
            assert(c.error == EAGAIN);
        }
        retry_pause();
    }
    atomic_store(&w->calls_done, 1);
    while (!atomic_load(&w->stop)) {
        if (atomic_load(&w->unblock)) {
            sigset_t mask;
            sigemptyset(&mask); sigaddset(&mask, SIGURG);
            assert(!pthread_sigmask(SIG_UNBLOCK, &mask, 0));
            atomic_store(&w->unblock, 0);
        }
        sched_yield();
    }
    llgo_traceback_unregister(w->node);
    return 0;
}

static pthread_t start(worker *w, unsigned id, unsigned parent)
{
    w->node = llgo_traceback_register(id, parent, 0);
    assert(w->node);
    pthread_t thread;
    assert(!pthread_create(&thread, 0, work, w));
    while (!atomic_load(&w->ready)) sched_yield();
    return thread;
}

static void stop(worker *w, pthread_t thread)
{
    atomic_store(&w->stop, 1);
    assert(!pthread_join(thread, 0));
}

static llgo_allthreads_call call(uintptr_t trap)
{
    llgo_allthreads_call c = {.trap = trap};
    if (trap == SYS_setresuid)
        c.args[0] = c.args[1] = c.args[2] = (uintptr_t)-1;
    errno = EDOM;
    llgo_traceback_all_threads_syscall(&c);
    assert(errno == EDOM);
    return c;
}

static void retry_pause(void)
{
    const struct timespec pause = {0, 1000000};
    nanosleep(&pause, 0);
}

static llgo_allthreads_call await_trap(uintptr_t trap, uintptr_t expected)
{
    struct timespec start, now;
    assert(!clock_gettime(CLOCK_MONOTONIC, &start));
    for (;;) {
        llgo_allthreads_call c = call(trap);
        if (c.error == expected) return c;
        assert(c.error == EAGAIN || c.error == ENOTSUP);
        assert(!clock_gettime(CLOCK_MONOTONIC, &now));
        assert(now.tv_sec-start.tv_sec < 5);
        retry_pause();
    }
}

static llgo_allthreads_call await_result(uintptr_t expected)
{
    return await_trap(SYS_setresuid, expected);
}

static void *foreign(void *raw)
{
    atomic_int *stop = raw;
    while (!atomic_load(stop)) sched_yield();
    return 0;
}

static void *creation(void *raw)
{
    worker *w = raw;
    llgo_traceback_attach(w->node);
    atomic_store(&w->ready, 1);
    for (unsigned i = 0; i < 150; i++) {
        worker child = {0};
        pthread_t thread = start(&child, 100+i, 10);
        stop(&child, thread);
        if ((i+1)%25 == 0) fprintf(stderr, "allthreads: created/joined %u threads\n", i+1);
    }
    atomic_store(&w->stop, 1);
    llgo_traceback_unregister(w->node);
    return 0;
}

int main(int argc, char **argv)
{
    void *main_node = llgo_traceback_register(1, 0, 0);
    assert(main_node);
    llgo_traceback_attach(main_node);
    llgo_traceback_mark_owned(main_node);
    if (argc == 2) {
        worker peer = {.filtered_trap = SYS_setresuid};
        if (!strcmp(argv[1], "caller-error")) {
            peer.filter_action = SECCOMP_RET_KILL_PROCESS;
        } else if (!strcmp(argv[1], "peer-error")) {
            peer.filter_action = SECCOMP_RET_ERRNO | EPERM;
        } else if (!strcmp(argv[1], "probe-mismatch")) {
            peer.filtered_trap = SYS_getpid;
            /* ERRNO with data zero returns a successful zero, different
             * from the caller's PID. Check values as well as errno. */
            peer.filter_action = SECCOMP_RET_ERRNO;
        } else {
            assert(!"unknown protocol scenario");
        }
        pthread_t thread = start(&peer, 2, 1);
        if (!strcmp(argv[1], "caller-error")) {
            filter_syscall(SYS_setresuid, SECCOMP_RET_ERRNO | EACCES);
            llgo_allthreads_call c = call(SYS_setresuid);
            assert(c.error == EACCES && c.r1 == (uintptr_t)-1 && c.r2 == 0);
            /* The peer would kill the process if it ran the failed call.
             * Cancellation must leave the cohort reusable. */
            assert(await_trap(SYS_getpid, 0).r1 == (uintptr_t)getpid());
            stop(&peer, thread);
            llgo_traceback_unregister(main_node);
            puts("allthreads-caller-error-ok");
            return 0;
        }
        call(peer.filtered_trap);
        assert(!"result disagreement must terminate the process");
    }
    worker children[3] = {0};
    pthread_t threads[3];
    for (unsigned i = 0; i < 3; i++) threads[i] = start(&children[i], i+2, 1);
    fputs("allthreads: owned-cohort calls\n", stderr);
    for (unsigned i = 0; i < 200; i++) {
        llgo_allthreads_call c = call(SYS_getpid);
        assert(!c.error && c.r1 == (uintptr_t)getpid());
        c = call(SYS_setresuid);
        assert(!c.error && c.r1 == 0);
    }
    /* Reject process-wide operations before any participant changes state.
     * Replaying umask would return different old masks and abort mid-call. */
    mode_t previous_mask = umask(0022);
    const uintptr_t rejected[] = {SYS_umask, SYS_setpgid, SYS_setsid,
        SYS_sethostname, SYS_setdomainname, SYS_chdir, SYS_fchdir, SYS_chroot,
        SYS_rt_sigprocmask, SYS_gettid};
    for (size_t i = 0; i < sizeof(rejected)/sizeof(rejected[0]); i++) {
        llgo_allthreads_call c = call(rejected[i]);
        assert(c.error == ENOTSUP && c.r1 == (uintptr_t)-1);
    }
    assert(umask(previous_mask) == 0022);
    llgo_allthreads_call c = call((uintptr_t)-1);
    assert(c.error == ENOTSUP && c.r1 == (uintptr_t)-1);
    c = call(SYS_rt_sigprocmask);
    assert(c.error == ENOTSUP);

    __llgo_allthreads_executable = 0;
    c = call(SYS_setresuid);
    assert(c.error == ENOTSUP);
    __llgo_allthreads_executable = 1;

    fputs("allthreads: pending creation\n", stderr);
    void *pending = llgo_traceback_register(9, 1, 0);
    assert(pending);
    c = call(SYS_getpid);
    assert(c.error == EAGAIN);
    llgo_traceback_unregister(pending);
    assert(await_trap(SYS_getpid, 0).r1 == (uintptr_t)getpid());

    fputs("allthreads: concurrent callers\n", stderr);
    worker callers[2] = {{.calls = 25}, {.calls = 25}};
    pthread_t callers_threads[2];
    for (unsigned i = 0; i < 2; i++) callers_threads[i] = start(&callers[i], 11+i, 1);
    for (unsigned i = 0; i < 25; i++)
        assert(await_trap(SYS_getpid, 0).r1 == (uintptr_t)getpid());
    for (unsigned i = 0; i < 2; i++)
        while (!atomic_load(&callers[i].calls_done)) sched_yield();
    for (unsigned i = 0; i < 2; i++) stop(&callers[i], callers_threads[i]);

    fputs("allthreads: foreign/adopted rejection\n", stderr);

    atomic_int foreign_stop = 0;
    pthread_t thread;
    assert(!pthread_create(&thread, 0, foreign, &foreign_stop));
    c = await_result(ENOTSUP);
    assert(c.error == ENOTSUP);
    atomic_store(&foreign_stop, 1);
    assert(!pthread_join(thread, 0));

    worker adopted = {0};
    thread = start(&adopted, 7, 0);
    c = call(SYS_setresuid);
    assert(c.error == ENOTSUP);
    stop(&adopted, thread);

    /* pthread_join's clear-child-TID wake can precede removal from procfs.
     * Establish the complete owned cohort before isolating signal masking. */
    assert(await_result(0).r1 == 0);

    fputs("allthreads: masked signal and cancellation\n", stderr);
    worker masked = {.masked = 1};
    thread = start(&masked, 8, 1);
    c = call(SYS_setresuid);
    fprintf(stderr, "allthreads: masked result=%zu\n", (size_t)c.error);
    assert(c.error == EAGAIN);
    atomic_store(&masked.unblock, 1);
    while (atomic_load(&masked.unblock)) sched_yield();
    c = await_result(0);
    assert(!c.error);
    stop(&masked, thread);

    fputs("allthreads: concurrent creation/exit\n", stderr);
    worker creator = {0};
    creator.node = llgo_traceback_register(10, 1, 0);
    assert(!pthread_create(&thread, 0, creation, &creator));
    while (!atomic_load(&creator.ready)) sched_yield();
    unsigned attempts = 0;
    while (!atomic_load(&creator.stop)) {
        c = call(SYS_setresuid);
        assert(!c.error || c.error == EAGAIN || c.error == ENOTSUP);
        attempts++;
        /* Do not starve attachment/unregister waiters by repeatedly taking
         * their registry lock while they try to finish a pthread lifetime. */
        retry_pause();
    }
    assert(!pthread_join(thread, 0));
    assert(attempts);
    assert(await_result(0).r1 == 0);
    fputs("allthreads: final teardown\n", stderr);
    for (unsigned i = 0; i < 3; i++) stop(&children[i], threads[i]);
    llgo_traceback_unregister(main_node);
    puts("allthreads-ok");
    return 0;
}
