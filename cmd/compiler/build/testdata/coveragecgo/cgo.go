package coveragecgo

/*
#include <errno.h>
typedef struct { int value; } record;
static int add(record *p, int n) { return p->value + n; }
static int fail(void) { errno = EINVAL; return -1; }
*/
import "C"

func Add(n int) int {
	p := C.record{value: 7}
	// Cgo emits pointer-checking wrappers which must not count as user blocks.
	return int(C.add(&p, C.int(n)))
}

func Failure() (int, error) {
	n, err := C.fail()
	return int(n), err
}

func Unused() int {
	return 9
}
