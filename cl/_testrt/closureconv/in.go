// LITTEST darwin/arm64 linux/amd64
package main

type Func func(a int, b int) int
type Func2 func(a int, b int) int

type Call struct {
	fn Func
	n  int
}

// CHECK-DAG: [[ADD_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @main.add, ptr null }
// CHECK-DAG: [[DEMO4_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @"main.demo4$1", ptr null }

// The bound method must forward both call arguments and read n from the same
// receiver captured in the function environment.
// CHECK-LABEL: define i64 @"main.(*Call).add"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[ADD_AB:%.*]] = add i64 %1, %2
// CHECK: [[ADD_N_FIELD:%.*]] = getelementptr inbounds nuw %main.Call, ptr %0, i32 0, i32 1
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

// CHECK-LABEL: define ptr @main.demo1(i64 %0){{.*}} {
// CHECK: [[DEMO1_CALL:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: [[DEMO1_N:%.*]] = getelementptr inbounds nuw %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 1
// CHECK: store i64 %0, ptr [[DEMO1_N]]
// CHECK: [[DEMO1_ALLOCATION:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT: [[DEMO1_DESCRIPTOR:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[DEMO1_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[DEMO1_ENV:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[DEMO1_ALLOCATION]], i32 0, i32 1
// CHECK-NEXT: [[DEMO1_CAPTURE_SLOT:%.*]] = getelementptr inbounds nuw { ptr }, ptr [[DEMO1_ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[DEMO1_CALL]], ptr [[DEMO1_CAPTURE_SLOT]]
// CHECK-NEXT: [[DEMO1_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO1_DESCRIPTOR]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.(*Call).add$bound", ptr [[DEMO1_CODE_SLOT]]
// CHECK-NEXT: [[DEMO1_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO1_DESCRIPTOR]], i32 0, i32 1
// CHECK-NEXT: store ptr [[DEMO1_ENV]], ptr [[DEMO1_ENV_SLOT]]
// CHECK: [[DEMO1_FN_FIELD:%.*]] = getelementptr inbounds nuw %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 0
// CHECK: store ptr [[DEMO1_DESCRIPTOR]], ptr [[DEMO1_FN_FIELD]]
// CHECK: [[DEMO1_RET_FIELD:%.*]] = getelementptr inbounds nuw %main.Call, ptr [[DEMO1_CALL]], i32 0, i32 0
// CHECK: [[DEMO1_RET:%.*]] = load ptr, ptr [[DEMO1_RET_FIELD]]
// CHECK: ret ptr [[DEMO1_RET]]
func demo1(n int) Func {
	m := &Call{n: n}
	m.fn = m.add
	return m.fn
}

// CHECK-LABEL: define ptr @main.demo2(){{.*}} {
// CHECK: [[DEMO2_CALL:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK: [[DEMO2_ALLOCATION:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT: [[DEMO2_DESCRIPTOR:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[DEMO2_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[DEMO2_ENV:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { ptr } }, ptr [[DEMO2_ALLOCATION]], i32 0, i32 1
// CHECK-NEXT: [[DEMO2_CAPTURE_SLOT:%.*]] = getelementptr inbounds nuw { ptr }, ptr [[DEMO2_ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[DEMO2_CALL]], ptr [[DEMO2_CAPTURE_SLOT]]
// CHECK-NEXT: [[DEMO2_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO2_DESCRIPTOR]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.(*Call).add$bound", ptr [[DEMO2_CODE_SLOT]]
// CHECK-NEXT: [[DEMO2_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO2_DESCRIPTOR]], i32 0, i32 1
// CHECK-NEXT: store ptr [[DEMO2_ENV]], ptr [[DEMO2_ENV_SLOT]]
// CHECK: ret ptr [[DEMO2_DESCRIPTOR]]

func demo2() Func {
	m := &Call{}
	return m.add
}

// CHECK-LABEL: define ptr @main.demo3(){{.*}} {
// CHECK: ret ptr [[ADD_DESCRIPTOR]]

func demo3() Func {
	return add
}

// CHECK-LABEL: define ptr @main.demo4(){{.*}} {
// CHECK: ret ptr [[DEMO4_DESCRIPTOR]]

