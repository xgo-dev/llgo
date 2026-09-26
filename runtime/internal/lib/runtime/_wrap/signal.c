#define _POSIX_C_SOURCE 200809L
#if defined(__APPLE__)
#define _DARWIN_C_SOURCE 1
#endif

#include <errno.h>
#include <fcntl.h>
#include <limits.h>
#include <pthread.h>
#include <sched.h>
#include <signal.h>
#include <stddef.h>
#include <string.h>
#include <unistd.h>

extern void llgo_traceback_urgent_action(void (*)(int));
/* All SIGURG actions in this file are simple one-argument handlers. */
static int llgo_signal_sigaction(int sig, const struct sigaction *action,
                                  struct sigaction *old)
{
    if (sig == SIGURG && action && !old && !(action->sa_flags & SA_SIGINFO)) {
        llgo_traceback_urgent_action(action->sa_handler);
        return 0;
    }
    return sigaction(sig, action, old);
}
#define sigaction(signum, action, old) llgo_signal_sigaction(signum, action, old)

_Static_assert(SIGPIPE == 13, "runtime signalPipe assumes SIGPIPE is 13");

#define LLGO_SIGNAL_COUNT 65
#define LLGO_SIGNAL_WORDS ((LLGO_SIGNAL_COUNT + 31) / 32)

_Static_assert(CHAR_BIT * sizeof(unsigned int) == 32,
               "signal bitmaps require 32-bit unsigned int");
#if defined(__clang__) || defined(__GNUC__)
_Static_assert(__atomic_always_lock_free(sizeof(unsigned int), 0),
               "signal-handler atomics must be lock-free");
#endif

static int llgo_signal_pipe[2] = {-1, -1};
static struct sigaction llgo_signal_default_action;
static unsigned int llgo_signal_delivering;
/* Raw ABI shared with LLGO_PROF_SIGNAL_* in profile.c and signalMode* in
 * signal_llgo.go. Keep the numeric values synchronized. */
enum {
    LLGO_SIGNAL_DEFAULT,
    LLGO_SIGNAL_WANTED,
    LLGO_SIGNAL_IGNORED,
};
static unsigned int llgo_signal_modes[LLGO_SIGNAL_COUNT];
static unsigned int llgo_signal_pending[LLGO_SIGNAL_WORDS];
/* Only the dedicated signal reader accesses this snapshot. */
static unsigned int llgo_signal_received[LLGO_SIGNAL_WORDS];

static void llgo_signal_handler(int signum);

static void llgo_signal_wanted_action(struct sigaction *action)
{
    memset(action, 0, sizeof(*action));
    action->sa_handler = llgo_signal_handler;
    sigemptyset(&action->sa_mask);
    action->sa_flags = SA_RESTART;
}

static void llgo_signal_ignored_action(struct sigaction *action)
{
    memset(action, 0, sizeof(*action));
    action->sa_handler = SIG_IGN;
    sigemptyset(&action->sa_mask);
}

static void llgo_signal_handler(int signum)
{
    int saved_errno = errno;
    unsigned int bit;
    unsigned int old;
    const int wake = 1;

    /* signalWaitUntilIdle uses this like the Go runtime's sig.delivering:
     * a pipe marker must not overtake a handler that has already entered. */
    __atomic_add_fetch(&llgo_signal_delivering, 1, __ATOMIC_SEQ_CST);
    if (llgo_signal_pipe[1] >= 0 && signum > 0 &&
        signum < LLGO_SIGNAL_COUNT) {
        unsigned int mode = __atomic_load_n(&llgo_signal_modes[signum],
                                             __ATOMIC_SEQ_CST);

        if (mode == LLGO_SIGNAL_IGNORED)
            goto done;
        if (mode != LLGO_SIGNAL_WANTED) {
            /* The kernel may select this handler immediately before Stop
             * restores SIG_DFL, then enter it only after Stop's pipe barrier.
             * In that case the signal must take its default action rather
             * than be queued after the stopping channel has been removed. */
            if (sigaction(signum, &llgo_signal_default_action, 0) != 0)
                _exit(128 + signum);
            __atomic_sub_fetch(&llgo_signal_delivering, 1,
                               __ATOMIC_SEQ_CST);
            if (raise(signum) != 0)
                _exit(128 + signum);
            errno = saved_errno;
            return;
        }
        bit = 1U << ((unsigned int)signum & 31U);
        old = __atomic_fetch_or(&llgo_signal_pending[(unsigned int)signum / 32U],
                                bit, __ATOMIC_SEQ_CST);
        if ((old & bit) == 0) {
            /* The pending bit preserves this signal if the nonblocking write
             * fails. In particular, EAGAIN means the full pipe already
             * contains a wakeup. */
            ssize_t count;
            do {
                count = write(llgo_signal_pipe[1], &wake, sizeof(wake));
            } while (count < 0 && errno == EINTR);
        }
    }
done:
    __atomic_sub_fetch(&llgo_signal_delivering, 1, __ATOMIC_SEQ_CST);
    errno = saved_errno;
}

