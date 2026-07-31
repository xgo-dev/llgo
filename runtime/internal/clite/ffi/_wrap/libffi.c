#include <ffi.h>

void *llgo_ffi_closure_alloc(void **code) {
    return ffi_closure_alloc(sizeof(ffi_closure), code);
}

/*
 * ffi_call does not expose a portable static-chain argument. Keep the real
 * target and environment in a per-thread context while it marshals arguments.
 * The trampoline is its final target: it saves the already-marshalled argument
 * registers, obtains that context, and installs LLVM's nest/swiftself register
 * for the real entry. A callee-saved swiftself register is restored afterward;
 * caller-saved transports can tail-jump. No generated function needs an adapter.
 *
 * Use libffi's optional Go-ABI entry when it is exported and its static-chain
 * register matches LLGo's nest transport. Common system libffi builds omit
 * that symbol, so the public-ffi_call trampoline remains the fallback.
 */
#if !defined(_WIN32) &&                                                     \
    (defined(__x86_64__) || defined(__i386__) || defined(__aarch64__) ||    \
     defined(__arm__) || defined(__riscv) || defined(__riscv__))

struct llgo_ffi_call_context {
    void (*target)(void);
    void *env;
    void *saved_callee;
    void *saved_self;
    void *saved_return;
};

static _Thread_local struct llgo_ffi_call_context llgo_ffi_call_current;

__attribute__((noinline, used)) static struct llgo_ffi_call_context *
llgo_ffi_current_call(void) {
    return &llgo_ffi_call_current;
}

/* libffi implements ffi_call_go by passing this final argument through its
 * private argument marshaller into the target's static-chain register. The
 * symbol is optional, so reference it weakly and retain the public API path. */
#if defined(__APPLE__)
#define LLGO_FFI_GO_WEAK __attribute__((weak_import))
#else
#define LLGO_FFI_GO_WEAK __attribute__((weak))
#endif

#if defined(__x86_64__) || defined(__i386__) || defined(__riscv) ||          \
    defined(__riscv__) ||                                                   \
    (defined(__aarch64__) && !defined(__APPLE__) && !defined(__ANDROID__))
#define LLGO_FFI_GO_ABI_MATCHES 1
extern void ffi_call_go(ffi_cif *, void (*)(void), void *, void **, void *)
    LLGO_FFI_GO_WEAK;
#endif

#if defined(__APPLE__)
#define LLGO_ASM_CSYM(name) "_" #name
#else
#define LLGO_ASM_CSYM(name) #name
#endif

#if defined(__x86_64__)

__attribute__((naked)) static void llgo_ffi_env_trampoline(void) {
    __asm__ volatile(
        "subq $200, %rsp\n\t"
        "movq %rdi, 0(%rsp)\n\t"
        "movq %rsi, 8(%rsp)\n\t"
        "movq %rdx, 16(%rsp)\n\t"
        "movq %rcx, 24(%rsp)\n\t"
        "movq %r8, 32(%rsp)\n\t"
        "movq %r9, 40(%rsp)\n\t"
        "movq %rax, 48(%rsp)\n\t"
        "movdqu %xmm0, 64(%rsp)\n\t"
        "movdqu %xmm1, 80(%rsp)\n\t"
        "movdqu %xmm2, 96(%rsp)\n\t"
        "movdqu %xmm3, 112(%rsp)\n\t"
        "movdqu %xmm4, 128(%rsp)\n\t"
        "movdqu %xmm5, 144(%rsp)\n\t"
        "movdqu %xmm6, 160(%rsp)\n\t"
        "movdqu %xmm7, 176(%rsp)\n\t"
        "callq " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "movq 0(%rax), %r11\n\t"
        "movq 8(%rax), %r10\n\t"
        "movdqu 64(%rsp), %xmm0\n\t"
        "movdqu 80(%rsp), %xmm1\n\t"
        "movdqu 96(%rsp), %xmm2\n\t"
        "movdqu 112(%rsp), %xmm3\n\t"
        "movdqu 128(%rsp), %xmm4\n\t"
        "movdqu 144(%rsp), %xmm5\n\t"
        "movdqu 160(%rsp), %xmm6\n\t"
        "movdqu 176(%rsp), %xmm7\n\t"
        "movq 0(%rsp), %rdi\n\t"
        "movq 8(%rsp), %rsi\n\t"
        "movq 16(%rsp), %rdx\n\t"
        "movq 24(%rsp), %rcx\n\t"
        "movq 32(%rsp), %r8\n\t"
        "movq 40(%rsp), %r9\n\t"
        "movq 48(%rsp), %rax\n\t"
        "addq $200, %rsp\n\t"
        "jmpq *%r11");
}

