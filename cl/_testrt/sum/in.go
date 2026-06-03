// LITTEST
package main

import (
	"github.com/goplus/lib/c"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/sum.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %1 = getelementptr inbounds i64, ptr %0, i64 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds i64, ptr %0, i64 1
// CHECK-NEXT:   store i64 2, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds i64, ptr %0, i64 2
// CHECK-NEXT:   store i64 3, ptr %3, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds i64, ptr %0, i64 3
// CHECK-NEXT:   store i64 4, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 4, 1
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, i64 4, 2
// CHECK-NEXT:   %8 = call i64 @"{{.*}}/cl/_testrt/sum.sum"(%"{{.*}}/runtime/internal/runtime.Slice" %7)
// CHECK-NEXT:   %9 = call i32 (ptr, ...) @printf(ptr @0, i64 %8)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	c.Printf(c.Str("Hello %d\n"), sum(1, 2, 3, 4))
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/sum.sum"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_2, %_llgo_0
// CHECK-NEXT:   %2 = phi i64 [ 0, %_llgo_0 ], [ %13, %_llgo_2 ]
// CHECK-NEXT:   %3 = phi i64 [ -1, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %4 = add i64 %3, 1
// CHECK-NEXT:   %5 = icmp slt i64 %4, %1
// CHECK-NEXT:   br i1 %5, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %8 = icmp slt i64 %4, 0
// CHECK-NEXT:   %9 = icmp uge i64 %4, %7
// CHECK-NEXT:   %10 = or i1 %9, %8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %10)
// CHECK-NEXT:   %11 = getelementptr inbounds i64, ptr %6, i64 %4
// CHECK-NEXT:   %12 = load i64, ptr %11, align 8
// CHECK-NEXT:   %13 = add i64 %2, %12
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }
func sum(args ...int) (ret int) {
	for _, v := range args {
		ret += v
	}
	return
}
