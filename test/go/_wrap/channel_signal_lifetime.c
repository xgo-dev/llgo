#include <errno.h>
#include <pthread.h>
#include <sched.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

int llgo_bind_channel_address_wait(void);
int llgo_channel_address_wait(uint32_t *notified);
int llgo_channel_address_wake(uint32_t *notified);

// Target only the armed test thread's next wait, an ordinary channel waiter.
// Select waiters have additional lifetime protection through channel cleanup.
static _Thread_local int armed;
static _Thread_local unsigned address_wait_attempts;
static _Thread_local pthread_mutex_t *unlocked;
static uint32_t *target_address;
static pthread_cond_t *target_cond;
static pthread_mutex_t *target_mutex;
static _Atomic int waiting;
static _Atomic int resume_wait;
static _Atomic int signaling;
static _Atomic int destroyed;
static _Atomic int address_woken;

static void fail(const char *message) {
    fprintf(stderr, "%s\n", message);
    _Exit(90);
}

void llgo_lifetime_arm(void) {
    armed = 1;
    unlocked = NULL;
    address_wait_attempts = 0;
}

int llgo_lifetime_bind_address_wait(void) {
    const char *backend = getenv("LLGO_CHANNEL_LIFETIME_BACKEND");
    if (backend == NULL || strcmp(backend, "address") != 0) {
        return 0;
    }
    if (!llgo_bind_channel_address_wait()) {
        _Exit(77);
    }
    return 1;
}

int llgo_lifetime_mutex_unlock(pthread_mutex_t *mutex) {
    if (armed) {
        unlocked = mutex;
    }
    return pthread_mutex_unlock(mutex);
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
    // channel waiter can observe completion and destroy its signal.
    return pthread_mutex_lock(mutex);
}

static void resume_notification(void) {
    atomic_store(&signaling, 1);
    // The waiter is still behind resume_wait, so it cannot own this mutex.
    // If finish has already unlocked, force the spurious return and hold
    // this call until destroy detects the lifetime violation. No timing
    // assumptions or access to a destroyed pthread object are needed.
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
}

int llgo_lifetime_cond_signal(pthread_cond_t *cond) {
    if (!atomic_load(&waiting) || cond != target_cond) {
        return pthread_cond_signal(cond);
    }
    resume_notification();
    int result = pthread_cond_signal(cond);
    atomic_store(&signaling, 0);
    return result;
}

int llgo_lifetime_address_wait(uint32_t *notified) {
    if (!armed) {
        return llgo_channel_address_wait(notified);
    }
    // Exercise the retry logic before the armed waiter reaches its test park.
    static const int transient[] = {EINTR, ENOMEM, EFAULT};
    if (address_wait_attempts < sizeof(transient) / sizeof(transient[0])) {
        errno = transient[address_wait_attempts++];
        return -1;
    }
    armed = 0;
    target_mutex = unlocked;
    target_address = notified;
    atomic_store(&waiting, 1);
    while (!atomic_load(&resume_wait)) {
        sched_yield();
    }
    return 0;
}

int llgo_lifetime_address_wake(uint32_t *notified) {
    if (!atomic_load(&waiting) || notified != target_address) {
        return llgo_channel_address_wake(notified);
    }
    if (target_mutex != NULL) {
        resume_notification();
        atomic_store(&signaling, 0);
    } else {
        // Lock-free 4-state address path (sigWaking -> sigDone): release the
        // waiter from wait_on_address while wake is still active and verify
        // that parkFast blocks in waitFastDone until after wake returns.
        atomic_store(&signaling, 1);
        atomic_store(&resume_wait, 1);
        usleep(2000);
        atomic_store(&address_woken, 1);
        atomic_store(&signaling, 0);
    }
    // The test rendezvous substitutes for a wait in the kernel.
    errno = ENOENT;
    return -1;
}

int llgo_lifetime_mutex_destroy(pthread_mutex_t *mutex) {
    if (atomic_load(&waiting) && target_address != NULL && mutex == target_mutex) {
        if (atomic_load(&signaling)) {
            fail("address waiter mutex destroyed while its wake call is active");
        }
        atomic_store(&destroyed, 1);
    }
    return pthread_mutex_destroy(mutex);
}

int llgo_lifetime_cond_destroy(pthread_cond_t *cond) {
    if (atomic_load(&waiting) && cond == target_cond) {
        if (atomic_load(&signaling)) {
            fail("condition destroyed while its signal call is active");
        }
        atomic_store(&destroyed, 1);
    }
    return pthread_cond_destroy(cond);
}

void llgo_lifetime_check(void) {
    if (atomic_load(&signaling)) {
        fail("address waiter returned while its wake call is active");
    }
    if (!atomic_load(&destroyed) && !atomic_load(&address_woken)) {
        fail("channel waiter lifetime was not exercised");
    }
    atomic_store(&waiting, 0);
}
