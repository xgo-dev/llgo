// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"bye", align 1{{$}}

func main() {
	syms := []int{}
	for range syms {
	}
	defer println("bye")
	for range syms {
	}
}

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
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_9), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %10, align 8
// CHECK-NEXT:   %11 = call i32 @{{(__)?}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %12 = icmp eq i32 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_8, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_8
// CHECK-NEXT:   %13 = phi i64 [ -1, %_llgo_8 ], [ %14, %_llgo_2 ]
// CHECK-NEXT:   %14 = add i64 %13, 1
// CHECK-NEXT:   %15 = icmp slt i64 %14, 0
// CHECK-NEXT:   br i1 %15, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %16 = load i64, ptr %7, align 8
// CHECK-NEXT:   %17 = or i64 %16, 1
// CHECK-NEXT:   store i64 %17, ptr %7, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %20 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %19, i32 0, i32 0
// CHECK-NEXT:   store ptr %18, ptr %20, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %19, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %19, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %22, align 8
// CHECK-NEXT:   store ptr %19, ptr %10, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_6, %_llgo_3
// CHECK-NEXT:   %23 = phi i64 [ -1, %_llgo_3 ], [ %24, %_llgo_6 ]
// CHECK-NEXT:   %24 = add i64 %23, 1
// CHECK-NEXT:   %25 = icmp slt i64 %24, 0
// CHECK-NEXT:   br i1 %25, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_12), ptr %9, align 8
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_11, %_llgo_7
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %8, align 8
// CHECK-NEXT:   %26 = load i64, ptr %7, align 8
// CHECK-NEXT:   %27 = and i64 %26, 1
// CHECK-NEXT:   %28 = icmp ne i64 %27, 0
// CHECK-NEXT:   br i1 %28, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_11, %_llgo_14
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.main, %_llgo_10), ptr %9, align 8
// CHECK-NEXT:   %29 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %29, [label %_llgo_10, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_14
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %30 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %31 = icmp ne ptr %30, null
// CHECK-NEXT:   br i1 %31, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_16, %_llgo_9
// CHECK-NEXT:   %32 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %32, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %33)
// CHECK-NEXT:   %34 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %34, [label %_llgo_10, label %_llgo_12]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_13
// CHECK-NEXT:   %35 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %36 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %35, align 8
// CHECK-NEXT:   %37 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %36, 0
// CHECK-NEXT:   store ptr %37, ptr %10, align 8
// CHECK-NEXT:   %38 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %36, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_15, %_llgo_13
// CHECK-NEXT:   br label %_llgo_14
// CHECK-NEXT: }
