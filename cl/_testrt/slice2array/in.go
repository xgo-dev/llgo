// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/slice2array.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %1 = getelementptr inbounds i8, ptr %0, i64 0
// CHECK-NEXT:   %2 = getelementptr inbounds i8, ptr %0, i64 1
// CHECK-NEXT:   %3 = getelementptr inbounds i8, ptr %0, i64 2
// CHECK-NEXT:   %4 = getelementptr inbounds i8, ptr %0, i64 3
// CHECK-NEXT:   store i8 1, ptr %1, align 1
// CHECK-NEXT:   store i8 2, ptr %2, align 1
// CHECK-NEXT:   store i8 3, ptr %3, align 1
// CHECK-NEXT:   store i8 4, ptr %4, align 1
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 4, 1
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 4, 2
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   %9 = icmp slt i64 %8, 4
// CHECK-NEXT:   br i1 %9, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 %10, i64 4)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %12 = load [4 x i8], ptr %0, align 1
// CHECK-NEXT:   %13 = load [4 x i8], ptr %11, align 1
// CHECK-NEXT:   %14 = extractvalue [4 x i8] %12, 0
// CHECK-NEXT:   %15 = extractvalue [4 x i8] %13, 0
// CHECK-NEXT:   %16 = icmp eq i8 %14, %15
// CHECK-NEXT:   %17 = and i1 true, %16
// CHECK-NEXT:   %18 = extractvalue [4 x i8] %12, 1
// CHECK-NEXT:   %19 = extractvalue [4 x i8] %13, 1
// CHECK-NEXT:   %20 = icmp eq i8 %18, %19
// CHECK-NEXT:   %21 = and i1 %17, %20
// CHECK-NEXT:   %22 = extractvalue [4 x i8] %12, 2
// CHECK-NEXT:   %23 = extractvalue [4 x i8] %13, 2
// CHECK-NEXT:   %24 = icmp eq i8 %22, %23
// CHECK-NEXT:   %25 = and i1 %21, %24
// CHECK-NEXT:   %26 = extractvalue [4 x i8] %12, 3
// CHECK-NEXT:   %27 = extractvalue [4 x i8] %13, 3
// CHECK-NEXT:   %28 = icmp eq i8 %26, %27
// CHECK-NEXT:   %29 = and i1 %25, %28
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %29)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %31 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %30, i64 4, 1
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %31, i64 4, 2
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, 1
// CHECK-NEXT:   %34 = icmp slt i64 %33, 2
// CHECK-NEXT:   br i1 %34, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %35 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 %35, i64 2)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3, %_llgo_2
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %32, 0
// CHECK-NEXT:   %37 = load [2 x i8], ptr %36, align 1
// CHECK-NEXT:   %38 = alloca [2 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset(ptr %38, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %39 = getelementptr inbounds i8, ptr %38, i64 0
// CHECK-NEXT:   %40 = getelementptr inbounds i8, ptr %38, i64 1
// CHECK-NEXT:   store i8 1, ptr %39, align 1
// CHECK-NEXT:   store i8 2, ptr %40, align 1
// CHECK-NEXT:   %41 = load [2 x i8], ptr %38, align 1
// CHECK-NEXT:   %42 = extractvalue [2 x i8] %37, 0
// CHECK-NEXT:   %43 = extractvalue [2 x i8] %41, 0
// CHECK-NEXT:   %44 = icmp eq i8 %42, %43
// CHECK-NEXT:   %45 = and i1 true, %44
// CHECK-NEXT:   %46 = extractvalue [2 x i8] %37, 1
// CHECK-NEXT:   %47 = extractvalue [2 x i8] %41, 1
// CHECK-NEXT:   %48 = icmp eq i8 %46, %47
// CHECK-NEXT:   %49 = and i1 %45, %48
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %49)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	array := [4]byte{1, 2, 3, 4}
	ptr := (*[4]byte)(array[:])
	println(array == *ptr)
	println(*(*[2]byte)(array[:]) == [2]byte{1, 2})
}
