// LITTEST
package main

// Test deferred closure and method-value lowering.

// Type for holding a function

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [16 x i8] c"deferred closure", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"closure value", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [17 x i8] c"field access test", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [19 x i8] c"callback from field", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"before return", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"deferred", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"captured", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [19 x i8] c"struct closure test", align 1{{$}}

type Handler struct {
	fn func(int)
}

func (h *Handler) SetHandler(f func(int)) {
	h.fn = f
}

// Test case 1: Deferred method call with function literal (issue #1488)
// This triggers the temporary register name check (name[0] == '%')
func testDeferMethodLiteral() {
	var h Handler
	h.SetHandler(func(int) {})
	defer h.SetHandler(func(x int) {
		println("deferred", x)
	})
	println("before return")
}

// Test case 2: Defer a closure value directly
// This triggers the v.kind != vkFuncDecl && v.kind != vkFuncPtr branch
func testDeferClosureValue() {
	x := 42
	fn := func() {
		println("closure value", x)
	}
	defer fn()
	println("deferred closure")
}

// Test case 3: Complex scenario with closure in struct
type Processor struct {
	callback func(string)
}

func (p *Processor) SetCallback(cb func(string)) {
	p.callback = cb
}

func testDeferStructClosure() {
	var p Processor
	msg := "captured"
	// Defer a method call that takes a closure capturing a variable
	defer p.SetCallback(func(s string) {
		println(s, msg)
	})
	println("struct closure test")
}

// Test case 4: Defer a function accessed through a struct field
// This should trigger the v.kind != vkFuncDecl && v.kind != vkFuncPtr branch
// because accessing p.callback returns a value that's not a function declaration
type FuncHolder struct {
	callback func()
}

func testDeferFieldAccess() {
	var holder FuncHolder
	holder.callback = func() {
		println("callback from field")
	}
	// When we defer holder.callback directly, it's accessed as a field load
	// which might have a different value kind than vkFuncDecl/vkFuncPtr
	defer holder.callback()
	println("field access test")
}

func main() {
	testDeferMethodLiteral()
	testDeferClosureValue()
	testDeferStructClosure()
	testDeferFieldAccess()
}

