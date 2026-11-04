#include "textflag.h"

// The last slot deliberately ends in CALL, with no synthetic RET slot.
TEXT callbackTable(SB), NOSPLIT|NOFRAME, $0-0
	CALL sharedHelper(SB)
	CALL sharedHelper(SB)
	CALL sharedHelper(SB)
	CALL sharedHelper(SB)
