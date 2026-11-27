//go:build llgo && (amd64 || 386)

package runtime

import _ "unsafe"

//go:linkname procyieldAsm llgo.asm
func procyieldAsm(instruction string)

// Procyield executes architecture pause instructions during short lock spins.
func Procyield(cycles uint32) {
	for i := uint32(0); i < cycles; i++ {
		procyieldAsm("pause")
	}
}
