#include "textflag.h"

TEXT ·Copy(SB), NOSPLIT, $56-32
	MOVQ dst+0(FP), AX
	MOVQ src+8(FP), BX
	MOVQ n+16(FP), CX
	MOVQ AX, 0(SP)
	MOVQ BX, 8(SP)
	MOVQ CX, 16(SP)
	MOVQ CX, 40(SP)
	CALL runtime·memmove(SB)
	MOVQ 40(SP), AX
	MOVQ AX, X0
	MOVQ X0, AX
	ADDQ $37, AX
	MOVQ AX, ret+24(FP)
	RET