// CHECK-LABEL: define i64 @"main.demo4$1"(i64 %0, i64 %1){{.*}} {
// CHECK: [[DEMO4_RESULT:%.*]] = add i64 %0, %1
// CHECK: ret i64 [[DEMO4_RESULT]]
func demo4() Func {
	return func(a, b int) int { return a + b }
}

// CHECK-LABEL: define ptr @main.demo5(i64 %0){{.*}} {
// CHECK: [[DEMO5_CAPTURE:%.*]] = alloca i64
// CHECK: store i64 %0, ptr [[DEMO5_CAPTURE]]
// CHECK-NEXT: [[DEMO5_COPY:%.*]] = load i64, ptr [[DEMO5_CAPTURE]]
// CHECK: [[DEMO5_ALLOCATION:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocAtomicU"(i64 24)
// CHECK-NEXT: [[DEMO5_DESCRIPTOR:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[DEMO5_ALLOCATION]], i32 0, i32 0
// CHECK-NEXT: [[DEMO5_ENV:%.*]] = getelementptr inbounds nuw { { ptr, ptr }, { i64 } }, ptr [[DEMO5_ALLOCATION]], i32 0, i32 1
// CHECK-NEXT: [[DEMO5_CAPTURE_SLOT:%.*]] = getelementptr inbounds nuw { i64 }, ptr [[DEMO5_ENV]], i32 0, i32 0
// CHECK-NEXT: store i64 [[DEMO5_COPY]], ptr [[DEMO5_CAPTURE_SLOT]]
// CHECK-NEXT: [[DEMO5_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO5_DESCRIPTOR]], i32 0, i32 0
// CHECK-NEXT: store ptr @"main.demo5$1", ptr [[DEMO5_CODE_SLOT]]
// CHECK-NEXT: [[DEMO5_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[DEMO5_DESCRIPTOR]], i32 0, i32 1
// CHECK-NEXT: store ptr [[DEMO5_ENV]], ptr [[DEMO5_ENV_SLOT]]
// CHECK: ret ptr [[DEMO5_DESCRIPTOR]]

