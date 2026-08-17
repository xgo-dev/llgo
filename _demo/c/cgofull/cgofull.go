package main

/*
#cgo windows,!amd64 CFLAGS: -D_WIN32
#cgo !windows CFLAGS: -D_POSIX
#cgo windows,amd64 CFLAGS: -D_WIN64
#cgo linux,amd64 CFLAGS: -D_LINUX64
#cgo !windows,amd64 CFLAGS: -D_UNIX64
#cgo pkg-config: python3-embed
#include <stdio.h>
#include <Python.h>
#include "foo.h"
typedef struct {
	int a;
} s4;

typedef struct {
  int a;
  int b;
} s8;

typedef struct {
  int a;
  int b;
  int c;
} s12;

typedef struct {
  int a;
  int b;
  int c;
  int d;
} s16;

typedef struct {
  int a;
  int b;
  int c;
  int d;
  int e;
} s20;

static int test_structs(s4* s4, s8* s8, s12* s12, s16* s16, s20* s20) {
  printf("s4.a: %d\n", s4->a);
  printf("s8.a: %d, s8.b: %d\n", s8->a, s8->b);
  printf("s12.a: %d, s12.b: %d, s12.c: %d\n", s12->a, s12->b, s12->c);
  printf("s16.a: %d, s16.b: %d, s16.c: %d, s16.d: %d\n", s16->a, s16->b, s16->c, s16->d);
  printf("s20.a: %d, s20.b: %d, s20.c: %d, s20.d: %d, s20.e: %d\n", s20->a, s20->b, s20->c, s20->d, s20->e);

  return s4->a + s8->a + s8->b + s12->a + s12->b + s12->c + s16->a + s16->b + s16->c + s16->d + s20->a + s20->b + s20->c + s20->d + s20->e;
}

static void test_macros() {
#ifdef FOO
	printf("FOO is defined\n");
#endif
#ifdef BAR
	printf("BAR is defined\n");
#endif
#ifdef _WIN32
	printf("WIN32 is defined\n");
#endif
#ifdef _POSIX
	printf("POSIX is defined\n");
#endif
#ifdef _WIN64
	printf("WIN64 is defined\n");
#endif
#ifdef _LINUX64
	printf("LINUX64 is defined\n");
#endif
#ifdef _UNIX64
	printf("UNIX64 is defined\n");
#endif
}

#define MY_VERSION "1.0.0"
#define MY_CODE 0x12345678

static void test_void() {
	printf("test_void\n");
}

typedef int (*Cb)(int);

extern int go_callback(int);

extern int c_callback(int i);

static void test_callback(Cb cb) {
	printf("test_callback, cb: %p, go_callback: %p, c_callback: %p\n", cb, go_callback, c_callback);
	printf("test_callback, *cb: %p, *go_callback: %p, *c_callback: %p\n", *(void**)cb, *(void**)(go_callback), *(void**)(c_callback));
	printf("cb result: %d\n", cb(123));
	printf("done\n");
}

extern int go_callback_not_use_in_go(int);

static void run_callback() {
	test_callback(c_callback);
	test_callback(go_callback_not_use_in_go);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/xgo-dev/llgo/_demo/c/cgofull/pymod1"
	"github.com/xgo-dev/llgo/_demo/c/cgofull/pymod2"
)

//export go_callback_not_use_in_go
func go_callback_not_use_in_go(i C.int) C.int {
	return i + 1
}

//export go_callback
func go_callback(i C.int) C.int {
	return i + 1
}

func main() {
	runPy()
	f := &C.Foo{a: 1}
	Foo(f)
	Bar(f)
	C.test_macros()
	r := C.test_structs(&C.s4{a: 1}, &C.s8{a: 1, b: 2}, &C.s12{a: 1, b: 2, c: 3}, &C.s16{a: 1, b: 2, c: 3, d: 4}, &C.s20{a: 1, b: 2, c: 3, d: 4, e: 5})
	fmt.Println(r)
	if r != 35 {
		panic("test_structs failed")
	}
	fmt.Println(C.MY_VERSION)
	fmt.Println(int(C.MY_CODE))
	C.test_void()

	println("call run_callback")
	C.run_callback()

	// test _Cgo_ptr and _cgoCheckResult
	println("call with go_callback")
	C.test_callback((C.Cb)(C.go_callback))

	println("call with c_callback")
	C.test_callback((C.Cb)(C.c_callback))
}

func runPy() {
	Initialize()
	defer Finalize()
	Run("print('Hello, Python!')")
	C.PyObject_Print((*C.PyObject)(unsafe.Pointer(pymod1.Float(1.23))), C.stderr, 0)
	C.PyObject_Print((*C.PyObject)(unsafe.Pointer(pymod2.Long(123))), C.stdout, 0)
	// test _Cgo_use
	C.PyObject_Print((*C.PyObject)(unsafe.Pointer(C.PyComplex_FromDoubles(C.double(1.23), C.double(4.56)))), C.stdout, 0)
}
