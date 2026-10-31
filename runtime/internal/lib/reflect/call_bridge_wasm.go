//go:build llgo && wasm && (wasip1 || (js && !llgo.wasm.reflect.libffi))

package reflect

import "unsafe"

const useWasmReflectBridges = true

func makeProviderFunc(ft *funcType, fn func([]Value) []Value, recoverTo unsafe.Pointer) Value {
	return makeCompiledFunc(ft, fn, recoverTo)
}