#elif defined(__i386__)

__attribute__((naked)) static void llgo_ffi_env_trampoline(void) {
    __asm__ volatile(
        "subl $12, %esp\n\t"
        "calll " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "movl 0(%eax), %edx\n\t"
        "movl 4(%eax), %ecx\n\t"
        "addl $12, %esp\n\t"
        "jmpl *%edx");
}

#elif defined(__aarch64__)

__attribute__((naked)) static void llgo_ffi_env_trampoline(void) {
    __asm__ volatile(
        "sub sp, sp, #224\n\t"
        "stp x0, x1, [sp, #0]\n\t"
        "stp x2, x3, [sp, #16]\n\t"
        "stp x4, x5, [sp, #32]\n\t"
        "stp x6, x7, [sp, #48]\n\t"
        "str x8, [sp, #64]\n\t"
        "str x30, [sp, #72]\n\t"
        "stp q0, q1, [sp, #80]\n\t"
        "stp q2, q3, [sp, #112]\n\t"
        "stp q4, q5, [sp, #144]\n\t"
        "stp q6, q7, [sp, #176]\n\t"
        "bl " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "mov x16, x0\n\t"
        "ldr x17, [x16, #0]\n\t"
#if defined(__APPLE__) || defined(__ANDROID__)
        /* X20 is callee-saved. Keep the TLS context in X19 across the real
         * target so it can restore both registers after the target returns.
         * The real target is entered with the original SP, preserving stack
         * argument offsets. */
        "str x19, [x16, #16]\n\t"
        "str x20, [x16, #24]\n\t"
        "ldr x15, [sp, #72]\n\t"
        "str x15, [x16, #32]\n\t"
        "mov x19, x16\n\t"
        "ldr x20, [x16, #8]\n\t"
#else
        "ldr x18, [x16, #8]\n\t"
#endif
        "ldp q0, q1, [sp, #80]\n\t"
        "ldp q2, q3, [sp, #112]\n\t"
        "ldp q4, q5, [sp, #144]\n\t"
        "ldp q6, q7, [sp, #176]\n\t"
        "ldr x8, [sp, #64]\n\t"
        "ldp x0, x1, [sp, #0]\n\t"
        "ldp x2, x3, [sp, #16]\n\t"
        "ldp x4, x5, [sp, #32]\n\t"
        "ldp x6, x7, [sp, #48]\n\t"
        "add sp, sp, #224\n\t"
#if defined(__APPLE__) || defined(__ANDROID__)
        "adr x30, 1f\n\t"
        "br x17\n\t"
        "1:\n\t"
        "ldr x16, [x19, #16]\n\t"
        "ldr x20, [x19, #24]\n\t"
        "ldr x30, [x19, #32]\n\t"
        "mov x19, x16\n\t"
        "ret"
#else
        "br x17"
#endif
    );
}

#elif defined(__arm__)

#if defined(__ARM_PCS_VFP)
#define LLGO_ARM_SAVED_LR_OFFSET 84
#else
#define LLGO_ARM_SAVED_LR_OFFSET 20
#endif
#define LLGO_ARM_ASM_STR1(value) #value
#define LLGO_ARM_ASM_STR(value) LLGO_ARM_ASM_STR1(value)

