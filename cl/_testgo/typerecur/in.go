// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"count:", align 1

type stateFn func(*counter) stateFn

type counter struct {
	value int
	max   int
	state stateFn
}

// CHECK-LABEL: define %"{{.*}}/cl/_testgo/typerecur.stateFn" @"{{.*}}/cl/_testgo/typerecur.countState"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   %5 = add i64 %4, 1
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %5, ptr %8, align 8
// CHECK-NEXT:   %9 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %14)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %16 = load i64, ptr %15, align 8
// CHECK-NEXT:   %17 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %17)
// CHECK-NEXT:   %18 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %18)
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %20 = load i64, ptr %19, align 8
// CHECK-NEXT:   %21 = icmp sge i64 %16, %20
// CHECK-NEXT:   br i1 %21, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/typerecur.stateFn" zeroinitializer
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/typerecur.stateFn" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/typerecur.countState", ptr null }
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/typerecur.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/typerecur.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/typerecur.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func countState(c *counter) stateFn {
	c.value++
	println("count:", c.value)

	if c.value >= c.max {
		return nil
	}
	return countState
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/typerecur.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store i64 5, ptr %2, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/typerecur.stateFn" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/typerecur.countState", ptr null }, ptr %4, align 8
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %8 = load %"{{.*}}/cl/_testgo/typerecur.stateFn", ptr %7, align 8
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/cl/_testgo/typerecur.stateFn" %8, 1
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/cl/_testgo/typerecur.stateFn" %8, 0
// CHECK-NEXT:   %11 = call %"{{.*}}/cl/_testgo/typerecur.stateFn" %10(ptr %9, ptr %0)
// CHECK-NEXT:   %12 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/typerecur.stateFn" %11, ptr %14, align 8
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %15 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %15)
// CHECK-NEXT:   %16 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %16)
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/typerecur.counter", ptr %0, i32 0, i32 2
// CHECK-NEXT:   %18 = load %"{{.*}}/cl/_testgo/typerecur.stateFn", ptr %17, align 8
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/cl/_testgo/typerecur.stateFn" %18, 0
// CHECK-NEXT:   %20 = icmp ne ptr %19, null
// CHECK-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

func main() {
	c := &counter{max: 5, state: countState}

	for c.state != nil {
		c.state = c.state(c)
	}
}

// CHECK-LABEL: define linkonce %"{{.*}}/cl/_testgo/typerecur.stateFn" @"__llgo_stub.{{.*}}/cl/_testgo/typerecur.countState"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call %"{{.*}}/cl/_testgo/typerecur.stateFn" @"{{.*}}/cl/_testgo/typerecur.countState"(ptr %1)
// CHECK-NEXT:   ret %"{{.*}}/cl/_testgo/typerecur.stateFn" %2
// CHECK-NEXT: }
