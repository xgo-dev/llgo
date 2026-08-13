// LITTEST
package main

/*
#cgo pkg-config: python3-embed
#include <stdio.h>
#include <Python.h>

void test_stdout() {
	printf("stdout ptr: %p\n", stdout);
	fputs("outputs to stdout\n", stdout);
}
*/
import "C"
import (
	"unsafe"

	"github.com/goplus/lib/c"
)

// C object-like macros are emitted as getters and then passed through the
// ordinary generated C wrappers.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call {{.*}} @main._Cfunc_test_stdout()
// CHECK: call {{.*}} @main._Cfunc_Py_Initialize()
// CHECK: call {{.*}} @main._Cfunc_Py_Finalize()
// CHECK-LABEL: define i32 @"main.main$2"(){{.*}} {
// CHECK: call ptr @main._Cmacro_Py_True()
// CHECK: call ptr @main._Cmacro_stdout()
// CHECK: call i32 @main._Cfunc_PyObject_Print
// CHECK-LABEL: define i32 @"main.main$4"(){{.*}} {
// CHECK: call ptr @main._Cmacro_Py_False()
// CHECK-LABEL: define i32 @"main.main$6"(){{.*}} {
// CHECK: call ptr @main._Cmacro_Py_None()

func main() {
	C.test_stdout()
	C.fputs((*C.char)(unsafe.Pointer(c.Str("hello\n"))), C.stdout)
	C.Py_Initialize()
	defer C.Py_Finalize()
	C.PyObject_Print(C.Py_True, C.stdout, 0)
	C.fputs((*C.char)(unsafe.Pointer(c.Str("\n"))), C.stdout)
	C.PyObject_Print(C.Py_False, C.stdout, 0)
	C.fputs((*C.char)(unsafe.Pointer(c.Str("\n"))), C.stdout)
	C.PyObject_Print(C.Py_None, C.stdout, 0)
	C.fputs((*C.char)(unsafe.Pointer(c.Str("\n"))), C.stdout)
}
