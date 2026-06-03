// LITTEST
package main

import "unsafe"

const pi = 3.14159265
const pi32bits = 0x40490fdb
const pi64lo = 0x53c8d4f1
const pi64hi = 0x400921fb

type eface struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

type u64parts struct {
	lo uint32
	hi uint32
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.check32"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: icmp eq ptr {{.*}}, @_llgo_float32
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1078530011
func check32(v any) {
	switch v.(type) {
	case float32:
	default:
		panic("error type f32")
	}
	e := *(*eface)(unsafe.Pointer(&v))
	if *(*uint32)(e.data) != pi32bits {
		panic("error bits f32")
	}
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.check64"(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: icmp eq ptr {{.*}}, @_llgo_float64
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1405670641
// CHECK: load i32, ptr
// CHECK: icmp ne i32 {{.*}}, 1074340347
func check64(v any) {
	switch v.(type) {
	case float64:
	default:
		panic("error type f64")
	}
	e := *(*eface)(unsafe.Pointer(&v))
	bits := *(*u64parts)(e.data)
	if bits.lo != pi64lo || bits.hi != pi64hi {
		panic("error bits f64")
	}
}

// CHECK-LABEL: define float @"{{.*}}/cl/_testrt/float2any.f32"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret float 0x400921FB60000000
// CHECK-NEXT: }
func f32() float32 {
	return pi
}

// CHECK-LABEL: define double @"{{.*}}/cl/_testrt/float2any.f64"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret double 0x400921FB53C8D4F1
// CHECK-NEXT: }
func f64() float64 {
	return pi
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/float2any.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call float @"{{.*}}/cl/_testrt/float2any.f32"()
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float %0, ptr %1, align 4
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/float2any.check32"(%"{{.*}}/runtime/internal/runtime.eface" %2)
// CHECK-NEXT:   %3 = call double @"{{.*}}/cl/_testrt/float2any.f64"()
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/cl/_testrt/float2any.check64"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	check32(f32())
	check64(f64())
}
