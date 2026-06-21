//go:build baremetal

package runtime

import (
	_ "unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
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
