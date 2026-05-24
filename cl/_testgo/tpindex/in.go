// LITTEST
package main

func main() {
	s := []int{1, 3, 5, 2, 4}
	println(index(s, 3))
	println(index(s, 6))
}

// The index function returns the index of the first occurrence of v in s,
// or -1 if not present.
func index[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpindex.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tpindex.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tpindex.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tpindex.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   store i64 3, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   store i64 5, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %0, i64 3
// CHECK-NEXT:   store i64 2, ptr %4, align 8
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %0, i64 4
// CHECK-NEXT:   store i64 4, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 5, 1
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %7, i64 5, 2
// CHECK-NEXT:   %9 = call i64 @"{{.*}}/cl/_testgo/tpindex.index[int]"(%"{{.*}}/runtime/internal/runtime.Slice" %8, i64 3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %10 = call i64 @"{{.*}}/cl/_testgo/tpindex.index[int]"(%"{{.*}}/runtime/internal/runtime.Slice" %8, i64 6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i64 @"{{.*}}/cl/_testgo/tpindex.index[int]"(%"{{.*}}/runtime/internal/runtime.Slice" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %4 = add i64 %3, 1
// CHECK-NEXT:   %5 = icmp slt i64 %4, %2
// CHECK-NEXT:   br i1 %5, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %8 = icmp slt i64 %4, 0
// CHECK-NEXT:   %9 = icmp uge i64 %4, %7
// CHECK-NEXT:   %10 = or i1 %9, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %10, i64 %4, i1 true, i64 %7)
// CHECK-NEXT:   %11 = getelementptr inbounds i64, ptr %6, i64 %4
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   %13 = icmp eq i64 %1, %12
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret i64 -1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }
