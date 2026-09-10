package main

import "unsafe"

const LLGoFiles = "_wrap/profile.c"

//go:linkname cPointerSize C.llgo_wasm_profile_pointer_size
func cPointerSize() uintptr

//go:linkname cLongSize C.llgo_wasm_profile_long_size
func cLongSize() uintptr

//go:linkname cEchoSize C.llgo_wasm_profile_echo_size
func cEchoSize(uintptr) uintptr

type pointerBox struct {
	p *byte
	b byte
}

func keepDynamic(fn func()) func() {
	return fn
}

func plainRecover() {
	if got := recover(); got != 42 {
		panic("plain function value did not recover")
	}
}

func checkClosureABI() {
	base := 41
	add1 := func() int { return base + 1 }
	if got := add1(); got != 42 {
		panic("captured closure call failed")
	}
	func() {
		defer keepDynamic(plainRecover)()
		panic(42)
	}()
}

func checkNativeNarrowing() {
	value := ^uintptr(0)
	if expectedCPointerSize == expectedGoWordSize {
		if got := cEchoSize(value); got != value {
			panic("native word round trip failed")
		}
		return
	}
	recovered := false
	func() {
		defer func() {
			recovered = recover() != nil
		}()
		cEchoSize(value)
	}()
	if !recovered {
		panic("unchecked native word narrowing")
	}
}

func main() {
	if got := unsafe.Sizeof(uintptr(0)); got != expectedGoWordSize {
		panic("unexpected Go word size")
	}
	if got := unsafe.Sizeof((*byte)(nil)); got != expectedGoWordSize {
		panic("unexpected Go pointer storage size")
	}
	if got := unsafe.Alignof((*byte)(nil)); got != expectedGoWordSize {
		panic("unexpected Go pointer alignment")
	}
	if got := cPointerSize(); got != expectedCPointerSize {
		panic("unexpected C pointer size")
	}
	if got := cLongSize(); got != expectedCLongSize {
		panic("unexpected C long size")
	}
	box := pointerBox{}
	if got := unsafe.Offsetof(box.b); got != expectedGoWordSize {
		panic("unexpected Go pointer field size")
	}
	*(*byte)(unsafe.Add(unsafe.Pointer(&box), unsafe.Offsetof(box.b))) = 42
	if box.b != 42 {
		panic("Go pointer field layout disagrees with unsafe.Offsetof")
	}
	checkClosureABI()
	checkNativeNarrowing()
	println("wasm ABI profile ok")
}
