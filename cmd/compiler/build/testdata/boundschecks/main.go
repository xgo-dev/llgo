package main

import "unsafe"

func indexString(v string, i uint8) byte {
	return v[i]
}

func indexSlice(v []byte, i uint8) byte {
	return v[i]
}

func indexArray(v [4]byte, i uint8) byte {
	return v[i]
}

func indexArrayPointer(v *[4]byte, i uint8) byte {
	return v[i]
}

func sliceString(v string, low uint8, high uint16) string {
	return v[low:high]
}

func sliceSlice(v []byte, low uint8, high uint16) []byte {
	return v[low:high]
}

func sliceArray(v [4]byte, low uint8, high uint16) []byte {
	return v[low:high]
}

func sliceArrayPointer(v *[4]byte, low uint8, high uint16) []byte {
	return v[low:high]
}

func sliceThree(v []byte, low uint8, high uint16, max uint32) []byte {
	return v[low:high:max]
}

func shortSliceToArrayPointer(v []byte) *[4]byte {
	return (*[4]byte)(v)
}

func shortSliceToArrayValue(v []byte) [4]byte {
	return [4]byte(v)
}

func makeUnsafeString(v *byte, n int) string {
	return unsafe.String(v, n)
}

func makeUnsafeSlice(v *byte, n int) []byte {
	return unsafe.Slice(v, n)
}

func main() {
	str := "abcd"
	slice := []byte{10, 20, 30, 40}
	array := [4]byte{10, 20, 30, 40}
	arrayPointer := &array

	println(indexString(str, 1), indexSlice(slice, 2), indexArray(array, 0), indexArrayPointer(arrayPointer, 3))
	println(
		sliceString(str, 1, 3),
		len(sliceSlice(slice, 1, 3)), cap(sliceSlice(slice, 1, 3)),
		len(sliceArray(array, 1, 3)), cap(sliceArray(array, 1, 3)),
		len(sliceArrayPointer(arrayPointer, 1, 3)), cap(sliceArrayPointer(arrayPointer, 1, 3)),
		len(sliceThree(slice, 1, 3, 4)), cap(sliceThree(slice, 1, 3, 4)),
	)
	arrayFromPointer := shortSliceToArrayPointer(slice)
	arrayValue := shortSliceToArrayValue(slice)
	println(arrayFromPointer[0], arrayFromPointer[3], arrayValue[1], arrayValue[2])
}
