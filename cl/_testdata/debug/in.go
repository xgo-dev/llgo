// LITTEST
package main

import "errors"

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [4 x i8] c"done", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"world", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"some error", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"i is 0", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"i:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"a:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"i is 1", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [4 x i8] c"i is", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"b:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"i is 2", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"c:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"d:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [1 x i8] c"a", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [1 x i8] c"b", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"Test error", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [10 x i8] c"globalInt:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [2 x i8] c"s:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [27 x i8] c"called function with struct", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"fn:", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"fn error", align 1{{$}}

type Base struct {
	name string
}

type E struct {
	// Base
	i int
}
type StructWithAllTypeFields struct {
	i8    int8
	i16   int16
	i32   int32
	i64   int64
	i     int
	u8    uint8
	u16   uint16
	u32   uint32
	u64   uint64
	u     uint
	f32   float32
	f64   float64
	b     bool
	c64   complex64
	c128  complex128
	slice []int
	arr   [3]int
	arr2  [3]E
	s     string
	e     E
	pf    *StructWithAllTypeFields // resursive
	pi    *int
	intr  Interface
	m     map[string]uint64
	c     chan int
	err   error
	fn    func(string) (int, error)
	pad1  int
	pad2  int
}

type Interface interface {
	Foo(a []int, b string) int
}

type Struct struct{}

func (s *Struct) Foo(a []int, b string) int {
	return 1
}

func FuncWithAllTypeStructParam(s StructWithAllTypeFields) {
	println(&s)
	// Expected:
	//   all variables: s
	//   s.i8: '\x01'
	//   s.i16: 2
	//   s.i32: 3
	//   s.i64: 4
	//   s.i: 5
	//   s.u8: '\x06'
	//   s.u16: 7
	//   s.u32: 8
	//   s.u64: 9
	//   s.u: 10
	//   s.f32: 11
	//   s.f64: 12
	//   s.b: true
	//   s.c64: complex64{real = 13, imag = 14}
	//   s.c128: complex128{real = 15, imag = 16}
	//   s.slice: []int{21, 22, 23}
	//   s.arr: [3]int{24, 25, 26}
	//   s.arr2: [3]github.com/goplus/llgo/cl/_testdata/debug.E{{i = 27}, {i = 28}, {i = 29}}
	//   s.s: "hello"
	//   s.e: github.com/goplus/llgo/cl/_testdata/debug.E{i = 30}
	//   s.pad1: 100
	//   s.pad2: 200
	s.i8 = '\b'
	// Expected:
	//   s.i8: '\b'
	//   s.i16: 2
	println(len(s.s), s.i8)
}

// Params is a function with all types of parameters.
func FuncWithAllTypeParams(
	i8 int8,
	i16 int16,
	i32 int32,
	i64 int64,
	i int,
	u8 uint8,
	u16 uint16,
	u32 uint32,
	u64 uint64,
	u uint,
	f32 float32,
	f64 float64,
	b bool,
	c64 complex64,
	c128 complex128,
	slice []int,
	arr [3]int,
	arr2 [3]E,
	s string,
	e E,
	f StructWithAllTypeFields,
	pf *StructWithAllTypeFields,
	pi *int,
	intr Interface,
	m map[string]uint64,
	c chan int,
	err error,
	fn func(string) (int, error),
) (int, error) {
	// Expected:
	//   all variables: i8 i16 i32 i64 i u8 u16 u32 u64 u f32 f64 b c64 c128 slice arr arr2 s e f pf pi intr m c err fn
	//   i32: 3
	//   i64: 4
	//   i: 5
	//   u32: 8
	//   u64: 9
	//   u: 10
	//   f32: 11
	//   f64: 12
	//   slice: []int{21, 22, 23}
	//   arr: [3]int{24, 25, 26}
	//   arr2: [3]github.com/goplus/llgo/cl/_testdata/debug.E{{i = 27}, {i = 28}, {i = 29}}
	//   slice[0]: 21
	//   slice[1]: 22
	//   slice[2]: 23
	//   arr[0]: 24
	//   arr[1]: 25
	//   arr[2]: 26
	//   arr2[0].i: 27
	//   arr2[1].i: 28
	//   arr2[2].i: 29
	//   e: github.com/goplus/llgo/cl/_testdata/debug.E{i = 30}

	// Expected(skip):
	//   i8: '\b'
	//   i16: 2
	//   u8: '\x06'
	//   u16: 7
	//   b: true
	println(
		i8, i16, i32, i64, i, u8, u16, u32, u64, u,
		f32, f64, b,
		c64, c128,
		slice, arr[0:],
		s,
		&e,
		&f, pf, pi, intr, m,
		c,
		err,
		fn,
	)
	i8 = 9
	i16 = 10
	i32 = 11
	i64 = 12
	i = 13
	u8 = 14
	u16 = 15
	u32 = 16
	u64 = 17
	u = 18
	f32 = 19
	f64 = 20
	b = false
	c64 = 21 + 22i
	c128 = 23 + 24i
	slice = []int{31, 32, 33}
	arr = [3]int{34, 35, 36}
	arr2 = [3]E{{i: 37}, {i: 38}, {i: 39}}
	s = "world"
	e = E{i: 40}

	println(i8, i16, i32, i64, i, u8, u16, u32, u64, u,
		f32, f64, b,
		c64, c128,
		slice, arr[0:], &arr2,
		s,
		&e,
		&f, pf, pi, intr, m,
		c,
		err,
		fn,
	)
	// Expected:
	//   i8: '\t'
	//   i16: 10
	//   i32: 11
	//   i64: 12
	//   i: 13
	//   u8: '\x0e'
	//   u16: 15
	//   u32: 16
	//   u64: 17
	//   u: 18
	//   f32: 19
	//   f64: 20
	//   b: false
	//   c64: complex64{real = 21, imag = 22}
	//   c128: complex128{real = 23, imag = 24}
	//   slice: []int{31, 32, 33}
	//   arr2: [3]github.com/goplus/llgo/cl/_testdata/debug.E{{i = 37}, {i = 38}, {i = 39}}
	//   s: "world"
	//   e: github.com/goplus/llgo/cl/_testdata/debug.E{i = 40}

	// Expected(skip):
	//   arr: [3]int{34, 35, 36}
	return 1, errors.New("some error")
}

type TinyStruct struct {
	I int
}

type SmallStruct struct {
	I int
	J int
}

type MidStruct struct {
	I int
	J int
	K int
}

type BigStruct struct {
	I int
	J int
	K int
	L int
	M int
	N int
	O int
	P int
	Q int
	R int
}

func FuncStructParams(t TinyStruct, s SmallStruct, m MidStruct, b BigStruct) {
	// println(&t, &s, &m, &b)
	// Expected:
	//   all variables: t s m b
	//   t.I: 1
	//   s.I: 2
	//   s.J: 3
	//   m.I: 4
	//   m.J: 5
	//   m.K: 6
	//   b.I: 7
	//   b.J: 8
	//   b.K: 9
	//   b.L: 10
	//   b.M: 11
	//   b.N: 12
	//   b.O: 13
	//   b.P: 14
	//   b.Q: 15
	//   b.R: 16
	println(t.I, s.I, s.J, m.I, m.J, m.K, b.I, b.J, b.K, b.L, b.M, b.N, b.O, b.P, b.Q, b.R)
	t.I = 10
	s.I = 20
	s.J = 21
	m.I = 40
	m.J = 41
	m.K = 42
	b.I = 70
	b.J = 71
	b.K = 72
	b.L = 73
	b.M = 74
	b.N = 75
	b.O = 76
	b.P = 77
	b.Q = 78
	b.R = 79
	// Expected:
	//   all variables: t s m b
	//   t.I: 10
	//   s.I: 20
	//   s.J: 21
	//   m.I: 40
	//   m.J: 41
	//   m.K: 42
	//   b.I: 70
	//   b.J: 71
	//   b.K: 72
	//   b.L: 73
	//   b.M: 74
	//   b.N: 75
	//   b.O: 76
	//   b.P: 77
	//   b.Q: 78
	//   b.R: 79
	println("done")
}

