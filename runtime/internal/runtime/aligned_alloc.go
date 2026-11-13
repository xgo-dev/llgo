/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// AllocUAligned allocates GC-owned storage with a power-of-two alignment.
// It may return an interior pointer; both supported collectors retain the
// entire allocation through interior pointers. This storage is not paired
// with AllocRoot/FreeRoot and must not be passed to a native free.
func AllocUAligned(size, alignment uintptr) unsafe.Pointer {
	return allocAligned(size, alignment, false, false)
}

// AllocZAligned is the zero-initialized counterpart of AllocUAligned.
func AllocZAligned(size, alignment uintptr) unsafe.Pointer {
	return allocAligned(size, alignment, true, false)
}

// AllocAtomicUAligned allocates aligned memory that must never contain pointers.
func AllocAtomicUAligned(size, alignment uintptr) unsafe.Pointer {
	return allocAligned(size, alignment, false, true)
}

// AllocAtomicZAligned is the zero-initialized counterpart of AllocAtomicUAligned.
func AllocAtomicZAligned(size, alignment uintptr) unsafe.Pointer {
	return allocAligned(size, alignment, true, true)
}

// allocZTyped selects the collector kind from the allocated type, rather than
// the type of the pointer holding the result, while preserving zero values.
func allocZTyped(size, alignment uintptr, hasPointers bool) unsafe.Pointer {
	if hasPointers {
		return AllocZAligned(size, alignment)
	}
	return AllocAtomicZAligned(size, alignment)
}

// All entry points share validation and the natural-alignment boundary.
func allocAligned(size, alignment uintptr, zero, noscan bool) unsafe.Pointer {
	if alignment == 0 || alignment&(alignment-1) != 0 {
		panic("runtime: invalid allocation alignment")
	}
	if size > maxAlloc {
		panic("runtime: allocation size out of range")
	}
	if alignment <= unsafe.Sizeof(uintptr(0)) {
		if noscan {
			if zero {
				return AllocAtomicZ(size)
			}
			return AllocAtomicU(size)
		}
		if zero {
			return AllocZ(size)
		}
		return AllocU(size)
	}
	// Keep even a zero-sized result inside its allocation.
	allocSize := size
	if allocSize == 0 {
		allocSize = 1
	}
	padding := alignment - 1
	if padding > maxAlloc || allocSize > maxAlloc-padding {
		panic("runtime: allocation size out of range")
	}
	var p unsafe.Pointer
	if noscan {
		// Not every backend supports aligned pointer-free allocations. Align
		// atomic storage ourselves; the supported GC backends retain its base
		// through the interior pointer.
		p = AllocAtomicU(allocSize + padding)
		p = unsafe.Add(p, -uintptr(p)&padding)
	} else {
		p = allocUAligned(allocSize, alignment)
	}
	if zero {
		c.Memset(p, 0, size)
	}
	return p
}

// allocUAlignedFallback is used when the backend cannot provide the requested
// alignment. AllocUAligned has already checked size plus padding for overflow.
func allocUAlignedFallback(size, alignment uintptr) unsafe.Pointer {
	p := AllocU(size + alignment - 1)
	return unsafe.Add(p, -uintptr(p)&(alignment-1))
}
