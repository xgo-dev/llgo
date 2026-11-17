//go:build !wasm

package runtime

func callerLocationHasContext() bool { return true }
