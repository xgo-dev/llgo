//go:build darwin || linux

package syscall

// Implemented in runtime. These fixed-arity hooks keep LLGo's C syscall
// bridge out of the standard package dependency graph.
func runtime_syscall3(trap, a1, a2, a3 uintptr) (r1 uintptr, errno int32)
func runtime_syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1 uintptr, errno int32)
