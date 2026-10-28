	.text
	.globl llgo_test_preserved
	.type llgo_test_preserved,@function
llgo_test_preserved:
	pushq %rbp
	pushq %rbx
	pushq %r12
	pushq %r13
	pushq %r14
	pushq %r15
	subq $136, %rsp
	movq %rdi, 128(%rsp)
	movq %rsp, %rdi
	movl $16, %ecx
	movabsq $0x1122334455667788, %rax
	rep stosq
	movq $101, %rbp
	movq $102, %rbx
	movq $103, %r12
	movq $104, %r13
	movq $105, %r14
	movq $106, %r15
	movq 128(%rsp), %rdi
	callq llgo_purego_sysv_call
	cmpl $0, %eax
	jne .Lfailed
	cmpq $101, %rbp
	jne .Lfailed
	cmpq $102, %rbx
	jne .Lfailed
	cmpq $103, %r12
	jne .Lfailed
	cmpq $104, %r13
	jne .Lfailed
	cmpq $105, %r14
	jne .Lfailed
	cmpq $106, %r15
	jne .Lfailed
	movq %rsp, %rdi
	movl $16, %ecx
	movabsq $0x1122334455667788, %rax
	repe scasq
	jne .Lfailed
	xorl %eax, %eax
	jmp .Ldone
.Lfailed:
	movl $1, %eax
.Ldone:
	addq $136, %rsp
	popq %r15
	popq %r14
	popq %r13
	popq %r12
	popq %rbx
	popq %rbp
	retq
	.size llgo_test_preserved, .-llgo_test_preserved

	.globl llgo_test_alignment
	.type llgo_test_alignment,@function
llgo_test_alignment:
	leaq 8(%rsp), %rax
	andq $15, %rax
	retq
	.size llgo_test_alignment, .-llgo_test_alignment

	.globl llgo_test_vector_count
	.type llgo_test_vector_count,@function
llgo_test_vector_count:
	movzbl %al, %eax
	retq
	.size llgo_test_vector_count, .-llgo_test_vector_count

	.section .note.GNU-stack,"",@progbits
