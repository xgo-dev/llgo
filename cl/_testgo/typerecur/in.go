// LITTEST
package main

type stateFn func(*counter) stateFn

type counter struct {
	value int
	max   int
	state stateFn
}

// CHECK-LABEL: define %main.stateFn @main.countState(ptr %0){{.*}} {
// CHECK: %[[VALUE_PTR1:[0-9]+]] = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT: %[[OLD_VALUE:[0-9]+]] = load i64, ptr %[[VALUE_PTR1]]
// CHECK-NEXT: %[[NEW_VALUE:[0-9]+]] = add i64 %[[OLD_VALUE]], 1
// CHECK: store i64 %[[NEW_VALUE]], ptr %{{[0-9]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %[[PRINTED_VALUE:[0-9]+]])
// CHECK: %[[DONE:[0-9]+]] = icmp sge i64 %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT: br i1 %[[DONE]], label %{{.*}}, label %{{.*}}
// CHECK: ret %main.stateFn zeroinitializer
// CHECK: ret %main.stateFn { ptr @main.countState, ptr null }
func countState(c *counter) stateFn {
	c.value++
	println("count:", c.value)

	if c.value >= c.max {
		return nil
	}
	return countState
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[COUNTER_OBJ:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK: %[[MAX_SLOT:[0-9]+]] = getelementptr inbounds %main.counter, ptr %[[COUNTER_OBJ]], i32 0, i32 1
// CHECK-NEXT: %[[STATE_SLOT:[0-9]+]] = getelementptr inbounds %main.counter, ptr %[[COUNTER_OBJ]], i32 0, i32 2
// CHECK-NEXT: store i64 5, ptr %[[MAX_SLOT]]
// CHECK-NEXT: store %main.stateFn { ptr @main.countState, ptr null }, ptr %[[STATE_SLOT]]
// CHECK: %[[STATE:[0-9]+]] = load %main.stateFn, ptr %{{[0-9]+}}
// CHECK-NEXT: %[[STATE_ENV:[0-9]+]] = extractvalue %main.stateFn %[[STATE]], 1
// CHECK-NEXT: %[[STATE_CODE:[0-9]+]] = extractvalue %main.stateFn %[[STATE]], 0
// CHECK: %[[NEXT_STATE:[0-9]+]] = call %main.stateFn %__llgo_funcval_code(ptr {{(nest|swiftself)}} %[[STATE_ENV]], ptr %[[COUNTER_OBJ]])
// CHECK: store %main.stateFn %[[NEXT_STATE]], ptr %{{[0-9]+}}
func main() {
	c := &counter{max: 5, state: countState}

	for c.state != nil {
		c.state = c.state(c)
	}
}
