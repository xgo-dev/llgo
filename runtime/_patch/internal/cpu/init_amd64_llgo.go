//go:build amd64 && go1.26 && !baremetal

package cpu

// Follow internal/cpu.Initialize, recording the OS-enabled register widths
// between detection and Go's feature overrides. A by-value vector call uses
// the same register even when the corresponding Go query has been disabled.
func llgoInitializeCPU(env string) uint64 {
	doinit()
	var widths uint64
	if X86.HasAVX {
		widths |= 1 << 17
	}
	if X86.HasAVX512F {
		widths |= 1 << 18
	}
	processOptions(env)
	if doDerived != nil {
		doDerived()
	}
	return widths
}
