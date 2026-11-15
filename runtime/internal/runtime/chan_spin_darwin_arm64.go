//go:build darwin && arm64

package runtime

const (
	chanSignalSpinInstruction = "yield"
	chanSignalSpinLimit       = 8192
)
