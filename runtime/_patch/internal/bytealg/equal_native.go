//go:build 386 || amd64 || arm64

package bytealg

import "unsafe"

// These are real LLGo runtime bodies for the complete equality source group.
// Other bytealg assembly remains selected and translated normally.

//go:linkname equalityMemcmp C.memcmp
func equalityMemcmp(a, b unsafe.Pointer, size uintptr) int32

//go:linkname equalityClosureEnv llgo.closureEnv
func equalityClosureEnv() unsafe.Pointer

//go:linkname equalityTypeSize github.com/xgo-dev/llgo/runtime/internal/runtime.equalityTypeSize
func equalityTypeSize(context unsafe.Pointer) uintptr

//go:linkname abigen_runtime_memequal runtime.memequal
func abigen_runtime_memequal(a, b unsafe.Pointer, size uintptr) bool {
	if size == 0 || a == b {
		return true
	}
	return equalityMemcmp(a, b, size) == 0
}

// This entry deliberately implements LLGo's typed environment contract. The
// compiler supplies its real Type*; normal closure lowering transports it.
// Stock gc's {entry, size} carrier and physical R26/DX entry are not supported
// by this body and are not inferred from the runtime symbol's spelling.
//
//go:linkname abigen_runtime_memequal_varlen runtime.memequal_varlen
//llgo:env
func abigen_runtime_memequal_varlen(a, b unsafe.Pointer) bool {
	return abigen_runtime_memequal(a, b, equalityTypeSize(equalityClosureEnv()))
}
