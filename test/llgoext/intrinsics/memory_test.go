//go:build llgo

package intrinsics

import (
	"testing"
	"unsafe"
)

//go:linkname copyMemory llvm.memcpy
func copyMemory(dst, src unsafe.Pointer, n uintptr, volatile bool)

//go:linkname copyMemoryInline llvm.memcpy.inline
func copyMemoryInline(dst, src unsafe.Pointer, n uintptr, volatile bool)

//go:linkname moveMemory llvm.memmove
func moveMemory(dst, src unsafe.Pointer, n uintptr, volatile bool)

//go:linkname fillMemory llvm.memset
func fillMemory(dst unsafe.Pointer, value byte, n uintptr, volatile bool)

//go:linkname fillMemoryInline llvm.memset.inline
func fillMemoryInline(dst unsafe.Pointer, value byte, n uintptr, volatile bool)

func TestMemoryIntrinsicOperations(t *testing.T) {
	var src, dst [32]byte
	for i := range src {
		src[i] = byte(i + 1)
	}
	copyMemory(unsafe.Pointer(&dst[1]), unsafe.Pointer(&src[3]), 17, false)
	for i := range dst {
		want := byte(0)
		if i >= 1 && i < 18 {
			want = byte(i + 3)
		}
		if dst[i] != want {
			t.Fatalf("copy byte %d: %d, want %d", i, dst[i], want)
		}
	}
	copyMemoryInline(unsafe.Pointer(&dst[1]), unsafe.Pointer(&src[2]), 16, true)
	for i := 0; i < 16; i++ {
		if dst[i+1] != byte(i+3) {
			t.Fatalf("inline copy byte %d", i)
		}
	}
	fillMemory(unsafe.Pointer(&dst[1]), 0xa5, 17, false)
	for i := 1; i < 18; i++ {
		if dst[i] != 0xa5 {
			t.Fatalf("fill byte %d", i)
		}
	}
	fillMemoryInline(unsafe.Pointer(&dst[1]), 0x5a, 16, true)
	for i := 1; i < 17; i++ {
		if dst[i] != 0x5a {
			t.Fatalf("inline fill byte %d", i)
		}
	}
	for _, backward := range []bool{false, true} {
		for i := range dst {
			dst[i] = byte(i + 1)
		}
		x, y := 1, 3
		if backward {
			x, y = y, x
		}
		moveMemory(unsafe.Pointer(&dst[x]), unsafe.Pointer(&dst[y]), 17, false)
		for i := range dst {
			want := byte(i + 1)
			if i >= x && i < x+17 {
				want = byte(i - x + y + 1)
			}
			if dst[i] != want {
				t.Fatalf("overlapping move %v, byte %d: %d, want %d", backward, i, dst[i], want)
			}
		}
	}
	copyMemory(nil, nil, 0, false)
	copyMemoryInline(nil, nil, 0, false)
	moveMemory(nil, nil, 0, false)
	fillMemory(nil, 0, 0, false)
	fillMemoryInline(nil, 0, 0, false)
}
