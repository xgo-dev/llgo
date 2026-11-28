#define _POSIX_C_SOURCE 200809L
#include <assert.h>
#include <pthread.h>
#include <stdatomic.h>
#include <stdint.h>
#include <time.h>

void llgo_wasi_gc_enter_begin(void);
void llgo_wasi_gc_enter_end(void);
void llgo_wasi_gc_leave_end(void);
int llgo_wasi_gc_stop(void);
void llgo_wasi_gc_resume(void);
void llgo_wasi_gc_mutex_lock(pthread_mutex_t *, uintptr_t, uintptr_t, uintptr_t);
void llgo_wasi_gc_cond_timedwait(pthread_cond_t *, pthread_mutex_t *, int64_t,
                                int, uintptr_t, uintptr_t, uintptr_t);

static pthread_mutex_t application = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t condition = PTHREAD_COND_INITIALIZER;
static atomic_int published;
static atomic_int returned;

void llgo_gcroot_publish_thread(uintptr_t chain, uintptr_t bottom,
                                uintptr_t top) {
  assert(chain == 11 && bottom == 22 && top == 33);
  atomic_store(&published, 1);
}
void llgo_gcroot_reset_thread_chain(void) {}
void *llgo_wasi_gc_mstart(void *arg) { return arg; }

static void pause_briefly(void) {
  struct timespec pause = {0, 1000000};
  nanosleep(&pause, 0);
}

static void *waiter(void *arg) {
  llgo_wasi_gc_enter_begin();
  llgo_wasi_gc_enter_end();
  if (arg) {
    assert(pthread_mutex_lock(&application) == 0);
    llgo_wasi_gc_cond_timedwait(&condition, &application, -1,
                               0, 11, 22, 33);
  } else {
    llgo_wasi_gc_mutex_lock(&application, 11, 22, 33);
  }
  atomic_store(&returned, 1);
  assert(pthread_mutex_unlock(&application) == 0);
  llgo_wasi_gc_leave_end();
  return 0;
}

int main(void) {
  llgo_wasi_gc_enter_begin();
  llgo_wasi_gc_enter_end();
  for (int mode = 0; mode != 2; ++mode) {
    for (int iteration = 0; iteration != 20; ++iteration) {
      atomic_store(&published, 0);
      atomic_store(&returned, 0);
      if (!mode)
        assert(pthread_mutex_lock(&application) == 0);
      pthread_t thread;
      assert(pthread_create(&thread, 0, waiter, (void *)(uintptr_t)mode) == 0);
      while (!atomic_load(&published))
        pause_briefly();
      if (mode)
        assert(pthread_mutex_lock(&application) == 0);

      // A stopped owner may hold the mutex a C waiter needs to reacquire.
      // Collection must succeed, and waking C must not resume Go early.
      assert(llgo_wasi_gc_stop());
      assert(pthread_cond_signal(&condition) == 0);
      assert(pthread_mutex_unlock(&application) == 0);
      pause_briefly();
      assert(!atomic_load(&returned));
      llgo_wasi_gc_resume();
      assert(pthread_join(thread, 0) == 0);
      assert(atomic_load(&returned));
    }
  }
  llgo_wasi_gc_leave_end();
  return 0;
}
