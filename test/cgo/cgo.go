//go:build llgo && !wasm
// +build llgo,!wasm

// Go's cgo frontend does not define a wasm target ABI. The LLGo Emscripten
// and WASI C boundaries are exercised by target-specific LLGoFiles fixtures.

package cgo

/*
#include <errno.h>
#include <stdlib.h>

typedef struct { int a; } s4;
typedef struct { int a; int b; } s8;
typedef struct { int a; int b; int c; } s12;
typedef struct { int a; int b; int c; int d; } s16;
typedef struct { int a; int b; int c; int d; int e; } s20;

static int c_add(int a, int b) {
	return a + b;
}

static int sum_structs(s4* a, s8* b, s12* c, s16* d, s20* e) {
	return a->a + b->a + b->b + c->a + c->b + c->c +
		d->a + d->b + d->c + d->d +
		e->a + e->b + e->c + e->d + e->e;
}

static int c_errno_wrap(int x) {
	if (x < 0) {
		errno = ERANGE;
		return -1;
	}
	errno = 0;
	return x + 1;
}

// volatile: a bare NULL store is UB and clang would otherwise propagate it
// into unreachable, turning the recursion into an infinite loop.
static int *volatile llgo_test_fault_null;
volatile int llgo_test_fault_marks;

void llgo_test_fault_leaf(void) { *llgo_test_fault_null = 42; }

void llgo_test_fault_mid(int depth) {
	if (depth > 0) {
		llgo_test_fault_mid(depth - 1);
		llgo_test_fault_marks++;
		return;
	}
	llgo_test_fault_leaf();
	llgo_test_fault_marks++;
}

void llgo_test_fault(int depth) {
	llgo_test_fault_mid(depth);
	llgo_test_fault_marks++;
}
*/
import "C"

import "unsafe"

func Add(a, b int) int {
	return int(C.c_add(C.int(a), C.int(b)))
}

func SumStructs() (int, error) {
	a := C.s4{a: 1}
	b := C.s8{a: 1, b: 2}
	c := C.s12{a: 1, b: 2, c: 3}
	d := C.s16{a: 1, b: 2, c: 3, d: 4}
	e := C.s20{a: 1, b: 2, c: 3, d: 4, e: 5}
	sum, err := C.sum_structs(&a, &b, &c, &d, &e)
	return int(sum), err
}

func ErrnoWrap(v int) (int, error) {
	r, err := C.c_errno_wrap(C.int(v))
	return int(r), err
}

func MallocFree(size uintptr) bool {
	p := C.malloc(C.size_t(size))
	if p == nil {
		return false
	}
	C.free(unsafe.Pointer(p))
	return true
}

func CauseFault(depth int) {
	C.llgo_test_fault(C.int(depth))
}
