//go:build js && wasm && (!llgo || !llgo.wasm.workers)

package js

const keepWasmCallbackPoll = false
