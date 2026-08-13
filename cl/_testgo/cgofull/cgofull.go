// LITTEST
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

	"github.com/goplus/llgo/cl/_testgo/cgofull/pymod1"
	"github.com/goplus/llgo/cl/_testgo/cgofull/pymod2"
)

// This is the broad CGo integration case. Check each boundary kind once:
// C2func errno result, exported Go callbacks, plain C wrappers, and macros.
// CHECK-LABEL: define { i32, %"{{.*}}iface" } @main._C2func_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK: [[C2_SLOT:%.*]] = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_C2func_test_structs
// CHECK: [[C2_FN:%.*]] = load ptr, ptr [[C2_SLOT]]
// CHECK: [[C2_RESULT:%.*]] = call i32 [[C2_FN]](ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK: [[C2_ERRNO:%.*]] = call i32 @cliteErrno()
// CHECK: [[C2_HAS_ERR:%.*]] = icmp ne i32 [[C2_ERRNO]], 0
// CHECK: [[C2_ERRNO64:%.*]] = sext i32 [[C2_ERRNO]] to i64
// CHECK: store i64 [[C2_ERRNO64]], ptr %{{.*}}
// CHECK: call ptr @"github.com/goplus/llgo/runtime/internal/runtime.NewItab"(ptr {{.*}}, ptr @_llgo_syscall.Errno)
// CHECK: br i1 [[C2_HAS_ERR]], label %{{.*}}, label %{{.*}}
// CHECK: insertvalue { i32, %"{{.*}}iface" } undef, i32 [[C2_RESULT]], 0
// CHECK: insertvalue { i32, %"{{.*}}iface" } undef, i32 [[C2_RESULT]], 0

// CHECK-LABEL: define ptr @main._Cgo_ptr(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT: ret ptr %0

// CHECK-LABEL: define i32 @go_callback(i32 %0){{.*}} {
// CHECK: [[GO_CTX:%.*]] = alloca %"{{.*}}LocalContext"
// CHECK: [[GO_TOKEN:%.*]] = call i64 @"github.com/goplus/llgo/runtime/internal/runtime.EnterLocalContext"(ptr [[GO_CTX]])
// CHECK: [[GO_RESULT:%.*]] = add i32 %0, 1
// CHECK: call void @"github.com/goplus/llgo/runtime/internal/runtime.LeaveLocalContext"(ptr [[GO_CTX]], i64 [[GO_TOKEN]])
// CHECK: ret i32 [[GO_RESULT]]

// CHECK-LABEL: define i32 @go_callback_not_use_in_go(i32 %0){{.*}} {
// CHECK: [[UNUSED_CTX:%.*]] = alloca %"{{.*}}LocalContext"
// CHECK: [[UNUSED_TOKEN:%.*]] = call i64 @"github.com/goplus/llgo/runtime/internal/runtime.EnterLocalContext"(ptr [[UNUSED_CTX]])
// CHECK: [[UNUSED_RESULT:%.*]] = add i32 %0, 1
// CHECK: call void @"github.com/goplus/llgo/runtime/internal/runtime.LeaveLocalContext"(ptr [[UNUSED_CTX]], i64 [[UNUSED_TOKEN]])
// CHECK: ret i32 [[UNUSED_RESULT]]

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.runPy()
// CHECK: call void @main.triggerC2func()
// CHECK: [[FOO:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK: [[FOO_A:%.*]] = getelementptr inbounds %{{.*}}, ptr [[FOO]], i32 0, i32 0
// CHECK: store i32 1, ptr [[FOO_A]]
// CHECK: call void @main.Foo(ptr [[FOO]])
// CHECK: call void @main.Bar(ptr [[FOO]])
// CHECK: call [0 x i8] @main._Cfunc_test_macros()
// CHECK: [[MAIN_S4:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK: store i32 1, ptr %{{.*}}
// CHECK: [[MAIN_S8:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: store i32 2, ptr %{{.*}}
// CHECK: [[MAIN_S12:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK: store i32 3, ptr %{{.*}}
// CHECK: [[MAIN_S16:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: store i32 4, ptr %{{.*}}
// CHECK: [[MAIN_S20:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK: store i32 5, ptr %{{.*}}
// CHECK: [[MAIN_RESULT:%.*]] = call i32 @main._Cfunc_test_structs(ptr [[MAIN_S4]], ptr [[MAIN_S8]], ptr [[MAIN_S12]], ptr [[MAIN_S16]], ptr [[MAIN_S20]])
// CHECK: store i32 [[MAIN_RESULT]], ptr %{{.*}}
// CHECK: [[MAIN_BAD:%.*]] = icmp ne i32 [[MAIN_RESULT]], 35
// CHECK: br i1 [[MAIN_BAD]], label %{{.*}}, label %{{.*}}
// CHECK: store %"{{.*}}String" { ptr {{.*}}, i64 5 }, ptr %{{.*}}
// CHECK: store i64 305419896, ptr %{{.*}}
// CHECK: call [0 x i8] @main._Cfunc_test_void()
// CHECK: call [0 x i8] @main._Cfunc_run_callback()
// CHECK: [[GO_CB:%.*]] = load ptr, ptr @main._Cfpvar_fp_go_callback
// CHECK: [[GO_CB_PTR:%.*]] = call ptr @main._Cgo_ptr(ptr [[GO_CB]])
// CHECK: call [0 x i8] @main._Cfunc_test_callback(ptr [[GO_CB_PTR]])
// CHECK: [[C_CB:%.*]] = load ptr, ptr @main._Cfpvar_fp_c_callback
// CHECK: [[C_CB_PTR:%.*]] = call ptr @main._Cgo_ptr(ptr [[C_CB]])
// CHECK: call [0 x i8] @main._Cfunc_test_callback(ptr [[C_CB_PTR]])

