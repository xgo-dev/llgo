package runtime

// CPUFeatures is published by internal/cpu after applying GODEBUG overrides.
// It is immutable after package initialization. Generated FMV resolvers use a
// baseline without caching while the initialized bit is still clear.
var CPUFeatures uint64
