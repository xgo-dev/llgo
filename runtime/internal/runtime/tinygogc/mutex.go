//go:build !llgo || !wasm || !llgo.wasm.gc.linear || ((!js || !llgo.wasm.workers) && (!wasip1 || !llgo.wasi_threads))

package tinygogc

// TODO(MeteorsLiu): mutex lock for baremetal GC
type mutex struct{}

func lock(m *mutex) {}

func unlock(m *mutex) {}