int llgo_signal_init(void)
{
    int flags;

    if (llgo_signal_pipe[0] >= 0)
        return llgo_signal_pipe[0];
    if (pipe(llgo_signal_pipe) != 0)
        return -errno;

    memset(&llgo_signal_default_action, 0,
           sizeof(llgo_signal_default_action));
    llgo_signal_default_action.sa_handler = SIG_DFL;
    sigemptyset(&llgo_signal_default_action.sa_mask);

    flags = fcntl(llgo_signal_pipe[1], F_GETFL, 0);
    if (flags < 0 || fcntl(llgo_signal_pipe[1], F_SETFL,
                           flags | O_NONBLOCK) != 0)
        goto fail;
    if (fcntl(llgo_signal_pipe[0], F_SETFD, FD_CLOEXEC) != 0 ||
        fcntl(llgo_signal_pipe[1], F_SETFD, FD_CLOEXEC) != 0)
        goto fail;
    return llgo_signal_pipe[0];

fail:
    flags = errno;
    close(llgo_signal_pipe[0]);
    close(llgo_signal_pipe[1]);
    llgo_signal_pipe[0] = -1;
    llgo_signal_pipe[1] = -1;
    return -flags;
}

int llgo_signal_enable(int signum)
{
    struct sigaction action;
    unsigned int previous;
    int code;

    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT)
        return EINVAL;

    llgo_signal_wanted_action(&action);
    previous = __atomic_exchange_n(&llgo_signal_modes[signum],
                                   LLGO_SIGNAL_WANTED, __ATOMIC_SEQ_CST);
    if (sigaction(signum, &action, 0) == 0)
        return 0;
    code = errno;
    __atomic_store_n(&llgo_signal_modes[signum], previous,
                     __ATOMIC_SEQ_CST);
    return code;
}

int llgo_signal_disable(int signum)
{
    unsigned int previous;
    int code;

    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT)
        return EINVAL;

    previous = __atomic_exchange_n(&llgo_signal_modes[signum],
                                   LLGO_SIGNAL_DEFAULT, __ATOMIC_SEQ_CST);
    if (sigaction(signum, &llgo_signal_default_action, 0) == 0)
        return 0;
    code = errno;
    __atomic_store_n(&llgo_signal_modes[signum], previous,
                     __ATOMIC_SEQ_CST);
    return code;
}

int llgo_signal_ignore(int signum)
{
    struct sigaction action;
    unsigned int previous;
    int code;

    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT)
        return EINVAL;

    llgo_signal_ignored_action(&action);
    previous = __atomic_exchange_n(&llgo_signal_modes[signum],
                                   LLGO_SIGNAL_IGNORED, __ATOMIC_SEQ_CST);
    if (sigaction(signum, &action, 0) == 0)
        return 0;
    code = errno;
    __atomic_store_n(&llgo_signal_modes[signum], previous,
                     __ATOMIC_SEQ_CST);
    return code;
}

/* CPU profiling owns the native SIGPROF disposition while active. Keep the
 * live os/signal handler mode ignored for that whole interval: a handler the
 * kernel selected just before profiling began may enter late and must not
 * forward a timer signal to SIG_DFL. */
int llgo_signal_profile_begin(int signum)
{
    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT)
        return -EINVAL;
    return (int)__atomic_exchange_n(&llgo_signal_modes[signum],
                                    LLGO_SIGNAL_IGNORED,
                                    __ATOMIC_SEQ_CST);
}

/* Restore the exact live mode when profiling failed before its handler or
 * timer became active. Unlike a completed Stop, this path has no pending
 * profiling signal that requires DEFAULT to be normalized to IGNORED. */
int llgo_signal_profile_abort(int signum, int mode)
{
    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT ||
        mode < LLGO_SIGNAL_DEFAULT || mode > LLGO_SIGNAL_IGNORED)
        return EINVAL;
    __atomic_store_n(&llgo_signal_modes[signum], (unsigned int)mode,
                     __ATOMIC_SEQ_CST);
    return 0;
}

