#include <pthread.h>
#include <stdint.h>

static pthread_mutex_t block_mutex = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t block_cond = PTHREAD_COND_INITIALIZER;
static int blocked;
static int released;

void llgo_gc_block_in_c(void *ptr) {
  volatile uintptr_t retained = (uintptr_t)ptr;
  pthread_mutex_lock(&block_mutex);
  blocked = 1;
  pthread_cond_broadcast(&block_cond);
  while (!released)
    pthread_cond_wait(&block_cond, &block_mutex);
  pthread_mutex_unlock(&block_mutex);
  (void)retained;
}

int32_t llgo_gc_c_blocked(void) {
  pthread_mutex_lock(&block_mutex);
  int result = blocked;
  pthread_mutex_unlock(&block_mutex);
  return result;
}

void llgo_gc_release_c(void) {
  pthread_mutex_lock(&block_mutex);
  released = 1;
  pthread_cond_broadcast(&block_cond);
  pthread_mutex_unlock(&block_mutex);
}

// Exercise the explicit C-to-Go registration contract. Returning from the
// callback currently retains registration until the foreign pthread exits.
extern void llgo_gc_foreign_callback(void);
static int foreign_idle;
static int foreign_released;

static void *foreign_thread(void *unused) {
  (void)unused;
  llgo_gc_foreign_callback();
  pthread_mutex_lock(&block_mutex);
  foreign_idle = 1;
  while (!foreign_released)
    pthread_cond_wait(&block_cond, &block_mutex);
  pthread_mutex_unlock(&block_mutex);
  return NULL;
}

int32_t llgo_gc_start_foreign_thread(void) {
  pthread_t thread;
  int status = pthread_create(&thread, NULL, foreign_thread, NULL);
  if (status != 0)
    return status;
  return pthread_detach(thread);
}

int32_t llgo_gc_foreign_thread_idle(void) {
  pthread_mutex_lock(&block_mutex);
  int idle = foreign_idle;
  pthread_mutex_unlock(&block_mutex);
  return idle;
}

void llgo_gc_release_foreign_thread(void) {
  pthread_mutex_lock(&block_mutex);
  foreign_released = 1;
  pthread_cond_broadcast(&block_cond);
  pthread_mutex_unlock(&block_mutex);
}
