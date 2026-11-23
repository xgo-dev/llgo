//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

const useWasmReflectBridges = false

func callWasmBridge(ft *abi.FuncType, fn, env unsafe.Pointer, method, native bool, prefix []unsafe.Pointer, in []Value) []Value {
	return nil
}

func resetWasmFuncBridge(ft *abi.FuncType) {}

func copyWasmFuncBridge(dst, src *abi.FuncType) {}
