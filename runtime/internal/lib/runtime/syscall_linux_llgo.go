//go:build linux && !baremetal && !wasm

package runtime

import (
	_ "unsafe"
)

type allThreadsCall struct {
	trap        uintptr
	args        [6]uintptr
	r1, r2, err uintptr
}

//go:linkname c_allThreadsCall C.llgo_traceback_all_threads_syscall
func c_allThreadsCall(*allThreadsCall)

// The native registry freezes creation/exit and parks the complete thread set
// before any syscall runs. Libraries and processes with unregistered C threads
// retain ENOTSUP. A masked transport or a pending creation returns EAGAIN before
// executing anything. Calls must be nonblocking and preserve signal/thread
// lifecycle; the native bridge rejects those incompatible syscall families.
//
//go:linkname syscall_runtime_doAllThreadsSyscall syscall.runtime_doAllThreadsSyscall
//go:uintptrescapes
func syscall_runtime_doAllThreadsSyscall(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	call := allThreadsCall{trap: trap, args: [6]uintptr{a1, a2, a3, a4, a5, a6}}
	c_allThreadsCall(&call)
	return call.r1, call.r2, call.err
}
