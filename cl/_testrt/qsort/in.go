// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

//go:linkname qsort C.qsort
func qsort(base c.Pointer, count, elem uintptr, compar func(a, b c.Pointer) c.Int)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/qsort.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %0, i64 3
// CHECK-NEXT:   %5 = getelementptr inbounds i64, ptr %0, i64 4
// CHECK-NEXT:   store i64 100, ptr %1, align 8
// CHECK-NEXT:   store i64 8, ptr %2, align 8
// CHECK-NEXT:   store i64 23, ptr %3, align 8
// CHECK-NEXT:   store i64 2, ptr %4, align 8
// CHECK-NEXT:   store i64 7, ptr %5, align 8
// CHECK-NEXT:   %6 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   call void @qsort(ptr %6, i64 5, i64 8, ptr @"{{.*}}/cl/_testrt/qsort.main$1")
// CHECK-NEXT:   %7 = load [5 x i64], ptr %0, align 8
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %8 = phi i64 [ -1, %_llgo_0 ], [ %9, %_llgo_2 ]
// CHECK-NEXT:   %9 = add i64 %8, 1
// CHECK-NEXT:   %10 = icmp slt i64 %9, 5
// CHECK-NEXT:   br i1 %10, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %11 = icmp slt i64 %9, 0
// CHECK-NEXT:   %12 = icmp uge i64 %9, 5
// CHECK-NEXT:   %13 = or i1 %12, %11
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %13)
// CHECK-NEXT:   %14 = getelementptr inbounds i64, ptr %0, i64 %9
// CHECK-NEXT:   %15 = load i64, ptr %14, align 8
// CHECK-NEXT:   %16 = call i32 (ptr, ...) @printf(ptr @0, i64 %15)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	a := [...]int{100, 8, 23, 2, 7}
	qsort(c.Pointer(&a[0]), 5, unsafe.Sizeof(0), func(a, b c.Pointer) c.Int {
		return c.Int(*(*int)(a) - *(*int)(b))
	})
	// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/qsort.main$1"(ptr %0, ptr %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   %3 = load i64, ptr %1, align 8
	// CHECK-NEXT:   %4 = sub i64 %2, %3
	// CHECK-NEXT:   %5 = trunc i64 %4 to i32
	// CHECK-NEXT:   ret i32 %5
	// CHECK-NEXT: }
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v)
	}
}
