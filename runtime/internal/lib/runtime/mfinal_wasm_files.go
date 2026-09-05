//go:build wasm && llgo.wasm.gc.linear

package runtime

const wasmFinalizerLLGoFiles = "; _wrap/finalizer_wasm.c"
