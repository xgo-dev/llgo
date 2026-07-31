//go:build !llgo || llgo_closure_env_explicit || (llgo && !llgo_closure_env_nest && !llgo_closure_env_swiftself)

package ffi

import "unsafe"

// ClosureEnvExplicit is a compile-time target property.
const ClosureEnvExplicit = true

// CallWithEnv calls fn through libffi. On explicit-context targets the signature
// passed by the caller already contains the leading env type; this function
// supplies the corresponding value before reaching this final call helper.
func CallWithEnv(cif *Signature, fn, _ unsafe.Pointer, ret unsafe.Pointer, args ...unsafe.Pointer) {
	Call(cif, fn, ret, args...)
}
