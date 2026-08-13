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

// check32 must test the dynamic type and then inspect the data pointer from the
// same interface value; the decimal constant is pi32bits.
// CHECK-LABEL: define void @main.check32(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[F32_SLOT:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}eface" %0, ptr [[F32_SLOT]]
// CHECK-NEXT: [[F32_IFACE:%[0-9]+]] = load %"{{.*}}eface", ptr [[F32_SLOT]]
// CHECK-NEXT: [[F32_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F32_IFACE]], 0
// CHECK-NEXT: [[F32_IS_TYPE:%[0-9]+]] = icmp eq ptr [[F32_TYPE]], @_llgo_float32
// CHECK-NEXT: br i1 [[F32_IS_TYPE]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: [[F32_VIEW:%[0-9]+]] = load %main.eface, ptr [[F32_SLOT]]
// CHECK-NEXT: store %main.eface [[F32_VIEW]], ptr [[F32_VIEW_ADDR:%[0-9]+]]
// CHECK-NEXT: [[F32_DATA_FIELD:%[0-9]+]] = getelementptr inbounds %main.eface, ptr [[F32_VIEW_ADDR]], i32 0, i32 1
// CHECK-NEXT: [[F32_DATA:%[0-9]+]] = load ptr, ptr [[F32_DATA_FIELD]]
// CHECK-NEXT: [[F32_BITS:%[0-9]+]] = load i32, ptr [[F32_DATA]]
// CHECK-NEXT: [[F32_BAD_BITS:%[0-9]+]] = icmp ne i32 [[F32_BITS]], 1078530011
// CHECK: [[F32_ASSERT_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F32_IFACE]], 1
// CHECK-NEXT: [[F32_VALUE:%[0-9]+]] = load float, ptr [[F32_ASSERT_DATA]]
// CHECK-NEXT: [[F32_OK0:%[0-9]+]] = insertvalue { float, i1 } undef, float [[F32_VALUE]], 0
// CHECK-NEXT: insertvalue { float, i1 } [[F32_OK0]], i1 true, 1
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

// CHECK-LABEL: define void @main.check64(%"{{.*}}eface" %0){{.*}} {
// CHECK: [[F64_SLOT:%[0-9]+]] = call ptr @"{{.*}}AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}eface" %0, ptr [[F64_SLOT]]
// CHECK-NEXT: [[F64_IFACE:%[0-9]+]] = load %"{{.*}}eface", ptr [[F64_SLOT]]
// CHECK-NEXT: [[F64_TYPE:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F64_IFACE]], 0
// CHECK-NEXT: [[F64_IS_TYPE:%[0-9]+]] = icmp eq ptr [[F64_TYPE]], @_llgo_float64
// CHECK-NEXT: br i1 [[F64_IS_TYPE]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: [[F64_VIEW:%[0-9]+]] = load %main.eface, ptr [[F64_SLOT]]
// CHECK-NEXT: store %main.eface [[F64_VIEW]], ptr [[F64_VIEW_ADDR:%[0-9]+]]
// CHECK: [[F64_DATA_FIELD:%[0-9]+]] = getelementptr inbounds %main.eface, ptr [[F64_VIEW_ADDR]], i32 0, i32 1
// CHECK-NEXT: [[F64_DATA:%[0-9]+]] = load ptr, ptr [[F64_DATA_FIELD]]
// CHECK-NEXT: [[F64_PARTS:%[0-9]+]] = load %main.u64parts, ptr [[F64_DATA]]
// CHECK-NEXT: store %main.u64parts [[F64_PARTS]], ptr [[F64_PARTS_ADDR:%[0-9]+]]
// CHECK-NEXT: [[F64_LO_FIELD:%[0-9]+]] = getelementptr inbounds %main.u64parts, ptr [[F64_PARTS_ADDR]], i32 0, i32 0
// CHECK-NEXT: [[F64_LO:%[0-9]+]] = load i32, ptr [[F64_LO_FIELD]]
// CHECK-NEXT: [[F64_BAD_LO:%[0-9]+]] = icmp ne i32 [[F64_LO]], 1405670641
// CHECK: [[F64_HI_FIELD:%[0-9]+]] = getelementptr inbounds %main.u64parts, ptr [[F64_PARTS_ADDR]], i32 0, i32 1
// CHECK-NEXT: [[F64_HI:%[0-9]+]] = load i32, ptr [[F64_HI_FIELD]]
// CHECK-NEXT: [[F64_BAD_HI:%[0-9]+]] = icmp ne i32 [[F64_HI]], 1074340347
// CHECK: [[F64_ASSERT_DATA:%[0-9]+]] = extractvalue %"{{.*}}eface" [[F64_IFACE]], 1
// CHECK-NEXT: [[F64_VALUE:%[0-9]+]] = load double, ptr [[F64_ASSERT_DATA]]
// CHECK-NEXT: [[F64_OK0:%[0-9]+]] = insertvalue { double, i1 } undef, double [[F64_VALUE]], 0
// CHECK-NEXT: insertvalue { double, i1 } [[F64_OK0]], i1 true, 1
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

// CHECK-LABEL: define float @main.f32(){{.*}} {
// CHECK: ret float 0x400921FB60000000
func f32() float32 {
	return pi
}

// CHECK-LABEL: define double @main.f64(){{.*}} {
// CHECK: ret double 0x400921FB53C8D4F1
func f64() float64 {
	return pi
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_F32:%[0-9]+]] = call float @main.f32()
// CHECK-NEXT: [[MAIN_F32_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 4)
// CHECK-NEXT: store float [[MAIN_F32]], ptr [[MAIN_F32_DATA]]
// CHECK-NEXT: [[MAIN_F32_IFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_float32, ptr undef }, ptr [[MAIN_F32_DATA]], 1
// CHECK-NEXT: call void @main.check32(%"{{.*}}eface" [[MAIN_F32_IFACE]])
// CHECK-NEXT: [[MAIN_F64:%[0-9]+]] = call double @main.f64()
// CHECK-NEXT: [[MAIN_F64_DATA:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK-NEXT: store double [[MAIN_F64]], ptr [[MAIN_F64_DATA]]
// CHECK-NEXT: [[MAIN_F64_IFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @_llgo_float64, ptr undef }, ptr [[MAIN_F64_DATA]], 1
// CHECK-NEXT: call void @main.check64(%"{{.*}}eface" [[MAIN_F64_IFACE]])
func main() {
	check32(f32())
	check64(f64())
}
