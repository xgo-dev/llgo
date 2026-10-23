//go:build amd64 && go1.26

package scan

import (
	"internal/runtime/gc"
	"unsafe"
)

//llgo:skip gcExpandersAVX512

// These bodies own the complete selected AMD64 assembly group. They do not
// implement the SDK's private AX -> Z1/Z2 ABI or manufacture expander PCs.
// CanAVX512, HasFastScanSpanPacked and the public scan dispatch stay SDK-owned.
func llgoGCScanFactor(sizeClass int) uintptr {
	if sizeClass <= 0 || sizeClass >= len(gc.SizeClassToSize) || gc.SizeClassToNPages[sizeClass] != 1 {
		panic("unsupported packed scan size class")
	}
	factor := uintptr(gc.SizeClassToSize[sizeClass]) / 8
	switch factor {
	case 1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32,
		36, 40, 44, 48, 52, 56, 60, 64:
		return factor
	}
	panic("unsupported packed scan size class")
}

func ExpandAVX512(sizeClass int, packed *gc.ObjMask, unpacked *gc.PtrMask) {
	factor := llgoGCScanFactor(sizeClass)
	marks := *packed // The original assembly loads its input before writing output.
	*unpacked = gc.PtrMask{}
	// Like the actual expander, cover all 1024 output bits, including a final
	// partial object's bits. Valid object masks have no marks past the last object.
	for word := uintptr(0); word < gc.PageWords; word++ {
		object := word / factor
		if marks[object/64]&(uintptr(1)<<(object%64)) != 0 {
			unpacked[word/64] |= uintptr(1) << (word % 64)
		}
	}
}

func scanSpanPackedAVX512(mem unsafe.Pointer, bufp *uintptr, objMarks *gc.ObjMask, sizeClass uintptr, ptrMask *gc.PtrMask) (count int32) {
	var expanded gc.PtrMask
	ExpandAVX512(int(sizeClass), objMarks, &expanded)
	pointers := *ptrMask
	buf := unsafe.Slice(bufp, gc.PageWords)
	for word := uintptr(0); word < gc.PageWords; word++ {
		bit := uintptr(1) << (word % 64)
		if expanded[word/64]&pointers[word/64]&bit != 0 {
			// Raw scan includes zero; its SDK caller performs zero-only filtering.
			buf[count] = *(*uintptr)(unsafe.Add(mem, word*8))
			count++
		}
	}
	return count
}

func FilterNilAVX512(bufp *uintptr, n int32) int32 {
	if n <= 0 {
		return 0
	}
	buf := unsafe.Slice(bufp, int(n))
	var count int32
	for _, value := range buf {
		if value != 0 {
			buf[count] = value
			count++
		}
	}
	return count
}
