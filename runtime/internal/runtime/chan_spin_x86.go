//go:build llgo && (amd64 || 386)

package runtime

import _ "unsafe"

//go:linkname chanSignalSpinAsm llgo.asm
func chanSignalSpinAsm(instruction string)

// Keep the PAUSE count small: its cost varies substantially between CPUs.
const chanSignalSpinLimit = 128

func chanSignalSpinHint() {
	chanSignalSpinAsm("pause")
}
