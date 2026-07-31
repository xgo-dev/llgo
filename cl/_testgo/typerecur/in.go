// LITTEST
package main

type stateFn func(*counter) stateFn

type counter struct {
	value int
	max   int
	state stateFn
}

// CHECK-LABEL: define %main.stateFn @main.countState(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %2 = load i64, ptr %1, align 8
// CHECK-NEXT:   %3 = add i64 %2, 1
// CHECK-NEXT:   %4 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %3, ptr %4, align 8
// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @0, i64 6 })
// CHECK: call void @"{{.*}}PrintInt"(i64 %6)
// CHECK: icmp sge i64 %8, %10
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
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 32)
	// CHECK: store i64 5, ptr %1, align 8
	// CHECK: store %main.stateFn { ptr @main.countState, ptr null }, ptr %2, align 8
	// CHECK: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %6)
	// CHECK: call %main.stateFn %__llgo_funcval_code(ptr {{(nest|swiftself)}} %5, ptr %0)
	// CHECK: store %main.stateFn %7, ptr %8, align 8
	// CHECK: icmp ne ptr %11, null
	// CHECK: br i1 %12, label %_llgo_1, label %_llgo_2
	c := &counter{max: 5, state: countState}

	for c.state != nil {
		c.state = c.state(c)
	}
}
