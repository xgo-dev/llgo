// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}tpindex.main"(){{.*}} {
func main() {
	// CHECK: call ptr @"{{.*}}AllocZ"(i64 40)
	// CHECK: store i64 1, ptr %1, align 8
	// CHECK: store i64 3, ptr %2, align 8
	// CHECK: store i64 5, ptr %3, align 8
	// CHECK: store i64 2, ptr %4, align 8
	// CHECK: store i64 4, ptr %5, align 8
	// CHECK: call i64 @"{{.*}}tpindex.index[int]"(%"{{.*}}Slice" %8, i64 3)
	// CHECK: call void @"{{.*}}PrintInt"(i64 %9)
	// CHECK: call i64 @"{{.*}}tpindex.index[int]"(%"{{.*}}Slice" %8, i64 6)
	// CHECK: call void @"{{.*}}PrintInt"(i64 %10)
	// CHECK: ret void
	s := []int{1, 3, 5, 2, 4}
	println(index(s, 3))
	println(index(s, 6))
}

// The index function returns the index of the first occurrence of v in s,
// or -1 if not present.
// CHECK-LABEL: define linkonce i64 @"{{.*}}tpindex.index[int]"(%"{{.*}}Slice" %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:
// CHECK-NEXT:   %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %4 = add i64 %3, 1
// CHECK-NEXT:   %5 = icmp slt i64 %4, %2
// CHECK-NEXT:   br i1 %5, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}Slice" %0, 0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}Slice" %0, 1
// CHECK-NEXT:   %8 = icmp slt i64 %4, 0
// CHECK-NEXT:   %9 = icmp uge i64 %4, %7
// CHECK-NEXT:   %10 = or i1 %9, %8
// CHECK-NEXT:   call void @"{{.*}}AssertIndexRange"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds i64, ptr %6, i64 %4
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   %13 = icmp eq i64 %1, %12
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:
// CHECK-NEXT:   ret i64 -1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:
// CHECK-NEXT:   ret i64 %4
func index[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}
