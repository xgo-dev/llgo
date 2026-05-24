// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [1 x i8] c"A", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [1 x i8] c"B", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [13 x i8] c"panic in main", align 1
// CHECK-LINE: @4 = private unnamed_addr constant [10 x i8] c"in defer 1", align 1
// CHECK-LINE: @5 = private unnamed_addr constant [16 x i8] c"panic in defer 1", align 1
// CHECK-LINE: @6 = private unnamed_addr constant [10 x i8] c"in defer 2", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [16 x i8] c"panic in defer 2", align 1

func main() {
	defer println("A")
	defer func() {
		if e := recover(); e != nil {
			println("in defer 1")
			panic("panic in defer 1")
		}
	}()
	defer func() {
		println("in defer 2")
		panic("panic in defer 2")
	}()
	defer println("B")
	panic("panic in main")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/defer5.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/defer5.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/defer5.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/defer5.main"(){{.*}} {
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
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_2), ptr %6, align 8
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
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_8), ptr %10, align 8
// CHECK-NEXT:   %17 = load i64, ptr %8, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %19 = icmp ne ptr %18, null
// CHECK-NEXT:   br i1 %19, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_12
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %21, i32 0, i32 0
// CHECK-NEXT:   store ptr %20, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %21, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %21, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 1 }, ptr %24, align 8
// CHECK-NEXT:   store ptr %21, ptr %14, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %26, i32 0, i32 0
// CHECK-NEXT:   store ptr %25, ptr %27, align 8
// CHECK-NEXT:   %28 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %26, i32 0, i32 1
// CHECK-NEXT:   store i64 3, ptr %28, align 8
// CHECK-NEXT:   %29 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %26, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 1 }, ptr %29, align 8
// CHECK-NEXT:   store ptr %26, ptr %14, align 8
// CHECK-NEXT:   %30 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 13 }, ptr %30, align 8
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %30, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %31)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %32 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %32, [label %_llgo_3, label %_llgo_6, label %_llgo_7, label %_llgo_8, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_7
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %33 = load i64, ptr %8, align 8
// CHECK-NEXT:   %34 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %35 = icmp ne ptr %34, null
// CHECK-NEXT:   br i1 %35, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_6), ptr %10, align 8
// CHECK-NEXT:   %36 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/defer5.main$1"()
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_5, %_llgo_10
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/defer5.main", %_llgo_7), ptr %10, align 8
// CHECK-NEXT:   %37 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/defer5.main$2"()
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %38 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %39 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %38, align 8
// CHECK-NEXT:   %40 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %39, 0
// CHECK-NEXT:   store ptr %40, ptr %14, align 8
// CHECK-NEXT:   %41 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %39, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %38)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_9, %_llgo_2
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_6
// CHECK-NEXT:   %42 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %43 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" }, ptr %42, align 8
// CHECK-NEXT:   %44 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %43, 0
// CHECK-NEXT:   store ptr %44, ptr %14, align 8
// CHECK-NEXT:   %45 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.String" } %43, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_11, %_llgo_6
// CHECK-NEXT:   %46 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %46, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %47)
// CHECK-NEXT:   %48 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %48, [label %_llgo_3]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/defer5.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %2 = xor i1 %1, true
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @4, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 16 }, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/defer5.main$2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 16 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %0, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %1)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
