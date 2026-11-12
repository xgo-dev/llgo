#include "textflag.h"

// Outgoing ABI0 arguments and saved locals use the Go frame, as in s2Decode.
TEXT ·copyABI0(SB), NOSPLIT, $56-32
	MOVD dst+0(FP), R7
	MOVD src+8(FP), R6
	MOVD n+16(FP), R4
	MOVD R7, 8(RSP)
	MOVD R6, 16(RSP)
	MOVD R4, 24(RSP)
	MOVD R4, 32(RSP)
	// These are scratch values, never memmove's C register arguments.
	MOVD $60, R1
	MOVD $99, R2
	CALL runtime·memmove(SB)
	MOVD 32(RSP), R3
	ADD $37, R3
	MOVD R3, ret+24(FP)
	RET
