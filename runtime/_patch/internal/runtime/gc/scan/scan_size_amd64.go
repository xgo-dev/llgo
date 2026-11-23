//go:build llgo && linux && llgo.size.native && go1.26

package scan

import (
	"internal/runtime/gc"
	"unsafe"
)

//llgo:skip HasFastScanSpanPacked CanAVX512 ScanSpanPacked ScanSpanPackedAVX512 scanSpanPackedAVX512 FilterNilAVX512 ExpandAVX512 avx512ScanPackedReqsMet

// LLGo uses its own collector. The size profile keeps the standard Go
// scanner's software fallback available without its assembly dispatch table.
func HasFastScanSpanPacked() bool { return false }

func CanAVX512() bool { return false }

func ScanSpanPacked(mem unsafe.Pointer, bufp *uintptr, objMarks *gc.ObjMask, sizeClass uintptr, ptrMask *gc.PtrMask) int32 {
	return ScanSpanPackedGo(mem, bufp, objMarks, sizeClass, ptrMask)
}

func ScanSpanPackedAVX512(mem unsafe.Pointer, bufp *uintptr, objMarks *gc.ObjMask, sizeClass uintptr, ptrMask *gc.PtrMask) int32 {
	return ScanSpanPackedGo(mem, bufp, objMarks, sizeClass, ptrMask)
}

func FilterNilAVX512(bufp *uintptr, n int32) int32 { return FilterNil(bufp, n) }

func ExpandAVX512(sizeClass int, packed *gc.ObjMask, unpacked *gc.PtrMask) {
	*unpacked = gc.PtrMask{}
	ExpandReference(sizeClass, packed, unpacked)
}
