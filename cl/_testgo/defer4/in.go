// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"bye", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"panic message", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"string", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"recover:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [9 x i8] c"reachable", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"world", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"hi", align 1{{$}}

func f(s string) bool {
	return len(s) > 2
}

func fail() {
	defer println("bye")
	defer func() {
		if e := recover(); e != nil {
			println("recover:", e.(string))
		}
	}()
	panic("panic message")
}

func main() {
	defer func() {
		println("hi")
	}()
	if s := "hello"; f(s) {
		defer println(s)
	} else {
		defer println("world")
		return
	}
	fail()
	println("reachable")
}

// CHECK-LABEL: define i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %0, 1
// CHECK-NEXT:   %2 = icmp sgt i64 %1, 2
// CHECK-NEXT:   ret i1 %2
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.fail(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.fail, %_llgo_2), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %10, align 8
// CHECK-NEXT:   %11 = call i32 @{{(__)?}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %12 = icmp eq i32 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store ptr blockaddress(@main.fail, %_llgo_6), ptr %8, align 8
// CHECK-NEXT:   %13 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @"main.fail$1"()
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %14 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %16 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   store ptr %14, ptr %16, align 8
// CHECK-NEXT:   %17 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %17, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %15, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %18, align 8
// CHECK-NEXT:   store ptr %15, ptr %10, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 }, ptr %19, align 8
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %19, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %20)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.fail, %_llgo_3), ptr %9, align 8
// CHECK-NEXT:   %21 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %21, [label %_llgo_3, label %_llgo_6, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@main.fail, %_llgo_3), ptr %8, align 8
// CHECK-NEXT:   %22 = load i64, ptr %7, align 8
// CHECK-NEXT:   %23 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %24 = icmp ne ptr %23, null
// CHECK-NEXT:   br i1 %24, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %25 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %26 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %25, align 8
// CHECK-NEXT:   %27 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %26, 0
// CHECK-NEXT:   store ptr %27, ptr %10, align 8
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %26, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %28)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %30)
// CHECK-NEXT:   %31 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_3]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.fail$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %2 = xor i1 %1, true
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %4 = icmp eq ptr %3, @_llgo_string
// CHECK-NEXT:   br i1 %4, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.String", ptr %5, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %3, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
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
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_4), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %10, align 8
// CHECK-NEXT:   %11 = call i32 @{{(__)?}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %12 = icmp eq i32 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %13 = load i64, ptr %7, align 8
// CHECK-NEXT:   %14 = or i64 %13, 1
// CHECK-NEXT:   store i64 %14, ptr %7, align 8
// CHECK-NEXT:   %15 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %17 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %16, i32 0, i32 0
// CHECK-NEXT:   store ptr %15, ptr %17, align 8
// CHECK-NEXT:   %18 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %16, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %18, align 8
// CHECK-NEXT:   %19 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %16, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %19, align 8
// CHECK-NEXT:   store ptr %16, ptr %10, align 8
// CHECK-NEXT:   call void @main.fail()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 9 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_8), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %20 = load i64, ptr %7, align 8
// CHECK-NEXT:   %21 = or i64 %20, 2
// CHECK-NEXT:   store i64 %21, ptr %7, align 8
// CHECK-NEXT:   %22 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %10, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_9), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_7, %_llgo_3, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_11), ptr %8, align 8
// CHECK-NEXT:   %27 = load i64, ptr %7, align 8
// CHECK-NEXT:   %28 = and i64 %27, 2
// CHECK-NEXT:   %29 = icmp ne i64 %28, 0
// CHECK-NEXT:   br i1 %29, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %30 = call i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT:   br i1 %30, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_5), ptr %9, align 8
// CHECK-NEXT:   %31 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_5, label %_llgo_10, label %_llgo_11, label %_llgo_4]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_7, %_llgo_17
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_5), ptr %8, align 8
// CHECK-NEXT:   %32 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @"main.main$1"()
// CHECK-NEXT:   %33 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %34 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %33, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %34)
// CHECK-NEXT:   %35 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %35, [label %_llgo_5, label %_llgo_8, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_7, %_llgo_13
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %8, align 8
// CHECK-NEXT:   %36 = load i64, ptr %7, align 8
// CHECK-NEXT:   %37 = and i64 %36, 1
// CHECK-NEXT:   %38 = icmp ne i64 %37, 0
// CHECK-NEXT:   br i1 %38, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %39 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %40 = icmp ne ptr %39, null
// CHECK-NEXT:   br i1 %40, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_15, %_llgo_4
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %41 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %42 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %41, align 8
// CHECK-NEXT:   %43 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %42, 0
// CHECK-NEXT:   store ptr %43, ptr %10, align 8
// CHECK-NEXT:   %44 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %42, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %44)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14, %_llgo_12
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %45 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %46 = icmp ne ptr %45, null
// CHECK-NEXT:   br i1 %46, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_19, %_llgo_11
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %47 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %48 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %47, align 8
// CHECK-NEXT:   %49 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %48, 0
// CHECK-NEXT:   store ptr %49, ptr %10, align 8
// CHECK-NEXT:   %50 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %48, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %50)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_18, %_llgo_16
// CHECK-NEXT:   br label %_llgo_17
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
