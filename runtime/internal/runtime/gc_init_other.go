//go:build nogc || baremetal || wasm

package runtime

// InitGC is a no-op when native BDWGC is not in use.
func InitGC() {}
