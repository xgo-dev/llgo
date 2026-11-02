//go:build linux && !baremetal
// +build linux,!baremetal

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

// auxv is the process auxiliary vector without the terminating AT_NULL pair.
// Keep this contract in sync with the Go runtime: x/sys/cpu and x/sys/unix
// access runtime.getAuxv through go:linkname.
var auxv = processAuxv()

func processAuxv() []uintptr {
	// argc == 0 is valid: argv[0] is the nil terminator and envp starts at
	// argv[1], so the same layout walk still applies.
	if c.Argc < 0 || c.Argv == nil {
		return nil
	}

	// argv is followed by a nil pointer, envp, another nil pointer, and auxv.
	// These scans intentionally trust the kernel-provided process-start layout;
	// do not reuse them for untrusted memory.
	p := c.Advance(c.Argv, int(c.Argc)+1)
	for *p != nil {
		p = c.Advance(p, 1)
	}
	p = c.Advance(p, 1)
	values := (*uintptr)(unsafe.Pointer(p))
	pairs := 0
	for c.Index(values, pairs*2) != 0 {
		pairs++
	}
	return unsafe.Slice(values, pairs*2)
}

//go:linkname getAuxv runtime.getAuxv
func getAuxv() []uintptr { return auxv }

//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string {
	argc := int(c.Argc)
	if argc <= 0 {
		return nil
	}
	if c.Argv == nil {
		return nil
	}
	args := make([]string, 0, argc)
	for i := 0; i < argc; i++ {
		p := c.Index(c.Argv, i)
		if p == nil {
			break
		}
		args = append(args, c.GoString(p))
	}
	return args
}

//go:linkname c_environ C.environ
var c_environ **c.Char

//go:linkname syscall_runtime_envs syscall.runtime_envs
func syscall_runtime_envs() []string {
	var out []string
	for p := c_environ; p != nil && *p != nil; p = c.Advance(p, 1) {
		out = append(out, c.GoString(*p))
	}
	return out
}

//go:linkname syscall_runtimeSetenv syscall.runtimeSetenv
func syscall_runtimeSetenv(key, value string) {
	cliteos.Setenv(c.AllocaCStr(key), c.AllocaCStr(value), 1)
	if key == "GODEBUG" {
		godebugEnvChanged(value)
	}
}

//go:linkname syscall_runtimeUnsetenv syscall.runtimeUnsetenv
func syscall_runtimeUnsetenv(key string) {
	cliteos.Unsetenv(c.AllocaCStr(key))
	if key == "GODEBUG" {
		godebugEnvChanged("")
	}
}

//go:linkname os_beforeExit os.runtime_beforeExit
func os_beforeExit(exitCode int) {
	runCoverageExitHook(exitCode)
}

//go:linkname os_sigpipe os.sigpipe
func os_sigpipe() { signalPipe() }

//go:linkname os_ignoreSIGSYS os.ignoreSIGSYS
func os_ignoreSIGSYS() {}

//go:linkname os_restoreSIGSYS os.restoreSIGSYS
func os_restoreSIGSYS() {}

//go:linkname c_getpagesize C.getpagesize
func c_getpagesize() c.Int

//go:linkname syscall_Getpagesize syscall.Getpagesize
func syscall_Getpagesize() int {
	return int(c_getpagesize())
}

//go:linkname syscall_Exit syscall.Exit
//go:nosplit
func syscall_Exit(code int) {
	cliteos.Exit(c.Int(code))
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

func fcntl(fd int32, cmd int32, arg int32) (int32, int32) {
	r := cliteos.Fcntl(c.Int(fd), c.Int(cmd), c.Int(arg))
	if r == -1 {
		return -1, int32(cliteos.Errno())
	}
	return int32(r), 0
}
