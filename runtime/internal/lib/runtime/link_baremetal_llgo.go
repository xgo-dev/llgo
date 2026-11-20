//go:build baremetal

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string {
	return nil
}

//go:linkname syscall_runtime_envs syscall.runtime_envs
func syscall_runtime_envs() []string {
	return nil
}

//go:linkname syscall_runtimeSetenv syscall.runtimeSetenv
func syscall_runtimeSetenv(key, value string) {
	_, _ = key, value
	if key == "GODEBUG" {
		godebugEnvChanged(value)
	}
}

//go:linkname syscall_runtimeUnsetenv syscall.runtimeUnsetenv
func syscall_runtimeUnsetenv(key string) {
	if key == "GODEBUG" {
		godebugEnvChanged("")
	}
}

//go:linkname os_beforeExit os.runtime_beforeExit
func os_beforeExit(exitCode int) {
	_ = exitCode
}

//go:linkname os_sigpipe os.sigpipe
func os_sigpipe() {}

//go:linkname os_ignoreSIGSYS os.ignoreSIGSYS
func os_ignoreSIGSYS() {}

//go:linkname os_restoreSIGSYS os.restoreSIGSYS
func os_restoreSIGSYS() {}

//go:linkname syscall_Getpagesize syscall.Getpagesize
func syscall_Getpagesize() int {
	return 4096
}

//go:linkname syscall_Exit syscall.Exit
//go:nosplit
func syscall_Exit(code int) {
	c.Exit(c.Int(code))
}

//go:linkname syscall_runtime_BeforeFork syscall.runtime_BeforeFork
func syscall_runtime_BeforeFork() {}

//go:linkname syscall_runtime_AfterFork syscall.runtime_AfterFork
func syscall_runtime_AfterFork() {}

//go:linkname syscall_runtime_AfterForkInChild syscall.runtime_AfterForkInChild
func syscall_runtime_AfterForkInChild() {}

//go:linkname syscall_runtime_BeforeExec syscall.runtime_BeforeExec
func syscall_runtime_BeforeExec() {}

//go:linkname syscall_runtime_AfterExec syscall.runtime_AfterExec
func syscall_runtime_AfterExec() {}

// The masqueraded GOOS=linux also expects the OS bridges that hosted
// link_linux files supply. There is no kernel here: report ENOSYS instead of
// routing through the C syscall bridge. cmpstring replaces the GOROOT bytealg
// assembly that the translator does not provide for this target; it is pure
// computation and keeps the real stdlib comparison semantics.

func cmpstring(x, y string) int {
	l := len(x)
	if len(y) < l {
		l = len(y)
	}
	for i := 0; i < l; i++ {
		c1, c2 := uint32(x[i]), uint32(y[i])
		if c1 != c2 {
			if c1 < c2 {
				return -1
			}
			return +1
		}
	}
	if len(x) < len(y) {
		return -1
	}
	if len(x) > len(y) {
		return +1
	}
	return 0
}

func fcntl(fd int32, cmd int32, arg int32) (int32, int32) {
	_, _, _ = fd, cmd, arg
	return -1, 38 // ENOSYS
}

//go:linkname syscall_rawVforkSyscall syscall.rawVforkSyscall
func syscall_rawVforkSyscall(trap, a1, a2, a3 uintptr) (r1 uintptr, err uintptr) {
	_, _, _, _ = trap, a1, a2, a3
	return 0, 38 // ENOSYS
}

//go:linkname syscall_seek syscall.seek
func syscall_seek(fd int32, offset int64, whence int32) (newoffset int64, errno uintptr) {
	_, _, _ = fd, offset, whence
	return 0, 38 // ENOSYS
}
