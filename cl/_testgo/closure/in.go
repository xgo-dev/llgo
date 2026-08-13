// LITTEST
package main

type T func(n int)

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// A closure without captures is represented by a direct code pointer, while
	// the captured closure is called with its environment.
	// CHECK: [[ENV_VALUE:%.*]] = call ptr @"{{.*}}AllocZ"(i64 16)
	// CHECK: store %"{{.*}}String" { ptr @{{[0-9]+}}, i64 3 }, ptr [[ENV_VALUE]]
	// CHECK: [[CLOSURE_ENV:%.*]] = call ptr @"{{.*}}AllocU"(i64 8)
	// CHECK: [[ENV_SLOT:%.*]] = getelementptr inbounds { ptr }, ptr [[CLOSURE_ENV]], i32 0, i32 0
	// CHECK: store ptr [[ENV_VALUE]], ptr [[ENV_SLOT]]
	// CHECK: [[V2:%.*]] = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr [[CLOSURE_ENV]], 1
	// CHECK: store { ptr, ptr } [[V2]], ptr [[V2_SLOT:%.*]]
	// CHECK: [[V2_VALUE:%.*]] = load %main.T, ptr [[V2_SLOT]]
	// CHECK: call void @"main.main$1"(i64 100)
	// CHECK: [[V2_CALL_ENV:%.*]] = extractvalue %main.T [[V2_VALUE]], 1
	// CHECK: [[V2_CALL_FN:%.*]] = extractvalue %main.T [[V2_VALUE]], 0
	// CHECK: [[V2_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[V2_CALL_FN]])
	// CHECK: call void [[V2_CODE]](ptr {{(nest|swiftself)}} [[V2_CALL_ENV]], i64 200)
	var env string = "env"
	var v1 T = func(i int) {
		// CHECK-LABEL: define void @"main.main$1"(i64 %0){{.*}} {
		// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
		println("func", i)
	}
	var v2 T = func(i int) {
		// CHECK-LABEL: define void @"main.main$2"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
		// CHECK: [[V2_ENV_VALUE:%.*]] = load { ptr }, ptr %0
		// CHECK-NEXT: [[V2_STRING_SLOT:%.*]] = extractvalue { ptr } [[V2_ENV_VALUE]], 0
		// CHECK-NEXT: [[V2_STRING:%.*]] = load %"{{.*}}String", ptr [[V2_STRING_SLOT]]
		// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
		// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}String" [[V2_STRING]])
		println("closure", i, env)
	}
	v1(100)
	v2(200)
}
