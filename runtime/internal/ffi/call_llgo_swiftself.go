//go:build llgo && llgo_closure_env_swiftself

package ffi

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/ffi"
)

const ClosureEnvExplicit = false

// CallWithEnv uses the runtime's libffi trampoline to preserve env across
// argument marshalling and place it in swiftself's register at the final call.
// Nil is written as well so native dynamic calls have one uniform path.
func CallWithEnv(cif *Signature, fn, env, ret unsafe.Pointer, args ...unsafe.Pointer) {
	var avalues *unsafe.Pointer
	if len(args) > 0 {
		avalues = &args[0]
	}
	ffi.CallWithEnv(cif, fn, ret, avalues, env)
}
