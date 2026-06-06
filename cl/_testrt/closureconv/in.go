// LITTEST
package main

type Func func(a int, b int) int
type Func2 func(a int, b int) int

type Call struct {
	fn Func
	n  int
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureconv.(*Call).add"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = add i64 %1, %2
// CHECK-NEXT:   %4 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %5)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testrt/closureconv.Call", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %7 = load i64, ptr %6, align 8
// CHECK-NEXT:   %8 = add i64 %3, %7
// CHECK-NEXT:   ret i64 %8
// CHECK-NEXT: }

func (c *Call) add(a int, b int) int {
	return a + b + c.n
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureconv.add"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

func add(a int, b int) int {
	return a + b
}

// CHECK-LABEL: define %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo1"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testrt/closureconv.Call", ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 %0, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %5 = getelementptr inbounds { ptr }, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closureconv.(*Call).add$bound", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %7 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %8)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/cl/_testrt/closureconv.Call", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %10 = alloca %"{{.*}}/cl/_testrt/closureconv.Func", align 8
// CHECK-NEXT:   store { ptr, ptr } %6, ptr %10, align 8
// CHECK-NEXT:   %11 = load %"{{.*}}/cl/_testrt/closureconv.Func", ptr %10, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/closureconv.Func" %11, ptr %9, align 8
// CHECK-NEXT:   %12 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %12)
// CHECK-NEXT:   %13 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/cl/_testrt/closureconv.Call", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %15 = load %"{{.*}}/cl/_testrt/closureconv.Func", ptr %14, align 8
// CHECK-NEXT:   ret %"{{.*}}/cl/_testrt/closureconv.Func" %15
// CHECK-NEXT: }

func demo1(n int) Func {
	m := &Call{n: n}
	m.fn = m.add
	return m.fn
}

// CHECK-LABEL: define %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closureconv.(*Call).add$bound", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = alloca %"{{.*}}/cl/_testrt/closureconv.Func", align 8
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testrt/closureconv.Func", ptr %4, align 8
// CHECK-NEXT:   ret %"{{.*}}/cl/_testrt/closureconv.Func" %5
// CHECK-NEXT: }

func demo2() Func {
	m := &Call{}
	return m.add
}

// CHECK-LABEL: define %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo3"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/cl/_testrt/closureconv.Func" { ptr @"__llgo_stub.{{.*}}/cl/_testrt/closureconv.add", ptr null }
// CHECK-NEXT: }

func demo3() Func {
	return add
}

// CHECK-LABEL: define %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo4"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %"{{.*}}/cl/_testrt/closureconv.Func" { ptr @"__llgo_stub.{{.*}}/cl/_testrt/closureconv.demo4$1", ptr null }
// CHECK-NEXT: }

func demo4() Func {

	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureconv.demo4$1"(i64 %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = add i64 %0, %1
	// CHECK-NEXT:   ret i64 %2
	// CHECK-NEXT: }

	return func(a, b int) int { return a + b }
}

// CHECK-LABEL: define %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo5"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 %0, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closureconv.demo5$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = alloca %"{{.*}}/cl/_testrt/closureconv.Func", align 8
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %5, align 8
// CHECK-NEXT:   %6 = load %"{{.*}}/cl/_testrt/closureconv.Func", ptr %5, align 8
// CHECK-NEXT:   ret %"{{.*}}/cl/_testrt/closureconv.Func" %6
// CHECK-NEXT: }

func demo5(n int) Func {

	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureconv.demo5$1"(ptr %0, i64 %1, i64 %2){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %3 = add i64 %1, %2
	// CHECK-NEXT:   %4 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %5 = extractvalue { ptr } %4, 0
	// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
	// CHECK-NEXT:   %7 = add i64 %3, %6
	// CHECK-NEXT:   ret i64 %7
	// CHECK-NEXT: }

	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/closureconv.init"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/closureconv.init$guard", align 1
	// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
	// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/closureconv.init$guard", align 1
	// CHECK-NEXT:   br label %_llgo_2
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	return func(a, b int) int { return a + b + n }
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/closureconv.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo1"(i64 1)
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %0, 0
// CHECK-NEXT:   %3 = call i64 %2(ptr %1, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo2"()
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %4, 1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %4, 0
// CHECK-NEXT:   %7 = call i64 %6(ptr %5, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo3"()
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %8, 1
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %8, 0
// CHECK-NEXT:   %11 = call i64 %10(ptr %9, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo4"()
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %12, 1
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %12, 0
// CHECK-NEXT:   %15 = call i64 %14(ptr %13, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo5"(i64 1)
// CHECK-NEXT:   %17 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %16, 1
// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %16, 0
// CHECK-NEXT:   %19 = call i64 %18(ptr %17, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %20 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo5"(i64 1)
// CHECK-NEXT:   %21 = alloca { ptr, ptr }, align 8
// CHECK-NEXT:   store %"{{.*}}/cl/_testrt/closureconv.Func" %20, ptr %21, align 8
// CHECK-NEXT:   %22 = load { ptr, ptr }, ptr %21, align 8
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr } %22, 1
// CHECK-NEXT:   %24 = extractvalue { ptr, ptr } %22, 0
// CHECK-NEXT:   %25 = call i64 %24(ptr %23, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %26 = call %"{{.*}}/cl/_testrt/closureconv.Func" @"{{.*}}/cl/_testrt/closureconv.demo5"(i64 1)
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %26, 0
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/cl/_testrt/closureconv.Func2" undef, ptr %27, 0
// CHECK-NEXT:   %29 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func" %26, 1
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/cl/_testrt/closureconv.Func2" %28, ptr %29, 1
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func2" %30, 1
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/cl/_testrt/closureconv.Func2" %30, 0
// CHECK-NEXT:   %33 = call i64 %32(ptr %31, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %33)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureconv.(*Call).add$bound"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %4 = extractvalue { ptr } %3, 0
// CHECK-NEXT:   %5 = call i64 @"{{.*}}/cl/_testrt/closureconv.(*Call).add"(ptr %4, i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testrt/closureconv.add"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testrt/closureconv.add"(i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testrt/closureconv.demo4$1"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i64 @"{{.*}}/cl/_testrt/closureconv.demo4$1"(i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }
