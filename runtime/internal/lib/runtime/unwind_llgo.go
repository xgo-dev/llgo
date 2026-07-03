//go:build !baremetal && !wasm

package runtime

import (
	"unsafe"

	_ "unsafe"
)

//go:linkname c_framepointer C.llgo_framepointer
func c_framepointer() unsafe.Pointer

// fpCallers walks the frame-pointer chain and fills pc with return
// addresses, Go-style: pc[0] is the return address in the frame `skip`
// levels above the caller of fpCallers. Every LLGo-compiled function keeps
// x29/rbp chained ("frame-pointer"="non-leaf" is set on all Go functions),
// so unlike the shadow stack this sees every physical frame; the walk stops
// at the first frame that breaks the chain discipline (e.g. foreign C code
// compiled without frame pointers).
//
//go:noinline
func fpCallers(skip int, pc []uintptr) int {
	if len(pc) == 0 {
		return 0
	}
	// The walk bound needs the frame table's text range; make sure it is
	// built (no-op when the prebuilt table was adopted at startup).
	initRuntimeFuncPCFrames()
	fp := uintptr(c_framepointer())
	n := 0
	// The helper's saved chain starts at our own frame; skip fpCallers
	// itself so skip counting matches the caller's view.
	skip++
	const maxFrames = 4096
	for i := 0; fp != 0 && n < len(pc) && i < maxFrames; i++ {
		prev := *(*uintptr)(unsafe.Pointer(fp))
		ret := *(*uintptr)(unsafe.Pointer(fp + unsafe.Sizeof(uintptr(0))))
		if ret < 4096 {
			break
		}
		// Beyond main the chain runs into libc frames without FP
		// discipline; their slots decode as wild pcs that nearest-below
		// symbolization would map to arbitrary functions. Bound the walk
		// to the program's own text (Go tracebacks stop at runtime.main
		// for the same reason).
		if !prebuiltTextContains(ret) {
			break
		}
		if skip > 0 {
			skip--
		} else {
			pc[n] = ret
			n++
		}
		// Stacks grow down, so the chain must strictly increase; bound the
		// stride so a corrupt slot cannot walk off the stack.
		if prev <= fp || prev-fp > 1<<20 || prev&(unsafe.Sizeof(uintptr(0))-1) != 0 {
			break
		}
		fp = prev
	}
	return n
}

// fpUnwindAvailable reports whether the physical walk can be used for the
// public stack APIs. The frame-pointer attribute ships with the same
// toolchain that builds this runtime, so presence of the funcinfo tables is
// the pairing signal; without them symbolization would fall back to dlsym
// anyway.
func fpUnwindAvailable() bool {
	return runtimeFuncInfoTable != nil && runtimeFuncInfoCount > 0
}
