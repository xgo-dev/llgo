// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"main.main$1"(i64 100, i64 200)
// CHECK-NEXT: [[FN1:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: store { ptr, ptr } { ptr @"main.main$2", ptr null }, ptr [[FN1]]
// CHECK-NEXT: [[ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: [[FN1_CAPTURE:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[FN1]], ptr [[FN1_CAPTURE]]
// CHECK-NEXT: [[FN2:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$3", ptr undef }, ptr [[ENV]], 1
// CHECK-NEXT: [[FN2_DATA:%[0-9]+]] = extractvalue { ptr, ptr } [[FN2]], 1
// CHECK-NEXT: [[FN2_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[FN2]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[FN2_CODE]])
// CHECK-NEXT: call void %__llgo_funcval_code(ptr swiftself [[FN2_DATA]])
func main() {
	// CHECK-LABEL: define void @"main.main$1"(i64 %0, i64 %1){{.*}} {
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 %0, i64 %1)
	func(n1, n2 int) {
		c.Printf(c.Str("%d %d\n"), n1, n2)
	}(100, 200)

	// CHECK-LABEL: define void @"main.main$2"(i64 %0, i64 %1){{.*}} {
	// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 %0, i64 %1)
	// CHECK-NEXT: ret void
	fn1 := func(n1, n2 int) {
		c.Printf(c.Str("%d %d\n"), n1, n2)
	}

	// CHECK-LABEL: define void @"main.main$3"(ptr swiftself %0){{.*}} {
	// CHECK: [[CAPTURED_ENV:%[0-9]+]] = load { ptr }, ptr %0
	// CHECK-NEXT: [[FN1_ADDR:%[0-9]+]] = extractvalue { ptr } [[CAPTURED_ENV]], 0
	// CHECK-NEXT: [[CAPTURED_FN1:%[0-9]+]] = load { ptr, ptr }, ptr [[FN1_ADDR]]
	// CHECK-NEXT: [[CAPTURED_FN1_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[CAPTURED_FN1]], 1
	// CHECK-NEXT: [[CAPTURED_FN1_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[CAPTURED_FN1]], 0
	// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[CAPTURED_FN1_RAW_CODE]])
	// CHECK-NEXT: call void %__llgo_funcval_code(ptr swiftself [[CAPTURED_FN1_ENV]], i64 100, i64 200)
	fn2 := func() {
		fn1(100, 200)
	}
	fn2()
}
