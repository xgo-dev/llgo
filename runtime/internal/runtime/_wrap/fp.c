#include <stddef.h>
#include <stdint.h>

/* llgo_framepointer lives in the runtime core (not the public runtime
 * package): Recover() records the recovering frame through it, and
 * programs that never import "runtime" still link the core. */
__attribute__((noinline)) void *llgo_framepointer(void)
{
#if defined(__GNUC__) || defined(__clang__)
    /* Read the saved caller FP before this helper's frame becomes invalid. */
    void **frame = (void **)__builtin_frame_address(0);
    return frame ? *frame : 0;
#else
    return 0;
#endif
}

/* llgo_scrub_stack zeroes 256 pointer-sized words below the caller's stack
 * pointer so pooled worker threads cannot conservatively pin finished
 * goroutine roots inside uninitialized park/GC_do_blocking context frames
 * (such as glibc's jmp_buf). */
#if defined(__GNUC__) || defined(__clang__)
__attribute__((noinline))
#elif defined(_MSC_VER)
__declspec(noinline)
#endif
void llgo_scrub_stack(void)
{
    volatile uintptr_t buf[256];
    for (size_t i = 0; i < 256; i++) {
        buf[i] = 0;
    }
}
