//go:build (darwin || linux) && !baremetal && !wasm

package runtime

import "unsafe"

// Native LLGo goroutines already execute on their OS thread's native stack.
// The packed-argument ABI therefore needs no Go g0-stack transition. Preserve
// the caller's G and collector registration across nested C calls and callbacks;
// native traceback/profiling continues to walk the physical stack.
//
// fn must be a C-ABI entry taking one pointer and returning int32. This is a
// separate contract from syscall's ordinary, actual-argument trampolines.
func libcCall(fn, arg unsafe.Pointer) int32 {
	if fn == nil {
		panic("runtime: nil libc trampoline")
	}
	result := c_libcCall(fn, arg)
	KeepAlive(arg)
	return result
}

//go:linkname c_libcCall C.llgo_libc_call
func c_libcCall(fn, arg unsafe.Pointer) int32

// asmcgocall's native packed-argument entry shares libcCall's ABI. It does not
// accept a Go-ABI function pointer or marshal an arbitrary C signature.
func asmcgocall(fn, arg unsafe.Pointer) int32 { return libcCall(fn, arg) }
