// LITTEST
// Scope: common
package main

import "unsafe"

var addressBuffer [16]byte

//go:noinline
func addressOne() any { return uintptr(unsafe.Pointer(&addressBuffer[1])) }

//go:noinline
func addressEight() any {
	return uintptr(unsafe.Pointer(&(*[2]uint64)(unsafe.Pointer(&addressBuffer))[1]))
}

type I interface {
	M() int
}

type T int

func (t T) M() int { return int(t) + 1 }

// Constant non-direct iface data (integers) go in a read-only box, not AllocU.
// CHECK-LABEL: define {{.*}} @main.boxed(){{.*}} {
// CHECK-NOT: AllocU
// CHECK: _llgo_ifacebox$
func boxed() any {
	return int64(42)
}

//go:noinline
func nulString() any { return "\x00" }

func twoStrings() (any, any) {
	return "aaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbbbbbbbbbbbbbbb"
}

var firstArray, secondArray [1]int
var firstBytes, secondBytes [1]byte
var firstNonzeroBytes = [1]byte{1}
var secondNonzeroBytes = [1]byte{1}

//go:noinline
func firstSlice() any { return firstArray[:] }

//go:noinline
func secondSlice() any { return secondArray[:] }

//go:noinline
func firstByteSlice() any { return firstBytes[:] }

//go:noinline
func secondByteSlice() any { return secondBytes[:] }

//go:noinline
func firstNonzeroByteSlice() any { return firstNonzeroBytes[:] }

//go:noinline
func secondNonzeroByteSlice() any { return secondNonzeroBytes[:] }

func main() {
	one, eight := addressOne().(uintptr), addressEight().(uintptr)
	if eight-one != 7 || one == eight {
		panic("GEP source element types shared an iface box")
	}
	v := boxed()
	if v.(int64) != 42 {
		panic(v)
	}
	if boxed() != any(int64(42)) {
		panic("eface equality")
	}
	if len(nulString().(string)) != 1 {
		panic("NUL string box")
	}
	var i I = T(41)
	if i.M() != 42 {
		panic(i.M())
	}
	a, b := twoStrings()
	if a.(string) == b.(string) {
		panic("same-length string boxes collided")
	}
	x, y := firstSlice().([]int), secondSlice().([]int)
	y[0] = 7
	if firstArray[0] != 0 || secondArray[0] != 7 || x[0] != 0 || &x[0] == &y[0] {
		panic("distinct global slice backing arrays shared an iface box")
	}
	bx, by := firstByteSlice().([]byte), secondByteSlice().([]byte)
	by[0] = 7
	if firstBytes[0] != 0 || secondBytes[0] != 7 || bx[0] != 0 || &bx[0] == &by[0] {
		panic("distinct zero byte-array globals shared an iface box")
	}
	nx, ny := firstNonzeroByteSlice().([]byte), secondNonzeroByteSlice().([]byte)
	ny[0] = 7
	if firstNonzeroBytes[0] != 1 || secondNonzeroBytes[0] != 7 || nx[0] != 1 || &nx[0] == &ny[0] {
		panic("distinct nonzero byte-array globals shared an iface box")
	}
	println("ok")
}
