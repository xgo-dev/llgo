// LITTEST
package main

/*
#include "in.h"
*/
import "C"
import "fmt"

// Check the C aggregate ABI boundary: all five differently sized structs are
// passed by pointer through one generated C wrapper.
// CHECK-LABEL: define i32 @main._Cfunc_test_structs(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK: [[C_SLOT:%[0-9]+]] = load ptr, ptr @main._cgo_{{.*}}_Cfunc_test_structs
// CHECK-NEXT: [[C_TARGET:%[0-9]+]] = load ptr, ptr [[C_SLOT]]
// CHECK-NEXT: [[C_RESULT:%[0-9]+]] = call i32 [[C_TARGET]](ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT: ret i32 [[C_RESULT]]
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[S4:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 4)
// CHECK-NEXT: [[S4_A:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S4]], i32 0, i32 0
// CHECK-NEXT: store i32 1, ptr [[S4_A]]
// CHECK-NEXT: [[S8:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT: [[S8_A:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S8]], i32 0, i32 0
// CHECK-NEXT: [[S8_B:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S8]], i32 0, i32 1
// CHECK-NEXT: store i32 1, ptr [[S8_A]]
// CHECK-NEXT: store i32 2, ptr [[S8_B]]
// CHECK-NEXT: [[S12:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 12)
// CHECK-NEXT: [[S12_A:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S12]], i32 0, i32 0
// CHECK-NEXT: [[S12_B:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S12]], i32 0, i32 1
// CHECK-NEXT: [[S12_C:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S12]], i32 0, i32 2
// CHECK-NEXT: store i32 1, ptr [[S12_A]]
// CHECK-NEXT: store i32 2, ptr [[S12_B]]
// CHECK-NEXT: store i32 3, ptr [[S12_C]]
// CHECK-NEXT: [[S16:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: [[S16_A:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S16]], i32 0, i32 0
// CHECK-NEXT: [[S16_B:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S16]], i32 0, i32 1
// CHECK-NEXT: [[S16_C:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S16]], i32 0, i32 2
// CHECK-NEXT: [[S16_D:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S16]], i32 0, i32 3
// CHECK-NEXT: store i32 1, ptr [[S16_A]]
// CHECK-NEXT: store i32 2, ptr [[S16_B]]
// CHECK-NEXT: store i32 3, ptr [[S16_C]]
// CHECK-NEXT: store i32 4, ptr [[S16_D]]
// CHECK-NEXT: [[S20:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 20)
// CHECK-NEXT: [[S20_A:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S20]], i32 0, i32 0
// CHECK-NEXT: [[S20_B:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S20]], i32 0, i32 1
// CHECK-NEXT: [[S20_C:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S20]], i32 0, i32 2
// CHECK-NEXT: [[S20_D:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S20]], i32 0, i32 3
// CHECK-NEXT: [[S20_E:%[0-9]+]] = getelementptr inbounds %{{.*}}, ptr [[S20]], i32 0, i32 4
// CHECK-NEXT: store i32 1, ptr [[S20_A]]
// CHECK-NEXT: store i32 2, ptr [[S20_B]]
// CHECK-NEXT: store i32 3, ptr [[S20_C]]
// CHECK-NEXT: store i32 4, ptr [[S20_D]]
// CHECK-NEXT: store i32 5, ptr [[S20_E]]
// CHECK-NEXT: [[RESULT:%[0-9]+]] = call i32 @main._Cfunc_test_structs(ptr [[S4]], ptr [[S8]], ptr [[S12]], ptr [[S16]], ptr [[S20]])
// CHECK: store i32 [[RESULT]], ptr
// CHECK: [[FAILED:%[0-9]+]] = icmp ne i32 [[RESULT]], 35
// CHECK-NEXT: br i1 [[FAILED]], label %{{.*}}, label %{{.*}}

func main() {
	r := C.test_structs(&C.s4{a: 1}, &C.s8{a: 1, b: 2}, &C.s12{a: 1, b: 2, c: 3}, &C.s16{a: 1, b: 2, c: 3, d: 4}, &C.s20{a: 1, b: 2, c: 3, d: 4, e: 5})
	fmt.Println(r)
	if r != 35 {
		panic("test_structs failed")
	}
}
