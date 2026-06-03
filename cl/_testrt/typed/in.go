// LITTEST
package main

type T string
type A [2]int

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/typed.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 5 }, ptr %0, align 8
// CHECK-NEXT:   %1 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/typed.T", ptr undef }, ptr %0, 1
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 0
// CHECK-NEXT:   %3 = icmp eq ptr %2, @"_llgo_github.com/goplus/llgo/cl/_testrt/typed.T"
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
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @3, i64 70 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %1, 1
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.String", ptr %10, align 8
// CHECK-NEXT:   %12 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" %11, 0
// CHECK-NEXT:   %13 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %12, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %14 = phi { %"{{.*}}/runtime/internal/runtime.String", i1 } [ %13, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %15 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %14, 0
// CHECK-NEXT:   %16 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %14, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %17 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %17, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %18 = getelementptr inbounds i64, ptr %17, i64 0
// CHECK-NEXT:   %19 = getelementptr inbounds i64, ptr %17, i64 1
// CHECK-NEXT:   store i64 1, ptr %18, align 8
// CHECK-NEXT:   store i64 2, ptr %19, align 8
// CHECK-NEXT:   %20 = load [2 x i64], ptr %17, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store [2 x i64] %20, ptr %21, align 8
// CHECK-NEXT:   %22 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_github.com/goplus/llgo/cl/_testrt/typed.A", ptr undef }, ptr %21, 1
// CHECK-NEXT:   %23 = alloca [2 x i64], align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %23, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %24 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 0
// CHECK-NEXT:   %25 = icmp eq ptr %24, @"_llgo_github.com/goplus/llgo/cl/_testrt/typed.A"
// CHECK-NEXT:   br i1 %25, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %22, 1
// CHECK-NEXT:   %27 = load [2 x i64], ptr %26, align 8
// CHECK-NEXT:   %28 = insertvalue { [2 x i64], i1 } undef, [2 x i64] %27, 0
// CHECK-NEXT:   %29 = insertvalue { [2 x i64], i1 } %28, i1 true, 1
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_6
// CHECK-NEXT:   %30 = phi { [2 x i64], i1 } [ %29, %_llgo_6 ], [ zeroinitializer, %_llgo_7 ]
// CHECK-NEXT:   %31 = extractvalue { [2 x i64], i1 } %30, 0
// CHECK-NEXT:   store [2 x i64] %31, ptr %23, align 8
// CHECK-NEXT:   %32 = extractvalue { [2 x i64], i1 } %30, 1
// CHECK-NEXT:   %33 = getelementptr inbounds i64, ptr %23, i64 0
// CHECK-NEXT:   %34 = load i64, ptr %33, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds i64, ptr %23, i64 1
// CHECK-NEXT:   %36 = load i64, ptr %35, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %34)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %36)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %32)
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
