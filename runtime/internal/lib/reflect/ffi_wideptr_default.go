//go:build (!llgo || !wasm || (llgo.wasm.emscripten.memory64 && !wasip1 && (!js || llgo.wasm.reflect.libffi))) && !(baremetal && llgo.baremetal.stdio && !wasm)

package reflect

import (
	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
)

func ffiNativeFuncType(_ *abi.Type) *ffi.Type {
	return ffi.TypePointer
}

func ffiWidenGoPointerStruct(ft *ffi.Type, _ *abi.Type) *ffi.Type {
	return ft
}
