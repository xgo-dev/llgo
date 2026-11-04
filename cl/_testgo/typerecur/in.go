// LITTEST darwin/arm64 linux/amd64
package main

type stateFn func(*counter) stateFn

type counter struct {
	value int
	max   int
	state stateFn
}

// CHECK: %main.counter = type { i64, i64, ptr }
// CHECK: [[COUNT_STATE_DESCRIPTOR:@"_llgo_funcval\$[0-9a-f]+"]] = linkonce_odr constant { ptr, ptr } { ptr @main.countState, ptr null }
// CHECK-LABEL: define ptr @main.countState(ptr %0){{.*}} {
// CHECK: %[[VALUE_PTR1:[0-9]+]] = getelementptr inbounds nuw %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT: %[[OLD_VALUE:[0-9]+]] = load i64, ptr %[[VALUE_PTR1]]
// CHECK-NEXT: %[[NEW_VALUE:[0-9]+]] = add i64 %[[OLD_VALUE]], 1
// CHECK: store i64 %[[NEW_VALUE]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[PRINTED_VALUE:[0-9]+]])
// CHECK: %[[DONE:[0-9]+]] = icmp sge i64 %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT: br i1 %[[DONE]], label %{{.*}}, label %{{.*}}
// CHECK: ret ptr null
// CHECK: ret ptr [[COUNT_STATE_DESCRIPTOR]]
func countState(c *counter) stateFn {
	c.value++
	println("count:", c.value)

	if c.value >= c.max {
		return nil
	}
	return countState
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[COUNTER_OBJ:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK: %[[MAX_SLOT:[0-9]+]] = getelementptr inbounds nuw %main.counter, ptr %[[COUNTER_OBJ]], i32 0, i32 1
// CHECK-NEXT: %[[STATE_SLOT:[0-9]+]] = getelementptr inbounds nuw %main.counter, ptr %[[COUNTER_OBJ]], i32 0, i32 2
// CHECK-NEXT: store i64 5, ptr %[[MAX_SLOT]]
// CHECK-NEXT: store ptr [[COUNT_STATE_DESCRIPTOR]], ptr %[[STATE_SLOT]]
// CHECK: %[[STATE:[0-9]+]] = load ptr, ptr %{{[0-9]+}}
// CHECK: [[STATE_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %[[STATE]]
// CHECK-NEXT: [[STATE_ENV_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[STATE_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: %[[STATE_ENV:[0-9]+]] = load ptr, ptr [[STATE_ENV_SLOT]]
// CHECK: [[STATE_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %[[STATE]]
// CHECK-NEXT: [[STATE_CODE_SLOT:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[STATE_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: %[[STATE_CODE:[0-9]+]] = load ptr, ptr [[STATE_CODE_SLOT]]
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %[[STATE_CODE]])
// ARM64-NEXT: %[[NEXT_STATE:[0-9]+]] = call ptr %__llgo_funcval_code(ptr swiftself %[[STATE_ENV]], ptr %[[COUNTER_OBJ]])
// AMD64-NEXT: %[[NEXT_STATE:[0-9]+]] = call ptr %__llgo_funcval_code(ptr nest %[[STATE_ENV]], ptr %[[COUNTER_OBJ]])
// CHECK: store ptr %[[NEXT_STATE]], ptr %{{[0-9]+}}
func main() {
	c := &counter{max: 5, state: countState}

	for c.state != nil {
		c.state = c.state(c)
	}
}
