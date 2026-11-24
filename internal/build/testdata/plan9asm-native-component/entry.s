#include "textflag.h"

TEXT addressRoot(SB), NOSPLIT|NOFRAME, $0-0
	RET

GLOBL ·entry(SB), RODATA|NOPTR, $8
DATA ·entry(SB)/8, $addressRoot(SB)
