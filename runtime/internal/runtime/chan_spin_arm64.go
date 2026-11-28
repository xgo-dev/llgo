//go:build llgo && !darwin && arm64

package runtime

import _ "unsafe"

//go:linkname chanSignalSpinAsm llgo.asm
func chanSignalSpinAsm(instruction string)

// On non-Apple ARM64 cores (Neoverse/Cortex/Oryon), YIELD is a lightweight
// pipeline hint; 512 iterations provides brief adaptive handoff spinning under
// the core-bounded slot cap and exponential miss backoff.
const chanSignalSpinLimit = 512

func chanSignalSpinHint() {
	chanSignalSpinAsm("yield")
}
