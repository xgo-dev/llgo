// LITTEST
// Scope: os (host CGo errno and aggregate ABI)
package main

/*
#include "in.h"
*/
import "C"

var checkErrno = func(error) {}

// The two-result spelling selects the errno-aware C2 wrapper. Follow the
// foreign result and errno into the Go pair. Errno's concrete interface type
// and the success/error branch are C2-specific compiler contracts.
// CHECK-LABEL: define { i32, %"{{.*}}iface" } @main._C2func_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK: [[C2_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_C2func_test_structs
// CHECK: [[C2_FN:%[0-9]+]] = load ptr, ptr [[C2_SLOT]]
// CHECK: [[C2_RESULT:%[0-9]+]] = call i32 [[C2_FN]](ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK: [[ERRNO:%[0-9]+]] = call i32 @cliteErrno()
// CHECK: [[NIL_ERR:%[0-9]+]] = load %"{{.*}}iface", ptr %{{[0-9]+}}
// CHECK: [[HAS_ERR:%[0-9]+]] = icmp ne i32 [[ERRNO]], 0
// CHECK: [[ERRNO_VALUE:%[0-9]+]] = sext i32 [[ERRNO]] to i64
// CHECK: store i64 [[ERRNO_VALUE]], ptr [[ERRNO_BOX:%[0-9]+]]
// CHECK: [[ERR_IFACE:%[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr [[ERRNO_BOX]], 1
// CHECK: br i1 [[HAS_ERR]], label %[[ERR_BLOCK:[^, ]+]], label %[[OK_BLOCK:[^, ]+]]
// CHECK: [[ERR_PAIR:%[0-9]+]] = insertvalue { i32, %"{{.*}}iface" } {{.*}}, %"{{.*}}iface" [[ERR_IFACE]], 1
// CHECK-NEXT: ret { i32, %"{{.*}}iface" } [[ERR_PAIR]]
// CHECK: [[OK_PAIR:%[0-9]+]] = insertvalue { i32, %"{{.*}}iface" } {{.*}}, %"{{.*}}iface" [[NIL_ERR]], 1
// CHECK-NEXT: ret { i32, %"{{.*}}iface" } [[OK_PAIR]]

// The same external function also retains the plain single-result wrapper.
// CHECK-LABEL: define i32 @main._Cfunc_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK: [[C_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_test_structs
// CHECK-NEXT: [[C_FN:%[0-9]+]] = load ptr, ptr [[C_SLOT]]
// CHECK-NEXT: [[C_RESULT:%[0-9]+]] = call i32 [[C_FN]](ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT: ret i32 [[C_RESULT]]

// Five sizes retain the local-header/local-C-file aggregate surface while the
// checks stay on allocations and the two genuinely different wrapper calls.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[S4:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 4)
// CHECK: [[S8:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK: [[S12:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 12)
// CHECK: [[S16:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK: [[S20:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 20)
// CHECK: [[PLAIN:%[0-9]+]] = call i32 @main._Cfunc_test_structs(ptr [[S4]], ptr [[S8]], ptr [[S12]], ptr [[S16]], ptr [[S20]])
// CHECK: [[PAIR:%[0-9]+]] = call { i32, %"{{.*}}iface" } @main._C2func_test_structs(ptr [[S4]], ptr [[S8]], ptr [[S12]], ptr [[S16]], ptr [[S20]])
// CHECK: extractvalue { i32, %"{{.*}}iface" } [[PAIR]], 0
// CHECK: extractvalue { i32, %"{{.*}}iface" } [[PAIR]], 1
func main() {
	s4 := C.s4{a: 1}
	s8 := C.s8{a: 1, b: 2}
	s12 := C.s12{a: 1, b: 2, c: 3}
	s16 := C.s16{a: 1, b: 2, c: 3, d: 4}
	s20 := C.s20{a: 1, b: 2, c: 3, d: 4, e: 5}

	plain := C.test_structs(&s4, &s8, &s12, &s16, &s20)
	if plain != 35 {
		panic("plain aggregate wrapper")
	}

	withErrno, err := C.test_structs(&s4, &s8, &s12, &s16, &s20)
	if withErrno != 35 {
		panic("errno aggregate wrapper")
	}
	checkErrno(err)
}
