//go:build llgo && arm64

package runtime

import _ "unsafe"

//go:linkname procyieldAsm llgo.asm
func procyieldAsm(instruction string)

// Procyield pauses the current hardware thread briefly during short lock spins.
// On Apple Silicon (arm64), YIELD is a 1-cycle no-op while ISB pauses the
// pipeline cleanly (~9ns) without consuming issue bandwidth.
func Procyield(cycles uint32) {
	for i := uint32(0); i < cycles; i++ {
		procyieldAsm("isb")
	}
}
