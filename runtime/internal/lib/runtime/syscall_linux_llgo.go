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

// Production builds retain ENOTSUP, including executables: the registry cannot
// exclude foreign C thread creators. The private native test transport freezes
// registration/exit and parks its controlled cohort before any syscall runs.
// A masked transport or pending creation returns EAGAIN before execution.
// See runtime/ALLTHREADS.md for the ownership and activation requirements.
//
//go:linkname syscall_runtime_doAllThreadsSyscall syscall.runtime_doAllThreadsSyscall
//go:uintptrescapes
func syscall_runtime_doAllThreadsSyscall(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2, err uintptr) {
	call := allThreadsCall{trap: trap, args: [6]uintptr{a1, a2, a3, a4, a5, a6}}
	c_allThreadsCall(&call)
	return call.r1, call.r2, call.err
}