// CHECK-LABEL: define i32 @"main.runPy$1"(){{.*}} {
// CHECK: [[FLOAT:%.*]] = call ptr @"github.com/goplus/llgo/cl/_testgo/cgofull/pymod1.Float"(double 1.230000e+00)
// CHECK: [[STDERR:%.*]] = call ptr @main._Cmacro_stderr()
// CHECK: [[FLOAT_RESULT:%.*]] = call i32 @main._Cfunc_PyObject_Print(ptr [[FLOAT]], ptr [[STDERR]], i32 0)
// CHECK: ret i32 [[FLOAT_RESULT]]

// CHECK-LABEL: define i32 @"main.runPy$2"(){{.*}} {
// CHECK: [[LONG:%.*]] = call ptr @"github.com/goplus/llgo/cl/_testgo/cgofull/pymod2.Long"(i64 123)
// CHECK: [[LONG_STDOUT:%.*]] = call ptr @main._Cmacro_stdout()
// CHECK: [[LONG_RESULT:%.*]] = call i32 @main._Cfunc_PyObject_Print(ptr [[LONG]], ptr [[LONG_STDOUT]], i32 0)
// CHECK: ret i32 [[LONG_RESULT]]

// CHECK-LABEL: define i32 @"main.runPy$3"(){{.*}} {
// CHECK: [[COMPLEX:%.*]] = call ptr @main._Cfunc_PyComplex_FromDoubles(double 1.230000e+00, double 4.560000e+00)
// CHECK: [[COMPLEX_STDOUT:%.*]] = call ptr @main._Cmacro_stdout()
// CHECK: [[COMPLEX_RESULT:%.*]] = call i32 @main._Cfunc_PyObject_Print(ptr [[COMPLEX]], ptr [[COMPLEX_STDOUT]], i32 0)
// CHECK: ret i32 [[COMPLEX_RESULT]]

// CHECK-LABEL: define void @main.triggerC2func(){{.*}} {
// CHECK: [[TRIGGER_S4:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK: store i32 1, ptr %{{.*}}
// CHECK: [[TRIGGER_S8:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: store i32 2, ptr %{{.*}}
// CHECK: [[TRIGGER_S12:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK: store i32 3, ptr %{{.*}}
// CHECK: [[TRIGGER_S16:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: store i32 4, ptr %{{.*}}
// CHECK: [[TRIGGER_S20:%.*]] = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK: store i32 5, ptr %{{.*}}
// CHECK: [[TRIGGER_PAIR:%.*]] = call { i32, %"{{.*}}iface" } @main._C2func_test_structs(ptr [[TRIGGER_S4]], ptr [[TRIGGER_S8]], ptr [[TRIGGER_S12]], ptr [[TRIGGER_S16]], ptr [[TRIGGER_S20]])
// CHECK: [[TRIGGER_RESULT:%.*]] = extractvalue { i32, %"{{.*}}iface" } [[TRIGGER_PAIR]], 0
// CHECK: [[TRIGGER_ERR:%.*]] = extractvalue { i32, %"{{.*}}iface" } [[TRIGGER_PAIR]], 1
// CHECK: call ptr @"github.com/goplus/llgo/runtime/internal/runtime.IfaceType"(%"{{.*}}iface" [[TRIGGER_ERR]])
// CHECK: [[ERR_IS_NIL:%.*]] = call i1 @"github.com/goplus/llgo/runtime/internal/runtime.EfaceEqual"
// CHECK: [[HAS_ERROR:%.*]] = xor i1 [[ERR_IS_NIL]], true
// CHECK: br i1 [[HAS_ERROR]], label %{{.*}}, label %{{.*}}
// CHECK: call void @"github.com/goplus/llgo/runtime/internal/runtime.Panic"
// CHECK: [[TRIGGER_BAD:%.*]] = icmp ne i32 [[TRIGGER_RESULT]], 35
// CHECK: br i1 [[TRIGGER_BAD]], label %{{.*}}, label %{{.*}}

//export go_callback_not_use_in_go
func go_callback_not_use_in_go(i C.int) C.int {
	return i + 1
}

//export go_callback
func go_callback(i C.int) C.int {
	return i + 1
}

func triggerC2func() {
	s4 := C.s4{a: 1}
	s8 := C.s8{a: 1, b: 2}
	s12 := C.s12{a: 1, b: 2, c: 3}
	s16 := C.s16{a: 1, b: 2, c: 3, d: 4}
	s20 := C.s20{a: 1, b: 2, c: 3, d: 4, e: 5}
	r, err := C.test_structs(&s4, &s8, &s12, &s16, &s20)
	if err != nil {
		panic(err)
	}
	if r != 35 {
		panic("C2func test_structs failed")
	}
	fmt.Println("c2func ok")
}

func main() {
	runPy()
	triggerC2func()
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
