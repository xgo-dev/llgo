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
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %2, label %3, label %4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %21
// CHECK-NEXT:   ret %main.stateFn zeroinitializer
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %21
// CHECK-NEXT:   ret %main.stateFn { ptr @__llgo_stub.main.countState, ptr null }
// CHECK-EMPTY:
// CHECK-NEXT: 3:                                                ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 4:                                                ; preds = %_llgo_0
// CHECK-NEXT:   %5 = load i64, ptr %1, align 8
// CHECK-NEXT:   %6 = add i64 %5, 1
// CHECK-NEXT:   %7 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %6, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %9, label %10, label %11
// CHECK-EMPTY:
// CHECK-NEXT: 10:                                               ; preds = %4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 11:                                               ; preds = %4
// CHECK-NEXT:   %12 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %13 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %14 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %14, label %15, label %16
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 16:                                               ; preds = %11
// CHECK-NEXT:   %17 = load i64, ptr %13, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds %main.counter, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %19 = icmp eq ptr %0, null
// CHECK-NEXT:   br i1 %19, label %20, label %21
// CHECK-EMPTY:
// CHECK-NEXT: 20:                                               ; preds = %16
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 21:                                               ; preds = %16
// CHECK-NEXT:   %22 = load i64, ptr %18, align 8
// CHECK-NEXT:   %23 = icmp sge i64 %17, %22
// CHECK-NEXT:   br i1 %23, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }
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
	// CHECK: store %main.stateFn { ptr @__llgo_stub.main.countState, ptr null }, ptr %2, align 8
	// CHECK: call %main.stateFn %6(ptr %5, ptr %0)
	// CHECK: store %main.stateFn %7, ptr %8, align 8
	// CHECK: icmp ne ptr %11, null
	// CHECK: br i1 %12, label %_llgo_1, label %_llgo_2
	c := &counter{max: 5, state: countState}

	for c.state != nil {
		c.state = c.state(c)
	}
}
