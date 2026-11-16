#include <errno.h>
#include <pthread.h>
#include <sched.h>
#include <stdatomic.h>
#include <stdio.h>
#include <stdlib.h>

// Target only the armed test thread's next wait, an ordinary channel waiter.
// Select waiters have additional lifetime protection through channel cleanup.
static _Thread_local int armed;
static pthread_cond_t *target_cond;
static pthread_mutex_t *target_mutex;
static _Atomic int waiting;
static _Atomic int resume_wait;
static _Atomic int signaling;

static void fail(const char *message) {
    fprintf(stderr, "%s\n", message);
    _Exit(90);
}

void llgo_lifetime_arm(void) {
    armed = 1;
}

void llgo_lifetime_await_wait(void) {
    while (!atomic_load(&waiting)) {
        sched_yield();
    }
}

int llgo_lifetime_cond_wait(pthread_cond_t *cond, pthread_mutex_t *mutex) {
    if (!armed) {
        return pthread_cond_wait(cond, mutex);
    }
    armed = 0;
    target_cond = cond;
    target_mutex = mutex;
    if (pthread_mutex_unlock(mutex) != 0) {
        fail("cannot release waiter mutex");
    }
    atomic_store(&waiting, 1);
    while (!atomic_load(&resume_wait)) {
        sched_yield();
    }
    // A legal spurious return: the mutex must be reacquired before the
    // channel waiter can observe completion and release or reuse its signal.
    return pthread_mutex_lock(mutex);
}

int llgo_lifetime_cond_signal(pthread_cond_t *cond) {
    if (!atomic_load(&waiting) || cond != target_cond) {
        return pthread_cond_signal(cond);
    }
    atomic_store(&signaling, 1);
    // The waiter is still behind resume_wait, so it cannot own this mutex.
    // If finish has already unlocked, force the spurious return and hold
    // this call until the completion check detects the lifetime violation.
    // No timing assumptions or access to a destroyed pthread object are needed.
    int result = pthread_mutex_trylock(target_mutex);
    if (result == 0) {
        pthread_mutex_unlock(target_mutex);
        atomic_store(&resume_wait, 1);
        for (;;) {
            sched_yield();
        }
    }
    if (result != EBUSY) {
        fail("unexpected waiter mutex trylock result");
    }
    atomic_store(&resume_wait, 1);
    result = pthread_cond_signal(cond);
    atomic_store(&signaling, 0);
    return result;
}

int llgo_lifetime_cond_destroy(pthread_cond_t *cond) {
    if (atomic_load(&waiting) && cond == target_cond) {
        if (atomic_load(&signaling)) {
            fail("condition destroyed while its signal call is active");
        }
    }
    return pthread_cond_destroy(cond);
}

void llgo_lifetime_check(void) {
    // A completed waiter may cache its condition variable until its G exits.
    // Check that the armed wait ran and notification finished before the
    // waiter was released for destruction or reuse.
    if (!atomic_load(&waiting) || !atomic_load(&resume_wait)) {
        fail("channel waiter lifetime was not exercised");
    }
    if (atomic_load(&signaling)) {
        fail("channel waiter completed while its signal call is active");
    }
    atomic_store(&waiting, 0);
}