__attribute__((naked)) static void llgo_ffi_env_trampoline(void) {
    __asm__ volatile(
        "push {r0-r3, r10, lr}\n\t"
#if defined(__ARM_PCS_VFP)
        "vpush {d0-d7}\n\t"
#endif
        "bl " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "mov r12, r0\n\t"
        "str r4, [r12, #8]\n\t"
        "str r10, [r12, #12]\n\t"
        "ldr r3, [sp, #" LLGO_ARM_ASM_STR(LLGO_ARM_SAVED_LR_OFFSET) "]\n\t"
        "str r3, [r12, #16]\n\t"
        "mov r4, r12\n\t"
        "ldr r10, [r12, #4]\n\t"
        "ldr r12, [r12, #0]\n\t"
#if defined(__ARM_PCS_VFP)
        "vpop {d0-d7}\n\t"
#endif
        "ldmia sp, {r0-r3}\n\t"
        "add sp, sp, #24\n\t"
        "adr lr, 1f\n\t"
        "bx r12\n\t"
        "1:\n\t"
        "ldr r12, [r4, #8]\n\t"
        "ldr r10, [r4, #12]\n\t"
        "ldr lr, [r4, #16]\n\t"
        "mov r4, r12\n\t"
        "bx lr");
}

#elif defined(__riscv) || defined(__riscv__)

#define LLGO_ASM_STR1(value) #value
#define LLGO_ASM_STR(value) LLGO_ASM_STR1(value)

#if __riscv_xlen == 64
#define LLGO_RISCV_FP_OFFSET 80
#define LLGO_RISCV_BASE_FRAME 80
#else
#define LLGO_RISCV_FP_OFFSET 48
#define LLGO_RISCV_BASE_FRAME 48
#endif

#if defined(__riscv_float_abi_double)
#define LLGO_RISCV_FRAME (LLGO_RISCV_BASE_FRAME + 64)
#define LLGO_RISCV_SAVE_FP                                                 \
    "fsd fa0, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 0) "(sp)\n\t"         \
    "fsd fa1, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 8) "(sp)\n\t"         \
    "fsd fa2, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 16) "(sp)\n\t"        \
    "fsd fa3, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 24) "(sp)\n\t"        \
    "fsd fa4, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 32) "(sp)\n\t"        \
    "fsd fa5, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 40) "(sp)\n\t"        \
    "fsd fa6, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 48) "(sp)\n\t"        \
    "fsd fa7, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 56) "(sp)\n\t"
#define LLGO_RISCV_RESTORE_FP                                              \
    "fld fa0, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 0) "(sp)\n\t"         \
    "fld fa1, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 8) "(sp)\n\t"         \
    "fld fa2, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 16) "(sp)\n\t"        \
    "fld fa3, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 24) "(sp)\n\t"        \
    "fld fa4, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 32) "(sp)\n\t"        \
    "fld fa5, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 40) "(sp)\n\t"        \
    "fld fa6, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 48) "(sp)\n\t"        \
    "fld fa7, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 56) "(sp)\n\t"
#elif defined(__riscv_float_abi_single)
#define LLGO_RISCV_FRAME (LLGO_RISCV_BASE_FRAME + 32)
#define LLGO_RISCV_SAVE_FP                                                 \
    "fsw fa0, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 0) "(sp)\n\t"         \
    "fsw fa1, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 4) "(sp)\n\t"         \
    "fsw fa2, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 8) "(sp)\n\t"         \
    "fsw fa3, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 12) "(sp)\n\t"        \
    "fsw fa4, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 16) "(sp)\n\t"        \
    "fsw fa5, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 20) "(sp)\n\t"        \
    "fsw fa6, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 24) "(sp)\n\t"        \
    "fsw fa7, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 28) "(sp)\n\t"
#define LLGO_RISCV_RESTORE_FP                                              \
    "flw fa0, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 0) "(sp)\n\t"         \
    "flw fa1, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 4) "(sp)\n\t"         \
    "flw fa2, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 8) "(sp)\n\t"         \
    "flw fa3, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 12) "(sp)\n\t"        \
    "flw fa4, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 16) "(sp)\n\t"        \
    "flw fa5, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 20) "(sp)\n\t"        \
    "flw fa6, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 24) "(sp)\n\t"        \
    "flw fa7, " LLGO_ASM_STR(LLGO_RISCV_FP_OFFSET + 28) "(sp)\n\t"
