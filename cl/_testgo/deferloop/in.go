// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"loop", align 1

func main() {
	for i := 0; i < 3; i++ {
		defer println("loop", i)
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/deferloop.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/deferloop.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/deferloop.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/deferloop.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8, i64 196, align 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/deferloop.main", %_llgo_6), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %9 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %11 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %11)
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %13 = icmp eq ptr %2, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %14, align 8
// CHECK-NEXT:   %15 = call i32 @sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_5, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_5
// CHECK-NEXT:   %17 = phi i64 [ 0, %_llgo_5 ], [ %25, %_llgo_2 ]
// CHECK-NEXT:   %18 = icmp slt i64 %17, 3
// CHECK-NEXT:   br i1 %18, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %19 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 40)
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 }, ptr %20, i32 0, i32 0
// CHECK-NEXT:   store ptr %19, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 }, ptr %20, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 }, ptr %20, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 }, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 }, ptr %20, i32 0, i32 3
// CHECK-NEXT:   store i64 %17, ptr %24, align 8
// CHECK-NEXT:   store ptr %20, ptr %14, align 8
// CHECK-NEXT:   %25 = add i64 %17, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/deferloop.main", %_llgo_9), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8, %_llgo_3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/deferloop.main", %_llgo_7), ptr %10, align 8
// CHECK-NEXT:   %26 = load i64, ptr %8, align 8
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_8, %_llgo_11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/deferloop.main", %_llgo_7), ptr %12, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %27, [label %_llgo_7, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_11
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_16, %_llgo_6
// CHECK-NEXT:   %28 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %29 = icmp ne ptr %28, null
// CHECK-NEXT:   br i1 %29, label %_llgo_12, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_13, %_llgo_10
// CHECK-NEXT:   %30 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %31 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %30, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %31)
// CHECK-NEXT:   %32 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %32, [label %_llgo_7, label %_llgo_9]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_10
// CHECK-NEXT:   %33 = load { ptr, i64 }, ptr %28, align 8
// CHECK-NEXT:   %34 = extractvalue { ptr, i64 } %33, 1
// CHECK-NEXT:   br label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_12
// CHECK-NEXT:   %35 = icmp eq i64 %34, 0
// CHECK-NEXT:   br i1 %35, label %_llgo_14, label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_13
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/deferloop.main", %_llgo_6), ptr %10, align 8
// CHECK-NEXT:   %36 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %37 = icmp ne ptr %36, null
// CHECK-NEXT:   br i1 %37, label %_llgo_15, label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %38 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %39 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 }, ptr %38, align 8
// CHECK-NEXT:   %40 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 } %39, 0
// CHECK-NEXT:   store ptr %40, ptr %14, align 8
// CHECK-NEXT:   %41 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 } %39, 2
// CHECK-NEXT:   %42 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String", i64 } %39, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_15, %_llgo_14
// CHECK-NEXT:   br label %_llgo_10
// CHECK-NEXT: }
