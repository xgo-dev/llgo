// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[CAPTURED_M:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: store i64 200, ptr [[CAPTURED_M]]
// CHECK: [[CLOSURE_ENV:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: [[CLOSURE_ENV_FIELD:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[CLOSURE_ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[CAPTURED_M]], ptr [[CLOSURE_ENV_FIELD]]
// CHECK: [[CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr [[CLOSURE_ENV]], 1
// CHECK: store { ptr, ptr } [[CLOSURE]], ptr [[CLOSURE_BOX:%[0-9]+]]
// CHECK-NEXT: [[CLOSURE_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr undef }, ptr [[CLOSURE_BOX]], 1
// CHECK-NEXT: [[CLOSURE_TYPE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[CLOSURE_EFACE]], 0
// CHECK-NEXT: [[CLOSURE_MATCH:%[0-9]+]] = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr [[CLOSURE_TYPE]])
// CHECK: [[ASSERTED_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[ASSERTED_CLOSURE:%[0-9]+]], 1
// CHECK-NEXT: [[ASSERTED_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[ASSERTED_CLOSURE]], 0
// CHECK: [[CLOSURE_RESULT:%[0-9]+]] = call i64 %{{.*}}(ptr {{(nest|swiftself)}} [[ASSERTED_ENV]], i64 100)
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[CLOSURE_RESULT]])
// CHECK: [[ASSERTED_CLOSURE]] = extractvalue { { ptr, ptr }, i1 } %{{[0-9]+}}, 0
func main() {
	var m int = 200
	fn := func(n int) int {
		return m + n
	}
	var i any = fn
	f, ok := i.(func(int) int)
	if !ok {
		panic("error")
	}
	println(f(100))
}

// CHECK-LABEL: define i64 @"main.main$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
// CHECK: [[CLOSURE_BODY_ENV:%[0-9]+]] = load { ptr }, ptr %0
// CHECK-NEXT: [[CLOSURE_BODY_M_PTR:%[0-9]+]] = extractvalue { ptr } [[CLOSURE_BODY_ENV]], 0
// CHECK-NEXT: [[CLOSURE_BODY_M:%[0-9]+]] = load i64, ptr [[CLOSURE_BODY_M_PTR]]
// CHECK-NEXT: [[CLOSURE_BODY_RESULT:%[0-9]+]] = add i64 [[CLOSURE_BODY_M]], %1
// CHECK-NEXT: ret i64 [[CLOSURE_BODY_RESULT]]
