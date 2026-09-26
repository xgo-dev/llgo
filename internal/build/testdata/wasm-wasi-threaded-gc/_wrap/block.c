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
