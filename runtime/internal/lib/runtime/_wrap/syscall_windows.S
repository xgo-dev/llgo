/*
 * Copyright 2025 The Go Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style license available at
 * https://go.dev/LICENSE.
 *
 * Adapted from internal/runtime/syscall/windows/asm_windows_386.s,
 * asm_windows_amd64.s, and asm_windows_arm64.s for LLGo's C ABI, including
 * explicit COFF unwind data where the platform uses table-based unwinding.
 */

/*
 * Variable-width Windows API call bridge. The layout of llgo_windows_syscall
 * is kept in sync with windowsSyscall in syscall_windows_llgo.go.
 */

#if defined(__x86_64__)

	.text
	.globl llgo_windows_syscall
	.def llgo_windows_syscall;
	.scl 2;
	.type 32;
	.endef
	.seh_proc llgo_windows_syscall
llgo_windows_syscall:
	pushq %rbp
	.seh_pushreg %rbp
	subq $48, %rsp
	.seh_stackalloc 48
	movq %rsp, %rbp
	.seh_setframe %rbp, 0
	.seh_endprologue
	movq %rcx, 32(%rbp)

	/* Keep the fixed frame recoverable while outgoing arguments grow below
	 * it. Include Win64 shadow space, and reject only size arithmetic wrap. */
	movq 8(%rcx), %rax
	movq %rax, %r10
	shrq $61, %r10
	jnz .Lbad_x64
	cmpq $4, %rax
	jae .Lsize_x64
	movl $4, %eax
.Lsize_x64:
	shlq $3, %rax
	addq $15, %rax
	jc .Lbad_x64
	andq $-16, %rax
#if defined(__MINGW32__)
	callq ___chkstk_ms
#else
	callq __chkstk
#endif
	subq %rax, %rsp
	movq 32(%rbp), %rcx

	/* SetLastError(0) through the TEB. */
	movq %gs:0x30, %r10
	movl $0, 0x68(%r10)

	/* The first four slots are Win64 shadow space. Remaining arguments are
	 * already in their final outgoing-stack positions. */
	movq 8(%rcx), %r10
	movq 16(%rcx), %r11
	xorl %eax, %eax
.Lcopy_x64:
	cmpq %r10, %rax
	jae .Lload_x64
	movq (%r11,%rax,8), %rdx
	movq %rdx, (%rsp,%rax,8)
	incq %rax
	jmp .Lcopy_x64

.Lload_x64:
	cmpq $4, %r10
	jb .Lless4_x64
	movq 24(%rsp), %r9
	movq %r9, %xmm3
.Lless4_x64:
	cmpq $3, %r10
	jb .Lless3_x64
	movq 16(%rsp), %r8
	movq %r8, %xmm2
.Lless3_x64:
	cmpq $2, %r10
	jb .Lless2_x64
	movq 8(%rsp), %rdx
	movq %rdx, %xmm1
.Lless2_x64:
	cmpq $1, %r10
	jb .Lcall_x64
	movq 0(%rsp), %rcx
	movq %rcx, %xmm0

.Lcall_x64:
	movq 32(%rbp), %r11
	movq 0(%r11), %rax
	callq *%rax

	movq 32(%rbp), %rcx
	movq %rax, 24(%rcx)
	movq %xmm0, 32(%rcx)
	movq %gs:0x30, %r10
	movl 0x68(%r10), %eax
	movq %rax, 40(%rcx)
	leaq 48(%rbp), %rsp
	popq %rbp
	retq
.Lbad_x64:
	ud2
	.seh_endproc

#elif defined(__aarch64__)

	.text
	.globl llgo_windows_syscall
	.def llgo_windows_syscall;
	.scl 2;
	.type 32;
	.endef
