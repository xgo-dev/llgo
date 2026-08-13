// LITTEST
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
)

var minhexdigits = 0

type slice struct {
	array unsafe.Pointer
	len   int
	cap   int
}

type stringStruct struct {
	str unsafe.Pointer
	len int
}

// Keep the output literals tied to the helpers that select them without
// depending on the compiler-assigned numeric global names.
// CHECK-DAG: [[FMT_CHAR:@[0-9]+]] = private unnamed_addr constant [3 x i8] c"%c\00"
// CHECK-DAG: [[STR_LPAREN:@[0-9]+]] = private unnamed_addr constant [1 x i8] c"("
// CHECK-DAG: [[STR_ICLOSE:@[0-9]+]] = private unnamed_addr constant [2 x i8] c"i)"
// CHECK-DAG: [[STR_TRUE:@[0-9]+]] = private unnamed_addr constant [4 x i8] c"true"
// CHECK-DAG: [[STR_FALSE:@[0-9]+]] = private unnamed_addr constant [5 x i8] c"false"
// CHECK-DAG: [[STR_NAN:@[0-9]+]] = private unnamed_addr constant [3 x i8] c"NaN"
// CHECK-DAG: [[STR_PINF:@[0-9]+]] = private unnamed_addr constant [4 x i8] c"+Inf"
// CHECK-DAG: [[STR_NINF:@[0-9]+]] = private unnamed_addr constant [4 x i8] c"-Inf"
// CHECK-DAG: [[HEX_DIGITS:@[0-9]+]] = private unnamed_addr constant [16 x i8] c"0123456789abcdef"
// CHECK-DAG: [[STR_MINUS:@[0-9]+]] = private unnamed_addr constant [1 x i8] c"-"
// CHECK-DAG: [[STR_SPACE:@[0-9]+]] = private unnamed_addr constant [1 x i8] c" "
// CHECK-DAG: [[STR_NL:@[0-9]+]] = private unnamed_addr constant [1 x i8] c"\0A"

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.Slice" @main.bytes(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK: [[BYTES_STRING_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT: store %"{{.*}}/runtime/internal/runtime.String" %0, ptr [[BYTES_STRING_ADDR]]
// CHECK: [[BYTES_SLICE_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK: [[BYTES_STRING_STRUCT:%[0-9]+]] = call ptr @main.stringStructOf(ptr [[BYTES_STRING_ADDR]])
// CHECK: [[BYTES_DATA_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringStruct, ptr [[BYTES_STRING_STRUCT]], i32 0, i32 0
// CHECK-NEXT: [[BYTES_DATA:%[0-9]+]] = load ptr, ptr [[BYTES_DATA_FIELD]]
// CHECK: [[BYTES_SLICE_DATA:%[0-9]+]] = getelementptr inbounds %main.slice, ptr [[BYTES_SLICE_ADDR]], i32 0, i32 0
// CHECK-NEXT: store ptr [[BYTES_DATA]], ptr [[BYTES_SLICE_DATA]]
// CHECK: [[BYTES_LEN_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringStruct, ptr [[BYTES_STRING_STRUCT]], i32 0, i32 1
// CHECK-NEXT: [[BYTES_LEN:%[0-9]+]] = load i64, ptr [[BYTES_LEN_FIELD]]
// CHECK: store i64 [[BYTES_LEN]], ptr %{{[0-9]+}}
// CHECK: [[BYTES_CAP_FIELD:%[0-9]+]] = getelementptr inbounds %main.stringStruct, ptr [[BYTES_STRING_STRUCT]], i32 0, i32 1
// CHECK-NEXT: [[BYTES_CAP:%[0-9]+]] = load i64, ptr [[BYTES_CAP_FIELD]]
// CHECK: store i64 [[BYTES_CAP]], ptr %{{[0-9]+}}
// CHECK: [[BYTES_RESULT:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.Slice", ptr [[BYTES_SLICE_ADDR]]
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.Slice" [[BYTES_RESULT]]

