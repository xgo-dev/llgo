package main

import "unsafe"

type Slices struct{ Data [64][]byte }
type Strings struct{ Data [64]string }
type TaggedSlices struct {
	Data [64][]byte
	Tag  int
}

//go:linkname slicesResult C.slicesResult
func slicesResult() Slices

//go:linkname stringsResult C.stringsResult
func stringsResult() Strings

//go:linkname taggedResult C.taggedResult
func taggedResult() TaggedSlices

//go:linkname consumeSlices C.consumeSlices
func consumeSlices(Slices)

//export FirstSliceLen
func FirstSliceLen(p Slices) int { return len(p.Data[0]) }

//export LastSliceCap
func LastSliceCap(p Slices) int { return cap(p.Data[63]) }

//export FirstStringLen
func FirstStringLen(p Strings) int { return len(p.Data[0]) }

//export NestedSliceLen
func NestedSliceLen(p [4][4][4][]byte) int { return len(p[3][2][1]) }

//export SliceTag
func SliceTag(p TaggedSlices) int { return p.Tag }

//export ResultSliceLen
func ResultSliceLen() int { return len(slicesResult().Data[0]) }

//export ResultSliceCap
func ResultSliceCap() int { return cap(slicesResult().Data[63]) }

//export ResultStringLen
func ResultStringLen() int { return len(stringsResult().Data[0]) }

//export ResultSliceTag
func ResultSliceTag() int { return taggedResult().Tag }

//export FirstSliceData
func FirstSliceData(p Slices) *byte { return unsafe.SliceData(p.Data[0]) }

//export ResultSliceData
func ResultSliceData() *byte { return unsafe.SliceData(slicesResult().Data[0]) }

//export TypedSlicePadding
func TypedSlicePadding(p Slices) uint32 {
	s := p.Data[0]
	return (*[2]uint32)(unsafe.Pointer(&s))[1]
}

//export FirstSliceValue
func FirstSliceValue(p Slices) []byte { return p.Data[0] }

//export ResultSliceValue
func ResultSliceValue() []byte { return slicesResult().Data[0] }

//export FirstSliceSlot
func FirstSliceSlot(p Slices) uintptr { return *(*uintptr)(unsafe.Pointer(&p.Data[0])) }

//export ResultSliceSlot
func ResultSliceSlot() uintptr {
	p := slicesResult()
	return *(*uintptr)(unsafe.Pointer(&p.Data[0]))
}

//export ForwardSlices
func ForwardSlices(p Slices) { consumeSlices(p) }

//export ForwardResult
func ForwardResult() { consumeSlices(slicesResult()) }

//export SliceLenOrSlot
func SliceLenOrSlot(p Slices, lengthOnly bool) uintptr {
	if lengthOnly {
		return uintptr(len(p.Data[0]))
	}
	return *(*uintptr)(unsafe.Pointer(&p.Data[0]))
}

//export ResultLenOrSlot
func ResultLenOrSlot(lengthOnly bool) uintptr {
	p := slicesResult()
	if lengthOnly {
		return uintptr(len(p.Data[0]))
	}
	return *(*uintptr)(unsafe.Pointer(&p.Data[0]))
}

var stored Slices

//export StoreSlices
func StoreSlices(p Slices) { stored = p }

//export StoreResult
func StoreResult() { stored = slicesResult() }

func main() {}
