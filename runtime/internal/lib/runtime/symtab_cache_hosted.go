//go:build !baremetal

package runtime

// Symbolization cache capacities. Hosted targets have whole megabytes of
// memory, so the caches are sized for whole-stack re-walks and deep
// CallersFrames traffic. See symtab_cache_baremetal.go for the baremetal
// sizing.
const (
	// frameSymbolCacheCapacity covers one entry per distinct return pc of a
	// deep stack.
	frameSymbolCacheCapacity = 4096
	// frameSymbolResultCacheCapacity memoizes full symbolization results.
	frameSymbolResultCacheCapacity = 4096
	// funcForPCCacheSetCapacity keeps FuncForPC set-associative lookups fast.
	funcForPCCacheSetCapacity = 1024
)
