//go:build llgo && linux && baremetal && llgo.baremetal.stdio && !go1.26

package syscall

import _ "unsafe"

// Go 1.26 moved this Linux implementation to internal/runtime/syscall/linux.
//
//llgo:skip Syscall6
//go:linkname Syscall6 runtime.llgoBaremetalSyscall6
func Syscall6(num, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, errno uintptr)
