// LITTEST
package main

import (
	"unsafe"
)

var a int64 = 1<<63 - 1
var b int64 = -1 << 63
var n uint64 = 1<<64 - 1

const (
	uvnan    = 0x7FF8000000000001
	uvinf    = 0x7FF0000000000000
	uvneginf = 0xFFF0000000000000
	uvone    = 0x3FF0000000000000
	mask     = 0x7FF
	shift    = 64 - 11 - 1
	bias     = 1023
	signMask = 1 << 63
	fracMask = 1<<shift - 1
)

// CHECK-LABEL: define double @main.Float64frombits(i64 %0){{.*}} {
// CHECK: [[BITS_ADDR:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: store i64 %0, ptr [[BITS_ADDR]]
// CHECK: [[FLOAT:%.*]] = load double, ptr [[BITS_ADDR]]
// CHECK: ret double [[FLOAT]]

func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }

// CHECK-LABEL: define double @main.Inf(i64 %0){{.*}} {
// CHECK: [[NONNEG:%.*]] = icmp sge i64 %0, 0
// CHECK: br i1 [[NONNEG]], label %{{.*}}, label %{{.*}}
// CHECK: [[INF_BITS:%.*]] = phi i64 [ 9218868437227405312, %{{.*}} ], [ -4503599627370496, %{{.*}} ]
// CHECK: [[INF:%.*]] = call double @main.Float64frombits(i64 [[INF_BITS]])
// CHECK: ret double [[INF]]

// Inf returns positive infinity if sign >= 0, negative infinity if sign < 0.
func Inf(sign int) float64 {
	var v uint64
	if sign >= 0 {
		v = uvinf
	} else {
		v = uvneginf
	}
	return Float64frombits(v)
}

// CHECK-LABEL: define i1 @main.IsNaN(double %0){{.*}} {
// CHECK: [[IS_NAN:%.*]] = fcmp une double %0, %0
// CHECK: ret i1 [[IS_NAN]]

func IsNaN(f float64) (is bool) {
	return f != f
}

// CHECK-LABEL: define double @main.NaN(){{.*}} {
// CHECK: [[NAN:%.*]] = call double @main.Float64frombits(i64 9221120237041090561)
// CHECK: ret double [[NAN]]

// NaN returns an IEEE 754 “not-a-number” value.
func NaN() float64 { return Float64frombits(uvnan) }

