//go:build arm64 && go1.24

package atomic

// Xchg8 completes the 49-entry ARM64 group selected by Go1.24 and later.
func Xchg8(ptr *uint8, next uint8) uint8 {
	return atomicXchg8(ptr, next)
}
