#include "textflag.h"

TEXT ·Enter(SB), NOSPLIT|NOFRAME, $0-24
 MOVD code+0(FP), R27
 MOVD context+8(FP), R0
 MOVD stack+16(FP), R26
 JMP (R27)

TEXT ·Resume(SB), NOSPLIT|NOFRAME, $0-24
 MOVD code+0(FP), R20
 MOVD context+8(FP), R0
 MOVD stack+16(FP), R19
 MOVD R29, 16(R0)
 MOVD RSP, R27
 MOVD R27, 24(R0)
 MOVD R30, 32(R0)
 MOVD R19, RSP
 JMP (R20)
