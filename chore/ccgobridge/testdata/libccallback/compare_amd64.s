#include "textflag.h"

TEXT ·comparePC(SB), NOSPLIT, $0-8
	MOVQ $·compare(SB), AX
	MOVQ AX, ret+0(FP)
	RET

TEXT ·compare(SB), 0, $40-28
	MOVQ tls+0(FP), AX
	MOVQ AX, 0(SP)
	MOVQ left+8(FP), AX
	MOVQ AX, 8(SP)
	MOVQ right+16(FP), AX
	MOVQ AX, 16(SP)
	CALL ·typedCompare(SB)
	MOVL 24(SP), AX
	MOVL AX, ret+24(FP)
	RET

TEXT ·typedCompare(SB), NOSPLIT, $0-28
	MOVQ left+8(FP), AX
	MOVQ right+16(FP), CX
	MOVQ (AX), AX
	MOVQ (CX), CX
	CMPQ AX, CX
	JL less
	JG greater
	MOVL $0, ret+24(FP)
	RET
less:
	MOVL $-1, ret+24(FP)
	RET
greater:
	MOVL $1, ret+24(FP)
	RET
