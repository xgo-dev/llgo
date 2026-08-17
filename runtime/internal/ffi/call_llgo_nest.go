//go:build llgo && llgo_closure_env_nest

package ffi

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/ffi"
)

const ClosureEnvExplicit = false

// CallWithEnv invokes fn with a semantic CIF that does not contain env. The
// native final hop passes env separately in LLVM's nest register, using
// ffi_call_go directly when libffi selects the same physical register.
func CallWithEnv(cif *Signature, fn, env, ret unsafe.Pointer, args ...unsafe.Pointer) {
	var avalues *unsafe.Pointer
	if len(args) > 0 {
		avalues = &args[0]
	}
	ffi.CallWithEnv(cif, fn, ret, avalues, env)
}
