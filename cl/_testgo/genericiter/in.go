// LITTEST
package main

// CHECK: {{^}}@0 = private unnamed_addr constant [17 x i8] c"bad Ascend result", align 1{{$}}
// CHECK: {{^}}@2 = private unnamed_addr constant [2 x i8] c"ok", align 1{{$}}

type IteratorG[T any] func(T) bool

type TreeG[T any] struct{}

func (*TreeG[T]) Ascend(iterator IteratorG[T]) {
	var zero T
	iterator(zero)
}

type Tree TreeG[int]

type Iterator IteratorG[int]

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/genericiter.(*Tree).Ascend"(ptr %0, %"{{.*}}/cl/_testgo/genericiter.Iterator" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testgo/genericiter.Iterator" %1, 0
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" undef, ptr %2, 0
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/cl/_testgo/genericiter.Iterator" %1, 1
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" %3, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/genericiter.(*TreeG[int]).Ascend"(ptr %0, %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func (t *Tree) Ascend(iterator Iterator) {
	(*TreeG[int])(t).Ascend((IteratorG[int])(iterator))
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/genericiter.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testgo/genericiter.main$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = alloca %"{{.*}}/cl/_testgo/genericiter.Iterator", align 8
// CHECK-NEXT:   store { ptr, ptr } %3, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/cl/_testgo/genericiter.Iterator", ptr %4, align 8
// CHECK-NEXT:   call void @"{{.*}}/cl/_testgo/genericiter.(*Tree).Ascend"(ptr @"__llgo.moduleZeroSizedAlloc$", %"{{.*}}/cl/_testgo/genericiter.Iterator" %5)
// CHECK-NEXT:   %6 = load i64, ptr %0, align 8
// CHECK-NEXT:   %7 = icmp ne i64 %6, 1
// CHECK-NEXT:   br i1 %7, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 17 }, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %8, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @2, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	var got int
	tree := (*Tree)(new(TreeG[int]))
	// CHECK-LABEL: define i1 @"{{.*}}/cl/_testgo/genericiter.main$1"(ptr %0, i64 %1){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %2 = add i64 %1, 1
	// CHECK-NEXT:   %3 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %4 = extractvalue { ptr } %3, 0
	// CHECK-NEXT:   store i64 %2, ptr %4, align 8
	// CHECK-NEXT:   ret i1 false
	// CHECK-NEXT: }

	tree.Ascend(func(v int) bool {
		got = v + 1
		return false
	})
	if got != 1 {
		panic("bad Ascend result")
	}
	println("ok")
}

// CHECK-LABEL: define linkonce void @"{{.*}}/cl/_testgo/genericiter.(*TreeG[int]).Ascend"(ptr %0, %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = extractvalue %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" %1, 1
// CHECK-NEXT:   %3 = extractvalue %"{{.*}}/cl/_testgo/genericiter.IteratorG[int]" %1, 0
// CHECK-NEXT:   %4 = call i1 %3(ptr %2, i64 0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
