//go:build goexperiment.simd && llgo && linux && amd64

package simd_test

import (
	"os"
	"simd/archsimd"
	"syscall"
	"testing"
	"unsafe"
)

func TestSIMDBridgeMaskedStoreGuardPage(t *testing.T) {
	page := os.Getpagesize()
	mem, err := syscall.Mmap(-1, 0, page*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(mem)
	if err := syscall.Mprotect(mem[page:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}
	values := [4]uint32{11, 22, 33, 44}
	x := archsimd.LoadUint32x4Array(&values)
	for count := 1; count < 4; count++ {
		ptr := (*[4]uint32)(unsafe.Pointer(&mem[page-count*4]))
		mask := archsimd.Mask32x4FromBits(uint8(1<<count - 1))
		x.StoreArrayMasked(ptr, mask)
		for i := 0; i < count; i++ {
			if ptr[i] != values[i] {
				t.Fatalf("masked store count=%d lane=%d = %d", count, i, ptr[i])
			}
		}
		// StorePart reaches this primitive through the official Go helper.
		if got := x.StorePart(unsafe.Slice(&ptr[0], count)); got != count {
			t.Fatalf("StorePart(%d) = %d", count, got)
		}
	}
	// Every lane is inaccessible, but none is enabled.
	x.StoreArrayMasked((*[4]uint32)(unsafe.Pointer(&mem[page])), archsimd.Mask32x4{})
}

func TestSIMD256MaskedStoreGuardPage(t *testing.T) {
	page := os.Getpagesize()
	mem, err := syscall.Mmap(-1, 0, page*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(mem)
	if err := syscall.Mprotect(mem[page:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}
	values := [8]uint32{11, 22, 33, 44, 55, 66, 77, 88}
	x := archsimd.LoadUint32x8Array(&values)
	for count := 1; count < 8; count++ {
		ptr := (*[8]uint32)(unsafe.Pointer(&mem[page-count*4]))
		x.StoreArrayMasked(ptr, archsimd.Mask32x8FromBits(uint8(1<<count-1)))
		for i := 0; i < count; i++ {
			if ptr[i] != values[i] {
				t.Fatalf("count %d lane %d: %d", count, i, ptr[i])
			}
		}
		if got := x.StorePart(unsafe.Slice(&ptr[0], count)); got != count {
			t.Fatalf("StorePart(%d)=%d", count, got)
		}
	}
	x.StoreArrayMasked((*[8]uint32)(unsafe.Pointer(&mem[page])), archsimd.Mask32x8{})
	x.StoreArrayMasked(nil, archsimd.Mask32x8{})
}
