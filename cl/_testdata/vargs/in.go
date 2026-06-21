// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK: @0 = private unnamed_addr constant [3 x i8] c"int", align 1
// CHECK: @1 = private unnamed_addr constant [4 x i8] c"%d\0A\00", align 1

func main() {
	test(1, 2, 3)
}

func test(a ...any) {
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v.(int))
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/vargs.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/vargs.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/vargs.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/vargs.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 48)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %0, i64 0
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %3, ptr %1, align 8
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %0, i64 1
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 2, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %5, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %6, ptr %4, align 8
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %0, i64 2
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 3, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %8, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %9, ptr %7, align 8
// CHECK-NEXT:   %10 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %0, 0
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %10, i64 3, 1
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %11, i64 3, 2
// CHECK-NEXT:   call void @"{{.*}}/cl/_testdata/vargs.test"(%"{{.*}}/runtime/internal/runtime.Slice" %12)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/vargs.test"(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %2 = phi i64 [ -1, %_llgo_0 ], [ %3, %_llgo_4 ]
// CHECK-NEXT:   %3 = add i64 %2, 1
// CHECK-NEXT:   %4 = icmp slt i64 %3, %1
// CHECK-NEXT:   br i1 %4, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT:   %7 = icmp slt i64 %3, 0
// CHECK-NEXT:   %8 = icmp uge i64 %3, %6
// CHECK-NEXT:   %9 = or i1 %8, %7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %9, i64 %3, i1 true, i64 %6)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %5, i64 %3
// CHECK-NEXT:   %11 = load %"{{.*}}/runtime/internal/runtime.eface", ptr %10, align 8
// CHECK-NEXT:   %12 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %11, 0
// CHECK-NEXT:   %13 = icmp eq ptr %12, @_llgo_int
// CHECK-NEXT:   br i1 %13, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %11, 1
// CHECK-NEXT:   %15 = load i64, ptr %14, align 8
// CHECK-NEXT:   %16 = call i32 (ptr, ...) @printf(ptr @1, i64 %15)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %12, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
