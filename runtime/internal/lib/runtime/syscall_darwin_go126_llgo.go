//go:build darwin && go1.26

package runtime

import _ "unsafe"

//go:linkname llgo_syscall5f64 llgo.syscall
func llgo_syscall5f64(fn, a1, a2, a3, a4, a5 uintptr, f1 float64) (r1, r2, err uintptr)

//go:linkname llgo_rawSyscall llgo.syscall
func llgo_rawSyscall(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_rawSyscall6 llgo.syscall
func llgo_rawSyscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr)

//go:linkname llgo_rawSyscall9 llgo.syscall
func llgo_rawSyscall9(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr)

// Go 1.26 moves errno normalization into syscall/syscall_darwin.go wrappers
// such as syscall9/rawSyscall9/syscallPtr, so runtime mirrors the raw runtime
// entry points and returns the libc errno unchanged here.

//go:linkname syscall_syscalln syscall.syscalln
func syscall_syscalln(fn uintptr, args ...uintptr) (r1, r2, err uintptr) {
	entersyscall()
	r1, r2, err = syscall_rawsyscalln(fn, args...)
	exitsyscall()
	return r1, r2, err
}

//go:linkname syscall_rawsyscalln syscall.rawsyscalln
func syscall_rawsyscalln(fn uintptr, args ...uintptr) (r1, r2, err uintptr) {
	switch len(args) {
	case 0:
		return llgo_rawSyscall(fn, 0, 0, 0)
	case 1:
		return llgo_rawSyscall(fn, args[0], 0, 0)
	case 2:
		return llgo_rawSyscall(fn, args[0], args[1], 0)
	case 3:
		return llgo_rawSyscall(fn, args[0], args[1], args[2])
	case 4:
		return llgo_rawSyscall6(fn, args[0], args[1], args[2], args[3], 0, 0)
	case 5:
		return llgo_rawSyscall6(fn, args[0], args[1], args[2], args[3], args[4], 0)
	case 6:
		return llgo_rawSyscall6(fn, args[0], args[1], args[2], args[3], args[4], args[5])
	case 7:
		return llgo_rawSyscall9(fn, args[0], args[1], args[2], args[3], args[4], args[5], args[6], 0, 0)
	case 8:
		return llgo_rawSyscall9(fn, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], 0)
	case 9:
		return llgo_rawSyscall9(fn, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8])
	default:
		panic("runtime: unsupported darwin syscall arg count")
	}
}

//go:linkname crypto_x509_syscall crypto/x509/internal/macos.syscall
func crypto_x509_syscall(fn, a1, a2, a3, a4, a5 uintptr, f1 float64) uintptr {
	r1, _, _ := llgo_syscall5f64(fn, a1, a2, a3, a4, a5, f1)
	return r1
}
