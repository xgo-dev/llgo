package runtime

// CPUFeatures records effective Go predicates and the available vector
// register widths. Only the predicates are subject to GODEBUG overrides.
// It is immutable after package initialization. Generated FMV resolvers use a
// baseline without caching while the initialized bit is still clear.
var CPUFeatures uint64

// SIMDHasAVX reports the OS-enabled AVX flag after CPU overrides.
func SIMDHasAVX() bool { return CPUFeatures&4 != 0 }

// Previously generated code marks the edge after this call unreachable.
func PanicSIMDX86Feature() {
	panic(errorString("SIMD instruction requires an unavailable x86 CPU feature"))
}

// Previous compilers called SIMDRequireX86Features before every x86 instruction
// leaf. Those call sites can return normally and execute the instruction.
func SIMDRequireX86Features(uint64) {}