func bytes(s string) (ret []byte) {
	rp := (*slice)(unsafe.Pointer(&ret))
	sp := stringStructOf(&s)
	rp.array = sp.str
	rp.len = sp.len
	rp.cap = sp.len
	return
}

// CHECK-LABEL: define void @main.gwrite(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK: [[GWRITE_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK-NEXT: [[GWRITE_EMPTY:%[0-9]+]] = icmp eq i64 [[GWRITE_LEN]], 0
// CHECK-NEXT: br i1 [[GWRITE_EMPTY]], label %{{.*}}, label %{{.*}}
// CHECK: [[GWRITE_RANGE_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: [[GWRITE_INDEX:%[0-9]+]] = add i64 %{{[0-9]+}}, 1
// CHECK-NEXT: [[GWRITE_MORE:%[0-9]+]] = icmp slt i64 [[GWRITE_INDEX]], [[GWRITE_RANGE_LEN]]
// CHECK: [[GWRITE_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK: [[GWRITE_BOUND:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 [[GWRITE_INDEX]], i1 true, i64 [[GWRITE_BOUND]])
// CHECK-NEXT: [[GWRITE_BYTE_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr [[GWRITE_DATA]], i64 [[GWRITE_INDEX]]
// CHECK-NEXT: [[GWRITE_BYTE:%[0-9]+]] = load i8, ptr [[GWRITE_BYTE_PTR]]
// CHECK-NEXT: {{%[0-9]+}} = call i32 (ptr, ...) @printf(ptr [[FMT_CHAR]], i8 [[GWRITE_BYTE]])

func gwrite(b []byte) {
	if len(b) == 0 {
		return
	}
	for _, v := range b {
		c.Printf(c.Str("%c"), v)
	}
}

// The driver only checks that representative source constants reach the
// helpers. The helper checks below own the printing and boxing contracts.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK: call void @main.printuint(i64 1024)
// CHECK: call void @main.printhex(i64 305441743)
// CHECK: call void @main.prinxor(i64 1)
// CHECK: call void @main.prinsub(i64 100)
// CHECK: call void @main.prinusub(i64 -1)
// CHECK: call void @main.prinfsub(double 1.001000e+02)
// CHECK: [[MAIN_F32_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT: store float 1.000000e+09, ptr [[MAIN_F32_ADDR]]
// CHECK-NEXT: [[MAIN_F32_BOX:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr [[MAIN_F32_ADDR]], 1
// CHECK-NEXT: call void @main.printany(%"{{.*}}/runtime/internal/runtime.eface" [[MAIN_F32_BOX]])
// CHECK: [[MAIN_F64_ADDR:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: store double 2.000000e+09, ptr [[MAIN_F64_ADDR]]
// CHECK-NEXT: [[MAIN_F64_BOX:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr [[MAIN_F64_ADDR]], 1
// CHECK-NEXT: call void @main.printany(%"{{.*}}/runtime/internal/runtime.eface" [[MAIN_F64_BOX]])

func main() {
	printstring("llgo")
	printnl()
	printuint(1024)
	printnl()
	printhex(0x1234abcf)
	printnl()
	prinxor(1)
	printnl()
	prinsub(100)
	printnl()
	prinusub(1<<64 - 1)
	printnl()
	prinfsub(100.1)
	printnl()
	printany(float32(1e9))
	printnl()
	printany(float64(2e9))
	printnl()
	var b bool = true
	if b == true && b != false {
		println("check bool", b)
	}
	n1 := 0b1001
	n2 := 0b0011
	println("check &^", n1&^n2 == 0b1000, n2&^n1 == 0b0010)
	println(true, false, 'a', 'A', rune('中'),
		int8(1), int16(2), int32(3), int64(4), 5,
		uint8(1), uint16(2), uint32(3), uint64(4), uintptr(5),
		"llgo")
	println(1 + 2i)
}

