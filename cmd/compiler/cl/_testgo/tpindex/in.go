// LITTEST
package main

func main() {
	// CHECK-LABEL: define void @main.main(){{.*}} {
	// CHECK: [[INDEX3:%[0-9]+]] = call i64 @"main.index[int]"(%"{{.*}}Slice" %{{[0-9]+}}, i64 3)
	// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[INDEX3]])
	// CHECK: [[INDEX6:%[0-9]+]] = call i64 @"main.index[int]"(%"{{.*}}Slice" %{{[0-9]+}}, i64 6)
	// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[INDEX6]])
	s := []int{1, 3, 5, 2, 4}
	println(index(s, 3))
	println(index(s, 6))
}

// The index function returns the index of the first occurrence of v in s,
// or -1 if not present.
// CHECK-LABEL: define linkonce i64 @"main.index[int]"(%"{{.*}}Slice" %0, i64 %1){{.*}} {
// CHECK: [[INDEX_POS:%[0-9]+]] = add i64 %{{[0-9]+}}, 1
// CHECK: [[INDEX_DATA:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 0
// CHECK: [[INDEX_LEN:%[0-9]+]] = extractvalue %"{{.*}}Slice" %0, 1
// CHECK: br i1 %{{[0-9]+}}, label %{{_llgo_[0-9]+}}, label %{{_llgo_[0-9]+}}
// CHECK: ret i64 -1
// CHECK: ret i64 [[INDEX_POS]]
// CHECK: call void @"{{.*}}PanicIndex"(i64 [[INDEX_POS]], i64 [[INDEX_LEN]])
// CHECK-NEXT: br label %{{_llgo_[0-9]+}}
// CHECK: [[INDEX_ITEM_PTR:%[0-9]+]] = getelementptr inbounds i64, ptr [[INDEX_DATA]], i64 [[INDEX_POS]]
// CHECK-NEXT: [[INDEX_ITEM:%[0-9]+]] = load i64, ptr [[INDEX_ITEM_PTR]]
// CHECK-NEXT: [[INDEX_MATCH:%[0-9]+]] = icmp eq i64 %1, [[INDEX_ITEM]]
func index[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}
