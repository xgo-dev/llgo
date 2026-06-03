// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/closureiface.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   store i64 200, ptr %0, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testrt/closureiface.main$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr undef }, ptr %4, 1
// CHECK-NEXT:   %6 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %5, 0
// CHECK-NEXT:   %7 = call i1 @"{{.*}}/runtime/internal/runtime.MatchesClosure"(ptr @"_llgo_closure$QIHBTaw1IFobr8yvWpq-2AJFm3xBNhdW_aNBicqUBGk", ptr %6)
// CHECK-NEXT:   br i1 %7, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{.*}}, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %10 = extractvalue { ptr, ptr } %18, 1
// CHECK-NEXT:   %11 = extractvalue { ptr, ptr } %18, 0
// CHECK-NEXT:   %12 = call i64 %11(ptr %10, i64 100)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %13 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %5, 1
// CHECK-NEXT:   %14 = load { ptr, ptr }, ptr %13, align 8
// CHECK-NEXT:   %15 = insertvalue { { ptr, ptr }, i1 } undef, { ptr, ptr } %14, 0
// CHECK-NEXT:   %16 = insertvalue { { ptr, ptr }, i1 } %15, i1 true, 1
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4, %_llgo_3
// CHECK-NEXT:   %17 = phi { { ptr, ptr }, i1 } [ %16, %_llgo_3 ], [ zeroinitializer, %_llgo_4 ]
// CHECK-NEXT:   %18 = extractvalue { { ptr, ptr }, i1 } %17, 0
// CHECK-NEXT:   %19 = extractvalue { { ptr, ptr }, i1 } %17, 1
// CHECK-NEXT:   br i1 %19, label %_llgo_2, label %_llgo_1
// CHECK-NEXT: }
func main() {
	var m int = 200
	// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/closureiface.main$1"(ptr %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %3 = extractvalue { ptr } %2, 0
	// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
	// CHECK-NEXT:   %5 = add i64 %4, %1
	// CHECK-NEXT:   ret i64 %5
	// CHECK-NEXT: }
	fn := func(n int) int {
		return m + n
	}
	var i any = fn
	f, ok := i.(func(int) int)
	if !ok {
		panic("error")
	}
	println(f(100))
}
