#include <pthread.h>
#include <stdint.h>

static pthread_mutex_t llgo_gcroot_registry = PTHREAD_MUTEX_INITIALIZER;

struct llgo_thread_context {
  // LLGo's wasm32 Go pointer/uintptr slots occupy eight bytes each.
  uint64_t next;
  uint64_t chain;
  uint64_t stack_bottom;
  uint64_t stack_top;
};

_Static_assert(sizeof(struct llgo_thread_context) == 32,
               "WASI GC root context must match LLGo's wasm32 layout");

static _Thread_local struct llgo_thread_context llgo_current_thread_context;

extern _Thread_local uint64_t llgo_gcroot_chain
    __asm__("github.com/xgo-dev/llgo/runtime/internal/gcroot.currentRootChain");

void llgo_gcroot_reset_thread_chain(void) {
  llgo_gcroot_chain = 0;
}

uintptr_t llgo_gcroot_thread_context(void) {
  return (uintptr_t)&llgo_current_thread_context;
}

void llgo_gcroot_publish_thread(uintptr_t chain, uintptr_t bottom,
                                uintptr_t top) {
  if (bottom == 0 || bottom >= top)
    __builtin_trap();
  llgo_current_thread_context.chain = chain;
  llgo_current_thread_context.stack_bottom = bottom;
  llgo_current_thread_context.stack_top = top;
}

void llgo_gcroot_lock(void) {
  if (pthread_mutex_lock(&llgo_gcroot_registry) != 0)
    __builtin_trap();
}

void llgo_gcroot_unlock(void) {
  if (pthread_mutex_unlock(&llgo_gcroot_registry) != 0)
    __builtin_trap();
}
