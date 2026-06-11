// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [19 x i8] c"will panic in defer", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [3 x i8] c"end", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [13 x i8] c"panic in main", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/recoverthenpanic.End"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @llvm.frameaddress.p0(i32 1)
// CHECK-NEXT:   %1 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"(ptr %0)
// CHECK-NEXT:   %2 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %1, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %3 = xor i1 %2, true
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %5 = alloca i8, i64 {{.*}}, align 1
// CHECK-NEXT:   %6 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 56)
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 0
// CHECK-NEXT:   store ptr %5, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 2
// CHECK-NEXT:   store ptr %4, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_5), ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 4
// CHECK-NEXT:   store ptr null, ptr %12, align 8
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %13, align 8
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 6
// CHECK-NEXT:   store ptr %6, ptr %14, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %7)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 1
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 3
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 4
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %18, align 8
// CHECK-NEXT:   %19 = call i32 @{{.*}}sigsetjmp(ptr %5, i32 0)
// CHECK-NEXT:   %20 = icmp eq i32 %19, 0
// CHECK-NEXT:   br i1 %20, label %_llgo_4, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4
// CHECK-NEXT:   %21 = load i64, ptr %15, align 8
// CHECK-NEXT:   %22 = or i64 %21, 1
// CHECK-NEXT:   store i64 %22, ptr %15, align 8
// CHECK-NEXT:   %23 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store ptr %23, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %24, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %26, align 8
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %24, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %1, ptr %27, align 8
// CHECK-NEXT:   store ptr %24, ptr %18, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 19 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 3 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_8), ptr %17, align 8
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_6), ptr %16, align 8
// CHECK-NEXT:   %28 = load i64, ptr %15, align 8
// CHECK-NEXT:   %29 = and i64 %28, 1
// CHECK-NEXT:   %30 = icmp ne i64 %29, 0
// CHECK-NEXT:   br i1 %30, label %_llgo_9, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7, %_llgo_10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %4)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.End", %_llgo_6), ptr %17, align 8
// CHECK-NEXT:   %31 = load ptr, ptr %16, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_6, label %_llgo_5]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_10
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %32 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %33 = icmp ne ptr %32, null
// CHECK-NEXT:   br i1 %33, label %_llgo_11, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_12, %_llgo_5
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %7, align 8
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %34, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %35)
// CHECK-NEXT:   %36 = load ptr, ptr %17, align 8
// CHECK-NEXT:   indirectbr ptr %36, [label %_llgo_6, label %_llgo_8]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %37 = load ptr, ptr %18, align 8
// CHECK-NEXT:   %38 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %37, align 8
// CHECK-NEXT:   %39 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %38, 0
// CHECK-NEXT:   store ptr %39, ptr %18, align 8
// CHECK-NEXT:   %40 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %38, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %37)
// CHECK-NEXT:   %41 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %40)
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
// CHECK-NEXT:   %2 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 56)
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_2), ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 4
// CHECK-NEXT:   store ptr null, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 6
// CHECK-NEXT:   store ptr %2, ptr %10, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %3)
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 3
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 4
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %14, align 8
// CHECK-NEXT:   %15 = call i32 @{{.*}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %17 = load i64, ptr %11, align 8
// CHECK-NEXT:   %18 = call ptr @llvm.frameaddress.p0(i32 0)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.StartRecoverFrame"(ptr %18)
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/recoverthenpanic.End"()
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.EndRecoverFrame"(ptr %19)
// CHECK-NEXT:   %20 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %3, align 8
// CHECK-NEXT:   %21 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %20, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %21)
// CHECK-NEXT:   %22 = load ptr, ptr %13, align 8
// CHECK-NEXT:   indirectbr ptr %22, [label %_llgo_3]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 13 }, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %23, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %24)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/recoverthenpanic.main", %_llgo_3), ptr %13, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %25, [label %_llgo_3, label %_llgo_2]
// CHECK-NEXT: }
