//go:build windows || (linux && !baremetal) || (llgo && wasip1 && wasm && llgo.wasi_threads)

package runtime

import (
	latomic "sync/atomic"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

func semaAcquire(addr *uint32) {
	for {
		value := latomic.LoadUint32(addr)
		if value != 0 && latomic.CompareAndSwapUint32(addr, value, value-1) {
			return
		}
		psync.WaitUint32(addr, 0)
	}
}

func semaReleaseNative(addr *uint32) {
	latomic.AddUint32(addr, 1)
	psync.WakeUint32(addr)
}
