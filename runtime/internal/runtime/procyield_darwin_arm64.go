//go:build llgo && darwin && arm64

package runtime

import _ "unsafe"

//go:linkname procyieldAsm llgo.asm
func procyieldAsm(instruction string)

// Procyield pauses the current hardware thread briefly during short lock spins.
// The count parameter specifies the number of hint instructions to issue (not
// CPU clock cycles). On Apple Silicon (darwin/arm64), YIELD retires as a
// single-cycle no-op whereas ISB (~9ns) pauses pipeline issue cleanly; 4 spin
// rounds of 10 ISBs (~360ns total) stays well within Go's 4x30 PAUSE budget on
// modern x86 CPUs.
func Procyield(count uint32) {
	for i := uint32(0); i < count; i++ {
		procyieldAsm("isb")
	}
}
