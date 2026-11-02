//go:build !llgo || !linux || baremetal || wasm

package runtime

func MarkMainThreadOwned() {}