// ARM64-LABEL: define hidden i64 @"main.demo5$1"(ptr swiftself %0, i64 %1, i64 %2){{.*}} {
// AMD64-LABEL: define hidden i64 @"main.demo5$1"(ptr nest %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[DEMO5_SUM:%.*]] = add i64 %1, %2
// CHECK: [[DEMO5_CAPTURE_ADDR:%.*]] = getelementptr inbounds nuw { i64 }, ptr %0, i32 0, i32 0
// CHECK: [[DEMO5_CAPTURE_VALUE:%.*]] = load i64, ptr [[DEMO5_CAPTURE_ADDR]]
// CHECK: [[DEMO5_RESULT:%.*]] = add i64 [[DEMO5_SUM]], [[DEMO5_CAPTURE_VALUE]]
// CHECK: ret i64 [[DEMO5_RESULT]]
func demo5(n int) Func {
	return func(a, b int) int { return a + b + n }
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_F1:%.*]] = call ptr @main.demo1(i64 1)
// CHECK: [[MAIN_F1_ENV_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[MAIN_F1]]
// CHECK-NEXT: [[MAIN_F1_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[MAIN_F1_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[MAIN_F1_ENV:%.*]] = load ptr, ptr [[MAIN_F1_ENV_SLOT]]
// CHECK: [[MAIN_F1_CODE_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[MAIN_F1]]
// CHECK-NEXT: [[MAIN_F1_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[MAIN_F1_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[MAIN_F1_CODE_RAW:%.*]] = load ptr, ptr [[MAIN_F1_CODE_SLOT]]
// CHECK: [[MAIN_F1_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[MAIN_F1_CODE_RAW]])
// ARM64: [[MAIN_R1:%.*]] = call i64 [[MAIN_F1_CODE]](ptr swiftself [[MAIN_F1_ENV]], i64 99, i64 200)
// AMD64: [[MAIN_R1:%.*]] = call i64 [[MAIN_F1_CODE]](ptr nest [[MAIN_F1_ENV]], i64 99, i64 200)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_R1]])
// CHECK: [[MAIN_F5:%.*]] = call ptr @main.demo5(i64 1)
// CHECK: [[MAIN_F5_ENV_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[MAIN_F5]]
// CHECK-NEXT: [[MAIN_F5_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[MAIN_F5_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[MAIN_F5_ENV:%.*]] = load ptr, ptr [[MAIN_F5_ENV_SLOT]]
// CHECK: [[MAIN_F5_CODE_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[MAIN_F5]]
// CHECK-NEXT: [[MAIN_F5_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[MAIN_F5_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[MAIN_F5_CODE_RAW:%.*]] = load ptr, ptr [[MAIN_F5_CODE_SLOT]]
// CHECK: [[MAIN_F5_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[MAIN_F5_CODE_RAW]])
// ARM64: [[MAIN_R5:%.*]] = call i64 [[MAIN_F5_CODE]](ptr swiftself [[MAIN_F5_ENV]], i64 99, i64 200)
// AMD64: [[MAIN_R5:%.*]] = call i64 [[MAIN_F5_CODE]](ptr nest [[MAIN_F5_ENV]], i64 99, i64 200)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[MAIN_R5]])
// CHECK: [[PLAIN_SOURCE:%.*]] = call ptr @main.demo5(i64 1)
// CHECK: [[PLAIN_ENV_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[PLAIN_SOURCE]]
// CHECK-NEXT: [[PLAIN_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[PLAIN_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[PLAIN_ENV:%.*]] = load ptr, ptr [[PLAIN_ENV_SLOT]]
// CHECK: [[PLAIN_CODE_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[PLAIN_SOURCE]]
// CHECK-NEXT: [[PLAIN_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[PLAIN_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[PLAIN_CODE_RAW:%.*]] = load ptr, ptr [[PLAIN_CODE_SLOT]]
// CHECK: [[PLAIN_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[PLAIN_CODE_RAW]])
// ARM64: call i64 [[PLAIN_CODE]](ptr swiftself [[PLAIN_ENV]], i64 99, i64 200)
// AMD64: call i64 [[PLAIN_CODE]](ptr nest [[PLAIN_ENV]], i64 99, i64 200)
// CHECK: [[FUNC2_SOURCE:%.*]] = call ptr @main.demo5(i64 1)
// CHECK: [[FUNC2_ENV_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FUNC2_SOURCE]]
// CHECK-NEXT: [[FUNC2_ENV_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FUNC2_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[FUNC2_ENV:%.*]] = load ptr, ptr [[FUNC2_ENV_SLOT]]
// CHECK: [[FUNC2_CODE_SAFE:%.*]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr [[FUNC2_SOURCE]]
// CHECK-NEXT: [[FUNC2_CODE_SLOT:%.*]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[FUNC2_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[FUNC2_CODE_RAW:%.*]] = load ptr, ptr [[FUNC2_CODE_SLOT]]
// CHECK: [[FUNC2_CODE:%.*]] = call ptr asm "", "=r,0"(ptr [[FUNC2_CODE_RAW]])
// ARM64: call i64 [[FUNC2_CODE]](ptr swiftself [[FUNC2_ENV]], i64 99, i64 200)
// AMD64: call i64 [[FUNC2_CODE]](ptr nest [[FUNC2_ENV]], i64 99, i64 200)

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

// ARM64-LABEL: define hidden i64 @"main.(*Call).add$bound"(ptr swiftself %0, i64 %1, i64 %2){{.*}} {
// AMD64-LABEL: define hidden i64 @"main.(*Call).add$bound"(ptr nest %0, i64 %1, i64 %2){{.*}} {
// CHECK: [[BOUND_ENV:%.*]] = load { ptr }, ptr %0
// CHECK: [[BOUND_CALL:%.*]] = extractvalue { ptr } [[BOUND_ENV]], 0
// CHECK: [[BOUND_RESULT:%.*]] = call i64 @"main.(*Call).add"(ptr [[BOUND_CALL]], i64 %1, i64 %2)
// CHECK: ret i64 [[BOUND_RESULT]]
