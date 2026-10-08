//go:build js && wasm && (!llgo || !llgo.wasm.workers)

package js

func isRemoteJSWorker() bool { return false }

func onJSWorker(fn func()) { fn() }
