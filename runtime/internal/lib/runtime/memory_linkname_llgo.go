package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

const (
	memoryHookWordSize   = unsafe.Sizeof(uintptr(0))
	memoryHookBlockWords = 8
)

// External packages use these Go runtime hooks through go:linkname, including
// taking their addresses. Keep wrappers with the Go signatures instead of
// exposing the C functions, which return a pointer.

//go:linkname memmove runtime.memmove
//go:nosplit
func memmove(dst, src unsafe.Pointer, n uintptr) {
	if n == 0 {
		return
	}
	if (uintptr(dst)|uintptr(src)|n)&(memoryHookWordSize-1) == 0 {
		if dst == src {
			return
		}
		// Go requires indivisible pointer-sized writes for aligned ranges.
		// Unordered stores preserve that guarantee without synchronization
		// fences, and prevent LLVM from introducing tearing stores or libc
		// calls. The contract applies to large copies too, so keep this path
		// for every aligned size. Source reads need no atomic ordering.
		if n <= 2*memoryHookWordSize {
			// Snapshot both ends before either store so one-word overlap is
			// safe without the direction check. The ends coincide for one word.
			first := *(*uintptr)(src)
			last := *(*uintptr)(unsafe.Add(src, n-memoryHookWordSize))
			atomic.StoreUnordered((*uintptr)(dst), first)
			atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n-memoryHookWordSize)), last)
			return
		}
		// Copy backward when dst overlaps the source at a higher address.
		if uintptr(dst) > uintptr(src) && uintptr(dst)-uintptr(src) < n {
			// Load a complete block before writing it, so even a one-word
			// overlap is safe. Visit blocks backward to preserve later input.
			for n >= memoryHookBlockWords*memoryHookWordSize {
				n -= memoryHookBlockWords * memoryHookWordSize
				value7 := *(*uintptr)(unsafe.Add(src, n+7*memoryHookWordSize))
				value6 := *(*uintptr)(unsafe.Add(src, n+6*memoryHookWordSize))
				value5 := *(*uintptr)(unsafe.Add(src, n+5*memoryHookWordSize))
				value4 := *(*uintptr)(unsafe.Add(src, n+4*memoryHookWordSize))
				value3 := *(*uintptr)(unsafe.Add(src, n+3*memoryHookWordSize))
				value2 := *(*uintptr)(unsafe.Add(src, n+2*memoryHookWordSize))
				value1 := *(*uintptr)(unsafe.Add(src, n+memoryHookWordSize))
				value0 := *(*uintptr)(unsafe.Add(src, n))
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+7*memoryHookWordSize)), value7)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+6*memoryHookWordSize)), value6)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+5*memoryHookWordSize)), value5)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+4*memoryHookWordSize)), value4)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+3*memoryHookWordSize)), value3)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+2*memoryHookWordSize)), value2)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n+memoryHookWordSize)), value1)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n)), value0)
			}
			for n != 0 {
				n -= memoryHookWordSize
				value := *(*uintptr)(unsafe.Add(src, n))
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, n)), value)
			}
		} else {
			// Keep every block's loads before its stores for forward overlap.
			offset := uintptr(0)
			for n-offset >= memoryHookBlockWords*memoryHookWordSize {
				value0 := *(*uintptr)(unsafe.Add(src, offset))
				value1 := *(*uintptr)(unsafe.Add(src, offset+memoryHookWordSize))
				value2 := *(*uintptr)(unsafe.Add(src, offset+2*memoryHookWordSize))
				value3 := *(*uintptr)(unsafe.Add(src, offset+3*memoryHookWordSize))
				value4 := *(*uintptr)(unsafe.Add(src, offset+4*memoryHookWordSize))
				value5 := *(*uintptr)(unsafe.Add(src, offset+5*memoryHookWordSize))
				value6 := *(*uintptr)(unsafe.Add(src, offset+6*memoryHookWordSize))
				value7 := *(*uintptr)(unsafe.Add(src, offset+7*memoryHookWordSize))
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset)), value0)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+memoryHookWordSize)), value1)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+2*memoryHookWordSize)), value2)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+3*memoryHookWordSize)), value3)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+4*memoryHookWordSize)), value4)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+5*memoryHookWordSize)), value5)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+6*memoryHookWordSize)), value6)
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset+7*memoryHookWordSize)), value7)
				offset += memoryHookBlockWords * memoryHookWordSize
			}
			for ; offset < n; offset += memoryHookWordSize {
				value := *(*uintptr)(unsafe.Add(src, offset))
				atomic.StoreUnordered((*uintptr)(unsafe.Add(dst, offset)), value)
			}
		}
		return
	}
	c.Memmove(dst, src, n)
}

//go:linkname memclrNoHeapPointers runtime.memclrNoHeapPointers
//go:nosplit
func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr) {
	if n == 0 {
		return
	}
	if (uintptr(ptr)|n)&(memoryHookWordSize-1) == 0 {
		// This hook also underlies pointer-bearing clears in Go, so aligned
		// pointer-sized words must become zero without tearing at every size.
		// Unordered stores provide atomicity without synchronization fences.
		if n <= 2*memoryHookWordSize {
			atomic.StoreUnordered((*uintptr)(ptr), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, n-memoryHookWordSize)), uintptr(0))
			return
		}
		offset := uintptr(0)
		for n-offset >= memoryHookBlockWords*memoryHookWordSize {
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+2*memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+3*memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+4*memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+5*memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+6*memoryHookWordSize)), uintptr(0))
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset+7*memoryHookWordSize)), uintptr(0))
			offset += memoryHookBlockWords * memoryHookWordSize
		}
		for ; offset < n; offset += memoryHookWordSize {
			atomic.StoreUnordered((*uintptr)(unsafe.Add(ptr, offset)), uintptr(0))
		}
		return
	}
	c.Memset(ptr, 0, n)
}
