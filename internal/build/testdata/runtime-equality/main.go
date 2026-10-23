package main

import (
	"math"
	"reflect"
	"unsafe"
)

var checks int

func check(name string, condition bool) {
	checks++
	if !condition {
		panic(name)
	}
}

func panics(call func()) (result bool) {
	defer func() { result = recover() != nil }()
	call()
	return false
}

type pointerTail struct {
	Pointer *uint64
	Tail    [2]uint64
}

type padded struct {
	Small byte
	Wide  uint64
}

type blank struct {
	A uint64
	_ uint64
	B uint64
}

func main() {
	// Pointer-free Size_ is 24 and PtrBytes is zero. The final byte matters,
	// so a zero length or the pointer-prefix length cannot accidentally pass.
	x := [3]uint64{1, 2, 3}
	y := x
	check("direct regular equality", x == y)
	check("captured regular equality", any(x) == any(y))
	y[0] ^= 1
	check("captured first-byte inequality", any(x) != any(y))
	y = x
	*(*byte)(unsafe.Add(unsafe.Pointer(&y), unsafe.Sizeof(y)-1)) ^= 0x80
	check("captured final-byte inequality", any(x) != any(y))
	check("captured regular nonzero", !reflect.ValueOf(&x).Elem().IsZero())
	zero := [3]uint64{}
	check("captured regular zero", reflect.ValueOf(&zero).Elem().IsZero())
	check("regular map equality", map[[3]uint64]int{x: 7}[x] == 7)

	// This object's pointer prefix ends before Tail. Equal pointees do not
	// make different pointer identities equal, and the last Tail byte matters.
	p, q := new(uint64), new(uint64)
	*p, *q = 42, 42
	a := pointerTail{Pointer: p, Tail: [2]uint64{5, 6}}
	b := a
	check("pointer-bearing captured equality", any(a) == any(b))
	b.Tail[1] ^= 1 << 63
	check("pointer-bearing final-tail inequality", any(a) != any(b))
	b = a
	b.Pointer = q
	check("distinct pointer identity", p != q && any(a) != any(b))

	// These types are not regular memory. Keep the existing per-element and
	// per-field closures, including IEEE equality rather than byte equality.
	f := [3]float64{math.NaN(), 1, 2}
	check("NaN array is not equal to itself", any(f) != any(f))
	positiveZero := [3]float64{}
	negativeZero := [3]float64{math.Copysign(0, -1)}
	check("signed-zero numerical equality", any(positiveZero) == any(negativeZero))
	check("signed-zero captured zero", reflect.ValueOf(&negativeZero).Elem().IsZero())

	// IsZero calls the actual type equality closure on this original address.
	// Boxing an ordinary value could discard padding before the comparison.
	pad := padded{}
	for i := uintptr(1); i < unsafe.Offsetof(pad.Wide); i++ {
		*(*byte)(unsafe.Add(unsafe.Pointer(&pad), i)) = 0xa5
	}
	check("padding is not numerical input", reflect.ValueOf(&pad).Elem().IsZero())
	hole := blank{}
	*(*uint64)(unsafe.Add(unsafe.Pointer(&hole), unsafe.Sizeof(uint64(0)))) = 0xdeadbeef
	check("blank field is ignored", reflect.ValueOf(&hole).Elem().IsZero())

	left := [2]any{0, []int{1}}
	right := [2]any{1, []int{1}}
	check("array stops before uncomparable tail", any(left) != any(right))
	right[0] = 0
	check("array still panics for reached slice", panics(func() { _ = any(left) == any(right) }))
	type fields struct {
		First int
		Tail  any
	}
	first := fields{First: 0, Tail: []int{1}}
	second := fields{First: 1, Tail: []int{1}}
	check("struct cheap mismatch retains short circuit", any(first) != any(second))
	second.First = 0
	check("struct still panics for reached slice", panics(func() { _ = any(first) == any(second) }))

	// Runtime-created types keep their real ordinary Go closure producers too.
	dynamic := reflect.ArrayOf(3, reflect.TypeOf(uint64(0)))
	value := reflect.New(dynamic).Elem()
	check("dynamic array zero", value.IsZero())
	value.Index(2).SetUint(1 << 63)
	check("dynamic array final-byte inequality", !value.IsZero())
	println("equality checks", checks)
}
