package main

import (
	"math"
	"unsafe"
)

type Key struct{ A, B, C uint64 }
type Other struct{ Data [3]uint64 }
type Blank struct {
	A uint64
	_ uint64
}
type Padded struct {
	A byte
	B uint64
}

func check[T comparable](a, b T) {
	m := map[T]int{a: 41}
	if m[a] != 41 || m[b] != 0 || a == b {
		panic("key equality")
	}
	m[b] = 42
	delete(m, a)
	if len(m) != 1 || m[b] != 42 {
		panic("key mutation")
	}
}

func main() {
	check(Key{1, 2, 3}, Key{1, 2, 4})
	check(Other{[3]uint64{1, 2, 3}}, Other{[3]uint64{1, 2, 4}})
	check([1]byte{1}, [1]byte{2})
	check([2]byte{1}, [2]byte{2})
	check([4]byte{1}, [4]byte{2})
	check([8]byte{1}, [8]byte{2})
	check([16]byte{1}, [16]byte{2})
	check(struct{ Key }{Key{1, 2, 3}}, struct{ Key }{Key{1, 2, 4}})
	check(Padded{1, 2}, Padded{1, 3})
	check(struct{ S string }{"one"}, struct{ S string }{"two"})
	check(struct{ F float64 }{1.5}, struct{ F float64 }{2.5})
	check(struct{ V any }{1}, struct{ V any }{2})
	x, y := 1, 2
	check(struct{ P *int }{&x}, struct{ P *int }{&y})
	pointers := map[struct{ P *int }]int{{&x}: 42}
	if pointers[struct{ P *int }{&x}] != 42 {
		panic("pointer field equality")
	}
	empty := map[[0]uint64]int{{}: 42}
	if empty[[0]uint64{}] != 42 {
		panic("zero length key")
	}
	if (map[[0]string]int{{}: 42})[[0]string{}] != 42 {
		panic("zero length nonregular element")
	}
	// Dirty bytes in blank fields and padding must not affect equality/hash.
	a, b := Blank{A: 7}, Blank{A: 7}
	*(*uint64)(unsafe.Add(unsafe.Pointer(&b), 8)) = 99
	if a != b || (map[Blank]int{a: 42})[b] != 42 {
		panic("blank field equality")
	}
	p, q := Padded{1, 2}, Padded{1, 2}
	*(*byte)(unsafe.Add(unsafe.Pointer(&q), 1)) = 99
	if p != q || (map[Padded]int{p: 42})[q] != 42 {
		panic("padding equality")
	}
	type Float struct{ F float64 }
	negativeZero := Float{math.Copysign(0, -1)}
	if (map[Float]int{{0}: 42})[negativeZero] != 42 {
		panic("signed zero equality")
	}
	nan := Float{math.NaN()}
	if nan == nan || (map[Float]int{nan: 42})[nan] != 0 {
		panic("NaN equality")
	}
	println("wasm type fields passed")
}
