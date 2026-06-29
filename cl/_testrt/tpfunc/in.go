// LITTEST
package main

import (
	"unsafe"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpfunc.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/tpfunc.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/tpfunc.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

type Func func(*int)

//llgo:type C
type CFunc func(*int)

//llgo:type C
type Callback[T any] func(*T)

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpfunc.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func main() {
	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpfunc.main$1"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = icmp eq ptr %0, null
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	var fn1 Func = func(v *int) {
		println(*v)
	}

	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpfunc.main$2"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = icmp eq ptr %0, null
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	var fn2 CFunc = func(v *int) {
		println(*v)
	}

	// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/tpfunc.main$3"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = icmp eq ptr %0, null
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %1)
	// CHECK-NEXT:   %2 = load i64, ptr %0, align 8
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
	// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	var fn3 Callback[int] = func(v *int) {
		println(*v)
	}
	println(unsafe.Sizeof(fn1), unsafe.Sizeof(fn2), unsafe.Sizeof(fn3))
}

// CHECK-LABEL: define linkonce void @"__llgo_stub.{{.*}}/cl/_testrt/tpfunc.main$1"(ptr %0, ptr %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   tail call void @"{{.*}}/cl/_testrt/tpfunc.main$1"(ptr %1)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
