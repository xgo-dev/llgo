//go:build !(llgo && js && wasm && llgo.wasm.workers)

// Package wasmworkers provides an explicit way to start independent work in
// LLGo's bounded browser worker pool.
package wasmworkers

// GoIndependent starts fn as a goroutine on targets without the bounded
// browser worker pool.
func GoIndependent(fn func()) { go fn() }
