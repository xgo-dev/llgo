//go:build llgo && darwin && arm64 && !ios && !baremetal

package runtime

// Match Go's internal/cpu/cpu_arm64_darwin.go: all Apple Silicon CPUs provide
// Armv8 AES instructions.
func aesHashHardwareSupport() bool {
	return true
}
