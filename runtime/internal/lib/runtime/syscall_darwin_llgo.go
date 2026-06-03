//go:build darwin && !go1.26

package runtime

import _ "unsafe"

//go:linkname llgo_syscall llgo.syscall
func llgo_syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_syscall6 llgo.syscall
func llgo_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_syscall6X llgo.syscall
func llgo_syscall6X(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_syscall9 llgo.syscall
func llgo_syscall9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_syscall5f64 llgo.syscall
func llgo_syscall5f64(fn, a1, a2, a3, a4, a5 uintptr, f1 float64) (r1, r2, err uintptr)

//go:linkname crypto_x509_syscall crypto/x509/internal/macos.syscall
func crypto_x509_syscall(fn, a1, a2, a3, a4, a5 uintptr, f1 float64) uintptr {
	r1, _, _ := llgo_syscall5f64(fn, a1, a2, a3, a4, a5, f1)
	return r1
}

//go:linkname llgo_syscallPtr llgo.syscall
func llgo_syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_rawSyscall llgo.syscall
func llgo_rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_rawSyscall6 llgo.syscall
func llgo_rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname syscall_syscall syscall.syscall
func syscall_syscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscall(fn, a1, a2, a3)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_syscall6 syscall.syscall6
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscall6(fn, a1, a2, a3, a4, a5, a6)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_syscall6X syscall.syscall6X
func syscall_syscall6X(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscall6X(fn, a1, a2, a3, a4, a5, a6)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_syscallPtr syscall.syscallPtr
func syscall_syscallPtr(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscallPtr(fn, a1, a2, a3)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_syscallX syscall.syscallX
func syscall_syscallX(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscall(fn, a1, a2, a3)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_syscall9 syscall.syscall9
func syscall_syscall9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_syscall9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_rawSyscall syscall.rawSyscall
func syscall_rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_rawSyscall(fn, a1, a2, a3)
	return normalizeSyscallErr(r1, r2, err)
}

//go:linkname syscall_rawSyscall6 syscall.rawSyscall6
func syscall_rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	r1, r2, err = llgo_rawSyscall6(fn, a1, a2, a3, a4, a5, a6)
	return normalizeSyscallErr(r1, r2, err)
}

func normalizeSyscallErr(r1, r2, err uintptr) (uintptr, uintptr, uintptr) {
	if r1 != ^uintptr(0) {
		return r1, r2, 0
	}
	return r1, r2, err
}
