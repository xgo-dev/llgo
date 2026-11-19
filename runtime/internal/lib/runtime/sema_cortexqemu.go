//go:build linux && cortexm && qemu && lm3s6965

package runtime

import "sync/atomic"

func semaAcquire(addr *uint32) {
	for {
		n := atomic.LoadUint32(addr)
		if n == 0 {
			throw("semaphore wait on the single-context Cortex-M runtime")
		}
		if atomic.CompareAndSwapUint32(addr, n, n-1) {
			return
		}
	}
}

func semaAcquireWaitGroup(addr *uint32, _ bool) {
	semaAcquire(addr)
}

func semaRelease(addr *uint32) {
	atomic.AddUint32(addr, 1)
}
