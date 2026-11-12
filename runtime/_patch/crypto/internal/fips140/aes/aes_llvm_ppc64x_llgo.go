//go:build llgo && (ppc64 || ppc64le) && !baremetal && go1.27 && !go1.28

package aes

const llgoAESName = "POWER8"
const llgoAESPreKey = false

// POWER8, including its vector crypto facility, is Go's PPC64 baseline.
func llgoAESHardware() bool { return true }
