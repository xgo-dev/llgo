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

/* llgo_scrub_stack zeroes 8 KiB (4 KiB on 32-bit) below the caller's stack
 * pointer and clears volatile integer and SIMD registers so pooled worker
 * threads cannot conservatively pin finished goroutine roots inside idle
 * park/GC_do_blocking context frames (such as glibc jmp_buf, Win64
 * RtlCaptureContext, or Windows ARM64 active wait frames). */
#if defined(__GNUC__) || defined(__clang__)
#define LLGO_NOINLINE __attribute__((noinline))
#elif defined(_MSC_VER)
#define LLGO_NOINLINE __declspec(noinline)
#else
#define LLGO_NOINLINE
#endif

static LLGO_NOINLINE void llgo_scrub_stack_chunk(int depth)
{
    volatile uintptr_t buf[256];
    for (size_t i = 0; i < 256; i++) {
        buf[i] = 0;
    }
    if (depth > 1) {
        llgo_scrub_stack_chunk(depth - 1);
        /* Volatile write after the recursive call prevents tail-call folding. */
        buf[0] = 0;
    }
}

LLGO_NOINLINE void llgo_scrub_stack(void)
{
    llgo_scrub_stack_chunk(4);
#if defined(__GNUC__) || defined(__clang__)
#if defined(__x86_64__) && defined(_WIN32)
    /* Win64 ABI: rax, rcx, rdx, r8-r11 and xmm0-xmm5 are volatile. */
    __asm__ volatile(
        "xorl %%eax, %%eax\n\t"
        "xorl %%ecx, %%ecx\n\t"
        "xorl %%edx, %%edx\n\t"
        "xorl %%r8d, %%r8d\n\t"
        "xorl %%r9d, %%r9d\n\t"
        "xorl %%r10d, %%r10d\n\t"
        "xorl %%r11d, %%r11d\n\t"
#if defined(__SSE2__)
        "pxor %%xmm0, %%xmm0\n\t"
        "pxor %%xmm1, %%xmm1\n\t"
        "pxor %%xmm2, %%xmm2\n\t"
        "pxor %%xmm3, %%xmm3\n\t"
        "pxor %%xmm4, %%xmm4\n\t"
        "pxor %%xmm5, %%xmm5\n\t"
#endif
        :
        :
        : "rax", "rcx", "rdx", "r8", "r9", "r10", "r11",
#if defined(__SSE2__)
          "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5",
#endif
          "cc", "memory");
#elif defined(__x86_64__)
    /* SysV AMD64 ABI: rax, rcx, rdx, rsi, rdi, r8-r11 and xmm0-xmm15 are volatile. */
    __asm__ volatile(
        "xorl %%eax, %%eax\n\t"
        "xorl %%ecx, %%ecx\n\t"
        "xorl %%edx, %%edx\n\t"
        "xorl %%esi, %%esi\n\t"
        "xorl %%edi, %%edi\n\t"
        "xorl %%r8d, %%r8d\n\t"
        "xorl %%r9d, %%r9d\n\t"
        "xorl %%r10d, %%r10d\n\t"
        "xorl %%r11d, %%r11d\n\t"
#if defined(__SSE2__)
        "pxor %%xmm0, %%xmm0\n\t"
        "pxor %%xmm1, %%xmm1\n\t"
        "pxor %%xmm2, %%xmm2\n\t"
        "pxor %%xmm3, %%xmm3\n\t"
        "pxor %%xmm4, %%xmm4\n\t"
        "pxor %%xmm5, %%xmm5\n\t"
        "pxor %%xmm6, %%xmm6\n\t"
        "pxor %%xmm7, %%xmm7\n\t"
        "pxor %%xmm8, %%xmm8\n\t"
        "pxor %%xmm9, %%xmm9\n\t"
        "pxor %%xmm10, %%xmm10\n\t"
        "pxor %%xmm11, %%xmm11\n\t"
        "pxor %%xmm12, %%xmm12\n\t"
        "pxor %%xmm13, %%xmm13\n\t"
        "pxor %%xmm14, %%xmm14\n\t"
        "pxor %%xmm15, %%xmm15\n\t"
#endif
        :
        :
        : "rax", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11",
#if defined(__SSE2__)
          "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7",
          "xmm8", "xmm9", "xmm10", "xmm11", "xmm12", "xmm13", "xmm14", "xmm15",
#endif
          "cc", "memory");
#elif defined(__aarch64__)
    /* AAPCS64 / Darwin / Windows ARM64: x0-x17, v0-v7, v16-v31 are volatile.
     * Never touch x18 (platform-reserved on Darwin and Windows). */
    __asm__ volatile(
        "mov x0, xzr\n\t"  "mov x1, xzr\n\t"  "mov x2, xzr\n\t"  "mov x3, xzr\n\t"
        "mov x4, xzr\n\t"  "mov x5, xzr\n\t"  "mov x6, xzr\n\t"  "mov x7, xzr\n\t"
        "mov x8, xzr\n\t"  "mov x9, xzr\n\t"  "mov x10, xzr\n\t" "mov x11, xzr\n\t"
        "mov x12, xzr\n\t" "mov x13, xzr\n\t" "mov x14, xzr\n\t" "mov x15, xzr\n\t"
        "mov x16, xzr\n\t" "mov x17, xzr\n\t"
#if defined(__ARM_NEON)
        "movi v0.2d, #0\n\t"  "movi v1.2d, #0\n\t"  "movi v2.2d, #0\n\t"  "movi v3.2d, #0\n\t"
        "movi v4.2d, #0\n\t"  "movi v5.2d, #0\n\t"  "movi v6.2d, #0\n\t"  "movi v7.2d, #0\n\t"
        "movi v16.2d, #0\n\t" "movi v17.2d, #0\n\t" "movi v18.2d, #0\n\t" "movi v19.2d, #0\n\t"
        "movi v20.2d, #0\n\t" "movi v21.2d, #0\n\t" "movi v22.2d, #0\n\t" "movi v23.2d, #0\n\t"
        "movi v24.2d, #0\n\t" "movi v25.2d, #0\n\t" "movi v26.2d, #0\n\t" "movi v27.2d, #0\n\t"
        "movi v28.2d, #0\n\t" "movi v29.2d, #0\n\t" "movi v30.2d, #0\n\t" "movi v31.2d, #0\n\t"
#endif
        :
        :
        : "x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7",
          "x8", "x9", "x10", "x11", "x12", "x13", "x14", "x15", "x16", "x17",
#if defined(__ARM_NEON)
          "v0", "v1", "v2", "v3", "v4", "v5", "v6", "v7",
          "v16", "v17", "v18", "v19", "v20", "v21", "v22", "v23",
          "v24", "v25", "v26", "v27", "v28", "v29", "v30", "v31",
#endif
          "memory");
#elif defined(__i386__)
    __asm__ volatile(
        "xorl %%eax, %%eax\n\t"
        "xorl %%ecx, %%ecx\n\t"
        "xorl %%edx, %%edx\n\t"
        :
        :
        : "eax", "ecx", "edx", "cc", "memory");
#endif
#endif
}