// CHECK-LABEL: define void @main.prinfsub(double %0){{.*}} {
// CHECK: [[PRINFSUB_NEG:%[0-9]+]] = fneg double %0
// CHECK-NEXT: call void @main.printfloat(double [[PRINFSUB_NEG]])

func prinfsub(n float64) {
	printfloat(-n)
}

// CHECK-LABEL: define void @main.prinsub(i64 %0){{.*}} {
// CHECK: [[PRINSUB_NEG:%[0-9]+]] = sub i64 0, %0
// CHECK-NEXT: call void @main.printint(i64 [[PRINSUB_NEG]])

func prinsub(n int64) {
	printint(-n)
}

// CHECK-LABEL: define void @main.printany(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK: [[PA_TYPE0:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: [[PA_IS_BOOL:%[0-9]+]] = icmp eq ptr [[PA_TYPE0]], @_llgo_bool
// CHECK: call void @main.printbool(i1 [[PA_BOOL:%[0-9]+]])
// CHECK: [[PA_TYPE1:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: {{%[0-9]+}} = icmp eq ptr [[PA_TYPE1]], @_llgo_int
// CHECK: call void @main.printint(i64 [[PA_INT:%[0-9]+]])
// CHECK: [[PA_TYPE2:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: {{%[0-9]+}} = icmp eq ptr [[PA_TYPE2]], @_llgo_int8
// CHECK: [[PA_INT8_EXT:%[0-9]+]] = sext i8 [[PA_INT8:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printint(i64 [[PA_INT8_EXT]])
// CHECK: [[PA_TYPE3:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE3]], @_llgo_int16
// CHECK: [[PA_INT16_EXT:%[0-9]+]] = sext i16 [[PA_INT16:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printint(i64 [[PA_INT16_EXT]])
// CHECK: [[PA_TYPE4:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE4]], @_llgo_int32
// CHECK: [[PA_INT32_EXT:%[0-9]+]] = sext i32 [[PA_INT32:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printint(i64 [[PA_INT32_EXT]])
// CHECK: [[PA_TYPE5:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE5]], @_llgo_int64
// CHECK: call void @main.printint(i64 [[PA_INT64:%[0-9]+]])
// CHECK: [[PA_TYPE6:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE6]], @_llgo_uint
// CHECK: call void @main.printuint(i64 [[PA_UINT:%[0-9]+]])
// CHECK: [[PA_TYPE7:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE7]], @_llgo_uint8
// CHECK: [[PA_UINT8_EXT:%[0-9]+]] = zext i8 [[PA_UINT8:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printuint(i64 [[PA_UINT8_EXT]])
// CHECK: [[PA_TYPE8:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE8]], @_llgo_uint16
// CHECK: [[PA_UINT16_EXT:%[0-9]+]] = zext i16 [[PA_UINT16:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printuint(i64 [[PA_UINT16_EXT]])
// CHECK: [[PA_TYPE9:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE9]], @_llgo_uint32
// CHECK: [[PA_UINT32_EXT:%[0-9]+]] = zext i32 [[PA_UINT32:%[0-9]+]] to i64
// CHECK-NEXT: call void @main.printuint(i64 [[PA_UINT32_EXT]])
// CHECK: [[PA_TYPE10:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE10]], @_llgo_uint64
// CHECK: call void @main.printuint(i64 [[PA_UINT64:%[0-9]+]])
// CHECK: [[PA_TYPE11:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE11]], @_llgo_uintptr
// CHECK: call void @main.printuint(i64 [[PA_UINTPTR:%[0-9]+]])
// CHECK: [[PA_TYPE12:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE12]], @_llgo_float32
// CHECK: [[PA_FLOAT32_EXT:%[0-9]+]] = fpext float [[PA_FLOAT32:%[0-9]+]] to double
// CHECK-NEXT: call void @main.printfloat(double [[PA_FLOAT32_EXT]])
// CHECK: [[PA_TYPE13:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE13]], @_llgo_float64
// CHECK: call void @main.printfloat(double [[PA_FLOAT64:%[0-9]+]])
// CHECK: [[PA_TYPE14:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE14]], @_llgo_complex64
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_LPAREN]], i64 1 })
// CHECK-NEXT: [[PA_COMPLEX64_REAL:%[0-9]+]] = extractvalue { float, float } [[PA_COMPLEX64:%[0-9]+]], 0
// CHECK-NEXT: [[PA_COMPLEX64_REAL_EXT:%[0-9]+]] = fpext float [[PA_COMPLEX64_REAL]] to double
// CHECK-NEXT: call void @main.printfloat(double [[PA_COMPLEX64_REAL_EXT]])
// CHECK-NEXT: [[PA_COMPLEX64_IMAG:%[0-9]+]] = extractvalue { float, float } [[PA_COMPLEX64]], 1
// CHECK-NEXT: [[PA_COMPLEX64_IMAG_EXT:%[0-9]+]] = fpext float [[PA_COMPLEX64_IMAG]] to double
// CHECK-NEXT: call void @main.printfloat(double [[PA_COMPLEX64_IMAG_EXT]])
// CHECK-NEXT: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_ICLOSE]], i64 2 })
// CHECK: [[PA_TYPE15:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE15]], @_llgo_complex128
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_LPAREN]], i64 1 })
// CHECK-NEXT: [[PA_COMPLEX128_REAL:%[0-9]+]] = extractvalue { double, double } [[PA_COMPLEX128:%[0-9]+]], 0
// CHECK-NEXT: call void @main.printfloat(double [[PA_COMPLEX128_REAL]])
// CHECK-NEXT: [[PA_COMPLEX128_IMAG:%[0-9]+]] = extractvalue { double, double } [[PA_COMPLEX128]], 1
// CHECK-NEXT: call void @main.printfloat(double [[PA_COMPLEX128_IMAG]])
// CHECK-NEXT: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_ICLOSE]], i64 2 })
// CHECK: [[PA_TYPE16:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 0
// CHECK-NEXT: icmp eq ptr [[PA_TYPE16]], @_llgo_string
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" [[PA_STRING:%[0-9]+]])
// Tie representative result values back to the payload loads in the delayed
// type-assertion blocks. This covers scalar, widened, aggregate, and string
// payloads without snapshotting every compiler-generated block.
// CHECK: [[PA_BOOL_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %0, 1
// CHECK-NEXT: [[PA_BOOL_LOAD:%[0-9]+]] = load i1, ptr [[PA_BOOL_DATA]]
// CHECK: [[PA_BOOL_PAIR:%[0-9]+]] = phi { i1, i1 }
// CHECK-NEXT: [[PA_BOOL]] = extractvalue { i1, i1 } [[PA_BOOL_PAIR]], 0
// CHECK: [[PA_INT8]] = extractvalue { i8, i1 } [[PA_INT8_PAIR:%[0-9]+]], 0
// CHECK: [[PA_UINT32]] = extractvalue { i32, i1 } [[PA_UINT32_PAIR:%[0-9]+]], 0
// CHECK: [[PA_FLOAT32]] = extractvalue { float, i1 } [[PA_FLOAT32_PAIR:%[0-9]+]], 0
// CHECK: [[PA_COMPLEX128]] = extractvalue { { double, double }, i1 } [[PA_COMPLEX128_PAIR:%[0-9]+]], 0
// CHECK: [[PA_STRING]] = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } [[PA_STRING_PAIR:%[0-9]+]], 0

func printany(v any) {
	switch v := v.(type) {
	case bool:
		printbool(v)
	case int:
		printint(int64(v))
	case int8:
		printint(int64(v))
	case int16:
		printint(int64(v))
	case int32:
		printint(int64(v))
	case int64:
		printint(int64(v))
	case uint:
		printuint(uint64(v))
	case uint8:
		printuint(uint64(v))
	case uint16:
		printuint(uint64(v))
	case uint32:
		printuint(uint64(v))
	case uint64:
		printuint(uint64(v))
	case uintptr:
		printuint(uint64(v))
	case float32:
		printfloat(float64(v))
	case float64:
		printfloat(float64(v))
	case complex64:
		printstring("(")
		printfloat(float64(real(v)))
		printfloat(float64(imag(v)))
		printstring("i)")
	case complex128:
		printstring("(")
		printfloat(real(v))
		printfloat(imag(v))
		printstring("i)")
	case string:
		printstring(v)
	}
}

// CHECK-LABEL: define void @main.printbool(i1 %0){{.*}} {
// CHECK: br i1 %0, label %{{.*}}, label %{{.*}}
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_TRUE]], i64 4 })
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_FALSE]], i64 5 })

func printbool(v bool) {
	if v {
		printstring("true")
	} else {
		printstring("false")
	}
}

