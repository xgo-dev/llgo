//go:build llgo && darwin && arm64

package runtime

import _ "unsafe"

//go:linkname chanSignalSpinAsm llgo.asm
func chanSignalSpinAsm(instruction string)

// On Apple Silicon, YIELD is a 1-cycle no-op (~0.3ns/iter), whereas ISB
// pauses the pipeline cleanly (~9ns/iter) without burning issue bandwidth.
// 5120 iterations covers ~47us of adaptive spinning for hot low-latency
// handoffs when spare CPU cores are idle; exponential miss backoff ensures
// longer-blocking channels immediately decay to minimal fastSpin (~0.3us).
const chanSignalSpinLimit = 5120

func chanSignalSpinHint() {
	chanSignalSpinAsm("isb")
}
