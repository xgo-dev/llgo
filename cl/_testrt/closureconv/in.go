// LITTEST
package main

type Func func(a int, b int) int
type Func2 func(a int, b int) int

type Call struct {
	fn Func
	n  int
}

// CHECK-LABEL: define i64 @"main.(*Call).add"(ptr %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = add i64 %1, %2
// CHECK-NEXT:   %4 = getelementptr inbounds %main.Call, ptr %0, i32 0, i32 1
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   %6 = add i64 %3, %5
// CHECK-NEXT:   ret i64 %6
// CHECK-NEXT: }
func (c *Call) add(a int, b int) int {
	return a + b + c.n
}

// CHECK-LABEL: define i64 @main.add(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func add(a int, b int) int {
	return a + b
}

// CHECK-LABEL: define %main.Func @main.demo1(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Call, ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } { ptr @"main.(*Call).add$bound", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %6 = getelementptr inbounds %main.Call, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %7 = alloca %main.Func, align 8
// CHECK-NEXT:   store { ptr, ptr } %5, ptr %7, align 8
// CHECK-NEXT:   %8 = load %main.Func, ptr %7, align 8
// CHECK-NEXT:   store %main.Func %8, ptr %6, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %main.Call, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %10 = load %main.Func, ptr %9, align 8
// CHECK-NEXT:   ret %main.Func %10
// CHECK-NEXT: }
func demo1(n int) Func {
	m := &Call{n: n}
	m.fn = m.add
	return m.fn
}

// CHECK-LABEL: define %main.Func @main.demo2(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"main.(*Call).add$bound", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = alloca %main.Func, align 8
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = load %main.Func, ptr %4, align 8
// CHECK-NEXT:   ret %main.Func %5
// CHECK-NEXT: }

func demo2() Func {
	m := &Call{}
	return m.add
}

// CHECK-LABEL: define %main.Func @main.demo3(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %main.Func { ptr @main.add, ptr null }
// CHECK-NEXT: }

func demo3() Func {
	return add
}

// CHECK-LABEL: define %main.Func @main.demo4(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret %main.Func { ptr @"main.demo4$1", ptr null }
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.demo4$1"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func demo4() Func {
	return func(a, b int) int { return a + b }
}

// CHECK-LABEL: define %main.Func @main.demo5(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 %0, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.demo5$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = alloca %main.Func, align 8
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %5, align 8
// CHECK-NEXT:   %6 = load %main.Func, ptr %5, align 8
// CHECK-NEXT:   ret %main.Func %6
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.demo5$1"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = add i64 %1, %2
// CHECK-NEXT:   %4 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %5 = extractvalue { ptr } %4, 0
// CHECK-NEXT:   %6 = load i64, ptr %5, align 8
// CHECK-NEXT:   %7 = add i64 %3, %6
// CHECK-NEXT:   ret i64 %7
// CHECK-NEXT: }
func demo5(n int) Func {
	return func(a, b int) int { return a + b + n }
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %main.Func @main.demo1(i64 1)
// CHECK-NEXT:   %1 = extractvalue %main.Func %0, 1
// CHECK-NEXT:   %2 = extractvalue %main.Func %0, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %2)
// CHECK-NEXT:   %3 = call i64 %__llgo_funcval_code(ptr {{(nest|swiftself)}} %1, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %4 = call %main.Func @main.demo2()
// CHECK-NEXT:   %5 = extractvalue %main.Func %4, 1
// CHECK-NEXT:   %6 = extractvalue %main.Func %4, 0
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %6)
// CHECK-NEXT:   %7 = call i64 %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %5, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = call %main.Func @main.demo3()
// CHECK-NEXT:   %9 = extractvalue %main.Func %8, 1
// CHECK-NEXT:   %10 = extractvalue %main.Func %8, 0
// CHECK-NEXT:   %__llgo_funcval_code2 = call ptr asm "", "=r,0"(ptr %10)
// CHECK-NEXT:   %11 = call i64 %__llgo_funcval_code2(ptr {{(nest|swiftself)}} %9, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = call %main.Func @main.demo4()
// CHECK-NEXT:   %13 = extractvalue %main.Func %12, 1
// CHECK-NEXT:   %14 = extractvalue %main.Func %12, 0
// CHECK-NEXT:   %__llgo_funcval_code3 = call ptr asm "", "=r,0"(ptr %14)
// CHECK-NEXT:   %15 = call i64 %__llgo_funcval_code3(ptr {{(nest|swiftself)}} %13, i64 100, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = call %main.Func @main.demo5(i64 1)
// CHECK-NEXT:   %17 = extractvalue %main.Func %16, 1
// CHECK-NEXT:   %18 = extractvalue %main.Func %16, 0
// CHECK-NEXT:   %__llgo_funcval_code4 = call ptr asm "", "=r,0"(ptr %18)
// CHECK-NEXT:   %19 = call i64 %__llgo_funcval_code4(ptr {{(nest|swiftself)}} %17, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %20 = call %main.Func @main.demo5(i64 1)
// CHECK-NEXT:   %21 = alloca { ptr, ptr }, align 8
// CHECK-NEXT:   store %main.Func %20, ptr %21, align 8
// CHECK-NEXT:   %22 = load { ptr, ptr }, ptr %21, align 8
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr } %22, 1
// CHECK-NEXT:   %24 = extractvalue { ptr, ptr } %22, 0
// CHECK-NEXT:   %__llgo_funcval_code5 = call ptr asm "", "=r,0"(ptr %24)
// CHECK-NEXT:   %25 = call i64 %__llgo_funcval_code5(ptr {{(nest|swiftself)}} %23, i64 99, i64 200)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %26 = call %main.Func @main.demo5(i64 1)
// CHECK-NEXT:   %27 = extractvalue %main.Func %26, 0
// CHECK-NEXT:   %28 = insertvalue %main.Func2 undef, ptr %27, 0
// CHECK-NEXT:   %29 = extractvalue %main.Func %26, 1
// CHECK-NEXT:   %30 = insertvalue %main.Func2 %28, ptr %29, 1
// CHECK-NEXT:   %31 = extractvalue %main.Func2 %30, 1
// CHECK-NEXT:   %32 = extractvalue %main.Func2 %30, 0
// CHECK-NEXT:   %__llgo_funcval_code6 = call ptr asm "", "=r,0"(ptr %32)
// CHECK-NEXT:   %33 = call i64 %__llgo_funcval_code6(ptr {{(nest|swiftself)}} %31, i64 99, i64 200)
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

// CHECK-LABEL: define i64 @"main.(*Call).add$bound"(ptr {{(nest|swiftself)}} %0, i64 %1, i64 %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %4 = extractvalue { ptr } %3, 0
// CHECK-NEXT:   %5 = call i64 @"main.(*Call).add"(ptr %4, i64 %1, i64 %2)
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }
