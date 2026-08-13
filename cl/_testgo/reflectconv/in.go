// LITTEST
package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"unsafe"
)

// Interface helper values must pass through the V function value and Elem.
// CHECK-LABEL: define %reflect.Value @main.EmptyInterfaceV(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: [[EV_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr [[EV_BOX]]
// CHECK-NEXT: [[EV_V:%[0-9]+]] = load { ptr, ptr }, ptr @main.V
// CHECK-NEXT: [[EV_ARG:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_any", ptr undef }, ptr [[EV_BOX]], 1
// CHECK-NEXT: [[EV_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[EV_V]], 1
// CHECK-NEXT: [[EV_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[EV_V]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[EV_RAW_CODE]])
// CHECK-NEXT: [[EV_PTR:%[0-9]+]] = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[EV_ENV]], %"{{.*}}/runtime/internal/runtime.eface" [[EV_ARG]])
// CHECK: [[EV_ELEM:%[0-9]+]] = call %reflect.Value @reflect.Value.Elem(%reflect.Value [[EV_PTR]])
// CHECK: ret %reflect.Value [[EV_ELEM]]

// IsRO reads flagStickyRO, while MakeRO sets that exact bit and returns the copy.
// CHECK-LABEL: define i1 @main.IsRO(%reflect.Value %0){{.*}} {
// CHECK: [[RO_VALUE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT: store %reflect.Value %0, ptr [[RO_VALUE]]
// CHECK-NEXT: [[RO_FLAG_PTR:%[0-9]+]] = getelementptr inbounds %main.Value, ptr [[RO_VALUE]], i32 0, i32 2
// CHECK: [[RO_FLAGS:%[0-9]+]] = load i64, ptr [[RO_FLAG_PTR]]
// CHECK: [[RO_BIT:%[0-9]+]] = and i64 [[RO_FLAGS]], 32
// CHECK: [[RO_SET:%[0-9]+]] = icmp ne i64 [[RO_BIT]], 0
// CHECK: ret i1 [[RO_SET]]

// CHECK-LABEL: define %reflect.Value @main.MakeRO(%reflect.Value %0){{.*}} {
// CHECK: [[MAKE_VALUE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT: store %reflect.Value %0, ptr [[MAKE_VALUE]]
// CHECK-NEXT: [[MAKE_FLAG_PTR:%[0-9]+]] = getelementptr inbounds %main.Value, ptr [[MAKE_VALUE]], i32 0, i32 2
// CHECK: [[MAKE_FLAGS:%[0-9]+]] = load i64, ptr [[MAKE_FLAG_PTR]]
// CHECK: [[MAKE_RO_FLAGS:%[0-9]+]] = or i64 [[MAKE_FLAGS]], 32
// CHECK: [[MAKE_STORE_PTR:%[0-9]+]] = getelementptr inbounds %main.Value, ptr [[MAKE_VALUE]], i32 0, i32 2
// CHECK-NEXT: store i64 [[MAKE_RO_FLAGS]], ptr [[MAKE_STORE_PTR]]
// CHECK-NEXT: [[MAKE_RESULT:%[0-9]+]] = load %reflect.Value, ptr [[MAKE_VALUE]]
// CHECK: ret %reflect.Value [[MAKE_RESULT]]

// CHECK-LABEL: define %reflect.Value @main.ReadWriterV(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: [[RW_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr [[RW_BOX]]
// CHECK-NEXT: [[RW_V:%[0-9]+]] = load { ptr, ptr }, ptr @main.V
// CHECK-NEXT: [[RW_ARG:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.ReadWriter", ptr undef }, ptr [[RW_BOX]], 1
// CHECK-NEXT: [[RW_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[RW_V]], 1
// CHECK-NEXT: [[RW_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[RW_V]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[RW_RAW_CODE]])
// CHECK-NEXT: [[RW_PTR:%[0-9]+]] = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[RW_ENV]], %"{{.*}}/runtime/internal/runtime.eface" [[RW_ARG]])
// CHECK: [[RW_ELEM:%[0-9]+]] = call %reflect.Value @reflect.Value.Elem(%reflect.Value [[RW_PTR]])
// CHECK: ret %reflect.Value [[RW_ELEM]]

// CHECK-LABEL: define %reflect.Value @main.ReaderV(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK: [[R_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr [[R_BOX]]
// CHECK-NEXT: [[R_V:%[0-9]+]] = load { ptr, ptr }, ptr @main.V
// CHECK-NEXT: [[R_ARG:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Reader", ptr undef }, ptr [[R_BOX]], 1
// CHECK-NEXT: [[R_ENV:%[0-9]+]] = extractvalue { ptr, ptr } [[R_V]], 1
// CHECK-NEXT: [[R_RAW_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[R_V]], 0
// CHECK-NEXT: %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr [[R_RAW_CODE]])
// CHECK-NEXT: [[R_PTR:%[0-9]+]] = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} [[R_ENV]], %"{{.*}}/runtime/internal/runtime.eface" [[R_ARG]])
// CHECK: [[R_ELEM:%[0-9]+]] = call %reflect.Value @reflect.Value.Elem(%reflect.Value [[R_PTR]])
// CHECK: ret %reflect.Value [[R_ELEM]]

// Each table entry records t1 -> t2, verifies both conversions, then proves the
// converted value can be assigned and that the read-only bit is preserved.
// CHECK-LABEL: define void @main.TestConvert(ptr %0){{.*}} {
// CHECK: [[CAN_MAP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map{{\[\[2\]}}_llgo_reflect.Type]_llgo_bool", i64 0)
// CHECK: [[ALL_MAP:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_reflect.Type]_llgo_bool", i64 0)
// CHECK: [[CONVERT_TABLE:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.Slice", ptr @main.convertTests
// CHECK: [[ENTRY_PTR:%[0-9]+]] = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %{{[0-9]+}}, i64 %{{[0-9]+}}
// CHECK: [[ENTRY:%[0-9]+]] = load { %reflect.Value, %reflect.Value }, ptr [[ENTRY_PTR]]
// CHECK: [[IN_VALUE:%[0-9]+]] = load %reflect.Value, ptr %{{[0-9]+}}
// CHECK: [[T1:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value [[IN_VALUE]])
// CHECK: [[SELF_CONVERTIBLE:%[0-9]+]] = call i1 %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" [[T1]])
// CHECK: br i1 [[SELF_CONVERTIBLE]],
// CHECK: [[OUT_VALUE:%[0-9]+]] = load %reflect.Value, ptr %{{[0-9]+}}
// CHECK: [[T2:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value [[OUT_VALUE]])
// CHECK: [[PAIR_CONVERTIBLE:%[0-9]+]] = call i1 %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" [[T2]])
// CHECK: br i1 [[PAIR_CONVERTIBLE]],
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr [[ALL_MAP]],
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr [[ALL_MAP]],
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map{{\[\[2\]}}_llgo_reflect.Type]_llgo_bool", ptr [[CAN_MAP]],
// CHECK: [[V1:%[0-9]+]] = load %reflect.Value, ptr %{{[0-9]+}}
// CHECK: [[V1_CAN_SELF:%[0-9]+]] = call i1 @reflect.Value.CanConvert(%reflect.Value [[V1]], %"{{.*}}/runtime/internal/runtime.iface" [[T1]])
// CHECK: br i1 [[V1_CAN_SELF]],
// CHECK: [[VOUT1:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[V1]], %"{{.*}}/runtime/internal/runtime.iface" [[T1]])
// CHECK: [[OUT1:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value [[VOUT1]])
// CHECK: [[V1_CAN_OUT:%[0-9]+]] = call i1 @reflect.Value.CanConvert(%reflect.Value [[V1]], %"{{.*}}/runtime/internal/runtime.iface" [[T2]])
// CHECK: br i1 [[V1_CAN_OUT]],
// CHECK: [[SELF_EQUAL:%[0-9]+]] = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" [[OUT1]], %"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK: br i1 [[SELF_EQUAL]],
// CHECK: [[VOUT2:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[V1]], %"{{.*}}/runtime/internal/runtime.iface" [[T2]])
// CHECK: [[OUT2:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value [[VOUT2]])
// CHECK: [[OUT_KIND:%[0-9]+]] = call i64 @reflect.Value.Kind(%reflect.Value [[VOUT2]])
// CHECK: [[TYPE_KIND:%[0-9]+]] = call i64 %{{[0-9]+}}(ptr %{{[0-9]+}})
// CHECK: [[BAD_KIND:%[0-9]+]] = icmp ne i64 [[OUT_KIND]], [[TYPE_KIND]]
// CHECK: [[OUT_EQUAL:%[0-9]+]] = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" [[OUT2]], %"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK: br i1 [[OUT_EQUAL]],
// CHECK: [[NEW_OUT:%[0-9]+]] = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" [[T2]])
// CHECK: [[VOUT3:%[0-9]+]] = call %reflect.Value @reflect.Value.Elem(%reflect.Value [[NEW_OUT]])
// CHECK: call void @reflect.Value.Set(%reflect.Value [[VOUT3]], %reflect.Value [[VOUT2]])
// CHECK: [[OUT3:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value [[VOUT3]])
// CHECK: call i1 @main.IsRO(%reflect.Value [[V1]])
// CHECK: [[OUT3_EQUAL:%[0-9]+]] = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" [[OUT3]], %"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK: br i1 [[OUT3_EQUAL]],
// CHECK: call i1 @main.IsRO(%reflect.Value [[VOUT1]])
// CHECK: call i1 @main.IsRO(%reflect.Value [[VOUT2]])
// CHECK: call i1 @main.IsRO(%reflect.Value [[VOUT3]])
// CHECK: [[RO_SELF_IN:%[0-9]+]] = call %reflect.Value @main.MakeRO(%reflect.Value [[V1]])
// CHECK: [[RO_SELF_OUT:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[RO_SELF_IN]], %"{{.*}}/runtime/internal/runtime.iface" [[T1]])
// CHECK: [[RO_SELF_OK:%[0-9]+]] = call i1 @main.IsRO(%reflect.Value [[RO_SELF_OUT]])
// CHECK: br i1 [[RO_SELF_OK]],
// CHECK: [[RO_PAIR_IN:%[0-9]+]] = call %reflect.Value @main.MakeRO(%reflect.Value [[V1]])
// CHECK: [[RO_PAIR_OUT:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[RO_PAIR_IN]], %"{{.*}}/runtime/internal/runtime.iface" [[T2]])
// CHECK: [[RO_PAIR_OK:%[0-9]+]] = call i1 @main.IsRO(%reflect.Value [[RO_PAIR_OUT]])
// CHECK: br i1 [[RO_PAIR_OK]],
// CHECK: call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr [[ALL_MAP]])
// CHECK: call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"
// CHECK: [[MATRIX_OK:%[0-9]+]] = call i1 %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}})
// CHECK: [[BAD_MATRIX:%[0-9]+]] = icmp ne i1 [[MATRIX_OK]], %{{[0-9]+}}
// CHECK: [[KNOWN_PAIR_PTR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map{{\[\[2\]}}_llgo_reflect.Type]_llgo_bool", ptr [[CAN_MAP]],
// CHECK: [[KNOWN_PAIR:%[0-9]+]] = load i1, ptr [[KNOWN_PAIR_PTR]]

// sNaN must survive both an ordinary store/load and a reflective named-float conversion.
// CHECK-LABEL: define void @main.TestConvertNaNs(ptr %0){{.*}} {
// CHECK: [[SNAN_VALUE:%[0-9]+]] = call float @math.Float32frombits(i32 2139095041)
// CHECK: store float [[SNAN_VALUE]], ptr @main.gFloat32
// CHECK: [[SNAN_LOAD:%[0-9]+]] = load float, ptr @main.gFloat32
// CHECK: [[SNAN_BITS:%[0-9]+]] = call i32 @math.Float32bits(float [[SNAN_LOAD]])
// CHECK: [[SNAN_BAD:%[0-9]+]] = icmp ne i32 [[SNAN_BITS]], 2139095041
// CHECK: [[SNAN_X:%[0-9]+]] = call %reflect.Value %{{[^ ]+}}(ptr {{(nest|swiftself)}} %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.eface" %{{[0-9]+}})
// CHECK: [[FLOAT32_TYPE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(
// CHECK: [[SNAN_Y:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[SNAN_X]], %"{{.*}}/runtime/internal/runtime.iface" [[FLOAT32_TYPE]])
// CHECK: [[SNAN_ANY:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value [[SNAN_Y]])
// CHECK: [[SNAN_TYPE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[SNAN_ANY]], 0
// CHECK: [[SNAN_IS_FLOAT:%[0-9]+]] = icmp eq ptr [[SNAN_TYPE]], @_llgo_float32
// CHECK: [[SNAN_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" [[SNAN_ANY]], 1
// CHECK: [[SNAN_FLOAT:%[0-9]+]] = load float, ptr [[SNAN_DATA]]
// CHECK: [[SNAN_CONVERT_BITS:%[0-9]+]] = call i32 @math.Float32bits(float [[SNAN_FLOAT]])
// CHECK: [[SNAN_CONVERT_BAD:%[0-9]+]] = icmp ne i32 [[SNAN_CONVERT_BITS]], 2139095041

// A four-byte slice is type-convertible to *[8]byte but not value-convertible;
// both pointer-to-array and array conversions must execute inside panic closures.
// CHECK-LABEL: define void @main.TestConvertPanic(ptr %0){{.*}} {
// CHECK: [[PANIC_SLICE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"({{.*}}i64 4,
// CHECK: [[PANIC_V:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[PANIC_PT:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(
// CHECK: [[PANIC_TYPE_OK:%[0-9]+]] = call i1 %{{[0-9]+}}(ptr %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}})
// CHECK: br i1 [[PANIC_TYPE_OK]],
// CHECK: [[PANIC_PTR_OK:%[0-9]+]] = call i1 @reflect.Value.CanConvert(%reflect.Value %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}})
// CHECK: br i1 [[PANIC_PTR_OK]],
// CHECK: [[PANIC_PTR_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.TestConvertPanic$1", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: call void @main.shouldPanic({{.*}}{ ptr, ptr } [[PANIC_PTR_CLOSURE]])
// CHECK: [[PANIC_ARRAY_TYPE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" %{{[0-9]+}}(ptr %{{[0-9]+}})
// CHECK: [[PANIC_ARRAY_OK:%[0-9]+]] = call i1 @reflect.Value.CanConvert(%reflect.Value %{{[0-9]+}}, %"{{.*}}/runtime/internal/runtime.iface" [[PANIC_ARRAY_TYPE]])
// CHECK: br i1 [[PANIC_ARRAY_OK]],
// CHECK: [[PANIC_ARRAY_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"main.TestConvertPanic$2", ptr undef }, ptr %{{[0-9]+}}, 1
// CHECK: call void @main.shouldPanic({{.*}}{ ptr, ptr } [[PANIC_ARRAY_CLOSURE]])

// CHECK-LABEL: define void @"main.TestConvertPanic$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[PANIC1_ENV:%[0-9]+]] = load { ptr, ptr }, ptr %0
// CHECK-NEXT: [[PANIC1_V_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC1_ENV]], 0
// CHECK: [[PANIC1_V_SAFE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr [[PANIC1_V_PTR]])
// CHECK-NEXT: [[PANIC1_V:%[0-9]+]] = load %reflect.Value, ptr [[PANIC1_V_SAFE]]
// CHECK-NEXT: [[PANIC1_T_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC1_ENV]], 1
// CHECK-NEXT: [[PANIC1_T:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.iface", ptr [[PANIC1_T_PTR]]
// CHECK-NEXT: call %reflect.Value @reflect.Value.Convert(%reflect.Value [[PANIC1_V]], %"{{.*}}/runtime/internal/runtime.iface" [[PANIC1_T]])

// CHECK-LABEL: define void @"main.TestConvertPanic$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[PANIC2_ENV:%[0-9]+]] = load { ptr, ptr }, ptr %0
// CHECK-NEXT: [[PANIC2_V_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC2_ENV]], 0
// CHECK: [[PANIC2_V_SAFE:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr [[PANIC2_V_PTR]])
// CHECK-NEXT: [[PANIC2_V:%[0-9]+]] = load %reflect.Value, ptr [[PANIC2_V_SAFE]]
// CHECK-NEXT: [[PANIC2_PT_PTR:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC2_ENV]], 1
// CHECK-NEXT: [[PANIC2_PT:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.iface", ptr [[PANIC2_PT_PTR]]
// CHECK-NEXT: [[PANIC2_DATA:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" [[PANIC2_PT]])
// CHECK-NEXT: [[PANIC2_TABLE:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" [[PANIC2_PT]], 0
// CHECK-NEXT: [[PANIC2_ELEM_SLOT:%[0-9]+]] = getelementptr ptr, ptr [[PANIC2_TABLE]], i64 11
// CHECK-NEXT: [[PANIC2_ELEM_METHOD:%[0-9]+]] = load ptr, ptr [[PANIC2_ELEM_SLOT]]
// CHECK-NEXT: [[PANIC2_PAIR0:%[0-9]+]] = insertvalue { ptr, ptr } undef, ptr [[PANIC2_ELEM_METHOD]], 0
// CHECK-NEXT: [[PANIC2_PAIR:%[0-9]+]] = insertvalue { ptr, ptr } [[PANIC2_PAIR0]], ptr [[PANIC2_DATA]], 1
// CHECK-NEXT: [[PANIC2_RECV:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC2_PAIR]], 1
// CHECK-NEXT: [[PANIC2_CODE:%[0-9]+]] = extractvalue { ptr, ptr } [[PANIC2_PAIR]], 0
// CHECK-NEXT: [[PANIC2_AT:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" [[PANIC2_CODE]](ptr [[PANIC2_RECV]])
// CHECK-NEXT: call %reflect.Value @reflect.Value.Convert(%reflect.Value [[PANIC2_V]], %"{{.*}}/runtime/internal/runtime.iface" [[PANIC2_AT]])

