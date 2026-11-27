//go:build baremetal && !nogc

package tinygogc

import (
	"unsafe"
	_ "unsafe"
)

// LLGoPackage instructs the LLGo linker to wrap C standard library memory allocation
// functions (malloc, realloc, calloc, free) so they use the tinygogc allocator
// instead. This ensures all memory allocations go through the GC, including C
// library calls. free belongs to the set: a buffer allocated through
// __wrap_malloc lives in the GC heap, so releasing it through the platform's
// free would hand a GC address to an allocator that never owns it (on ESP-IDF
// that corrupts TLSF metadata; tinygogc itself reports "freeing pointer outside
// heap"). The wrapped pair is what keeps C-requested buffers reachable: the
// collector scans the GC heap, not the platform heap.
const LLGoPackage = "link: --wrap=malloc --wrap=realloc --wrap=calloc --wrap=free"

//export __wrap_malloc
func __wrap_malloc(size uintptr) unsafe.Pointer {
	return Alloc(size)
}

//export __wrap_calloc
func __wrap_calloc(nmemb, size uintptr) unsafe.Pointer {
	totalSize := nmemb * size
	// Check for multiplication overflow
	if nmemb != 0 && totalSize/nmemb != size {
		return nil // Overflow
	}
	return Alloc(totalSize)
}

//export __wrap_realloc
func __wrap_realloc(ptr unsafe.Pointer, size uintptr) unsafe.Pointer {
	return Realloc(ptr, size)
}

//export __wrap_free
func __wrap_free(ptr unsafe.Pointer) {
	Free(ptr)
}

//go:linkname _heapStart C._heapStart
var _heapStart [0]byte

//go:linkname _heapEnd C._heapEnd
var _heapEnd [0]byte

//go:linkname _stackStart C._stack_top
var _stackStart [0]byte

//go:linkname _stackEnd C._stack_end
var _stackEnd [0]byte

//go:linkname _globals_start C._globals_start
var _globals_start [0]byte

//go:linkname _globals_end C._globals_end
var _globals_end [0]byte

// Baremetal heap bases come from the target linker script; do not assume
// stronger alignment than the machine word.
const allocationAlignment = unsafe.Sizeof(uintptr(0))

func gcMemoryLayout() (heapStart, heapEnd, globalsStart, globalsEnd, stackTop uintptr) {
	// Reserve 2 KiB for libc internal allocation that cannot be wrapped.
	// Keep a guard gap below the bootstrap stack: the linker places _heapEnd
	// and _stack_end at the same address, so any off-by-one write at the top
	// of the heap would land on the scheduler stack's floor word.
	return uintptr(unsafe.Pointer(&_heapStart)) + 2048,
		uintptr(unsafe.Pointer(&_heapEnd)) - 64,
		uintptr(unsafe.Pointer(&_globals_start)),
		uintptr(unsafe.Pointer(&_globals_end)),
		uintptr(unsafe.Pointer(&_stackStart))
}

func gcGrowMemory(oldHeapEnd uintptr) uintptr {
	return oldHeapEnd
}

// StackRootsHook is installed by scheduler backends that own additional
// execution stacks (the baremetal fiber scheduler). When set, it replaces the
// default single-context stack scan and reports every mapped stack range
// through the mark callback.
var StackRootsHook func(mark func(start, end uintptr))

func gcMarkReachable() {
	if globalsStart < globalsEnd {
		markRoots(globalsStart, globalsEnd)
	}
	if StackRootsHook != nil {
		StackRootsHook(markRoots)
		return
	}
	sp := uintptr(getsp())
	if sp < stackTop {
		markRoots(sp, stackTop)
	}
}

func gcStackStats() (inuse, sys uintptr) {
	sp := uintptr(getsp())
	if sp < stackTop {
		inuse = stackTop - sp
	}
	stackEnd := uintptr(unsafe.Pointer(&_stackEnd))
	if stackEnd < stackTop {
		sys = stackTop - stackEnd
	}
	return
}
