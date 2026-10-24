//go:build llgo && (darwin || linux) && !wasm && !baremetal

package abicall

import (
	"runtime"
	"strings"
	"unsafe"
)

const LLGoFiles = "_wrap/libcall.c"

type block struct {
	a, b, result, depth, signal int64
	payload                     unsafe.Pointer
	traced                      int64
}

//go:linkname libcCall runtime.libcCall
func libcCall(unsafe.Pointer, unsafe.Pointer) int32

//go:linkname asmcgocall runtime.asmcgocall
func asmcgocall(unsafe.Pointer, unsafe.Pointer) int32

//go:linkname trampoline C.llgo_test_libcall_address
func trampoline() unsafe.Pointer

//go:linkname currentErrno C.llgo_test_libcall_errno
func currentErrno() int32

//go:linkname errnoRange C.llgo_test_libcall_erange
func errnoRange() int32

//export LLGoLibcallCallback
//go:noinline
func LLGoLibcallCallback(raw unsafe.Pointer) int32 {
	b := (*block)(raw)
	runtime.GC()
	if b.depth > 0 {
		child := block{a: b.a, b: b.b, depth: b.depth - 1, payload: b.payload}
		if asmcgocall(trampoline(), unsafe.Pointer(&child)) != 18 || child.result != 42 {
			panic("nested argument block")
		}
	}
	var pcs [32]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(0, pcs[:])])
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.Function, "LLGoLibcallCallback") {
			b.traced = 1
		}
		if !more {
			break
		}
	}
	if b.payload == nil || *(*byte)(b.payload) != 42 {
		panic("callback lost pointer root")
	}
	b.result = b.a + b.b
	return 17
}
