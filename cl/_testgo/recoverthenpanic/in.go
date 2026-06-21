// LITTEST
package main

// CHECK: @0 = private unnamed_addr constant [19 x i8] c"will panic in defer", align 1
// CHECK: @1 = private unnamed_addr constant [3 x i8] c"end", align 1
// CHECK: @2 = private unnamed_addr constant [13 x i8] c"panic in main", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/recoverthenpanic.End"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %2 = xor i1 %1, true
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %4 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 2
// CHECK-NEXT:   store ptr %3, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_5), ptr %9, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %5)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 4
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %13, align 8
// CHECK-NEXT:   %14 = call i32 @{{.*}}sigsetjmp(ptr %4, i32 0)
// CHECK-NEXT:   %15 = icmp eq i32 %14, 0
// CHECK-NEXT:   br i1 %15, label %_llgo_4, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %16 = load i64, ptr %10, align 8
// CHECK-NEXT:   %17 = or i64 %16, 1
// CHECK-NEXT:   store i64 %17, ptr %10, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %20 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %19, i32 0, i32 0
// CHECK-NEXT:   store ptr %18, ptr %20, align 8
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %19, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %19, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %22, align 8
// CHECK-NEXT:   store ptr %19, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 19 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 3 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_8), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_6), ptr %11, align 8
// CHECK-NEXT:   %23 = load i64, ptr %10, align 8
// CHECK-NEXT:   %24 = and i64 %23, 1
// CHECK-NEXT:   %25 = icmp ne i64 %24, 0
// CHECK-NEXT:   br i1 %25, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7, %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %3)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   %26 = load ptr, ptr %11, align 8
// CHECK-NEXT:   indirectbr ptr %26, [label %_llgo_6, label %_llgo_5]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %27 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %28 = icmp ne ptr %27, null
// CHECK-NEXT:   br i1 %28, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_12, %_llgo_5
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %30)
// CHECK-NEXT:   %31 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_6, label %_llgo_8]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %32 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %33 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %32, align 8
// CHECK-NEXT:   %34 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %33, 0
// CHECK-NEXT:   store ptr %34, ptr %13, align 8
// CHECK-NEXT:   %35 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %33, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_9
// CHECK-NEXT:   br label %_llgo_10
// CHECK-NEXT: }

func End() {
	if recovered := recover(); recovered != nil {
		// Record but don't stop the panic.
		defer panic(recovered)
		println("will panic in defer")
	}
	println("end")
}

func main() {
	defer End()
	panic("panic in main")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/recoverthenpanic.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/recoverthenpanic.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/recoverthenpanic.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/recoverthenpanic.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_2), ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %2)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 4
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %10, align 8
// CHECK-NEXT:   %11 = call i32 @{{.*}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %12 = icmp eq i32 %11, 0
// CHECK-NEXT:   br i1 %12, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_3), ptr %8, align 8
// CHECK-NEXT:   %13 = load i64, ptr %7, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/recoverthenpanic.End"()
// CHECK-NEXT:   %14 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %14, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %15)
// CHECK-NEXT:   %16 = load ptr, ptr %9, align 8
// CHECK-NEXT:   indirectbr ptr %16, [label %_llgo_3]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 13 }, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %17, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_3), ptr %9, align 8
// CHECK-NEXT:   %19 = load ptr, ptr %8, align 8
// CHECK-NEXT:   indirectbr ptr %19, [label %_llgo_3, label %_llgo_2]
// CHECK-NEXT: }
