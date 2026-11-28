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
// CHECK-DAG: ret i64 14
func layoutSize() uintptr {
	return unsafe.Sizeof(packedLayout{})
}

func main() {
	m1 := M[bool]{}
	var probe1 struct {
		Pad   byte
		Value N[bool]
	}
	m1.check(uintptr(unsafe.Pointer(&probe1.Value))-uintptr(unsafe.Pointer(&probe1)),
		uintptr(unsafe.Pointer(&m1.m2))-uintptr(unsafe.Pointer(&m1)),
		uintptr(unsafe.Pointer(&m1.m2.n2))-uintptr(unsafe.Pointer(&m1.m2)))
	m2 := M[int64]{}
	var probe2 struct {
		Pad   byte
		Value N[int64]
	}
	m2.check(uintptr(unsafe.Pointer(&probe2.Value))-uintptr(unsafe.Pointer(&probe2)),
		uintptr(unsafe.Pointer(&m2.m2))-uintptr(unsafe.Pointer(&m2)),
		uintptr(unsafe.Pointer(&m2.m2.n2))-uintptr(unsafe.Pointer(&m2.m2)))
}

// Each instantiation folds Alignof/Offsetof to its concrete layout.
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

// CHECK-LABEL: define linkonce void @"main.(*M[bool]).check"(
// CHECK-DAG: icmp ne i64 1, %{{[0-9]+}}
// CHECK-DAG: icmp ne i64 8, %{{[0-9]+}}
// CHECK-DAG: icmp ne i64 1, %{{[0-9]+}}

// CHECK-LABEL: define linkonce void @"main.(*M[int64]).check"(
// CHECK-DAG: icmp ne i64 8, %{{[0-9]+}}
// CHECK-DAG: icmp ne i64 16, %{{[0-9]+}}
// CHECK-DAG: icmp ne i64 8, %{{[0-9]+}}