func FuncStructPtrParams(t *TinyStruct, s *SmallStruct, m *MidStruct, b *BigStruct) {
	// Expected:
	//   all variables: t s m b
	//   t.I: 1
	//   s.I: 2
	//   s.J: 3
	//   m.I: 4
	//   m.J: 5
	//   m.K: 6
	//   b.I: 7
	//   b.J: 8
	//   b.K: 9
	//   b.L: 10
	//   b.M: 11
	//   b.N: 12
	//   b.O: 13
	//   b.P: 14
	//   b.Q: 15
	//   b.R: 16
	println(t, s, m, b)
	t.I = 10
	s.I = 20
	s.J = 21
	m.I = 40
	m.J = 41
	m.K = 42
	b.I = 70
	b.J = 71
	b.K = 72
	b.L = 73
	b.M = 74
	b.N = 75
	b.O = 76
	b.P = 77
	b.Q = 78
	b.R = 79
	// Expected:
	//   all variables: t s m b
	//   t.I: 10
	//   s.I: 20
	//   s.J: 21
	//   m.I: 40
	//   m.J: 41
	//   m.K: 42
	//   b.I: 70
	//   b.J: 71
	//   b.K: 72
	//   b.L: 73
	//   b.M: 74
	//   b.N: 75
	//   b.O: 76
	//   b.P: 77
	//   b.Q: 78
	//   b.R: 79
	println(t.I, s.I, s.J, m.I, m.J, m.K, b.I, b.J, b.K, b.L, b.M, b.N, b.O, b.P, b.Q, b.R)
	println("done")
}

func ScopeIf(branch int) {
	a := 1
	// Expected:
	//   all variables: a branch
	//   a: 1
	if branch == 1 {
		b := 2
		c := 3
		// Expected:
		//   all variables: a b c branch
		//   a: 1
		//   b: 2
		//   c: 3
		//   branch: 1
		println(a, b, c)
	} else {
		c := 3
		d := 4
		// Expected:
		//   all variables: a c d branch
		//   a: 1
		//   c: 3
		//   d: 4
		//   branch: 0
		println(a, c, d)
	}
	// Expected:
	//   all variables: a branch
	//   a: 1
	println("a:", a)
}

func ScopeFor() {
	a := 1
	for i := 0; i < 10; i++ {
		switch i {
		case 0:
			println("i is 0")
			// Expected:
			//   all variables: i a
			//   i: 0
			//   a: 1
			println("i:", i)
		case 1:
			println("i is 1")
			// Expected:
			//   all variables: i a
			//   i: 1
			//   a: 1
			println("i:", i)
		default:
			println("i is", i)
		}
	}
	println("a:", a)
}

func ScopeSwitch(i int) {
	a := 0
	switch i {
	case 1:
		b := 1
		println("i is 1")
		// Expected:
		//   all variables: i a b
		//   i: 1
		//   a: 0
		//   b: 1
		println("i:", i, "a:", a, "b:", b)
	case 2:
		c := 2
		println("i is 2")
		// Expected:
		//   all variables: i a c
		//   i: 2
		//   a: 0
		//   c: 2
		println("i:", i, "a:", a, "c:", c)
	default:
		d := 3
		println("i is", i)
		// Expected:
		//   all variables: i a d
		//   i: 3
		//   a: 0
		//   d: 3
		println("i:", i, "a:", a, "d:", d)
	}
	// Expected:
	//   all variables: a i
	//   a: 0
	println("a:", a)
}

func main() {
	FuncStructParams(TinyStruct{I: 1}, SmallStruct{I: 2, J: 3}, MidStruct{I: 4, J: 5, K: 6}, BigStruct{I: 7, J: 8, K: 9, L: 10, M: 11, N: 12, O: 13, P: 14, Q: 15, R: 16})
	FuncStructPtrParams(&TinyStruct{I: 1}, &SmallStruct{I: 2, J: 3}, &MidStruct{I: 4, J: 5, K: 6}, &BigStruct{I: 7, J: 8, K: 9, L: 10, M: 11, N: 12, O: 13, P: 14, Q: 15, R: 16})
	i := 100
	s := StructWithAllTypeFields{
		i8:    1,
		i16:   2,
		i32:   3,
		i64:   4,
		i:     5,
		u8:    6,
		u16:   7,
		u32:   8,
		u64:   9,
		u:     10,
		f32:   11,
		f64:   12,
		b:     true,
		c64:   13 + 14i,
		c128:  15 + 16i,
		slice: []int{21, 22, 23},
		arr:   [3]int{24, 25, 26},
		arr2:  [3]E{{i: 27}, {i: 28}, {i: 29}},
		s:     "hello",
		e:     E{i: 30},
		pf:    &StructWithAllTypeFields{i16: 100},
		pi:    &i,
		intr:  &Struct{},
		m:     map[string]uint64{"a": 31, "b": 32},
		c:     make(chan int),
		err:   errors.New("Test error"),
		fn: func(s string) (int, error) {
			println("fn:", s)
			i = 201
			return 1, errors.New("fn error")
		},
		pad1: 100,
		pad2: 200,
	}
	// Expected:
	//   all variables: s i err
	//   s.i8: '\x01'
	//   s.i16: 2
	//   s.i32: 3
	//   s.i64: 4
	//   s.i: 5
	//   s.u8: '\x06'
	//   s.u16: 7
	//   s.u32: 8
	//   s.u64: 9
	//   s.u: 10
	//   s.f32: 11
	//   s.f64: 12
	//   s.b: true
	//   s.c64: complex64{real = 13, imag = 14}
	//   s.c128: complex128{real = 15, imag = 16}
	//   s.slice: []int{21, 22, 23}
	//   s.arr: [3]int{24, 25, 26}
	//   s.arr2: [3]github.com/goplus/llgo/cl/_testdata/debug.E{{i = 27}, {i = 28}, {i = 29}}
	//   s.s: "hello"
	//   s.e: github.com/goplus/llgo/cl/_testdata/debug.E{i = 30}
	//   s.pf.i16: 100
	//   *(s.pf).i16: 100
	//   *(s.pi): 100
	globalStructPtr = &s
	globalStruct = s
	println("globalInt:", globalInt)
	// Expected(skip):
	//   all variables: globalInt globalStruct globalStructPtr s i err
	println("s:", &s)
	FuncWithAllTypeStructParam(s)
	println("called function with struct")
	i, err := FuncWithAllTypeParams(
		s.i8, s.i16, s.i32, s.i64, s.i, s.u8, s.u16, s.u32, s.u64, s.u,
		s.f32, s.f64, s.b,
		s.c64, s.c128,
		s.slice, s.arr, s.arr2,
		s.s,
		s.e, s,
		s.pf, s.pi,
		s.intr,
		s.m,
		s.c,
		s.err,
		s.fn,
	)
	println(i, err)
	ScopeIf(1)
	ScopeIf(0)
	ScopeFor()
	ScopeSwitch(1)
	ScopeSwitch(2)
	ScopeSwitch(3)
	println(globalStructPtr)
	println(&globalStruct)
	s.i8 = 0x12
	println(s.i8)
	// Expected:
	//   all variables: s i err
	//   s.i8: '\x12'

	// Expected(skip):
	//   globalStruct.i8: '\x01'
	println((*globalStructPtr).i8)
	println("done")
	println("")
	println(&s, &globalStruct, globalStructPtr.i16, globalStructPtr)
	globalStructPtr = nil
}

var globalInt int = 301
var globalStruct StructWithAllTypeFields
var globalStructPtr *StructWithAllTypeFields

