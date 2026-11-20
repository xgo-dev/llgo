//go:build baremetal

package runtime

// Baremetal SRAM budgets cannot host the hosted symbolization caches, so
// every table shrinks to a small power-of-two (masking in symtab.go relies on
// the capacity being a power of two).
const (
	frameSymbolCacheCapacity       = 64
	frameSymbolResultCacheCapacity = 64
	funcForPCCacheSetCapacity      = 64
)
