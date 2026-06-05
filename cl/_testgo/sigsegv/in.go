// LITTEST
package main

// CHECK-LINE: @0 = private unnamed_addr constant [4 x i8] c"init", align 1
// CHECK-LINE: @7 = private unnamed_addr constant [7 x i8] c"recover", align 1
// CHECK-LINE: @8 = private unnamed_addr constant [4 x i8] c"main", align 1

type T struct {
	s int
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testgo/sigsegv.f"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

func f() *T {
	return nil
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/sigsegv.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/sigsegv.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/sigsegv.init$guard", align 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/sigsegv.init#1"()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func init() {
	println("init")
	defer func() {
		r := recover()
		if e, ok := r.(error); ok {
			println("recover", e.Error())
		}
	}()
	println(f().s)
}

func main() {
	println("main")
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/sigsegv.init#1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
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
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/sigsegv.init#1", %_llgo_2), ptr %6, align 8
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
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/sigsegv.init#1", %_llgo_3), ptr %10, align 8
// CHECK-NEXT:   %17 = load i64, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/sigsegv.init#1$1"()
// CHECK-NEXT:   %18 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %2, align 8
// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %18, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %19)
// CHECK-NEXT:   %20 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %20, [label %_llgo_3, label %_llgo_6]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/cl/_testgo/sigsegv.f"()
// CHECK-NEXT:   %22 = icmp eq ptr %21, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %22)
// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/cl/_testgo/sigsegv.T", ptr %21, i32 0, i32 0
// CHECK-NEXT:   %24 = load i64, ptr %23, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/sigsegv.init#1", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"{{.*}}/cl/_testgo/sigsegv.init#1", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %10, align 8
// CHECK-NEXT:   indirectbr ptr %25, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/sigsegv.init#1$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT:   %2 = call i1 @"{{.*}}/runtime/internal/runtime.Implements"(ptr @_llgo_error, ptr %1)
// CHECK-NEXT:   br i1 %2, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %19)
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %19, 0
// CHECK-NEXT:   %5 = getelementptr ptr, ptr %4, i64 3
// CHECK-NEXT:   %6 = load ptr, ptr %5, align 8
// CHECK-NEXT:   %7 = insertvalue { ptr, ptr } undef, ptr %6, 0
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } %7, ptr %3, 1
// CHECK-NEXT:   %9 = extractvalue { ptr, ptr } %8, 1
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %8, 0
// CHECK-NEXT:   %11 = call %"{{.*}}/runtime/internal/runtime.String" %10(ptr %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_5
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Fh8eUJ-Gw4e6TYuajcFIOSCuqSPKAt5nS4ow7xeGXEU", ptr %1)
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %13, 0
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %14, ptr %12, 1
// CHECK-NEXT:   %16 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } undef, %"{{.*}}/runtime/internal/runtime.iface" %15, 0
// CHECK-NEXT:   %17 = insertvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %16, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %18 = phi { %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %17, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %19 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %18, 0
// CHECK-NEXT:   %20 = extractvalue { %"{{.*}}/runtime/internal/runtime.iface", i1 } %18, 1
// CHECK-NEXT:   br i1 %20, label %_llgo_1, label %_llgo_2
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/sigsegv.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.interequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.interequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
