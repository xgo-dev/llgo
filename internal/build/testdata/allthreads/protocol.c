#define _GNU_SOURCE 1
#include "../../../../runtime/internal/stacktrace/_wrap/traceback.h"
#include <assert.h>
#include <errno.h>
#include <linux/audit.h>
#include <linux/filter.h>
#include <linux/seccomp.h>
#include <pthread.h>
#include <sched.h>
#include <signal.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
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
    atomic_int ready, stop, masked, unblock;
    unsigned filter_trap, filter_action;
} worker;

static void filter_syscall(unsigned trap, unsigned action)
{
#if defined(__x86_64__)
    const unsigned architecture = AUDIT_ARCH_X86_64;
#else
    const unsigned architecture = AUDIT_ARCH_AARCH64;
#endif
    struct sock_filter rules[] = {
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS, offsetof(struct seccomp_data, arch)),
        BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, architecture, 1, 0),
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ALLOW),
        BPF_STMT(BPF_LD | BPF_W | BPF_ABS, offsetof(struct seccomp_data, nr)),
        BPF_JUMP(BPF_JMP | BPF_JEQ | BPF_K, trap, 0, 1),
        BPF_STMT(BPF_RET | BPF_K, action),
        BPF_STMT(BPF_RET | BPF_K, SECCOMP_RET_ALLOW),
    };
    struct sock_fprog filter = {sizeof(rules)/sizeof(rules[0]), rules};
    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0) ||
        prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &filter)) _exit(77);
}

static void *work(void *raw)
{
    worker *w = raw;
    llgo_traceback_attach(w->node);
    if (w->filter_action) {
        filter_syscall(w->filter_trap, w->filter_action);
        if (w->filter_trap == SYS_getpid &&
            (syscall(SYS_getpid) != -1 || errno != EPERM)) _exit(77);
    }
    if (atomic_load(&w->masked)) {
        sigset_t mask;
        sigemptyset(&mask); sigaddset(&mask, SIGURG);
        assert(!pthread_sigmask(SIG_BLOCK, &mask, 0));
    }
    atomic_store(&w->ready, 1);
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
    w->node = llgo_traceback_register(id, parent, 0, 0);
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

static llgo_allthreads_call await_result(uintptr_t expected)
{
    struct timespec start, now;
    assert(!clock_gettime(CLOCK_MONOTONIC, &start));
    for (;;) {
        llgo_allthreads_call c = call(SYS_setresuid);
        if (c.error == expected) return c;
        assert(c.error == EAGAIN || c.error == ENOTSUP);
        assert(!clock_gettime(CLOCK_MONOTONIC, &now));
        assert(now.tv_sec-start.tv_sec < 5);
        retry_pause();
    }
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
    void *main_node = llgo_traceback_register(1, 0, 0, 0);
    assert(main_node);
    llgo_traceback_attach(main_node);
    llgo_traceback_mark_owned(main_node);
    if (argc == 2) {
        const int mismatch = !strcmp(argv[1], "mismatch");
        worker peer = {0};
        if (mismatch) {
            peer.filter_trap = SYS_getpid;
            peer.filter_action = SECCOMP_RET_ERRNO | EPERM;
        } else {
            filter_syscall(SYS_setgroups, SECCOMP_RET_ERRNO | E2BIG);
            if (syscall(SYS_setgroups, -1, 0) != -1 || errno != E2BIG) return 77;
            peer.filter_trap = SYS_setgroups;
            peer.filter_action = SECCOMP_RET_KILL_PROCESS;
        }
        pthread_t thread = start(&peer, 2, 1);
        llgo_allthreads_call result = call(mismatch ? SYS_getpid : SYS_setgroups);
        assert(!mismatch); /* A peer result mismatch must terminate with 2. */
        assert(result.error == E2BIG && result.r1 == (uintptr_t)-1);
        /* Killing the process on a peer's setgroups proves a caller error
         * never executed it. Its handler must also be reusable afterward. */
        for (unsigned retry = 0; retry < 5000; retry++) {
            result = call(SYS_getpid);
            if (result.error != EAGAIN) break;
            retry_pause();
        }
        assert(!result.error && result.r1 == (uintptr_t)getpid());
        stop(&peer, thread);
        llgo_traceback_unregister(main_node);
        return 0;
    }
    worker children[3] = {0};
    pthread_t threads[3];
    for (unsigned i = 0; i < 3; i++) threads[i] = start(&children[i], i+2, 1);
    fputs("allthreads: owned-cohort calls\n", stderr);
    for (unsigned i = 0; i < 200; i++) {
        llgo_allthreads_call probe = call(SYS_getpid);
        assert(!probe.error && probe.r1 == (uintptr_t)getpid() && probe.r2 == 0);
        llgo_allthreads_call c = call(SYS_setresuid);
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
    creator.node = llgo_traceback_register(10, 1, 0, 0);
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
    const char *cases[] = {"mismatch", "caller-error"};
    for (unsigned i = 0; i < 2; i++) {
        pid_t child = fork();
        assert(child >= 0);
        if (!child) {
            char *args[] = {argv[0], (char *)cases[i], 0};
            execv(args[0], args);
            _exit(1);
        }
        int status;
        assert(waitpid(child, &status, 0) == child);
        assert(WIFEXITED(status));
        if (WEXITSTATUS(status) == 77) {
            fprintf(stderr, "allthreads: seccomp unavailable for %s\n", cases[i]);
        } else {
            assert(WEXITSTATUS(status) == (i == 0 ? 2 : 0));
            fprintf(stderr, "allthreads: %s result verified\n", cases[i]);
        }
    }
    puts("allthreads-ok");
    return 0;
}
