// LITTEST
// Scope: common
package main

import (
	"math"
	"reflect"
)

type namedString string
type namedBytes []byte
type namedFloat32 float32
type namedFunc func(int) int

// Keep this fixture at the reflection/compiler boundary. The standard library
// owns the full conversion matrix; these cases retain one representative for
// each distinct Value.Convert path LLGo must lower.
//
// FileCheck follows LLGo's deterministic lexical function order. Each block
// checks only the conversion bridge and the value consumed from it.
// CHECK-LABEL: define void @main.functionConversion(){{.*}} {
// CHECK: [[FUNC_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[FUNC_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[FUNC_RESULT:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[FUNC_VALUE]], {{.*}} [[FUNC_TYPE]])
// CHECK: call %{{.*}} @reflect.Value.Interface(%reflect.Value [[FUNC_RESULT]])

// CHECK-LABEL: define void @main.functionPointers(){{.*}} {
// CHECK: [[CAPTURED_ENV:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 8)
// CHECK: [[CAPTURED_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.functionPointers$1", ptr undef }, ptr [[CAPTURED_ENV]], 1
// CHECK-NEXT: [[CAPTURED_BOX:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store { ptr, ptr } [[CAPTURED_PAIR]], ptr [[CAPTURED_BOX]]
// CHECK-NEXT: [[CAPTURED_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @{{.*}}, ptr undef }, ptr [[CAPTURED_BOX]], 1
// CHECK-NEXT: [[CAPTURED_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[CAPTURED_EFACE]])
// CHECK-NEXT: call ptr @reflect.Value.UnsafePointer(%reflect.Value [[CAPTURED_VALUE]])
// CHECK: [[DECLARED_BOX:%[0-9]+]] = call ptr @"{{.*}}AllocU"(i64 16)
// CHECK-NEXT: store { ptr, ptr } { ptr @main.numericConversions, ptr null }, ptr [[DECLARED_BOX]]
// CHECK-NEXT: [[DECLARED_EFACE:%[0-9]+]] = insertvalue %"{{.*}}eface" { ptr @{{.*}}, ptr undef }, ptr [[DECLARED_BOX]], 1
// CHECK-NEXT: [[DECLARED_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(%"{{.*}}eface" [[DECLARED_EFACE]])
// CHECK-NEXT: call ptr @reflect.Value.UnsafePointer(%reflect.Value [[DECLARED_VALUE]])

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.numericConversions()
// CHECK: call void @main.textConversions()
// CHECK: call void @main.sliceArrayConversions()
// CHECK: call void @main.functionConversion()
// CHECK: call void @main.functionPointers()
// CHECK: call void @main.nanConversion()

// CHECK-LABEL: define void @main.nanConversion(){{.*}} {
// CHECK: [[NAN:%[0-9]+]] = call float @math.Float32frombits(i32 2139095041)
// CHECK: [[NAN_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[FLOAT_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[FLOAT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[NAN_VALUE]], {{.*}} [[FLOAT_TYPE]])
// CHECK: call i32 @math.Float32bits(

// CHECK-LABEL: define void @main.numericConversions(){{.*}} {
// CHECK: [[INT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[UINT_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[UINT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[INT_VALUE]], {{.*}} [[UINT_TYPE]])
// CHECK: call i64 @reflect.Value.Uint(%reflect.Value [[UINT_VALUE]])
// CHECK: [[FLOAT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[INT_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[TRUNCATED:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[FLOAT_VALUE]], {{.*}} [[INT_TYPE]])
// CHECK: call i64 @reflect.Value.Int(%reflect.Value [[TRUNCATED]])

// CHECK-LABEL: define void @main.sliceArrayConversions(){{.*}} {
// CHECK: [[SLICE_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[ARRAY_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[ARRAY_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[SLICE_VALUE]], {{.*}} [[ARRAY_TYPE]])
// CHECK: [[POINTER_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[POINTER_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert({{.*}}[[POINTER_TYPE]])
// CHECK: call i1 @reflect.Value.CanConvert(
// CHECK: call %reflect.Value @reflect.Value.Index(%reflect.Value [[ARRAY_VALUE]], i64 0)
// CHECK: call %reflect.Value @reflect.Value.Elem(%reflect.Value [[POINTER_VALUE]])

