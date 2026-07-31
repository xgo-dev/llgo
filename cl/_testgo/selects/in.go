// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [4 x i8] c"c1<-", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [4 x i8] c"<-c2", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [4 x i8] c"<-c4", align 1{{$}}
// CHECK: {{^}}@3 = private unnamed_addr constant [31 x i8] c"blocking select matched no case", align 1{{$}}
// CHECK: {{^}}@5 = private unnamed_addr constant [4 x i8] c"<-c1", align 1{{$}}
// CHECK: {{^}}@6 = private unnamed_addr constant [4 x i8] c"c2<-", align 1{{$}}
// CHECK: {{^}}@7 = private unnamed_addr constant [4 x i8] c"<-c3", align 1{{$}}

func main() {
	c1 := make(chan struct{}, 1)
	c2 := make(chan struct{}, 1)
	c3 := make(chan struct{}, 1)
	c4 := make(chan struct{}, 1)

	go func() {
		<-c1
		println("<-c1")

		select {
		case c2 <- struct{}{}:
			println("c2<-")
		case <-c3:
			println("<-c3")
		}
	}()

	c1 <- struct{}{}
	println("c1<-")

	select {
	case <-c2:
		println("<-c2")
	case <-c4:
		println("<-c4")
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
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK-NEXT:   store ptr %5, ptr %4, align 8
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 0, i64 1)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   %8 = getelementptr inbounds { ptr, ptr, ptr }, ptr %7, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %8, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds { ptr, ptr, ptr }, ptr %7, i32 0, i32 1
// CHECK-NEXT:   store ptr %2, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds { ptr, ptr, ptr }, ptr %7, i32 0, i32 2
// CHECK-NEXT:   store ptr %4, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %7, 1
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds { { ptr, ptr } }, ptr %12, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %11, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %12, i64 0)
// CHECK-NEXT:   %14 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %15 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %16 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %16, i8 0, i64 0, i1 false)
// CHECK-NEXT:   store {} zeroinitializer, ptr %16, align 1
// CHECK-NEXT:   %17 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %14, ptr %16, i64 0)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %18 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %19 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %20 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %20, i8 0, i64 0, i1 false)
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %18, 0
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %21, ptr %20, 1
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %22, i32 0, 2
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %23, i1 false, 3
// CHECK-NEXT:   %25 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %25, i8 0, i64 0, i1 false)
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %6, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %26, ptr %25, 1
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %27, i32 0, 2
// CHECK-NEXT:   %29 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %28, i1 false, 3
// CHECK-NEXT:   %30 = alloca i8, i64 48, align 1
// CHECK-NEXT:   %31 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %30, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %24, ptr %31, align 8
// CHECK-NEXT:   %32 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %30, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %29, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %30, 0
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 2, 1
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, i64 2, 2
// CHECK-NEXT:   %36 = call { i64, i1 } @"{{.*}}/runtime/internal/runtime.Select"(%"{{.*}}/runtime/internal/runtime.Slice" %35)
// CHECK-NEXT:   %37 = extractvalue { i64, i1 } %36, 0
// CHECK-NEXT:   %38 = extractvalue { i64, i1 } %36, 1
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %24, 1
// CHECK-NEXT:   %40 = icmp eq ptr %39, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %40)
// CHECK-NEXT:   %41 = extractvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %29, 1
// CHECK-NEXT:   %42 = icmp eq ptr %41, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %42)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %19)
// CHECK-NEXT:   %43 = insertvalue { i64, i1, {}, {} } undef, i64 %37, 0
// CHECK-NEXT:   %44 = insertvalue { i64, i1, {}, {} } %43, i1 %38, 1
// CHECK-NEXT:   %45 = insertvalue { i64, i1, {}, {} } %44, {} zeroinitializer, 2
// CHECK-NEXT:   %46 = insertvalue { i64, i1, {}, {} } %45, {} zeroinitializer, 3
// CHECK-NEXT:   %47 = extractvalue { i64, i1, {}, {} } %46, 0
// CHECK-NEXT:   %48 = icmp eq i64 %47, 0
// CHECK-NEXT:   br i1 %48, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %49 = icmp eq i64 %47, 1
// CHECK-NEXT:   br i1 %49, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 31 }, ptr %50, align 8
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %50, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %51)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr, ptr, ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr, ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 0, i1 false)
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %3, ptr %5, i64 0)
// CHECK-NEXT:   %7 = icmp eq ptr %5, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %7)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @5, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr, ptr } %1, 1
// CHECK-NEXT:   %9 = load ptr, ptr %8, align 8
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr, ptr } %1, 2
// CHECK-NEXT:   %11 = load ptr, ptr %10, align 8
// CHECK-NEXT:   %12 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %13 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %13, i8 0, i64 0, i1 false)
// CHECK-NEXT:   store {} zeroinitializer, ptr %13, align 1
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %9, 0
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %14, ptr %13, 1
// CHECK-NEXT:   %16 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %15, i32 0, 2
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %16, i1 true, 3
// CHECK-NEXT:   %18 = alloca {}, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %18, i8 0, i64 0, i1 false)
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" undef, ptr %11, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %19, ptr %18, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %20, i32 0, 2
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %21, i1 false, 3
// CHECK-NEXT:   %23 = alloca i8, i64 48, align 1
// CHECK-NEXT:   %24 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %23, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %17, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr %"{{.*}}/runtime/internal/runtime.ChanOp", ptr %23, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.ChanOp" %22, ptr %25, align 8
// CHECK-NEXT:   %26 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %23, 0
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %26, i64 2, 1
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %27, i64 2, 2
// CHECK-NEXT:   %29 = call { i64, i1 } @"{{.*}}/runtime/internal/runtime.Select"(%"{{.*}}/runtime/internal/runtime.Slice" %28)
// CHECK-NEXT:   %30 = extractvalue { i64, i1 } %29, 0
// CHECK-NEXT:   %31 = extractvalue { i64, i1 } %29, 1
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.ChanOp" %22, 1
// CHECK-NEXT:   %33 = icmp eq ptr %32, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %33)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %12)
// CHECK-NEXT:   %34 = insertvalue { i64, i1, {} } undef, i64 %30, 0
// CHECK-NEXT:   %35 = insertvalue { i64, i1, {} } %34, i1 %31, 1
// CHECK-NEXT:   %36 = insertvalue { i64, i1, {} } %35, {} zeroinitializer, 2
// CHECK-NEXT:   %37 = extractvalue { i64, i1, {} } %36, 0
// CHECK-NEXT:   %38 = icmp eq i64 %37, 0
// CHECK-NEXT:   br i1 %38, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %39 = icmp eq i64 %37, 1
// CHECK-NEXT:   br i1 %39, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %40 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 31 }, ptr %40, align 8
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %40, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %41)
// CHECK-NEXT:   unreachable
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
