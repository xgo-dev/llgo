//go:build amd64 && go1.26 && !baremetal

package cpu

import "internal/goos"

// Follow internal/cpu.Initialize, recording the OS-enabled register widths
// between detection and Go's feature overrides. A by-value vector call uses
// the same register even when the corresponding Go query has been disabled.
func llgoInitializeCPU(env string) uint64 {
	doinit()
	var widths uint64
	if X86.HasAVX || goos.IsLinux != 0 && llgoRosettaHasYMM() {
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

// Linux Rosetta can mask the AVX instruction bit while translating AVX.
// A caller with a fixed YMM transfer can use its AVX body in that case.
// Keep Go's public feature flags as reported by the standard detector.
func llgoRosettaHasYMM() bool {
	const brand = "VirtualApple "
	name := Name()
	if len(name) < len(brand) || name[:len(brand)] != brand {
		return false
	}
	const xmmYMM = 1<<1 | 1<<2
	if X86.HasOSXSAVE {
		state, _ := xgetbv()
		return state&xmmYMM == xmmYMM
	}
	// Recent Linux Rosetta still enumerates XMM/YMM in CPUID leaf D when
	// AVX and OSXSAVE are both hidden. Older translators may lack this leaf.
	// On native x86, supported XSAVE components alone cannot authorize AVX.
	maxLeaf, _, _, _ := cpuid(0, 0)
	if maxLeaf < 0xd {
		return false
	}
	state, _, _, _ := cpuid(0xd, 0)
	return state&xmmYMM == xmmYMM
}
