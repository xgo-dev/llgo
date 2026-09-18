//go:build llgo && js && wasm && !llgo.wasm.emscripten

package os_test

// The GoJS browser host has the same process fallback as Go's wasm_exec.js:
// identity values are -1 and process-dependent operations return ENOSYS.
const goJSBrowserProcessUnavailable = true
