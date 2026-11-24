//go:build js && wasm && (!llgo || !llgo.wasm.fs_snapshot || llgo.wasm.workers)

package js

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

func initEmvalGlobals() {
	if emvalGlobalsReady {
		return
	}
	valueGlobal = emval_get_global(nil)
	objectConstructor = emval_get_global(c.Str("Object"))
	stringConstructor = emval_get_global(c.Str("String"))
	arrayConstructor = emval_get_global(c.Str("Array"))
	functionConstructor = emval_get_global(c.Str("Function"))
	valueNaN = emval_get_global(c.Str("NaN"))
	valueZero = emval_new_double(0)
	uint8Array = emval_get_global(c.Str("Uint8Array"))
	uint8ClampedArray = emval_get_global(c.Str("Uint8ClampedArray"))
	emvalGlobalsReady = true
}
