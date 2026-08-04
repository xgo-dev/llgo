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

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [18 x i8] c"failed to run code", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [19 x i8] c"test_structs failed", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"1.0.0", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [17 x i8] c"call run_callback", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [21 x i8] c"call with go_callback", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [20 x i8] c"call with c_callback", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [23 x i8] c"print('Hello, Python!')", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [26 x i8] c"C2func test_structs failed", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [9 x i8] c"c2func ok", align 1{{$}}

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

// CHECK-LABEL: define void @main.Bar(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call [0 x i8] @main._Cfunc_print_foo(ptr %0)
// CHECK-NEXT:   %2 = call [0 x i8] @main._Cfunc_foo(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.Finalize(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call [0 x i8] @main._Cfunc_Py_Finalize()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.Foo(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call [0 x i8] @main._Cfunc_print_foo(ptr %0)
// CHECK-NEXT:   %2 = call [0 x i8] @main._Cfunc_foo(ptr %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.Initialize(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call [0 x i8] @main._Cfunc_Py_Initialize()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.iface" @main.Run(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.CString"(%"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT:   %2 = call i32 @main._Cfunc_PyRun_SimpleString(ptr %1)
// CHECK-NEXT:   %3 = icmp ne i32 %2, 0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call [0 x i8] @main._Cfunc_PyErr_Print()
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.iface" @fmt.Errorf(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 18 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" %5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer
// CHECK-NEXT: }