func demo() {
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// make([]byte, 4, 10) must preserve both the requested length and capacity.
// CHECK: [[D_STORAGE:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 10)
// CHECK: [[D:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr [[D_STORAGE]], i64 1, i64 10, i64 0, i64 4, i1 true, i1 true, i1 true)
// append(s, 5, 6, 7, 8) forwards the materialized four-element tail.
// CHECK: call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringSlice2"(%"{{.*}}String" { ptr @{{.*}}, i64 5 }, i64 5, i64 5, i1 true, i1 true)
// CHECK: [[INT_TAIL:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK: [[INT_TAIL_0:%.*]] = getelementptr inbounds i64, ptr [[INT_TAIL]], i64 0
// CHECK: store i64 5, ptr [[INT_TAIL_0]]
// CHECK: [[INT_TAIL_PTR:%.*]] = extractvalue %"{{.*}}Slice" %{{.*}}, 0
// CHECK: [[INT_TAIL_LEN:%.*]] = extractvalue %"{{.*}}Slice" %{{.*}}, 1
// CHECK: [[INT_APPEND:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}Slice" %{{.*}}, ptr [[INT_TAIL_PTR]], i64 [[INT_TAIL_LEN]], i64 8)
// append(data, "def"...) uses three one-byte elements.
// CHECK: [[DATA_APPEND:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}Slice" %{{.*}}, ptr @{{.*}}, i64 3, i64 1)
// Appending a function closure uses the two-word function representation.
// CHECK: [[FN_PTR:%.*]] = extractvalue %"{{.*}}Slice" [[FN_SLICE:%.*]], 0
// CHECK: [[FN_LEN:%.*]] = extractvalue %"{{.*}}Slice" [[FN_SLICE]], 1
// CHECK: [[FN_APPEND:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.SliceAppend"(%"{{.*}}Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr [[FN_PTR]], i64 [[FN_LEN]], i64 16)
// Both copy forms retain destination/source lengths and the byte element size.
// CHECK: [[COPY_DATA:%.*]] = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}Slice" %{{.*}}, ptr %{{.*}}, i64 %{{.*}}, i64 1)
// CHECK: store i64 [[COPY_DATA]], ptr [[COPY_N:%.*]]
// CHECK: [[COPY_LITERAL:%.*]] = call i64 @"{{.*}}/runtime/internal/runtime.SliceCopy"(%"{{.*}}Slice" %{{.*}}, ptr @{{.*}}, i64 4, i64 1)
// CHECK: store i64 [[COPY_LITERAL]], ptr [[COPY_N]]
// String range is lowered to one iterator whose yielded index/rune are consumed.
// CHECK: [[ITER:%.*]] = call ptr @"{{.*}}/runtime/internal/runtime.NewStringIter"(%"{{.*}}String" { ptr @{{.*}}, i64 7 })
// CHECK: [[NEXT:%.*]] = call { i1, i64, i32 } @"{{.*}}/runtime/internal/runtime.StringIterNext"(ptr [[ITER]])
// CHECK: [[HAS_NEXT:%.*]] = extractvalue { i1, i64, i32 } [[NEXT]], 0
// CHECK: br i1 [[HAS_NEXT]], label %{{.*}}, label %{{.*}}
// Floating-point helpers are exercised with both infinity signs and NaN results.
// CHECK: [[POS_INF:%.*]] = call double @main.Inf(i64 1)
// CHECK: [[NEG_INF:%.*]] = call double @main.Inf(i64 -1)
// CHECK: [[NAN_VALUE:%.*]] = call double @main.NaN()
// CHECK: [[NAN_INPUT:%.*]] = call double @main.NaN()
// CHECK: [[NAN_TRUE:%.*]] = call i1 @main.IsNaN(double [[NAN_INPUT]])
// CHECK: [[NAN_FALSE:%.*]] = call i1 @main.IsNaN(double 1.000000e+00)
// Byte/rune conversions are round-tripped through their originating slices.
// CHECK: [[BYTES:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}String" { ptr @{{.*}}, i64 7 })
// CHECK: [[RUNES:%.*]] = call %"{{.*}}Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}String" { ptr @{{.*}}, i64 7 })
// CHECK: call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringFromBytes"(%"{{.*}}Slice" [[BYTES]])
// CHECK: call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringFromRunes"(%"{{.*}}Slice" [[RUNES]])
// CHECK: [[BYTE_VALUE:%.*]] = load i8, ptr %{{.*}}
// CHECK: [[BYTE64:%.*]] = zext i8 [[BYTE_VALUE]] to i64
// CHECK: call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringFromUint64"(i64 [[BYTE64]])
// CHECK: [[RUNE_VALUE:%.*]] = load i32, ptr %{{.*}}
// CHECK: [[RUNE64:%.*]] = sext i32 [[RUNE_VALUE]] to i64
// CHECK: call %"{{.*}}String" @"{{.*}}/runtime/internal/runtime.StringFromInt64"(i64 [[RUNE64]])

func main() {
	var s = []int{1, 2, 3, 4}
	var a = [...]int{1, 2, 3, 4}
	d := make([]byte, 4, 10)
	println(s, len(s), cap(s))
	println(d, len(d), cap(d))
	println(len(a), cap(a), cap(&a), len(&a))
	println(len([]int{1, 2, 3, 4}), len([4]int{1, 2, 3, 4}))
	println(len(s[1:]), cap(s[1:]), len(s[1:2]), cap(s[1:2]), len(s[1:2:2]), cap(s[1:2:2]))
	println(len(a[1:]), cap(a[1:]), len(a[1:2]), cap(a[1:2]), len(a[1:2:2]), cap(a[1:2:2]))

	println("hello", "hello"[1:], "hello"[1:2], len("hello"[5:]))
	println(append(s, 5, 6, 7, 8))
	data := []byte{'a', 'b', 'c'}
	data = append(data, "def"...)
	println(data)
	fns := []func(){}

	fns = append(fns, func() {})
	println(fns)
	var i any = 100
	println(true, 0, 100, -100, uint(255), int32(-100), 0.0, 100.5, i, &i, uintptr(unsafe.Pointer(&i)))
	var dst [3]byte
	n := copy(dst[:], data)
	println(n, dst[0], dst[1], dst[2])
	n = copy(dst[1:], "ABCD")
	println(n, dst[0], dst[1], dst[2])

	fn1 := demo

	fn2 := func() {
		println("fn")
	}

	fn3 := func() {
		println(n)
	}
	println(demo, fn1, fn2, fn3)

	for i, v := range "中abcd" {
		println(i, v)
	}

	println(Inf(1), Inf(-1), NaN(), IsNaN(NaN()), IsNaN(1.0))

	data1 := []byte("中abcd")
	data2 := []rune("中abcd")
	println(data1, data2)
	println(string(data1), string(data2), string(data1[3]), string(data2[0]))
	s1 := "abc"
	s2 := "abd"
	println(s1 == "abc", s1 == s2, s1 != s2, s1 < s2, s1 <= s2, s1 > s2, s1 >= s2)
}

// CHECK-LABEL: define void @"main.main$3"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK: [[CAPTURE:%.*]] = load { ptr }, ptr %0
// CHECK: [[N_ADDR:%.*]] = extractvalue { ptr } [[CAPTURE]], 0
// CHECK: [[N:%.*]] = load i64, ptr [[N_ADDR]]
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 [[N]])
