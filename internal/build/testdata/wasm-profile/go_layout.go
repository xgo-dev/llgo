package main

import (
	"reflect"
	"strconv"
	"unsafe"
)

// This source runs unchanged with gc and LLGo on js/wasm and wasip1/wasm.
// Function values are tested separately: LLGo stores a code/environment pair.
func checkGoLayout() {
	type record struct {
		B    byte
		P    *byte
		N    int
		Tail [0]uint64
	}
	for _, tc := range []struct {
		value any
		size  uintptr
	}{
		{int(0), 8}, {uint(0), 8}, {uintptr(0), 8}, {(*byte)(nil), 8},
		{"", 16}, {[]byte(nil), 24}, {map[int]int(nil), 8}, {chan int(nil), 8},
		{record{}, 32}, {[2]record{}, 64},
	} {
		t := reflect.TypeOf(tc.value)
		if t.Size() != tc.size || t.Align() != 8 {
			println(t.String(), t.Size(), t.Align(), tc.size)
			panic("Go64 reflected layout mismatch")
		}
	}
	var iface any
	var r record
	if strconv.IntSize != 64 || unsafe.Sizeof(iface) != 16 || unsafe.Alignof(iface) != 8 ||
		unsafe.Sizeof(r) != 32 || unsafe.Offsetof(r.P) != 8 || unsafe.Offsetof(r.N) != 16 || unsafe.Offsetof(r.Tail) != 24 {
		panic("Go64 unsafe layout mismatch")
	}
	n := int64(1<<63 - 1)
	word := int(n)
	word++
	if int64(word) != -1<<63 {
		panic("Go int overflow is not 64-bit")
	}
	u := ^uintptr(0)
	if uint64(u) != ^uint64(0) || u+1 != 0 {
		panic("Go uintptr overflow is not 64-bit")
	}
}
