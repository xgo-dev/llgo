// LITTEST
// Scope: arch (arm64/amd64 closure/defer calling convention)
package main

/*
#include <stdlib.h>
#include "callback.h"

typedef int (*callback_t)(int);
extern int go_callback(int);
extern int go_callback_c_only(int);

static int invoke_callback(callback_t callback, int value) {
	return callback(value);
}
static int invoke_export_only(int value) {
	return go_callback_c_only(value);
}
*/
import "C"

// CHECK-LABEL: define [0 x i8] @main._Cfunc_free(ptr %0){{.*}} {
// CHECK: [[FREE_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_free
// CHECK-NEXT: [[FREE_FN:%[0-9]+]] = load ptr, ptr [[FREE_SLOT]]
// CHECK-NEXT: [[FREE_RESULT:%[0-9]+]] = call [0 x i8] [[FREE_FN]](ptr %0)
// CHECK-NEXT: ret [0 x i8] [[FREE_RESULT]]

// Exported Go callbacks establish a local runtime context before entering Go.
// go_callback_c_only is deliberately never referenced by Go: the local C
// invoke_export_only wrapper is its sole caller, preserving the export-only
// reachability regression.
// CHECK inspects package IR; ENTRY inspects the final-link module through
// TestCgoDeferExportEntries. Both stages retain the callback's signature.
// ENTRY-LABEL: define i32 @go_callback(i32 %0){{.*}} {
// ENTRY: [[REGISTERED:%[0-9]+]] = call i1 @"{{.*}}EnterForeignThread"()
// ENTRY-NEXT: [[RESULT:%[0-9]+]] = call i32 @main.go_callback(i32 %0)
// ENTRY-NEXT: call void @"{{.*}}ExitForeignThread"(i1 [[REGISTERED]])
// ENTRY-NEXT: ret i32 [[RESULT]]
// ENTRY-LABEL: define i32 @go_callback_c_only(i32 %0){{.*}} {
// ENTRY: [[ONLY_REGISTERED:%[0-9]+]] = call i1 @"{{.*}}EnterForeignThread"()
// ENTRY-NEXT: [[ONLY_RESULT:%[0-9]+]] = call i32 @main.go_callback_c_only(i32 %0)
// ENTRY-NEXT: call void @"{{.*}}ExitForeignThread"(i1 [[ONLY_REGISTERED]])
// ENTRY-NEXT: ret i32 [[ONLY_RESULT]]

// CHECK-LABEL: define i32 @main.go_callback(i32 %0){{.*}} {
// CHECK: [[GO_CONTEXT:%[0-9]+]] = alloca %"{{.*}}LocalContext"
// CHECK: [[GO_TOKEN:%[0-9]+]] = call i64 @"{{.*}}EnterLocalContext"(ptr [[GO_CONTEXT]])
// CHECK: [[GO_RESULT:%[0-9]+]] = add i32 %0, 2
// CHECK: call void @"{{.*}}LeaveLocalContext"(ptr [[GO_CONTEXT]], i64 [[GO_TOKEN]])
// CHECK: ret i32 [[GO_RESULT]]

// CHECK-LABEL: define i32 @main.go_callback_c_only(i32 %0){{.*}} {
// CHECK: [[ONLY_CONTEXT:%[0-9]+]] = alloca %"{{.*}}LocalContext"
// CHECK: call i64 @"{{.*}}EnterLocalContext"(ptr [[ONLY_CONTEXT]])
// CHECK: [[ONLY_RESULT:%[0-9]+]] = add i32 %0, 2
// CHECK: call void @"{{.*}}LeaveLocalContext"(ptr [[ONLY_CONTEXT]], i64 %{{[0-9]+}})
// CHECK: ret i32 [[ONLY_RESULT]]

