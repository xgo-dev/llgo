//go:build llgo && llgo_closure_env_swiftself

package ffi

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/ffi"
)

const ClosureEnvExplicit = false

// CallWithEnv invokes fn with a semantic CIF that does not contain env. The
// native final hop passes env separately in LLVM's swiftself register: ARM32
// bridges libffi's R12 static chain to R10, while AArch64 targets that reserve
// X18 use a TLS trampoline to install X20. Nil is installed as well so native
// dynamic calls keep one uniform path.
func CallWithEnv(cif *Signature, fn, env, ret unsafe.Pointer, args ...unsafe.Pointer) {
	var avalues *unsafe.Pointer
	if len(args) > 0 {
		avalues = &args[0]
	}
	ffi.CallWithEnv(cif, fn, ret, avalues, env)
}
