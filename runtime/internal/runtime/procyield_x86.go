//go:build llgo && (amd64 || 386)

package runtime

import _ "unsafe"

//go:linkname procyieldAsm llgo.asm
func procyieldAsm(instruction string)

// Procyield executes architecture pause instructions during short lock spins.
// The count parameter specifies the number of pause instructions to issue.
func Procyield(count uint32) {
	for i := uint32(0); i < count; i++ {
		procyieldAsm("pause")
	}
}