#else
#define LLGO_RISCV_FRAME LLGO_RISCV_BASE_FRAME
#define LLGO_RISCV_SAVE_FP
#define LLGO_RISCV_RESTORE_FP
#endif

__attribute__((naked)) static void llgo_ffi_env_trampoline(void) {
#if __riscv_xlen == 64
    __asm__ volatile(
        "addi sp, sp, -" LLGO_ASM_STR(LLGO_RISCV_FRAME) "\n\t"
        "sd a0, 0(sp)\n\t"
        "sd a1, 8(sp)\n\t"
        "sd a2, 16(sp)\n\t"
        "sd a3, 24(sp)\n\t"
        "sd a4, 32(sp)\n\t"
        "sd a5, 40(sp)\n\t"
        "sd a6, 48(sp)\n\t"
        "sd a7, 56(sp)\n\t"
        "sd ra, 64(sp)\n\t"
        LLGO_RISCV_SAVE_FP
        "call " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "mv t1, a0\n\t"
        "ld t0, 0(t1)\n\t"
        "ld t2, 8(t1)\n\t"
        LLGO_RISCV_RESTORE_FP
        "ld ra, 64(sp)\n\t"
        "ld a0, 0(sp)\n\t"
        "ld a1, 8(sp)\n\t"
        "ld a2, 16(sp)\n\t"
        "ld a3, 24(sp)\n\t"
        "ld a4, 32(sp)\n\t"
        "ld a5, 40(sp)\n\t"
        "ld a6, 48(sp)\n\t"
        "ld a7, 56(sp)\n\t"
        "addi sp, sp, " LLGO_ASM_STR(LLGO_RISCV_FRAME) "\n\t"
        "jr t0");
#else
    __asm__ volatile(
        "addi sp, sp, -" LLGO_ASM_STR(LLGO_RISCV_FRAME) "\n\t"
        "sw a0, 0(sp)\n\t"
        "sw a1, 4(sp)\n\t"
        "sw a2, 8(sp)\n\t"
        "sw a3, 12(sp)\n\t"
        "sw a4, 16(sp)\n\t"
        "sw a5, 20(sp)\n\t"
        "sw a6, 24(sp)\n\t"
        "sw a7, 28(sp)\n\t"
        "sw ra, 32(sp)\n\t"
        LLGO_RISCV_SAVE_FP
        "call " LLGO_ASM_CSYM(llgo_ffi_current_call) "\n\t"
        "mv t1, a0\n\t"
        "lw t0, 0(t1)\n\t"
        "lw t2, 4(t1)\n\t"
        LLGO_RISCV_RESTORE_FP
        "lw ra, 32(sp)\n\t"
        "lw a0, 0(sp)\n\t"
        "lw a1, 4(sp)\n\t"
        "lw a2, 8(sp)\n\t"
        "lw a3, 12(sp)\n\t"
        "lw a4, 16(sp)\n\t"
        "lw a5, 20(sp)\n\t"
        "lw a6, 24(sp)\n\t"
        "lw a7, 28(sp)\n\t"
        "addi sp, sp, " LLGO_ASM_STR(LLGO_RISCV_FRAME) "\n\t"
        "jr t0");
#endif
}

#endif

void llgo_ffi_call_with_env(ffi_cif *cif, void (*fn)(void), void *rvalue,
                            void **avalue, void *env) {
#if defined(LLGO_FFI_GO_ABI_MATCHES)
    if (ffi_call_go != NULL) {
        ffi_call_go(cif, fn, rvalue, avalue, env);
        return;
    }
#endif
    struct llgo_ffi_call_context previous = llgo_ffi_call_current;
    llgo_ffi_call_current.target = fn;
    llgo_ffi_call_current.env = env;
    ffi_call(cif, llgo_ffi_env_trampoline, rvalue, avalue);
    llgo_ffi_call_current = previous;
}

#endif
