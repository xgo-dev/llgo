#include "textflag.h"
#include "funcdata.h"
#include "go_asm.h"

// Copy all Uint128 fields through the outgoing ABI0 frame, then read the
// 32-bit result. The frame and field constants come from the selected package.
TEXT ·wrap(SB), $56-52
	GO_ARGS
	NO_LOCAL_POINTERS
	MOVQ t+0(FP), AX
	MOVQ AX, 0(SP)
	MOVQ a_Lo+8(FP), AX
	MOVQ AX, 8(SP)
	MOVQ a_Hi+(8+Uint128_Hi)(FP), AX
	MOVQ AX, 16(SP)
	MOVQ b_Lo+24(FP), AX
	MOVQ AX, 24(SP)
	MOVQ b_Hi+(24+Uint128_Hi)(FP), AX
	MOVQ AX, 32(SP)
	MOVQ result+40(FP), AX
	MOVQ AX, 40(SP)
	CALL ·impl(SB)
	MOVL 48(SP), AX
	MOVL AX, ret+48(FP)
	RET

TEXT ·roundTrip(SB), NOSPLIT, $0-16
	MOVQ value+0(FP), AX
	VMOVQ AX, X0
	VMOVQ X0, AX
	MOVQ AX, ret+8(FP)
	RET
