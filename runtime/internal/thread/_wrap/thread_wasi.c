#include <pthread.h>
#include <stdint.h>

__attribute__((import_module("wasix_32v1"), import_name("thread_exit"), noreturn))
extern void llgo_wasix_thread_exit(uint32_t status);

// wasi-libc supplies pthread creation and TLS; the host retires the thread.
_Noreturn void pthread_exit(void *result) {
  (void)result;
  llgo_wasix_thread_exit(0);
}
