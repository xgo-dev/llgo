//go:build darwin && arm64

package runtime

const (
	// On Apple Silicon, YIELD is a 1-cycle no-op (~0.3ns/iter), whereas ISB
	// pauses the pipeline cleanly (~9ns/iter) without burning issue bandwidth.
	// 5120 iterations covers ~47us of adaptive spinning for hot low-latency
	// handoffs when spare CPU cores are idle; exponential miss backoff ensures
	// longer-blocking channels immediately decay to minimal fastSpin (~0.3us).
	chanSignalSpinInstruction = "isb"
	chanSignalSpinLimit       = 5120
)
