//go:build !llgo || !linux || baremetal || wasm

package runtime

func markTracebackOwned(*g) {}
