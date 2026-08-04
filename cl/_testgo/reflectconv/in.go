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

// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [7 x i8] c"reflect", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [58 x i8] c"ValueOf(%T(%[1]v)).Convert(%s) = %T(%[3]v), want %T(%[4]v)", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [59 x i8] c"ValueOf(%T(%[1]v)).Convert(%s) has internal kind %v want %v", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [63 x i8] c"Set(ValueOf(%T(%[1]v)).Convert(%s)) = %T(%[3]v), want %T(%[4]v)", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [35 x i8] c"table entry %v is RO, should not be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [46 x i8] c"self-conversion output %v is RO, should not be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [41 x i8] c"conversion output %v is RO, should not be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [46 x i8] c"set(conversion output) %v is RO, should not be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [41 x i8] c"(%s).ConvertibleTo(%s) = false, want true", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [52 x i8] c"ValueOf(%T(%[1]v)).CanConvert(%s) = false, want true", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [53 x i8] c" ValueOf(%T(%[1]v)).CanConvert(%s) = false, want true", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [49 x i8] c"RO self-conversion output %v is not RO, should be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [44 x i8] c"RO conversion output %v is not RO, should be", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [41 x i8] c"@(%s).ConvertibleTo(%s) = %v, want %v: %v", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [47 x i8] c"store/load of sNaN not faithful, got %x want %x", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [7 x i8] c"float32", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [40 x i8] c"signaling nan conversion got %x, want %x", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [40 x i8] c"[]byte should be convertible to *[8]byte", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [57 x i8] c"slice with length 4 should not be convertible to *[8]byte", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [77 x i8] c"reflect: cannot convert slice with length 4 to pointer to array with length 8", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [56 x i8] c"slice with length 4 should not be convertible to [8]byte", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [66 x i8] c"reflect: cannot convert slice with length 4 to array with length 8", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [65 x i8] c"convert slice to non-empty array returns a addressable copy array", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [52 x i8] c"slice (%v) mutation visible in converted result (%v)", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [5 x i8] c"hello", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"bytes1", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"bytes2", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [6 x i8] c"bytes3", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"runes\E2\99\9D", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [8 x i8] c"runes\E2\99\95", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [17 x i8] c"runes\F0\9F\99\88\F0\9F\99\89\F0\9F\99\8A", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [1 x i8] c"a", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"\EF\BF\BD", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [13 x i8] c"did not panic", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [32 x i8] c"panicked with unexpected type %T", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [44 x i8] c"panic string does not start with \22reflect\22: ", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [31 x i8] c"panic string does not contain \22", align 1{{$}}
// CHECK: {{^}}@{{[0-9]+}} = private unnamed_addr constant [3 x i8] c"\22: ", align 1{{$}}

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

// CHECK-LABEL: define %reflect.Value @main.EmptyInterfaceV(%"{{.*}}/runtime/internal/runtime.eface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_any", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   %6 = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, %"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %6)
// CHECK-NEXT:   ret %reflect.Value %7
// CHECK-NEXT: }

// CHECK-LABEL: define i1 @main.IsRO(%reflect.Value %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   store %reflect.Value %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Value, ptr %1, i32 0, i32 2
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = and i64 %3, 32
// CHECK-NEXT:   %5 = icmp ne i64 %4, 0
// CHECK-NEXT:   ret i1 %5
// CHECK-NEXT: }

// CHECK-LABEL: define %reflect.Value @main.MakeRO(%reflect.Value %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   store %reflect.Value %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %main.Value, ptr %1, i32 0, i32 2
// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
// CHECK-NEXT:   %4 = or i64 %3, 32
// CHECK-NEXT:   %5 = getelementptr inbounds %main.Value, ptr %1, i32 0, i32 2
// CHECK-NEXT:   store i64 %4, ptr %5, align 8
// CHECK-NEXT:   %6 = load %reflect.Value, ptr %1, align 8
// CHECK-NEXT:   ret %reflect.Value %6
// CHECK-NEXT: }

// CHECK-LABEL: define %reflect.Value @main.ReadWriterV(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.ReadWriter", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   %6 = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, %"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %6)
// CHECK-NEXT:   ret %reflect.Value %7
// CHECK-NEXT: }

// CHECK-LABEL: define %reflect.Value @main.ReaderV(%"{{.*}}/runtime/internal/runtime.iface" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Reader", ptr undef }, ptr %1, 1
// CHECK-NEXT:   %4 = extractvalue { ptr, ptr } %2, 1
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %2, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %5)
// CHECK-NEXT:   %6 = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} %4, %"{{.*}}/runtime/internal/runtime.eface" %3)
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %6)
// CHECK-NEXT:   ret %reflect.Value %7
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.TestConvert(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map{{\[\[}}2]_llgo_reflect.Type]_llgo_bool", i64 0)
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_reflect.Type]_llgo_bool", i64 0)
// CHECK-NEXT:   %3 = load %"{{.*}}/runtime/internal/runtime.Slice", ptr @main.convertTests, align 8
// CHECK-NEXT:   %4 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_33, %_llgo_6, %_llgo_4, %_llgo_32, %_llgo_0
// CHECK-NEXT:   %5 = phi i64 [ -1, %_llgo_0 ], [ %6, %_llgo_4 ], [ %6, %_llgo_6 ], [ %6, %_llgo_32 ], [ %6, %_llgo_33 ]
// CHECK-NEXT:   %6 = add i64 %5, 1
// CHECK-NEXT:   %7 = icmp slt i64 %6, %4
// CHECK-NEXT:   br i1 %7, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 0
// CHECK-NEXT:   %9 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %3, 1
// CHECK-NEXT:   %10 = icmp slt i64 %6, 0
// CHECK-NEXT:   %11 = icmp uge i64 %6, %9
// CHECK-NEXT:   %12 = or i1 %11, %10
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.CheckIndexRange"(i1 %12, i64 %6, i1 true, i64 %9)
// CHECK-NEXT:   %13 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %8, i64 %6
// CHECK-NEXT:   %14 = load { %reflect.Value, %reflect.Value }, ptr %13, align 8
// CHECK-NEXT:   %15 = alloca { %reflect.Value, %reflect.Value }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %15, i8 0, i64 48, i1 false)
// CHECK-NEXT:   store { %reflect.Value, %reflect.Value } %14, ptr %15, align 8
// CHECK-NEXT:   %16 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %17 = load %reflect.Value, ptr %16, align 8
// CHECK-NEXT:   %18 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %17)
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 0
// CHECK-NEXT:   %21 = getelementptr ptr, ptr %20, i64 10
// CHECK-NEXT:   %22 = load ptr, ptr %21, align 8
// CHECK-NEXT:   %23 = insertvalue { ptr, ptr } undef, ptr %22, 0
// CHECK-NEXT:   %24 = insertvalue { ptr, ptr } %23, ptr %19, 1
// CHECK-NEXT:   %25 = extractvalue { ptr, ptr } %24, 1
// CHECK-NEXT:   %26 = extractvalue { ptr, ptr } %24, 0
// CHECK-NEXT:   %27 = call i1 %26(ptr %25, %"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   br i1 %27, label %_llgo_5, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_1
// CHECK-NEXT:   %28 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr %2)
// CHECK-NEXT:   br label %_llgo_34
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %30 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %29, i64 0
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %32 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %31, 0
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %33, ptr %32, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %34, ptr %30, align 8
// CHECK-NEXT:   %35 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %29, i64 1
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %36, 0
// CHECK-NEXT:   %39 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %38, ptr %37, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %39, ptr %35, align 8
// CHECK-NEXT:   %40 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %29, 0
// CHECK-NEXT:   %41 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %40, i64 2, 1
// CHECK-NEXT:   %42 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %41, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 41 }, %"{{.*}}/runtime/internal/runtime.Slice" %42)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %43 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %44 = load %reflect.Value, ptr %43, align 8
// CHECK-NEXT:   %45 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %44)
// CHECK-NEXT:   %46 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %47 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 0
// CHECK-NEXT:   %48 = getelementptr ptr, ptr %47, i64 10
// CHECK-NEXT:   %49 = load ptr, ptr %48, align 8
// CHECK-NEXT:   %50 = insertvalue { ptr, ptr } undef, ptr %49, 0
// CHECK-NEXT:   %51 = insertvalue { ptr, ptr } %50, ptr %46, 1
// CHECK-NEXT:   %52 = extractvalue { ptr, ptr } %51, 1
// CHECK-NEXT:   %53 = extractvalue { ptr, ptr } %51, 0
// CHECK-NEXT:   %54 = call i1 %53(ptr %52, %"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   br i1 %54, label %_llgo_7, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %55 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %56 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 0
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %58 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %59 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %57, 0
// CHECK-NEXT:   %60 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %59, ptr %58, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %60, ptr %56, align 8
// CHECK-NEXT:   %61 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %55, i64 1
// CHECK-NEXT:   %62 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %63 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %45, 1
// CHECK-NEXT:   %64 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %62, 0
// CHECK-NEXT:   %65 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %64, ptr %63, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %65, ptr %61, align 8
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %55, 0
// CHECK-NEXT:   %67 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %66, i64 2, 1
// CHECK-NEXT:   %68 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %67, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 41 }, %"{{.*}}/runtime/internal/runtime.Slice" %68)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %69 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %18, ptr %69, align 8
// CHECK-NEXT:   %70 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr %2, ptr %69)
// CHECK-NEXT:   store i1 true, ptr %70, align 1
// CHECK-NEXT:   %71 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %45, ptr %71, align 8
// CHECK-NEXT:   %72 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr %2, ptr %71)
// CHECK-NEXT:   store i1 true, ptr %72, align 1
// CHECK-NEXT:   %73 = alloca [2 x %"{{.*}}/runtime/internal/runtime.iface"], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %73, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %74 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %73, i64 0
// CHECK-NEXT:   %75 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %73, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %18, ptr %74, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %45, ptr %75, align 8
// CHECK-NEXT:   %76 = load [2 x %"{{.*}}/runtime/internal/runtime.iface"], ptr %73, align 8
// CHECK-NEXT:   %77 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   store [2 x %"{{.*}}/runtime/internal/runtime.iface"] %76, ptr %77, align 8
// CHECK-NEXT:   %78 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map{{\[\[}}2]_llgo_reflect.Type]_llgo_bool", ptr %1, ptr %77)
// CHECK-NEXT:   store i1 true, ptr %78, align 1
// CHECK-NEXT:   %79 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %80 = load %reflect.Value, ptr %79, align 8
// CHECK-NEXT:   %81 = call i1 @reflect.Value.CanConvert(%reflect.Value %80, %"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   br i1 %81, label %_llgo_9, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %82 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %83 = load %reflect.Value, ptr %82, align 8
// CHECK-NEXT:   %84 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %83)
// CHECK-NEXT:   %85 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %86 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %85, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %84, ptr %86, align 8
// CHECK-NEXT:   %87 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %85, i64 1
// CHECK-NEXT:   %88 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %89 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %90 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %88, 0
// CHECK-NEXT:   %91 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %90, ptr %89, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %91, ptr %87, align 8
// CHECK-NEXT:   %92 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %85, 0
// CHECK-NEXT:   %93 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %92, i64 2, 1
// CHECK-NEXT:   %94 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %93, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 52 }, %"{{.*}}/runtime/internal/runtime.Slice" %94)
// CHECK-NEXT:   br label %_llgo_9
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_8, %_llgo_7
// CHECK-NEXT:   %95 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %80, %"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %96 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %95)
// CHECK-NEXT:   %97 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %95)
// CHECK-NEXT:   %98 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %99 = load %reflect.Value, ptr %98, align 8
// CHECK-NEXT:   %100 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %99)
// CHECK-NEXT:   %101 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %97)
// CHECK-NEXT:   %102 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %97, 1
// CHECK-NEXT:   %103 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %101, 0
// CHECK-NEXT:   %104 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %103, ptr %102, 1
// CHECK-NEXT:   %105 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %100)
// CHECK-NEXT:   %106 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %100, 1
// CHECK-NEXT:   %107 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %105, 0
// CHECK-NEXT:   %108 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %107, ptr %106, 1
// CHECK-NEXT:   %109 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %104, %"{{.*}}/runtime/internal/runtime.eface" %108)
// CHECK-NEXT:   %110 = xor i1 %109, true
// CHECK-NEXT:   br i1 %110, label %_llgo_10, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_12, %_llgo_9
// CHECK-NEXT:   %111 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %112 = load %reflect.Value, ptr %111, align 8
// CHECK-NEXT:   %113 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %112)
// CHECK-NEXT:   %114 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %115 = load %reflect.Value, ptr %114, align 8
// CHECK-NEXT:   %116 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %115)
// CHECK-NEXT:   %117 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %118 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %117, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %113, ptr %118, align 8
// CHECK-NEXT:   %119 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %117, i64 1
// CHECK-NEXT:   %120 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %121 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %122 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %120, 0
// CHECK-NEXT:   %123 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %122, ptr %121, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %123, ptr %119, align 8
// CHECK-NEXT:   %124 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %117, i64 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %96, ptr %124, align 8
// CHECK-NEXT:   %125 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %117, i64 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %116, ptr %125, align 8
// CHECK-NEXT:   %126 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %117, 0
// CHECK-NEXT:   %127 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %126, i64 4, 1
// CHECK-NEXT:   %128 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %127, i64 4, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 58 }, %"{{.*}}/runtime/internal/runtime.Slice" %128)
// CHECK-NEXT:   br label %_llgo_11
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_12, %_llgo_10
// CHECK-NEXT:   %129 = call i1 @reflect.Value.CanConvert(%reflect.Value %80, %"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   br i1 %129, label %_llgo_14, label %_llgo_13
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_9
// CHECK-NEXT:   %130 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %131 = load %reflect.Value, ptr %130, align 8
// CHECK-NEXT:   %132 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %131)
// CHECK-NEXT:   %133 = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" %96, %"{{.*}}/runtime/internal/runtime.eface" %132)
// CHECK-NEXT:   br i1 %133, label %_llgo_11, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %134 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %135 = load %reflect.Value, ptr %134, align 8
// CHECK-NEXT:   %136 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %135)
// CHECK-NEXT:   %137 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %138 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %137, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %136, ptr %138, align 8
// CHECK-NEXT:   %139 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %137, i64 1
// CHECK-NEXT:   %140 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %141 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %45, 1
// CHECK-NEXT:   %142 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %140, 0
// CHECK-NEXT:   %143 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %142, ptr %141, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %143, ptr %139, align 8
// CHECK-NEXT:   %144 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %137, 0
// CHECK-NEXT:   %145 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %144, i64 2, 1
// CHECK-NEXT:   %146 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %145, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 53 }, %"{{.*}}/runtime/internal/runtime.Slice" %146)
// CHECK-NEXT:   br label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_13, %_llgo_11
// CHECK-NEXT:   %147 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %80, %"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %148 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %147)
// CHECK-NEXT:   %149 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %147)
// CHECK-NEXT:   %150 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %151 = load %reflect.Value, ptr %150, align 8
// CHECK-NEXT:   %152 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %151)
// CHECK-NEXT:   %153 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %149)
// CHECK-NEXT:   %154 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %149, 1
// CHECK-NEXT:   %155 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %153, 0
// CHECK-NEXT:   %156 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %155, ptr %154, 1
// CHECK-NEXT:   %157 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %152)
// CHECK-NEXT:   %158 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %152, 1
// CHECK-NEXT:   %159 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %157, 0
// CHECK-NEXT:   %160 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %159, ptr %158, 1
// CHECK-NEXT:   %161 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %156, %"{{.*}}/runtime/internal/runtime.eface" %160)
// CHECK-NEXT:   %162 = xor i1 %161, true
// CHECK-NEXT:   br i1 %162, label %_llgo_15, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_17, %_llgo_14
// CHECK-NEXT:   %163 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %164 = load %reflect.Value, ptr %163, align 8
// CHECK-NEXT:   %165 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %164)
// CHECK-NEXT:   %166 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %167 = load %reflect.Value, ptr %166, align 8
// CHECK-NEXT:   %168 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %167)
// CHECK-NEXT:   %169 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %170 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %169, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %165, ptr %170, align 8
// CHECK-NEXT:   %171 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %169, i64 1
// CHECK-NEXT:   %172 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %173 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %45, 1
// CHECK-NEXT:   %174 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %172, 0
// CHECK-NEXT:   %175 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %174, ptr %173, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %175, ptr %171, align 8
// CHECK-NEXT:   %176 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %169, i64 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %148, ptr %176, align 8
// CHECK-NEXT:   %177 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %169, i64 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %168, ptr %177, align 8
// CHECK-NEXT:   %178 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %169, 0
// CHECK-NEXT:   %179 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %178, i64 4, 1
// CHECK-NEXT:   %180 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %179, i64 4, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 58 }, %"{{.*}}/runtime/internal/runtime.Slice" %180)
// CHECK-NEXT:   br label %_llgo_16
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_17, %_llgo_15
// CHECK-NEXT:   %181 = call i64 @reflect.Value.Kind(%reflect.Value %147)
// CHECK-NEXT:   %182 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %147)
// CHECK-NEXT:   %183 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %182)
// CHECK-NEXT:   %184 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %182, 0
// CHECK-NEXT:   %185 = getelementptr ptr, ptr %184, i64 {{(21|23)}}
// CHECK-NEXT:   %186 = load ptr, ptr %185, align 8
// CHECK-NEXT:   %187 = insertvalue { ptr, ptr } undef, ptr %186, 0
// CHECK-NEXT:   %188 = insertvalue { ptr, ptr } %187, ptr %183, 1
// CHECK-NEXT:   %189 = extractvalue { ptr, ptr } %188, 1
// CHECK-NEXT:   %190 = extractvalue { ptr, ptr } %188, 0
// CHECK-NEXT:   %191 = call i64 %190(ptr %189)
// CHECK-NEXT:   %192 = icmp ne i64 %181, %191
// CHECK-NEXT:   br i1 %192, label %_llgo_18, label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_14
// CHECK-NEXT:   %193 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %194 = load %reflect.Value, ptr %193, align 8
// CHECK-NEXT:   %195 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %194)
// CHECK-NEXT:   %196 = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" %148, %"{{.*}}/runtime/internal/runtime.eface" %195)
// CHECK-NEXT:   br i1 %196, label %_llgo_16, label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_16
// CHECK-NEXT:   %197 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %198 = load %reflect.Value, ptr %197, align 8
// CHECK-NEXT:   %199 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %198)
// CHECK-NEXT:   %200 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %201 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %200, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %199, ptr %201, align 8
// CHECK-NEXT:   %202 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %200, i64 1
// CHECK-NEXT:   %203 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %204 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %18, 1
// CHECK-NEXT:   %205 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %203, 0
// CHECK-NEXT:   %206 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %205, ptr %204, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %206, ptr %202, align 8
// CHECK-NEXT:   %207 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %200, i64 2
// CHECK-NEXT:   %208 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %181, ptr %208, align 8
// CHECK-NEXT:   %209 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Kind, ptr undef }, ptr %208, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %209, ptr %207, align 8
// CHECK-NEXT:   %210 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %200, i64 3
// CHECK-NEXT:   %211 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %191, ptr %211, align 8
// CHECK-NEXT:   %212 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Kind, ptr undef }, ptr %211, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %212, ptr %210, align 8
// CHECK-NEXT:   %213 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %200, 0
// CHECK-NEXT:   %214 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %213, i64 4, 1
// CHECK-NEXT:   %215 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %214, i64 4, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 59 }, %"{{.*}}/runtime/internal/runtime.Slice" %215)
// CHECK-NEXT:   br label %_llgo_19
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_19:                                         ; preds = %_llgo_18, %_llgo_16
// CHECK-NEXT:   %216 = call %reflect.Value @reflect.New(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %217 = call %reflect.Value @reflect.Value.Elem(%reflect.Value %216)
// CHECK-NEXT:   call void @reflect.Value.Set(%reflect.Value %217, %reflect.Value %147)
// CHECK-NEXT:   %218 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %217)
// CHECK-NEXT:   %219 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %217)
// CHECK-NEXT:   %220 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %221 = load %reflect.Value, ptr %220, align 8
// CHECK-NEXT:   %222 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %221)
// CHECK-NEXT:   %223 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %219)
// CHECK-NEXT:   %224 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %219, 1
// CHECK-NEXT:   %225 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %223, 0
// CHECK-NEXT:   %226 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %225, ptr %224, 1
// CHECK-NEXT:   %227 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %222)
// CHECK-NEXT:   %228 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %222, 1
// CHECK-NEXT:   %229 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %227, 0
// CHECK-NEXT:   %230 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %229, ptr %228, 1
// CHECK-NEXT:   %231 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %226, %"{{.*}}/runtime/internal/runtime.eface" %230)
// CHECK-NEXT:   %232 = xor i1 %231, true
// CHECK-NEXT:   br i1 %232, label %_llgo_20, label %_llgo_22
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_20:                                         ; preds = %_llgo_22, %_llgo_19
// CHECK-NEXT:   %233 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 0
// CHECK-NEXT:   %234 = load %reflect.Value, ptr %233, align 8
// CHECK-NEXT:   %235 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %234)
// CHECK-NEXT:   %236 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %237 = load %reflect.Value, ptr %236, align 8
// CHECK-NEXT:   %238 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %237)
// CHECK-NEXT:   %239 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 64)
// CHECK-NEXT:   %240 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %239, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %235, ptr %240, align 8
// CHECK-NEXT:   %241 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %239, i64 1
// CHECK-NEXT:   %242 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %243 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %45, 1
// CHECK-NEXT:   %244 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %242, 0
// CHECK-NEXT:   %245 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %244, ptr %243, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %245, ptr %241, align 8
// CHECK-NEXT:   %246 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %239, i64 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %218, ptr %246, align 8
// CHECK-NEXT:   %247 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %239, i64 3
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %238, ptr %247, align 8
// CHECK-NEXT:   %248 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %239, 0
// CHECK-NEXT:   %249 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %248, i64 4, 1
// CHECK-NEXT:   %250 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %249, i64 4, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 63 }, %"{{.*}}/runtime/internal/runtime.Slice" %250)
// CHECK-NEXT:   br label %_llgo_21
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_21:                                         ; preds = %_llgo_22, %_llgo_20
// CHECK-NEXT:   %251 = call i1 @main.IsRO(%reflect.Value %80)
// CHECK-NEXT:   br i1 %251, label %_llgo_23, label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_22:                                         ; preds = %_llgo_19
// CHECK-NEXT:   %252 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %15, i32 0, i32 1
// CHECK-NEXT:   %253 = load %reflect.Value, ptr %252, align 8
// CHECK-NEXT:   %254 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %253)
// CHECK-NEXT:   %255 = call i1 @reflect.DeepEqual(%"{{.*}}/runtime/internal/runtime.eface" %218, %"{{.*}}/runtime/internal/runtime.eface" %254)
// CHECK-NEXT:   br i1 %255, label %_llgo_21, label %_llgo_20
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_23:                                         ; preds = %_llgo_21
// CHECK-NEXT:   %256 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %257 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %256, i64 0
// CHECK-NEXT:   %258 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %80, ptr %258, align 8
// CHECK-NEXT:   %259 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %258, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %259, ptr %257, align 8
// CHECK-NEXT:   %260 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %256, 0
// CHECK-NEXT:   %261 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %260, i64 1, 1
// CHECK-NEXT:   %262 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %261, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 35 }, %"{{.*}}/runtime/internal/runtime.Slice" %262)
// CHECK-NEXT:   br label %_llgo_24
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_24:                                         ; preds = %_llgo_23, %_llgo_21
// CHECK-NEXT:   %263 = call i1 @main.IsRO(%reflect.Value %95)
// CHECK-NEXT:   br i1 %263, label %_llgo_25, label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_25:                                         ; preds = %_llgo_24
// CHECK-NEXT:   %264 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %265 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %264, i64 0
// CHECK-NEXT:   %266 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %95, ptr %266, align 8
// CHECK-NEXT:   %267 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %266, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %267, ptr %265, align 8
// CHECK-NEXT:   %268 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %264, 0
// CHECK-NEXT:   %269 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %268, i64 1, 1
// CHECK-NEXT:   %270 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %269, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 46 }, %"{{.*}}/runtime/internal/runtime.Slice" %270)
// CHECK-NEXT:   br label %_llgo_26
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_26:                                         ; preds = %_llgo_25, %_llgo_24
// CHECK-NEXT:   %271 = call i1 @main.IsRO(%reflect.Value %147)
// CHECK-NEXT:   br i1 %271, label %_llgo_27, label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_27:                                         ; preds = %_llgo_26
// CHECK-NEXT:   %272 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %273 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %272, i64 0
// CHECK-NEXT:   %274 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %147, ptr %274, align 8
// CHECK-NEXT:   %275 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %274, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %275, ptr %273, align 8
// CHECK-NEXT:   %276 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %272, 0
// CHECK-NEXT:   %277 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %276, i64 1, 1
// CHECK-NEXT:   %278 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %277, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 41 }, %"{{.*}}/runtime/internal/runtime.Slice" %278)
// CHECK-NEXT:   br label %_llgo_28
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_28:                                         ; preds = %_llgo_27, %_llgo_26
// CHECK-NEXT:   %279 = call i1 @main.IsRO(%reflect.Value %217)
// CHECK-NEXT:   br i1 %279, label %_llgo_29, label %_llgo_30
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_29:                                         ; preds = %_llgo_28
// CHECK-NEXT:   %280 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %281 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %280, i64 0
// CHECK-NEXT:   %282 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %217, ptr %282, align 8
// CHECK-NEXT:   %283 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %282, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %283, ptr %281, align 8
// CHECK-NEXT:   %284 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %280, 0
// CHECK-NEXT:   %285 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %284, i64 1, 1
// CHECK-NEXT:   %286 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %285, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 46 }, %"{{.*}}/runtime/internal/runtime.Slice" %286)
// CHECK-NEXT:   br label %_llgo_30
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_30:                                         ; preds = %_llgo_29, %_llgo_28
// CHECK-NEXT:   %287 = call %reflect.Value @main.MakeRO(%reflect.Value %80)
// CHECK-NEXT:   %288 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %287, %"{{.*}}/runtime/internal/runtime.iface" %18)
// CHECK-NEXT:   %289 = call i1 @main.IsRO(%reflect.Value %288)
// CHECK-NEXT:   br i1 %289, label %_llgo_32, label %_llgo_31
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_31:                                         ; preds = %_llgo_30
// CHECK-NEXT:   %290 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %291 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %290, i64 0
// CHECK-NEXT:   %292 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %80, ptr %292, align 8
// CHECK-NEXT:   %293 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %292, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %293, ptr %291, align 8
// CHECK-NEXT:   %294 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %290, 0
// CHECK-NEXT:   %295 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %294, i64 1, 1
// CHECK-NEXT:   %296 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %295, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 49 }, %"{{.*}}/runtime/internal/runtime.Slice" %296)
// CHECK-NEXT:   br label %_llgo_32
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_32:                                         ; preds = %_llgo_31, %_llgo_30
// CHECK-NEXT:   %297 = call %reflect.Value @main.MakeRO(%reflect.Value %80)
// CHECK-NEXT:   %298 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %297, %"{{.*}}/runtime/internal/runtime.iface" %45)
// CHECK-NEXT:   %299 = call i1 @main.IsRO(%reflect.Value %298)
// CHECK-NEXT:   br i1 %299, label %_llgo_1, label %_llgo_33
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_33:                                         ; preds = %_llgo_32
// CHECK-NEXT:   %300 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %301 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %300, i64 0
// CHECK-NEXT:   %302 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %80, ptr %302, align 8
// CHECK-NEXT:   %303 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %302, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %303, ptr %301, align 8
// CHECK-NEXT:   %304 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %300, 0
// CHECK-NEXT:   %305 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %304, i64 1, 1
// CHECK-NEXT:   %306 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %305, i64 1, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 44 }, %"{{.*}}/runtime/internal/runtime.Slice" %306)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_34:                                         ; preds = %_llgo_50, %_llgo_3
// CHECK-NEXT:   %307 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %28)
// CHECK-NEXT:   %308 = extractvalue { i1, ptr, ptr } %307, 0
// CHECK-NEXT:   br i1 %308, label %_llgo_45, label %_llgo_46
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_35:                                         ; preds = %_llgo_47
// CHECK-NEXT:   %309 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %398, 1
// CHECK-NEXT:   %310 = call ptr @"{{.*}}/runtime/internal/runtime.NewMapIter"(ptr @"map[_llgo_reflect.Type]_llgo_bool", ptr %2)
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_36:                                         ; preds = %_llgo_47
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_37:                                         ; preds = %_llgo_44, %_llgo_40, %_llgo_35
// CHECK-NEXT:   %311 = call { i1, ptr, ptr } @"{{.*}}/runtime/internal/runtime.MapIterNext"(ptr %310)
// CHECK-NEXT:   %312 = extractvalue { i1, ptr, ptr } %311, 0
// CHECK-NEXT:   br i1 %312, label %_llgo_48, label %_llgo_49
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_38:                                         ; preds = %_llgo_50
// CHECK-NEXT:   %313 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %406, 1
// CHECK-NEXT:   %314 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %309)
// CHECK-NEXT:   %315 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %309, 1
// CHECK-NEXT:   %316 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %314, 0
// CHECK-NEXT:   %317 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %316, ptr %315, 1
// CHECK-NEXT:   %318 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %313)
// CHECK-NEXT:   %319 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %313, 1
// CHECK-NEXT:   %320 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %318, 0
// CHECK-NEXT:   %321 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %320, ptr %319, 1
// CHECK-NEXT:   %322 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %317, %"{{.*}}/runtime/internal/runtime.eface" %321)
// CHECK-NEXT:   br i1 %322, label %_llgo_40, label %_llgo_41
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_39:                                         ; preds = %_llgo_41
// CHECK-NEXT:   %323 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %313)
// CHECK-NEXT:   %324 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %313, 0
// CHECK-NEXT:   %325 = getelementptr ptr, ptr %324, i64 {{(21|23)}}
// CHECK-NEXT:   %326 = load ptr, ptr %325, align 8
// CHECK-NEXT:   %327 = insertvalue { ptr, ptr } undef, ptr %326, 0
// CHECK-NEXT:   %328 = insertvalue { ptr, ptr } %327, ptr %323, 1
// CHECK-NEXT:   %329 = extractvalue { ptr, ptr } %328, 1
// CHECK-NEXT:   %330 = extractvalue { ptr, ptr } %328, 0
// CHECK-NEXT:   %331 = call i64 %330(ptr %329)
// CHECK-NEXT:   %332 = icmp eq i64 %331, 20
// CHECK-NEXT:   br i1 %332, label %_llgo_42, label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_40:                                         ; preds = %_llgo_43, %_llgo_41, %_llgo_38
// CHECK-NEXT:   %333 = phi i1 [ true, %_llgo_38 ], [ true, %_llgo_41 ], [ %361, %_llgo_43 ]
// CHECK-NEXT:   %334 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %309)
// CHECK-NEXT:   %335 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %309, 0
// CHECK-NEXT:   %336 = getelementptr ptr, ptr %335, i64 10
// CHECK-NEXT:   %337 = load ptr, ptr %336, align 8
// CHECK-NEXT:   %338 = insertvalue { ptr, ptr } undef, ptr %337, 0
// CHECK-NEXT:   %339 = insertvalue { ptr, ptr } %338, ptr %334, 1
// CHECK-NEXT:   %340 = extractvalue { ptr, ptr } %339, 1
// CHECK-NEXT:   %341 = extractvalue { ptr, ptr } %339, 0
// CHECK-NEXT:   %342 = call i1 %341(ptr %340, %"{{.*}}/runtime/internal/runtime.iface" %313)
// CHECK-NEXT:   %343 = icmp ne i1 %342, %333
// CHECK-NEXT:   br i1 %343, label %_llgo_44, label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_41:                                         ; preds = %_llgo_38
// CHECK-NEXT:   %344 = alloca [2 x %"{{.*}}/runtime/internal/runtime.iface"], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %344, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %345 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %344, i64 0
// CHECK-NEXT:   %346 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %344, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %309, ptr %345, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %313, ptr %346, align 8
// CHECK-NEXT:   %347 = load [2 x %"{{.*}}/runtime/internal/runtime.iface"], ptr %344, align 8
// CHECK-NEXT:   %348 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   store [2 x %"{{.*}}/runtime/internal/runtime.iface"] %347, ptr %348, align 8
// CHECK-NEXT:   %349 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map{{\[\[}}2]_llgo_reflect.Type]_llgo_bool", ptr %1, ptr %348)
// CHECK-NEXT:   %350 = load i1, ptr %349, align 1
// CHECK-NEXT:   br i1 %350, label %_llgo_40, label %_llgo_39
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_42:                                         ; preds = %_llgo_39
// CHECK-NEXT:   %351 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %313)
// CHECK-NEXT:   %352 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %313, 0
// CHECK-NEXT:   %353 = getelementptr ptr, ptr %352, i64 {{(28|31)}}
// CHECK-NEXT:   %354 = load ptr, ptr %353, align 8
// CHECK-NEXT:   %355 = insertvalue { ptr, ptr } undef, ptr %354, 0
// CHECK-NEXT:   %356 = insertvalue { ptr, ptr } %355, ptr %351, 1
// CHECK-NEXT:   %357 = extractvalue { ptr, ptr } %356, 1
// CHECK-NEXT:   %358 = extractvalue { ptr, ptr } %356, 0
// CHECK-NEXT:   %359 = call i64 %358(ptr %357)
// CHECK-NEXT:   %360 = icmp eq i64 %359, 0
// CHECK-NEXT:   br label %_llgo_43
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_43:                                         ; preds = %_llgo_42, %_llgo_39
// CHECK-NEXT:   %361 = phi i1 [ false, %_llgo_39 ], [ %360, %_llgo_42 ]
// CHECK-NEXT:   br label %_llgo_40
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_44:                                         ; preds = %_llgo_40
// CHECK-NEXT:   %362 = alloca [2 x %"{{.*}}/runtime/internal/runtime.iface"], align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %362, i8 0, i64 32, i1 false)
// CHECK-NEXT:   %363 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %362, i64 0
// CHECK-NEXT:   %364 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.iface", ptr %362, i64 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %309, ptr %363, align 8
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %313, ptr %364, align 8
// CHECK-NEXT:   %365 = load [2 x %"{{.*}}/runtime/internal/runtime.iface"], ptr %362, align 8
// CHECK-NEXT:   %366 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   store [2 x %"{{.*}}/runtime/internal/runtime.iface"] %365, ptr %366, align 8
// CHECK-NEXT:   %367 = call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"(ptr @"map{{\[\[}}2]_llgo_reflect.Type]_llgo_bool", ptr %1, ptr %366)
// CHECK-NEXT:   %368 = load i1, ptr %367, align 1
// CHECK-NEXT:   %369 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 80)
// CHECK-NEXT:   %370 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %369, i64 0
// CHECK-NEXT:   %371 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %309)
// CHECK-NEXT:   %372 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %309, 1
// CHECK-NEXT:   %373 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %371, 0
// CHECK-NEXT:   %374 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %373, ptr %372, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %374, ptr %370, align 8
// CHECK-NEXT:   %375 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %369, i64 1
// CHECK-NEXT:   %376 = call ptr @"{{.*}}/runtime/internal/runtime.IfaceType"(%"{{.*}}/runtime/internal/runtime.iface" %313)
// CHECK-NEXT:   %377 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %313, 1
// CHECK-NEXT:   %378 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" undef, ptr %376, 0
// CHECK-NEXT:   %379 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" %378, ptr %377, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %379, ptr %375, align 8
// CHECK-NEXT:   %380 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %369, i64 2
// CHECK-NEXT:   %381 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 %342, ptr %381, align 1
// CHECK-NEXT:   %382 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %381, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %382, ptr %380, align 8
// CHECK-NEXT:   %383 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %369, i64 3
// CHECK-NEXT:   %384 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 %333, ptr %384, align 1
// CHECK-NEXT:   %385 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %384, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %385, ptr %383, align 8
// CHECK-NEXT:   %386 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %369, i64 4
// CHECK-NEXT:   %387 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i1 %368, ptr %387, align 1
// CHECK-NEXT:   %388 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_bool, ptr undef }, ptr %387, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %388, ptr %386, align 8
// CHECK-NEXT:   %389 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %369, 0
// CHECK-NEXT:   %390 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %389, i64 5, 1
// CHECK-NEXT:   %391 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %390, i64 5, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 41 }, %"{{.*}}/runtime/internal/runtime.Slice" %391)
// CHECK-NEXT:   br label %_llgo_37
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_45:                                         ; preds = %_llgo_34
// CHECK-NEXT:   %392 = extractvalue { i1, ptr, ptr } %307, 1
// CHECK-NEXT:   %393 = extractvalue { i1, ptr, ptr } %307, 2
// CHECK-NEXT:   %394 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %392, align 8
// CHECK-NEXT:   %395 = load i1, ptr %393, align 1
// CHECK-NEXT:   %396 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } { i1 true, %"{{.*}}/runtime/internal/runtime.iface" undef, i1 undef }, %"{{.*}}/runtime/internal/runtime.iface" %394, 1
// CHECK-NEXT:   %397 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %396, i1 %395, 2
// CHECK-NEXT:   br label %_llgo_47
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_46:                                         ; preds = %_llgo_34
// CHECK-NEXT:   br label %_llgo_47
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_47:                                         ; preds = %_llgo_46, %_llgo_45
// CHECK-NEXT:   %398 = phi { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %397, %_llgo_45 ], [ zeroinitializer, %_llgo_46 ]
// CHECK-NEXT:   %399 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %398, 0
// CHECK-NEXT:   br i1 %399, label %_llgo_35, label %_llgo_36
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_48:                                         ; preds = %_llgo_37
// CHECK-NEXT:   %400 = extractvalue { i1, ptr, ptr } %311, 1
// CHECK-NEXT:   %401 = extractvalue { i1, ptr, ptr } %311, 2
// CHECK-NEXT:   %402 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %400, align 8
// CHECK-NEXT:   %403 = load i1, ptr %401, align 1
// CHECK-NEXT:   %404 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } { i1 true, %"{{.*}}/runtime/internal/runtime.iface" undef, i1 undef }, %"{{.*}}/runtime/internal/runtime.iface" %402, 1
// CHECK-NEXT:   %405 = insertvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %404, i1 %403, 2
// CHECK-NEXT:   br label %_llgo_50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_49:                                         ; preds = %_llgo_37
// CHECK-NEXT:   br label %_llgo_50
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_50:                                         ; preds = %_llgo_49, %_llgo_48
// CHECK-NEXT:   %406 = phi { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } [ %405, %_llgo_48 ], [ zeroinitializer, %_llgo_49 ]
// CHECK-NEXT:   %407 = extractvalue { i1, %"{{.*}}/runtime/internal/runtime.iface", i1 } %406, 0
// CHECK-NEXT:   br i1 %407, label %_llgo_38, label %_llgo_34
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.TestConvertNaNs(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call float @math.Float32frombits(i32 2139095041)
// CHECK-NEXT:   store float %1, ptr @main.gFloat32, align 4
// CHECK-NEXT:   %2 = load float, ptr @main.gFloat32, align 4
// CHECK-NEXT:   %3 = call i32 @math.Float32bits(float %2)
// CHECK-NEXT:   %4 = icmp ne i32 %3, 2139095041
// CHECK-NEXT:   br i1 %4, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %5, i64 0
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 %3, ptr %7, align 4
// CHECK-NEXT:   %8 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %7, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %8, ptr %6, align 8
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %5, i64 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 2139095041, ptr %10, align 4
// CHECK-NEXT:   %11 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %10, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %11, ptr %9, align 8
// CHECK-NEXT:   %12 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %5, 0
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %12, i64 2, 1
// CHECK-NEXT:   %14 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %13, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 47 }, %"{{.*}}/runtime/internal/runtime.Slice" %14)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %15 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %16 = call float @math.Float32frombits(i32 2139095041)
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float %16, ptr %17, align 4
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.myFloat32.14.0, ptr undef }, ptr %17, 1
// CHECK-NEXT:   %19 = extractvalue { ptr, ptr } %15, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %15, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %20)
// CHECK-NEXT:   %21 = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} %19, %"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   %22 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 0.000000e+00, ptr %22, align 4
// CHECK-NEXT:   %23 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %22, 1
// CHECK-NEXT:   %24 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %23)
// CHECK-NEXT:   %25 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %21, %"{{.*}}/runtime/internal/runtime.iface" %24)
// CHECK-NEXT:   %26 = call %"{{.*}}/runtime/internal/runtime.eface" @reflect.Value.Interface(%reflect.Value %25)
// CHECK-NEXT:   %27 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %26, 0
// CHECK-NEXT:   %28 = icmp eq ptr %27, @_llgo_float32
// CHECK-NEXT:   br i1 %28, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %30 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %29, i64 0
// CHECK-NEXT:   %31 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 %41, ptr %31, align 4
// CHECK-NEXT:   %32 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %31, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %32, ptr %30, align 8
// CHECK-NEXT:   %33 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %29, i64 1
// CHECK-NEXT:   %34 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 2139095041, ptr %34, align 4
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %34, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %35, ptr %33, align 8
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %29, 0
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %36, i64 2, 1
// CHECK-NEXT:   %38 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %37, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 40 }, %"{{.*}}/runtime/internal/runtime.Slice" %38)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3, %_llgo_5
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %39 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %26, 1
// CHECK-NEXT:   %40 = load float, ptr %39, align 4
// CHECK-NEXT:   %41 = call i32 @math.Float32bits(float %40)
// CHECK-NEXT:   %42 = icmp ne i32 %41, 2139095041
// CHECK-NEXT:   br i1 %42, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicTypeAssert"(ptr %27, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 7 }, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.TestConvertPanic(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %1, i64 1, i64 4, i64 0, i64 4, i1 true, i1 true, i1 true)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2, ptr %5, align 8
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %5, 1
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   store %reflect.Value %7, ptr %4, align 8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[8]_llgo_uint8", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %10 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %9)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.iface" %10, ptr %8, align 8
// CHECK-NEXT:   %11 = load %reflect.Value, ptr %4, align 8
// CHECK-NEXT:   %12 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.Value.Type(%reflect.Value %11)
// CHECK-NEXT:   %13 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %8, align 8
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %12)
// CHECK-NEXT:   %15 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %12, 0
// CHECK-NEXT:   %16 = getelementptr ptr, ptr %15, i64 10
// CHECK-NEXT:   %17 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %18 = insertvalue { ptr, ptr } undef, ptr %17, 0
// CHECK-NEXT:   %19 = insertvalue { ptr, ptr } %18, ptr %14, 1
// CHECK-NEXT:   %20 = extractvalue { ptr, ptr } %19, 1
// CHECK-NEXT:   %21 = extractvalue { ptr, ptr } %19, 0
// CHECK-NEXT:   %22 = call i1 %21(ptr %20, %"{{.*}}/runtime/internal/runtime.iface" %13)
// CHECK-NEXT:   br i1 %22, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 40 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %23 = load %reflect.Value, ptr %4, align 8
// CHECK-NEXT:   %24 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %8, align 8
// CHECK-NEXT:   %25 = call i1 @reflect.Value.CanConvert(%reflect.Value %23, %"{{.*}}/runtime/internal/runtime.iface" %24)
// CHECK-NEXT:   br i1 %25, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 57 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   br label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3, %_llgo_2
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %27 = getelementptr inbounds { ptr, ptr }, ptr %26, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %27, align 8
// CHECK-NEXT:   %28 = getelementptr inbounds { ptr, ptr }, ptr %26, i32 0, i32 1
// CHECK-NEXT:   store ptr %8, ptr %28, align 8
// CHECK-NEXT:   %29 = insertvalue { ptr, ptr } { ptr @"main.TestConvertPanic$1", ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @main.shouldPanic(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 77 }, { ptr, ptr } %29)
// CHECK-NEXT:   %30 = load %reflect.Value, ptr %4, align 8
// CHECK-NEXT:   %31 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %8, align 8
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %31)
// CHECK-NEXT:   %33 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %31, 0
// CHECK-NEXT:   %34 = getelementptr ptr, ptr %33, i64 11
// CHECK-NEXT:   %35 = load ptr, ptr %34, align 8
// CHECK-NEXT:   %36 = insertvalue { ptr, ptr } undef, ptr %35, 0
// CHECK-NEXT:   %37 = insertvalue { ptr, ptr } %36, ptr %32, 1
// CHECK-NEXT:   %38 = extractvalue { ptr, ptr } %37, 1
// CHECK-NEXT:   %39 = extractvalue { ptr, ptr } %37, 0
// CHECK-NEXT:   %40 = call %"{{.*}}/runtime/internal/runtime.iface" %39(ptr %38)
// CHECK-NEXT:   %41 = call i1 @reflect.Value.CanConvert(%reflect.Value %30, %"{{.*}}/runtime/internal/runtime.iface" %40)
// CHECK-NEXT:   br i1 %41, label %_llgo_5, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_4
// CHECK-NEXT:   call void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 56 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   %43 = getelementptr inbounds { ptr, ptr }, ptr %42, i32 0, i32 0
// CHECK-NEXT:   store ptr %4, ptr %43, align 8
// CHECK-NEXT:   %44 = getelementptr inbounds { ptr, ptr }, ptr %42, i32 0, i32 1
// CHECK-NEXT:   store ptr %8, ptr %44, align 8
// CHECK-NEXT:   %45 = insertvalue { ptr, ptr } { ptr @"main.TestConvertPanic$2", ptr undef }, ptr %42, 1
// CHECK-NEXT:   call void @main.shouldPanic(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 66 }, { ptr, ptr } %45)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.TestConvertPanic$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr, ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %2)
// CHECK-NEXT:   %4 = load %reflect.Value, ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %5, align 8
// CHECK-NEXT:   %7 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %4, %"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.TestConvertPanic$2"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr, ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AssertNilDerefPtr"(ptr %2)
// CHECK-NEXT:   %4 = load %reflect.Value, ptr %3, align 8
// CHECK-NEXT:   %5 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %6 = load %"{{.*}}/runtime/internal/runtime.iface", ptr %5, align 8
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %6)
// CHECK-NEXT:   %8 = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %6, 0
// CHECK-NEXT:   %9 = getelementptr ptr, ptr %8, i64 11
// CHECK-NEXT:   %10 = load ptr, ptr %9, align 8
// CHECK-NEXT:   %11 = insertvalue { ptr, ptr } undef, ptr %10, 0
// CHECK-NEXT:   %12 = insertvalue { ptr, ptr } %11, ptr %7, 1
// CHECK-NEXT:   %13 = extractvalue { ptr, ptr } %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %12, 0
// CHECK-NEXT:   %15 = call %"{{.*}}/runtime/internal/runtime.iface" %14(ptr %13)
// CHECK-NEXT:   %16 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %4, %"{{.*}}/runtime/internal/runtime.iface" %15)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.TestConvertSlice2Array(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.NewSlice2"(ptr %1, i64 8, i64 4, i64 0, i64 4, i1 true, i1 true, i1 true)
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   store [4 x i64] zeroinitializer, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[4]_llgo_int", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %5 = call %"{{.*}}/runtime/internal/runtime.iface" @reflect.TypeOf(%"{{.*}}/runtime/internal/runtime.eface" %4)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int", ptr undef }, ptr %6, 1
// CHECK-NEXT:   %8 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %7)
// CHECK-NEXT:   %9 = call %reflect.Value @reflect.Value.Convert(%reflect.Value %8, %"{{.*}}/runtime/internal/runtime.iface" %5)
// CHECK-NEXT:   %10 = call i1 @reflect.Value.CanAddr(%reflect.Value %9)
// CHECK-NEXT:   br i1 %10, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   call void @"main.(*testingT).Fatalf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 65 }, %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer)
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   %11 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %2, 1
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_4, %_llgo_2
// CHECK-NEXT:   %12 = phi i64 [ -1, %_llgo_2 ], [ %13, %_llgo_4 ]
// CHECK-NEXT:   %13 = add i64 %12, 1
// CHECK-NEXT:   %14 = icmp slt i64 %13, %11
// CHECK-NEXT:   br i1 %14, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %15 = call %reflect.Value @reflect.Value.Index(%reflect.Value %8, i64 %13)
// CHECK-NEXT:   %16 = add i64 %13, 1
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 %16, ptr %17, align 8
// CHECK-NEXT:   %18 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %17, 1
// CHECK-NEXT:   %19 = call %reflect.Value @reflect.ValueOf(%"{{.*}}/runtime/internal/runtime.eface" %18)
// CHECK-NEXT:   call void @reflect.Value.Set(%reflect.Value %15, %reflect.Value %19)
// CHECK-NEXT:   br label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_3
// CHECK-NEXT:   %20 = extractvalue %"{{.*}}/runtime/internal/runtime.Slice" %2, 1
// CHECK-NEXT:   br label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_9, %_llgo_7, %_llgo_5
// CHECK-NEXT:   %21 = phi i64 [ -1, %_llgo_5 ], [ %22, %_llgo_7 ], [ %22, %_llgo_9 ]
// CHECK-NEXT:   %22 = add i64 %21, 1
// CHECK-NEXT:   %23 = icmp slt i64 %22, %20
// CHECK-NEXT:   br i1 %23, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_6
// CHECK-NEXT:   %24 = call %reflect.Value @reflect.Value.Index(%reflect.Value %9, i64 %22)
// CHECK-NEXT:   %25 = call i64 @reflect.Value.Int(%reflect.Value %24)
// CHECK-NEXT:   %26 = icmp ne i64 %25, 0
// CHECK-NEXT:   br i1 %26, label %_llgo_9, label %_llgo_6
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_6
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_7
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
// CHECK-NEXT:   %28 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 0
// CHECK-NEXT:   %29 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %8, ptr %29, align 8
// CHECK-NEXT:   %30 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %29, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %30, ptr %28, align 8
// CHECK-NEXT:   %31 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %27, i64 1
// CHECK-NEXT:   %32 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %reflect.Value %9, ptr %32, align 8
// CHECK-NEXT:   %33 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_reflect.Value, ptr undef }, ptr %32, 1
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %33, ptr %31, align 8
// CHECK-NEXT:   %34 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %27, 0
// CHECK-NEXT:   %35 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %34, i64 2, 1
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %35, i64 2, 2
// CHECK-NEXT:   call void @"main.(*testingT).Fatalf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 52 }, %"{{.*}}/runtime/internal/runtime.Slice" %36)
// CHECK-NEXT:   br label %_llgo_6
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"main.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"main.init$guard", align 1
// CHECK-NEXT:   call void @bytes.init()
// CHECK-NEXT:   call void @fmt.init()
// CHECK-NEXT:   call void @io.init()
// CHECK-NEXT:   call void @math.init()
// CHECK-NEXT:   call void @reflect.init()
// CHECK-NEXT:   call void @strings.init()
// CHECK-NEXT:   store { ptr, ptr } { ptr @reflect.ValueOf, ptr null }, ptr @main.V, align 8
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 17808)
// CHECK-NEXT:   %2 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 0
// CHECK-NEXT:   %3 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %5, align 1
// CHECK-NEXT:   %6 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %5, 1
// CHECK-NEXT:   %7 = extractvalue { ptr, ptr } %4, 1
// CHECK-NEXT:   %8 = extractvalue { ptr, ptr } %4, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %8)
// CHECK-NEXT:   %9 = call %reflect.Value %__llgo_funcval_code(ptr {{(nest|swiftself)}} %7, %"{{.*}}/runtime/internal/runtime.eface" %6)
// CHECK-NEXT:   %10 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2, i32 0, i32 1
// CHECK-NEXT:   %11 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 0, ptr %12, align 1
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %12, 1
// CHECK-NEXT:   %14 = extractvalue { ptr, ptr } %11, 1
// CHECK-NEXT:   %15 = extractvalue { ptr, ptr } %11, 0
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %15)
// CHECK-NEXT:   %16 = call %reflect.Value %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %14, %"{{.*}}/runtime/internal/runtime.eface" %13)
// CHECK-NEXT:   store %reflect.Value %9, ptr %3, align 8
// CHECK-NEXT:   store %reflect.Value %16, ptr %10, align 8
// CHECK-NEXT:   %17 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 1
// CHECK-NEXT:   %18 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %17, i32 0, i32 0
// CHECK-NEXT:   %19 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 1, ptr %20, align 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %20, 1
// CHECK-NEXT:   %22 = extractvalue { ptr, ptr } %19, 1
// CHECK-NEXT:   %23 = extractvalue { ptr, ptr } %19, 0
// CHECK-NEXT:   %__llgo_funcval_code2 = call ptr asm "", "=r,0"(ptr %23)
// CHECK-NEXT:   %24 = call %reflect.Value %__llgo_funcval_code2(ptr {{(nest|swiftself)}} %22, %"{{.*}}/runtime/internal/runtime.eface" %21)
// CHECK-NEXT:   %25 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %17, i32 0, i32 1
// CHECK-NEXT:   %26 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %27 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 1, ptr %27, align 1
// CHECK-NEXT:   %28 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %27, 1
// CHECK-NEXT:   %29 = extractvalue { ptr, ptr } %26, 1
// CHECK-NEXT:   %30 = extractvalue { ptr, ptr } %26, 0
// CHECK-NEXT:   %__llgo_funcval_code3 = call ptr asm "", "=r,0"(ptr %30)
// CHECK-NEXT:   %31 = call %reflect.Value %__llgo_funcval_code3(ptr {{(nest|swiftself)}} %29, %"{{.*}}/runtime/internal/runtime.eface" %28)
// CHECK-NEXT:   store %reflect.Value %24, ptr %18, align 8
// CHECK-NEXT:   store %reflect.Value %31, ptr %25, align 8
// CHECK-NEXT:   %32 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 2
// CHECK-NEXT:   %33 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %32, i32 0, i32 0
// CHECK-NEXT:   %34 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %35 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 2, ptr %35, align 1
// CHECK-NEXT:   %36 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %35, 1
// CHECK-NEXT:   %37 = extractvalue { ptr, ptr } %34, 1
// CHECK-NEXT:   %38 = extractvalue { ptr, ptr } %34, 0
// CHECK-NEXT:   %__llgo_funcval_code4 = call ptr asm "", "=r,0"(ptr %38)
// CHECK-NEXT:   %39 = call %reflect.Value %__llgo_funcval_code4(ptr {{(nest|swiftself)}} %37, %"{{.*}}/runtime/internal/runtime.eface" %36)
// CHECK-NEXT:   %40 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %32, i32 0, i32 1
// CHECK-NEXT:   %41 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %42 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 2, ptr %42, align 1
// CHECK-NEXT:   %43 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %42, 1
// CHECK-NEXT:   %44 = extractvalue { ptr, ptr } %41, 1
// CHECK-NEXT:   %45 = extractvalue { ptr, ptr } %41, 0
// CHECK-NEXT:   %__llgo_funcval_code5 = call ptr asm "", "=r,0"(ptr %45)
// CHECK-NEXT:   %46 = call %reflect.Value %__llgo_funcval_code5(ptr {{(nest|swiftself)}} %44, %"{{.*}}/runtime/internal/runtime.eface" %43)
// CHECK-NEXT:   store %reflect.Value %39, ptr %33, align 8
// CHECK-NEXT:   store %reflect.Value %46, ptr %40, align 8
// CHECK-NEXT:   %47 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 3
// CHECK-NEXT:   %48 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %47, i32 0, i32 0
// CHECK-NEXT:   %49 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %50 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 3, ptr %50, align 1
// CHECK-NEXT:   %51 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %50, 1
// CHECK-NEXT:   %52 = extractvalue { ptr, ptr } %49, 1
// CHECK-NEXT:   %53 = extractvalue { ptr, ptr } %49, 0
// CHECK-NEXT:   %__llgo_funcval_code6 = call ptr asm "", "=r,0"(ptr %53)
// CHECK-NEXT:   %54 = call %reflect.Value %__llgo_funcval_code6(ptr {{(nest|swiftself)}} %52, %"{{.*}}/runtime/internal/runtime.eface" %51)
// CHECK-NEXT:   %55 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %47, i32 0, i32 1
// CHECK-NEXT:   %56 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %57 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 3, ptr %57, align 1
// CHECK-NEXT:   %58 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %57, 1
// CHECK-NEXT:   %59 = extractvalue { ptr, ptr } %56, 1
// CHECK-NEXT:   %60 = extractvalue { ptr, ptr } %56, 0
// CHECK-NEXT:   %__llgo_funcval_code7 = call ptr asm "", "=r,0"(ptr %60)
// CHECK-NEXT:   %61 = call %reflect.Value %__llgo_funcval_code7(ptr {{(nest|swiftself)}} %59, %"{{.*}}/runtime/internal/runtime.eface" %58)
// CHECK-NEXT:   store %reflect.Value %54, ptr %48, align 8
// CHECK-NEXT:   store %reflect.Value %61, ptr %55, align 8
// CHECK-NEXT:   %62 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 4
// CHECK-NEXT:   %63 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %62, i32 0, i32 0
// CHECK-NEXT:   %64 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %65 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 4, ptr %65, align 1
// CHECK-NEXT:   %66 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %65, 1
// CHECK-NEXT:   %67 = extractvalue { ptr, ptr } %64, 1
// CHECK-NEXT:   %68 = extractvalue { ptr, ptr } %64, 0
// CHECK-NEXT:   %__llgo_funcval_code8 = call ptr asm "", "=r,0"(ptr %68)
// CHECK-NEXT:   %69 = call %reflect.Value %__llgo_funcval_code8(ptr {{(nest|swiftself)}} %67, %"{{.*}}/runtime/internal/runtime.eface" %66)
// CHECK-NEXT:   %70 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %62, i32 0, i32 1
// CHECK-NEXT:   %71 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %72 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 4, ptr %72, align 2
// CHECK-NEXT:   %73 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %72, 1
// CHECK-NEXT:   %74 = extractvalue { ptr, ptr } %71, 1
// CHECK-NEXT:   %75 = extractvalue { ptr, ptr } %71, 0
// CHECK-NEXT:   %__llgo_funcval_code9 = call ptr asm "", "=r,0"(ptr %75)
// CHECK-NEXT:   %76 = call %reflect.Value %__llgo_funcval_code9(ptr {{(nest|swiftself)}} %74, %"{{.*}}/runtime/internal/runtime.eface" %73)
// CHECK-NEXT:   store %reflect.Value %69, ptr %63, align 8
// CHECK-NEXT:   store %reflect.Value %76, ptr %70, align 8
// CHECK-NEXT:   %77 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 5
// CHECK-NEXT:   %78 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %77, i32 0, i32 0
// CHECK-NEXT:   %79 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %80 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 5, ptr %80, align 2
// CHECK-NEXT:   %81 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %80, 1
// CHECK-NEXT:   %82 = extractvalue { ptr, ptr } %79, 1
// CHECK-NEXT:   %83 = extractvalue { ptr, ptr } %79, 0
// CHECK-NEXT:   %__llgo_funcval_code10 = call ptr asm "", "=r,0"(ptr %83)
// CHECK-NEXT:   %84 = call %reflect.Value %__llgo_funcval_code10(ptr {{(nest|swiftself)}} %82, %"{{.*}}/runtime/internal/runtime.eface" %81)
// CHECK-NEXT:   %85 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %77, i32 0, i32 1
// CHECK-NEXT:   %86 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %87 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 5, ptr %87, align 1
// CHECK-NEXT:   %88 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %87, 1
// CHECK-NEXT:   %89 = extractvalue { ptr, ptr } %86, 1
// CHECK-NEXT:   %90 = extractvalue { ptr, ptr } %86, 0
// CHECK-NEXT:   %__llgo_funcval_code11 = call ptr asm "", "=r,0"(ptr %90)
// CHECK-NEXT:   %91 = call %reflect.Value %__llgo_funcval_code11(ptr {{(nest|swiftself)}} %89, %"{{.*}}/runtime/internal/runtime.eface" %88)
// CHECK-NEXT:   store %reflect.Value %84, ptr %78, align 8
// CHECK-NEXT:   store %reflect.Value %91, ptr %85, align 8
// CHECK-NEXT:   %92 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 6
// CHECK-NEXT:   %93 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %92, i32 0, i32 0
// CHECK-NEXT:   %94 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %95 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 6, ptr %95, align 1
// CHECK-NEXT:   %96 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %95, 1
// CHECK-NEXT:   %97 = extractvalue { ptr, ptr } %94, 1
// CHECK-NEXT:   %98 = extractvalue { ptr, ptr } %94, 0
// CHECK-NEXT:   %__llgo_funcval_code12 = call ptr asm "", "=r,0"(ptr %98)
// CHECK-NEXT:   %99 = call %reflect.Value %__llgo_funcval_code12(ptr {{(nest|swiftself)}} %97, %"{{.*}}/runtime/internal/runtime.eface" %96)
// CHECK-NEXT:   %100 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %92, i32 0, i32 1
// CHECK-NEXT:   %101 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %102 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 6, ptr %102, align 2
// CHECK-NEXT:   %103 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %102, 1
// CHECK-NEXT:   %104 = extractvalue { ptr, ptr } %101, 1
// CHECK-NEXT:   %105 = extractvalue { ptr, ptr } %101, 0
// CHECK-NEXT:   %__llgo_funcval_code13 = call ptr asm "", "=r,0"(ptr %105)
// CHECK-NEXT:   %106 = call %reflect.Value %__llgo_funcval_code13(ptr {{(nest|swiftself)}} %104, %"{{.*}}/runtime/internal/runtime.eface" %103)
// CHECK-NEXT:   store %reflect.Value %99, ptr %93, align 8
// CHECK-NEXT:   store %reflect.Value %106, ptr %100, align 8
// CHECK-NEXT:   %107 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 7
// CHECK-NEXT:   %108 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %107, i32 0, i32 0
// CHECK-NEXT:   %109 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %110 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 7, ptr %110, align 2
// CHECK-NEXT:   %111 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %110, 1
// CHECK-NEXT:   %112 = extractvalue { ptr, ptr } %109, 1
// CHECK-NEXT:   %113 = extractvalue { ptr, ptr } %109, 0
// CHECK-NEXT:   %__llgo_funcval_code14 = call ptr asm "", "=r,0"(ptr %113)
// CHECK-NEXT:   %114 = call %reflect.Value %__llgo_funcval_code14(ptr {{(nest|swiftself)}} %112, %"{{.*}}/runtime/internal/runtime.eface" %111)
// CHECK-NEXT:   %115 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %107, i32 0, i32 1
// CHECK-NEXT:   %116 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %117 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 7, ptr %117, align 1
// CHECK-NEXT:   %118 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %117, 1
// CHECK-NEXT:   %119 = extractvalue { ptr, ptr } %116, 1
// CHECK-NEXT:   %120 = extractvalue { ptr, ptr } %116, 0
// CHECK-NEXT:   %__llgo_funcval_code15 = call ptr asm "", "=r,0"(ptr %120)
// CHECK-NEXT:   %121 = call %reflect.Value %__llgo_funcval_code15(ptr {{(nest|swiftself)}} %119, %"{{.*}}/runtime/internal/runtime.eface" %118)
// CHECK-NEXT:   store %reflect.Value %114, ptr %108, align 8
// CHECK-NEXT:   store %reflect.Value %121, ptr %115, align 8
// CHECK-NEXT:   %122 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 8
// CHECK-NEXT:   %123 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %122, i32 0, i32 0
// CHECK-NEXT:   %124 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %125 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 8, ptr %125, align 1
// CHECK-NEXT:   %126 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %125, 1
// CHECK-NEXT:   %127 = extractvalue { ptr, ptr } %124, 1
// CHECK-NEXT:   %128 = extractvalue { ptr, ptr } %124, 0
// CHECK-NEXT:   %__llgo_funcval_code16 = call ptr asm "", "=r,0"(ptr %128)
// CHECK-NEXT:   %129 = call %reflect.Value %__llgo_funcval_code16(ptr {{(nest|swiftself)}} %127, %"{{.*}}/runtime/internal/runtime.eface" %126)
// CHECK-NEXT:   %130 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %122, i32 0, i32 1
// CHECK-NEXT:   %131 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %132 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 8, ptr %132, align 4
// CHECK-NEXT:   %133 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %132, 1
// CHECK-NEXT:   %134 = extractvalue { ptr, ptr } %131, 1
// CHECK-NEXT:   %135 = extractvalue { ptr, ptr } %131, 0
// CHECK-NEXT:   %__llgo_funcval_code17 = call ptr asm "", "=r,0"(ptr %135)
// CHECK-NEXT:   %136 = call %reflect.Value %__llgo_funcval_code17(ptr {{(nest|swiftself)}} %134, %"{{.*}}/runtime/internal/runtime.eface" %133)
// CHECK-NEXT:   store %reflect.Value %129, ptr %123, align 8
// CHECK-NEXT:   store %reflect.Value %136, ptr %130, align 8
// CHECK-NEXT:   %137 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 9
// CHECK-NEXT:   %138 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %137, i32 0, i32 0
// CHECK-NEXT:   %139 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %140 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 9, ptr %140, align 4
// CHECK-NEXT:   %141 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %140, 1
// CHECK-NEXT:   %142 = extractvalue { ptr, ptr } %139, 1
// CHECK-NEXT:   %143 = extractvalue { ptr, ptr } %139, 0
// CHECK-NEXT:   %__llgo_funcval_code18 = call ptr asm "", "=r,0"(ptr %143)
// CHECK-NEXT:   %144 = call %reflect.Value %__llgo_funcval_code18(ptr {{(nest|swiftself)}} %142, %"{{.*}}/runtime/internal/runtime.eface" %141)
// CHECK-NEXT:   %145 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %137, i32 0, i32 1
// CHECK-NEXT:   %146 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %147 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 9, ptr %147, align 1
// CHECK-NEXT:   %148 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %147, 1
// CHECK-NEXT:   %149 = extractvalue { ptr, ptr } %146, 1
// CHECK-NEXT:   %150 = extractvalue { ptr, ptr } %146, 0
// CHECK-NEXT:   %__llgo_funcval_code19 = call ptr asm "", "=r,0"(ptr %150)
// CHECK-NEXT:   %151 = call %reflect.Value %__llgo_funcval_code19(ptr {{(nest|swiftself)}} %149, %"{{.*}}/runtime/internal/runtime.eface" %148)
// CHECK-NEXT:   store %reflect.Value %144, ptr %138, align 8
// CHECK-NEXT:   store %reflect.Value %151, ptr %145, align 8
// CHECK-NEXT:   %152 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 10
// CHECK-NEXT:   %153 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %152, i32 0, i32 0
// CHECK-NEXT:   %154 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %155 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 10, ptr %155, align 1
// CHECK-NEXT:   %156 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %155, 1
// CHECK-NEXT:   %157 = extractvalue { ptr, ptr } %154, 1
// CHECK-NEXT:   %158 = extractvalue { ptr, ptr } %154, 0
// CHECK-NEXT:   %__llgo_funcval_code20 = call ptr asm "", "=r,0"(ptr %158)
// CHECK-NEXT:   %159 = call %reflect.Value %__llgo_funcval_code20(ptr {{(nest|swiftself)}} %157, %"{{.*}}/runtime/internal/runtime.eface" %156)
// CHECK-NEXT:   %160 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %152, i32 0, i32 1
// CHECK-NEXT:   %161 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %162 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 10, ptr %162, align 4
// CHECK-NEXT:   %163 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %162, 1
// CHECK-NEXT:   %164 = extractvalue { ptr, ptr } %161, 1
// CHECK-NEXT:   %165 = extractvalue { ptr, ptr } %161, 0
// CHECK-NEXT:   %__llgo_funcval_code21 = call ptr asm "", "=r,0"(ptr %165)
// CHECK-NEXT:   %166 = call %reflect.Value %__llgo_funcval_code21(ptr {{(nest|swiftself)}} %164, %"{{.*}}/runtime/internal/runtime.eface" %163)
// CHECK-NEXT:   store %reflect.Value %159, ptr %153, align 8
// CHECK-NEXT:   store %reflect.Value %166, ptr %160, align 8
// CHECK-NEXT:   %167 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 11
// CHECK-NEXT:   %168 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %167, i32 0, i32 0
// CHECK-NEXT:   %169 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %170 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 11, ptr %170, align 4
// CHECK-NEXT:   %171 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %170, 1
// CHECK-NEXT:   %172 = extractvalue { ptr, ptr } %169, 1
// CHECK-NEXT:   %173 = extractvalue { ptr, ptr } %169, 0
// CHECK-NEXT:   %__llgo_funcval_code22 = call ptr asm "", "=r,0"(ptr %173)
// CHECK-NEXT:   %174 = call %reflect.Value %__llgo_funcval_code22(ptr {{(nest|swiftself)}} %172, %"{{.*}}/runtime/internal/runtime.eface" %171)
// CHECK-NEXT:   %175 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %167, i32 0, i32 1
// CHECK-NEXT:   %176 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %177 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 11, ptr %177, align 1
// CHECK-NEXT:   %178 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %177, 1
// CHECK-NEXT:   %179 = extractvalue { ptr, ptr } %176, 1
// CHECK-NEXT:   %180 = extractvalue { ptr, ptr } %176, 0
// CHECK-NEXT:   %__llgo_funcval_code23 = call ptr asm "", "=r,0"(ptr %180)
// CHECK-NEXT:   %181 = call %reflect.Value %__llgo_funcval_code23(ptr {{(nest|swiftself)}} %179, %"{{.*}}/runtime/internal/runtime.eface" %178)
// CHECK-NEXT:   store %reflect.Value %174, ptr %168, align 8
// CHECK-NEXT:   store %reflect.Value %181, ptr %175, align 8
// CHECK-NEXT:   %182 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 12
// CHECK-NEXT:   %183 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %182, i32 0, i32 0
// CHECK-NEXT:   %184 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %185 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 12, ptr %185, align 1
// CHECK-NEXT:   %186 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %185, 1
// CHECK-NEXT:   %187 = extractvalue { ptr, ptr } %184, 1
// CHECK-NEXT:   %188 = extractvalue { ptr, ptr } %184, 0
// CHECK-NEXT:   %__llgo_funcval_code24 = call ptr asm "", "=r,0"(ptr %188)
// CHECK-NEXT:   %189 = call %reflect.Value %__llgo_funcval_code24(ptr {{(nest|swiftself)}} %187, %"{{.*}}/runtime/internal/runtime.eface" %186)
// CHECK-NEXT:   %190 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %182, i32 0, i32 1
// CHECK-NEXT:   %191 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %192 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 12, ptr %192, align 8
// CHECK-NEXT:   %193 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %192, 1
// CHECK-NEXT:   %194 = extractvalue { ptr, ptr } %191, 1
// CHECK-NEXT:   %195 = extractvalue { ptr, ptr } %191, 0
// CHECK-NEXT:   %__llgo_funcval_code25 = call ptr asm "", "=r,0"(ptr %195)
// CHECK-NEXT:   %196 = call %reflect.Value %__llgo_funcval_code25(ptr {{(nest|swiftself)}} %194, %"{{.*}}/runtime/internal/runtime.eface" %193)
// CHECK-NEXT:   store %reflect.Value %189, ptr %183, align 8
// CHECK-NEXT:   store %reflect.Value %196, ptr %190, align 8
// CHECK-NEXT:   %197 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 13
// CHECK-NEXT:   %198 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %197, i32 0, i32 0
// CHECK-NEXT:   %199 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %200 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 13, ptr %200, align 8
// CHECK-NEXT:   %201 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %200, 1
// CHECK-NEXT:   %202 = extractvalue { ptr, ptr } %199, 1
// CHECK-NEXT:   %203 = extractvalue { ptr, ptr } %199, 0
// CHECK-NEXT:   %__llgo_funcval_code26 = call ptr asm "", "=r,0"(ptr %203)
// CHECK-NEXT:   %204 = call %reflect.Value %__llgo_funcval_code26(ptr {{(nest|swiftself)}} %202, %"{{.*}}/runtime/internal/runtime.eface" %201)
// CHECK-NEXT:   %205 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %197, i32 0, i32 1
// CHECK-NEXT:   %206 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %207 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 13, ptr %207, align 1
// CHECK-NEXT:   %208 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %207, 1
// CHECK-NEXT:   %209 = extractvalue { ptr, ptr } %206, 1
// CHECK-NEXT:   %210 = extractvalue { ptr, ptr } %206, 0
// CHECK-NEXT:   %__llgo_funcval_code27 = call ptr asm "", "=r,0"(ptr %210)
// CHECK-NEXT:   %211 = call %reflect.Value %__llgo_funcval_code27(ptr {{(nest|swiftself)}} %209, %"{{.*}}/runtime/internal/runtime.eface" %208)
// CHECK-NEXT:   store %reflect.Value %204, ptr %198, align 8
// CHECK-NEXT:   store %reflect.Value %211, ptr %205, align 8
// CHECK-NEXT:   %212 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 14
// CHECK-NEXT:   %213 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %212, i32 0, i32 0
// CHECK-NEXT:   %214 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %215 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 14, ptr %215, align 1
// CHECK-NEXT:   %216 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %215, 1
// CHECK-NEXT:   %217 = extractvalue { ptr, ptr } %214, 1
// CHECK-NEXT:   %218 = extractvalue { ptr, ptr } %214, 0
// CHECK-NEXT:   %__llgo_funcval_code28 = call ptr asm "", "=r,0"(ptr %218)
// CHECK-NEXT:   %219 = call %reflect.Value %__llgo_funcval_code28(ptr {{(nest|swiftself)}} %217, %"{{.*}}/runtime/internal/runtime.eface" %216)
// CHECK-NEXT:   %220 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %212, i32 0, i32 1
// CHECK-NEXT:   %221 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %222 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 14, ptr %222, align 8
// CHECK-NEXT:   %223 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %222, 1
// CHECK-NEXT:   %224 = extractvalue { ptr, ptr } %221, 1
// CHECK-NEXT:   %225 = extractvalue { ptr, ptr } %221, 0
// CHECK-NEXT:   %__llgo_funcval_code29 = call ptr asm "", "=r,0"(ptr %225)
// CHECK-NEXT:   %226 = call %reflect.Value %__llgo_funcval_code29(ptr {{(nest|swiftself)}} %224, %"{{.*}}/runtime/internal/runtime.eface" %223)
// CHECK-NEXT:   store %reflect.Value %219, ptr %213, align 8
// CHECK-NEXT:   store %reflect.Value %226, ptr %220, align 8
// CHECK-NEXT:   %227 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 15
// CHECK-NEXT:   %228 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %227, i32 0, i32 0
// CHECK-NEXT:   %229 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %230 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 15, ptr %230, align 8
// CHECK-NEXT:   %231 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %230, 1
// CHECK-NEXT:   %232 = extractvalue { ptr, ptr } %229, 1
// CHECK-NEXT:   %233 = extractvalue { ptr, ptr } %229, 0
// CHECK-NEXT:   %__llgo_funcval_code30 = call ptr asm "", "=r,0"(ptr %233)
// CHECK-NEXT:   %234 = call %reflect.Value %__llgo_funcval_code30(ptr {{(nest|swiftself)}} %232, %"{{.*}}/runtime/internal/runtime.eface" %231)
// CHECK-NEXT:   %235 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %227, i32 0, i32 1
// CHECK-NEXT:   %236 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %237 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 15, ptr %237, align 1
// CHECK-NEXT:   %238 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %237, 1
// CHECK-NEXT:   %239 = extractvalue { ptr, ptr } %236, 1
// CHECK-NEXT:   %240 = extractvalue { ptr, ptr } %236, 0
// CHECK-NEXT:   %__llgo_funcval_code31 = call ptr asm "", "=r,0"(ptr %240)
// CHECK-NEXT:   %241 = call %reflect.Value %__llgo_funcval_code31(ptr {{(nest|swiftself)}} %239, %"{{.*}}/runtime/internal/runtime.eface" %238)
// CHECK-NEXT:   store %reflect.Value %234, ptr %228, align 8
// CHECK-NEXT:   store %reflect.Value %241, ptr %235, align 8
// CHECK-NEXT:   %242 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 16
// CHECK-NEXT:   %243 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %242, i32 0, i32 0
// CHECK-NEXT:   %244 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %245 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 16, ptr %245, align 1
// CHECK-NEXT:   %246 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %245, 1
// CHECK-NEXT:   %247 = extractvalue { ptr, ptr } %244, 1
// CHECK-NEXT:   %248 = extractvalue { ptr, ptr } %244, 0
// CHECK-NEXT:   %__llgo_funcval_code32 = call ptr asm "", "=r,0"(ptr %248)
// CHECK-NEXT:   %249 = call %reflect.Value %__llgo_funcval_code32(ptr {{(nest|swiftself)}} %247, %"{{.*}}/runtime/internal/runtime.eface" %246)
// CHECK-NEXT:   %250 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %242, i32 0, i32 1
// CHECK-NEXT:   %251 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %252 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 16, ptr %252, align 8
// CHECK-NEXT:   %253 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %252, 1
// CHECK-NEXT:   %254 = extractvalue { ptr, ptr } %251, 1
// CHECK-NEXT:   %255 = extractvalue { ptr, ptr } %251, 0
// CHECK-NEXT:   %__llgo_funcval_code33 = call ptr asm "", "=r,0"(ptr %255)
// CHECK-NEXT:   %256 = call %reflect.Value %__llgo_funcval_code33(ptr {{(nest|swiftself)}} %254, %"{{.*}}/runtime/internal/runtime.eface" %253)
// CHECK-NEXT:   store %reflect.Value %249, ptr %243, align 8
// CHECK-NEXT:   store %reflect.Value %256, ptr %250, align 8
// CHECK-NEXT:   %257 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 17
// CHECK-NEXT:   %258 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %257, i32 0, i32 0
// CHECK-NEXT:   %259 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %260 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 17, ptr %260, align 8
// CHECK-NEXT:   %261 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %260, 1
// CHECK-NEXT:   %262 = extractvalue { ptr, ptr } %259, 1
// CHECK-NEXT:   %263 = extractvalue { ptr, ptr } %259, 0
// CHECK-NEXT:   %__llgo_funcval_code34 = call ptr asm "", "=r,0"(ptr %263)
// CHECK-NEXT:   %264 = call %reflect.Value %__llgo_funcval_code34(ptr {{(nest|swiftself)}} %262, %"{{.*}}/runtime/internal/runtime.eface" %261)
// CHECK-NEXT:   %265 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %257, i32 0, i32 1
// CHECK-NEXT:   %266 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %267 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 17, ptr %267, align 1
// CHECK-NEXT:   %268 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %267, 1
// CHECK-NEXT:   %269 = extractvalue { ptr, ptr } %266, 1
// CHECK-NEXT:   %270 = extractvalue { ptr, ptr } %266, 0
// CHECK-NEXT:   %__llgo_funcval_code35 = call ptr asm "", "=r,0"(ptr %270)
// CHECK-NEXT:   %271 = call %reflect.Value %__llgo_funcval_code35(ptr {{(nest|swiftself)}} %269, %"{{.*}}/runtime/internal/runtime.eface" %268)
// CHECK-NEXT:   store %reflect.Value %264, ptr %258, align 8
// CHECK-NEXT:   store %reflect.Value %271, ptr %265, align 8
// CHECK-NEXT:   %272 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 18
// CHECK-NEXT:   %273 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %272, i32 0, i32 0
// CHECK-NEXT:   %274 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %275 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 18, ptr %275, align 1
// CHECK-NEXT:   %276 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %275, 1
// CHECK-NEXT:   %277 = extractvalue { ptr, ptr } %274, 1
// CHECK-NEXT:   %278 = extractvalue { ptr, ptr } %274, 0
// CHECK-NEXT:   %__llgo_funcval_code36 = call ptr asm "", "=r,0"(ptr %278)
// CHECK-NEXT:   %279 = call %reflect.Value %__llgo_funcval_code36(ptr {{(nest|swiftself)}} %277, %"{{.*}}/runtime/internal/runtime.eface" %276)
// CHECK-NEXT:   %280 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %272, i32 0, i32 1
// CHECK-NEXT:   %281 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %282 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 18, ptr %282, align 8
// CHECK-NEXT:   %283 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %282, 1
// CHECK-NEXT:   %284 = extractvalue { ptr, ptr } %281, 1
// CHECK-NEXT:   %285 = extractvalue { ptr, ptr } %281, 0
// CHECK-NEXT:   %__llgo_funcval_code37 = call ptr asm "", "=r,0"(ptr %285)
// CHECK-NEXT:   %286 = call %reflect.Value %__llgo_funcval_code37(ptr {{(nest|swiftself)}} %284, %"{{.*}}/runtime/internal/runtime.eface" %283)
// CHECK-NEXT:   store %reflect.Value %279, ptr %273, align 8
// CHECK-NEXT:   store %reflect.Value %286, ptr %280, align 8
// CHECK-NEXT:   %287 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 19
// CHECK-NEXT:   %288 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %287, i32 0, i32 0
// CHECK-NEXT:   %289 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %290 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 19, ptr %290, align 8
// CHECK-NEXT:   %291 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %290, 1
// CHECK-NEXT:   %292 = extractvalue { ptr, ptr } %289, 1
// CHECK-NEXT:   %293 = extractvalue { ptr, ptr } %289, 0
// CHECK-NEXT:   %__llgo_funcval_code38 = call ptr asm "", "=r,0"(ptr %293)
// CHECK-NEXT:   %294 = call %reflect.Value %__llgo_funcval_code38(ptr {{(nest|swiftself)}} %292, %"{{.*}}/runtime/internal/runtime.eface" %291)
// CHECK-NEXT:   %295 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %287, i32 0, i32 1
// CHECK-NEXT:   %296 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %297 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 19, ptr %297, align 1
// CHECK-NEXT:   %298 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %297, 1
// CHECK-NEXT:   %299 = extractvalue { ptr, ptr } %296, 1
// CHECK-NEXT:   %300 = extractvalue { ptr, ptr } %296, 0
// CHECK-NEXT:   %__llgo_funcval_code39 = call ptr asm "", "=r,0"(ptr %300)
// CHECK-NEXT:   %301 = call %reflect.Value %__llgo_funcval_code39(ptr {{(nest|swiftself)}} %299, %"{{.*}}/runtime/internal/runtime.eface" %298)
// CHECK-NEXT:   store %reflect.Value %294, ptr %288, align 8
// CHECK-NEXT:   store %reflect.Value %301, ptr %295, align 8
// CHECK-NEXT:   %302 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 20
// CHECK-NEXT:   %303 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %302, i32 0, i32 0
// CHECK-NEXT:   %304 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %305 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 20, ptr %305, align 1
// CHECK-NEXT:   %306 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %305, 1
// CHECK-NEXT:   %307 = extractvalue { ptr, ptr } %304, 1
// CHECK-NEXT:   %308 = extractvalue { ptr, ptr } %304, 0
// CHECK-NEXT:   %__llgo_funcval_code40 = call ptr asm "", "=r,0"(ptr %308)
// CHECK-NEXT:   %309 = call %reflect.Value %__llgo_funcval_code40(ptr {{(nest|swiftself)}} %307, %"{{.*}}/runtime/internal/runtime.eface" %306)
// CHECK-NEXT:   %310 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %302, i32 0, i32 1
// CHECK-NEXT:   %311 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %312 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 20, ptr %312, align 8
// CHECK-NEXT:   %313 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %312, 1
// CHECK-NEXT:   %314 = extractvalue { ptr, ptr } %311, 1
// CHECK-NEXT:   %315 = extractvalue { ptr, ptr } %311, 0
// CHECK-NEXT:   %__llgo_funcval_code41 = call ptr asm "", "=r,0"(ptr %315)
// CHECK-NEXT:   %316 = call %reflect.Value %__llgo_funcval_code41(ptr {{(nest|swiftself)}} %314, %"{{.*}}/runtime/internal/runtime.eface" %313)
// CHECK-NEXT:   store %reflect.Value %309, ptr %303, align 8
// CHECK-NEXT:   store %reflect.Value %316, ptr %310, align 8
// CHECK-NEXT:   %317 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 21
// CHECK-NEXT:   %318 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %317, i32 0, i32 0
// CHECK-NEXT:   %319 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %320 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 21, ptr %320, align 8
// CHECK-NEXT:   %321 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %320, 1
// CHECK-NEXT:   %322 = extractvalue { ptr, ptr } %319, 1
// CHECK-NEXT:   %323 = extractvalue { ptr, ptr } %319, 0
// CHECK-NEXT:   %__llgo_funcval_code42 = call ptr asm "", "=r,0"(ptr %323)
// CHECK-NEXT:   %324 = call %reflect.Value %__llgo_funcval_code42(ptr {{(nest|swiftself)}} %322, %"{{.*}}/runtime/internal/runtime.eface" %321)
// CHECK-NEXT:   %325 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %317, i32 0, i32 1
// CHECK-NEXT:   %326 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %327 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 21, ptr %327, align 1
// CHECK-NEXT:   %328 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %327, 1
// CHECK-NEXT:   %329 = extractvalue { ptr, ptr } %326, 1
// CHECK-NEXT:   %330 = extractvalue { ptr, ptr } %326, 0
// CHECK-NEXT:   %__llgo_funcval_code43 = call ptr asm "", "=r,0"(ptr %330)
// CHECK-NEXT:   %331 = call %reflect.Value %__llgo_funcval_code43(ptr {{(nest|swiftself)}} %329, %"{{.*}}/runtime/internal/runtime.eface" %328)
// CHECK-NEXT:   store %reflect.Value %324, ptr %318, align 8
// CHECK-NEXT:   store %reflect.Value %331, ptr %325, align 8
// CHECK-NEXT:   %332 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 22
// CHECK-NEXT:   %333 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %332, i32 0, i32 0
// CHECK-NEXT:   %334 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %335 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 22, ptr %335, align 1
// CHECK-NEXT:   %336 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %335, 1
// CHECK-NEXT:   %337 = extractvalue { ptr, ptr } %334, 1
// CHECK-NEXT:   %338 = extractvalue { ptr, ptr } %334, 0
// CHECK-NEXT:   %__llgo_funcval_code44 = call ptr asm "", "=r,0"(ptr %338)
// CHECK-NEXT:   %339 = call %reflect.Value %__llgo_funcval_code44(ptr {{(nest|swiftself)}} %337, %"{{.*}}/runtime/internal/runtime.eface" %336)
// CHECK-NEXT:   %340 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %332, i32 0, i32 1
// CHECK-NEXT:   %341 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %342 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 2.200000e+01, ptr %342, align 4
// CHECK-NEXT:   %343 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %342, 1
// CHECK-NEXT:   %344 = extractvalue { ptr, ptr } %341, 1
// CHECK-NEXT:   %345 = extractvalue { ptr, ptr } %341, 0
// CHECK-NEXT:   %__llgo_funcval_code45 = call ptr asm "", "=r,0"(ptr %345)
// CHECK-NEXT:   %346 = call %reflect.Value %__llgo_funcval_code45(ptr {{(nest|swiftself)}} %344, %"{{.*}}/runtime/internal/runtime.eface" %343)
// CHECK-NEXT:   store %reflect.Value %339, ptr %333, align 8
// CHECK-NEXT:   store %reflect.Value %346, ptr %340, align 8
// CHECK-NEXT:   %347 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 23
// CHECK-NEXT:   %348 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %347, i32 0, i32 0
// CHECK-NEXT:   %349 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %350 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 2.300000e+01, ptr %350, align 4
// CHECK-NEXT:   %351 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %350, 1
// CHECK-NEXT:   %352 = extractvalue { ptr, ptr } %349, 1
// CHECK-NEXT:   %353 = extractvalue { ptr, ptr } %349, 0
// CHECK-NEXT:   %__llgo_funcval_code46 = call ptr asm "", "=r,0"(ptr %353)
// CHECK-NEXT:   %354 = call %reflect.Value %__llgo_funcval_code46(ptr {{(nest|swiftself)}} %352, %"{{.*}}/runtime/internal/runtime.eface" %351)
// CHECK-NEXT:   %355 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %347, i32 0, i32 1
// CHECK-NEXT:   %356 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %357 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 23, ptr %357, align 1
// CHECK-NEXT:   %358 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %357, 1
// CHECK-NEXT:   %359 = extractvalue { ptr, ptr } %356, 1
// CHECK-NEXT:   %360 = extractvalue { ptr, ptr } %356, 0
// CHECK-NEXT:   %__llgo_funcval_code47 = call ptr asm "", "=r,0"(ptr %360)
// CHECK-NEXT:   %361 = call %reflect.Value %__llgo_funcval_code47(ptr {{(nest|swiftself)}} %359, %"{{.*}}/runtime/internal/runtime.eface" %358)
// CHECK-NEXT:   store %reflect.Value %354, ptr %348, align 8
// CHECK-NEXT:   store %reflect.Value %361, ptr %355, align 8
// CHECK-NEXT:   %362 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 24
// CHECK-NEXT:   %363 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %362, i32 0, i32 0
// CHECK-NEXT:   %364 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %365 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 24, ptr %365, align 1
// CHECK-NEXT:   %366 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %365, 1
// CHECK-NEXT:   %367 = extractvalue { ptr, ptr } %364, 1
// CHECK-NEXT:   %368 = extractvalue { ptr, ptr } %364, 0
// CHECK-NEXT:   %__llgo_funcval_code48 = call ptr asm "", "=r,0"(ptr %368)
// CHECK-NEXT:   %369 = call %reflect.Value %__llgo_funcval_code48(ptr {{(nest|swiftself)}} %367, %"{{.*}}/runtime/internal/runtime.eface" %366)
// CHECK-NEXT:   %370 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %362, i32 0, i32 1
// CHECK-NEXT:   %371 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %372 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.400000e+01, ptr %372, align 8
// CHECK-NEXT:   %373 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %372, 1
// CHECK-NEXT:   %374 = extractvalue { ptr, ptr } %371, 1
// CHECK-NEXT:   %375 = extractvalue { ptr, ptr } %371, 0
// CHECK-NEXT:   %__llgo_funcval_code49 = call ptr asm "", "=r,0"(ptr %375)
// CHECK-NEXT:   %376 = call %reflect.Value %__llgo_funcval_code49(ptr {{(nest|swiftself)}} %374, %"{{.*}}/runtime/internal/runtime.eface" %373)
// CHECK-NEXT:   store %reflect.Value %369, ptr %363, align 8
// CHECK-NEXT:   store %reflect.Value %376, ptr %370, align 8
// CHECK-NEXT:   %377 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 25
// CHECK-NEXT:   %378 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %377, i32 0, i32 0
// CHECK-NEXT:   %379 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %380 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 2.500000e+01, ptr %380, align 8
// CHECK-NEXT:   %381 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %380, 1
// CHECK-NEXT:   %382 = extractvalue { ptr, ptr } %379, 1
// CHECK-NEXT:   %383 = extractvalue { ptr, ptr } %379, 0
// CHECK-NEXT:   %__llgo_funcval_code50 = call ptr asm "", "=r,0"(ptr %383)
// CHECK-NEXT:   %384 = call %reflect.Value %__llgo_funcval_code50(ptr {{(nest|swiftself)}} %382, %"{{.*}}/runtime/internal/runtime.eface" %381)
// CHECK-NEXT:   %385 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %377, i32 0, i32 1
// CHECK-NEXT:   %386 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %387 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 25, ptr %387, align 1
// CHECK-NEXT:   %388 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %387, 1
// CHECK-NEXT:   %389 = extractvalue { ptr, ptr } %386, 1
// CHECK-NEXT:   %390 = extractvalue { ptr, ptr } %386, 0
// CHECK-NEXT:   %__llgo_funcval_code51 = call ptr asm "", "=r,0"(ptr %390)
// CHECK-NEXT:   %391 = call %reflect.Value %__llgo_funcval_code51(ptr {{(nest|swiftself)}} %389, %"{{.*}}/runtime/internal/runtime.eface" %388)
// CHECK-NEXT:   store %reflect.Value %384, ptr %378, align 8
// CHECK-NEXT:   store %reflect.Value %391, ptr %385, align 8
// CHECK-NEXT:   %392 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 26
// CHECK-NEXT:   %393 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %392, i32 0, i32 0
// CHECK-NEXT:   %394 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %395 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 26, ptr %395, align 1
// CHECK-NEXT:   %396 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %395, 1
// CHECK-NEXT:   %397 = extractvalue { ptr, ptr } %394, 1
// CHECK-NEXT:   %398 = extractvalue { ptr, ptr } %394, 0
// CHECK-NEXT:   %__llgo_funcval_code52 = call ptr asm "", "=r,0"(ptr %398)
// CHECK-NEXT:   %399 = call %reflect.Value %__llgo_funcval_code52(ptr {{(nest|swiftself)}} %397, %"{{.*}}/runtime/internal/runtime.eface" %396)
// CHECK-NEXT:   %400 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %392, i32 0, i32 1
// CHECK-NEXT:   %401 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %402 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 26, ptr %402, align 1
// CHECK-NEXT:   %403 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %402, 1
// CHECK-NEXT:   %404 = extractvalue { ptr, ptr } %401, 1
// CHECK-NEXT:   %405 = extractvalue { ptr, ptr } %401, 0
// CHECK-NEXT:   %__llgo_funcval_code53 = call ptr asm "", "=r,0"(ptr %405)
// CHECK-NEXT:   %406 = call %reflect.Value %__llgo_funcval_code53(ptr {{(nest|swiftself)}} %404, %"{{.*}}/runtime/internal/runtime.eface" %403)
// CHECK-NEXT:   store %reflect.Value %399, ptr %393, align 8
// CHECK-NEXT:   store %reflect.Value %406, ptr %400, align 8
// CHECK-NEXT:   %407 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 27
// CHECK-NEXT:   %408 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %407, i32 0, i32 0
// CHECK-NEXT:   %409 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %410 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 27, ptr %410, align 1
// CHECK-NEXT:   %411 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %410, 1
// CHECK-NEXT:   %412 = extractvalue { ptr, ptr } %409, 1
// CHECK-NEXT:   %413 = extractvalue { ptr, ptr } %409, 0
// CHECK-NEXT:   %__llgo_funcval_code54 = call ptr asm "", "=r,0"(ptr %413)
// CHECK-NEXT:   %414 = call %reflect.Value %__llgo_funcval_code54(ptr {{(nest|swiftself)}} %412, %"{{.*}}/runtime/internal/runtime.eface" %411)
// CHECK-NEXT:   %415 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %407, i32 0, i32 1
// CHECK-NEXT:   %416 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %417 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 27, ptr %417, align 2
// CHECK-NEXT:   %418 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %417, 1
// CHECK-NEXT:   %419 = extractvalue { ptr, ptr } %416, 1
// CHECK-NEXT:   %420 = extractvalue { ptr, ptr } %416, 0
// CHECK-NEXT:   %__llgo_funcval_code55 = call ptr asm "", "=r,0"(ptr %420)
// CHECK-NEXT:   %421 = call %reflect.Value %__llgo_funcval_code55(ptr {{(nest|swiftself)}} %419, %"{{.*}}/runtime/internal/runtime.eface" %418)
// CHECK-NEXT:   store %reflect.Value %414, ptr %408, align 8
// CHECK-NEXT:   store %reflect.Value %421, ptr %415, align 8
// CHECK-NEXT:   %422 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 28
// CHECK-NEXT:   %423 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %422, i32 0, i32 0
// CHECK-NEXT:   %424 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %425 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 28, ptr %425, align 2
// CHECK-NEXT:   %426 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %425, 1
// CHECK-NEXT:   %427 = extractvalue { ptr, ptr } %424, 1
// CHECK-NEXT:   %428 = extractvalue { ptr, ptr } %424, 0
// CHECK-NEXT:   %__llgo_funcval_code56 = call ptr asm "", "=r,0"(ptr %428)
// CHECK-NEXT:   %429 = call %reflect.Value %__llgo_funcval_code56(ptr {{(nest|swiftself)}} %427, %"{{.*}}/runtime/internal/runtime.eface" %426)
// CHECK-NEXT:   %430 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %422, i32 0, i32 1
// CHECK-NEXT:   %431 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %432 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 28, ptr %432, align 1
// CHECK-NEXT:   %433 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %432, 1
// CHECK-NEXT:   %434 = extractvalue { ptr, ptr } %431, 1
// CHECK-NEXT:   %435 = extractvalue { ptr, ptr } %431, 0
// CHECK-NEXT:   %__llgo_funcval_code57 = call ptr asm "", "=r,0"(ptr %435)
// CHECK-NEXT:   %436 = call %reflect.Value %__llgo_funcval_code57(ptr {{(nest|swiftself)}} %434, %"{{.*}}/runtime/internal/runtime.eface" %433)
// CHECK-NEXT:   store %reflect.Value %429, ptr %423, align 8
// CHECK-NEXT:   store %reflect.Value %436, ptr %430, align 8
// CHECK-NEXT:   %437 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 29
// CHECK-NEXT:   %438 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %437, i32 0, i32 0
// CHECK-NEXT:   %439 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %440 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 29, ptr %440, align 1
// CHECK-NEXT:   %441 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %440, 1
// CHECK-NEXT:   %442 = extractvalue { ptr, ptr } %439, 1
// CHECK-NEXT:   %443 = extractvalue { ptr, ptr } %439, 0
// CHECK-NEXT:   %__llgo_funcval_code58 = call ptr asm "", "=r,0"(ptr %443)
// CHECK-NEXT:   %444 = call %reflect.Value %__llgo_funcval_code58(ptr {{(nest|swiftself)}} %442, %"{{.*}}/runtime/internal/runtime.eface" %441)
// CHECK-NEXT:   %445 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %437, i32 0, i32 1
// CHECK-NEXT:   %446 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %447 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 29, ptr %447, align 2
// CHECK-NEXT:   %448 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %447, 1
// CHECK-NEXT:   %449 = extractvalue { ptr, ptr } %446, 1
// CHECK-NEXT:   %450 = extractvalue { ptr, ptr } %446, 0
// CHECK-NEXT:   %__llgo_funcval_code59 = call ptr asm "", "=r,0"(ptr %450)
// CHECK-NEXT:   %451 = call %reflect.Value %__llgo_funcval_code59(ptr {{(nest|swiftself)}} %449, %"{{.*}}/runtime/internal/runtime.eface" %448)
// CHECK-NEXT:   store %reflect.Value %444, ptr %438, align 8
// CHECK-NEXT:   store %reflect.Value %451, ptr %445, align 8
// CHECK-NEXT:   %452 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 30
// CHECK-NEXT:   %453 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %452, i32 0, i32 0
// CHECK-NEXT:   %454 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %455 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 30, ptr %455, align 2
// CHECK-NEXT:   %456 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %455, 1
// CHECK-NEXT:   %457 = extractvalue { ptr, ptr } %454, 1
// CHECK-NEXT:   %458 = extractvalue { ptr, ptr } %454, 0
// CHECK-NEXT:   %__llgo_funcval_code60 = call ptr asm "", "=r,0"(ptr %458)
// CHECK-NEXT:   %459 = call %reflect.Value %__llgo_funcval_code60(ptr {{(nest|swiftself)}} %457, %"{{.*}}/runtime/internal/runtime.eface" %456)
// CHECK-NEXT:   %460 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %452, i32 0, i32 1
// CHECK-NEXT:   %461 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %462 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 30, ptr %462, align 1
// CHECK-NEXT:   %463 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %462, 1
// CHECK-NEXT:   %464 = extractvalue { ptr, ptr } %461, 1
// CHECK-NEXT:   %465 = extractvalue { ptr, ptr } %461, 0
// CHECK-NEXT:   %__llgo_funcval_code61 = call ptr asm "", "=r,0"(ptr %465)
// CHECK-NEXT:   %466 = call %reflect.Value %__llgo_funcval_code61(ptr {{(nest|swiftself)}} %464, %"{{.*}}/runtime/internal/runtime.eface" %463)
// CHECK-NEXT:   store %reflect.Value %459, ptr %453, align 8
// CHECK-NEXT:   store %reflect.Value %466, ptr %460, align 8
// CHECK-NEXT:   %467 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 31
// CHECK-NEXT:   %468 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %467, i32 0, i32 0
// CHECK-NEXT:   %469 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %470 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 31, ptr %470, align 1
// CHECK-NEXT:   %471 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %470, 1
// CHECK-NEXT:   %472 = extractvalue { ptr, ptr } %469, 1
// CHECK-NEXT:   %473 = extractvalue { ptr, ptr } %469, 0
// CHECK-NEXT:   %__llgo_funcval_code62 = call ptr asm "", "=r,0"(ptr %473)
// CHECK-NEXT:   %474 = call %reflect.Value %__llgo_funcval_code62(ptr {{(nest|swiftself)}} %472, %"{{.*}}/runtime/internal/runtime.eface" %471)
// CHECK-NEXT:   %475 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %467, i32 0, i32 1
// CHECK-NEXT:   %476 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %477 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 31, ptr %477, align 4
// CHECK-NEXT:   %478 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %477, 1
// CHECK-NEXT:   %479 = extractvalue { ptr, ptr } %476, 1
// CHECK-NEXT:   %480 = extractvalue { ptr, ptr } %476, 0
// CHECK-NEXT:   %__llgo_funcval_code63 = call ptr asm "", "=r,0"(ptr %480)
// CHECK-NEXT:   %481 = call %reflect.Value %__llgo_funcval_code63(ptr {{(nest|swiftself)}} %479, %"{{.*}}/runtime/internal/runtime.eface" %478)
// CHECK-NEXT:   store %reflect.Value %474, ptr %468, align 8
// CHECK-NEXT:   store %reflect.Value %481, ptr %475, align 8
// CHECK-NEXT:   %482 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 32
// CHECK-NEXT:   %483 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %482, i32 0, i32 0
// CHECK-NEXT:   %484 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %485 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 32, ptr %485, align 4
// CHECK-NEXT:   %486 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %485, 1
// CHECK-NEXT:   %487 = extractvalue { ptr, ptr } %484, 1
// CHECK-NEXT:   %488 = extractvalue { ptr, ptr } %484, 0
// CHECK-NEXT:   %__llgo_funcval_code64 = call ptr asm "", "=r,0"(ptr %488)
// CHECK-NEXT:   %489 = call %reflect.Value %__llgo_funcval_code64(ptr {{(nest|swiftself)}} %487, %"{{.*}}/runtime/internal/runtime.eface" %486)
// CHECK-NEXT:   %490 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %482, i32 0, i32 1
// CHECK-NEXT:   %491 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %492 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 32, ptr %492, align 1
// CHECK-NEXT:   %493 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %492, 1
// CHECK-NEXT:   %494 = extractvalue { ptr, ptr } %491, 1
// CHECK-NEXT:   %495 = extractvalue { ptr, ptr } %491, 0
// CHECK-NEXT:   %__llgo_funcval_code65 = call ptr asm "", "=r,0"(ptr %495)
// CHECK-NEXT:   %496 = call %reflect.Value %__llgo_funcval_code65(ptr {{(nest|swiftself)}} %494, %"{{.*}}/runtime/internal/runtime.eface" %493)
// CHECK-NEXT:   store %reflect.Value %489, ptr %483, align 8
// CHECK-NEXT:   store %reflect.Value %496, ptr %490, align 8
// CHECK-NEXT:   %497 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 33
// CHECK-NEXT:   %498 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %497, i32 0, i32 0
// CHECK-NEXT:   %499 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %500 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 33, ptr %500, align 1
// CHECK-NEXT:   %501 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %500, 1
// CHECK-NEXT:   %502 = extractvalue { ptr, ptr } %499, 1
// CHECK-NEXT:   %503 = extractvalue { ptr, ptr } %499, 0
// CHECK-NEXT:   %__llgo_funcval_code66 = call ptr asm "", "=r,0"(ptr %503)
// CHECK-NEXT:   %504 = call %reflect.Value %__llgo_funcval_code66(ptr {{(nest|swiftself)}} %502, %"{{.*}}/runtime/internal/runtime.eface" %501)
// CHECK-NEXT:   %505 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %497, i32 0, i32 1
// CHECK-NEXT:   %506 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %507 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 33, ptr %507, align 4
// CHECK-NEXT:   %508 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %507, 1
// CHECK-NEXT:   %509 = extractvalue { ptr, ptr } %506, 1
// CHECK-NEXT:   %510 = extractvalue { ptr, ptr } %506, 0
// CHECK-NEXT:   %__llgo_funcval_code67 = call ptr asm "", "=r,0"(ptr %510)
// CHECK-NEXT:   %511 = call %reflect.Value %__llgo_funcval_code67(ptr {{(nest|swiftself)}} %509, %"{{.*}}/runtime/internal/runtime.eface" %508)
// CHECK-NEXT:   store %reflect.Value %504, ptr %498, align 8
// CHECK-NEXT:   store %reflect.Value %511, ptr %505, align 8
// CHECK-NEXT:   %512 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 34
// CHECK-NEXT:   %513 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %512, i32 0, i32 0
// CHECK-NEXT:   %514 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %515 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 34, ptr %515, align 4
// CHECK-NEXT:   %516 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %515, 1
// CHECK-NEXT:   %517 = extractvalue { ptr, ptr } %514, 1
// CHECK-NEXT:   %518 = extractvalue { ptr, ptr } %514, 0
// CHECK-NEXT:   %__llgo_funcval_code68 = call ptr asm "", "=r,0"(ptr %518)
// CHECK-NEXT:   %519 = call %reflect.Value %__llgo_funcval_code68(ptr {{(nest|swiftself)}} %517, %"{{.*}}/runtime/internal/runtime.eface" %516)
// CHECK-NEXT:   %520 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %512, i32 0, i32 1
// CHECK-NEXT:   %521 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %522 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 34, ptr %522, align 1
// CHECK-NEXT:   %523 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %522, 1
// CHECK-NEXT:   %524 = extractvalue { ptr, ptr } %521, 1
// CHECK-NEXT:   %525 = extractvalue { ptr, ptr } %521, 0
// CHECK-NEXT:   %__llgo_funcval_code69 = call ptr asm "", "=r,0"(ptr %525)
// CHECK-NEXT:   %526 = call %reflect.Value %__llgo_funcval_code69(ptr {{(nest|swiftself)}} %524, %"{{.*}}/runtime/internal/runtime.eface" %523)
// CHECK-NEXT:   store %reflect.Value %519, ptr %513, align 8
// CHECK-NEXT:   store %reflect.Value %526, ptr %520, align 8
// CHECK-NEXT:   %527 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 35
// CHECK-NEXT:   %528 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %527, i32 0, i32 0
// CHECK-NEXT:   %529 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %530 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 35, ptr %530, align 1
// CHECK-NEXT:   %531 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %530, 1
// CHECK-NEXT:   %532 = extractvalue { ptr, ptr } %529, 1
// CHECK-NEXT:   %533 = extractvalue { ptr, ptr } %529, 0
// CHECK-NEXT:   %__llgo_funcval_code70 = call ptr asm "", "=r,0"(ptr %533)
// CHECK-NEXT:   %534 = call %reflect.Value %__llgo_funcval_code70(ptr {{(nest|swiftself)}} %532, %"{{.*}}/runtime/internal/runtime.eface" %531)
// CHECK-NEXT:   %535 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %527, i32 0, i32 1
// CHECK-NEXT:   %536 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %537 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 35, ptr %537, align 8
// CHECK-NEXT:   %538 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %537, 1
// CHECK-NEXT:   %539 = extractvalue { ptr, ptr } %536, 1
// CHECK-NEXT:   %540 = extractvalue { ptr, ptr } %536, 0
// CHECK-NEXT:   %__llgo_funcval_code71 = call ptr asm "", "=r,0"(ptr %540)
// CHECK-NEXT:   %541 = call %reflect.Value %__llgo_funcval_code71(ptr {{(nest|swiftself)}} %539, %"{{.*}}/runtime/internal/runtime.eface" %538)
// CHECK-NEXT:   store %reflect.Value %534, ptr %528, align 8
// CHECK-NEXT:   store %reflect.Value %541, ptr %535, align 8
// CHECK-NEXT:   %542 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 36
// CHECK-NEXT:   %543 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %542, i32 0, i32 0
// CHECK-NEXT:   %544 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %545 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 36, ptr %545, align 8
// CHECK-NEXT:   %546 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %545, 1
// CHECK-NEXT:   %547 = extractvalue { ptr, ptr } %544, 1
// CHECK-NEXT:   %548 = extractvalue { ptr, ptr } %544, 0
// CHECK-NEXT:   %__llgo_funcval_code72 = call ptr asm "", "=r,0"(ptr %548)
// CHECK-NEXT:   %549 = call %reflect.Value %__llgo_funcval_code72(ptr {{(nest|swiftself)}} %547, %"{{.*}}/runtime/internal/runtime.eface" %546)
// CHECK-NEXT:   %550 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %542, i32 0, i32 1
// CHECK-NEXT:   %551 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %552 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 36, ptr %552, align 1
// CHECK-NEXT:   %553 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %552, 1
// CHECK-NEXT:   %554 = extractvalue { ptr, ptr } %551, 1
// CHECK-NEXT:   %555 = extractvalue { ptr, ptr } %551, 0
// CHECK-NEXT:   %__llgo_funcval_code73 = call ptr asm "", "=r,0"(ptr %555)
// CHECK-NEXT:   %556 = call %reflect.Value %__llgo_funcval_code73(ptr {{(nest|swiftself)}} %554, %"{{.*}}/runtime/internal/runtime.eface" %553)
// CHECK-NEXT:   store %reflect.Value %549, ptr %543, align 8
// CHECK-NEXT:   store %reflect.Value %556, ptr %550, align 8
// CHECK-NEXT:   %557 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 37
// CHECK-NEXT:   %558 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %557, i32 0, i32 0
// CHECK-NEXT:   %559 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %560 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 37, ptr %560, align 1
// CHECK-NEXT:   %561 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %560, 1
// CHECK-NEXT:   %562 = extractvalue { ptr, ptr } %559, 1
// CHECK-NEXT:   %563 = extractvalue { ptr, ptr } %559, 0
// CHECK-NEXT:   %__llgo_funcval_code74 = call ptr asm "", "=r,0"(ptr %563)
// CHECK-NEXT:   %564 = call %reflect.Value %__llgo_funcval_code74(ptr {{(nest|swiftself)}} %562, %"{{.*}}/runtime/internal/runtime.eface" %561)
// CHECK-NEXT:   %565 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %557, i32 0, i32 1
// CHECK-NEXT:   %566 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %567 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 37, ptr %567, align 8
// CHECK-NEXT:   %568 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %567, 1
// CHECK-NEXT:   %569 = extractvalue { ptr, ptr } %566, 1
// CHECK-NEXT:   %570 = extractvalue { ptr, ptr } %566, 0
// CHECK-NEXT:   %__llgo_funcval_code75 = call ptr asm "", "=r,0"(ptr %570)
// CHECK-NEXT:   %571 = call %reflect.Value %__llgo_funcval_code75(ptr {{(nest|swiftself)}} %569, %"{{.*}}/runtime/internal/runtime.eface" %568)
// CHECK-NEXT:   store %reflect.Value %564, ptr %558, align 8
// CHECK-NEXT:   store %reflect.Value %571, ptr %565, align 8
// CHECK-NEXT:   %572 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 38
// CHECK-NEXT:   %573 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %572, i32 0, i32 0
// CHECK-NEXT:   %574 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %575 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 38, ptr %575, align 8
// CHECK-NEXT:   %576 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %575, 1
// CHECK-NEXT:   %577 = extractvalue { ptr, ptr } %574, 1
// CHECK-NEXT:   %578 = extractvalue { ptr, ptr } %574, 0
// CHECK-NEXT:   %__llgo_funcval_code76 = call ptr asm "", "=r,0"(ptr %578)
// CHECK-NEXT:   %579 = call %reflect.Value %__llgo_funcval_code76(ptr {{(nest|swiftself)}} %577, %"{{.*}}/runtime/internal/runtime.eface" %576)
// CHECK-NEXT:   %580 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %572, i32 0, i32 1
// CHECK-NEXT:   %581 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %582 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 38, ptr %582, align 1
// CHECK-NEXT:   %583 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %582, 1
// CHECK-NEXT:   %584 = extractvalue { ptr, ptr } %581, 1
// CHECK-NEXT:   %585 = extractvalue { ptr, ptr } %581, 0
// CHECK-NEXT:   %__llgo_funcval_code77 = call ptr asm "", "=r,0"(ptr %585)
// CHECK-NEXT:   %586 = call %reflect.Value %__llgo_funcval_code77(ptr {{(nest|swiftself)}} %584, %"{{.*}}/runtime/internal/runtime.eface" %583)
// CHECK-NEXT:   store %reflect.Value %579, ptr %573, align 8
// CHECK-NEXT:   store %reflect.Value %586, ptr %580, align 8
// CHECK-NEXT:   %587 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 39
// CHECK-NEXT:   %588 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %587, i32 0, i32 0
// CHECK-NEXT:   %589 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %590 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 39, ptr %590, align 1
// CHECK-NEXT:   %591 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %590, 1
// CHECK-NEXT:   %592 = extractvalue { ptr, ptr } %589, 1
// CHECK-NEXT:   %593 = extractvalue { ptr, ptr } %589, 0
// CHECK-NEXT:   %__llgo_funcval_code78 = call ptr asm "", "=r,0"(ptr %593)
// CHECK-NEXT:   %594 = call %reflect.Value %__llgo_funcval_code78(ptr {{(nest|swiftself)}} %592, %"{{.*}}/runtime/internal/runtime.eface" %591)
// CHECK-NEXT:   %595 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %587, i32 0, i32 1
// CHECK-NEXT:   %596 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %597 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 39, ptr %597, align 8
// CHECK-NEXT:   %598 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %597, 1
// CHECK-NEXT:   %599 = extractvalue { ptr, ptr } %596, 1
// CHECK-NEXT:   %600 = extractvalue { ptr, ptr } %596, 0
// CHECK-NEXT:   %__llgo_funcval_code79 = call ptr asm "", "=r,0"(ptr %600)
// CHECK-NEXT:   %601 = call %reflect.Value %__llgo_funcval_code79(ptr {{(nest|swiftself)}} %599, %"{{.*}}/runtime/internal/runtime.eface" %598)
// CHECK-NEXT:   store %reflect.Value %594, ptr %588, align 8
// CHECK-NEXT:   store %reflect.Value %601, ptr %595, align 8
// CHECK-NEXT:   %602 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 40
// CHECK-NEXT:   %603 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %602, i32 0, i32 0
// CHECK-NEXT:   %604 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %605 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 40, ptr %605, align 8
// CHECK-NEXT:   %606 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %605, 1
// CHECK-NEXT:   %607 = extractvalue { ptr, ptr } %604, 1
// CHECK-NEXT:   %608 = extractvalue { ptr, ptr } %604, 0
// CHECK-NEXT:   %__llgo_funcval_code80 = call ptr asm "", "=r,0"(ptr %608)
// CHECK-NEXT:   %609 = call %reflect.Value %__llgo_funcval_code80(ptr {{(nest|swiftself)}} %607, %"{{.*}}/runtime/internal/runtime.eface" %606)
// CHECK-NEXT:   %610 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %602, i32 0, i32 1
// CHECK-NEXT:   %611 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %612 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 40, ptr %612, align 1
// CHECK-NEXT:   %613 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %612, 1
// CHECK-NEXT:   %614 = extractvalue { ptr, ptr } %611, 1
// CHECK-NEXT:   %615 = extractvalue { ptr, ptr } %611, 0
// CHECK-NEXT:   %__llgo_funcval_code81 = call ptr asm "", "=r,0"(ptr %615)
// CHECK-NEXT:   %616 = call %reflect.Value %__llgo_funcval_code81(ptr {{(nest|swiftself)}} %614, %"{{.*}}/runtime/internal/runtime.eface" %613)
// CHECK-NEXT:   store %reflect.Value %609, ptr %603, align 8
// CHECK-NEXT:   store %reflect.Value %616, ptr %610, align 8
// CHECK-NEXT:   %617 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 41
// CHECK-NEXT:   %618 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %617, i32 0, i32 0
// CHECK-NEXT:   %619 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %620 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 41, ptr %620, align 1
// CHECK-NEXT:   %621 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %620, 1
// CHECK-NEXT:   %622 = extractvalue { ptr, ptr } %619, 1
// CHECK-NEXT:   %623 = extractvalue { ptr, ptr } %619, 0
// CHECK-NEXT:   %__llgo_funcval_code82 = call ptr asm "", "=r,0"(ptr %623)
// CHECK-NEXT:   %624 = call %reflect.Value %__llgo_funcval_code82(ptr {{(nest|swiftself)}} %622, %"{{.*}}/runtime/internal/runtime.eface" %621)
// CHECK-NEXT:   %625 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %617, i32 0, i32 1
// CHECK-NEXT:   %626 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %627 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 41, ptr %627, align 8
// CHECK-NEXT:   %628 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %627, 1
// CHECK-NEXT:   %629 = extractvalue { ptr, ptr } %626, 1
// CHECK-NEXT:   %630 = extractvalue { ptr, ptr } %626, 0
// CHECK-NEXT:   %__llgo_funcval_code83 = call ptr asm "", "=r,0"(ptr %630)
// CHECK-NEXT:   %631 = call %reflect.Value %__llgo_funcval_code83(ptr {{(nest|swiftself)}} %629, %"{{.*}}/runtime/internal/runtime.eface" %628)
// CHECK-NEXT:   store %reflect.Value %624, ptr %618, align 8
// CHECK-NEXT:   store %reflect.Value %631, ptr %625, align 8
// CHECK-NEXT:   %632 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 42
// CHECK-NEXT:   %633 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %632, i32 0, i32 0
// CHECK-NEXT:   %634 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %635 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %635, align 8
// CHECK-NEXT:   %636 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %635, 1
// CHECK-NEXT:   %637 = extractvalue { ptr, ptr } %634, 1
// CHECK-NEXT:   %638 = extractvalue { ptr, ptr } %634, 0
// CHECK-NEXT:   %__llgo_funcval_code84 = call ptr asm "", "=r,0"(ptr %638)
// CHECK-NEXT:   %639 = call %reflect.Value %__llgo_funcval_code84(ptr {{(nest|swiftself)}} %637, %"{{.*}}/runtime/internal/runtime.eface" %636)
// CHECK-NEXT:   %640 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %632, i32 0, i32 1
// CHECK-NEXT:   %641 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %642 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 42, ptr %642, align 1
// CHECK-NEXT:   %643 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %642, 1
// CHECK-NEXT:   %644 = extractvalue { ptr, ptr } %641, 1
// CHECK-NEXT:   %645 = extractvalue { ptr, ptr } %641, 0
// CHECK-NEXT:   %__llgo_funcval_code85 = call ptr asm "", "=r,0"(ptr %645)
// CHECK-NEXT:   %646 = call %reflect.Value %__llgo_funcval_code85(ptr {{(nest|swiftself)}} %644, %"{{.*}}/runtime/internal/runtime.eface" %643)
// CHECK-NEXT:   store %reflect.Value %639, ptr %633, align 8
// CHECK-NEXT:   store %reflect.Value %646, ptr %640, align 8
// CHECK-NEXT:   %647 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 43
// CHECK-NEXT:   %648 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %647, i32 0, i32 0
// CHECK-NEXT:   %649 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %650 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 43, ptr %650, align 1
// CHECK-NEXT:   %651 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %650, 1
// CHECK-NEXT:   %652 = extractvalue { ptr, ptr } %649, 1
// CHECK-NEXT:   %653 = extractvalue { ptr, ptr } %649, 0
// CHECK-NEXT:   %__llgo_funcval_code86 = call ptr asm "", "=r,0"(ptr %653)
// CHECK-NEXT:   %654 = call %reflect.Value %__llgo_funcval_code86(ptr {{(nest|swiftself)}} %652, %"{{.*}}/runtime/internal/runtime.eface" %651)
// CHECK-NEXT:   %655 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %647, i32 0, i32 1
// CHECK-NEXT:   %656 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %657 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 43, ptr %657, align 8
// CHECK-NEXT:   %658 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %657, 1
// CHECK-NEXT:   %659 = extractvalue { ptr, ptr } %656, 1
// CHECK-NEXT:   %660 = extractvalue { ptr, ptr } %656, 0
// CHECK-NEXT:   %__llgo_funcval_code87 = call ptr asm "", "=r,0"(ptr %660)
// CHECK-NEXT:   %661 = call %reflect.Value %__llgo_funcval_code87(ptr {{(nest|swiftself)}} %659, %"{{.*}}/runtime/internal/runtime.eface" %658)
// CHECK-NEXT:   store %reflect.Value %654, ptr %648, align 8
// CHECK-NEXT:   store %reflect.Value %661, ptr %655, align 8
// CHECK-NEXT:   %662 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 44
// CHECK-NEXT:   %663 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %662, i32 0, i32 0
// CHECK-NEXT:   %664 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %665 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 44, ptr %665, align 8
// CHECK-NEXT:   %666 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %665, 1
// CHECK-NEXT:   %667 = extractvalue { ptr, ptr } %664, 1
// CHECK-NEXT:   %668 = extractvalue { ptr, ptr } %664, 0
// CHECK-NEXT:   %__llgo_funcval_code88 = call ptr asm "", "=r,0"(ptr %668)
// CHECK-NEXT:   %669 = call %reflect.Value %__llgo_funcval_code88(ptr {{(nest|swiftself)}} %667, %"{{.*}}/runtime/internal/runtime.eface" %666)
// CHECK-NEXT:   %670 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %662, i32 0, i32 1
// CHECK-NEXT:   %671 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %672 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 44, ptr %672, align 1
// CHECK-NEXT:   %673 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %672, 1
// CHECK-NEXT:   %674 = extractvalue { ptr, ptr } %671, 1
// CHECK-NEXT:   %675 = extractvalue { ptr, ptr } %671, 0
// CHECK-NEXT:   %__llgo_funcval_code89 = call ptr asm "", "=r,0"(ptr %675)
// CHECK-NEXT:   %676 = call %reflect.Value %__llgo_funcval_code89(ptr {{(nest|swiftself)}} %674, %"{{.*}}/runtime/internal/runtime.eface" %673)
// CHECK-NEXT:   store %reflect.Value %669, ptr %663, align 8
// CHECK-NEXT:   store %reflect.Value %676, ptr %670, align 8
// CHECK-NEXT:   %677 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 45
// CHECK-NEXT:   %678 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %677, i32 0, i32 0
// CHECK-NEXT:   %679 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %680 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 45, ptr %680, align 1
// CHECK-NEXT:   %681 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %680, 1
// CHECK-NEXT:   %682 = extractvalue { ptr, ptr } %679, 1
// CHECK-NEXT:   %683 = extractvalue { ptr, ptr } %679, 0
// CHECK-NEXT:   %__llgo_funcval_code90 = call ptr asm "", "=r,0"(ptr %683)
// CHECK-NEXT:   %684 = call %reflect.Value %__llgo_funcval_code90(ptr {{(nest|swiftself)}} %682, %"{{.*}}/runtime/internal/runtime.eface" %681)
// CHECK-NEXT:   %685 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %677, i32 0, i32 1
// CHECK-NEXT:   %686 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %687 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 4.500000e+01, ptr %687, align 4
// CHECK-NEXT:   %688 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %687, 1
// CHECK-NEXT:   %689 = extractvalue { ptr, ptr } %686, 1
// CHECK-NEXT:   %690 = extractvalue { ptr, ptr } %686, 0
// CHECK-NEXT:   %__llgo_funcval_code91 = call ptr asm "", "=r,0"(ptr %690)
// CHECK-NEXT:   %691 = call %reflect.Value %__llgo_funcval_code91(ptr {{(nest|swiftself)}} %689, %"{{.*}}/runtime/internal/runtime.eface" %688)
// CHECK-NEXT:   store %reflect.Value %684, ptr %678, align 8
// CHECK-NEXT:   store %reflect.Value %691, ptr %685, align 8
// CHECK-NEXT:   %692 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 46
// CHECK-NEXT:   %693 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %692, i32 0, i32 0
// CHECK-NEXT:   %694 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %695 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 4.600000e+01, ptr %695, align 4
// CHECK-NEXT:   %696 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %695, 1
// CHECK-NEXT:   %697 = extractvalue { ptr, ptr } %694, 1
// CHECK-NEXT:   %698 = extractvalue { ptr, ptr } %694, 0
// CHECK-NEXT:   %__llgo_funcval_code92 = call ptr asm "", "=r,0"(ptr %698)
// CHECK-NEXT:   %699 = call %reflect.Value %__llgo_funcval_code92(ptr {{(nest|swiftself)}} %697, %"{{.*}}/runtime/internal/runtime.eface" %696)
// CHECK-NEXT:   %700 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %692, i32 0, i32 1
// CHECK-NEXT:   %701 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %702 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 46, ptr %702, align 1
// CHECK-NEXT:   %703 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %702, 1
// CHECK-NEXT:   %704 = extractvalue { ptr, ptr } %701, 1
// CHECK-NEXT:   %705 = extractvalue { ptr, ptr } %701, 0
// CHECK-NEXT:   %__llgo_funcval_code93 = call ptr asm "", "=r,0"(ptr %705)
// CHECK-NEXT:   %706 = call %reflect.Value %__llgo_funcval_code93(ptr {{(nest|swiftself)}} %704, %"{{.*}}/runtime/internal/runtime.eface" %703)
// CHECK-NEXT:   store %reflect.Value %699, ptr %693, align 8
// CHECK-NEXT:   store %reflect.Value %706, ptr %700, align 8
// CHECK-NEXT:   %707 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 47
// CHECK-NEXT:   %708 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %707, i32 0, i32 0
// CHECK-NEXT:   %709 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %710 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 47, ptr %710, align 1
// CHECK-NEXT:   %711 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %710, 1
// CHECK-NEXT:   %712 = extractvalue { ptr, ptr } %709, 1
// CHECK-NEXT:   %713 = extractvalue { ptr, ptr } %709, 0
// CHECK-NEXT:   %__llgo_funcval_code94 = call ptr asm "", "=r,0"(ptr %713)
// CHECK-NEXT:   %714 = call %reflect.Value %__llgo_funcval_code94(ptr {{(nest|swiftself)}} %712, %"{{.*}}/runtime/internal/runtime.eface" %711)
// CHECK-NEXT:   %715 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %707, i32 0, i32 1
// CHECK-NEXT:   %716 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %717 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 4.700000e+01, ptr %717, align 8
// CHECK-NEXT:   %718 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %717, 1
// CHECK-NEXT:   %719 = extractvalue { ptr, ptr } %716, 1
// CHECK-NEXT:   %720 = extractvalue { ptr, ptr } %716, 0
// CHECK-NEXT:   %__llgo_funcval_code95 = call ptr asm "", "=r,0"(ptr %720)
// CHECK-NEXT:   %721 = call %reflect.Value %__llgo_funcval_code95(ptr {{(nest|swiftself)}} %719, %"{{.*}}/runtime/internal/runtime.eface" %718)
// CHECK-NEXT:   store %reflect.Value %714, ptr %708, align 8
// CHECK-NEXT:   store %reflect.Value %721, ptr %715, align 8
// CHECK-NEXT:   %722 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 48
// CHECK-NEXT:   %723 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %722, i32 0, i32 0
// CHECK-NEXT:   %724 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %725 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 4.800000e+01, ptr %725, align 8
// CHECK-NEXT:   %726 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %725, 1
// CHECK-NEXT:   %727 = extractvalue { ptr, ptr } %724, 1
// CHECK-NEXT:   %728 = extractvalue { ptr, ptr } %724, 0
// CHECK-NEXT:   %__llgo_funcval_code96 = call ptr asm "", "=r,0"(ptr %728)
// CHECK-NEXT:   %729 = call %reflect.Value %__llgo_funcval_code96(ptr {{(nest|swiftself)}} %727, %"{{.*}}/runtime/internal/runtime.eface" %726)
// CHECK-NEXT:   %730 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %722, i32 0, i32 1
// CHECK-NEXT:   %731 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %732 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 48, ptr %732, align 1
// CHECK-NEXT:   %733 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %732, 1
// CHECK-NEXT:   %734 = extractvalue { ptr, ptr } %731, 1
// CHECK-NEXT:   %735 = extractvalue { ptr, ptr } %731, 0
// CHECK-NEXT:   %__llgo_funcval_code97 = call ptr asm "", "=r,0"(ptr %735)
// CHECK-NEXT:   %736 = call %reflect.Value %__llgo_funcval_code97(ptr {{(nest|swiftself)}} %734, %"{{.*}}/runtime/internal/runtime.eface" %733)
// CHECK-NEXT:   store %reflect.Value %729, ptr %723, align 8
// CHECK-NEXT:   store %reflect.Value %736, ptr %730, align 8
// CHECK-NEXT:   %737 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 49
// CHECK-NEXT:   %738 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %737, i32 0, i32 0
// CHECK-NEXT:   %739 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %740 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 49, ptr %740, align 2
// CHECK-NEXT:   %741 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %740, 1
// CHECK-NEXT:   %742 = extractvalue { ptr, ptr } %739, 1
// CHECK-NEXT:   %743 = extractvalue { ptr, ptr } %739, 0
// CHECK-NEXT:   %__llgo_funcval_code98 = call ptr asm "", "=r,0"(ptr %743)
// CHECK-NEXT:   %744 = call %reflect.Value %__llgo_funcval_code98(ptr {{(nest|swiftself)}} %742, %"{{.*}}/runtime/internal/runtime.eface" %741)
// CHECK-NEXT:   %745 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %737, i32 0, i32 1
// CHECK-NEXT:   %746 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %747 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 49, ptr %747, align 2
// CHECK-NEXT:   %748 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %747, 1
// CHECK-NEXT:   %749 = extractvalue { ptr, ptr } %746, 1
// CHECK-NEXT:   %750 = extractvalue { ptr, ptr } %746, 0
// CHECK-NEXT:   %__llgo_funcval_code99 = call ptr asm "", "=r,0"(ptr %750)
// CHECK-NEXT:   %751 = call %reflect.Value %__llgo_funcval_code99(ptr {{(nest|swiftself)}} %749, %"{{.*}}/runtime/internal/runtime.eface" %748)
// CHECK-NEXT:   store %reflect.Value %744, ptr %738, align 8
// CHECK-NEXT:   store %reflect.Value %751, ptr %745, align 8
// CHECK-NEXT:   %752 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 50
// CHECK-NEXT:   %753 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %752, i32 0, i32 0
// CHECK-NEXT:   %754 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %755 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 50, ptr %755, align 2
// CHECK-NEXT:   %756 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %755, 1
// CHECK-NEXT:   %757 = extractvalue { ptr, ptr } %754, 1
// CHECK-NEXT:   %758 = extractvalue { ptr, ptr } %754, 0
// CHECK-NEXT:   %__llgo_funcval_code100 = call ptr asm "", "=r,0"(ptr %758)
// CHECK-NEXT:   %759 = call %reflect.Value %__llgo_funcval_code100(ptr {{(nest|swiftself)}} %757, %"{{.*}}/runtime/internal/runtime.eface" %756)
// CHECK-NEXT:   %760 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %752, i32 0, i32 1
// CHECK-NEXT:   %761 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %762 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 50, ptr %762, align 2
// CHECK-NEXT:   %763 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %762, 1
// CHECK-NEXT:   %764 = extractvalue { ptr, ptr } %761, 1
// CHECK-NEXT:   %765 = extractvalue { ptr, ptr } %761, 0
// CHECK-NEXT:   %__llgo_funcval_code101 = call ptr asm "", "=r,0"(ptr %765)
// CHECK-NEXT:   %766 = call %reflect.Value %__llgo_funcval_code101(ptr {{(nest|swiftself)}} %764, %"{{.*}}/runtime/internal/runtime.eface" %763)
// CHECK-NEXT:   store %reflect.Value %759, ptr %753, align 8
// CHECK-NEXT:   store %reflect.Value %766, ptr %760, align 8
// CHECK-NEXT:   %767 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 51
// CHECK-NEXT:   %768 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %767, i32 0, i32 0
// CHECK-NEXT:   %769 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %770 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 51, ptr %770, align 2
// CHECK-NEXT:   %771 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %770, 1
// CHECK-NEXT:   %772 = extractvalue { ptr, ptr } %769, 1
// CHECK-NEXT:   %773 = extractvalue { ptr, ptr } %769, 0
// CHECK-NEXT:   %__llgo_funcval_code102 = call ptr asm "", "=r,0"(ptr %773)
// CHECK-NEXT:   %774 = call %reflect.Value %__llgo_funcval_code102(ptr {{(nest|swiftself)}} %772, %"{{.*}}/runtime/internal/runtime.eface" %771)
// CHECK-NEXT:   %775 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %767, i32 0, i32 1
// CHECK-NEXT:   %776 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %777 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 51, ptr %777, align 2
// CHECK-NEXT:   %778 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %777, 1
// CHECK-NEXT:   %779 = extractvalue { ptr, ptr } %776, 1
// CHECK-NEXT:   %780 = extractvalue { ptr, ptr } %776, 0
// CHECK-NEXT:   %__llgo_funcval_code103 = call ptr asm "", "=r,0"(ptr %780)
// CHECK-NEXT:   %781 = call %reflect.Value %__llgo_funcval_code103(ptr {{(nest|swiftself)}} %779, %"{{.*}}/runtime/internal/runtime.eface" %778)
// CHECK-NEXT:   store %reflect.Value %774, ptr %768, align 8
// CHECK-NEXT:   store %reflect.Value %781, ptr %775, align 8
// CHECK-NEXT:   %782 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 52
// CHECK-NEXT:   %783 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %782, i32 0, i32 0
// CHECK-NEXT:   %784 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %785 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 52, ptr %785, align 2
// CHECK-NEXT:   %786 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %785, 1
// CHECK-NEXT:   %787 = extractvalue { ptr, ptr } %784, 1
// CHECK-NEXT:   %788 = extractvalue { ptr, ptr } %784, 0
// CHECK-NEXT:   %__llgo_funcval_code104 = call ptr asm "", "=r,0"(ptr %788)
// CHECK-NEXT:   %789 = call %reflect.Value %__llgo_funcval_code104(ptr {{(nest|swiftself)}} %787, %"{{.*}}/runtime/internal/runtime.eface" %786)
// CHECK-NEXT:   %790 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %782, i32 0, i32 1
// CHECK-NEXT:   %791 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %792 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 52, ptr %792, align 4
// CHECK-NEXT:   %793 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %792, 1
// CHECK-NEXT:   %794 = extractvalue { ptr, ptr } %791, 1
// CHECK-NEXT:   %795 = extractvalue { ptr, ptr } %791, 0
// CHECK-NEXT:   %__llgo_funcval_code105 = call ptr asm "", "=r,0"(ptr %795)
// CHECK-NEXT:   %796 = call %reflect.Value %__llgo_funcval_code105(ptr {{(nest|swiftself)}} %794, %"{{.*}}/runtime/internal/runtime.eface" %793)
// CHECK-NEXT:   store %reflect.Value %789, ptr %783, align 8
// CHECK-NEXT:   store %reflect.Value %796, ptr %790, align 8
// CHECK-NEXT:   %797 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 53
// CHECK-NEXT:   %798 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %797, i32 0, i32 0
// CHECK-NEXT:   %799 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %800 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 53, ptr %800, align 4
// CHECK-NEXT:   %801 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %800, 1
// CHECK-NEXT:   %802 = extractvalue { ptr, ptr } %799, 1
// CHECK-NEXT:   %803 = extractvalue { ptr, ptr } %799, 0
// CHECK-NEXT:   %__llgo_funcval_code106 = call ptr asm "", "=r,0"(ptr %803)
// CHECK-NEXT:   %804 = call %reflect.Value %__llgo_funcval_code106(ptr {{(nest|swiftself)}} %802, %"{{.*}}/runtime/internal/runtime.eface" %801)
// CHECK-NEXT:   %805 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %797, i32 0, i32 1
// CHECK-NEXT:   %806 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %807 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 53, ptr %807, align 2
// CHECK-NEXT:   %808 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %807, 1
// CHECK-NEXT:   %809 = extractvalue { ptr, ptr } %806, 1
// CHECK-NEXT:   %810 = extractvalue { ptr, ptr } %806, 0
// CHECK-NEXT:   %__llgo_funcval_code107 = call ptr asm "", "=r,0"(ptr %810)
// CHECK-NEXT:   %811 = call %reflect.Value %__llgo_funcval_code107(ptr {{(nest|swiftself)}} %809, %"{{.*}}/runtime/internal/runtime.eface" %808)
// CHECK-NEXT:   store %reflect.Value %804, ptr %798, align 8
// CHECK-NEXT:   store %reflect.Value %811, ptr %805, align 8
// CHECK-NEXT:   %812 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 54
// CHECK-NEXT:   %813 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %812, i32 0, i32 0
// CHECK-NEXT:   %814 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %815 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 54, ptr %815, align 2
// CHECK-NEXT:   %816 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %815, 1
// CHECK-NEXT:   %817 = extractvalue { ptr, ptr } %814, 1
// CHECK-NEXT:   %818 = extractvalue { ptr, ptr } %814, 0
// CHECK-NEXT:   %__llgo_funcval_code108 = call ptr asm "", "=r,0"(ptr %818)
// CHECK-NEXT:   %819 = call %reflect.Value %__llgo_funcval_code108(ptr {{(nest|swiftself)}} %817, %"{{.*}}/runtime/internal/runtime.eface" %816)
// CHECK-NEXT:   %820 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %812, i32 0, i32 1
// CHECK-NEXT:   %821 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %822 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 54, ptr %822, align 4
// CHECK-NEXT:   %823 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %822, 1
// CHECK-NEXT:   %824 = extractvalue { ptr, ptr } %821, 1
// CHECK-NEXT:   %825 = extractvalue { ptr, ptr } %821, 0
// CHECK-NEXT:   %__llgo_funcval_code109 = call ptr asm "", "=r,0"(ptr %825)
// CHECK-NEXT:   %826 = call %reflect.Value %__llgo_funcval_code109(ptr {{(nest|swiftself)}} %824, %"{{.*}}/runtime/internal/runtime.eface" %823)
// CHECK-NEXT:   store %reflect.Value %819, ptr %813, align 8
// CHECK-NEXT:   store %reflect.Value %826, ptr %820, align 8
// CHECK-NEXT:   %827 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 55
// CHECK-NEXT:   %828 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %827, i32 0, i32 0
// CHECK-NEXT:   %829 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %830 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 55, ptr %830, align 4
// CHECK-NEXT:   %831 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %830, 1
// CHECK-NEXT:   %832 = extractvalue { ptr, ptr } %829, 1
// CHECK-NEXT:   %833 = extractvalue { ptr, ptr } %829, 0
// CHECK-NEXT:   %__llgo_funcval_code110 = call ptr asm "", "=r,0"(ptr %833)
// CHECK-NEXT:   %834 = call %reflect.Value %__llgo_funcval_code110(ptr {{(nest|swiftself)}} %832, %"{{.*}}/runtime/internal/runtime.eface" %831)
// CHECK-NEXT:   %835 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %827, i32 0, i32 1
// CHECK-NEXT:   %836 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %837 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 55, ptr %837, align 2
// CHECK-NEXT:   %838 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %837, 1
// CHECK-NEXT:   %839 = extractvalue { ptr, ptr } %836, 1
// CHECK-NEXT:   %840 = extractvalue { ptr, ptr } %836, 0
// CHECK-NEXT:   %__llgo_funcval_code111 = call ptr asm "", "=r,0"(ptr %840)
// CHECK-NEXT:   %841 = call %reflect.Value %__llgo_funcval_code111(ptr {{(nest|swiftself)}} %839, %"{{.*}}/runtime/internal/runtime.eface" %838)
// CHECK-NEXT:   store %reflect.Value %834, ptr %828, align 8
// CHECK-NEXT:   store %reflect.Value %841, ptr %835, align 8
// CHECK-NEXT:   %842 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 56
// CHECK-NEXT:   %843 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %842, i32 0, i32 0
// CHECK-NEXT:   %844 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %845 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 56, ptr %845, align 2
// CHECK-NEXT:   %846 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %845, 1
// CHECK-NEXT:   %847 = extractvalue { ptr, ptr } %844, 1
// CHECK-NEXT:   %848 = extractvalue { ptr, ptr } %844, 0
// CHECK-NEXT:   %__llgo_funcval_code112 = call ptr asm "", "=r,0"(ptr %848)
// CHECK-NEXT:   %849 = call %reflect.Value %__llgo_funcval_code112(ptr {{(nest|swiftself)}} %847, %"{{.*}}/runtime/internal/runtime.eface" %846)
// CHECK-NEXT:   %850 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %842, i32 0, i32 1
// CHECK-NEXT:   %851 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %852 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 56, ptr %852, align 8
// CHECK-NEXT:   %853 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %852, 1
// CHECK-NEXT:   %854 = extractvalue { ptr, ptr } %851, 1
// CHECK-NEXT:   %855 = extractvalue { ptr, ptr } %851, 0
// CHECK-NEXT:   %__llgo_funcval_code113 = call ptr asm "", "=r,0"(ptr %855)
// CHECK-NEXT:   %856 = call %reflect.Value %__llgo_funcval_code113(ptr {{(nest|swiftself)}} %854, %"{{.*}}/runtime/internal/runtime.eface" %853)
// CHECK-NEXT:   store %reflect.Value %849, ptr %843, align 8
// CHECK-NEXT:   store %reflect.Value %856, ptr %850, align 8
// CHECK-NEXT:   %857 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 57
// CHECK-NEXT:   %858 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %857, i32 0, i32 0
// CHECK-NEXT:   %859 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %860 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 57, ptr %860, align 8
// CHECK-NEXT:   %861 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %860, 1
// CHECK-NEXT:   %862 = extractvalue { ptr, ptr } %859, 1
// CHECK-NEXT:   %863 = extractvalue { ptr, ptr } %859, 0
// CHECK-NEXT:   %__llgo_funcval_code114 = call ptr asm "", "=r,0"(ptr %863)
// CHECK-NEXT:   %864 = call %reflect.Value %__llgo_funcval_code114(ptr {{(nest|swiftself)}} %862, %"{{.*}}/runtime/internal/runtime.eface" %861)
// CHECK-NEXT:   %865 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %857, i32 0, i32 1
// CHECK-NEXT:   %866 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %867 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 57, ptr %867, align 2
// CHECK-NEXT:   %868 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %867, 1
// CHECK-NEXT:   %869 = extractvalue { ptr, ptr } %866, 1
// CHECK-NEXT:   %870 = extractvalue { ptr, ptr } %866, 0
// CHECK-NEXT:   %__llgo_funcval_code115 = call ptr asm "", "=r,0"(ptr %870)
// CHECK-NEXT:   %871 = call %reflect.Value %__llgo_funcval_code115(ptr {{(nest|swiftself)}} %869, %"{{.*}}/runtime/internal/runtime.eface" %868)
// CHECK-NEXT:   store %reflect.Value %864, ptr %858, align 8
// CHECK-NEXT:   store %reflect.Value %871, ptr %865, align 8
// CHECK-NEXT:   %872 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 58
// CHECK-NEXT:   %873 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %872, i32 0, i32 0
// CHECK-NEXT:   %874 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %875 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 58, ptr %875, align 2
// CHECK-NEXT:   %876 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %875, 1
// CHECK-NEXT:   %877 = extractvalue { ptr, ptr } %874, 1
// CHECK-NEXT:   %878 = extractvalue { ptr, ptr } %874, 0
// CHECK-NEXT:   %__llgo_funcval_code116 = call ptr asm "", "=r,0"(ptr %878)
// CHECK-NEXT:   %879 = call %reflect.Value %__llgo_funcval_code116(ptr {{(nest|swiftself)}} %877, %"{{.*}}/runtime/internal/runtime.eface" %876)
// CHECK-NEXT:   %880 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %872, i32 0, i32 1
// CHECK-NEXT:   %881 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %882 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 58, ptr %882, align 8
// CHECK-NEXT:   %883 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %882, 1
// CHECK-NEXT:   %884 = extractvalue { ptr, ptr } %881, 1
// CHECK-NEXT:   %885 = extractvalue { ptr, ptr } %881, 0
// CHECK-NEXT:   %__llgo_funcval_code117 = call ptr asm "", "=r,0"(ptr %885)
// CHECK-NEXT:   %886 = call %reflect.Value %__llgo_funcval_code117(ptr {{(nest|swiftself)}} %884, %"{{.*}}/runtime/internal/runtime.eface" %883)
// CHECK-NEXT:   store %reflect.Value %879, ptr %873, align 8
// CHECK-NEXT:   store %reflect.Value %886, ptr %880, align 8
// CHECK-NEXT:   %887 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 59
// CHECK-NEXT:   %888 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %887, i32 0, i32 0
// CHECK-NEXT:   %889 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %890 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 59, ptr %890, align 8
// CHECK-NEXT:   %891 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %890, 1
// CHECK-NEXT:   %892 = extractvalue { ptr, ptr } %889, 1
// CHECK-NEXT:   %893 = extractvalue { ptr, ptr } %889, 0
// CHECK-NEXT:   %__llgo_funcval_code118 = call ptr asm "", "=r,0"(ptr %893)
// CHECK-NEXT:   %894 = call %reflect.Value %__llgo_funcval_code118(ptr {{(nest|swiftself)}} %892, %"{{.*}}/runtime/internal/runtime.eface" %891)
// CHECK-NEXT:   %895 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %887, i32 0, i32 1
// CHECK-NEXT:   %896 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %897 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 59, ptr %897, align 2
// CHECK-NEXT:   %898 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %897, 1
// CHECK-NEXT:   %899 = extractvalue { ptr, ptr } %896, 1
// CHECK-NEXT:   %900 = extractvalue { ptr, ptr } %896, 0
// CHECK-NEXT:   %__llgo_funcval_code119 = call ptr asm "", "=r,0"(ptr %900)
// CHECK-NEXT:   %901 = call %reflect.Value %__llgo_funcval_code119(ptr {{(nest|swiftself)}} %899, %"{{.*}}/runtime/internal/runtime.eface" %898)
// CHECK-NEXT:   store %reflect.Value %894, ptr %888, align 8
// CHECK-NEXT:   store %reflect.Value %901, ptr %895, align 8
// CHECK-NEXT:   %902 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 60
// CHECK-NEXT:   %903 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %902, i32 0, i32 0
// CHECK-NEXT:   %904 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %905 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 60, ptr %905, align 2
// CHECK-NEXT:   %906 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %905, 1
// CHECK-NEXT:   %907 = extractvalue { ptr, ptr } %904, 1
// CHECK-NEXT:   %908 = extractvalue { ptr, ptr } %904, 0
// CHECK-NEXT:   %__llgo_funcval_code120 = call ptr asm "", "=r,0"(ptr %908)
// CHECK-NEXT:   %909 = call %reflect.Value %__llgo_funcval_code120(ptr {{(nest|swiftself)}} %907, %"{{.*}}/runtime/internal/runtime.eface" %906)
// CHECK-NEXT:   %910 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %902, i32 0, i32 1
// CHECK-NEXT:   %911 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %912 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 60, ptr %912, align 8
// CHECK-NEXT:   %913 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %912, 1
// CHECK-NEXT:   %914 = extractvalue { ptr, ptr } %911, 1
// CHECK-NEXT:   %915 = extractvalue { ptr, ptr } %911, 0
// CHECK-NEXT:   %__llgo_funcval_code121 = call ptr asm "", "=r,0"(ptr %915)
// CHECK-NEXT:   %916 = call %reflect.Value %__llgo_funcval_code121(ptr {{(nest|swiftself)}} %914, %"{{.*}}/runtime/internal/runtime.eface" %913)
// CHECK-NEXT:   store %reflect.Value %909, ptr %903, align 8
// CHECK-NEXT:   store %reflect.Value %916, ptr %910, align 8
// CHECK-NEXT:   %917 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 61
// CHECK-NEXT:   %918 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %917, i32 0, i32 0
// CHECK-NEXT:   %919 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %920 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 61, ptr %920, align 8
// CHECK-NEXT:   %921 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %920, 1
// CHECK-NEXT:   %922 = extractvalue { ptr, ptr } %919, 1
// CHECK-NEXT:   %923 = extractvalue { ptr, ptr } %919, 0
// CHECK-NEXT:   %__llgo_funcval_code122 = call ptr asm "", "=r,0"(ptr %923)
// CHECK-NEXT:   %924 = call %reflect.Value %__llgo_funcval_code122(ptr {{(nest|swiftself)}} %922, %"{{.*}}/runtime/internal/runtime.eface" %921)
// CHECK-NEXT:   %925 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %917, i32 0, i32 1
// CHECK-NEXT:   %926 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %927 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 61, ptr %927, align 2
// CHECK-NEXT:   %928 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %927, 1
// CHECK-NEXT:   %929 = extractvalue { ptr, ptr } %926, 1
// CHECK-NEXT:   %930 = extractvalue { ptr, ptr } %926, 0
// CHECK-NEXT:   %__llgo_funcval_code123 = call ptr asm "", "=r,0"(ptr %930)
// CHECK-NEXT:   %931 = call %reflect.Value %__llgo_funcval_code123(ptr {{(nest|swiftself)}} %929, %"{{.*}}/runtime/internal/runtime.eface" %928)
// CHECK-NEXT:   store %reflect.Value %924, ptr %918, align 8
// CHECK-NEXT:   store %reflect.Value %931, ptr %925, align 8
// CHECK-NEXT:   %932 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 62
// CHECK-NEXT:   %933 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %932, i32 0, i32 0
// CHECK-NEXT:   %934 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %935 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 62, ptr %935, align 2
// CHECK-NEXT:   %936 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %935, 1
// CHECK-NEXT:   %937 = extractvalue { ptr, ptr } %934, 1
// CHECK-NEXT:   %938 = extractvalue { ptr, ptr } %934, 0
// CHECK-NEXT:   %__llgo_funcval_code124 = call ptr asm "", "=r,0"(ptr %938)
// CHECK-NEXT:   %939 = call %reflect.Value %__llgo_funcval_code124(ptr {{(nest|swiftself)}} %937, %"{{.*}}/runtime/internal/runtime.eface" %936)
// CHECK-NEXT:   %940 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %932, i32 0, i32 1
// CHECK-NEXT:   %941 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %942 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 62, ptr %942, align 8
// CHECK-NEXT:   %943 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %942, 1
// CHECK-NEXT:   %944 = extractvalue { ptr, ptr } %941, 1
// CHECK-NEXT:   %945 = extractvalue { ptr, ptr } %941, 0
// CHECK-NEXT:   %__llgo_funcval_code125 = call ptr asm "", "=r,0"(ptr %945)
// CHECK-NEXT:   %946 = call %reflect.Value %__llgo_funcval_code125(ptr {{(nest|swiftself)}} %944, %"{{.*}}/runtime/internal/runtime.eface" %943)
// CHECK-NEXT:   store %reflect.Value %939, ptr %933, align 8
// CHECK-NEXT:   store %reflect.Value %946, ptr %940, align 8
// CHECK-NEXT:   %947 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 63
// CHECK-NEXT:   %948 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %947, i32 0, i32 0
// CHECK-NEXT:   %949 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %950 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 63, ptr %950, align 8
// CHECK-NEXT:   %951 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %950, 1
// CHECK-NEXT:   %952 = extractvalue { ptr, ptr } %949, 1
// CHECK-NEXT:   %953 = extractvalue { ptr, ptr } %949, 0
// CHECK-NEXT:   %__llgo_funcval_code126 = call ptr asm "", "=r,0"(ptr %953)
// CHECK-NEXT:   %954 = call %reflect.Value %__llgo_funcval_code126(ptr {{(nest|swiftself)}} %952, %"{{.*}}/runtime/internal/runtime.eface" %951)
// CHECK-NEXT:   %955 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %947, i32 0, i32 1
// CHECK-NEXT:   %956 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %957 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 63, ptr %957, align 2
// CHECK-NEXT:   %958 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %957, 1
// CHECK-NEXT:   %959 = extractvalue { ptr, ptr } %956, 1
// CHECK-NEXT:   %960 = extractvalue { ptr, ptr } %956, 0
// CHECK-NEXT:   %__llgo_funcval_code127 = call ptr asm "", "=r,0"(ptr %960)
// CHECK-NEXT:   %961 = call %reflect.Value %__llgo_funcval_code127(ptr {{(nest|swiftself)}} %959, %"{{.*}}/runtime/internal/runtime.eface" %958)
// CHECK-NEXT:   store %reflect.Value %954, ptr %948, align 8
// CHECK-NEXT:   store %reflect.Value %961, ptr %955, align 8
// CHECK-NEXT:   %962 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 64
// CHECK-NEXT:   %963 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %962, i32 0, i32 0
// CHECK-NEXT:   %964 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %965 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 64, ptr %965, align 2
// CHECK-NEXT:   %966 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %965, 1
// CHECK-NEXT:   %967 = extractvalue { ptr, ptr } %964, 1
// CHECK-NEXT:   %968 = extractvalue { ptr, ptr } %964, 0
// CHECK-NEXT:   %__llgo_funcval_code128 = call ptr asm "", "=r,0"(ptr %968)
// CHECK-NEXT:   %969 = call %reflect.Value %__llgo_funcval_code128(ptr {{(nest|swiftself)}} %967, %"{{.*}}/runtime/internal/runtime.eface" %966)
// CHECK-NEXT:   %970 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %962, i32 0, i32 1
// CHECK-NEXT:   %971 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %972 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 64, ptr %972, align 8
// CHECK-NEXT:   %973 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %972, 1
// CHECK-NEXT:   %974 = extractvalue { ptr, ptr } %971, 1
// CHECK-NEXT:   %975 = extractvalue { ptr, ptr } %971, 0
// CHECK-NEXT:   %__llgo_funcval_code129 = call ptr asm "", "=r,0"(ptr %975)
// CHECK-NEXT:   %976 = call %reflect.Value %__llgo_funcval_code129(ptr {{(nest|swiftself)}} %974, %"{{.*}}/runtime/internal/runtime.eface" %973)
// CHECK-NEXT:   store %reflect.Value %969, ptr %963, align 8
// CHECK-NEXT:   store %reflect.Value %976, ptr %970, align 8
// CHECK-NEXT:   %977 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 65
// CHECK-NEXT:   %978 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %977, i32 0, i32 0
// CHECK-NEXT:   %979 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %980 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 65, ptr %980, align 8
// CHECK-NEXT:   %981 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %980, 1
// CHECK-NEXT:   %982 = extractvalue { ptr, ptr } %979, 1
// CHECK-NEXT:   %983 = extractvalue { ptr, ptr } %979, 0
// CHECK-NEXT:   %__llgo_funcval_code130 = call ptr asm "", "=r,0"(ptr %983)
// CHECK-NEXT:   %984 = call %reflect.Value %__llgo_funcval_code130(ptr {{(nest|swiftself)}} %982, %"{{.*}}/runtime/internal/runtime.eface" %981)
// CHECK-NEXT:   %985 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %977, i32 0, i32 1
// CHECK-NEXT:   %986 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %987 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 65, ptr %987, align 2
// CHECK-NEXT:   %988 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %987, 1
// CHECK-NEXT:   %989 = extractvalue { ptr, ptr } %986, 1
// CHECK-NEXT:   %990 = extractvalue { ptr, ptr } %986, 0
// CHECK-NEXT:   %__llgo_funcval_code131 = call ptr asm "", "=r,0"(ptr %990)
// CHECK-NEXT:   %991 = call %reflect.Value %__llgo_funcval_code131(ptr {{(nest|swiftself)}} %989, %"{{.*}}/runtime/internal/runtime.eface" %988)
// CHECK-NEXT:   store %reflect.Value %984, ptr %978, align 8
// CHECK-NEXT:   store %reflect.Value %991, ptr %985, align 8
// CHECK-NEXT:   %992 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 66
// CHECK-NEXT:   %993 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %992, i32 0, i32 0
// CHECK-NEXT:   %994 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %995 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 66, ptr %995, align 2
// CHECK-NEXT:   %996 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %995, 1
// CHECK-NEXT:   %997 = extractvalue { ptr, ptr } %994, 1
// CHECK-NEXT:   %998 = extractvalue { ptr, ptr } %994, 0
// CHECK-NEXT:   %__llgo_funcval_code132 = call ptr asm "", "=r,0"(ptr %998)
// CHECK-NEXT:   %999 = call %reflect.Value %__llgo_funcval_code132(ptr {{(nest|swiftself)}} %997, %"{{.*}}/runtime/internal/runtime.eface" %996)
// CHECK-NEXT:   %1000 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %992, i32 0, i32 1
// CHECK-NEXT:   %1001 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1002 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 6.600000e+01, ptr %1002, align 4
// CHECK-NEXT:   %1003 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1002, 1
// CHECK-NEXT:   %1004 = extractvalue { ptr, ptr } %1001, 1
// CHECK-NEXT:   %1005 = extractvalue { ptr, ptr } %1001, 0
// CHECK-NEXT:   %__llgo_funcval_code133 = call ptr asm "", "=r,0"(ptr %1005)
// CHECK-NEXT:   %1006 = call %reflect.Value %__llgo_funcval_code133(ptr {{(nest|swiftself)}} %1004, %"{{.*}}/runtime/internal/runtime.eface" %1003)
// CHECK-NEXT:   store %reflect.Value %999, ptr %993, align 8
// CHECK-NEXT:   store %reflect.Value %1006, ptr %1000, align 8
// CHECK-NEXT:   %1007 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 67
// CHECK-NEXT:   %1008 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1007, i32 0, i32 0
// CHECK-NEXT:   %1009 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1010 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 6.700000e+01, ptr %1010, align 4
// CHECK-NEXT:   %1011 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1010, 1
// CHECK-NEXT:   %1012 = extractvalue { ptr, ptr } %1009, 1
// CHECK-NEXT:   %1013 = extractvalue { ptr, ptr } %1009, 0
// CHECK-NEXT:   %__llgo_funcval_code134 = call ptr asm "", "=r,0"(ptr %1013)
// CHECK-NEXT:   %1014 = call %reflect.Value %__llgo_funcval_code134(ptr {{(nest|swiftself)}} %1012, %"{{.*}}/runtime/internal/runtime.eface" %1011)
// CHECK-NEXT:   %1015 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1007, i32 0, i32 1
// CHECK-NEXT:   %1016 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1017 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 67, ptr %1017, align 2
// CHECK-NEXT:   %1018 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %1017, 1
// CHECK-NEXT:   %1019 = extractvalue { ptr, ptr } %1016, 1
// CHECK-NEXT:   %1020 = extractvalue { ptr, ptr } %1016, 0
// CHECK-NEXT:   %__llgo_funcval_code135 = call ptr asm "", "=r,0"(ptr %1020)
// CHECK-NEXT:   %1021 = call %reflect.Value %__llgo_funcval_code135(ptr {{(nest|swiftself)}} %1019, %"{{.*}}/runtime/internal/runtime.eface" %1018)
// CHECK-NEXT:   store %reflect.Value %1014, ptr %1008, align 8
// CHECK-NEXT:   store %reflect.Value %1021, ptr %1015, align 8
// CHECK-NEXT:   %1022 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 68
// CHECK-NEXT:   %1023 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1022, i32 0, i32 0
// CHECK-NEXT:   %1024 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1025 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 68, ptr %1025, align 2
// CHECK-NEXT:   %1026 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %1025, 1
// CHECK-NEXT:   %1027 = extractvalue { ptr, ptr } %1024, 1
// CHECK-NEXT:   %1028 = extractvalue { ptr, ptr } %1024, 0
// CHECK-NEXT:   %__llgo_funcval_code136 = call ptr asm "", "=r,0"(ptr %1028)
// CHECK-NEXT:   %1029 = call %reflect.Value %__llgo_funcval_code136(ptr {{(nest|swiftself)}} %1027, %"{{.*}}/runtime/internal/runtime.eface" %1026)
// CHECK-NEXT:   %1030 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1022, i32 0, i32 1
// CHECK-NEXT:   %1031 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1032 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 6.800000e+01, ptr %1032, align 8
// CHECK-NEXT:   %1033 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1032, 1
// CHECK-NEXT:   %1034 = extractvalue { ptr, ptr } %1031, 1
// CHECK-NEXT:   %1035 = extractvalue { ptr, ptr } %1031, 0
// CHECK-NEXT:   %__llgo_funcval_code137 = call ptr asm "", "=r,0"(ptr %1035)
// CHECK-NEXT:   %1036 = call %reflect.Value %__llgo_funcval_code137(ptr {{(nest|swiftself)}} %1034, %"{{.*}}/runtime/internal/runtime.eface" %1033)
// CHECK-NEXT:   store %reflect.Value %1029, ptr %1023, align 8
// CHECK-NEXT:   store %reflect.Value %1036, ptr %1030, align 8
// CHECK-NEXT:   %1037 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 69
// CHECK-NEXT:   %1038 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1037, i32 0, i32 0
// CHECK-NEXT:   %1039 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1040 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 6.900000e+01, ptr %1040, align 8
// CHECK-NEXT:   %1041 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1040, 1
// CHECK-NEXT:   %1042 = extractvalue { ptr, ptr } %1039, 1
// CHECK-NEXT:   %1043 = extractvalue { ptr, ptr } %1039, 0
// CHECK-NEXT:   %__llgo_funcval_code138 = call ptr asm "", "=r,0"(ptr %1043)
// CHECK-NEXT:   %1044 = call %reflect.Value %__llgo_funcval_code138(ptr {{(nest|swiftself)}} %1042, %"{{.*}}/runtime/internal/runtime.eface" %1041)
// CHECK-NEXT:   %1045 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1037, i32 0, i32 1
// CHECK-NEXT:   %1046 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1047 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 69, ptr %1047, align 2
// CHECK-NEXT:   %1048 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %1047, 1
// CHECK-NEXT:   %1049 = extractvalue { ptr, ptr } %1046, 1
// CHECK-NEXT:   %1050 = extractvalue { ptr, ptr } %1046, 0
// CHECK-NEXT:   %__llgo_funcval_code139 = call ptr asm "", "=r,0"(ptr %1050)
// CHECK-NEXT:   %1051 = call %reflect.Value %__llgo_funcval_code139(ptr {{(nest|swiftself)}} %1049, %"{{.*}}/runtime/internal/runtime.eface" %1048)
// CHECK-NEXT:   store %reflect.Value %1044, ptr %1038, align 8
// CHECK-NEXT:   store %reflect.Value %1051, ptr %1045, align 8
// CHECK-NEXT:   %1052 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 70
// CHECK-NEXT:   %1053 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1052, i32 0, i32 0
// CHECK-NEXT:   %1054 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1055 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 70, ptr %1055, align 2
// CHECK-NEXT:   %1056 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1055, 1
// CHECK-NEXT:   %1057 = extractvalue { ptr, ptr } %1054, 1
// CHECK-NEXT:   %1058 = extractvalue { ptr, ptr } %1054, 0
// CHECK-NEXT:   %__llgo_funcval_code140 = call ptr asm "", "=r,0"(ptr %1058)
// CHECK-NEXT:   %1059 = call %reflect.Value %__llgo_funcval_code140(ptr {{(nest|swiftself)}} %1057, %"{{.*}}/runtime/internal/runtime.eface" %1056)
// CHECK-NEXT:   %1060 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1052, i32 0, i32 1
// CHECK-NEXT:   %1061 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1062 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 70, ptr %1062, align 2
// CHECK-NEXT:   %1063 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1062, 1
// CHECK-NEXT:   %1064 = extractvalue { ptr, ptr } %1061, 1
// CHECK-NEXT:   %1065 = extractvalue { ptr, ptr } %1061, 0
// CHECK-NEXT:   %__llgo_funcval_code141 = call ptr asm "", "=r,0"(ptr %1065)
// CHECK-NEXT:   %1066 = call %reflect.Value %__llgo_funcval_code141(ptr {{(nest|swiftself)}} %1064, %"{{.*}}/runtime/internal/runtime.eface" %1063)
// CHECK-NEXT:   store %reflect.Value %1059, ptr %1053, align 8
// CHECK-NEXT:   store %reflect.Value %1066, ptr %1060, align 8
// CHECK-NEXT:   %1067 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 71
// CHECK-NEXT:   %1068 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1067, i32 0, i32 0
// CHECK-NEXT:   %1069 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1070 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 71, ptr %1070, align 2
// CHECK-NEXT:   %1071 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1070, 1
// CHECK-NEXT:   %1072 = extractvalue { ptr, ptr } %1069, 1
// CHECK-NEXT:   %1073 = extractvalue { ptr, ptr } %1069, 0
// CHECK-NEXT:   %__llgo_funcval_code142 = call ptr asm "", "=r,0"(ptr %1073)
// CHECK-NEXT:   %1074 = call %reflect.Value %__llgo_funcval_code142(ptr {{(nest|swiftself)}} %1072, %"{{.*}}/runtime/internal/runtime.eface" %1071)
// CHECK-NEXT:   %1075 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1067, i32 0, i32 1
// CHECK-NEXT:   %1076 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1077 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 71, ptr %1077, align 4
// CHECK-NEXT:   %1078 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1077, 1
// CHECK-NEXT:   %1079 = extractvalue { ptr, ptr } %1076, 1
// CHECK-NEXT:   %1080 = extractvalue { ptr, ptr } %1076, 0
// CHECK-NEXT:   %__llgo_funcval_code143 = call ptr asm "", "=r,0"(ptr %1080)
// CHECK-NEXT:   %1081 = call %reflect.Value %__llgo_funcval_code143(ptr {{(nest|swiftself)}} %1079, %"{{.*}}/runtime/internal/runtime.eface" %1078)
// CHECK-NEXT:   store %reflect.Value %1074, ptr %1068, align 8
// CHECK-NEXT:   store %reflect.Value %1081, ptr %1075, align 8
// CHECK-NEXT:   %1082 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 72
// CHECK-NEXT:   %1083 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1082, i32 0, i32 0
// CHECK-NEXT:   %1084 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1085 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 72, ptr %1085, align 4
// CHECK-NEXT:   %1086 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1085, 1
// CHECK-NEXT:   %1087 = extractvalue { ptr, ptr } %1084, 1
// CHECK-NEXT:   %1088 = extractvalue { ptr, ptr } %1084, 0
// CHECK-NEXT:   %__llgo_funcval_code144 = call ptr asm "", "=r,0"(ptr %1088)
// CHECK-NEXT:   %1089 = call %reflect.Value %__llgo_funcval_code144(ptr {{(nest|swiftself)}} %1087, %"{{.*}}/runtime/internal/runtime.eface" %1086)
// CHECK-NEXT:   %1090 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1082, i32 0, i32 1
// CHECK-NEXT:   %1091 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1092 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 72, ptr %1092, align 2
// CHECK-NEXT:   %1093 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1092, 1
// CHECK-NEXT:   %1094 = extractvalue { ptr, ptr } %1091, 1
// CHECK-NEXT:   %1095 = extractvalue { ptr, ptr } %1091, 0
// CHECK-NEXT:   %__llgo_funcval_code145 = call ptr asm "", "=r,0"(ptr %1095)
// CHECK-NEXT:   %1096 = call %reflect.Value %__llgo_funcval_code145(ptr {{(nest|swiftself)}} %1094, %"{{.*}}/runtime/internal/runtime.eface" %1093)
// CHECK-NEXT:   store %reflect.Value %1089, ptr %1083, align 8
// CHECK-NEXT:   store %reflect.Value %1096, ptr %1090, align 8
// CHECK-NEXT:   %1097 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 73
// CHECK-NEXT:   %1098 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1097, i32 0, i32 0
// CHECK-NEXT:   %1099 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1100 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 73, ptr %1100, align 2
// CHECK-NEXT:   %1101 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1100, 1
// CHECK-NEXT:   %1102 = extractvalue { ptr, ptr } %1099, 1
// CHECK-NEXT:   %1103 = extractvalue { ptr, ptr } %1099, 0
// CHECK-NEXT:   %__llgo_funcval_code146 = call ptr asm "", "=r,0"(ptr %1103)
// CHECK-NEXT:   %1104 = call %reflect.Value %__llgo_funcval_code146(ptr {{(nest|swiftself)}} %1102, %"{{.*}}/runtime/internal/runtime.eface" %1101)
// CHECK-NEXT:   %1105 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1097, i32 0, i32 1
// CHECK-NEXT:   %1106 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1107 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 73, ptr %1107, align 4
// CHECK-NEXT:   %1108 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1107, 1
// CHECK-NEXT:   %1109 = extractvalue { ptr, ptr } %1106, 1
// CHECK-NEXT:   %1110 = extractvalue { ptr, ptr } %1106, 0
// CHECK-NEXT:   %__llgo_funcval_code147 = call ptr asm "", "=r,0"(ptr %1110)
// CHECK-NEXT:   %1111 = call %reflect.Value %__llgo_funcval_code147(ptr {{(nest|swiftself)}} %1109, %"{{.*}}/runtime/internal/runtime.eface" %1108)
// CHECK-NEXT:   store %reflect.Value %1104, ptr %1098, align 8
// CHECK-NEXT:   store %reflect.Value %1111, ptr %1105, align 8
// CHECK-NEXT:   %1112 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 74
// CHECK-NEXT:   %1113 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1112, i32 0, i32 0
// CHECK-NEXT:   %1114 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1115 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 74, ptr %1115, align 4
// CHECK-NEXT:   %1116 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1115, 1
// CHECK-NEXT:   %1117 = extractvalue { ptr, ptr } %1114, 1
// CHECK-NEXT:   %1118 = extractvalue { ptr, ptr } %1114, 0
// CHECK-NEXT:   %__llgo_funcval_code148 = call ptr asm "", "=r,0"(ptr %1118)
// CHECK-NEXT:   %1119 = call %reflect.Value %__llgo_funcval_code148(ptr {{(nest|swiftself)}} %1117, %"{{.*}}/runtime/internal/runtime.eface" %1116)
// CHECK-NEXT:   %1120 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1112, i32 0, i32 1
// CHECK-NEXT:   %1121 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1122 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 74, ptr %1122, align 2
// CHECK-NEXT:   %1123 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1122, 1
// CHECK-NEXT:   %1124 = extractvalue { ptr, ptr } %1121, 1
// CHECK-NEXT:   %1125 = extractvalue { ptr, ptr } %1121, 0
// CHECK-NEXT:   %__llgo_funcval_code149 = call ptr asm "", "=r,0"(ptr %1125)
// CHECK-NEXT:   %1126 = call %reflect.Value %__llgo_funcval_code149(ptr {{(nest|swiftself)}} %1124, %"{{.*}}/runtime/internal/runtime.eface" %1123)
// CHECK-NEXT:   store %reflect.Value %1119, ptr %1113, align 8
// CHECK-NEXT:   store %reflect.Value %1126, ptr %1120, align 8
// CHECK-NEXT:   %1127 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 75
// CHECK-NEXT:   %1128 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1127, i32 0, i32 0
// CHECK-NEXT:   %1129 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1130 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 75, ptr %1130, align 2
// CHECK-NEXT:   %1131 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1130, 1
// CHECK-NEXT:   %1132 = extractvalue { ptr, ptr } %1129, 1
// CHECK-NEXT:   %1133 = extractvalue { ptr, ptr } %1129, 0
// CHECK-NEXT:   %__llgo_funcval_code150 = call ptr asm "", "=r,0"(ptr %1133)
// CHECK-NEXT:   %1134 = call %reflect.Value %__llgo_funcval_code150(ptr {{(nest|swiftself)}} %1132, %"{{.*}}/runtime/internal/runtime.eface" %1131)
// CHECK-NEXT:   %1135 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1127, i32 0, i32 1
// CHECK-NEXT:   %1136 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1137 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 75, ptr %1137, align 8
// CHECK-NEXT:   %1138 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1137, 1
// CHECK-NEXT:   %1139 = extractvalue { ptr, ptr } %1136, 1
// CHECK-NEXT:   %1140 = extractvalue { ptr, ptr } %1136, 0
// CHECK-NEXT:   %__llgo_funcval_code151 = call ptr asm "", "=r,0"(ptr %1140)
// CHECK-NEXT:   %1141 = call %reflect.Value %__llgo_funcval_code151(ptr {{(nest|swiftself)}} %1139, %"{{.*}}/runtime/internal/runtime.eface" %1138)
// CHECK-NEXT:   store %reflect.Value %1134, ptr %1128, align 8
// CHECK-NEXT:   store %reflect.Value %1141, ptr %1135, align 8
// CHECK-NEXT:   %1142 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 76
// CHECK-NEXT:   %1143 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1142, i32 0, i32 0
// CHECK-NEXT:   %1144 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1145 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 76, ptr %1145, align 8
// CHECK-NEXT:   %1146 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1145, 1
// CHECK-NEXT:   %1147 = extractvalue { ptr, ptr } %1144, 1
// CHECK-NEXT:   %1148 = extractvalue { ptr, ptr } %1144, 0
// CHECK-NEXT:   %__llgo_funcval_code152 = call ptr asm "", "=r,0"(ptr %1148)
// CHECK-NEXT:   %1149 = call %reflect.Value %__llgo_funcval_code152(ptr {{(nest|swiftself)}} %1147, %"{{.*}}/runtime/internal/runtime.eface" %1146)
// CHECK-NEXT:   %1150 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1142, i32 0, i32 1
// CHECK-NEXT:   %1151 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1152 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 76, ptr %1152, align 2
// CHECK-NEXT:   %1153 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1152, 1
// CHECK-NEXT:   %1154 = extractvalue { ptr, ptr } %1151, 1
// CHECK-NEXT:   %1155 = extractvalue { ptr, ptr } %1151, 0
// CHECK-NEXT:   %__llgo_funcval_code153 = call ptr asm "", "=r,0"(ptr %1155)
// CHECK-NEXT:   %1156 = call %reflect.Value %__llgo_funcval_code153(ptr {{(nest|swiftself)}} %1154, %"{{.*}}/runtime/internal/runtime.eface" %1153)
// CHECK-NEXT:   store %reflect.Value %1149, ptr %1143, align 8
// CHECK-NEXT:   store %reflect.Value %1156, ptr %1150, align 8
// CHECK-NEXT:   %1157 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 77
// CHECK-NEXT:   %1158 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1157, i32 0, i32 0
// CHECK-NEXT:   %1159 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1160 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 77, ptr %1160, align 2
// CHECK-NEXT:   %1161 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1160, 1
// CHECK-NEXT:   %1162 = extractvalue { ptr, ptr } %1159, 1
// CHECK-NEXT:   %1163 = extractvalue { ptr, ptr } %1159, 0
// CHECK-NEXT:   %__llgo_funcval_code154 = call ptr asm "", "=r,0"(ptr %1163)
// CHECK-NEXT:   %1164 = call %reflect.Value %__llgo_funcval_code154(ptr {{(nest|swiftself)}} %1162, %"{{.*}}/runtime/internal/runtime.eface" %1161)
// CHECK-NEXT:   %1165 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1157, i32 0, i32 1
// CHECK-NEXT:   %1166 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1167 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 77, ptr %1167, align 8
// CHECK-NEXT:   %1168 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1167, 1
// CHECK-NEXT:   %1169 = extractvalue { ptr, ptr } %1166, 1
// CHECK-NEXT:   %1170 = extractvalue { ptr, ptr } %1166, 0
// CHECK-NEXT:   %__llgo_funcval_code155 = call ptr asm "", "=r,0"(ptr %1170)
// CHECK-NEXT:   %1171 = call %reflect.Value %__llgo_funcval_code155(ptr {{(nest|swiftself)}} %1169, %"{{.*}}/runtime/internal/runtime.eface" %1168)
// CHECK-NEXT:   store %reflect.Value %1164, ptr %1158, align 8
// CHECK-NEXT:   store %reflect.Value %1171, ptr %1165, align 8
// CHECK-NEXT:   %1172 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 78
// CHECK-NEXT:   %1173 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1172, i32 0, i32 0
// CHECK-NEXT:   %1174 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1175 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 78, ptr %1175, align 8
// CHECK-NEXT:   %1176 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1175, 1
// CHECK-NEXT:   %1177 = extractvalue { ptr, ptr } %1174, 1
// CHECK-NEXT:   %1178 = extractvalue { ptr, ptr } %1174, 0
// CHECK-NEXT:   %__llgo_funcval_code156 = call ptr asm "", "=r,0"(ptr %1178)
// CHECK-NEXT:   %1179 = call %reflect.Value %__llgo_funcval_code156(ptr {{(nest|swiftself)}} %1177, %"{{.*}}/runtime/internal/runtime.eface" %1176)
// CHECK-NEXT:   %1180 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1172, i32 0, i32 1
// CHECK-NEXT:   %1181 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1182 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 78, ptr %1182, align 2
// CHECK-NEXT:   %1183 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1182, 1
// CHECK-NEXT:   %1184 = extractvalue { ptr, ptr } %1181, 1
// CHECK-NEXT:   %1185 = extractvalue { ptr, ptr } %1181, 0
// CHECK-NEXT:   %__llgo_funcval_code157 = call ptr asm "", "=r,0"(ptr %1185)
// CHECK-NEXT:   %1186 = call %reflect.Value %__llgo_funcval_code157(ptr {{(nest|swiftself)}} %1184, %"{{.*}}/runtime/internal/runtime.eface" %1183)
// CHECK-NEXT:   store %reflect.Value %1179, ptr %1173, align 8
// CHECK-NEXT:   store %reflect.Value %1186, ptr %1180, align 8
// CHECK-NEXT:   %1187 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 79
// CHECK-NEXT:   %1188 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1187, i32 0, i32 0
// CHECK-NEXT:   %1189 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1190 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 79, ptr %1190, align 2
// CHECK-NEXT:   %1191 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1190, 1
// CHECK-NEXT:   %1192 = extractvalue { ptr, ptr } %1189, 1
// CHECK-NEXT:   %1193 = extractvalue { ptr, ptr } %1189, 0
// CHECK-NEXT:   %__llgo_funcval_code158 = call ptr asm "", "=r,0"(ptr %1193)
// CHECK-NEXT:   %1194 = call %reflect.Value %__llgo_funcval_code158(ptr {{(nest|swiftself)}} %1192, %"{{.*}}/runtime/internal/runtime.eface" %1191)
// CHECK-NEXT:   %1195 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1187, i32 0, i32 1
// CHECK-NEXT:   %1196 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1197 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 79, ptr %1197, align 8
// CHECK-NEXT:   %1198 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1197, 1
// CHECK-NEXT:   %1199 = extractvalue { ptr, ptr } %1196, 1
// CHECK-NEXT:   %1200 = extractvalue { ptr, ptr } %1196, 0
// CHECK-NEXT:   %__llgo_funcval_code159 = call ptr asm "", "=r,0"(ptr %1200)
// CHECK-NEXT:   %1201 = call %reflect.Value %__llgo_funcval_code159(ptr {{(nest|swiftself)}} %1199, %"{{.*}}/runtime/internal/runtime.eface" %1198)
// CHECK-NEXT:   store %reflect.Value %1194, ptr %1188, align 8
// CHECK-NEXT:   store %reflect.Value %1201, ptr %1195, align 8
// CHECK-NEXT:   %1202 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 80
// CHECK-NEXT:   %1203 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1202, i32 0, i32 0
// CHECK-NEXT:   %1204 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1205 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 80, ptr %1205, align 8
// CHECK-NEXT:   %1206 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1205, 1
// CHECK-NEXT:   %1207 = extractvalue { ptr, ptr } %1204, 1
// CHECK-NEXT:   %1208 = extractvalue { ptr, ptr } %1204, 0
// CHECK-NEXT:   %__llgo_funcval_code160 = call ptr asm "", "=r,0"(ptr %1208)
// CHECK-NEXT:   %1209 = call %reflect.Value %__llgo_funcval_code160(ptr {{(nest|swiftself)}} %1207, %"{{.*}}/runtime/internal/runtime.eface" %1206)
// CHECK-NEXT:   %1210 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1202, i32 0, i32 1
// CHECK-NEXT:   %1211 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1212 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 80, ptr %1212, align 2
// CHECK-NEXT:   %1213 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1212, 1
// CHECK-NEXT:   %1214 = extractvalue { ptr, ptr } %1211, 1
// CHECK-NEXT:   %1215 = extractvalue { ptr, ptr } %1211, 0
// CHECK-NEXT:   %__llgo_funcval_code161 = call ptr asm "", "=r,0"(ptr %1215)
// CHECK-NEXT:   %1216 = call %reflect.Value %__llgo_funcval_code161(ptr {{(nest|swiftself)}} %1214, %"{{.*}}/runtime/internal/runtime.eface" %1213)
// CHECK-NEXT:   store %reflect.Value %1209, ptr %1203, align 8
// CHECK-NEXT:   store %reflect.Value %1216, ptr %1210, align 8
// CHECK-NEXT:   %1217 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 81
// CHECK-NEXT:   %1218 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1217, i32 0, i32 0
// CHECK-NEXT:   %1219 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1220 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 81, ptr %1220, align 2
// CHECK-NEXT:   %1221 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1220, 1
// CHECK-NEXT:   %1222 = extractvalue { ptr, ptr } %1219, 1
// CHECK-NEXT:   %1223 = extractvalue { ptr, ptr } %1219, 0
// CHECK-NEXT:   %__llgo_funcval_code162 = call ptr asm "", "=r,0"(ptr %1223)
// CHECK-NEXT:   %1224 = call %reflect.Value %__llgo_funcval_code162(ptr {{(nest|swiftself)}} %1222, %"{{.*}}/runtime/internal/runtime.eface" %1221)
// CHECK-NEXT:   %1225 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1217, i32 0, i32 1
// CHECK-NEXT:   %1226 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1227 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 81, ptr %1227, align 8
// CHECK-NEXT:   %1228 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1227, 1
// CHECK-NEXT:   %1229 = extractvalue { ptr, ptr } %1226, 1
// CHECK-NEXT:   %1230 = extractvalue { ptr, ptr } %1226, 0
// CHECK-NEXT:   %__llgo_funcval_code163 = call ptr asm "", "=r,0"(ptr %1230)
// CHECK-NEXT:   %1231 = call %reflect.Value %__llgo_funcval_code163(ptr {{(nest|swiftself)}} %1229, %"{{.*}}/runtime/internal/runtime.eface" %1228)
// CHECK-NEXT:   store %reflect.Value %1224, ptr %1218, align 8
// CHECK-NEXT:   store %reflect.Value %1231, ptr %1225, align 8
// CHECK-NEXT:   %1232 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 82
// CHECK-NEXT:   %1233 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1232, i32 0, i32 0
// CHECK-NEXT:   %1234 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1235 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 82, ptr %1235, align 8
// CHECK-NEXT:   %1236 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1235, 1
// CHECK-NEXT:   %1237 = extractvalue { ptr, ptr } %1234, 1
// CHECK-NEXT:   %1238 = extractvalue { ptr, ptr } %1234, 0
// CHECK-NEXT:   %__llgo_funcval_code164 = call ptr asm "", "=r,0"(ptr %1238)
// CHECK-NEXT:   %1239 = call %reflect.Value %__llgo_funcval_code164(ptr {{(nest|swiftself)}} %1237, %"{{.*}}/runtime/internal/runtime.eface" %1236)
// CHECK-NEXT:   %1240 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1232, i32 0, i32 1
// CHECK-NEXT:   %1241 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1242 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 82, ptr %1242, align 2
// CHECK-NEXT:   %1243 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1242, 1
// CHECK-NEXT:   %1244 = extractvalue { ptr, ptr } %1241, 1
// CHECK-NEXT:   %1245 = extractvalue { ptr, ptr } %1241, 0
// CHECK-NEXT:   %__llgo_funcval_code165 = call ptr asm "", "=r,0"(ptr %1245)
// CHECK-NEXT:   %1246 = call %reflect.Value %__llgo_funcval_code165(ptr {{(nest|swiftself)}} %1244, %"{{.*}}/runtime/internal/runtime.eface" %1243)
// CHECK-NEXT:   store %reflect.Value %1239, ptr %1233, align 8
// CHECK-NEXT:   store %reflect.Value %1246, ptr %1240, align 8
// CHECK-NEXT:   %1247 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 83
// CHECK-NEXT:   %1248 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1247, i32 0, i32 0
// CHECK-NEXT:   %1249 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1250 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 83, ptr %1250, align 2
// CHECK-NEXT:   %1251 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1250, 1
// CHECK-NEXT:   %1252 = extractvalue { ptr, ptr } %1249, 1
// CHECK-NEXT:   %1253 = extractvalue { ptr, ptr } %1249, 0
// CHECK-NEXT:   %__llgo_funcval_code166 = call ptr asm "", "=r,0"(ptr %1253)
// CHECK-NEXT:   %1254 = call %reflect.Value %__llgo_funcval_code166(ptr {{(nest|swiftself)}} %1252, %"{{.*}}/runtime/internal/runtime.eface" %1251)
// CHECK-NEXT:   %1255 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1247, i32 0, i32 1
// CHECK-NEXT:   %1256 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1257 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 83, ptr %1257, align 8
// CHECK-NEXT:   %1258 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1257, 1
// CHECK-NEXT:   %1259 = extractvalue { ptr, ptr } %1256, 1
// CHECK-NEXT:   %1260 = extractvalue { ptr, ptr } %1256, 0
// CHECK-NEXT:   %__llgo_funcval_code167 = call ptr asm "", "=r,0"(ptr %1260)
// CHECK-NEXT:   %1261 = call %reflect.Value %__llgo_funcval_code167(ptr {{(nest|swiftself)}} %1259, %"{{.*}}/runtime/internal/runtime.eface" %1258)
// CHECK-NEXT:   store %reflect.Value %1254, ptr %1248, align 8
// CHECK-NEXT:   store %reflect.Value %1261, ptr %1255, align 8
// CHECK-NEXT:   %1262 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 84
// CHECK-NEXT:   %1263 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1262, i32 0, i32 0
// CHECK-NEXT:   %1264 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1265 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 84, ptr %1265, align 8
// CHECK-NEXT:   %1266 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1265, 1
// CHECK-NEXT:   %1267 = extractvalue { ptr, ptr } %1264, 1
// CHECK-NEXT:   %1268 = extractvalue { ptr, ptr } %1264, 0
// CHECK-NEXT:   %__llgo_funcval_code168 = call ptr asm "", "=r,0"(ptr %1268)
// CHECK-NEXT:   %1269 = call %reflect.Value %__llgo_funcval_code168(ptr {{(nest|swiftself)}} %1267, %"{{.*}}/runtime/internal/runtime.eface" %1266)
// CHECK-NEXT:   %1270 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1262, i32 0, i32 1
// CHECK-NEXT:   %1271 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1272 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 84, ptr %1272, align 2
// CHECK-NEXT:   %1273 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1272, 1
// CHECK-NEXT:   %1274 = extractvalue { ptr, ptr } %1271, 1
// CHECK-NEXT:   %1275 = extractvalue { ptr, ptr } %1271, 0
// CHECK-NEXT:   %__llgo_funcval_code169 = call ptr asm "", "=r,0"(ptr %1275)
// CHECK-NEXT:   %1276 = call %reflect.Value %__llgo_funcval_code169(ptr {{(nest|swiftself)}} %1274, %"{{.*}}/runtime/internal/runtime.eface" %1273)
// CHECK-NEXT:   store %reflect.Value %1269, ptr %1263, align 8
// CHECK-NEXT:   store %reflect.Value %1276, ptr %1270, align 8
// CHECK-NEXT:   %1277 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 85
// CHECK-NEXT:   %1278 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1277, i32 0, i32 0
// CHECK-NEXT:   %1279 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1280 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 85, ptr %1280, align 2
// CHECK-NEXT:   %1281 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1280, 1
// CHECK-NEXT:   %1282 = extractvalue { ptr, ptr } %1279, 1
// CHECK-NEXT:   %1283 = extractvalue { ptr, ptr } %1279, 0
// CHECK-NEXT:   %__llgo_funcval_code170 = call ptr asm "", "=r,0"(ptr %1283)
// CHECK-NEXT:   %1284 = call %reflect.Value %__llgo_funcval_code170(ptr {{(nest|swiftself)}} %1282, %"{{.*}}/runtime/internal/runtime.eface" %1281)
// CHECK-NEXT:   %1285 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1277, i32 0, i32 1
// CHECK-NEXT:   %1286 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1287 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 8.500000e+01, ptr %1287, align 4
// CHECK-NEXT:   %1288 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1287, 1
// CHECK-NEXT:   %1289 = extractvalue { ptr, ptr } %1286, 1
// CHECK-NEXT:   %1290 = extractvalue { ptr, ptr } %1286, 0
// CHECK-NEXT:   %__llgo_funcval_code171 = call ptr asm "", "=r,0"(ptr %1290)
// CHECK-NEXT:   %1291 = call %reflect.Value %__llgo_funcval_code171(ptr {{(nest|swiftself)}} %1289, %"{{.*}}/runtime/internal/runtime.eface" %1288)
// CHECK-NEXT:   store %reflect.Value %1284, ptr %1278, align 8
// CHECK-NEXT:   store %reflect.Value %1291, ptr %1285, align 8
// CHECK-NEXT:   %1292 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 86
// CHECK-NEXT:   %1293 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1292, i32 0, i32 0
// CHECK-NEXT:   %1294 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1295 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 8.600000e+01, ptr %1295, align 4
// CHECK-NEXT:   %1296 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1295, 1
// CHECK-NEXT:   %1297 = extractvalue { ptr, ptr } %1294, 1
// CHECK-NEXT:   %1298 = extractvalue { ptr, ptr } %1294, 0
// CHECK-NEXT:   %__llgo_funcval_code172 = call ptr asm "", "=r,0"(ptr %1298)
// CHECK-NEXT:   %1299 = call %reflect.Value %__llgo_funcval_code172(ptr {{(nest|swiftself)}} %1297, %"{{.*}}/runtime/internal/runtime.eface" %1296)
// CHECK-NEXT:   %1300 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1292, i32 0, i32 1
// CHECK-NEXT:   %1301 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1302 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 86, ptr %1302, align 2
// CHECK-NEXT:   %1303 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1302, 1
// CHECK-NEXT:   %1304 = extractvalue { ptr, ptr } %1301, 1
// CHECK-NEXT:   %1305 = extractvalue { ptr, ptr } %1301, 0
// CHECK-NEXT:   %__llgo_funcval_code173 = call ptr asm "", "=r,0"(ptr %1305)
// CHECK-NEXT:   %1306 = call %reflect.Value %__llgo_funcval_code173(ptr {{(nest|swiftself)}} %1304, %"{{.*}}/runtime/internal/runtime.eface" %1303)
// CHECK-NEXT:   store %reflect.Value %1299, ptr %1293, align 8
// CHECK-NEXT:   store %reflect.Value %1306, ptr %1300, align 8
// CHECK-NEXT:   %1307 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 87
// CHECK-NEXT:   %1308 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1307, i32 0, i32 0
// CHECK-NEXT:   %1309 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1310 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 87, ptr %1310, align 2
// CHECK-NEXT:   %1311 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1310, 1
// CHECK-NEXT:   %1312 = extractvalue { ptr, ptr } %1309, 1
// CHECK-NEXT:   %1313 = extractvalue { ptr, ptr } %1309, 0
// CHECK-NEXT:   %__llgo_funcval_code174 = call ptr asm "", "=r,0"(ptr %1313)
// CHECK-NEXT:   %1314 = call %reflect.Value %__llgo_funcval_code174(ptr {{(nest|swiftself)}} %1312, %"{{.*}}/runtime/internal/runtime.eface" %1311)
// CHECK-NEXT:   %1315 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1307, i32 0, i32 1
// CHECK-NEXT:   %1316 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1317 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 8.700000e+01, ptr %1317, align 8
// CHECK-NEXT:   %1318 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1317, 1
// CHECK-NEXT:   %1319 = extractvalue { ptr, ptr } %1316, 1
// CHECK-NEXT:   %1320 = extractvalue { ptr, ptr } %1316, 0
// CHECK-NEXT:   %__llgo_funcval_code175 = call ptr asm "", "=r,0"(ptr %1320)
// CHECK-NEXT:   %1321 = call %reflect.Value %__llgo_funcval_code175(ptr {{(nest|swiftself)}} %1319, %"{{.*}}/runtime/internal/runtime.eface" %1318)
// CHECK-NEXT:   store %reflect.Value %1314, ptr %1308, align 8
// CHECK-NEXT:   store %reflect.Value %1321, ptr %1315, align 8
// CHECK-NEXT:   %1322 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 88
// CHECK-NEXT:   %1323 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1322, i32 0, i32 0
// CHECK-NEXT:   %1324 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1325 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 8.800000e+01, ptr %1325, align 8
// CHECK-NEXT:   %1326 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1325, 1
// CHECK-NEXT:   %1327 = extractvalue { ptr, ptr } %1324, 1
// CHECK-NEXT:   %1328 = extractvalue { ptr, ptr } %1324, 0
// CHECK-NEXT:   %__llgo_funcval_code176 = call ptr asm "", "=r,0"(ptr %1328)
// CHECK-NEXT:   %1329 = call %reflect.Value %__llgo_funcval_code176(ptr {{(nest|swiftself)}} %1327, %"{{.*}}/runtime/internal/runtime.eface" %1326)
// CHECK-NEXT:   %1330 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1322, i32 0, i32 1
// CHECK-NEXT:   %1331 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1332 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 88, ptr %1332, align 2
// CHECK-NEXT:   %1333 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %1332, 1
// CHECK-NEXT:   %1334 = extractvalue { ptr, ptr } %1331, 1
// CHECK-NEXT:   %1335 = extractvalue { ptr, ptr } %1331, 0
// CHECK-NEXT:   %__llgo_funcval_code177 = call ptr asm "", "=r,0"(ptr %1335)
// CHECK-NEXT:   %1336 = call %reflect.Value %__llgo_funcval_code177(ptr {{(nest|swiftself)}} %1334, %"{{.*}}/runtime/internal/runtime.eface" %1333)
// CHECK-NEXT:   store %reflect.Value %1329, ptr %1323, align 8
// CHECK-NEXT:   store %reflect.Value %1336, ptr %1330, align 8
// CHECK-NEXT:   %1337 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 89
// CHECK-NEXT:   %1338 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1337, i32 0, i32 0
// CHECK-NEXT:   %1339 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1340 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 89, ptr %1340, align 4
// CHECK-NEXT:   %1341 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1340, 1
// CHECK-NEXT:   %1342 = extractvalue { ptr, ptr } %1339, 1
// CHECK-NEXT:   %1343 = extractvalue { ptr, ptr } %1339, 0
// CHECK-NEXT:   %__llgo_funcval_code178 = call ptr asm "", "=r,0"(ptr %1343)
// CHECK-NEXT:   %1344 = call %reflect.Value %__llgo_funcval_code178(ptr {{(nest|swiftself)}} %1342, %"{{.*}}/runtime/internal/runtime.eface" %1341)
// CHECK-NEXT:   %1345 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1337, i32 0, i32 1
// CHECK-NEXT:   %1346 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1347 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 89, ptr %1347, align 4
// CHECK-NEXT:   %1348 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1347, 1
// CHECK-NEXT:   %1349 = extractvalue { ptr, ptr } %1346, 1
// CHECK-NEXT:   %1350 = extractvalue { ptr, ptr } %1346, 0
// CHECK-NEXT:   %__llgo_funcval_code179 = call ptr asm "", "=r,0"(ptr %1350)
// CHECK-NEXT:   %1351 = call %reflect.Value %__llgo_funcval_code179(ptr {{(nest|swiftself)}} %1349, %"{{.*}}/runtime/internal/runtime.eface" %1348)
// CHECK-NEXT:   store %reflect.Value %1344, ptr %1338, align 8
// CHECK-NEXT:   store %reflect.Value %1351, ptr %1345, align 8
// CHECK-NEXT:   %1352 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 90
// CHECK-NEXT:   %1353 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1352, i32 0, i32 0
// CHECK-NEXT:   %1354 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1355 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 90, ptr %1355, align 4
// CHECK-NEXT:   %1356 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1355, 1
// CHECK-NEXT:   %1357 = extractvalue { ptr, ptr } %1354, 1
// CHECK-NEXT:   %1358 = extractvalue { ptr, ptr } %1354, 0
// CHECK-NEXT:   %__llgo_funcval_code180 = call ptr asm "", "=r,0"(ptr %1358)
// CHECK-NEXT:   %1359 = call %reflect.Value %__llgo_funcval_code180(ptr {{(nest|swiftself)}} %1357, %"{{.*}}/runtime/internal/runtime.eface" %1356)
// CHECK-NEXT:   %1360 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1352, i32 0, i32 1
// CHECK-NEXT:   %1361 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1362 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 90, ptr %1362, align 4
// CHECK-NEXT:   %1363 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1362, 1
// CHECK-NEXT:   %1364 = extractvalue { ptr, ptr } %1361, 1
// CHECK-NEXT:   %1365 = extractvalue { ptr, ptr } %1361, 0
// CHECK-NEXT:   %__llgo_funcval_code181 = call ptr asm "", "=r,0"(ptr %1365)
// CHECK-NEXT:   %1366 = call %reflect.Value %__llgo_funcval_code181(ptr {{(nest|swiftself)}} %1364, %"{{.*}}/runtime/internal/runtime.eface" %1363)
// CHECK-NEXT:   store %reflect.Value %1359, ptr %1353, align 8
// CHECK-NEXT:   store %reflect.Value %1366, ptr %1360, align 8
// CHECK-NEXT:   %1367 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 91
// CHECK-NEXT:   %1368 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1367, i32 0, i32 0
// CHECK-NEXT:   %1369 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1370 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 91, ptr %1370, align 4
// CHECK-NEXT:   %1371 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1370, 1
// CHECK-NEXT:   %1372 = extractvalue { ptr, ptr } %1369, 1
// CHECK-NEXT:   %1373 = extractvalue { ptr, ptr } %1369, 0
// CHECK-NEXT:   %__llgo_funcval_code182 = call ptr asm "", "=r,0"(ptr %1373)
// CHECK-NEXT:   %1374 = call %reflect.Value %__llgo_funcval_code182(ptr {{(nest|swiftself)}} %1372, %"{{.*}}/runtime/internal/runtime.eface" %1371)
// CHECK-NEXT:   %1375 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1367, i32 0, i32 1
// CHECK-NEXT:   %1376 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1377 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 91, ptr %1377, align 4
// CHECK-NEXT:   %1378 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1377, 1
// CHECK-NEXT:   %1379 = extractvalue { ptr, ptr } %1376, 1
// CHECK-NEXT:   %1380 = extractvalue { ptr, ptr } %1376, 0
// CHECK-NEXT:   %__llgo_funcval_code183 = call ptr asm "", "=r,0"(ptr %1380)
// CHECK-NEXT:   %1381 = call %reflect.Value %__llgo_funcval_code183(ptr {{(nest|swiftself)}} %1379, %"{{.*}}/runtime/internal/runtime.eface" %1378)
// CHECK-NEXT:   store %reflect.Value %1374, ptr %1368, align 8
// CHECK-NEXT:   store %reflect.Value %1381, ptr %1375, align 8
// CHECK-NEXT:   %1382 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 92
// CHECK-NEXT:   %1383 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1382, i32 0, i32 0
// CHECK-NEXT:   %1384 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1385 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 92, ptr %1385, align 4
// CHECK-NEXT:   %1386 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1385, 1
// CHECK-NEXT:   %1387 = extractvalue { ptr, ptr } %1384, 1
// CHECK-NEXT:   %1388 = extractvalue { ptr, ptr } %1384, 0
// CHECK-NEXT:   %__llgo_funcval_code184 = call ptr asm "", "=r,0"(ptr %1388)
// CHECK-NEXT:   %1389 = call %reflect.Value %__llgo_funcval_code184(ptr {{(nest|swiftself)}} %1387, %"{{.*}}/runtime/internal/runtime.eface" %1386)
// CHECK-NEXT:   %1390 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1382, i32 0, i32 1
// CHECK-NEXT:   %1391 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1392 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 92, ptr %1392, align 8
// CHECK-NEXT:   %1393 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1392, 1
// CHECK-NEXT:   %1394 = extractvalue { ptr, ptr } %1391, 1
// CHECK-NEXT:   %1395 = extractvalue { ptr, ptr } %1391, 0
// CHECK-NEXT:   %__llgo_funcval_code185 = call ptr asm "", "=r,0"(ptr %1395)
// CHECK-NEXT:   %1396 = call %reflect.Value %__llgo_funcval_code185(ptr {{(nest|swiftself)}} %1394, %"{{.*}}/runtime/internal/runtime.eface" %1393)
// CHECK-NEXT:   store %reflect.Value %1389, ptr %1383, align 8
// CHECK-NEXT:   store %reflect.Value %1396, ptr %1390, align 8
// CHECK-NEXT:   %1397 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 93
// CHECK-NEXT:   %1398 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1397, i32 0, i32 0
// CHECK-NEXT:   %1399 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1400 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 93, ptr %1400, align 8
// CHECK-NEXT:   %1401 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1400, 1
// CHECK-NEXT:   %1402 = extractvalue { ptr, ptr } %1399, 1
// CHECK-NEXT:   %1403 = extractvalue { ptr, ptr } %1399, 0
// CHECK-NEXT:   %__llgo_funcval_code186 = call ptr asm "", "=r,0"(ptr %1403)
// CHECK-NEXT:   %1404 = call %reflect.Value %__llgo_funcval_code186(ptr {{(nest|swiftself)}} %1402, %"{{.*}}/runtime/internal/runtime.eface" %1401)
// CHECK-NEXT:   %1405 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1397, i32 0, i32 1
// CHECK-NEXT:   %1406 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1407 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 93, ptr %1407, align 4
// CHECK-NEXT:   %1408 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1407, 1
// CHECK-NEXT:   %1409 = extractvalue { ptr, ptr } %1406, 1
// CHECK-NEXT:   %1410 = extractvalue { ptr, ptr } %1406, 0
// CHECK-NEXT:   %__llgo_funcval_code187 = call ptr asm "", "=r,0"(ptr %1410)
// CHECK-NEXT:   %1411 = call %reflect.Value %__llgo_funcval_code187(ptr {{(nest|swiftself)}} %1409, %"{{.*}}/runtime/internal/runtime.eface" %1408)
// CHECK-NEXT:   store %reflect.Value %1404, ptr %1398, align 8
// CHECK-NEXT:   store %reflect.Value %1411, ptr %1405, align 8
// CHECK-NEXT:   %1412 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 94
// CHECK-NEXT:   %1413 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1412, i32 0, i32 0
// CHECK-NEXT:   %1414 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1415 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 94, ptr %1415, align 4
// CHECK-NEXT:   %1416 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1415, 1
// CHECK-NEXT:   %1417 = extractvalue { ptr, ptr } %1414, 1
// CHECK-NEXT:   %1418 = extractvalue { ptr, ptr } %1414, 0
// CHECK-NEXT:   %__llgo_funcval_code188 = call ptr asm "", "=r,0"(ptr %1418)
// CHECK-NEXT:   %1419 = call %reflect.Value %__llgo_funcval_code188(ptr {{(nest|swiftself)}} %1417, %"{{.*}}/runtime/internal/runtime.eface" %1416)
// CHECK-NEXT:   %1420 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1412, i32 0, i32 1
// CHECK-NEXT:   %1421 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1422 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 94, ptr %1422, align 8
// CHECK-NEXT:   %1423 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1422, 1
// CHECK-NEXT:   %1424 = extractvalue { ptr, ptr } %1421, 1
// CHECK-NEXT:   %1425 = extractvalue { ptr, ptr } %1421, 0
// CHECK-NEXT:   %__llgo_funcval_code189 = call ptr asm "", "=r,0"(ptr %1425)
// CHECK-NEXT:   %1426 = call %reflect.Value %__llgo_funcval_code189(ptr {{(nest|swiftself)}} %1424, %"{{.*}}/runtime/internal/runtime.eface" %1423)
// CHECK-NEXT:   store %reflect.Value %1419, ptr %1413, align 8
// CHECK-NEXT:   store %reflect.Value %1426, ptr %1420, align 8
// CHECK-NEXT:   %1427 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 95
// CHECK-NEXT:   %1428 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1427, i32 0, i32 0
// CHECK-NEXT:   %1429 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1430 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 95, ptr %1430, align 8
// CHECK-NEXT:   %1431 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1430, 1
// CHECK-NEXT:   %1432 = extractvalue { ptr, ptr } %1429, 1
// CHECK-NEXT:   %1433 = extractvalue { ptr, ptr } %1429, 0
// CHECK-NEXT:   %__llgo_funcval_code190 = call ptr asm "", "=r,0"(ptr %1433)
// CHECK-NEXT:   %1434 = call %reflect.Value %__llgo_funcval_code190(ptr {{(nest|swiftself)}} %1432, %"{{.*}}/runtime/internal/runtime.eface" %1431)
// CHECK-NEXT:   %1435 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1427, i32 0, i32 1
// CHECK-NEXT:   %1436 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1437 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 95, ptr %1437, align 4
// CHECK-NEXT:   %1438 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1437, 1
// CHECK-NEXT:   %1439 = extractvalue { ptr, ptr } %1436, 1
// CHECK-NEXT:   %1440 = extractvalue { ptr, ptr } %1436, 0
// CHECK-NEXT:   %__llgo_funcval_code191 = call ptr asm "", "=r,0"(ptr %1440)
// CHECK-NEXT:   %1441 = call %reflect.Value %__llgo_funcval_code191(ptr {{(nest|swiftself)}} %1439, %"{{.*}}/runtime/internal/runtime.eface" %1438)
// CHECK-NEXT:   store %reflect.Value %1434, ptr %1428, align 8
// CHECK-NEXT:   store %reflect.Value %1441, ptr %1435, align 8
// CHECK-NEXT:   %1442 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 96
// CHECK-NEXT:   %1443 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1442, i32 0, i32 0
// CHECK-NEXT:   %1444 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1445 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 96, ptr %1445, align 4
// CHECK-NEXT:   %1446 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1445, 1
// CHECK-NEXT:   %1447 = extractvalue { ptr, ptr } %1444, 1
// CHECK-NEXT:   %1448 = extractvalue { ptr, ptr } %1444, 0
// CHECK-NEXT:   %__llgo_funcval_code192 = call ptr asm "", "=r,0"(ptr %1448)
// CHECK-NEXT:   %1449 = call %reflect.Value %__llgo_funcval_code192(ptr {{(nest|swiftself)}} %1447, %"{{.*}}/runtime/internal/runtime.eface" %1446)
// CHECK-NEXT:   %1450 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1442, i32 0, i32 1
// CHECK-NEXT:   %1451 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1452 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 96, ptr %1452, align 8
// CHECK-NEXT:   %1453 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1452, 1
// CHECK-NEXT:   %1454 = extractvalue { ptr, ptr } %1451, 1
// CHECK-NEXT:   %1455 = extractvalue { ptr, ptr } %1451, 0
// CHECK-NEXT:   %__llgo_funcval_code193 = call ptr asm "", "=r,0"(ptr %1455)
// CHECK-NEXT:   %1456 = call %reflect.Value %__llgo_funcval_code193(ptr {{(nest|swiftself)}} %1454, %"{{.*}}/runtime/internal/runtime.eface" %1453)
// CHECK-NEXT:   store %reflect.Value %1449, ptr %1443, align 8
// CHECK-NEXT:   store %reflect.Value %1456, ptr %1450, align 8
// CHECK-NEXT:   %1457 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 97
// CHECK-NEXT:   %1458 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1457, i32 0, i32 0
// CHECK-NEXT:   %1459 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1460 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %1460, align 8
// CHECK-NEXT:   %1461 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1460, 1
// CHECK-NEXT:   %1462 = extractvalue { ptr, ptr } %1459, 1
// CHECK-NEXT:   %1463 = extractvalue { ptr, ptr } %1459, 0
// CHECK-NEXT:   %__llgo_funcval_code194 = call ptr asm "", "=r,0"(ptr %1463)
// CHECK-NEXT:   %1464 = call %reflect.Value %__llgo_funcval_code194(ptr {{(nest|swiftself)}} %1462, %"{{.*}}/runtime/internal/runtime.eface" %1461)
// CHECK-NEXT:   %1465 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1457, i32 0, i32 1
// CHECK-NEXT:   %1466 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1467 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %1467, align 4
// CHECK-NEXT:   %1468 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1467, 1
// CHECK-NEXT:   %1469 = extractvalue { ptr, ptr } %1466, 1
// CHECK-NEXT:   %1470 = extractvalue { ptr, ptr } %1466, 0
// CHECK-NEXT:   %__llgo_funcval_code195 = call ptr asm "", "=r,0"(ptr %1470)
// CHECK-NEXT:   %1471 = call %reflect.Value %__llgo_funcval_code195(ptr {{(nest|swiftself)}} %1469, %"{{.*}}/runtime/internal/runtime.eface" %1468)
// CHECK-NEXT:   store %reflect.Value %1464, ptr %1458, align 8
// CHECK-NEXT:   store %reflect.Value %1471, ptr %1465, align 8
// CHECK-NEXT:   %1472 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 98
// CHECK-NEXT:   %1473 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1472, i32 0, i32 0
// CHECK-NEXT:   %1474 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1475 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 98, ptr %1475, align 4
// CHECK-NEXT:   %1476 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1475, 1
// CHECK-NEXT:   %1477 = extractvalue { ptr, ptr } %1474, 1
// CHECK-NEXT:   %1478 = extractvalue { ptr, ptr } %1474, 0
// CHECK-NEXT:   %__llgo_funcval_code196 = call ptr asm "", "=r,0"(ptr %1478)
// CHECK-NEXT:   %1479 = call %reflect.Value %__llgo_funcval_code196(ptr {{(nest|swiftself)}} %1477, %"{{.*}}/runtime/internal/runtime.eface" %1476)
// CHECK-NEXT:   %1480 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1472, i32 0, i32 1
// CHECK-NEXT:   %1481 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1482 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 98, ptr %1482, align 8
// CHECK-NEXT:   %1483 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1482, 1
// CHECK-NEXT:   %1484 = extractvalue { ptr, ptr } %1481, 1
// CHECK-NEXT:   %1485 = extractvalue { ptr, ptr } %1481, 0
// CHECK-NEXT:   %__llgo_funcval_code197 = call ptr asm "", "=r,0"(ptr %1485)
// CHECK-NEXT:   %1486 = call %reflect.Value %__llgo_funcval_code197(ptr {{(nest|swiftself)}} %1484, %"{{.*}}/runtime/internal/runtime.eface" %1483)
// CHECK-NEXT:   store %reflect.Value %1479, ptr %1473, align 8
// CHECK-NEXT:   store %reflect.Value %1486, ptr %1480, align 8
// CHECK-NEXT:   %1487 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 99
// CHECK-NEXT:   %1488 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1487, i32 0, i32 0
// CHECK-NEXT:   %1489 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1490 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 99, ptr %1490, align 8
// CHECK-NEXT:   %1491 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1490, 1
// CHECK-NEXT:   %1492 = extractvalue { ptr, ptr } %1489, 1
// CHECK-NEXT:   %1493 = extractvalue { ptr, ptr } %1489, 0
// CHECK-NEXT:   %__llgo_funcval_code198 = call ptr asm "", "=r,0"(ptr %1493)
// CHECK-NEXT:   %1494 = call %reflect.Value %__llgo_funcval_code198(ptr {{(nest|swiftself)}} %1492, %"{{.*}}/runtime/internal/runtime.eface" %1491)
// CHECK-NEXT:   %1495 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1487, i32 0, i32 1
// CHECK-NEXT:   %1496 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1497 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 99, ptr %1497, align 4
// CHECK-NEXT:   %1498 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1497, 1
// CHECK-NEXT:   %1499 = extractvalue { ptr, ptr } %1496, 1
// CHECK-NEXT:   %1500 = extractvalue { ptr, ptr } %1496, 0
// CHECK-NEXT:   %__llgo_funcval_code199 = call ptr asm "", "=r,0"(ptr %1500)
// CHECK-NEXT:   %1501 = call %reflect.Value %__llgo_funcval_code199(ptr {{(nest|swiftself)}} %1499, %"{{.*}}/runtime/internal/runtime.eface" %1498)
// CHECK-NEXT:   store %reflect.Value %1494, ptr %1488, align 8
// CHECK-NEXT:   store %reflect.Value %1501, ptr %1495, align 8
// CHECK-NEXT:   %1502 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 100
// CHECK-NEXT:   %1503 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1502, i32 0, i32 0
// CHECK-NEXT:   %1504 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1505 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 100, ptr %1505, align 4
// CHECK-NEXT:   %1506 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1505, 1
// CHECK-NEXT:   %1507 = extractvalue { ptr, ptr } %1504, 1
// CHECK-NEXT:   %1508 = extractvalue { ptr, ptr } %1504, 0
// CHECK-NEXT:   %__llgo_funcval_code200 = call ptr asm "", "=r,0"(ptr %1508)
// CHECK-NEXT:   %1509 = call %reflect.Value %__llgo_funcval_code200(ptr {{(nest|swiftself)}} %1507, %"{{.*}}/runtime/internal/runtime.eface" %1506)
// CHECK-NEXT:   %1510 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1502, i32 0, i32 1
// CHECK-NEXT:   %1511 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1512 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 100, ptr %1512, align 8
// CHECK-NEXT:   %1513 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1512, 1
// CHECK-NEXT:   %1514 = extractvalue { ptr, ptr } %1511, 1
// CHECK-NEXT:   %1515 = extractvalue { ptr, ptr } %1511, 0
// CHECK-NEXT:   %__llgo_funcval_code201 = call ptr asm "", "=r,0"(ptr %1515)
// CHECK-NEXT:   %1516 = call %reflect.Value %__llgo_funcval_code201(ptr {{(nest|swiftself)}} %1514, %"{{.*}}/runtime/internal/runtime.eface" %1513)
// CHECK-NEXT:   store %reflect.Value %1509, ptr %1503, align 8
// CHECK-NEXT:   store %reflect.Value %1516, ptr %1510, align 8
// CHECK-NEXT:   %1517 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 101
// CHECK-NEXT:   %1518 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1517, i32 0, i32 0
// CHECK-NEXT:   %1519 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1520 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 101, ptr %1520, align 8
// CHECK-NEXT:   %1521 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1520, 1
// CHECK-NEXT:   %1522 = extractvalue { ptr, ptr } %1519, 1
// CHECK-NEXT:   %1523 = extractvalue { ptr, ptr } %1519, 0
// CHECK-NEXT:   %__llgo_funcval_code202 = call ptr asm "", "=r,0"(ptr %1523)
// CHECK-NEXT:   %1524 = call %reflect.Value %__llgo_funcval_code202(ptr {{(nest|swiftself)}} %1522, %"{{.*}}/runtime/internal/runtime.eface" %1521)
// CHECK-NEXT:   %1525 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1517, i32 0, i32 1
// CHECK-NEXT:   %1526 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1527 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 101, ptr %1527, align 4
// CHECK-NEXT:   %1528 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1527, 1
// CHECK-NEXT:   %1529 = extractvalue { ptr, ptr } %1526, 1
// CHECK-NEXT:   %1530 = extractvalue { ptr, ptr } %1526, 0
// CHECK-NEXT:   %__llgo_funcval_code203 = call ptr asm "", "=r,0"(ptr %1530)
// CHECK-NEXT:   %1531 = call %reflect.Value %__llgo_funcval_code203(ptr {{(nest|swiftself)}} %1529, %"{{.*}}/runtime/internal/runtime.eface" %1528)
// CHECK-NEXT:   store %reflect.Value %1524, ptr %1518, align 8
// CHECK-NEXT:   store %reflect.Value %1531, ptr %1525, align 8
// CHECK-NEXT:   %1532 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 102
// CHECK-NEXT:   %1533 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1532, i32 0, i32 0
// CHECK-NEXT:   %1534 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1535 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 102, ptr %1535, align 4
// CHECK-NEXT:   %1536 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1535, 1
// CHECK-NEXT:   %1537 = extractvalue { ptr, ptr } %1534, 1
// CHECK-NEXT:   %1538 = extractvalue { ptr, ptr } %1534, 0
// CHECK-NEXT:   %__llgo_funcval_code204 = call ptr asm "", "=r,0"(ptr %1538)
// CHECK-NEXT:   %1539 = call %reflect.Value %__llgo_funcval_code204(ptr {{(nest|swiftself)}} %1537, %"{{.*}}/runtime/internal/runtime.eface" %1536)
// CHECK-NEXT:   %1540 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1532, i32 0, i32 1
// CHECK-NEXT:   %1541 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1542 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.020000e+02, ptr %1542, align 4
// CHECK-NEXT:   %1543 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1542, 1
// CHECK-NEXT:   %1544 = extractvalue { ptr, ptr } %1541, 1
// CHECK-NEXT:   %1545 = extractvalue { ptr, ptr } %1541, 0
// CHECK-NEXT:   %__llgo_funcval_code205 = call ptr asm "", "=r,0"(ptr %1545)
// CHECK-NEXT:   %1546 = call %reflect.Value %__llgo_funcval_code205(ptr {{(nest|swiftself)}} %1544, %"{{.*}}/runtime/internal/runtime.eface" %1543)
// CHECK-NEXT:   store %reflect.Value %1539, ptr %1533, align 8
// CHECK-NEXT:   store %reflect.Value %1546, ptr %1540, align 8
// CHECK-NEXT:   %1547 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 103
// CHECK-NEXT:   %1548 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1547, i32 0, i32 0
// CHECK-NEXT:   %1549 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1550 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.030000e+02, ptr %1550, align 4
// CHECK-NEXT:   %1551 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1550, 1
// CHECK-NEXT:   %1552 = extractvalue { ptr, ptr } %1549, 1
// CHECK-NEXT:   %1553 = extractvalue { ptr, ptr } %1549, 0
// CHECK-NEXT:   %__llgo_funcval_code206 = call ptr asm "", "=r,0"(ptr %1553)
// CHECK-NEXT:   %1554 = call %reflect.Value %__llgo_funcval_code206(ptr {{(nest|swiftself)}} %1552, %"{{.*}}/runtime/internal/runtime.eface" %1551)
// CHECK-NEXT:   %1555 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1547, i32 0, i32 1
// CHECK-NEXT:   %1556 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1557 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 103, ptr %1557, align 4
// CHECK-NEXT:   %1558 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1557, 1
// CHECK-NEXT:   %1559 = extractvalue { ptr, ptr } %1556, 1
// CHECK-NEXT:   %1560 = extractvalue { ptr, ptr } %1556, 0
// CHECK-NEXT:   %__llgo_funcval_code207 = call ptr asm "", "=r,0"(ptr %1560)
// CHECK-NEXT:   %1561 = call %reflect.Value %__llgo_funcval_code207(ptr {{(nest|swiftself)}} %1559, %"{{.*}}/runtime/internal/runtime.eface" %1558)
// CHECK-NEXT:   store %reflect.Value %1554, ptr %1548, align 8
// CHECK-NEXT:   store %reflect.Value %1561, ptr %1555, align 8
// CHECK-NEXT:   %1562 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 104
// CHECK-NEXT:   %1563 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1562, i32 0, i32 0
// CHECK-NEXT:   %1564 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1565 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 104, ptr %1565, align 4
// CHECK-NEXT:   %1566 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1565, 1
// CHECK-NEXT:   %1567 = extractvalue { ptr, ptr } %1564, 1
// CHECK-NEXT:   %1568 = extractvalue { ptr, ptr } %1564, 0
// CHECK-NEXT:   %__llgo_funcval_code208 = call ptr asm "", "=r,0"(ptr %1568)
// CHECK-NEXT:   %1569 = call %reflect.Value %__llgo_funcval_code208(ptr {{(nest|swiftself)}} %1567, %"{{.*}}/runtime/internal/runtime.eface" %1566)
// CHECK-NEXT:   %1570 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1562, i32 0, i32 1
// CHECK-NEXT:   %1571 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1572 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.040000e+02, ptr %1572, align 8
// CHECK-NEXT:   %1573 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1572, 1
// CHECK-NEXT:   %1574 = extractvalue { ptr, ptr } %1571, 1
// CHECK-NEXT:   %1575 = extractvalue { ptr, ptr } %1571, 0
// CHECK-NEXT:   %__llgo_funcval_code209 = call ptr asm "", "=r,0"(ptr %1575)
// CHECK-NEXT:   %1576 = call %reflect.Value %__llgo_funcval_code209(ptr {{(nest|swiftself)}} %1574, %"{{.*}}/runtime/internal/runtime.eface" %1573)
// CHECK-NEXT:   store %reflect.Value %1569, ptr %1563, align 8
// CHECK-NEXT:   store %reflect.Value %1576, ptr %1570, align 8
// CHECK-NEXT:   %1577 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 105
// CHECK-NEXT:   %1578 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1577, i32 0, i32 0
// CHECK-NEXT:   %1579 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1580 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.050000e+02, ptr %1580, align 8
// CHECK-NEXT:   %1581 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1580, 1
// CHECK-NEXT:   %1582 = extractvalue { ptr, ptr } %1579, 1
// CHECK-NEXT:   %1583 = extractvalue { ptr, ptr } %1579, 0
// CHECK-NEXT:   %__llgo_funcval_code210 = call ptr asm "", "=r,0"(ptr %1583)
// CHECK-NEXT:   %1584 = call %reflect.Value %__llgo_funcval_code210(ptr {{(nest|swiftself)}} %1582, %"{{.*}}/runtime/internal/runtime.eface" %1581)
// CHECK-NEXT:   %1585 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1577, i32 0, i32 1
// CHECK-NEXT:   %1586 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1587 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 105, ptr %1587, align 4
// CHECK-NEXT:   %1588 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %1587, 1
// CHECK-NEXT:   %1589 = extractvalue { ptr, ptr } %1586, 1
// CHECK-NEXT:   %1590 = extractvalue { ptr, ptr } %1586, 0
// CHECK-NEXT:   %__llgo_funcval_code211 = call ptr asm "", "=r,0"(ptr %1590)
// CHECK-NEXT:   %1591 = call %reflect.Value %__llgo_funcval_code211(ptr {{(nest|swiftself)}} %1589, %"{{.*}}/runtime/internal/runtime.eface" %1588)
// CHECK-NEXT:   store %reflect.Value %1584, ptr %1578, align 8
// CHECK-NEXT:   store %reflect.Value %1591, ptr %1585, align 8
// CHECK-NEXT:   %1592 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 106
// CHECK-NEXT:   %1593 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1592, i32 0, i32 0
// CHECK-NEXT:   %1594 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1595 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 106, ptr %1595, align 4
// CHECK-NEXT:   %1596 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1595, 1
// CHECK-NEXT:   %1597 = extractvalue { ptr, ptr } %1594, 1
// CHECK-NEXT:   %1598 = extractvalue { ptr, ptr } %1594, 0
// CHECK-NEXT:   %__llgo_funcval_code212 = call ptr asm "", "=r,0"(ptr %1598)
// CHECK-NEXT:   %1599 = call %reflect.Value %__llgo_funcval_code212(ptr {{(nest|swiftself)}} %1597, %"{{.*}}/runtime/internal/runtime.eface" %1596)
// CHECK-NEXT:   %1600 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1592, i32 0, i32 1
// CHECK-NEXT:   %1601 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1602 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 106, ptr %1602, align 4
// CHECK-NEXT:   %1603 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1602, 1
// CHECK-NEXT:   %1604 = extractvalue { ptr, ptr } %1601, 1
// CHECK-NEXT:   %1605 = extractvalue { ptr, ptr } %1601, 0
// CHECK-NEXT:   %__llgo_funcval_code213 = call ptr asm "", "=r,0"(ptr %1605)
// CHECK-NEXT:   %1606 = call %reflect.Value %__llgo_funcval_code213(ptr {{(nest|swiftself)}} %1604, %"{{.*}}/runtime/internal/runtime.eface" %1603)
// CHECK-NEXT:   store %reflect.Value %1599, ptr %1593, align 8
// CHECK-NEXT:   store %reflect.Value %1606, ptr %1600, align 8
// CHECK-NEXT:   %1607 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 107
// CHECK-NEXT:   %1608 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1607, i32 0, i32 0
// CHECK-NEXT:   %1609 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1610 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 107, ptr %1610, align 4
// CHECK-NEXT:   %1611 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1610, 1
// CHECK-NEXT:   %1612 = extractvalue { ptr, ptr } %1609, 1
// CHECK-NEXT:   %1613 = extractvalue { ptr, ptr } %1609, 0
// CHECK-NEXT:   %__llgo_funcval_code214 = call ptr asm "", "=r,0"(ptr %1613)
// CHECK-NEXT:   %1614 = call %reflect.Value %__llgo_funcval_code214(ptr {{(nest|swiftself)}} %1612, %"{{.*}}/runtime/internal/runtime.eface" %1611)
// CHECK-NEXT:   %1615 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1607, i32 0, i32 1
// CHECK-NEXT:   %1616 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1617 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 107, ptr %1617, align 8
// CHECK-NEXT:   %1618 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1617, 1
// CHECK-NEXT:   %1619 = extractvalue { ptr, ptr } %1616, 1
// CHECK-NEXT:   %1620 = extractvalue { ptr, ptr } %1616, 0
// CHECK-NEXT:   %__llgo_funcval_code215 = call ptr asm "", "=r,0"(ptr %1620)
// CHECK-NEXT:   %1621 = call %reflect.Value %__llgo_funcval_code215(ptr {{(nest|swiftself)}} %1619, %"{{.*}}/runtime/internal/runtime.eface" %1618)
// CHECK-NEXT:   store %reflect.Value %1614, ptr %1608, align 8
// CHECK-NEXT:   store %reflect.Value %1621, ptr %1615, align 8
// CHECK-NEXT:   %1622 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 108
// CHECK-NEXT:   %1623 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1622, i32 0, i32 0
// CHECK-NEXT:   %1624 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1625 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 108, ptr %1625, align 8
// CHECK-NEXT:   %1626 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1625, 1
// CHECK-NEXT:   %1627 = extractvalue { ptr, ptr } %1624, 1
// CHECK-NEXT:   %1628 = extractvalue { ptr, ptr } %1624, 0
// CHECK-NEXT:   %__llgo_funcval_code216 = call ptr asm "", "=r,0"(ptr %1628)
// CHECK-NEXT:   %1629 = call %reflect.Value %__llgo_funcval_code216(ptr {{(nest|swiftself)}} %1627, %"{{.*}}/runtime/internal/runtime.eface" %1626)
// CHECK-NEXT:   %1630 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1622, i32 0, i32 1
// CHECK-NEXT:   %1631 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1632 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 108, ptr %1632, align 4
// CHECK-NEXT:   %1633 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1632, 1
// CHECK-NEXT:   %1634 = extractvalue { ptr, ptr } %1631, 1
// CHECK-NEXT:   %1635 = extractvalue { ptr, ptr } %1631, 0
// CHECK-NEXT:   %__llgo_funcval_code217 = call ptr asm "", "=r,0"(ptr %1635)
// CHECK-NEXT:   %1636 = call %reflect.Value %__llgo_funcval_code217(ptr {{(nest|swiftself)}} %1634, %"{{.*}}/runtime/internal/runtime.eface" %1633)
// CHECK-NEXT:   store %reflect.Value %1629, ptr %1623, align 8
// CHECK-NEXT:   store %reflect.Value %1636, ptr %1630, align 8
// CHECK-NEXT:   %1637 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 109
// CHECK-NEXT:   %1638 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1637, i32 0, i32 0
// CHECK-NEXT:   %1639 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1640 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 109, ptr %1640, align 4
// CHECK-NEXT:   %1641 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1640, 1
// CHECK-NEXT:   %1642 = extractvalue { ptr, ptr } %1639, 1
// CHECK-NEXT:   %1643 = extractvalue { ptr, ptr } %1639, 0
// CHECK-NEXT:   %__llgo_funcval_code218 = call ptr asm "", "=r,0"(ptr %1643)
// CHECK-NEXT:   %1644 = call %reflect.Value %__llgo_funcval_code218(ptr {{(nest|swiftself)}} %1642, %"{{.*}}/runtime/internal/runtime.eface" %1641)
// CHECK-NEXT:   %1645 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1637, i32 0, i32 1
// CHECK-NEXT:   %1646 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1647 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 109, ptr %1647, align 8
// CHECK-NEXT:   %1648 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1647, 1
// CHECK-NEXT:   %1649 = extractvalue { ptr, ptr } %1646, 1
// CHECK-NEXT:   %1650 = extractvalue { ptr, ptr } %1646, 0
// CHECK-NEXT:   %__llgo_funcval_code219 = call ptr asm "", "=r,0"(ptr %1650)
// CHECK-NEXT:   %1651 = call %reflect.Value %__llgo_funcval_code219(ptr {{(nest|swiftself)}} %1649, %"{{.*}}/runtime/internal/runtime.eface" %1648)
// CHECK-NEXT:   store %reflect.Value %1644, ptr %1638, align 8
// CHECK-NEXT:   store %reflect.Value %1651, ptr %1645, align 8
// CHECK-NEXT:   %1652 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 110
// CHECK-NEXT:   %1653 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1652, i32 0, i32 0
// CHECK-NEXT:   %1654 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1655 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 110, ptr %1655, align 8
// CHECK-NEXT:   %1656 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1655, 1
// CHECK-NEXT:   %1657 = extractvalue { ptr, ptr } %1654, 1
// CHECK-NEXT:   %1658 = extractvalue { ptr, ptr } %1654, 0
// CHECK-NEXT:   %__llgo_funcval_code220 = call ptr asm "", "=r,0"(ptr %1658)
// CHECK-NEXT:   %1659 = call %reflect.Value %__llgo_funcval_code220(ptr {{(nest|swiftself)}} %1657, %"{{.*}}/runtime/internal/runtime.eface" %1656)
// CHECK-NEXT:   %1660 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1652, i32 0, i32 1
// CHECK-NEXT:   %1661 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1662 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 110, ptr %1662, align 4
// CHECK-NEXT:   %1663 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1662, 1
// CHECK-NEXT:   %1664 = extractvalue { ptr, ptr } %1661, 1
// CHECK-NEXT:   %1665 = extractvalue { ptr, ptr } %1661, 0
// CHECK-NEXT:   %__llgo_funcval_code221 = call ptr asm "", "=r,0"(ptr %1665)
// CHECK-NEXT:   %1666 = call %reflect.Value %__llgo_funcval_code221(ptr {{(nest|swiftself)}} %1664, %"{{.*}}/runtime/internal/runtime.eface" %1663)
// CHECK-NEXT:   store %reflect.Value %1659, ptr %1653, align 8
// CHECK-NEXT:   store %reflect.Value %1666, ptr %1660, align 8
// CHECK-NEXT:   %1667 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 111
// CHECK-NEXT:   %1668 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1667, i32 0, i32 0
// CHECK-NEXT:   %1669 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1670 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 111, ptr %1670, align 4
// CHECK-NEXT:   %1671 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1670, 1
// CHECK-NEXT:   %1672 = extractvalue { ptr, ptr } %1669, 1
// CHECK-NEXT:   %1673 = extractvalue { ptr, ptr } %1669, 0
// CHECK-NEXT:   %__llgo_funcval_code222 = call ptr asm "", "=r,0"(ptr %1673)
// CHECK-NEXT:   %1674 = call %reflect.Value %__llgo_funcval_code222(ptr {{(nest|swiftself)}} %1672, %"{{.*}}/runtime/internal/runtime.eface" %1671)
// CHECK-NEXT:   %1675 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1667, i32 0, i32 1
// CHECK-NEXT:   %1676 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1677 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 111, ptr %1677, align 8
// CHECK-NEXT:   %1678 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1677, 1
// CHECK-NEXT:   %1679 = extractvalue { ptr, ptr } %1676, 1
// CHECK-NEXT:   %1680 = extractvalue { ptr, ptr } %1676, 0
// CHECK-NEXT:   %__llgo_funcval_code223 = call ptr asm "", "=r,0"(ptr %1680)
// CHECK-NEXT:   %1681 = call %reflect.Value %__llgo_funcval_code223(ptr {{(nest|swiftself)}} %1679, %"{{.*}}/runtime/internal/runtime.eface" %1678)
// CHECK-NEXT:   store %reflect.Value %1674, ptr %1668, align 8
// CHECK-NEXT:   store %reflect.Value %1681, ptr %1675, align 8
// CHECK-NEXT:   %1682 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 112
// CHECK-NEXT:   %1683 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1682, i32 0, i32 0
// CHECK-NEXT:   %1684 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1685 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 112, ptr %1685, align 8
// CHECK-NEXT:   %1686 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1685, 1
// CHECK-NEXT:   %1687 = extractvalue { ptr, ptr } %1684, 1
// CHECK-NEXT:   %1688 = extractvalue { ptr, ptr } %1684, 0
// CHECK-NEXT:   %__llgo_funcval_code224 = call ptr asm "", "=r,0"(ptr %1688)
// CHECK-NEXT:   %1689 = call %reflect.Value %__llgo_funcval_code224(ptr {{(nest|swiftself)}} %1687, %"{{.*}}/runtime/internal/runtime.eface" %1686)
// CHECK-NEXT:   %1690 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1682, i32 0, i32 1
// CHECK-NEXT:   %1691 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1692 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 112, ptr %1692, align 4
// CHECK-NEXT:   %1693 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1692, 1
// CHECK-NEXT:   %1694 = extractvalue { ptr, ptr } %1691, 1
// CHECK-NEXT:   %1695 = extractvalue { ptr, ptr } %1691, 0
// CHECK-NEXT:   %__llgo_funcval_code225 = call ptr asm "", "=r,0"(ptr %1695)
// CHECK-NEXT:   %1696 = call %reflect.Value %__llgo_funcval_code225(ptr {{(nest|swiftself)}} %1694, %"{{.*}}/runtime/internal/runtime.eface" %1693)
// CHECK-NEXT:   store %reflect.Value %1689, ptr %1683, align 8
// CHECK-NEXT:   store %reflect.Value %1696, ptr %1690, align 8
// CHECK-NEXT:   %1697 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 113
// CHECK-NEXT:   %1698 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1697, i32 0, i32 0
// CHECK-NEXT:   %1699 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1700 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 113, ptr %1700, align 4
// CHECK-NEXT:   %1701 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1700, 1
// CHECK-NEXT:   %1702 = extractvalue { ptr, ptr } %1699, 1
// CHECK-NEXT:   %1703 = extractvalue { ptr, ptr } %1699, 0
// CHECK-NEXT:   %__llgo_funcval_code226 = call ptr asm "", "=r,0"(ptr %1703)
// CHECK-NEXT:   %1704 = call %reflect.Value %__llgo_funcval_code226(ptr {{(nest|swiftself)}} %1702, %"{{.*}}/runtime/internal/runtime.eface" %1701)
// CHECK-NEXT:   %1705 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1697, i32 0, i32 1
// CHECK-NEXT:   %1706 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1707 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 113, ptr %1707, align 8
// CHECK-NEXT:   %1708 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1707, 1
// CHECK-NEXT:   %1709 = extractvalue { ptr, ptr } %1706, 1
// CHECK-NEXT:   %1710 = extractvalue { ptr, ptr } %1706, 0
// CHECK-NEXT:   %__llgo_funcval_code227 = call ptr asm "", "=r,0"(ptr %1710)
// CHECK-NEXT:   %1711 = call %reflect.Value %__llgo_funcval_code227(ptr {{(nest|swiftself)}} %1709, %"{{.*}}/runtime/internal/runtime.eface" %1708)
// CHECK-NEXT:   store %reflect.Value %1704, ptr %1698, align 8
// CHECK-NEXT:   store %reflect.Value %1711, ptr %1705, align 8
// CHECK-NEXT:   %1712 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 114
// CHECK-NEXT:   %1713 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1712, i32 0, i32 0
// CHECK-NEXT:   %1714 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1715 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 114, ptr %1715, align 8
// CHECK-NEXT:   %1716 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1715, 1
// CHECK-NEXT:   %1717 = extractvalue { ptr, ptr } %1714, 1
// CHECK-NEXT:   %1718 = extractvalue { ptr, ptr } %1714, 0
// CHECK-NEXT:   %__llgo_funcval_code228 = call ptr asm "", "=r,0"(ptr %1718)
// CHECK-NEXT:   %1719 = call %reflect.Value %__llgo_funcval_code228(ptr {{(nest|swiftself)}} %1717, %"{{.*}}/runtime/internal/runtime.eface" %1716)
// CHECK-NEXT:   %1720 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1712, i32 0, i32 1
// CHECK-NEXT:   %1721 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1722 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 114, ptr %1722, align 4
// CHECK-NEXT:   %1723 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1722, 1
// CHECK-NEXT:   %1724 = extractvalue { ptr, ptr } %1721, 1
// CHECK-NEXT:   %1725 = extractvalue { ptr, ptr } %1721, 0
// CHECK-NEXT:   %__llgo_funcval_code229 = call ptr asm "", "=r,0"(ptr %1725)
// CHECK-NEXT:   %1726 = call %reflect.Value %__llgo_funcval_code229(ptr {{(nest|swiftself)}} %1724, %"{{.*}}/runtime/internal/runtime.eface" %1723)
// CHECK-NEXT:   store %reflect.Value %1719, ptr %1713, align 8
// CHECK-NEXT:   store %reflect.Value %1726, ptr %1720, align 8
// CHECK-NEXT:   %1727 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 115
// CHECK-NEXT:   %1728 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1727, i32 0, i32 0
// CHECK-NEXT:   %1729 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1730 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 115, ptr %1730, align 4
// CHECK-NEXT:   %1731 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1730, 1
// CHECK-NEXT:   %1732 = extractvalue { ptr, ptr } %1729, 1
// CHECK-NEXT:   %1733 = extractvalue { ptr, ptr } %1729, 0
// CHECK-NEXT:   %__llgo_funcval_code230 = call ptr asm "", "=r,0"(ptr %1733)
// CHECK-NEXT:   %1734 = call %reflect.Value %__llgo_funcval_code230(ptr {{(nest|swiftself)}} %1732, %"{{.*}}/runtime/internal/runtime.eface" %1731)
// CHECK-NEXT:   %1735 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1727, i32 0, i32 1
// CHECK-NEXT:   %1736 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1737 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 115, ptr %1737, align 8
// CHECK-NEXT:   %1738 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1737, 1
// CHECK-NEXT:   %1739 = extractvalue { ptr, ptr } %1736, 1
// CHECK-NEXT:   %1740 = extractvalue { ptr, ptr } %1736, 0
// CHECK-NEXT:   %__llgo_funcval_code231 = call ptr asm "", "=r,0"(ptr %1740)
// CHECK-NEXT:   %1741 = call %reflect.Value %__llgo_funcval_code231(ptr {{(nest|swiftself)}} %1739, %"{{.*}}/runtime/internal/runtime.eface" %1738)
// CHECK-NEXT:   store %reflect.Value %1734, ptr %1728, align 8
// CHECK-NEXT:   store %reflect.Value %1741, ptr %1735, align 8
// CHECK-NEXT:   %1742 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 116
// CHECK-NEXT:   %1743 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1742, i32 0, i32 0
// CHECK-NEXT:   %1744 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1745 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 116, ptr %1745, align 8
// CHECK-NEXT:   %1746 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1745, 1
// CHECK-NEXT:   %1747 = extractvalue { ptr, ptr } %1744, 1
// CHECK-NEXT:   %1748 = extractvalue { ptr, ptr } %1744, 0
// CHECK-NEXT:   %__llgo_funcval_code232 = call ptr asm "", "=r,0"(ptr %1748)
// CHECK-NEXT:   %1749 = call %reflect.Value %__llgo_funcval_code232(ptr {{(nest|swiftself)}} %1747, %"{{.*}}/runtime/internal/runtime.eface" %1746)
// CHECK-NEXT:   %1750 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1742, i32 0, i32 1
// CHECK-NEXT:   %1751 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1752 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 116, ptr %1752, align 4
// CHECK-NEXT:   %1753 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1752, 1
// CHECK-NEXT:   %1754 = extractvalue { ptr, ptr } %1751, 1
// CHECK-NEXT:   %1755 = extractvalue { ptr, ptr } %1751, 0
// CHECK-NEXT:   %__llgo_funcval_code233 = call ptr asm "", "=r,0"(ptr %1755)
// CHECK-NEXT:   %1756 = call %reflect.Value %__llgo_funcval_code233(ptr {{(nest|swiftself)}} %1754, %"{{.*}}/runtime/internal/runtime.eface" %1753)
// CHECK-NEXT:   store %reflect.Value %1749, ptr %1743, align 8
// CHECK-NEXT:   store %reflect.Value %1756, ptr %1750, align 8
// CHECK-NEXT:   %1757 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 117
// CHECK-NEXT:   %1758 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1757, i32 0, i32 0
// CHECK-NEXT:   %1759 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1760 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 117, ptr %1760, align 4
// CHECK-NEXT:   %1761 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1760, 1
// CHECK-NEXT:   %1762 = extractvalue { ptr, ptr } %1759, 1
// CHECK-NEXT:   %1763 = extractvalue { ptr, ptr } %1759, 0
// CHECK-NEXT:   %__llgo_funcval_code234 = call ptr asm "", "=r,0"(ptr %1763)
// CHECK-NEXT:   %1764 = call %reflect.Value %__llgo_funcval_code234(ptr {{(nest|swiftself)}} %1762, %"{{.*}}/runtime/internal/runtime.eface" %1761)
// CHECK-NEXT:   %1765 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1757, i32 0, i32 1
// CHECK-NEXT:   %1766 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1767 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.170000e+02, ptr %1767, align 4
// CHECK-NEXT:   %1768 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1767, 1
// CHECK-NEXT:   %1769 = extractvalue { ptr, ptr } %1766, 1
// CHECK-NEXT:   %1770 = extractvalue { ptr, ptr } %1766, 0
// CHECK-NEXT:   %__llgo_funcval_code235 = call ptr asm "", "=r,0"(ptr %1770)
// CHECK-NEXT:   %1771 = call %reflect.Value %__llgo_funcval_code235(ptr {{(nest|swiftself)}} %1769, %"{{.*}}/runtime/internal/runtime.eface" %1768)
// CHECK-NEXT:   store %reflect.Value %1764, ptr %1758, align 8
// CHECK-NEXT:   store %reflect.Value %1771, ptr %1765, align 8
// CHECK-NEXT:   %1772 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 118
// CHECK-NEXT:   %1773 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1772, i32 0, i32 0
// CHECK-NEXT:   %1774 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1775 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.180000e+02, ptr %1775, align 4
// CHECK-NEXT:   %1776 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1775, 1
// CHECK-NEXT:   %1777 = extractvalue { ptr, ptr } %1774, 1
// CHECK-NEXT:   %1778 = extractvalue { ptr, ptr } %1774, 0
// CHECK-NEXT:   %__llgo_funcval_code236 = call ptr asm "", "=r,0"(ptr %1778)
// CHECK-NEXT:   %1779 = call %reflect.Value %__llgo_funcval_code236(ptr {{(nest|swiftself)}} %1777, %"{{.*}}/runtime/internal/runtime.eface" %1776)
// CHECK-NEXT:   %1780 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1772, i32 0, i32 1
// CHECK-NEXT:   %1781 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1782 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 118, ptr %1782, align 4
// CHECK-NEXT:   %1783 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1782, 1
// CHECK-NEXT:   %1784 = extractvalue { ptr, ptr } %1781, 1
// CHECK-NEXT:   %1785 = extractvalue { ptr, ptr } %1781, 0
// CHECK-NEXT:   %__llgo_funcval_code237 = call ptr asm "", "=r,0"(ptr %1785)
// CHECK-NEXT:   %1786 = call %reflect.Value %__llgo_funcval_code237(ptr {{(nest|swiftself)}} %1784, %"{{.*}}/runtime/internal/runtime.eface" %1783)
// CHECK-NEXT:   store %reflect.Value %1779, ptr %1773, align 8
// CHECK-NEXT:   store %reflect.Value %1786, ptr %1780, align 8
// CHECK-NEXT:   %1787 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 119
// CHECK-NEXT:   %1788 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1787, i32 0, i32 0
// CHECK-NEXT:   %1789 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1790 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 119, ptr %1790, align 4
// CHECK-NEXT:   %1791 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1790, 1
// CHECK-NEXT:   %1792 = extractvalue { ptr, ptr } %1789, 1
// CHECK-NEXT:   %1793 = extractvalue { ptr, ptr } %1789, 0
// CHECK-NEXT:   %__llgo_funcval_code238 = call ptr asm "", "=r,0"(ptr %1793)
// CHECK-NEXT:   %1794 = call %reflect.Value %__llgo_funcval_code238(ptr {{(nest|swiftself)}} %1792, %"{{.*}}/runtime/internal/runtime.eface" %1791)
// CHECK-NEXT:   %1795 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1787, i32 0, i32 1
// CHECK-NEXT:   %1796 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1797 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.190000e+02, ptr %1797, align 8
// CHECK-NEXT:   %1798 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1797, 1
// CHECK-NEXT:   %1799 = extractvalue { ptr, ptr } %1796, 1
// CHECK-NEXT:   %1800 = extractvalue { ptr, ptr } %1796, 0
// CHECK-NEXT:   %__llgo_funcval_code239 = call ptr asm "", "=r,0"(ptr %1800)
// CHECK-NEXT:   %1801 = call %reflect.Value %__llgo_funcval_code239(ptr {{(nest|swiftself)}} %1799, %"{{.*}}/runtime/internal/runtime.eface" %1798)
// CHECK-NEXT:   store %reflect.Value %1794, ptr %1788, align 8
// CHECK-NEXT:   store %reflect.Value %1801, ptr %1795, align 8
// CHECK-NEXT:   %1802 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 120
// CHECK-NEXT:   %1803 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1802, i32 0, i32 0
// CHECK-NEXT:   %1804 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1805 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.200000e+02, ptr %1805, align 8
// CHECK-NEXT:   %1806 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1805, 1
// CHECK-NEXT:   %1807 = extractvalue { ptr, ptr } %1804, 1
// CHECK-NEXT:   %1808 = extractvalue { ptr, ptr } %1804, 0
// CHECK-NEXT:   %__llgo_funcval_code240 = call ptr asm "", "=r,0"(ptr %1808)
// CHECK-NEXT:   %1809 = call %reflect.Value %__llgo_funcval_code240(ptr {{(nest|swiftself)}} %1807, %"{{.*}}/runtime/internal/runtime.eface" %1806)
// CHECK-NEXT:   %1810 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1802, i32 0, i32 1
// CHECK-NEXT:   %1811 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1812 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 120, ptr %1812, align 4
// CHECK-NEXT:   %1813 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %1812, 1
// CHECK-NEXT:   %1814 = extractvalue { ptr, ptr } %1811, 1
// CHECK-NEXT:   %1815 = extractvalue { ptr, ptr } %1811, 0
// CHECK-NEXT:   %__llgo_funcval_code241 = call ptr asm "", "=r,0"(ptr %1815)
// CHECK-NEXT:   %1816 = call %reflect.Value %__llgo_funcval_code241(ptr {{(nest|swiftself)}} %1814, %"{{.*}}/runtime/internal/runtime.eface" %1813)
// CHECK-NEXT:   store %reflect.Value %1809, ptr %1803, align 8
// CHECK-NEXT:   store %reflect.Value %1816, ptr %1810, align 8
// CHECK-NEXT:   %1817 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 121
// CHECK-NEXT:   %1818 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1817, i32 0, i32 0
// CHECK-NEXT:   %1819 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1820 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 121, ptr %1820, align 8
// CHECK-NEXT:   %1821 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1820, 1
// CHECK-NEXT:   %1822 = extractvalue { ptr, ptr } %1819, 1
// CHECK-NEXT:   %1823 = extractvalue { ptr, ptr } %1819, 0
// CHECK-NEXT:   %__llgo_funcval_code242 = call ptr asm "", "=r,0"(ptr %1823)
// CHECK-NEXT:   %1824 = call %reflect.Value %__llgo_funcval_code242(ptr {{(nest|swiftself)}} %1822, %"{{.*}}/runtime/internal/runtime.eface" %1821)
// CHECK-NEXT:   %1825 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1817, i32 0, i32 1
// CHECK-NEXT:   %1826 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1827 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 121, ptr %1827, align 8
// CHECK-NEXT:   %1828 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1827, 1
// CHECK-NEXT:   %1829 = extractvalue { ptr, ptr } %1826, 1
// CHECK-NEXT:   %1830 = extractvalue { ptr, ptr } %1826, 0
// CHECK-NEXT:   %__llgo_funcval_code243 = call ptr asm "", "=r,0"(ptr %1830)
// CHECK-NEXT:   %1831 = call %reflect.Value %__llgo_funcval_code243(ptr {{(nest|swiftself)}} %1829, %"{{.*}}/runtime/internal/runtime.eface" %1828)
// CHECK-NEXT:   store %reflect.Value %1824, ptr %1818, align 8
// CHECK-NEXT:   store %reflect.Value %1831, ptr %1825, align 8
// CHECK-NEXT:   %1832 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 122
// CHECK-NEXT:   %1833 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1832, i32 0, i32 0
// CHECK-NEXT:   %1834 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1835 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 122, ptr %1835, align 8
// CHECK-NEXT:   %1836 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1835, 1
// CHECK-NEXT:   %1837 = extractvalue { ptr, ptr } %1834, 1
// CHECK-NEXT:   %1838 = extractvalue { ptr, ptr } %1834, 0
// CHECK-NEXT:   %__llgo_funcval_code244 = call ptr asm "", "=r,0"(ptr %1838)
// CHECK-NEXT:   %1839 = call %reflect.Value %__llgo_funcval_code244(ptr {{(nest|swiftself)}} %1837, %"{{.*}}/runtime/internal/runtime.eface" %1836)
// CHECK-NEXT:   %1840 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1832, i32 0, i32 1
// CHECK-NEXT:   %1841 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1842 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 122, ptr %1842, align 8
// CHECK-NEXT:   %1843 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1842, 1
// CHECK-NEXT:   %1844 = extractvalue { ptr, ptr } %1841, 1
// CHECK-NEXT:   %1845 = extractvalue { ptr, ptr } %1841, 0
// CHECK-NEXT:   %__llgo_funcval_code245 = call ptr asm "", "=r,0"(ptr %1845)
// CHECK-NEXT:   %1846 = call %reflect.Value %__llgo_funcval_code245(ptr {{(nest|swiftself)}} %1844, %"{{.*}}/runtime/internal/runtime.eface" %1843)
// CHECK-NEXT:   store %reflect.Value %1839, ptr %1833, align 8
// CHECK-NEXT:   store %reflect.Value %1846, ptr %1840, align 8
// CHECK-NEXT:   %1847 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 123
// CHECK-NEXT:   %1848 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1847, i32 0, i32 0
// CHECK-NEXT:   %1849 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1850 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 123, ptr %1850, align 8
// CHECK-NEXT:   %1851 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %1850, 1
// CHECK-NEXT:   %1852 = extractvalue { ptr, ptr } %1849, 1
// CHECK-NEXT:   %1853 = extractvalue { ptr, ptr } %1849, 0
// CHECK-NEXT:   %__llgo_funcval_code246 = call ptr asm "", "=r,0"(ptr %1853)
// CHECK-NEXT:   %1854 = call %reflect.Value %__llgo_funcval_code246(ptr {{(nest|swiftself)}} %1852, %"{{.*}}/runtime/internal/runtime.eface" %1851)
// CHECK-NEXT:   %1855 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1847, i32 0, i32 1
// CHECK-NEXT:   %1856 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1857 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 123, ptr %1857, align 8
// CHECK-NEXT:   %1858 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1857, 1
// CHECK-NEXT:   %1859 = extractvalue { ptr, ptr } %1856, 1
// CHECK-NEXT:   %1860 = extractvalue { ptr, ptr } %1856, 0
// CHECK-NEXT:   %__llgo_funcval_code247 = call ptr asm "", "=r,0"(ptr %1860)
// CHECK-NEXT:   %1861 = call %reflect.Value %__llgo_funcval_code247(ptr {{(nest|swiftself)}} %1859, %"{{.*}}/runtime/internal/runtime.eface" %1858)
// CHECK-NEXT:   store %reflect.Value %1854, ptr %1848, align 8
// CHECK-NEXT:   store %reflect.Value %1861, ptr %1855, align 8
// CHECK-NEXT:   %1862 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 124
// CHECK-NEXT:   %1863 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1862, i32 0, i32 0
// CHECK-NEXT:   %1864 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1865 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 124, ptr %1865, align 8
// CHECK-NEXT:   %1866 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1865, 1
// CHECK-NEXT:   %1867 = extractvalue { ptr, ptr } %1864, 1
// CHECK-NEXT:   %1868 = extractvalue { ptr, ptr } %1864, 0
// CHECK-NEXT:   %__llgo_funcval_code248 = call ptr asm "", "=r,0"(ptr %1868)
// CHECK-NEXT:   %1869 = call %reflect.Value %__llgo_funcval_code248(ptr {{(nest|swiftself)}} %1867, %"{{.*}}/runtime/internal/runtime.eface" %1866)
// CHECK-NEXT:   %1870 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1862, i32 0, i32 1
// CHECK-NEXT:   %1871 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1872 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 124, ptr %1872, align 8
// CHECK-NEXT:   %1873 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1872, 1
// CHECK-NEXT:   %1874 = extractvalue { ptr, ptr } %1871, 1
// CHECK-NEXT:   %1875 = extractvalue { ptr, ptr } %1871, 0
// CHECK-NEXT:   %__llgo_funcval_code249 = call ptr asm "", "=r,0"(ptr %1875)
// CHECK-NEXT:   %1876 = call %reflect.Value %__llgo_funcval_code249(ptr {{(nest|swiftself)}} %1874, %"{{.*}}/runtime/internal/runtime.eface" %1873)
// CHECK-NEXT:   store %reflect.Value %1869, ptr %1863, align 8
// CHECK-NEXT:   store %reflect.Value %1876, ptr %1870, align 8
// CHECK-NEXT:   %1877 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 125
// CHECK-NEXT:   %1878 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1877, i32 0, i32 0
// CHECK-NEXT:   %1879 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1880 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 125, ptr %1880, align 8
// CHECK-NEXT:   %1881 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1880, 1
// CHECK-NEXT:   %1882 = extractvalue { ptr, ptr } %1879, 1
// CHECK-NEXT:   %1883 = extractvalue { ptr, ptr } %1879, 0
// CHECK-NEXT:   %__llgo_funcval_code250 = call ptr asm "", "=r,0"(ptr %1883)
// CHECK-NEXT:   %1884 = call %reflect.Value %__llgo_funcval_code250(ptr {{(nest|swiftself)}} %1882, %"{{.*}}/runtime/internal/runtime.eface" %1881)
// CHECK-NEXT:   %1885 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1877, i32 0, i32 1
// CHECK-NEXT:   %1886 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1887 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 125, ptr %1887, align 8
// CHECK-NEXT:   %1888 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1887, 1
// CHECK-NEXT:   %1889 = extractvalue { ptr, ptr } %1886, 1
// CHECK-NEXT:   %1890 = extractvalue { ptr, ptr } %1886, 0
// CHECK-NEXT:   %__llgo_funcval_code251 = call ptr asm "", "=r,0"(ptr %1890)
// CHECK-NEXT:   %1891 = call %reflect.Value %__llgo_funcval_code251(ptr {{(nest|swiftself)}} %1889, %"{{.*}}/runtime/internal/runtime.eface" %1888)
// CHECK-NEXT:   store %reflect.Value %1884, ptr %1878, align 8
// CHECK-NEXT:   store %reflect.Value %1891, ptr %1885, align 8
// CHECK-NEXT:   %1892 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 126
// CHECK-NEXT:   %1893 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1892, i32 0, i32 0
// CHECK-NEXT:   %1894 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1895 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 126, ptr %1895, align 8
// CHECK-NEXT:   %1896 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1895, 1
// CHECK-NEXT:   %1897 = extractvalue { ptr, ptr } %1894, 1
// CHECK-NEXT:   %1898 = extractvalue { ptr, ptr } %1894, 0
// CHECK-NEXT:   %__llgo_funcval_code252 = call ptr asm "", "=r,0"(ptr %1898)
// CHECK-NEXT:   %1899 = call %reflect.Value %__llgo_funcval_code252(ptr {{(nest|swiftself)}} %1897, %"{{.*}}/runtime/internal/runtime.eface" %1896)
// CHECK-NEXT:   %1900 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1892, i32 0, i32 1
// CHECK-NEXT:   %1901 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1902 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 126, ptr %1902, align 8
// CHECK-NEXT:   %1903 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1902, 1
// CHECK-NEXT:   %1904 = extractvalue { ptr, ptr } %1901, 1
// CHECK-NEXT:   %1905 = extractvalue { ptr, ptr } %1901, 0
// CHECK-NEXT:   %__llgo_funcval_code253 = call ptr asm "", "=r,0"(ptr %1905)
// CHECK-NEXT:   %1906 = call %reflect.Value %__llgo_funcval_code253(ptr {{(nest|swiftself)}} %1904, %"{{.*}}/runtime/internal/runtime.eface" %1903)
// CHECK-NEXT:   store %reflect.Value %1899, ptr %1893, align 8
// CHECK-NEXT:   store %reflect.Value %1906, ptr %1900, align 8
// CHECK-NEXT:   %1907 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 127
// CHECK-NEXT:   %1908 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1907, i32 0, i32 0
// CHECK-NEXT:   %1909 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1910 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 127, ptr %1910, align 8
// CHECK-NEXT:   %1911 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %1910, 1
// CHECK-NEXT:   %1912 = extractvalue { ptr, ptr } %1909, 1
// CHECK-NEXT:   %1913 = extractvalue { ptr, ptr } %1909, 0
// CHECK-NEXT:   %__llgo_funcval_code254 = call ptr asm "", "=r,0"(ptr %1913)
// CHECK-NEXT:   %1914 = call %reflect.Value %__llgo_funcval_code254(ptr {{(nest|swiftself)}} %1912, %"{{.*}}/runtime/internal/runtime.eface" %1911)
// CHECK-NEXT:   %1915 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1907, i32 0, i32 1
// CHECK-NEXT:   %1916 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1917 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 127, ptr %1917, align 8
// CHECK-NEXT:   %1918 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1917, 1
// CHECK-NEXT:   %1919 = extractvalue { ptr, ptr } %1916, 1
// CHECK-NEXT:   %1920 = extractvalue { ptr, ptr } %1916, 0
// CHECK-NEXT:   %__llgo_funcval_code255 = call ptr asm "", "=r,0"(ptr %1920)
// CHECK-NEXT:   %1921 = call %reflect.Value %__llgo_funcval_code255(ptr {{(nest|swiftself)}} %1919, %"{{.*}}/runtime/internal/runtime.eface" %1918)
// CHECK-NEXT:   store %reflect.Value %1914, ptr %1908, align 8
// CHECK-NEXT:   store %reflect.Value %1921, ptr %1915, align 8
// CHECK-NEXT:   %1922 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 128
// CHECK-NEXT:   %1923 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1922, i32 0, i32 0
// CHECK-NEXT:   %1924 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1925 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 128, ptr %1925, align 8
// CHECK-NEXT:   %1926 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1925, 1
// CHECK-NEXT:   %1927 = extractvalue { ptr, ptr } %1924, 1
// CHECK-NEXT:   %1928 = extractvalue { ptr, ptr } %1924, 0
// CHECK-NEXT:   %__llgo_funcval_code256 = call ptr asm "", "=r,0"(ptr %1928)
// CHECK-NEXT:   %1929 = call %reflect.Value %__llgo_funcval_code256(ptr {{(nest|swiftself)}} %1927, %"{{.*}}/runtime/internal/runtime.eface" %1926)
// CHECK-NEXT:   %1930 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1922, i32 0, i32 1
// CHECK-NEXT:   %1931 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1932 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 128, ptr %1932, align 8
// CHECK-NEXT:   %1933 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1932, 1
// CHECK-NEXT:   %1934 = extractvalue { ptr, ptr } %1931, 1
// CHECK-NEXT:   %1935 = extractvalue { ptr, ptr } %1931, 0
// CHECK-NEXT:   %__llgo_funcval_code257 = call ptr asm "", "=r,0"(ptr %1935)
// CHECK-NEXT:   %1936 = call %reflect.Value %__llgo_funcval_code257(ptr {{(nest|swiftself)}} %1934, %"{{.*}}/runtime/internal/runtime.eface" %1933)
// CHECK-NEXT:   store %reflect.Value %1929, ptr %1923, align 8
// CHECK-NEXT:   store %reflect.Value %1936, ptr %1930, align 8
// CHECK-NEXT:   %1937 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 129
// CHECK-NEXT:   %1938 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1937, i32 0, i32 0
// CHECK-NEXT:   %1939 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1940 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 129, ptr %1940, align 8
// CHECK-NEXT:   %1941 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %1940, 1
// CHECK-NEXT:   %1942 = extractvalue { ptr, ptr } %1939, 1
// CHECK-NEXT:   %1943 = extractvalue { ptr, ptr } %1939, 0
// CHECK-NEXT:   %__llgo_funcval_code258 = call ptr asm "", "=r,0"(ptr %1943)
// CHECK-NEXT:   %1944 = call %reflect.Value %__llgo_funcval_code258(ptr {{(nest|swiftself)}} %1942, %"{{.*}}/runtime/internal/runtime.eface" %1941)
// CHECK-NEXT:   %1945 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1937, i32 0, i32 1
// CHECK-NEXT:   %1946 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1947 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 129, ptr %1947, align 8
// CHECK-NEXT:   %1948 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1947, 1
// CHECK-NEXT:   %1949 = extractvalue { ptr, ptr } %1946, 1
// CHECK-NEXT:   %1950 = extractvalue { ptr, ptr } %1946, 0
// CHECK-NEXT:   %__llgo_funcval_code259 = call ptr asm "", "=r,0"(ptr %1950)
// CHECK-NEXT:   %1951 = call %reflect.Value %__llgo_funcval_code259(ptr {{(nest|swiftself)}} %1949, %"{{.*}}/runtime/internal/runtime.eface" %1948)
// CHECK-NEXT:   store %reflect.Value %1944, ptr %1938, align 8
// CHECK-NEXT:   store %reflect.Value %1951, ptr %1945, align 8
// CHECK-NEXT:   %1952 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 130
// CHECK-NEXT:   %1953 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1952, i32 0, i32 0
// CHECK-NEXT:   %1954 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1955 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 130, ptr %1955, align 8
// CHECK-NEXT:   %1956 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1955, 1
// CHECK-NEXT:   %1957 = extractvalue { ptr, ptr } %1954, 1
// CHECK-NEXT:   %1958 = extractvalue { ptr, ptr } %1954, 0
// CHECK-NEXT:   %__llgo_funcval_code260 = call ptr asm "", "=r,0"(ptr %1958)
// CHECK-NEXT:   %1959 = call %reflect.Value %__llgo_funcval_code260(ptr {{(nest|swiftself)}} %1957, %"{{.*}}/runtime/internal/runtime.eface" %1956)
// CHECK-NEXT:   %1960 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1952, i32 0, i32 1
// CHECK-NEXT:   %1961 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1962 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.300000e+02, ptr %1962, align 4
// CHECK-NEXT:   %1963 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1962, 1
// CHECK-NEXT:   %1964 = extractvalue { ptr, ptr } %1961, 1
// CHECK-NEXT:   %1965 = extractvalue { ptr, ptr } %1961, 0
// CHECK-NEXT:   %__llgo_funcval_code261 = call ptr asm "", "=r,0"(ptr %1965)
// CHECK-NEXT:   %1966 = call %reflect.Value %__llgo_funcval_code261(ptr {{(nest|swiftself)}} %1964, %"{{.*}}/runtime/internal/runtime.eface" %1963)
// CHECK-NEXT:   store %reflect.Value %1959, ptr %1953, align 8
// CHECK-NEXT:   store %reflect.Value %1966, ptr %1960, align 8
// CHECK-NEXT:   %1967 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 131
// CHECK-NEXT:   %1968 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1967, i32 0, i32 0
// CHECK-NEXT:   %1969 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1970 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.310000e+02, ptr %1970, align 4
// CHECK-NEXT:   %1971 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %1970, 1
// CHECK-NEXT:   %1972 = extractvalue { ptr, ptr } %1969, 1
// CHECK-NEXT:   %1973 = extractvalue { ptr, ptr } %1969, 0
// CHECK-NEXT:   %__llgo_funcval_code262 = call ptr asm "", "=r,0"(ptr %1973)
// CHECK-NEXT:   %1974 = call %reflect.Value %__llgo_funcval_code262(ptr {{(nest|swiftself)}} %1972, %"{{.*}}/runtime/internal/runtime.eface" %1971)
// CHECK-NEXT:   %1975 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1967, i32 0, i32 1
// CHECK-NEXT:   %1976 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1977 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 131, ptr %1977, align 8
// CHECK-NEXT:   %1978 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1977, 1
// CHECK-NEXT:   %1979 = extractvalue { ptr, ptr } %1976, 1
// CHECK-NEXT:   %1980 = extractvalue { ptr, ptr } %1976, 0
// CHECK-NEXT:   %__llgo_funcval_code263 = call ptr asm "", "=r,0"(ptr %1980)
// CHECK-NEXT:   %1981 = call %reflect.Value %__llgo_funcval_code263(ptr {{(nest|swiftself)}} %1979, %"{{.*}}/runtime/internal/runtime.eface" %1978)
// CHECK-NEXT:   store %reflect.Value %1974, ptr %1968, align 8
// CHECK-NEXT:   store %reflect.Value %1981, ptr %1975, align 8
// CHECK-NEXT:   %1982 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 132
// CHECK-NEXT:   %1983 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1982, i32 0, i32 0
// CHECK-NEXT:   %1984 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1985 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 132, ptr %1985, align 8
// CHECK-NEXT:   %1986 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %1985, 1
// CHECK-NEXT:   %1987 = extractvalue { ptr, ptr } %1984, 1
// CHECK-NEXT:   %1988 = extractvalue { ptr, ptr } %1984, 0
// CHECK-NEXT:   %__llgo_funcval_code264 = call ptr asm "", "=r,0"(ptr %1988)
// CHECK-NEXT:   %1989 = call %reflect.Value %__llgo_funcval_code264(ptr {{(nest|swiftself)}} %1987, %"{{.*}}/runtime/internal/runtime.eface" %1986)
// CHECK-NEXT:   %1990 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1982, i32 0, i32 1
// CHECK-NEXT:   %1991 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %1992 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.320000e+02, ptr %1992, align 8
// CHECK-NEXT:   %1993 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %1992, 1
// CHECK-NEXT:   %1994 = extractvalue { ptr, ptr } %1991, 1
// CHECK-NEXT:   %1995 = extractvalue { ptr, ptr } %1991, 0
// CHECK-NEXT:   %__llgo_funcval_code265 = call ptr asm "", "=r,0"(ptr %1995)
// CHECK-NEXT:   %1996 = call %reflect.Value %__llgo_funcval_code265(ptr {{(nest|swiftself)}} %1994, %"{{.*}}/runtime/internal/runtime.eface" %1993)
// CHECK-NEXT:   store %reflect.Value %1989, ptr %1983, align 8
// CHECK-NEXT:   store %reflect.Value %1996, ptr %1990, align 8
// CHECK-NEXT:   %1997 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 133
// CHECK-NEXT:   %1998 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1997, i32 0, i32 0
// CHECK-NEXT:   %1999 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2000 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.330000e+02, ptr %2000, align 8
// CHECK-NEXT:   %2001 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2000, 1
// CHECK-NEXT:   %2002 = extractvalue { ptr, ptr } %1999, 1
// CHECK-NEXT:   %2003 = extractvalue { ptr, ptr } %1999, 0
// CHECK-NEXT:   %__llgo_funcval_code266 = call ptr asm "", "=r,0"(ptr %2003)
// CHECK-NEXT:   %2004 = call %reflect.Value %__llgo_funcval_code266(ptr {{(nest|swiftself)}} %2002, %"{{.*}}/runtime/internal/runtime.eface" %2001)
// CHECK-NEXT:   %2005 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1997, i32 0, i32 1
// CHECK-NEXT:   %2006 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2007 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 133, ptr %2007, align 8
// CHECK-NEXT:   %2008 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %2007, 1
// CHECK-NEXT:   %2009 = extractvalue { ptr, ptr } %2006, 1
// CHECK-NEXT:   %2010 = extractvalue { ptr, ptr } %2006, 0
// CHECK-NEXT:   %__llgo_funcval_code267 = call ptr asm "", "=r,0"(ptr %2010)
// CHECK-NEXT:   %2011 = call %reflect.Value %__llgo_funcval_code267(ptr {{(nest|swiftself)}} %2009, %"{{.*}}/runtime/internal/runtime.eface" %2008)
// CHECK-NEXT:   store %reflect.Value %2004, ptr %1998, align 8
// CHECK-NEXT:   store %reflect.Value %2011, ptr %2005, align 8
// CHECK-NEXT:   %2012 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 134
// CHECK-NEXT:   %2013 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2012, i32 0, i32 0
// CHECK-NEXT:   %2014 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2015 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 134, ptr %2015, align 8
// CHECK-NEXT:   %2016 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2015, 1
// CHECK-NEXT:   %2017 = extractvalue { ptr, ptr } %2014, 1
// CHECK-NEXT:   %2018 = extractvalue { ptr, ptr } %2014, 0
// CHECK-NEXT:   %__llgo_funcval_code268 = call ptr asm "", "=r,0"(ptr %2018)
// CHECK-NEXT:   %2019 = call %reflect.Value %__llgo_funcval_code268(ptr {{(nest|swiftself)}} %2017, %"{{.*}}/runtime/internal/runtime.eface" %2016)
// CHECK-NEXT:   %2020 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2012, i32 0, i32 1
// CHECK-NEXT:   %2021 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2022 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 134, ptr %2022, align 8
// CHECK-NEXT:   %2023 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2022, 1
// CHECK-NEXT:   %2024 = extractvalue { ptr, ptr } %2021, 1
// CHECK-NEXT:   %2025 = extractvalue { ptr, ptr } %2021, 0
// CHECK-NEXT:   %__llgo_funcval_code269 = call ptr asm "", "=r,0"(ptr %2025)
// CHECK-NEXT:   %2026 = call %reflect.Value %__llgo_funcval_code269(ptr {{(nest|swiftself)}} %2024, %"{{.*}}/runtime/internal/runtime.eface" %2023)
// CHECK-NEXT:   store %reflect.Value %2019, ptr %2013, align 8
// CHECK-NEXT:   store %reflect.Value %2026, ptr %2020, align 8
// CHECK-NEXT:   %2027 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 135
// CHECK-NEXT:   %2028 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2027, i32 0, i32 0
// CHECK-NEXT:   %2029 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2030 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 135, ptr %2030, align 8
// CHECK-NEXT:   %2031 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2030, 1
// CHECK-NEXT:   %2032 = extractvalue { ptr, ptr } %2029, 1
// CHECK-NEXT:   %2033 = extractvalue { ptr, ptr } %2029, 0
// CHECK-NEXT:   %__llgo_funcval_code270 = call ptr asm "", "=r,0"(ptr %2033)
// CHECK-NEXT:   %2034 = call %reflect.Value %__llgo_funcval_code270(ptr {{(nest|swiftself)}} %2032, %"{{.*}}/runtime/internal/runtime.eface" %2031)
// CHECK-NEXT:   %2035 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2027, i32 0, i32 1
// CHECK-NEXT:   %2036 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2037 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 135, ptr %2037, align 8
// CHECK-NEXT:   %2038 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2037, 1
// CHECK-NEXT:   %2039 = extractvalue { ptr, ptr } %2036, 1
// CHECK-NEXT:   %2040 = extractvalue { ptr, ptr } %2036, 0
// CHECK-NEXT:   %__llgo_funcval_code271 = call ptr asm "", "=r,0"(ptr %2040)
// CHECK-NEXT:   %2041 = call %reflect.Value %__llgo_funcval_code271(ptr {{(nest|swiftself)}} %2039, %"{{.*}}/runtime/internal/runtime.eface" %2038)
// CHECK-NEXT:   store %reflect.Value %2034, ptr %2028, align 8
// CHECK-NEXT:   store %reflect.Value %2041, ptr %2035, align 8
// CHECK-NEXT:   %2042 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 136
// CHECK-NEXT:   %2043 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2042, i32 0, i32 0
// CHECK-NEXT:   %2044 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2045 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 136, ptr %2045, align 8
// CHECK-NEXT:   %2046 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2045, 1
// CHECK-NEXT:   %2047 = extractvalue { ptr, ptr } %2044, 1
// CHECK-NEXT:   %2048 = extractvalue { ptr, ptr } %2044, 0
// CHECK-NEXT:   %__llgo_funcval_code272 = call ptr asm "", "=r,0"(ptr %2048)
// CHECK-NEXT:   %2049 = call %reflect.Value %__llgo_funcval_code272(ptr {{(nest|swiftself)}} %2047, %"{{.*}}/runtime/internal/runtime.eface" %2046)
// CHECK-NEXT:   %2050 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2042, i32 0, i32 1
// CHECK-NEXT:   %2051 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2052 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 136, ptr %2052, align 8
// CHECK-NEXT:   %2053 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2052, 1
// CHECK-NEXT:   %2054 = extractvalue { ptr, ptr } %2051, 1
// CHECK-NEXT:   %2055 = extractvalue { ptr, ptr } %2051, 0
// CHECK-NEXT:   %__llgo_funcval_code273 = call ptr asm "", "=r,0"(ptr %2055)
// CHECK-NEXT:   %2056 = call %reflect.Value %__llgo_funcval_code273(ptr {{(nest|swiftself)}} %2054, %"{{.*}}/runtime/internal/runtime.eface" %2053)
// CHECK-NEXT:   store %reflect.Value %2049, ptr %2043, align 8
// CHECK-NEXT:   store %reflect.Value %2056, ptr %2050, align 8
// CHECK-NEXT:   %2057 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 137
// CHECK-NEXT:   %2058 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2057, i32 0, i32 0
// CHECK-NEXT:   %2059 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2060 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 137, ptr %2060, align 8
// CHECK-NEXT:   %2061 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2060, 1
// CHECK-NEXT:   %2062 = extractvalue { ptr, ptr } %2059, 1
// CHECK-NEXT:   %2063 = extractvalue { ptr, ptr } %2059, 0
// CHECK-NEXT:   %__llgo_funcval_code274 = call ptr asm "", "=r,0"(ptr %2063)
// CHECK-NEXT:   %2064 = call %reflect.Value %__llgo_funcval_code274(ptr {{(nest|swiftself)}} %2062, %"{{.*}}/runtime/internal/runtime.eface" %2061)
// CHECK-NEXT:   %2065 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2057, i32 0, i32 1
// CHECK-NEXT:   %2066 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2067 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 137, ptr %2067, align 8
// CHECK-NEXT:   %2068 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2067, 1
// CHECK-NEXT:   %2069 = extractvalue { ptr, ptr } %2066, 1
// CHECK-NEXT:   %2070 = extractvalue { ptr, ptr } %2066, 0
// CHECK-NEXT:   %__llgo_funcval_code275 = call ptr asm "", "=r,0"(ptr %2070)
// CHECK-NEXT:   %2071 = call %reflect.Value %__llgo_funcval_code275(ptr {{(nest|swiftself)}} %2069, %"{{.*}}/runtime/internal/runtime.eface" %2068)
// CHECK-NEXT:   store %reflect.Value %2064, ptr %2058, align 8
// CHECK-NEXT:   store %reflect.Value %2071, ptr %2065, align 8
// CHECK-NEXT:   %2072 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 138
// CHECK-NEXT:   %2073 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2072, i32 0, i32 0
// CHECK-NEXT:   %2074 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2075 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 138, ptr %2075, align 8
// CHECK-NEXT:   %2076 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2075, 1
// CHECK-NEXT:   %2077 = extractvalue { ptr, ptr } %2074, 1
// CHECK-NEXT:   %2078 = extractvalue { ptr, ptr } %2074, 0
// CHECK-NEXT:   %__llgo_funcval_code276 = call ptr asm "", "=r,0"(ptr %2078)
// CHECK-NEXT:   %2079 = call %reflect.Value %__llgo_funcval_code276(ptr {{(nest|swiftself)}} %2077, %"{{.*}}/runtime/internal/runtime.eface" %2076)
// CHECK-NEXT:   %2080 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2072, i32 0, i32 1
// CHECK-NEXT:   %2081 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2082 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 138, ptr %2082, align 8
// CHECK-NEXT:   %2083 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2082, 1
// CHECK-NEXT:   %2084 = extractvalue { ptr, ptr } %2081, 1
// CHECK-NEXT:   %2085 = extractvalue { ptr, ptr } %2081, 0
// CHECK-NEXT:   %__llgo_funcval_code277 = call ptr asm "", "=r,0"(ptr %2085)
// CHECK-NEXT:   %2086 = call %reflect.Value %__llgo_funcval_code277(ptr {{(nest|swiftself)}} %2084, %"{{.*}}/runtime/internal/runtime.eface" %2083)
// CHECK-NEXT:   store %reflect.Value %2079, ptr %2073, align 8
// CHECK-NEXT:   store %reflect.Value %2086, ptr %2080, align 8
// CHECK-NEXT:   %2087 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 139
// CHECK-NEXT:   %2088 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2087, i32 0, i32 0
// CHECK-NEXT:   %2089 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2090 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 139, ptr %2090, align 8
// CHECK-NEXT:   %2091 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2090, 1
// CHECK-NEXT:   %2092 = extractvalue { ptr, ptr } %2089, 1
// CHECK-NEXT:   %2093 = extractvalue { ptr, ptr } %2089, 0
// CHECK-NEXT:   %__llgo_funcval_code278 = call ptr asm "", "=r,0"(ptr %2093)
// CHECK-NEXT:   %2094 = call %reflect.Value %__llgo_funcval_code278(ptr {{(nest|swiftself)}} %2092, %"{{.*}}/runtime/internal/runtime.eface" %2091)
// CHECK-NEXT:   %2095 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2087, i32 0, i32 1
// CHECK-NEXT:   %2096 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2097 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 139, ptr %2097, align 8
// CHECK-NEXT:   %2098 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2097, 1
// CHECK-NEXT:   %2099 = extractvalue { ptr, ptr } %2096, 1
// CHECK-NEXT:   %2100 = extractvalue { ptr, ptr } %2096, 0
// CHECK-NEXT:   %__llgo_funcval_code279 = call ptr asm "", "=r,0"(ptr %2100)
// CHECK-NEXT:   %2101 = call %reflect.Value %__llgo_funcval_code279(ptr {{(nest|swiftself)}} %2099, %"{{.*}}/runtime/internal/runtime.eface" %2098)
// CHECK-NEXT:   store %reflect.Value %2094, ptr %2088, align 8
// CHECK-NEXT:   store %reflect.Value %2101, ptr %2095, align 8
// CHECK-NEXT:   %2102 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 140
// CHECK-NEXT:   %2103 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2102, i32 0, i32 0
// CHECK-NEXT:   %2104 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2105 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 140, ptr %2105, align 8
// CHECK-NEXT:   %2106 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2105, 1
// CHECK-NEXT:   %2107 = extractvalue { ptr, ptr } %2104, 1
// CHECK-NEXT:   %2108 = extractvalue { ptr, ptr } %2104, 0
// CHECK-NEXT:   %__llgo_funcval_code280 = call ptr asm "", "=r,0"(ptr %2108)
// CHECK-NEXT:   %2109 = call %reflect.Value %__llgo_funcval_code280(ptr {{(nest|swiftself)}} %2107, %"{{.*}}/runtime/internal/runtime.eface" %2106)
// CHECK-NEXT:   %2110 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2102, i32 0, i32 1
// CHECK-NEXT:   %2111 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2112 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 140, ptr %2112, align 8
// CHECK-NEXT:   %2113 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2112, 1
// CHECK-NEXT:   %2114 = extractvalue { ptr, ptr } %2111, 1
// CHECK-NEXT:   %2115 = extractvalue { ptr, ptr } %2111, 0
// CHECK-NEXT:   %__llgo_funcval_code281 = call ptr asm "", "=r,0"(ptr %2115)
// CHECK-NEXT:   %2116 = call %reflect.Value %__llgo_funcval_code281(ptr {{(nest|swiftself)}} %2114, %"{{.*}}/runtime/internal/runtime.eface" %2113)
// CHECK-NEXT:   store %reflect.Value %2109, ptr %2103, align 8
// CHECK-NEXT:   store %reflect.Value %2116, ptr %2110, align 8
// CHECK-NEXT:   %2117 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 141
// CHECK-NEXT:   %2118 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2117, i32 0, i32 0
// CHECK-NEXT:   %2119 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2120 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 141, ptr %2120, align 8
// CHECK-NEXT:   %2121 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2120, 1
// CHECK-NEXT:   %2122 = extractvalue { ptr, ptr } %2119, 1
// CHECK-NEXT:   %2123 = extractvalue { ptr, ptr } %2119, 0
// CHECK-NEXT:   %__llgo_funcval_code282 = call ptr asm "", "=r,0"(ptr %2123)
// CHECK-NEXT:   %2124 = call %reflect.Value %__llgo_funcval_code282(ptr {{(nest|swiftself)}} %2122, %"{{.*}}/runtime/internal/runtime.eface" %2121)
// CHECK-NEXT:   %2125 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2117, i32 0, i32 1
// CHECK-NEXT:   %2126 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2127 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.410000e+02, ptr %2127, align 4
// CHECK-NEXT:   %2128 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2127, 1
// CHECK-NEXT:   %2129 = extractvalue { ptr, ptr } %2126, 1
// CHECK-NEXT:   %2130 = extractvalue { ptr, ptr } %2126, 0
// CHECK-NEXT:   %__llgo_funcval_code283 = call ptr asm "", "=r,0"(ptr %2130)
// CHECK-NEXT:   %2131 = call %reflect.Value %__llgo_funcval_code283(ptr {{(nest|swiftself)}} %2129, %"{{.*}}/runtime/internal/runtime.eface" %2128)
// CHECK-NEXT:   store %reflect.Value %2124, ptr %2118, align 8
// CHECK-NEXT:   store %reflect.Value %2131, ptr %2125, align 8
// CHECK-NEXT:   %2132 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 142
// CHECK-NEXT:   %2133 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2132, i32 0, i32 0
// CHECK-NEXT:   %2134 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2135 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.420000e+02, ptr %2135, align 4
// CHECK-NEXT:   %2136 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2135, 1
// CHECK-NEXT:   %2137 = extractvalue { ptr, ptr } %2134, 1
// CHECK-NEXT:   %2138 = extractvalue { ptr, ptr } %2134, 0
// CHECK-NEXT:   %__llgo_funcval_code284 = call ptr asm "", "=r,0"(ptr %2138)
// CHECK-NEXT:   %2139 = call %reflect.Value %__llgo_funcval_code284(ptr {{(nest|swiftself)}} %2137, %"{{.*}}/runtime/internal/runtime.eface" %2136)
// CHECK-NEXT:   %2140 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2132, i32 0, i32 1
// CHECK-NEXT:   %2141 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2142 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 142, ptr %2142, align 8
// CHECK-NEXT:   %2143 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2142, 1
// CHECK-NEXT:   %2144 = extractvalue { ptr, ptr } %2141, 1
// CHECK-NEXT:   %2145 = extractvalue { ptr, ptr } %2141, 0
// CHECK-NEXT:   %__llgo_funcval_code285 = call ptr asm "", "=r,0"(ptr %2145)
// CHECK-NEXT:   %2146 = call %reflect.Value %__llgo_funcval_code285(ptr {{(nest|swiftself)}} %2144, %"{{.*}}/runtime/internal/runtime.eface" %2143)
// CHECK-NEXT:   store %reflect.Value %2139, ptr %2133, align 8
// CHECK-NEXT:   store %reflect.Value %2146, ptr %2140, align 8
// CHECK-NEXT:   %2147 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 143
// CHECK-NEXT:   %2148 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2147, i32 0, i32 0
// CHECK-NEXT:   %2149 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2150 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 143, ptr %2150, align 8
// CHECK-NEXT:   %2151 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2150, 1
// CHECK-NEXT:   %2152 = extractvalue { ptr, ptr } %2149, 1
// CHECK-NEXT:   %2153 = extractvalue { ptr, ptr } %2149, 0
// CHECK-NEXT:   %__llgo_funcval_code286 = call ptr asm "", "=r,0"(ptr %2153)
// CHECK-NEXT:   %2154 = call %reflect.Value %__llgo_funcval_code286(ptr {{(nest|swiftself)}} %2152, %"{{.*}}/runtime/internal/runtime.eface" %2151)
// CHECK-NEXT:   %2155 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2147, i32 0, i32 1
// CHECK-NEXT:   %2156 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2157 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.430000e+02, ptr %2157, align 8
// CHECK-NEXT:   %2158 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2157, 1
// CHECK-NEXT:   %2159 = extractvalue { ptr, ptr } %2156, 1
// CHECK-NEXT:   %2160 = extractvalue { ptr, ptr } %2156, 0
// CHECK-NEXT:   %__llgo_funcval_code287 = call ptr asm "", "=r,0"(ptr %2160)
// CHECK-NEXT:   %2161 = call %reflect.Value %__llgo_funcval_code287(ptr {{(nest|swiftself)}} %2159, %"{{.*}}/runtime/internal/runtime.eface" %2158)
// CHECK-NEXT:   store %reflect.Value %2154, ptr %2148, align 8
// CHECK-NEXT:   store %reflect.Value %2161, ptr %2155, align 8
// CHECK-NEXT:   %2162 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 144
// CHECK-NEXT:   %2163 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2162, i32 0, i32 0
// CHECK-NEXT:   %2164 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2165 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.440000e+02, ptr %2165, align 8
// CHECK-NEXT:   %2166 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2165, 1
// CHECK-NEXT:   %2167 = extractvalue { ptr, ptr } %2164, 1
// CHECK-NEXT:   %2168 = extractvalue { ptr, ptr } %2164, 0
// CHECK-NEXT:   %__llgo_funcval_code288 = call ptr asm "", "=r,0"(ptr %2168)
// CHECK-NEXT:   %2169 = call %reflect.Value %__llgo_funcval_code288(ptr {{(nest|swiftself)}} %2167, %"{{.*}}/runtime/internal/runtime.eface" %2166)
// CHECK-NEXT:   %2170 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2162, i32 0, i32 1
// CHECK-NEXT:   %2171 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2172 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 144, ptr %2172, align 8
// CHECK-NEXT:   %2173 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2172, 1
// CHECK-NEXT:   %2174 = extractvalue { ptr, ptr } %2171, 1
// CHECK-NEXT:   %2175 = extractvalue { ptr, ptr } %2171, 0
// CHECK-NEXT:   %__llgo_funcval_code289 = call ptr asm "", "=r,0"(ptr %2175)
// CHECK-NEXT:   %2176 = call %reflect.Value %__llgo_funcval_code289(ptr {{(nest|swiftself)}} %2174, %"{{.*}}/runtime/internal/runtime.eface" %2173)
// CHECK-NEXT:   store %reflect.Value %2169, ptr %2163, align 8
// CHECK-NEXT:   store %reflect.Value %2176, ptr %2170, align 8
// CHECK-NEXT:   %2177 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 145
// CHECK-NEXT:   %2178 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2177, i32 0, i32 0
// CHECK-NEXT:   %2179 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2180 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 145, ptr %2180, align 8
// CHECK-NEXT:   %2181 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2180, 1
// CHECK-NEXT:   %2182 = extractvalue { ptr, ptr } %2179, 1
// CHECK-NEXT:   %2183 = extractvalue { ptr, ptr } %2179, 0
// CHECK-NEXT:   %__llgo_funcval_code290 = call ptr asm "", "=r,0"(ptr %2183)
// CHECK-NEXT:   %2184 = call %reflect.Value %__llgo_funcval_code290(ptr {{(nest|swiftself)}} %2182, %"{{.*}}/runtime/internal/runtime.eface" %2181)
// CHECK-NEXT:   %2185 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2177, i32 0, i32 1
// CHECK-NEXT:   %2186 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2187 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 145, ptr %2187, align 8
// CHECK-NEXT:   %2188 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2187, 1
// CHECK-NEXT:   %2189 = extractvalue { ptr, ptr } %2186, 1
// CHECK-NEXT:   %2190 = extractvalue { ptr, ptr } %2186, 0
// CHECK-NEXT:   %__llgo_funcval_code291 = call ptr asm "", "=r,0"(ptr %2190)
// CHECK-NEXT:   %2191 = call %reflect.Value %__llgo_funcval_code291(ptr {{(nest|swiftself)}} %2189, %"{{.*}}/runtime/internal/runtime.eface" %2188)
// CHECK-NEXT:   store %reflect.Value %2184, ptr %2178, align 8
// CHECK-NEXT:   store %reflect.Value %2191, ptr %2185, align 8
// CHECK-NEXT:   %2192 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 146
// CHECK-NEXT:   %2193 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2192, i32 0, i32 0
// CHECK-NEXT:   %2194 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2195 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 146, ptr %2195, align 8
// CHECK-NEXT:   %2196 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2195, 1
// CHECK-NEXT:   %2197 = extractvalue { ptr, ptr } %2194, 1
// CHECK-NEXT:   %2198 = extractvalue { ptr, ptr } %2194, 0
// CHECK-NEXT:   %__llgo_funcval_code292 = call ptr asm "", "=r,0"(ptr %2198)
// CHECK-NEXT:   %2199 = call %reflect.Value %__llgo_funcval_code292(ptr {{(nest|swiftself)}} %2197, %"{{.*}}/runtime/internal/runtime.eface" %2196)
// CHECK-NEXT:   %2200 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2192, i32 0, i32 1
// CHECK-NEXT:   %2201 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2202 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 146, ptr %2202, align 8
// CHECK-NEXT:   %2203 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2202, 1
// CHECK-NEXT:   %2204 = extractvalue { ptr, ptr } %2201, 1
// CHECK-NEXT:   %2205 = extractvalue { ptr, ptr } %2201, 0
// CHECK-NEXT:   %__llgo_funcval_code293 = call ptr asm "", "=r,0"(ptr %2205)
// CHECK-NEXT:   %2206 = call %reflect.Value %__llgo_funcval_code293(ptr {{(nest|swiftself)}} %2204, %"{{.*}}/runtime/internal/runtime.eface" %2203)
// CHECK-NEXT:   store %reflect.Value %2199, ptr %2193, align 8
// CHECK-NEXT:   store %reflect.Value %2206, ptr %2200, align 8
// CHECK-NEXT:   %2207 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 147
// CHECK-NEXT:   %2208 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2207, i32 0, i32 0
// CHECK-NEXT:   %2209 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2210 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 147, ptr %2210, align 8
// CHECK-NEXT:   %2211 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2210, 1
// CHECK-NEXT:   %2212 = extractvalue { ptr, ptr } %2209, 1
// CHECK-NEXT:   %2213 = extractvalue { ptr, ptr } %2209, 0
// CHECK-NEXT:   %__llgo_funcval_code294 = call ptr asm "", "=r,0"(ptr %2213)
// CHECK-NEXT:   %2214 = call %reflect.Value %__llgo_funcval_code294(ptr {{(nest|swiftself)}} %2212, %"{{.*}}/runtime/internal/runtime.eface" %2211)
// CHECK-NEXT:   %2215 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2207, i32 0, i32 1
// CHECK-NEXT:   %2216 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2217 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 147, ptr %2217, align 8
// CHECK-NEXT:   %2218 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2217, 1
// CHECK-NEXT:   %2219 = extractvalue { ptr, ptr } %2216, 1
// CHECK-NEXT:   %2220 = extractvalue { ptr, ptr } %2216, 0
// CHECK-NEXT:   %__llgo_funcval_code295 = call ptr asm "", "=r,0"(ptr %2220)
// CHECK-NEXT:   %2221 = call %reflect.Value %__llgo_funcval_code295(ptr {{(nest|swiftself)}} %2219, %"{{.*}}/runtime/internal/runtime.eface" %2218)
// CHECK-NEXT:   store %reflect.Value %2214, ptr %2208, align 8
// CHECK-NEXT:   store %reflect.Value %2221, ptr %2215, align 8
// CHECK-NEXT:   %2222 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 148
// CHECK-NEXT:   %2223 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2222, i32 0, i32 0
// CHECK-NEXT:   %2224 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2225 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 148, ptr %2225, align 8
// CHECK-NEXT:   %2226 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2225, 1
// CHECK-NEXT:   %2227 = extractvalue { ptr, ptr } %2224, 1
// CHECK-NEXT:   %2228 = extractvalue { ptr, ptr } %2224, 0
// CHECK-NEXT:   %__llgo_funcval_code296 = call ptr asm "", "=r,0"(ptr %2228)
// CHECK-NEXT:   %2229 = call %reflect.Value %__llgo_funcval_code296(ptr {{(nest|swiftself)}} %2227, %"{{.*}}/runtime/internal/runtime.eface" %2226)
// CHECK-NEXT:   %2230 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2222, i32 0, i32 1
// CHECK-NEXT:   %2231 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2232 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 148, ptr %2232, align 8
// CHECK-NEXT:   %2233 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2232, 1
// CHECK-NEXT:   %2234 = extractvalue { ptr, ptr } %2231, 1
// CHECK-NEXT:   %2235 = extractvalue { ptr, ptr } %2231, 0
// CHECK-NEXT:   %__llgo_funcval_code297 = call ptr asm "", "=r,0"(ptr %2235)
// CHECK-NEXT:   %2236 = call %reflect.Value %__llgo_funcval_code297(ptr {{(nest|swiftself)}} %2234, %"{{.*}}/runtime/internal/runtime.eface" %2233)
// CHECK-NEXT:   store %reflect.Value %2229, ptr %2223, align 8
// CHECK-NEXT:   store %reflect.Value %2236, ptr %2230, align 8
// CHECK-NEXT:   %2237 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 149
// CHECK-NEXT:   %2238 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2237, i32 0, i32 0
// CHECK-NEXT:   %2239 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2240 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 149, ptr %2240, align 8
// CHECK-NEXT:   %2241 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2240, 1
// CHECK-NEXT:   %2242 = extractvalue { ptr, ptr } %2239, 1
// CHECK-NEXT:   %2243 = extractvalue { ptr, ptr } %2239, 0
// CHECK-NEXT:   %__llgo_funcval_code298 = call ptr asm "", "=r,0"(ptr %2243)
// CHECK-NEXT:   %2244 = call %reflect.Value %__llgo_funcval_code298(ptr {{(nest|swiftself)}} %2242, %"{{.*}}/runtime/internal/runtime.eface" %2241)
// CHECK-NEXT:   %2245 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2237, i32 0, i32 1
// CHECK-NEXT:   %2246 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2247 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 149, ptr %2247, align 8
// CHECK-NEXT:   %2248 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2247, 1
// CHECK-NEXT:   %2249 = extractvalue { ptr, ptr } %2246, 1
// CHECK-NEXT:   %2250 = extractvalue { ptr, ptr } %2246, 0
// CHECK-NEXT:   %__llgo_funcval_code299 = call ptr asm "", "=r,0"(ptr %2250)
// CHECK-NEXT:   %2251 = call %reflect.Value %__llgo_funcval_code299(ptr {{(nest|swiftself)}} %2249, %"{{.*}}/runtime/internal/runtime.eface" %2248)
// CHECK-NEXT:   store %reflect.Value %2244, ptr %2238, align 8
// CHECK-NEXT:   store %reflect.Value %2251, ptr %2245, align 8
// CHECK-NEXT:   %2252 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 150
// CHECK-NEXT:   %2253 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2252, i32 0, i32 0
// CHECK-NEXT:   %2254 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2255 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 150, ptr %2255, align 8
// CHECK-NEXT:   %2256 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2255, 1
// CHECK-NEXT:   %2257 = extractvalue { ptr, ptr } %2254, 1
// CHECK-NEXT:   %2258 = extractvalue { ptr, ptr } %2254, 0
// CHECK-NEXT:   %__llgo_funcval_code300 = call ptr asm "", "=r,0"(ptr %2258)
// CHECK-NEXT:   %2259 = call %reflect.Value %__llgo_funcval_code300(ptr {{(nest|swiftself)}} %2257, %"{{.*}}/runtime/internal/runtime.eface" %2256)
// CHECK-NEXT:   %2260 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2252, i32 0, i32 1
// CHECK-NEXT:   %2261 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2262 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.500000e+02, ptr %2262, align 4
// CHECK-NEXT:   %2263 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2262, 1
// CHECK-NEXT:   %2264 = extractvalue { ptr, ptr } %2261, 1
// CHECK-NEXT:   %2265 = extractvalue { ptr, ptr } %2261, 0
// CHECK-NEXT:   %__llgo_funcval_code301 = call ptr asm "", "=r,0"(ptr %2265)
// CHECK-NEXT:   %2266 = call %reflect.Value %__llgo_funcval_code301(ptr {{(nest|swiftself)}} %2264, %"{{.*}}/runtime/internal/runtime.eface" %2263)
// CHECK-NEXT:   store %reflect.Value %2259, ptr %2253, align 8
// CHECK-NEXT:   store %reflect.Value %2266, ptr %2260, align 8
// CHECK-NEXT:   %2267 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 151
// CHECK-NEXT:   %2268 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2267, i32 0, i32 0
// CHECK-NEXT:   %2269 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2270 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.510000e+02, ptr %2270, align 4
// CHECK-NEXT:   %2271 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2270, 1
// CHECK-NEXT:   %2272 = extractvalue { ptr, ptr } %2269, 1
// CHECK-NEXT:   %2273 = extractvalue { ptr, ptr } %2269, 0
// CHECK-NEXT:   %__llgo_funcval_code302 = call ptr asm "", "=r,0"(ptr %2273)
// CHECK-NEXT:   %2274 = call %reflect.Value %__llgo_funcval_code302(ptr {{(nest|swiftself)}} %2272, %"{{.*}}/runtime/internal/runtime.eface" %2271)
// CHECK-NEXT:   %2275 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2267, i32 0, i32 1
// CHECK-NEXT:   %2276 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2277 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 151, ptr %2277, align 8
// CHECK-NEXT:   %2278 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2277, 1
// CHECK-NEXT:   %2279 = extractvalue { ptr, ptr } %2276, 1
// CHECK-NEXT:   %2280 = extractvalue { ptr, ptr } %2276, 0
// CHECK-NEXT:   %__llgo_funcval_code303 = call ptr asm "", "=r,0"(ptr %2280)
// CHECK-NEXT:   %2281 = call %reflect.Value %__llgo_funcval_code303(ptr {{(nest|swiftself)}} %2279, %"{{.*}}/runtime/internal/runtime.eface" %2278)
// CHECK-NEXT:   store %reflect.Value %2274, ptr %2268, align 8
// CHECK-NEXT:   store %reflect.Value %2281, ptr %2275, align 8
// CHECK-NEXT:   %2282 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 152
// CHECK-NEXT:   %2283 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2282, i32 0, i32 0
// CHECK-NEXT:   %2284 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2285 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 152, ptr %2285, align 8
// CHECK-NEXT:   %2286 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2285, 1
// CHECK-NEXT:   %2287 = extractvalue { ptr, ptr } %2284, 1
// CHECK-NEXT:   %2288 = extractvalue { ptr, ptr } %2284, 0
// CHECK-NEXT:   %__llgo_funcval_code304 = call ptr asm "", "=r,0"(ptr %2288)
// CHECK-NEXT:   %2289 = call %reflect.Value %__llgo_funcval_code304(ptr {{(nest|swiftself)}} %2287, %"{{.*}}/runtime/internal/runtime.eface" %2286)
// CHECK-NEXT:   %2290 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2282, i32 0, i32 1
// CHECK-NEXT:   %2291 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2292 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.520000e+02, ptr %2292, align 8
// CHECK-NEXT:   %2293 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2292, 1
// CHECK-NEXT:   %2294 = extractvalue { ptr, ptr } %2291, 1
// CHECK-NEXT:   %2295 = extractvalue { ptr, ptr } %2291, 0
// CHECK-NEXT:   %__llgo_funcval_code305 = call ptr asm "", "=r,0"(ptr %2295)
// CHECK-NEXT:   %2296 = call %reflect.Value %__llgo_funcval_code305(ptr {{(nest|swiftself)}} %2294, %"{{.*}}/runtime/internal/runtime.eface" %2293)
// CHECK-NEXT:   store %reflect.Value %2289, ptr %2283, align 8
// CHECK-NEXT:   store %reflect.Value %2296, ptr %2290, align 8
// CHECK-NEXT:   %2297 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 153
// CHECK-NEXT:   %2298 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2297, i32 0, i32 0
// CHECK-NEXT:   %2299 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2300 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.530000e+02, ptr %2300, align 8
// CHECK-NEXT:   %2301 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2300, 1
// CHECK-NEXT:   %2302 = extractvalue { ptr, ptr } %2299, 1
// CHECK-NEXT:   %2303 = extractvalue { ptr, ptr } %2299, 0
// CHECK-NEXT:   %__llgo_funcval_code306 = call ptr asm "", "=r,0"(ptr %2303)
// CHECK-NEXT:   %2304 = call %reflect.Value %__llgo_funcval_code306(ptr {{(nest|swiftself)}} %2302, %"{{.*}}/runtime/internal/runtime.eface" %2301)
// CHECK-NEXT:   %2305 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2297, i32 0, i32 1
// CHECK-NEXT:   %2306 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2307 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 153, ptr %2307, align 8
// CHECK-NEXT:   %2308 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2307, 1
// CHECK-NEXT:   %2309 = extractvalue { ptr, ptr } %2306, 1
// CHECK-NEXT:   %2310 = extractvalue { ptr, ptr } %2306, 0
// CHECK-NEXT:   %__llgo_funcval_code307 = call ptr asm "", "=r,0"(ptr %2310)
// CHECK-NEXT:   %2311 = call %reflect.Value %__llgo_funcval_code307(ptr {{(nest|swiftself)}} %2309, %"{{.*}}/runtime/internal/runtime.eface" %2308)
// CHECK-NEXT:   store %reflect.Value %2304, ptr %2298, align 8
// CHECK-NEXT:   store %reflect.Value %2311, ptr %2305, align 8
// CHECK-NEXT:   %2312 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 154
// CHECK-NEXT:   %2313 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2312, i32 0, i32 0
// CHECK-NEXT:   %2314 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2315 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 154, ptr %2315, align 8
// CHECK-NEXT:   %2316 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2315, 1
// CHECK-NEXT:   %2317 = extractvalue { ptr, ptr } %2314, 1
// CHECK-NEXT:   %2318 = extractvalue { ptr, ptr } %2314, 0
// CHECK-NEXT:   %__llgo_funcval_code308 = call ptr asm "", "=r,0"(ptr %2318)
// CHECK-NEXT:   %2319 = call %reflect.Value %__llgo_funcval_code308(ptr {{(nest|swiftself)}} %2317, %"{{.*}}/runtime/internal/runtime.eface" %2316)
// CHECK-NEXT:   %2320 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2312, i32 0, i32 1
// CHECK-NEXT:   %2321 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2322 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 154, ptr %2322, align 8
// CHECK-NEXT:   %2323 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2322, 1
// CHECK-NEXT:   %2324 = extractvalue { ptr, ptr } %2321, 1
// CHECK-NEXT:   %2325 = extractvalue { ptr, ptr } %2321, 0
// CHECK-NEXT:   %__llgo_funcval_code309 = call ptr asm "", "=r,0"(ptr %2325)
// CHECK-NEXT:   %2326 = call %reflect.Value %__llgo_funcval_code309(ptr {{(nest|swiftself)}} %2324, %"{{.*}}/runtime/internal/runtime.eface" %2323)
// CHECK-NEXT:   store %reflect.Value %2319, ptr %2313, align 8
// CHECK-NEXT:   store %reflect.Value %2326, ptr %2320, align 8
// CHECK-NEXT:   %2327 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 155
// CHECK-NEXT:   %2328 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2327, i32 0, i32 0
// CHECK-NEXT:   %2329 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2330 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 155, ptr %2330, align 8
// CHECK-NEXT:   %2331 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2330, 1
// CHECK-NEXT:   %2332 = extractvalue { ptr, ptr } %2329, 1
// CHECK-NEXT:   %2333 = extractvalue { ptr, ptr } %2329, 0
// CHECK-NEXT:   %__llgo_funcval_code310 = call ptr asm "", "=r,0"(ptr %2333)
// CHECK-NEXT:   %2334 = call %reflect.Value %__llgo_funcval_code310(ptr {{(nest|swiftself)}} %2332, %"{{.*}}/runtime/internal/runtime.eface" %2331)
// CHECK-NEXT:   %2335 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2327, i32 0, i32 1
// CHECK-NEXT:   %2336 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2337 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 155, ptr %2337, align 8
// CHECK-NEXT:   %2338 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2337, 1
// CHECK-NEXT:   %2339 = extractvalue { ptr, ptr } %2336, 1
// CHECK-NEXT:   %2340 = extractvalue { ptr, ptr } %2336, 0
// CHECK-NEXT:   %__llgo_funcval_code311 = call ptr asm "", "=r,0"(ptr %2340)
// CHECK-NEXT:   %2341 = call %reflect.Value %__llgo_funcval_code311(ptr {{(nest|swiftself)}} %2339, %"{{.*}}/runtime/internal/runtime.eface" %2338)
// CHECK-NEXT:   store %reflect.Value %2334, ptr %2328, align 8
// CHECK-NEXT:   store %reflect.Value %2341, ptr %2335, align 8
// CHECK-NEXT:   %2342 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 156
// CHECK-NEXT:   %2343 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2342, i32 0, i32 0
// CHECK-NEXT:   %2344 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2345 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 156, ptr %2345, align 8
// CHECK-NEXT:   %2346 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2345, 1
// CHECK-NEXT:   %2347 = extractvalue { ptr, ptr } %2344, 1
// CHECK-NEXT:   %2348 = extractvalue { ptr, ptr } %2344, 0
// CHECK-NEXT:   %__llgo_funcval_code312 = call ptr asm "", "=r,0"(ptr %2348)
// CHECK-NEXT:   %2349 = call %reflect.Value %__llgo_funcval_code312(ptr {{(nest|swiftself)}} %2347, %"{{.*}}/runtime/internal/runtime.eface" %2346)
// CHECK-NEXT:   %2350 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2342, i32 0, i32 1
// CHECK-NEXT:   %2351 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2352 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 156, ptr %2352, align 8
// CHECK-NEXT:   %2353 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2352, 1
// CHECK-NEXT:   %2354 = extractvalue { ptr, ptr } %2351, 1
// CHECK-NEXT:   %2355 = extractvalue { ptr, ptr } %2351, 0
// CHECK-NEXT:   %__llgo_funcval_code313 = call ptr asm "", "=r,0"(ptr %2355)
// CHECK-NEXT:   %2356 = call %reflect.Value %__llgo_funcval_code313(ptr {{(nest|swiftself)}} %2354, %"{{.*}}/runtime/internal/runtime.eface" %2353)
// CHECK-NEXT:   store %reflect.Value %2349, ptr %2343, align 8
// CHECK-NEXT:   store %reflect.Value %2356, ptr %2350, align 8
// CHECK-NEXT:   %2357 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 157
// CHECK-NEXT:   %2358 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2357, i32 0, i32 0
// CHECK-NEXT:   %2359 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2360 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 157, ptr %2360, align 8
// CHECK-NEXT:   %2361 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2360, 1
// CHECK-NEXT:   %2362 = extractvalue { ptr, ptr } %2359, 1
// CHECK-NEXT:   %2363 = extractvalue { ptr, ptr } %2359, 0
// CHECK-NEXT:   %__llgo_funcval_code314 = call ptr asm "", "=r,0"(ptr %2363)
// CHECK-NEXT:   %2364 = call %reflect.Value %__llgo_funcval_code314(ptr {{(nest|swiftself)}} %2362, %"{{.*}}/runtime/internal/runtime.eface" %2361)
// CHECK-NEXT:   %2365 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2357, i32 0, i32 1
// CHECK-NEXT:   %2366 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2367 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.570000e+02, ptr %2367, align 4
// CHECK-NEXT:   %2368 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2367, 1
// CHECK-NEXT:   %2369 = extractvalue { ptr, ptr } %2366, 1
// CHECK-NEXT:   %2370 = extractvalue { ptr, ptr } %2366, 0
// CHECK-NEXT:   %__llgo_funcval_code315 = call ptr asm "", "=r,0"(ptr %2370)
// CHECK-NEXT:   %2371 = call %reflect.Value %__llgo_funcval_code315(ptr {{(nest|swiftself)}} %2369, %"{{.*}}/runtime/internal/runtime.eface" %2368)
// CHECK-NEXT:   store %reflect.Value %2364, ptr %2358, align 8
// CHECK-NEXT:   store %reflect.Value %2371, ptr %2365, align 8
// CHECK-NEXT:   %2372 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 158
// CHECK-NEXT:   %2373 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2372, i32 0, i32 0
// CHECK-NEXT:   %2374 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2375 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.580000e+02, ptr %2375, align 4
// CHECK-NEXT:   %2376 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2375, 1
// CHECK-NEXT:   %2377 = extractvalue { ptr, ptr } %2374, 1
// CHECK-NEXT:   %2378 = extractvalue { ptr, ptr } %2374, 0
// CHECK-NEXT:   %__llgo_funcval_code316 = call ptr asm "", "=r,0"(ptr %2378)
// CHECK-NEXT:   %2379 = call %reflect.Value %__llgo_funcval_code316(ptr {{(nest|swiftself)}} %2377, %"{{.*}}/runtime/internal/runtime.eface" %2376)
// CHECK-NEXT:   %2380 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2372, i32 0, i32 1
// CHECK-NEXT:   %2381 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2382 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 158, ptr %2382, align 8
// CHECK-NEXT:   %2383 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2382, 1
// CHECK-NEXT:   %2384 = extractvalue { ptr, ptr } %2381, 1
// CHECK-NEXT:   %2385 = extractvalue { ptr, ptr } %2381, 0
// CHECK-NEXT:   %__llgo_funcval_code317 = call ptr asm "", "=r,0"(ptr %2385)
// CHECK-NEXT:   %2386 = call %reflect.Value %__llgo_funcval_code317(ptr {{(nest|swiftself)}} %2384, %"{{.*}}/runtime/internal/runtime.eface" %2383)
// CHECK-NEXT:   store %reflect.Value %2379, ptr %2373, align 8
// CHECK-NEXT:   store %reflect.Value %2386, ptr %2380, align 8
// CHECK-NEXT:   %2387 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 159
// CHECK-NEXT:   %2388 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2387, i32 0, i32 0
// CHECK-NEXT:   %2389 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2390 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 159, ptr %2390, align 8
// CHECK-NEXT:   %2391 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2390, 1
// CHECK-NEXT:   %2392 = extractvalue { ptr, ptr } %2389, 1
// CHECK-NEXT:   %2393 = extractvalue { ptr, ptr } %2389, 0
// CHECK-NEXT:   %__llgo_funcval_code318 = call ptr asm "", "=r,0"(ptr %2393)
// CHECK-NEXT:   %2394 = call %reflect.Value %__llgo_funcval_code318(ptr {{(nest|swiftself)}} %2392, %"{{.*}}/runtime/internal/runtime.eface" %2391)
// CHECK-NEXT:   %2395 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2387, i32 0, i32 1
// CHECK-NEXT:   %2396 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2397 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.590000e+02, ptr %2397, align 8
// CHECK-NEXT:   %2398 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2397, 1
// CHECK-NEXT:   %2399 = extractvalue { ptr, ptr } %2396, 1
// CHECK-NEXT:   %2400 = extractvalue { ptr, ptr } %2396, 0
// CHECK-NEXT:   %__llgo_funcval_code319 = call ptr asm "", "=r,0"(ptr %2400)
// CHECK-NEXT:   %2401 = call %reflect.Value %__llgo_funcval_code319(ptr {{(nest|swiftself)}} %2399, %"{{.*}}/runtime/internal/runtime.eface" %2398)
// CHECK-NEXT:   store %reflect.Value %2394, ptr %2388, align 8
// CHECK-NEXT:   store %reflect.Value %2401, ptr %2395, align 8
// CHECK-NEXT:   %2402 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 160
// CHECK-NEXT:   %2403 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2402, i32 0, i32 0
// CHECK-NEXT:   %2404 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2405 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.600000e+02, ptr %2405, align 8
// CHECK-NEXT:   %2406 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2405, 1
// CHECK-NEXT:   %2407 = extractvalue { ptr, ptr } %2404, 1
// CHECK-NEXT:   %2408 = extractvalue { ptr, ptr } %2404, 0
// CHECK-NEXT:   %__llgo_funcval_code320 = call ptr asm "", "=r,0"(ptr %2408)
// CHECK-NEXT:   %2409 = call %reflect.Value %__llgo_funcval_code320(ptr {{(nest|swiftself)}} %2407, %"{{.*}}/runtime/internal/runtime.eface" %2406)
// CHECK-NEXT:   %2410 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2402, i32 0, i32 1
// CHECK-NEXT:   %2411 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2412 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 160, ptr %2412, align 8
// CHECK-NEXT:   %2413 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2412, 1
// CHECK-NEXT:   %2414 = extractvalue { ptr, ptr } %2411, 1
// CHECK-NEXT:   %2415 = extractvalue { ptr, ptr } %2411, 0
// CHECK-NEXT:   %__llgo_funcval_code321 = call ptr asm "", "=r,0"(ptr %2415)
// CHECK-NEXT:   %2416 = call %reflect.Value %__llgo_funcval_code321(ptr {{(nest|swiftself)}} %2414, %"{{.*}}/runtime/internal/runtime.eface" %2413)
// CHECK-NEXT:   store %reflect.Value %2409, ptr %2403, align 8
// CHECK-NEXT:   store %reflect.Value %2416, ptr %2410, align 8
// CHECK-NEXT:   %2417 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 161
// CHECK-NEXT:   %2418 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2417, i32 0, i32 0
// CHECK-NEXT:   %2419 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2420 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 161, ptr %2420, align 8
// CHECK-NEXT:   %2421 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2420, 1
// CHECK-NEXT:   %2422 = extractvalue { ptr, ptr } %2419, 1
// CHECK-NEXT:   %2423 = extractvalue { ptr, ptr } %2419, 0
// CHECK-NEXT:   %__llgo_funcval_code322 = call ptr asm "", "=r,0"(ptr %2423)
// CHECK-NEXT:   %2424 = call %reflect.Value %__llgo_funcval_code322(ptr {{(nest|swiftself)}} %2422, %"{{.*}}/runtime/internal/runtime.eface" %2421)
// CHECK-NEXT:   %2425 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2417, i32 0, i32 1
// CHECK-NEXT:   %2426 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2427 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 161, ptr %2427, align 8
// CHECK-NEXT:   %2428 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2427, 1
// CHECK-NEXT:   %2429 = extractvalue { ptr, ptr } %2426, 1
// CHECK-NEXT:   %2430 = extractvalue { ptr, ptr } %2426, 0
// CHECK-NEXT:   %__llgo_funcval_code323 = call ptr asm "", "=r,0"(ptr %2430)
// CHECK-NEXT:   %2431 = call %reflect.Value %__llgo_funcval_code323(ptr {{(nest|swiftself)}} %2429, %"{{.*}}/runtime/internal/runtime.eface" %2428)
// CHECK-NEXT:   store %reflect.Value %2424, ptr %2418, align 8
// CHECK-NEXT:   store %reflect.Value %2431, ptr %2425, align 8
// CHECK-NEXT:   %2432 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 162
// CHECK-NEXT:   %2433 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2432, i32 0, i32 0
// CHECK-NEXT:   %2434 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2435 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 162, ptr %2435, align 8
// CHECK-NEXT:   %2436 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2435, 1
// CHECK-NEXT:   %2437 = extractvalue { ptr, ptr } %2434, 1
// CHECK-NEXT:   %2438 = extractvalue { ptr, ptr } %2434, 0
// CHECK-NEXT:   %__llgo_funcval_code324 = call ptr asm "", "=r,0"(ptr %2438)
// CHECK-NEXT:   %2439 = call %reflect.Value %__llgo_funcval_code324(ptr {{(nest|swiftself)}} %2437, %"{{.*}}/runtime/internal/runtime.eface" %2436)
// CHECK-NEXT:   %2440 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2432, i32 0, i32 1
// CHECK-NEXT:   %2441 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2442 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.620000e+02, ptr %2442, align 4
// CHECK-NEXT:   %2443 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2442, 1
// CHECK-NEXT:   %2444 = extractvalue { ptr, ptr } %2441, 1
// CHECK-NEXT:   %2445 = extractvalue { ptr, ptr } %2441, 0
// CHECK-NEXT:   %__llgo_funcval_code325 = call ptr asm "", "=r,0"(ptr %2445)
// CHECK-NEXT:   %2446 = call %reflect.Value %__llgo_funcval_code325(ptr {{(nest|swiftself)}} %2444, %"{{.*}}/runtime/internal/runtime.eface" %2443)
// CHECK-NEXT:   store %reflect.Value %2439, ptr %2433, align 8
// CHECK-NEXT:   store %reflect.Value %2446, ptr %2440, align 8
// CHECK-NEXT:   %2447 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 163
// CHECK-NEXT:   %2448 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2447, i32 0, i32 0
// CHECK-NEXT:   %2449 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2450 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.630000e+02, ptr %2450, align 4
// CHECK-NEXT:   %2451 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2450, 1
// CHECK-NEXT:   %2452 = extractvalue { ptr, ptr } %2449, 1
// CHECK-NEXT:   %2453 = extractvalue { ptr, ptr } %2449, 0
// CHECK-NEXT:   %__llgo_funcval_code326 = call ptr asm "", "=r,0"(ptr %2453)
// CHECK-NEXT:   %2454 = call %reflect.Value %__llgo_funcval_code326(ptr {{(nest|swiftself)}} %2452, %"{{.*}}/runtime/internal/runtime.eface" %2451)
// CHECK-NEXT:   %2455 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2447, i32 0, i32 1
// CHECK-NEXT:   %2456 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2457 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 163, ptr %2457, align 8
// CHECK-NEXT:   %2458 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2457, 1
// CHECK-NEXT:   %2459 = extractvalue { ptr, ptr } %2456, 1
// CHECK-NEXT:   %2460 = extractvalue { ptr, ptr } %2456, 0
// CHECK-NEXT:   %__llgo_funcval_code327 = call ptr asm "", "=r,0"(ptr %2460)
// CHECK-NEXT:   %2461 = call %reflect.Value %__llgo_funcval_code327(ptr {{(nest|swiftself)}} %2459, %"{{.*}}/runtime/internal/runtime.eface" %2458)
// CHECK-NEXT:   store %reflect.Value %2454, ptr %2448, align 8
// CHECK-NEXT:   store %reflect.Value %2461, ptr %2455, align 8
// CHECK-NEXT:   %2462 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 164
// CHECK-NEXT:   %2463 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2462, i32 0, i32 0
// CHECK-NEXT:   %2464 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2465 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 164, ptr %2465, align 8
// CHECK-NEXT:   %2466 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2465, 1
// CHECK-NEXT:   %2467 = extractvalue { ptr, ptr } %2464, 1
// CHECK-NEXT:   %2468 = extractvalue { ptr, ptr } %2464, 0
// CHECK-NEXT:   %__llgo_funcval_code328 = call ptr asm "", "=r,0"(ptr %2468)
// CHECK-NEXT:   %2469 = call %reflect.Value %__llgo_funcval_code328(ptr {{(nest|swiftself)}} %2467, %"{{.*}}/runtime/internal/runtime.eface" %2466)
// CHECK-NEXT:   %2470 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2462, i32 0, i32 1
// CHECK-NEXT:   %2471 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2472 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.640000e+02, ptr %2472, align 8
// CHECK-NEXT:   %2473 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2472, 1
// CHECK-NEXT:   %2474 = extractvalue { ptr, ptr } %2471, 1
// CHECK-NEXT:   %2475 = extractvalue { ptr, ptr } %2471, 0
// CHECK-NEXT:   %__llgo_funcval_code329 = call ptr asm "", "=r,0"(ptr %2475)
// CHECK-NEXT:   %2476 = call %reflect.Value %__llgo_funcval_code329(ptr {{(nest|swiftself)}} %2474, %"{{.*}}/runtime/internal/runtime.eface" %2473)
// CHECK-NEXT:   store %reflect.Value %2469, ptr %2463, align 8
// CHECK-NEXT:   store %reflect.Value %2476, ptr %2470, align 8
// CHECK-NEXT:   %2477 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 165
// CHECK-NEXT:   %2478 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2477, i32 0, i32 0
// CHECK-NEXT:   %2479 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2480 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.650000e+02, ptr %2480, align 8
// CHECK-NEXT:   %2481 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2480, 1
// CHECK-NEXT:   %2482 = extractvalue { ptr, ptr } %2479, 1
// CHECK-NEXT:   %2483 = extractvalue { ptr, ptr } %2479, 0
// CHECK-NEXT:   %__llgo_funcval_code330 = call ptr asm "", "=r,0"(ptr %2483)
// CHECK-NEXT:   %2484 = call %reflect.Value %__llgo_funcval_code330(ptr {{(nest|swiftself)}} %2482, %"{{.*}}/runtime/internal/runtime.eface" %2481)
// CHECK-NEXT:   %2485 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2477, i32 0, i32 1
// CHECK-NEXT:   %2486 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2487 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 165, ptr %2487, align 8
// CHECK-NEXT:   %2488 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2487, 1
// CHECK-NEXT:   %2489 = extractvalue { ptr, ptr } %2486, 1
// CHECK-NEXT:   %2490 = extractvalue { ptr, ptr } %2486, 0
// CHECK-NEXT:   %__llgo_funcval_code331 = call ptr asm "", "=r,0"(ptr %2490)
// CHECK-NEXT:   %2491 = call %reflect.Value %__llgo_funcval_code331(ptr {{(nest|swiftself)}} %2489, %"{{.*}}/runtime/internal/runtime.eface" %2488)
// CHECK-NEXT:   store %reflect.Value %2484, ptr %2478, align 8
// CHECK-NEXT:   store %reflect.Value %2491, ptr %2485, align 8
// CHECK-NEXT:   %2492 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 166
// CHECK-NEXT:   %2493 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2492, i32 0, i32 0
// CHECK-NEXT:   %2494 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2495 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.660000e+02, ptr %2495, align 4
// CHECK-NEXT:   %2496 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2495, 1
// CHECK-NEXT:   %2497 = extractvalue { ptr, ptr } %2494, 1
// CHECK-NEXT:   %2498 = extractvalue { ptr, ptr } %2494, 0
// CHECK-NEXT:   %__llgo_funcval_code332 = call ptr asm "", "=r,0"(ptr %2498)
// CHECK-NEXT:   %2499 = call %reflect.Value %__llgo_funcval_code332(ptr {{(nest|swiftself)}} %2497, %"{{.*}}/runtime/internal/runtime.eface" %2496)
// CHECK-NEXT:   %2500 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2492, i32 0, i32 1
// CHECK-NEXT:   %2501 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2502 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.660000e+02, ptr %2502, align 4
// CHECK-NEXT:   %2503 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2502, 1
// CHECK-NEXT:   %2504 = extractvalue { ptr, ptr } %2501, 1
// CHECK-NEXT:   %2505 = extractvalue { ptr, ptr } %2501, 0
// CHECK-NEXT:   %__llgo_funcval_code333 = call ptr asm "", "=r,0"(ptr %2505)
// CHECK-NEXT:   %2506 = call %reflect.Value %__llgo_funcval_code333(ptr {{(nest|swiftself)}} %2504, %"{{.*}}/runtime/internal/runtime.eface" %2503)
// CHECK-NEXT:   store %reflect.Value %2499, ptr %2493, align 8
// CHECK-NEXT:   store %reflect.Value %2506, ptr %2500, align 8
// CHECK-NEXT:   %2507 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 167
// CHECK-NEXT:   %2508 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2507, i32 0, i32 0
// CHECK-NEXT:   %2509 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2510 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.670000e+02, ptr %2510, align 4
// CHECK-NEXT:   %2511 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2510, 1
// CHECK-NEXT:   %2512 = extractvalue { ptr, ptr } %2509, 1
// CHECK-NEXT:   %2513 = extractvalue { ptr, ptr } %2509, 0
// CHECK-NEXT:   %__llgo_funcval_code334 = call ptr asm "", "=r,0"(ptr %2513)
// CHECK-NEXT:   %2514 = call %reflect.Value %__llgo_funcval_code334(ptr {{(nest|swiftself)}} %2512, %"{{.*}}/runtime/internal/runtime.eface" %2511)
// CHECK-NEXT:   %2515 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2507, i32 0, i32 1
// CHECK-NEXT:   %2516 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2517 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.670000e+02, ptr %2517, align 8
// CHECK-NEXT:   %2518 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2517, 1
// CHECK-NEXT:   %2519 = extractvalue { ptr, ptr } %2516, 1
// CHECK-NEXT:   %2520 = extractvalue { ptr, ptr } %2516, 0
// CHECK-NEXT:   %__llgo_funcval_code335 = call ptr asm "", "=r,0"(ptr %2520)
// CHECK-NEXT:   %2521 = call %reflect.Value %__llgo_funcval_code335(ptr {{(nest|swiftself)}} %2519, %"{{.*}}/runtime/internal/runtime.eface" %2518)
// CHECK-NEXT:   store %reflect.Value %2514, ptr %2508, align 8
// CHECK-NEXT:   store %reflect.Value %2521, ptr %2515, align 8
// CHECK-NEXT:   %2522 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 168
// CHECK-NEXT:   %2523 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2522, i32 0, i32 0
// CHECK-NEXT:   %2524 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2525 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.680000e+02, ptr %2525, align 8
// CHECK-NEXT:   %2526 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2525, 1
// CHECK-NEXT:   %2527 = extractvalue { ptr, ptr } %2524, 1
// CHECK-NEXT:   %2528 = extractvalue { ptr, ptr } %2524, 0
// CHECK-NEXT:   %__llgo_funcval_code336 = call ptr asm "", "=r,0"(ptr %2528)
// CHECK-NEXT:   %2529 = call %reflect.Value %__llgo_funcval_code336(ptr {{(nest|swiftself)}} %2527, %"{{.*}}/runtime/internal/runtime.eface" %2526)
// CHECK-NEXT:   %2530 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2522, i32 0, i32 1
// CHECK-NEXT:   %2531 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2532 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store float 1.680000e+02, ptr %2532, align 4
// CHECK-NEXT:   %2533 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float32, ptr undef }, ptr %2532, 1
// CHECK-NEXT:   %2534 = extractvalue { ptr, ptr } %2531, 1
// CHECK-NEXT:   %2535 = extractvalue { ptr, ptr } %2531, 0
// CHECK-NEXT:   %__llgo_funcval_code337 = call ptr asm "", "=r,0"(ptr %2535)
// CHECK-NEXT:   %2536 = call %reflect.Value %__llgo_funcval_code337(ptr {{(nest|swiftself)}} %2534, %"{{.*}}/runtime/internal/runtime.eface" %2533)
// CHECK-NEXT:   store %reflect.Value %2529, ptr %2523, align 8
// CHECK-NEXT:   store %reflect.Value %2536, ptr %2530, align 8
// CHECK-NEXT:   %2537 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 169
// CHECK-NEXT:   %2538 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2537, i32 0, i32 0
// CHECK-NEXT:   %2539 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2540 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.690000e+02, ptr %2540, align 8
// CHECK-NEXT:   %2541 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2540, 1
// CHECK-NEXT:   %2542 = extractvalue { ptr, ptr } %2539, 1
// CHECK-NEXT:   %2543 = extractvalue { ptr, ptr } %2539, 0
// CHECK-NEXT:   %__llgo_funcval_code338 = call ptr asm "", "=r,0"(ptr %2543)
// CHECK-NEXT:   %2544 = call %reflect.Value %__llgo_funcval_code338(ptr {{(nest|swiftself)}} %2542, %"{{.*}}/runtime/internal/runtime.eface" %2541)
// CHECK-NEXT:   %2545 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2537, i32 0, i32 1
// CHECK-NEXT:   %2546 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2547 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.690000e+02, ptr %2547, align 8
// CHECK-NEXT:   %2548 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2547, 1
// CHECK-NEXT:   %2549 = extractvalue { ptr, ptr } %2546, 1
// CHECK-NEXT:   %2550 = extractvalue { ptr, ptr } %2546, 0
// CHECK-NEXT:   %__llgo_funcval_code339 = call ptr asm "", "=r,0"(ptr %2550)
// CHECK-NEXT:   %2551 = call %reflect.Value %__llgo_funcval_code339(ptr {{(nest|swiftself)}} %2549, %"{{.*}}/runtime/internal/runtime.eface" %2548)
// CHECK-NEXT:   store %reflect.Value %2544, ptr %2538, align 8
// CHECK-NEXT:   store %reflect.Value %2551, ptr %2545, align 8
// CHECK-NEXT:   %2552 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 170
// CHECK-NEXT:   %2553 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2552, i32 0, i32 0
// CHECK-NEXT:   %2554 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2555 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store double 1.500000e+00, ptr %2555, align 8
// CHECK-NEXT:   %2556 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_float64, ptr undef }, ptr %2555, 1
// CHECK-NEXT:   %2557 = extractvalue { ptr, ptr } %2554, 1
// CHECK-NEXT:   %2558 = extractvalue { ptr, ptr } %2554, 0
// CHECK-NEXT:   %__llgo_funcval_code340 = call ptr asm "", "=r,0"(ptr %2558)
// CHECK-NEXT:   %2559 = call %reflect.Value %__llgo_funcval_code340(ptr {{(nest|swiftself)}} %2557, %"{{.*}}/runtime/internal/runtime.eface" %2556)
// CHECK-NEXT:   %2560 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2552, i32 0, i32 1
// CHECK-NEXT:   %2561 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2562 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %2562, align 8
// CHECK-NEXT:   %2563 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2562, 1
// CHECK-NEXT:   %2564 = extractvalue { ptr, ptr } %2561, 1
// CHECK-NEXT:   %2565 = extractvalue { ptr, ptr } %2561, 0
// CHECK-NEXT:   %__llgo_funcval_code341 = call ptr asm "", "=r,0"(ptr %2565)
// CHECK-NEXT:   %2566 = call %reflect.Value %__llgo_funcval_code341(ptr {{(nest|swiftself)}} %2564, %"{{.*}}/runtime/internal/runtime.eface" %2563)
// CHECK-NEXT:   store %reflect.Value %2559, ptr %2553, align 8
// CHECK-NEXT:   store %reflect.Value %2566, ptr %2560, align 8
// CHECK-NEXT:   %2567 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 171
// CHECK-NEXT:   %2568 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2567, i32 0, i32 0
// CHECK-NEXT:   %2569 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2570 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { float, float } { float 0.000000e+00, float 1.000000e+00 }, ptr %2570, align 4
// CHECK-NEXT:   %2571 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex64, ptr undef }, ptr %2570, 1
// CHECK-NEXT:   %2572 = extractvalue { ptr, ptr } %2569, 1
// CHECK-NEXT:   %2573 = extractvalue { ptr, ptr } %2569, 0
// CHECK-NEXT:   %__llgo_funcval_code342 = call ptr asm "", "=r,0"(ptr %2573)
// CHECK-NEXT:   %2574 = call %reflect.Value %__llgo_funcval_code342(ptr {{(nest|swiftself)}} %2572, %"{{.*}}/runtime/internal/runtime.eface" %2571)
// CHECK-NEXT:   %2575 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2567, i32 0, i32 1
// CHECK-NEXT:   %2576 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2577 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { float, float } { float 0.000000e+00, float 1.000000e+00 }, ptr %2577, align 4
// CHECK-NEXT:   %2578 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex64, ptr undef }, ptr %2577, 1
// CHECK-NEXT:   %2579 = extractvalue { ptr, ptr } %2576, 1
// CHECK-NEXT:   %2580 = extractvalue { ptr, ptr } %2576, 0
// CHECK-NEXT:   %__llgo_funcval_code343 = call ptr asm "", "=r,0"(ptr %2580)
// CHECK-NEXT:   %2581 = call %reflect.Value %__llgo_funcval_code343(ptr {{(nest|swiftself)}} %2579, %"{{.*}}/runtime/internal/runtime.eface" %2578)
// CHECK-NEXT:   store %reflect.Value %2574, ptr %2568, align 8
// CHECK-NEXT:   store %reflect.Value %2581, ptr %2575, align 8
// CHECK-NEXT:   %2582 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 172
// CHECK-NEXT:   %2583 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2582, i32 0, i32 0
// CHECK-NEXT:   %2584 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2585 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { float, float } { float 0.000000e+00, float 2.000000e+00 }, ptr %2585, align 4
// CHECK-NEXT:   %2586 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex64, ptr undef }, ptr %2585, 1
// CHECK-NEXT:   %2587 = extractvalue { ptr, ptr } %2584, 1
// CHECK-NEXT:   %2588 = extractvalue { ptr, ptr } %2584, 0
// CHECK-NEXT:   %__llgo_funcval_code344 = call ptr asm "", "=r,0"(ptr %2588)
// CHECK-NEXT:   %2589 = call %reflect.Value %__llgo_funcval_code344(ptr {{(nest|swiftself)}} %2587, %"{{.*}}/runtime/internal/runtime.eface" %2586)
// CHECK-NEXT:   %2590 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2582, i32 0, i32 1
// CHECK-NEXT:   %2591 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2592 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { double, double } { double 0.000000e+00, double 2.000000e+00 }, ptr %2592, align 8
// CHECK-NEXT:   %2593 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex128, ptr undef }, ptr %2592, 1
// CHECK-NEXT:   %2594 = extractvalue { ptr, ptr } %2591, 1
// CHECK-NEXT:   %2595 = extractvalue { ptr, ptr } %2591, 0
// CHECK-NEXT:   %__llgo_funcval_code345 = call ptr asm "", "=r,0"(ptr %2595)
// CHECK-NEXT:   %2596 = call %reflect.Value %__llgo_funcval_code345(ptr {{(nest|swiftself)}} %2594, %"{{.*}}/runtime/internal/runtime.eface" %2593)
// CHECK-NEXT:   store %reflect.Value %2589, ptr %2583, align 8
// CHECK-NEXT:   store %reflect.Value %2596, ptr %2590, align 8
// CHECK-NEXT:   %2597 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 173
// CHECK-NEXT:   %2598 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2597, i32 0, i32 0
// CHECK-NEXT:   %2599 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2600 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { double, double } { double 0.000000e+00, double 3.000000e+00 }, ptr %2600, align 8
// CHECK-NEXT:   %2601 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex128, ptr undef }, ptr %2600, 1
// CHECK-NEXT:   %2602 = extractvalue { ptr, ptr } %2599, 1
// CHECK-NEXT:   %2603 = extractvalue { ptr, ptr } %2599, 0
// CHECK-NEXT:   %__llgo_funcval_code346 = call ptr asm "", "=r,0"(ptr %2603)
// CHECK-NEXT:   %2604 = call %reflect.Value %__llgo_funcval_code346(ptr {{(nest|swiftself)}} %2602, %"{{.*}}/runtime/internal/runtime.eface" %2601)
// CHECK-NEXT:   %2605 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2597, i32 0, i32 1
// CHECK-NEXT:   %2606 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2607 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { float, float } { float 0.000000e+00, float 3.000000e+00 }, ptr %2607, align 4
// CHECK-NEXT:   %2608 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex64, ptr undef }, ptr %2607, 1
// CHECK-NEXT:   %2609 = extractvalue { ptr, ptr } %2606, 1
// CHECK-NEXT:   %2610 = extractvalue { ptr, ptr } %2606, 0
// CHECK-NEXT:   %__llgo_funcval_code347 = call ptr asm "", "=r,0"(ptr %2610)
// CHECK-NEXT:   %2611 = call %reflect.Value %__llgo_funcval_code347(ptr {{(nest|swiftself)}} %2609, %"{{.*}}/runtime/internal/runtime.eface" %2608)
// CHECK-NEXT:   store %reflect.Value %2604, ptr %2598, align 8
// CHECK-NEXT:   store %reflect.Value %2611, ptr %2605, align 8
// CHECK-NEXT:   %2612 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 174
// CHECK-NEXT:   %2613 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2612, i32 0, i32 0
// CHECK-NEXT:   %2614 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2615 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { double, double } { double 0.000000e+00, double 4.000000e+00 }, ptr %2615, align 8
// CHECK-NEXT:   %2616 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex128, ptr undef }, ptr %2615, 1
// CHECK-NEXT:   %2617 = extractvalue { ptr, ptr } %2614, 1
// CHECK-NEXT:   %2618 = extractvalue { ptr, ptr } %2614, 0
// CHECK-NEXT:   %__llgo_funcval_code348 = call ptr asm "", "=r,0"(ptr %2618)
// CHECK-NEXT:   %2619 = call %reflect.Value %__llgo_funcval_code348(ptr {{(nest|swiftself)}} %2617, %"{{.*}}/runtime/internal/runtime.eface" %2616)
// CHECK-NEXT:   %2620 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2612, i32 0, i32 1
// CHECK-NEXT:   %2621 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2622 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { double, double } { double 0.000000e+00, double 4.000000e+00 }, ptr %2622, align 8
// CHECK-NEXT:   %2623 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_complex128, ptr undef }, ptr %2622, 1
// CHECK-NEXT:   %2624 = extractvalue { ptr, ptr } %2621, 1
// CHECK-NEXT:   %2625 = extractvalue { ptr, ptr } %2621, 0
// CHECK-NEXT:   %__llgo_funcval_code349 = call ptr asm "", "=r,0"(ptr %2625)
// CHECK-NEXT:   %2626 = call %reflect.Value %__llgo_funcval_code349(ptr {{(nest|swiftself)}} %2624, %"{{.*}}/runtime/internal/runtime.eface" %2623)
// CHECK-NEXT:   store %reflect.Value %2619, ptr %2613, align 8
// CHECK-NEXT:   store %reflect.Value %2626, ptr %2620, align 8
// CHECK-NEXT:   %2627 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 175
// CHECK-NEXT:   %2628 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2627, i32 0, i32 0
// CHECK-NEXT:   %2629 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2630 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %2630, align 8
// CHECK-NEXT:   %2631 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2630, 1
// CHECK-NEXT:   %2632 = extractvalue { ptr, ptr } %2629, 1
// CHECK-NEXT:   %2633 = extractvalue { ptr, ptr } %2629, 0
// CHECK-NEXT:   %__llgo_funcval_code350 = call ptr asm "", "=r,0"(ptr %2633)
// CHECK-NEXT:   %2634 = call %reflect.Value %__llgo_funcval_code350(ptr {{(nest|swiftself)}} %2632, %"{{.*}}/runtime/internal/runtime.eface" %2631)
// CHECK-NEXT:   %2635 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2627, i32 0, i32 1
// CHECK-NEXT:   %2636 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2637 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %2637, align 8
// CHECK-NEXT:   %2638 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2637, 1
// CHECK-NEXT:   %2639 = extractvalue { ptr, ptr } %2636, 1
// CHECK-NEXT:   %2640 = extractvalue { ptr, ptr } %2636, 0
// CHECK-NEXT:   %__llgo_funcval_code351 = call ptr asm "", "=r,0"(ptr %2640)
// CHECK-NEXT:   %2641 = call %reflect.Value %__llgo_funcval_code351(ptr {{(nest|swiftself)}} %2639, %"{{.*}}/runtime/internal/runtime.eface" %2638)
// CHECK-NEXT:   store %reflect.Value %2634, ptr %2628, align 8
// CHECK-NEXT:   store %reflect.Value %2641, ptr %2635, align 8
// CHECK-NEXT:   %2642 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 176
// CHECK-NEXT:   %2643 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2642, i32 0, i32 0
// CHECK-NEXT:   %2644 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2645 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %2645, align 8
// CHECK-NEXT:   %2646 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2645, 1
// CHECK-NEXT:   %2647 = extractvalue { ptr, ptr } %2644, 1
// CHECK-NEXT:   %2648 = extractvalue { ptr, ptr } %2644, 0
// CHECK-NEXT:   %__llgo_funcval_code352 = call ptr asm "", "=r,0"(ptr %2648)
// CHECK-NEXT:   %2649 = call %reflect.Value %__llgo_funcval_code352(ptr {{(nest|swiftself)}} %2647, %"{{.*}}/runtime/internal/runtime.eface" %2646)
// CHECK-NEXT:   %2650 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2642, i32 0, i32 1
// CHECK-NEXT:   %2651 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2652 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %2653 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2652, ptr %2653, align 8
// CHECK-NEXT:   %2654 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %2653, 1
// CHECK-NEXT:   %2655 = extractvalue { ptr, ptr } %2651, 1
// CHECK-NEXT:   %2656 = extractvalue { ptr, ptr } %2651, 0
// CHECK-NEXT:   %__llgo_funcval_code353 = call ptr asm "", "=r,0"(ptr %2656)
// CHECK-NEXT:   %2657 = call %reflect.Value %__llgo_funcval_code353(ptr {{(nest|swiftself)}} %2655, %"{{.*}}/runtime/internal/runtime.eface" %2654)
// CHECK-NEXT:   store %reflect.Value %2649, ptr %2643, align 8
// CHECK-NEXT:   store %reflect.Value %2657, ptr %2650, align 8
// CHECK-NEXT:   %2658 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 177
// CHECK-NEXT:   %2659 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2658, i32 0, i32 0
// CHECK-NEXT:   %2660 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2661 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %2662 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2661, ptr %2662, align 8
// CHECK-NEXT:   %2663 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %2662, 1
// CHECK-NEXT:   %2664 = extractvalue { ptr, ptr } %2660, 1
// CHECK-NEXT:   %2665 = extractvalue { ptr, ptr } %2660, 0
// CHECK-NEXT:   %__llgo_funcval_code354 = call ptr asm "", "=r,0"(ptr %2665)
// CHECK-NEXT:   %2666 = call %reflect.Value %__llgo_funcval_code354(ptr {{(nest|swiftself)}} %2664, %"{{.*}}/runtime/internal/runtime.eface" %2663)
// CHECK-NEXT:   %2667 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2658, i32 0, i32 1
// CHECK-NEXT:   %2668 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2669 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %2669, align 8
// CHECK-NEXT:   %2670 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2669, 1
// CHECK-NEXT:   %2671 = extractvalue { ptr, ptr } %2668, 1
// CHECK-NEXT:   %2672 = extractvalue { ptr, ptr } %2668, 0
// CHECK-NEXT:   %__llgo_funcval_code355 = call ptr asm "", "=r,0"(ptr %2672)
// CHECK-NEXT:   %2673 = call %reflect.Value %__llgo_funcval_code355(ptr {{(nest|swiftself)}} %2671, %"{{.*}}/runtime/internal/runtime.eface" %2670)
// CHECK-NEXT:   store %reflect.Value %2666, ptr %2659, align 8
// CHECK-NEXT:   store %reflect.Value %2673, ptr %2667, align 8
// CHECK-NEXT:   %2674 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 178
// CHECK-NEXT:   %2675 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2674, i32 0, i32 0
// CHECK-NEXT:   %2676 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2677 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %2678 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2677, ptr %2678, align 8
// CHECK-NEXT:   %2679 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %2678, 1
// CHECK-NEXT:   %2680 = extractvalue { ptr, ptr } %2676, 1
// CHECK-NEXT:   %2681 = extractvalue { ptr, ptr } %2676, 0
// CHECK-NEXT:   %__llgo_funcval_code356 = call ptr asm "", "=r,0"(ptr %2681)
// CHECK-NEXT:   %2682 = call %reflect.Value %__llgo_funcval_code356(ptr {{(nest|swiftself)}} %2680, %"{{.*}}/runtime/internal/runtime.eface" %2679)
// CHECK-NEXT:   %2683 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2674, i32 0, i32 1
// CHECK-NEXT:   %2684 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2685 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %2686 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2685, ptr %2686, align 8
// CHECK-NEXT:   %2687 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %2686, 1
// CHECK-NEXT:   %2688 = extractvalue { ptr, ptr } %2684, 1
// CHECK-NEXT:   %2689 = extractvalue { ptr, ptr } %2684, 0
// CHECK-NEXT:   %__llgo_funcval_code357 = call ptr asm "", "=r,0"(ptr %2689)
// CHECK-NEXT:   %2690 = call %reflect.Value %__llgo_funcval_code357(ptr {{(nest|swiftself)}} %2688, %"{{.*}}/runtime/internal/runtime.eface" %2687)
// CHECK-NEXT:   store %reflect.Value %2682, ptr %2675, align 8
// CHECK-NEXT:   store %reflect.Value %2690, ptr %2683, align 8
// CHECK-NEXT:   %2691 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 179
// CHECK-NEXT:   %2692 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2691, i32 0, i32 0
// CHECK-NEXT:   %2693 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2694 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %2694, align 8
// CHECK-NEXT:   %2695 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2694, 1
// CHECK-NEXT:   %2696 = extractvalue { ptr, ptr } %2693, 1
// CHECK-NEXT:   %2697 = extractvalue { ptr, ptr } %2693, 0
// CHECK-NEXT:   %__llgo_funcval_code358 = call ptr asm "", "=r,0"(ptr %2697)
// CHECK-NEXT:   %2698 = call %reflect.Value %__llgo_funcval_code358(ptr {{(nest|swiftself)}} %2696, %"{{.*}}/runtime/internal/runtime.eface" %2695)
// CHECK-NEXT:   %2699 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2691, i32 0, i32 1
// CHECK-NEXT:   %2700 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2701 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %2702 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2701, ptr %2702, align 8
// CHECK-NEXT:   %2703 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %2702, 1
// CHECK-NEXT:   %2704 = extractvalue { ptr, ptr } %2700, 1
// CHECK-NEXT:   %2705 = extractvalue { ptr, ptr } %2700, 0
// CHECK-NEXT:   %__llgo_funcval_code359 = call ptr asm "", "=r,0"(ptr %2705)
// CHECK-NEXT:   %2706 = call %reflect.Value %__llgo_funcval_code359(ptr {{(nest|swiftself)}} %2704, %"{{.*}}/runtime/internal/runtime.eface" %2703)
// CHECK-NEXT:   store %reflect.Value %2698, ptr %2692, align 8
// CHECK-NEXT:   store %reflect.Value %2706, ptr %2699, align 8
// CHECK-NEXT:   %2707 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 180
// CHECK-NEXT:   %2708 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2707, i32 0, i32 0
// CHECK-NEXT:   %2709 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2710 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %2711 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2710, ptr %2711, align 8
// CHECK-NEXT:   %2712 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %2711, 1
// CHECK-NEXT:   %2713 = extractvalue { ptr, ptr } %2709, 1
// CHECK-NEXT:   %2714 = extractvalue { ptr, ptr } %2709, 0
// CHECK-NEXT:   %__llgo_funcval_code360 = call ptr asm "", "=r,0"(ptr %2714)
// CHECK-NEXT:   %2715 = call %reflect.Value %__llgo_funcval_code360(ptr {{(nest|swiftself)}} %2713, %"{{.*}}/runtime/internal/runtime.eface" %2712)
// CHECK-NEXT:   %2716 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2707, i32 0, i32 1
// CHECK-NEXT:   %2717 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2718 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %2718, align 8
// CHECK-NEXT:   %2719 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2718, 1
// CHECK-NEXT:   %2720 = extractvalue { ptr, ptr } %2717, 1
// CHECK-NEXT:   %2721 = extractvalue { ptr, ptr } %2717, 0
// CHECK-NEXT:   %__llgo_funcval_code361 = call ptr asm "", "=r,0"(ptr %2721)
// CHECK-NEXT:   %2722 = call %reflect.Value %__llgo_funcval_code361(ptr {{(nest|swiftself)}} %2720, %"{{.*}}/runtime/internal/runtime.eface" %2719)
// CHECK-NEXT:   store %reflect.Value %2715, ptr %2708, align 8
// CHECK-NEXT:   store %reflect.Value %2722, ptr %2716, align 8
// CHECK-NEXT:   %2723 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 181
// CHECK-NEXT:   %2724 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2723, i32 0, i32 0
// CHECK-NEXT:   %2725 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2726 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %2727 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2726, ptr %2727, align 8
// CHECK-NEXT:   %2728 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %2727, 1
// CHECK-NEXT:   %2729 = extractvalue { ptr, ptr } %2725, 1
// CHECK-NEXT:   %2730 = extractvalue { ptr, ptr } %2725, 0
// CHECK-NEXT:   %__llgo_funcval_code362 = call ptr asm "", "=r,0"(ptr %2730)
// CHECK-NEXT:   %2731 = call %reflect.Value %__llgo_funcval_code362(ptr {{(nest|swiftself)}} %2729, %"{{.*}}/runtime/internal/runtime.eface" %2728)
// CHECK-NEXT:   %2732 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2723, i32 0, i32 1
// CHECK-NEXT:   %2733 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2734 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %2735 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %2734, ptr %2735, align 8
// CHECK-NEXT:   %2736 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %2735, 1
// CHECK-NEXT:   %2737 = extractvalue { ptr, ptr } %2733, 1
// CHECK-NEXT:   %2738 = extractvalue { ptr, ptr } %2733, 0
// CHECK-NEXT:   %__llgo_funcval_code363 = call ptr asm "", "=r,0"(ptr %2738)
// CHECK-NEXT:   %2739 = call %reflect.Value %__llgo_funcval_code363(ptr {{(nest|swiftself)}} %2737, %"{{.*}}/runtime/internal/runtime.eface" %2736)
// CHECK-NEXT:   store %reflect.Value %2731, ptr %2724, align 8
// CHECK-NEXT:   store %reflect.Value %2739, ptr %2732, align 8
// CHECK-NEXT:   %2740 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 182
// CHECK-NEXT:   %2741 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2740, i32 0, i32 0
// CHECK-NEXT:   %2742 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2743 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %2743, align 8
// CHECK-NEXT:   %2744 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2743, 1
// CHECK-NEXT:   %2745 = extractvalue { ptr, ptr } %2742, 1
// CHECK-NEXT:   %2746 = extractvalue { ptr, ptr } %2742, 0
// CHECK-NEXT:   %__llgo_funcval_code364 = call ptr asm "", "=r,0"(ptr %2746)
// CHECK-NEXT:   %2747 = call %reflect.Value %__llgo_funcval_code364(ptr {{(nest|swiftself)}} %2745, %"{{.*}}/runtime/internal/runtime.eface" %2744)
// CHECK-NEXT:   %2748 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2740, i32 0, i32 1
// CHECK-NEXT:   %2749 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2750 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2750, align 8
// CHECK-NEXT:   %2751 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2750, 1
// CHECK-NEXT:   %2752 = extractvalue { ptr, ptr } %2749, 1
// CHECK-NEXT:   %2753 = extractvalue { ptr, ptr } %2749, 0
// CHECK-NEXT:   %__llgo_funcval_code365 = call ptr asm "", "=r,0"(ptr %2753)
// CHECK-NEXT:   %2754 = call %reflect.Value %__llgo_funcval_code365(ptr {{(nest|swiftself)}} %2752, %"{{.*}}/runtime/internal/runtime.eface" %2751)
// CHECK-NEXT:   store %reflect.Value %2747, ptr %2741, align 8
// CHECK-NEXT:   store %reflect.Value %2754, ptr %2748, align 8
// CHECK-NEXT:   %2755 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 183
// CHECK-NEXT:   %2756 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2755, i32 0, i32 0
// CHECK-NEXT:   %2757 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2758 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 97, ptr %2758, align 1
// CHECK-NEXT:   %2759 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %2758, 1
// CHECK-NEXT:   %2760 = extractvalue { ptr, ptr } %2757, 1
// CHECK-NEXT:   %2761 = extractvalue { ptr, ptr } %2757, 0
// CHECK-NEXT:   %__llgo_funcval_code366 = call ptr asm "", "=r,0"(ptr %2761)
// CHECK-NEXT:   %2762 = call %reflect.Value %__llgo_funcval_code366(ptr {{(nest|swiftself)}} %2760, %"{{.*}}/runtime/internal/runtime.eface" %2759)
// CHECK-NEXT:   %2763 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2755, i32 0, i32 1
// CHECK-NEXT:   %2764 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2765 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2765, align 8
// CHECK-NEXT:   %2766 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2765, 1
// CHECK-NEXT:   %2767 = extractvalue { ptr, ptr } %2764, 1
// CHECK-NEXT:   %2768 = extractvalue { ptr, ptr } %2764, 0
// CHECK-NEXT:   %__llgo_funcval_code367 = call ptr asm "", "=r,0"(ptr %2768)
// CHECK-NEXT:   %2769 = call %reflect.Value %__llgo_funcval_code367(ptr {{(nest|swiftself)}} %2767, %"{{.*}}/runtime/internal/runtime.eface" %2766)
// CHECK-NEXT:   store %reflect.Value %2762, ptr %2756, align 8
// CHECK-NEXT:   store %reflect.Value %2769, ptr %2763, align 8
// CHECK-NEXT:   %2770 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 184
// CHECK-NEXT:   %2771 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2770, i32 0, i32 0
// CHECK-NEXT:   %2772 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2773 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 97, ptr %2773, align 2
// CHECK-NEXT:   %2774 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %2773, 1
// CHECK-NEXT:   %2775 = extractvalue { ptr, ptr } %2772, 1
// CHECK-NEXT:   %2776 = extractvalue { ptr, ptr } %2772, 0
// CHECK-NEXT:   %__llgo_funcval_code368 = call ptr asm "", "=r,0"(ptr %2776)
// CHECK-NEXT:   %2777 = call %reflect.Value %__llgo_funcval_code368(ptr {{(nest|swiftself)}} %2775, %"{{.*}}/runtime/internal/runtime.eface" %2774)
// CHECK-NEXT:   %2778 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2770, i32 0, i32 1
// CHECK-NEXT:   %2779 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2780 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2780, align 8
// CHECK-NEXT:   %2781 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2780, 1
// CHECK-NEXT:   %2782 = extractvalue { ptr, ptr } %2779, 1
// CHECK-NEXT:   %2783 = extractvalue { ptr, ptr } %2779, 0
// CHECK-NEXT:   %__llgo_funcval_code369 = call ptr asm "", "=r,0"(ptr %2783)
// CHECK-NEXT:   %2784 = call %reflect.Value %__llgo_funcval_code369(ptr {{(nest|swiftself)}} %2782, %"{{.*}}/runtime/internal/runtime.eface" %2781)
// CHECK-NEXT:   store %reflect.Value %2777, ptr %2771, align 8
// CHECK-NEXT:   store %reflect.Value %2784, ptr %2778, align 8
// CHECK-NEXT:   %2785 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 185
// CHECK-NEXT:   %2786 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2785, i32 0, i32 0
// CHECK-NEXT:   %2787 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2788 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %2788, align 4
// CHECK-NEXT:   %2789 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %2788, 1
// CHECK-NEXT:   %2790 = extractvalue { ptr, ptr } %2787, 1
// CHECK-NEXT:   %2791 = extractvalue { ptr, ptr } %2787, 0
// CHECK-NEXT:   %__llgo_funcval_code370 = call ptr asm "", "=r,0"(ptr %2791)
// CHECK-NEXT:   %2792 = call %reflect.Value %__llgo_funcval_code370(ptr {{(nest|swiftself)}} %2790, %"{{.*}}/runtime/internal/runtime.eface" %2789)
// CHECK-NEXT:   %2793 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2785, i32 0, i32 1
// CHECK-NEXT:   %2794 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2795 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2795, align 8
// CHECK-NEXT:   %2796 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2795, 1
// CHECK-NEXT:   %2797 = extractvalue { ptr, ptr } %2794, 1
// CHECK-NEXT:   %2798 = extractvalue { ptr, ptr } %2794, 0
// CHECK-NEXT:   %__llgo_funcval_code371 = call ptr asm "", "=r,0"(ptr %2798)
// CHECK-NEXT:   %2799 = call %reflect.Value %__llgo_funcval_code371(ptr {{(nest|swiftself)}} %2797, %"{{.*}}/runtime/internal/runtime.eface" %2796)
// CHECK-NEXT:   store %reflect.Value %2792, ptr %2786, align 8
// CHECK-NEXT:   store %reflect.Value %2799, ptr %2793, align 8
// CHECK-NEXT:   %2800 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 186
// CHECK-NEXT:   %2801 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2800, i32 0, i32 0
// CHECK-NEXT:   %2802 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2803 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %2803, align 8
// CHECK-NEXT:   %2804 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %2803, 1
// CHECK-NEXT:   %2805 = extractvalue { ptr, ptr } %2802, 1
// CHECK-NEXT:   %2806 = extractvalue { ptr, ptr } %2802, 0
// CHECK-NEXT:   %__llgo_funcval_code372 = call ptr asm "", "=r,0"(ptr %2806)
// CHECK-NEXT:   %2807 = call %reflect.Value %__llgo_funcval_code372(ptr {{(nest|swiftself)}} %2805, %"{{.*}}/runtime/internal/runtime.eface" %2804)
// CHECK-NEXT:   %2808 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2800, i32 0, i32 1
// CHECK-NEXT:   %2809 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2810 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2810, align 8
// CHECK-NEXT:   %2811 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2810, 1
// CHECK-NEXT:   %2812 = extractvalue { ptr, ptr } %2809, 1
// CHECK-NEXT:   %2813 = extractvalue { ptr, ptr } %2809, 0
// CHECK-NEXT:   %__llgo_funcval_code373 = call ptr asm "", "=r,0"(ptr %2813)
// CHECK-NEXT:   %2814 = call %reflect.Value %__llgo_funcval_code373(ptr {{(nest|swiftself)}} %2812, %"{{.*}}/runtime/internal/runtime.eface" %2811)
// CHECK-NEXT:   store %reflect.Value %2807, ptr %2801, align 8
// CHECK-NEXT:   store %reflect.Value %2814, ptr %2808, align 8
// CHECK-NEXT:   %2815 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 187
// CHECK-NEXT:   %2816 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2815, i32 0, i32 0
// CHECK-NEXT:   %2817 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2818 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %2818, align 8
// CHECK-NEXT:   %2819 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %2818, 1
// CHECK-NEXT:   %2820 = extractvalue { ptr, ptr } %2817, 1
// CHECK-NEXT:   %2821 = extractvalue { ptr, ptr } %2817, 0
// CHECK-NEXT:   %__llgo_funcval_code374 = call ptr asm "", "=r,0"(ptr %2821)
// CHECK-NEXT:   %2822 = call %reflect.Value %__llgo_funcval_code374(ptr {{(nest|swiftself)}} %2820, %"{{.*}}/runtime/internal/runtime.eface" %2819)
// CHECK-NEXT:   %2823 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2815, i32 0, i32 1
// CHECK-NEXT:   %2824 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2825 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2825, align 8
// CHECK-NEXT:   %2826 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2825, 1
// CHECK-NEXT:   %2827 = extractvalue { ptr, ptr } %2824, 1
// CHECK-NEXT:   %2828 = extractvalue { ptr, ptr } %2824, 0
// CHECK-NEXT:   %__llgo_funcval_code375 = call ptr asm "", "=r,0"(ptr %2828)
// CHECK-NEXT:   %2829 = call %reflect.Value %__llgo_funcval_code375(ptr {{(nest|swiftself)}} %2827, %"{{.*}}/runtime/internal/runtime.eface" %2826)
// CHECK-NEXT:   store %reflect.Value %2822, ptr %2816, align 8
// CHECK-NEXT:   store %reflect.Value %2829, ptr %2823, align 8
// CHECK-NEXT:   %2830 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 188
// CHECK-NEXT:   %2831 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2830, i32 0, i32 0
// CHECK-NEXT:   %2832 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2833 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 97, ptr %2833, align 1
// CHECK-NEXT:   %2834 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %2833, 1
// CHECK-NEXT:   %2835 = extractvalue { ptr, ptr } %2832, 1
// CHECK-NEXT:   %2836 = extractvalue { ptr, ptr } %2832, 0
// CHECK-NEXT:   %__llgo_funcval_code376 = call ptr asm "", "=r,0"(ptr %2836)
// CHECK-NEXT:   %2837 = call %reflect.Value %__llgo_funcval_code376(ptr {{(nest|swiftself)}} %2835, %"{{.*}}/runtime/internal/runtime.eface" %2834)
// CHECK-NEXT:   %2838 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2830, i32 0, i32 1
// CHECK-NEXT:   %2839 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2840 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2840, align 8
// CHECK-NEXT:   %2841 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2840, 1
// CHECK-NEXT:   %2842 = extractvalue { ptr, ptr } %2839, 1
// CHECK-NEXT:   %2843 = extractvalue { ptr, ptr } %2839, 0
// CHECK-NEXT:   %__llgo_funcval_code377 = call ptr asm "", "=r,0"(ptr %2843)
// CHECK-NEXT:   %2844 = call %reflect.Value %__llgo_funcval_code377(ptr {{(nest|swiftself)}} %2842, %"{{.*}}/runtime/internal/runtime.eface" %2841)
// CHECK-NEXT:   store %reflect.Value %2837, ptr %2831, align 8
// CHECK-NEXT:   store %reflect.Value %2844, ptr %2838, align 8
// CHECK-NEXT:   %2845 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 189
// CHECK-NEXT:   %2846 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2845, i32 0, i32 0
// CHECK-NEXT:   %2847 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2848 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 97, ptr %2848, align 2
// CHECK-NEXT:   %2849 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %2848, 1
// CHECK-NEXT:   %2850 = extractvalue { ptr, ptr } %2847, 1
// CHECK-NEXT:   %2851 = extractvalue { ptr, ptr } %2847, 0
// CHECK-NEXT:   %__llgo_funcval_code378 = call ptr asm "", "=r,0"(ptr %2851)
// CHECK-NEXT:   %2852 = call %reflect.Value %__llgo_funcval_code378(ptr {{(nest|swiftself)}} %2850, %"{{.*}}/runtime/internal/runtime.eface" %2849)
// CHECK-NEXT:   %2853 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2845, i32 0, i32 1
// CHECK-NEXT:   %2854 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2855 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2855, align 8
// CHECK-NEXT:   %2856 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2855, 1
// CHECK-NEXT:   %2857 = extractvalue { ptr, ptr } %2854, 1
// CHECK-NEXT:   %2858 = extractvalue { ptr, ptr } %2854, 0
// CHECK-NEXT:   %__llgo_funcval_code379 = call ptr asm "", "=r,0"(ptr %2858)
// CHECK-NEXT:   %2859 = call %reflect.Value %__llgo_funcval_code379(ptr {{(nest|swiftself)}} %2857, %"{{.*}}/runtime/internal/runtime.eface" %2856)
// CHECK-NEXT:   store %reflect.Value %2852, ptr %2846, align 8
// CHECK-NEXT:   store %reflect.Value %2859, ptr %2853, align 8
// CHECK-NEXT:   %2860 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 190
// CHECK-NEXT:   %2861 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2860, i32 0, i32 0
// CHECK-NEXT:   %2862 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2863 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %2863, align 4
// CHECK-NEXT:   %2864 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %2863, 1
// CHECK-NEXT:   %2865 = extractvalue { ptr, ptr } %2862, 1
// CHECK-NEXT:   %2866 = extractvalue { ptr, ptr } %2862, 0
// CHECK-NEXT:   %__llgo_funcval_code380 = call ptr asm "", "=r,0"(ptr %2866)
// CHECK-NEXT:   %2867 = call %reflect.Value %__llgo_funcval_code380(ptr {{(nest|swiftself)}} %2865, %"{{.*}}/runtime/internal/runtime.eface" %2864)
// CHECK-NEXT:   %2868 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2860, i32 0, i32 1
// CHECK-NEXT:   %2869 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2870 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2870, align 8
// CHECK-NEXT:   %2871 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2870, 1
// CHECK-NEXT:   %2872 = extractvalue { ptr, ptr } %2869, 1
// CHECK-NEXT:   %2873 = extractvalue { ptr, ptr } %2869, 0
// CHECK-NEXT:   %__llgo_funcval_code381 = call ptr asm "", "=r,0"(ptr %2873)
// CHECK-NEXT:   %2874 = call %reflect.Value %__llgo_funcval_code381(ptr {{(nest|swiftself)}} %2872, %"{{.*}}/runtime/internal/runtime.eface" %2871)
// CHECK-NEXT:   store %reflect.Value %2867, ptr %2861, align 8
// CHECK-NEXT:   store %reflect.Value %2874, ptr %2868, align 8
// CHECK-NEXT:   %2875 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 191
// CHECK-NEXT:   %2876 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2875, i32 0, i32 0
// CHECK-NEXT:   %2877 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2878 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %2878, align 8
// CHECK-NEXT:   %2879 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %2878, 1
// CHECK-NEXT:   %2880 = extractvalue { ptr, ptr } %2877, 1
// CHECK-NEXT:   %2881 = extractvalue { ptr, ptr } %2877, 0
// CHECK-NEXT:   %__llgo_funcval_code382 = call ptr asm "", "=r,0"(ptr %2881)
// CHECK-NEXT:   %2882 = call %reflect.Value %__llgo_funcval_code382(ptr {{(nest|swiftself)}} %2880, %"{{.*}}/runtime/internal/runtime.eface" %2879)
// CHECK-NEXT:   %2883 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2875, i32 0, i32 1
// CHECK-NEXT:   %2884 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2885 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2885, align 8
// CHECK-NEXT:   %2886 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2885, 1
// CHECK-NEXT:   %2887 = extractvalue { ptr, ptr } %2884, 1
// CHECK-NEXT:   %2888 = extractvalue { ptr, ptr } %2884, 0
// CHECK-NEXT:   %__llgo_funcval_code383 = call ptr asm "", "=r,0"(ptr %2888)
// CHECK-NEXT:   %2889 = call %reflect.Value %__llgo_funcval_code383(ptr {{(nest|swiftself)}} %2887, %"{{.*}}/runtime/internal/runtime.eface" %2886)
// CHECK-NEXT:   store %reflect.Value %2882, ptr %2876, align 8
// CHECK-NEXT:   store %reflect.Value %2889, ptr %2883, align 8
// CHECK-NEXT:   %2890 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 192
// CHECK-NEXT:   %2891 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2890, i32 0, i32 0
// CHECK-NEXT:   %2892 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2893 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %2893, align 8
// CHECK-NEXT:   %2894 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %2893, 1
// CHECK-NEXT:   %2895 = extractvalue { ptr, ptr } %2892, 1
// CHECK-NEXT:   %2896 = extractvalue { ptr, ptr } %2892, 0
// CHECK-NEXT:   %__llgo_funcval_code384 = call ptr asm "", "=r,0"(ptr %2896)
// CHECK-NEXT:   %2897 = call %reflect.Value %__llgo_funcval_code384(ptr {{(nest|swiftself)}} %2895, %"{{.*}}/runtime/internal/runtime.eface" %2894)
// CHECK-NEXT:   %2898 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2890, i32 0, i32 1
// CHECK-NEXT:   %2899 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2900 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %2900, align 8
// CHECK-NEXT:   %2901 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2900, 1
// CHECK-NEXT:   %2902 = extractvalue { ptr, ptr } %2899, 1
// CHECK-NEXT:   %2903 = extractvalue { ptr, ptr } %2899, 0
// CHECK-NEXT:   %__llgo_funcval_code385 = call ptr asm "", "=r,0"(ptr %2903)
// CHECK-NEXT:   %2904 = call %reflect.Value %__llgo_funcval_code385(ptr {{(nest|swiftself)}} %2902, %"{{.*}}/runtime/internal/runtime.eface" %2901)
// CHECK-NEXT:   store %reflect.Value %2897, ptr %2891, align 8
// CHECK-NEXT:   store %reflect.Value %2904, ptr %2898, align 8
// CHECK-NEXT:   %2905 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 193
// CHECK-NEXT:   %2906 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2905, i32 0, i32 0
// CHECK-NEXT:   %2907 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2908 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 -1, ptr %2908, align 8
// CHECK-NEXT:   %2909 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2908, 1
// CHECK-NEXT:   %2910 = extractvalue { ptr, ptr } %2907, 1
// CHECK-NEXT:   %2911 = extractvalue { ptr, ptr } %2907, 0
// CHECK-NEXT:   %__llgo_funcval_code386 = call ptr asm "", "=r,0"(ptr %2911)
// CHECK-NEXT:   %2912 = call %reflect.Value %__llgo_funcval_code386(ptr {{(nest|swiftself)}} %2910, %"{{.*}}/runtime/internal/runtime.eface" %2909)
// CHECK-NEXT:   %2913 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2905, i32 0, i32 1
// CHECK-NEXT:   %2914 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2915 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2915, align 8
// CHECK-NEXT:   %2916 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2915, 1
// CHECK-NEXT:   %2917 = extractvalue { ptr, ptr } %2914, 1
// CHECK-NEXT:   %2918 = extractvalue { ptr, ptr } %2914, 0
// CHECK-NEXT:   %__llgo_funcval_code387 = call ptr asm "", "=r,0"(ptr %2918)
// CHECK-NEXT:   %2919 = call %reflect.Value %__llgo_funcval_code387(ptr {{(nest|swiftself)}} %2917, %"{{.*}}/runtime/internal/runtime.eface" %2916)
// CHECK-NEXT:   store %reflect.Value %2912, ptr %2906, align 8
// CHECK-NEXT:   store %reflect.Value %2919, ptr %2913, align 8
// CHECK-NEXT:   %2920 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 194
// CHECK-NEXT:   %2921 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2920, i32 0, i32 0
// CHECK-NEXT:   %2922 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2923 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 -2, ptr %2923, align 1
// CHECK-NEXT:   %2924 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %2923, 1
// CHECK-NEXT:   %2925 = extractvalue { ptr, ptr } %2922, 1
// CHECK-NEXT:   %2926 = extractvalue { ptr, ptr } %2922, 0
// CHECK-NEXT:   %__llgo_funcval_code388 = call ptr asm "", "=r,0"(ptr %2926)
// CHECK-NEXT:   %2927 = call %reflect.Value %__llgo_funcval_code388(ptr {{(nest|swiftself)}} %2925, %"{{.*}}/runtime/internal/runtime.eface" %2924)
// CHECK-NEXT:   %2928 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2920, i32 0, i32 1
// CHECK-NEXT:   %2929 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2930 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2930, align 8
// CHECK-NEXT:   %2931 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2930, 1
// CHECK-NEXT:   %2932 = extractvalue { ptr, ptr } %2929, 1
// CHECK-NEXT:   %2933 = extractvalue { ptr, ptr } %2929, 0
// CHECK-NEXT:   %__llgo_funcval_code389 = call ptr asm "", "=r,0"(ptr %2933)
// CHECK-NEXT:   %2934 = call %reflect.Value %__llgo_funcval_code389(ptr {{(nest|swiftself)}} %2932, %"{{.*}}/runtime/internal/runtime.eface" %2931)
// CHECK-NEXT:   store %reflect.Value %2927, ptr %2921, align 8
// CHECK-NEXT:   store %reflect.Value %2934, ptr %2928, align 8
// CHECK-NEXT:   %2935 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 195
// CHECK-NEXT:   %2936 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2935, i32 0, i32 0
// CHECK-NEXT:   %2937 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2938 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 -3, ptr %2938, align 2
// CHECK-NEXT:   %2939 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %2938, 1
// CHECK-NEXT:   %2940 = extractvalue { ptr, ptr } %2937, 1
// CHECK-NEXT:   %2941 = extractvalue { ptr, ptr } %2937, 0
// CHECK-NEXT:   %__llgo_funcval_code390 = call ptr asm "", "=r,0"(ptr %2941)
// CHECK-NEXT:   %2942 = call %reflect.Value %__llgo_funcval_code390(ptr {{(nest|swiftself)}} %2940, %"{{.*}}/runtime/internal/runtime.eface" %2939)
// CHECK-NEXT:   %2943 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2935, i32 0, i32 1
// CHECK-NEXT:   %2944 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2945 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2945, align 8
// CHECK-NEXT:   %2946 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2945, 1
// CHECK-NEXT:   %2947 = extractvalue { ptr, ptr } %2944, 1
// CHECK-NEXT:   %2948 = extractvalue { ptr, ptr } %2944, 0
// CHECK-NEXT:   %__llgo_funcval_code391 = call ptr asm "", "=r,0"(ptr %2948)
// CHECK-NEXT:   %2949 = call %reflect.Value %__llgo_funcval_code391(ptr {{(nest|swiftself)}} %2947, %"{{.*}}/runtime/internal/runtime.eface" %2946)
// CHECK-NEXT:   store %reflect.Value %2942, ptr %2936, align 8
// CHECK-NEXT:   store %reflect.Value %2949, ptr %2943, align 8
// CHECK-NEXT:   %2950 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 196
// CHECK-NEXT:   %2951 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2950, i32 0, i32 0
// CHECK-NEXT:   %2952 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2953 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 -4, ptr %2953, align 4
// CHECK-NEXT:   %2954 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %2953, 1
// CHECK-NEXT:   %2955 = extractvalue { ptr, ptr } %2952, 1
// CHECK-NEXT:   %2956 = extractvalue { ptr, ptr } %2952, 0
// CHECK-NEXT:   %__llgo_funcval_code392 = call ptr asm "", "=r,0"(ptr %2956)
// CHECK-NEXT:   %2957 = call %reflect.Value %__llgo_funcval_code392(ptr {{(nest|swiftself)}} %2955, %"{{.*}}/runtime/internal/runtime.eface" %2954)
// CHECK-NEXT:   %2958 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2950, i32 0, i32 1
// CHECK-NEXT:   %2959 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2960 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2960, align 8
// CHECK-NEXT:   %2961 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2960, 1
// CHECK-NEXT:   %2962 = extractvalue { ptr, ptr } %2959, 1
// CHECK-NEXT:   %2963 = extractvalue { ptr, ptr } %2959, 0
// CHECK-NEXT:   %__llgo_funcval_code393 = call ptr asm "", "=r,0"(ptr %2963)
// CHECK-NEXT:   %2964 = call %reflect.Value %__llgo_funcval_code393(ptr {{(nest|swiftself)}} %2962, %"{{.*}}/runtime/internal/runtime.eface" %2961)
// CHECK-NEXT:   store %reflect.Value %2957, ptr %2951, align 8
// CHECK-NEXT:   store %reflect.Value %2964, ptr %2958, align 8
// CHECK-NEXT:   %2965 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 197
// CHECK-NEXT:   %2966 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2965, i32 0, i32 0
// CHECK-NEXT:   %2967 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2968 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 -5, ptr %2968, align 8
// CHECK-NEXT:   %2969 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %2968, 1
// CHECK-NEXT:   %2970 = extractvalue { ptr, ptr } %2967, 1
// CHECK-NEXT:   %2971 = extractvalue { ptr, ptr } %2967, 0
// CHECK-NEXT:   %__llgo_funcval_code394 = call ptr asm "", "=r,0"(ptr %2971)
// CHECK-NEXT:   %2972 = call %reflect.Value %__llgo_funcval_code394(ptr {{(nest|swiftself)}} %2970, %"{{.*}}/runtime/internal/runtime.eface" %2969)
// CHECK-NEXT:   %2973 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2965, i32 0, i32 1
// CHECK-NEXT:   %2974 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2975 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2975, align 8
// CHECK-NEXT:   %2976 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2975, 1
// CHECK-NEXT:   %2977 = extractvalue { ptr, ptr } %2974, 1
// CHECK-NEXT:   %2978 = extractvalue { ptr, ptr } %2974, 0
// CHECK-NEXT:   %__llgo_funcval_code395 = call ptr asm "", "=r,0"(ptr %2978)
// CHECK-NEXT:   %2979 = call %reflect.Value %__llgo_funcval_code395(ptr {{(nest|swiftself)}} %2977, %"{{.*}}/runtime/internal/runtime.eface" %2976)
// CHECK-NEXT:   store %reflect.Value %2972, ptr %2966, align 8
// CHECK-NEXT:   store %reflect.Value %2979, ptr %2973, align 8
// CHECK-NEXT:   %2980 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 198
// CHECK-NEXT:   %2981 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2980, i32 0, i32 0
// CHECK-NEXT:   %2982 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2983 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 -4294967296, ptr %2983, align 8
// CHECK-NEXT:   %2984 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %2983, 1
// CHECK-NEXT:   %2985 = extractvalue { ptr, ptr } %2982, 1
// CHECK-NEXT:   %2986 = extractvalue { ptr, ptr } %2982, 0
// CHECK-NEXT:   %__llgo_funcval_code396 = call ptr asm "", "=r,0"(ptr %2986)
// CHECK-NEXT:   %2987 = call %reflect.Value %__llgo_funcval_code396(ptr {{(nest|swiftself)}} %2985, %"{{.*}}/runtime/internal/runtime.eface" %2984)
// CHECK-NEXT:   %2988 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2980, i32 0, i32 1
// CHECK-NEXT:   %2989 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2990 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %2990, align 8
// CHECK-NEXT:   %2991 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %2990, 1
// CHECK-NEXT:   %2992 = extractvalue { ptr, ptr } %2989, 1
// CHECK-NEXT:   %2993 = extractvalue { ptr, ptr } %2989, 0
// CHECK-NEXT:   %__llgo_funcval_code397 = call ptr asm "", "=r,0"(ptr %2993)
// CHECK-NEXT:   %2994 = call %reflect.Value %__llgo_funcval_code397(ptr {{(nest|swiftself)}} %2992, %"{{.*}}/runtime/internal/runtime.eface" %2991)
// CHECK-NEXT:   store %reflect.Value %2987, ptr %2981, align 8
// CHECK-NEXT:   store %reflect.Value %2994, ptr %2988, align 8
// CHECK-NEXT:   %2995 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 199
// CHECK-NEXT:   %2996 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2995, i32 0, i32 0
// CHECK-NEXT:   %2997 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %2998 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 4294967296, ptr %2998, align 8
// CHECK-NEXT:   %2999 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %2998, 1
// CHECK-NEXT:   %3000 = extractvalue { ptr, ptr } %2997, 1
// CHECK-NEXT:   %3001 = extractvalue { ptr, ptr } %2997, 0
// CHECK-NEXT:   %__llgo_funcval_code398 = call ptr asm "", "=r,0"(ptr %3001)
// CHECK-NEXT:   %3002 = call %reflect.Value %__llgo_funcval_code398(ptr {{(nest|swiftself)}} %3000, %"{{.*}}/runtime/internal/runtime.eface" %2999)
// CHECK-NEXT:   %3003 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %2995, i32 0, i32 1
// CHECK-NEXT:   %3004 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3005 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3005, align 8
// CHECK-NEXT:   %3006 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3005, 1
// CHECK-NEXT:   %3007 = extractvalue { ptr, ptr } %3004, 1
// CHECK-NEXT:   %3008 = extractvalue { ptr, ptr } %3004, 0
// CHECK-NEXT:   %__llgo_funcval_code399 = call ptr asm "", "=r,0"(ptr %3008)
// CHECK-NEXT:   %3009 = call %reflect.Value %__llgo_funcval_code399(ptr {{(nest|swiftself)}} %3007, %"{{.*}}/runtime/internal/runtime.eface" %3006)
// CHECK-NEXT:   store %reflect.Value %3002, ptr %2996, align 8
// CHECK-NEXT:   store %reflect.Value %3009, ptr %3003, align 8
// CHECK-NEXT:   %3010 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 200
// CHECK-NEXT:   %3011 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3010, i32 0, i32 0
// CHECK-NEXT:   %3012 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3013 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114113, ptr %3013, align 8
// CHECK-NEXT:   %3014 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %3013, 1
// CHECK-NEXT:   %3015 = extractvalue { ptr, ptr } %3012, 1
// CHECK-NEXT:   %3016 = extractvalue { ptr, ptr } %3012, 0
// CHECK-NEXT:   %__llgo_funcval_code400 = call ptr asm "", "=r,0"(ptr %3016)
// CHECK-NEXT:   %3017 = call %reflect.Value %__llgo_funcval_code400(ptr {{(nest|swiftself)}} %3015, %"{{.*}}/runtime/internal/runtime.eface" %3014)
// CHECK-NEXT:   %3018 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3010, i32 0, i32 1
// CHECK-NEXT:   %3019 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3020 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3020, align 8
// CHECK-NEXT:   %3021 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3020, 1
// CHECK-NEXT:   %3022 = extractvalue { ptr, ptr } %3019, 1
// CHECK-NEXT:   %3023 = extractvalue { ptr, ptr } %3019, 0
// CHECK-NEXT:   %__llgo_funcval_code401 = call ptr asm "", "=r,0"(ptr %3023)
// CHECK-NEXT:   %3024 = call %reflect.Value %__llgo_funcval_code401(ptr {{(nest|swiftself)}} %3022, %"{{.*}}/runtime/internal/runtime.eface" %3021)
// CHECK-NEXT:   store %reflect.Value %3017, ptr %3011, align 8
// CHECK-NEXT:   store %reflect.Value %3024, ptr %3018, align 8
// CHECK-NEXT:   %3025 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 201
// CHECK-NEXT:   %3026 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3025, i32 0, i32 0
// CHECK-NEXT:   %3027 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3028 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 1114114, ptr %3028, align 4
// CHECK-NEXT:   %3029 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %3028, 1
// CHECK-NEXT:   %3030 = extractvalue { ptr, ptr } %3027, 1
// CHECK-NEXT:   %3031 = extractvalue { ptr, ptr } %3027, 0
// CHECK-NEXT:   %__llgo_funcval_code402 = call ptr asm "", "=r,0"(ptr %3031)
// CHECK-NEXT:   %3032 = call %reflect.Value %__llgo_funcval_code402(ptr {{(nest|swiftself)}} %3030, %"{{.*}}/runtime/internal/runtime.eface" %3029)
// CHECK-NEXT:   %3033 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3025, i32 0, i32 1
// CHECK-NEXT:   %3034 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3035 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3035, align 8
// CHECK-NEXT:   %3036 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3035, 1
// CHECK-NEXT:   %3037 = extractvalue { ptr, ptr } %3034, 1
// CHECK-NEXT:   %3038 = extractvalue { ptr, ptr } %3034, 0
// CHECK-NEXT:   %__llgo_funcval_code403 = call ptr asm "", "=r,0"(ptr %3038)
// CHECK-NEXT:   %3039 = call %reflect.Value %__llgo_funcval_code403(ptr {{(nest|swiftself)}} %3037, %"{{.*}}/runtime/internal/runtime.eface" %3036)
// CHECK-NEXT:   store %reflect.Value %3032, ptr %3026, align 8
// CHECK-NEXT:   store %reflect.Value %3039, ptr %3033, align 8
// CHECK-NEXT:   %3040 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 202
// CHECK-NEXT:   %3041 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3040, i32 0, i32 0
// CHECK-NEXT:   %3042 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3043 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114115, ptr %3043, align 8
// CHECK-NEXT:   %3044 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %3043, 1
// CHECK-NEXT:   %3045 = extractvalue { ptr, ptr } %3042, 1
// CHECK-NEXT:   %3046 = extractvalue { ptr, ptr } %3042, 0
// CHECK-NEXT:   %__llgo_funcval_code404 = call ptr asm "", "=r,0"(ptr %3046)
// CHECK-NEXT:   %3047 = call %reflect.Value %__llgo_funcval_code404(ptr {{(nest|swiftself)}} %3045, %"{{.*}}/runtime/internal/runtime.eface" %3044)
// CHECK-NEXT:   %3048 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3040, i32 0, i32 1
// CHECK-NEXT:   %3049 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3050 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3050, align 8
// CHECK-NEXT:   %3051 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3050, 1
// CHECK-NEXT:   %3052 = extractvalue { ptr, ptr } %3049, 1
// CHECK-NEXT:   %3053 = extractvalue { ptr, ptr } %3049, 0
// CHECK-NEXT:   %__llgo_funcval_code405 = call ptr asm "", "=r,0"(ptr %3053)
// CHECK-NEXT:   %3054 = call %reflect.Value %__llgo_funcval_code405(ptr {{(nest|swiftself)}} %3052, %"{{.*}}/runtime/internal/runtime.eface" %3051)
// CHECK-NEXT:   store %reflect.Value %3047, ptr %3041, align 8
// CHECK-NEXT:   store %reflect.Value %3054, ptr %3048, align 8
// CHECK-NEXT:   %3055 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 203
// CHECK-NEXT:   %3056 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3055, i32 0, i32 0
// CHECK-NEXT:   %3057 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3058 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 4294967296, ptr %3058, align 8
// CHECK-NEXT:   %3059 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %3058, 1
// CHECK-NEXT:   %3060 = extractvalue { ptr, ptr } %3057, 1
// CHECK-NEXT:   %3061 = extractvalue { ptr, ptr } %3057, 0
// CHECK-NEXT:   %__llgo_funcval_code406 = call ptr asm "", "=r,0"(ptr %3061)
// CHECK-NEXT:   %3062 = call %reflect.Value %__llgo_funcval_code406(ptr {{(nest|swiftself)}} %3060, %"{{.*}}/runtime/internal/runtime.eface" %3059)
// CHECK-NEXT:   %3063 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3055, i32 0, i32 1
// CHECK-NEXT:   %3064 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3065 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3065, align 8
// CHECK-NEXT:   %3066 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3065, 1
// CHECK-NEXT:   %3067 = extractvalue { ptr, ptr } %3064, 1
// CHECK-NEXT:   %3068 = extractvalue { ptr, ptr } %3064, 0
// CHECK-NEXT:   %__llgo_funcval_code407 = call ptr asm "", "=r,0"(ptr %3068)
// CHECK-NEXT:   %3069 = call %reflect.Value %__llgo_funcval_code407(ptr {{(nest|swiftself)}} %3067, %"{{.*}}/runtime/internal/runtime.eface" %3066)
// CHECK-NEXT:   store %reflect.Value %3062, ptr %3056, align 8
// CHECK-NEXT:   store %reflect.Value %3069, ptr %3063, align 8
// CHECK-NEXT:   %3070 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 204
// CHECK-NEXT:   %3071 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3070, i32 0, i32 0
// CHECK-NEXT:   %3072 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3073 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114116, ptr %3073, align 8
// CHECK-NEXT:   %3074 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %3073, 1
// CHECK-NEXT:   %3075 = extractvalue { ptr, ptr } %3072, 1
// CHECK-NEXT:   %3076 = extractvalue { ptr, ptr } %3072, 0
// CHECK-NEXT:   %__llgo_funcval_code408 = call ptr asm "", "=r,0"(ptr %3076)
// CHECK-NEXT:   %3077 = call %reflect.Value %__llgo_funcval_code408(ptr {{(nest|swiftself)}} %3075, %"{{.*}}/runtime/internal/runtime.eface" %3074)
// CHECK-NEXT:   %3078 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3070, i32 0, i32 1
// CHECK-NEXT:   %3079 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3080 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3080, align 8
// CHECK-NEXT:   %3081 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3080, 1
// CHECK-NEXT:   %3082 = extractvalue { ptr, ptr } %3079, 1
// CHECK-NEXT:   %3083 = extractvalue { ptr, ptr } %3079, 0
// CHECK-NEXT:   %__llgo_funcval_code409 = call ptr asm "", "=r,0"(ptr %3083)
// CHECK-NEXT:   %3084 = call %reflect.Value %__llgo_funcval_code409(ptr {{(nest|swiftself)}} %3082, %"{{.*}}/runtime/internal/runtime.eface" %3081)
// CHECK-NEXT:   store %reflect.Value %3077, ptr %3071, align 8
// CHECK-NEXT:   store %reflect.Value %3084, ptr %3078, align 8
// CHECK-NEXT:   %3085 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 205
// CHECK-NEXT:   %3086 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3085, i32 0, i32 0
// CHECK-NEXT:   %3087 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3088 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3088, align 8
// CHECK-NEXT:   %3089 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3088, 1
// CHECK-NEXT:   %3090 = extractvalue { ptr, ptr } %3087, 1
// CHECK-NEXT:   %3091 = extractvalue { ptr, ptr } %3087, 0
// CHECK-NEXT:   %__llgo_funcval_code410 = call ptr asm "", "=r,0"(ptr %3091)
// CHECK-NEXT:   %3092 = call %reflect.Value %__llgo_funcval_code410(ptr {{(nest|swiftself)}} %3090, %"{{.*}}/runtime/internal/runtime.eface" %3089)
// CHECK-NEXT:   %3093 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3085, i32 0, i32 1
// CHECK-NEXT:   %3094 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3095 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3095, align 8
// CHECK-NEXT:   %3096 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3095, 1
// CHECK-NEXT:   %3097 = extractvalue { ptr, ptr } %3094, 1
// CHECK-NEXT:   %3098 = extractvalue { ptr, ptr } %3094, 0
// CHECK-NEXT:   %__llgo_funcval_code411 = call ptr asm "", "=r,0"(ptr %3098)
// CHECK-NEXT:   %3099 = call %reflect.Value %__llgo_funcval_code411(ptr {{(nest|swiftself)}} %3097, %"{{.*}}/runtime/internal/runtime.eface" %3096)
// CHECK-NEXT:   store %reflect.Value %3092, ptr %3086, align 8
// CHECK-NEXT:   store %reflect.Value %3099, ptr %3093, align 8
// CHECK-NEXT:   %3100 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 206
// CHECK-NEXT:   %3101 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3100, i32 0, i32 0
// CHECK-NEXT:   %3102 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3103 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3103, align 8
// CHECK-NEXT:   %3104 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3103, 1
// CHECK-NEXT:   %3105 = extractvalue { ptr, ptr } %3102, 1
// CHECK-NEXT:   %3106 = extractvalue { ptr, ptr } %3102, 0
// CHECK-NEXT:   %__llgo_funcval_code412 = call ptr asm "", "=r,0"(ptr %3106)
// CHECK-NEXT:   %3107 = call %reflect.Value %__llgo_funcval_code412(ptr {{(nest|swiftself)}} %3105, %"{{.*}}/runtime/internal/runtime.eface" %3104)
// CHECK-NEXT:   %3108 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3100, i32 0, i32 1
// CHECK-NEXT:   %3109 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3110 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3110, align 8
// CHECK-NEXT:   %3111 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3110, 1
// CHECK-NEXT:   %3112 = extractvalue { ptr, ptr } %3109, 1
// CHECK-NEXT:   %3113 = extractvalue { ptr, ptr } %3109, 0
// CHECK-NEXT:   %__llgo_funcval_code413 = call ptr asm "", "=r,0"(ptr %3113)
// CHECK-NEXT:   %3114 = call %reflect.Value %__llgo_funcval_code413(ptr {{(nest|swiftself)}} %3112, %"{{.*}}/runtime/internal/runtime.eface" %3111)
// CHECK-NEXT:   store %reflect.Value %3107, ptr %3101, align 8
// CHECK-NEXT:   store %reflect.Value %3114, ptr %3108, align 8
// CHECK-NEXT:   %3115 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 207
// CHECK-NEXT:   %3116 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3115, i32 0, i32 0
// CHECK-NEXT:   %3117 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3118 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3118, align 8
// CHECK-NEXT:   %3119 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3118, 1
// CHECK-NEXT:   %3120 = extractvalue { ptr, ptr } %3117, 1
// CHECK-NEXT:   %3121 = extractvalue { ptr, ptr } %3117, 0
// CHECK-NEXT:   %__llgo_funcval_code414 = call ptr asm "", "=r,0"(ptr %3121)
// CHECK-NEXT:   %3122 = call %reflect.Value %__llgo_funcval_code414(ptr {{(nest|swiftself)}} %3120, %"{{.*}}/runtime/internal/runtime.eface" %3119)
// CHECK-NEXT:   %3123 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3115, i32 0, i32 1
// CHECK-NEXT:   %3124 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3125 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3125, align 8
// CHECK-NEXT:   %3126 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3125, 1
// CHECK-NEXT:   %3127 = extractvalue { ptr, ptr } %3124, 1
// CHECK-NEXT:   %3128 = extractvalue { ptr, ptr } %3124, 0
// CHECK-NEXT:   %__llgo_funcval_code415 = call ptr asm "", "=r,0"(ptr %3128)
// CHECK-NEXT:   %3129 = call %reflect.Value %__llgo_funcval_code415(ptr {{(nest|swiftself)}} %3127, %"{{.*}}/runtime/internal/runtime.eface" %3126)
// CHECK-NEXT:   store %reflect.Value %3122, ptr %3116, align 8
// CHECK-NEXT:   store %reflect.Value %3129, ptr %3123, align 8
// CHECK-NEXT:   %3130 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 208
// CHECK-NEXT:   %3131 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3130, i32 0, i32 0
// CHECK-NEXT:   %3132 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3133 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3133, align 8
// CHECK-NEXT:   %3134 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3133, 1
// CHECK-NEXT:   %3135 = extractvalue { ptr, ptr } %3132, 1
// CHECK-NEXT:   %3136 = extractvalue { ptr, ptr } %3132, 0
// CHECK-NEXT:   %__llgo_funcval_code416 = call ptr asm "", "=r,0"(ptr %3136)
// CHECK-NEXT:   %3137 = call %reflect.Value %__llgo_funcval_code416(ptr {{(nest|swiftself)}} %3135, %"{{.*}}/runtime/internal/runtime.eface" %3134)
// CHECK-NEXT:   %3138 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3130, i32 0, i32 1
// CHECK-NEXT:   %3139 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3140 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %3140, align 8
// CHECK-NEXT:   %3141 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3140, 1
// CHECK-NEXT:   %3142 = extractvalue { ptr, ptr } %3139, 1
// CHECK-NEXT:   %3143 = extractvalue { ptr, ptr } %3139, 0
// CHECK-NEXT:   %__llgo_funcval_code417 = call ptr asm "", "=r,0"(ptr %3143)
// CHECK-NEXT:   %3144 = call %reflect.Value %__llgo_funcval_code417(ptr {{(nest|swiftself)}} %3142, %"{{.*}}/runtime/internal/runtime.eface" %3141)
// CHECK-NEXT:   store %reflect.Value %3137, ptr %3131, align 8
// CHECK-NEXT:   store %reflect.Value %3144, ptr %3138, align 8
// CHECK-NEXT:   %3145 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 209
// CHECK-NEXT:   %3146 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3145, i32 0, i32 0
// CHECK-NEXT:   %3147 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3148 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3148, align 8
// CHECK-NEXT:   %3149 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3148, 1
// CHECK-NEXT:   %3150 = extractvalue { ptr, ptr } %3147, 1
// CHECK-NEXT:   %3151 = extractvalue { ptr, ptr } %3147, 0
// CHECK-NEXT:   %__llgo_funcval_code418 = call ptr asm "", "=r,0"(ptr %3151)
// CHECK-NEXT:   %3152 = call %reflect.Value %__llgo_funcval_code418(ptr {{(nest|swiftself)}} %3150, %"{{.*}}/runtime/internal/runtime.eface" %3149)
// CHECK-NEXT:   %3153 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3145, i32 0, i32 1
// CHECK-NEXT:   %3154 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3155 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3156 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3155, ptr %3156, align 8
// CHECK-NEXT:   %3157 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3156, 1
// CHECK-NEXT:   %3158 = extractvalue { ptr, ptr } %3154, 1
// CHECK-NEXT:   %3159 = extractvalue { ptr, ptr } %3154, 0
// CHECK-NEXT:   %__llgo_funcval_code419 = call ptr asm "", "=r,0"(ptr %3159)
// CHECK-NEXT:   %3160 = call %reflect.Value %__llgo_funcval_code419(ptr {{(nest|swiftself)}} %3158, %"{{.*}}/runtime/internal/runtime.eface" %3157)
// CHECK-NEXT:   store %reflect.Value %3152, ptr %3146, align 8
// CHECK-NEXT:   store %reflect.Value %3160, ptr %3153, align 8
// CHECK-NEXT:   %3161 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 210
// CHECK-NEXT:   %3162 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3161, i32 0, i32 0
// CHECK-NEXT:   %3163 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3164 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3165 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3164, ptr %3165, align 8
// CHECK-NEXT:   %3166 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3165, 1
// CHECK-NEXT:   %3167 = extractvalue { ptr, ptr } %3163, 1
// CHECK-NEXT:   %3168 = extractvalue { ptr, ptr } %3163, 0
// CHECK-NEXT:   %__llgo_funcval_code420 = call ptr asm "", "=r,0"(ptr %3168)
// CHECK-NEXT:   %3169 = call %reflect.Value %__llgo_funcval_code420(ptr {{(nest|swiftself)}} %3167, %"{{.*}}/runtime/internal/runtime.eface" %3166)
// CHECK-NEXT:   %3170 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3161, i32 0, i32 1
// CHECK-NEXT:   %3171 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3172 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3172, align 8
// CHECK-NEXT:   %3173 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3172, 1
// CHECK-NEXT:   %3174 = extractvalue { ptr, ptr } %3171, 1
// CHECK-NEXT:   %3175 = extractvalue { ptr, ptr } %3171, 0
// CHECK-NEXT:   %__llgo_funcval_code421 = call ptr asm "", "=r,0"(ptr %3175)
// CHECK-NEXT:   %3176 = call %reflect.Value %__llgo_funcval_code421(ptr {{(nest|swiftself)}} %3174, %"{{.*}}/runtime/internal/runtime.eface" %3173)
// CHECK-NEXT:   store %reflect.Value %3169, ptr %3162, align 8
// CHECK-NEXT:   store %reflect.Value %3176, ptr %3170, align 8
// CHECK-NEXT:   %3177 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 211
// CHECK-NEXT:   %3178 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3177, i32 0, i32 0
// CHECK-NEXT:   %3179 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3180 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3181 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3180, ptr %3181, align 8
// CHECK-NEXT:   %3182 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3181, 1
// CHECK-NEXT:   %3183 = extractvalue { ptr, ptr } %3179, 1
// CHECK-NEXT:   %3184 = extractvalue { ptr, ptr } %3179, 0
// CHECK-NEXT:   %__llgo_funcval_code422 = call ptr asm "", "=r,0"(ptr %3184)
// CHECK-NEXT:   %3185 = call %reflect.Value %__llgo_funcval_code422(ptr {{(nest|swiftself)}} %3183, %"{{.*}}/runtime/internal/runtime.eface" %3182)
// CHECK-NEXT:   %3186 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3177, i32 0, i32 1
// CHECK-NEXT:   %3187 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3188 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3189 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3188, ptr %3189, align 8
// CHECK-NEXT:   %3190 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3189, 1
// CHECK-NEXT:   %3191 = extractvalue { ptr, ptr } %3187, 1
// CHECK-NEXT:   %3192 = extractvalue { ptr, ptr } %3187, 0
// CHECK-NEXT:   %__llgo_funcval_code423 = call ptr asm "", "=r,0"(ptr %3192)
// CHECK-NEXT:   %3193 = call %reflect.Value %__llgo_funcval_code423(ptr {{(nest|swiftself)}} %3191, %"{{.*}}/runtime/internal/runtime.eface" %3190)
// CHECK-NEXT:   store %reflect.Value %3185, ptr %3178, align 8
// CHECK-NEXT:   store %reflect.Value %3193, ptr %3186, align 8
// CHECK-NEXT:   %3194 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 212
// CHECK-NEXT:   %3195 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3194, i32 0, i32 0
// CHECK-NEXT:   %3196 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3197 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3197, align 8
// CHECK-NEXT:   %3198 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3197, 1
// CHECK-NEXT:   %3199 = extractvalue { ptr, ptr } %3196, 1
// CHECK-NEXT:   %3200 = extractvalue { ptr, ptr } %3196, 0
// CHECK-NEXT:   %__llgo_funcval_code424 = call ptr asm "", "=r,0"(ptr %3200)
// CHECK-NEXT:   %3201 = call %reflect.Value %__llgo_funcval_code424(ptr {{(nest|swiftself)}} %3199, %"{{.*}}/runtime/internal/runtime.eface" %3198)
// CHECK-NEXT:   %3202 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3194, i32 0, i32 1
// CHECK-NEXT:   %3203 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3204 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3205 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3204, ptr %3205, align 8
// CHECK-NEXT:   %3206 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3205, 1
// CHECK-NEXT:   %3207 = extractvalue { ptr, ptr } %3203, 1
// CHECK-NEXT:   %3208 = extractvalue { ptr, ptr } %3203, 0
// CHECK-NEXT:   %__llgo_funcval_code425 = call ptr asm "", "=r,0"(ptr %3208)
// CHECK-NEXT:   %3209 = call %reflect.Value %__llgo_funcval_code425(ptr {{(nest|swiftself)}} %3207, %"{{.*}}/runtime/internal/runtime.eface" %3206)
// CHECK-NEXT:   store %reflect.Value %3201, ptr %3195, align 8
// CHECK-NEXT:   store %reflect.Value %3209, ptr %3202, align 8
// CHECK-NEXT:   %3210 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 213
// CHECK-NEXT:   %3211 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3210, i32 0, i32 0
// CHECK-NEXT:   %3212 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3213 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3214 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3213, ptr %3214, align 8
// CHECK-NEXT:   %3215 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3214, 1
// CHECK-NEXT:   %3216 = extractvalue { ptr, ptr } %3212, 1
// CHECK-NEXT:   %3217 = extractvalue { ptr, ptr } %3212, 0
// CHECK-NEXT:   %__llgo_funcval_code426 = call ptr asm "", "=r,0"(ptr %3217)
// CHECK-NEXT:   %3218 = call %reflect.Value %__llgo_funcval_code426(ptr {{(nest|swiftself)}} %3216, %"{{.*}}/runtime/internal/runtime.eface" %3215)
// CHECK-NEXT:   %3219 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3210, i32 0, i32 1
// CHECK-NEXT:   %3220 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3221 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3221, align 8
// CHECK-NEXT:   %3222 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3221, 1
// CHECK-NEXT:   %3223 = extractvalue { ptr, ptr } %3220, 1
// CHECK-NEXT:   %3224 = extractvalue { ptr, ptr } %3220, 0
// CHECK-NEXT:   %__llgo_funcval_code427 = call ptr asm "", "=r,0"(ptr %3224)
// CHECK-NEXT:   %3225 = call %reflect.Value %__llgo_funcval_code427(ptr {{(nest|swiftself)}} %3223, %"{{.*}}/runtime/internal/runtime.eface" %3222)
// CHECK-NEXT:   store %reflect.Value %3218, ptr %3211, align 8
// CHECK-NEXT:   store %reflect.Value %3225, ptr %3219, align 8
// CHECK-NEXT:   %3226 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 214
// CHECK-NEXT:   %3227 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3226, i32 0, i32 0
// CHECK-NEXT:   %3228 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3229 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3230 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3229, ptr %3230, align 8
// CHECK-NEXT:   %3231 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3230, 1
// CHECK-NEXT:   %3232 = extractvalue { ptr, ptr } %3228, 1
// CHECK-NEXT:   %3233 = extractvalue { ptr, ptr } %3228, 0
// CHECK-NEXT:   %__llgo_funcval_code428 = call ptr asm "", "=r,0"(ptr %3233)
// CHECK-NEXT:   %3234 = call %reflect.Value %__llgo_funcval_code428(ptr {{(nest|swiftself)}} %3232, %"{{.*}}/runtime/internal/runtime.eface" %3231)
// CHECK-NEXT:   %3235 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3226, i32 0, i32 1
// CHECK-NEXT:   %3236 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3237 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3238 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3237, ptr %3238, align 8
// CHECK-NEXT:   %3239 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3238, 1
// CHECK-NEXT:   %3240 = extractvalue { ptr, ptr } %3236, 1
// CHECK-NEXT:   %3241 = extractvalue { ptr, ptr } %3236, 0
// CHECK-NEXT:   %__llgo_funcval_code429 = call ptr asm "", "=r,0"(ptr %3241)
// CHECK-NEXT:   %3242 = call %reflect.Value %__llgo_funcval_code429(ptr {{(nest|swiftself)}} %3240, %"{{.*}}/runtime/internal/runtime.eface" %3239)
// CHECK-NEXT:   store %reflect.Value %3234, ptr %3227, align 8
// CHECK-NEXT:   store %reflect.Value %3242, ptr %3235, align 8
// CHECK-NEXT:   %3243 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 215
// CHECK-NEXT:   %3244 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3243, i32 0, i32 0
// CHECK-NEXT:   %3245 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3246 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3247 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3246, ptr %3247, align 8
// CHECK-NEXT:   %3248 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3247, 1
// CHECK-NEXT:   %3249 = extractvalue { ptr, ptr } %3245, 1
// CHECK-NEXT:   %3250 = extractvalue { ptr, ptr } %3245, 0
// CHECK-NEXT:   %__llgo_funcval_code430 = call ptr asm "", "=r,0"(ptr %3250)
// CHECK-NEXT:   %3251 = call %reflect.Value %__llgo_funcval_code430(ptr {{(nest|swiftself)}} %3249, %"{{.*}}/runtime/internal/runtime.eface" %3248)
// CHECK-NEXT:   %3252 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3243, i32 0, i32 1
// CHECK-NEXT:   %3253 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3254 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3255 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3254, ptr %3255, align 8
// CHECK-NEXT:   %3256 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3255, 1
// CHECK-NEXT:   %3257 = extractvalue { ptr, ptr } %3253, 1
// CHECK-NEXT:   %3258 = extractvalue { ptr, ptr } %3253, 0
// CHECK-NEXT:   %__llgo_funcval_code431 = call ptr asm "", "=r,0"(ptr %3258)
// CHECK-NEXT:   %3259 = call %reflect.Value %__llgo_funcval_code431(ptr {{(nest|swiftself)}} %3257, %"{{.*}}/runtime/internal/runtime.eface" %3256)
// CHECK-NEXT:   store %reflect.Value %3251, ptr %3244, align 8
// CHECK-NEXT:   store %reflect.Value %3259, ptr %3252, align 8
// CHECK-NEXT:   %3260 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 216
// CHECK-NEXT:   %3261 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3260, i32 0, i32 0
// CHECK-NEXT:   %3262 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3263 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3264 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3263, ptr %3264, align 8
// CHECK-NEXT:   %3265 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3264, 1
// CHECK-NEXT:   %3266 = extractvalue { ptr, ptr } %3262, 1
// CHECK-NEXT:   %3267 = extractvalue { ptr, ptr } %3262, 0
// CHECK-NEXT:   %__llgo_funcval_code432 = call ptr asm "", "=r,0"(ptr %3267)
// CHECK-NEXT:   %3268 = call %reflect.Value %__llgo_funcval_code432(ptr {{(nest|swiftself)}} %3266, %"{{.*}}/runtime/internal/runtime.eface" %3265)
// CHECK-NEXT:   %3269 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3260, i32 0, i32 1
// CHECK-NEXT:   %3270 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3271 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3272 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3271, ptr %3272, align 8
// CHECK-NEXT:   %3273 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int32", ptr undef }, ptr %3272, 1
// CHECK-NEXT:   %3274 = extractvalue { ptr, ptr } %3270, 1
// CHECK-NEXT:   %3275 = extractvalue { ptr, ptr } %3270, 0
// CHECK-NEXT:   %__llgo_funcval_code433 = call ptr asm "", "=r,0"(ptr %3275)
// CHECK-NEXT:   %3276 = call %reflect.Value %__llgo_funcval_code433(ptr {{(nest|swiftself)}} %3274, %"{{.*}}/runtime/internal/runtime.eface" %3273)
// CHECK-NEXT:   store %reflect.Value %3268, ptr %3261, align 8
// CHECK-NEXT:   store %reflect.Value %3276, ptr %3269, align 8
// CHECK-NEXT:   %3277 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 217
// CHECK-NEXT:   %3278 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3277, i32 0, i32 0
// CHECK-NEXT:   %3279 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3280 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %3280, align 8
// CHECK-NEXT:   %3281 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %3280, 1
// CHECK-NEXT:   %3282 = extractvalue { ptr, ptr } %3279, 1
// CHECK-NEXT:   %3283 = extractvalue { ptr, ptr } %3279, 0
// CHECK-NEXT:   %__llgo_funcval_code434 = call ptr asm "", "=r,0"(ptr %3283)
// CHECK-NEXT:   %3284 = call %reflect.Value %__llgo_funcval_code434(ptr {{(nest|swiftself)}} %3282, %"{{.*}}/runtime/internal/runtime.eface" %3281)
// CHECK-NEXT:   %3285 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3277, i32 0, i32 1
// CHECK-NEXT:   %3286 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3287 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3287, align 8
// CHECK-NEXT:   %3288 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3287, 1
// CHECK-NEXT:   %3289 = extractvalue { ptr, ptr } %3286, 1
// CHECK-NEXT:   %3290 = extractvalue { ptr, ptr } %3286, 0
// CHECK-NEXT:   %__llgo_funcval_code435 = call ptr asm "", "=r,0"(ptr %3290)
// CHECK-NEXT:   %3291 = call %reflect.Value %__llgo_funcval_code435(ptr {{(nest|swiftself)}} %3289, %"{{.*}}/runtime/internal/runtime.eface" %3288)
// CHECK-NEXT:   store %reflect.Value %3284, ptr %3278, align 8
// CHECK-NEXT:   store %reflect.Value %3291, ptr %3285, align 8
// CHECK-NEXT:   %3292 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 218
// CHECK-NEXT:   %3293 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3292, i32 0, i32 0
// CHECK-NEXT:   %3294 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3295 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 97, ptr %3295, align 1
// CHECK-NEXT:   %3296 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %3295, 1
// CHECK-NEXT:   %3297 = extractvalue { ptr, ptr } %3294, 1
// CHECK-NEXT:   %3298 = extractvalue { ptr, ptr } %3294, 0
// CHECK-NEXT:   %__llgo_funcval_code436 = call ptr asm "", "=r,0"(ptr %3298)
// CHECK-NEXT:   %3299 = call %reflect.Value %__llgo_funcval_code436(ptr {{(nest|swiftself)}} %3297, %"{{.*}}/runtime/internal/runtime.eface" %3296)
// CHECK-NEXT:   %3300 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3292, i32 0, i32 1
// CHECK-NEXT:   %3301 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3302 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3302, align 8
// CHECK-NEXT:   %3303 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3302, 1
// CHECK-NEXT:   %3304 = extractvalue { ptr, ptr } %3301, 1
// CHECK-NEXT:   %3305 = extractvalue { ptr, ptr } %3301, 0
// CHECK-NEXT:   %__llgo_funcval_code437 = call ptr asm "", "=r,0"(ptr %3305)
// CHECK-NEXT:   %3306 = call %reflect.Value %__llgo_funcval_code437(ptr {{(nest|swiftself)}} %3304, %"{{.*}}/runtime/internal/runtime.eface" %3303)
// CHECK-NEXT:   store %reflect.Value %3299, ptr %3293, align 8
// CHECK-NEXT:   store %reflect.Value %3306, ptr %3300, align 8
// CHECK-NEXT:   %3307 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 219
// CHECK-NEXT:   %3308 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3307, i32 0, i32 0
// CHECK-NEXT:   %3309 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3310 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 97, ptr %3310, align 2
// CHECK-NEXT:   %3311 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %3310, 1
// CHECK-NEXT:   %3312 = extractvalue { ptr, ptr } %3309, 1
// CHECK-NEXT:   %3313 = extractvalue { ptr, ptr } %3309, 0
// CHECK-NEXT:   %__llgo_funcval_code438 = call ptr asm "", "=r,0"(ptr %3313)
// CHECK-NEXT:   %3314 = call %reflect.Value %__llgo_funcval_code438(ptr {{(nest|swiftself)}} %3312, %"{{.*}}/runtime/internal/runtime.eface" %3311)
// CHECK-NEXT:   %3315 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3307, i32 0, i32 1
// CHECK-NEXT:   %3316 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3317 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3317, align 8
// CHECK-NEXT:   %3318 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3317, 1
// CHECK-NEXT:   %3319 = extractvalue { ptr, ptr } %3316, 1
// CHECK-NEXT:   %3320 = extractvalue { ptr, ptr } %3316, 0
// CHECK-NEXT:   %__llgo_funcval_code439 = call ptr asm "", "=r,0"(ptr %3320)
// CHECK-NEXT:   %3321 = call %reflect.Value %__llgo_funcval_code439(ptr {{(nest|swiftself)}} %3319, %"{{.*}}/runtime/internal/runtime.eface" %3318)
// CHECK-NEXT:   store %reflect.Value %3314, ptr %3308, align 8
// CHECK-NEXT:   store %reflect.Value %3321, ptr %3315, align 8
// CHECK-NEXT:   %3322 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 220
// CHECK-NEXT:   %3323 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3322, i32 0, i32 0
// CHECK-NEXT:   %3324 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3325 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %3325, align 4
// CHECK-NEXT:   %3326 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %3325, 1
// CHECK-NEXT:   %3327 = extractvalue { ptr, ptr } %3324, 1
// CHECK-NEXT:   %3328 = extractvalue { ptr, ptr } %3324, 0
// CHECK-NEXT:   %__llgo_funcval_code440 = call ptr asm "", "=r,0"(ptr %3328)
// CHECK-NEXT:   %3329 = call %reflect.Value %__llgo_funcval_code440(ptr {{(nest|swiftself)}} %3327, %"{{.*}}/runtime/internal/runtime.eface" %3326)
// CHECK-NEXT:   %3330 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3322, i32 0, i32 1
// CHECK-NEXT:   %3331 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3332 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3332, align 8
// CHECK-NEXT:   %3333 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3332, 1
// CHECK-NEXT:   %3334 = extractvalue { ptr, ptr } %3331, 1
// CHECK-NEXT:   %3335 = extractvalue { ptr, ptr } %3331, 0
// CHECK-NEXT:   %__llgo_funcval_code441 = call ptr asm "", "=r,0"(ptr %3335)
// CHECK-NEXT:   %3336 = call %reflect.Value %__llgo_funcval_code441(ptr {{(nest|swiftself)}} %3334, %"{{.*}}/runtime/internal/runtime.eface" %3333)
// CHECK-NEXT:   store %reflect.Value %3329, ptr %3323, align 8
// CHECK-NEXT:   store %reflect.Value %3336, ptr %3330, align 8
// CHECK-NEXT:   %3337 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 221
// CHECK-NEXT:   %3338 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3337, i32 0, i32 0
// CHECK-NEXT:   %3339 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3340 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %3340, align 8
// CHECK-NEXT:   %3341 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %3340, 1
// CHECK-NEXT:   %3342 = extractvalue { ptr, ptr } %3339, 1
// CHECK-NEXT:   %3343 = extractvalue { ptr, ptr } %3339, 0
// CHECK-NEXT:   %__llgo_funcval_code442 = call ptr asm "", "=r,0"(ptr %3343)
// CHECK-NEXT:   %3344 = call %reflect.Value %__llgo_funcval_code442(ptr {{(nest|swiftself)}} %3342, %"{{.*}}/runtime/internal/runtime.eface" %3341)
// CHECK-NEXT:   %3345 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3337, i32 0, i32 1
// CHECK-NEXT:   %3346 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3347 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3347, align 8
// CHECK-NEXT:   %3348 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3347, 1
// CHECK-NEXT:   %3349 = extractvalue { ptr, ptr } %3346, 1
// CHECK-NEXT:   %3350 = extractvalue { ptr, ptr } %3346, 0
// CHECK-NEXT:   %__llgo_funcval_code443 = call ptr asm "", "=r,0"(ptr %3350)
// CHECK-NEXT:   %3351 = call %reflect.Value %__llgo_funcval_code443(ptr {{(nest|swiftself)}} %3349, %"{{.*}}/runtime/internal/runtime.eface" %3348)
// CHECK-NEXT:   store %reflect.Value %3344, ptr %3338, align 8
// CHECK-NEXT:   store %reflect.Value %3351, ptr %3345, align 8
// CHECK-NEXT:   %3352 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 222
// CHECK-NEXT:   %3353 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3352, i32 0, i32 0
// CHECK-NEXT:   %3354 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3355 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %3355, align 8
// CHECK-NEXT:   %3356 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %3355, 1
// CHECK-NEXT:   %3357 = extractvalue { ptr, ptr } %3354, 1
// CHECK-NEXT:   %3358 = extractvalue { ptr, ptr } %3354, 0
// CHECK-NEXT:   %__llgo_funcval_code444 = call ptr asm "", "=r,0"(ptr %3358)
// CHECK-NEXT:   %3359 = call %reflect.Value %__llgo_funcval_code444(ptr {{(nest|swiftself)}} %3357, %"{{.*}}/runtime/internal/runtime.eface" %3356)
// CHECK-NEXT:   %3360 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3352, i32 0, i32 1
// CHECK-NEXT:   %3361 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3362 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3362, align 8
// CHECK-NEXT:   %3363 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3362, 1
// CHECK-NEXT:   %3364 = extractvalue { ptr, ptr } %3361, 1
// CHECK-NEXT:   %3365 = extractvalue { ptr, ptr } %3361, 0
// CHECK-NEXT:   %__llgo_funcval_code445 = call ptr asm "", "=r,0"(ptr %3365)
// CHECK-NEXT:   %3366 = call %reflect.Value %__llgo_funcval_code445(ptr {{(nest|swiftself)}} %3364, %"{{.*}}/runtime/internal/runtime.eface" %3363)
// CHECK-NEXT:   store %reflect.Value %3359, ptr %3353, align 8
// CHECK-NEXT:   store %reflect.Value %3366, ptr %3360, align 8
// CHECK-NEXT:   %3367 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 223
// CHECK-NEXT:   %3368 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3367, i32 0, i32 0
// CHECK-NEXT:   %3369 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3370 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 97, ptr %3370, align 1
// CHECK-NEXT:   %3371 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint8, ptr undef }, ptr %3370, 1
// CHECK-NEXT:   %3372 = extractvalue { ptr, ptr } %3369, 1
// CHECK-NEXT:   %3373 = extractvalue { ptr, ptr } %3369, 0
// CHECK-NEXT:   %__llgo_funcval_code446 = call ptr asm "", "=r,0"(ptr %3373)
// CHECK-NEXT:   %3374 = call %reflect.Value %__llgo_funcval_code446(ptr {{(nest|swiftself)}} %3372, %"{{.*}}/runtime/internal/runtime.eface" %3371)
// CHECK-NEXT:   %3375 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3367, i32 0, i32 1
// CHECK-NEXT:   %3376 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3377 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3377, align 8
// CHECK-NEXT:   %3378 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3377, 1
// CHECK-NEXT:   %3379 = extractvalue { ptr, ptr } %3376, 1
// CHECK-NEXT:   %3380 = extractvalue { ptr, ptr } %3376, 0
// CHECK-NEXT:   %__llgo_funcval_code447 = call ptr asm "", "=r,0"(ptr %3380)
// CHECK-NEXT:   %3381 = call %reflect.Value %__llgo_funcval_code447(ptr {{(nest|swiftself)}} %3379, %"{{.*}}/runtime/internal/runtime.eface" %3378)
// CHECK-NEXT:   store %reflect.Value %3374, ptr %3368, align 8
// CHECK-NEXT:   store %reflect.Value %3381, ptr %3375, align 8
// CHECK-NEXT:   %3382 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 224
// CHECK-NEXT:   %3383 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3382, i32 0, i32 0
// CHECK-NEXT:   %3384 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3385 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 97, ptr %3385, align 2
// CHECK-NEXT:   %3386 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint16, ptr undef }, ptr %3385, 1
// CHECK-NEXT:   %3387 = extractvalue { ptr, ptr } %3384, 1
// CHECK-NEXT:   %3388 = extractvalue { ptr, ptr } %3384, 0
// CHECK-NEXT:   %__llgo_funcval_code448 = call ptr asm "", "=r,0"(ptr %3388)
// CHECK-NEXT:   %3389 = call %reflect.Value %__llgo_funcval_code448(ptr {{(nest|swiftself)}} %3387, %"{{.*}}/runtime/internal/runtime.eface" %3386)
// CHECK-NEXT:   %3390 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3382, i32 0, i32 1
// CHECK-NEXT:   %3391 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3392 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3392, align 8
// CHECK-NEXT:   %3393 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3392, 1
// CHECK-NEXT:   %3394 = extractvalue { ptr, ptr } %3391, 1
// CHECK-NEXT:   %3395 = extractvalue { ptr, ptr } %3391, 0
// CHECK-NEXT:   %__llgo_funcval_code449 = call ptr asm "", "=r,0"(ptr %3395)
// CHECK-NEXT:   %3396 = call %reflect.Value %__llgo_funcval_code449(ptr {{(nest|swiftself)}} %3394, %"{{.*}}/runtime/internal/runtime.eface" %3393)
// CHECK-NEXT:   store %reflect.Value %3389, ptr %3383, align 8
// CHECK-NEXT:   store %reflect.Value %3396, ptr %3390, align 8
// CHECK-NEXT:   %3397 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 225
// CHECK-NEXT:   %3398 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3397, i32 0, i32 0
// CHECK-NEXT:   %3399 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3400 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 97, ptr %3400, align 4
// CHECK-NEXT:   %3401 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %3400, 1
// CHECK-NEXT:   %3402 = extractvalue { ptr, ptr } %3399, 1
// CHECK-NEXT:   %3403 = extractvalue { ptr, ptr } %3399, 0
// CHECK-NEXT:   %__llgo_funcval_code450 = call ptr asm "", "=r,0"(ptr %3403)
// CHECK-NEXT:   %3404 = call %reflect.Value %__llgo_funcval_code450(ptr {{(nest|swiftself)}} %3402, %"{{.*}}/runtime/internal/runtime.eface" %3401)
// CHECK-NEXT:   %3405 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3397, i32 0, i32 1
// CHECK-NEXT:   %3406 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3407 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3407, align 8
// CHECK-NEXT:   %3408 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3407, 1
// CHECK-NEXT:   %3409 = extractvalue { ptr, ptr } %3406, 1
// CHECK-NEXT:   %3410 = extractvalue { ptr, ptr } %3406, 0
// CHECK-NEXT:   %__llgo_funcval_code451 = call ptr asm "", "=r,0"(ptr %3410)
// CHECK-NEXT:   %3411 = call %reflect.Value %__llgo_funcval_code451(ptr {{(nest|swiftself)}} %3409, %"{{.*}}/runtime/internal/runtime.eface" %3408)
// CHECK-NEXT:   store %reflect.Value %3404, ptr %3398, align 8
// CHECK-NEXT:   store %reflect.Value %3411, ptr %3405, align 8
// CHECK-NEXT:   %3412 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 226
// CHECK-NEXT:   %3413 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3412, i32 0, i32 0
// CHECK-NEXT:   %3414 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3415 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %3415, align 8
// CHECK-NEXT:   %3416 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %3415, 1
// CHECK-NEXT:   %3417 = extractvalue { ptr, ptr } %3414, 1
// CHECK-NEXT:   %3418 = extractvalue { ptr, ptr } %3414, 0
// CHECK-NEXT:   %__llgo_funcval_code452 = call ptr asm "", "=r,0"(ptr %3418)
// CHECK-NEXT:   %3419 = call %reflect.Value %__llgo_funcval_code452(ptr {{(nest|swiftself)}} %3417, %"{{.*}}/runtime/internal/runtime.eface" %3416)
// CHECK-NEXT:   %3420 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3412, i32 0, i32 1
// CHECK-NEXT:   %3421 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3422 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3422, align 8
// CHECK-NEXT:   %3423 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3422, 1
// CHECK-NEXT:   %3424 = extractvalue { ptr, ptr } %3421, 1
// CHECK-NEXT:   %3425 = extractvalue { ptr, ptr } %3421, 0
// CHECK-NEXT:   %__llgo_funcval_code453 = call ptr asm "", "=r,0"(ptr %3425)
// CHECK-NEXT:   %3426 = call %reflect.Value %__llgo_funcval_code453(ptr {{(nest|swiftself)}} %3424, %"{{.*}}/runtime/internal/runtime.eface" %3423)
// CHECK-NEXT:   store %reflect.Value %3419, ptr %3413, align 8
// CHECK-NEXT:   store %reflect.Value %3426, ptr %3420, align 8
// CHECK-NEXT:   %3427 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 227
// CHECK-NEXT:   %3428 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3427, i32 0, i32 0
// CHECK-NEXT:   %3429 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3430 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 97, ptr %3430, align 8
// CHECK-NEXT:   %3431 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %3430, 1
// CHECK-NEXT:   %3432 = extractvalue { ptr, ptr } %3429, 1
// CHECK-NEXT:   %3433 = extractvalue { ptr, ptr } %3429, 0
// CHECK-NEXT:   %__llgo_funcval_code454 = call ptr asm "", "=r,0"(ptr %3433)
// CHECK-NEXT:   %3434 = call %reflect.Value %__llgo_funcval_code454(ptr {{(nest|swiftself)}} %3432, %"{{.*}}/runtime/internal/runtime.eface" %3431)
// CHECK-NEXT:   %3435 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3427, i32 0, i32 1
// CHECK-NEXT:   %3436 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3437 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 1 }, ptr %3437, align 8
// CHECK-NEXT:   %3438 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3437, 1
// CHECK-NEXT:   %3439 = extractvalue { ptr, ptr } %3436, 1
// CHECK-NEXT:   %3440 = extractvalue { ptr, ptr } %3436, 0
// CHECK-NEXT:   %__llgo_funcval_code455 = call ptr asm "", "=r,0"(ptr %3440)
// CHECK-NEXT:   %3441 = call %reflect.Value %__llgo_funcval_code455(ptr {{(nest|swiftself)}} %3439, %"{{.*}}/runtime/internal/runtime.eface" %3438)
// CHECK-NEXT:   store %reflect.Value %3434, ptr %3428, align 8
// CHECK-NEXT:   store %reflect.Value %3441, ptr %3435, align 8
// CHECK-NEXT:   %3442 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 228
// CHECK-NEXT:   %3443 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3442, i32 0, i32 0
// CHECK-NEXT:   %3444 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3445 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 -1, ptr %3445, align 8
// CHECK-NEXT:   %3446 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %3445, 1
// CHECK-NEXT:   %3447 = extractvalue { ptr, ptr } %3444, 1
// CHECK-NEXT:   %3448 = extractvalue { ptr, ptr } %3444, 0
// CHECK-NEXT:   %__llgo_funcval_code456 = call ptr asm "", "=r,0"(ptr %3448)
// CHECK-NEXT:   %3449 = call %reflect.Value %__llgo_funcval_code456(ptr {{(nest|swiftself)}} %3447, %"{{.*}}/runtime/internal/runtime.eface" %3446)
// CHECK-NEXT:   %3450 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3442, i32 0, i32 1
// CHECK-NEXT:   %3451 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3452 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3452, align 8
// CHECK-NEXT:   %3453 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3452, 1
// CHECK-NEXT:   %3454 = extractvalue { ptr, ptr } %3451, 1
// CHECK-NEXT:   %3455 = extractvalue { ptr, ptr } %3451, 0
// CHECK-NEXT:   %__llgo_funcval_code457 = call ptr asm "", "=r,0"(ptr %3455)
// CHECK-NEXT:   %3456 = call %reflect.Value %__llgo_funcval_code457(ptr {{(nest|swiftself)}} %3454, %"{{.*}}/runtime/internal/runtime.eface" %3453)
// CHECK-NEXT:   store %reflect.Value %3449, ptr %3443, align 8
// CHECK-NEXT:   store %reflect.Value %3456, ptr %3450, align 8
// CHECK-NEXT:   %3457 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 229
// CHECK-NEXT:   %3458 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3457, i32 0, i32 0
// CHECK-NEXT:   %3459 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3460 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store i8 -2, ptr %3460, align 1
// CHECK-NEXT:   %3461 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int8, ptr undef }, ptr %3460, 1
// CHECK-NEXT:   %3462 = extractvalue { ptr, ptr } %3459, 1
// CHECK-NEXT:   %3463 = extractvalue { ptr, ptr } %3459, 0
// CHECK-NEXT:   %__llgo_funcval_code458 = call ptr asm "", "=r,0"(ptr %3463)
// CHECK-NEXT:   %3464 = call %reflect.Value %__llgo_funcval_code458(ptr {{(nest|swiftself)}} %3462, %"{{.*}}/runtime/internal/runtime.eface" %3461)
// CHECK-NEXT:   %3465 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3457, i32 0, i32 1
// CHECK-NEXT:   %3466 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3467 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3467, align 8
// CHECK-NEXT:   %3468 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3467, 1
// CHECK-NEXT:   %3469 = extractvalue { ptr, ptr } %3466, 1
// CHECK-NEXT:   %3470 = extractvalue { ptr, ptr } %3466, 0
// CHECK-NEXT:   %__llgo_funcval_code459 = call ptr asm "", "=r,0"(ptr %3470)
// CHECK-NEXT:   %3471 = call %reflect.Value %__llgo_funcval_code459(ptr {{(nest|swiftself)}} %3469, %"{{.*}}/runtime/internal/runtime.eface" %3468)
// CHECK-NEXT:   store %reflect.Value %3464, ptr %3458, align 8
// CHECK-NEXT:   store %reflect.Value %3471, ptr %3465, align 8
// CHECK-NEXT:   %3472 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 230
// CHECK-NEXT:   %3473 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3472, i32 0, i32 0
// CHECK-NEXT:   %3474 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3475 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store i16 -3, ptr %3475, align 2
// CHECK-NEXT:   %3476 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int16, ptr undef }, ptr %3475, 1
// CHECK-NEXT:   %3477 = extractvalue { ptr, ptr } %3474, 1
// CHECK-NEXT:   %3478 = extractvalue { ptr, ptr } %3474, 0
// CHECK-NEXT:   %__llgo_funcval_code460 = call ptr asm "", "=r,0"(ptr %3478)
// CHECK-NEXT:   %3479 = call %reflect.Value %__llgo_funcval_code460(ptr {{(nest|swiftself)}} %3477, %"{{.*}}/runtime/internal/runtime.eface" %3476)
// CHECK-NEXT:   %3480 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3472, i32 0, i32 1
// CHECK-NEXT:   %3481 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3482 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3482, align 8
// CHECK-NEXT:   %3483 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3482, 1
// CHECK-NEXT:   %3484 = extractvalue { ptr, ptr } %3481, 1
// CHECK-NEXT:   %3485 = extractvalue { ptr, ptr } %3481, 0
// CHECK-NEXT:   %__llgo_funcval_code461 = call ptr asm "", "=r,0"(ptr %3485)
// CHECK-NEXT:   %3486 = call %reflect.Value %__llgo_funcval_code461(ptr {{(nest|swiftself)}} %3484, %"{{.*}}/runtime/internal/runtime.eface" %3483)
// CHECK-NEXT:   store %reflect.Value %3479, ptr %3473, align 8
// CHECK-NEXT:   store %reflect.Value %3486, ptr %3480, align 8
// CHECK-NEXT:   %3487 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 231
// CHECK-NEXT:   %3488 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3487, i32 0, i32 0
// CHECK-NEXT:   %3489 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3490 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 -4, ptr %3490, align 4
// CHECK-NEXT:   %3491 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int32, ptr undef }, ptr %3490, 1
// CHECK-NEXT:   %3492 = extractvalue { ptr, ptr } %3489, 1
// CHECK-NEXT:   %3493 = extractvalue { ptr, ptr } %3489, 0
// CHECK-NEXT:   %__llgo_funcval_code462 = call ptr asm "", "=r,0"(ptr %3493)
// CHECK-NEXT:   %3494 = call %reflect.Value %__llgo_funcval_code462(ptr {{(nest|swiftself)}} %3492, %"{{.*}}/runtime/internal/runtime.eface" %3491)
// CHECK-NEXT:   %3495 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3487, i32 0, i32 1
// CHECK-NEXT:   %3496 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3497 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3497, align 8
// CHECK-NEXT:   %3498 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3497, 1
// CHECK-NEXT:   %3499 = extractvalue { ptr, ptr } %3496, 1
// CHECK-NEXT:   %3500 = extractvalue { ptr, ptr } %3496, 0
// CHECK-NEXT:   %__llgo_funcval_code463 = call ptr asm "", "=r,0"(ptr %3500)
// CHECK-NEXT:   %3501 = call %reflect.Value %__llgo_funcval_code463(ptr {{(nest|swiftself)}} %3499, %"{{.*}}/runtime/internal/runtime.eface" %3498)
// CHECK-NEXT:   store %reflect.Value %3494, ptr %3488, align 8
// CHECK-NEXT:   store %reflect.Value %3501, ptr %3495, align 8
// CHECK-NEXT:   %3502 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 232
// CHECK-NEXT:   %3503 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3502, i32 0, i32 0
// CHECK-NEXT:   %3504 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3505 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 -5, ptr %3505, align 8
// CHECK-NEXT:   %3506 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int64, ptr undef }, ptr %3505, 1
// CHECK-NEXT:   %3507 = extractvalue { ptr, ptr } %3504, 1
// CHECK-NEXT:   %3508 = extractvalue { ptr, ptr } %3504, 0
// CHECK-NEXT:   %__llgo_funcval_code464 = call ptr asm "", "=r,0"(ptr %3508)
// CHECK-NEXT:   %3509 = call %reflect.Value %__llgo_funcval_code464(ptr {{(nest|swiftself)}} %3507, %"{{.*}}/runtime/internal/runtime.eface" %3506)
// CHECK-NEXT:   %3510 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3502, i32 0, i32 1
// CHECK-NEXT:   %3511 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3512 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3512, align 8
// CHECK-NEXT:   %3513 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3512, 1
// CHECK-NEXT:   %3514 = extractvalue { ptr, ptr } %3511, 1
// CHECK-NEXT:   %3515 = extractvalue { ptr, ptr } %3511, 0
// CHECK-NEXT:   %__llgo_funcval_code465 = call ptr asm "", "=r,0"(ptr %3515)
// CHECK-NEXT:   %3516 = call %reflect.Value %__llgo_funcval_code465(ptr {{(nest|swiftself)}} %3514, %"{{.*}}/runtime/internal/runtime.eface" %3513)
// CHECK-NEXT:   store %reflect.Value %3509, ptr %3503, align 8
// CHECK-NEXT:   store %reflect.Value %3516, ptr %3510, align 8
// CHECK-NEXT:   %3517 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 233
// CHECK-NEXT:   %3518 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3517, i32 0, i32 0
// CHECK-NEXT:   %3519 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3520 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114113, ptr %3520, align 8
// CHECK-NEXT:   %3521 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint, ptr undef }, ptr %3520, 1
// CHECK-NEXT:   %3522 = extractvalue { ptr, ptr } %3519, 1
// CHECK-NEXT:   %3523 = extractvalue { ptr, ptr } %3519, 0
// CHECK-NEXT:   %__llgo_funcval_code466 = call ptr asm "", "=r,0"(ptr %3523)
// CHECK-NEXT:   %3524 = call %reflect.Value %__llgo_funcval_code466(ptr {{(nest|swiftself)}} %3522, %"{{.*}}/runtime/internal/runtime.eface" %3521)
// CHECK-NEXT:   %3525 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3517, i32 0, i32 1
// CHECK-NEXT:   %3526 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3527 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3527, align 8
// CHECK-NEXT:   %3528 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3527, 1
// CHECK-NEXT:   %3529 = extractvalue { ptr, ptr } %3526, 1
// CHECK-NEXT:   %3530 = extractvalue { ptr, ptr } %3526, 0
// CHECK-NEXT:   %__llgo_funcval_code467 = call ptr asm "", "=r,0"(ptr %3530)
// CHECK-NEXT:   %3531 = call %reflect.Value %__llgo_funcval_code467(ptr {{(nest|swiftself)}} %3529, %"{{.*}}/runtime/internal/runtime.eface" %3528)
// CHECK-NEXT:   store %reflect.Value %3524, ptr %3518, align 8
// CHECK-NEXT:   store %reflect.Value %3531, ptr %3525, align 8
// CHECK-NEXT:   %3532 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 234
// CHECK-NEXT:   %3533 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3532, i32 0, i32 0
// CHECK-NEXT:   %3534 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3535 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store i32 1114114, ptr %3535, align 4
// CHECK-NEXT:   %3536 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint32, ptr undef }, ptr %3535, 1
// CHECK-NEXT:   %3537 = extractvalue { ptr, ptr } %3534, 1
// CHECK-NEXT:   %3538 = extractvalue { ptr, ptr } %3534, 0
// CHECK-NEXT:   %__llgo_funcval_code468 = call ptr asm "", "=r,0"(ptr %3538)
// CHECK-NEXT:   %3539 = call %reflect.Value %__llgo_funcval_code468(ptr {{(nest|swiftself)}} %3537, %"{{.*}}/runtime/internal/runtime.eface" %3536)
// CHECK-NEXT:   %3540 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3532, i32 0, i32 1
// CHECK-NEXT:   %3541 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3542 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3542, align 8
// CHECK-NEXT:   %3543 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3542, 1
// CHECK-NEXT:   %3544 = extractvalue { ptr, ptr } %3541, 1
// CHECK-NEXT:   %3545 = extractvalue { ptr, ptr } %3541, 0
// CHECK-NEXT:   %__llgo_funcval_code469 = call ptr asm "", "=r,0"(ptr %3545)
// CHECK-NEXT:   %3546 = call %reflect.Value %__llgo_funcval_code469(ptr {{(nest|swiftself)}} %3544, %"{{.*}}/runtime/internal/runtime.eface" %3543)
// CHECK-NEXT:   store %reflect.Value %3539, ptr %3533, align 8
// CHECK-NEXT:   store %reflect.Value %3546, ptr %3540, align 8
// CHECK-NEXT:   %3547 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 235
// CHECK-NEXT:   %3548 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3547, i32 0, i32 0
// CHECK-NEXT:   %3549 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3550 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114115, ptr %3550, align 8
// CHECK-NEXT:   %3551 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uint64, ptr undef }, ptr %3550, 1
// CHECK-NEXT:   %3552 = extractvalue { ptr, ptr } %3549, 1
// CHECK-NEXT:   %3553 = extractvalue { ptr, ptr } %3549, 0
// CHECK-NEXT:   %__llgo_funcval_code470 = call ptr asm "", "=r,0"(ptr %3553)
// CHECK-NEXT:   %3554 = call %reflect.Value %__llgo_funcval_code470(ptr {{(nest|swiftself)}} %3552, %"{{.*}}/runtime/internal/runtime.eface" %3551)
// CHECK-NEXT:   %3555 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3547, i32 0, i32 1
// CHECK-NEXT:   %3556 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3557 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3557, align 8
// CHECK-NEXT:   %3558 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3557, 1
// CHECK-NEXT:   %3559 = extractvalue { ptr, ptr } %3556, 1
// CHECK-NEXT:   %3560 = extractvalue { ptr, ptr } %3556, 0
// CHECK-NEXT:   %__llgo_funcval_code471 = call ptr asm "", "=r,0"(ptr %3560)
// CHECK-NEXT:   %3561 = call %reflect.Value %__llgo_funcval_code471(ptr {{(nest|swiftself)}} %3559, %"{{.*}}/runtime/internal/runtime.eface" %3558)
// CHECK-NEXT:   store %reflect.Value %3554, ptr %3548, align 8
// CHECK-NEXT:   store %reflect.Value %3561, ptr %3555, align 8
// CHECK-NEXT:   %3562 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 236
// CHECK-NEXT:   %3563 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3562, i32 0, i32 0
// CHECK-NEXT:   %3564 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3565 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1114116, ptr %3565, align 8
// CHECK-NEXT:   %3566 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_uintptr, ptr undef }, ptr %3565, 1
// CHECK-NEXT:   %3567 = extractvalue { ptr, ptr } %3564, 1
// CHECK-NEXT:   %3568 = extractvalue { ptr, ptr } %3564, 0
// CHECK-NEXT:   %__llgo_funcval_code472 = call ptr asm "", "=r,0"(ptr %3568)
// CHECK-NEXT:   %3569 = call %reflect.Value %__llgo_funcval_code472(ptr {{(nest|swiftself)}} %3567, %"{{.*}}/runtime/internal/runtime.eface" %3566)
// CHECK-NEXT:   %3570 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3562, i32 0, i32 1
// CHECK-NEXT:   %3571 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3572 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 }, ptr %3572, align 8
// CHECK-NEXT:   %3573 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3572, 1
// CHECK-NEXT:   %3574 = extractvalue { ptr, ptr } %3571, 1
// CHECK-NEXT:   %3575 = extractvalue { ptr, ptr } %3571, 0
// CHECK-NEXT:   %__llgo_funcval_code473 = call ptr asm "", "=r,0"(ptr %3575)
// CHECK-NEXT:   %3576 = call %reflect.Value %__llgo_funcval_code473(ptr {{(nest|swiftself)}} %3574, %"{{.*}}/runtime/internal/runtime.eface" %3573)
// CHECK-NEXT:   store %reflect.Value %3569, ptr %3563, align 8
// CHECK-NEXT:   store %reflect.Value %3576, ptr %3570, align 8
// CHECK-NEXT:   %3577 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 237
// CHECK-NEXT:   %3578 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3577, i32 0, i32 0
// CHECK-NEXT:   %3579 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3580 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3580, align 8
// CHECK-NEXT:   %3581 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3580, 1
// CHECK-NEXT:   %3582 = extractvalue { ptr, ptr } %3579, 1
// CHECK-NEXT:   %3583 = extractvalue { ptr, ptr } %3579, 0
// CHECK-NEXT:   %__llgo_funcval_code474 = call ptr asm "", "=r,0"(ptr %3583)
// CHECK-NEXT:   %3584 = call %reflect.Value %__llgo_funcval_code474(ptr {{(nest|swiftself)}} %3582, %"{{.*}}/runtime/internal/runtime.eface" %3581)
// CHECK-NEXT:   %3585 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3577, i32 0, i32 1
// CHECK-NEXT:   %3586 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3587 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3588 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3587, ptr %3588, align 8
// CHECK-NEXT:   %3589 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3588, 1
// CHECK-NEXT:   %3590 = extractvalue { ptr, ptr } %3586, 1
// CHECK-NEXT:   %3591 = extractvalue { ptr, ptr } %3586, 0
// CHECK-NEXT:   %__llgo_funcval_code475 = call ptr asm "", "=r,0"(ptr %3591)
// CHECK-NEXT:   %3592 = call %reflect.Value %__llgo_funcval_code475(ptr {{(nest|swiftself)}} %3590, %"{{.*}}/runtime/internal/runtime.eface" %3589)
// CHECK-NEXT:   store %reflect.Value %3584, ptr %3578, align 8
// CHECK-NEXT:   store %reflect.Value %3592, ptr %3585, align 8
// CHECK-NEXT:   %3593 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 238
// CHECK-NEXT:   %3594 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3593, i32 0, i32 0
// CHECK-NEXT:   %3595 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3596 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3597 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3596, ptr %3597, align 8
// CHECK-NEXT:   %3598 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3597, 1
// CHECK-NEXT:   %3599 = extractvalue { ptr, ptr } %3595, 1
// CHECK-NEXT:   %3600 = extractvalue { ptr, ptr } %3595, 0
// CHECK-NEXT:   %__llgo_funcval_code476 = call ptr asm "", "=r,0"(ptr %3600)
// CHECK-NEXT:   %3601 = call %reflect.Value %__llgo_funcval_code476(ptr {{(nest|swiftself)}} %3599, %"{{.*}}/runtime/internal/runtime.eface" %3598)
// CHECK-NEXT:   %3602 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3593, i32 0, i32 1
// CHECK-NEXT:   %3603 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3604 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3604, align 8
// CHECK-NEXT:   %3605 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3604, 1
// CHECK-NEXT:   %3606 = extractvalue { ptr, ptr } %3603, 1
// CHECK-NEXT:   %3607 = extractvalue { ptr, ptr } %3603, 0
// CHECK-NEXT:   %__llgo_funcval_code477 = call ptr asm "", "=r,0"(ptr %3607)
// CHECK-NEXT:   %3608 = call %reflect.Value %__llgo_funcval_code477(ptr {{(nest|swiftself)}} %3606, %"{{.*}}/runtime/internal/runtime.eface" %3605)
// CHECK-NEXT:   store %reflect.Value %3601, ptr %3594, align 8
// CHECK-NEXT:   store %reflect.Value %3608, ptr %3602, align 8
// CHECK-NEXT:   %3609 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 239
// CHECK-NEXT:   %3610 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3609, i32 0, i32 0
// CHECK-NEXT:   %3611 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3612 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3613 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3612, ptr %3613, align 8
// CHECK-NEXT:   %3614 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3613, 1
// CHECK-NEXT:   %3615 = extractvalue { ptr, ptr } %3611, 1
// CHECK-NEXT:   %3616 = extractvalue { ptr, ptr } %3611, 0
// CHECK-NEXT:   %__llgo_funcval_code478 = call ptr asm "", "=r,0"(ptr %3616)
// CHECK-NEXT:   %3617 = call %reflect.Value %__llgo_funcval_code478(ptr {{(nest|swiftself)}} %3615, %"{{.*}}/runtime/internal/runtime.eface" %3614)
// CHECK-NEXT:   %3618 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3609, i32 0, i32 1
// CHECK-NEXT:   %3619 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3620 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3621 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3620, ptr %3621, align 8
// CHECK-NEXT:   %3622 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3621, 1
// CHECK-NEXT:   %3623 = extractvalue { ptr, ptr } %3619, 1
// CHECK-NEXT:   %3624 = extractvalue { ptr, ptr } %3619, 0
// CHECK-NEXT:   %__llgo_funcval_code479 = call ptr asm "", "=r,0"(ptr %3624)
// CHECK-NEXT:   %3625 = call %reflect.Value %__llgo_funcval_code479(ptr {{(nest|swiftself)}} %3623, %"{{.*}}/runtime/internal/runtime.eface" %3622)
// CHECK-NEXT:   store %reflect.Value %3617, ptr %3610, align 8
// CHECK-NEXT:   store %reflect.Value %3625, ptr %3618, align 8
// CHECK-NEXT:   %3626 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 240
// CHECK-NEXT:   %3627 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3626, i32 0, i32 0
// CHECK-NEXT:   %3628 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3629 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3629, align 8
// CHECK-NEXT:   %3630 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3629, 1
// CHECK-NEXT:   %3631 = extractvalue { ptr, ptr } %3628, 1
// CHECK-NEXT:   %3632 = extractvalue { ptr, ptr } %3628, 0
// CHECK-NEXT:   %__llgo_funcval_code480 = call ptr asm "", "=r,0"(ptr %3632)
// CHECK-NEXT:   %3633 = call %reflect.Value %__llgo_funcval_code480(ptr {{(nest|swiftself)}} %3631, %"{{.*}}/runtime/internal/runtime.eface" %3630)
// CHECK-NEXT:   %3634 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3626, i32 0, i32 1
// CHECK-NEXT:   %3635 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3636 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3637 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3636, ptr %3637, align 8
// CHECK-NEXT:   %3638 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3637, 1
// CHECK-NEXT:   %3639 = extractvalue { ptr, ptr } %3635, 1
// CHECK-NEXT:   %3640 = extractvalue { ptr, ptr } %3635, 0
// CHECK-NEXT:   %__llgo_funcval_code481 = call ptr asm "", "=r,0"(ptr %3640)
// CHECK-NEXT:   %3641 = call %reflect.Value %__llgo_funcval_code481(ptr {{(nest|swiftself)}} %3639, %"{{.*}}/runtime/internal/runtime.eface" %3638)
// CHECK-NEXT:   store %reflect.Value %3633, ptr %3627, align 8
// CHECK-NEXT:   store %reflect.Value %3641, ptr %3634, align 8
// CHECK-NEXT:   %3642 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 241
// CHECK-NEXT:   %3643 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3642, i32 0, i32 0
// CHECK-NEXT:   %3644 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3645 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToBytes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 })
// CHECK-NEXT:   %3646 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3645, ptr %3646, align 8
// CHECK-NEXT:   %3647 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3646, 1
// CHECK-NEXT:   %3648 = extractvalue { ptr, ptr } %3644, 1
// CHECK-NEXT:   %3649 = extractvalue { ptr, ptr } %3644, 0
// CHECK-NEXT:   %__llgo_funcval_code482 = call ptr asm "", "=r,0"(ptr %3649)
// CHECK-NEXT:   %3650 = call %reflect.Value %__llgo_funcval_code482(ptr {{(nest|swiftself)}} %3648, %"{{.*}}/runtime/internal/runtime.eface" %3647)
// CHECK-NEXT:   %3651 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3642, i32 0, i32 1
// CHECK-NEXT:   %3652 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3653 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 6 }, ptr %3653, align 8
// CHECK-NEXT:   %3654 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3653, 1
// CHECK-NEXT:   %3655 = extractvalue { ptr, ptr } %3652, 1
// CHECK-NEXT:   %3656 = extractvalue { ptr, ptr } %3652, 0
// CHECK-NEXT:   %__llgo_funcval_code483 = call ptr asm "", "=r,0"(ptr %3656)
// CHECK-NEXT:   %3657 = call %reflect.Value %__llgo_funcval_code483(ptr {{(nest|swiftself)}} %3655, %"{{.*}}/runtime/internal/runtime.eface" %3654)
// CHECK-NEXT:   store %reflect.Value %3650, ptr %3643, align 8
// CHECK-NEXT:   store %reflect.Value %3657, ptr %3651, align 8
// CHECK-NEXT:   %3658 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 242
// CHECK-NEXT:   %3659 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3658, i32 0, i32 0
// CHECK-NEXT:   %3660 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3661 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3661, align 8
// CHECK-NEXT:   %3662 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3661, 1
// CHECK-NEXT:   %3663 = extractvalue { ptr, ptr } %3660, 1
// CHECK-NEXT:   %3664 = extractvalue { ptr, ptr } %3660, 0
// CHECK-NEXT:   %__llgo_funcval_code484 = call ptr asm "", "=r,0"(ptr %3664)
// CHECK-NEXT:   %3665 = call %reflect.Value %__llgo_funcval_code484(ptr {{(nest|swiftself)}} %3663, %"{{.*}}/runtime/internal/runtime.eface" %3662)
// CHECK-NEXT:   %3666 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3658, i32 0, i32 1
// CHECK-NEXT:   %3667 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3668 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3669 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3668, ptr %3669, align 8
// CHECK-NEXT:   %3670 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3669, 1
// CHECK-NEXT:   %3671 = extractvalue { ptr, ptr } %3667, 1
// CHECK-NEXT:   %3672 = extractvalue { ptr, ptr } %3667, 0
// CHECK-NEXT:   %__llgo_funcval_code485 = call ptr asm "", "=r,0"(ptr %3672)
// CHECK-NEXT:   %3673 = call %reflect.Value %__llgo_funcval_code485(ptr {{(nest|swiftself)}} %3671, %"{{.*}}/runtime/internal/runtime.eface" %3670)
// CHECK-NEXT:   store %reflect.Value %3665, ptr %3659, align 8
// CHECK-NEXT:   store %reflect.Value %3673, ptr %3666, align 8
// CHECK-NEXT:   %3674 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 243
// CHECK-NEXT:   %3675 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3674, i32 0, i32 0
// CHECK-NEXT:   %3676 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3677 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3678 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3677, ptr %3678, align 8
// CHECK-NEXT:   %3679 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3678, 1
// CHECK-NEXT:   %3680 = extractvalue { ptr, ptr } %3676, 1
// CHECK-NEXT:   %3681 = extractvalue { ptr, ptr } %3676, 0
// CHECK-NEXT:   %__llgo_funcval_code486 = call ptr asm "", "=r,0"(ptr %3681)
// CHECK-NEXT:   %3682 = call %reflect.Value %__llgo_funcval_code486(ptr {{(nest|swiftself)}} %3680, %"{{.*}}/runtime/internal/runtime.eface" %3679)
// CHECK-NEXT:   %3683 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3674, i32 0, i32 1
// CHECK-NEXT:   %3684 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3685 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3685, align 8
// CHECK-NEXT:   %3686 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %3685, 1
// CHECK-NEXT:   %3687 = extractvalue { ptr, ptr } %3684, 1
// CHECK-NEXT:   %3688 = extractvalue { ptr, ptr } %3684, 0
// CHECK-NEXT:   %__llgo_funcval_code487 = call ptr asm "", "=r,0"(ptr %3688)
// CHECK-NEXT:   %3689 = call %reflect.Value %__llgo_funcval_code487(ptr {{(nest|swiftself)}} %3687, %"{{.*}}/runtime/internal/runtime.eface" %3686)
// CHECK-NEXT:   store %reflect.Value %3682, ptr %3675, align 8
// CHECK-NEXT:   store %reflect.Value %3689, ptr %3683, align 8
// CHECK-NEXT:   %3690 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 244
// CHECK-NEXT:   %3691 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3690, i32 0, i32 0
// CHECK-NEXT:   %3692 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3693 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3694 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3693, ptr %3694, align 8
// CHECK-NEXT:   %3695 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3694, 1
// CHECK-NEXT:   %3696 = extractvalue { ptr, ptr } %3692, 1
// CHECK-NEXT:   %3697 = extractvalue { ptr, ptr } %3692, 0
// CHECK-NEXT:   %__llgo_funcval_code488 = call ptr asm "", "=r,0"(ptr %3697)
// CHECK-NEXT:   %3698 = call %reflect.Value %__llgo_funcval_code488(ptr {{(nest|swiftself)}} %3696, %"{{.*}}/runtime/internal/runtime.eface" %3695)
// CHECK-NEXT:   %3699 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3690, i32 0, i32 1
// CHECK-NEXT:   %3700 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3701 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 17 })
// CHECK-NEXT:   %3702 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3701, ptr %3702, align 8
// CHECK-NEXT:   %3703 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3702, 1
// CHECK-NEXT:   %3704 = extractvalue { ptr, ptr } %3700, 1
// CHECK-NEXT:   %3705 = extractvalue { ptr, ptr } %3700, 0
// CHECK-NEXT:   %__llgo_funcval_code489 = call ptr asm "", "=r,0"(ptr %3705)
// CHECK-NEXT:   %3706 = call %reflect.Value %__llgo_funcval_code489(ptr {{(nest|swiftself)}} %3704, %"{{.*}}/runtime/internal/runtime.eface" %3703)
// CHECK-NEXT:   store %reflect.Value %3698, ptr %3691, align 8
// CHECK-NEXT:   store %reflect.Value %3706, ptr %3699, align 8
// CHECK-NEXT:   %3707 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 245
// CHECK-NEXT:   %3708 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3707, i32 0, i32 0
// CHECK-NEXT:   %3709 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3710 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3710, align 8
// CHECK-NEXT:   %3711 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3710, 1
// CHECK-NEXT:   %3712 = extractvalue { ptr, ptr } %3709, 1
// CHECK-NEXT:   %3713 = extractvalue { ptr, ptr } %3709, 0
// CHECK-NEXT:   %__llgo_funcval_code490 = call ptr asm "", "=r,0"(ptr %3713)
// CHECK-NEXT:   %3714 = call %reflect.Value %__llgo_funcval_code490(ptr {{(nest|swiftself)}} %3712, %"{{.*}}/runtime/internal/runtime.eface" %3711)
// CHECK-NEXT:   %3715 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3707, i32 0, i32 1
// CHECK-NEXT:   %3716 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3717 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3718 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3717, ptr %3718, align 8
// CHECK-NEXT:   %3719 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3718, 1
// CHECK-NEXT:   %3720 = extractvalue { ptr, ptr } %3716, 1
// CHECK-NEXT:   %3721 = extractvalue { ptr, ptr } %3716, 0
// CHECK-NEXT:   %__llgo_funcval_code491 = call ptr asm "", "=r,0"(ptr %3721)
// CHECK-NEXT:   %3722 = call %reflect.Value %__llgo_funcval_code491(ptr {{(nest|swiftself)}} %3720, %"{{.*}}/runtime/internal/runtime.eface" %3719)
// CHECK-NEXT:   store %reflect.Value %3714, ptr %3708, align 8
// CHECK-NEXT:   store %reflect.Value %3722, ptr %3715, align 8
// CHECK-NEXT:   %3723 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 246
// CHECK-NEXT:   %3724 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3723, i32 0, i32 0
// CHECK-NEXT:   %3725 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3726 = call %"{{.*}}/runtime/internal/runtime.Slice" @"{{.*}}/runtime/internal/runtime.StringToRunes"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 })
// CHECK-NEXT:   %3727 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3726, ptr %3727, align 8
// CHECK-NEXT:   %3728 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyRunes, ptr undef }, ptr %3727, 1
// CHECK-NEXT:   %3729 = extractvalue { ptr, ptr } %3725, 1
// CHECK-NEXT:   %3730 = extractvalue { ptr, ptr } %3725, 0
// CHECK-NEXT:   %__llgo_funcval_code492 = call ptr asm "", "=r,0"(ptr %3730)
// CHECK-NEXT:   %3731 = call %reflect.Value %__llgo_funcval_code492(ptr {{(nest|swiftself)}} %3729, %"{{.*}}/runtime/internal/runtime.eface" %3728)
// CHECK-NEXT:   %3732 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3723, i32 0, i32 1
// CHECK-NEXT:   %3733 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3734 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 8 }, ptr %3734, align 8
// CHECK-NEXT:   %3735 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyString, ptr undef }, ptr %3734, 1
// CHECK-NEXT:   %3736 = extractvalue { ptr, ptr } %3733, 1
// CHECK-NEXT:   %3737 = extractvalue { ptr, ptr } %3733, 0
// CHECK-NEXT:   %__llgo_funcval_code493 = call ptr asm "", "=r,0"(ptr %3737)
// CHECK-NEXT:   %3738 = call %reflect.Value %__llgo_funcval_code493(ptr {{(nest|swiftself)}} %3736, %"{{.*}}/runtime/internal/runtime.eface" %3735)
// CHECK-NEXT:   store %reflect.Value %3731, ptr %3724, align 8
// CHECK-NEXT:   store %reflect.Value %3738, ptr %3732, align 8
// CHECK-NEXT:   %3739 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 247
// CHECK-NEXT:   %3740 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3739, i32 0, i32 0
// CHECK-NEXT:   %3741 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3742 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %3742, align 8
// CHECK-NEXT:   %3743 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3742, 1
// CHECK-NEXT:   %3744 = extractvalue { ptr, ptr } %3741, 1
// CHECK-NEXT:   %3745 = extractvalue { ptr, ptr } %3741, 0
// CHECK-NEXT:   %__llgo_funcval_code494 = call ptr asm "", "=r,0"(ptr %3745)
// CHECK-NEXT:   %3746 = call %reflect.Value %__llgo_funcval_code494(ptr {{(nest|swiftself)}} %3744, %"{{.*}}/runtime/internal/runtime.eface" %3743)
// CHECK-NEXT:   %3747 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3739, i32 0, i32 1
// CHECK-NEXT:   %3748 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3749 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3749, align 1
// CHECK-NEXT:   %3750 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %3749, 1
// CHECK-NEXT:   %3751 = extractvalue { ptr, ptr } %3748, 1
// CHECK-NEXT:   %3752 = extractvalue { ptr, ptr } %3748, 0
// CHECK-NEXT:   %__llgo_funcval_code495 = call ptr asm "", "=r,0"(ptr %3752)
// CHECK-NEXT:   %3753 = call %reflect.Value %__llgo_funcval_code495(ptr {{(nest|swiftself)}} %3751, %"{{.*}}/runtime/internal/runtime.eface" %3750)
// CHECK-NEXT:   store %reflect.Value %3746, ptr %3740, align 8
// CHECK-NEXT:   store %reflect.Value %3753, ptr %3747, align 8
// CHECK-NEXT:   %3754 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 248
// CHECK-NEXT:   %3755 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3754, i32 0, i32 0
// CHECK-NEXT:   %3756 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3757 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %3757, align 8
// CHECK-NEXT:   %3758 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3757, 1
// CHECK-NEXT:   %3759 = extractvalue { ptr, ptr } %3756, 1
// CHECK-NEXT:   %3760 = extractvalue { ptr, ptr } %3756, 0
// CHECK-NEXT:   %__llgo_funcval_code496 = call ptr asm "", "=r,0"(ptr %3760)
// CHECK-NEXT:   %3761 = call %reflect.Value %__llgo_funcval_code496(ptr {{(nest|swiftself)}} %3759, %"{{.*}}/runtime/internal/runtime.eface" %3758)
// CHECK-NEXT:   %3762 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3754, i32 0, i32 1
// CHECK-NEXT:   %3763 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3764 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3764, align 1
// CHECK-NEXT:   %3765 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %3764, 1
// CHECK-NEXT:   %3766 = extractvalue { ptr, ptr } %3763, 1
// CHECK-NEXT:   %3767 = extractvalue { ptr, ptr } %3763, 0
// CHECK-NEXT:   %__llgo_funcval_code497 = call ptr asm "", "=r,0"(ptr %3767)
// CHECK-NEXT:   %3768 = call %reflect.Value %__llgo_funcval_code497(ptr {{(nest|swiftself)}} %3766, %"{{.*}}/runtime/internal/runtime.eface" %3765)
// CHECK-NEXT:   store %reflect.Value %3761, ptr %3755, align 8
// CHECK-NEXT:   store %reflect.Value %3768, ptr %3762, align 8
// CHECK-NEXT:   %3769 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 249
// CHECK-NEXT:   %3770 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3769, i32 0, i32 0
// CHECK-NEXT:   %3771 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3772 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %3773 = getelementptr inbounds i8, ptr %3772, i64 0
// CHECK-NEXT:   store i8 1, ptr %3773, align 1
// CHECK-NEXT:   %3774 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3772, 0
// CHECK-NEXT:   %3775 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3774, i64 1, 1
// CHECK-NEXT:   %3776 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3775, i64 1, 2
// CHECK-NEXT:   %3777 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3776, ptr %3777, align 8
// CHECK-NEXT:   %3778 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3777, 1
// CHECK-NEXT:   %3779 = extractvalue { ptr, ptr } %3771, 1
// CHECK-NEXT:   %3780 = extractvalue { ptr, ptr } %3771, 0
// CHECK-NEXT:   %__llgo_funcval_code498 = call ptr asm "", "=r,0"(ptr %3780)
// CHECK-NEXT:   %3781 = call %reflect.Value %__llgo_funcval_code498(ptr {{(nest|swiftself)}} %3779, %"{{.*}}/runtime/internal/runtime.eface" %3778)
// CHECK-NEXT:   %3782 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3769, i32 0, i32 1
// CHECK-NEXT:   %3783 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3784 = alloca [1 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3784, i8 0, i64 1, i1 false)
// CHECK-NEXT:   %3785 = getelementptr inbounds i8, ptr %3784, i64 0
// CHECK-NEXT:   store i8 1, ptr %3785, align 1
// CHECK-NEXT:   %3786 = load [1 x i8], ptr %3784, align 1
// CHECK-NEXT:   %3787 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store [1 x i8] %3786, ptr %3787, align 1
// CHECK-NEXT:   %3788 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[1]_llgo_uint8", ptr undef }, ptr %3787, 1
// CHECK-NEXT:   %3789 = extractvalue { ptr, ptr } %3783, 1
// CHECK-NEXT:   %3790 = extractvalue { ptr, ptr } %3783, 0
// CHECK-NEXT:   %__llgo_funcval_code499 = call ptr asm "", "=r,0"(ptr %3790)
// CHECK-NEXT:   %3791 = call %reflect.Value %__llgo_funcval_code499(ptr {{(nest|swiftself)}} %3789, %"{{.*}}/runtime/internal/runtime.eface" %3788)
// CHECK-NEXT:   store %reflect.Value %3781, ptr %3770, align 8
// CHECK-NEXT:   store %reflect.Value %3791, ptr %3782, align 8
// CHECK-NEXT:   %3792 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 250
// CHECK-NEXT:   %3793 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3792, i32 0, i32 0
// CHECK-NEXT:   %3794 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3795 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %3796 = getelementptr inbounds i8, ptr %3795, i64 0
// CHECK-NEXT:   store i8 1, ptr %3796, align 1
// CHECK-NEXT:   %3797 = getelementptr inbounds i8, ptr %3795, i64 1
// CHECK-NEXT:   store i8 2, ptr %3797, align 1
// CHECK-NEXT:   %3798 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3795, 0
// CHECK-NEXT:   %3799 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3798, i64 2, 1
// CHECK-NEXT:   %3800 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3799, i64 2, 2
// CHECK-NEXT:   %3801 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3800, ptr %3801, align 8
// CHECK-NEXT:   %3802 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3801, 1
// CHECK-NEXT:   %3803 = extractvalue { ptr, ptr } %3794, 1
// CHECK-NEXT:   %3804 = extractvalue { ptr, ptr } %3794, 0
// CHECK-NEXT:   %__llgo_funcval_code500 = call ptr asm "", "=r,0"(ptr %3804)
// CHECK-NEXT:   %3805 = call %reflect.Value %__llgo_funcval_code500(ptr {{(nest|swiftself)}} %3803, %"{{.*}}/runtime/internal/runtime.eface" %3802)
// CHECK-NEXT:   %3806 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3792, i32 0, i32 1
// CHECK-NEXT:   %3807 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3808 = alloca [2 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3808, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %3809 = getelementptr inbounds i8, ptr %3808, i64 0
// CHECK-NEXT:   %3810 = getelementptr inbounds i8, ptr %3808, i64 1
// CHECK-NEXT:   store i8 1, ptr %3809, align 1
// CHECK-NEXT:   store i8 2, ptr %3810, align 1
// CHECK-NEXT:   %3811 = load [2 x i8], ptr %3808, align 1
// CHECK-NEXT:   %3812 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] %3811, ptr %3812, align 1
// CHECK-NEXT:   %3813 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %3812, 1
// CHECK-NEXT:   %3814 = extractvalue { ptr, ptr } %3807, 1
// CHECK-NEXT:   %3815 = extractvalue { ptr, ptr } %3807, 0
// CHECK-NEXT:   %__llgo_funcval_code501 = call ptr asm "", "=r,0"(ptr %3815)
// CHECK-NEXT:   %3816 = call %reflect.Value %__llgo_funcval_code501(ptr {{(nest|swiftself)}} %3814, %"{{.*}}/runtime/internal/runtime.eface" %3813)
// CHECK-NEXT:   store %reflect.Value %3805, ptr %3793, align 8
// CHECK-NEXT:   store %reflect.Value %3816, ptr %3806, align 8
// CHECK-NEXT:   %3817 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 251
// CHECK-NEXT:   %3818 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3817, i32 0, i32 0
// CHECK-NEXT:   %3819 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3820 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 3)
// CHECK-NEXT:   %3821 = getelementptr inbounds i8, ptr %3820, i64 0
// CHECK-NEXT:   store i8 1, ptr %3821, align 1
// CHECK-NEXT:   %3822 = getelementptr inbounds i8, ptr %3820, i64 1
// CHECK-NEXT:   store i8 2, ptr %3822, align 1
// CHECK-NEXT:   %3823 = getelementptr inbounds i8, ptr %3820, i64 2
// CHECK-NEXT:   store i8 3, ptr %3823, align 1
// CHECK-NEXT:   %3824 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3820, 0
// CHECK-NEXT:   %3825 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3824, i64 3, 1
// CHECK-NEXT:   %3826 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3825, i64 3, 2
// CHECK-NEXT:   %3827 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3826, ptr %3827, align 8
// CHECK-NEXT:   %3828 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3827, 1
// CHECK-NEXT:   %3829 = extractvalue { ptr, ptr } %3819, 1
// CHECK-NEXT:   %3830 = extractvalue { ptr, ptr } %3819, 0
// CHECK-NEXT:   %__llgo_funcval_code502 = call ptr asm "", "=r,0"(ptr %3830)
// CHECK-NEXT:   %3831 = call %reflect.Value %__llgo_funcval_code502(ptr {{(nest|swiftself)}} %3829, %"{{.*}}/runtime/internal/runtime.eface" %3828)
// CHECK-NEXT:   %3832 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3817, i32 0, i32 1
// CHECK-NEXT:   %3833 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3834 = alloca [3 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3834, i8 0, i64 3, i1 false)
// CHECK-NEXT:   %3835 = getelementptr inbounds i8, ptr %3834, i64 0
// CHECK-NEXT:   %3836 = getelementptr inbounds i8, ptr %3834, i64 1
// CHECK-NEXT:   %3837 = getelementptr inbounds i8, ptr %3834, i64 2
// CHECK-NEXT:   store i8 1, ptr %3835, align 1
// CHECK-NEXT:   store i8 2, ptr %3836, align 1
// CHECK-NEXT:   store i8 3, ptr %3837, align 1
// CHECK-NEXT:   %3838 = load [3 x i8], ptr %3834, align 1
// CHECK-NEXT:   %3839 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 3)
// CHECK-NEXT:   store [3 x i8] %3838, ptr %3839, align 1
// CHECK-NEXT:   %3840 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[3]_llgo_uint8", ptr undef }, ptr %3839, 1
// CHECK-NEXT:   %3841 = extractvalue { ptr, ptr } %3833, 1
// CHECK-NEXT:   %3842 = extractvalue { ptr, ptr } %3833, 0
// CHECK-NEXT:   %__llgo_funcval_code503 = call ptr asm "", "=r,0"(ptr %3842)
// CHECK-NEXT:   %3843 = call %reflect.Value %__llgo_funcval_code503(ptr {{(nest|swiftself)}} %3841, %"{{.*}}/runtime/internal/runtime.eface" %3840)
// CHECK-NEXT:   store %reflect.Value %3831, ptr %3818, align 8
// CHECK-NEXT:   store %reflect.Value %3843, ptr %3832, align 8
// CHECK-NEXT:   %3844 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 252
// CHECK-NEXT:   %3845 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3844, i32 0, i32 0
// CHECK-NEXT:   %3846 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3847 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %3847, align 8
// CHECK-NEXT:   %3848 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3847, 1
// CHECK-NEXT:   %3849 = extractvalue { ptr, ptr } %3846, 1
// CHECK-NEXT:   %3850 = extractvalue { ptr, ptr } %3846, 0
// CHECK-NEXT:   %__llgo_funcval_code504 = call ptr asm "", "=r,0"(ptr %3850)
// CHECK-NEXT:   %3851 = call %reflect.Value %__llgo_funcval_code504(ptr {{(nest|swiftself)}} %3849, %"{{.*}}/runtime/internal/runtime.eface" %3848)
// CHECK-NEXT:   %3852 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3844, i32 0, i32 1
// CHECK-NEXT:   %3853 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3854 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3854, align 1
// CHECK-NEXT:   %3855 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %3854, 1
// CHECK-NEXT:   %3856 = extractvalue { ptr, ptr } %3853, 1
// CHECK-NEXT:   %3857 = extractvalue { ptr, ptr } %3853, 0
// CHECK-NEXT:   %__llgo_funcval_code505 = call ptr asm "", "=r,0"(ptr %3857)
// CHECK-NEXT:   %3858 = call %reflect.Value %__llgo_funcval_code505(ptr {{(nest|swiftself)}} %3856, %"{{.*}}/runtime/internal/runtime.eface" %3855)
// CHECK-NEXT:   store %reflect.Value %3851, ptr %3845, align 8
// CHECK-NEXT:   store %reflect.Value %3858, ptr %3852, align 8
// CHECK-NEXT:   %3859 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 253
// CHECK-NEXT:   %3860 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3859, i32 0, i32 0
// CHECK-NEXT:   %3861 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3862 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %3862, align 8
// CHECK-NEXT:   %3863 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3862, 1
// CHECK-NEXT:   %3864 = extractvalue { ptr, ptr } %3861, 1
// CHECK-NEXT:   %3865 = extractvalue { ptr, ptr } %3861, 0
// CHECK-NEXT:   %__llgo_funcval_code506 = call ptr asm "", "=r,0"(ptr %3865)
// CHECK-NEXT:   %3866 = call %reflect.Value %__llgo_funcval_code506(ptr {{(nest|swiftself)}} %3864, %"{{.*}}/runtime/internal/runtime.eface" %3863)
// CHECK-NEXT:   %3867 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3859, i32 0, i32 1
// CHECK-NEXT:   %3868 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3869 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3869, align 1
// CHECK-NEXT:   %3870 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %3869, 1
// CHECK-NEXT:   %3871 = extractvalue { ptr, ptr } %3868, 1
// CHECK-NEXT:   %3872 = extractvalue { ptr, ptr } %3868, 0
// CHECK-NEXT:   %__llgo_funcval_code507 = call ptr asm "", "=r,0"(ptr %3872)
// CHECK-NEXT:   %3873 = call %reflect.Value %__llgo_funcval_code507(ptr {{(nest|swiftself)}} %3871, %"{{.*}}/runtime/internal/runtime.eface" %3870)
// CHECK-NEXT:   store %reflect.Value %3866, ptr %3860, align 8
// CHECK-NEXT:   store %reflect.Value %3873, ptr %3867, align 8
// CHECK-NEXT:   %3874 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 254
// CHECK-NEXT:   %3875 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3874, i32 0, i32 0
// CHECK-NEXT:   %3876 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3877 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %3878 = getelementptr inbounds i8, ptr %3877, i64 0
// CHECK-NEXT:   store i8 1, ptr %3878, align 1
// CHECK-NEXT:   %3879 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3877, 0
// CHECK-NEXT:   %3880 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3879, i64 1, 1
// CHECK-NEXT:   %3881 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3880, i64 1, 2
// CHECK-NEXT:   %3882 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3881, ptr %3882, align 8
// CHECK-NEXT:   %3883 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3882, 1
// CHECK-NEXT:   %3884 = extractvalue { ptr, ptr } %3876, 1
// CHECK-NEXT:   %3885 = extractvalue { ptr, ptr } %3876, 0
// CHECK-NEXT:   %__llgo_funcval_code508 = call ptr asm "", "=r,0"(ptr %3885)
// CHECK-NEXT:   %3886 = call %reflect.Value %__llgo_funcval_code508(ptr {{(nest|swiftself)}} %3884, %"{{.*}}/runtime/internal/runtime.eface" %3883)
// CHECK-NEXT:   %3887 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3874, i32 0, i32 1
// CHECK-NEXT:   %3888 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3889 = alloca [1 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3889, i8 0, i64 1, i1 false)
// CHECK-NEXT:   %3890 = getelementptr inbounds i8, ptr %3889, i64 0
// CHECK-NEXT:   store i8 1, ptr %3890, align 1
// CHECK-NEXT:   %3891 = load [1 x i8], ptr %3889, align 1
// CHECK-NEXT:   %3892 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 1)
// CHECK-NEXT:   store [1 x i8] %3891, ptr %3892, align 1
// CHECK-NEXT:   %3893 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[1]_llgo_uint8", ptr undef }, ptr %3892, 1
// CHECK-NEXT:   %3894 = extractvalue { ptr, ptr } %3888, 1
// CHECK-NEXT:   %3895 = extractvalue { ptr, ptr } %3888, 0
// CHECK-NEXT:   %__llgo_funcval_code509 = call ptr asm "", "=r,0"(ptr %3895)
// CHECK-NEXT:   %3896 = call %reflect.Value %__llgo_funcval_code509(ptr {{(nest|swiftself)}} %3894, %"{{.*}}/runtime/internal/runtime.eface" %3893)
// CHECK-NEXT:   store %reflect.Value %3886, ptr %3875, align 8
// CHECK-NEXT:   store %reflect.Value %3896, ptr %3887, align 8
// CHECK-NEXT:   %3897 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 255
// CHECK-NEXT:   %3898 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3897, i32 0, i32 0
// CHECK-NEXT:   %3899 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3900 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %3901 = getelementptr inbounds i8, ptr %3900, i64 0
// CHECK-NEXT:   store i8 1, ptr %3901, align 1
// CHECK-NEXT:   %3902 = getelementptr inbounds i8, ptr %3900, i64 1
// CHECK-NEXT:   store i8 2, ptr %3902, align 1
// CHECK-NEXT:   %3903 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3900, 0
// CHECK-NEXT:   %3904 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3903, i64 2, 1
// CHECK-NEXT:   %3905 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3904, i64 2, 2
// CHECK-NEXT:   %3906 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3905, ptr %3906, align 8
// CHECK-NEXT:   %3907 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3906, 1
// CHECK-NEXT:   %3908 = extractvalue { ptr, ptr } %3899, 1
// CHECK-NEXT:   %3909 = extractvalue { ptr, ptr } %3899, 0
// CHECK-NEXT:   %__llgo_funcval_code510 = call ptr asm "", "=r,0"(ptr %3909)
// CHECK-NEXT:   %3910 = call %reflect.Value %__llgo_funcval_code510(ptr {{(nest|swiftself)}} %3908, %"{{.*}}/runtime/internal/runtime.eface" %3907)
// CHECK-NEXT:   %3911 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3897, i32 0, i32 1
// CHECK-NEXT:   %3912 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3913 = alloca [2 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3913, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %3914 = getelementptr inbounds i8, ptr %3913, i64 0
// CHECK-NEXT:   %3915 = getelementptr inbounds i8, ptr %3913, i64 1
// CHECK-NEXT:   store i8 1, ptr %3914, align 1
// CHECK-NEXT:   store i8 2, ptr %3915, align 1
// CHECK-NEXT:   %3916 = load [2 x i8], ptr %3913, align 1
// CHECK-NEXT:   %3917 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] %3916, ptr %3917, align 1
// CHECK-NEXT:   %3918 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %3917, 1
// CHECK-NEXT:   %3919 = extractvalue { ptr, ptr } %3912, 1
// CHECK-NEXT:   %3920 = extractvalue { ptr, ptr } %3912, 0
// CHECK-NEXT:   %__llgo_funcval_code511 = call ptr asm "", "=r,0"(ptr %3920)
// CHECK-NEXT:   %3921 = call %reflect.Value %__llgo_funcval_code511(ptr {{(nest|swiftself)}} %3919, %"{{.*}}/runtime/internal/runtime.eface" %3918)
// CHECK-NEXT:   store %reflect.Value %3910, ptr %3898, align 8
// CHECK-NEXT:   store %reflect.Value %3921, ptr %3911, align 8
// CHECK-NEXT:   %3922 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 256
// CHECK-NEXT:   %3923 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3922, i32 0, i32 0
// CHECK-NEXT:   %3924 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3925 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 3)
// CHECK-NEXT:   %3926 = getelementptr inbounds i8, ptr %3925, i64 0
// CHECK-NEXT:   store i8 1, ptr %3926, align 1
// CHECK-NEXT:   %3927 = getelementptr inbounds i8, ptr %3925, i64 1
// CHECK-NEXT:   store i8 2, ptr %3927, align 1
// CHECK-NEXT:   %3928 = getelementptr inbounds i8, ptr %3925, i64 2
// CHECK-NEXT:   store i8 3, ptr %3928, align 1
// CHECK-NEXT:   %3929 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3925, 0
// CHECK-NEXT:   %3930 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3929, i64 3, 1
// CHECK-NEXT:   %3931 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3930, i64 3, 2
// CHECK-NEXT:   %3932 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3931, ptr %3932, align 8
// CHECK-NEXT:   %3933 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %3932, 1
// CHECK-NEXT:   %3934 = extractvalue { ptr, ptr } %3924, 1
// CHECK-NEXT:   %3935 = extractvalue { ptr, ptr } %3924, 0
// CHECK-NEXT:   %__llgo_funcval_code512 = call ptr asm "", "=r,0"(ptr %3935)
// CHECK-NEXT:   %3936 = call %reflect.Value %__llgo_funcval_code512(ptr {{(nest|swiftself)}} %3934, %"{{.*}}/runtime/internal/runtime.eface" %3933)
// CHECK-NEXT:   %3937 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3922, i32 0, i32 1
// CHECK-NEXT:   %3938 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3939 = alloca [3 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3939, i8 0, i64 3, i1 false)
// CHECK-NEXT:   %3940 = getelementptr inbounds i8, ptr %3939, i64 0
// CHECK-NEXT:   %3941 = getelementptr inbounds i8, ptr %3939, i64 1
// CHECK-NEXT:   %3942 = getelementptr inbounds i8, ptr %3939, i64 2
// CHECK-NEXT:   store i8 1, ptr %3940, align 1
// CHECK-NEXT:   store i8 2, ptr %3941, align 1
// CHECK-NEXT:   store i8 3, ptr %3942, align 1
// CHECK-NEXT:   %3943 = load [3 x i8], ptr %3939, align 1
// CHECK-NEXT:   %3944 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 3)
// CHECK-NEXT:   store [3 x i8] %3943, ptr %3944, align 1
// CHECK-NEXT:   %3945 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[3]_llgo_uint8", ptr undef }, ptr %3944, 1
// CHECK-NEXT:   %3946 = extractvalue { ptr, ptr } %3938, 1
// CHECK-NEXT:   %3947 = extractvalue { ptr, ptr } %3938, 0
// CHECK-NEXT:   %__llgo_funcval_code513 = call ptr asm "", "=r,0"(ptr %3947)
// CHECK-NEXT:   %3948 = call %reflect.Value %__llgo_funcval_code513(ptr {{(nest|swiftself)}} %3946, %"{{.*}}/runtime/internal/runtime.eface" %3945)
// CHECK-NEXT:   store %reflect.Value %3936, ptr %3923, align 8
// CHECK-NEXT:   store %reflect.Value %3948, ptr %3937, align 8
// CHECK-NEXT:   %3949 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 257
// CHECK-NEXT:   %3950 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3949, i32 0, i32 0
// CHECK-NEXT:   %3951 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3952 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %3952, align 8
// CHECK-NEXT:   %3953 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3952, 1
// CHECK-NEXT:   %3954 = extractvalue { ptr, ptr } %3951, 1
// CHECK-NEXT:   %3955 = extractvalue { ptr, ptr } %3951, 0
// CHECK-NEXT:   %__llgo_funcval_code514 = call ptr asm "", "=r,0"(ptr %3955)
// CHECK-NEXT:   %3956 = call %reflect.Value %__llgo_funcval_code514(ptr {{(nest|swiftself)}} %3954, %"{{.*}}/runtime/internal/runtime.eface" %3953)
// CHECK-NEXT:   %3957 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3949, i32 0, i32 1
// CHECK-NEXT:   %3958 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3959 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3959, align 1
// CHECK-NEXT:   %3960 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray0, ptr undef }, ptr %3959, 1
// CHECK-NEXT:   %3961 = extractvalue { ptr, ptr } %3958, 1
// CHECK-NEXT:   %3962 = extractvalue { ptr, ptr } %3958, 0
// CHECK-NEXT:   %__llgo_funcval_code515 = call ptr asm "", "=r,0"(ptr %3962)
// CHECK-NEXT:   %3963 = call %reflect.Value %__llgo_funcval_code515(ptr {{(nest|swiftself)}} %3961, %"{{.*}}/runtime/internal/runtime.eface" %3960)
// CHECK-NEXT:   store %reflect.Value %3956, ptr %3950, align 8
// CHECK-NEXT:   store %reflect.Value %3963, ptr %3957, align 8
// CHECK-NEXT:   %3964 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 258
// CHECK-NEXT:   %3965 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3964, i32 0, i32 0
// CHECK-NEXT:   %3966 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3967 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %3967, align 8
// CHECK-NEXT:   %3968 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3967, 1
// CHECK-NEXT:   %3969 = extractvalue { ptr, ptr } %3966, 1
// CHECK-NEXT:   %3970 = extractvalue { ptr, ptr } %3966, 0
// CHECK-NEXT:   %__llgo_funcval_code516 = call ptr asm "", "=r,0"(ptr %3970)
// CHECK-NEXT:   %3971 = call %reflect.Value %__llgo_funcval_code516(ptr {{(nest|swiftself)}} %3969, %"{{.*}}/runtime/internal/runtime.eface" %3968)
// CHECK-NEXT:   %3972 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3964, i32 0, i32 1
// CHECK-NEXT:   %3973 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3974 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %3974, align 1
// CHECK-NEXT:   %3975 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray0, ptr undef }, ptr %3974, 1
// CHECK-NEXT:   %3976 = extractvalue { ptr, ptr } %3973, 1
// CHECK-NEXT:   %3977 = extractvalue { ptr, ptr } %3973, 0
// CHECK-NEXT:   %__llgo_funcval_code517 = call ptr asm "", "=r,0"(ptr %3977)
// CHECK-NEXT:   %3978 = call %reflect.Value %__llgo_funcval_code517(ptr {{(nest|swiftself)}} %3976, %"{{.*}}/runtime/internal/runtime.eface" %3975)
// CHECK-NEXT:   store %reflect.Value %3971, ptr %3965, align 8
// CHECK-NEXT:   store %reflect.Value %3978, ptr %3972, align 8
// CHECK-NEXT:   %3979 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 259
// CHECK-NEXT:   %3980 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3979, i32 0, i32 0
// CHECK-NEXT:   %3981 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3982 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %3983 = getelementptr inbounds i8, ptr %3982, i64 0
// CHECK-NEXT:   store i8 1, ptr %3983, align 1
// CHECK-NEXT:   %3984 = getelementptr inbounds i8, ptr %3982, i64 1
// CHECK-NEXT:   store i8 2, ptr %3984, align 1
// CHECK-NEXT:   %3985 = getelementptr inbounds i8, ptr %3982, i64 2
// CHECK-NEXT:   store i8 3, ptr %3985, align 1
// CHECK-NEXT:   %3986 = getelementptr inbounds i8, ptr %3982, i64 3
// CHECK-NEXT:   store i8 4, ptr %3986, align 1
// CHECK-NEXT:   %3987 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %3982, 0
// CHECK-NEXT:   %3988 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3987, i64 4, 1
// CHECK-NEXT:   %3989 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %3988, i64 4, 2
// CHECK-NEXT:   %3990 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %3989, ptr %3990, align 8
// CHECK-NEXT:   %3991 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %3990, 1
// CHECK-NEXT:   %3992 = extractvalue { ptr, ptr } %3981, 1
// CHECK-NEXT:   %3993 = extractvalue { ptr, ptr } %3981, 0
// CHECK-NEXT:   %__llgo_funcval_code518 = call ptr asm "", "=r,0"(ptr %3993)
// CHECK-NEXT:   %3994 = call %reflect.Value %__llgo_funcval_code518(ptr {{(nest|swiftself)}} %3992, %"{{.*}}/runtime/internal/runtime.eface" %3991)
// CHECK-NEXT:   %3995 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %3979, i32 0, i32 1
// CHECK-NEXT:   %3996 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %3997 = alloca [4 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %3997, i8 0, i64 4, i1 false)
// CHECK-NEXT:   %3998 = getelementptr inbounds i8, ptr %3997, i64 0
// CHECK-NEXT:   %3999 = getelementptr inbounds i8, ptr %3997, i64 1
// CHECK-NEXT:   %4000 = getelementptr inbounds i8, ptr %3997, i64 2
// CHECK-NEXT:   %4001 = getelementptr inbounds i8, ptr %3997, i64 3
// CHECK-NEXT:   store i8 1, ptr %3998, align 1
// CHECK-NEXT:   store i8 2, ptr %3999, align 1
// CHECK-NEXT:   store i8 3, ptr %4000, align 1
// CHECK-NEXT:   store i8 4, ptr %4001, align 1
// CHECK-NEXT:   %4002 = load [4 x i8], ptr %3997, align 1
// CHECK-NEXT:   %4003 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store [4 x i8] %4002, ptr %4003, align 1
// CHECK-NEXT:   %4004 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray, ptr undef }, ptr %4003, 1
// CHECK-NEXT:   %4005 = extractvalue { ptr, ptr } %3996, 1
// CHECK-NEXT:   %4006 = extractvalue { ptr, ptr } %3996, 0
// CHECK-NEXT:   %__llgo_funcval_code519 = call ptr asm "", "=r,0"(ptr %4006)
// CHECK-NEXT:   %4007 = call %reflect.Value %__llgo_funcval_code519(ptr {{(nest|swiftself)}} %4005, %"{{.*}}/runtime/internal/runtime.eface" %4004)
// CHECK-NEXT:   store %reflect.Value %3994, ptr %3980, align 8
// CHECK-NEXT:   store %reflect.Value %4007, ptr %3995, align 8
// CHECK-NEXT:   %4008 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 260
// CHECK-NEXT:   %4009 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4008, i32 0, i32 0
// CHECK-NEXT:   %4010 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4011 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4011, align 8
// CHECK-NEXT:   %4012 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4011, 1
// CHECK-NEXT:   %4013 = extractvalue { ptr, ptr } %4010, 1
// CHECK-NEXT:   %4014 = extractvalue { ptr, ptr } %4010, 0
// CHECK-NEXT:   %__llgo_funcval_code520 = call ptr asm "", "=r,0"(ptr %4014)
// CHECK-NEXT:   %4015 = call %reflect.Value %__llgo_funcval_code520(ptr {{(nest|swiftself)}} %4013, %"{{.*}}/runtime/internal/runtime.eface" %4012)
// CHECK-NEXT:   %4016 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4008, i32 0, i32 1
// CHECK-NEXT:   %4017 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4018 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4018, align 1
// CHECK-NEXT:   %4019 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray0, ptr undef }, ptr %4018, 1
// CHECK-NEXT:   %4020 = extractvalue { ptr, ptr } %4017, 1
// CHECK-NEXT:   %4021 = extractvalue { ptr, ptr } %4017, 0
// CHECK-NEXT:   %__llgo_funcval_code521 = call ptr asm "", "=r,0"(ptr %4021)
// CHECK-NEXT:   %4022 = call %reflect.Value %__llgo_funcval_code521(ptr {{(nest|swiftself)}} %4020, %"{{.*}}/runtime/internal/runtime.eface" %4019)
// CHECK-NEXT:   store %reflect.Value %4015, ptr %4009, align 8
// CHECK-NEXT:   store %reflect.Value %4022, ptr %4016, align 8
// CHECK-NEXT:   %4023 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 261
// CHECK-NEXT:   %4024 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4023, i32 0, i32 0
// CHECK-NEXT:   %4025 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4026 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4027 = getelementptr inbounds i8, ptr %4026, i64 0
// CHECK-NEXT:   store i8 5, ptr %4027, align 1
// CHECK-NEXT:   %4028 = getelementptr inbounds i8, ptr %4026, i64 1
// CHECK-NEXT:   store i8 6, ptr %4028, align 1
// CHECK-NEXT:   %4029 = getelementptr inbounds i8, ptr %4026, i64 2
// CHECK-NEXT:   store i8 7, ptr %4029, align 1
// CHECK-NEXT:   %4030 = getelementptr inbounds i8, ptr %4026, i64 3
// CHECK-NEXT:   store i8 8, ptr %4030, align 1
// CHECK-NEXT:   %4031 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4026, 0
// CHECK-NEXT:   %4032 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4031, i64 4, 1
// CHECK-NEXT:   %4033 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4032, i64 4, 2
// CHECK-NEXT:   %4034 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4033, ptr %4034, align 8
// CHECK-NEXT:   %4035 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4034, 1
// CHECK-NEXT:   %4036 = extractvalue { ptr, ptr } %4025, 1
// CHECK-NEXT:   %4037 = extractvalue { ptr, ptr } %4025, 0
// CHECK-NEXT:   %__llgo_funcval_code522 = call ptr asm "", "=r,0"(ptr %4037)
// CHECK-NEXT:   %4038 = call %reflect.Value %__llgo_funcval_code522(ptr {{(nest|swiftself)}} %4036, %"{{.*}}/runtime/internal/runtime.eface" %4035)
// CHECK-NEXT:   %4039 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4023, i32 0, i32 1
// CHECK-NEXT:   %4040 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4041 = alloca [4 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %4041, i8 0, i64 4, i1 false)
// CHECK-NEXT:   %4042 = getelementptr inbounds i8, ptr %4041, i64 0
// CHECK-NEXT:   %4043 = getelementptr inbounds i8, ptr %4041, i64 1
// CHECK-NEXT:   %4044 = getelementptr inbounds i8, ptr %4041, i64 2
// CHECK-NEXT:   %4045 = getelementptr inbounds i8, ptr %4041, i64 3
// CHECK-NEXT:   store i8 5, ptr %4042, align 1
// CHECK-NEXT:   store i8 6, ptr %4043, align 1
// CHECK-NEXT:   store i8 7, ptr %4044, align 1
// CHECK-NEXT:   store i8 8, ptr %4045, align 1
// CHECK-NEXT:   %4046 = load [4 x i8], ptr %4041, align 1
// CHECK-NEXT:   %4047 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 4)
// CHECK-NEXT:   store [4 x i8] %4046, ptr %4047, align 1
// CHECK-NEXT:   %4048 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray, ptr undef }, ptr %4047, 1
// CHECK-NEXT:   %4049 = extractvalue { ptr, ptr } %4040, 1
// CHECK-NEXT:   %4050 = extractvalue { ptr, ptr } %4040, 0
// CHECK-NEXT:   %__llgo_funcval_code523 = call ptr asm "", "=r,0"(ptr %4050)
// CHECK-NEXT:   %4051 = call %reflect.Value %__llgo_funcval_code523(ptr {{(nest|swiftself)}} %4049, %"{{.*}}/runtime/internal/runtime.eface" %4048)
// CHECK-NEXT:   store %reflect.Value %4038, ptr %4024, align 8
// CHECK-NEXT:   store %reflect.Value %4051, ptr %4039, align 8
// CHECK-NEXT:   %4052 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 262
// CHECK-NEXT:   %4053 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4052, i32 0, i32 0
// CHECK-NEXT:   %4054 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4055 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4055, align 8
// CHECK-NEXT:   %4056 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_main.MyByte", ptr undef }, ptr %4055, 1
// CHECK-NEXT:   %4057 = extractvalue { ptr, ptr } %4054, 1
// CHECK-NEXT:   %4058 = extractvalue { ptr, ptr } %4054, 0
// CHECK-NEXT:   %__llgo_funcval_code524 = call ptr asm "", "=r,0"(ptr %4058)
// CHECK-NEXT:   %4059 = call %reflect.Value %__llgo_funcval_code524(ptr {{(nest|swiftself)}} %4057, %"{{.*}}/runtime/internal/runtime.eface" %4056)
// CHECK-NEXT:   %4060 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4052, i32 0, i32 1
// CHECK-NEXT:   %4061 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4062 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4062, align 1
// CHECK-NEXT:   %4063 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_main.MyByte", ptr undef }, ptr %4062, 1
// CHECK-NEXT:   %4064 = extractvalue { ptr, ptr } %4061, 1
// CHECK-NEXT:   %4065 = extractvalue { ptr, ptr } %4061, 0
// CHECK-NEXT:   %__llgo_funcval_code525 = call ptr asm "", "=r,0"(ptr %4065)
// CHECK-NEXT:   %4066 = call %reflect.Value %__llgo_funcval_code525(ptr {{(nest|swiftself)}} %4064, %"{{.*}}/runtime/internal/runtime.eface" %4063)
// CHECK-NEXT:   store %reflect.Value %4059, ptr %4053, align 8
// CHECK-NEXT:   store %reflect.Value %4066, ptr %4060, align 8
// CHECK-NEXT:   %4067 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 263
// CHECK-NEXT:   %4068 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4067, i32 0, i32 0
// CHECK-NEXT:   %4069 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4070 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 2)
// CHECK-NEXT:   %4071 = getelementptr inbounds i8, ptr %4070, i64 0
// CHECK-NEXT:   store i8 1, ptr %4071, align 1
// CHECK-NEXT:   %4072 = getelementptr inbounds i8, ptr %4070, i64 1
// CHECK-NEXT:   store i8 2, ptr %4072, align 1
// CHECK-NEXT:   %4073 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4070, 0
// CHECK-NEXT:   %4074 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4073, i64 2, 1
// CHECK-NEXT:   %4075 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4074, i64 2, 2
// CHECK-NEXT:   %4076 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4075, ptr %4076, align 8
// CHECK-NEXT:   %4077 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_main.MyByte", ptr undef }, ptr %4076, 1
// CHECK-NEXT:   %4078 = extractvalue { ptr, ptr } %4069, 1
// CHECK-NEXT:   %4079 = extractvalue { ptr, ptr } %4069, 0
// CHECK-NEXT:   %__llgo_funcval_code526 = call ptr asm "", "=r,0"(ptr %4079)
// CHECK-NEXT:   %4080 = call %reflect.Value %__llgo_funcval_code526(ptr {{(nest|swiftself)}} %4078, %"{{.*}}/runtime/internal/runtime.eface" %4077)
// CHECK-NEXT:   %4081 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4067, i32 0, i32 1
// CHECK-NEXT:   %4082 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4083 = alloca [2 x i8], align 1
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %4083, i8 0, i64 2, i1 false)
// CHECK-NEXT:   %4084 = getelementptr inbounds i8, ptr %4083, i64 0
// CHECK-NEXT:   %4085 = getelementptr inbounds i8, ptr %4083, i64 1
// CHECK-NEXT:   store i8 1, ptr %4084, align 1
// CHECK-NEXT:   store i8 2, ptr %4085, align 1
// CHECK-NEXT:   %4086 = load [2 x i8], ptr %4083, align 1
// CHECK-NEXT:   %4087 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] %4086, ptr %4087, align 1
// CHECK-NEXT:   %4088 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_main.MyByte", ptr undef }, ptr %4087, 1
// CHECK-NEXT:   %4089 = extractvalue { ptr, ptr } %4082, 1
// CHECK-NEXT:   %4090 = extractvalue { ptr, ptr } %4082, 0
// CHECK-NEXT:   %__llgo_funcval_code527 = call ptr asm "", "=r,0"(ptr %4090)
// CHECK-NEXT:   %4091 = call %reflect.Value %__llgo_funcval_code527(ptr {{(nest|swiftself)}} %4089, %"{{.*}}/runtime/internal/runtime.eface" %4088)
// CHECK-NEXT:   store %reflect.Value %4080, ptr %4068, align 8
// CHECK-NEXT:   store %reflect.Value %4091, ptr %4081, align 8
// CHECK-NEXT:   %4092 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 264
// CHECK-NEXT:   %4093 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4092, i32 0, i32 0
// CHECK-NEXT:   %4094 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4095 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4095, align 8
// CHECK-NEXT:   %4096 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4095, 1
// CHECK-NEXT:   %4097 = extractvalue { ptr, ptr } %4094, 1
// CHECK-NEXT:   %4098 = extractvalue { ptr, ptr } %4094, 0
// CHECK-NEXT:   %__llgo_funcval_code528 = call ptr asm "", "=r,0"(ptr %4098)
// CHECK-NEXT:   %4099 = call %reflect.Value %__llgo_funcval_code528(ptr {{(nest|swiftself)}} %4097, %"{{.*}}/runtime/internal/runtime.eface" %4096)
// CHECK-NEXT:   %4100 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4092, i32 0, i32 1
// CHECK-NEXT:   %4101 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4102 = extractvalue { ptr, ptr } %4101, 1
// CHECK-NEXT:   %4103 = extractvalue { ptr, ptr } %4101, 0
// CHECK-NEXT:   %__llgo_funcval_code529 = call ptr asm "", "=r,0"(ptr %4103)
// CHECK-NEXT:   %4104 = call %reflect.Value %__llgo_funcval_code529(ptr {{(nest|swiftself)}} %4102, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4099, ptr %4093, align 8
// CHECK-NEXT:   store %reflect.Value %4104, ptr %4100, align 8
// CHECK-NEXT:   %4105 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 265
// CHECK-NEXT:   %4106 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4105, i32 0, i32 0
// CHECK-NEXT:   %4107 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4108 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4108, align 8
// CHECK-NEXT:   %4109 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4108, 1
// CHECK-NEXT:   %4110 = extractvalue { ptr, ptr } %4107, 1
// CHECK-NEXT:   %4111 = extractvalue { ptr, ptr } %4107, 0
// CHECK-NEXT:   %__llgo_funcval_code530 = call ptr asm "", "=r,0"(ptr %4111)
// CHECK-NEXT:   %4112 = call %reflect.Value %__llgo_funcval_code530(ptr {{(nest|swiftself)}} %4110, %"{{.*}}/runtime/internal/runtime.eface" %4109)
// CHECK-NEXT:   %4113 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4105, i32 0, i32 1
// CHECK-NEXT:   %4114 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4115 = extractvalue { ptr, ptr } %4114, 1
// CHECK-NEXT:   %4116 = extractvalue { ptr, ptr } %4114, 0
// CHECK-NEXT:   %__llgo_funcval_code531 = call ptr asm "", "=r,0"(ptr %4116)
// CHECK-NEXT:   %4117 = call %reflect.Value %__llgo_funcval_code531(ptr {{(nest|swiftself)}} %4115, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4112, ptr %4106, align 8
// CHECK-NEXT:   store %reflect.Value %4117, ptr %4113, align 8
// CHECK-NEXT:   %4118 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 266
// CHECK-NEXT:   %4119 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4118, i32 0, i32 0
// CHECK-NEXT:   %4120 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4121 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %4122 = getelementptr inbounds i8, ptr %4121, i64 0
// CHECK-NEXT:   store i8 7, ptr %4122, align 1
// CHECK-NEXT:   %4123 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4121, 0
// CHECK-NEXT:   %4124 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4123, i64 1, 1
// CHECK-NEXT:   %4125 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4124, i64 1, 2
// CHECK-NEXT:   %4126 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4125, ptr %4126, align 8
// CHECK-NEXT:   %4127 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4126, 1
// CHECK-NEXT:   %4128 = extractvalue { ptr, ptr } %4120, 1
// CHECK-NEXT:   %4129 = extractvalue { ptr, ptr } %4120, 0
// CHECK-NEXT:   %__llgo_funcval_code532 = call ptr asm "", "=r,0"(ptr %4129)
// CHECK-NEXT:   %4130 = call %reflect.Value %__llgo_funcval_code532(ptr {{(nest|swiftself)}} %4128, %"{{.*}}/runtime/internal/runtime.eface" %4127)
// CHECK-NEXT:   %4131 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4118, i32 0, i32 1
// CHECK-NEXT:   %4132 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4133 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %4134 = getelementptr inbounds i8, ptr %4133, i64 0
// CHECK-NEXT:   store i8 7, ptr %4134, align 1
// CHECK-NEXT:   %4135 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[1]_llgo_uint8", ptr undef }, ptr %4133, 1
// CHECK-NEXT:   %4136 = extractvalue { ptr, ptr } %4132, 1
// CHECK-NEXT:   %4137 = extractvalue { ptr, ptr } %4132, 0
// CHECK-NEXT:   %__llgo_funcval_code533 = call ptr asm "", "=r,0"(ptr %4137)
// CHECK-NEXT:   %4138 = call %reflect.Value %__llgo_funcval_code533(ptr {{(nest|swiftself)}} %4136, %"{{.*}}/runtime/internal/runtime.eface" %4135)
// CHECK-NEXT:   store %reflect.Value %4130, ptr %4119, align 8
// CHECK-NEXT:   store %reflect.Value %4138, ptr %4131, align 8
// CHECK-NEXT:   %4139 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 267
// CHECK-NEXT:   %4140 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4139, i32 0, i32 0
// CHECK-NEXT:   %4141 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4142 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4142, align 8
// CHECK-NEXT:   %4143 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4142, 1
// CHECK-NEXT:   %4144 = extractvalue { ptr, ptr } %4141, 1
// CHECK-NEXT:   %4145 = extractvalue { ptr, ptr } %4141, 0
// CHECK-NEXT:   %__llgo_funcval_code534 = call ptr asm "", "=r,0"(ptr %4145)
// CHECK-NEXT:   %4146 = call %reflect.Value %__llgo_funcval_code534(ptr {{(nest|swiftself)}} %4144, %"{{.*}}/runtime/internal/runtime.eface" %4143)
// CHECK-NEXT:   %4147 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4139, i32 0, i32 1
// CHECK-NEXT:   %4148 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4149 = extractvalue { ptr, ptr } %4148, 1
// CHECK-NEXT:   %4150 = extractvalue { ptr, ptr } %4148, 0
// CHECK-NEXT:   %__llgo_funcval_code535 = call ptr asm "", "=r,0"(ptr %4150)
// CHECK-NEXT:   %4151 = call %reflect.Value %__llgo_funcval_code535(ptr {{(nest|swiftself)}} %4149, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4146, ptr %4140, align 8
// CHECK-NEXT:   store %reflect.Value %4151, ptr %4147, align 8
// CHECK-NEXT:   %4152 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 268
// CHECK-NEXT:   %4153 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4152, i32 0, i32 0
// CHECK-NEXT:   %4154 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4155 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4155, align 8
// CHECK-NEXT:   %4156 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4155, 1
// CHECK-NEXT:   %4157 = extractvalue { ptr, ptr } %4154, 1
// CHECK-NEXT:   %4158 = extractvalue { ptr, ptr } %4154, 0
// CHECK-NEXT:   %__llgo_funcval_code536 = call ptr asm "", "=r,0"(ptr %4158)
// CHECK-NEXT:   %4159 = call %reflect.Value %__llgo_funcval_code536(ptr {{(nest|swiftself)}} %4157, %"{{.*}}/runtime/internal/runtime.eface" %4156)
// CHECK-NEXT:   %4160 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4152, i32 0, i32 1
// CHECK-NEXT:   %4161 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4162 = extractvalue { ptr, ptr } %4161, 1
// CHECK-NEXT:   %4163 = extractvalue { ptr, ptr } %4161, 0
// CHECK-NEXT:   %__llgo_funcval_code537 = call ptr asm "", "=r,0"(ptr %4163)
// CHECK-NEXT:   %4164 = call %reflect.Value %__llgo_funcval_code537(ptr {{(nest|swiftself)}} %4162, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4159, ptr %4153, align 8
// CHECK-NEXT:   store %reflect.Value %4164, ptr %4160, align 8
// CHECK-NEXT:   %4165 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 269
// CHECK-NEXT:   %4166 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4165, i32 0, i32 0
// CHECK-NEXT:   %4167 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4168 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %4169 = getelementptr inbounds i8, ptr %4168, i64 0
// CHECK-NEXT:   store i8 9, ptr %4169, align 1
// CHECK-NEXT:   %4170 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4168, 0
// CHECK-NEXT:   %4171 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4170, i64 1, 1
// CHECK-NEXT:   %4172 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4171, i64 1, 2
// CHECK-NEXT:   %4173 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4172, ptr %4173, align 8
// CHECK-NEXT:   %4174 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4173, 1
// CHECK-NEXT:   %4175 = extractvalue { ptr, ptr } %4167, 1
// CHECK-NEXT:   %4176 = extractvalue { ptr, ptr } %4167, 0
// CHECK-NEXT:   %__llgo_funcval_code538 = call ptr asm "", "=r,0"(ptr %4176)
// CHECK-NEXT:   %4177 = call %reflect.Value %__llgo_funcval_code538(ptr {{(nest|swiftself)}} %4175, %"{{.*}}/runtime/internal/runtime.eface" %4174)
// CHECK-NEXT:   %4178 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4165, i32 0, i32 1
// CHECK-NEXT:   %4179 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4180 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 1)
// CHECK-NEXT:   %4181 = getelementptr inbounds i8, ptr %4180, i64 0
// CHECK-NEXT:   store i8 9, ptr %4181, align 1
// CHECK-NEXT:   %4182 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[1]_llgo_uint8", ptr undef }, ptr %4180, 1
// CHECK-NEXT:   %4183 = extractvalue { ptr, ptr } %4179, 1
// CHECK-NEXT:   %4184 = extractvalue { ptr, ptr } %4179, 0
// CHECK-NEXT:   %__llgo_funcval_code539 = call ptr asm "", "=r,0"(ptr %4184)
// CHECK-NEXT:   %4185 = call %reflect.Value %__llgo_funcval_code539(ptr {{(nest|swiftself)}} %4183, %"{{.*}}/runtime/internal/runtime.eface" %4182)
// CHECK-NEXT:   store %reflect.Value %4177, ptr %4166, align 8
// CHECK-NEXT:   store %reflect.Value %4185, ptr %4178, align 8
// CHECK-NEXT:   %4186 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 270
// CHECK-NEXT:   %4187 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4186, i32 0, i32 0
// CHECK-NEXT:   %4188 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4189 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4189, align 8
// CHECK-NEXT:   %4190 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4189, 1
// CHECK-NEXT:   %4191 = extractvalue { ptr, ptr } %4188, 1
// CHECK-NEXT:   %4192 = extractvalue { ptr, ptr } %4188, 0
// CHECK-NEXT:   %__llgo_funcval_code540 = call ptr asm "", "=r,0"(ptr %4192)
// CHECK-NEXT:   %4193 = call %reflect.Value %__llgo_funcval_code540(ptr {{(nest|swiftself)}} %4191, %"{{.*}}/runtime/internal/runtime.eface" %4190)
// CHECK-NEXT:   %4194 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4186, i32 0, i32 1
// CHECK-NEXT:   %4195 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4196 = extractvalue { ptr, ptr } %4195, 1
// CHECK-NEXT:   %4197 = extractvalue { ptr, ptr } %4195, 0
// CHECK-NEXT:   %__llgo_funcval_code541 = call ptr asm "", "=r,0"(ptr %4197)
// CHECK-NEXT:   %4198 = call %reflect.Value %__llgo_funcval_code541(ptr {{(nest|swiftself)}} %4196, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr0, ptr null })
// CHECK-NEXT:   store %reflect.Value %4193, ptr %4187, align 8
// CHECK-NEXT:   store %reflect.Value %4198, ptr %4194, align 8
// CHECK-NEXT:   %4199 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 271
// CHECK-NEXT:   %4200 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4199, i32 0, i32 0
// CHECK-NEXT:   %4201 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4202 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4202, align 8
// CHECK-NEXT:   %4203 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4202, 1
// CHECK-NEXT:   %4204 = extractvalue { ptr, ptr } %4201, 1
// CHECK-NEXT:   %4205 = extractvalue { ptr, ptr } %4201, 0
// CHECK-NEXT:   %__llgo_funcval_code542 = call ptr asm "", "=r,0"(ptr %4205)
// CHECK-NEXT:   %4206 = call %reflect.Value %__llgo_funcval_code542(ptr {{(nest|swiftself)}} %4204, %"{{.*}}/runtime/internal/runtime.eface" %4203)
// CHECK-NEXT:   %4207 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4199, i32 0, i32 1
// CHECK-NEXT:   %4208 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4209 = extractvalue { ptr, ptr } %4208, 1
// CHECK-NEXT:   %4210 = extractvalue { ptr, ptr } %4208, 0
// CHECK-NEXT:   %__llgo_funcval_code543 = call ptr asm "", "=r,0"(ptr %4210)
// CHECK-NEXT:   %4211 = call %reflect.Value %__llgo_funcval_code543(ptr {{(nest|swiftself)}} %4209, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr0, ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4206, ptr %4200, align 8
// CHECK-NEXT:   store %reflect.Value %4211, ptr %4207, align 8
// CHECK-NEXT:   %4212 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 272
// CHECK-NEXT:   %4213 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4212, i32 0, i32 0
// CHECK-NEXT:   %4214 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4215 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4216 = getelementptr inbounds i8, ptr %4215, i64 0
// CHECK-NEXT:   store i8 1, ptr %4216, align 1
// CHECK-NEXT:   %4217 = getelementptr inbounds i8, ptr %4215, i64 1
// CHECK-NEXT:   store i8 2, ptr %4217, align 1
// CHECK-NEXT:   %4218 = getelementptr inbounds i8, ptr %4215, i64 2
// CHECK-NEXT:   store i8 3, ptr %4218, align 1
// CHECK-NEXT:   %4219 = getelementptr inbounds i8, ptr %4215, i64 3
// CHECK-NEXT:   store i8 4, ptr %4219, align 1
// CHECK-NEXT:   %4220 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4215, 0
// CHECK-NEXT:   %4221 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4220, i64 4, 1
// CHECK-NEXT:   %4222 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4221, i64 4, 2
// CHECK-NEXT:   %4223 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4222, ptr %4223, align 8
// CHECK-NEXT:   %4224 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4223, 1
// CHECK-NEXT:   %4225 = extractvalue { ptr, ptr } %4214, 1
// CHECK-NEXT:   %4226 = extractvalue { ptr, ptr } %4214, 0
// CHECK-NEXT:   %__llgo_funcval_code544 = call ptr asm "", "=r,0"(ptr %4226)
// CHECK-NEXT:   %4227 = call %reflect.Value %__llgo_funcval_code544(ptr {{(nest|swiftself)}} %4225, %"{{.*}}/runtime/internal/runtime.eface" %4224)
// CHECK-NEXT:   %4228 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4212, i32 0, i32 1
// CHECK-NEXT:   %4229 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4230 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4231 = getelementptr inbounds i8, ptr %4230, i64 0
// CHECK-NEXT:   %4232 = getelementptr inbounds i8, ptr %4230, i64 1
// CHECK-NEXT:   %4233 = getelementptr inbounds i8, ptr %4230, i64 2
// CHECK-NEXT:   %4234 = getelementptr inbounds i8, ptr %4230, i64 3
// CHECK-NEXT:   store i8 1, ptr %4231, align 1
// CHECK-NEXT:   store i8 2, ptr %4232, align 1
// CHECK-NEXT:   store i8 3, ptr %4233, align 1
// CHECK-NEXT:   store i8 4, ptr %4234, align 1
// CHECK-NEXT:   %4235 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr, ptr undef }, ptr %4230, 1
// CHECK-NEXT:   %4236 = extractvalue { ptr, ptr } %4229, 1
// CHECK-NEXT:   %4237 = extractvalue { ptr, ptr } %4229, 0
// CHECK-NEXT:   %__llgo_funcval_code545 = call ptr asm "", "=r,0"(ptr %4237)
// CHECK-NEXT:   %4238 = call %reflect.Value %__llgo_funcval_code545(ptr {{(nest|swiftself)}} %4236, %"{{.*}}/runtime/internal/runtime.eface" %4235)
// CHECK-NEXT:   store %reflect.Value %4227, ptr %4213, align 8
// CHECK-NEXT:   store %reflect.Value %4238, ptr %4228, align 8
// CHECK-NEXT:   %4239 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 273
// CHECK-NEXT:   %4240 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4239, i32 0, i32 0
// CHECK-NEXT:   %4241 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4242 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4242, align 8
// CHECK-NEXT:   %4243 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4242, 1
// CHECK-NEXT:   %4244 = extractvalue { ptr, ptr } %4241, 1
// CHECK-NEXT:   %4245 = extractvalue { ptr, ptr } %4241, 0
// CHECK-NEXT:   %__llgo_funcval_code546 = call ptr asm "", "=r,0"(ptr %4245)
// CHECK-NEXT:   %4246 = call %reflect.Value %__llgo_funcval_code546(ptr {{(nest|swiftself)}} %4244, %"{{.*}}/runtime/internal/runtime.eface" %4243)
// CHECK-NEXT:   %4247 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4239, i32 0, i32 1
// CHECK-NEXT:   %4248 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4249 = extractvalue { ptr, ptr } %4248, 1
// CHECK-NEXT:   %4250 = extractvalue { ptr, ptr } %4248, 0
// CHECK-NEXT:   %__llgo_funcval_code547 = call ptr asm "", "=r,0"(ptr %4250)
// CHECK-NEXT:   %4251 = call %reflect.Value %__llgo_funcval_code547(ptr {{(nest|swiftself)}} %4249, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr0, ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4246, ptr %4240, align 8
// CHECK-NEXT:   store %reflect.Value %4251, ptr %4247, align 8
// CHECK-NEXT:   %4252 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 274
// CHECK-NEXT:   %4253 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4252, i32 0, i32 0
// CHECK-NEXT:   %4254 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4255 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4256 = getelementptr inbounds i8, ptr %4255, i64 0
// CHECK-NEXT:   store i8 5, ptr %4256, align 1
// CHECK-NEXT:   %4257 = getelementptr inbounds i8, ptr %4255, i64 1
// CHECK-NEXT:   store i8 6, ptr %4257, align 1
// CHECK-NEXT:   %4258 = getelementptr inbounds i8, ptr %4255, i64 2
// CHECK-NEXT:   store i8 7, ptr %4258, align 1
// CHECK-NEXT:   %4259 = getelementptr inbounds i8, ptr %4255, i64 3
// CHECK-NEXT:   store i8 8, ptr %4259, align 1
// CHECK-NEXT:   %4260 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4255, 0
// CHECK-NEXT:   %4261 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4260, i64 4, 1
// CHECK-NEXT:   %4262 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4261, i64 4, 2
// CHECK-NEXT:   %4263 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4262, ptr %4263, align 8
// CHECK-NEXT:   %4264 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4263, 1
// CHECK-NEXT:   %4265 = extractvalue { ptr, ptr } %4254, 1
// CHECK-NEXT:   %4266 = extractvalue { ptr, ptr } %4254, 0
// CHECK-NEXT:   %__llgo_funcval_code548 = call ptr asm "", "=r,0"(ptr %4266)
// CHECK-NEXT:   %4267 = call %reflect.Value %__llgo_funcval_code548(ptr {{(nest|swiftself)}} %4265, %"{{.*}}/runtime/internal/runtime.eface" %4264)
// CHECK-NEXT:   %4268 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4252, i32 0, i32 1
// CHECK-NEXT:   %4269 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4270 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4271 = getelementptr inbounds i8, ptr %4270, i64 0
// CHECK-NEXT:   %4272 = getelementptr inbounds i8, ptr %4270, i64 1
// CHECK-NEXT:   %4273 = getelementptr inbounds i8, ptr %4270, i64 2
// CHECK-NEXT:   %4274 = getelementptr inbounds i8, ptr %4270, i64 3
// CHECK-NEXT:   store i8 5, ptr %4271, align 1
// CHECK-NEXT:   store i8 6, ptr %4272, align 1
// CHECK-NEXT:   store i8 7, ptr %4273, align 1
// CHECK-NEXT:   store i8 8, ptr %4274, align 1
// CHECK-NEXT:   %4275 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr, ptr undef }, ptr %4270, 1
// CHECK-NEXT:   %4276 = extractvalue { ptr, ptr } %4269, 1
// CHECK-NEXT:   %4277 = extractvalue { ptr, ptr } %4269, 0
// CHECK-NEXT:   %__llgo_funcval_code549 = call ptr asm "", "=r,0"(ptr %4277)
// CHECK-NEXT:   %4278 = call %reflect.Value %__llgo_funcval_code549(ptr {{(nest|swiftself)}} %4276, %"{{.*}}/runtime/internal/runtime.eface" %4275)
// CHECK-NEXT:   store %reflect.Value %4267, ptr %4253, align 8
// CHECK-NEXT:   store %reflect.Value %4278, ptr %4268, align 8
// CHECK-NEXT:   %4279 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 275
// CHECK-NEXT:   %4280 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4279, i32 0, i32 0
// CHECK-NEXT:   %4281 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4282 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4282, align 8
// CHECK-NEXT:   %4283 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4282, 1
// CHECK-NEXT:   %4284 = extractvalue { ptr, ptr } %4281, 1
// CHECK-NEXT:   %4285 = extractvalue { ptr, ptr } %4281, 0
// CHECK-NEXT:   %__llgo_funcval_code550 = call ptr asm "", "=r,0"(ptr %4285)
// CHECK-NEXT:   %4286 = call %reflect.Value %__llgo_funcval_code550(ptr {{(nest|swiftself)}} %4284, %"{{.*}}/runtime/internal/runtime.eface" %4283)
// CHECK-NEXT:   %4287 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4279, i32 0, i32 1
// CHECK-NEXT:   %4288 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4289 = extractvalue { ptr, ptr } %4288, 1
// CHECK-NEXT:   %4290 = extractvalue { ptr, ptr } %4288, 0
// CHECK-NEXT:   %__llgo_funcval_code551 = call ptr asm "", "=r,0"(ptr %4290)
// CHECK-NEXT:   %4291 = call %reflect.Value %__llgo_funcval_code551(ptr {{(nest|swiftself)}} %4289, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr null })
// CHECK-NEXT:   store %reflect.Value %4286, ptr %4280, align 8
// CHECK-NEXT:   store %reflect.Value %4291, ptr %4287, align 8
// CHECK-NEXT:   %4292 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 276
// CHECK-NEXT:   %4293 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4292, i32 0, i32 0
// CHECK-NEXT:   %4294 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4295 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4295, align 8
// CHECK-NEXT:   %4296 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4295, 1
// CHECK-NEXT:   %4297 = extractvalue { ptr, ptr } %4294, 1
// CHECK-NEXT:   %4298 = extractvalue { ptr, ptr } %4294, 0
// CHECK-NEXT:   %__llgo_funcval_code552 = call ptr asm "", "=r,0"(ptr %4298)
// CHECK-NEXT:   %4299 = call %reflect.Value %__llgo_funcval_code552(ptr {{(nest|swiftself)}} %4297, %"{{.*}}/runtime/internal/runtime.eface" %4296)
// CHECK-NEXT:   %4300 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4292, i32 0, i32 1
// CHECK-NEXT:   %4301 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4302 = extractvalue { ptr, ptr } %4301, 1
// CHECK-NEXT:   %4303 = extractvalue { ptr, ptr } %4301, 0
// CHECK-NEXT:   %__llgo_funcval_code553 = call ptr asm "", "=r,0"(ptr %4303)
// CHECK-NEXT:   %4304 = call %reflect.Value %__llgo_funcval_code553(ptr {{(nest|swiftself)}} %4302, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4299, ptr %4293, align 8
// CHECK-NEXT:   store %reflect.Value %4304, ptr %4300, align 8
// CHECK-NEXT:   %4305 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 277
// CHECK-NEXT:   %4306 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4305, i32 0, i32 0
// CHECK-NEXT:   %4307 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4308 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4309 = getelementptr inbounds i8, ptr %4308, i64 0
// CHECK-NEXT:   store i8 1, ptr %4309, align 1
// CHECK-NEXT:   %4310 = getelementptr inbounds i8, ptr %4308, i64 1
// CHECK-NEXT:   store i8 2, ptr %4310, align 1
// CHECK-NEXT:   %4311 = getelementptr inbounds i8, ptr %4308, i64 2
// CHECK-NEXT:   store i8 3, ptr %4311, align 1
// CHECK-NEXT:   %4312 = getelementptr inbounds i8, ptr %4308, i64 3
// CHECK-NEXT:   store i8 4, ptr %4312, align 1
// CHECK-NEXT:   %4313 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4308, 0
// CHECK-NEXT:   %4314 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4313, i64 4, 1
// CHECK-NEXT:   %4315 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4314, i64 4, 2
// CHECK-NEXT:   %4316 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4315, ptr %4316, align 8
// CHECK-NEXT:   %4317 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4316, 1
// CHECK-NEXT:   %4318 = extractvalue { ptr, ptr } %4307, 1
// CHECK-NEXT:   %4319 = extractvalue { ptr, ptr } %4307, 0
// CHECK-NEXT:   %__llgo_funcval_code554 = call ptr asm "", "=r,0"(ptr %4319)
// CHECK-NEXT:   %4320 = call %reflect.Value %__llgo_funcval_code554(ptr {{(nest|swiftself)}} %4318, %"{{.*}}/runtime/internal/runtime.eface" %4317)
// CHECK-NEXT:   %4321 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4305, i32 0, i32 1
// CHECK-NEXT:   %4322 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4323 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4324 = getelementptr inbounds i8, ptr %4323, i64 0
// CHECK-NEXT:   %4325 = getelementptr inbounds i8, ptr %4323, i64 1
// CHECK-NEXT:   %4326 = getelementptr inbounds i8, ptr %4323, i64 2
// CHECK-NEXT:   %4327 = getelementptr inbounds i8, ptr %4323, i64 3
// CHECK-NEXT:   store i8 1, ptr %4324, align 1
// CHECK-NEXT:   store i8 2, ptr %4325, align 1
// CHECK-NEXT:   store i8 3, ptr %4326, align 1
// CHECK-NEXT:   store i8 4, ptr %4327, align 1
// CHECK-NEXT:   %4328 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray", ptr undef }, ptr %4323, 1
// CHECK-NEXT:   %4329 = extractvalue { ptr, ptr } %4322, 1
// CHECK-NEXT:   %4330 = extractvalue { ptr, ptr } %4322, 0
// CHECK-NEXT:   %__llgo_funcval_code555 = call ptr asm "", "=r,0"(ptr %4330)
// CHECK-NEXT:   %4331 = call %reflect.Value %__llgo_funcval_code555(ptr {{(nest|swiftself)}} %4329, %"{{.*}}/runtime/internal/runtime.eface" %4328)
// CHECK-NEXT:   store %reflect.Value %4320, ptr %4306, align 8
// CHECK-NEXT:   store %reflect.Value %4331, ptr %4321, align 8
// CHECK-NEXT:   %4332 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 278
// CHECK-NEXT:   %4333 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4332, i32 0, i32 0
// CHECK-NEXT:   %4334 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4335 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4335, align 8
// CHECK-NEXT:   %4336 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4335, 1
// CHECK-NEXT:   %4337 = extractvalue { ptr, ptr } %4334, 1
// CHECK-NEXT:   %4338 = extractvalue { ptr, ptr } %4334, 0
// CHECK-NEXT:   %__llgo_funcval_code556 = call ptr asm "", "=r,0"(ptr %4338)
// CHECK-NEXT:   %4339 = call %reflect.Value %__llgo_funcval_code556(ptr {{(nest|swiftself)}} %4337, %"{{.*}}/runtime/internal/runtime.eface" %4336)
// CHECK-NEXT:   %4340 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4332, i32 0, i32 1
// CHECK-NEXT:   %4341 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4342 = extractvalue { ptr, ptr } %4341, 1
// CHECK-NEXT:   %4343 = extractvalue { ptr, ptr } %4341, 0
// CHECK-NEXT:   %__llgo_funcval_code557 = call ptr asm "", "=r,0"(ptr %4343)
// CHECK-NEXT:   %4344 = call %reflect.Value %__llgo_funcval_code557(ptr {{(nest|swiftself)}} %4342, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr null })
// CHECK-NEXT:   store %reflect.Value %4339, ptr %4333, align 8
// CHECK-NEXT:   store %reflect.Value %4344, ptr %4340, align 8
// CHECK-NEXT:   %4345 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 279
// CHECK-NEXT:   %4346 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4345, i32 0, i32 0
// CHECK-NEXT:   %4347 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4348 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4348, align 8
// CHECK-NEXT:   %4349 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4348, 1
// CHECK-NEXT:   %4350 = extractvalue { ptr, ptr } %4347, 1
// CHECK-NEXT:   %4351 = extractvalue { ptr, ptr } %4347, 0
// CHECK-NEXT:   %__llgo_funcval_code558 = call ptr asm "", "=r,0"(ptr %4351)
// CHECK-NEXT:   %4352 = call %reflect.Value %__llgo_funcval_code558(ptr {{(nest|swiftself)}} %4350, %"{{.*}}/runtime/internal/runtime.eface" %4349)
// CHECK-NEXT:   %4353 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4345, i32 0, i32 1
// CHECK-NEXT:   %4354 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4355 = extractvalue { ptr, ptr } %4354, 1
// CHECK-NEXT:   %4356 = extractvalue { ptr, ptr } %4354, 0
// CHECK-NEXT:   %__llgo_funcval_code559 = call ptr asm "", "=r,0"(ptr %4356)
// CHECK-NEXT:   %4357 = call %reflect.Value %__llgo_funcval_code559(ptr {{(nest|swiftself)}} %4355, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4352, ptr %4346, align 8
// CHECK-NEXT:   store %reflect.Value %4357, ptr %4353, align 8
// CHECK-NEXT:   %4358 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 280
// CHECK-NEXT:   %4359 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4358, i32 0, i32 0
// CHECK-NEXT:   %4360 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4361 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4362 = getelementptr inbounds i8, ptr %4361, i64 0
// CHECK-NEXT:   store i8 5, ptr %4362, align 1
// CHECK-NEXT:   %4363 = getelementptr inbounds i8, ptr %4361, i64 1
// CHECK-NEXT:   store i8 6, ptr %4363, align 1
// CHECK-NEXT:   %4364 = getelementptr inbounds i8, ptr %4361, i64 2
// CHECK-NEXT:   store i8 7, ptr %4364, align 1
// CHECK-NEXT:   %4365 = getelementptr inbounds i8, ptr %4361, i64 3
// CHECK-NEXT:   store i8 8, ptr %4365, align 1
// CHECK-NEXT:   %4366 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %4361, 0
// CHECK-NEXT:   %4367 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4366, i64 4, 1
// CHECK-NEXT:   %4368 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %4367, i64 4, 2
// CHECK-NEXT:   %4369 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %4368, ptr %4369, align 8
// CHECK-NEXT:   %4370 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4369, 1
// CHECK-NEXT:   %4371 = extractvalue { ptr, ptr } %4360, 1
// CHECK-NEXT:   %4372 = extractvalue { ptr, ptr } %4360, 0
// CHECK-NEXT:   %__llgo_funcval_code560 = call ptr asm "", "=r,0"(ptr %4372)
// CHECK-NEXT:   %4373 = call %reflect.Value %__llgo_funcval_code560(ptr {{(nest|swiftself)}} %4371, %"{{.*}}/runtime/internal/runtime.eface" %4370)
// CHECK-NEXT:   %4374 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4358, i32 0, i32 1
// CHECK-NEXT:   %4375 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4376 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %4377 = getelementptr inbounds i8, ptr %4376, i64 0
// CHECK-NEXT:   %4378 = getelementptr inbounds i8, ptr %4376, i64 1
// CHECK-NEXT:   %4379 = getelementptr inbounds i8, ptr %4376, i64 2
// CHECK-NEXT:   %4380 = getelementptr inbounds i8, ptr %4376, i64 3
// CHECK-NEXT:   store i8 5, ptr %4377, align 1
// CHECK-NEXT:   store i8 6, ptr %4378, align 1
// CHECK-NEXT:   store i8 7, ptr %4379, align 1
// CHECK-NEXT:   store i8 8, ptr %4380, align 1
// CHECK-NEXT:   %4381 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray", ptr undef }, ptr %4376, 1
// CHECK-NEXT:   %4382 = extractvalue { ptr, ptr } %4375, 1
// CHECK-NEXT:   %4383 = extractvalue { ptr, ptr } %4375, 0
// CHECK-NEXT:   %__llgo_funcval_code561 = call ptr asm "", "=r,0"(ptr %4383)
// CHECK-NEXT:   %4384 = call %reflect.Value %__llgo_funcval_code561(ptr {{(nest|swiftself)}} %4382, %"{{.*}}/runtime/internal/runtime.eface" %4381)
// CHECK-NEXT:   store %reflect.Value %4373, ptr %4359, align 8
// CHECK-NEXT:   store %reflect.Value %4384, ptr %4374, align 8
// CHECK-NEXT:   %4385 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 281
// CHECK-NEXT:   %4386 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4385, i32 0, i32 0
// CHECK-NEXT:   %4387 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4388 = extractvalue { ptr, ptr } %4387, 1
// CHECK-NEXT:   %4389 = extractvalue { ptr, ptr } %4387, 0
// CHECK-NEXT:   %__llgo_funcval_code562 = call ptr asm "", "=r,0"(ptr %4389)
// CHECK-NEXT:   %4390 = call %reflect.Value %__llgo_funcval_code562(ptr {{(nest|swiftself)}} %4388, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   %4391 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4385, i32 0, i32 1
// CHECK-NEXT:   %4392 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4393 = extractvalue { ptr, ptr } %4392, 1
// CHECK-NEXT:   %4394 = extractvalue { ptr, ptr } %4392, 0
// CHECK-NEXT:   %__llgo_funcval_code563 = call ptr asm "", "=r,0"(ptr %4394)
// CHECK-NEXT:   %4395 = call %reflect.Value %__llgo_funcval_code563(ptr {{(nest|swiftself)}} %4393, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4390, ptr %4386, align 8
// CHECK-NEXT:   store %reflect.Value %4395, ptr %4391, align 8
// CHECK-NEXT:   %4396 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 282
// CHECK-NEXT:   %4397 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4396, i32 0, i32 0
// CHECK-NEXT:   %4398 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4399 = extractvalue { ptr, ptr } %4398, 1
// CHECK-NEXT:   %4400 = extractvalue { ptr, ptr } %4398, 0
// CHECK-NEXT:   %__llgo_funcval_code564 = call ptr asm "", "=r,0"(ptr %4400)
// CHECK-NEXT:   %4401 = call %reflect.Value %__llgo_funcval_code564(ptr {{(nest|swiftself)}} %4399, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyBytesArray0", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   %4402 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4396, i32 0, i32 1
// CHECK-NEXT:   %4403 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4404 = extractvalue { ptr, ptr } %4403, 1
// CHECK-NEXT:   %4405 = extractvalue { ptr, ptr } %4403, 0
// CHECK-NEXT:   %__llgo_funcval_code565 = call ptr asm "", "=r,0"(ptr %4405)
// CHECK-NEXT:   %4406 = call %reflect.Value %__llgo_funcval_code565(ptr {{(nest|swiftself)}} %4404, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4401, ptr %4397, align 8
// CHECK-NEXT:   store %reflect.Value %4406, ptr %4402, align 8
// CHECK-NEXT:   %4407 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 283
// CHECK-NEXT:   %4408 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4407, i32 0, i32 0
// CHECK-NEXT:   %4409 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4410 = extractvalue { ptr, ptr } %4409, 1
// CHECK-NEXT:   %4411 = extractvalue { ptr, ptr } %4409, 0
// CHECK-NEXT:   %__llgo_funcval_code566 = call ptr asm "", "=r,0"(ptr %4411)
// CHECK-NEXT:   %4412 = call %reflect.Value %__llgo_funcval_code566(ptr {{(nest|swiftself)}} %4410, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr0, ptr null })
// CHECK-NEXT:   %4413 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4407, i32 0, i32 1
// CHECK-NEXT:   %4414 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4415 = extractvalue { ptr, ptr } %4414, 1
// CHECK-NEXT:   %4416 = extractvalue { ptr, ptr } %4414, 0
// CHECK-NEXT:   %__llgo_funcval_code567 = call ptr asm "", "=r,0"(ptr %4416)
// CHECK-NEXT:   %4417 = call %reflect.Value %__llgo_funcval_code567(ptr {{(nest|swiftself)}} %4415, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4412, ptr %4408, align 8
// CHECK-NEXT:   store %reflect.Value %4417, ptr %4413, align 8
// CHECK-NEXT:   %4418 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 284
// CHECK-NEXT:   %4419 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4418, i32 0, i32 0
// CHECK-NEXT:   %4420 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4421 = extractvalue { ptr, ptr } %4420, 1
// CHECK-NEXT:   %4422 = extractvalue { ptr, ptr } %4420, 0
// CHECK-NEXT:   %__llgo_funcval_code568 = call ptr asm "", "=r,0"(ptr %4422)
// CHECK-NEXT:   %4423 = call %reflect.Value %__llgo_funcval_code568(ptr {{(nest|swiftself)}} %4421, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*[0]_llgo_uint8", ptr null })
// CHECK-NEXT:   %4424 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4418, i32 0, i32 1
// CHECK-NEXT:   %4425 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4426 = extractvalue { ptr, ptr } %4425, 1
// CHECK-NEXT:   %4427 = extractvalue { ptr, ptr } %4425, 0
// CHECK-NEXT:   %__llgo_funcval_code569 = call ptr asm "", "=r,0"(ptr %4427)
// CHECK-NEXT:   %4428 = call %reflect.Value %__llgo_funcval_code569(ptr {{(nest|swiftself)}} %4426, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArrayPtr0, ptr null })
// CHECK-NEXT:   store %reflect.Value %4423, ptr %4419, align 8
// CHECK-NEXT:   store %reflect.Value %4428, ptr %4424, align 8
// CHECK-NEXT:   %4429 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 285
// CHECK-NEXT:   %4430 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4429, i32 0, i32 0
// CHECK-NEXT:   %4431 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4432 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4433 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_int", ptr undef }, ptr %4432, 1
// CHECK-NEXT:   %4434 = extractvalue { ptr, ptr } %4431, 1
// CHECK-NEXT:   %4435 = extractvalue { ptr, ptr } %4431, 0
// CHECK-NEXT:   %__llgo_funcval_code570 = call ptr asm "", "=r,0"(ptr %4435)
// CHECK-NEXT:   %4436 = call %reflect.Value %__llgo_funcval_code570(ptr {{(nest|swiftself)}} %4434, %"{{.*}}/runtime/internal/runtime.eface" %4433)
// CHECK-NEXT:   %4437 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4429, i32 0, i32 1
// CHECK-NEXT:   %4438 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4439 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4440 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.integer", ptr undef }, ptr %4439, 1
// CHECK-NEXT:   %4441 = extractvalue { ptr, ptr } %4438, 1
// CHECK-NEXT:   %4442 = extractvalue { ptr, ptr } %4438, 0
// CHECK-NEXT:   %__llgo_funcval_code571 = call ptr asm "", "=r,0"(ptr %4442)
// CHECK-NEXT:   %4443 = call %reflect.Value %__llgo_funcval_code571(ptr {{(nest|swiftself)}} %4441, %"{{.*}}/runtime/internal/runtime.eface" %4440)
// CHECK-NEXT:   store %reflect.Value %4436, ptr %4430, align 8
// CHECK-NEXT:   store %reflect.Value %4443, ptr %4437, align 8
// CHECK-NEXT:   %4444 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 286
// CHECK-NEXT:   %4445 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4444, i32 0, i32 0
// CHECK-NEXT:   %4446 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4447 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4448 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.integer", ptr undef }, ptr %4447, 1
// CHECK-NEXT:   %4449 = extractvalue { ptr, ptr } %4446, 1
// CHECK-NEXT:   %4450 = extractvalue { ptr, ptr } %4446, 0
// CHECK-NEXT:   %__llgo_funcval_code572 = call ptr asm "", "=r,0"(ptr %4450)
// CHECK-NEXT:   %4451 = call %reflect.Value %__llgo_funcval_code572(ptr {{(nest|swiftself)}} %4449, %"{{.*}}/runtime/internal/runtime.eface" %4448)
// CHECK-NEXT:   %4452 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4444, i32 0, i32 1
// CHECK-NEXT:   %4453 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4454 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %4455 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_int", ptr undef }, ptr %4454, 1
// CHECK-NEXT:   %4456 = extractvalue { ptr, ptr } %4453, 1
// CHECK-NEXT:   %4457 = extractvalue { ptr, ptr } %4453, 0
// CHECK-NEXT:   %__llgo_funcval_code573 = call ptr asm "", "=r,0"(ptr %4457)
// CHECK-NEXT:   %4458 = call %reflect.Value %__llgo_funcval_code573(ptr {{(nest|swiftself)}} %4456, %"{{.*}}/runtime/internal/runtime.eface" %4455)
// CHECK-NEXT:   store %reflect.Value %4451, ptr %4445, align 8
// CHECK-NEXT:   store %reflect.Value %4458, ptr %4452, align 8
// CHECK-NEXT:   %4459 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 287
// CHECK-NEXT:   %4460 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4459, i32 0, i32 0
// CHECK-NEXT:   %4461 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4462 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %main.Empty zeroinitializer, ptr %4462, align 1
// CHECK-NEXT:   %4463 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.Empty, ptr undef }, ptr %4462, 1
// CHECK-NEXT:   %4464 = extractvalue { ptr, ptr } %4461, 1
// CHECK-NEXT:   %4465 = extractvalue { ptr, ptr } %4461, 0
// CHECK-NEXT:   %__llgo_funcval_code574 = call ptr asm "", "=r,0"(ptr %4465)
// CHECK-NEXT:   %4466 = call %reflect.Value %__llgo_funcval_code574(ptr {{(nest|swiftself)}} %4464, %"{{.*}}/runtime/internal/runtime.eface" %4463)
// CHECK-NEXT:   %4467 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4459, i32 0, i32 1
// CHECK-NEXT:   %4468 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4469 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store {} zeroinitializer, ptr %4469, align 1
// CHECK-NEXT:   %4470 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr undef }, ptr %4469, 1
// CHECK-NEXT:   %4471 = extractvalue { ptr, ptr } %4468, 1
// CHECK-NEXT:   %4472 = extractvalue { ptr, ptr } %4468, 0
// CHECK-NEXT:   %__llgo_funcval_code575 = call ptr asm "", "=r,0"(ptr %4472)
// CHECK-NEXT:   %4473 = call %reflect.Value %__llgo_funcval_code575(ptr {{(nest|swiftself)}} %4471, %"{{.*}}/runtime/internal/runtime.eface" %4470)
// CHECK-NEXT:   store %reflect.Value %4466, ptr %4460, align 8
// CHECK-NEXT:   store %reflect.Value %4473, ptr %4467, align 8
// CHECK-NEXT:   %4474 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 288
// CHECK-NEXT:   %4475 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4474, i32 0, i32 0
// CHECK-NEXT:   %4476 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4477 = extractvalue { ptr, ptr } %4476, 1
// CHECK-NEXT:   %4478 = extractvalue { ptr, ptr } %4476, 0
// CHECK-NEXT:   %__llgo_funcval_code576 = call ptr asm "", "=r,0"(ptr %4478)
// CHECK-NEXT:   %4479 = call %reflect.Value %__llgo_funcval_code576(ptr {{(nest|swiftself)}} %4477, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.Empty", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   %4480 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4474, i32 0, i32 1
// CHECK-NEXT:   %4481 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4482 = extractvalue { ptr, ptr } %4481, 1
// CHECK-NEXT:   %4483 = extractvalue { ptr, ptr } %4481, 0
// CHECK-NEXT:   %__llgo_funcval_code577 = call ptr asm "", "=r,0"(ptr %4483)
// CHECK-NEXT:   %4484 = call %reflect.Value %__llgo_funcval_code577(ptr {{(nest|swiftself)}} %4482, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4479, ptr %4475, align 8
// CHECK-NEXT:   store %reflect.Value %4484, ptr %4480, align 8
// CHECK-NEXT:   %4485 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 289
// CHECK-NEXT:   %4486 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4485, i32 0, i32 0
// CHECK-NEXT:   %4487 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4488 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store {} zeroinitializer, ptr %4488, align 1
// CHECK-NEXT:   %4489 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr undef }, ptr %4488, 1
// CHECK-NEXT:   %4490 = extractvalue { ptr, ptr } %4487, 1
// CHECK-NEXT:   %4491 = extractvalue { ptr, ptr } %4487, 0
// CHECK-NEXT:   %__llgo_funcval_code578 = call ptr asm "", "=r,0"(ptr %4491)
// CHECK-NEXT:   %4492 = call %reflect.Value %__llgo_funcval_code578(ptr {{(nest|swiftself)}} %4490, %"{{.*}}/runtime/internal/runtime.eface" %4489)
// CHECK-NEXT:   %4493 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4485, i32 0, i32 1
// CHECK-NEXT:   %4494 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4495 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %main.Empty zeroinitializer, ptr %4495, align 1
// CHECK-NEXT:   %4496 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.Empty, ptr undef }, ptr %4495, 1
// CHECK-NEXT:   %4497 = extractvalue { ptr, ptr } %4494, 1
// CHECK-NEXT:   %4498 = extractvalue { ptr, ptr } %4494, 0
// CHECK-NEXT:   %__llgo_funcval_code579 = call ptr asm "", "=r,0"(ptr %4498)
// CHECK-NEXT:   %4499 = call %reflect.Value %__llgo_funcval_code579(ptr {{(nest|swiftself)}} %4497, %"{{.*}}/runtime/internal/runtime.eface" %4496)
// CHECK-NEXT:   store %reflect.Value %4492, ptr %4486, align 8
// CHECK-NEXT:   store %reflect.Value %4499, ptr %4493, align 8
// CHECK-NEXT:   %4500 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 290
// CHECK-NEXT:   %4501 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4500, i32 0, i32 0
// CHECK-NEXT:   %4502 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4503 = extractvalue { ptr, ptr } %4502, 1
// CHECK-NEXT:   %4504 = extractvalue { ptr, ptr } %4502, 0
// CHECK-NEXT:   %__llgo_funcval_code580 = call ptr asm "", "=r,0"(ptr %4504)
// CHECK-NEXT:   %4505 = call %reflect.Value %__llgo_funcval_code580(ptr {{(nest|swiftself)}} %4503, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_struct$n1H8J_3prDN3firMwPxBLVTkE5hJ9Di-AqNvaC9jczw", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   %4506 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4500, i32 0, i32 1
// CHECK-NEXT:   %4507 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4508 = extractvalue { ptr, ptr } %4507, 1
// CHECK-NEXT:   %4509 = extractvalue { ptr, ptr } %4507, 0
// CHECK-NEXT:   %__llgo_funcval_code581 = call ptr asm "", "=r,0"(ptr %4509)
// CHECK-NEXT:   %4510 = call %reflect.Value %__llgo_funcval_code581(ptr {{(nest|swiftself)}} %4508, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.Empty", ptr @"__llgo.moduleZeroSizedAlloc$" })
// CHECK-NEXT:   store %reflect.Value %4505, ptr %4501, align 8
// CHECK-NEXT:   store %reflect.Value %4510, ptr %4506, align 8
// CHECK-NEXT:   %4511 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 291
// CHECK-NEXT:   %4512 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4511, i32 0, i32 0
// CHECK-NEXT:   %4513 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4514 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %main.Empty zeroinitializer, ptr %4514, align 1
// CHECK-NEXT:   %4515 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.Empty, ptr undef }, ptr %4514, 1
// CHECK-NEXT:   %4516 = extractvalue { ptr, ptr } %4513, 1
// CHECK-NEXT:   %4517 = extractvalue { ptr, ptr } %4513, 0
// CHECK-NEXT:   %__llgo_funcval_code582 = call ptr asm "", "=r,0"(ptr %4517)
// CHECK-NEXT:   %4518 = call %reflect.Value %__llgo_funcval_code582(ptr {{(nest|swiftself)}} %4516, %"{{.*}}/runtime/internal/runtime.eface" %4515)
// CHECK-NEXT:   %4519 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4511, i32 0, i32 1
// CHECK-NEXT:   %4520 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4521 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store %main.Empty zeroinitializer, ptr %4521, align 1
// CHECK-NEXT:   %4522 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.Empty, ptr undef }, ptr %4521, 1
// CHECK-NEXT:   %4523 = extractvalue { ptr, ptr } %4520, 1
// CHECK-NEXT:   %4524 = extractvalue { ptr, ptr } %4520, 0
// CHECK-NEXT:   %__llgo_funcval_code583 = call ptr asm "", "=r,0"(ptr %4524)
// CHECK-NEXT:   %4525 = call %reflect.Value %__llgo_funcval_code583(ptr {{(nest|swiftself)}} %4523, %"{{.*}}/runtime/internal/runtime.eface" %4522)
// CHECK-NEXT:   store %reflect.Value %4518, ptr %4512, align 8
// CHECK-NEXT:   store %reflect.Value %4525, ptr %4519, align 8
// CHECK-NEXT:   %4526 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 292
// CHECK-NEXT:   %4527 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4526, i32 0, i32 0
// CHECK-NEXT:   %4528 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4529 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4529, align 8
// CHECK-NEXT:   %4530 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4529, 1
// CHECK-NEXT:   %4531 = extractvalue { ptr, ptr } %4528, 1
// CHECK-NEXT:   %4532 = extractvalue { ptr, ptr } %4528, 0
// CHECK-NEXT:   %__llgo_funcval_code584 = call ptr asm "", "=r,0"(ptr %4532)
// CHECK-NEXT:   %4533 = call %reflect.Value %__llgo_funcval_code584(ptr {{(nest|swiftself)}} %4531, %"{{.*}}/runtime/internal/runtime.eface" %4530)
// CHECK-NEXT:   %4534 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4526, i32 0, i32 1
// CHECK-NEXT:   %4535 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4536 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4536, align 8
// CHECK-NEXT:   %4537 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4536, 1
// CHECK-NEXT:   %4538 = extractvalue { ptr, ptr } %4535, 1
// CHECK-NEXT:   %4539 = extractvalue { ptr, ptr } %4535, 0
// CHECK-NEXT:   %__llgo_funcval_code585 = call ptr asm "", "=r,0"(ptr %4539)
// CHECK-NEXT:   %4540 = call %reflect.Value %__llgo_funcval_code585(ptr {{(nest|swiftself)}} %4538, %"{{.*}}/runtime/internal/runtime.eface" %4537)
// CHECK-NEXT:   store %reflect.Value %4533, ptr %4527, align 8
// CHECK-NEXT:   store %reflect.Value %4540, ptr %4534, align 8
// CHECK-NEXT:   %4541 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 293
// CHECK-NEXT:   %4542 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4541, i32 0, i32 0
// CHECK-NEXT:   %4543 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4544 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4544, align 8
// CHECK-NEXT:   %4545 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4544, 1
// CHECK-NEXT:   %4546 = extractvalue { ptr, ptr } %4543, 1
// CHECK-NEXT:   %4547 = extractvalue { ptr, ptr } %4543, 0
// CHECK-NEXT:   %__llgo_funcval_code586 = call ptr asm "", "=r,0"(ptr %4547)
// CHECK-NEXT:   %4548 = call %reflect.Value %__llgo_funcval_code586(ptr {{(nest|swiftself)}} %4546, %"{{.*}}/runtime/internal/runtime.eface" %4545)
// CHECK-NEXT:   %4549 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4541, i32 0, i32 1
// CHECK-NEXT:   %4550 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4551 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" { ptr @"__llgo.moduleZeroSizedAlloc$", i64 0, i64 0 }, ptr %4551, align 8
// CHECK-NEXT:   %4552 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytes, ptr undef }, ptr %4551, 1
// CHECK-NEXT:   %4553 = extractvalue { ptr, ptr } %4550, 1
// CHECK-NEXT:   %4554 = extractvalue { ptr, ptr } %4550, 0
// CHECK-NEXT:   %__llgo_funcval_code587 = call ptr asm "", "=r,0"(ptr %4554)
// CHECK-NEXT:   %4555 = call %reflect.Value %__llgo_funcval_code587(ptr {{(nest|swiftself)}} %4553, %"{{.*}}/runtime/internal/runtime.eface" %4552)
// CHECK-NEXT:   store %reflect.Value %4548, ptr %4542, align 8
// CHECK-NEXT:   store %reflect.Value %4555, ptr %4549, align 8
// CHECK-NEXT:   %4556 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 294
// CHECK-NEXT:   %4557 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4556, i32 0, i32 0
// CHECK-NEXT:   %4558 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4559 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } zeroinitializer, ptr %4559, align 8
// CHECK-NEXT:   %4560 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %4559, 1
// CHECK-NEXT:   %4561 = extractvalue { ptr, ptr } %4558, 1
// CHECK-NEXT:   %4562 = extractvalue { ptr, ptr } %4558, 0
// CHECK-NEXT:   %__llgo_funcval_code588 = call ptr asm "", "=r,0"(ptr %4562)
// CHECK-NEXT:   %4563 = call %reflect.Value %__llgo_funcval_code588(ptr {{(nest|swiftself)}} %4561, %"{{.*}}/runtime/internal/runtime.eface" %4560)
// CHECK-NEXT:   %4564 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4556, i32 0, i32 1
// CHECK-NEXT:   %4565 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4566 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %main.MyFunc zeroinitializer, ptr %4566, align 8
// CHECK-NEXT:   %4567 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyFunc, ptr undef }, ptr %4566, 1
// CHECK-NEXT:   %4568 = extractvalue { ptr, ptr } %4565, 1
// CHECK-NEXT:   %4569 = extractvalue { ptr, ptr } %4565, 0
// CHECK-NEXT:   %__llgo_funcval_code589 = call ptr asm "", "=r,0"(ptr %4569)
// CHECK-NEXT:   %4570 = call %reflect.Value %__llgo_funcval_code589(ptr {{(nest|swiftself)}} %4568, %"{{.*}}/runtime/internal/runtime.eface" %4567)
// CHECK-NEXT:   store %reflect.Value %4563, ptr %4557, align 8
// CHECK-NEXT:   store %reflect.Value %4570, ptr %4564, align 8
// CHECK-NEXT:   %4571 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 295
// CHECK-NEXT:   %4572 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4571, i32 0, i32 0
// CHECK-NEXT:   %4573 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4574 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %main.MyFunc zeroinitializer, ptr %4574, align 8
// CHECK-NEXT:   %4575 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyFunc, ptr undef }, ptr %4574, 1
// CHECK-NEXT:   %4576 = extractvalue { ptr, ptr } %4573, 1
// CHECK-NEXT:   %4577 = extractvalue { ptr, ptr } %4573, 0
// CHECK-NEXT:   %__llgo_funcval_code590 = call ptr asm "", "=r,0"(ptr %4577)
// CHECK-NEXT:   %4578 = call %reflect.Value %__llgo_funcval_code590(ptr {{(nest|swiftself)}} %4576, %"{{.*}}/runtime/internal/runtime.eface" %4575)
// CHECK-NEXT:   %4579 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4571, i32 0, i32 1
// CHECK-NEXT:   %4580 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4581 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store { ptr, ptr } zeroinitializer, ptr %4581, align 8
// CHECK-NEXT:   %4582 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_closure$b7Su1hWaFih-M0M9hMk6nO_RD1K_GQu5WjIXQp6Q2e8", ptr undef }, ptr %4581, 1
// CHECK-NEXT:   %4583 = extractvalue { ptr, ptr } %4580, 1
// CHECK-NEXT:   %4584 = extractvalue { ptr, ptr } %4580, 0
// CHECK-NEXT:   %__llgo_funcval_code591 = call ptr asm "", "=r,0"(ptr %4584)
// CHECK-NEXT:   %4585 = call %reflect.Value %__llgo_funcval_code591(ptr {{(nest|swiftself)}} %4583, %"{{.*}}/runtime/internal/runtime.eface" %4582)
// CHECK-NEXT:   store %reflect.Value %4578, ptr %4572, align 8
// CHECK-NEXT:   store %reflect.Value %4585, ptr %4579, align 8
// CHECK-NEXT:   %4586 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 296
// CHECK-NEXT:   %4587 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4586, i32 0, i32 0
// CHECK-NEXT:   %4588 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4589 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4589, align 8
// CHECK-NEXT:   %4590 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$qCYMduDqkoVspSDbQftiUv7WF_sipKSWG3oLt28TNlI", ptr undef }, ptr %4589, 1
// CHECK-NEXT:   %4591 = extractvalue { ptr, ptr } %4588, 1
// CHECK-NEXT:   %4592 = extractvalue { ptr, ptr } %4588, 0
// CHECK-NEXT:   %__llgo_funcval_code592 = call ptr asm "", "=r,0"(ptr %4592)
// CHECK-NEXT:   %4593 = call %reflect.Value %__llgo_funcval_code592(ptr {{(nest|swiftself)}} %4591, %"{{.*}}/runtime/internal/runtime.eface" %4590)
// CHECK-NEXT:   %4594 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4586, i32 0, i32 1
// CHECK-NEXT:   %4595 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4596 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4596, align 8
// CHECK-NEXT:   %4597 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$8xdZyDeCx3tqPdt33CJnJ-JMAObkKtPXr9OL-lBc3yU", ptr undef }, ptr %4596, 1
// CHECK-NEXT:   %4598 = extractvalue { ptr, ptr } %4595, 1
// CHECK-NEXT:   %4599 = extractvalue { ptr, ptr } %4595, 0
// CHECK-NEXT:   %__llgo_funcval_code593 = call ptr asm "", "=r,0"(ptr %4599)
// CHECK-NEXT:   %4600 = call %reflect.Value %__llgo_funcval_code593(ptr {{(nest|swiftself)}} %4598, %"{{.*}}/runtime/internal/runtime.eface" %4597)
// CHECK-NEXT:   store %reflect.Value %4593, ptr %4587, align 8
// CHECK-NEXT:   store %reflect.Value %4600, ptr %4594, align 8
// CHECK-NEXT:   %4601 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 297
// CHECK-NEXT:   %4602 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4601, i32 0, i32 0
// CHECK-NEXT:   %4603 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4604 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4604, align 8
// CHECK-NEXT:   %4605 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$8xdZyDeCx3tqPdt33CJnJ-JMAObkKtPXr9OL-lBc3yU", ptr undef }, ptr %4604, 1
// CHECK-NEXT:   %4606 = extractvalue { ptr, ptr } %4603, 1
// CHECK-NEXT:   %4607 = extractvalue { ptr, ptr } %4603, 0
// CHECK-NEXT:   %__llgo_funcval_code594 = call ptr asm "", "=r,0"(ptr %4607)
// CHECK-NEXT:   %4608 = call %reflect.Value %__llgo_funcval_code594(ptr {{(nest|swiftself)}} %4606, %"{{.*}}/runtime/internal/runtime.eface" %4605)
// CHECK-NEXT:   %4609 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4601, i32 0, i32 1
// CHECK-NEXT:   %4610 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4611 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4611, align 8
// CHECK-NEXT:   %4612 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$qCYMduDqkoVspSDbQftiUv7WF_sipKSWG3oLt28TNlI", ptr undef }, ptr %4611, 1
// CHECK-NEXT:   %4613 = extractvalue { ptr, ptr } %4610, 1
// CHECK-NEXT:   %4614 = extractvalue { ptr, ptr } %4610, 0
// CHECK-NEXT:   %__llgo_funcval_code595 = call ptr asm "", "=r,0"(ptr %4614)
// CHECK-NEXT:   %4615 = call %reflect.Value %__llgo_funcval_code595(ptr {{(nest|swiftself)}} %4613, %"{{.*}}/runtime/internal/runtime.eface" %4612)
// CHECK-NEXT:   store %reflect.Value %4608, ptr %4602, align 8
// CHECK-NEXT:   store %reflect.Value %4615, ptr %4609, align 8
// CHECK-NEXT:   %4616 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 298
// CHECK-NEXT:   %4617 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4616, i32 0, i32 0
// CHECK-NEXT:   %4618 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4619 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct zeroinitializer, ptr %4619, align 8
// CHECK-NEXT:   %4620 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct, ptr undef }, ptr %4619, 1
// CHECK-NEXT:   %4621 = extractvalue { ptr, ptr } %4618, 1
// CHECK-NEXT:   %4622 = extractvalue { ptr, ptr } %4618, 0
// CHECK-NEXT:   %__llgo_funcval_code596 = call ptr asm "", "=r,0"(ptr %4622)
// CHECK-NEXT:   %4623 = call %reflect.Value %__llgo_funcval_code596(ptr {{(nest|swiftself)}} %4621, %"{{.*}}/runtime/internal/runtime.eface" %4620)
// CHECK-NEXT:   %4624 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4616, i32 0, i32 1
// CHECK-NEXT:   %4625 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4626 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4626, align 8
// CHECK-NEXT:   %4627 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$qCYMduDqkoVspSDbQftiUv7WF_sipKSWG3oLt28TNlI", ptr undef }, ptr %4626, 1
// CHECK-NEXT:   %4628 = extractvalue { ptr, ptr } %4625, 1
// CHECK-NEXT:   %4629 = extractvalue { ptr, ptr } %4625, 0
// CHECK-NEXT:   %__llgo_funcval_code597 = call ptr asm "", "=r,0"(ptr %4629)
// CHECK-NEXT:   %4630 = call %reflect.Value %__llgo_funcval_code597(ptr {{(nest|swiftself)}} %4628, %"{{.*}}/runtime/internal/runtime.eface" %4627)
// CHECK-NEXT:   store %reflect.Value %4623, ptr %4617, align 8
// CHECK-NEXT:   store %reflect.Value %4630, ptr %4624, align 8
// CHECK-NEXT:   %4631 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 299
// CHECK-NEXT:   %4632 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4631, i32 0, i32 0
// CHECK-NEXT:   %4633 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4634 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4634, align 8
// CHECK-NEXT:   %4635 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$qCYMduDqkoVspSDbQftiUv7WF_sipKSWG3oLt28TNlI", ptr undef }, ptr %4634, 1
// CHECK-NEXT:   %4636 = extractvalue { ptr, ptr } %4633, 1
// CHECK-NEXT:   %4637 = extractvalue { ptr, ptr } %4633, 0
// CHECK-NEXT:   %__llgo_funcval_code598 = call ptr asm "", "=r,0"(ptr %4637)
// CHECK-NEXT:   %4638 = call %reflect.Value %__llgo_funcval_code598(ptr {{(nest|swiftself)}} %4636, %"{{.*}}/runtime/internal/runtime.eface" %4635)
// CHECK-NEXT:   %4639 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4631, i32 0, i32 1
// CHECK-NEXT:   %4640 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4641 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct zeroinitializer, ptr %4641, align 8
// CHECK-NEXT:   %4642 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct, ptr undef }, ptr %4641, 1
// CHECK-NEXT:   %4643 = extractvalue { ptr, ptr } %4640, 1
// CHECK-NEXT:   %4644 = extractvalue { ptr, ptr } %4640, 0
// CHECK-NEXT:   %__llgo_funcval_code599 = call ptr asm "", "=r,0"(ptr %4644)
// CHECK-NEXT:   %4645 = call %reflect.Value %__llgo_funcval_code599(ptr {{(nest|swiftself)}} %4643, %"{{.*}}/runtime/internal/runtime.eface" %4642)
// CHECK-NEXT:   store %reflect.Value %4638, ptr %4632, align 8
// CHECK-NEXT:   store %reflect.Value %4645, ptr %4639, align 8
// CHECK-NEXT:   %4646 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 300
// CHECK-NEXT:   %4647 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4646, i32 0, i32 0
// CHECK-NEXT:   %4648 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4649 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct zeroinitializer, ptr %4649, align 8
// CHECK-NEXT:   %4650 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct, ptr undef }, ptr %4649, 1
// CHECK-NEXT:   %4651 = extractvalue { ptr, ptr } %4648, 1
// CHECK-NEXT:   %4652 = extractvalue { ptr, ptr } %4648, 0
// CHECK-NEXT:   %__llgo_funcval_code600 = call ptr asm "", "=r,0"(ptr %4652)
// CHECK-NEXT:   %4653 = call %reflect.Value %__llgo_funcval_code600(ptr {{(nest|swiftself)}} %4651, %"{{.*}}/runtime/internal/runtime.eface" %4650)
// CHECK-NEXT:   %4654 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4646, i32 0, i32 1
// CHECK-NEXT:   %4655 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4656 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4656, align 8
// CHECK-NEXT:   %4657 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$8xdZyDeCx3tqPdt33CJnJ-JMAObkKtPXr9OL-lBc3yU", ptr undef }, ptr %4656, 1
// CHECK-NEXT:   %4658 = extractvalue { ptr, ptr } %4655, 1
// CHECK-NEXT:   %4659 = extractvalue { ptr, ptr } %4655, 0
// CHECK-NEXT:   %__llgo_funcval_code601 = call ptr asm "", "=r,0"(ptr %4659)
// CHECK-NEXT:   %4660 = call %reflect.Value %__llgo_funcval_code601(ptr {{(nest|swiftself)}} %4658, %"{{.*}}/runtime/internal/runtime.eface" %4657)
// CHECK-NEXT:   store %reflect.Value %4653, ptr %4647, align 8
// CHECK-NEXT:   store %reflect.Value %4660, ptr %4654, align 8
// CHECK-NEXT:   %4661 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 301
// CHECK-NEXT:   %4662 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4661, i32 0, i32 0
// CHECK-NEXT:   %4663 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4664 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } zeroinitializer, ptr %4664, align 8
// CHECK-NEXT:   %4665 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testgo/reflectconv.struct$8xdZyDeCx3tqPdt33CJnJ-JMAObkKtPXr9OL-lBc3yU", ptr undef }, ptr %4664, 1
// CHECK-NEXT:   %4666 = extractvalue { ptr, ptr } %4663, 1
// CHECK-NEXT:   %4667 = extractvalue { ptr, ptr } %4663, 0
// CHECK-NEXT:   %__llgo_funcval_code602 = call ptr asm "", "=r,0"(ptr %4667)
// CHECK-NEXT:   %4668 = call %reflect.Value %__llgo_funcval_code602(ptr {{(nest|swiftself)}} %4666, %"{{.*}}/runtime/internal/runtime.eface" %4665)
// CHECK-NEXT:   %4669 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4661, i32 0, i32 1
// CHECK-NEXT:   %4670 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4671 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct zeroinitializer, ptr %4671, align 8
// CHECK-NEXT:   %4672 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct, ptr undef }, ptr %4671, 1
// CHECK-NEXT:   %4673 = extractvalue { ptr, ptr } %4670, 1
// CHECK-NEXT:   %4674 = extractvalue { ptr, ptr } %4670, 0
// CHECK-NEXT:   %__llgo_funcval_code603 = call ptr asm "", "=r,0"(ptr %4674)
// CHECK-NEXT:   %4675 = call %reflect.Value %__llgo_funcval_code603(ptr {{(nest|swiftself)}} %4673, %"{{.*}}/runtime/internal/runtime.eface" %4672)
// CHECK-NEXT:   store %reflect.Value %4668, ptr %4662, align 8
// CHECK-NEXT:   store %reflect.Value %4675, ptr %4669, align 8
// CHECK-NEXT:   %4676 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 302
// CHECK-NEXT:   %4677 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4676, i32 0, i32 0
// CHECK-NEXT:   %4678 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4679 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct1 zeroinitializer, ptr %4679, align 8
// CHECK-NEXT:   %4680 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct1, ptr undef }, ptr %4679, 1
// CHECK-NEXT:   %4681 = extractvalue { ptr, ptr } %4678, 1
// CHECK-NEXT:   %4682 = extractvalue { ptr, ptr } %4678, 0
// CHECK-NEXT:   %__llgo_funcval_code604 = call ptr asm "", "=r,0"(ptr %4682)
// CHECK-NEXT:   %4683 = call %reflect.Value %__llgo_funcval_code604(ptr {{(nest|swiftself)}} %4681, %"{{.*}}/runtime/internal/runtime.eface" %4680)
// CHECK-NEXT:   %4684 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4676, i32 0, i32 1
// CHECK-NEXT:   %4685 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4686 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct2 zeroinitializer, ptr %4686, align 8
// CHECK-NEXT:   %4687 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct2, ptr undef }, ptr %4686, 1
// CHECK-NEXT:   %4688 = extractvalue { ptr, ptr } %4685, 1
// CHECK-NEXT:   %4689 = extractvalue { ptr, ptr } %4685, 0
// CHECK-NEXT:   %__llgo_funcval_code605 = call ptr asm "", "=r,0"(ptr %4689)
// CHECK-NEXT:   %4690 = call %reflect.Value %__llgo_funcval_code605(ptr {{(nest|swiftself)}} %4688, %"{{.*}}/runtime/internal/runtime.eface" %4687)
// CHECK-NEXT:   store %reflect.Value %4683, ptr %4677, align 8
// CHECK-NEXT:   store %reflect.Value %4690, ptr %4684, align 8
// CHECK-NEXT:   %4691 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 303
// CHECK-NEXT:   %4692 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4691, i32 0, i32 0
// CHECK-NEXT:   %4693 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4694 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct2 zeroinitializer, ptr %4694, align 8
// CHECK-NEXT:   %4695 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct2, ptr undef }, ptr %4694, 1
// CHECK-NEXT:   %4696 = extractvalue { ptr, ptr } %4693, 1
// CHECK-NEXT:   %4697 = extractvalue { ptr, ptr } %4693, 0
// CHECK-NEXT:   %__llgo_funcval_code606 = call ptr asm "", "=r,0"(ptr %4697)
// CHECK-NEXT:   %4698 = call %reflect.Value %__llgo_funcval_code606(ptr {{(nest|swiftself)}} %4696, %"{{.*}}/runtime/internal/runtime.eface" %4695)
// CHECK-NEXT:   %4699 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4691, i32 0, i32 1
// CHECK-NEXT:   %4700 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4701 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store %main.MyStruct1 zeroinitializer, ptr %4701, align 8
// CHECK-NEXT:   %4702 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyStruct1, ptr undef }, ptr %4701, 1
// CHECK-NEXT:   %4703 = extractvalue { ptr, ptr } %4700, 1
// CHECK-NEXT:   %4704 = extractvalue { ptr, ptr } %4700, 0
// CHECK-NEXT:   %__llgo_funcval_code607 = call ptr asm "", "=r,0"(ptr %4704)
// CHECK-NEXT:   %4705 = call %reflect.Value %__llgo_funcval_code607(ptr {{(nest|swiftself)}} %4703, %"{{.*}}/runtime/internal/runtime.eface" %4702)
// CHECK-NEXT:   store %reflect.Value %4698, ptr %4692, align 8
// CHECK-NEXT:   store %reflect.Value %4705, ptr %4699, align 8
// CHECK-NEXT:   %4706 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 304
// CHECK-NEXT:   %4707 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4706, i32 0, i32 0
// CHECK-NEXT:   %4708 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4709 = extractvalue { ptr, ptr } %4708, 1
// CHECK-NEXT:   %4710 = extractvalue { ptr, ptr } %4708, 0
// CHECK-NEXT:   %__llgo_funcval_code608 = call ptr asm "", "=r,0"(ptr %4710)
// CHECK-NEXT:   %4711 = call %reflect.Value %__llgo_funcval_code608(ptr {{(nest|swiftself)}} %4709, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_uint8", ptr null })
// CHECK-NEXT:   %4712 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4706, i32 0, i32 1
// CHECK-NEXT:   %4713 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4714 = extractvalue { ptr, ptr } %4713, 1
// CHECK-NEXT:   %4715 = extractvalue { ptr, ptr } %4713, 0
// CHECK-NEXT:   %__llgo_funcval_code609 = call ptr asm "", "=r,0"(ptr %4715)
// CHECK-NEXT:   %4716 = call %reflect.Value %__llgo_funcval_code609(ptr {{(nest|swiftself)}} %4714, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   store %reflect.Value %4711, ptr %4707, align 8
// CHECK-NEXT:   store %reflect.Value %4716, ptr %4712, align 8
// CHECK-NEXT:   %4717 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 305
// CHECK-NEXT:   %4718 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4717, i32 0, i32 0
// CHECK-NEXT:   %4719 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4720 = extractvalue { ptr, ptr } %4719, 1
// CHECK-NEXT:   %4721 = extractvalue { ptr, ptr } %4719, 0
// CHECK-NEXT:   %__llgo_funcval_code610 = call ptr asm "", "=r,0"(ptr %4721)
// CHECK-NEXT:   %4722 = call %reflect.Value %__llgo_funcval_code610(ptr {{(nest|swiftself)}} %4720, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   %4723 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4717, i32 0, i32 1
// CHECK-NEXT:   %4724 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4725 = extractvalue { ptr, ptr } %4724, 1
// CHECK-NEXT:   %4726 = extractvalue { ptr, ptr } %4724, 0
// CHECK-NEXT:   %__llgo_funcval_code611 = call ptr asm "", "=r,0"(ptr %4726)
// CHECK-NEXT:   %4727 = call %reflect.Value %__llgo_funcval_code611(ptr {{(nest|swiftself)}} %4725, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4722, ptr %4718, align 8
// CHECK-NEXT:   store %reflect.Value %4727, ptr %4723, align 8
// CHECK-NEXT:   %4728 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 306
// CHECK-NEXT:   %4729 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4728, i32 0, i32 0
// CHECK-NEXT:   %4730 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4731 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4731, align 1
// CHECK-NEXT:   %4732 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %4731, 1
// CHECK-NEXT:   %4733 = extractvalue { ptr, ptr } %4730, 1
// CHECK-NEXT:   %4734 = extractvalue { ptr, ptr } %4730, 0
// CHECK-NEXT:   %__llgo_funcval_code612 = call ptr asm "", "=r,0"(ptr %4734)
// CHECK-NEXT:   %4735 = call %reflect.Value %__llgo_funcval_code612(ptr {{(nest|swiftself)}} %4733, %"{{.*}}/runtime/internal/runtime.eface" %4732)
// CHECK-NEXT:   %4736 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4728, i32 0, i32 1
// CHECK-NEXT:   %4737 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4738 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4738, align 1
// CHECK-NEXT:   %4739 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %4738, 1
// CHECK-NEXT:   %4740 = extractvalue { ptr, ptr } %4737, 1
// CHECK-NEXT:   %4741 = extractvalue { ptr, ptr } %4737, 0
// CHECK-NEXT:   %__llgo_funcval_code613 = call ptr asm "", "=r,0"(ptr %4741)
// CHECK-NEXT:   %4742 = call %reflect.Value %__llgo_funcval_code613(ptr {{(nest|swiftself)}} %4740, %"{{.*}}/runtime/internal/runtime.eface" %4739)
// CHECK-NEXT:   store %reflect.Value %4735, ptr %4729, align 8
// CHECK-NEXT:   store %reflect.Value %4742, ptr %4736, align 8
// CHECK-NEXT:   %4743 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 307
// CHECK-NEXT:   %4744 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4743, i32 0, i32 0
// CHECK-NEXT:   %4745 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4746 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 3)
// CHECK-NEXT:   store [3 x i8] zeroinitializer, ptr %4746, align 1
// CHECK-NEXT:   %4747 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[3]_llgo_uint8", ptr undef }, ptr %4746, 1
// CHECK-NEXT:   %4748 = extractvalue { ptr, ptr } %4745, 1
// CHECK-NEXT:   %4749 = extractvalue { ptr, ptr } %4745, 0
// CHECK-NEXT:   %__llgo_funcval_code614 = call ptr asm "", "=r,0"(ptr %4749)
// CHECK-NEXT:   %4750 = call %reflect.Value %__llgo_funcval_code614(ptr {{(nest|swiftself)}} %4748, %"{{.*}}/runtime/internal/runtime.eface" %4747)
// CHECK-NEXT:   %4751 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4743, i32 0, i32 1
// CHECK-NEXT:   %4752 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4753 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 3)
// CHECK-NEXT:   store [3 x i8] zeroinitializer, ptr %4753, align 1
// CHECK-NEXT:   %4754 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[3]_llgo_uint8", ptr undef }, ptr %4753, 1
// CHECK-NEXT:   %4755 = extractvalue { ptr, ptr } %4752, 1
// CHECK-NEXT:   %4756 = extractvalue { ptr, ptr } %4752, 0
// CHECK-NEXT:   %__llgo_funcval_code615 = call ptr asm "", "=r,0"(ptr %4756)
// CHECK-NEXT:   %4757 = call %reflect.Value %__llgo_funcval_code615(ptr {{(nest|swiftself)}} %4755, %"{{.*}}/runtime/internal/runtime.eface" %4754)
// CHECK-NEXT:   store %reflect.Value %4750, ptr %4744, align 8
// CHECK-NEXT:   store %reflect.Value %4757, ptr %4751, align 8
// CHECK-NEXT:   %4758 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 308
// CHECK-NEXT:   %4759 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4758, i32 0, i32 0
// CHECK-NEXT:   %4760 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4761 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4761, align 1
// CHECK-NEXT:   %4762 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray0, ptr undef }, ptr %4761, 1
// CHECK-NEXT:   %4763 = extractvalue { ptr, ptr } %4760, 1
// CHECK-NEXT:   %4764 = extractvalue { ptr, ptr } %4760, 0
// CHECK-NEXT:   %__llgo_funcval_code616 = call ptr asm "", "=r,0"(ptr %4764)
// CHECK-NEXT:   %4765 = call %reflect.Value %__llgo_funcval_code616(ptr {{(nest|swiftself)}} %4763, %"{{.*}}/runtime/internal/runtime.eface" %4762)
// CHECK-NEXT:   %4766 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4758, i32 0, i32 1
// CHECK-NEXT:   %4767 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4768 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4768, align 1
// CHECK-NEXT:   %4769 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %4768, 1
// CHECK-NEXT:   %4770 = extractvalue { ptr, ptr } %4767, 1
// CHECK-NEXT:   %4771 = extractvalue { ptr, ptr } %4767, 0
// CHECK-NEXT:   %__llgo_funcval_code617 = call ptr asm "", "=r,0"(ptr %4771)
// CHECK-NEXT:   %4772 = call %reflect.Value %__llgo_funcval_code617(ptr {{(nest|swiftself)}} %4770, %"{{.*}}/runtime/internal/runtime.eface" %4769)
// CHECK-NEXT:   store %reflect.Value %4765, ptr %4759, align 8
// CHECK-NEXT:   store %reflect.Value %4772, ptr %4766, align 8
// CHECK-NEXT:   %4773 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 309
// CHECK-NEXT:   %4774 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4773, i32 0, i32 0
// CHECK-NEXT:   %4775 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4776 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4776, align 1
// CHECK-NEXT:   %4777 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[0]_llgo_uint8", ptr undef }, ptr %4776, 1
// CHECK-NEXT:   %4778 = extractvalue { ptr, ptr } %4775, 1
// CHECK-NEXT:   %4779 = extractvalue { ptr, ptr } %4775, 0
// CHECK-NEXT:   %__llgo_funcval_code618 = call ptr asm "", "=r,0"(ptr %4779)
// CHECK-NEXT:   %4780 = call %reflect.Value %__llgo_funcval_code618(ptr {{(nest|swiftself)}} %4778, %"{{.*}}/runtime/internal/runtime.eface" %4777)
// CHECK-NEXT:   %4781 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4773, i32 0, i32 1
// CHECK-NEXT:   %4782 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4783 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 0)
// CHECK-NEXT:   store [0 x i8] zeroinitializer, ptr %4783, align 1
// CHECK-NEXT:   %4784 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.MyBytesArray0, ptr undef }, ptr %4783, 1
// CHECK-NEXT:   %4785 = extractvalue { ptr, ptr } %4782, 1
// CHECK-NEXT:   %4786 = extractvalue { ptr, ptr } %4782, 0
// CHECK-NEXT:   %__llgo_funcval_code619 = call ptr asm "", "=r,0"(ptr %4786)
// CHECK-NEXT:   %4787 = call %reflect.Value %__llgo_funcval_code619(ptr {{(nest|swiftself)}} %4785, %"{{.*}}/runtime/internal/runtime.eface" %4784)
// CHECK-NEXT:   store %reflect.Value %4780, ptr %4774, align 8
// CHECK-NEXT:   store %reflect.Value %4787, ptr %4781, align 8
// CHECK-NEXT:   %4788 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 310
// CHECK-NEXT:   %4789 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4788, i32 0, i32 0
// CHECK-NEXT:   %4790 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4791 = extractvalue { ptr, ptr } %4790, 1
// CHECK-NEXT:   %4792 = extractvalue { ptr, ptr } %4790, 0
// CHECK-NEXT:   %__llgo_funcval_code620 = call ptr asm "", "=r,0"(ptr %4792)
// CHECK-NEXT:   %4793 = call %reflect.Value %__llgo_funcval_code620(ptr {{(nest|swiftself)}} %4791, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"**_llgo_uint8", ptr null })
// CHECK-NEXT:   %4794 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4788, i32 0, i32 1
// CHECK-NEXT:   %4795 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4796 = extractvalue { ptr, ptr } %4795, 1
// CHECK-NEXT:   %4797 = extractvalue { ptr, ptr } %4795, 0
// CHECK-NEXT:   %__llgo_funcval_code621 = call ptr asm "", "=r,0"(ptr %4797)
// CHECK-NEXT:   %4798 = call %reflect.Value %__llgo_funcval_code621(ptr {{(nest|swiftself)}} %4796, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"**_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4793, ptr %4789, align 8
// CHECK-NEXT:   store %reflect.Value %4798, ptr %4794, align 8
// CHECK-NEXT:   %4799 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 311
// CHECK-NEXT:   %4800 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4799, i32 0, i32 0
// CHECK-NEXT:   %4801 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4802 = extractvalue { ptr, ptr } %4801, 1
// CHECK-NEXT:   %4803 = extractvalue { ptr, ptr } %4801, 0
// CHECK-NEXT:   %__llgo_funcval_code622 = call ptr asm "", "=r,0"(ptr %4803)
// CHECK-NEXT:   %4804 = call %reflect.Value %__llgo_funcval_code622(ptr {{(nest|swiftself)}} %4802, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"**_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   %4805 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4799, i32 0, i32 1
// CHECK-NEXT:   %4806 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4807 = extractvalue { ptr, ptr } %4806, 1
// CHECK-NEXT:   %4808 = extractvalue { ptr, ptr } %4806, 0
// CHECK-NEXT:   %__llgo_funcval_code623 = call ptr asm "", "=r,0"(ptr %4808)
// CHECK-NEXT:   %4809 = call %reflect.Value %__llgo_funcval_code623(ptr {{(nest|swiftself)}} %4807, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"**_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   store %reflect.Value %4804, ptr %4800, align 8
// CHECK-NEXT:   store %reflect.Value %4809, ptr %4805, align 8
// CHECK-NEXT:   %4810 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 312
// CHECK-NEXT:   %4811 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4810, i32 0, i32 0
// CHECK-NEXT:   %4812 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4813 = extractvalue { ptr, ptr } %4812, 1
// CHECK-NEXT:   %4814 = extractvalue { ptr, ptr } %4812, 0
// CHECK-NEXT:   %__llgo_funcval_code624 = call ptr asm "", "=r,0"(ptr %4814)
// CHECK-NEXT:   %4815 = call %reflect.Value %__llgo_funcval_code624(ptr {{(nest|swiftself)}} %4813, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_uint8", ptr null })
// CHECK-NEXT:   %4816 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4810, i32 0, i32 1
// CHECK-NEXT:   %4817 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4818 = extractvalue { ptr, ptr } %4817, 1
// CHECK-NEXT:   %4819 = extractvalue { ptr, ptr } %4817, 0
// CHECK-NEXT:   %__llgo_funcval_code625 = call ptr asm "", "=r,0"(ptr %4819)
// CHECK-NEXT:   %4820 = call %reflect.Value %__llgo_funcval_code625(ptr {{(nest|swiftself)}} %4818, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4815, ptr %4811, align 8
// CHECK-NEXT:   store %reflect.Value %4820, ptr %4816, align 8
// CHECK-NEXT:   %4821 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 313
// CHECK-NEXT:   %4822 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4821, i32 0, i32 0
// CHECK-NEXT:   %4823 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4824 = extractvalue { ptr, ptr } %4823, 1
// CHECK-NEXT:   %4825 = extractvalue { ptr, ptr } %4823, 0
// CHECK-NEXT:   %__llgo_funcval_code626 = call ptr asm "", "=r,0"(ptr %4825)
// CHECK-NEXT:   %4826 = call %reflect.Value %__llgo_funcval_code626(ptr {{(nest|swiftself)}} %4824, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_main.MyByte", ptr null })
// CHECK-NEXT:   %4827 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4821, i32 0, i32 1
// CHECK-NEXT:   %4828 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4829 = extractvalue { ptr, ptr } %4828, 1
// CHECK-NEXT:   %4830 = extractvalue { ptr, ptr } %4828, 0
// CHECK-NEXT:   %__llgo_funcval_code627 = call ptr asm "", "=r,0"(ptr %4830)
// CHECK-NEXT:   %4831 = call %reflect.Value %__llgo_funcval_code627(ptr {{(nest|swiftself)}} %4829, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_main.MyByte", ptr null })
// CHECK-NEXT:   store %reflect.Value %4826, ptr %4822, align 8
// CHECK-NEXT:   store %reflect.Value %4831, ptr %4827, align 8
// CHECK-NEXT:   %4832 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 314
// CHECK-NEXT:   %4833 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4832, i32 0, i32 0
// CHECK-NEXT:   %4834 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4835 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4835, align 8
// CHECK-NEXT:   %4836 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4835, 1
// CHECK-NEXT:   %4837 = extractvalue { ptr, ptr } %4834, 1
// CHECK-NEXT:   %4838 = extractvalue { ptr, ptr } %4834, 0
// CHECK-NEXT:   %__llgo_funcval_code628 = call ptr asm "", "=r,0"(ptr %4838)
// CHECK-NEXT:   %4839 = call %reflect.Value %__llgo_funcval_code628(ptr {{(nest|swiftself)}} %4837, %"{{.*}}/runtime/internal/runtime.eface" %4836)
// CHECK-NEXT:   %4840 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4832, i32 0, i32 1
// CHECK-NEXT:   %4841 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4842 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4842, align 8
// CHECK-NEXT:   %4843 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint8", ptr undef }, ptr %4842, 1
// CHECK-NEXT:   %4844 = extractvalue { ptr, ptr } %4841, 1
// CHECK-NEXT:   %4845 = extractvalue { ptr, ptr } %4841, 0
// CHECK-NEXT:   %__llgo_funcval_code629 = call ptr asm "", "=r,0"(ptr %4845)
// CHECK-NEXT:   %4846 = call %reflect.Value %__llgo_funcval_code629(ptr {{(nest|swiftself)}} %4844, %"{{.*}}/runtime/internal/runtime.eface" %4843)
// CHECK-NEXT:   store %reflect.Value %4839, ptr %4833, align 8
// CHECK-NEXT:   store %reflect.Value %4846, ptr %4840, align 8
// CHECK-NEXT:   %4847 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 315
// CHECK-NEXT:   %4848 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4847, i32 0, i32 0
// CHECK-NEXT:   %4849 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4850 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4850, align 8
// CHECK-NEXT:   %4851 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_main.MyByte", ptr undef }, ptr %4850, 1
// CHECK-NEXT:   %4852 = extractvalue { ptr, ptr } %4849, 1
// CHECK-NEXT:   %4853 = extractvalue { ptr, ptr } %4849, 0
// CHECK-NEXT:   %__llgo_funcval_code630 = call ptr asm "", "=r,0"(ptr %4853)
// CHECK-NEXT:   %4854 = call %reflect.Value %__llgo_funcval_code630(ptr {{(nest|swiftself)}} %4852, %"{{.*}}/runtime/internal/runtime.eface" %4851)
// CHECK-NEXT:   %4855 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4847, i32 0, i32 1
// CHECK-NEXT:   %4856 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4857 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %4857, align 8
// CHECK-NEXT:   %4858 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_main.MyByte", ptr undef }, ptr %4857, 1
// CHECK-NEXT:   %4859 = extractvalue { ptr, ptr } %4856, 1
// CHECK-NEXT:   %4860 = extractvalue { ptr, ptr } %4856, 0
// CHECK-NEXT:   %__llgo_funcval_code631 = call ptr asm "", "=r,0"(ptr %4860)
// CHECK-NEXT:   %4861 = call %reflect.Value %__llgo_funcval_code631(ptr {{(nest|swiftself)}} %4859, %"{{.*}}/runtime/internal/runtime.eface" %4858)
// CHECK-NEXT:   store %reflect.Value %4854, ptr %4848, align 8
// CHECK-NEXT:   store %reflect.Value %4861, ptr %4855, align 8
// CHECK-NEXT:   %4862 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 316
// CHECK-NEXT:   %4863 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4862, i32 0, i32 0
// CHECK-NEXT:   %4864 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4865 = extractvalue { ptr, ptr } %4864, 1
// CHECK-NEXT:   %4866 = extractvalue { ptr, ptr } %4864, 0
// CHECK-NEXT:   %__llgo_funcval_code632 = call ptr asm "", "=r,0"(ptr %4866)
// CHECK-NEXT:   %4867 = call %reflect.Value %__llgo_funcval_code632(ptr {{(nest|swiftself)}} %4865, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_uint8", ptr null })
// CHECK-NEXT:   %4868 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4862, i32 0, i32 1
// CHECK-NEXT:   %4869 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4870 = extractvalue { ptr, ptr } %4869, 1
// CHECK-NEXT:   %4871 = extractvalue { ptr, ptr } %4869, 0
// CHECK-NEXT:   %__llgo_funcval_code633 = call ptr asm "", "=r,0"(ptr %4871)
// CHECK-NEXT:   %4872 = call %reflect.Value %__llgo_funcval_code633(ptr {{(nest|swiftself)}} %4870, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4867, ptr %4863, align 8
// CHECK-NEXT:   store %reflect.Value %4872, ptr %4868, align 8
// CHECK-NEXT:   %4873 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 317
// CHECK-NEXT:   %4874 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4873, i32 0, i32 0
// CHECK-NEXT:   %4875 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4876 = extractvalue { ptr, ptr } %4875, 1
// CHECK-NEXT:   %4877 = extractvalue { ptr, ptr } %4875, 0
// CHECK-NEXT:   %__llgo_funcval_code634 = call ptr asm "", "=r,0"(ptr %4877)
// CHECK-NEXT:   %4878 = call %reflect.Value %__llgo_funcval_code634(ptr {{(nest|swiftself)}} %4876, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   %4879 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4873, i32 0, i32 1
// CHECK-NEXT:   %4880 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4881 = extractvalue { ptr, ptr } %4880, 1
// CHECK-NEXT:   %4882 = extractvalue { ptr, ptr } %4880, 0
// CHECK-NEXT:   %__llgo_funcval_code635 = call ptr asm "", "=r,0"(ptr %4882)
// CHECK-NEXT:   %4883 = call %reflect.Value %__llgo_funcval_code635(ptr {{(nest|swiftself)}} %4881, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_main.MyByte", ptr null })
// CHECK-NEXT:   store %reflect.Value %4878, ptr %4874, align 8
// CHECK-NEXT:   store %reflect.Value %4883, ptr %4879, align 8
// CHECK-NEXT:   %4884 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 318
// CHECK-NEXT:   %4885 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4884, i32 0, i32 0
// CHECK-NEXT:   %4886 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4887 = extractvalue { ptr, ptr } %4886, 1
// CHECK-NEXT:   %4888 = extractvalue { ptr, ptr } %4886, 0
// CHECK-NEXT:   %__llgo_funcval_code636 = call ptr asm "", "=r,0"(ptr %4888)
// CHECK-NEXT:   %4889 = call %reflect.Value %__llgo_funcval_code636(ptr {{(nest|swiftself)}} %4887, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_uint8]_llgo_int", ptr null })
// CHECK-NEXT:   %4890 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4884, i32 0, i32 1
// CHECK-NEXT:   %4891 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4892 = extractvalue { ptr, ptr } %4891, 1
// CHECK-NEXT:   %4893 = extractvalue { ptr, ptr } %4891, 0
// CHECK-NEXT:   %__llgo_funcval_code637 = call ptr asm "", "=r,0"(ptr %4893)
// CHECK-NEXT:   %4894 = call %reflect.Value %__llgo_funcval_code637(ptr {{(nest|swiftself)}} %4892, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_uint8]_llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %4889, ptr %4885, align 8
// CHECK-NEXT:   store %reflect.Value %4894, ptr %4890, align 8
// CHECK-NEXT:   %4895 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 319
// CHECK-NEXT:   %4896 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4895, i32 0, i32 0
// CHECK-NEXT:   %4897 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4898 = extractvalue { ptr, ptr } %4897, 1
// CHECK-NEXT:   %4899 = extractvalue { ptr, ptr } %4897, 0
// CHECK-NEXT:   %__llgo_funcval_code638 = call ptr asm "", "=r,0"(ptr %4899)
// CHECK-NEXT:   %4900 = call %reflect.Value %__llgo_funcval_code638(ptr {{(nest|swiftself)}} %4898, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_main.MyByte]_llgo_int", ptr null })
// CHECK-NEXT:   %4901 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4895, i32 0, i32 1
// CHECK-NEXT:   %4902 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4903 = extractvalue { ptr, ptr } %4902, 1
// CHECK-NEXT:   %4904 = extractvalue { ptr, ptr } %4902, 0
// CHECK-NEXT:   %__llgo_funcval_code639 = call ptr asm "", "=r,0"(ptr %4904)
// CHECK-NEXT:   %4905 = call %reflect.Value %__llgo_funcval_code639(ptr {{(nest|swiftself)}} %4903, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_main.MyByte]_llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %4900, ptr %4896, align 8
// CHECK-NEXT:   store %reflect.Value %4905, ptr %4901, align 8
// CHECK-NEXT:   %4906 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 320
// CHECK-NEXT:   %4907 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4906, i32 0, i32 0
// CHECK-NEXT:   %4908 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4909 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4909, align 1
// CHECK-NEXT:   %4910 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %4909, 1
// CHECK-NEXT:   %4911 = extractvalue { ptr, ptr } %4908, 1
// CHECK-NEXT:   %4912 = extractvalue { ptr, ptr } %4908, 0
// CHECK-NEXT:   %__llgo_funcval_code640 = call ptr asm "", "=r,0"(ptr %4912)
// CHECK-NEXT:   %4913 = call %reflect.Value %__llgo_funcval_code640(ptr {{(nest|swiftself)}} %4911, %"{{.*}}/runtime/internal/runtime.eface" %4910)
// CHECK-NEXT:   %4914 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4906, i32 0, i32 1
// CHECK-NEXT:   %4915 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4916 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4916, align 1
// CHECK-NEXT:   %4917 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_uint8", ptr undef }, ptr %4916, 1
// CHECK-NEXT:   %4918 = extractvalue { ptr, ptr } %4915, 1
// CHECK-NEXT:   %4919 = extractvalue { ptr, ptr } %4915, 0
// CHECK-NEXT:   %__llgo_funcval_code641 = call ptr asm "", "=r,0"(ptr %4919)
// CHECK-NEXT:   %4920 = call %reflect.Value %__llgo_funcval_code641(ptr {{(nest|swiftself)}} %4918, %"{{.*}}/runtime/internal/runtime.eface" %4917)
// CHECK-NEXT:   store %reflect.Value %4913, ptr %4907, align 8
// CHECK-NEXT:   store %reflect.Value %4920, ptr %4914, align 8
// CHECK-NEXT:   %4921 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 321
// CHECK-NEXT:   %4922 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4921, i32 0, i32 0
// CHECK-NEXT:   %4923 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4924 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4924, align 1
// CHECK-NEXT:   %4925 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_main.MyByte", ptr undef }, ptr %4924, 1
// CHECK-NEXT:   %4926 = extractvalue { ptr, ptr } %4923, 1
// CHECK-NEXT:   %4927 = extractvalue { ptr, ptr } %4923, 0
// CHECK-NEXT:   %__llgo_funcval_code642 = call ptr asm "", "=r,0"(ptr %4927)
// CHECK-NEXT:   %4928 = call %reflect.Value %__llgo_funcval_code642(ptr {{(nest|swiftself)}} %4926, %"{{.*}}/runtime/internal/runtime.eface" %4925)
// CHECK-NEXT:   %4929 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4921, i32 0, i32 1
// CHECK-NEXT:   %4930 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4931 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 2)
// CHECK-NEXT:   store [2 x i8] zeroinitializer, ptr %4931, align 1
// CHECK-NEXT:   %4932 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[2]_llgo_main.MyByte", ptr undef }, ptr %4931, 1
// CHECK-NEXT:   %4933 = extractvalue { ptr, ptr } %4930, 1
// CHECK-NEXT:   %4934 = extractvalue { ptr, ptr } %4930, 0
// CHECK-NEXT:   %__llgo_funcval_code643 = call ptr asm "", "=r,0"(ptr %4934)
// CHECK-NEXT:   %4935 = call %reflect.Value %__llgo_funcval_code643(ptr {{(nest|swiftself)}} %4933, %"{{.*}}/runtime/internal/runtime.eface" %4932)
// CHECK-NEXT:   store %reflect.Value %4928, ptr %4922, align 8
// CHECK-NEXT:   store %reflect.Value %4935, ptr %4929, align 8
// CHECK-NEXT:   %4936 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 322
// CHECK-NEXT:   %4937 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4936, i32 0, i32 0
// CHECK-NEXT:   %4938 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4939 = extractvalue { ptr, ptr } %4938, 1
// CHECK-NEXT:   %4940 = extractvalue { ptr, ptr } %4938, 0
// CHECK-NEXT:   %__llgo_funcval_code644 = call ptr asm "", "=r,0"(ptr %4940)
// CHECK-NEXT:   %4941 = call %reflect.Value %__llgo_funcval_code644(ptr {{(nest|swiftself)}} %4939, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int", ptr null })
// CHECK-NEXT:   %4942 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4936, i32 0, i32 1
// CHECK-NEXT:   %4943 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4944 = extractvalue { ptr, ptr } %4943, 1
// CHECK-NEXT:   %4945 = extractvalue { ptr, ptr } %4943, 0
// CHECK-NEXT:   %__llgo_funcval_code645 = call ptr asm "", "=r,0"(ptr %4945)
// CHECK-NEXT:   %4946 = call %reflect.Value %__llgo_funcval_code645(ptr {{(nest|swiftself)}} %4944, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %4941, ptr %4937, align 8
// CHECK-NEXT:   store %reflect.Value %4946, ptr %4942, align 8
// CHECK-NEXT:   %4947 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 323
// CHECK-NEXT:   %4948 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4947, i32 0, i32 0
// CHECK-NEXT:   %4949 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4950 = extractvalue { ptr, ptr } %4949, 1
// CHECK-NEXT:   %4951 = extractvalue { ptr, ptr } %4949, 0
// CHECK-NEXT:   %__llgo_funcval_code646 = call ptr asm "", "=r,0"(ptr %4951)
// CHECK-NEXT:   %4952 = call %reflect.Value %__llgo_funcval_code646(ptr {{(nest|swiftself)}} %4950, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_uint8", ptr null })
// CHECK-NEXT:   %4953 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4947, i32 0, i32 1
// CHECK-NEXT:   %4954 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4955 = extractvalue { ptr, ptr } %4954, 1
// CHECK-NEXT:   %4956 = extractvalue { ptr, ptr } %4954, 0
// CHECK-NEXT:   %__llgo_funcval_code647 = call ptr asm "", "=r,0"(ptr %4956)
// CHECK-NEXT:   %4957 = call %reflect.Value %__llgo_funcval_code647(ptr {{(nest|swiftself)}} %4955, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4952, ptr %4948, align 8
// CHECK-NEXT:   store %reflect.Value %4957, ptr %4953, align 8
// CHECK-NEXT:   %4958 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 324
// CHECK-NEXT:   %4959 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4958, i32 0, i32 0
// CHECK-NEXT:   %4960 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4961 = extractvalue { ptr, ptr } %4960, 1
// CHECK-NEXT:   %4962 = extractvalue { ptr, ptr } %4960, 0
// CHECK-NEXT:   %__llgo_funcval_code648 = call ptr asm "", "=r,0"(ptr %4962)
// CHECK-NEXT:   %4963 = call %reflect.Value %__llgo_funcval_code648(ptr {{(nest|swiftself)}} %4961, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int32", ptr null })
// CHECK-NEXT:   %4964 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4958, i32 0, i32 1
// CHECK-NEXT:   %4965 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4966 = extractvalue { ptr, ptr } %4965, 1
// CHECK-NEXT:   %4967 = extractvalue { ptr, ptr } %4965, 0
// CHECK-NEXT:   %__llgo_funcval_code649 = call ptr asm "", "=r,0"(ptr %4967)
// CHECK-NEXT:   %4968 = call %reflect.Value %__llgo_funcval_code649(ptr {{(nest|swiftself)}} %4966, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int32", ptr null })
// CHECK-NEXT:   store %reflect.Value %4963, ptr %4959, align 8
// CHECK-NEXT:   store %reflect.Value %4968, ptr %4964, align 8
// CHECK-NEXT:   %4969 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 325
// CHECK-NEXT:   %4970 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4969, i32 0, i32 0
// CHECK-NEXT:   %4971 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4972 = extractvalue { ptr, ptr } %4971, 1
// CHECK-NEXT:   %4973 = extractvalue { ptr, ptr } %4971, 0
// CHECK-NEXT:   %__llgo_funcval_code650 = call ptr asm "", "=r,0"(ptr %4973)
// CHECK-NEXT:   %4974 = call %reflect.Value %__llgo_funcval_code650(ptr {{(nest|swiftself)}} %4972, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int64", ptr null })
// CHECK-NEXT:   %4975 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4969, i32 0, i32 1
// CHECK-NEXT:   %4976 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4977 = extractvalue { ptr, ptr } %4976, 1
// CHECK-NEXT:   %4978 = extractvalue { ptr, ptr } %4976, 0
// CHECK-NEXT:   %__llgo_funcval_code651 = call ptr asm "", "=r,0"(ptr %4978)
// CHECK-NEXT:   %4979 = call %reflect.Value %__llgo_funcval_code651(ptr {{(nest|swiftself)}} %4977, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"***_llgo_int64", ptr null })
// CHECK-NEXT:   store %reflect.Value %4974, ptr %4970, align 8
// CHECK-NEXT:   store %reflect.Value %4979, ptr %4975, align 8
// CHECK-NEXT:   %4980 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 326
// CHECK-NEXT:   %4981 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4980, i32 0, i32 0
// CHECK-NEXT:   %4982 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4983 = extractvalue { ptr, ptr } %4982, 1
// CHECK-NEXT:   %4984 = extractvalue { ptr, ptr } %4982, 0
// CHECK-NEXT:   %__llgo_funcval_code652 = call ptr asm "", "=r,0"(ptr %4984)
// CHECK-NEXT:   %4985 = call %reflect.Value %__llgo_funcval_code652(ptr {{(nest|swiftself)}} %4983, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_uint8", ptr null })
// CHECK-NEXT:   %4986 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4980, i32 0, i32 1
// CHECK-NEXT:   %4987 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4988 = extractvalue { ptr, ptr } %4987, 1
// CHECK-NEXT:   %4989 = extractvalue { ptr, ptr } %4987, 0
// CHECK-NEXT:   %__llgo_funcval_code653 = call ptr asm "", "=r,0"(ptr %4989)
// CHECK-NEXT:   %4990 = call %reflect.Value %__llgo_funcval_code653(ptr {{(nest|swiftself)}} %4988, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %4985, ptr %4981, align 8
// CHECK-NEXT:   store %reflect.Value %4990, ptr %4986, align 8
// CHECK-NEXT:   %4991 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 327
// CHECK-NEXT:   %4992 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4991, i32 0, i32 0
// CHECK-NEXT:   %4993 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4994 = extractvalue { ptr, ptr } %4993, 1
// CHECK-NEXT:   %4995 = extractvalue { ptr, ptr } %4993, 0
// CHECK-NEXT:   %__llgo_funcval_code654 = call ptr asm "", "=r,0"(ptr %4995)
// CHECK-NEXT:   %4996 = call %reflect.Value %__llgo_funcval_code654(ptr {{(nest|swiftself)}} %4994, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_main.MyByte", ptr null })
// CHECK-NEXT:   %4997 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %4991, i32 0, i32 1
// CHECK-NEXT:   %4998 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %4999 = extractvalue { ptr, ptr } %4998, 1
// CHECK-NEXT:   %5000 = extractvalue { ptr, ptr } %4998, 0
// CHECK-NEXT:   %__llgo_funcval_code655 = call ptr asm "", "=r,0"(ptr %5000)
// CHECK-NEXT:   %5001 = call %reflect.Value %__llgo_funcval_code655(ptr {{(nest|swiftself)}} %4999, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_main.MyByte", ptr null })
// CHECK-NEXT:   store %reflect.Value %4996, ptr %4992, align 8
// CHECK-NEXT:   store %reflect.Value %5001, ptr %4997, align 8
// CHECK-NEXT:   %5002 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 328
// CHECK-NEXT:   %5003 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5002, i32 0, i32 0
// CHECK-NEXT:   %5004 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5005 = extractvalue { ptr, ptr } %5004, 1
// CHECK-NEXT:   %5006 = extractvalue { ptr, ptr } %5004, 0
// CHECK-NEXT:   %__llgo_funcval_code656 = call ptr asm "", "=r,0"(ptr %5006)
// CHECK-NEXT:   %5007 = call %reflect.Value %__llgo_funcval_code656(ptr {{(nest|swiftself)}} %5005, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_bool", ptr null })
// CHECK-NEXT:   %5008 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5002, i32 0, i32 1
// CHECK-NEXT:   %5009 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5010 = extractvalue { ptr, ptr } %5009, 1
// CHECK-NEXT:   %5011 = extractvalue { ptr, ptr } %5009, 0
// CHECK-NEXT:   %__llgo_funcval_code657 = call ptr asm "", "=r,0"(ptr %5011)
// CHECK-NEXT:   %5012 = call %reflect.Value %__llgo_funcval_code657(ptr {{(nest|swiftself)}} %5010, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_bool", ptr null })
// CHECK-NEXT:   store %reflect.Value %5007, ptr %5003, align 8
// CHECK-NEXT:   store %reflect.Value %5012, ptr %5008, align 8
// CHECK-NEXT:   %5013 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 329
// CHECK-NEXT:   %5014 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5013, i32 0, i32 0
// CHECK-NEXT:   %5015 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5016 = extractvalue { ptr, ptr } %5015, 1
// CHECK-NEXT:   %5017 = extractvalue { ptr, ptr } %5015, 0
// CHECK-NEXT:   %__llgo_funcval_code658 = call ptr asm "", "=r,0"(ptr %5017)
// CHECK-NEXT:   %5018 = call %reflect.Value %__llgo_funcval_code658(ptr {{(nest|swiftself)}} %5016, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_uint8", ptr null })
// CHECK-NEXT:   %5019 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5013, i32 0, i32 1
// CHECK-NEXT:   %5020 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5021 = extractvalue { ptr, ptr } %5020, 1
// CHECK-NEXT:   %5022 = extractvalue { ptr, ptr } %5020, 0
// CHECK-NEXT:   %__llgo_funcval_code659 = call ptr asm "", "=r,0"(ptr %5022)
// CHECK-NEXT:   %5023 = call %reflect.Value %__llgo_funcval_code659(ptr {{(nest|swiftself)}} %5021, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_int]_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5018, ptr %5014, align 8
// CHECK-NEXT:   store %reflect.Value %5023, ptr %5019, align 8
// CHECK-NEXT:   %5024 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 330
// CHECK-NEXT:   %5025 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5024, i32 0, i32 0
// CHECK-NEXT:   %5026 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5027 = extractvalue { ptr, ptr } %5026, 1
// CHECK-NEXT:   %5028 = extractvalue { ptr, ptr } %5026, 0
// CHECK-NEXT:   %__llgo_funcval_code660 = call ptr asm "", "=r,0"(ptr %5028)
// CHECK-NEXT:   %5029 = call %reflect.Value %__llgo_funcval_code660(ptr {{(nest|swiftself)}} %5027, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_uint]_llgo_bool", ptr null })
// CHECK-NEXT:   %5030 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5024, i32 0, i32 1
// CHECK-NEXT:   %5031 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5032 = extractvalue { ptr, ptr } %5031, 1
// CHECK-NEXT:   %5033 = extractvalue { ptr, ptr } %5031, 0
// CHECK-NEXT:   %__llgo_funcval_code661 = call ptr asm "", "=r,0"(ptr %5033)
// CHECK-NEXT:   %5034 = call %reflect.Value %__llgo_funcval_code661(ptr {{(nest|swiftself)}} %5032, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"map[_llgo_uint]_llgo_bool", ptr null })
// CHECK-NEXT:   store %reflect.Value %5029, ptr %5025, align 8
// CHECK-NEXT:   store %reflect.Value %5034, ptr %5030, align 8
// CHECK-NEXT:   %5035 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 331
// CHECK-NEXT:   %5036 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5035, i32 0, i32 0
// CHECK-NEXT:   %5037 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5038 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %5038, align 8
// CHECK-NEXT:   %5039 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint", ptr undef }, ptr %5038, 1
// CHECK-NEXT:   %5040 = extractvalue { ptr, ptr } %5037, 1
// CHECK-NEXT:   %5041 = extractvalue { ptr, ptr } %5037, 0
// CHECK-NEXT:   %__llgo_funcval_code662 = call ptr asm "", "=r,0"(ptr %5041)
// CHECK-NEXT:   %5042 = call %reflect.Value %__llgo_funcval_code662(ptr {{(nest|swiftself)}} %5040, %"{{.*}}/runtime/internal/runtime.eface" %5039)
// CHECK-NEXT:   %5043 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5035, i32 0, i32 1
// CHECK-NEXT:   %5044 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5045 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %5045, align 8
// CHECK-NEXT:   %5046 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_uint", ptr undef }, ptr %5045, 1
// CHECK-NEXT:   %5047 = extractvalue { ptr, ptr } %5044, 1
// CHECK-NEXT:   %5048 = extractvalue { ptr, ptr } %5044, 0
// CHECK-NEXT:   %__llgo_funcval_code663 = call ptr asm "", "=r,0"(ptr %5048)
// CHECK-NEXT:   %5049 = call %reflect.Value %__llgo_funcval_code663(ptr {{(nest|swiftself)}} %5047, %"{{.*}}/runtime/internal/runtime.eface" %5046)
// CHECK-NEXT:   store %reflect.Value %5042, ptr %5036, align 8
// CHECK-NEXT:   store %reflect.Value %5049, ptr %5043, align 8
// CHECK-NEXT:   %5050 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 332
// CHECK-NEXT:   %5051 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5050, i32 0, i32 0
// CHECK-NEXT:   %5052 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5053 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %5053, align 8
// CHECK-NEXT:   %5054 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int", ptr undef }, ptr %5053, 1
// CHECK-NEXT:   %5055 = extractvalue { ptr, ptr } %5052, 1
// CHECK-NEXT:   %5056 = extractvalue { ptr, ptr } %5052, 0
// CHECK-NEXT:   %__llgo_funcval_code664 = call ptr asm "", "=r,0"(ptr %5056)
// CHECK-NEXT:   %5057 = call %reflect.Value %__llgo_funcval_code664(ptr {{(nest|swiftself)}} %5055, %"{{.*}}/runtime/internal/runtime.eface" %5054)
// CHECK-NEXT:   %5058 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5050, i32 0, i32 1
// CHECK-NEXT:   %5059 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5060 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" zeroinitializer, ptr %5060, align 8
// CHECK-NEXT:   %5061 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"[]_llgo_int", ptr undef }, ptr %5060, 1
// CHECK-NEXT:   %5062 = extractvalue { ptr, ptr } %5059, 1
// CHECK-NEXT:   %5063 = extractvalue { ptr, ptr } %5059, 0
// CHECK-NEXT:   %__llgo_funcval_code665 = call ptr asm "", "=r,0"(ptr %5063)
// CHECK-NEXT:   %5064 = call %reflect.Value %__llgo_funcval_code665(ptr {{(nest|swiftself)}} %5062, %"{{.*}}/runtime/internal/runtime.eface" %5061)
// CHECK-NEXT:   store %reflect.Value %5057, ptr %5051, align 8
// CHECK-NEXT:   store %reflect.Value %5064, ptr %5058, align 8
// CHECK-NEXT:   %5065 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 333
// CHECK-NEXT:   %5066 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5065, i32 0, i32 0
// CHECK-NEXT:   %5067 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5068 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5069 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_any", ptr undef }, ptr %5068, 1
// CHECK-NEXT:   %5070 = extractvalue { ptr, ptr } %5067, 1
// CHECK-NEXT:   %5071 = extractvalue { ptr, ptr } %5067, 0
// CHECK-NEXT:   %__llgo_funcval_code666 = call ptr asm "", "=r,0"(ptr %5071)
// CHECK-NEXT:   %5072 = call %reflect.Value %__llgo_funcval_code666(ptr {{(nest|swiftself)}} %5070, %"{{.*}}/runtime/internal/runtime.eface" %5069)
// CHECK-NEXT:   %5073 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5065, i32 0, i32 1
// CHECK-NEXT:   %5074 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5075 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5076 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_any", ptr undef }, ptr %5075, 1
// CHECK-NEXT:   %5077 = extractvalue { ptr, ptr } %5074, 1
// CHECK-NEXT:   %5078 = extractvalue { ptr, ptr } %5074, 0
// CHECK-NEXT:   %__llgo_funcval_code667 = call ptr asm "", "=r,0"(ptr %5078)
// CHECK-NEXT:   %5079 = call %reflect.Value %__llgo_funcval_code667(ptr {{(nest|swiftself)}} %5077, %"{{.*}}/runtime/internal/runtime.eface" %5076)
// CHECK-NEXT:   store %reflect.Value %5072, ptr %5066, align 8
// CHECK-NEXT:   store %reflect.Value %5079, ptr %5073, align 8
// CHECK-NEXT:   %5080 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 334
// CHECK-NEXT:   %5081 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5080, i32 0, i32 0
// CHECK-NEXT:   %5082 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5083 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5084 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Reader", ptr undef }, ptr %5083, 1
// CHECK-NEXT:   %5085 = extractvalue { ptr, ptr } %5082, 1
// CHECK-NEXT:   %5086 = extractvalue { ptr, ptr } %5082, 0
// CHECK-NEXT:   %__llgo_funcval_code668 = call ptr asm "", "=r,0"(ptr %5086)
// CHECK-NEXT:   %5087 = call %reflect.Value %__llgo_funcval_code668(ptr {{(nest|swiftself)}} %5085, %"{{.*}}/runtime/internal/runtime.eface" %5084)
// CHECK-NEXT:   %5088 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5080, i32 0, i32 1
// CHECK-NEXT:   %5089 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5090 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5091 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Reader", ptr undef }, ptr %5090, 1
// CHECK-NEXT:   %5092 = extractvalue { ptr, ptr } %5089, 1
// CHECK-NEXT:   %5093 = extractvalue { ptr, ptr } %5089, 0
// CHECK-NEXT:   %__llgo_funcval_code669 = call ptr asm "", "=r,0"(ptr %5093)
// CHECK-NEXT:   %5094 = call %reflect.Value %__llgo_funcval_code669(ptr {{(nest|swiftself)}} %5092, %"{{.*}}/runtime/internal/runtime.eface" %5091)
// CHECK-NEXT:   store %reflect.Value %5087, ptr %5081, align 8
// CHECK-NEXT:   store %reflect.Value %5094, ptr %5088, align 8
// CHECK-NEXT:   %5095 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 335
// CHECK-NEXT:   %5096 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5095, i32 0, i32 0
// CHECK-NEXT:   %5097 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5098 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5099 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Writer", ptr undef }, ptr %5098, 1
// CHECK-NEXT:   %5100 = extractvalue { ptr, ptr } %5097, 1
// CHECK-NEXT:   %5101 = extractvalue { ptr, ptr } %5097, 0
// CHECK-NEXT:   %__llgo_funcval_code670 = call ptr asm "", "=r,0"(ptr %5101)
// CHECK-NEXT:   %5102 = call %reflect.Value %__llgo_funcval_code670(ptr {{(nest|swiftself)}} %5100, %"{{.*}}/runtime/internal/runtime.eface" %5099)
// CHECK-NEXT:   %5103 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5095, i32 0, i32 1
// CHECK-NEXT:   %5104 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5105 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %5106 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_io.Writer", ptr undef }, ptr %5105, 1
// CHECK-NEXT:   %5107 = extractvalue { ptr, ptr } %5104, 1
// CHECK-NEXT:   %5108 = extractvalue { ptr, ptr } %5104, 0
// CHECK-NEXT:   %__llgo_funcval_code671 = call ptr asm "", "=r,0"(ptr %5108)
// CHECK-NEXT:   %5109 = call %reflect.Value %__llgo_funcval_code671(ptr {{(nest|swiftself)}} %5107, %"{{.*}}/runtime/internal/runtime.eface" %5106)
// CHECK-NEXT:   store %reflect.Value %5102, ptr %5096, align 8
// CHECK-NEXT:   store %reflect.Value %5109, ptr %5103, align 8
// CHECK-NEXT:   %5110 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 336
// CHECK-NEXT:   %5111 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5110, i32 0, i32 0
// CHECK-NEXT:   %5112 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5113 = extractvalue { ptr, ptr } %5112, 1
// CHECK-NEXT:   %5114 = extractvalue { ptr, ptr } %5112, 0
// CHECK-NEXT:   %__llgo_funcval_code672 = call ptr asm "", "=r,0"(ptr %5114)
// CHECK-NEXT:   %5115 = call %reflect.Value %__llgo_funcval_code672(ptr {{(nest|swiftself)}} %5113, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   %5116 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5110, i32 0, i32 1
// CHECK-NEXT:   %5117 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5118 = extractvalue { ptr, ptr } %5117, 1
// CHECK-NEXT:   %5119 = extractvalue { ptr, ptr } %5117, 0
// CHECK-NEXT:   %__llgo_funcval_code673 = call ptr asm "", "=r,0"(ptr %5119)
// CHECK-NEXT:   %5120 = call %reflect.Value %__llgo_funcval_code673(ptr {{(nest|swiftself)}} %5118, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5115, ptr %5111, align 8
// CHECK-NEXT:   store %reflect.Value %5120, ptr %5116, align 8
// CHECK-NEXT:   %5121 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 337
// CHECK-NEXT:   %5122 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5121, i32 0, i32 0
// CHECK-NEXT:   %5123 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5124 = extractvalue { ptr, ptr } %5123, 1
// CHECK-NEXT:   %5125 = extractvalue { ptr, ptr } %5123, 0
// CHECK-NEXT:   %__llgo_funcval_code674 = call ptr asm "", "=r,0"(ptr %5125)
// CHECK-NEXT:   %5126 = call %reflect.Value %__llgo_funcval_code674(ptr {{(nest|swiftself)}} %5124, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   %5127 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5121, i32 0, i32 1
// CHECK-NEXT:   %5128 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5129 = extractvalue { ptr, ptr } %5128, 1
// CHECK-NEXT:   %5130 = extractvalue { ptr, ptr } %5128, 0
// CHECK-NEXT:   %__llgo_funcval_code675 = call ptr asm "", "=r,0"(ptr %5130)
// CHECK-NEXT:   %5131 = call %reflect.Value %__llgo_funcval_code675(ptr {{(nest|swiftself)}} %5129, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5126, ptr %5122, align 8
// CHECK-NEXT:   store %reflect.Value %5131, ptr %5127, align 8
// CHECK-NEXT:   %5132 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 338
// CHECK-NEXT:   %5133 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5132, i32 0, i32 0
// CHECK-NEXT:   %5134 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5135 = extractvalue { ptr, ptr } %5134, 1
// CHECK-NEXT:   %5136 = extractvalue { ptr, ptr } %5134, 0
// CHECK-NEXT:   %__llgo_funcval_code676 = call ptr asm "", "=r,0"(ptr %5136)
// CHECK-NEXT:   %5137 = call %reflect.Value %__llgo_funcval_code676(ptr {{(nest|swiftself)}} %5135, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   %5138 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5132, i32 0, i32 1
// CHECK-NEXT:   %5139 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5140 = extractvalue { ptr, ptr } %5139, 1
// CHECK-NEXT:   %5141 = extractvalue { ptr, ptr } %5139, 0
// CHECK-NEXT:   %__llgo_funcval_code677 = call ptr asm "", "=r,0"(ptr %5141)
// CHECK-NEXT:   %5142 = call %reflect.Value %__llgo_funcval_code677(ptr {{(nest|swiftself)}} %5140, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5137, ptr %5133, align 8
// CHECK-NEXT:   store %reflect.Value %5142, ptr %5138, align 8
// CHECK-NEXT:   %5143 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 339
// CHECK-NEXT:   %5144 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5143, i32 0, i32 0
// CHECK-NEXT:   %5145 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5146 = extractvalue { ptr, ptr } %5145, 1
// CHECK-NEXT:   %5147 = extractvalue { ptr, ptr } %5145, 0
// CHECK-NEXT:   %__llgo_funcval_code678 = call ptr asm "", "=r,0"(ptr %5147)
// CHECK-NEXT:   %5148 = call %reflect.Value %__llgo_funcval_code678(ptr {{(nest|swiftself)}} %5146, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   %5149 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5143, i32 0, i32 1
// CHECK-NEXT:   %5150 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5151 = extractvalue { ptr, ptr } %5150, 1
// CHECK-NEXT:   %5152 = extractvalue { ptr, ptr } %5150, 0
// CHECK-NEXT:   %__llgo_funcval_code679 = call ptr asm "", "=r,0"(ptr %5152)
// CHECK-NEXT:   %5153 = call %reflect.Value %__llgo_funcval_code679(ptr {{(nest|swiftself)}} %5151, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5148, ptr %5144, align 8
// CHECK-NEXT:   store %reflect.Value %5153, ptr %5149, align 8
// CHECK-NEXT:   %5154 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 340
// CHECK-NEXT:   %5155 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5154, i32 0, i32 0
// CHECK-NEXT:   %5156 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5157 = extractvalue { ptr, ptr } %5156, 1
// CHECK-NEXT:   %5158 = extractvalue { ptr, ptr } %5156, 0
// CHECK-NEXT:   %__llgo_funcval_code680 = call ptr asm "", "=r,0"(ptr %5158)
// CHECK-NEXT:   %5159 = call %reflect.Value %__llgo_funcval_code680(ptr {{(nest|swiftself)}} %5157, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanRecv, ptr null })
// CHECK-NEXT:   %5160 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5154, i32 0, i32 1
// CHECK-NEXT:   %5161 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5162 = extractvalue { ptr, ptr } %5161, 1
// CHECK-NEXT:   %5163 = extractvalue { ptr, ptr } %5161, 0
// CHECK-NEXT:   %__llgo_funcval_code681 = call ptr asm "", "=r,0"(ptr %5163)
// CHECK-NEXT:   %5164 = call %reflect.Value %__llgo_funcval_code681(ptr {{(nest|swiftself)}} %5162, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5159, ptr %5155, align 8
// CHECK-NEXT:   store %reflect.Value %5164, ptr %5160, align 8
// CHECK-NEXT:   %5165 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 341
// CHECK-NEXT:   %5166 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5165, i32 0, i32 0
// CHECK-NEXT:   %5167 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5168 = extractvalue { ptr, ptr } %5167, 1
// CHECK-NEXT:   %5169 = extractvalue { ptr, ptr } %5167, 0
// CHECK-NEXT:   %__llgo_funcval_code682 = call ptr asm "", "=r,0"(ptr %5169)
// CHECK-NEXT:   %5170 = call %reflect.Value %__llgo_funcval_code682(ptr {{(nest|swiftself)}} %5168, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan _llgo_int", ptr null })
// CHECK-NEXT:   %5171 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5165, i32 0, i32 1
// CHECK-NEXT:   %5172 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5173 = extractvalue { ptr, ptr } %5172, 1
// CHECK-NEXT:   %5174 = extractvalue { ptr, ptr } %5172, 0
// CHECK-NEXT:   %__llgo_funcval_code683 = call ptr asm "", "=r,0"(ptr %5174)
// CHECK-NEXT:   %5175 = call %reflect.Value %__llgo_funcval_code683(ptr {{(nest|swiftself)}} %5173, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5170, ptr %5166, align 8
// CHECK-NEXT:   store %reflect.Value %5175, ptr %5171, align 8
// CHECK-NEXT:   %5176 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 342
// CHECK-NEXT:   %5177 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5176, i32 0, i32 0
// CHECK-NEXT:   %5178 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5179 = extractvalue { ptr, ptr } %5178, 1
// CHECK-NEXT:   %5180 = extractvalue { ptr, ptr } %5178, 0
// CHECK-NEXT:   %__llgo_funcval_code684 = call ptr asm "", "=r,0"(ptr %5180)
// CHECK-NEXT:   %5181 = call %reflect.Value %__llgo_funcval_code684(ptr {{(nest|swiftself)}} %5179, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanSend, ptr null })
// CHECK-NEXT:   %5182 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5176, i32 0, i32 1
// CHECK-NEXT:   %5183 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5184 = extractvalue { ptr, ptr } %5183, 1
// CHECK-NEXT:   %5185 = extractvalue { ptr, ptr } %5183, 0
// CHECK-NEXT:   %__llgo_funcval_code685 = call ptr asm "", "=r,0"(ptr %5185)
// CHECK-NEXT:   %5186 = call %reflect.Value %__llgo_funcval_code685(ptr {{(nest|swiftself)}} %5184, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5181, ptr %5177, align 8
// CHECK-NEXT:   store %reflect.Value %5186, ptr %5182, align 8
// CHECK-NEXT:   %5187 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 343
// CHECK-NEXT:   %5188 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5187, i32 0, i32 0
// CHECK-NEXT:   %5189 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5190 = extractvalue { ptr, ptr } %5189, 1
// CHECK-NEXT:   %5191 = extractvalue { ptr, ptr } %5189, 0
// CHECK-NEXT:   %__llgo_funcval_code686 = call ptr asm "", "=r,0"(ptr %5191)
// CHECK-NEXT:   %5192 = call %reflect.Value %__llgo_funcval_code686(ptr {{(nest|swiftself)}} %5190, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- _llgo_int", ptr null })
// CHECK-NEXT:   %5193 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5187, i32 0, i32 1
// CHECK-NEXT:   %5194 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5195 = extractvalue { ptr, ptr } %5194, 1
// CHECK-NEXT:   %5196 = extractvalue { ptr, ptr } %5194, 0
// CHECK-NEXT:   %__llgo_funcval_code687 = call ptr asm "", "=r,0"(ptr %5196)
// CHECK-NEXT:   %5197 = call %reflect.Value %__llgo_funcval_code687(ptr {{(nest|swiftself)}} %5195, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5192, ptr %5188, align 8
// CHECK-NEXT:   store %reflect.Value %5197, ptr %5193, align 8
// CHECK-NEXT:   %5198 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 344
// CHECK-NEXT:   %5199 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5198, i32 0, i32 0
// CHECK-NEXT:   %5200 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5201 = extractvalue { ptr, ptr } %5200, 1
// CHECK-NEXT:   %5202 = extractvalue { ptr, ptr } %5200, 0
// CHECK-NEXT:   %__llgo_funcval_code688 = call ptr asm "", "=r,0"(ptr %5202)
// CHECK-NEXT:   %5203 = call %reflect.Value %__llgo_funcval_code688(ptr {{(nest|swiftself)}} %5201, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   %5204 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5198, i32 0, i32 1
// CHECK-NEXT:   %5205 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5206 = extractvalue { ptr, ptr } %5205, 1
// CHECK-NEXT:   %5207 = extractvalue { ptr, ptr } %5205, 0
// CHECK-NEXT:   %__llgo_funcval_code689 = call ptr asm "", "=r,0"(ptr %5207)
// CHECK-NEXT:   %5208 = call %reflect.Value %__llgo_funcval_code689(ptr {{(nest|swiftself)}} %5206, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5203, ptr %5199, align 8
// CHECK-NEXT:   store %reflect.Value %5208, ptr %5204, align 8
// CHECK-NEXT:   %5209 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 345
// CHECK-NEXT:   %5210 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5209, i32 0, i32 0
// CHECK-NEXT:   %5211 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5212 = extractvalue { ptr, ptr } %5211, 1
// CHECK-NEXT:   %5213 = extractvalue { ptr, ptr } %5211, 0
// CHECK-NEXT:   %__llgo_funcval_code690 = call ptr asm "", "=r,0"(ptr %5213)
// CHECK-NEXT:   %5214 = call %reflect.Value %__llgo_funcval_code690(ptr {{(nest|swiftself)}} %5212, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   %5215 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5209, i32 0, i32 1
// CHECK-NEXT:   %5216 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5217 = extractvalue { ptr, ptr } %5216, 1
// CHECK-NEXT:   %5218 = extractvalue { ptr, ptr } %5216, 0
// CHECK-NEXT:   %__llgo_funcval_code691 = call ptr asm "", "=r,0"(ptr %5218)
// CHECK-NEXT:   %5219 = call %reflect.Value %__llgo_funcval_code691(ptr {{(nest|swiftself)}} %5217, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   store %reflect.Value %5214, ptr %5210, align 8
// CHECK-NEXT:   store %reflect.Value %5219, ptr %5215, align 8
// CHECK-NEXT:   %5220 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 346
// CHECK-NEXT:   %5221 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5220, i32 0, i32 0
// CHECK-NEXT:   %5222 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5223 = extractvalue { ptr, ptr } %5222, 1
// CHECK-NEXT:   %5224 = extractvalue { ptr, ptr } %5222, 0
// CHECK-NEXT:   %__llgo_funcval_code692 = call ptr asm "", "=r,0"(ptr %5224)
// CHECK-NEXT:   %5225 = call %reflect.Value %__llgo_funcval_code692(ptr {{(nest|swiftself)}} %5223, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   %5226 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5220, i32 0, i32 1
// CHECK-NEXT:   %5227 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5228 = extractvalue { ptr, ptr } %5227, 1
// CHECK-NEXT:   %5229 = extractvalue { ptr, ptr } %5227, 0
// CHECK-NEXT:   %__llgo_funcval_code693 = call ptr asm "", "=r,0"(ptr %5229)
// CHECK-NEXT:   %5230 = call %reflect.Value %__llgo_funcval_code693(ptr {{(nest|swiftself)}} %5228, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5225, ptr %5221, align 8
// CHECK-NEXT:   store %reflect.Value %5230, ptr %5226, align 8
// CHECK-NEXT:   %5231 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 347
// CHECK-NEXT:   %5232 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5231, i32 0, i32 0
// CHECK-NEXT:   %5233 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5234 = extractvalue { ptr, ptr } %5233, 1
// CHECK-NEXT:   %5235 = extractvalue { ptr, ptr } %5233, 0
// CHECK-NEXT:   %__llgo_funcval_code694 = call ptr asm "", "=r,0"(ptr %5235)
// CHECK-NEXT:   %5236 = call %reflect.Value %__llgo_funcval_code694(ptr {{(nest|swiftself)}} %5234, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan _llgo_int", ptr null })
// CHECK-NEXT:   %5237 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5231, i32 0, i32 1
// CHECK-NEXT:   %5238 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5239 = extractvalue { ptr, ptr } %5238, 1
// CHECK-NEXT:   %5240 = extractvalue { ptr, ptr } %5238, 0
// CHECK-NEXT:   %__llgo_funcval_code695 = call ptr asm "", "=r,0"(ptr %5240)
// CHECK-NEXT:   %5241 = call %reflect.Value %__llgo_funcval_code695(ptr {{(nest|swiftself)}} %5239, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- _llgo_int", ptr null })
// CHECK-NEXT:   store %reflect.Value %5236, ptr %5232, align 8
// CHECK-NEXT:   store %reflect.Value %5241, ptr %5237, align 8
// CHECK-NEXT:   %5242 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 348
// CHECK-NEXT:   %5243 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5242, i32 0, i32 0
// CHECK-NEXT:   %5244 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5245 = extractvalue { ptr, ptr } %5244, 1
// CHECK-NEXT:   %5246 = extractvalue { ptr, ptr } %5244, 0
// CHECK-NEXT:   %__llgo_funcval_code696 = call ptr asm "", "=r,0"(ptr %5246)
// CHECK-NEXT:   %5247 = call %reflect.Value %__llgo_funcval_code696(ptr {{(nest|swiftself)}} %5245, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   %5248 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5242, i32 0, i32 1
// CHECK-NEXT:   %5249 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5250 = extractvalue { ptr, ptr } %5249, 1
// CHECK-NEXT:   %5251 = extractvalue { ptr, ptr } %5249, 0
// CHECK-NEXT:   %__llgo_funcval_code697 = call ptr asm "", "=r,0"(ptr %5251)
// CHECK-NEXT:   %5252 = call %reflect.Value %__llgo_funcval_code697(ptr {{(nest|swiftself)}} %5250, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5247, ptr %5243, align 8
// CHECK-NEXT:   store %reflect.Value %5252, ptr %5248, align 8
// CHECK-NEXT:   %5253 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 349
// CHECK-NEXT:   %5254 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5253, i32 0, i32 0
// CHECK-NEXT:   %5255 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5256 = extractvalue { ptr, ptr } %5255, 1
// CHECK-NEXT:   %5257 = extractvalue { ptr, ptr } %5255, 0
// CHECK-NEXT:   %__llgo_funcval_code698 = call ptr asm "", "=r,0"(ptr %5257)
// CHECK-NEXT:   %5258 = call %reflect.Value %__llgo_funcval_code698(ptr {{(nest|swiftself)}} %5256, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   %5259 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5253, i32 0, i32 1
// CHECK-NEXT:   %5260 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5261 = extractvalue { ptr, ptr } %5260, 1
// CHECK-NEXT:   %5262 = extractvalue { ptr, ptr } %5260, 0
// CHECK-NEXT:   %__llgo_funcval_code699 = call ptr asm "", "=r,0"(ptr %5262)
// CHECK-NEXT:   %5263 = call %reflect.Value %__llgo_funcval_code699(ptr {{(nest|swiftself)}} %5261, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5258, ptr %5254, align 8
// CHECK-NEXT:   store %reflect.Value %5263, ptr %5259, align 8
// CHECK-NEXT:   %5264 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 350
// CHECK-NEXT:   %5265 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5264, i32 0, i32 0
// CHECK-NEXT:   %5266 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5267 = extractvalue { ptr, ptr } %5266, 1
// CHECK-NEXT:   %5268 = extractvalue { ptr, ptr } %5266, 0
// CHECK-NEXT:   %__llgo_funcval_code700 = call ptr asm "", "=r,0"(ptr %5268)
// CHECK-NEXT:   %5269 = call %reflect.Value %__llgo_funcval_code700(ptr {{(nest|swiftself)}} %5267, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5270 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5264, i32 0, i32 1
// CHECK-NEXT:   %5271 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5272 = extractvalue { ptr, ptr } %5271, 1
// CHECK-NEXT:   %5273 = extractvalue { ptr, ptr } %5271, 0
// CHECK-NEXT:   %__llgo_funcval_code701 = call ptr asm "", "=r,0"(ptr %5273)
// CHECK-NEXT:   %5274 = call %reflect.Value %__llgo_funcval_code701(ptr {{(nest|swiftself)}} %5272, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5269, ptr %5265, align 8
// CHECK-NEXT:   store %reflect.Value %5274, ptr %5270, align 8
// CHECK-NEXT:   %5275 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 351
// CHECK-NEXT:   %5276 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5275, i32 0, i32 0
// CHECK-NEXT:   %5277 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5278 = extractvalue { ptr, ptr } %5277, 1
// CHECK-NEXT:   %5279 = extractvalue { ptr, ptr } %5277, 0
// CHECK-NEXT:   %__llgo_funcval_code702 = call ptr asm "", "=r,0"(ptr %5279)
// CHECK-NEXT:   %5280 = call %reflect.Value %__llgo_funcval_code702(ptr {{(nest|swiftself)}} %5278, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5281 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5275, i32 0, i32 1
// CHECK-NEXT:   %5282 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5283 = extractvalue { ptr, ptr } %5282, 1
// CHECK-NEXT:   %5284 = extractvalue { ptr, ptr } %5282, 0
// CHECK-NEXT:   %__llgo_funcval_code703 = call ptr asm "", "=r,0"(ptr %5284)
// CHECK-NEXT:   %5285 = call %reflect.Value %__llgo_funcval_code703(ptr {{(nest|swiftself)}} %5283, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5280, ptr %5276, align 8
// CHECK-NEXT:   store %reflect.Value %5285, ptr %5281, align 8
// CHECK-NEXT:   %5286 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 352
// CHECK-NEXT:   %5287 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5286, i32 0, i32 0
// CHECK-NEXT:   %5288 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5289 = extractvalue { ptr, ptr } %5288, 1
// CHECK-NEXT:   %5290 = extractvalue { ptr, ptr } %5288, 0
// CHECK-NEXT:   %__llgo_funcval_code704 = call ptr asm "", "=r,0"(ptr %5290)
// CHECK-NEXT:   %5291 = call %reflect.Value %__llgo_funcval_code704(ptr {{(nest|swiftself)}} %5289, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanRecv, ptr null })
// CHECK-NEXT:   %5292 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5286, i32 0, i32 1
// CHECK-NEXT:   %5293 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5294 = extractvalue { ptr, ptr } %5293, 1
// CHECK-NEXT:   %5295 = extractvalue { ptr, ptr } %5293, 0
// CHECK-NEXT:   %__llgo_funcval_code705 = call ptr asm "", "=r,0"(ptr %5295)
// CHECK-NEXT:   %5296 = call %reflect.Value %__llgo_funcval_code705(ptr {{(nest|swiftself)}} %5294, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5291, ptr %5287, align 8
// CHECK-NEXT:   store %reflect.Value %5296, ptr %5292, align 8
// CHECK-NEXT:   %5297 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 353
// CHECK-NEXT:   %5298 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5297, i32 0, i32 0
// CHECK-NEXT:   %5299 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5300 = extractvalue { ptr, ptr } %5299, 1
// CHECK-NEXT:   %5301 = extractvalue { ptr, ptr } %5299, 0
// CHECK-NEXT:   %__llgo_funcval_code706 = call ptr asm "", "=r,0"(ptr %5301)
// CHECK-NEXT:   %5302 = call %reflect.Value %__llgo_funcval_code706(ptr {{(nest|swiftself)}} %5300, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5303 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5297, i32 0, i32 1
// CHECK-NEXT:   %5304 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5305 = extractvalue { ptr, ptr } %5304, 1
// CHECK-NEXT:   %5306 = extractvalue { ptr, ptr } %5304, 0
// CHECK-NEXT:   %__llgo_funcval_code707 = call ptr asm "", "=r,0"(ptr %5306)
// CHECK-NEXT:   %5307 = call %reflect.Value %__llgo_funcval_code707(ptr {{(nest|swiftself)}} %5305, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5302, ptr %5298, align 8
// CHECK-NEXT:   store %reflect.Value %5307, ptr %5303, align 8
// CHECK-NEXT:   %5308 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 354
// CHECK-NEXT:   %5309 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5308, i32 0, i32 0
// CHECK-NEXT:   %5310 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5311 = extractvalue { ptr, ptr } %5310, 1
// CHECK-NEXT:   %5312 = extractvalue { ptr, ptr } %5310, 0
// CHECK-NEXT:   %__llgo_funcval_code708 = call ptr asm "", "=r,0"(ptr %5312)
// CHECK-NEXT:   %5313 = call %reflect.Value %__llgo_funcval_code708(ptr {{(nest|swiftself)}} %5311, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanSend, ptr null })
// CHECK-NEXT:   %5314 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5308, i32 0, i32 1
// CHECK-NEXT:   %5315 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5316 = extractvalue { ptr, ptr } %5315, 1
// CHECK-NEXT:   %5317 = extractvalue { ptr, ptr } %5315, 0
// CHECK-NEXT:   %__llgo_funcval_code709 = call ptr asm "", "=r,0"(ptr %5317)
// CHECK-NEXT:   %5318 = call %reflect.Value %__llgo_funcval_code709(ptr {{(nest|swiftself)}} %5316, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5313, ptr %5309, align 8
// CHECK-NEXT:   store %reflect.Value %5318, ptr %5314, align 8
// CHECK-NEXT:   %5319 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 355
// CHECK-NEXT:   %5320 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5319, i32 0, i32 0
// CHECK-NEXT:   %5321 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5322 = extractvalue { ptr, ptr } %5321, 1
// CHECK-NEXT:   %5323 = extractvalue { ptr, ptr } %5321, 0
// CHECK-NEXT:   %__llgo_funcval_code710 = call ptr asm "", "=r,0"(ptr %5323)
// CHECK-NEXT:   %5324 = call %reflect.Value %__llgo_funcval_code710(ptr {{(nest|swiftself)}} %5322, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5325 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5319, i32 0, i32 1
// CHECK-NEXT:   %5326 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5327 = extractvalue { ptr, ptr } %5326, 1
// CHECK-NEXT:   %5328 = extractvalue { ptr, ptr } %5326, 0
// CHECK-NEXT:   %__llgo_funcval_code711 = call ptr asm "", "=r,0"(ptr %5328)
// CHECK-NEXT:   %5329 = call %reflect.Value %__llgo_funcval_code711(ptr {{(nest|swiftself)}} %5327, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5324, ptr %5320, align 8
// CHECK-NEXT:   store %reflect.Value %5329, ptr %5325, align 8
// CHECK-NEXT:   %5330 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 356
// CHECK-NEXT:   %5331 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5330, i32 0, i32 0
// CHECK-NEXT:   %5332 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5333 = extractvalue { ptr, ptr } %5332, 1
// CHECK-NEXT:   %5334 = extractvalue { ptr, ptr } %5332, 0
// CHECK-NEXT:   %__llgo_funcval_code712 = call ptr asm "", "=r,0"(ptr %5334)
// CHECK-NEXT:   %5335 = call %reflect.Value %__llgo_funcval_code712(ptr {{(nest|swiftself)}} %5333, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   %5336 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5330, i32 0, i32 1
// CHECK-NEXT:   %5337 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5338 = extractvalue { ptr, ptr } %5337, 1
// CHECK-NEXT:   %5339 = extractvalue { ptr, ptr } %5337, 0
// CHECK-NEXT:   %__llgo_funcval_code713 = call ptr asm "", "=r,0"(ptr %5339)
// CHECK-NEXT:   %5340 = call %reflect.Value %__llgo_funcval_code713(ptr {{(nest|swiftself)}} %5338, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5335, ptr %5331, align 8
// CHECK-NEXT:   store %reflect.Value %5340, ptr %5336, align 8
// CHECK-NEXT:   %5341 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 357
// CHECK-NEXT:   %5342 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5341, i32 0, i32 0
// CHECK-NEXT:   %5343 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5344 = extractvalue { ptr, ptr } %5343, 1
// CHECK-NEXT:   %5345 = extractvalue { ptr, ptr } %5343, 0
// CHECK-NEXT:   %__llgo_funcval_code714 = call ptr asm "", "=r,0"(ptr %5345)
// CHECK-NEXT:   %5346 = call %reflect.Value %__llgo_funcval_code714(ptr {{(nest|swiftself)}} %5344, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5347 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5341, i32 0, i32 1
// CHECK-NEXT:   %5348 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5349 = extractvalue { ptr, ptr } %5348, 1
// CHECK-NEXT:   %5350 = extractvalue { ptr, ptr } %5348, 0
// CHECK-NEXT:   %__llgo_funcval_code715 = call ptr asm "", "=r,0"(ptr %5350)
// CHECK-NEXT:   %5351 = call %reflect.Value %__llgo_funcval_code715(ptr {{(nest|swiftself)}} %5349, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   store %reflect.Value %5346, ptr %5342, align 8
// CHECK-NEXT:   store %reflect.Value %5351, ptr %5347, align 8
// CHECK-NEXT:   %5352 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 358
// CHECK-NEXT:   %5353 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5352, i32 0, i32 0
// CHECK-NEXT:   %5354 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5355 = extractvalue { ptr, ptr } %5354, 1
// CHECK-NEXT:   %5356 = extractvalue { ptr, ptr } %5354, 0
// CHECK-NEXT:   %__llgo_funcval_code716 = call ptr asm "", "=r,0"(ptr %5356)
// CHECK-NEXT:   %5357 = call %reflect.Value %__llgo_funcval_code716(ptr {{(nest|swiftself)}} %5355, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5358 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5352, i32 0, i32 1
// CHECK-NEXT:   %5359 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5360 = extractvalue { ptr, ptr } %5359, 1
// CHECK-NEXT:   %5361 = extractvalue { ptr, ptr } %5359, 0
// CHECK-NEXT:   %__llgo_funcval_code717 = call ptr asm "", "=r,0"(ptr %5361)
// CHECK-NEXT:   %5362 = call %reflect.Value %__llgo_funcval_code717(ptr {{(nest|swiftself)}} %5360, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"<-chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5357, ptr %5353, align 8
// CHECK-NEXT:   store %reflect.Value %5362, ptr %5358, align 8
// CHECK-NEXT:   %5363 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 359
// CHECK-NEXT:   %5364 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5363, i32 0, i32 0
// CHECK-NEXT:   %5365 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5366 = extractvalue { ptr, ptr } %5365, 1
// CHECK-NEXT:   %5367 = extractvalue { ptr, ptr } %5365, 0
// CHECK-NEXT:   %__llgo_funcval_code718 = call ptr asm "", "=r,0"(ptr %5367)
// CHECK-NEXT:   %5368 = call %reflect.Value %__llgo_funcval_code718(ptr {{(nest|swiftself)}} %5366, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan []_llgo_uint8", ptr null })
// CHECK-NEXT:   %5369 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5363, i32 0, i32 1
// CHECK-NEXT:   %5370 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5371 = extractvalue { ptr, ptr } %5370, 1
// CHECK-NEXT:   %5372 = extractvalue { ptr, ptr } %5370, 0
// CHECK-NEXT:   %__llgo_funcval_code719 = call ptr asm "", "=r,0"(ptr %5372)
// CHECK-NEXT:   %5373 = call %reflect.Value %__llgo_funcval_code719(ptr {{(nest|swiftself)}} %5371, %"{{.*}}/runtime/internal/runtime.eface" { ptr @"chan<- []_llgo_uint8", ptr null })
// CHECK-NEXT:   store %reflect.Value %5368, ptr %5364, align 8
// CHECK-NEXT:   store %reflect.Value %5373, ptr %5369, align 8
// CHECK-NEXT:   %5374 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 360
// CHECK-NEXT:   %5375 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5374, i32 0, i32 0
// CHECK-NEXT:   %5376 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5377 = extractvalue { ptr, ptr } %5376, 1
// CHECK-NEXT:   %5378 = extractvalue { ptr, ptr } %5376, 0
// CHECK-NEXT:   %__llgo_funcval_code720 = call ptr asm "", "=r,0"(ptr %5378)
// CHECK-NEXT:   %5379 = call %reflect.Value %__llgo_funcval_code720(ptr {{(nest|swiftself)}} %5377, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   %5380 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5374, i32 0, i32 1
// CHECK-NEXT:   %5381 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5382 = extractvalue { ptr, ptr } %5381, 1
// CHECK-NEXT:   %5383 = extractvalue { ptr, ptr } %5381, 0
// CHECK-NEXT:   %__llgo_funcval_code721 = call ptr asm "", "=r,0"(ptr %5383)
// CHECK-NEXT:   %5384 = call %reflect.Value %__llgo_funcval_code721(ptr {{(nest|swiftself)}} %5382, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChan, ptr null })
// CHECK-NEXT:   store %reflect.Value %5379, ptr %5375, align 8
// CHECK-NEXT:   store %reflect.Value %5384, ptr %5380, align 8
// CHECK-NEXT:   %5385 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 361
// CHECK-NEXT:   %5386 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5385, i32 0, i32 0
// CHECK-NEXT:   %5387 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5388 = extractvalue { ptr, ptr } %5387, 1
// CHECK-NEXT:   %5389 = extractvalue { ptr, ptr } %5387, 0
// CHECK-NEXT:   %__llgo_funcval_code722 = call ptr asm "", "=r,0"(ptr %5389)
// CHECK-NEXT:   %5390 = call %reflect.Value %__llgo_funcval_code722(ptr {{(nest|swiftself)}} %5388, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanRecv, ptr null })
// CHECK-NEXT:   %5391 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5385, i32 0, i32 1
// CHECK-NEXT:   %5392 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5393 = extractvalue { ptr, ptr } %5392, 1
// CHECK-NEXT:   %5394 = extractvalue { ptr, ptr } %5392, 0
// CHECK-NEXT:   %__llgo_funcval_code723 = call ptr asm "", "=r,0"(ptr %5394)
// CHECK-NEXT:   %5395 = call %reflect.Value %__llgo_funcval_code723(ptr {{(nest|swiftself)}} %5393, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5390, ptr %5386, align 8
// CHECK-NEXT:   store %reflect.Value %5395, ptr %5391, align 8
// CHECK-NEXT:   %5396 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 362
// CHECK-NEXT:   %5397 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5396, i32 0, i32 0
// CHECK-NEXT:   %5398 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5399 = extractvalue { ptr, ptr } %5398, 1
// CHECK-NEXT:   %5400 = extractvalue { ptr, ptr } %5398, 0
// CHECK-NEXT:   %__llgo_funcval_code724 = call ptr asm "", "=r,0"(ptr %5400)
// CHECK-NEXT:   %5401 = call %reflect.Value %__llgo_funcval_code724(ptr {{(nest|swiftself)}} %5399, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanSend, ptr null })
// CHECK-NEXT:   %5402 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5396, i32 0, i32 1
// CHECK-NEXT:   %5403 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5404 = extractvalue { ptr, ptr } %5403, 1
// CHECK-NEXT:   %5405 = extractvalue { ptr, ptr } %5403, 0
// CHECK-NEXT:   %__llgo_funcval_code725 = call ptr asm "", "=r,0"(ptr %5405)
// CHECK-NEXT:   %5406 = call %reflect.Value %__llgo_funcval_code725(ptr {{(nest|swiftself)}} %5404, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.IntChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5401, ptr %5397, align 8
// CHECK-NEXT:   store %reflect.Value %5406, ptr %5402, align 8
// CHECK-NEXT:   %5407 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 363
// CHECK-NEXT:   %5408 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5407, i32 0, i32 0
// CHECK-NEXT:   %5409 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5410 = extractvalue { ptr, ptr } %5409, 1
// CHECK-NEXT:   %5411 = extractvalue { ptr, ptr } %5409, 0
// CHECK-NEXT:   %__llgo_funcval_code726 = call ptr asm "", "=r,0"(ptr %5411)
// CHECK-NEXT:   %5412 = call %reflect.Value %__llgo_funcval_code726(ptr {{(nest|swiftself)}} %5410, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   %5413 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5407, i32 0, i32 1
// CHECK-NEXT:   %5414 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5415 = extractvalue { ptr, ptr } %5414, 1
// CHECK-NEXT:   %5416 = extractvalue { ptr, ptr } %5414, 0
// CHECK-NEXT:   %__llgo_funcval_code727 = call ptr asm "", "=r,0"(ptr %5416)
// CHECK-NEXT:   %5417 = call %reflect.Value %__llgo_funcval_code727(ptr {{(nest|swiftself)}} %5415, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChan, ptr null })
// CHECK-NEXT:   store %reflect.Value %5412, ptr %5408, align 8
// CHECK-NEXT:   store %reflect.Value %5417, ptr %5413, align 8
// CHECK-NEXT:   %5418 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 364
// CHECK-NEXT:   %5419 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5418, i32 0, i32 0
// CHECK-NEXT:   %5420 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5421 = extractvalue { ptr, ptr } %5420, 1
// CHECK-NEXT:   %5422 = extractvalue { ptr, ptr } %5420, 0
// CHECK-NEXT:   %__llgo_funcval_code728 = call ptr asm "", "=r,0"(ptr %5422)
// CHECK-NEXT:   %5423 = call %reflect.Value %__llgo_funcval_code728(ptr {{(nest|swiftself)}} %5421, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanRecv, ptr null })
// CHECK-NEXT:   %5424 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5418, i32 0, i32 1
// CHECK-NEXT:   %5425 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5426 = extractvalue { ptr, ptr } %5425, 1
// CHECK-NEXT:   %5427 = extractvalue { ptr, ptr } %5425, 0
// CHECK-NEXT:   %__llgo_funcval_code729 = call ptr asm "", "=r,0"(ptr %5427)
// CHECK-NEXT:   %5428 = call %reflect.Value %__llgo_funcval_code729(ptr {{(nest|swiftself)}} %5426, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanRecv, ptr null })
// CHECK-NEXT:   store %reflect.Value %5423, ptr %5419, align 8
// CHECK-NEXT:   store %reflect.Value %5428, ptr %5424, align 8
// CHECK-NEXT:   %5429 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 365
// CHECK-NEXT:   %5430 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5429, i32 0, i32 0
// CHECK-NEXT:   %5431 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5432 = extractvalue { ptr, ptr } %5431, 1
// CHECK-NEXT:   %5433 = extractvalue { ptr, ptr } %5431, 0
// CHECK-NEXT:   %__llgo_funcval_code730 = call ptr asm "", "=r,0"(ptr %5433)
// CHECK-NEXT:   %5434 = call %reflect.Value %__llgo_funcval_code730(ptr {{(nest|swiftself)}} %5432, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanSend, ptr null })
// CHECK-NEXT:   %5435 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5429, i32 0, i32 1
// CHECK-NEXT:   %5436 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5437 = extractvalue { ptr, ptr } %5436, 1
// CHECK-NEXT:   %5438 = extractvalue { ptr, ptr } %5436, 0
// CHECK-NEXT:   %__llgo_funcval_code731 = call ptr asm "", "=r,0"(ptr %5438)
// CHECK-NEXT:   %5439 = call %reflect.Value %__llgo_funcval_code731(ptr {{(nest|swiftself)}} %5437, %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_main.BytesChanSend, ptr null })
// CHECK-NEXT:   store %reflect.Value %5434, ptr %5430, align 8
// CHECK-NEXT:   store %reflect.Value %5439, ptr %5435, align 8
// CHECK-NEXT:   %5440 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 366
// CHECK-NEXT:   %5441 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5440, i32 0, i32 0
// CHECK-NEXT:   %5442 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5443 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %5443, align 8
// CHECK-NEXT:   %5444 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %5443, 1
// CHECK-NEXT:   %5445 = extractvalue { ptr, ptr } %5442, 1
// CHECK-NEXT:   %5446 = extractvalue { ptr, ptr } %5442, 0
// CHECK-NEXT:   %__llgo_funcval_code732 = call ptr asm "", "=r,0"(ptr %5446)
// CHECK-NEXT:   %5447 = call %reflect.Value %__llgo_funcval_code732(ptr {{(nest|swiftself)}} %5445, %"{{.*}}/runtime/internal/runtime.eface" %5444)
// CHECK-NEXT:   %5448 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5440, i32 0, i32 1
// CHECK-NEXT:   %5449 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 1, ptr %5449, align 8
// CHECK-NEXT:   %5450 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %5449, 1
// CHECK-NEXT:   %5451 = call %reflect.Value @main.EmptyInterfaceV(%"{{.*}}/runtime/internal/runtime.eface" %5450)
// CHECK-NEXT:   store %reflect.Value %5447, ptr %5441, align 8
// CHECK-NEXT:   store %reflect.Value %5451, ptr %5448, align 8
// CHECK-NEXT:   %5452 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 367
// CHECK-NEXT:   %5453 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5452, i32 0, i32 0
// CHECK-NEXT:   %5454 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5455 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %5455, align 8
// CHECK-NEXT:   %5456 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5455, 1
// CHECK-NEXT:   %5457 = extractvalue { ptr, ptr } %5454, 1
// CHECK-NEXT:   %5458 = extractvalue { ptr, ptr } %5454, 0
// CHECK-NEXT:   %__llgo_funcval_code733 = call ptr asm "", "=r,0"(ptr %5458)
// CHECK-NEXT:   %5459 = call %reflect.Value %__llgo_funcval_code733(ptr {{(nest|swiftself)}} %5457, %"{{.*}}/runtime/internal/runtime.eface" %5456)
// CHECK-NEXT:   %5460 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5452, i32 0, i32 1
// CHECK-NEXT:   %5461 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 5 }, ptr %5461, align 8
// CHECK-NEXT:   %5462 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %5461, 1
// CHECK-NEXT:   %5463 = call %reflect.Value @main.EmptyInterfaceV(%"{{.*}}/runtime/internal/runtime.eface" %5462)
// CHECK-NEXT:   store %reflect.Value %5459, ptr %5453, align 8
// CHECK-NEXT:   store %reflect.Value %5463, ptr %5460, align 8
// CHECK-NEXT:   %5464 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 368
// CHECK-NEXT:   %5465 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5464, i32 0, i32 0
// CHECK-NEXT:   %5466 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5467 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5468 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_bytes.Buffer", ptr undef }, ptr %5467, 1
// CHECK-NEXT:   %5469 = extractvalue { ptr, ptr } %5466, 1
// CHECK-NEXT:   %5470 = extractvalue { ptr, ptr } %5466, 0
// CHECK-NEXT:   %__llgo_funcval_code734 = call ptr asm "", "=r,0"(ptr %5470)
// CHECK-NEXT:   %5471 = call %reflect.Value %__llgo_funcval_code734(ptr {{(nest|swiftself)}} %5469, %"{{.*}}/runtime/internal/runtime.eface" %5468)
// CHECK-NEXT:   %5472 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5464, i32 0, i32 1
// CHECK-NEXT:   %5473 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5474 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uycIKA3bbxRhudEjW1hHKWKdLqHQsCVy8NdW1bkQmNw", ptr @"*_llgo_bytes.Buffer")
// CHECK-NEXT:   %5475 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5474, 0
// CHECK-NEXT:   %5476 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5475, ptr %5473, 1
// CHECK-NEXT:   %5477 = call %reflect.Value @main.ReaderV(%"{{.*}}/runtime/internal/runtime.iface" %5476)
// CHECK-NEXT:   store %reflect.Value %5471, ptr %5465, align 8
// CHECK-NEXT:   store %reflect.Value %5477, ptr %5472, align 8
// CHECK-NEXT:   %5478 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 369
// CHECK-NEXT:   %5479 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5478, i32 0, i32 0
// CHECK-NEXT:   %5480 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5481 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Zutt7i_AwOTtBOIzyS7ZA5vhcNcbk0kRAoRC98HJDos", ptr @"*_llgo_bytes.Buffer")
// CHECK-NEXT:   %5482 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5481, 0
// CHECK-NEXT:   %5483 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5482, ptr %5480, 1
// CHECK-NEXT:   %5484 = call %reflect.Value @main.ReadWriterV(%"{{.*}}/runtime/internal/runtime.iface" %5483)
// CHECK-NEXT:   %5485 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5478, i32 0, i32 1
// CHECK-NEXT:   %5486 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5487 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$uycIKA3bbxRhudEjW1hHKWKdLqHQsCVy8NdW1bkQmNw", ptr @"*_llgo_bytes.Buffer")
// CHECK-NEXT:   %5488 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5487, 0
// CHECK-NEXT:   %5489 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5488, ptr %5486, 1
// CHECK-NEXT:   %5490 = call %reflect.Value @main.ReaderV(%"{{.*}}/runtime/internal/runtime.iface" %5489)
// CHECK-NEXT:   store %reflect.Value %5484, ptr %5479, align 8
// CHECK-NEXT:   store %reflect.Value %5490, ptr %5485, align 8
// CHECK-NEXT:   %5491 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %1, i64 370
// CHECK-NEXT:   %5492 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5491, i32 0, i32 0
// CHECK-NEXT:   %5493 = load { ptr, ptr }, ptr @main.V, align 8
// CHECK-NEXT:   %5494 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5495 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_bytes.Buffer", ptr undef }, ptr %5494, 1
// CHECK-NEXT:   %5496 = extractvalue { ptr, ptr } %5493, 1
// CHECK-NEXT:   %5497 = extractvalue { ptr, ptr } %5493, 0
// CHECK-NEXT:   %__llgo_funcval_code735 = call ptr asm "", "=r,0"(ptr %5497)
// CHECK-NEXT:   %5498 = call %reflect.Value %__llgo_funcval_code735(ptr {{(nest|swiftself)}} %5496, %"{{.*}}/runtime/internal/runtime.eface" %5495)
// CHECK-NEXT:   %5499 = getelementptr inbounds { %reflect.Value, %reflect.Value }, ptr %5491, i32 0, i32 1
// CHECK-NEXT:   %5500 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 40)
// CHECK-NEXT:   %5501 = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr @"_llgo_iface$Zutt7i_AwOTtBOIzyS7ZA5vhcNcbk0kRAoRC98HJDos", ptr @"*_llgo_bytes.Buffer")
// CHECK-NEXT:   %5502 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %5501, 0
// CHECK-NEXT:   %5503 = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %5502, ptr %5500, 1
// CHECK-NEXT:   %5504 = call %reflect.Value @main.ReadWriterV(%"{{.*}}/runtime/internal/runtime.iface" %5503)
// CHECK-NEXT:   store %reflect.Value %5498, ptr %5492, align 8
// CHECK-NEXT:   store %reflect.Value %5504, ptr %5499, align 8
// CHECK-NEXT:   %5505 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %1, 0
// CHECK-NEXT:   %5506 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5505, i64 371, 1
// CHECK-NEXT:   %5507 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %5506, i64 371, 2
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.Slice" %5507, ptr @main.convertTests, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @main.TestConvert(ptr @"__llgo.moduleZeroSizedAlloc$")
// CHECK-NEXT:   call void @main.TestConvertPanic(ptr @"__llgo.moduleZeroSizedAlloc$")
// CHECK-NEXT:   call void @main.TestConvertSlice2Array(ptr @"__llgo.moduleZeroSizedAlloc$")
// CHECK-NEXT:   call void @main.TestConvertNaNs(ptr @"__llgo.moduleZeroSizedAlloc$")
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @main.shouldPanic(%"{{.*}}/runtime/internal/runtime.String" %0, { ptr, ptr } %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %0, ptr %2, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %4 = getelementptr inbounds { ptr }, ptr %3, i32 0, i32 0
// CHECK-NEXT:   store ptr %2, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue { ptr, ptr } { ptr @"main.shouldPanic$1", ptr undef }, ptr %3, 1
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.GetThreadDefer"()
// CHECK-NEXT:   %7 = alloca i8
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 48)
// CHECK-NEXT:   %9 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 0
// CHECK-NEXT:   store ptr %7, ptr %9, align 8
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %10, align 8
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 2
// CHECK-NEXT:   store ptr %6, ptr %11, align 8
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   store ptr blockaddress(@main.shouldPanic, %_llgo_2), ptr %12, align 8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %8)
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 1
// CHECK-NEXT:   %14 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 3
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 4
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, i32 0, i32 5
// CHECK-NEXT:   store ptr null, ptr %16, align 8
// CHECK-NEXT:   %17 = call i32 @{{(__)?}}sigsetjmp(ptr %7, i32 0)
// CHECK-NEXT:   %18 = icmp eq i32 %17, 0
// CHECK-NEXT:   br i1 %18, label %_llgo_4, label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_3
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_5, %_llgo_4
// CHECK-NEXT:   store ptr blockaddress(@main.shouldPanic, %_llgo_3), ptr %14, align 8
// CHECK-NEXT:   %19 = load i64, ptr %13, align 8
// CHECK-NEXT:   %20 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %21 = icmp ne ptr %20, null
// CHECK-NEXT:   br i1 %21, label %_llgo_7, label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_5, %_llgo_8
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Rethrow"(ptr %6)
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %22 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 32)
// CHECK-NEXT:   %24 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 0
// CHECK-NEXT:   store ptr %22, ptr %24, align 8
// CHECK-NEXT:   %25 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 1
// CHECK-NEXT:   store i64 0, ptr %25, align 8
// CHECK-NEXT:   %26 = getelementptr inbounds { ptr, i64, { ptr, ptr } }, ptr %23, i32 0, i32 2
// CHECK-NEXT:   store { ptr, ptr } %5, ptr %26, align 8
// CHECK-NEXT:   store ptr %23, ptr %16, align 8
// CHECK-NEXT:   %27 = extractvalue { ptr, ptr } %1, 1
// CHECK-NEXT:   %28 = extractvalue { ptr, ptr } %1, 0
// CHECK-NEXT:   %__llgo_funcval_code = call ptr asm "", "=r,0"(ptr %28)
// CHECK-NEXT:   call void %__llgo_funcval_code(ptr {{(nest|swiftself)}} %27)
// CHECK-NEXT:   store ptr blockaddress(@main.shouldPanic, %_llgo_6), ptr %15, align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store ptr blockaddress(@main.shouldPanic, %_llgo_3), ptr %15, align 8
// CHECK-NEXT:   %29 = load ptr, ptr %14, align 8
// CHECK-NEXT:   indirectbr ptr %29, [label %_llgo_3, label %_llgo_2]
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_8
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %30 = load ptr, ptr %16, align 8
// CHECK-NEXT:   %31 = load { ptr, i64, { ptr, ptr } }, ptr %30, align 8
// CHECK-NEXT:   %32 = extractvalue { ptr, i64, { ptr, ptr } } %31, 0
// CHECK-NEXT:   store ptr %32, ptr %16, align 8
// CHECK-NEXT:   %33 = extractvalue { ptr, i64, { ptr, ptr } } %31, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.FreeDeferNode"(ptr %30)
// CHECK-NEXT:   %34 = extractvalue { ptr, ptr } %33, 1
// CHECK-NEXT:   %35 = extractvalue { ptr, ptr } %33, 0
// CHECK-NEXT:   %__llgo_funcval_code1 = call ptr asm "", "=r,0"(ptr %35)
// CHECK-NEXT:   call void %__llgo_funcval_code1(ptr {{(nest|swiftself)}} %34)
// CHECK-NEXT:   br label %_llgo_8
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_7, %_llgo_2
// CHECK-NEXT:   %36 = load %"{{.*}}/runtime/internal/runtime.Defer", ptr %8, align 8
// CHECK-NEXT:   %37 = extractvalue %"{{.*}}/runtime/internal/runtime.Defer" %36, 2
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.SetThreadDefer"(ptr %37)
// CHECK-NEXT:   %38 = load ptr, ptr %15, align 8
// CHECK-NEXT:   indirectbr ptr %38, [label %_llgo_3, label %_llgo_6]
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.shouldPanic$1"(ptr {{(nest|swiftself)}} %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
// CHECK-NEXT:   %2 = call %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/runtime/internal/runtime.Recover"()
// CHECK-NEXT:   %3 = call i1 @"{{.*}}/runtime/internal/runtime.EfaceEqual"(%"{{.*}}/runtime/internal/runtime.eface" %2, %"{{.*}}/runtime/internal/runtime.eface" zeroinitializer)
// CHECK-NEXT:   br i1 %3, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 13 }, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_0
// CHECK-NEXT:   %6 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %7 = load %"{{.*}}/runtime/internal/runtime.String", ptr %6, align 8
// CHECK-NEXT:   %8 = call i1 @"{{.*}}/runtime/internal/runtime.StringEqual"(%"{{.*}}/runtime/internal/runtime.String" %7, %"{{.*}}/runtime/internal/runtime.String" zeroinitializer)
// CHECK-NEXT:   %9 = xor i1 %8, true
// CHECK-NEXT:   br i1 %9, label %_llgo_3, label %_llgo_4
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:                                          ; preds = %_llgo_2
// CHECK-NEXT:   %10 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 0
// CHECK-NEXT:   %11 = icmp eq ptr %10, @_llgo_string
// CHECK-NEXT:   br i1 %11, label %_llgo_13, label %_llgo_14
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_4:                                          ; preds = %_llgo_11, %_llgo_2
// CHECK-NEXT:   ret void
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_5:                                          ; preds = %_llgo_8, %_llgo_6
// CHECK-NEXT:   %12 = phi %"{{.*}}/runtime/internal/runtime.String" [ %43, %_llgo_6 ], [ %16, %_llgo_8 ]
// CHECK-NEXT:   %13 = call i1 @strings.HasPrefix(%"{{.*}}/runtime/internal/runtime.String" %12, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 7 })
// CHECK-NEXT:   br i1 %13, label %_llgo_11, label %_llgo_10
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_6:                                          ; preds = %_llgo_15
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_7:                                          ; preds = %_llgo_15
// CHECK-NEXT:   %14 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 0
// CHECK-NEXT:   %15 = icmp eq ptr %14, @"*_llgo_reflect.ValueError"
// CHECK-NEXT:   br i1 %15, label %_llgo_16, label %_llgo_17
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_8:                                          ; preds = %_llgo_18
// CHECK-NEXT:   %16 = call %"{{.*}}/runtime/internal/runtime.String" @"reflect.(*ValueError).Error"(ptr %49)
// CHECK-NEXT:   br label %_llgo_5
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_9:                                          ; preds = %_llgo_18
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/runtime/internal/runtime.eface", ptr %17, i64 0
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" undef, ptr %17, 0
// CHECK-NEXT:   %20 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %19, i64 1, 1
// CHECK-NEXT:   %21 = insertvalue %"{{.*}}/runtime/internal/runtime.Slice" %20, i64 1, 2
// CHECK-NEXT:   %22 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 32 }, %"{{.*}}/runtime/internal/runtime.Slice" %21)
// CHECK-NEXT:   %23 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %22, ptr %23, align 8
// CHECK-NEXT:   %24 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %23, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %24)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_10:                                         ; preds = %_llgo_5
// CHECK-NEXT:   %25 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 44 }, %"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   %26 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %25, ptr %26, align 8
// CHECK-NEXT:   %27 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %26, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %27)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_11:                                         ; preds = %_llgo_5
// CHECK-NEXT:   %28 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %29 = load %"{{.*}}/runtime/internal/runtime.String", ptr %28, align 8
// CHECK-NEXT:   %30 = call i1 @strings.Contains(%"{{.*}}/runtime/internal/runtime.String" %12, %"{{.*}}/runtime/internal/runtime.String" %29)
// CHECK-NEXT:   br i1 %30, label %_llgo_4, label %_llgo_12
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_12:                                         ; preds = %_llgo_11
// CHECK-NEXT:   %31 = extractvalue { ptr } %1, 0
// CHECK-NEXT:   %32 = load %"{{.*}}/runtime/internal/runtime.String", ptr %31, align 8
// CHECK-NEXT:   %33 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 31 }, %"{{.*}}/runtime/internal/runtime.String" %32)
// CHECK-NEXT:   %34 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %33, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 3 })
// CHECK-NEXT:   %35 = call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"(%"{{.*}}/runtime/internal/runtime.String" %34, %"{{.*}}/runtime/internal/runtime.String" %12)
// CHECK-NEXT:   %36 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %35, ptr %36, align 8
// CHECK-NEXT:   %37 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %36, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %37)
// CHECK-NEXT:   unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_13:                                         ; preds = %_llgo_3
// CHECK-NEXT:   %38 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 1
// CHECK-NEXT:   %39 = load %"{{.*}}/runtime/internal/runtime.String", ptr %38, align 8
// CHECK-NEXT:   %40 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } undef, %"{{.*}}/runtime/internal/runtime.String" %39, 0
// CHECK-NEXT:   %41 = insertvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %40, i1 true, 1
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_14:                                         ; preds = %_llgo_3
// CHECK-NEXT:   br label %_llgo_15
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_15:                                         ; preds = %_llgo_14, %_llgo_13
// CHECK-NEXT:   %42 = phi { %"{{.*}}/runtime/internal/runtime.String", i1 } [ %41, %_llgo_13 ], [ zeroinitializer, %_llgo_14 ]
// CHECK-NEXT:   %43 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %42, 0
// CHECK-NEXT:   %44 = extractvalue { %"{{.*}}/runtime/internal/runtime.String", i1 } %42, 1
// CHECK-NEXT:   br i1 %44, label %_llgo_6, label %_llgo_7
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_16:                                         ; preds = %_llgo_7
// CHECK-NEXT:   %45 = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %2, 1
// CHECK-NEXT:   %46 = insertvalue { ptr, i1 } undef, ptr %45, 0
// CHECK-NEXT:   %47 = insertvalue { ptr, i1 } %46, i1 true, 1
// CHECK-NEXT:   br label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_17:                                         ; preds = %_llgo_7
// CHECK-NEXT:   br label %_llgo_18
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_18:                                         ; preds = %_llgo_17, %_llgo_16
// CHECK-NEXT:   %48 = phi { ptr, i1 } [ %47, %_llgo_16 ], [ zeroinitializer, %_llgo_17 ]
// CHECK-NEXT:   %49 = extractvalue { ptr, i1 } %48, 0
// CHECK-NEXT:   %50 = extractvalue { ptr, i1 } %48, 1
// CHECK-NEXT:   br i1 %50, label %_llgo_8, label %_llgo_9
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.(*testingT).Errorf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }

// CHECK-LABEL: define void @"main.(*testingT).Fatalf"(ptr %0, %"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = call %"{{.*}}/runtime/internal/runtime.String" @fmt.Sprintf(%"{{.*}}/runtime/internal/runtime.String" %1, %"{{.*}}/runtime/internal/runtime.Slice" %2)
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" %3, ptr %4, align 8
// CHECK-NEXT:   %5 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_string, ptr undef }, ptr %4, 1
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.Panic"(%"{{.*}}/runtime/internal/runtime.eface" %5)
// CHECK-NEXT:   unreachable
// CHECK-NEXT: }
