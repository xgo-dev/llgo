//go:build wasm && !baremetal

package runtime

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
)

//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string {
	argc := int(c.Argc)
	if argc <= 0 || c.Argv == nil {
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

// syscall's standard-library environment implementation obtains its initial
// snapshot from runtime. Emscripten and WASI both expose the C environment, so
// use the same source as LLGo's native Unix runtimes.
//
//go:linkname syscall_runtime_envs syscall.runtime_envs
func syscall_runtime_envs() []string {
	var out []string
	for p := cliteos.Environ; p != nil; p = c.Advance(p, 1) {
		value := c.Index(p, 0)
		if value == nil {
			break
		}
		out = append(out, c.GoString(value))
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
	_ = exitCode
}

// WebAssembly has no SIGPIPE delivery. This matches the official Go wasm
// runtime and keeps os.File's broken-pipe hook a no-op.
//
//go:linkname os_sigpipe os.sigpipe
func os_sigpipe() {}

//go:linkname syscall_Exit syscall.Exit
//go:nosplit
func syscall_Exit(code int) {
	c.Exit(c.Int(code))
}
