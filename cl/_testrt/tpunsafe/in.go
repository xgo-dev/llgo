// LITTEST
// Scope: common
package main

import (
	"unsafe"
)

type N[T any] struct {
	n1 T
	n2 T
}

type M[T any] struct {
	m0 T
	m1 int32
	m2 N[T]
}

type packedLayout struct {
	a byte
	b uint8
	c uint16
	d byte
	e [8]int8
}

// CHECK-LABEL: define i64 @main.layoutSize(){{.*}} {
// CHECK: ret i64 14
func layoutSize() uintptr {
	return unsafe.Sizeof(packedLayout{})
}

func main() {
	m1 := M[bool]{}
	m1.check(1, 8, 1)
	m2 := M[int64]{}
	m2.check(8, 16, 8)
}

// Each instantiation folds Alignof/Offsetof to its concrete layout while still
// addressing the instantiated fields used by the expressions.
func (m *M[T]) check(align, offset1, offset2 uintptr) {
	if v := unsafe.Alignof(m.m2); v != align {
		println("have", v, "want", align)
		panic("unsafe.Alignof error")
	}
	if v := unsafe.Offsetof(m.m2); v != offset1 {
		println("have", v, "want", offset1)
		panic("unsafe.Offsetof error")
	}
	if v := unsafe.Offsetof(m.m2.n2); v != offset2 {
		println("have", v, "want", offset2)
		panic("unsafe.Offsetof error")
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[BOOL:[0-9]+]] = call noalias dereferenceable(12) ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK: call void @"main.(*M[bool]).check"(ptr %[[BOOL]], i64 1, i64 8, i64 1)
// CHECK: %[[INT64:[0-9]+]] = call noalias dereferenceable(32) ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK: call void @"main.(*M[int64]).check"(ptr %[[INT64]], i64 8, i64 16, i64 8)

// CHECK-LABEL: define linkonce void @"main.(*M[bool]).check"(
// CHECK: getelementptr inbounds nuw %"main.M[bool]", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK: icmp ne i64 1, %{{[0-9]+}}
// CHECK: getelementptr inbounds nuw %"main.M[bool]", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK: icmp ne i64 8, %{{[0-9]+}}
// CHECK: getelementptr inbounds nuw %"main.N[bool]", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK: icmp ne i64 1, %{{[0-9]+}}

// CHECK-LABEL: define linkonce void @"main.(*M[int64]).check"(
// CHECK: getelementptr inbounds nuw %"main.M[int64]", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK: icmp ne i64 8, %{{[0-9]+}}
// CHECK: getelementptr inbounds nuw %"main.M[int64]", ptr %{{[0-9]+}}, i32 0, i32 2
// CHECK: icmp ne i64 16, %{{[0-9]+}}
// CHECK: getelementptr inbounds nuw %"main.N[int64]", ptr %{{[0-9]+}}, i32 0, i32 1
// CHECK: icmp ne i64 8, %{{[0-9]+}}
