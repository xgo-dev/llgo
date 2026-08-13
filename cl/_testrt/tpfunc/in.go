// LITTEST
package main

import (
	"unsafe"
)

type Func func(*int)

//llgo:type C
type CFunc func(*int)

//llgo:type C
type Callback[T any] func(*T)

// The Go function value is two words, while both llgo:type C forms are one word.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 16)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 8)

// All three source literals still dereference and print the same *int argument.
// CHECK-LABEL: define void @"main.main$1"(ptr %0){{.*}} {
// CHECK: [[GO_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 [[GO_NIL]])
// CHECK: [[GO_VALUE:%[0-9]+]] = load i64, ptr %0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[GO_VALUE]])

// CHECK-LABEL: define void @"main.main$2"(ptr %0){{.*}} {
// CHECK: [[C_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 [[C_NIL]])
// CHECK: [[C_VALUE:%[0-9]+]] = load i64, ptr %0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[C_VALUE]])

// CHECK-LABEL: define void @"main.main$3"(ptr %0){{.*}} {
// CHECK: [[GENERIC_C_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 [[GENERIC_C_NIL]])
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
