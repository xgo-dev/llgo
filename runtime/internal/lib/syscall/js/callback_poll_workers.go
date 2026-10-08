//go:build js && wasm && llgo && llgo.wasm.workers

package js

// Keep the shared poll hook installed while worker schedulers can enter it.
// Registered functions and emval handles are owned by Go's main JS worker.
const keepWasmCallbackPoll = true
