#include <pthread.h>
#include <stdatomic.h>
#include <sched.h>

static pthread_t thread;
static atomic_int ready, stop;

static void *worker(void *unused)
{
    (void)unused;
    atomic_store(&ready, 1);
    while (!atomic_load(&stop)) sched_yield();
    return 0;
}

int foreign_thread_start(void)
{
    int error = pthread_create(&thread, 0, worker, 0);
    if (error) return error;
    while (!atomic_load(&ready)) sched_yield();
    return 0;
}

int foreign_thread_join(void)
{
    atomic_store(&stop, 1);
    return pthread_join(thread, 0);
}
