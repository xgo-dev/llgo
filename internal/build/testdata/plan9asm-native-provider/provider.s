#include "textflag.h"

TEXT nativeRoot(SB), NOSPLIT|NOFRAME, $0-0
	CALL native_component_export(SB)
	RET

GLOBL ·entry(SB), RODATA|NOPTR, $8
DATA ·entry(SB)/8, $nativeRoot(SB)
