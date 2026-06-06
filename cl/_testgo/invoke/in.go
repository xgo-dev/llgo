// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [6 x i8] c"invoke", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [42 x i8] c"{{.*}}/cl/_testgo/invoke.T", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [6 x i8] c"Invoke", align 1
// CHECK-LINE: @3 = private unnamed_addr constant [7 x i8] c"invoke1", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testgo/invoke.T1", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [7 x i8] c"invoke2", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testgo/invoke.T2", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [7 x i8] c"invoke3", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [7 x i8] c"invoke4", align 1
// CHECK-LINE: @9 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testgo/invoke.T4", align 1
// CHECK-LINE: @10 = private unnamed_addr constant [7 x i8] c"invoke5", align 1
// CHECK-LINE: @11 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testgo/invoke.T5", align 1
// CHECK-LINE: @12 = private unnamed_addr constant [7 x i8] c"invoke6", align 1
// CHECK-LINE: @13 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testgo/invoke.T6", align 1
// CHECK-LINE: @14 = private unnamed_addr constant [5 x i8] c"hello", align 1
// CHECK-LINE: @36 = private unnamed_addr constant [5 x i8] c"world", align 1
// CHECK-LINE: @38 = private unnamed_addr constant [42 x i8] c"{{.*}}/cl/_testgo/invoke.I", align 1
// CHECK-LINE: @40 = private unnamed_addr constant [3 x i8] c"any", align 1
// CHECK-LINE: @41 = private unnamed_addr constant [23 x i8] c"interface{Invoke() int}", align 1