// CHECK-LABEL: define { i32, %"{{.*}}/runtime/internal/runtime.iface" } @main._C2func_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %6 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_C2func_test_structs, align 8
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 %7(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT:   %9 = call i32 @cliteErrno()
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" zeroinitializer, ptr %10, align 8
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %10, align 8
// CHECK-NEXT:   %12 = icmp ne i32 %9, 0
// CHECK-NEXT:   %13 = sext i32 %9 to i64
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %13, ptr %14, align 8
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Fh8eUJ-Gw4e6TYuajcFIOSCuqSPKAt5nS4ow7xeGXEU", ptr @_llgo_syscall.Errno)
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %15, 0
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %16, ptr %14, 1
// CHECK-NEXT:   br i1 %12, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %18 = insertvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %8, 0
// CHECK-NEXT:   %19 = insertvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } %18, %"{{.*}}/runtime/internal/runtime.iface" %17, 1
// CHECK-NEXT:   ret { i32, %"{{.*}}/runtime/internal/runtime.iface" } %19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = insertvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } undef, i32 %8, 0
// CHECK-NEXT:   %21 = insertvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } %20, %"{{.*}}/runtime/internal/runtime.iface" %11, 1
// CHECK-NEXT:   ret { i32, %"{{.*}}/runtime/internal/runtime.iface" } %21
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @main._Cfunc_PyComplex_FromDoubles(double %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyComplex_FromDoubles, align 8
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call ptr %4(double %0, double %1)
// CHECK-NEXT:   ret ptr %5
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_PyErr_Print(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyErr_Print, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @main._Cfunc_PyObject_Print(ptr %0, ptr %1, i32 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyObject_Print, align 8
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call i32 %5(ptr %0, ptr %1, i32 %2)
// CHECK-NEXT:   ret i32 %6
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @main._Cfunc_PyRun_SimpleString(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyRun_SimpleString, align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 %3(ptr %0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_Py_Finalize(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_Py_Finalize, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_Py_Initialize(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_Py_Initialize, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_foo(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_foo, align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_print_foo(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_print_foo, align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_run_callback(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_run_callback, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_test_callback(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_callback, align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call [0 x i8] %3(ptr %0)
// CHECK-NEXT:   ret [0 x i8] %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_test_macros(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_macros, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @main._Cfunc_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %6 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_structs, align 8
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 %7(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT:   ret i32 %8
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @main._Cfunc_test_void(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_void, align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @main._Cgo_ptr(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr %0
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @main._Cmacro_stderr(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cmacro_stderr, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @main._Cmacro_stdout(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @main._cgo_{{[0-9a-f]+}}_Cmacro_stdout, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @go_callback(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/runtime/internal/runtime.LocalContext", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/runtime/internal/runtime.EnterLocalContext"(ptr %1)
// CHECK-NEXT:   %3 = add i32 %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.LeaveLocalContext"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i32 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @go_callback_not_use_in_go(i32 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/runtime/internal/runtime.LocalContext", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/runtime/internal/runtime.EnterLocalContext"(ptr %1)
// CHECK-NEXT:   %3 = add i32 %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.LeaveLocalContext"(ptr %1, i64 %2)
// CHECK-NEXT:   ret i32 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/cgofull/pymod1.init"()
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/cgofull/pymod2.init"()
// CHECK-NEXT:   store ptr @main.__cgo_c_callback, ptr @main._Cfpvar_fp_c_callback, align 8
// CHECK-NEXT:   store ptr @main.__cgo_go_callback, ptr @main._Cfpvar_fp_go_callback, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_C2func_test_structs, ptr @main._cgo_{{[0-9a-f]+}}_C2func_test_structs, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_PyComplex_FromDoubles, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyComplex_FromDoubles, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_PyErr_Print, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyErr_Print, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_PyObject_Print, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyObject_Print, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_PyRun_SimpleString, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_PyRun_SimpleString, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_Py_Finalize, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_Py_Finalize, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_Py_Initialize, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_Py_Initialize, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_foo, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_foo, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_print_foo, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_print_foo, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_run_callback, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_run_callback, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cmacro_stderr, ptr @main._cgo_{{[0-9a-f]+}}_Cmacro_stderr, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cmacro_stdout, ptr @main._cgo_{{[0-9a-f]+}}_Cmacro_stdout, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_test_callback, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_callback, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_test_macros, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_macros, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_test_structs, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_structs, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc_test_void, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc_test_void, align 8
// CHECK-NEXT:   store ptr @_cgo_{{[0-9a-f]+}}_Cfunc__Cmalloc, ptr @main._cgo_{{[0-9a-f]+}}_Cfunc__Cmalloc, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @main.runPy()
// CHECK-NEXT:   call void @main.triggerC2func()
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %1 = getelementptr inbounds %main._Ctype_struct___0, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %1, align 4
// CHECK-NEXT:   call void @main.Foo(ptr %0)
// CHECK-NEXT:   call void @main.Bar(ptr %0)
// CHECK-NEXT:   %2 = call [0 x i8] @main._Cfunc_test_macros()
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4 = getelementptr inbounds %main._Ctype_struct___12, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %4, align 4
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %6 = getelementptr inbounds %main._Ctype_struct___13, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = getelementptr inbounds %main._Ctype_struct___13, ptr %5, i32 0, i32 1
// CHECK-NEXT:   store i32 1, ptr %6, align 4
// CHECK-NEXT:   store i32 2, ptr %7, align 4
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   %9 = getelementptr inbounds %main._Ctype_struct___9, ptr %8, i32 0, i32 0
// CHECK-NEXT:   %10 = getelementptr inbounds %main._Ctype_struct___9, ptr %8, i32 0, i32 1
// CHECK-NEXT:   %11 = getelementptr inbounds %main._Ctype_struct___9, ptr %8, i32 0, i32 2
// CHECK-NEXT:   store i32 1, ptr %9, align 4
// CHECK-NEXT:   store i32 2, ptr %10, align 4
// CHECK-NEXT:   store i32 3, ptr %11, align 4
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds %main._Ctype_struct___10, ptr %12, i32 0, i32 0
// CHECK-NEXT:   %14 = getelementptr inbounds %main._Ctype_struct___10, ptr %12, i32 0, i32 1
// CHECK-NEXT:   %15 = getelementptr inbounds %main._Ctype_struct___10, ptr %12, i32 0, i32 2
// CHECK-NEXT:   %16 = getelementptr inbounds %main._Ctype_struct___10, ptr %12, i32 0, i32 3
// CHECK-NEXT:   store i32 1, ptr %13, align 4
// CHECK-NEXT:   store i32 2, ptr %14, align 4
// CHECK-NEXT:   store i32 3, ptr %15, align 4
// CHECK-NEXT:   store i32 4, ptr %16, align 4
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK-NEXT:   %18 = getelementptr inbounds %main._Ctype_struct___11, ptr %17, i32 0, i32 0
// CHECK-NEXT:   %19 = getelementptr inbounds %main._Ctype_struct___11, ptr %17, i32 0, i32 1
// CHECK-NEXT:   %20 = getelementptr inbounds %main._Ctype_struct___11, ptr %17, i32 0, i32 2
// CHECK-NEXT:   %21 = getelementptr inbounds %main._Ctype_struct___11, ptr %17, i32 0, i32 3
// CHECK-NEXT:   %22 = getelementptr inbounds %main._Ctype_struct___11, ptr %17, i32 0, i32 4
// CHECK-NEXT:   store i32 1, ptr %18, align 4
// CHECK-NEXT:   store i32 2, ptr %19, align 4
// CHECK-NEXT:   store i32 3, ptr %20, align 4
// CHECK-NEXT:   store i32 4, ptr %21, align 4
// CHECK-NEXT:   store i32 5, ptr %22, align 4
// CHECK-NEXT:   %23 = call i32 @main._Cfunc_test_structs(ptr %3, ptr %5, ptr %8, ptr %12, ptr %17)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %25 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %24, i64 0
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 %23, ptr %26, align 4
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main._Ctype_int, ptr undef }, ptr %26, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %27, ptr %25, align 8
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %24, 0
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %28, i64 1, 1
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %29, i64 1, 2
// CHECK-NEXT:   %31 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %30)
// CHECK-NEXT:   %32 = icmp ne i32 %23, 35
// CHECK-NEXT:   br i1 %32, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 19 }, ptr %33, align 8
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %33, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %34)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %36 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %35, i64 0
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %37, align 8
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %37, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %38, ptr %36, align 8
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %35, 0
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %39, i64 1, 1
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %40, i64 1, 2
// CHECK-NEXT:   %42 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %41)
// CHECK-NEXT:   %43 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %44 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %43, i64 0
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 305419896, ptr %45, align 8
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %45, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %46, ptr %44, align 8
// CHECK-NEXT:   %47 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %43, 0
// CHECK-NEXT:   %48 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %47, i64 1, 1
// CHECK-NEXT:   %49 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %48, i64 1, 2
// CHECK-NEXT:   %50 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %49)
// CHECK-NEXT:   %51 = call [0 x i8] @main._Cfunc_test_void()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %52 = call [0 x i8] @main._Cfunc_run_callback()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 21 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %53 = load ptr, ptr @main._Cfpvar_fp_go_callback, align 8
// CHECK-NEXT:   %54 = call ptr @main._Cgo_ptr(ptr %53)
// CHECK-NEXT:   %55 = call [0 x i8] @main._Cfunc_test_callback(ptr %54)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 20 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %56 = load ptr, ptr @main._Cfpvar_fp_c_callback, align 8
// CHECK-NEXT:   %57 = call ptr @main._Cgo_ptr(ptr %56)
// CHECK-NEXT:   %58 = call [0 x i8] @main._Cfunc_test_callback(ptr %57)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.runPy(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @main.Initialize()
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.runPy, %_llgo_2), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %10, align 8
// CHECK-NEXT:   %11 = call i32 @{{(__)?}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %12 = icmp eq i32 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.runPy, %_llgo_3), ptr %8, align 8
// CHECK-NEXT:   %13 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @main.Finalize()
// CHECK-NEXT:   %14 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %14, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %15)
// CHECK-NEXT:   %16 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %16, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call %"{{.*}}/runtime/internal/runtime.iface" @main.Run(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 23 })
// CHECK-NEXT:   %18 = call i32 @"main.runPy$1"()
// CHECK-NEXT:   %19 = call i32 @"main.runPy$2"()
// CHECK-NEXT:   %20 = call i32 @"main.runPy$3"()
// CHECK-NEXT:   store ptr blockaddress(@main.runPy, %_llgo_6), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.runPy, %_llgo_3), ptr %9, align 8
// CHECK-NEXT:   %21 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %21, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"main.runPy$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgofull/pymod1.Float"(double 1.230000e+00)
// CHECK-NEXT:   %1 = call ptr @main._Cmacro_stderr()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct_{{(__sFILE|_IO_FILE)}}", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @main._Cfunc_PyObject_Print(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"main.runPy$2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgofull/pymod2.Long"(i64 123)
// CHECK-NEXT:   %1 = call ptr @main._Cmacro_stdout()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct_{{(__sFILE|_IO_FILE)}}", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @main._Cfunc_PyObject_Print(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"main.runPy$3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @main._Cfunc_PyComplex_FromDoubles(double 1.230000e+00, double 4.560000e+00)
// CHECK-NEXT:   %1 = call ptr @main._Cmacro_stdout()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main._Ctype_struct_{{(__sFILE|_IO_FILE)}}", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @main._Cfunc_PyObject_Print(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.triggerC2func(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %1 = getelementptr inbounds %main._Ctype_struct___12, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %1, align 4
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds %main._Ctype_struct___13, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %main._Ctype_struct___13, ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i32 1, ptr %3, align 4
// CHECK-NEXT:   store i32 2, ptr %4, align 4
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   %6 = getelementptr inbounds %main._Ctype_struct___9, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = getelementptr inbounds %main._Ctype_struct___9, ptr %5, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %main._Ctype_struct___9, ptr %5, i32 0, i32 2
// CHECK-NEXT:   store i32 1, ptr %6, align 4
// CHECK-NEXT:   store i32 2, ptr %7, align 4
// CHECK-NEXT:   store i32 3, ptr %8, align 4
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %10 = getelementptr inbounds %main._Ctype_struct___10, ptr %9, i32 0, i32 0
// CHECK-NEXT:   %11 = getelementptr inbounds %main._Ctype_struct___10, ptr %9, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %main._Ctype_struct___10, ptr %9, i32 0, i32 2
// CHECK-NEXT:   %13 = getelementptr inbounds %main._Ctype_struct___10, ptr %9, i32 0, i32 3
// CHECK-NEXT:   store i32 1, ptr %10, align 4
// CHECK-NEXT:   store i32 2, ptr %11, align 4
// CHECK-NEXT:   store i32 3, ptr %12, align 4
// CHECK-NEXT:   store i32 4, ptr %13, align 4
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK-NEXT:   %15 = getelementptr inbounds %main._Ctype_struct___11, ptr %14, i32 0, i32 0
// CHECK-NEXT:   %16 = getelementptr inbounds %main._Ctype_struct___11, ptr %14, i32 0, i32 1
// CHECK-NEXT:   %17 = getelementptr inbounds %main._Ctype_struct___11, ptr %14, i32 0, i32 2
// CHECK-NEXT:   %18 = getelementptr inbounds %main._Ctype_struct___11, ptr %14, i32 0, i32 3
// CHECK-NEXT:   %19 = getelementptr inbounds %main._Ctype_struct___11, ptr %14, i32 0, i32 4
// CHECK-NEXT:   store i32 1, ptr %15, align 4
// CHECK-NEXT:   store i32 2, ptr %16, align 4
// CHECK-NEXT:   store i32 3, ptr %17, align 4
// CHECK-NEXT:   store i32 4, ptr %18, align 4
// CHECK-NEXT:   store i32 5, ptr %19, align 4
// CHECK-NEXT:   %20 = call { i32, %"{{.*}}/runtime/internal/runtime.iface" } @main._C2func_test_structs(ptr %0, ptr %2, ptr %5, ptr %9, ptr %14)
// CHECK-NEXT:   %21 = extractvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } %20, 0
// CHECK-NEXT:   %22 = extractvalue { i32, %"{{.*}}/runtime/internal/runtime.iface" } %20, 1
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %22)
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %22, 1
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %23, 0
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %25, ptr %24, 1
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %27, 0
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %28, ptr null, 1
// CHECK-NEXT:   %30 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %26, %"{{.*}}/runtime/internal/runtime.eface" %29)
// CHECK-NEXT:   %31 = xor i1 %30, true
// CHECK-NEXT:   br i1 %31, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %22)
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %22, 1
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %32, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %34, ptr %33, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %36 = icmp ne i32 %21, 35
// CHECK-NEXT:   br i1 %36, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %37 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 26 }, ptr %37, align 8
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %37, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %38)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %39 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %40 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %39, i64 0
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 9 }, ptr %41, align 8
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %41, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %42, ptr %40, align 8
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %39, 0
// CHECK-NEXT:   %44 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %43, i64 1, 1
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %44, i64 1, 2
// CHECK-NEXT:   %46 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @fmt.Println(%"{{.*}}/runtime/internal/runtime.Slice" %45)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
