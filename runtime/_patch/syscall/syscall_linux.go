//go:build linux

package syscall

func rawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr) {
	r1, _ = runtime_syscall3(trap, a1, a2, a3)
	return r1, 0
}
