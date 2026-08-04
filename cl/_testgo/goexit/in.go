// LITTEST
package main

import (
	"runtime"
)

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"must nil", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"error", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"must error", align 1{{$}}

func main() {
	demo1()
	demo2()
	demo3()
}

func demo1() {
	ch := make(chan bool)
	go func() {
		defer func() {
			ch <- true
		}()
		runtime.Goexit()
	}()
	<-ch
}

func demo2() {
	ch := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				panic("must nil")
			}
			ch <- true
		}()
		runtime.Goexit()
	}()
	<-ch
}

func demo3() {
	ch := make(chan bool)
	go func() {
		defer func() {
			r := recover()
			if r != "error" {
				panic("must error")
			}
			ch <- true
		}()
		defer func() {
			if r := recover(); r != nil {
				panic("must nil")
			}
			panic("error")
		}()
		runtime.Goexit()
	}()
	<-ch
}

// CHECK-LABEL: define void @main.demo1(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 1, i64 0)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.demo1$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %6 = getelementptr inbounds { { ptr, ptr } }, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %5, i64 0)
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %9 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %9, i8 0, i64 1, i1 false)
// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %7, ptr %9, i64 1)
// CHECK-NEXT:   %11 = load i1, ptr %9, align 1
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } { ptr @"main.demo1$1$1", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %7 = alloca i8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store ptr %7, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 2
// CHECK-NEXT:   store ptr %6, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"main.demo1$1", %_llgo_2), ptr %12, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %8)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 4
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %16, align 8
// CHECK-NEXT:   %17 = call i32 @{{(__)?}}sigsetjmp(ptr %7, i32 0)
// CHECK-NEXT:   %18 = icmp eq i32 %17, 0
// CHECK-NEXT:   br i1 %18, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"main.demo1$1", %_llgo_3), ptr %14, align 8
// CHECK-NEXT:   %19 = load i64, ptr %13, align 8
// CHECK-NEXT:   %20 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %21 = icmp ne ptr %20, null
// CHECK-NEXT:   br i1 %21, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %6)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %5, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %16, align 8
// CHECK-NEXT:   call void @runtime.Goexit()
// CHECK-NEXT:   store ptr blockaddress(@"main.demo1$1", %_llgo_6), ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"main.demo1$1", %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %27, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %28 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %29 = load { ptr, i64, { ptr, ptr } }, ptr %28, align 8
// CHECK-NEXT:   %30 = extractvalue { ptr, i64, { ptr, ptr } } %29, 0
// CHECK-NEXT:   store ptr %30, ptr %16, align 8
// CHECK-NEXT:   %31 = extractvalue { ptr, i64, { ptr, ptr } } %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %28)
// CHECK-NEXT:   %32 = extractvalue { ptr, ptr } %31, 1
// CHECK-NEXT:   %33 = extractvalue { ptr, ptr } %31, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %33)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %32)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, align 8
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %34, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %35)
// CHECK-NEXT:   %36 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %36, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo1$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 1, i1 false)
// CHECK-NEXT:   store i1 true, ptr %5, align 1
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %3, ptr %5, i64 1)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.demo2(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 1, i64 0)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.demo2$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %6 = getelementptr inbounds { { ptr, ptr } }, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$2", ptr %5, i64 0)
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %9 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %9, i8 0, i64 1, i1 false)
// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %7, ptr %9, i64 1)
// CHECK-NEXT:   %11 = load i1, ptr %9, align 1
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo2$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } { ptr @"main.demo2$1$1", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %7 = alloca i8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store ptr %7, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 2
// CHECK-NEXT:   store ptr %6, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"main.demo2$1", %_llgo_2), ptr %12, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %8)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 4
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %16, align 8
// CHECK-NEXT:   %17 = call i32 @{{(__)?}}sigsetjmp(ptr %7, i32 0)
// CHECK-NEXT:   %18 = icmp eq i32 %17, 0
// CHECK-NEXT:   br i1 %18, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"main.demo2$1", %_llgo_3), ptr %14, align 8
// CHECK-NEXT:   %19 = load i64, ptr %13, align 8
// CHECK-NEXT:   %20 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %21 = icmp ne ptr %20, null
// CHECK-NEXT:   br i1 %21, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %6)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %5, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %16, align 8
// CHECK-NEXT:   call void @runtime.Goexit()
// CHECK-NEXT:   store ptr blockaddress(@"main.demo2$1", %_llgo_6), ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"main.demo2$1", %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %27, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %28 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %29 = load { ptr, i64, { ptr, ptr } }, ptr %28, align 8
// CHECK-NEXT:   %30 = extractvalue { ptr, i64, { ptr, ptr } } %29, 0
// CHECK-NEXT:   store ptr %30, ptr %16, align 8
// CHECK-NEXT:   %31 = extractvalue { ptr, i64, { ptr, ptr } } %29, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %28)
// CHECK-NEXT:   %32 = extractvalue { ptr, ptr } %31, 1
// CHECK-NEXT:   %33 = extractvalue { ptr, ptr } %31, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %33)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %32)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %34 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, align 8
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %34, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %35)
// CHECK-NEXT:   %36 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %36, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo2$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %2, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %4 = xor i1 %3, true
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %10 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %10, i8 0, i64 1, i1 false)
// CHECK-NEXT:   store i1 true, ptr %10, align 1
// CHECK-NEXT:   %11 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %8, ptr %10, i64 1)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %9)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.demo3(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 1, i64 0)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds { ptr }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue { ptr, ptr } { ptr @"main.demo3$1", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %6 = getelementptr inbounds { { ptr, ptr } }, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %4, ptr %6, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$3", ptr %5, i64 0)
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %9 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %9, i8 0, i64 1, i1 false)
// CHECK-NEXT:   %10 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %7, ptr %9, i64 1)
// CHECK-NEXT:   %11 = load i1, ptr %9, align 1
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo3$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } { ptr @"main.demo3$1$1", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %7 = alloca i8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store ptr %7, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 2
// CHECK-NEXT:   store ptr %6, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@"main.demo3$1", %_llgo_2), ptr %12, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %8)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 4
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %16, align 8
// CHECK-NEXT:   %17 = call i32 @{{(__)?}}sigsetjmp(ptr %7, i32 0)
// CHECK-NEXT:   %18 = icmp eq i32 %17, 0
// CHECK-NEXT:   br i1 %18, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@"main.demo3$1", %_llgo_7), ptr %14, align 8
// CHECK-NEXT:   %19 = load i64, ptr %13, align 8
// CHECK-NEXT:   call void @"main.demo3$1$2"()
// CHECK-NEXT:   br label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_9
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %6)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %20 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 0
// CHECK-NEXT:   store ptr %20, ptr %22, align 8
// CHECK-NEXT:   %23 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %23, align 8
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %21, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %5, ptr %24, align 8
// CHECK-NEXT:   store ptr %21, ptr %16, align 8
// CHECK-NEXT:   call void @runtime.Goexit()
// CHECK-NEXT:   store ptr blockaddress(@"main.demo3$1", %_llgo_6), ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@"main.demo3$1", %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %25 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %25, [label %_llgo_3, label %_llgo_7, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_9
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5, %_llgo_2
// CHECK-NEXT:   store ptr blockaddress(@"main.demo3$1", %_llgo_3), ptr %14, align 8
// CHECK-NEXT:   %26 = load i64, ptr %13, align 8
// CHECK-NEXT:   %27 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %28 = icmp ne ptr %27, null
// CHECK-NEXT:   br i1 %28, label %_llgo_8, label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %29 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %30 = load { ptr, i64, { ptr, ptr } }, ptr %29, align 8
// CHECK-NEXT:   %31 = extractvalue { ptr, i64, { ptr, ptr } } %30, 0
// CHECK-NEXT:   store ptr %31, ptr %16, align 8
// CHECK-NEXT:   %32 = extractvalue { ptr, i64, { ptr, ptr } } %30, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %29)
// CHECK-NEXT:   %33 = extractvalue { ptr, ptr } %32, 1
// CHECK-NEXT:   %34 = extractvalue { ptr, ptr } %32, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %34)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %33)
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8, %_llgo_7
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, align 8
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %35, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %36)
// CHECK-NEXT:   %37 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %37, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo3$1$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %2, %"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = xor i1 %5, true
// CHECK-NEXT:   br i1 %6, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 }, ptr %7, align 8
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %9 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %12 = alloca i1, align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %12, i8 0, i64 1, i1 false)
// CHECK-NEXT:   store i1 true, ptr %12, align 1
// CHECK-NEXT:   %13 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %10, ptr %12, i64 1)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %11)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.demo3$1$2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %1 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %0, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   %2 = xor i1 %1, true
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   unreachable
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
// CHECK-NEXT:   call void @main.demo1()
// CHECK-NEXT:   call void @main.demo2()
// CHECK-NEXT:   call void @main.demo3()
// CHECK-NEXT:   ret void
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

// CHECK-LABEL: define ptr @"main._llgo_routine$3"(ptr %0){{.*}} {
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
