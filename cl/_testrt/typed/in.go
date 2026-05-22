// LITTEST
package main

// CHECK: @0 = private unnamed_addr constant [5 x i8] c"hello", align 1

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typed.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/typed.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/typed.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type T string
type A [2]int

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typed.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/typed.T", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// CHECK-NEXT:   %3 = icmp eq ptr %2, @"_llgo_{{.*}}/cl/_testrt/typed.T"
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// CHECK-NEXT:   %7 = icmp eq ptr %6, @_llgo_string
// CHECK-NEXT:   br i1 %7, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.TypeAssertError"(ptr %2, ptr @"_llgo_{{.*}}/cl/_testrt/typed.T", ptr @_llgo_any)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %8)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// CHECK-NEXT:   %10 = load %"{{.*}}/runtime/internal/runtime.String", ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" %10, 0
// CHECK-NEXT:   %12 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %11, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %13 = phi { %"{{.*}}/runtime/internal/runtime.String", i1 } [ %12, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %14 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %13, 0
// CHECK-NEXT:   %15 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %13, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %16, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %17 = getelementptr inbounds i64, ptr %16, i64 0
// CHECK-NEXT:   %18 = getelementptr inbounds i64, ptr %16, i64 1
// CHECK-NEXT:   store i64 1, ptr %17, align 8
// CHECK-NEXT:   store i64 2, ptr %18, align 8
// CHECK-NEXT:   %19 = load [2 x i64], ptr %16, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store [2 x i64] %19, ptr %20, align 8
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_{{.*}}/cl/_testrt/typed.A", ptr undef }, ptr %20, 1
// CHECK-NEXT:   %22 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %22, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %23 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %21, 0
// CHECK-NEXT:   %24 = icmp eq ptr %23, @"_llgo_{{.*}}/cl/_testrt/typed.A"
// CHECK-NEXT:   br i1 %24, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %25 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %21, 1
// CHECK-NEXT:   %26 = load [2 x i64], ptr %25, align 8
// CHECK-NEXT:   %27 = insertvalue { [2 x i64], i1 } undef, [2 x i64] %26, 0
// CHECK-NEXT:   %28 = insertvalue { [2 x i64], i1 } %27, i1 true, 1
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   %29 = phi { [2 x i64], i1 } [ %28, %_llgo_6 ], [ zeroinitializer, %_llgo_7 ]
// CHECK-NEXT:   %30 = extractvalue { [2 x i64], i1 } %29, 0
// CHECK-NEXT:   store [2 x i64] %30, ptr %22, align 8
// CHECK-NEXT:   %31 = extractvalue { [2 x i64], i1 } %29, 1
// CHECK-NEXT:   %32 = getelementptr inbounds i64, ptr %22, i64 0
// CHECK-NEXT:   %33 = load i64, ptr %32, align 8
// CHECK-NEXT:   %34 = getelementptr inbounds i64, ptr %22, i64 1
// CHECK-NEXT:   %35 = load i64, ptr %34, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %33)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %31)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	var v any = T("hello")
	println(v.(T))
	s, ok := v.(string)
	println(s, ok)

	var a any = A{1, 2}
	ar, ok := a.(A)
	println(ar[0], ar[1], ok)
}

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
