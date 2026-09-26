//go:build js && wasm && llgo.wasm.workers

package syscall

import (
	"syscall/js"
	_ "unsafe"
)

//go:linkname llgoHostJSGlobal syscall/js.GlobalForHost
func llgoHostJSGlobal() js.Value

// Node objects are represented by Emscripten emval handles, whose ownership
// is local to one JavaScript worker. Integer flags and the Go file table stay
// process-wide; only the handles are initialized independently per worker.
//
//llgointernal:tls
var (
	jsProcess  = llgoHostJSGlobal().Get("process")
	jsPath     = llgoHostJSGlobal().Get("path")
	jsFS       = llgoHostJSGlobal().Get("fs")
	constants  = jsFS.Get("constants")
	uint8Array = llgoHostJSGlobal().Get("Uint8Array")
)