// CHECK-LABEL: define void @main.printfloat(double %0){{.*}} {
// CHECK: [[PF_NAN:%[0-9]+]] = fcmp une double %0, %0
// CHECK-NEXT: br i1 [[PF_NAN]], label %{{.*}}, label %{{.*}}
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_NAN]], i64 3 })
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_PINF]], i64 4 })
// CHECK: [[PF_INF_SUM:%[0-9]+]] = fadd double %0, %0
// CHECK-NEXT: [[PF_IS_INF:%[0-9]+]] = fcmp oeq double [[PF_INF_SUM]], %0
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_NINF]], i64 4 })
// CHECK: [[PF_NINF_SUM:%[0-9]+]] = fadd double %0, %0
// CHECK-NEXT: [[PF_IS_NINF:%[0-9]+]] = fcmp oeq double [[PF_NINF_SUM]], %0
// CHECK: [[PF_BUFFER:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 14)
// CHECK: [[PF_IS_ZERO:%[0-9]+]] = fcmp oeq double %0, 0.000000e+00
// CHECK: [[PF_ZERO_SIGN:%[0-9]+]] = fdiv double 1.000000e+00, %0
// CHECK-NEXT: {{%[0-9]+}} = fcmp olt double [[PF_ZERO_SIGN]], 0.000000e+00
// CHECK: [[PF_NEGATED:%[0-9]+]] = fneg double %0
// CHECK: [[PF_SCALE_DOWN:%[0-9]+]] = fdiv double [[PF_NORMAL:%[0-9]+]], 1.000000e+01
// CHECK: [[PF_NORMAL]] = phi double [ %0, %{{.*}} ], [ [[PF_SCALE_DOWN]], %{{.*}} ], [ [[PF_NEGATED]], %{{.*}} ]
// CHECK: [[PF_TOO_LARGE:%[0-9]+]] = fcmp oge double [[PF_NORMAL]], 1.000000e+01
// CHECK: [[PF_SCALE_UP:%[0-9]+]] = fmul double [[PF_SMALL_VALUE:%[0-9]+]], 1.000000e+01
// CHECK: [[PF_SMALL_VALUE]] = phi double [ [[PF_NORMAL]], %{{.*}} ], [ [[PF_SCALE_UP]], %{{.*}} ]
// CHECK: [[PF_TOO_SMALL:%[0-9]+]] = fcmp olt double [[PF_SMALL_VALUE]], 1.000000e+00
// CHECK: [[PF_ROUND_COUNT:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ %{{[0-9]+}}, %{{.*}} ]
// CHECK-NEXT: [[PF_ROUND_MORE:%[0-9]+]] = icmp slt i64 [[PF_ROUND_COUNT]], 7
// CHECK: [[PF_ROUNDED:%[0-9]+]] = fadd double %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT: [[PF_ROUND_OVERFLOW:%[0-9]+]] = fcmp oge double [[PF_ROUNDED]], 1.000000e+01
// CHECK: [[PF_DIGIT_INDEX:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ %{{[0-9]+}}, %{{.*}} ]
// CHECK-NEXT: [[PF_MORE_DIGITS:%[0-9]+]] = icmp slt i64 [[PF_DIGIT_INDEX]], 7
// CHECK: [[PF_DIGIT:%[0-9]+]] = fptosi double %{{[0-9]+}} to i64
// CHECK: [[PF_DIGIT_CHAR:%[0-9]+]] = trunc i64 %{{[0-9]+}} to i8
// CHECK: store i8 [[PF_DIGIT_CHAR]], ptr %{{[0-9]+}}
// CHECK: [[PF_REMAINDER:%[0-9]+]] = fsub double %{{[0-9]+}}, %{{[0-9]+}}
// CHECK-NEXT: [[PF_NEXT_DIGIT:%[0-9]+]] = fmul double [[PF_REMAINDER]], 1.000000e+01
// CHECK: store i8 46, ptr %{{[0-9]+}}
// CHECK: store i8 101, ptr %{{[0-9]+}}
// CHECK: [[PF_EXP_NEG:%[0-9]+]] = icmp slt i64 [[PF_EXP:%[0-9]+]], 0
// CHECK: [[PF_EXP_ABS:%[0-9]+]] = phi i64 [ [[PF_EXP]], %{{.*}} ], [ %{{[0-9]+}}, %{{.*}} ]
// CHECK: [[PF_SLICE0:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr [[PF_BUFFER]], 0
// CHECK-NEXT: [[PF_SLICE1:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" [[PF_SLICE0]], i64 14, 1
// CHECK-NEXT: [[PF_SLICE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" [[PF_SLICE1]], i64 14, 2
// CHECK-NEXT: call void @main.gwrite(%"{{.*}}/runtime/internal/runtime.Slice" [[PF_SLICE]])

