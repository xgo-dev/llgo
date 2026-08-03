// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
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
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 4, i64 %10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, 0
// CHECK-NEXT:   %12 = load [4 x i8], ptr %0, align 1
// CHECK-NEXT:   %13 = icmp eq ptr %11, null
// CHECK-NEXT:   br i1 %13, label %14, label %15
// CHECK-EMPTY:
// CHECK-NEXT: 14:                                               ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 15:                                               ; preds = %_llgo_2
// CHECK-NEXT:   %16 = load [4 x i8], ptr %11, align 1
// CHECK-NEXT:   %17 = extractvalue [4 x i8] %12, 0
// CHECK-NEXT:   %18 = extractvalue [4 x i8] %16, 0
// CHECK-NEXT:   %19 = icmp eq i8 %17, %18
// CHECK-NEXT:   %20 = and i1 true, %19
// CHECK-NEXT:   %21 = extractvalue [4 x i8] %12, 1
// CHECK-NEXT:   %22 = extractvalue [4 x i8] %16, 1
// CHECK-NEXT:   %23 = icmp eq i8 %21, %22
// CHECK-NEXT:   %24 = and i1 %20, %23
// CHECK-NEXT:   %25 = extractvalue [4 x i8] %12, 2
// CHECK-NEXT:   %26 = extractvalue [4 x i8] %16, 2
// CHECK-NEXT:   %27 = icmp eq i8 %25, %26
// CHECK-NEXT:   %28 = and i1 %24, %27
// CHECK-NEXT:   %29 = extractvalue [4 x i8] %12, 3
// CHECK-NEXT:   %30 = extractvalue [4 x i8] %16, 3
// CHECK-NEXT:   %31 = icmp eq i8 %29, %30
// CHECK-NEXT:   %32 = and i1 %28, %31
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %33, i64 4, 1
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, i64 4, 2
// CHECK-NEXT:   %36 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   %37 = icmp slt i64 %36, 2
// CHECK-NEXT:   br i1 %37, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %15
// CHECK-NEXT:   %38 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicSliceConvert"(i64 2, i64 %38)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3, %15
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, 0
// CHECK-NEXT:   %40 = icmp eq ptr %39, null
// CHECK-NEXT:   br i1 %40, label %41, label %42
// CHECK-EMPTY:
// CHECK-NEXT: 41:                                               ; preds = %_llgo_4
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: 42:                                               ; preds = %_llgo_4
// CHECK-NEXT:   %43 = load [2 x i8], ptr %39, align 1
// CHECK-NEXT:   %44 = alloca [2 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %44, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %45 = getelementptr inbounds i8, ptr %44, i64 0
// CHECK-NEXT:   %46 = getelementptr inbounds i8, ptr %44, i64 1
// CHECK-NEXT:   store i8 1, ptr %45, align 1
// CHECK-NEXT:   store i8 2, ptr %46, align 1
// CHECK-NEXT:   %47 = load [2 x i8], ptr %44, align 1
// CHECK-NEXT:   %48 = extractvalue [2 x i8] %43, 0
// CHECK-NEXT:   %49 = extractvalue [2 x i8] %47, 0
// CHECK-NEXT:   %50 = icmp eq i8 %48, %49
// CHECK-NEXT:   %51 = and i1 true, %50
// CHECK-NEXT:   %52 = extractvalue [2 x i8] %43, 1
// CHECK-NEXT:   %53 = extractvalue [2 x i8] %47, 1
// CHECK-NEXT:   %54 = icmp eq i8 %52, %53
// CHECK-NEXT:   %55 = and i1 %51, %54
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %55)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	array := [4]byte{1, 2, 3, 4}
	ptr := (*[4]byte)(array[:])
	println(array == *ptr)
	println(*(*[2]byte)(array[:]) == [2]byte{1, 2})
}
