/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

	"github.com/xgo-dev/llgo/runtime/abi"
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/math"
)

// -----------------------------------------------------------------------------

// Slice is the runtime representation of a slice.
type Slice struct {
	data unsafe.Pointer
	len  int
	cap  int
}

func NewSlice2(base unsafe.Pointer, eltSize, cap int, i, j int64, iSigned, jSigned, upperIsLen bool) (s Slice) {
	// cap is non-negative. Converting both indexes to uint64 makes the
	// successful-path checks independent of the source index signedness: a
	// negative signed index, and an unsigned index whose high bit is set, both
	// compare above the non-negative bound. Keep the signed flags only for the
	// panic text on the cold path.
	if uint64(j) > uint64(cap) {
		upperCode := boundsSliceAcap
		if upperIsLen {
			upperCode = boundsSliceAlen
		}
		panicBounds(j, jSigned, cap, upperCode)
	}
	if uint64(i) > uint64(j) {
		panicBounds(i, iSigned, int(j), boundsSliceB)
	}
	return newSliceUnchecked(base, eltSize, int(i), int(j), cap)
}

func NewSlice3Bounds(base unsafe.Pointer, eltSize, cap int, i, j, k int64, iSigned, jSigned, kSigned, upperIsLen bool) (s Slice) {
	// As in NewSlice2, check from the upper bound down so each successful
	// uint64 comparison establishes 0 <= index <= bound. Since cap fits int,
	// k and then j also fit int before the subsequent checks and conversions.
	if uint64(k) > uint64(cap) {
		upperCode := boundsSlice3Acap
		if upperIsLen {
			upperCode = boundsSlice3Alen
		}
		panicBounds(k, kSigned, cap, upperCode)
	}
	if uint64(j) > uint64(k) {
		panicBounds(j, jSigned, int(k), boundsSlice3B)
	}
	if uint64(i) > uint64(j) {
		panicBounds(i, iSigned, int(j), boundsSlice3C)
	}
	return newSliceUnchecked(base, eltSize, int(i), int(j), int(k))
}

func NewSlice3(base unsafe.Pointer, eltSize, cap, i, j, k int) (s Slice) {
	if k < 0 || k > cap {
		panic(boundsError{x: int64(k), signed: true, y: cap, code: boundsSlice3Acap})
	}
	if j < 0 || j > k {
		panic(boundsError{x: int64(j), signed: true, y: k, code: boundsSlice3B})
	}
	if i < 0 || i > j {
		panic(boundsError{x: int64(i), signed: true, y: j, code: boundsSlice3C})
	}
	return newSliceUnchecked(base, eltSize, i, j, k)
}

func newSliceUnchecked(base unsafe.Pointer, eltSize, i, j, k int) (s Slice) {
	s.len = j - i
	s.cap = k - i
	if k-i > 0 {
		s.data = c.Advance(base, i*eltSize)
	} else {
		s.data = base
	}
	return
}

// SliceAppend append elem data and returns a slice.
func SliceAppend(src Slice, data unsafe.Pointer, num, etSize int, hasPointers bool) Slice {
	return SliceAppendAligned(src, data, num, etSize, 1, hasPointers)
}

func SliceAppendAligned(src Slice, data unsafe.Pointer, num, etSize, etAlign int, hasPointers bool) Slice {
	oldLen := src.len
	src = GrowSliceAligned(src, num, etSize, etAlign, hasPointers)
	if etSize == 0 {
		return src
	}
	c.Memcpy(c.Advance(src.data, oldLen*etSize), data, uintptr(num*etSize))
	return src
}

// GrowSlice grows slice and returns the grown slice.
func GrowSlice(src Slice, num, etSize int, hasPointers bool) Slice {
	return GrowSliceAligned(src, num, etSize, 1, hasPointers)
}

func GrowSliceAligned(src Slice, num, etSize, etAlign int, hasPointers bool) Slice {
	oldLen := src.len
	newLen := oldLen + num
	if newLen < 0 {
		panicgrowslicelen()
	}
	if newLen > src.cap {
		newCap := nextslicecap(newLen, src.cap)
		mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(newCap))
		if overflow || mem > maxAlloc {
			panicgrowslicelen()
		}
		var p unsafe.Pointer
		if hasPointers {
			p = AllocZAligned(mem, uintptr(etAlign))
		} else {
			p = AllocAtomicUAligned(mem, uintptr(etAlign))
			// The old prefix is overwritten by the copy below. Clear the
			// remaining capacity, including newly exposed elements: reflect.Grow
			// uses this helper without subsequently writing append arguments.
			copied := uintptr(oldLen) * uintptr(etSize)
			c.Memset(unsafe.Add(p, copied), 0, mem-copied)
		}
		if oldLen != 0 {
			c.Memcpy(p, src.data, uintptr(oldLen*etSize))
		}
		src.data = p
		src.cap = newCap
	}
	src.len = newLen
	return src
}

