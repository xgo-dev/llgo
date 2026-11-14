//go:build linux && !baremetal && !wasm

package syscall

import "unsafe"

// LLGo's pthread/libc runtime cannot exclude all foreign C thread creators.
// Ordinary executables and all C library modes keep Go's cgo boundary. The
// explicit controlled-thread mode is a caller contract; a TID snapshot alone
// never authorizes future foreign pthread creation.
//
//go:uintptrescapes
func AllThreadsSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	if cgo_libc_setegid != nil && !runtime_allThreadsSyscallSupported() {
		return minus1, minus1, ENOTSUP
	}
	r1, r2, errno := runtime_doAllThreadsSyscall(trap, a1, a2, a3, 0, 0, 0)
	return r1, r2, Errno(errno)
}

//go:uintptrescapes
func AllThreadsSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	if cgo_libc_setegid != nil && !runtime_allThreadsSyscallSupported() {
		return minus1, minus1, ENOTSUP
	}
	r1, r2, errno := runtime_doAllThreadsSyscall(trap, a1, a2, a3, a4, a5, a6)
	return r1, r2, Errno(errno)
}

func runtime_allThreadsSyscallSupported() bool

// The runtime owns these pointers even when syscall is not imported.

//go:linkname cgo_libc_setuid go:llgo.cgo_setuid
var cgo_libc_setuid unsafe.Pointer

//go:linkname cgo_libc_seteuid go:llgo.cgo_seteuid
var cgo_libc_seteuid unsafe.Pointer

//go:linkname cgo_libc_setgid go:llgo.cgo_setgid
var cgo_libc_setgid unsafe.Pointer

//go:linkname cgo_libc_setegid go:llgo.cgo_setegid
var cgo_libc_setegid unsafe.Pointer

//go:linkname cgo_libc_setreuid go:llgo.cgo_setreuid
var cgo_libc_setreuid unsafe.Pointer

//go:linkname cgo_libc_setregid go:llgo.cgo_setregid
var cgo_libc_setregid unsafe.Pointer

//go:linkname cgo_libc_setresuid go:llgo.cgo_setresuid
var cgo_libc_setresuid unsafe.Pointer

//go:linkname cgo_libc_setresgid go:llgo.cgo_setresgid
var cgo_libc_setresgid unsafe.Pointer

//go:linkname cgo_libc_setgroups go:llgo.cgo_setgroups
var cgo_libc_setgroups unsafe.Pointer
