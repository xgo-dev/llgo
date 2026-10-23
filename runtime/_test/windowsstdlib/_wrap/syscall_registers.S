/* Native Windows ARM64 execution only. This is a real C ABI caller of the
 * shared six-word thunk, not a gc register frame or a mock callee. */
#if defined(__aarch64__)
	.text
	.globl llgo_windows_syscall_register_canary
	.def llgo_windows_syscall_register_canary;
	.scl 2;
	.type 32;
	.endef
	.seh_proc llgo_windows_syscall_register_canary
llgo_windows_syscall_register_canary:
	stp x29, x30, [sp, #-48]!
	.seh_save_fplr_x 48
	stp x19, x20, [sp, #16]
	.seh_save_regp x19, 16
	stp x21, x22, [sp, #32]
	.seh_save_regp x21, 32
	mov x29, sp
	.seh_set_fp
	.seh_endprologue

	mov x21, sp
	mov x19, #0x1928
	mov x20, #0x2039
	bl llgo_windows_syscall
	mov w0, #0
	mov x9, #0x1928
	cmp x19, x9
	b.ne .Lrestore_register_canary
	mov x9, #0x2039
	cmp x20, x9
	b.ne .Lrestore_register_canary
	cmp x29, x21
	b.ne .Lrestore_register_canary
	mov w0, #1

.Lrestore_register_canary:
	/* Measure first, then re-establish our own real frame for cleanup even
	 * when the tested callee violated the x29 preservation contract. */
	mov x29, x21
	.seh_startepilogue
	mov sp, x29
	.seh_set_fp
	ldp x21, x22, [sp, #32]
	.seh_save_regp x21, 32
	ldp x19, x20, [sp, #16]
	.seh_save_regp x19, 16
	ldp x29, x30, [sp], #48
	.seh_save_fplr_x 48
	.seh_endepilogue
	ret
	.seh_endproc
#endif
