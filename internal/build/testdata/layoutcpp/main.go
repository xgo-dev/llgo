package main

import "unsafe"

const LLGoFiles = "_wrap/layout.cpp"

type Empty struct{}
type Pad struct {
	//llgo:padding
	E Empty
	B byte
}
type NoPad struct {
	X uint64
	//llgo:no-padding
	E Empty
}
type Pair[A, B any] struct {
	//llgo:padding-if
	A A
	//llgo:padding-if
	B B
}

//llgo:cdecl
//llgo:packed
type Packed struct {
	B byte
	X uint32
	Z byte
}

//llgo:cdecl
//llgo:pack 2
type Pack2 struct {
	B byte
	X uint32
	Y uint16
}

//llgo:cdecl
//llgo:packed
type Lane struct {
	X    uint32
	Tail byte
}

//llgo:cdecl
type PackedArray struct{ Lanes [2]Lane }

//go:linkname roundArray C.llgo_layout_round_array
func roundArray(x PackedArray) PackedArray

//go:linkname metric C.llgo_layout_metric
func metric(i int32) uint64

//go:linkname roundPacked C.llgo_layout_round_packed
func roundPacked(x Packed) Packed

//go:linkname roundPack2 C.llgo_layout_round_pack2
func roundPack2(x Pack2) Pack2

//go:linkname callback C.llgo_layout_callback
var callback func(int32) int32

//llgo:cdecl
type Callback func(int32) int32

func main() {
	if unsafe.Sizeof(callback) != unsafe.Sizeof(Callback(nil)) || callback(35) != 42 {
		panic("native callback variable")
	}
	values := []uintptr{
		unsafe.Sizeof(Pad{}), unsafe.Alignof(Pad{}), unsafe.Offsetof(Pad{}.B),
		unsafe.Sizeof(NoPad{}), unsafe.Sizeof([2]NoPad{}),
		unsafe.Sizeof(Pair[Empty, Empty]{}), unsafe.Offsetof(Pair[Empty, Empty]{}.B),
		unsafe.Sizeof(Packed{}), unsafe.Alignof(Packed{}), unsafe.Offsetof(Packed{}.X), unsafe.Offsetof(Packed{}.Z),
		unsafe.Sizeof(Pack2{}), unsafe.Alignof(Pack2{}), unsafe.Offsetof(Pack2{}.X), unsafe.Offsetof(Pack2{}.Y),
	}
	for i, value := range values {
		if got := metric(int32(i)); got != uint64(value) {
			println("layout mismatch", i, got, value)
			panic("C++ layout")
		}
	}
	if got := roundPacked(Packed{1, 2, 3}); got != (Packed{2, 4, 6}) {
		panic("packed C ABI")
	}
	if got := roundPack2(Pack2{1, 2, 3}); got != (Pack2{2, 4, 6}) {
		panic("pack2 C ABI")
	}
	if got := roundArray(PackedArray{[2]Lane{{3, 7}, {5, 9}}}); got != (PackedArray{[2]Lane{{4, 7}, {7, 9}}}) {
		panic("packed array C ABI")
	}
	if unsafe.Sizeof(roundPacked) != unsafe.Sizeof((func(Packed) Packed)(nil)) {
		panic("native function size")
	}
	println("layout cpp ok")
}