llgo_windows_syscall:
	.seh_proc llgo_windows_syscall
	/* Chained alloca frame: unwind always recovers SP through x29. */
	stp x29, x30, [sp, #-32]!
	.seh_save_fplr_x 32
	str x19, [sp, #16]
	.seh_save_reg x19, 16
	mov x29, sp
	.seh_set_fp
	.seh_endprologue
	mov x19, x0

	ldr x9, [x19, #8]
	cmp x9, #8
	b.ls .Lallocated_arm64
	sub x15, x9, #8
	lsr x16, x15, #61
	cbnz x16, .Lbad_arm64
	lsl x15, x15, #3
	adds x15, x15, #15
	b.cs .Lbad_arm64
	lsr x15, x15, #4
	/* Windows __chkstk takes x15 in 16-byte units and preserves it. */
	bl __chkstk
	sub sp, sp, x15, lsl #4
.Lallocated_arm64:

	/* SetLastError(0) through the Windows ARM64 TEB register. */
	str wzr, [x18, #0x68]

	ldr x9, [x19, #8]
	ldr x12, [x19, #16]
	cmp x9, #8
	b.ls .Lload_arm64
	sub x10, x9, #8
	mov x11, #0
.Lcopy_arm64:
	add x13, x12, #64
	ldr x13, [x13, x11, lsl #3]
	str x13, [sp, x11, lsl #3]
	add x11, x11, #1
	cmp x11, x10
	b.lo .Lcopy_arm64

.Lload_arm64:
	cmp x9, #8
	b.lo .Lless8_arm64
	ldr x7, [x12, #56]
.Lless8_arm64:
	cmp x9, #7
	b.lo .Lless7_arm64
	ldr x6, [x12, #48]
.Lless7_arm64:
	cmp x9, #6
	b.lo .Lless6_arm64
	ldr x5, [x12, #40]
.Lless6_arm64:
	cmp x9, #5
	b.lo .Lless5_arm64
	ldr x4, [x12, #32]
.Lless5_arm64:
	cmp x9, #4
	b.lo .Lless4_arm64
	ldr x3, [x12, #24]
.Lless4_arm64:
	cmp x9, #3
	b.lo .Lless3_arm64
	ldr x2, [x12, #16]
.Lless3_arm64:
	cmp x9, #2
	b.lo .Lless2_arm64
	ldr x1, [x12, #8]
.Lless2_arm64:
	cmp x9, #1
	b.lo .Lcall_arm64
	ldr x0, [x12]

.Lcall_arm64:
	ldr x16, [x19]
	blr x16
	str x0, [x19, #24]
	str xzr, [x19, #32]
	ldr w0, [x18, #0x68]
	str x0, [x19, #40]

	.seh_startepilogue
	mov sp, x29
	.seh_set_fp
	ldr x19, [sp, #16]
	.seh_save_reg x19, 16
	ldp x29, x30, [sp], #32
	.seh_save_fplr_x 32
	.seh_endepilogue
	ret
.Lbad_arm64:
	brk #0
	.seh_endfunclet
	.seh_endproc

#elif defined(__i386__)

	/* This object defines no exception handlers. Mark it SafeSEH-compatible so
	 * the 32-bit MSVC linker can retain its default /safeseh policy. */
	.def @feat.00;
	.scl 3;
	.type 0;
	.endef
	.globl @feat.00
	.set @feat.00, 1

	.text
	.globl _llgo_windows_syscall
	.def _llgo_windows_syscall;
	.scl 2;
	.type 32;
	.endef
_llgo_windows_syscall:
	pushl %ebp
	movl %esp, %ebp
	pushl %ebx
	pushl %esi
	pushl %edi

	/* The LLGo C ABI passes the windowsSyscall pointer on the stack. */
	movl 8(%ebp), %ebx

	/* SetLastError(0) through the x86 TEB. */
	movl $0, %fs:0x34

	/* Both stdcall and cdecl receive their arguments on the stack. The
	 * explicit stack restoration after the call accommodates either cleanup
	 * convention, matching the Go runtime's x86 bridge. */
	andl $-16, %esp
	movl 4(%ebx), %ecx
	testl %ecx, %ecx
	jz .Lcall_x86
	testl $0xc0000000, %ecx
	jnz .Lbad_x86
	movl %ecx, %eax
	shll $2, %eax
	addl $15, %eax
	jc .Lbad_x86
	andl $-16, %eax
#if defined(__MINGW32__)
	/* GNU's probe leaves ESP unchanged; MSVC's allocates the probed frame. */
	calll ___chkstk_ms
	subl %eax, %esp
#else
	calll __chkstk
#endif
	movl %esp, %edi
	movl 8(%ebx), %esi
	cld
	rep movsl

.Lcall_x86:
	call *0(%ebx)
	leal -12(%ebp), %esp

	movl %eax, 12(%ebx)
	movl %edx, 16(%ebx)
	movl %fs:0x34, %eax
	movl %eax, 20(%ebx)

	popl %edi
	popl %esi
	popl %ebx
	popl %ebp
	retl
.Lbad_x86:
	ud2

#else
#error "LLGo Windows SyscallN currently supports 386, amd64, and arm64"
#endif
