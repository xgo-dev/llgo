//go:build llgo && js && wasm && llgo.wasm.gc.linear && llgo.wasm.workers

package tinygogc

// Multi-worker collection stops goroutines only at compiler safepoints. Their
// live pointer values are therefore completely represented by the per-worker
// compiler root chains. Conservatively rescanning each current fiber stack is
// both redundant and unsafe: integer words in Emscripten's pthread frames can
// resemble heap addresses and retain arbitrarily large native allocation
// graphs.
func gcMarkStackRoots() {}
