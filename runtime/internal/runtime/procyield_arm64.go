//go:build llgo && !darwin && arm64

package runtime

import _ "unsafe"

//go:linkname procyieldAsm llgo.asm
func procyieldAsm(instruction string)

// Procyield pauses the current hardware thread briefly during short lock spins.
// The count parameter specifies the number of architecture hint instructions
// to issue (not CPU clock cycles).
func Procyield(count uint32) {
	for i := uint32(0); i < count; i++ {
		procyieldAsm("yield")
	}
}
