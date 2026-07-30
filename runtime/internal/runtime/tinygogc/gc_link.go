//go:build baremetal && !nogc

package tinygogc

import (
	"unsafe"
	_ "unsafe"
)

// LLGoPackage instructs the LLGo linker to wrap C standard library memory allocation
// functions (malloc, realloc, calloc) so they use the tinygogc allocator instead.
// This ensures all memory allocations go through the GC, including C library calls.
const LLGoPackage = "link: --wrap=malloc --wrap=realloc --wrap=calloc"

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

//go:linkname _heapStart _heapStart
var _heapStart [0]byte

//go:linkname _heapEnd _heapEnd
var _heapEnd [0]byte

//go:linkname _stackStart _stack_top
var _stackStart [0]byte

//go:linkname _stackEnd _stack_end
var _stackEnd [0]byte

//go:linkname _globals_start _globals_start
var _globals_start [0]byte

//go:linkname _globals_end _globals_end
var _globals_end [0]byte

func gcMemoryLayout() (heapStart, heapEnd, globalsStart, globalsEnd, stackTop uintptr) {
	// Reserve 2 KiB for libc internal allocation that cannot be wrapped.
	return uintptr(unsafe.Pointer(&_heapStart)) + 2048,
		uintptr(unsafe.Pointer(&_heapEnd)),
		uintptr(unsafe.Pointer(&_globals_start)),
		uintptr(unsafe.Pointer(&_globals_end)),
		uintptr(unsafe.Pointer(&_stackStart))
}

func gcGrowMemory(oldHeapEnd uintptr) uintptr {
	return oldHeapEnd
}

func gcMarkReachable() {
	sp := uintptr(getsp())
	if sp < stackTop {
		markRoots(sp, stackTop)
	}
	if globalsStart < globalsEnd {
		markRoots(globalsStart, globalsEnd)
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