// CHECK-LABEL: define void @main.FuncStructParams(%main.TinyStruct %0, %main.SmallStruct %1, %main.MidStruct %2, %main.BigStruct %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %4 = alloca %main.TinyStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %4, i8 0, i64 8, i1 false)
// CHECK-NEXT:     #dbg_declare(ptr %4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.TinyStruct %0, ptr %4, align 8
// CHECK-NEXT:   %5 = alloca %main.SmallStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %5, i8 0, i64 16, i1 false)
// CHECK-NEXT:     #dbg_declare(ptr %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.SmallStruct %1, ptr %5, align 8
// CHECK-NEXT:   %6 = alloca %main.MidStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %6, i8 0, i64 24, i1 false)
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.MidStruct %2, ptr %6, align 8
// CHECK-NEXT:   %7 = alloca %main.BigStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %7, i8 0, i64 80, i1 false)
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.BigStruct %3, ptr %7, align 8
// CHECK-NEXT:     #dbg_declare(ptr %4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %8 = getelementptr inbounds %main.TinyStruct, ptr %4, i32 0, i32 0
// CHECK-NEXT:   %9 = load i64, ptr %8, align 8
// CHECK-NEXT:     #dbg_declare(ptr %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %10 = getelementptr inbounds %main.SmallStruct, ptr %5, i32 0, i32 0
// CHECK-NEXT:   %11 = load i64, ptr %10, align 8
// CHECK-NEXT:     #dbg_declare(ptr %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %12 = getelementptr inbounds %main.SmallStruct, ptr %5, i32 0, i32 1
// CHECK-NEXT:   %13 = load i64, ptr %12, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %14 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 0
// CHECK-NEXT:   %15 = load i64, ptr %14, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %16 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 1
// CHECK-NEXT:   %17 = load i64, ptr %16, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %18 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 2
// CHECK-NEXT:   %19 = load i64, ptr %18, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %20 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 0
// CHECK-NEXT:   %21 = load i64, ptr %20, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %22 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 1
// CHECK-NEXT:   %23 = load i64, ptr %22, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %24 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 2
// CHECK-NEXT:   %25 = load i64, ptr %24, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %26 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 3
// CHECK-NEXT:   %27 = load i64, ptr %26, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %28 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 4
// CHECK-NEXT:   %29 = load i64, ptr %28, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %30 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 5
// CHECK-NEXT:   %31 = load i64, ptr %30, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %32 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 6
// CHECK-NEXT:   %33 = load i64, ptr %32, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %34 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 7
// CHECK-NEXT:   %35 = load i64, ptr %34, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %36 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 8
// CHECK-NEXT:   %37 = load i64, ptr %36, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %38 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 9
// CHECK-NEXT:   %39 = load i64, ptr %38, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %19)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %27)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %29)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %31)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %33)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %37)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %39)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %40 = getelementptr inbounds %main.TinyStruct, ptr %4, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %40, align 8
// CHECK-NEXT:     #dbg_declare(ptr %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %41 = getelementptr inbounds %main.SmallStruct, ptr %5, i32 0, i32 0
// CHECK-NEXT:   store i64 20, ptr %41, align 8
// CHECK-NEXT:     #dbg_declare(ptr %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %42 = getelementptr inbounds %main.SmallStruct, ptr %5, i32 0, i32 1
// CHECK-NEXT:   store i64 21, ptr %42, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %43 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 0
// CHECK-NEXT:   store i64 40, ptr %43, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %44 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 1
// CHECK-NEXT:   store i64 41, ptr %44, align 8
// CHECK-NEXT:     #dbg_declare(ptr %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %45 = getelementptr inbounds %main.MidStruct, ptr %6, i32 0, i32 2
// CHECK-NEXT:   store i64 42, ptr %45, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %46 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 0
// CHECK-NEXT:   store i64 70, ptr %46, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %47 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 1
// CHECK-NEXT:   store i64 71, ptr %47, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %48 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 2
// CHECK-NEXT:   store i64 72, ptr %48, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %49 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 3
// CHECK-NEXT:   store i64 73, ptr %49, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %50 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 4
// CHECK-NEXT:   store i64 74, ptr %50, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %51 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 5
// CHECK-NEXT:   store i64 75, ptr %51, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %52 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 6
// CHECK-NEXT:   store i64 76, ptr %52, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %53 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 7
// CHECK-NEXT:   store i64 77, ptr %53, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %54 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 8
// CHECK-NEXT:   store i64 78, ptr %54, align 8
// CHECK-NEXT:     #dbg_declare(ptr %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %55 = getelementptr inbounds %main.BigStruct, ptr %7, i32 0, i32 9
// CHECK-NEXT:   store i64 79, ptr %55, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.FuncStructPtrParams(ptr %0, ptr %1, ptr %2, ptr %3){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(ptr %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(ptr %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %4 = getelementptr inbounds %main.TinyStruct, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 10, ptr %4, align 8
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %5 = getelementptr inbounds %main.SmallStruct, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store i64 20, ptr %5, align 8
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %6 = getelementptr inbounds %main.SmallStruct, ptr %1, i32 0, i32 1
// CHECK-NEXT:   store i64 21, ptr %6, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %7 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 0
// CHECK-NEXT:   store i64 40, ptr %7, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %8 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i64 41, ptr %8, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %9 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 2
// CHECK-NEXT:   store i64 42, ptr %9, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %10 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store i64 70, ptr %10, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %11 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i64 71, ptr %11, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %12 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 2
// CHECK-NEXT:   store i64 72, ptr %12, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %13 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 3
// CHECK-NEXT:   store i64 73, ptr %13, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %14 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 4
// CHECK-NEXT:   store i64 74, ptr %14, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %15 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 5
// CHECK-NEXT:   store i64 75, ptr %15, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %16 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 6
// CHECK-NEXT:   store i64 76, ptr %16, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %17 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 7
// CHECK-NEXT:   store i64 77, ptr %17, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %18 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 8
// CHECK-NEXT:   store i64 78, ptr %18, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %19 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 9
// CHECK-NEXT:   store i64 79, ptr %19, align 8
// CHECK-NEXT:     #dbg_value(ptr %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %20 = getelementptr inbounds %main.TinyStruct, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %21 = load i64, ptr %20, align 8
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %22 = getelementptr inbounds %main.SmallStruct, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %23 = load i64, ptr %22, align 8
// CHECK-NEXT:     #dbg_value(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %24 = getelementptr inbounds %main.SmallStruct, ptr %1, i32 0, i32 1
// CHECK-NEXT:   %25 = load i64, ptr %24, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %26 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %27 = load i64, ptr %26, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %28 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %29 = load i64, ptr %28, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %30 = getelementptr inbounds %main.MidStruct, ptr %2, i32 0, i32 2
// CHECK-NEXT:   %31 = load i64, ptr %30, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %32 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 0
// CHECK-NEXT:   %33 = load i64, ptr %32, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %34 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 1
// CHECK-NEXT:   %35 = load i64, ptr %34, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %36 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 2
// CHECK-NEXT:   %37 = load i64, ptr %36, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %38 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 3
// CHECK-NEXT:   %39 = load i64, ptr %38, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %40 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 4
// CHECK-NEXT:   %41 = load i64, ptr %40, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %42 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 5
// CHECK-NEXT:   %43 = load i64, ptr %42, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %44 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 6
// CHECK-NEXT:   %45 = load i64, ptr %44, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %46 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 7
// CHECK-NEXT:   %47 = load i64, ptr %46, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %48 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 8
// CHECK-NEXT:   %49 = load i64, ptr %48, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %50 = getelementptr inbounds %main.BigStruct, ptr %3, i32 0, i32 9
// CHECK-NEXT:   %51 = load i64, ptr %50, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %27)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %29)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %31)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %33)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %35)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %37)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %39)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %41)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %43)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %49)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %51)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.FuncWithAllTypeParams(i8 %0, i16 %1, i32 %2, i64 %3, i64 %4, i8 %5, i16 %6, i32 %7, i64 %8, i64 %9, float %10, double %11, i1 %12, { float, float } %13, { double, double } %14, %"{{.*}}/runtime/internal/runtime.Slice" %15, [3 x i64] %16, [3 x %main.E] %17, %"{{.*}}/runtime/internal/runtime.String" %18, %main.E %19, %main.StructWithAllTypeFields %20, ptr %21, ptr %22, %"{{.*}}/runtime/internal/runtime.iface" %23, ptr %24, ptr %25, %"{{.*}}/runtime/internal/runtime.iface" %26, { ptr, ptr } %27){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(i8 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i8 %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %8, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %9, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(float %10, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(double %11, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i1 %12, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %28 = alloca { float, float }, align 8
// CHECK-NEXT:   store { float, float } %13, ptr %28, align 4
// CHECK-NEXT:   %29 = load { float, float }, ptr %28, align 4
// CHECK-NEXT:     #dbg_value(ptr %28, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %30 = alloca { double, double }, align 8
// CHECK-NEXT:   store { double, double } %14, ptr %30, align 8
// CHECK-NEXT:   %31 = load { double, double }, ptr %30, align 8
// CHECK-NEXT:     #dbg_value(ptr %30, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %32 = alloca { ptr, i64, i64 }, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %15, ptr %32, align 8
// CHECK-NEXT:   %33 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %32, align 8
// CHECK-NEXT:     #dbg_value(ptr %32, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %34 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %18, ptr %34, align 8
// CHECK-NEXT:   %35 = load %"{{.*}}/runtime/internal/runtime.String", ptr %34, align 8
// CHECK-NEXT:     #dbg_value(ptr %34, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %21, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %22, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %36 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %23, ptr %36, align 8
// CHECK-NEXT:   %37 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %36, align 8
// CHECK-NEXT:     #dbg_value(ptr %36, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %38 = alloca { i64, i8, i8, i16, i32, ptr, ptr, i64, ptr }, align 8
// CHECK-NEXT:   store ptr %24, ptr %38, align 8
// CHECK-NEXT:   %39 = load ptr, ptr %38, align 8
// CHECK-NEXT:     #dbg_value(ptr %38, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %40 = alloca ptr, align 8
// CHECK-NEXT:   store ptr %25, ptr %40, align 8
// CHECK-NEXT:   %41 = load ptr, ptr %40, align 8
// CHECK-NEXT:     #dbg_value(ptr %40, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %42 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %26, ptr %42, align 8
// CHECK-NEXT:   %43 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %42, align 8
// CHECK-NEXT:     #dbg_value(ptr %42, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %44 = alloca { ptr, ptr }, align 8
// CHECK-NEXT:   store { ptr, ptr } %27, ptr %44, align 8
// CHECK-NEXT:   %45 = load { ptr, ptr }, ptr %44, align 8
// CHECK-NEXT:     #dbg_value(ptr %44, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:     #dbg_declare(ptr %46, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store [3 x i64] %16, ptr %46, align 8
// CHECK-NEXT:   %47 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:     #dbg_declare(ptr %47, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store [3 x %main.E] %17, ptr %47, align 8
// CHECK-NEXT:   %48 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:     #dbg_declare(ptr %48, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.E %19, ptr %48, align 8
// CHECK-NEXT:   %49 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 288)
// CHECK-NEXT:     #dbg_declare(ptr %49, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.StructWithAllTypeFields %20, ptr %49, align 8
// CHECK-NEXT:     #dbg_value(i8 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 %2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i8 %5, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 %6, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 %7, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %8, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %9, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(float %10, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(double %11, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i1 %12, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %50 = alloca { float, float }, align 8
// CHECK-NEXT:   store { float, float } %13, ptr %50, align 4
// CHECK-NEXT:   %51 = load { float, float }, ptr %50, align 4
// CHECK-NEXT:     #dbg_value(ptr %50, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %52 = alloca { double, double }, align 8
// CHECK-NEXT:   store { double, double } %14, ptr %52, align 8
// CHECK-NEXT:   %53 = load { double, double }, ptr %52, align 8
// CHECK-NEXT:     #dbg_value(ptr %52, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %54 = alloca { ptr, i64, i64 }, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %15, ptr %54, align 8
// CHECK-NEXT:   %55 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %54, align 8
// CHECK-NEXT:     #dbg_value(ptr %54, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %46, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %56 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %46, i64 8, i64 3, i64 0, i64 3, i1 true, i1 true, i1 true)
// CHECK-NEXT:   %57 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %18, ptr %57, align 8
// CHECK-NEXT:   %58 = load %"{{.*}}/runtime/internal/runtime.String", ptr %57, align 8
// CHECK-NEXT:     #dbg_value(ptr %57, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %48, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %49, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %21, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %22, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %59 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %23, ptr %59, align 8
// CHECK-NEXT:   %60 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %59, align 8
// CHECK-NEXT:     #dbg_value(ptr %59, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %61 = alloca { i64, i8, i8, i16, i32, ptr, ptr, i64, ptr }, align 8
// CHECK-NEXT:   store ptr %24, ptr %61, align 8
// CHECK-NEXT:   %62 = load ptr, ptr %61, align 8
// CHECK-NEXT:     #dbg_value(ptr %61, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %63 = alloca ptr, align 8
// CHECK-NEXT:   store ptr %25, ptr %63, align 8
// CHECK-NEXT:   %64 = load ptr, ptr %63, align 8
// CHECK-NEXT:     #dbg_value(ptr %63, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %65 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %26, ptr %65, align 8
// CHECK-NEXT:   %66 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %65, align 8
// CHECK-NEXT:     #dbg_value(ptr %65, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %67 = alloca { ptr, ptr }, align 8
// CHECK-NEXT:   store { ptr, ptr } %27, ptr %67, align 8
// CHECK-NEXT:   %68 = load { ptr, ptr }, ptr %67, align 8
// CHECK-NEXT:     #dbg_value(ptr %67, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %69 = sext i8 %0 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %69)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %70 = sext i16 %1 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %70)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %71 = sext i32 %2 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %71)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %72 = zext i8 %5 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %72)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %73 = zext i16 %6 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %73)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %74 = zext i32 %7 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %74)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %75 = fpext float %10 to double
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %75)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %76 = extractvalue { float, float } %13, 0
// CHECK-NEXT:   %77 = extractvalue { float, float } %13, 1
// CHECK-NEXT:   %78 = fpext float %76 to double
// CHECK-NEXT:   %79 = fpext float %77 to double
// CHECK-NEXT:   %80 = insertvalue { double, double } undef, double %78, 0
// CHECK-NEXT:   %81 = insertvalue { double, double } %80, double %79, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } %81)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } %14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %56)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %48)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %49)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %22)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %26)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %82 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %82)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i8 9, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 10, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 11, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 12, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 13, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i8 14, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 15, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 16, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 17, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 18, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(float 1.900000e+01, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(double 2.000000e+01, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i1 false, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %83 = alloca { float, float }, align 8
// CHECK-NEXT:   store { float, float } { float 2.100000e+01, float 2.200000e+01 }, ptr %83, align 4
// CHECK-NEXT:   %84 = load { float, float }, ptr %83, align 4
// CHECK-NEXT:     #dbg_value(ptr %83, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %85 = alloca { double, double }, align 8
// CHECK-NEXT:   store { double, double } { double 2.300000e+01, double 2.400000e+01 }, ptr %85, align 8
// CHECK-NEXT:   %86 = load { double, double }, ptr %85, align 8
// CHECK-NEXT:     #dbg_value(ptr %85, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %87 = alloca { ptr, i64, i64 }, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %15, ptr %87, align 8
// CHECK-NEXT:   %88 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %87, align 8
// CHECK-NEXT:     #dbg_value(ptr %87, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %89 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %90 = getelementptr inbounds i64, ptr %89, i64 0
// CHECK-NEXT:   store i64 31, ptr %90, align 8
// CHECK-NEXT:   %91 = getelementptr inbounds i64, ptr %89, i64 1
// CHECK-NEXT:   store i64 32, ptr %91, align 8
// CHECK-NEXT:   %92 = getelementptr inbounds i64, ptr %89, i64 2
// CHECK-NEXT:   store i64 33, ptr %92, align 8
// CHECK-NEXT:   %93 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %89, 0
// CHECK-NEXT:   %94 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %93, i64 3, 1
// CHECK-NEXT:   %95 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %94, i64 3, 2
// CHECK-NEXT:     #dbg_declare(ptr %46, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %96 = getelementptr inbounds i64, ptr %46, i64 0
// CHECK-NEXT:   %97 = getelementptr inbounds i64, ptr %46, i64 1
// CHECK-NEXT:   %98 = getelementptr inbounds i64, ptr %46, i64 2
// CHECK-NEXT:   store i64 34, ptr %96, align 8
// CHECK-NEXT:   store i64 35, ptr %97, align 8
// CHECK-NEXT:   store i64 36, ptr %98, align 8
// CHECK-NEXT:     #dbg_declare(ptr %47, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %99 = getelementptr inbounds %main.E, ptr %47, i64 0
// CHECK-NEXT:   %100 = getelementptr inbounds %main.E, ptr %99, i32 0, i32 0
// CHECK-NEXT:   %101 = getelementptr inbounds %main.E, ptr %47, i64 1
// CHECK-NEXT:   %102 = getelementptr inbounds %main.E, ptr %101, i32 0, i32 0
// CHECK-NEXT:   %103 = getelementptr inbounds %main.E, ptr %47, i64 2
// CHECK-NEXT:   %104 = getelementptr inbounds %main.E, ptr %103, i32 0, i32 0
// CHECK-NEXT:   store i64 37, ptr %100, align 8
// CHECK-NEXT:   store i64 38, ptr %102, align 8
// CHECK-NEXT:   store i64 39, ptr %104, align 8
// CHECK-NEXT:   %105 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %105, align 8
// CHECK-NEXT:   %106 = load %"{{.*}}/runtime/internal/runtime.String", ptr %105, align 8
// CHECK-NEXT:     #dbg_value(ptr %105, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %48, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %107 = getelementptr inbounds %main.E, ptr %48, i32 0, i32 0
// CHECK-NEXT:   store i64 40, ptr %107, align 8
// CHECK-NEXT:     #dbg_value(i8 9, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 10, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 11, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 12, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 13, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i8 14, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i16 15, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i32 16, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 17, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 18, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(float 1.900000e+01, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(double 2.000000e+01, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i1 false, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %108 = alloca { float, float }, align 8
// CHECK-NEXT:   store { float, float } { float 2.100000e+01, float 2.200000e+01 }, ptr %108, align 4
// CHECK-NEXT:   %109 = load { float, float }, ptr %108, align 4
// CHECK-NEXT:     #dbg_value(ptr %108, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %110 = alloca { double, double }, align 8
// CHECK-NEXT:   store { double, double } { double 2.300000e+01, double 2.400000e+01 }, ptr %110, align 8
// CHECK-NEXT:   %111 = load { double, double }, ptr %110, align 8
// CHECK-NEXT:     #dbg_value(ptr %110, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %112 = alloca { ptr, i64, i64 }, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %95, ptr %112, align 8
// CHECK-NEXT:   %113 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %112, align 8
// CHECK-NEXT:     #dbg_value(ptr %112, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %46, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %114 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %46, i64 8, i64 3, i64 0, i64 3, i1 true, i1 true, i1 true)
// CHECK-NEXT:     #dbg_declare(ptr %47, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %115 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %115, align 8
// CHECK-NEXT:   %116 = load %"{{.*}}/runtime/internal/runtime.String", ptr %115, align 8
// CHECK-NEXT:     #dbg_value(ptr %115, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %48, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %49, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %21, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(ptr %22, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %117 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %23, ptr %117, align 8
// CHECK-NEXT:   %118 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %117, align 8
// CHECK-NEXT:     #dbg_value(ptr %117, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %119 = alloca { i64, i8, i8, i16, i32, ptr, ptr, i64, ptr }, align 8
// CHECK-NEXT:   store ptr %24, ptr %119, align 8
// CHECK-NEXT:   %120 = load ptr, ptr %119, align 8
// CHECK-NEXT:     #dbg_value(ptr %119, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %121 = alloca ptr, align 8
// CHECK-NEXT:   store ptr %25, ptr %121, align 8
// CHECK-NEXT:   %122 = load ptr, ptr %121, align 8
// CHECK-NEXT:     #dbg_value(ptr %121, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %123 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %26, ptr %123, align 8
// CHECK-NEXT:   %124 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %123, align 8
// CHECK-NEXT:     #dbg_value(ptr %123, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %125 = alloca { ptr, ptr }, align 8
// CHECK-NEXT:   store { ptr, ptr } %27, ptr %125, align 8
// CHECK-NEXT:   %126 = load { ptr, ptr }, ptr %125, align 8
// CHECK-NEXT:     #dbg_value(ptr %125, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 9)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 14)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 16)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 18)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double 1.900000e+01)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintFloat"(double 2.000000e+01)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintBool"(i1 false)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double 2.100000e+01, double 2.200000e+01 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintComplex"({ double, double } { double 2.300000e+01, double 2.400000e+01 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %95)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintSlice"(%"{{.*}}/runtime/internal/runtime.Slice" %114)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %47)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %48)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %49)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %21)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %22)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %23)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %24)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %25)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %26)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %127 = extractvalue { ptr, ptr } %27, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %127)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %128 = call %"{{.*}}/runtime/internal/runtime.iface" @errors.New(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 })
// CHECK-NEXT:   %129 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 1, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %128, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %129
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.FuncWithAllTypeStructParam(%main.StructWithAllTypeFields %0){{.*}} !dbg !{{[0-9]+}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 288)
// CHECK-NEXT:     #dbg_declare(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store %main.StructWithAllTypeFields %0, ptr %1, align 8
// CHECK-NEXT:     #dbg_declare(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %2 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store i8 8, ptr %2, align 1
// CHECK-NEXT:     #dbg_declare(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %3 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %1, i32 0, i32 18
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.String", ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue %"{{.*}}/runtime/internal/runtime.String" %4, 1
// CHECK-NEXT:     #dbg_declare(ptr %1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %6 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %1, i32 0, i32 0
// CHECK-NEXT:   %7 = load i8, ptr %6, align 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %8 = sext i8 %7 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.ScopeFor(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_4, %_llgo_0
// CHECK-NEXT:   %0 = phi i64 [ 0, %_llgo_0 ], [ %3, %_llgo_4 ]
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %1 = icmp slt i64 %0, 10
// CHECK-NEXT:   br i1 %1, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %2 = icmp eq i64 %0, 0
// CHECK-NEXT:   br i1 %2, label %_llgo_5, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_8, %_llgo_6, %_llgo_5
// CHECK-NEXT:   %3 = add i64 %0, 1
// CHECK-NEXT:     #dbg_value(i64 %3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_7
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %4 = icmp eq i64 %0, 1
// CHECK-NEXT:   br i1 %4, label %_llgo_6, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.ScopeIf(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %1 = icmp eq i64 %0, 1
// CHECK-NEXT:   br i1 %1, label %_llgo_1, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:     #dbg_value(i64 2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_3, %_llgo_1
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_0
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 4, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.ScopeSwitch(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %1 = icmp eq i64 %0, 1
// CHECK-NEXT:   br i1 %1, label %_llgo_2, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_5, %_llgo_3, %_llgo_2
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 1, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4
// CHECK-NEXT:     #dbg_value(i64 2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 2, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %2 = icmp eq i64 %0, 2
// CHECK-NEXT:   br i1 %2, label %_llgo_3, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_value(i64 %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_value(i64 3, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 3)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-NEXT: }

// CHECK-LABEL: define i64 @"main.(*Struct).Foo"(ptr %0, %"{{.*}}/runtime/internal/runtime.Slice" %1, %"{{.*}}/runtime/internal/runtime.String" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:     #dbg_value(ptr %0, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %3 = alloca { ptr, i64, i64 }, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %1, ptr %3, align 8
// CHECK-NEXT:   %4 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %3, align 8
// CHECK-NEXT:     #dbg_value(ptr %3, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %5 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %2, ptr %5, align 8
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.String", ptr %5, align 8
// CHECK-NEXT:     #dbg_value(ptr %5, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   ret i64 1
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @errors.init()
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca %main.TinyStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.TinyStruct, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load %main.TinyStruct, ptr %0, align 8
// CHECK-NEXT:   %3 = alloca %main.SmallStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3, i8 0, i64 16, i1 false)
// CHECK-NEXT:   %4 = getelementptr inbounds %main.SmallStruct, ptr %3, i32 0, i32 0
// CHECK-NEXT:   %5 = getelementptr inbounds %main.SmallStruct, ptr %3, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %4, align 8
// CHECK-NEXT:   store i64 3, ptr %5, align 8
// CHECK-NEXT:   %6 = load %main.SmallStruct, ptr %3, align 8
// CHECK-NEXT:   %7 = alloca %main.MidStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %7, i8 0, i64 24, i1 false)
// CHECK-NEXT:   %8 = getelementptr inbounds %main.MidStruct, ptr %7, i32 0, i32 0
// CHECK-NEXT:   %9 = getelementptr inbounds %main.MidStruct, ptr %7, i32 0, i32 1
// CHECK-NEXT:   %10 = getelementptr inbounds %main.MidStruct, ptr %7, i32 0, i32 2
// CHECK-NEXT:   store i64 4, ptr %8, align 8
// CHECK-NEXT:   store i64 5, ptr %9, align 8
// CHECK-NEXT:   store i64 6, ptr %10, align 8
// CHECK-NEXT:   %11 = load %main.MidStruct, ptr %7, align 8
// CHECK-NEXT:   %12 = alloca %main.BigStruct, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %12, i8 0, i64 80, i1 false)
// CHECK-NEXT:   %13 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 0
// CHECK-NEXT:   %14 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 1
// CHECK-NEXT:   %15 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 2
// CHECK-NEXT:   %16 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 3
// CHECK-NEXT:   %17 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 4
// CHECK-NEXT:   %18 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 5
// CHECK-NEXT:   %19 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 6
// CHECK-NEXT:   %20 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 7
// CHECK-NEXT:   %21 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 8
// CHECK-NEXT:   %22 = getelementptr inbounds %main.BigStruct, ptr %12, i32 0, i32 9
// CHECK-NEXT:   store i64 7, ptr %13, align 8
// CHECK-NEXT:   store i64 8, ptr %14, align 8
// CHECK-NEXT:   store i64 9, ptr %15, align 8
// CHECK-NEXT:   store i64 10, ptr %16, align 8
// CHECK-NEXT:   store i64 11, ptr %17, align 8
// CHECK-NEXT:   store i64 12, ptr %18, align 8
// CHECK-NEXT:   store i64 13, ptr %19, align 8
// CHECK-NEXT:   store i64 14, ptr %20, align 8
// CHECK-NEXT:   store i64 15, ptr %21, align 8
// CHECK-NEXT:   store i64 16, ptr %22, align 8
// CHECK-NEXT:   %23 = load %main.BigStruct, ptr %12, align 8
// CHECK-NEXT:   call void @main.FuncStructParams(%main.TinyStruct %2, %main.SmallStruct %6, %main.MidStruct %11, %main.BigStruct %23)
// CHECK-NEXT:   %24 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %25 = getelementptr inbounds %main.TinyStruct, ptr %24, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %25, align 8
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %27 = getelementptr inbounds %main.SmallStruct, ptr %26, i32 0, i32 0
// CHECK-NEXT:   %28 = getelementptr inbounds %main.SmallStruct, ptr %26, i32 0, i32 1
// CHECK-NEXT:   store i64 2, ptr %27, align 8
// CHECK-NEXT:   store i64 3, ptr %28, align 8
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %30 = getelementptr inbounds %main.MidStruct, ptr %29, i32 0, i32 0
// CHECK-NEXT:   %31 = getelementptr inbounds %main.MidStruct, ptr %29, i32 0, i32 1
// CHECK-NEXT:   %32 = getelementptr inbounds %main.MidStruct, ptr %29, i32 0, i32 2
// CHECK-NEXT:   store i64 4, ptr %30, align 8
// CHECK-NEXT:   store i64 5, ptr %31, align 8
// CHECK-NEXT:   store i64 6, ptr %32, align 8
// CHECK-NEXT:   %33 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 80)
// CHECK-NEXT:   %34 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 0
// CHECK-NEXT:   %35 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 1
// CHECK-NEXT:   %36 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 2
// CHECK-NEXT:   %37 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 3
// CHECK-NEXT:   %38 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 4
// CHECK-NEXT:   %39 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 5
// CHECK-NEXT:   %40 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 6
// CHECK-NEXT:   %41 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 7
// CHECK-NEXT:   %42 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 8
// CHECK-NEXT:   %43 = getelementptr inbounds %main.BigStruct, ptr %33, i32 0, i32 9
// CHECK-NEXT:   store i64 7, ptr %34, align 8
// CHECK-NEXT:   store i64 8, ptr %35, align 8
// CHECK-NEXT:   store i64 9, ptr %36, align 8
// CHECK-NEXT:   store i64 10, ptr %37, align 8
// CHECK-NEXT:   store i64 11, ptr %38, align 8
// CHECK-NEXT:   store i64 12, ptr %39, align 8
// CHECK-NEXT:   store i64 13, ptr %40, align 8
// CHECK-NEXT:   store i64 14, ptr %41, align 8
// CHECK-NEXT:   store i64 15, ptr %42, align 8
// CHECK-NEXT:   store i64 16, ptr %43, align 8
// CHECK-NEXT:   call void @main.FuncStructPtrParams(ptr %24, ptr %26, ptr %29, ptr %33)
// CHECK-NEXT:   %44 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:     #dbg_declare(ptr %44, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store i64 100, ptr %44, align 8
// CHECK-NEXT:     #dbg_value(i64 100, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %45 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 288)
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %46 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 0
// CHECK-NEXT:   %47 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 1
// CHECK-NEXT:   %48 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 2
// CHECK-NEXT:   %49 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 3
// CHECK-NEXT:   %50 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 4
// CHECK-NEXT:   %51 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 5
// CHECK-NEXT:   %52 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 6
// CHECK-NEXT:   %53 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 7
// CHECK-NEXT:   %54 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 8
// CHECK-NEXT:   %55 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 9
// CHECK-NEXT:   %56 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 10
// CHECK-NEXT:   %57 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 11
// CHECK-NEXT:   %58 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 12
// CHECK-NEXT:   %59 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 13
// CHECK-NEXT:   %60 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 14
// CHECK-NEXT:   %61 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 15
// CHECK-NEXT:   %62 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %63 = getelementptr inbounds i64, ptr %62, i64 0
// CHECK-NEXT:   store i64 21, ptr %63, align 8
// CHECK-NEXT:   %64 = getelementptr inbounds i64, ptr %62, i64 1
// CHECK-NEXT:   store i64 22, ptr %64, align 8
// CHECK-NEXT:   %65 = getelementptr inbounds i64, ptr %62, i64 2
// CHECK-NEXT:   store i64 23, ptr %65, align 8
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %62, 0
// CHECK-NEXT:   %67 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, i64 3, 1
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, i64 3, 2
// CHECK-NEXT:   %69 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 16
// CHECK-NEXT:   %70 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %70)
// CHECK-NEXT:   %71 = getelementptr inbounds i64, ptr %69, i64 0
// CHECK-NEXT:   %72 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %72)
// CHECK-NEXT:   %73 = getelementptr inbounds i64, ptr %69, i64 1
// CHECK-NEXT:   %74 = icmp eq ptr %69, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %74)
// CHECK-NEXT:   %75 = getelementptr inbounds i64, ptr %69, i64 2
// CHECK-NEXT:   %76 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 17
// CHECK-NEXT:   %77 = icmp eq ptr %76, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %77)
// CHECK-NEXT:   %78 = getelementptr inbounds %main.E, ptr %76, i64 0
// CHECK-NEXT:   %79 = getelementptr inbounds %main.E, ptr %78, i32 0, i32 0
// CHECK-NEXT:   %80 = icmp eq ptr %76, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %80)
// CHECK-NEXT:   %81 = getelementptr inbounds %main.E, ptr %76, i64 1
// CHECK-NEXT:   %82 = getelementptr inbounds %main.E, ptr %81, i32 0, i32 0
// CHECK-NEXT:   %83 = icmp eq ptr %76, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNilDeref"(i1 %83)
// CHECK-NEXT:   %84 = getelementptr inbounds %main.E, ptr %76, i64 2
// CHECK-NEXT:   %85 = getelementptr inbounds %main.E, ptr %84, i32 0, i32 0
// CHECK-NEXT:   %86 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 18
// CHECK-NEXT:   %87 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 19
// CHECK-NEXT:   %88 = getelementptr inbounds %main.E, ptr %87, i32 0, i32 0
// CHECK-NEXT:   %89 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 20
// CHECK-NEXT:   %90 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 288)
// CHECK-NEXT:   %91 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %90, i32 0, i32 1
// CHECK-NEXT:   store i16 100, ptr %91, align 2
// CHECK-NEXT:   %92 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 21
// CHECK-NEXT:     #dbg_declare(ptr %44, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %93 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 22
// CHECK-NEXT:   %94 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 23
// CHECK-NEXT:   %95 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_uint64", i64 2)
// CHECK-NEXT:   %96 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %96, align 8
// CHECK-NEXT:   %97 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_uint64", ptr %95, ptr %96)
// CHECK-NEXT:   store i64 31, ptr %97, align 8
// CHECK-NEXT:   %98 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %98, align 8
// CHECK-NEXT:   %99 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_uint64", ptr %95, ptr %98)
// CHECK-NEXT:   store i64 32, ptr %99, align 8
// CHECK-NEXT:   %100 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 24
// CHECK-NEXT:   %101 = call ptr @"{{.*}}/runtime/internal/runtime.NewChan"(i64 8, i64 0)
// CHECK-NEXT:   %102 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 25
// CHECK-NEXT:   %103 = call %"{{.*}}/runtime/internal/runtime.iface" @errors.New(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 })
// CHECK-NEXT:   %104 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 26
// CHECK-NEXT:   %105 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %106 = getelementptr inbounds { ptr }, ptr %105, i32 0, i32 0
// CHECK-NEXT:   store ptr %44, ptr %106, align 8
// CHECK-NEXT:   %107 = insertvalue { ptr, ptr } { ptr @"main.main$1", ptr undef }, ptr %105, 1
// CHECK-NEXT:   %108 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 27
// CHECK-NEXT:   %109 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 28
// CHECK-NEXT:   store i8 1, ptr %46, align 1
// CHECK-NEXT:   store i16 2, ptr %47, align 2
// CHECK-NEXT:   store i32 3, ptr %48, align 4
// CHECK-NEXT:   store i64 4, ptr %49, align 8
// CHECK-NEXT:   store i64 5, ptr %50, align 8
// CHECK-NEXT:   store i8 6, ptr %51, align 1
// CHECK-NEXT:   store i16 7, ptr %52, align 2
// CHECK-NEXT:   store i32 8, ptr %53, align 4
// CHECK-NEXT:   store i64 9, ptr %54, align 8
// CHECK-NEXT:   store i64 10, ptr %55, align 8
// CHECK-NEXT:   store float 1.100000e+01, ptr %56, align 4
// CHECK-NEXT:   store double 1.200000e+01, ptr %57, align 8
// CHECK-NEXT:   store i1 true, ptr %58, align 1
// CHECK-NEXT:   store { float, float } { float 1.300000e+01, float 1.400000e+01 }, ptr %59, align 4
// CHECK-NEXT:   store { double, double } { double 1.500000e+01, double 1.600000e+01 }, ptr %60, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %68, ptr %61, align 8
// CHECK-NEXT:   store i64 24, ptr %71, align 8
// CHECK-NEXT:   store i64 25, ptr %73, align 8
// CHECK-NEXT:   store i64 26, ptr %75, align 8
// CHECK-NEXT:   store i64 27, ptr %79, align 8
// CHECK-NEXT:   store i64 28, ptr %82, align 8
// CHECK-NEXT:   store i64 29, ptr %85, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %86, align 8
// CHECK-NEXT:   store i64 30, ptr %88, align 8
// CHECK-NEXT:   store ptr %90, ptr %89, align 8
// CHECK-NEXT:   store ptr %44, ptr %92, align 8
// CHECK-NEXT:   %110 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$opv3stH14p-JT6UN0WEYD-Tr6bHK3MHpC4KSk10pjNU", ptr @"*_llgo_main.Struct")
// CHECK-NEXT:   %111 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %110, 0
// CHECK-NEXT:   %112 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %111, ptr @"__llgo.moduleZeroSizedAlloc$", 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %112, ptr %93, align 8
// CHECK-NEXT:   store ptr %95, ptr %94, align 8
// CHECK-NEXT:   store ptr %101, ptr %100, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %103, ptr %102, align 8
// CHECK-NEXT:   store { ptr, ptr } %107, ptr %104, align 8
// CHECK-NEXT:   store i64 100, ptr %108, align 8
// CHECK-NEXT:   store i64 200, ptr %109, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   store ptr %45, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   %113 = load %main.StructWithAllTypeFields, ptr %45, align 8
// CHECK-NEXT:   %114 = alloca %main.StructWithAllTypeFields, align 8
// CHECK-NEXT:   store %main.StructWithAllTypeFields %113, ptr %114, align 8
// CHECK-NEXT:   %115 = load %main.StructWithAllTypeFields, ptr %114, align 8
// CHECK-NEXT:     #dbg_value(ptr %114, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   store %main.StructWithAllTypeFields %113, ptr @main.globalStruct, align 8
// CHECK-NEXT:   %116 = load i64, ptr @main.globalInt, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 10 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %116)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %117 = load %main.StructWithAllTypeFields, ptr %45, align 8
// CHECK-NEXT:   %118 = alloca %main.StructWithAllTypeFields, align 8
// CHECK-NEXT:   store %main.StructWithAllTypeFields %117, ptr %118, align 8
// CHECK-NEXT:   %119 = load %main.StructWithAllTypeFields, ptr %118, align 8
// CHECK-NEXT:     #dbg_value(ptr %118, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   call void @main.FuncWithAllTypeStructParam(%main.StructWithAllTypeFields %117)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 27 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %120 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 0
// CHECK-NEXT:   %121 = load i8, ptr %120, align 1
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %122 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 1
// CHECK-NEXT:   %123 = load i16, ptr %122, align 2
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %124 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 2
// CHECK-NEXT:   %125 = load i32, ptr %124, align 4
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %126 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 3
// CHECK-NEXT:   %127 = load i64, ptr %126, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %128 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 4
// CHECK-NEXT:   %129 = load i64, ptr %128, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %130 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 5
// CHECK-NEXT:   %131 = load i8, ptr %130, align 1
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %132 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 6
// CHECK-NEXT:   %133 = load i16, ptr %132, align 2
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %134 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 7
// CHECK-NEXT:   %135 = load i32, ptr %134, align 4
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %136 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 8
// CHECK-NEXT:   %137 = load i64, ptr %136, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %138 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 9
// CHECK-NEXT:   %139 = load i64, ptr %138, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %140 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 10
// CHECK-NEXT:   %141 = load float, ptr %140, align 4
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %142 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 11
// CHECK-NEXT:   %143 = load double, ptr %142, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %144 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 12
// CHECK-NEXT:   %145 = load i1, ptr %144, align 1
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %146 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 13
// CHECK-NEXT:   %147 = load { float, float }, ptr %146, align 4
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %148 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 14
// CHECK-NEXT:   %149 = load { double, double }, ptr %148, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %150 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 15
// CHECK-NEXT:   %151 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr %150, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %152 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 16
// CHECK-NEXT:   %153 = load [3 x i64], ptr %152, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %154 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 17
// CHECK-NEXT:   %155 = load [3 x %main.E], ptr %154, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %156 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 18
// CHECK-NEXT:   %157 = load %"{{.*}}/runtime/internal/runtime.String", ptr %156, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %158 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 19
// CHECK-NEXT:   %159 = load %main.E, ptr %158, align 8
// CHECK-NEXT:   %160 = load %main.StructWithAllTypeFields, ptr %45, align 8
// CHECK-NEXT:   %161 = alloca %main.StructWithAllTypeFields, align 8
// CHECK-NEXT:   store %main.StructWithAllTypeFields %160, ptr %161, align 8
// CHECK-NEXT:   %162 = load %main.StructWithAllTypeFields, ptr %161, align 8
// CHECK-NEXT:     #dbg_value(ptr %161, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %163 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 20
// CHECK-NEXT:   %164 = load ptr, ptr %163, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %165 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 21
// CHECK-NEXT:   %166 = load ptr, ptr %165, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %167 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 22
// CHECK-NEXT:   %168 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %167, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %169 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 23
// CHECK-NEXT:   %170 = load ptr, ptr %169, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %171 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 24
// CHECK-NEXT:   %172 = load ptr, ptr %171, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %173 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 25
// CHECK-NEXT:   %174 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %173, align 8
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %175 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 26
// CHECK-NEXT:   %176 = load { ptr, ptr }, ptr %175, align 8
// CHECK-NEXT:   %177 = call { i64, %"{{.*}}/runtime/internal/runtime.iface" } @main.FuncWithAllTypeParams(i8 %121, i16 %123, i32 %125, i64 %127, i64 %129, i8 %131, i16 %133, i32 %135, i64 %137, i64 %139, float %141, double %143, i1 %145, { float, float } %147, { double, double } %149, %"{{.*}}/runtime/internal/runtime.Slice" %151, [3 x i64] %153, [3 x %main.E] %155, %"{{.*}}/runtime/internal/runtime.String" %157, %main.E %159, %main.StructWithAllTypeFields %160, ptr %164, ptr %166, %"{{.*}}/runtime/internal/runtime.iface" %168, ptr %170, ptr %172, %"{{.*}}/runtime/internal/runtime.iface" %174, { ptr, ptr } %176)
// CHECK-NEXT:   %178 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %177, 0
// CHECK-NEXT:   store i64 %178, ptr %44, align 8
// CHECK-NEXT:     #dbg_value(i64 %178, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %179 = extractvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } %177, 1
// CHECK-NEXT:   %180 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %179, ptr %180, align 8
// CHECK-NEXT:   %181 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %180, align 8
// CHECK-NEXT:     #dbg_value(ptr %180, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %182 = load i64, ptr %44, align 8
// CHECK-NEXT:     #dbg_value(i64 %182, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %183 = alloca %"{{.*}}/runtime/internal/runtime.iface", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %179, ptr %183, align 8
// CHECK-NEXT:   %184 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %183, align 8
// CHECK-NEXT:     #dbg_value(ptr %183, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %182)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintIface"(%"{{.*}}/runtime/internal/runtime.iface" %179)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @main.ScopeIf(i64 1)
// CHECK-NEXT:   call void @main.ScopeIf(i64 0)
// CHECK-NEXT:   call void @main.ScopeFor()
// CHECK-NEXT:   call void @main.ScopeSwitch(i64 1)
// CHECK-NEXT:   call void @main.ScopeSwitch(i64 2)
// CHECK-NEXT:   call void @main.ScopeSwitch(i64 3)
// CHECK-NEXT:   %185 = load ptr, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %185)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr @main.globalStruct)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %186 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 0
// CHECK-NEXT:   store i8 18, ptr %186, align 1
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %187 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %45, i32 0, i32 0
// CHECK-NEXT:   %188 = load i8, ptr %187, align 1
// CHECK-NEXT:   %189 = sext i8 %188 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %189)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %190 = load ptr, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   %191 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %190, i32 0, i32 0
// CHECK-NEXT:   %192 = load i8, ptr %191, align 1
// CHECK-NEXT:   %193 = sext i8 %192 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %193)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:     #dbg_declare(ptr %45, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %194 = load ptr, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   %195 = getelementptr inbounds %main.StructWithAllTypeFields, ptr %194, i32 0, i32 1
// CHECK-NEXT:   %196 = load i16, ptr %195, align 2
// CHECK-NEXT:   %197 = load ptr, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %45)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr @main.globalStruct)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   %198 = sext i16 %196 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %198)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintPointer"(ptr %197)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   store ptr null, ptr @main.globalStructPtr, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define { i64, %"{{.*}}/runtime/internal/runtime.iface" } @"main.main$1"(ptr {{(nest|swiftself)}} %0, %"{{.*}}/runtime/internal/runtime.String" %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %1, ptr %2, align 8
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.String", ptr %2, align 8
// CHECK-NEXT:     #dbg_value(ptr %2, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   %4 = alloca %"{{.*}}/runtime/internal/runtime.String", align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %1, ptr %4, align 8
// CHECK-NEXT:   %5 = load %"{{.*}}/runtime/internal/runtime.String", ptr %4, align 8
// CHECK-NEXT:     #dbg_value(ptr %4, !{{[0-9]+}}, !DIExpression(DW_OP_deref), !{{[0-9]+}})
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %6 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %7 = extractvalue { ptr } %6, 0
// CHECK-NEXT:   store i64 201, ptr %7, align 8
// CHECK-NEXT:     #dbg_value(i64 201, !{{[0-9]+}}, !DIExpression(), !{{[0-9]+}})
// CHECK-NEXT:   %8 = call %"{{.*}}/runtime/internal/runtime.iface" @errors.New(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %9 = insertvalue { i64, %"{{.*}}/runtime/internal/runtime.iface" } { i64 1, %"{{.*}}/runtime/internal/runtime.iface" undef }, %"{{.*}}/runtime/internal/runtime.iface" %8, 1
// CHECK-NEXT:   ret { i64, %"{{.*}}/runtime/internal/runtime.iface" } %9
// CHECK-NEXT: }

// CHECK: !llvm.dbg.cu = !{!{{[0-9]+}}}
// CHECK: !{{[0-9]+}} = distinct !DICompileUnit({{.*}}producer: "LLGo"{{.*}}emissionKind: FullDebug{{.*}})
// CHECK: !{{[0-9]+}} = !DILocalVariable(name: "s", arg: 1, {{.*}})
