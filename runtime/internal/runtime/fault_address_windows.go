//go:build windows

package runtime

import "unsafe"

// PanicOnFault reports whether the current goroutine opted into recovering
// unexpected non-nil memory faults. It avoids creating a runtime context when
// called from a platform exception handler on an unrelated host thread.
func PanicOnFault() bool {
	gp := (*g)(unsafe.Pointer(currentG))
	return gp != nil && gp.paniconfault
}

// errorAddressString matches the optional fault-address contract documented
// by runtime/debug.SetPanicOnFault.
type errorAddressString struct {
	msg  string
	addr uintptr
}

func (e errorAddressString) RuntimeError() {}

func (e errorAddressString) Error() string {
	return "runtime error: " + e.msg
}

func (e errorAddressString) Addr() uintptr {
	return e.addr
}

// PanicSignalAddr converts an unexpected memory fault enabled through
// runtime/debug.SetPanicOnFault and preserves the best-effort fault address.
//
//llgo:cold
//llgo:noreturn
func PanicSignalAddr(addr uintptr) {
	panic(errorAddressString{
		msg:  "invalid memory address or nil pointer dereference",
		addr: addr,
	})
}
