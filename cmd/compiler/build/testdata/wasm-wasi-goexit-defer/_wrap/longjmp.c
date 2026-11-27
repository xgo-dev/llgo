#include <setjmp.h>

__attribute__((noinline)) static void jump(jmp_buf target) {
  longjmp(target, 7);
}

int llgo_wasi_thread_longjmp(void) {
  jmp_buf target;
  int value = setjmp(target);
  if (!value)
    jump(target);
  return value;
}
