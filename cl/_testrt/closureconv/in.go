// LITTEST
package main

type Func func(a int, b int) int
type Func2 func(a int, b int) int

type Call struct {
	fn Func
	n  int
}

// The bound method must forward both call arguments and read n from the same
// receiver captured in the function environment.
// CHECK-LABEL: define i64 @"main.(*Call).add"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[ADD_AB:%.*]] = add i64 %1, %2
// CHECK: [[ADD_N_FIELD:%.*]] = getelementptr inbounds %main.Call, ptr %0, i32 0, i32 1
// CHECK-NEXT: [[ADD_N:%.*]] = load i64, ptr [[ADD_N_FIELD]]
// CHECK-NEXT: [[ADD_RESULT:%.*]] = add i64 [[ADD_AB]], [[ADD_N]]
// CHECK-NEXT: ret i64 [[ADD_RESULT]]
func (c *Call) add(a int, b int) int {
	return a + b + c.n
}

// CHECK-LABEL: define i64 @main.add(i64 %0, i64 %1){{.*}} {
// CHECK: [[ADD_PLAIN_RESULT:%.*]] = add i64 %0, %1
// CHECK-NEXT: ret i64 [[ADD_PLAIN_RESULT]]
func add(a int, b int) int {
	return a + b
}

// CHECK-LABEL: define %main.Func @main.demo1(i64 %0){{.*}} {
// CHECK: [[DEMO1_CALL:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK: [[DEMO1_N:%.*]] = getelementptr inbounds %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 1
// CHECK: store i64 %0, ptr [[DEMO1_N]]
// CHECK: [[DEMO1_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: [[DEMO1_ENV_SLOT:%.*]] = getelementptr inbounds { ptr }, ptr [[DEMO1_ENV]], i32 0, i32 0
// CHECK: store ptr [[DEMO1_CALL]], ptr [[DEMO1_ENV_SLOT]]
// CHECK: [[DEMO1_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.(*Call).add$bound", ptr undef }, ptr [[DEMO1_ENV]], 1
// CHECK: [[DEMO1_FN_FIELD:%.*]] = getelementptr inbounds %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 0
// CHECK: store { ptr, ptr } [[DEMO1_BOUND]], ptr [[DEMO1_BOUND_SLOT:%.*]]
// CHECK-NEXT: [[DEMO1_BOUND_VALUE:%.*]] = load %main.Func, ptr [[DEMO1_BOUND_SLOT]]
// CHECK-NEXT: store %main.Func [[DEMO1_BOUND_VALUE]], ptr [[DEMO1_FN_FIELD]]
// CHECK: [[DEMO1_RET_FIELD:%.*]] = getelementptr inbounds %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 0
// CHECK: [[DEMO1_RET:%.*]] = load %main.Func, ptr [[DEMO1_RET_FIELD]]
// CHECK: ret %main.Func [[DEMO1_RET]]
func demo1(n int) Func {
	m := &Call{n: n}
	m.fn = m.add
	return m.fn
}

// CHECK-LABEL: define %main.Func @main.demo2(){{.*}} {
// CHECK: [[DEMO2_CALL:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK: [[DEMO2_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: [[DEMO2_ENV_SLOT:%.*]] = getelementptr inbounds { ptr }, ptr [[DEMO2_ENV]], i32 0, i32 0
// CHECK: store ptr [[DEMO2_CALL]], ptr [[DEMO2_ENV_SLOT]]
// CHECK: [[DEMO2_BOUND:%.*]] = insertvalue { ptr, ptr } { ptr @"main.(*Call).add$bound", ptr undef }, ptr [[DEMO2_ENV]], 1
// CHECK: store { ptr, ptr } [[DEMO2_BOUND]], ptr [[DEMO2_OUT:%.*]]
// CHECK: [[DEMO2_RET:%.*]] = load %main.Func, ptr [[DEMO2_OUT]]
// CHECK: ret %main.Func [[DEMO2_RET]]

func demo2() Func {
	m := &Call{}
	return m.add
}

// CHECK-LABEL: define %main.Func @main.demo3(){{.*}} {
// CHECK: ret %main.Func { ptr @main.add, ptr null }

func demo3() Func {
	return add
}

// CHECK-LABEL: define %main.Func @main.demo4(){{.*}} {
// CHECK:   ret %main.Func { ptr @"main.demo4$1", ptr null }

// CHECK-LABEL: define i64 @"main.demo4$1"(i64 %0, i64 %1){{.*}} {
// CHECK: [[DEMO4_RESULT:%.*]] = add i64 %0, %1
// CHECK: ret i64 [[DEMO4_RESULT]]
func demo4() Func {
	return func(a, b int) int { return a + b }
}

// CHECK-LABEL: define %main.Func @main.demo5(i64 %0){{.*}} {
// CHECK: [[DEMO5_CAPTURE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: store i64 %0, ptr [[DEMO5_CAPTURE]]
// CHECK: [[DEMO5_ENV:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK: [[DEMO5_ENV_SLOT:%.*]] = getelementptr inbounds { ptr }, ptr [[DEMO5_ENV]], i32 0, i32 0
// CHECK: store ptr [[DEMO5_CAPTURE]], ptr [[DEMO5_ENV_SLOT]]
// CHECK: [[DEMO5_FN:%.*]] = insertvalue { ptr, ptr } { ptr @"main.demo5$1", ptr undef }, ptr [[DEMO5_ENV]], 1
// CHECK: store { ptr, ptr } [[DEMO5_FN]], ptr [[DEMO5_OUT:%.*]]
// CHECK: [[DEMO5_RET:%.*]] = load %main.Func, ptr [[DEMO5_OUT]]
// CHECK: ret %main.Func [[DEMO5_RET]]

