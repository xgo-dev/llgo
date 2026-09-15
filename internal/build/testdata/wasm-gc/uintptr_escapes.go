package main

import (
	"runtime"
	"unsafe"
)

type uintptrRootObject struct {
	value   int
	padding [32]byte
}

//go:noinline
func newUintptrRoot() unsafe.Pointer {
	p := &uintptrRootObject{value: 47}
	runtime.SetFinalizer(p, func(p *uintptrRootObject) { p.value = -1 })
	return unsafe.Pointer(p)
}

//go:noinline
func laterUintptrRoot() unsafe.Pointer {
	// The preceding argument has already become uintptr, but the marked
	// callee has not entered and cannot publish its parameters yet.
	runtime.GC()
	runtime.GC()
	return newUintptrRoot()
}

//go:noinline
//go:uintptrescapes
func checkUintptrRoots(first, second uintptr, rest ...uintptr) {
	runtime.GC()
	runtime.GC()
	if (*uintptrRootObject)(unsafe.Pointer(first)).value != 47 ||
		(*uintptrRootObject)(unsafe.Pointer(second)).value != 47 {
		panic("pragma-designated uintptr root was finalized")
	}
	for _, p := range rest {
		if (*uintptrRootObject)(unsafe.Pointer(p)).value != 47 {
			panic("variadic uintptr root was finalized")
		}
	}
}

// This extends the existing Emscripten wasm32/wasm64 and WASI GC fixture, so it runs
// in CI without an additional test binary, workflow job, or dependency.
func testUintptrEscapesRoots() {
	checkUintptrRoots(uintptr(newUintptrRoot()), uintptr(laterUintptrRoot()),
		uintptr(newUintptrRoot()), uintptr(laterUintptrRoot()))
	func() {
		for i := 0; i < 4; i++ {
			defer checkUintptrRoots(uintptr(newUintptrRoot()), uintptr(newUintptrRoot()), uintptr(newUintptrRoot()))
		}
		runtime.GC()
	}()
}
