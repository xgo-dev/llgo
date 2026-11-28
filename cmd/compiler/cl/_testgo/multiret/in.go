// LITTEST darwin/arm64 linux/amd64
// Scope: arch (arm64/amd64 closure-call ABI; aggregate results otherwise common)
package main

var a int = 1

// Returning a closure inside an aggregate must preserve both function words;
// this is the historical result/funcPtr regression owner.
// CHECK-LABEL: define { { ptr, ptr }, i64 } @main.closureResult(){{.*}} {
// CHECK: ret { { ptr, ptr }, i64 } { { ptr, ptr } { ptr @"main.closureResult$1", ptr null }, i64 1 }
// CHECK-LABEL: define i64 @"main.closureResult$1"(i64 %0){{.*}} {
// CHECK: [[CLOSURE_RESULT:%[0-9]+]] = add i64 %0, 1
// CHECK-NEXT: ret i64 [[CLOSURE_RESULT]]
func closureResult() (func(int) int, int) {
	return func(v int) int { return v + 1 }, 1
}

// CHECK-LABEL: define { i64, double } @main.foo(double %0){{.*}} {
// CHECK: [[FOO_INT:%[0-9]+]] = load i64, ptr @main.a
// CHECK-NEXT: [[FOO_PAIR0:%[0-9]+]] = insertvalue { i64, double } undef, i64 [[FOO_INT]], 0
// CHECK-NEXT: [[FOO_PAIR:%[0-9]+]] = insertvalue { i64, double } [[FOO_PAIR0]], double %0, 1
// CHECK-NEXT: ret { i64, double } [[FOO_PAIR]]
func foo(f float64) (int, float64) {
	return a, f
}

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[FOO_RESULT:%[0-9]+]] = call { i64, double } @main.foo(double 2.000000e+00)
	// CHECK-NEXT: [[MAIN_INT:%[0-9]+]] = extractvalue { i64, double } [[FOO_RESULT]], 0
	// CHECK-NEXT: [[MAIN_FLOAT:%[0-9]+]] = extractvalue { i64, double } [[FOO_RESULT]], 1
	// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_INT]])
	// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double [[MAIN_FLOAT]])
	// CHECK: [[CLOSURE_PAIR:%[0-9]+]] = call { { ptr, ptr }, i64 } @main.closureResult()
	// CHECK-NEXT: [[CLOSURE_FN:%[0-9]+]] = extractvalue { { ptr, ptr }, i64 } [[CLOSURE_PAIR]], 0
	// CHECK-NEXT: [[CLOSURE_N:%[0-9]+]] = extractvalue { { ptr, ptr }, i64 } [[CLOSURE_PAIR]], 1
	// CHECK: [[CLOSURE_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[CLOSURE_FN]], 1
	// CHECK-NEXT: [[CLOSURE_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[CLOSURE_FN]], 0
	// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[CLOSURE_RAW_CODE]])
	// ARM64-NEXT: [[DYNAMIC_RESULT:%[0-9]+]] = call i64 %__llgo_funcval_code(ptr swiftself [[CLOSURE_ENV]], i64 41)
	// AMD64-NEXT: [[DYNAMIC_RESULT:%[0-9]+]] = call i64 %__llgo_funcval_code(ptr nest [[CLOSURE_ENV]], i64 41)
	// CHECK-NEXT: [[BAD_CLOSURE_RESULT:%[0-9]+]] = icmp ne i64 [[DYNAMIC_RESULT]], 42
	i, f := foo(2.0)
	println(i, f)
	fn, n := closureResult()
	if fn(41) != 42 || n != 1 {
		panic("bad closure result")
	}
}
