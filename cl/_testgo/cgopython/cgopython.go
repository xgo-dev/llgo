// LITTEST
package main

/*
#cgo pkg-config: python3-embed
#include <Python.h>
*/
import "C"

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgopython._Cfunc_PyRun_SimpleString"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %2 = load ptr, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_PyRun_SimpleString", align 8
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call i32 %3(ptr %0)
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgopython._Cfunc_Py_Finalize"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_Py_Finalize", align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define [0 x i8] @"{{.*}}/cl/_testgo/cgopython._Cfunc_Py_Initialize"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load ptr, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_Py_Initialize", align 8
// CHECK-NEXT:   %1 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %2 = call [0 x i8] %1()
// CHECK-NEXT:   ret [0 x i8] %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgopython.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/cgopython.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/cgopython.init$guard", align 1
// CHECK-NEXT:   call void @syscall.init()
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_PyRun_SimpleString, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_PyRun_SimpleString", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_Py_Finalize, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_Py_Finalize", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc_Py_Initialize, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc_Py_Initialize", align 8
// CHECK-NEXT:   store ptr @_cgo_{{.*}}_Cfunc__Cmalloc, ptr @"{{.*}}/cl/_testgo/cgopython._cgo_{{.*}}_Cfunc__Cmalloc", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgopython.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call [0 x i8] @"{{.*}}/cl/_testgo/cgopython._Cfunc_Py_Initialize"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %2 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 2
// CHECK-NEXT:   store ptr %1, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgopython.main", %_llgo_2), ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %3)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 4
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %11, align 8
// CHECK-NEXT:   %12 = call i32 @{{.*}}sigsetjmp(ptr %2, i32 0)
// CHECK-NEXT:   %13 = icmp eq i32 %12, 0
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgopython.main", %_llgo_3), ptr %9, align 8
// CHECK-NEXT:   %14 = load i64, ptr %8, align 8
// CHECK-NEXT:   %15 = call [0 x i8] @"{{.*}}/cl/_testgo/cgopython._Cfunc_Py_Finalize"()
// CHECK-NEXT:   %16 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, align 8
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %16, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %17)
// CHECK-NEXT:   %18 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %18, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %1)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.CString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 23 })
// CHECK-NEXT:   %20 = call i32 @"{{.*}}/cl/_testgo/cgopython._Cfunc_PyRun_SimpleString"(ptr %19)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgopython.main", %_llgo_6), ptr %10, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/cgopython.main", %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %21 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %21, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	C.Py_Initialize()
	defer C.Py_Finalize()
	C.PyRun_SimpleString(C.CString("print('Hello, Python!')"))
}
