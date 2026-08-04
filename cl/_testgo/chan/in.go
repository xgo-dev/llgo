// LITTEST
package main

func main() {
	ch := make(chan int, 10)
	var v any = ch
	println(ch, len(ch), cap(ch), v)
	go func() {
		ch <- 100
	}()
	n := <-ch
	println(n)

	ch2 := make(chan int, 10)
	go func() {
		close(ch2)
	}()
	n2, ok := <-ch2
	println(n2, ok)
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
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 10)
// CHECK-NEXT:   store ptr %1, ptr %0, align 8
// CHECK-NEXT:   %2 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr undef }, ptr %2, 1
// CHECK-NEXT:   %4 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %5 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %6 = call i64 @"{{.*}}/runtime/internal/runtime.ChanLen"(ptr %5)
// CHECK-NEXT:   %7 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %8 = call i64 @"{{.*}}/runtime/internal/runtime.ChanCap"(ptr %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintEface"(%"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %10 = getelementptr inbounds { ptr }, ptr %9, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %10, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %9, 1
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %13 = getelementptr inbounds { { ptr, ptr } }, ptr %12, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %11, ptr %13, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$1", ptr %12, i64 0)
// CHECK-NEXT:   %14 = load ptr, ptr %0, align 8
// CHECK-NEXT:   %15 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %16 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %16, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %17 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %14, ptr %16, i64 8)
// CHECK-NEXT:   %18 = load i64, ptr %16, align 8
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 10)
// CHECK-NEXT:   store ptr %20, ptr %19, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %22 = getelementptr inbounds { ptr }, ptr %21, i32 0, i32 0
// CHECK-NEXT:   store ptr %19, ptr %22, align 8
// CHECK-NEXT:   %23 = insertvalue { ptr, ptr } { ptr @"main.main$2", ptr undef }, ptr %21, 1
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocRoot"(i64 16)
// CHECK-NEXT:   %25 = getelementptr inbounds { { ptr, ptr } }, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } %23, ptr %25, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.NewProc"(ptr @"main._llgo_routine$2", ptr %24, i64 0)
// CHECK-NEXT:   %26 = load ptr, ptr %19, align 8
// CHECK-NEXT:   %27 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %28 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %28, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %29 = call i1 @"{{.*}}/runtime/internal/runtime.ChanRecv"(ptr %26, ptr %28, i64 8)
// CHECK-NEXT:   %30 = load i64, ptr %28, align 8
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %27)
// CHECK-NEXT:   %31 = insertvalue { i64, i1 } undef, i64 %30, 0
// CHECK-NEXT:   %32 = insertvalue { i64, i1 } %31, i1 %29, 1
// CHECK-NEXT:   %33 = extractvalue { i64, i1 } %32, 0
// CHECK-NEXT:   %34 = extractvalue { i64, i1 } %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %33)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %34)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   %4 = call ptr @llvm.stacksave.p0()
// CHECK-NEXT:   %5 = alloca i64, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 8, i1 false)
// CHECK-NEXT:   store i64 100, ptr %5, align 8
// CHECK-NEXT:   %6 = call i1 @"{{.*}}/runtime/internal/runtime.ChanSend"(ptr %3, ptr %5, i64 8)
// CHECK-NEXT:   call void @llvm.stackrestore.p0(ptr %4)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.main$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.ChanClose"(ptr %3)
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
