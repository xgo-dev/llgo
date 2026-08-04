// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"bye", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"world", align 1{{$}}

func f(s string) bool {
	return len(s) > 2
}

func main() {
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
// CHECK-NEXT:   %0 = call i1 @main.f(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
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
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_5), ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %3)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 4
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %11, align 8
// CHECK-NEXT:   %12 = call i32 @{{(__)?}}sigsetjmp(ptr %2, i32 0)
// CHECK-NEXT:   %13 = icmp eq i32 %12, 0
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %14 = load i64, ptr %8, align 8
// CHECK-NEXT:   %15 = or i64 %14, 1
// CHECK-NEXT:   store i64 %15, ptr %8, align 8
// CHECK-NEXT:   %16 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %18 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %17, i32 0, i32 0
// CHECK-NEXT:   store ptr %16, ptr %18, align 8
// CHECK-NEXT:   %19 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %17, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %19, align 8
// CHECK-NEXT:   %20 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %17, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %20, align 8
// CHECK-NEXT:   store ptr %17, ptr %11, align 8
// CHECK-NEXT:   %21 = load i64, ptr %8, align 8
// CHECK-NEXT:   %22 = or i64 %21, 2
// CHECK-NEXT:   store i64 %22, ptr %8, align 8
// CHECK-NEXT:   %23 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store ptr %23, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %24, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %24, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %27, align 8
// CHECK-NEXT:   store ptr %24, ptr %11, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_8), ptr %10, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %28 = load i64, ptr %8, align 8
// CHECK-NEXT:   %29 = or i64 %28, 4
// CHECK-NEXT:   store i64 %29, ptr %8, align 8
// CHECK-NEXT:   %30 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %32 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %31, i32 0, i32 0
// CHECK-NEXT:   store ptr %30, ptr %32, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %31, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %33, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %31, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %34, align 8
// CHECK-NEXT:   store ptr %31, ptr %11, align 8
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_9), ptr %10, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 %0, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_2, %_llgo_1
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_11), ptr %9, align 8
// CHECK-NEXT:   %35 = load i64, ptr %8, align 8
// CHECK-NEXT:   %36 = and i64 %35, 4
// CHECK-NEXT:   %37 = icmp ne i64 %36, 0
// CHECK-NEXT:   br i1 %37, label %_llgo_12, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7, %_llgo_21
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %1)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_6), ptr %10, align 8
// CHECK-NEXT:   %38 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %38, [label %_llgo_6, label %_llgo_10, label %_llgo_11, label %_llgo_5]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_21
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_21
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_7, %_llgo_17
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_6), ptr %9, align 8
// CHECK-NEXT:   %39 = load i64, ptr %8, align 8
// CHECK-NEXT:   %40 = and i64 %39, 1
// CHECK-NEXT:   %41 = icmp ne i64 %40, 0
// CHECK-NEXT:   br i1 %41, label %_llgo_20, label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_7, %_llgo_13
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %9, align 8
// CHECK-NEXT:   %42 = load i64, ptr %8, align 8
// CHECK-NEXT:   %43 = and i64 %42, 2
// CHECK-NEXT:   %44 = icmp ne i64 %43, 0
// CHECK-NEXT:   br i1 %44, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_5
// CHECK-NEXT:   %45 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %46 = icmp ne ptr %45, null
// CHECK-NEXT:   br i1 %46, label %_llgo_14, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_15, %_llgo_5
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %47 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %48 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %47, align 8
// CHECK-NEXT:   %49 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %48, 0
// CHECK-NEXT:   store ptr %49, ptr %11, align 8
// CHECK-NEXT:   %50 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %48, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %50)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14, %_llgo_12
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %51 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %52 = icmp ne ptr %51, null
// CHECK-NEXT:   br i1 %52, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_19, %_llgo_11
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %53 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %54 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %53, align 8
// CHECK-NEXT:   %55 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %54, 0
// CHECK-NEXT:   store ptr %55, ptr %11, align 8
// CHECK-NEXT:   %56 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %54, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %53)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %56)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_18, %_llgo_16
// CHECK-NEXT:   br label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %57 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %58 = icmp ne ptr %57, null
// CHECK-NEXT:   br i1 %58, label %_llgo_22, label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_23, %_llgo_10
// CHECK-NEXT:   %59 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, align 8
// CHECK-NEXT:   %60 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %59, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %60)
// CHECK-NEXT:   %61 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %61, [label %_llgo_6, label %_llgo_8, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_20
// CHECK-NEXT:   %62 = load ptr, ptr %11, align 8
// CHECK-NEXT:   %63 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %62, align 8
// CHECK-NEXT:   %64 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %63, 0
// CHECK-NEXT:   store ptr %64, ptr %11, align 8
// CHECK-NEXT:   %65 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %63, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %62)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %65)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_23
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_22, %_llgo_20
// CHECK-NEXT:   br label %_llgo_21
// CHECK-NEXT: }
