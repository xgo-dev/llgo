//go:build llgo

package binary

import "unsafe"

//llgo:template llgoSizeScalar llgoSizeSlice
func Size(v any) int

// These scalar types have a fixed wire size equal to their Go size.
type llgoFixedScalar interface {
	bool | int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64 |
		float32 | float64 | complex64 | complex128
}

func llgoSizeScalar[T llgoFixedScalar](v T) int {
	return int(unsafe.Sizeof(v))
}

func llgoSizeSlice[T llgoFixedScalar](v []T) int {
	var elem T
	return len(v) * int(unsafe.Sizeof(elem))
}