// CHECK-LABEL: define void @"main.(*Handler).SetHandler"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Handler, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.(*Processor).SetCallback"(ptr %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Processor, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %1, ptr %2, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @main.testDeferMethodLiteral()
// CHECK-NEXT:   call void @main.testDeferClosureValue()
// CHECK-NEXT:   call void @main.testDeferStructClosure()
// CHECK-NEXT:   call void @main.testDeferFieldAccess()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.testDeferClosureValue(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"main.testDeferClosureValue$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %5 = alloca i8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 2
// CHECK-NEXT:   store ptr %4, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferClosureValue, %_llgo_2), ptr %10, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %6)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 4
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %14, align 8
// CHECK-NEXT:   %15 = call i32 @{{(__)?}}sigsetjmp(ptr %5, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferClosureValue, %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %17 = load i64, ptr %11, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %19 = icmp ne ptr %18, null
// CHECK-NEXT:   br i1 %19, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %4)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 0
// CHECK-NEXT:   store ptr %20, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %24, align 8
// CHECK-NEXT:   store ptr %21, ptr %14, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 16 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferClosureValue, %_llgo_6), ptr %13, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferClosureValue, %_llgo_3), ptr %13, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %25, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %26 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %27 = load { ptr, i64, { ptr, ptr } }, ptr %26, align 8
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, { ptr, ptr } } %27, 0
// CHECK-NEXT:   store ptr %28, ptr %14, align 8
// CHECK-NEXT:   %29 = extractvalue { ptr, i64, { ptr, ptr } } %27, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %26)
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %29, 1
// CHECK-NEXT:   %31 = extractvalue { ptr, ptr } %29, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %31)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %30)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %32 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, align 8
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %32, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %33)
// CHECK-NEXT:   %34 = load ptr, ptr %13, align 8
// CHECK-NEXT:   indirectbr ptr %34, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.testDeferClosureValue$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.testDeferFieldAccess(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %main.FuncHolder, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.FuncHolder, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } { ptr @"main.testDeferFieldAccess$1", ptr null }, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.FuncHolder, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %3 = load { ptr, ptr }, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %5 = alloca i8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 2
// CHECK-NEXT:   store ptr %4, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferFieldAccess, %_llgo_2), ptr %10, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %6)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 3
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 4
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %14, align 8
// CHECK-NEXT:   %15 = call i32 @{{(__)?}}sigsetjmp(ptr %5, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferFieldAccess, %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %17 = load i64, ptr %11, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %19 = icmp ne ptr %18, null
// CHECK-NEXT:   br i1 %19, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %4)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 0
// CHECK-NEXT:   store ptr %20, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %24, align 8
// CHECK-NEXT:   store ptr %21, ptr %14, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferFieldAccess, %_llgo_6), ptr %13, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferFieldAccess, %_llgo_3), ptr %13, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %25, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %26 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %27 = load { ptr, i64, { ptr, ptr } }, ptr %26, align 8
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, { ptr, ptr } } %27, 0
// CHECK-NEXT:   store ptr %28, ptr %14, align 8
// CHECK-NEXT:   %29 = extractvalue { ptr, i64, { ptr, ptr } } %27, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %26)
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %29, 1
// CHECK-NEXT:   %31 = extractvalue { ptr, ptr } %29, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %31)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %30)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %32 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %6, align 8
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %32, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %33)
// CHECK-NEXT:   %34 = load ptr, ptr %13, align 8
// CHECK-NEXT:   indirectbr ptr %34, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.testDeferFieldAccess$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 19 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.testDeferMethodLiteral(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   call void @"main.(*Handler).SetHandler"(ptr %0, { ptr, ptr } { ptr @"main.testDeferMethodLiteral$1", ptr null })
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %2 = alloca i8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 2
// CHECK-NEXT:   store ptr %1, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferMethodLiteral, %_llgo_2), ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %3)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 4
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %11, align 8
// CHECK-NEXT:   %12 = call i32 @{{(__)?}}sigsetjmp(ptr %2, i32 0)
// CHECK-NEXT:   %13 = icmp eq i32 %12, 0
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferMethodLiteral, %_llgo_3), ptr %9, align 8
// CHECK-NEXT:   %14 = load i64, ptr %8, align 8
// CHECK-NEXT:   %15 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %16 = icmp ne ptr %15, null
// CHECK-NEXT:   br i1 %16, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %1)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %19 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %18, i32 0, i32 0
// CHECK-NEXT:   store ptr %17, ptr %19, align 8
// CHECK-NEXT:   %20 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %18, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %20, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %18, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, ptr, ptr }, ptr %18, i32 0, i32 3
// CHECK-NEXT:   store ptr @"main.testDeferMethodLiteral$2", ptr %22, align 8
// CHECK-NEXT:   store ptr %18, ptr %11, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferMethodLiteral, %_llgo_6), ptr %10, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferMethodLiteral, %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %23 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %23, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %24 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %25 = load { ptr, i64, ptr, ptr }, ptr %24, align 8
// CHECK-NEXT:   %26 = extractvalue { ptr, i64, ptr, ptr } %25, 0
// CHECK-NEXT:   store ptr %26, ptr %11, align 8
// CHECK-NEXT:   %27 = extractvalue { ptr, i64, ptr, ptr } %25, 2
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, ptr, ptr } %25, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %24)
// CHECK-NEXT:   %29 = insertvalue { ptr, ptr } undef, ptr %28, 0
// CHECK-NEXT:   %30 = insertvalue { ptr, ptr } %29, ptr null, 1
// CHECK-NEXT:   call void @"main.(*Handler).SetHandler"(ptr %27, { ptr, ptr } %30)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %31 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, align 8
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %31, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %32)
// CHECK-NEXT:   %33 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %33, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.testDeferMethodLiteral$1"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.testDeferMethodLiteral$2"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.testDeferStructClosure(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %1, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.testDeferStructClosure$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %6 = alloca i8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 0
// CHECK-NEXT:   store ptr %6, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 2
// CHECK-NEXT:   store ptr %5, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferStructClosure, %_llgo_2), ptr %11, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %7)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 4
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %15, align 8
// CHECK-NEXT:   %16 = call i32 @{{(__)?}}sigsetjmp(ptr %6, i32 0)
// CHECK-NEXT:   %17 = icmp eq i32 %16, 0
// CHECK-NEXT:   br i1 %17, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferStructClosure, %_llgo_3), ptr %13, align 8
// CHECK-NEXT:   %18 = load i64, ptr %12, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %20 = icmp ne ptr %19, null
// CHECK-NEXT:   br i1 %20, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %5)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %21 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 40)
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, ptr, { ptr, ptr } }, ptr %22, i32 0, i32 0
// CHECK-NEXT:   store ptr %21, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, ptr, { ptr, ptr } }, ptr %22, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, ptr, { ptr, ptr } }, ptr %22, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, ptr, { ptr, ptr } }, ptr %22, i32 0, i32 3
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %26, align 8
// CHECK-NEXT:   store ptr %22, ptr %15, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 19 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferStructClosure, %_llgo_6), ptr %14, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.testDeferStructClosure, %_llgo_3), ptr %14, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %13, align 8
// CHECK-NEXT:   indirectbr ptr %27, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %28 = load ptr, ptr %15, align 8
// CHECK-NEXT:   %29 = load { ptr, i64, ptr, { ptr, ptr } }, ptr %28, align 8
// CHECK-NEXT:   %30 = extractvalue { ptr, i64, ptr, { ptr, ptr } } %29, 0
// CHECK-NEXT:   store ptr %30, ptr %15, align 8
// CHECK-NEXT:   %31 = extractvalue { ptr, i64, ptr, { ptr, ptr } } %29, 2
// CHECK-NEXT:   %32 = extractvalue { ptr, i64, ptr, { ptr, ptr } } %29, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %28)
// CHECK-NEXT:   call void @"main.(*Processor).SetCallback"(ptr %31, { ptr, ptr } %32)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %33 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, align 8
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %33, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %34)
// CHECK-NEXT:   %35 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %35, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.testDeferStructClosure$1"(ptr {{(nest|swiftself)}} %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
