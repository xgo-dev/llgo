// LITTEST
package main

/*
#cgo pkg-config: python3-embed
#include <Python.h>
*/
import "C"

// The wrappers must call the C entry points installed in their cgo slots.
// CHECK-LABEL: define i32 @main._Cfunc_PyRun_SimpleString(ptr %0){{.*}} {
// CHECK: [[RUN_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_PyRun_SimpleString
// CHECK-NEXT: [[RUN_TARGET:%[0-9]+]] = load ptr, ptr [[RUN_SLOT]]
// CHECK-NEXT: [[RUN_RESULT:%[0-9]+]] = call i32 [[RUN_TARGET]](ptr %0)
// CHECK-NEXT: ret i32 [[RUN_RESULT]]
// CHECK-LABEL: define [0 x i8] @main._Cfunc_Py_Finalize(){{.*}} {
// CHECK: [[FINALIZE_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_Py_Finalize
// CHECK-NEXT: [[FINALIZE_TARGET:%[0-9]+]] = load ptr, ptr [[FINALIZE_SLOT]]
// CHECK-NEXT: [[FINALIZE_RESULT:%[0-9]+]] = call [0 x i8] [[FINALIZE_TARGET]]()
// CHECK-NEXT: ret [0 x i8] [[FINALIZE_RESULT]]
// CHECK-LABEL: define [0 x i8] @main._Cfunc_Py_Initialize(){{.*}} {
// CHECK: [[INIT_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_Py_Initialize
// CHECK-NEXT: [[INIT_TARGET:%[0-9]+]] = load ptr, ptr [[INIT_SLOT]]
// CHECK-NEXT: [[INIT_RESULT:%[0-9]+]] = call [0 x i8] [[INIT_TARGET]]()
// CHECK-NEXT: ret [0 x i8] [[INIT_RESULT]]

// main must register Finalize as a real defer before running Python, execute it
// on both normal and panic paths, restore the previous defer chain, and rethrow.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call [0 x i8] @main._Cfunc_Py_Initialize()
// CHECK-NEXT: [[OLD_DEFER:%[0-9]+]] = call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[DEFER:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 48)
// CHECK: [[DEFER_PREV:%[0-9]+]] = getelementptr inbounds %"{{.*}}Defer", ptr [[DEFER]], i32 0, i32 2
// CHECK-NEXT: store ptr [[OLD_DEFER]], ptr [[DEFER_PREV]]
// CHECK: call void @"{{.*}}SetThreadDefer"(ptr [[DEFER]])
// CHECK: [[JMP_RESULT:%[0-9]+]] = call i32 @{{(__)?sigsetjmp}}
// CHECK-NEXT: [[JMP_NORMAL:%[0-9]+]] = icmp eq i32 [[JMP_RESULT]], 0
// CHECK-NEXT: br i1 [[JMP_NORMAL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call [0 x i8] @main._Cfunc_Py_Finalize()
// CHECK: [[DEFER_VALUE:%[0-9]+]] = load %"{{.*}}Defer", ptr [[DEFER]]
// CHECK-NEXT: [[RESTORE:%[0-9]+]] = extractvalue %"{{.*}}Defer" [[DEFER_VALUE]], 2
// CHECK-NEXT: call void @"{{.*}}SetThreadDefer"(ptr [[RESTORE]])
// CHECK: call void @"{{.*}}Rethrow"(ptr [[OLD_DEFER]])
// CHECK: {{^_llgo_[0-9]+:}}
// CHECK-NEXT: [[CODE:%[0-9]+]] = call ptr @"{{.*}}CString"(%"{{.*}}String" { ptr {{.*}}, i64 23 })
// CHECK-NEXT: call i32 @main._Cfunc_PyRun_SimpleString(ptr [[CODE]])
// CHECK: {{^_llgo_[0-9]+:}}

func main() {
	C.Py_Initialize()
	defer C.Py_Finalize()
	C.PyRun_SimpleString(C.CString("print('Hello, Python!')"))
}