func printfloat(v float64) {
	switch {
	case v != v:
		printstring("NaN")
		return
	case v+v == v && v > 0:
		printstring("+Inf")
		return
	case v+v == v && v < 0:
		printstring("-Inf")
		return
	}

	const n = 7 // digits printed
	var buf [n + 7]byte
	buf[0] = '+'
	e := 0 // exp
	if v == 0 {
		if 1/v < 0 {
			buf[0] = '-'
		}
	} else {
		if v < 0 {
			v = -v
			buf[0] = '-'
		}

		// normalize
		for v >= 10 {
			e++
			v /= 10
		}
		for v < 1 {
			e--
			v *= 10
		}

		// round
		h := 5.0
		for i := 0; i < n; i++ {
			h /= 10
		}
		v += h
		if v >= 10 {
			e++
			v /= 10
		}
	}

	// format +d.dddd+edd
	for i := 0; i < n; i++ {
		s := int(v)
		buf[i+2] = byte(s + '0')
		v -= float64(s)
		v *= 10
	}
	buf[1] = buf[2]
	buf[2] = '.'

	buf[n+2] = 'e'
	buf[n+3] = '+'
	if e < 0 {
		e = -e
		buf[n+3] = '-'
	}

	buf[n+4] = byte(e/100) + '0'
	buf[n+5] = byte(e/10)%10 + '0'
	buf[n+6] = byte(e%10) + '0'
	gwrite(buf[:])
}

