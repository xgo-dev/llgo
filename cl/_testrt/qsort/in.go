// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//go:linkname qsort C.qsort
func qsort(base c.Pointer, count, elem uintptr, compar func(a, b c.Pointer) c.Int)

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[ARRAY:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 40)
// CHECK-NEXT: [[E0:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 0
// CHECK-NEXT: [[E1:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 1
// CHECK-NEXT: [[E2:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 2
// CHECK-NEXT: [[E3:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 3
// CHECK-NEXT: [[E4:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 4
// CHECK-NEXT: store i64 100, ptr [[E0]]
// CHECK-NEXT: store i64 8, ptr [[E1]]
// CHECK-NEXT: store i64 23, ptr [[E2]]
// CHECK-NEXT: store i64 2, ptr [[E3]]
// CHECK-NEXT: store i64 7, ptr [[E4]]
// CHECK: [[BASE:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 0
// CHECK-NEXT: call void @qsort(ptr [[BASE]], i64 5, i64 8, ptr @"main.main$1")
// CHECK-NEXT: load [5 x i64], ptr [[ARRAY]]
// CHECK: call void @"{{.*}}CheckIndexRange"(i1 {{.*}}, i64 [[QSORT_INDEX:%[0-9]+]], i1 true, i64 5)
// CHECK-NEXT: [[SORTED_ADDR:%[0-9]+]] = getelementptr inbounds i64, ptr [[ARRAY]], i64 [[QSORT_INDEX]]
// CHECK-NEXT: [[SORTED_VALUE:%[0-9]+]] = load i64, ptr [[SORTED_ADDR]]
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i64 [[SORTED_VALUE]])
func main() {
	a := [...]int{100, 8, 23, 2, 7}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), func(a, b c.Pointer) c.Int {
		// CHECK-LABEL: define i32 @"main.main$1"(ptr %0, ptr %1){{.*}} {
		// CHECK: [[QSORT_A:%[0-9]+]] = load i64, ptr %0
		// CHECK-NEXT: [[QSORT_B:%[0-9]+]] = load i64, ptr %1
		// CHECK-NEXT: [[QSORT_DIFF:%[0-9]+]] = sub i64 [[QSORT_A]], [[QSORT_B]]
		// CHECK-NEXT: [[QSORT_RESULT:%[0-9]+]] = trunc i64 [[QSORT_DIFF]] to i32
		// CHECK-NEXT: ret i32 [[QSORT_RESULT]]
		return c.Int(*(*int)(a) - *(*int)(b))
	})
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}
