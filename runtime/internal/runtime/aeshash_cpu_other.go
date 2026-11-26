//go:build !llgo || baremetal || (!amd64 && !386 && !(arm64 && ((darwin && !ios) || linux || android || windows)))

package runtime

func aesHashHardwareSupport() bool {
	return false
}