// CHECK-LABEL: define void @main.printhex(i64 %0){{.*}} {
// CHECK: [[PH_BUFFER:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 100)
// CHECK: [[PH_DIGIT:%[0-9]+]] = urem i64 [[PH_VALUE:%[0-9]+]], 16
// CHECK: [[PH_DIGIT_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr [[HEX_DIGITS]], i64 [[PH_DIGIT]]
// CHECK-NEXT: [[PH_DIGIT_CHAR:%[0-9]+]] = load i8, ptr [[PH_DIGIT_PTR]]
// CHECK: [[PH_BUFFER_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr [[PH_BUFFER]], i64 [[PH_INDEX:%[0-9]+]]
// CHECK-NEXT: store i8 [[PH_DIGIT_CHAR]], ptr [[PH_BUFFER_PTR]]
// CHECK: [[PH_LAST_DIGIT:%[0-9]+]] = icmp ult i64 [[PH_VALUE]], 16
// CHECK: store i8 120, ptr %{{[0-9]+}}
// CHECK: store i8 48, ptr %{{[0-9]+}}
// CHECK: [[PH_SLICE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr [[PH_BUFFER]], i64 1, i64 100, i64 %{{[0-9]+}}, i64 100, i1 true, i1 true, i1 true)
// CHECK-NEXT: call void @main.gwrite(%"{{.*}}/runtime/internal/runtime.Slice" [[PH_SLICE]])
// CHECK: [[PH_VALUE]] = phi i64 [ %0, %{{.*}} ], [ [[PH_QUOTIENT:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[PH_INDEX]] = phi i64 [ 99, %{{.*}} ], [ [[PH_PREV_INDEX:%[0-9]+]], %{{.*}} ]
// CHECK: [[PH_QUOTIENT]] = udiv i64 [[PH_VALUE]], 16
// CHECK-NEXT: [[PH_PREV_INDEX]] = sub i64 [[PH_INDEX]], 1
// CHECK: [[PH_WIDTH:%[0-9]+]] = sub i64 100, [[PH_INDEX]]
// CHECK-NEXT: [[PH_MIN_WIDTH:%[0-9]+]] = load i64, ptr @main.minhexdigits
// CHECK-NEXT: [[PH_WIDE_ENOUGH:%[0-9]+]] = icmp sge i64 [[PH_WIDTH]], [[PH_MIN_WIDTH]]

func printhex(v uint64) {
	const dig = "0123456789abcdef"
	var buf [100]byte
	i := len(buf)
	for i--; i > 0; i-- {
		buf[i] = dig[v%16]
		if v < 16 && len(buf)-i >= minhexdigits {
			break
		}
		v /= 16
	}
	i--
	buf[i] = 'x'
	i--
	buf[i] = '0'
	gwrite(buf[i:])
}

// CHECK-LABEL: define void @main.printint(i64 %0){{.*}} {
// CHECK: [[PI_NEGATIVE:%[0-9]+]] = icmp slt i64 %0, 0
// CHECK-NEXT: br i1 [[PI_NEGATIVE]], label %{{.*}}, label %{{.*}}
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_MINUS]], i64 1 })
// CHECK-NEXT: [[PI_ABS:%[0-9]+]] = sub i64 0, %0
// CHECK: [[PI_MAGNITUDE:%[0-9]+]] = phi i64 [ %0, %{{.*}} ], [ [[PI_ABS]], %{{.*}} ]
// CHECK-NEXT: call void @main.printuint(i64 [[PI_MAGNITUDE]])

func printint(v int64) {
	if v < 0 {
		printstring("-")
		v = -v
	}
	printuint(uint64(v))
}

// CHECK-LABEL: define void @main.println(%"{{.*}}/runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK: [[PL_LEN:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: [[PL_INDEX:%[0-9]+]] = add i64 %{{[0-9]+}}, 1
// CHECK-NEXT: [[PL_MORE:%[0-9]+]] = icmp slt i64 [[PL_INDEX]], [[PL_LEN]]
// CHECK: [[PL_DATA:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 0
// CHECK: [[PL_BOUND:%[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %0, 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %{{[0-9]+}}, i64 [[PL_INDEX]], i1 true, i64 [[PL_BOUND]])
// CHECK-NEXT: [[PL_ITEM_PTR:%[0-9]+]] = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr [[PL_DATA]], i64 [[PL_INDEX]]
// CHECK-NEXT: [[PL_ITEM:%[0-9]+]] = load %"{{.*}}/runtime/internal/runtime.eface", ptr [[PL_ITEM_PTR]]
// CHECK: [[PL_NEEDS_SPACE:%[0-9]+]] = icmp ne i64 [[PL_INDEX]], 0
// CHECK: call void @main.printnl()
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_SPACE]], i64 1 })
// CHECK: call void @main.printany(%"{{.*}}/runtime/internal/runtime.eface" [[PL_ITEM]])

