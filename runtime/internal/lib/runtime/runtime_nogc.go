//go:build nogc && (!wasm || !llgo_wasm_gc)

package runtime

func GC() {

}
