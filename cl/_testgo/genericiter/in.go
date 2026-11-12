// LITTEST darwin/arm64 linux/amd64
package main

type IteratorG[T any] func(T) bool

type TreeG[T any] struct{}

func (*TreeG[T]) Ascend(iterator IteratorG[T]) {
	var zero T
	iterator(zero)
}

type Tree TreeG[int]

type Iterator IteratorG[int]

// CHECK-LABEL: define void @"main.(*Tree).Ascend"(ptr %0, ptr %1){{.*}} {
// The named conversion preserves the descriptor and its captured identity.
// CHECK-NOT: AllocU
// CHECK: call void @"main.(*TreeG[int]).Ascend"(ptr %0, ptr %1)

func (t *Tree) Ascend(iterator Iterator) {
	(*TreeG[int])(t).Ascend((IteratorG[int])(iterator))
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK:   call void @"main.(*Tree).Ascend"(ptr @"__llgo.moduleZeroSizedAlloc$", ptr %{{[0-9]+}})

func main() {
	var got int
	tree := (*Tree)(new(TreeG[int]))
	// ARM64-LABEL: define hidden i1 @"main.main$1"(ptr swiftself %0, i64 %1){{.*}} {
	// AMD64-LABEL: define hidden i1 @"main.main$1"(ptr nest %0, i64 %1){{.*}} {
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

// CHECK-LABEL: define linkonce void @"main.(*TreeG[int]).Ascend"(ptr %0, ptr %1){{.*}} {
// CHECK: [[GENERIC_ENV_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %1
// CHECK-NEXT: [[GENERIC_ENV_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[GENERIC_ENV_SAFE]], i32 0, i32 1
// CHECK-NEXT: [[GENERIC_ENV:%[0-9]+]] = load ptr, ptr [[GENERIC_ENV_FIELD]]
// CHECK: [[GENERIC_CODE_SAFE:%[0-9]+]] = select i1 %{{[0-9]+}}, ptr @_llgo_funcval_nil, ptr %1
// CHECK-NEXT: [[GENERIC_CODE_FIELD:%[0-9]+]] = getelementptr inbounds nuw { ptr, ptr }, ptr [[GENERIC_CODE_SAFE]], i32 0, i32 0
// CHECK-NEXT: [[GENERIC_CODE:%[0-9]+]] = load ptr, ptr [[GENERIC_CODE_FIELD]]
// ARM64: call i1 %__llgo_funcval_code(ptr swiftself [[GENERIC_ENV]], i64 0)
// AMD64: call i1 %__llgo_funcval_code(ptr nest [[GENERIC_ENV]], i64 0)
