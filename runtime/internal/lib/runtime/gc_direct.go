//go:build (!windows || !386) && !nogc && !baremetal && !wasm

package runtime

func requestExplicitGC() {
	collectExplicitGCBatch()
}
