#include <pthread.h>
#include <stdint.h>

typedef struct {
    void *code;
    uint64_t input;
    uint64_t depth;
    uint64_t result;
    uint64_t callbacks;
    uint64_t entry_mod;
    uint64_t call_mod;
} callback_frame;

extern void crosscall2(void *fn, void *frame, int32_t size, uintptr_t context);

static int32_t callback_entry(void *raw)
{
    callback_frame *frame = raw;
    crosscall2(frame->code, frame, 0, 0);
    return -19;
}

static void *foreign_start(void *raw)
{
    callback_entry(raw);
    return 0;
}

static int32_t foreign_entry(void *raw)
{
    pthread_t thread;
    if (pthread_create(&thread, 0, foreign_start, raw) != 0) {
        return -20;
    }
    if (pthread_join(thread, 0) != 0) {
        return -21;
    }
    return -19;
}

void *callback_entry_address(void)
{
    return (void *)&callback_entry;
}

void *foreign_entry_address(void)
{
    return (void *)&foreign_entry;
}

#if defined(__linux__) && defined(__x86_64__)
/* Both stubs enter with normal SysV alignment. The second reproduces a
 * word-aligned Go-assembly call to crosscall2, not an arbitrary C call. */
__attribute__((naked)) static int32_t stack_aligned_entry(void *raw)
{
    __asm__("movq %rsp, %rax\n\tandq $15, %rax\n\tmovq %rax, 40(%rdi)\n\t"
            "movq %rdi, %rsi\n\tpushq %rbx\n\t"
            "movq %rsp, %rax\n\tandq $15, %rax\n\tmovq %rax, 48(%rsi)\n\t"
            "movq (%rdi), %rdi\n\txorl %edx, %edx\n\txorl %ecx, %ecx\n\t"
            "callq crosscall2@PLT\n\tpopq %rbx\n\tmovl $-19, %eax\n\tretq");
}

__attribute__((naked)) static int32_t stack_eight_aligned_entry(void *raw)
{
    __asm__("movq %rsp, %rax\n\tandq $15, %rax\n\tmovq %rax, 40(%rdi)\n\t"
            "movq %rdi, %rsi\n\t"
            "movq %rsp, %rax\n\tandq $15, %rax\n\tmovq %rax, 48(%rsi)\n\t"
            "movq (%rdi), %rdi\n\txorl %edx, %edx\n\txorl %ecx, %ecx\n\t"
            "callq crosscall2@PLT\n\tmovl $-19, %eax\n\tretq");
}
#endif

void *stack_entry_address(int aligned)
{
#if defined(__linux__) && defined(__x86_64__)
    return aligned ? (void *)&stack_aligned_entry : (void *)&stack_eight_aligned_entry;
#else
    return 0; /* Only the Linux/amd64 owner selects these two modes. */
#endif
}
