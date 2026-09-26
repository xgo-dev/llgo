package deeppanic

import "unsafe"

//go:noinline
func panicSite() {
	panic("deep-panic")
}

//go:noinline
func deepCall(depth int) {
	if depth == 0 {
		panicSite()
		return
	}
	deepCall(depth - 1)
}

//go:noinline
func faultSite() byte { return *(*byte)(unsafe.Pointer(uintptr(0x12345))) }

//go:noinline
func deepFault(depth int) byte {
	if depth == 0 {
		return faultSite()
	}
	return 1 + deepFault(depth-1)
}
