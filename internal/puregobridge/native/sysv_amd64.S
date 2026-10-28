/* Explicit Linux/amd64 C ABI implementation of purego's packed call layout.
 * Do not substitute this entry for an arbitrary Go runtime trampoline. */

#if !defined(__linux__) || !defined(__x86_64__)
#error "the purego SysV adapter requires Linux/amd64"
#endif

	.text
	.globl llgo_purego_sysv_call
	.type llgo_purego_sysv_call,@function
llgo_purego_sysv_call:
	.cfi_startproc
	pushq %rbp
	.cfi_def_cfa_offset 16
	.cfi_offset %rbp, -16
	movq %rsp, %rbp
	.cfi_def_cfa_register %rbp
	subq $80, %rsp

	/* Nine outgoing spill words and one private saved frame pointer. Keep
	 * the pointer below BP, never in the caller's stack frame. R10/R11 are
	 * caller-saved scratch; none of RBX or R12-R15 may be used as scratch. */
	movq %rdi, 72(%rsp)
	movq %rdi, %r11
	movq 56(%r11), %r10
	movq %r10, 0(%rsp)
	movq 64(%r11), %r10
	movq %r10, 8(%rsp)
	movq 72(%r11), %r10
	movq %r10, 16(%rsp)
	movq 80(%r11), %r10
	movq %r10, 24(%rsp)
	movq 88(%r11), %r10
	movq %r10, 32(%rsp)
	movq 96(%r11), %r10
	movq %r10, 40(%rsp)
	movq 104(%r11), %r10
	movq %r10, 48(%rsp)
	movq 112(%r11), %r10
	movq %r10, 56(%rsp)
	movq 120(%r11), %r10
	movq %r10, 64(%rsp)

	movq 128(%r11), %xmm0
	movq 136(%r11), %xmm1
	movq 144(%r11), %xmm2
	movq 152(%r11), %xmm3
	movq 160(%r11), %xmm4
	movq 168(%r11), %xmm5
	movq 176(%r11), %xmm6
	movq 184(%r11), %xmm7
	movq 8(%r11), %rdi
	movq 16(%r11), %rsi
	movq 24(%r11), %rdx
	movq 32(%r11), %rcx
	movq 40(%r11), %r8
	movq 48(%r11), %r9
	movq 0(%r11), %r10
	/* SysV variadic AL is an upper bound on occupied vector registers.
	 * Supplying all eight is valid for fixed and variadic callees alike. */
	movl $8, %eax
	callq *%r10

	movq 72(%rsp), %rdi
	movq %rax, 8(%rdi)
	movq %rdx, 16(%rdi)
	movq %xmm0, 128(%rdi)
	movq %xmm1, 136(%rdi)
	xorl %eax, %eax
	movq %rbp, %rsp
	popq %rbp
	.cfi_def_cfa %rsp, 8
	.cfi_restore %rbp
	retq
	.cfi_endproc
	.size llgo_purego_sysv_call, .-llgo_purego_sysv_call

	.section .note.GNU-stack,"",@progbits
