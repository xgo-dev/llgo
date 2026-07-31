//go:build llgo && llgo_closure_env_nest

package ffi

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/ffi"
)

const ClosureEnvExplicit = false

// CallWithEnv uses the runtime's libffi trampoline to place env in LLVM's nest
// register at the final call without changing the user-visible signature.
func CallWithEnv(cif *Signature, fn, env, ret unsafe.Pointer, args ...unsafe.Pointer) {
	var avalues *unsafe.Pointer
	if len(args) > 0 {
		avalues = &args[0]
	}
	ffi.CallWithEnv(cif, fn, ret, avalues, env)
}
