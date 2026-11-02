//go:build linux && (baremetal || wasm)

package runtime

import (
	clitesyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
	"unsafe"
)

//go:linkname syscall_cgocaller syscall.cgocaller
//go:uintptrescapes
func syscall_cgocaller(_ unsafe.Pointer, _ ...uintptr) uintptr {
	return uintptr(clitesyscall.ENOTSUP)
}

//go:linkname syscall_runtime_doAllThreadsSyscall syscall.runtime_doAllThreadsSyscall
//go:uintptrescapes
func syscall_runtime_doAllThreadsSyscall(_, _, _, _, _, _, _ uintptr) (r1, r2, err uintptr) {
	return ^uintptr(0), ^uintptr(0), uintptr(clitesyscall.ENOTSUP)
}
