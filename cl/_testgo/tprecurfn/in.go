// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecurfn.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testgo/tprecurfn.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testgo/tprecurfn.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type My[T any] struct {
	fn   func(n T)
	next *My[T]
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecurfn.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testgo/tprecurfn.My[int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %4 = icmp eq ptr %3, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %4)
// CHECK-NEXT:   %5 = getelementptr inbounds %"{{.*}}/cl/_testgo/tprecurfn.My[int]", ptr %3, i32 0, i32 0
// CHECK-NEXT:   store { ptr, ptr } { ptr @"__llgo_stub.{{.*}}/cl/_testgo/tprecurfn.main$1", ptr null }, ptr %5, align 8
// CHECK-NEXT:   store ptr %3, ptr %2, align 8
// CHECK-NEXT:   %6 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %6)
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/tprecurfn.My[int]", ptr %0, i32 0, i32 1
// CHECK-NEXT:   %8 = load ptr, ptr %7, align 8
// CHECK-NEXT:   %9 = icmp eq ptr %8, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %9)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/tprecurfn.My[int]", ptr %8, i32 0, i32 0
// CHECK-NEXT:   %11 = load { ptr, ptr }, ptr %10, align 8
// CHECK-NEXT:   %12 = extractvalue { ptr, ptr } %11, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %11, 0
// CHECK-NEXT:   call void %13(ptr %12, i64 100)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {

	// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/tprecurfn.main$1"(i64 %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	m := &My[int]{next: &My[int]{fn: func(n int) { println(n) }}}
	m.next.fn(100)
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testgo/tprecurfn.main$1"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testgo/tprecurfn.main$1"(i64 %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