// CHECK-LABEL: define i64 @"main.demo5$1"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[DEMO5_SUM:%.*]] = add i64 %1, %2
// CHECK: [[DEMO5_ENV_VALUE:%.*]] = load { ptr }, ptr %0
// CHECK: [[DEMO5_CAPTURE_ADDR:%.*]] = extractvalue { ptr } [[DEMO5_ENV_VALUE]], 0
// CHECK: [[DEMO5_CAPTURE_VALUE:%.*]] = load i64, ptr [[DEMO5_CAPTURE_ADDR]]
// CHECK: [[DEMO5_RESULT:%.*]] = add i64 [[DEMO5_SUM]], [[DEMO5_CAPTURE_VALUE]]
// CHECK: ret i64 [[DEMO5_RESULT]]
func demo5(n int) Func {
	return func(a, b int) int { return a + b + n }
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// Each returned function value supplies both the code and environment used by its call.
// CHECK: [[MAIN_F1:%.*]] = call %main.Func @main.demo1(i64 1)
// CHECK: [[MAIN_F1_ENV:%.*]] = extractvalue %main.Func [[MAIN_F1]], 1
// CHECK: [[MAIN_F1_CODE_RAW:%.*]] = extractvalue %main.Func [[MAIN_F1]], 0
// CHECK: [[MAIN_F1_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[MAIN_F1_CODE_RAW]])
// CHECK: [[MAIN_R1:%.*]] = call i64 [[MAIN_F1_CODE]](ptr {{(nest|swiftself)}} [[MAIN_F1_ENV]], i64 99, i64 200)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_R1]])
// The individual demo2, demo3, and demo4 functions above check their distinct
// construction forms; one captured closure is enough to cover invocation here.
// CHECK: [[MAIN_F5:%.*]] = call %main.Func @main.demo5(i64 1)
// CHECK: [[MAIN_F5_ENV:%.*]] = extractvalue %main.Func [[MAIN_F5]], 1
// CHECK: [[MAIN_F5_CODE_RAW:%.*]] = extractvalue %main.Func [[MAIN_F5]], 0
// CHECK: [[MAIN_F5_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[MAIN_F5_CODE_RAW]])
// CHECK: [[MAIN_R5:%.*]] = call i64 [[MAIN_F5_CODE]](ptr {{(nest|swiftself)}} [[MAIN_F5_ENV]], i64 99, i64 200)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_R5]])
// Conversion to the unnamed function type preserves the two function words.
// CHECK: [[PLAIN_SOURCE:%.*]] = call %main.Func @main.demo5(i64 1)
// CHECK: store %main.Func [[PLAIN_SOURCE]], ptr [[PLAIN_SLOT:%.*]]
// CHECK: [[PLAIN_FN:%.*]] = load { ptr, ptr }, ptr [[PLAIN_SLOT]]
// CHECK: [[PLAIN_ENV:%.*]] = extractvalue { ptr, ptr } [[PLAIN_FN]], 1
// CHECK: [[PLAIN_CODE_RAW:%.*]] = extractvalue { ptr, ptr } [[PLAIN_FN]], 0
// CHECK: [[PLAIN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[PLAIN_CODE_RAW]])
// CHECK: call i64 [[PLAIN_CODE]](ptr {{(nest|swiftself)}} [[PLAIN_ENV]], i64 99, i64 200)
// Conversion from Func to Func2 preserves code and environment independently.
// CHECK: [[FUNC2_SOURCE:%.*]] = call %main.Func @main.demo5(i64 1)
// CHECK: [[FUNC2_CODE:%.*]] = extractvalue %main.Func [[FUNC2_SOURCE]], 0
// CHECK: [[FUNC2_0:%.*]] = insertvalue %main.Func2 undef, ptr [[FUNC2_CODE]], 0
// CHECK: [[FUNC2_ENV:%.*]] = extractvalue %main.Func [[FUNC2_SOURCE]], 1
// CHECK: [[FUNC2:%.*]] = insertvalue %main.Func2 [[FUNC2_0]], ptr [[FUNC2_ENV]], 1
// CHECK: [[FUNC2_CALL_ENV:%.*]] = extractvalue %main.Func2 [[FUNC2]], 1
// CHECK: [[FUNC2_CALL_CODE_RAW:%.*]] = extractvalue %main.Func2 [[FUNC2]], 0
// CHECK: [[FUNC2_CALL_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[FUNC2_CALL_CODE_RAW]])
// CHECK: call i64 [[FUNC2_CALL_CODE]](ptr {{(nest|swiftself)}} [[FUNC2_CALL_ENV]], i64 99, i64 200)

func main() {
	n1 := demo1(1)(99, 200)
	println(n1)

	n2 := demo2()(100, 200)
	println(n2)

	n3 := demo3()(100, 200)
	println(n3)

	n4 := demo4()(100, 200)
	println(n4)

	n5 := demo5(1)(99, 200)
	println(n5)

	var fn func(a int, b int) int = demo5(1)
	println(fn(99, 200))

	var fn2 Func2 = (Func2)(demo5(1))
	println(fn2(99, 200))
}

// CHECK-LABEL: define i64 @"main.(*Call).add$bound"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[BOUND_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[BOUND_CALL:%.*]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK: [[BOUND_RESULT:%.*]] = call i64 @"main.(*Call).add"(ptr [[BOUND_CALL]], i64 %1, i64 %2)
// CHECK: ret i64 [[BOUND_RESULT]]
