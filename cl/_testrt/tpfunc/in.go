// LITTEST
package main

import (
	"unsafe"
)

type Func func(*int)

//llgo:cdecl
type CFunc func(*int)

//llgo:cdecl
type Callback[T any] func(*T)

// All three values occupy one word; the Go value points to its descriptor,
// while the native C values store their entry pointers directly.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)

// All three source literals still dereference and print the same *int argument.
// CHECK-LABEL: define void @"main.main$1"(ptr %0){{.*}} {
// CHECK: [[GO_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[GO_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[GO_VALUE:%[0-9]+]] = load i64, ptr %0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[GO_VALUE]])

// CHECK-LABEL: define void @"main.main$2"(ptr %0){{.*}} {
// CHECK: [[C_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[C_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[C_VALUE:%[0-9]+]] = load i64, ptr %0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[C_VALUE]])

// CHECK-LABEL: define void @"main.main$3"(ptr %0){{.*}} {
// CHECK: [[GENERIC_C_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: br i1 [[GENERIC_C_NIL]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 true)
// CHECK: [[GENERIC_C_VALUE:%[0-9]+]] = load i64, ptr %0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[GENERIC_C_VALUE]])

func main() {
	var fn1 Func = func(v *int) {
		println(*v)
	}

	var fn2 CFunc = func(v *int) {
		println(*v)
	}

	var fn3 Callback[int] = func(v *int) {
		println(*v)
	}
	println(unsafe.Sizeof(fn1), unsafe.Sizeof(fn2), unsafe.Sizeof(fn3))
}
