//go:build !llgo || !wasm || (!wasip1 && (!js || llgo.wasm.reflect.libffi))

package reflect

const useWasmReflectBridges = false
