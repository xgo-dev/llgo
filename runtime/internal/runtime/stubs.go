// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"unsafe"

	"github.com/goplus/llgo/runtime/internal/clite/sync/atomic"
	"github.com/goplus/llgo/runtime/internal/clite/time"
	"github.com/goplus/llgo/runtime/internal/runtime/math"
)

//go:linkname fastrand C.rand
func fastrand() uint32

//go:linkname srand C.srand
func srand(uint32)

func fastrand64() uint64 {
	n := uint64(fastrand())
	n += 0xa0761d6478bd642f
	hi, lo := math.Mul64(n, n^0xe7037ed1a0b428db)
	return hi ^ lo
}

func init() {
	srand(uint32(time.Time(nil)))
	hashkey[0] = uintptr(fastrand()) | 1
	hashkey[1] = uintptr(fastrand()) | 1
	hashkey[2] = uintptr(fastrand()) | 1
	hashkey[3] = uintptr(fastrand()) | 1
}

/* TODO(xsw):
func fastrand() uint32 {
	mp := getg().m
	// Implement wyrand: https://github.com/wangyi-fudan/wyhash
	// Only the platform that math.Mul64 can be lowered
	// by the compiler should be in this list.
	if goarch.IsAmd64|goarch.IsArm64|goarch.IsPpc64|
		goarch.IsPpc64le|goarch.IsMips64|goarch.IsMips64le|
		goarch.IsS390x|goarch.IsRiscv64|goarch.IsLoong64 == 1 {
		mp.fastrand += 0xa0761d6478bd642f
		hi, lo := math.Mul64(mp.fastrand, mp.fastrand^0xe7037ed1a0b428db)
		return uint32(hi ^ lo)
	}

	// Implement xorshift64+: 2 32-bit xorshift sequences added together.
	// Shift triplet [17,7,16] was calculated as indicated in Marsaglia's
	// Xorshift paper: https://www.jstatsoft.org/article/view/v008i14/xorshift.pdf
	// This generator passes the SmallCrush suite, part of TestU01 framework:
	// http://simul.iro.umontreal.ca/testu01/tu01.html
	t := (*[2]uint32)(unsafe.Pointer(&mp.fastrand))
	s1, s0 := t[0], t[1]
	s1 ^= s1 << 17
	s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
	t[0], t[1] = s0, s1
	return s0 + s1
}
*/

//go:nosplit
func add(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// implementation of new builtin
// compiler (both frontend and SSA backend) knows the signature
// of this function.
func newobject(typ *_type) unsafe.Pointer {
	return AllocZ(typ.Size_)
}

// TODO
func roundupsize(size uintptr) uintptr {
	// if size < _MaxSmallSize {
	// 	if size <= smallSizeMax-8 {
	// 		return uintptr(class_to_size[size_to_class8[divRoundUp(size, smallSizeDiv)]])
	// 	} else {
	// 		return uintptr(class_to_size[size_to_class128[divRoundUp(size-smallSizeMax, largeSizeDiv)]])
	// 	}
	// }
	// if size+_PageSize < size {
	// 	return size
	// }
	// return alignUp(size, _PageSize)
	return size
}

// newarray allocates an array of n elements of type typ.
func newarray(typ *_type, n int) unsafe.Pointer {
	if n == 1 {
		return AllocZ(typ.Size_)
	}
	mem, overflow := math.MulUintptr(typ.Size_, uintptr(n))
	if overflow || mem > maxAlloc || n < 0 {
		panic(plainError("runtime: allocation size out of range"))
	}
	return AllocZ(mem)
}

const (
	// _64bit = 1 on 64-bit systems, 0 on 32-bit systems
	_64bit       = 1 << (^uintptr(0) >> 63) / 2
	heapAddrBits = (_64bit)*48 + (1-_64bit)*(32)
	maxAlloc     = (1 << heapAddrBits) - (1-_64bit)*1
)

func memclrHasPointers(ptr unsafe.Pointer, n uintptr) {
	// bulkBarrierPreWrite(uintptr(ptr), 0, n)
	// memclrNoHeapPointers(ptr, n)
}

func memclrNoHeapPointers(ptr unsafe.Pointer, n uintptr) {
}

func throw(s string) {
	print("fatal error: ", s, "\n")
}

func atomicOr8(ptr *uint8, v uint8) uint8 {
	return (uint8)(atomic.Or((*uint)(unsafe.Pointer(ptr)), uint(v)))
}

func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}
