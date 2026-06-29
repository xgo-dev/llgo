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

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_PyObject_Print"(ptr %0, ptr %1, i32 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_PyObject_Print", align 8
// CHECK-NEXT:   %5 = load ptr, ptr %4, align 8
// CHECK-NEXT:   %6 = call i32 %5(ptr %0, ptr %1, i32 %2)
// CHECK-NEXT:   ret i32 %6
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_Py_Finalize"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_Py_Finalize", align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_Py_Initialize"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_Py_Initialize", align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_fputs"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_fputs", align 8
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = call i32 %4(ptr %0, ptr %1)
// CHECK-NEXT:   ret i32 %5
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_test_stdout"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_test_stdout", align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_False"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_False", align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_None"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_None", align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_True"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_True", align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = load ptr, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_stdout", align 8
// CHECK-NEXT:   %2 = load ptr, ptr %1, align 8
// CHECK-NEXT:   ret ptr %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgomacro.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cgomacro.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cgomacro.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_PyObject_Print, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_PyObject_Print", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cmacro_Py_False, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_False", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_Py_Finalize, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_Py_Finalize", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_Py_Initialize, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_Py_Initialize", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cmacro_Py_None, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_None", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cmacro_Py_True, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_Py_True", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_fputs, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_fputs", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cmacro_stdout, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cmacro_stdout", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_test_stdout, ptr @"{{.*}}/cl/_testgo/cgomacro._cgo_{{.*}}_Cfunc_test_stdout", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgomacro.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_test_stdout"()
// CHECK-NEXT:   %1 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$1"()
// CHECK-NEXT:   %2 = call [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_Py_Initialize"()
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %4 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %5 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 56)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 2
// CHECK-NEXT:   store ptr %3, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgomacro.main", %_llgo_2), ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 4
// CHECK-NEXT:   store ptr null, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 6
// CHECK-NEXT:   store ptr %5, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %6)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 4
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %17, align 8
// CHECK-NEXT:   %18 = call i32 @{{.*}}sigsetjmp(ptr %4, i32 0)
// CHECK-NEXT:   %19 = icmp eq i32 %18, 0
// CHECK-NEXT:   br i1 %19, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgomacro.main", %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %20 = load i64, ptr %14, align 8
// CHECK-NEXT:   %21 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr %21)
// CHECK-NEXT:   %23 = call [0 x i8] @"{{.*}}/cl/_testgo/cgomacro._Cfunc_Py_Finalize"()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(ptr %22)
// CHECK-NEXT:   %24 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, align 8
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %24, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %25)
// CHECK-NEXT:   %26 = load ptr, ptr %16, align 8
// CHECK-NEXT:   indirectbr ptr %26, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %3)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %27 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$2"()
// CHECK-NEXT:   %28 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$3"()
// CHECK-NEXT:   %29 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$4"()
// CHECK-NEXT:   %30 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$5"()
// CHECK-NEXT:   %31 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$6"()
// CHECK-NEXT:   %32 = call i32 @"{{.*}}/cl/_testgo/cgomacro.main$7"()
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgomacro.main", %_llgo_6), ptr %16, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgomacro.main", %_llgo_3), ptr %16, align 8
// CHECK-NEXT:   %33 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %33, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
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

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_fputs"(ptr @0, ptr %0)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_True"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_PyObject_Print"(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_fputs"(ptr @{{.*}}, ptr %0)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_False"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_PyObject_Print"(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$5"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_fputs"(ptr @{{.*}}, ptr %0)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$6"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_Py_None"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__object", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_PyObject_Print"(ptr %0, ptr %1, i32 0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgomacro.main$7"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/cl/_testgo/cgomacro._Cmacro_stdout"()
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/cgomacro._Ctype_struct__{{.*}}FILE", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = call i32 @"{{.*}}/cl/_testgo/cgomacro._Cfunc_fputs"(ptr @{{.*}}, ptr %0)
// CHECK-NEXT:   ret i32 %2
// CHECK-NEXT: }