type T struct {
	s string
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T.Invoke"(%"{{.*}}/cl/_testgo/invoke.T" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/invoke.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/invoke.T", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 0
// CHECK-NEXT: }

func (t T) Invoke() int {
	println("invoke", t.s)
	return 0
}

func (t *T) Method() {}

type T1 int

func (t T1) Invoke() int {
	println("invoke1", t)
	return 1
}

type T2 float64

func (t T2) Invoke() int {
	println("invoke2", t)
	return 2
}

type T3 int8

func (t *T3) Invoke() int {
	println("invoke3", *t)
	return 3
}

type T4 [1]int

func (t T4) Invoke() int {
	println("invoke4", t[0])
	return 4
}

type T5 struct {
	n int
}

func (t T5) Invoke() int {
	println("invoke5", t.n)
	return 5
}

type T6 func() int

func (t T6) Invoke() int {
	println("invoke6", t())
	return 6
}

type I interface {
	Invoke() int
}

func invoke(i I) {
	println(i.Invoke())
}

func main() {
	var t = T{"hello"}
	var t1 = T1(100)
	var t2 = T2(100.1)
	var t3 = T3(127)
	var t4 = T4{200}
	var t5 = T5{300}
	var t6 = T6(func() int { return 400 })

	invoke(t)

	invoke(&t)

	invoke(t1)

	invoke(&t1)

	invoke(t2)

	invoke(&t2)

	invoke(&t3)

	invoke(t4)

	invoke(&t4)

	invoke(t5)

	invoke(&t5)

	invoke(t6)

	invoke(&t6)

	var m M
	var i I = m

	println(i, m)

	m = &t

	invoke(m)

	var a any = T{"world"}

	invoke(a.(I))

	invoke(a.(interface{}).(interface{ Invoke() int }))

	//panic
	//invoke(nil)
}

type M interface {
	Invoke() int
	Method()
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 42 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/invoke.T", ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T.Invoke"(%"{{.*}}/cl/_testgo/invoke.T" %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/invoke.(*T).Method"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T1.Invoke"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 1
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T1).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T1.Invoke"(i64 %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T2.Invoke"(double %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 2
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T2).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load double, ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T2.Invoke"(double %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T3).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = load i8, ptr %0, align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %3 = sext i8 %2 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T4.Invoke"([1 x i64] %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca [1 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store [1 x i64] %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %1, i64 0
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 4
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T4).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @9, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load [1 x i64], ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T4.Invoke"([1 x i64] %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T5.Invoke"(%"{{.*}}/cl/_testgo/invoke.T5" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testgo/invoke.T5", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T5" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %2)
// CHECK-NEXT:   %3 = icmp eq ptr %1, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %3)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/invoke.T5", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %5 = load i64, ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @10, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 5
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T5).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @11, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/invoke.T5", ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T5.Invoke"(%"{{.*}}/cl/_testgo/invoke.T5" %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.T6.Invoke"(%"{{.*}}/cl/_testgo/invoke.T6" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/cl/_testgo/invoke.T6" %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testgo/invoke.T6" %0, 0
// CHECK-NEXT:   %3 = call i64 %2(ptr %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @12, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret i64 6
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.(*T6).Invoke"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @13, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testgo/invoke.T6", ptr %0, align 8
// CHECK-NEXT:   %3 = call i64 @"{{.*}}/cl/_testgo/invoke.T6.Invoke"(%"{{.*}}/cl/_testgo/invoke.T6" %2)
// CHECK-NEXT:   ret i64 %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/invoke.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/invoke.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/invoke.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %0)
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %0, 0
// CHECK-NEXT:   %3 = getelementptr ptr, ptr %2, i64 3
// CHECK-NEXT:   %4 = load ptr, ptr %3, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %6, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %6, 0
// CHECK-NEXT:   %9 = call i64 %8(ptr %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/invoke.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/invoke.T", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @14, i64 5 }, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store double 1.001000e+02, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   store i8 127, ptr %5, align 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds i64, ptr %6, i64 0
// CHECK-NEXT:   store i64 200, ptr %7, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %9 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/invoke.T5", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store i64 300, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T6" { ptr @"__llgo_stub.{{.*}}/cl/_testgo/invoke.main$1", ptr null }, ptr %11, align 8
// CHECK-NEXT:   %12 = load %"{{.*}}/cl/_testgo/invoke.T", ptr %0, align 8
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T" %12, ptr %13, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T")
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %14, 0
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %15, ptr %13, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %16)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T")
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %17, 0
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %18, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %19)
// CHECK-NEXT:   %20 = load i64, ptr %3, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %20, ptr %21, align 8
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T1")
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %22, 0
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %23, ptr %21, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %24)
// CHECK-NEXT:   %25 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T1")
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %25, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %26, ptr %3, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %27)
// CHECK-NEXT:   %28 = load double, ptr %4, align 8
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %28, ptr %29, align 8
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T2")
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %30, 0
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %31, ptr %29, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %32)
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T2")
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %33, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %34, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %35)
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T3")
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %36, 0
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %37, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %38)
// CHECK-NEXT:   %39 = load [1 x i64], ptr %6, align 8
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store [1 x i64] %39, ptr %40, align 8
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T4")
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %41, 0
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %42, ptr %40, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %43)
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T4")
// CHECK-NEXT:   %45 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %44, 0
// CHECK-NEXT:   %46 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %45, ptr %6, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %46)
// CHECK-NEXT:   %47 = load %"{{.*}}/cl/_testgo/invoke.T5", ptr %8, align 8
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T5" %47, ptr %48, align 8
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T5")
// CHECK-NEXT:   %50 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %49, 0
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %50, ptr %48, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %51)
// CHECK-NEXT:   %52 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T5")
// CHECK-NEXT:   %53 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %52, 0
// CHECK-NEXT:   %54 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %53, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %54)
// CHECK-NEXT:   %55 = load %"{{.*}}/cl/_testgo/invoke.T6", ptr %11, align 8
// CHECK-NEXT:   %56 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T6" %55, ptr %56, align 8
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T6")
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %57, 0
// CHECK-NEXT:   %59 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %58, ptr %56, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %59)
// CHECK-NEXT:   %60 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T6")
// CHECK-NEXT:   %61 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %60, 0
// CHECK-NEXT:   %62 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %61, ptr %11, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %62)
// CHECK-NEXT:   %63 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   %64 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr %63)
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %64, 0
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %65, ptr null, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %66)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" zeroinitializer)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %67 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$jwmSdgh1zvY_TDIgLzCkvkbiyrdwl9N806DH0JGcyMI", ptr @"*_llgo_{{.*}}/cl/_testgo/invoke.T")
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %67, 0
// CHECK-NEXT:   %69 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %68, ptr %0, 1
// CHECK-NEXT:   %70 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %69)
// CHECK-NEXT:   %71 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %69, 1
// CHECK-NEXT:   %72 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr %70)
// CHECK-NEXT:   %73 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %72, 0
// CHECK-NEXT:   %74 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %73, ptr %71, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %74)
// CHECK-NEXT:   %75 = alloca %"{{.*}}/cl/_testgo/invoke.T", align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %75, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %76 = icmp eq ptr %75, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %76)
// CHECK-NEXT:   %77 = getelementptr inbounds %"{{.*}}/cl/_testgo/invoke.T", ptr %75, i32 0, i32 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @36, i64 5 }, ptr %77, align 8
// CHECK-NEXT:   %78 = load %"{{.*}}/cl/_testgo/invoke.T", ptr %75, align 8
// CHECK-NEXT:   %79 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/cl/_testgo/invoke.T" %78, ptr %79, align 8
// CHECK-NEXT:   %80 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testgo/invoke.T", ptr undef }, ptr %79, 1
// CHECK-NEXT:   %81 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %80, 0
// CHECK-NEXT:   %82 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_{{.*}}/cl/_testgo/invoke.I", ptr %81)
// CHECK-NEXT:   br i1 %82, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %83 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %80, 1
// CHECK-NEXT:   %84 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr %81)
// CHECK-NEXT:   %85 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %84, 0
// CHECK-NEXT:   %86 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %85, ptr %83, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %86)
// CHECK-NEXT:   %87 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %80, 0
// CHECK-NEXT:   %88 = icmp ne ptr %87, null
// CHECK-NEXT:   br i1 %88, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %81, %"{{.*}}/runtime/internal/runtime.String" { ptr @38, i64 42 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %89 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %80, 0
// CHECK-NEXT:   %90 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr %89)
// CHECK-NEXT:   br i1 %90, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %87, %"{{.*}}/runtime/internal/runtime.String" { ptr @40, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %91 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %80, 1
// CHECK-NEXT:   %92 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uRUteI7wmSy7y7ODhGzk0FdDaxGKMhVSSu6HZEv9aa0", ptr %89)
// CHECK-NEXT:   %93 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %92, 0
// CHECK-NEXT:   %94 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %93, ptr %91, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/invoke.invoke"(%"{{.*}}/runtime/internal/runtime.iface" %94)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %89, %"{{.*}}/runtime/internal/runtime.String" { ptr @41, i64 23 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 6 })
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testgo/invoke.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 400
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.main$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = tail call i64 @"{{.*}}/cl/_testgo/invoke.main$1"()
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T).Method"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/invoke.(*T).Method"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T.Invoke"(ptr %0, %"{{.*}}/cl/_testgo/invoke.T" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T.Invoke"(%"{{.*}}/cl/_testgo/invoke.T" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T1).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T1).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T1.Invoke"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T1.Invoke"(i64 %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.f64equal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.f64equal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T2).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T2).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T2.Invoke"(ptr %0, double %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T2.Invoke"(double %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T3).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T3).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T4).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T4).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T4.Invoke"(ptr %0, [1 x i64] %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T4.Invoke"([1 x i64] %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T5).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T5).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T5.Invoke"(ptr %0, %"{{.*}}/cl/_testgo/invoke.T5" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T5.Invoke"(%"{{.*}}/cl/_testgo/invoke.T5" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.(*T6).Invoke"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.(*T6).Invoke"(ptr %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"__llgo_stub.{{.*}}/cl/_testgo/invoke.T6.Invoke"(ptr %0, %"{{.*}}/cl/_testgo/invoke.T6" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = tail call i64 @"{{.*}}/cl/_testgo/invoke.T6.Invoke"(%"{{.*}}/cl/_testgo/invoke.T6" %1)
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
