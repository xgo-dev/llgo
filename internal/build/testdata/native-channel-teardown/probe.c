#include <pthread.h>
#include <sched.h>
#include <stdatomic.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>

extern int GoChannelWait(void);

static _Thread_local int armed;
static pthread_t owner;
static pthread_cond_t *target_cond;
static pthread_mutex_t *target_mutex;
static _Atomic int waiting;
static _Atomic int cond_destroyed;
static _Atomic int mutex_destroyed;

static void fail(const char *message) {
    fprintf(stderr, "%s\n", message);
    _Exit(90);
}

void llgo_teardown_arm(void) {
    armed = 1;
}

void llgo_teardown_await_wait(void) {
    while (!atomic_load(&waiting)) sched_yield();
}

int llgo_teardown_cond_wait(pthread_cond_t *cond, pthread_mutex_t *mutex) {
    if (armed) {
        armed = 0;
        owner = pthread_self();
        target_cond = cond;
        target_mutex = mutex;
        atomic_store(&waiting, 1);
    }
    return pthread_cond_wait(cond, mutex);
}

int llgo_teardown_cond_destroy(pthread_cond_t *cond) {
    int result = pthread_cond_destroy(cond);
    if (atomic_load(&waiting) && cond == target_cond) {
        if (result != 0) fail("cached channel condition destroy failed");
        atomic_fetch_add(&cond_destroyed, 1);
    }
    return result;
}

int llgo_teardown_mutex_destroy(pthread_mutex_t *mutex) {
    int result = pthread_mutex_destroy(mutex);
    if (atomic_load(&waiting) && mutex == target_mutex) {
        if (result != 0) fail("cached channel mutex destroy failed");
        atomic_fetch_add(&mutex_destroyed, 1);
    }
    return result;
}

void llgo_teardown_check_retained(void) {
    if (!atomic_load(&waiting)) fail("channel waiter was not exercised");
    if (atomic_load(&cond_destroyed) || atomic_load(&mutex_destroyed)) {
        fail("channel signal was not retained until goroutine teardown");
    }
}

static void check_destroyed(void) {
    if (atomic_load(&cond_destroyed) != 1 || atomic_load(&mutex_destroyed) != 1) {
        fail("cached channel signal was not destroyed exactly once");
    }
}

void llgo_teardown_finish_main(void) {
    if (pthread_join(owner, NULL) != 0) fail("cannot join initial thread");
    check_destroyed();
    puts("ok");
    fflush(stdout);
    _Exit(0);
}

static void *foreign_wait(void *unused) {
    return (void *)(intptr_t)GoChannelWait();
}

int llgo_teardown_foreign_thread(void) {
    pthread_t thread;
    if (pthread_create(&thread, NULL, foreign_wait, NULL) != 0) {
        fail("cannot create foreign thread");
    }
    void *result;
    if (pthread_join(thread, &result) != 0) fail("cannot join foreign thread");
    check_destroyed();
    return (int)(intptr_t)result;
}
