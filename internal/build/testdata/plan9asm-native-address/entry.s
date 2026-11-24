#include "textflag.h"

TEXT rawEntry(SB), NOSPLIT|NOFRAME, $0-0
	RET

GLOBL ·entry(SB), RODATA|NOPTR, $8
DATA ·entry(SB)/8, $rawEntry(SB)
