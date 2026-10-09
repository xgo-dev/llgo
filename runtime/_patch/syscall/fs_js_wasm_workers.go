//go:build js && wasm && llgo.wasm.workers

package syscall

import (
	"syscall/js"
	_ "unsafe"
)

//go:linkname llgoHostJSGlobal syscall/js.GlobalForHost
func llgoHostJSGlobal() js.Value

// The upstream syscall implementation reads these names directly. Initialize
// each worker's cached handles on first use; syscall/js routes their operations
// through Go's main JavaScript realm.
// Integer flags and the Go file table remain process-wide.
type workerJSHostValue uint8

const (
	jsProcess workerJSHostValue = iota
	jsPath
	jsFS
	constants
	uint8Array
)

//llgointernal:tls
var (
	workerJSHostReady bool
	workerJSProcess   js.Value
	workerJSPath      js.Value
	workerJSFS        js.Value
	workerJSConstants js.Value
	workerUint8Array  js.Value
)

func ensureWorkerJSHost() {
	if workerJSHostReady {
		return
	}
	global := llgoHostJSGlobal()
	workerJSProcess = global.Get("process")
	workerJSPath = global.Get("path")
	workerJSFS = global.Get("fs")
	workerJSConstants = workerJSFS.Get("constants")
	workerUint8Array = global.Get("Uint8Array")
	workerJSHostReady = true
}

func (v workerJSHostValue) value() js.Value {
	ensureWorkerJSHost()
	switch v {
	case jsProcess:
		return workerJSProcess
	case jsPath:
		return workerJSPath
	case jsFS:
		return workerJSFS
	case constants:
		return workerJSConstants
	case uint8Array:
		return workerUint8Array
	}
	panic("unknown worker JavaScript host value")
}

func (v workerJSHostValue) Get(name string) js.Value { return v.value().Get(name) }

func (v workerJSHostValue) Call(name string, args ...any) js.Value {
	return v.value().Call(name, args...)
}

func (v workerJSHostValue) New(args ...any) js.Value { return v.value().New(args...) }
