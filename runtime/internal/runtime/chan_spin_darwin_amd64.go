//go:build darwin && amd64

package runtime

// Keep the PAUSE count small: its cost varies substantially between CPUs.
const (
	chanSignalSpinInstruction = "pause"
	chanSignalSpinLimit       = 128
)
