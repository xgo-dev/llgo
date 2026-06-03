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

	"github.com/goplus/llgo/runtime/abi"
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/runtime/math"
)

// -----------------------------------------------------------------------------

// Slice is the runtime representation of a slice.
type Slice struct {
	data unsafe.Pointer
	len  int
	cap  int
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
func SliceAppend(src Slice, data unsafe.Pointer, num, etSize int) Slice {
	if etSize == 0 {
		return src
	}
	oldLen := src.len
	src = GrowSlice(src, num, etSize)
	c.Memcpy(c.Advance(src.data, oldLen*etSize), data, uintptr(num*etSize))
	return src
}

// GrowSlice grows slice and returns the grown slice.
func GrowSlice(src Slice, num, etSize int) Slice {
	oldLen := src.len
	newLen := oldLen + num
	if newLen > src.cap {
		newCap := nextslicecap(newLen, src.cap)
		p := AllocZ(uintptr(newCap * etSize))
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

func MakeSlice(len, cap int, etSize int) Slice {
	mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(cap))
	if overflow || mem > maxAlloc || len < 0 || len > cap {
		mem, overflow := math.MulUintptr(uintptr(etSize), uintptr(len))
		if overflow || mem > maxAlloc || len < 0 {
			panicmakeslicelen()
		}
		panicmakeslicecap()
	}
	return Slice{AllocZ(mem), len, cap}
}

func panicmakeslicelen() {
	panic(errorString("makeslice: len out of range"))
}

func panicmakeslicecap() {
	panic(errorString("makeslice: cap out of range"))
}

func SliceClear(t *abi.SliceType, s Slice) {
	c.Memset(s.data, 0, uintptr(s.len)*t.Elem.Size())
}

// -----------------------------------------------------------------------------
