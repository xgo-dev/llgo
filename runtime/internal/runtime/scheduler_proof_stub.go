//go:build !llgo || !js || !wasm || !llgo.wasm.schedulerproof || llgo.wasm.workers

package runtime

const wasmSchedulerProof = false

func requireWasmScheduler() {}
