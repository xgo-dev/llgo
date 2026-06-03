// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibc/allocacstrs.main"(){{.*}} {
func main() {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 48)
	// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 0
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 1 }, ptr %1, align 8
	// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 1
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 1 }, ptr %2, align 8
	// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %0, i64 2
	// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 1 }, ptr %3, align 8
	// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
	// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4, i64 3, 1
	// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5, i64 3, 2
	// CHECK-NEXT:   %7 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 1
	// CHECK-NEXT:   %8 = add i64 %7, 1
	// CHECK-NEXT:   %9 = alloca ptr, i64 %8, align 8
	// CHECK-NEXT:   br label %_llgo_4
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3, %_llgo_6
	// CHECK-NEXT:   %10 = phi i64 [ 0, %_llgo_6 ], [ %15, %_llgo_3 ]
	// CHECK-NEXT:   %11 = getelementptr ptr, ptr %9, i64 %10
	// CHECK-NEXT:   %12 = load ptr, ptr %11, align 8
	// CHECK-NEXT:   %13 = icmp eq ptr %12, null
	// CHECK-NEXT:   br i1 %13, label %_llgo_2, label %_llgo_3
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
	// CHECK-NEXT:   ret void
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
	// CHECK-NEXT:   %14 = call i32 (ptr, ...) @printf(ptr @3, ptr %12)
	// CHECK-NEXT:   %15 = add i64 %10, 1
	// CHECK-NEXT:   br label %_llgo_1
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_5, %_llgo_0
	// CHECK-NEXT:   %16 = phi i64 [ 0, %_llgo_0 ], [ %30, %_llgo_5 ]
	// CHECK-NEXT:   %17 = icmp slt i64 %16, %7
	// CHECK-NEXT:   br i1 %17, label %_llgo_5, label %_llgo_6
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
	// CHECK-NEXT:   %18 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 0
	// CHECK-NEXT:   %19 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %6, 1
	// CHECK-NEXT:   %20 = icmp slt i64 %16, 0
	// CHECK-NEXT:   %21 = icmp uge i64 %16, %19
	// CHECK-NEXT:   %22 = or i1 %21, %20
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertIndexRange"(i1 %22)
	// CHECK-NEXT:   %23 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String", ptr %18, i64 %16
	// CHECK-NEXT:   %24 = load %"{{.*}}/runtime/internal/runtime.String", ptr %23, align 8
	// CHECK-NEXT:   %25 = getelementptr ptr, ptr %9, i64 %16
	// CHECK-NEXT:   %26 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %24, 1
	// CHECK-NEXT:   %27 = add i64 %26, 1
	// CHECK-NEXT:   %28 = alloca i8, i64 %27, align 1
	// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.CStrCopy"(ptr %28, %"{{.*}}/runtime/internal/runtime.String" %24)
	// CHECK-NEXT:   store ptr %29, ptr %25, align 8
	// CHECK-NEXT:   %30 = add i64 %16, 1
	// CHECK-NEXT:   br label %_llgo_4
	// CHECK-EMPTY:
	// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_4
	// CHECK-NEXT:   %31 = getelementptr ptr, ptr %9, i64 %7
	// CHECK-NEXT:   store ptr null, ptr %31, align 8
	// CHECK-NEXT:   br label %_llgo_1
	// CHECK-NEXT: }
	cstrs := c.AllocaCStrs([]string{"a", "b", "c"}, true)
	n := 0
	for {
		cstr := *c.Advance(cstrs, n)
		if cstr == nil {
			break
		}
		c.Printf(c.Str("%s\n"), cstr)
		n++
	}
}