// Slice-to-array conversion must be a non-addressable copy: mutate ov, then read zeros from v.
// CHECK-LABEL: define void @main.TestConvertSlice2Array(ptr %0){{.*}} {
// CHECK: [[ARRAY_TYPE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(
// CHECK: [[SLICE_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: [[ARRAY_VALUE:%[0-9]+]] = call %reflect.Value @reflect.Value.Convert(%reflect.Value [[SLICE_VALUE]], %"{{.*}}/runtime/internal/runtime.iface" [[ARRAY_TYPE]])
// CHECK: [[ARRAY_ADDR:%[0-9]+]] = call i1 @reflect.Value.CanAddr(%reflect.Value [[ARRAY_VALUE]])
// CHECK: br i1 [[ARRAY_ADDR]],
// CHECK: [[SLICE_ELEM:%[0-9]+]] = call %reflect.Value @reflect.Value.Index(%reflect.Value [[SLICE_VALUE]], i64 [[MUT_I:%[0-9]+]])
// CHECK: [[NEXT_I:%[0-9]+]] = add i64 [[MUT_I]], 1
// CHECK: [[NEXT_VALUE:%[0-9]+]] = call %reflect.Value @reflect.ValueOf(
// CHECK: call void @reflect.Value.Set(%reflect.Value [[SLICE_ELEM]], %reflect.Value [[NEXT_VALUE]])
// CHECK: [[ARRAY_ELEM:%[0-9]+]] = call %reflect.Value @reflect.Value.Index(%reflect.Value [[ARRAY_VALUE]], i64 %{{[0-9]+}})
// CHECK: [[ARRAY_INT:%[0-9]+]] = call i64 @reflect.Value.Int(%reflect.Value [[ARRAY_ELEM]])
// CHECK: [[ARRAY_CHANGED:%[0-9]+]] = icmp ne i64 [[ARRAY_INT]], 0

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.TestConvert(ptr
// CHECK: call void @main.TestConvertPanic(ptr
// CHECK: call void @main.TestConvertSlice2Array(ptr
// CHECK: call void @main.TestConvertNaNs(ptr

type Value struct {
	typ_ unsafe.Pointer
	ptr  unsafe.Pointer
	flag uintptr
}

const flagStickyRO uintptr = 1 << 5

// MakeRO returns a copy of v with the read-only flag set.
func MakeRO(v reflect.Value) reflect.Value {
	(*Value)(unsafe.Pointer(&v)).flag |= flagStickyRO
	return v
}

// IsRO reports whether v's read-only flag is set.
func IsRO(v reflect.Value) bool {
	return (*Value)(unsafe.Pointer(&v)).flag&flagStickyRO != 0
}

type testingT struct {
}

func (t *testingT) Errorf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func (t *testingT) Fatalf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

func main() {
	TestConvert(&testingT{})
	TestConvertPanic(&testingT{})
	TestConvertSlice2Array(&testingT{})
	TestConvertNaNs(&testingT{})
}

var V = reflect.ValueOf

func EmptyInterfaceV(x any) reflect.Value {
	return V(&x).Elem()
}

func ReaderV(x io.Reader) reflect.Value {
	return V(&x).Elem()
}

func ReadWriterV(x io.ReadWriter) reflect.Value {
	return V(&x).Elem()
}

type integer int
type T struct {
	a int
	b float64
	c string
	d *int
}

type Empty struct{}
type MyStruct struct {
	x int `some:"tag"`
}
type MyStruct1 struct {
	x struct {
		int `some:"bar"`
	}
}
type MyStruct2 struct {
	x struct {
		int `some:"foo"`
	}
}
type MyString string
type MyBytes []byte
type MyBytesArrayPtr0 *[0]byte
type MyBytesArrayPtr *[4]byte
type MyBytesArray0 [0]byte
type MyBytesArray [4]byte
type MyRunes []int32
type MyFunc func()
type MyByte byte

type IntChan chan int
type IntChanRecv <-chan int
type IntChanSend chan<- int
type BytesChan chan []byte
type BytesChanRecv <-chan []byte
type BytesChanSend chan<- []byte

var convertTests = []struct {
	in  reflect.Value
	out reflect.Value
}{
	// numbers
	/*
		Edit .+1,/\*\//-1>cat >/tmp/x.go && go run /tmp/x.go

		package main

		import "fmt"

		var numbers = []string{
			"int8", "uint8", "int16", "uint16",
			"int32", "uint32", "int64", "uint64",
			"int", "uint", "uintptr",
			"float32", "float64",
		}

		func main() {
			// all pairs but in an unusual order,
			// to emit all the int8, uint8 cases
			// before n grows too big.
			n := 1
			for i, f := range numbers {
				for _, g := range numbers[i:] {
					fmt.Printf("\t{V(%s(%d)), V(%s(%d))},\n", f, n, g, n)
					n++
					if f != g {
						fmt.Printf("\t{V(%s(%d)), V(%s(%d))},\n", g, n, f, n)
						n++
					}
				}
			}
		}
	*/
	{V(byte(0)), V(byte(0))},
	{V(int8(1)), V(int8(1))},
	{V(int8(2)), V(uint8(2))},
	{V(uint8(3)), V(int8(3))},
	{V(int8(4)), V(int16(4))},
	{V(int16(5)), V(int8(5))},
	{V(int8(6)), V(uint16(6))},
	{V(uint16(7)), V(int8(7))},
	{V(int8(8)), V(int32(8))},
	{V(int32(9)), V(int8(9))},
	{V(int8(10)), V(uint32(10))},
	{V(uint32(11)), V(int8(11))},
	{V(int8(12)), V(int64(12))},
	{V(int64(13)), V(int8(13))},
	{V(int8(14)), V(uint64(14))},
	{V(uint64(15)), V(int8(15))},
	{V(int8(16)), V(int(16))},
	{V(int(17)), V(int8(17))},
	{V(int8(18)), V(uint(18))},
	{V(uint(19)), V(int8(19))},
	{V(int8(20)), V(uintptr(20))},
	{V(uintptr(21)), V(int8(21))},
	{V(int8(22)), V(float32(22))},
	{V(float32(23)), V(int8(23))},
	{V(int8(24)), V(float64(24))},
	{V(float64(25)), V(int8(25))},
	{V(uint8(26)), V(uint8(26))},
	{V(uint8(27)), V(int16(27))},
	{V(int16(28)), V(uint8(28))},
	{V(uint8(29)), V(uint16(29))},
	{V(uint16(30)), V(uint8(30))},
	{V(uint8(31)), V(int32(31))},
	{V(int32(32)), V(uint8(32))},
	{V(uint8(33)), V(uint32(33))},
	{V(uint32(34)), V(uint8(34))},
	{V(uint8(35)), V(int64(35))},
	{V(int64(36)), V(uint8(36))},
	{V(uint8(37)), V(uint64(37))},
	{V(uint64(38)), V(uint8(38))},
	{V(uint8(39)), V(int(39))},
	{V(int(40)), V(uint8(40))},
	{V(uint8(41)), V(uint(41))},
	{V(uint(42)), V(uint8(42))},
	{V(uint8(43)), V(uintptr(43))},
	{V(uintptr(44)), V(uint8(44))},
	{V(uint8(45)), V(float32(45))},
	{V(float32(46)), V(uint8(46))},
	{V(uint8(47)), V(float64(47))},
	{V(float64(48)), V(uint8(48))},
	{V(int16(49)), V(int16(49))},
	{V(int16(50)), V(uint16(50))},
	{V(uint16(51)), V(int16(51))},
	{V(int16(52)), V(int32(52))},
	{V(int32(53)), V(int16(53))},
	{V(int16(54)), V(uint32(54))},
	{V(uint32(55)), V(int16(55))},
	{V(int16(56)), V(int64(56))},
	{V(int64(57)), V(int16(57))},
	{V(int16(58)), V(uint64(58))},
	{V(uint64(59)), V(int16(59))},
	{V(int16(60)), V(int(60))},
	{V(int(61)), V(int16(61))},
	{V(int16(62)), V(uint(62))},
	{V(uint(63)), V(int16(63))},
	{V(int16(64)), V(uintptr(64))},
	{V(uintptr(65)), V(int16(65))},
	{V(int16(66)), V(float32(66))},
	{V(float32(67)), V(int16(67))},
	{V(int16(68)), V(float64(68))},
	{V(float64(69)), V(int16(69))},
	{V(uint16(70)), V(uint16(70))},
	{V(uint16(71)), V(int32(71))},
	{V(int32(72)), V(uint16(72))},
	{V(uint16(73)), V(uint32(73))},
	{V(uint32(74)), V(uint16(74))},
	{V(uint16(75)), V(int64(75))},
	{V(int64(76)), V(uint16(76))},
	{V(uint16(77)), V(uint64(77))},
	{V(uint64(78)), V(uint16(78))},
	{V(uint16(79)), V(int(79))},
	{V(int(80)), V(uint16(80))},
	{V(uint16(81)), V(uint(81))},
	{V(uint(82)), V(uint16(82))},
	{V(uint16(83)), V(uintptr(83))},
	{V(uintptr(84)), V(uint16(84))},
	{V(uint16(85)), V(float32(85))},
	{V(float32(86)), V(uint16(86))},
	{V(uint16(87)), V(float64(87))},
	{V(float64(88)), V(uint16(88))},
	{V(int32(89)), V(int32(89))},
	{V(int32(90)), V(uint32(90))},
	{V(uint32(91)), V(int32(91))},
	{V(int32(92)), V(int64(92))},
	{V(int64(93)), V(int32(93))},
	{V(int32(94)), V(uint64(94))},
	{V(uint64(95)), V(int32(95))},
	{V(int32(96)), V(int(96))},
	{V(int(97)), V(int32(97))},
	{V(int32(98)), V(uint(98))},
	{V(uint(99)), V(int32(99))},
	{V(int32(100)), V(uintptr(100))},
	{V(uintptr(101)), V(int32(101))},
	{V(int32(102)), V(float32(102))},
	{V(float32(103)), V(int32(103))},
	{V(int32(104)), V(float64(104))},
	{V(float64(105)), V(int32(105))},
	{V(uint32(106)), V(uint32(106))},
	{V(uint32(107)), V(int64(107))},
	{V(int64(108)), V(uint32(108))},
	{V(uint32(109)), V(uint64(109))},
	{V(uint64(110)), V(uint32(110))},
	{V(uint32(111)), V(int(111))},
	{V(int(112)), V(uint32(112))},
	{V(uint32(113)), V(uint(113))},
	{V(uint(114)), V(uint32(114))},
	{V(uint32(115)), V(uintptr(115))},
	{V(uintptr(116)), V(uint32(116))},
	{V(uint32(117)), V(float32(117))},
	{V(float32(118)), V(uint32(118))},
	{V(uint32(119)), V(float64(119))},
	{V(float64(120)), V(uint32(120))},
	{V(int64(121)), V(int64(121))},
	{V(int64(122)), V(uint64(122))},
	{V(uint64(123)), V(int64(123))},
	{V(int64(124)), V(int(124))},
	{V(int(125)), V(int64(125))},
	{V(int64(126)), V(uint(126))},
	{V(uint(127)), V(int64(127))},
	{V(int64(128)), V(uintptr(128))},
	{V(uintptr(129)), V(int64(129))},
	{V(int64(130)), V(float32(130))},
	{V(float32(131)), V(int64(131))},
	{V(int64(132)), V(float64(132))},
	{V(float64(133)), V(int64(133))},
	{V(uint64(134)), V(uint64(134))},
	{V(uint64(135)), V(int(135))},
	{V(int(136)), V(uint64(136))},
	{V(uint64(137)), V(uint(137))},
	{V(uint(138)), V(uint64(138))},
	{V(uint64(139)), V(uintptr(139))},
	{V(uintptr(140)), V(uint64(140))},
	{V(uint64(141)), V(float32(141))},
	{V(float32(142)), V(uint64(142))},
	{V(uint64(143)), V(float64(143))},
	{V(float64(144)), V(uint64(144))},
	{V(int(145)), V(int(145))},
	{V(int(146)), V(uint(146))},
	{V(uint(147)), V(int(147))},
	{V(int(148)), V(uintptr(148))},
	{V(uintptr(149)), V(int(149))},
	{V(int(150)), V(float32(150))},
	{V(float32(151)), V(int(151))},
	{V(int(152)), V(float64(152))},
	{V(float64(153)), V(int(153))},
	{V(uint(154)), V(uint(154))},
	{V(uint(155)), V(uintptr(155))},
	{V(uintptr(156)), V(uint(156))},
	{V(uint(157)), V(float32(157))},
	{V(float32(158)), V(uint(158))},
	{V(uint(159)), V(float64(159))},
	{V(float64(160)), V(uint(160))},
	{V(uintptr(161)), V(uintptr(161))},
	{V(uintptr(162)), V(float32(162))},
	{V(float32(163)), V(uintptr(163))},
	{V(uintptr(164)), V(float64(164))},
	{V(float64(165)), V(uintptr(165))},
	{V(float32(166)), V(float32(166))},
	{V(float32(167)), V(float64(167))},
	{V(float64(168)), V(float32(168))},
	{V(float64(169)), V(float64(169))},

	// truncation
	{V(float64(1.5)), V(int(1))},

	// complex
	{V(complex64(1i)), V(complex64(1i))},
	{V(complex64(2i)), V(complex128(2i))},
	{V(complex128(3i)), V(complex64(3i))},
	{V(complex128(4i)), V(complex128(4i))},

	// string
	{V(string("hello")), V(string("hello"))},
	{V(string("bytes1")), V([]byte("bytes1"))},
	{V([]byte("bytes2")), V(string("bytes2"))},
	{V([]byte("bytes3")), V([]byte("bytes3"))},
	{V(string("runes♝")), V([]rune("runes♝"))},
	{V([]rune("runes♕")), V(string("runes♕"))},
	{V([]rune("runes🙈🙉🙊")), V([]rune("runes🙈🙉🙊"))},
	{V(int('a')), V(string("a"))},
	{V(int8('a')), V(string("a"))},
	{V(int16('a')), V(string("a"))},
	{V(int32('a')), V(string("a"))},
	{V(int64('a')), V(string("a"))},
	{V(uint('a')), V(string("a"))},
	{V(uint8('a')), V(string("a"))},
	{V(uint16('a')), V(string("a"))},
	{V(uint32('a')), V(string("a"))},
	{V(uint64('a')), V(string("a"))},
	{V(uintptr('a')), V(string("a"))},
	{V(int(-1)), V(string("\uFFFD"))},
	{V(int8(-2)), V(string("\uFFFD"))},
	{V(int16(-3)), V(string("\uFFFD"))},
	{V(int32(-4)), V(string("\uFFFD"))},
	{V(int64(-5)), V(string("\uFFFD"))},
	{V(int64(-1 << 32)), V(string("\uFFFD"))},
	{V(int64(1 << 32)), V(string("\uFFFD"))},
	{V(uint(0x110001)), V(string("\uFFFD"))},
	{V(uint32(0x110002)), V(string("\uFFFD"))},
	{V(uint64(0x110003)), V(string("\uFFFD"))},
	{V(uint64(1 << 32)), V(string("\uFFFD"))},
	{V(uintptr(0x110004)), V(string("\uFFFD"))},

	// named string
	{V(MyString("hello")), V(string("hello"))},
	{V(string("hello")), V(MyString("hello"))},
	{V(string("hello")), V(string("hello"))},
	{V(MyString("hello")), V(MyString("hello"))},
	{V(MyString("bytes1")), V([]byte("bytes1"))},
	{V([]byte("bytes2")), V(MyString("bytes2"))},
	{V([]byte("bytes3")), V([]byte("bytes3"))},
	{V(MyString("runes♝")), V([]rune("runes♝"))},
	{V([]rune("runes♕")), V(MyString("runes♕"))},
	{V([]rune("runes🙈🙉🙊")), V([]rune("runes🙈🙉🙊"))},
	{V([]rune("runes🙈🙉🙊")), V(MyRunes("runes🙈🙉🙊"))},
	{V(MyRunes("runes🙈🙉🙊")), V([]rune("runes🙈🙉🙊"))},
	{V(int('a')), V(MyString("a"))},
	{V(int8('a')), V(MyString("a"))},
	{V(int16('a')), V(MyString("a"))},
	{V(int32('a')), V(MyString("a"))},
	{V(int64('a')), V(MyString("a"))},
	{V(uint('a')), V(MyString("a"))},
	{V(uint8('a')), V(MyString("a"))},
	{V(uint16('a')), V(MyString("a"))},
	{V(uint32('a')), V(MyString("a"))},
	{V(uint64('a')), V(MyString("a"))},
	{V(uintptr('a')), V(MyString("a"))},
	{V(int(-1)), V(MyString("\uFFFD"))},
	{V(int8(-2)), V(MyString("\uFFFD"))},
	{V(int16(-3)), V(MyString("\uFFFD"))},
	{V(int32(-4)), V(MyString("\uFFFD"))},
	{V(int64(-5)), V(MyString("\uFFFD"))},
	{V(uint(0x110001)), V(MyString("\uFFFD"))},
	{V(uint32(0x110002)), V(MyString("\uFFFD"))},
	{V(uint64(0x110003)), V(MyString("\uFFFD"))},
	{V(uintptr(0x110004)), V(MyString("\uFFFD"))},

	// named []byte
	{V(string("bytes1")), V(MyBytes("bytes1"))},
	{V(MyBytes("bytes2")), V(string("bytes2"))},
	{V(MyBytes("bytes3")), V(MyBytes("bytes3"))},
	{V(MyString("bytes1")), V(MyBytes("bytes1"))},
	{V(MyBytes("bytes2")), V(MyString("bytes2"))},

	// named []rune
	{V(string("runes♝")), V(MyRunes("runes♝"))},
	{V(MyRunes("runes♕")), V(string("runes♕"))},
	{V(MyRunes("runes🙈🙉🙊")), V(MyRunes("runes🙈🙉🙊"))},
	{V(MyString("runes♝")), V(MyRunes("runes♝"))},
	{V(MyRunes("runes♕")), V(MyString("runes♕"))},

	// slice to array
	{V([]byte(nil)), V([0]byte{})},
	{V([]byte{}), V([0]byte{})},
	{V([]byte{1}), V([1]byte{1})},
	{V([]byte{1, 2}), V([2]byte{1, 2})},
	{V([]byte{1, 2, 3}), V([3]byte{1, 2, 3})},
	{V(MyBytes([]byte(nil))), V([0]byte{})},
	{V(MyBytes{}), V([0]byte{})},
	{V(MyBytes{1}), V([1]byte{1})},
	{V(MyBytes{1, 2}), V([2]byte{1, 2})},
	{V(MyBytes{1, 2, 3}), V([3]byte{1, 2, 3})},
	{V([]byte(nil)), V(MyBytesArray0{})},
	{V([]byte{}), V(MyBytesArray0([0]byte{}))},
	{V([]byte{1, 2, 3, 4}), V(MyBytesArray([4]byte{1, 2, 3, 4}))},
	{V(MyBytes{}), V(MyBytesArray0([0]byte{}))},
	{V(MyBytes{5, 6, 7, 8}), V(MyBytesArray([4]byte{5, 6, 7, 8}))},
	{V([]MyByte{}), V([0]MyByte{})},
	{V([]MyByte{1, 2}), V([2]MyByte{1, 2})},

	// slice to array pointer
	{V([]byte(nil)), V((*[0]byte)(nil))},
	{V([]byte{}), V(new([0]byte))},
	{V([]byte{7}), V(&[1]byte{7})},
	{V(MyBytes([]byte(nil))), V((*[0]byte)(nil))},
	{V(MyBytes([]byte{})), V(new([0]byte))},
	{V(MyBytes([]byte{9})), V(&[1]byte{9})},
	{V([]byte(nil)), V(MyBytesArrayPtr0(nil))},
	{V([]byte{}), V(MyBytesArrayPtr0(new([0]byte)))},
	{V([]byte{1, 2, 3, 4}), V(MyBytesArrayPtr(&[4]byte{1, 2, 3, 4}))},
	{V(MyBytes([]byte{})), V(MyBytesArrayPtr0(new([0]byte)))},
	{V(MyBytes([]byte{5, 6, 7, 8})), V(MyBytesArrayPtr(&[4]byte{5, 6, 7, 8}))},

	{V([]byte(nil)), V((*MyBytesArray0)(nil))},
	{V([]byte{}), V((*MyBytesArray0)(new([0]byte)))},
	{V([]byte{1, 2, 3, 4}), V(&MyBytesArray{1, 2, 3, 4})},
	{V(MyBytes([]byte(nil))), V((*MyBytesArray0)(nil))},
	{V(MyBytes([]byte{})), V((*MyBytesArray0)(new([0]byte)))},
	{V(MyBytes([]byte{5, 6, 7, 8})), V(&MyBytesArray{5, 6, 7, 8})},
	{V(new([0]byte)), V(new(MyBytesArray0))},
	{V(new(MyBytesArray0)), V(new([0]byte))},
	{V(MyBytesArrayPtr0(nil)), V((*[0]byte)(nil))},
	{V((*[0]byte)(nil)), V(MyBytesArrayPtr0(nil))},

	// named types and equal underlying types
	{V(new(int)), V(new(integer))},
	{V(new(integer)), V(new(int))},
	{V(Empty{}), V(struct{}{})},
	{V(new(Empty)), V(new(struct{}))},
	{V(struct{}{}), V(Empty{})},
	{V(new(struct{})), V(new(Empty))},
	{V(Empty{}), V(Empty{})},
	{V(MyBytes{}), V([]byte{})},
	{V([]byte{}), V(MyBytes{})},
	{V((func())(nil)), V(MyFunc(nil))},
	{V((MyFunc)(nil)), V((func())(nil))},

	// structs with different tags
	{V(struct {
		x int `some:"foo"`
	}{}), V(struct {
		x int `some:"bar"`
	}{})},

	{V(struct {
		x int `some:"bar"`
	}{}), V(struct {
		x int `some:"foo"`
	}{})},

	{V(MyStruct{}), V(struct {
		x int `some:"foo"`
	}{})},

	{V(struct {
		x int `some:"foo"`
	}{}), V(MyStruct{})},

	{V(MyStruct{}), V(struct {
		x int `some:"bar"`
	}{})},

	{V(struct {
		x int `some:"bar"`
	}{}), V(MyStruct{})},

	{V(MyStruct1{}), V(MyStruct2{})},
	{V(MyStruct2{}), V(MyStruct1{})},

	// can convert *byte and *MyByte
	{V((*byte)(nil)), V((*MyByte)(nil))},
	{V((*MyByte)(nil)), V((*byte)(nil))},

	// cannot convert mismatched array sizes
	{V([2]byte{}), V([2]byte{})},
	{V([3]byte{}), V([3]byte{})},
	{V(MyBytesArray0{}), V([0]byte{})},
	{V([0]byte{}), V(MyBytesArray0{})},

	// cannot convert other instances
	{V((**byte)(nil)), V((**byte)(nil))},
	{V((**MyByte)(nil)), V((**MyByte)(nil))},
	{V((chan byte)(nil)), V((chan byte)(nil))},
	{V((chan MyByte)(nil)), V((chan MyByte)(nil))},
	{V(([]byte)(nil)), V(([]byte)(nil))},
	{V(([]MyByte)(nil)), V(([]MyByte)(nil))},
	{V((map[int]byte)(nil)), V((map[int]byte)(nil))},
	{V((map[int]MyByte)(nil)), V((map[int]MyByte)(nil))},
	{V((map[byte]int)(nil)), V((map[byte]int)(nil))},
	{V((map[MyByte]int)(nil)), V((map[MyByte]int)(nil))},
	{V([2]byte{}), V([2]byte{})},
	{V([2]MyByte{}), V([2]MyByte{})},

	// other
	{V((***int)(nil)), V((***int)(nil))},
	{V((***byte)(nil)), V((***byte)(nil))},
	{V((***int32)(nil)), V((***int32)(nil))},
	{V((***int64)(nil)), V((***int64)(nil))},
	{V((chan byte)(nil)), V((chan byte)(nil))},
	{V((chan MyByte)(nil)), V((chan MyByte)(nil))},
	{V((map[int]bool)(nil)), V((map[int]bool)(nil))},
	{V((map[int]byte)(nil)), V((map[int]byte)(nil))},
	{V((map[uint]bool)(nil)), V((map[uint]bool)(nil))},
	{V([]uint(nil)), V([]uint(nil))},
	{V([]int(nil)), V([]int(nil))},
	{V(new(any)), V(new(any))},
	{V(new(io.Reader)), V(new(io.Reader))},
	{V(new(io.Writer)), V(new(io.Writer))},

	// channels
	{V(IntChan(nil)), V((chan<- int)(nil))},
	{V(IntChan(nil)), V((<-chan int)(nil))},
	{V((chan int)(nil)), V(IntChanRecv(nil))},
	{V((chan int)(nil)), V(IntChanSend(nil))},
	{V(IntChanRecv(nil)), V((<-chan int)(nil))},
	{V((<-chan int)(nil)), V(IntChanRecv(nil))},
	{V(IntChanSend(nil)), V((chan<- int)(nil))},
	{V((chan<- int)(nil)), V(IntChanSend(nil))},
	{V(IntChan(nil)), V((chan int)(nil))},
	{V((chan int)(nil)), V(IntChan(nil))},
	{V((chan int)(nil)), V((<-chan int)(nil))},
	{V((chan int)(nil)), V((chan<- int)(nil))},
	{V(BytesChan(nil)), V((chan<- []byte)(nil))},
	{V(BytesChan(nil)), V((<-chan []byte)(nil))},
	{V((chan []byte)(nil)), V(BytesChanRecv(nil))},
	{V((chan []byte)(nil)), V(BytesChanSend(nil))},
	{V(BytesChanRecv(nil)), V((<-chan []byte)(nil))},
	{V((<-chan []byte)(nil)), V(BytesChanRecv(nil))},
	{V(BytesChanSend(nil)), V((chan<- []byte)(nil))},
	{V((chan<- []byte)(nil)), V(BytesChanSend(nil))},
	{V(BytesChan(nil)), V((chan []byte)(nil))},
	{V((chan []byte)(nil)), V(BytesChan(nil))},
	{V((chan []byte)(nil)), V((<-chan []byte)(nil))},
	{V((chan []byte)(nil)), V((chan<- []byte)(nil))},

	// cannot convert other instances (channels)
	{V(IntChan(nil)), V(IntChan(nil))},
	{V(IntChanRecv(nil)), V(IntChanRecv(nil))},
	{V(IntChanSend(nil)), V(IntChanSend(nil))},
	{V(BytesChan(nil)), V(BytesChan(nil))},
	{V(BytesChanRecv(nil)), V(BytesChanRecv(nil))},
	{V(BytesChanSend(nil)), V(BytesChanSend(nil))},

	// interfaces
	{V(int(1)), EmptyInterfaceV(int(1))},
	{V(string("hello")), EmptyInterfaceV(string("hello"))},
	{V(new(bytes.Buffer)), ReaderV(new(bytes.Buffer))},
	{ReadWriterV(new(bytes.Buffer)), ReaderV(new(bytes.Buffer))},
	{V(new(bytes.Buffer)), ReadWriterV(new(bytes.Buffer))},
}

func TestConvert(t *testingT) {
	canConvert := map[[2]reflect.Type]bool{}
	all := map[reflect.Type]bool{}

	for _, tt := range convertTests {
		t1 := tt.in.Type()
		if !t1.ConvertibleTo(t1) {
			t.Errorf("(%s).ConvertibleTo(%s) = false, want true", t1, t1)
			continue
		}

		t2 := tt.out.Type()
		if !t1.ConvertibleTo(t2) {
			t.Errorf("(%s).ConvertibleTo(%s) = false, want true", t1, t2)
			continue
		}

		all[t1] = true
		all[t2] = true
		canConvert[[2]reflect.Type{t1, t2}] = true

		// vout1 represents the in value converted to the in type.
		v1 := tt.in
		if !v1.CanConvert(t1) {
			t.Errorf("ValueOf(%T(%[1]v)).CanConvert(%s) = false, want true", tt.in.Interface(), t1)
		}
		vout1 := v1.Convert(t1)
		out1 := vout1.Interface()
		if vout1.Type() != tt.in.Type() || !reflect.DeepEqual(out1, tt.in.Interface()) {
			t.Errorf("ValueOf(%T(%[1]v)).Convert(%s) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t1, out1, tt.in.Interface())
		}

		// vout2 represents the in value converted to the out type.
		if !v1.CanConvert(t2) {
			t.Errorf(" ValueOf(%T(%[1]v)).CanConvert(%s) = false, want true", tt.in.Interface(), t2)
		}
		vout2 := v1.Convert(t2)
		out2 := vout2.Interface()
		if vout2.Type() != tt.out.Type() || !reflect.DeepEqual(out2, tt.out.Interface()) {
			t.Errorf("ValueOf(%T(%[1]v)).Convert(%s) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t2, out2, tt.out.Interface())
		}
		if got, want := vout2.Kind(), vout2.Type().Kind(); got != want {
			t.Errorf("ValueOf(%T(%[1]v)).Convert(%s) has internal kind %v want %v", tt.in.Interface(), t1, got, want)
		}

		// vout3 represents a new value of the out type, set to vout2.  This makes
		// sure the converted value vout2 is really usable as a regular value.
		vout3 := reflect.New(t2).Elem()
		vout3.Set(vout2)
		out3 := vout3.Interface()
		if vout3.Type() != tt.out.Type() || !reflect.DeepEqual(out3, tt.out.Interface()) {
			t.Errorf("Set(ValueOf(%T(%[1]v)).Convert(%s)) = %T(%[3]v), want %T(%[4]v)", tt.in.Interface(), t2, out3, tt.out.Interface())
		}

		if IsRO(v1) {
			t.Errorf("table entry %v is RO, should not be", v1)
		}
		if IsRO(vout1) {
			t.Errorf("self-conversion output %v is RO, should not be", vout1)
		}
		if IsRO(vout2) {
			t.Errorf("conversion output %v is RO, should not be", vout2)
		}
		if IsRO(vout3) {
			t.Errorf("set(conversion output) %v is RO, should not be", vout3)
		}
		if !IsRO(MakeRO(v1).Convert(t1)) {
			t.Errorf("RO self-conversion output %v is not RO, should be", v1)
		}
		if !IsRO(MakeRO(v1).Convert(t2)) {
			t.Errorf("RO conversion output %v is not RO, should be", v1)
		}
	}

	// Assume that of all the types we saw during the tests,
	// if there wasn't an explicit entry for a conversion between
	// a pair of types, then it's not to be allowed. This checks for
	// things like 'int64' converting to '*int'.
	for t1 := range all {
		for t2 := range all {
			expectOK := t1 == t2 || canConvert[[2]reflect.Type{t1, t2}] || t2.Kind() == reflect.Interface && t2.NumMethod() == 0
			ok := t1.ConvertibleTo(t2)
			if ok != expectOK {
				t.Errorf("@(%s).ConvertibleTo(%s) = %v, want %v: %v", t1, t2, ok, expectOK, canConvert[[2]reflect.Type{t1, t2}])
			}
		}
	}
}

func TestConvertPanic(t *testingT) {
	s := make([]byte, 4)
	p := new([8]byte)
	v := reflect.ValueOf(s)
	pt := reflect.TypeOf(p)
	if !v.Type().ConvertibleTo(pt) {
		t.Errorf("[]byte should be convertible to *[8]byte")
	}
	if v.CanConvert(pt) {
		t.Errorf("slice with length 4 should not be convertible to *[8]byte")
	}
	shouldPanic("reflect: cannot convert slice with length 4 to pointer to array with length 8", func() {
		_ = v.Convert(pt)
	})

	if v.CanConvert(pt.Elem()) {
		t.Errorf("slice with length 4 should not be convertible to [8]byte")
	}
	shouldPanic("reflect: cannot convert slice with length 4 to array with length 8", func() {
		_ = v.Convert(pt.Elem())
	})
}

func TestConvertSlice2Array(t *testingT) {
	s := make([]int, 4)
	p := [4]int{}
	pt := reflect.TypeOf(p)
	ov := reflect.ValueOf(s)
	v := ov.Convert(pt)
	// Converting a slice to non-empty array needs to return
	// a non-addressable copy of the original memory.
	if v.CanAddr() {
		t.Fatalf("convert slice to non-empty array returns a addressable copy array")
	}
	for i := range s {
		ov.Index(i).Set(reflect.ValueOf(i + 1))
	}
	for i := range s {
		if v.Index(i).Int() != 0 {
			t.Fatalf("slice (%v) mutation visible in converted result (%v)", ov, v)
		}
	}
}

var gFloat32 float32

const snan uint32 = 0x7f800001

func TestConvertNaNs(t *testingT) {
	// Test to see if a store followed by a load of a signaling NaN
	// maintains the signaling bit. (This used to fail on the 387 port.)
	gFloat32 = math.Float32frombits(snan)
	// runtime.Gosched() // make sure we don't optimize the store/load away
	if got := math.Float32bits(gFloat32); got != snan {
		t.Errorf("store/load of sNaN not faithful, got %x want %x", got, snan)
	}
	// Test reflect's conversion between float32s. See issue 36400.
	type myFloat32 float32
	x := V(myFloat32(math.Float32frombits(snan)))
	y := x.Convert(reflect.TypeOf(float32(0)))
	z := y.Interface().(float32)
	if got := math.Float32bits(z); got != snan {
		t.Errorf("signaling nan conversion got %x, want %x", got, snan)
	}
}

func shouldPanic(expect string, f func()) {
	defer func() {
		r := recover()
		if r == nil {
			panic("did not panic")
		}
		if expect != "" {
			var s string
			switch r := r.(type) {
			case string:
				s = r
			case *reflect.ValueError:
				s = r.Error()
			default:
				panic(fmt.Sprintf("panicked with unexpected type %T", r))
			}
			if !strings.HasPrefix(s, "reflect") {
				panic(`panic string does not start with "reflect": ` + s)
			}
			if !strings.Contains(s, expect) {
				panic(`panic string does not contain "` + expect + `": ` + s)
			}
		}
	}()
	f()
}
