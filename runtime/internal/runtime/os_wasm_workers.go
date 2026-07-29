//go:build llgo && js && wasm && llgo.wasm_workers

package runtime

// Worker threads are owned by the scheduler pool rather than individual M
// records, so the host-specific M payload stays empty.
type mOS struct{}
