// LITTEST
package main

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [4 x i8] c"exit", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"ch1", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"ch2", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [31 x i8] c"blocking select matched no case", align 1{{$}}

func main() {
	send()
	recv()
}

func send() {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go func() {
		println(<-ch1)
	}()
	go func() {
		println(<-ch2)
	}()

	select {
	case ch1 <- 100:
	case ch2 <- 200:
	}
}

func recv() {
	c1 := make(chan string)
	c2 := make(chan string)
	go func() {
		c1 <- "ch1"
	}()
	go func() {
		c2 <- "ch2"
	}()

	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-c1:
			println(msg1)
		case msg2 := <-c2:
			println(msg2)
		default:
			println("exit")
		}
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
// CHECK-NEXT:   call void @main.send()
// CHECK-NEXT:   call void @main.recv()
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.recv(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 16, i64 0)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 16, i64 0)
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %5 = getelementptr inbounds { ptr }, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } { ptr @"main.recv$1", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %8 = getelementptr inbounds { { ptr, ptr } }, ptr %7, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %6, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %7, i64 0)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %10 = getelementptr inbounds { ptr }, ptr %9, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } { ptr @"main.recv$2", ptr undef }, ptr %9, 1
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds { { ptr, ptr } }, ptr %12, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %11, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$2", ptr %12, i64 0)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %14 = phi i64 [ 0, %_llgo_0 ], [ %50, %_llgo_4 ]
// CHECK-NEXT:   %15 = icmp slt i64 %14, 2
// CHECK-NEXT:   br i1 %15, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %16 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %17 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %18 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %19 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %19, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %16, 0
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %20, ptr %19, 1
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %21, i32 16, 2
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %22, i1 false, 3
// CHECK-NEXT:   %24 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %24, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %17, 0
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %25, ptr %24, 1
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %26, i32 16, 2
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %27, i1 false, 3
// CHECK-NEXT:   %29 = alloca i8, i64 48, align 1
// CHECK-NEXT:   %30 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %29, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %23, ptr %30, align 8
// CHECK-NEXT:   %31 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %29, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %28, ptr %31, align 8
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %29, 0
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, i64 2, 1
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 2, 2
// CHECK-NEXT:   %35 = call { i64, i1, i1 } @"{{.*}}/runtime/internal/runtime.TrySelect"(%"{{.*}}/runtime/internal/runtime.Slice" %34)
// CHECK-NEXT:   %36 = extractvalue { i64, i1, i1 } %35, 0
// CHECK-NEXT:   %37 = extractvalue { i64, i1, i1 } %35, 1
// CHECK-NEXT:   %38 = extractvalue { i64, i1, i1 } %35, 2
// CHECK-NEXT:   %39 = select i1 %38, i64 %36, i64 -1
// CHECK-NEXT:   %40 = extractvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %23, 1
// CHECK-NEXT:   %41 = load %"{{.*}}/runtime/internal/runtime.String", ptr %40, align 8
// CHECK-NEXT:   %42 = extractvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %28, 1
// CHECK-NEXT:   %43 = load %"{{.*}}/runtime/internal/runtime.String", ptr %42, align 8
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %18)
// CHECK-NEXT:   %44 = insertvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } undef, i64 %39, 0
// CHECK-NEXT:   %45 = insertvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %44, i1 %37, 1
// CHECK-NEXT:   %46 = insertvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %45, %"{{.*}}/runtime/internal/runtime.String" %41, 2
// CHECK-NEXT:   %47 = insertvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %46, %"{{.*}}/runtime/internal/runtime.String" %43, 3
// CHECK-NEXT:   %48 = extractvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %47, 0
// CHECK-NEXT:   %49 = icmp eq i64 %48, 0
// CHECK-NEXT:   br i1 %49, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_8, %_llgo_7, %_llgo_5
// CHECK-NEXT:   %50 = add i64 %14, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %51 = extractvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %47, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %51)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %52 = icmp eq i64 %48, 1
// CHECK-NEXT:   br i1 %52, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %53 = extractvalue { i64, i1, %"{{.*}}/runtime/internal/runtime.String", %"{{.*}}/runtime/internal/runtime.String" } %47, 3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %53)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.recv$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %5, align 8
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %3, ptr %5, i64 16)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.recv$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %5, align 8
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %3, ptr %5, i64 16)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.send(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %5 = getelementptr inbounds { ptr }, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue { ptr, ptr } { ptr @"main.send$1", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %8 = getelementptr inbounds { { ptr, ptr } }, ptr %7, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %6, ptr %8, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$3", ptr %7, i64 0)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %10 = getelementptr inbounds { ptr }, ptr %9, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } { ptr @"main.send$2", ptr undef }, ptr %9, 1
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds { { ptr, ptr } }, ptr %12, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %11, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$4", ptr %12, i64 0)
// CHECK-NEXT:   %14 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %15 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %16 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %17 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %17, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store i64 100, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %14, 0
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %18, ptr %17, 1
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %19, i32 8, 2
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %20, i1 true, 3
// CHECK-NEXT:   %22 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %22, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store i64 200, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %15, 0
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %23, ptr %22, 1
// CHECK-NEXT:   %25 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %24, i32 8, 2
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %25, i1 true, 3
// CHECK-NEXT:   %27 = alloca i8, i64 48, align 1
// CHECK-NEXT:   %28 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %27, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %21, ptr %28, align 8
// CHECK-NEXT:   %29 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %27, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %26, ptr %29, align 8
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %27, 0
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, i64 2, 1
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, i64 2, 2
// CHECK-NEXT:   %33 = call { i64, i1 } @"{{.*}}/runtime/internal/runtime.Select"(%"{{.*}}/runtime/internal/runtime.Slice" %32)
// CHECK-NEXT:   %34 = extractvalue { i64, i1 } %33, 0
// CHECK-NEXT:   %35 = extractvalue { i64, i1 } %33, 1
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %16)
// CHECK-NEXT:   %36 = insertvalue { i64, i1 } undef, i64 %34, 0
// CHECK-NEXT:   %37 = insertvalue { i64, i1 } %36, i1 %35, 1
// CHECK-NEXT:   %38 = extractvalue { i64, i1 } %37, 0
// CHECK-NEXT:   %39 = icmp eq i64 %38, 0
// CHECK-NEXT:   br i1 %39, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %40 = icmp eq i64 %38, 1
// CHECK-NEXT:   br i1 %40, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %41 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 31 }, ptr %41, align 8
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %41, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %42)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.send$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %3, ptr %5, i64 8)
// CHECK-NEXT:   %7 = load i64, ptr %5, align 8
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.send$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %3, ptr %5, i64 8)
// CHECK-NEXT:   %7 = load i64, ptr %5, align 8
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr } } %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %4)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %3)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$2"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr } } %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %4)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %3)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$3"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr } } %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %4)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %3)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }

// CHECK-LABEL: define ptr @"main._llgo_routine$4"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { { ptr, ptr } }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { { ptr, ptr } } %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeRoot"(ptr %0)
// CHECK-NEXT:   %3 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %4)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %3)
// CHECK-NEXT:   ret ptr null
// CHECK-NEXT: }