int llgo_signal_profile_finish(int signum, int mode)
{
    unsigned int live_mode;

    if (signum <= 0 || signum >= LLGO_SIGNAL_COUNT ||
        mode < LLGO_SIGNAL_DEFAULT || mode > LLGO_SIGNAL_IGNORED)
        return EINVAL;
    live_mode = mode == LLGO_SIGNAL_WANTED ? LLGO_SIGNAL_WANTED
                                           : LLGO_SIGNAL_IGNORED;
    __atomic_store_n(&llgo_signal_modes[signum], live_mode,
                     __ATOMIC_SEQ_CST);
    return 0;
}

/* Apply the os/signal state requested while profiling was active. Default is
 * restored as ignore, matching the Go runtime's protection against a final
 * pending profiling signal. */
int llgo_signal_profile_restore(int signum, int mode)
{
    struct sigaction action;

    if (llgo_signal_profile_finish(signum, mode) != 0)
        return EINVAL;
    if (mode == LLGO_SIGNAL_WANTED)
        llgo_signal_wanted_action(&action);
    else
        llgo_signal_ignored_action(&action);
    if (sigaction(signum, &action, 0) == 0)
        return 0;
    return errno;
}

int llgo_signal_ignore_pipe(void)
{
    struct sigaction action;
    struct sigaction previous;

    memset(&action, 0, sizeof(action));
    action.sa_handler = SIG_IGN;
    sigemptyset(&action.sa_mask);
    if (sigaction(SIGPIPE, &action, &previous) != 0)
        return -errno;
    return previous.sa_handler == SIG_IGN;
}

int llgo_signal_raise_pipe(void)
{
    return raise(SIGPIPE);
}

int llgo_signal_die_pipe(void)
{
    struct sigaction action;
    sigset_t mask;
    int code;

    memset(&action, 0, sizeof(action));
    action.sa_handler = SIG_DFL;
    sigemptyset(&action.sa_mask);
    if (sigaction(SIGPIPE, &action, 0) != 0)
        return errno;

    sigemptyset(&mask);
    sigaddset(&mask, SIGPIPE);
    code = pthread_sigmask(SIG_UNBLOCK, &mask, 0);
    if (code != 0)
        return code;
    if (raise(SIGPIPE) != 0)
        return errno;
    return EIO;
}

/* Place a marker after every signal already written to the pipe. The write
 * end must stay nonblocking for the signal handler, so an ordinary runtime
 * thread retries while the reader makes room. */
int llgo_signal_barrier(void)
{
    const int marker = 0;

    while (__atomic_load_n(&llgo_signal_delivering, __ATOMIC_ACQUIRE) != 0)
        sched_yield();
    for (;;) {
        ssize_t count = write(llgo_signal_pipe[1], &marker, sizeof(marker));
        if (count == (ssize_t)sizeof(marker))
            return 0;
        if (count < 0 && errno == EINTR)
            continue;
        if (count < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            sched_yield();
            continue;
        }
        return count < 0 ? errno : EIO;
    }
}

static int llgo_signal_take_received(void)
{
    unsigned int signum;

    for (signum = 1; signum < LLGO_SIGNAL_COUNT; signum++) {
        unsigned int bit = 1U << (signum & 31U);
        unsigned int *word = &llgo_signal_received[signum / 32U];

        if ((*word & bit) != 0) {
            *word &= ~bit;
            return (int)signum;
        }
    }
    return 0;
}

static void llgo_signal_receive_pending(void)
{
    unsigned int word;

    for (word = 0; word < LLGO_SIGNAL_WORDS; word++)
        llgo_signal_received[word] =
            __atomic_exchange_n(&llgo_signal_pending[word], 0,
                                __ATOMIC_SEQ_CST);
}

int llgo_signal_recv(int fd, int *signum)
{
    if (signum == NULL)
        return EINVAL;

    for (;;) {
        int received;
        int token;
        unsigned char *out;
        size_t offset = 0;

        received = llgo_signal_take_received();
        if (received != 0) {
            *signum = received;
            return 0;
        }

        out = (unsigned char *)&token;
        while (offset < sizeof(token)) {
            ssize_t count = read(fd, out + offset, sizeof(token) - offset);
            if (count > 0) {
                offset += (size_t)count;
                continue;
            }
            if (count < 0 && errno == EINTR)
                continue;
            return count == 0 ? EPIPE : errno;
        }
        if (token == 0) {
            *signum = 0;
            return 0;
        }
        llgo_signal_receive_pending();
    }
}
