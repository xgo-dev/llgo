// LITTEST
package main

type IteratorG[T any] func(T) bool

type TreeG[T any] struct{}

func (*TreeG[T]) Ascend(iterator IteratorG[T]) {
	var zero T
	iterator(zero)
}

type Tree TreeG[int]

type Iterator IteratorG[int]

// CHECK-LABEL: define void @"main.(*Tree).Ascend"(ptr %0, %main.Iterator %1){{.*}} {
// CHECK: [[TREE_CODE:%[0-9]+]] = extractvalue %main.Iterator %1, 0
// CHECK-NEXT: [[TREE_GENERIC0:%[0-9]+]] = insertvalue %"main.IteratorG[int]" undef, ptr [[TREE_CODE]], 0
// CHECK-NEXT: [[TREE_ENV:%[0-9]+]] = extractvalue %main.Iterator %1, 1
// CHECK-NEXT: [[TREE_GENERIC:%[0-9]+]] = insertvalue %"main.IteratorG[int]" [[TREE_GENERIC0]], ptr [[TREE_ENV]], 1
// CHECK-NEXT: call void @"main.(*TreeG[int]).Ascend"(ptr %0, %"main.IteratorG[int]" [[TREE_GENERIC]])

func (t *Tree) Ascend(iterator Iterator) {
	(*TreeG[int])(t).Ascend((IteratorG[int])(iterator))
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK:   call void @"main.(*Tree).Ascend"(ptr @"__llgo.moduleZeroSizedAlloc$", %main.Iterator %{{[0-9]+}})

func main() {
	var got int
	tree := (*Tree)(new(TreeG[int]))
	// CHECK-LABEL: define i1 @"main.main$1"(ptr {{(nest|swiftself)}} %0, i64 %1){{.*}} {
	// CHECK: [[ITER_VALUE:%[0-9]+]] = add i64 %1, 1
	// CHECK: [[ITER_ENV:%[0-9]+]] = load { ptr }, ptr %0
	// CHECK-NEXT: [[ITER_GOT:%[0-9]+]] = extractvalue { ptr } [[ITER_ENV]], 0
	// CHECK-NEXT: store i64 [[ITER_VALUE]], ptr [[ITER_GOT]]
	// CHECK-NEXT: ret i1 false

	tree.Ascend(func(v int) bool {
		got = v + 1
		return false
	})
	if got != 1 {
		panic("bad Ascend result")
	}
	println("ok")
}

// CHECK-LABEL: define linkonce void @"main.(*TreeG[int]).Ascend"(ptr %0, %"main.IteratorG[int]" %1){{.*}} {
// CHECK: [[GENERIC_ENV:%[0-9]+]] = extractvalue %"main.IteratorG[int]" %1, 1
// CHECK-NEXT: [[GENERIC_CODE:%[0-9]+]] = extractvalue %"main.IteratorG[int]" %1, 0
// CHECK: call i1 %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[GENERIC_ENV]], i64 0)
