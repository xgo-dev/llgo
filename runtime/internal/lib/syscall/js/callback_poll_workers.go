//go:build js && wasm && llgo && llgo.wasm.workers

package js

// The hook polls all JavaScript worker realms. A callback on one worker must
// not disable polling while another worker still has registered functions.
const keepWasmCallbackPoll = true
