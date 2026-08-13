// LITTEST
package main

/*
#include <stdlib.h>
*/
import "C"

// CHECK-LABEL: define {{.*}} @main._Cfunc_free(ptr %0){{.*}} {
// CHECK: [[FREE_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_free
// CHECK-NEXT: [[FREE_FN:%[0-9]+]] = load ptr, ptr [[FREE_SLOT]]
// CHECK-NEXT: [[FREE_RESULT:%[0-9]+]] = call [0 x i8] [[FREE_FN]](ptr %0)
// CHECK-NEXT: ret [0 x i8] [[FREE_RESULT]]
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call ptr @malloc(i64 1024)
// CHECK: call ptr @"{{.*}}GetThreadDefer"()
// CHECK: call void @"{{.*}}FreeDeferNode"
// CHECK-LABEL: define void @"main.main$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[DEFER_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK: [[DEFER_KEEPALIVE_SLOT:%[0-9]+]] = extractvalue { ptr } [[DEFER_ENV]], 0
// CHECK-NEXT: [[DEFER_KEEPALIVE:%[0-9]+]] = load ptr, ptr [[DEFER_KEEPALIVE_SLOT]]
// CHECK-NEXT: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_Pointer, ptr undef }, ptr [[DEFER_KEEPALIVE]], 1
// CHECK-NEXT: [[DEFER_ARG_SLOT:%[0-9]+]] = extractvalue { ptr } [[DEFER_ENV]], 0
// CHECK-NEXT: [[DEFER_ARG:%[0-9]+]] = load ptr, ptr [[DEFER_ARG_SLOT]]
// CHECK-NEXT: call [0 x i8] @main._Cfunc_free(ptr [[DEFER_ARG]])

func main() {
	p := C.malloc(1024)
	defer C.free(p)
}
