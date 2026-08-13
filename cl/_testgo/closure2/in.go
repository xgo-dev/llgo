// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// Both nested functions are invoked through function values with closure
	// environments, and the inner closure keeps the original x slot alive.
	// CHECK: [[X_SLOT:%.*]] = call ptr @"{{.*}}AllocZ"(i64 8)
	// CHECK: store i64 1, ptr [[X_SLOT]]
	// CHECK: [[OUTER_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
	// CHECK: store ptr [[X_SLOT]], ptr {{%.*}}
	// CHECK: [[OUTER:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[OUTER_ENV]], 1
	// CHECK: [[OUTER_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[OUTER]], 1
	// CHECK: [[OUTER_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[OUTER]], 0
	// CHECK: [[OUTER_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[OUTER_CALL_FN]])
	// CHECK: [[INNER:%.*]] = call { ptr, ptr } [[OUTER_CODE]](ptr {{(nest|swiftself)}} [[OUTER_CALL_ENV]], i64 1)
	// CHECK: [[INNER_CALL_ENV:%.*]] = extractvalue { ptr, ptr } [[INNER]], 1
	// CHECK: [[INNER_CALL_FN:%.*]] = extractvalue { ptr, ptr } [[INNER]], 0
	// CHECK: [[INNER_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[INNER_CALL_FN]])
	// CHECK: call void [[INNER_CODE]](ptr {{(nest|swiftself)}} [[INNER_CALL_ENV]], i64 2)
	x := 1
	f := func(i int) func(int) {
		// CHECK-LABEL: define { ptr, ptr } @"main.main$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
		// CHECK: [[OUTER_CAPTURE:%.*]] = load { ptr }, ptr %0
		// CHECK-NEXT: [[CAPTURED_X_SLOT:%.*]] = extractvalue { ptr } [[OUTER_CAPTURE]], 0
		// CHECK: [[INNER_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
		// CHECK: store ptr [[CAPTURED_X_SLOT]], ptr {{%.*}}
		// CHECK: [[INNER_VALUE:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$1$1", ptr undef }, ptr [[INNER_ENV]], 1
		// CHECK-NEXT: ret { ptr, ptr } [[INNER_VALUE]]
		return func(i int) {
			// CHECK-LABEL: define void @"main.main$1$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
			// CHECK: [[INNER_CAPTURE:%.*]] = load { ptr }, ptr %0
			// CHECK-NEXT: [[INNER_X_SLOT:%.*]] = extractvalue { ptr } [[INNER_CAPTURE]], 0
			// CHECK-NEXT: [[INNER_X:%.*]] = load i64, ptr [[INNER_X_SLOT]]
			// CHECK: call void @"{{.*}}PrintInt"(i64 %1)
			// CHECK: call void @"{{.*}}PrintInt"(i64 [[INNER_X]])
			println("closure", i, x)
		}
	}
	f(1)(2)
}
