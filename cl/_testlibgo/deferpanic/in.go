// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [14 x i8] c"panic in defer", align 1
// CHECK-LINE: @1 = private unnamed_addr constant [6 x i8] c"string", align 1
// CHECK-LINE: @2 = private unnamed_addr constant [8 x i8] c"run main", align 1

func main() {
	defer func() {
		e := recover()
		println(e.(string))
	}()
	defer panic("panic in defer")
	println("run main")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/deferpanic.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testlibgo/deferpanic.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testlibgo/deferpanic.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/deferpanic.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %1 = alloca {{.*}}, align {{[0-9]+}}
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %1, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 2
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/deferpanic.main", %_llgo_2), ptr %6, align 8
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
// CHECK-NEXT:   %15 = call i32 @{{.*}}sigsetjmp(ptr %1, i32 0)
// CHECK-NEXT:   %16 = icmp eq i32 %15, 0
// CHECK-NEXT:   br i1 %16, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/deferpanic.main", %_llgo_7), ptr %10, align 8
// CHECK-NEXT:   %17 = load i64, ptr %8, align 8
// CHECK-NEXT:   %18 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %19 = icmp ne ptr %18, null
// CHECK-NEXT:   br i1 %19, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 14 }, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %20, 1
// CHECK-NEXT:   %22 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 1, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %21, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %14, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 8 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/deferpanic.main", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/deferpanic.main", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %27, [label %_llgo_3, label %_llgo_7, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5, %_llgo_9
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testlibgo/deferpanic.main", %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %28 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testlibgo/deferpanic.main$1"()
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %30)
// CHECK-NEXT:   %31 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %32 = load ptr, ptr %14, align 8
// CHECK-NEXT:   %33 = load { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" }, ptr %32, align 8
// CHECK-NEXT:   %34 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %33, 0
// CHECK-NEXT:   store ptr %34, ptr %14, align 8
// CHECK-NEXT:   %35 = extractvalue { ptr, i64, %"{{.*}}/runtime/internal/runtime.eface" } %33, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %35)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_2
// CHECK-NEXT:   br label %_llgo_7
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/deferpanic.main$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %2 = icmp eq ptr %1, @_llgo_string
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 6 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
