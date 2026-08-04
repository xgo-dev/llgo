#include <stddef.h>
#include <stdlib.h>

size_t llgo_test_sizeof_long(void) {
	return sizeof(long);
}

int llgo_test_scheduler_deadlock(void) {
	return getenv("LLGO_WASM_SCHEDULER_DEADLOCK") != NULL;
}
