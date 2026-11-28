//go:build llgo && js && wasm && llgo.wasm.schedulerproof && !llgo.wasm.workers

package runtime

import _ "unsafe"

const wasmSchedulerProof = true

// The normal Emscripten link supplies an empty definition. A link without
// the scheduler succeeds only if it can discard all calls to this symbol.
//
//go:linkname requireWasmScheduler C.__llgo_wasm_requires_scheduler
func requireWasmScheduler()
