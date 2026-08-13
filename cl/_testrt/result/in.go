// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define { ptr, ptr } @main.add(){{.*}} {
// CHECK: ret { ptr, ptr } { ptr @"main.add$1", ptr null }
// CHECK-LABEL: define i64 @"main.add$1"(i64 %0, i64 %1){{.*}} {
// CHECK: [[ADD_SUM:%[0-9]+]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[ADD_SUM]]
func add() func(int, int) int {
	return func(x, y int) int {
		return x + y
	}
}

// CHECK-LABEL: define { { ptr, ptr }, i64 } @main.add2(){{.*}} {
// CHECK:   ret { { ptr, ptr }, i64 } { { ptr, ptr } { ptr @"main.add2$1", ptr null }, i64 1 }
func add2() (func(int, int) int, int) {
	// CHECK-LABEL: define i64 @"main.add2$1"(i64 %0, i64 %1){{.*}} {
	// CHECK: [[ADD2_SUM:%[0-9]+]] = add i64 %0, %1
	// CHECK-NEXT: ret i64 [[ADD2_SUM]]
	return func(x, y int) int {
		return x + y
	}, 1
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[LOCAL_FN:%[0-9]+]] = call { ptr, ptr } @"main.main$1"()
// CHECK-NEXT: [[LOCAL_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[LOCAL_FN]], 1
// CHECK-NEXT: [[LOCAL_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[LOCAL_FN]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[LOCAL_RAW_CODE]])
// CHECK-NEXT: [[LOCAL_SUM:%[0-9]+]] = call i64 %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[LOCAL_ENV]], i64 100, i64 200)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[LOCAL_SUM]])
// CHECK-NEXT: [[ADD_FN:%[0-9]+]] = call { ptr, ptr } @main.add()
// CHECK-NEXT: [[ADD_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[ADD_FN]], 1
// CHECK-NEXT: [[ADD_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[ADD_FN]], 0
// CHECK-NEXT: %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr [[ADD_RAW_CODE]])
// CHECK-NEXT: [[ADD_VALUE:%[0-9]+]] = call i64 %__llgo_funcval_code1(ptr {{(nest|swiftself)}} [[ADD_ENV]], i64 100, i64 200)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[ADD_VALUE]])
// CHECK-NEXT: [[ADD2:%[0-9]+]] = call { { ptr, ptr }, i64 } @main.add2()
// CHECK-NEXT: extractvalue { { ptr, ptr }, i64 } [[ADD2]], 0
// CHECK-NEXT: [[ADD2_N:%[0-9]+]] = extractvalue { { ptr, ptr }, i64 } [[ADD2]], 1
// CHECK-NEXT: [[FINAL_FN:%[0-9]+]] = call { ptr, ptr } @main.add()
// CHECK-NEXT: [[FINAL_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[FINAL_FN]], 1
// CHECK-NEXT: [[FINAL_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[FINAL_FN]], 0
// CHECK-NEXT: %__llgo_funcval_code2 = call ptr asm "", "=r,0"(ptr [[FINAL_RAW_CODE]])
// CHECK-NEXT: [[FINAL_SUM:%[0-9]+]] = call i64 %__llgo_funcval_code2(ptr {{(nest|swiftself)}} [[FINAL_ENV]], i64 100, i64 200)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[FINAL_SUM]], i64 [[ADD2_N]])
func main() {
	// CHECK-LABEL: define { ptr, ptr } @"main.main$1"(){{.*}} {
	// CHECK:   ret { ptr, ptr } { ptr @"main.main$1$1", ptr null }
	fn := func() func(int, int) int {
		// CHECK-LABEL: define i64 @"main.main$1$1"(i64 %0, i64 %1){{.*}} {
		// CHECK: [[MAIN_SUM:%[0-9]+]] = add i64 %0, %1
		// CHECK-NEXT: ret i64 [[MAIN_SUM]]
		return func(x, y int) int {
			return x + y
		}
	}()
	c.Printf(c.Str("%d\n"), fn(100, 200))
	c.Printf(c.Str("%d\n"), add()(100, 200))
	fn, n := add2()
	c.Printf(c.Str("%d %d\n"), add()(100, 200), n)
}
