package runtime

// CPUFeatures is published by internal/cpu after applying GODEBUG overrides.
// It is immutable after package initialization. Generated FMV resolvers use a
// baseline without caching while the initialized bit is still clear.
var CPUFeatures uint64

// SIMDHasAVX reports the OS-enabled AVX flag after CPU overrides.
func SIMDHasAVX() bool { return CPUFeatures&4 != 0 }
