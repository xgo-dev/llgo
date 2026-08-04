// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"bye", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"world", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"hi", align 1{{$}}

func f(s string) bool {
	return len(s) > 2
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
	defer println("bye")
}

// CHECK-LABEL: define i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %0, 1
// CHECK-NEXT:   %2 = icmp sgt i64 %1, 2
// CHECK-NEXT:   ret i1 %2
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
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %10, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_8), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %27 = load i64, ptr %7, align 8
// CHECK-NEXT:   %28 = or i64 %27, 4
// CHECK-NEXT:   store i64 %28, ptr %7, align 8
// CHECK-NEXT:   %29 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %31 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 0
// CHECK-NEXT:   store ptr %29, ptr %31, align 8
// CHECK-NEXT:   %32 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 1
// CHECK-NEXT:   store i64 3, ptr %32, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %30, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %33, align 8
// CHECK-NEXT:   store ptr %30, ptr %10, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_9), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_7, %_llgo_3, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_12), ptr %8, align 8
// CHECK-NEXT:   %34 = load i64, ptr %7, align 8
// CHECK-NEXT:   %35 = and i64 %34, 4
// CHECK-NEXT:   %36 = icmp ne i64 %35, 0
// CHECK-NEXT:   br i1 %36, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %37 = call i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT:   br i1 %37, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_5), ptr %9, align 8
// CHECK-NEXT:   %38 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %38, [label %_llgo_5, label %_llgo_10, label %_llgo_11, label %_llgo_12, label %_llgo_4]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_7, %_llgo_22
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_5), ptr %8, align 8
// CHECK-NEXT:   %39 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @"main.main$1"()
// CHECK-NEXT:   %40 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %40, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %41)
// CHECK-NEXT:   %42 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %42, [label %_llgo_5, label %_llgo_8, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_7, %_llgo_18
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %8, align 8
// CHECK-NEXT:   %43 = load i64, ptr %7, align 8
// CHECK-NEXT:   %44 = and i64 %43, 1
// CHECK-NEXT:   %45 = icmp ne i64 %44, 0
// CHECK-NEXT:   br i1 %45, label %_llgo_21, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_7, %_llgo_14
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_11), ptr %8, align 8
// CHECK-NEXT:   %46 = load i64, ptr %7, align 8
// CHECK-NEXT:   %47 = and i64 %46, 2
// CHECK-NEXT:   %48 = icmp ne i64 %47, 0
// CHECK-NEXT:   br i1 %48, label %_llgo_17, label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_4
// CHECK-NEXT:   %49 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %50 = icmp ne ptr %49, null
// CHECK-NEXT:   br i1 %50, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_16, %_llgo_4
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_13
// CHECK-NEXT:   %51 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %52 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %51, align 8
// CHECK-NEXT:   %53 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %52, 0
// CHECK-NEXT:   store ptr %53, ptr %10, align 8
// CHECK-NEXT:   %54 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %52, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %51)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %54)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_15, %_llgo_13
// CHECK-NEXT:   br label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %55 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %56 = icmp ne ptr %55, null
// CHECK-NEXT:   br i1 %56, label %_llgo_19, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_20, %_llgo_12
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_17
// CHECK-NEXT:   %57 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %58 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %57, align 8
// CHECK-NEXT:   %59 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %58, 0
// CHECK-NEXT:   store ptr %59, ptr %10, align 8
// CHECK-NEXT:   %60 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %58, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %57)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %60)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_19, %_llgo_17
// CHECK-NEXT:   br label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %61 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %62 = icmp ne ptr %61, null
// CHECK-NEXT:   br i1 %62, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_24, %_llgo_11
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %63 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %64 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %63, align 8
// CHECK-NEXT:   %65 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %64, 0
// CHECK-NEXT:   store ptr %65, ptr %10, align 8
// CHECK-NEXT:   %66 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %64, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %63)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %66)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_23, %_llgo_21
// CHECK-NEXT:   br label %_llgo_22
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
