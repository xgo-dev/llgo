//go:build baremetal || wasm

package runtime

func savePanicStack(skip int) {
}

func savedPanicStack(pc []uintptr) int {
	return 0
}
