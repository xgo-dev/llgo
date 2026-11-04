//go:build !baremetal && !wasm && windows

package runtime

// Windows keeps its platform walker: SEH-aware on amd64/arm64 to recover
// biased frame registers, and frame-pointer based on 386.
func fpCreatorCallers(pcs []uintptr) int { return fpCallers(0, pcs) }
