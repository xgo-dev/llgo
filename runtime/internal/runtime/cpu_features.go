package runtime

// CPUFeatures records effective Go predicates and the available vector
// register widths. Only the predicates are subject to GODEBUG overrides.
// It is immutable after package initialization. Generated FMV resolvers use a
// baseline without caching while the initialized bit is still clear.
var CPUFeatures uint64

// SIMDHasAVX reports the OS-enabled AVX flag after CPU overrides.
func SIMDHasAVX() bool { return CPUFeatures&4 != 0 }

// PanicSIMDX86Feature is called only on the missing-feature path.
func PanicSIMDX86Feature() {
	panic(errorString("SIMD instruction requires an unavailable x86 CPU feature"))
}

// SIMDRequireX86Features remains available to previously generated packages.
func SIMDRequireX86Features(required uint64) {
	if CPUFeatures&required != required {
		PanicSIMDX86Feature()
	}
}
