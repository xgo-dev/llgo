// LITTEST
package main

import (
	"sync"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [4 x i8] c"done", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"work 1", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"work 2", align 1{{$}}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		println("work 1")
	}()
	go func() {
		defer wg.Done()
		println("work 2")
	}()
	wg.Wait()
	println("done")
}

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @sync.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   call void @"sync.(*WaitGroup).Add"(ptr %0, i64 2)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %5 = getelementptr inbounds { { ptr, ptr } }, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %5, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %4, i64 0)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %7 = getelementptr inbounds { ptr }, ptr %6, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr %6, 1
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %10 = getelementptr inbounds { { ptr, ptr } }, ptr %9, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %8, ptr %10, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$2", ptr %9, i64 0)
// CHECK-NEXT:   call void @"sync.(*WaitGroup).Wait"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %4 = alloca i8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 2
// CHECK-NEXT:   store ptr %3, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"main.main$1", %_llgo_2), ptr %9, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %5)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 4
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %13, align 8
// CHECK-NEXT:   %14 = call i32 @{{(__)?}}sigsetjmp(ptr %4, i32 0)
// CHECK-NEXT:   %15 = icmp eq i32 %14, 0
// CHECK-NEXT:   br i1 %15, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"main.main$1", %_llgo_3), ptr %11, align 8
// CHECK-NEXT:   %16 = load i64, ptr %10, align 8
// CHECK-NEXT:   %17 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %18 = icmp ne ptr %17, null
// CHECK-NEXT:   br i1 %18, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %3)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 0
// CHECK-NEXT:   store ptr %19, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 2
// CHECK-NEXT:   store ptr %2, ptr %23, align 8
// CHECK-NEXT:   store ptr %20, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"main.main$1", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"main.main$1", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %24 = load ptr, ptr %11, align 8
// CHECK-NEXT:   indirectbr ptr %24, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %25 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %26 = load { ptr, i64, ptr }, ptr %25, align 8
// CHECK-NEXT:   %27 = extractvalue { ptr, i64, ptr } %26, 0
// CHECK-NEXT:   store ptr %27, ptr %13, align 8
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, ptr } %26, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %25)
// CHECK-NEXT:   call void @"sync.(*WaitGroup).Done"(ptr %28)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %30)
// CHECK-NEXT:   %31 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %4 = alloca i8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %6, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %7, align 8
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 2
// CHECK-NEXT:   store ptr %3, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"main.main$2", %_llgo_2), ptr %9, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %5)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 3
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 4
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %13, align 8
// CHECK-NEXT:   %14 = call i32 @{{(__)?}}sigsetjmp(ptr %4, i32 0)
// CHECK-NEXT:   %15 = icmp eq i32 %14, 0
// CHECK-NEXT:   br i1 %15, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"main.main$2", %_llgo_3), ptr %11, align 8
// CHECK-NEXT:   %16 = load i64, ptr %10, align 8
// CHECK-NEXT:   %17 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %18 = icmp ne ptr %17, null
// CHECK-NEXT:   br i1 %18, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %3)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %19 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   %21 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 0
// CHECK-NEXT:   store ptr %19, ptr %21, align 8
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, ptr }, ptr %20, i32 0, i32 2
// CHECK-NEXT:   store ptr %2, ptr %23, align 8
// CHECK-NEXT:   store ptr %20, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr blockaddress(@"main.main$2", %_llgo_6), ptr %12, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"main.main$2", %_llgo_3), ptr %12, align 8
// CHECK-NEXT:   %24 = load ptr, ptr %11, align 8
// CHECK-NEXT:   indirectbr ptr %24, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %25 = load ptr, ptr %13, align 8
// CHECK-NEXT:   %26 = load { ptr, i64, ptr }, ptr %25, align 8
// CHECK-NEXT:   %27 = extractvalue { ptr, i64, ptr } %26, 0
// CHECK-NEXT:   store ptr %27, ptr %13, align 8
// CHECK-NEXT:   %28 = extractvalue { ptr, i64, ptr } %26, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %25)
// CHECK-NEXT:   call void @"sync.(*WaitGroup).Done"(ptr %28)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %5, align 8
// CHECK-NEXT:   %30 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %30)
// CHECK-NEXT:   %31 = load ptr, ptr %12, align 8
// CHECK-NEXT:   indirectbr ptr %31, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/runtime/internal/runtime.LocalContext", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/runtime/internal/runtime.EnterLocalContext"(ptr %1)
// CHECK-NEXT:   %3 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %4 = extractvalue { { ptr, ptr } } %3, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %4, 1
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %4, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %6)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.LeaveLocalContext"(ptr %1, i64 %2)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/runtime/internal/runtime.LocalContext", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/runtime/internal/runtime.EnterLocalContext"(ptr %1)
// CHECK-NEXT:   %3 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %4 = extractvalue { { ptr, ptr } } %3, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %4, 1
// CHECK-NEXT:   %6 = extractvalue { ptr, ptr } %4, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %6)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.LeaveLocalContext"(ptr %1, i64 %2)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }
