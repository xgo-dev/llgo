//go:build wasm

package runtime

import (
	_ "unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

//go:linkname c_environ environ
var c_environ **c.Char

//go:linkname syscall_runtime_envs syscall.runtime_envs
func syscall_runtime_envs() []string {
	var out []string
	for p := c_environ; p != nil && *p != nil; p = c.Advance(p, 1) {
		out = append(out, c.GoString(*p))
	}
	return out
}