// main covers Go and C callback pointers, the C-only exported callback, and a
// deferred C free. The deferred pointer is boxed as keepalive data before the
// wrapper call, which is the compiler-owned lifetime edge.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MEMORY:%[0-9]+]] = call ptr @malloc(i64 64)
// CHECK: call ptr @"{{.*}}GetThreadDefer"()
// CHECK: [[HEAD:%[0-9]+]] = getelementptr inbounds nuw %"{{.*}}Defer", ptr %{{[0-9]+}}, i32 0, i32 5
// CHECK: [[C_CALLBACK:%[0-9]+]] = load ptr, ptr @main._Cfpvar_fp_c_callback
// CHECK-NEXT: [[C_CALLBACK_PTR:%[0-9]+]] = call ptr @main._Cgo_ptr(ptr [[C_CALLBACK]])
// CHECK-NEXT: call i32 @main._Cfunc_invoke_callback(ptr [[C_CALLBACK_PTR]], i32 39)
// CHECK: call i32 @main._Cfunc_invoke_export_only(i32 40)
// CHECK: [[GO_CALLBACK:%[0-9]+]] = load ptr, ptr @main._Cfpvar_fp_go_callback
// CHECK-NEXT: [[GO_CALLBACK_PTR:%[0-9]+]] = call ptr @main._Cgo_ptr(ptr [[GO_CALLBACK]])
// CHECK-NEXT: call i32 @main._Cfunc_invoke_callback(ptr [[GO_CALLBACK_PTR]], i32 40)
// Pop and release the defer node before invoking its function value. Keeping
// the node alive across the C call both leaks it on a non-returning call and
// lets a nested panic observe stale defer state.
// CHECK: [[ACTIVE_NODE:%[0-9]+]] = load ptr, ptr [[HEAD]]
// CHECK-NEXT: [[ACTIVE_DEFER:%[0-9]+]] = load { ptr, i64, { ptr, ptr } }, ptr [[ACTIVE_NODE]]
// CHECK: [[DEFER_FUNC:%[0-9]+]] = extractvalue { ptr, i64, { ptr, ptr } } [[ACTIVE_DEFER]], 2
// CHECK-NOT: FreeDeferNode
// CHECK: [[DEFER_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[DEFER_FUNC]], 0
// CHECK: call void %{{.*}}

// ARM64-LABEL: define void @"main.main$1$1"(ptr swiftself %0){{.*}} {
// AMD64-LABEL: define void @"main.main$1$1"(ptr nest %0){{.*}} {
// CHECK: [[DEFER_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK: [[KEEPALIVE_SLOT:%[0-9]+]] = extractvalue { ptr } [[DEFER_ENV]], 0
// CHECK-NEXT: [[KEEPALIVE:%[0-9]+]] = load ptr, ptr [[KEEPALIVE_SLOT]]
// CHECK-NEXT: insertvalue %"{{.*}}eface" { ptr @_llgo_Pointer, ptr undef }, ptr [[KEEPALIVE]], 1
// CHECK-NEXT: [[DEFER_ARG_SLOT:%[0-9]+]] = extractvalue { ptr } [[DEFER_ENV]], 0
// CHECK-NEXT: [[DEFER_ARG:%[0-9]+]] = load ptr, ptr [[DEFER_ARG_SLOT]]
// CHECK-NEXT: call [0 x i8] @main._Cfunc_free(ptr [[DEFER_ARG]])

//export go_callback
func go_callback(value C.int) C.int {
	return value + 2
}

//export go_callback_c_only
func go_callback_c_only(value C.int) C.int {
	return value + 2
}

func main() {
	memory := C.malloc(64)
	defer C.free(memory)

	if C.invoke_callback((C.callback_t)(C.go_callback), 40) != 42 {
		panic("Go callback")
	}
	if C.invoke_callback((C.callback_t)(C.c_callback), 39) != 42 {
		panic("C callback")
	}
	if C.invoke_export_only(40) != 42 {
		panic("C-only exported callback")
	}
}
