//go:build !llgo || !js || !wasm || !llgo_wasm_gc || !llgo.wasm_workers

package tinygogc

// TODO(MeteorsLiu): mutex lock for baremetal GC
type mutex struct{}

func lock(m *mutex) {}

func unlock(m *mutex) {}
