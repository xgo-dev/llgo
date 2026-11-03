//go:build !baremetal && !wasm && windows

package runtime

// Windows needs its SEH-aware walker to recover biased frame registers.
func fpCreatorCallers(pcs []uintptr) int { return fpCallers(0, pcs) }