// nextslicecap computes the next appropriate slice length.
func nextslicecap(newLen, oldCap int) int {
	newcap := oldCap
	doublecap := newcap + newcap
	if newLen > doublecap {
		return newLen
	}

	const threshold = 256
	if oldCap < threshold {
		return doublecap
	}
	for {
		// Transition from growing 2x for small slices
		// to growing 1.25x for large slices. This formula
		// gives a smooth-ish transition between the two.
		newcap += (newcap + 3*threshold) >> 2

		// We need to check `newcap >= newLen` and whether `newcap` overflowed.
		// newLen is guaranteed to be larger than zero, hence
		// when newcap overflows then `uint(newcap) > uint(newLen)`.
		// This allows to check for both with the same comparison.
		if uint(newcap) >= uint(newLen) {
			break
		}
	}

	// Set newcap to the requested cap when
	// the newcap calculation overflowed.
	if newcap <= 0 {
		return newLen
	}
	return newcap
}

// SliceCopy copy data to slice and returns a slice.
func SliceCopy(dst Slice, data unsafe.Pointer, num int, etSize int) int {
	n := dst.len
	if n > num {
		n = num
	}
	if n > 0 {
		c.Memmove(dst.data, data, uintptr(n*etSize))
	}
	return n
}

func MakeSlice(len, cap int, etSize int, hasPointers bool) Slice {
	mem := makeSliceMemory(len, cap, etSize)
	// Ordinary Go slices need only natural alignment. Keep their allocation
	// independent of the dynamic over-alignment and validation paths.
	var data unsafe.Pointer
	if hasPointers {
		data = AllocZ(mem)
	} else {
		data = AllocAtomicZ(mem)
	}
	return Slice{data, len, cap}
}

func MakeSliceAligned(len, cap, etSize, etAlign int, hasPointers bool) Slice {
	return Slice{allocZTyped(makeSliceMemory(len, cap, etSize), uintptr(etAlign), hasPointers), len, cap}
}

func makeSliceMemory(len, cap, etSize int) uintptr {
	mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(cap))
	if overflow || mem > maxAlloc || len < 0 || len > cap {
		mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(len))
		if overflow || mem > maxAlloc || len < 0 {
			panicmakeslicelen()
		}
		panicmakeslicecap()
	}
	return mem
}

// MakeSliceBuffer uses caller-owned aligned storage for a valid small make.
// Fallbacks enter the original helper, preserving both its checks and the
// existing heap allocation path for callers without an escape proof.
func MakeSliceBuffer(len, cap, etSize, etAlign int, buffer unsafe.Pointer, bufferCap int, hasPointers bool) Slice {
	if buffer != nil && len >= 0 && len <= cap && cap <= bufferCap {
		mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(cap))
		if !overflow && mem <= maxAlloc {
			c.Memset(buffer, 0, mem)
			return Slice{buffer, len, cap}
		}
	}
	return MakeSliceAligned(len, cap, etSize, etAlign, hasPointers)
}

func panicmakeslicelen() {
	panic(errorString("makeslice: len out of range"))
}

func panicmakeslicecap() {
	panic(errorString("makeslice: cap out of range"))
}

func panicgrowslicelen() {
	panic(errorString("growslice: len out of range"))
}

func SliceClear(t *abi.SliceType, s Slice) {
	c.Memset(s.data, 0, uintptr(s.len)*t.Elem.Size())
}

func unsafeslice(et *abi.Type, ptr unsafe.Pointer, len int) {
	if len < 0 {
		panic(errorString("unsafe.Slice: len out of range"))
	}

	if et.Size_ == 0 {
		if ptr == nil && len > 0 {
			panic(errorString("unsafe.Slice: ptr is nil and len is not zero"))
		}
	}

	mem, overflow := math.MulUintptr(et.Size_, uintptr(len))
	if overflow || mem > -uintptr(ptr) {
		if ptr == nil {
			panic(errorString("unsafe.Slice: ptr is nil and len is not zero"))
		}
		panic(errorString("unsafe.Slice: len out of range"))
	}
}

// -----------------------------------------------------------------------------