// CHECK-LABEL: define void @main.textConversions(){{.*}} {
// CHECK: [[STRING_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[BYTES_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[BYTES_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[STRING_VALUE]], {{.*}} [[BYTES_TYPE]])
// CHECK: call %{{.*}} @reflect.Value.Interface(%reflect.Value [[BYTES_VALUE]])
// CHECK: [[NAMED_BYTES:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[STRING_TYPE:%[0-9]+]] = call %{{.*}} @reflect.TypeOf(
// CHECK: [[STRING_RESULT:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[NAMED_BYTES]], {{.*}} [[STRING_TYPE]])
// CHECK: call %{{.*}} @reflect.Value.String(%reflect.Value [[STRING_RESULT]])

func main() {
	numericConversions()
	textConversions()
	sliceArrayConversions()
	functionConversion()
	functionPointers()
	nanConversion()
}

// Numeric conversion exercises signed-to-unsigned overflow and floating-point
// truncation without enumerating the standard library's Cartesian matrix.
func numericConversions() {
	overflow := reflect.ValueOf(int64(257)).Convert(reflect.TypeOf(uint8(0)))
	if overflow.Uint() != 1 {
		panic("integer overflow conversion")
	}

	truncated := reflect.ValueOf(float64(-3.75)).Convert(reflect.TypeOf(int32(0)))
	if truncated.Int() != -3 {
		panic("floating-point truncation")
	}
}

// String/slice conversions use allocation-backed conversion helpers in both
// directions, including named source types.
func textConversions() {
	b := reflect.ValueOf(namedString("go")).Convert(reflect.TypeOf([]byte(nil))).Interface().([]byte)
	if len(b) != 2 || b[0] != 'g' || b[1] != 'o' {
		panic("string to byte slice")
	}

	s := reflect.ValueOf(namedBytes{'l', 'l'}).Convert(reflect.TypeOf("")).String()
	if s != "ll" {
		panic("byte slice to string")
	}
}

// Array conversion copies, pointer-to-array conversion aliases, and a short
// slice is rejected before the conversion helper is called.
func sliceArrayConversions() {
	src := []byte{1, 2, 3, 4}
	sv := reflect.ValueOf(src)
	av := sv.Convert(reflect.TypeOf([4]byte{}))
	src[0] = 9
	if av.Index(0).Uint() != 1 || av.CanAddr() {
		panic("slice-to-array must be an unaddressable copy")
	}

	pv := sv.Convert(reflect.TypeOf((*[4]byte)(nil)))
	src[1] = 8
	if pv.Elem().Index(1).Uint() != 8 {
		panic("slice-to-array-pointer must alias")
	}

	short := reflect.ValueOf(src[:2])
	arrayType := reflect.TypeOf([4]byte{})
	if short.CanConvert(arrayType) {
		panic("short slice reported convertible")
	}
	assertPanics(func() { short.Convert(arrayType) })
}

// Named-to-unnamed function conversion must preserve the closure environment.
func functionConversion() {
	bias := 40
	var fn namedFunc = func(v int) int { return bias + v }
	converted := reflect.ValueOf(fn).Convert(reflect.TypeOf((func(int) int)(nil))).Interface().(func(int) int)
	if converted(2) != 42 {
		panic("function conversion lost closure environment")
	}
}

// Func-kind UnsafePointer is a separate reflect path from conversion. Keep a
// captured closure and a declared function: the former carries an environment,
// while both must expose their non-nil code pointer.
func functionPointers() {
	bias := 1
	captured := func(v int) int { return bias + v }
	capturedPointer := reflect.ValueOf(captured).UnsafePointer()
	declaredPointer := reflect.ValueOf(numericConversions).UnsafePointer()
	if capturedPointer == nil || declaredPointer == nil || capturedPointer == declaredPointer {
		panic("bad reflected function pointer")
	}
}

// A signaling NaN's payload must survive conversion from a named float. This
// is the non-ordinary numeric edge that previously caught backend differences.
func nanConversion() {
	const signalingNaN = uint32(0x7f800001)
	in := namedFloat32(math.Float32frombits(signalingNaN))
	out := reflect.ValueOf(in).Convert(reflect.TypeOf(float32(0))).Interface().(float32)
	if math.Float32bits(out) != signalingNaN {
		panic("signaling NaN payload changed")
	}
}

func assertPanics(fn func()) {
	defer func() {
		if recover() == nil {
			panic("conversion did not panic")
		}
	}()
	fn()
}
