//go:build baremetal

package runtime

// The bare-metal scheduler does not use 64-bit atomic counters.
type atomic64Alignment struct{}
