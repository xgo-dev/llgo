//go:build llgo && linux && baremetal && llgo.baremetal.stdio

package linux

import _ "unsafe"

//llgo:skip Syscall6
//go:linkname Syscall6 runtime.llgoBaremetalSyscall6
func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)
