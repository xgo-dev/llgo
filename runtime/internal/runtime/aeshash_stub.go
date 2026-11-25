//go:build !llgo || !(amd64 || arm64 || 386) || baremetal

package runtime

import "unsafe"

const useAeshash = false

func initAESHash() {}

func memhashAES(p unsafe.Pointer, seed, s uintptr) uintptr {
	return 0
}