func println(args ...any) {
	for i, v := range args {
		if i != 0 {
			printstring(" ")
		}
		printany(v)
	}
	printnl()
}

// CHECK-LABEL: define void @main.printnl(){{.*}} {
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_NL]], i64 1 })

func printnl() {
	printstring("\n")
}

// CHECK-LABEL: define void @main.printsp(){{.*}} {
// CHECK: call void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" { ptr [[STR_SPACE]], i64 1 })

func printsp() {
	printstring(" ")
}

// CHECK-LABEL: define void @main.printstring(%"{{.*}}/runtime/internal/runtime.String" %0){{.*}} {
// CHECK: [[PS_BYTES:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @main.bytes(%"{{.*}}/runtime/internal/runtime.String" %0)
// CHECK-NEXT: call void @main.gwrite(%"{{.*}}/runtime/internal/runtime.Slice" [[PS_BYTES]])

func printstring(s string) {
	gwrite(bytes(s))
}

// CHECK-LABEL: define void @main.printuint(i64 %0){{.*}} {
// CHECK: [[PU_BUFFER:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 100)
// CHECK: [[PU_DIGIT:%[0-9]+]] = urem i64 [[PU_VALUE:%[0-9]+]], 10
// CHECK-NEXT: [[PU_ASCII:%[0-9]+]] = add i64 [[PU_DIGIT]], 48
// CHECK-NEXT: [[PU_CHAR:%[0-9]+]] = trunc i64 [[PU_ASCII]] to i8
// CHECK: [[PU_CHAR_PTR:%[0-9]+]] = getelementptr inbounds i8, ptr [[PU_BUFFER]], i64 [[PU_INDEX:%[0-9]+]]
// CHECK-NEXT: store i8 [[PU_CHAR]], ptr [[PU_CHAR_PTR]]
// CHECK-NEXT: [[PU_LAST_DIGIT:%[0-9]+]] = icmp ult i64 [[PU_VALUE]], 10
// CHECK: [[PU_SLICE:%[0-9]+]] = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr [[PU_BUFFER]], i64 1, i64 100, i64 [[PU_INDEX]], i64 100, i1 true, i1 true, i1 true)
// CHECK-NEXT: call void @main.gwrite(%"{{.*}}/runtime/internal/runtime.Slice" [[PU_SLICE]])
// CHECK: [[PU_VALUE]] = phi i64 [ %0, %{{.*}} ], [ [[PU_QUOTIENT:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[PU_INDEX]] = phi i64 [ 99, %{{.*}} ], [ [[PU_PREV_INDEX:%[0-9]+]], %{{.*}} ]
// CHECK: [[PU_QUOTIENT]] = udiv i64 [[PU_VALUE]], 10
// CHECK-NEXT: [[PU_PREV_INDEX]] = sub i64 [[PU_INDEX]], 1

func printuint(v uint64) {
	var buf [100]byte
	i := len(buf)
	for i--; i > 0; i-- {
		buf[i] = byte(v%10 + '0')
		if v < 10 {
			break
		}
		v /= 10
	}
	gwrite(buf[i:])
}

// CHECK-LABEL: define void @main.prinusub(i64 %0){{.*}} {
// CHECK: [[PRINUSUB_NEG:%[0-9]+]] = sub i64 0, %0
// CHECK-NEXT: call void @main.printuint(i64 [[PRINUSUB_NEG]])

func prinusub(n uint64) {
	printuint(-n)
}

// CHECK-LABEL: define void @main.prinxor(i64 %0){{.*}} {
// CHECK: [[PRINXOR_NOT:%[0-9]+]] = xor i64 %0, -1
// CHECK-NEXT: call void @main.printint(i64 [[PRINXOR_NOT]])

func prinxor(n int64) {
	printint(^n)
}

// CHECK-LABEL: define ptr @main.stringStructOf(ptr %0){{.*}} {
// CHECK: ret ptr %0

func stringStructOf(sp *string) *stringStruct {
	return (*stringStruct)(unsafe.Pointer(sp))
}
