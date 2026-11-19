//go:build llgo && baremetal && llgo.baremetal.stdio && go1.27

package linux

import "unsafe"

// These targets use GOROOT's Linux/ARM package layout and have no Linux
// kernel. Standard input and output use the target libc.
//
//llgo:skip Syscall6
func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr) {
	const (
		armWrite = 4
		badFD    = 9
		ioError  = 5
		noSys    = 38
	)
	switch num {
	case SYS_READ:
		if a1 != 0 {
			return ^uintptr(0), 0, badFD
		}
		n := readConsole(int32(a1), unsafe.Pointer(a2), a3)
		if n < 0 {
			return ^uintptr(0), 0, ioError
		}
		return uintptr(n), 0, 0
	case armWrite:
		if a1 != 1 && a1 != 2 {
			return ^uintptr(0), 0, badFD
		}
		n := writeConsole(int32(a1), unsafe.Pointer(a2), a3)
		if n < 0 {
			return ^uintptr(0), 0, ioError
		}
		return uintptr(n), 0, 0
	}
	return ^uintptr(0), 0, noSys
}

//go:linkname readConsole C.read
func readConsole(fd int32, buf unsafe.Pointer, n uintptr) int32

//go:linkname writeConsole C.write
func writeConsole(fd int32, buf unsafe.Pointer, n uintptr) int32
