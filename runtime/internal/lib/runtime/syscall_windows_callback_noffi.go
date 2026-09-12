//go:build windows && llgo_noffi

package runtime

import _ "unsafe"

// Keep syscall's linkname target available without pulling in the libffi
// callback implementation. A noffi program cannot compile C callbacks.
//
//go:linkname syscall_compileCallback syscall.compileCallback
func syscall_compileCallback(fn any, cleanstack bool) uintptr {
	_, _ = fn, cleanstack
	panic("syscall callback support is unavailable without libffi")
}
