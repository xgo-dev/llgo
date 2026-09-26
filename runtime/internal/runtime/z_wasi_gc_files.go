//go:build llgo && wasip1 && wasm && llgo.wasi_threads && llgo.wasm.gc.linear

package runtime

const wasiGCLLGoFiles = "; _wrap/wasi_gc_world.c"
